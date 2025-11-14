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
	// Build list of diff strategies to try
	var strategies [][]string

	// Strategy 1: origin/base..HEAD (two-dot, shows changes between branches)
	if baseBranch != "" {
		strategies = append(strategies, []string{"diff", "origin/" + baseBranch + "..HEAD", "--", ".gitleaksignore"})
	}

	// Strategy 2: base..HEAD (without origin/ prefix)
	if baseBranch != "" {
		strategies = append(strategies, []string{"diff", baseBranch + "..HEAD", "--", ".gitleaksignore"})
	}

	// Strategy 3: origin/base...HEAD (three-dot, shows changes since common ancestor)
	if baseBranch != "" {
		strategies = append(strategies, []string{"diff", "origin/" + baseBranch + "...HEAD", "--", ".gitleaksignore"})
	}

	// Strategy 4: HEAD~1..HEAD (single commit diff)
	strategies = append(strategies, []string{"diff", "HEAD~1..HEAD", "--", ".gitleaksignore"})

	// Strategy 5: Simple HEAD diff (uncommitted changes)
	strategies = append(strategies, []string{"diff", "HEAD", "--", ".gitleaksignore"})

	var lastErr error
	var lastOutput []byte

	for i, args := range strategies {
		cmd := exec.Command("git", args...)
		output, err := cmd.CombinedOutput()

		if err == nil {
			// Success! Parse the output
			result, parseErr := parseDiffOutput(output)
			if parseErr == nil && len(result) > 0 {
				return result, nil
			}
			// If parsing failed or no results, try next strategy
			if parseErr != nil {
				lastErr = fmt.Errorf("strategy %d (%v) parse failed: %w", i+1, args, parseErr)
			}
			continue
		}

		// Save error for later
		lastOutput = output
		if exitErr, ok := err.(*exec.ExitError); ok {
			lastErr = fmt.Errorf("strategy %d (%v) failed (exit %d): %s", i+1, args, exitErr.ExitCode(), string(output))
		} else {
			lastErr = fmt.Errorf("strategy %d (%v) failed: %w", i+1, args, err)
		}
	}

	// All strategies failed or returned no results
	if lastErr != nil {
		return nil, fmt.Errorf("all git diff strategies failed, last error: %w (output: %s)", lastErr, string(lastOutput))
	}

	// No changes found
	return []DiffChange{}, nil
}

// parseDiffOutput parses the git diff output
func parseDiffOutput(output []byte) ([]DiffChange, error) {

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
