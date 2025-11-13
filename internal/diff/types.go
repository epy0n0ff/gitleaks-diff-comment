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

	// Check for line number suffix (path:42)
	if parts := strings.Split(line, ":"); len(parts) == 2 {
		if lineNum, err := strconv.Atoi(parts[1]); err == nil {
			entry.FilePattern = parts[0]
			entry.LineNumber = lineNum
			return entry, nil
		}
	}

	entry.FilePattern = line
	return entry, nil
}

// FileLink generates a GitHub file link for this entry
func (e *GitleaksEntry) FileLink(repo, commitSHA string) string {
	// For patterns with wildcards, link to parent directory
	path := e.FilePattern
	if e.IsPattern {
		path = filepath.Dir(e.FilePattern)
		if path == "." {
			path = ""
		}
	}

	return fmt.Sprintf("https://github.com/%s/blob/%s/%s", repo, commitSHA, path)
}

// HasLineNumber returns true if this entry has a line number
func (e *GitleaksEntry) HasLineNumber() bool {
	return e.LineNumber > 0
}
