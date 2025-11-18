package comment

import (
	_ "embed"
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/epy0n0ff/gitleaks-diff-comment/internal/diff"
)

//go:embed templates/addition.md
var additionTemplate string

//go:embed templates/deletion.md
var deletionTemplate string

// NewGeneratedComment creates a new GeneratedComment from a DiffChange
// ghHost should be the GitHub Enterprise Server hostname (e.g., "github.company.com")
// or empty string for GitHub.com
func NewGeneratedComment(change *diff.DiffChange, repo, commitSHA, ghHost string) (*GeneratedComment, error) {
	// Parse the gitleaks entry
	entry, err := diff.ParseGitleaksEntry(change.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse gitleaks entry: %w", err)
	}

	// Prepare template data
	data := CommentData{
		FilePattern:   entry.FilePattern,
		FileLink:      entry.FileLink(repo, commitSHA, ghHost),
		Operation:     string(change.Operation),
		HasLineNumber: entry.HasLineNumber(),
		LineNumber:    entry.LineNumber,
		IsPattern:     entry.IsPattern,
	}

	// Render template
	body, err := renderTemplate(change.Operation, data)
	if err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	// Determine side based on operation
	side := "RIGHT" // Default for additions
	if change.Operation == diff.OperationDeletion {
		side = "LEFT"
	}

	// Use LineNumber from the change (this is the line number in the file)
	line := change.LineNumber
	if line <= 0 {
		line = 1 // Fallback to line 1 if not set
	}

	// Add invisible marker for comment identification (for override mode)
	// Format: <!-- gitleaks-diff-comment: {path}:{content}:{side} -->
	// Use content instead of line number to handle line shifts when file changes
	contentID := change.Content // Use the actual gitleaks pattern as identifier
	marker := fmt.Sprintf("<!-- gitleaks-diff-comment: %s:%s:%s -->", ".gitleaksignore", contentID, side)
	bodyWithMarker := marker + "\n" + body

	return &GeneratedComment{
		Body:         bodyWithMarker,
		Path:         ".gitleaksignore",
		Line:         line,
		Side:         side,
		Position:     change.Position,
		CommitID:     commitSHA,
		SourceChange: change,
	}, nil
}

// renderTemplate renders the appropriate template based on operation type
func renderTemplate(operation diff.OperationType, data CommentData) (string, error) {
	var tmplStr string
	var tmplName string

	switch operation {
	case diff.OperationAddition:
		tmplStr = additionTemplate
		tmplName = "addition"
	case diff.OperationDeletion:
		tmplStr = deletionTemplate
		tmplName = "deletion"
	default:
		return "", fmt.Errorf("unknown operation type: %s", operation)
	}

	// Parse template
	tmpl, err := template.New(tmplName).Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	// Clean up extra whitespace
	result := strings.TrimSpace(buf.String())
	return result, nil
}

// GetBodyPreview returns a short preview of the comment body for logging
func (g *GeneratedComment) GetBodyPreview() string {
	const maxLen = 80
	body := strings.ReplaceAll(g.Body, "\n", " ")
	if len(body) > maxLen {
		return body[:maxLen] + "..."
	}
	return body
}
