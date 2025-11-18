package github

import (
	"strings"

	"github.com/google/go-github/v57/github"
)

// IsBotComment checks if a comment was created by the gitleaks-diff-comment bot
// It uses two identification methods:
// 1. Primary: Check for invisible HTML marker in comment body
// 2. Fallback: Check if comment author is "github-actions[bot]"
func IsBotComment(comment *github.IssueComment) bool {
	if comment == nil {
		return false
	}

	body := comment.GetBody()

	// Primary: Check for invisible marker
	// All bot comments include: <!-- gitleaks-diff-comment: ... -->
	if strings.Contains(body, "<!-- gitleaks-diff-comment:") {
		return true
	}

	// Fallback: Check comment author
	// Handles old comments that may not have the marker
	if comment.GetUser().GetLogin() == "github-actions[bot]" {
		return true
	}

	return false
}

// FilterBotComments separates bot comments from human comments
// Returns only comments that were created by the gitleaks-diff-comment bot
func FilterBotComments(comments []*github.IssueComment) []*github.IssueComment {
	var botComments []*github.IssueComment

	for _, comment := range comments {
		if IsBotComment(comment) {
			botComments = append(botComments, comment)
		}
	}

	return botComments
}
