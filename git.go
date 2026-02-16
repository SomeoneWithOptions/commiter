package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func isGitRepo() error {
	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errorf("not a git repository: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func stageAll() error {
	cmd := exec.Command("git", "add", "-A")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errorf("failed to stage changes: %s", strings.TrimSpace(string(output)))
	}
	return nil
}

func getStagedDiff() (string, error) {
	cmd := exec.Command("git", "diff", "--cached")
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
	cmd := exec.Command("git", "commit", "-m", message)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errorf("failed to commit: %s", strings.TrimSpace(string(output)))
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
