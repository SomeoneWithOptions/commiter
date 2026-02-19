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
	// We need to snapshot and potentially rollback if we are doing a real commit (!staged && !dryRun)
	// OR if we are doing a full dry-run (dryRun && !staged) where we need to temporarily stage everything
	// to get the diff including untracked files.
	needsSnapshot := !staged
	rollbackOnError := !staged && !dryRun // Only error rollbacks for actual commits

	if needsSnapshot {
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

	// For a full run (real or dry), we stage all changes
	if !staged {
		if !dryRun {
			spinner = NewSpinner("Staging all changes...")
			spinner.Start()
		}
		if err := stageAll(); err != nil {
			handleError(err, rollbackOnError)
		}
	} else if staged {
		spinner = NewSpinner("Checking staged changes...")
		spinner.Start()
	}

	var diff string
	var err error

	// Now everything we want is staged (either by us or by the user)
	diff, err = getStagedDiff()

	// If it's a dry-run and we temporarily staged everything, restore the index now
	if dryRun && !staged {
		if restoreErr := restoreIndex(indexSnapshot); restoreErr != nil {
			handleError(fmt.Errorf("failed to restore index after diff: %w", restoreErr), false)
		}
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
