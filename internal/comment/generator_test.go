package comment

import (
	"strings"
	"testing"

	"github.com/epy0n0ff/gitleaks-diff-comment/internal/diff"
)

func TestNewGeneratedComment_WithLineNumber(t *testing.T) {
	change := &diff.DiffChange{
		FilePath:   ".gitleaksignore",
		Operation:  diff.OperationAddition,
		LineNumber: 10,
		Content:    "config/secrets.yml:42",
		Position:   5,
	}

	comment, err := NewGeneratedComment(change, "owner/repo", "abc123", "")
	if err != nil {
		t.Fatalf("NewGeneratedComment() unexpected error: %v", err)
	}

	// Verify body contains line number mention
	if !strings.Contains(comment.Body, "(line 42)") {
		t.Errorf("Comment body should mention line number: %s", comment.Body)
	}

	// Verify body contains file link
	if !strings.Contains(comment.Body, "https://github.com/owner/repo/blob/abc123/config/secrets.yml") {
		t.Errorf("Comment body should contain file link: %s", comment.Body)
	}

	// Verify security note for specific file
	if !strings.Contains(comment.Body, "This file will no longer be scanned") {
		t.Errorf("Comment body should contain specific file security note: %s", comment.Body)
	}
}

func TestNewGeneratedComment_WildcardPattern(t *testing.T) {
	change := &diff.DiffChange{
		FilePath:   ".gitleaksignore",
		Operation:  diff.OperationAddition,
		LineNumber: 12,
		Content:    "config/*.env",
		Position:   7,
	}

	comment, err := NewGeneratedComment(change, "owner/repo", "abc123", "")
	if err != nil {
		t.Fatalf("NewGeneratedComment() unexpected error: %v", err)
	}

	// Verify body does NOT contain line number (wildcard patterns don't have line numbers)
	if strings.Contains(comment.Body, "(line") {
		t.Errorf("Comment body should not mention line number for wildcard: %s", comment.Body)
	}

	// Verify body contains directory link (parent of pattern)
	if !strings.Contains(comment.Body, "https://github.com/owner/repo/blob/abc123/config") {
		t.Errorf("Comment body should contain directory link: %s", comment.Body)
	}

	// Verify wildcard pattern security note
	if !strings.Contains(comment.Body, "wildcard pattern will match multiple files") {
		t.Errorf("Comment body should contain wildcard pattern note: %s", comment.Body)
	}
}

func TestNewGeneratedComment_Deletion(t *testing.T) {
	change := &diff.DiffChange{
		FilePath:  ".gitleaksignore",
		Operation: diff.OperationDeletion,
		Content:   "old-secrets.yml",
		Position:  10,
	}

	comment, err := NewGeneratedComment(change, "owner/repo", "abc123", "")
	if err != nil {
		t.Fatalf("NewGeneratedComment() unexpected error: %v", err)
	}

	// Verify deletion indicator
	if !strings.Contains(comment.Body, "✅") {
		t.Errorf("Comment body should contain deletion emoji: %s", comment.Body)
	}

	// Verify "will now be scanned" message
	if !strings.Contains(comment.Body, "will now be scanned by gitleaks") {
		t.Errorf("Comment body should indicate file will be scanned: %s", comment.Body)
	}
}

func TestRenderTemplate_Addition(t *testing.T) {
	data := CommentData{
		FilePattern:   "config/secrets.yml",
		FileLink:      "https://github.com/owner/repo/blob/abc123/config/secrets.yml",
		Operation:     "addition",
		HasLineNumber: true,
		LineNumber:    42,
		IsPattern:     false,
	}

	body, err := renderTemplate(diff.OperationAddition, data)
	if err != nil {
		t.Fatalf("renderTemplate() unexpected error: %v", err)
	}

	// Verify addition emoji
	if !strings.Contains(body, "🔒") {
		t.Errorf("Template should contain addition emoji: %s", body)
	}

	// Verify file pattern
	if !strings.Contains(body, "config/secrets.yml") {
		t.Errorf("Template should contain file pattern: %s", body)
	}

	// Verify line number
	if !strings.Contains(body, "(line 42)") {
		t.Errorf("Template should contain line number: %s", body)
	}
}

