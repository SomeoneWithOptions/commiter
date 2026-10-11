package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

const defaultModel = "google/gemini-2.5-flash-lite"

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) (runErr error) {
	flags := flag.NewFlagSet("commiter", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	model := flags.String("model", defaultModel, "OpenRouter model to use")
	configPath := flags.String("config", "", "Config file path")
	pushChanges := flags.Bool("push", false, "Push to remote after committing")
	staged := flags.Bool("staged", false, "Only use staged changes")
	dryRun := flags.Bool("dry-run", false, "Preview diff and message without committing")
	edit := flags.Bool("edit", false, "Open Git's editor before committing")
	suggest := flags.Bool("suggest", false, "Print only the generated message without committing")
	messageFile := flags.String("message-file", "", "Read message from a file, or - for stdin; skip AI generation")
	var help bool
	flags.BoolVar(&help, "help", false, "Show detailed help and examples")
	flags.BoolVar(&help, "h", false, "Show detailed help and examples")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if help {
		return printHelp(stdout)
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected argument %q; use --help for usage", flags.Arg(0))
	}
	messageFileSet := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "message-file" {
			messageFileSet = true
		}
	})
	if messageFileSet && *messageFile == "" {
		return fmt.Errorf("--message-file requires a path or - for stdin")
	}
	if *suggest && (*dryRun || *edit || *pushChanges || messageFileSet) {
		return fmt.Errorf("--suggest cannot be combined with --dry-run, --edit, --push, or --message-file")
	}
	if messageFileSet && *edit {
		return fmt.Errorf("--message-file cannot be combined with --edit; edit the file before committing")
	}

	// Validate supplied messages before touching the index or calling Git.
	var message string
	if messageFileSet {
		var err error
		message, err = readCommitMessage(*messageFile, stdin)
		if err != nil {
			return err
		}
	}
	if err := isGitRepo(); err != nil {
		return err
	}

	var spinner *Spinner
	// Agent-oriented modes never emit a spinner. Other progress goes to stderr.
	progress := func(text string) {
		if *suggest || messageFileSet {
			return
		}
		if spinner == nil {
			spinner = NewSpinner(text)
			spinner.output = stderr
			spinner.Start()
		} else {
			spinner.UpdateMessage(text)
		}
	}
	stopProgress := func() {
		if spinner != nil {
			spinner.Stop()
			spinner = nil
		}
	}
	defer stopProgress()

	var indexSnapshot string
	restoreNeeded := false
	defer func() {
		if restoreNeeded {
			stopProgress()
			if err := restoreIndex(indexSnapshot); err != nil {
				if runErr != nil {
					runErr = fmt.Errorf("%w (also failed to restore staged state: %v)", runErr, err)
				} else {
					runErr = fmt.Errorf("failed to restore staged state: %w", err)
				}
			}
		}
	}()
	if !*staged {
		var err error
		indexSnapshot, err = snapshotIndex()
		if err != nil {
			return err
		}
		restoreNeeded = true
		progress("Staging all changes...")
		if err := stageAll(); err != nil {
			return err
		}
	} else {
		progress("Checking staged changes...")
	}
	diff, err := getStagedDiff()
	// Restore preview staging before generation, including when reading the diff fails.
	if restoreNeeded && (*dryRun || *suggest) {
		if restoreErr := restoreIndex(indexSnapshot); restoreErr != nil {
			restoreNeeded = false
			return fmt.Errorf("failed to restore index after preview: %w", restoreErr)
		}
		restoreNeeded = false
	}
	if err != nil {
		return err
	}
	if !messageFileSet {
		progress("Generating commit message...")
		message, err = generateCommitMessageWithConfig(diff, *model, *configPath)
		if err != nil {
			return err
		}
	}
	stopProgress()
	if *suggest {
		_, err := fmt.Fprintln(stdout, message)
		return err
	}
	if *dryRun {
		_, err := fmt.Fprintf(stdout, "--- Diff ---\n%s\n\n--- Commit Message ---\n%s\n", diff, message)
		return err
	}
	if _, err := fmt.Fprintln(stdout, message); err != nil {
		return err
	}
	if *edit {
		fmt.Fprintln(stderr, "Opening editor to confirm/edit commit message...")
		err = commitWithEditor(message)
	} else {
		progress("Committing...")
		err = commit(message)
	}
	if err != nil {
		return err
	}
	restoreNeeded = false // A push failure must not restore a pre-commit index.
	if *pushChanges {
		progress("Pushing to remote...")
		if err := push(); err != nil {
			return err
		}
		stopProgress()
		_, err = fmt.Fprintln(stdout, "Committed and pushed successfully.")
	} else {
		stopProgress()
		_, err = fmt.Fprintln(stdout, "Committed successfully.")
	}
	return err
}

func readCommitMessage(path string, stdin io.Reader) (string, error) {
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return "", fmt.Errorf("failed to read commit message from %q: %w", path, err)
	}
	message := string(data)
	if strings.TrimSpace(message) == "" {
		return "", fmt.Errorf("commit message from %q is empty", path)
	}
	if strings.ContainsRune(message, '\x00') {
		return "", fmt.Errorf("commit message from %q contains a NUL byte", path)
	}
	return message, nil
}
