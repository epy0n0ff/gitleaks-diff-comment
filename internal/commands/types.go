package commands

import "time"

// Command represents a user-issued command detected in a PR comment
type Command struct {
	// Type is the command type (e.g., "clear")
	Type string

	// IssueNumber is the pull request number
	IssueNumber int

	// CommentID is the GitHub comment ID containing the command
	CommentID int64

	// RequestedBy is the GitHub login of the user who issued the command
	RequestedBy string

	// RequestedAt is the timestamp when the command was detected
	RequestedAt time.Time

	// Raw is the original comment body text
	Raw string
}

// Authorization represents the permission check result for a command requester
type Authorization struct {
	// Username is the GitHub login being checked
	Username string

	// PermissionLevel is the GitHub permission level (none/read/write/admin/maintain)
	PermissionLevel string

	// IsAuthorized indicates whether the user can execute the command
	IsAuthorized bool

	// CheckedAt is when the permission was verified
	CheckedAt time.Time

	// Reason provides explanation if not authorized
	Reason string
}