func TestRenderTemplate_Deletion(t *testing.T) {
	data := CommentData{
		FilePattern:   "*.env",
		FileLink:      "https://github.com/owner/repo/blob/abc123/",
		Operation:     "deletion",
		HasLineNumber: false,
		LineNumber:    0,
		IsPattern:     true,
	}

	body, err := renderTemplate(diff.OperationDeletion, data)
	if err != nil {
		t.Fatalf("renderTemplate() unexpected error: %v", err)
	}

	// Verify deletion emoji
	if !strings.Contains(body, "✅") {
		t.Errorf("Template should contain deletion emoji: %s", body)
	}

	// Verify pattern indication
	if !strings.Contains(body, "All files matching this pattern") {
		t.Errorf("Template should indicate pattern matching: %s", body)
	}

	// Verify no line number mention
	if strings.Contains(body, "(line") {
		t.Errorf("Template should not mention line number for pattern: %s", body)
	}
}

func TestNewGeneratedComment_EnterpriseServer(t *testing.T) {
	change := &diff.DiffChange{
		FilePath:   ".gitleaksignore",
		Operation:  diff.OperationAddition,
		LineNumber: 15,
		Content:    "database/credentials.json:23",
		Position:   8,
	}

	// Test with GitHub Enterprise Server hostname
	comment, err := NewGeneratedComment(change, "owner/repo", "abc123", "github.company.com")
	if err != nil {
		t.Fatalf("NewGeneratedComment() unexpected error: %v", err)
	}

	// Verify body contains enterprise server link
	expectedLink := "https://github.company.com/owner/repo/blob/abc123/database/credentials.json"
	if !strings.Contains(comment.Body, expectedLink) {
		t.Errorf("Comment body should contain enterprise server link %s, got: %s", expectedLink, comment.Body)
	}

	// Verify NOT contains github.com
	if strings.Contains(comment.Body, "https://github.com/") {
		t.Errorf("Comment body should NOT contain github.com link: %s", comment.Body)
	}
}

func TestNewGeneratedComment_EnterpriseServerWithPort(t *testing.T) {
	change := &diff.DiffChange{
		FilePath:   ".gitleaksignore",
		Operation:  diff.OperationAddition,
		LineNumber: 20,
		Content:    "config/*.env",
		Position:   10,
	}

	// Test with GitHub Enterprise Server hostname with port
	comment, err := NewGeneratedComment(change, "owner/repo", "abc123", "github.company.com:8443")
	if err != nil {
		t.Fatalf("NewGeneratedComment() unexpected error: %v", err)
	}

	// Verify body contains enterprise server link with port
	expectedLink := "https://github.company.com:8443/owner/repo/blob/abc123/config"
	if !strings.Contains(comment.Body, expectedLink) {
		t.Errorf("Comment body should contain enterprise server link with port %s, got: %s", expectedLink, comment.Body)
	}
}

func TestGetBodyPreview(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		maxLen   int
		expected string
	}{
		{
			name:     "short body",
			body:     "Short comment",
			expected: "Short comment",
		},
		{
			name:     "long body with newlines",
			body:     strings.Repeat("a", 100) + "\n" + strings.Repeat("b", 50),
			expected: strings.Repeat("a", 80) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comment := &GeneratedComment{Body: tt.body}
			preview := comment.GetBodyPreview()

			if len(preview) > 83 { // 80 chars + "..."
				t.Errorf("Preview too long: %d chars", len(preview))
			}

			if tt.body != "" && len(tt.body) <= 80 {
				if preview != tt.body {
					t.Errorf("Short body preview = %v, want %v", preview, tt.body)
				}
			}
		})
	}
}
