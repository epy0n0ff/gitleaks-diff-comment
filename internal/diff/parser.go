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
	// First, check if .gitleaksignore file exists
	checkCmd := exec.Command("git", "ls-files", ".gitleaksignore")
	checkOutput, _ := checkCmd.Output()
	if len(checkOutput) == 0 {
		// File doesn't exist in the repository
		return []DiffChange{}, nil
	}

	// Build list of diff strategies to try
	var strategies [][]string

	// Try to use merge-base to find common ancestor
	if baseBranch != "" {
		// Try finding merge-base with origin/base
		mergeBaseCmd := exec.Command("git", "merge-base", "origin/"+baseBranch, "HEAD")
		if mergeBase, err := mergeBaseCmd.Output(); err == nil && len(mergeBase) > 0 {
			baseCommit := strings.TrimSpace(string(mergeBase))
			strategies = append(strategies, []string{"diff", baseCommit + "..HEAD", "--", ".gitleaksignore"})
		}

		// Standard PR strategies
		strategies = append(strategies, []string{"diff", "origin/" + baseBranch + "..HEAD", "--", ".gitleaksignore"})
		strategies = append(strategies, []string{"diff", "origin/" + baseBranch + "...HEAD", "--", ".gitleaksignore"})
	}

	// Try with FETCH_HEAD (GitHub Actions sets this)
	strategies = append(strategies, []string{"diff", "FETCH_HEAD..HEAD", "--", ".gitleaksignore"})

	// Try refs/remotes/origin/main pattern
	if baseBranch != "" {
		strategies = append(strategies, []string{"diff", "refs/remotes/origin/" + baseBranch + "..HEAD", "--", ".gitleaksignore"})
	}

	// Single commit strategies
	strategies = append(strategies, []string{"diff", "HEAD~1..HEAD", "--", ".gitleaksignore"})
	strategies = append(strategies, []string{"diff", "HEAD~1", "HEAD", "--", ".gitleaksignore"})

	// Use git log -p as a fallback (shows full history with diffs)
	strategies = append(strategies, []string{"log", "-p", "-1", "--", ".gitleaksignore"})

	var lastErr error
	var lastOutput []byte
	var successCount int

	for i, args := range strategies {
		cmd := exec.Command("git", args...)
		output, err := cmd.CombinedOutput()

		if err == nil {
			successCount++
			// Success! Parse the output
			result, parseErr := parseDiffOutput(output)
			if parseErr == nil && len(result) > 0 {
				return result, nil
			}
			// If parsing succeeded but no results, continue trying other strategies
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

	// If at least one strategy succeeded but found no changes, that's OK
	if successCount > 0 {
		return []DiffChange{}, nil
	}

	// All strategies failed
	if lastErr != nil {
		return nil, fmt.Errorf("all %d git diff strategies failed, last error: %w", len(strategies), lastErr)
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
