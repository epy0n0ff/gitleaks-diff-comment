package commands

import (
	"regexp"
	"strings"
)

// commandPattern matches @github-actions mentions followed by /clear command (case-insensitive)
// Pattern: @github-actions + whitespace + /clear
var commandPattern = regexp.MustCompile(`(?i)@github-actions\s+/(clear)`)

// DetectCommand detects if a comment body contains a valid command
// Returns the command type (lowercase) and a boolean indicating if a command was found
func DetectCommand(commentBody string) (string, bool) {
	matches := commandPattern.FindStringSubmatch(commentBody)
	if len(matches) < 2 {
		return "", false
	}

	// Return lowercase command type
	return strings.ToLower(matches[1]), true
}
