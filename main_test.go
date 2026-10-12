package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type testTransport func(*http.Request) (*http.Response, error)

func (f testTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func mockGeneration(t *testing.T, wantDiff string, failure bool) *int {
	t.Helper()
	calls := new(int)
	previous := http.DefaultTransport
	http.DefaultTransport = testTransport(func(r *http.Request) (*http.Response, error) {
		*calls++
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		if wantDiff != "" && !strings.Contains(string(body), wantDiff) {
			t.Errorf("API request missing diff %q: %s", wantDiff, body)
		}
		if failure {
			return nil, errors.New("generation failed")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":"fix: update tracked file"}}]}`)),
			Header:     make(http.Header),
		}, nil
	})
	t.Cleanup(func() { http.DefaultTransport = previous })
	return calls
}

func gitInTest(t *testing.T, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return string(output)
}

func writeInTest(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func setupRepo(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	// Keep tests independent of user hooks, signing, editor, and Git config.
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	gitInTest(t, "init", "-q")
	gitInTest(t, "config", "user.name", "Test")
	gitInTest(t, "config", "user.email", "test@example.com")
	writeInTest(t, "tracked.txt", "original\n")
	gitInTest(t, "add", "tracked.txt")
	gitInTest(t, "commit", "-qm", "initial")
}

func generationConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	writeInTest(t, path, `{"openrouter":{"api_key":"test-key","model":"test/model"}}`)
	return path
}

func TestHelpOutsideRepository(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, arg := range []string{"--help", "-h"} {
		var stdout, stderr bytes.Buffer
		if err := run([]string{arg}, strings.NewReader(""), &stdout, &stderr); err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"--staged", "--suggest", "--message-file", "--dry-run", "--edit", "--push", "--model", "--config", "--help", "AGENT WORKFLOW", "Example:", "openrouter.model"} {
			if !strings.Contains(stdout.String(), want) {
				t.Errorf("help missing %q", want)
			}
		}
		if stderr.Len() != 0 {
			t.Errorf("help wrote stderr: %q", stderr.String())
		}
	}
}

func TestInvalidOptionsBeforeGit(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, args := range [][]string{
		{"--suggest", "--push"}, {"--suggest", "--edit"},
		{"--suggest", "--dry-run"}, {"--suggest", "--message-file", "x"},
		{"--message-file", "x", "--edit"}, {"--message-file="},
		{"--message-file"}, {"--unknown"}, {"unexpected"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := run(args, strings.NewReader(""), &stdout, &stderr)
			if err == nil || strings.Contains(err.Error(), "not a git repository") {
				t.Fatalf("expected option error before Git, got %v", err)
			}
			if stdout.Len() != 0 {
				t.Errorf("invalid options wrote stdout: %q", stdout.String())
			}
		})
	}
}

