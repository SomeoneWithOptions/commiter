package main

import (
	"flag"
	"fmt"
	"os"
)

const defaultModel = "arcee-ai/trinity-mini:free"

func main() {
	var model string
	flag.StringVar(&model, "model", defaultModel, "OpenRouter model to use")
	flag.Parse()

	if err := isGitRepo(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	spinner := NewSpinner("Staging all changes...")
	spinner.Start()

	if err := stageAll(); err != nil {
		spinner.Stop()
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
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

	spinner.Stop()
	fmt.Println("Committed successfully.")
}
