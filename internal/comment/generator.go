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
func NewGeneratedComment(change *diff.DiffChange, repo, commitSHA string) (*GeneratedComment, error) {
	// Parse the gitleaks entry
	entry, err := diff.ParseGitleaksEntry(change.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse gitleaks entry: %w", err)
	}

	// Prepare template data
	data := CommentData{
		FilePattern:   entry.FilePattern,
		FileLink:      entry.FileLink(repo, commitSHA),
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

	return &GeneratedComment{
		Body:         body,
		Path:         ".gitleaksignore",
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
