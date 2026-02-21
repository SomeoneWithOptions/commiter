package main

import "testing"

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
