package comment

import "github.com/epy0n0ff/gitleaks-diff-comment/internal/diff"

// GeneratedComment represents a comment ready to be posted to GitHub
type GeneratedComment struct {
	// Comment body in markdown format
	Body string `json:"body"`

	// File path for the comment (always ".gitleaksignore")
	Path string `json:"path"`

	// Position in the diff (1-indexed, relative to diff output)
	Position int `json:"position"`

	// Commit ID for the comment
	CommitID string `json:"commit_id"`

	// Source diff change (not serialized to JSON)
	SourceChange *diff.DiffChange `json:"-"`
}

// CommentData is the data passed to comment templates
type CommentData struct {
	FilePattern   string
	FileLink      string
	Operation     string
	HasLineNumber bool
	LineNumber    int
	IsPattern     bool
}
