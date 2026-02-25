package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCleanCommitMessage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "Feat: add new feature",
			expected: "feat: add new feature",
		},
		{
			input:    "fix(ui): fix button color.",
			expected: "fix(ui): fix button color",
		},
		{
			input:    "  Update   documentation.  ",
			expected: "update documentation",
		},
		{
			input:    "Refactor: CLEAN UP CODE",
			expected: "refactor: clean up code",
		},
		{
			input:    "simple message",
			expected: "simple message",
		},
		{
			input:    "feat(auth): add login endpoint",
			expected: "feat(auth): add login endpoint",
		},
		{
			input:    "implementgetfulldiff function and test",
			expected: "implement getfulldiff function and test",
		},
		{
			input:    "addtest cases for git commands",
			expected: "add test cases for git commands",
		},
		{
			input:    "addpushand staged flags",
			expected: "add push and staged flags",
		},
		// Real-world regressions: correctly-spaced variants of messages the LLM
		// previously produced with merged words. cleanCommitMessage must not
		// disturb already-correct spacing.
		{
			input:    "feat(httpapi): add local usage fallback handling",
			expected: "feat(httpapi): add local usage fallback handling",
		},
		{
			input:    "style(styles.css): change position to right",
			expected: "style(styles.css): change position to right",
		},
		{
			input:    "test: add test for copying user messages",
			expected: "test: add test for copying user messages",
		},
		{
			input:    "chore: sync models on a schedule",
			expected: "chore: sync models on a schedule",
		},
	}

	for _, tt := range tests {
		result := cleanCommitMessage(tt.input)
		if result != tt.expected {
			t.Errorf("cleanCommitMessage(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestExtractMessageContent(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{
			name: "plain string content",
			raw:  `"feat: add tests"`,
			want: "feat: add tests",
		},
		{
			name: "multipart text content",
			raw:  `[{"type":"text","text":"feat:"},{"type":"text","text":"add tests"}]`,
			want: "feat: add tests",
		},
		{
			name:    "unsupported object content",
			raw:     `{"text":"feat: add tests"}`,
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractMessageContent(json.RawMessage(tt.raw))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("extractMessageContent(%s) expected error", tt.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("extractMessageContent(%s) unexpected error: %v", tt.raw, err)
			}
			if got != tt.want {
				t.Fatalf("extractMessageContent(%s) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestLimitDiffForPrompt(t *testing.T) {
	short := "diff --git a/file b/file\n+line\n"
	if got := limitDiffForPrompt(short, 100); got != short {
		t.Fatalf("expected short diff unchanged, got %q", got)
	}

	long := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const marker = "\n\n[... diff truncated to fit model context ...]\n\n"
	gotTiny := limitDiffForPrompt(long, 30)
	if len(gotTiny) != 30 {
		t.Fatalf("expected length 30, got %d (%q)", len(gotTiny), gotTiny)
	}
	if gotTiny != long[:30] {
		t.Fatalf("expected prefix-only truncation for tiny budget, got %q", gotTiny)
	}

	got := limitDiffForPrompt(long, 59)
	if len(got) != 59 {
		t.Fatalf("expected length 59, got %d (%q)", len(got), got)
	}
	if got[0:4] != "abcd" {
		t.Fatalf("expected preserved head, got %q", got[0:4])
	}
	if got[len(got)-4:] != "6789" {
		t.Fatalf("expected preserved tail, got %q", got[len(got)-4:])
	}
	if !strings.Contains(got, marker) {
		t.Fatalf("expected marker in truncated diff: %q", got)
	}
}