func TestSuggestPreservesStaging(t *testing.T) {
	for _, stagedOnly := range []bool{false, true} {
		t.Run(map[bool]string{false: "all", true: "staged"}[stagedOnly], func(t *testing.T) {
			setupRepo(t)
			writeInTest(t, "tracked.txt", "staged version\n")
			gitInTest(t, "add", "tracked.txt")
			writeInTest(t, "tracked.txt", "unstaged version\n")
			writeInTest(t, "new.txt", "untracked content\n")
			beforeIndex := gitInTest(t, "diff", "--cached", "--binary")
			beforeStatus := gitInTest(t, "status", "--porcelain")
			beforeHead := gitInTest(t, "rev-parse", "HEAD")
			wantDiff := "untracked content"
			args := []string{"--suggest", "--config", generationConfig(t)}
			if stagedOnly {
				args = append(args, "--staged")
				wantDiff = "staged version"
			}
			calls := mockGeneration(t, wantDiff, false)
			var stdout, stderr bytes.Buffer
			if err := run(args, strings.NewReader(""), &stdout, &stderr); err != nil {
				t.Fatal(err)
			}
			if stdout.String() != "fix: update tracked file\n" || stderr.Len() != 0 {
				t.Fatalf("unclean suggestion output: stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
			if *calls != 1 {
				t.Errorf("generation calls = %d", *calls)
			}
			if got := gitInTest(t, "diff", "--cached", "--binary"); got != beforeIndex {
				t.Errorf("staging changed: %q", got)
			}
			if got := gitInTest(t, "status", "--porcelain"); got != beforeStatus {
				t.Errorf("status changed: %q", got)
			}
			if gitInTest(t, "rev-parse", "HEAD") != beforeHead {
				t.Error("suggest committed changes")
			}
		})
	}
}

func TestSuggestFailureRestoresStaging(t *testing.T) {
	setupRepo(t)
	writeInTest(t, "tracked.txt", "staged version\n")
	gitInTest(t, "add", "tracked.txt")
	writeInTest(t, "tracked.txt", "unstaged version\n")
	before := gitInTest(t, "diff", "--cached", "--binary")
	mockGeneration(t, "unstaged version", true)
	var stdout, stderr bytes.Buffer
	err := run([]string{"--suggest", "--config", generationConfig(t)}, strings.NewReader(""), &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "generation failed") {
		t.Fatalf("expected generation failure, got %v", err)
	}
	if stdout.Len() != 0 || gitInTest(t, "diff", "--cached", "--binary") != before {
		t.Fatal("failed suggestion changed staging or printed a message")
	}
}

func TestMessageFileCommitsWithoutAPI(t *testing.T) {
	for _, pathMode := range []string{"file", "stdin"} {
		for _, stagedOnly := range []bool{false, true} {
			t.Run(fmtTestName(pathMode, stagedOnly), func(t *testing.T) {
				setupRepo(t)
				writeInTest(t, "tracked.txt", "changed\n")
				gitInTest(t, "add", "tracked.txt")
				writeInTest(t, "new.txt", "new\n")
				message := "fix(auth): preserve Upload body\n\nKeep API payload intact.\nVerified with integration tests.\n"
				path := "-"
				if pathMode == "file" {
					path = filepath.Join(t.TempDir(), "message.txt")
					writeInTest(t, path, message)
				}
				calls := mockGeneration(t, "", true)
				args := []string{"--message-file", path, "--config", "missing-config.json"}
				if stagedOnly {
					args = append(args, "--staged")
				}
				var stdout, stderr bytes.Buffer
				if err := run(args, strings.NewReader(message), &stdout, &stderr); err != nil {
					t.Fatal(err)
				}
				if *calls != 0 || stderr.Len() != 0 {
					t.Fatalf("unexpected generation/progress: calls=%d stderr=%q", *calls, stderr.String())
				}
				if got := gitInTest(t, "log", "-1", "--format=%B"); strings.TrimSpace(got) != strings.TrimSpace(message) {
					t.Fatalf("message changed: %q", got)
				}
				files := gitInTest(t, "show", "--format=", "--name-only", "HEAD")
				if strings.Contains(files, "new.txt") == stagedOnly {
					t.Errorf("wrong commit scope: %q", files)
				}
			})
		}
	}
}

func fmtTestName(pathMode string, staged bool) string {
	if staged {
		return pathMode + "/staged"
	}
	return pathMode + "/all"
}

func TestMessageFileDryRun(t *testing.T) {
	setupRepo(t)
	writeInTest(t, "tracked.txt", "changed\n")
	before := gitInTest(t, "status", "--porcelain")
	head := gitInTest(t, "rev-parse", "HEAD")
	var stdout, stderr bytes.Buffer
	err := run([]string{"--message-file", "-", "--dry-run"}, strings.NewReader("fix: preview\n\nBody.\n"), &stdout, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "--- Diff ---") || !strings.Contains(stdout.String(), "fix: preview\n\nBody.") {
		t.Errorf("missing preview: %q", stdout.String())
	}
	if gitInTest(t, "rev-parse", "HEAD") != head || gitInTest(t, "status", "--porcelain") != before {
		t.Error("dry run changed repo state")
	}
}

func TestInvalidMessageBeforeStaging(t *testing.T) {
	setupRepo(t)
	writeInTest(t, "tracked.txt", "changed\n")
	before := gitInTest(t, "status", "--porcelain")
	for _, message := range []string{"", " \n\t", "fix: invalid\x00message"} {
		var stdout, stderr bytes.Buffer
		if err := run([]string{"--message-file", "-"}, strings.NewReader(message), &stdout, &stderr); err == nil {
			t.Errorf("accepted invalid message %q", message)
		}
		if stdout.Len() != 0 || gitInTest(t, "status", "--porcelain") != before {
			t.Error("invalid message changed state or printed stdout")
		}
	}
	var stdout, stderr bytes.Buffer
	if err := run([]string{"--message-file", "missing.txt"}, strings.NewReader(""), &stdout, &stderr); err == nil {
		t.Error("accepted missing message file")
	}
}

func TestPushFailureKeepsCommit(t *testing.T) {
	setupRepo(t)
	writeInTest(t, "tracked.txt", "changed\n")
	before := gitInTest(t, "rev-parse", "HEAD")
	var stdout, stderr bytes.Buffer
	err := run([]string{"--message-file", "-", "--push"}, strings.NewReader("fix: update\n"), &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "failed to push") {
		t.Fatalf("expected push failure, got %v", err)
	}
	if gitInTest(t, "rev-parse", "HEAD") == before || gitInTest(t, "status", "--porcelain") != "" {
		t.Error("push failure lost commit or restored old index")
	}
}

func TestCommitFailureRestoresStaging(t *testing.T) {
	setupRepo(t)
	writeInTest(t, "tracked.txt", "staged version\n")
	gitInTest(t, "add", "tracked.txt")
	writeInTest(t, "tracked.txt", "unstaged version\n")
	before := gitInTest(t, "diff", "--cached", "--binary")
	if err := os.WriteFile(".git/hooks/pre-commit", []byte("#!/bin/sh\nexit 1\n"), 0700); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err := run([]string{"--message-file", "-"}, strings.NewReader("fix: update\n"), &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "failed to commit") {
		t.Fatalf("expected commit failure, got %v", err)
	}
	if gitInTest(t, "diff", "--cached", "--binary") != before {
		t.Error("commit failure changed staging")
	}
}

func TestGeneratedCommitAndDryRun(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		t.Run(fmtTestName("generated", dryRun), func(t *testing.T) {
			setupRepo(t)
			writeInTest(t, "tracked.txt", "changed\n")
			before := gitInTest(t, "status", "--porcelain")
			head := gitInTest(t, "rev-parse", "HEAD")
			calls := mockGeneration(t, "changed", false)
			args := []string{"--config", generationConfig(t)}
			if dryRun {
				args = append(args, "--dry-run")
			}
			var stdout, stderr bytes.Buffer
			if err := run(args, strings.NewReader(""), &stdout, &stderr); err != nil {
				t.Fatal(err)
			}
			if *calls != 1 || !strings.Contains(stdout.String(), "fix: update tracked file") {
				t.Errorf("missing generated message: %q (calls=%d)", stdout.String(), *calls)
			}
			if strings.ContainsAny(stdout.String(), "\r\x1b") {
				t.Errorf("spinner leaked to stdout: %q", stdout.String())
			}
			if dryRun {
				if gitInTest(t, "rev-parse", "HEAD") != head || gitInTest(t, "status", "--porcelain") != before {
					t.Error("generated dry run changed repository")
				}
			} else if gitInTest(t, "rev-parse", "HEAD") == head || gitInTest(t, "status", "--porcelain") != "" {
				t.Error("default run failed to commit changes")
			}
		})
	}
}

func TestSuggestWithoutChanges(t *testing.T) {
	setupRepo(t)
	calls := mockGeneration(t, "", true)
	var stdout, stderr bytes.Buffer
	err := run([]string{"--suggest"}, strings.NewReader(""), &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "no changes to commit") {
		t.Fatalf("expected no-changes error, got %v", err)
	}
	if *calls != 0 || stdout.Len() != 0 || gitInTest(t, "status", "--porcelain") != "" {
		t.Error("empty suggestion called API, wrote output, or changed state")
	}
}

func TestSuggestBeforeFirstCommit(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	gitInTest(t, "init", "-q")
	writeInTest(t, "staged.txt", "staged\n")
	gitInTest(t, "add", "staged.txt")
	writeInTest(t, "new.txt", "untracked content\n")
	before := gitInTest(t, "status", "--porcelain")
	mockGeneration(t, "untracked content", false)
	var stdout, stderr bytes.Buffer
	if err := run([]string{"--suggest", "--config", generationConfig(t)}, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if gitInTest(t, "status", "--porcelain") != before {
		t.Error("suggest changed staging before first commit")
	}
}
