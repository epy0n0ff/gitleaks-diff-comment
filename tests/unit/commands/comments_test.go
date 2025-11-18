package commands_test

import (
	"testing"

	"github.com/epy0n0ff/gitleaks-diff-comment/internal/github"
	gh "github.com/google/go-github/v57/github"
)

func TestIsBotComment(t *testing.T) {
	tests := []struct {
		name     string
		comment  *gh.IssueComment
		expected bool
	}{
		{
			name: "comment with invisible marker",
			comment: &gh.IssueComment{
				Body: gh.String("<!-- gitleaks-diff-comment: .gitleaksignore:64:RIGHT -->\n🔒 **Gitleaks Exclusion Added**"),
				User: &gh.User{Login: gh.String("github-actions[bot]")},
			},
			expected: true,
		},
		{
			name: "comment with marker but different author",
			comment: &gh.IssueComment{
				Body: gh.String("<!-- gitleaks-diff-comment: .gitleaksignore:65:RIGHT -->\nSome content"),
				User: &gh.User{Login: gh.String("human-user")},
			},
			expected: true, // Marker takes precedence
		},
		{
			name: "bot author without marker",
			comment: &gh.IssueComment{
				Body: gh.String("This is a bot comment without marker"),
				User: &gh.User{Login: gh.String("github-actions[bot]")},
			},
			expected: true, // Fallback to author check
		},
		{
			name: "human comment",
			comment: &gh.IssueComment{
				Body: gh.String("LGTM! This looks good to merge."),
				User: &gh.User{Login: gh.String("octocat")},
			},
			expected: false,
		},
		{
			name: "human comment mentioning bot",
			comment: &gh.IssueComment{
				Body: gh.String("Thanks @github-actions for the report!"),
				User: &gh.User{Login: gh.String("octocat")},
			},
			expected: false,
		},
		{
			name: "comment with similar marker",
			comment: &gh.IssueComment{
				Body: gh.String("<!-- some-other-bot: data -->\nContent"),
				User: &gh.User{Login: gh.String("other-bot[bot]")},
			},
			expected: false,
		},
		{
			name: "empty comment body",
			comment: &gh.IssueComment{
				Body: gh.String(""),
				User: &gh.User{Login: gh.String("octocat")},
			},
			expected: false,
		},
		{
			name:     "nil comment",
			comment:  nil,
			expected: false,
		},
		{
			name: "marker in code block",
			comment: &gh.IssueComment{
				Body: gh.String("Check this:\n```\n<!-- gitleaks-diff-comment: test -->\n```"),
				User: &gh.User{Login: gh.String("octocat")},
			},
			expected: true, // Still detects marker even in code block
		},
		{
			name: "multiple markers",
			comment: &gh.IssueComment{
				Body: gh.String("<!-- gitleaks-diff-comment: line1 -->\nContent\n<!-- gitleaks-diff-comment: line2 -->"),
				User: &gh.User{Login: gh.String("github-actions[bot]")},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := github.IsBotComment(tt.comment)
			if result != tt.expected {
				t.Errorf("IsBotComment() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFilterBotComments(t *testing.T) {
	comments := []*gh.IssueComment{
		{
			ID:   gh.Int64(1),
			Body: gh.String("<!-- gitleaks-diff-comment: .gitleaksignore:10:RIGHT -->\nBot comment 1"),
			User: &gh.User{Login: gh.String("github-actions[bot]")},
		},
		{
			ID:   gh.Int64(2),
			Body: gh.String("This is a human comment"),
			User: &gh.User{Login: gh.String("octocat")},
		},
		{
			ID:   gh.Int64(3),
			Body: gh.String("Another human comment with @github-actions mention"),
			User: &gh.User{Login: gh.String("developer")},
		},
		{
			ID:   gh.Int64(4),
			Body: gh.String("<!-- gitleaks-diff-comment: .gitleaksignore:20:LEFT -->\nBot comment 2"),
			User: &gh.User{Login: gh.String("github-actions[bot]")},
		},
		{
			ID:   gh.Int64(5),
			Body: gh.String("Bot comment without marker (old format)"),
			User: &gh.User{Login: gh.String("github-actions[bot]")},
		},
		{
			ID:   gh.Int64(6),
			Body: gh.String("LGTM"),
			User: &gh.User{Login: gh.String("reviewer")},
		},
	}

	botComments := github.FilterBotComments(comments)

	// Should find 3 bot comments (IDs 1, 4, 5)
	expectedCount := 3
	if len(botComments) != expectedCount {
		t.Errorf("FilterBotComments() returned %d comments, want %d", len(botComments), expectedCount)
	}

	// Verify correct comments were filtered
	expectedIDs := map[int64]bool{1: true, 4: true, 5: true}
	for _, comment := range botComments {
		id := comment.GetID()
		if !expectedIDs[id] {
			t.Errorf("FilterBotComments() included unexpected comment ID %d", id)
		}
	}

	// Verify human comments were excluded
	excludedIDs := map[int64]bool{2: true, 3: true, 6: true}
	for _, comment := range botComments {
		id := comment.GetID()
		if excludedIDs[id] {
			t.Errorf("FilterBotComments() should not include human comment ID %d", id)
		}
	}
}

func TestFilterBotComments_EmptyList(t *testing.T) {
	comments := []*gh.IssueComment{}
	botComments := github.FilterBotComments(comments)

	if len(botComments) != 0 {
		t.Errorf("FilterBotComments([]) should return empty list, got %d comments", len(botComments))
	}
}

func TestFilterBotComments_AllBot(t *testing.T) {
	comments := []*gh.IssueComment{
		{
			ID:   gh.Int64(1),
			Body: gh.String("<!-- gitleaks-diff-comment: test1 -->\nBot 1"),
			User: &gh.User{Login: gh.String("github-actions[bot]")},
		},
		{
			ID:   gh.Int64(2),
			Body: gh.String("<!-- gitleaks-diff-comment: test2 -->\nBot 2"),
			User: &gh.User{Login: gh.String("github-actions[bot]")},
		},
	}

	botComments := github.FilterBotComments(comments)

	if len(botComments) != 2 {
		t.Errorf("FilterBotComments() should return all 2 comments, got %d", len(botComments))
	}
}

func TestFilterBotComments_AllHuman(t *testing.T) {
	comments := []*gh.IssueComment{
		{
			ID:   gh.Int64(1),
			Body: gh.String("Human comment 1"),
			User: &gh.User{Login: gh.String("user1")},
		},
		{
			ID:   gh.Int64(2),
			Body: gh.String("Human comment 2"),
			User: &gh.User{Login: gh.String("user2")},
		},
	}

	botComments := github.FilterBotComments(comments)

	if len(botComments) != 0 {
		t.Errorf("FilterBotComments() should return empty list for all human comments, got %d", len(botComments))
	}
}
