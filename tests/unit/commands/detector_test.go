package commands_test

import (
	"testing"

	"github.com/epy0n0ff/gitleaks-diff-comment/internal/commands"
)

func TestDetectCommand(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedCmd   string
		expectedFound bool
	}{
		{
			name:          "basic clear command",
			input:         "@github-actions /clear",
			expectedCmd:   "clear",
			expectedFound: true,
		},
		{
			name:          "uppercase CLEAR command",
			input:         "@github-actions /CLEAR",
			expectedCmd:   "clear",
			expectedFound: true,
		},
		{
			name:          "mixed case Clear command",
			input:         "@github-actions /Clear",
			expectedCmd:   "clear",
			expectedFound: true,
		},
		{
			name:          "clear command with additional text",
			input:         "@github-actions /clear please remove old warnings",
			expectedCmd:   "clear",
			expectedFound: true,
		},
		{
			name:          "clear command with newline before",
			input:         "Some text\n@github-actions /clear",
			expectedCmd:   "clear",
			expectedFound: true,
		},
		{
			name:          "clear command with newline after",
			input:         "@github-actions /clear\nThank you!",
			expectedCmd:   "clear",
			expectedFound: true,
		},
		{
			name:          "clear command with multiple spaces",
			input:         "@github-actions    /clear",
			expectedCmd:   "clear",
			expectedFound: true,
		},
		{
			name:          "clear command with tab",
			input:         "@github-actions\t/clear",
			expectedCmd:   "clear",
			expectedFound: true,
		},
		{
			name:          "mention without command",
			input:         "@github-actions hello",
			expectedCmd:   "",
			expectedFound: false,
		},
		{
			name:          "command without mention",
			input:         "/clear without mention",
			expectedCmd:   "",
			expectedFound: false,
		},
		{
			name:          "wrong bot mention",
			input:         "@github-actions-bot /clear",
			expectedCmd:   "",
			expectedFound: false,
		},
		{
			name:          "empty comment",
			input:         "",
			expectedCmd:   "",
			expectedFound: false,
		},
		{
			name:          "only whitespace",
			input:         "   \n\t  ",
			expectedCmd:   "",
			expectedFound: false,
		},
		{
			name:          "partial match - clearance",
			input:         "@github-actions /clearance",
			expectedCmd:   "",
			expectedFound: false,
		},
		{
			name:          "command in middle of text",
			input:         "Hey @github-actions /clear this please, thanks!",
			expectedCmd:   "clear",
			expectedFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, found := commands.DetectCommand(tt.input)

			if found != tt.expectedFound {
				t.Errorf("DetectCommand(%q) found = %v, want %v", tt.input, found, tt.expectedFound)
			}

			if cmd != tt.expectedCmd {
				t.Errorf("DetectCommand(%q) cmd = %q, want %q", tt.input, cmd, tt.expectedCmd)
			}
		})
	}
}

func TestDetectCommand_CaseInsensitivity(t *testing.T) {
	// Test all case variations of the command
	variations := []string{
		"@github-actions /clear",
		"@github-actions /Clear",
		"@github-actions /CLEAR",
		"@github-actions /cLeAr",
		"@GITHUB-ACTIONS /clear",  // bot mention is case-sensitive in pattern, but GitHub normalizes
	}

	for _, input := range variations {
		cmd, found := commands.DetectCommand(input)

		if !found {
			t.Errorf("DetectCommand(%q) should detect command", input)
		}

		if cmd != "clear" {
			t.Errorf("DetectCommand(%q) should return lowercase 'clear', got %q", input, cmd)
		}
	}
}
