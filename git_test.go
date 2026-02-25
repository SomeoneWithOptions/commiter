package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func hasArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}

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
				if hasArg(subargs, "--binary") {
					fmt.Print(os.Getenv("MOCK_GIT_DIFF_BINARY"))
					os.Exit(0)
				}

				// getStagedDiff
				// Return diff content if checking for changes, or empty if testing no changes
				if os.Getenv("MOCK_GIT_DIFF_EMPTY") == "1" {
					fmt.Print("")
				} else {
					fmt.Print("diff content")
				}
				os.Exit(0)
			case "commit":
				if os.Getenv("MOCK_GIT_EXPECT_EDIT") == "1" && !hasArg(subargs, "--edit") {
					fmt.Fprint(os.Stderr, "missing --edit")
					os.Exit(1)
				}
				if os.Getenv("MOCK_GIT_EXPECT_MESSAGE_FLAG") == "1" && !hasArg(subargs, "-m") {
					fmt.Fprint(os.Stderr, "missing -m")
					os.Exit(1)
				}
				if os.Getenv("MOCK_GIT_COMMIT_FAIL") == "1" {
					fmt.Fprint(os.Stderr, "commit failed")
					os.Exit(1)
				}
				os.Exit(0)
			case "push":
				// push
				os.Exit(0)
			case "reset":
				if os.Getenv("MOCK_GIT_RESET_FAIL") == "1" {
					fmt.Fprint(os.Stderr, "reset failed")
					os.Exit(1)
				}
				os.Exit(0)
			case "apply":
				if os.Getenv("MOCK_GIT_APPLY_FAIL") == "1" {
					fmt.Fprint(os.Stderr, "apply failed")
					os.Exit(1)
				}
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
	if val := os.Getenv("MOCK_GIT_DIFF_BINARY"); val != "" {
		cmd.Env = append(cmd.Env, "MOCK_GIT_DIFF_BINARY="+val)
	}
	if val := os.Getenv("MOCK_GIT_RESET_FAIL"); val != "" {
		cmd.Env = append(cmd.Env, "MOCK_GIT_RESET_FAIL="+val)
	}
	if val := os.Getenv("MOCK_GIT_APPLY_FAIL"); val != "" {
		cmd.Env = append(cmd.Env, "MOCK_GIT_APPLY_FAIL="+val)
	}
	if val := os.Getenv("MOCK_GIT_EXPECT_EDIT"); val != "" {
		cmd.Env = append(cmd.Env, "MOCK_GIT_EXPECT_EDIT="+val)
	}
	if val := os.Getenv("MOCK_GIT_EXPECT_MESSAGE_FLAG"); val != "" {
		cmd.Env = append(cmd.Env, "MOCK_GIT_EXPECT_MESSAGE_FLAG="+val)
	}
	if val := os.Getenv("MOCK_GIT_COMMIT_FAIL"); val != "" {
		cmd.Env = append(cmd.Env, "MOCK_GIT_COMMIT_FAIL="+val)
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

func TestGetFullDiff(t *testing.T) {
	t.Skip("getFullDiff is currently unused but kept for potential future use")
	execCommand = mockExecCommand
	defer func() { execCommand = exec.Command }()

	// Test with changes
	diff, err := getFullDiff()
	if err != nil {
		t.Errorf("getFullDiff() failed: %v", err)
	}
	if diff != "diff content" {
		t.Errorf("Expected 'diff content', got %q", diff)
	}

	// Test with no changes
	os.Setenv("MOCK_GIT_DIFF_EMPTY", "1")
	defer os.Unsetenv("MOCK_GIT_DIFF_EMPTY")

	_, err = getFullDiff()
	if err == nil {
		t.Error("Expected error when no changes are present, got nil")
	} else if !strings.Contains(err.Error(), "no changes to commit") {
		t.Errorf("Expected 'no changes to commit' error, got %v", err)
	}
}

func TestCommit(t *testing.T) {
	execCommand = mockExecCommand
	defer func() { execCommand = exec.Command }()

	os.Setenv("MOCK_GIT_EXPECT_MESSAGE_FLAG", "1")
	defer os.Unsetenv("MOCK_GIT_EXPECT_MESSAGE_FLAG")

	if err := commit("test message"); err != nil {
		t.Errorf("commit() failed: %v", err)
	}
}

func TestCommitWithEditor(t *testing.T) {
	execCommand = mockExecCommand
	defer func() { execCommand = exec.Command }()

	os.Setenv("MOCK_GIT_EXPECT_EDIT", "1")
	defer os.Unsetenv("MOCK_GIT_EXPECT_EDIT")
	os.Setenv("MOCK_GIT_EXPECT_MESSAGE_FLAG", "1")
	defer os.Unsetenv("MOCK_GIT_EXPECT_MESSAGE_FLAG")

	if err := commitWithEditor("test message"); err != nil {
		t.Errorf("commitWithEditor() failed: %v", err)
	}
}

func TestCommitWithEditorFailure(t *testing.T) {
	execCommand = mockExecCommand
	defer func() { execCommand = exec.Command }()

	os.Setenv("MOCK_GIT_COMMIT_FAIL", "1")
	defer os.Unsetenv("MOCK_GIT_COMMIT_FAIL")

	err := commitWithEditor("test message")
	if err == nil {
		t.Fatal("Expected commitWithEditor() to fail")
	}
	if !strings.Contains(err.Error(), "failed to commit") {
		t.Errorf("Expected commit failure message, got %v", err)
	}
}

func TestSnapshotIndex(t *testing.T) {
	execCommand = mockExecCommand
	defer func() { execCommand = exec.Command }()

	os.Setenv("MOCK_GIT_DIFF_BINARY", "binary patch")
	defer os.Unsetenv("MOCK_GIT_DIFF_BINARY")

	snapshot, err := snapshotIndex()
	if err != nil {
		t.Errorf("snapshotIndex() failed: %v", err)
	}
	if snapshot != "binary patch" {
		t.Errorf("Expected 'binary patch', got %q", snapshot)
	}
}

func TestRestoreIndex(t *testing.T) {
	execCommand = mockExecCommand
	defer func() { execCommand = exec.Command }()

	if err := restoreIndex("some patch"); err != nil {
		t.Errorf("restoreIndex() failed: %v", err)
	}
}

func TestRestoreIndexSkipsApplyForEmptySnapshot(t *testing.T) {
	execCommand = mockExecCommand
	defer func() { execCommand = exec.Command }()

	os.Setenv("MOCK_GIT_APPLY_FAIL", "1")
	defer os.Unsetenv("MOCK_GIT_APPLY_FAIL")

	if err := restoreIndex(""); err != nil {
		t.Errorf("restoreIndex() with empty snapshot failed: %v", err)
	}
}

func TestRestoreIndexResetFailure(t *testing.T) {
	execCommand = mockExecCommand
	defer func() { execCommand = exec.Command }()

	os.Setenv("MOCK_GIT_RESET_FAIL", "1")
	defer os.Unsetenv("MOCK_GIT_RESET_FAIL")

	err := restoreIndex("some patch")
	if err == nil {
		t.Fatal("Expected restoreIndex() to fail when reset fails")
	}
	if !strings.Contains(err.Error(), "failed to reset index") {
		t.Errorf("Expected reset failure message, got %v", err)
	}
}

func TestRestoreIndexApplyFailure(t *testing.T) {
	execCommand = mockExecCommand
	defer func() { execCommand = exec.Command }()

	os.Setenv("MOCK_GIT_APPLY_FAIL", "1")
	defer os.Unsetenv("MOCK_GIT_APPLY_FAIL")

	err := restoreIndex("some patch")
	if err == nil {
		t.Fatal("Expected restoreIndex() to fail when apply fails")
	}
	if !strings.Contains(err.Error(), "failed to restore staged changes") {
		t.Errorf("Expected apply failure message, got %v", err)
	}
}
