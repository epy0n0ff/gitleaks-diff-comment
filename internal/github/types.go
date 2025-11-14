package github

import "time"

// PostCommentRequest represents a request to post a PR review comment
type PostCommentRequest struct {
	Body     string `json:"body"`
	CommitID string `json:"commit_id"`
	Path     string `json:"path"`
	Line     int    `json:"line"`      // Line number in the file
	Side     string `json:"side"`      // "LEFT" or "RIGHT"
	Position int    `json:"position"`  // Deprecated, kept for backwards compatibility
}

// PostCommentResponse represents the response from posting a comment
type PostCommentResponse struct {
	ID        int64     `json:"id"`
	HTMLURL   string    `json:"html_url"`
	CreatedAt time.Time `json:"created_at"`
}

// ExistingComment represents a comment fetched from GitHub
type ExistingComment struct {
	ID       int64  `json:"id"`
	Body     string `json:"body"`
	Path     string `json:"path"`
	Position int    `json:"position"`
}

// CommentResult represents the result of posting a comment
type CommentResult struct {
	// Status: "posted", "skipped_duplicate", "error"
	Status string `json:"status"`

	// Comment ID if successfully posted
	CommentID int64 `json:"comment_id,omitempty"`

	// Comment URL if successfully posted
	CommentURL string `json:"comment_url,omitempty"`

	// Error message if status is "error"
	Error string `json:"error,omitempty"`

	// Body preview for logging
	BodyPreview string `json:"body_preview,omitempty"`
}

// ActionOutput represents the final output of the action
type ActionOutput struct {
	Posted            int             `json:"posted"`
	SkippedDuplicates int             `json:"skipped_duplicates"`
	Errors            int             `json:"errors"`
	Results           []CommentResult `json:"results"`
}
