package main

import (
	"flag"
	"fmt"
	"os"
)

const defaultModel = "arcee-ai/trinity-mini:free"

func main() {
	var model string
	var pushChanges bool
	var staged bool
	var dryRun bool
	flag.StringVar(&model, "model", defaultModel, "OpenRouter model to use")
	flag.BoolVar(&pushChanges, "push", false, "Push to remote after committing")
	flag.BoolVar(&staged, "staged", false, "Only commit staged changes")
	flag.BoolVar(&dryRun, "dry-run", false, "Preview diff and commit message without committing")
	flag.Parse()

	if err := isGitRepo(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var indexSnapshot string
	rollbackOnError := !staged && !dryRun
	if rollbackOnError {
		var err error
		indexSnapshot, err = snapshotIndex()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	var spinner *Spinner
	handleError := func(err error, shouldRollback bool) {
		if spinner != nil {
			spinner.Stop()
		}

		if shouldRollback {
			if restoreErr := restoreIndex(indexSnapshot); restoreErr != nil {
				fmt.Fprintf(os.Stderr, "Error: %v (also failed to restore staged state: %v)\n", err, restoreErr)
				os.Exit(1)
			}
		}

		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !staged && !dryRun {
		spinner = NewSpinner("Staging all changes...")
		spinner.Start()

		if err := stageAll(); err != nil {
			handleError(err, rollbackOnError)
		}
	} else if staged {
		spinner = NewSpinner("Checking staged changes...")
		spinner.Start()
	}

	var diff string
	var err error
	if dryRun && !staged {
		diff, err = getFullDiff()
	} else {
		diff, err = getStagedDiff()
	}
	if err != nil {
		handleError(err, rollbackOnError)
	}

	if spinner == nil {
		spinner = NewSpinner("Generating commit message...")
		spinner.Start()
	} else {
		spinner.UpdateMessage("Generating commit message...")
	}

	message, err := generateCommitMessage(diff, model)
	if err != nil {
		handleError(err, rollbackOnError)
	}

	spinner.Stop()

	if dryRun {
		fmt.Println("--- Diff ---")
		fmt.Println(diff)
		fmt.Println()
		fmt.Println("--- Commit Message ---")
		fmt.Println(message)
		return
	}

	fmt.Println(message)

	spinner = NewSpinner("Committing...")
	spinner.Start()

	if err := commit(message); err != nil {
		handleError(err, rollbackOnError)
	}

	rollbackOnError = false

	if pushChanges {
		spinner.UpdateMessage("Pushing to remote...")
		if err := push(); err != nil {
			handleError(err, false)
		}
		spinner.Stop()
		fmt.Println("Committed and pushed successfully.")
	} else {
		spinner.Stop()
		fmt.Println("Committed successfully.")
	}
}
