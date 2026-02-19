package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

var execCommand = exec.Command

func isGitRepo() error {
	cmd := execCommand("git", "rev-parse", "--is-inside-work-tree")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errorf("not a git repository: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func stageAll() error {
	cmd := execCommand("git", "add", "--all", ":/")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errorf("failed to stage changes: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func snapshotIndex() (string, error) {
	cmd := execCommand("git", "diff", "--cached", "--binary")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errorf("failed to snapshot staged changes: %s", strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func restoreIndex(snapshot string) error {
	resetCmd := execCommand("git", "reset")
	resetOutput, err := resetCmd.CombinedOutput()
	if err != nil {
		return errorf("failed to reset index: %s", strings.TrimSpace(string(resetOutput)))
	}

	if strings.TrimSpace(snapshot) == "" {
		return nil
	}

	applyCmd := execCommand("git", "apply", "--cached", "--whitespace=nowarn", "-")
	applyCmd.Stdin = bytes.NewBufferString(snapshot)
	applyOutput, err := applyCmd.CombinedOutput()
	if err != nil {
		return errorf("failed to restore staged changes: %s", strings.TrimSpace(string(applyOutput)))
	}

	return nil
}

func getStagedDiff() (string, error) {
	cmd := execCommand("git", "diff", "--cached")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errorf("failed to get diff: %s", strings.TrimSpace(string(output)))
	}
	diff := strings.TrimSpace(string(output))
	if diff == "" {
		return "", errorf("no changes to commit")
	}
	return diff, nil
}

func getFullDiff() (string, error) {
	cmd := execCommand("git", "diff", "HEAD")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errorf("failed to get diff: %s", strings.TrimSpace(string(output)))
	}
	diff := strings.TrimSpace(string(output))
	if diff == "" {
		return "", errorf("no changes to commit")
	}
	return diff, nil
}

func commit(message string) error {
	cmd := execCommand("git", "commit", "-m", message)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errorf("failed to commit: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func push() error {
	cmd := execCommand("git", "push")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errorf("failed to push: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func errorf(format string, args ...interface{}) error {
	return &gitError{msg: formatArgs(format, args)}
}

func formatArgs(format string, args []interface{}) string {
	return fmt.Sprintf(format, args...)
}

type gitError struct {
	msg string
}

func (e *gitError) Error() string {
	return e.msg
}
