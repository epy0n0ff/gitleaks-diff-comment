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
	// Check if .gitleaksignore file exists in either HEAD or working directory
	// This handles cases where the file is newly added
	checkCmd := exec.Command("git", "ls-files", ".gitleaksignore")
	checkOutput, _ := checkCmd.Output()

	// Also check working directory
	workingDirCmd := exec.Command("ls", "-la", ".gitleaksignore")
	workingDirOutput, _ := workingDirCmd.Output()

	fmt.Printf("DEBUG: git ls-files .gitleaksignore: %q\n", string(checkOutput))
	fmt.Printf("DEBUG: ls -la .gitleaksignore: %q\n", string(workingDirOutput))

	// If file doesn't exist in git and not in working dir, skip
	if len(checkOutput) == 0 && len(workingDirOutput) == 0 {
		fmt.Printf("DEBUG: .gitleaksignore not found in git or working directory\n")
		// Don't return early - the file might exist in the diff even if not in current HEAD
		// This happens when a file is added in a PR
	} else {
		fmt.Printf("DEBUG: .gitleaksignore detected (git tracked: %v, in working dir: %v)\n",
			len(checkOutput) > 0, len(workingDirOutput) > 0)
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

	fmt.Printf("DEBUG: Trying %d strategies for base=%s, head=%s\n", len(strategies), baseBranch, headRef)

	for i, args := range strategies {
		cmd := exec.Command("git", args...)
		output, err := cmd.CombinedOutput()

		fmt.Printf("DEBUG: Strategy %d: git %v\n", i+1, args)
		fmt.Printf("DEBUG: Output length: %d bytes, Error: %v\n", len(output), err)

		if err == nil {
			successCount++
			fmt.Printf("DEBUG: Strategy %d succeeded, parsing output...\n", i+1)
			if len(output) > 0 {
				previewLen := 200
				if len(output) < previewLen {
					previewLen = len(output)
				}
				fmt.Printf("DEBUG: Output preview (first %d chars): %s\n", previewLen, string(output[:previewLen]))
			}

			// Success! Parse the output
			result, parseErr := parseDiffOutput(output)
			if parseErr == nil && len(result) > 0 {
				fmt.Printf("DEBUG: Found %d changes!\n", len(result))
				return result, nil
			}
			// If parsing succeeded but no results, continue trying other strategies
			if parseErr != nil {
				fmt.Printf("DEBUG: Parse error: %v\n", parseErr)
				lastErr = fmt.Errorf("strategy %d (%v) parse failed: %w", i+1, args, parseErr)
			} else {
				fmt.Printf("DEBUG: Parse succeeded but 0 results\n")
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

	// If at least one strategy succeeded (command ran without error)
	// but no changes were found, this could mean:
	// 1. The file truly hasn't changed (legitimate case)
	// 2. All strategies returned empty output (potential issue)

	fmt.Printf("DEBUG: Completed all strategies. Success count: %d, last error: %v\n", successCount, lastErr)

	// If at least one strategy succeeded without error, treat as "no changes"
	if successCount > 0 {
		fmt.Printf("DEBUG: Returning empty result (no changes detected)\n")
		return []DiffChange{}, nil
	}

	// All strategies failed with errors
	if lastErr != nil {
		if len(lastOutput) > 0 {
			return nil, fmt.Errorf("all %d git diff strategies failed, last error: %w (output: %s)", len(strategies), lastErr, string(lastOutput))
		}
		return nil, fmt.Errorf("all %d git diff strategies failed, last error: %w", len(strategies), lastErr)
	}

	// No strategies were attempted (shouldn't happen)
	fmt.Printf("DEBUG: No strategies attempted, returning empty\n")
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
