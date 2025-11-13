package diff

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// ParseGitleaksDiff parses git diff output for .gitleaksignore
func ParseGitleaksDiff(baseBranch, headRef string) ([]DiffChange, error) {
	// Execute git diff command
	cmd := exec.Command("git", "diff", baseBranch+"..."+headRef, "--", ".gitleaksignore")
	output, err := cmd.Output()
	if err != nil {
		// If the command fails, it might be because there are no changes
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 1 with empty output means no changes
			if len(exitErr.Stderr) == 0 && len(output) == 0 {
				return []DiffChange{}, nil
			}
		}
		return nil, fmt.Errorf("git diff command failed: %w", err)
	}

	// If output is empty, no changes to .gitleaksignore
	if len(output) == 0 {
		return []DiffChange{}, nil
	}

	var changes []DiffChange
	scanner := bufio.NewScanner(bytes.NewReader(output))
	lineNum := 0
	position := 0

	// Regex to parse hunk headers: @@ -old_start,old_count +new_start,new_count @@
	hunkRegex := regexp.MustCompile(`^@@ -(\d+),?(\d*) \+(\d+),?(\d*) @@`)

	for scanner.Scan() {
		line := scanner.Text()
		position++

		// Check for hunk header
		if matches := hunkRegex.FindStringSubmatch(line); matches != nil {
			// matches[3] is the new file starting line number
			lineNum, _ = strconv.Atoi(matches[3])
			continue
		}

		// Skip file headers
		if strings.HasPrefix(line, "diff --git") ||
			strings.HasPrefix(line, "index ") ||
			strings.HasPrefix(line, "--- ") ||
			strings.HasPrefix(line, "+++ ") {
			continue
		}

		// Handle additions
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			content := strings.TrimPrefix(line, "+")
			content = strings.TrimSpace(content)

			// Skip empty lines and comments
			if content != "" && !strings.HasPrefix(content, "#") {
				changes = append(changes, DiffChange{
					FilePath:   ".gitleaksignore",
					Operation:  OperationAddition,
					LineNumber: lineNum,
					Content:    content,
					Position:   position,
				})
			}
			lineNum++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			// Handle deletions
			content := strings.TrimPrefix(line, "-")
			content = strings.TrimSpace(content)

			// Skip empty lines and comments
			if content != "" && !strings.HasPrefix(content, "#") {
				changes = append(changes, DiffChange{
					FilePath:  ".gitleaksignore",
					Operation: OperationDeletion,
					Content:   content,
					Position:  position,
				})
			}
		} else if !strings.HasPrefix(line, "\\") {
			// Context lines (no change)
			lineNum++
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning diff output: %w", err)
	}

	return changes, nil
}
