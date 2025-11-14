package diff

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// DiffChange represents a single line change in .gitleaksignore
type DiffChange struct {
	// File path (always ".gitleaksignore" for this feature)
	FilePath string `json:"file_path"`

	// Operation type: "addition" or "deletion"
	Operation OperationType `json:"operation"`

	// Line number in the new version (0 if deletion)
	LineNumber int `json:"line_number"`

	// Raw line content (the gitleaks pattern/file path)
	Content string `json:"content"`

	// Position in the diff for PR comment placement (1-indexed)
	Position int `json:"position"`
}

// OperationType represents the type of change
type OperationType string

const (
	OperationAddition OperationType = "addition"
	OperationDeletion OperationType = "deletion"
)

// IsAddition returns true if this is an addition
func (d *DiffChange) IsAddition() bool {
	return d.Operation == OperationAddition
}

// IsDeletion returns true if this is a deletion
func (d *DiffChange) IsDeletion() bool {
	return d.Operation == OperationDeletion
}

// GitleaksEntry represents a parsed entry from .gitleaksignore
type GitleaksEntry struct {
	// File path or pattern being ignored
	FilePattern string `json:"file_pattern"`

	// Optional line number in the file (0 if not specified)
	LineNumber int `json:"line_number,omitempty"`

	// Whether the pattern contains wildcards
	IsPattern bool `json:"is_pattern"`

	// Original line from .gitleaksignore
	OriginalLine string `json:"original_line"`
}

// ParseGitleaksEntry parses a line from .gitleaksignore into a GitleaksEntry
func ParseGitleaksEntry(line string) (*GitleaksEntry, error) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return nil, fmt.Errorf("empty or comment line")
	}

	entry := &GitleaksEntry{
		OriginalLine: line,
		IsPattern:    strings.ContainsAny(line, "*?[]"),
	}

	// Parse gitleaks format: file:rule:line or file:line
	// Examples:
	//   DUMMY.txt:base64-encoded-secrets:1 -> file=DUMMY.txt, line=1
	//   config/secrets.yml:23 -> file=config/secrets.yml, line=23
	//   *.env -> file=*.env, line=0
	parts := strings.Split(line, ":")

	if len(parts) >= 2 {
		// Check if last part is a line number
		lastPart := parts[len(parts)-1]
		if lineNum, err := strconv.Atoi(lastPart); err == nil {
			// Last part is a line number
			// Everything before the last colon is the file path
			filePath := strings.Join(parts[:len(parts)-1], ":")

			// If there are 3+ parts, extract just the file name (first part)
			if len(parts) >= 3 {
				// Format: file:rule:line -> use first part only
				entry.FilePattern = parts[0]
			} else {
				// Format: file:line -> use everything before last colon
				entry.FilePattern = filePath
			}
			entry.LineNumber = lineNum
			return entry, nil
		}
	}

	// No line number found, use the whole line as pattern
	entry.FilePattern = line
	return entry, nil
}

// FileLink generates a GitHub file link for this entry
// ghHost should be the GitHub Enterprise Server hostname (e.g., "github.company.com")
// or empty string for GitHub.com
func (e *GitleaksEntry) FileLink(repo, commitSHA, ghHost string) string {
	// Determine base URL based on ghHost
	baseURL := "https://github.com"
	if ghHost != "" {
		baseURL = "https://" + ghHost
	}

	// For patterns with wildcards, link to parent directory
	path := e.FilePattern
	if e.IsPattern {
		path = filepath.Dir(e.FilePattern)
		if path == "." {
			path = ""
		}
		return fmt.Sprintf("%s/%s/blob/%s/%s", baseURL, repo, commitSHA, path)
	}

	// For specific files with line numbers, create a permalink to that line
	if e.HasLineNumber() {
		return fmt.Sprintf("%s/%s/blob/%s/%s#L%d", baseURL, repo, commitSHA, path, e.LineNumber)
	}

	// Default: link to the file
	return fmt.Sprintf("%s/%s/blob/%s/%s", baseURL, repo, commitSHA, path)
}

// HasLineNumber returns true if this entry has a line number
func (e *GitleaksEntry) HasLineNumber() bool {
	return e.LineNumber > 0
}
