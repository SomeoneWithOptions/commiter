package main

import "testing"

func TestCleanCommitMessage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "Feat: add new feature",
			expected: "add new feature",
		},
		{
			input:    "fix(ui): fix button color.",
			expected: "fix button color",
		},
		{
			input:    "  Update   documentation.  ",
			expected: "update documentation",
		},
		{
			input:    "Refactor: CLEAN UP CODE",
			expected: "clean up code",
		},
		{
			input:    "simple message",
			expected: "simple message",
		},
		{
			input:    "feat(auth): add login endpoint",
			expected: "add login endpoint",
		},
	}

	for _, tt := range tests {
		result := cleanCommitMessage(tt.input)
		if result != tt.expected {
			t.Errorf("cleanCommitMessage(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
