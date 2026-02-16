package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// HelperProcess is used to mock exec.Command
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	// Check the commant add being run
	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "No command\n")
		os.Exit(2)
	}

	cmd, subargs := args[0], args[1:]

	switch cmd {
	case "git":
		if len(subargs) > 0 {
			switch subargs[0] {
			case "rev-parse":
				// isGitRepo
				os.Exit(0)
			case "add":
				// stageAll
				os.Exit(0)
			case "diff":
				// getStagedDiff
				// Return diff content if checking for changes, or empty if testing no changes
				if os.Getenv("MOCK_GIT_DIFF_EMPTY") == "1" {
					fmt.Print("")
				} else {
					fmt.Print("diff content")
				}
				os.Exit(0)
			case "commit":
				// commit
				os.Exit(0)
			case "push":
				// push
				os.Exit(0)
			}
		}
	}
	fmt.Fprintf(os.Stderr, "Unknown command %q\n", cmd)
	os.Exit(2)
}

func mockExecCommand(command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess", "--", command}
	cs = append(cs, args...)
	cmd := exec.Command(os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	// Pass through specific mock env vars
	if val := os.Getenv("MOCK_GIT_DIFF_EMPTY"); val != "" {
		cmd.Env = append(cmd.Env, "MOCK_GIT_DIFF_EMPTY="+val)
	}
	return cmd
}

func TestStageAll(t *testing.T) {
	execCommand = mockExecCommand
	defer func() { execCommand = exec.Command }()

	if err := stageAll(); err != nil {
		t.Errorf("stageAll() failed: %v", err)
	}
}

func TestGetStagedDiff(t *testing.T) {
	execCommand = mockExecCommand
	defer func() { execCommand = exec.Command }()

	// Test with changes
	diff, err := getStagedDiff()
	if err != nil {
		t.Errorf("getStagedDiff() failed: %v", err)
	}
	if diff != "diff content" {
		t.Errorf("Expected 'diff content', got %q", diff)
	}

	// Test with no changes
	os.Setenv("MOCK_GIT_DIFF_EMPTY", "1")
	defer os.Unsetenv("MOCK_GIT_DIFF_EMPTY")

	_, err = getStagedDiff()
	if err == nil {
		t.Error("Expected error when no changes are staged, got nil")
	} else if !strings.Contains(err.Error(), "no changes to commit") {
		t.Errorf("Expected 'no changes to commit' error, got %v", err)
	}
}

func TestCommit(t *testing.T) {
	execCommand = mockExecCommand
	defer func() { execCommand = exec.Command }()

	if err := commit("test message"); err != nil {
		t.Errorf("commit() failed: %v", err)
	}
}
