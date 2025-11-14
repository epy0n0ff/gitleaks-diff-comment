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
	// Try multiple strategies to get the diff
	strategies := []struct {
		name    string
		baseRef string
		headRef string
	}{
		// Strategy 1: origin/base...HEAD (standard PR diff)
		{"origin/base...HEAD", "origin/" + baseBranch, "HEAD"},
		// Strategy 2: base...HEAD (if origin/ doesn't exist)
		{"base...HEAD", baseBranch, "HEAD"},
		// Strategy 3: HEAD^...HEAD (fallback for single-commit PRs)
		{"HEAD^...HEAD", "HEAD^", "HEAD"},
	}

	var lastErr error
	for _, strategy := range strategies {
		if strategy.baseRef == "origin/" || strategy.baseRef == "" {
			continue // skip if base branch is empty
		}

		cmd := exec.Command("git", "diff", strategy.baseRef+"..."+strategy.headRef, "--", ".gitleaksignore")
		output, err := cmd.CombinedOutput()

		if err == nil {
			// Success! Parse the output
			return parseDiffOutput(output)
		}

		// Save error for later
		if exitErr, ok := err.(*exec.ExitError); ok {
			lastErr = fmt.Errorf("strategy '%s' failed (exit %d): %s", strategy.name, exitErr.ExitCode(), string(output))
		} else {
			lastErr = fmt.Errorf("strategy '%s' failed: %w", strategy.name, err)
		}
	}

	// All strategies failed
	if lastErr != nil {
		return nil, fmt.Errorf("all git diff strategies failed, last error: %w", lastErr)
	}

	return nil, fmt.Errorf("unable to determine diff strategy (base: %s, head: %s)", baseBranch, headRef)
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
