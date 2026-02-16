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
	flag.StringVar(&model, "model", defaultModel, "OpenRouter model to use")
	flag.BoolVar(&pushChanges, "push", false, "Push to remote after committing")
	flag.BoolVar(&staged, "staged", false, "Only commit staged changes")
	flag.Parse()

	if err := isGitRepo(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	var spinner *Spinner
	if !staged {
		spinner = NewSpinner("Staging all changes...")
		spinner.Start()

		if err := stageAll(); err != nil {
			spinner.Stop()
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		spinner = NewSpinner("Checking staged changes...")
		spinner.Start()
	}

	diff, err := getStagedDiff()
	if err != nil {
		spinner.Stop()
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	spinner.UpdateMessage("Generating commit message...")
	message, err := generateCommitMessage(diff, model)
	if err != nil {
		spinner.Stop()
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	spinner.Stop()
	fmt.Println(message)

	spinner = NewSpinner("Committing...")
	spinner.Start()

	if err := commit(message); err != nil {
		spinner.Stop()
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if pushChanges {
		spinner.UpdateMessage("Pushing to remote...")
		if err := push(); err != nil {
			spinner.Stop()
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		spinner.Stop()
		fmt.Println("Committed and pushed successfully.")
	} else {
		spinner.Stop()
		fmt.Println("Committed successfully.")
	}
}
