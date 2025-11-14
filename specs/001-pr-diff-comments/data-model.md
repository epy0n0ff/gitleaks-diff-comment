# Data Model: Automated PR Diff Comment Explanations (Go Implementation)

**Feature**: 001-pr-diff-comments
**Date**: 2025-11-13
**Purpose**: Define Go struct definitions and data structures for the custom action

## Overview

This is a stateless GitHub Action with no persistent storage. All data exists transiently during action execution. Data structures are defined as Go structs with JSON tags for serialization/deserialization.

## Core Data Structures

### 1. Action Configuration

**Package**: `internal/config`

```go
// Config holds all configuration parsed from action inputs and environment
type Config struct {
    // GitHub API token for authentication
    GitHubToken string `env:"INPUT_GITHUB-TOKEN,required"`

    // Pull request number
    PRNumber int `env:"INPUT_PR-NUMBER,required"`

    // Repository in format "owner/repo"
    Repository string `env:"GITHUB_REPOSITORY,required"`

    // Commit SHA that triggered the action
    CommitSHA string `env:"GITHUB_SHA,required"`

    // Base branch reference (e.g., "main")
    BaseRef string `env:"GITHUB_BASE_REF"`

    // Head branch reference (e.g., "feature/update-ignore")
    HeadRef string `env:"GITHUB_HEAD_REF"`

    // Workspace directory (git repository root)
    Workspace string `env:"GITHUB_WORKSPACE"`

    // Enable debug logging
    Debug bool `env:"INPUT_DEBUG"`
}

// Owner returns the repository owner from Repository field
func (c *Config) Owner() string {
    parts := strings.Split(c.Repository, "/")
    if len(parts) != 2 {
        return ""
    }
    return parts[0]
}

// Repo returns the repository name from Repository field
func (c *Config) Repo() string {
    parts := strings.Split(c.Repository, "/")
    if len(parts) != 2 {
        return ""
    }
    return parts[1]
}
```

### 2. Diff Change

**Package**: `internal/diff`

```go
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
```

### 3. Gitleaks Entry

**Package**: `internal/diff`

```go
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
        return nil, errors.New("empty or comment line")
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
```

### 4. Generated Comment

**Package**: `internal/comment`

```go
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
    FilePattern string
    FileLink    string
    Operation   string
    HasLineNumber bool
    LineNumber  int
}

// NewGeneratedComment creates a new GeneratedComment from a DiffChange
func NewGeneratedComment(change *diff.DiffChange, repo, commitSHA string) (*GeneratedComment, error) {
    entry, err := diff.ParseGitleaksEntry(change.Content)
    if err != nil {
        return nil, fmt.Errorf("failed to parse gitleaks entry: %w", err)
    }

    data := CommentData{
        FilePattern:   entry.FilePattern,
        FileLink:      entry.FileLink(repo, commitSHA),
        Operation:     string(change.Operation),
        HasLineNumber: entry.LineNumber > 0,
        LineNumber:    entry.LineNumber,
    }

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
```

### 5. GitHub API Types

**Package**: `internal/github`

```go
// PostCommentRequest represents a request to post a PR review comment
type PostCommentRequest struct {
    Body     string `json:"body"`
    CommitID string `json:"commit_id"`
    Path     string `json:"path"`
    Position int    `json:"position"`
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
```

### 6. GitHub Client Interface

**Package**: `internal/github`

```go
// Client defines the interface for GitHub API operations
type Client interface {
    // CreateReviewComment posts a line-level review comment on a PR
    CreateReviewComment(ctx context.Context, req *PostCommentRequest) (*PostCommentResponse, error)

    // ListReviewComments fetches all review comments for a PR
    ListReviewComments(ctx context.Context) ([]*ExistingComment, error)

    // CreateIssueComment posts a PR-level comment (fallback)
    CreateIssueComment(ctx context.Context, body string) (*PostCommentResponse, error)

    // CheckRateLimit returns remaining API calls
    CheckRateLimit(ctx context.Context) (int, error)
}

// ClientImpl is the concrete implementation using go-github
type ClientImpl struct {
    client   *github.Client
    owner    string
    repo     string
    prNumber int
}

// NewClient creates a new GitHub API client
func NewClient(token, owner, repo string, prNumber int) (Client, error) {
    if token == "" {
        return nil, errors.New("GitHub token is required")
    }

    ctx := context.Background()
    ts := oauth2.StaticTokenSource(
        &oauth2.Token{AccessToken: token},
    )
    tc := oauth2.NewClient(ctx, ts)

    return &ClientImpl{
        client:   github.NewClient(tc),
        owner:    owner,
        repo:     repo,
        prNumber: prNumber,
    }, nil
}
```

## Data Flow

```
1. Action Execution Start
   ↓
2. Parse Config from environment variables
   → Config struct
   ↓
3. Execute git diff command
   → Raw diff output (string)
   ↓
4. Parse diff line-by-line
   → []DiffChange
   ↓
5. For each DiffChange:
   a. Parse gitleaks entry
      → GitleaksEntry
   b. Generate comment text
      → GeneratedComment
   ↓
6. Fetch existing PR comments (deduplication)
   → []ExistingComment
   ↓
7. For each GeneratedComment:
   a. Check if duplicate exists
   b. If not duplicate:
      i. Post to GitHub API
         → PostCommentResponse
      ii. Record result
         → CommentResult
   ↓
8. Aggregate results
   → ActionOutput
   ↓
9. Output JSON summary to stdout
   → Exit with status code
```

## Validation Rules

### Config Validation

```go
func (c *Config) Validate() error {
    if c.GitHubToken == "" {
        return errors.New("GitHub token is required")
    }
    if c.PRNumber <= 0 {
        return errors.New("PR number must be positive")
    }
    if c.Repository == "" {
        return errors.New("repository is required")
    }
    if !strings.Contains(c.Repository, "/") {
        return errors.New("repository must be in format owner/repo")
    }
    if c.CommitSHA == "" {
        return errors.New("commit SHA is required")
    }
    return nil
}
```

### DiffChange Validation

```go
func (d *DiffChange) Validate() error {
    if d.FilePath != ".gitleaksignore" {
        return fmt.Errorf("unexpected file path: %s", d.FilePath)
    }
    if d.Operation != OperationAddition && d.Operation != OperationDeletion {
        return fmt.Errorf("invalid operation: %s", d.Operation)
    }
    if d.Position <= 0 {
        return errors.New("position must be positive")
    }
    if strings.TrimSpace(d.Content) == "" {
        return errors.New("content cannot be empty")
    }
    return nil
}
```

### GeneratedComment Validation

```go
func (g *GeneratedComment) Validate() error {
    if g.Body == "" {
        return errors.New("comment body cannot be empty")
    }
    if len(g.Body) > 65536 {
        return errors.New("comment body exceeds GitHub limit (65536 chars)")
    }
    if g.Path != ".gitleaksignore" {
        return fmt.Errorf("unexpected path: %s", g.Path)
    }
    if g.Position <= 0 {
        return errors.New("position must be positive")
    }
    if g.CommitID == "" {
        return errors.New("commit ID is required")
    }
    return nil
}
```

## Error Types

```go
// Package: internal/errors

// ErrGitCommand represents a git command execution failure
type ErrGitCommand struct {
    Command string
    Output  string
    Err     error
}

func (e *ErrGitCommand) Error() string {
    return fmt.Sprintf("git command failed: %s: %v\nOutput: %s", e.Command, e.Err, e.Output)
}

// ErrAPIRateLimit represents a GitHub API rate limit error
type ErrAPIRateLimit struct {
    Limit     int
    Remaining int
    ResetAt   time.Time
}

func (e *ErrAPIRateLimit) Error() string {
    return fmt.Sprintf("GitHub API rate limit exceeded (remaining: %d/%d, resets at %s)",
        e.Remaining, e.Limit, e.ResetAt.Format(time.RFC3339))
}

// ErrInvalidDiff represents a parsing error for diff output
type ErrInvalidDiff struct {
    Line   string
    Reason string
}

func (e *ErrInvalidDiff) Error() string {
    return fmt.Sprintf("invalid diff line: %s (reason: %s)", e.Line, e.Reason)
}
```

## Example Data Instances

### Example 1: Adding a file to ignore list

```go
change := &diff.DiffChange{
    FilePath:   ".gitleaksignore",
    Operation:  diff.OperationAddition,
    LineNumber: 6,
    Content:    "database/credentials.json:23",
    Position:   1,
}

entry, _ := diff.ParseGitleaksEntry(change.Content)
// entry.FilePattern = "database/credentials.json"
// entry.LineNumber = 23
// entry.IsPattern = false

comment, _ := comment.NewGeneratedComment(change, "owner/repo", "abc123")
// comment.Body = "🔒 **Gitleaks Exclusion Added**\n\n..."
// comment.Path = ".gitleaksignore"
// comment.Position = 1
// comment.CommitID = "abc123"
```

### Example 2: Removing a pattern from ignore list

```go
change := &diff.DiffChange{
    FilePath:   ".gitleaksignore",
    Operation:  diff.OperationDeletion,
    LineNumber: 0, // Deletions don't have new line numbers
    Content:    "old-config/*.env",
    Position:   2,
}

entry, _ := diff.ParseGitleaksEntry(change.Content)
// entry.FilePattern = "old-config/*.env"
// entry.LineNumber = 0
// entry.IsPattern = true

comment, _ := comment.NewGeneratedComment(change, "owner/repo", "def456")
// comment.Body = "✅ **Gitleaks Exclusion Removed**\n\n..."
// comment.Path = ".gitleaksignore"
// comment.Position = 2
// comment.CommitID = "def456"
```

### Example 3: Action Output

```go
output := &github.ActionOutput{
    Posted:            2,
    SkippedDuplicates: 1,
    Errors:            0,
    Results: []github.CommentResult{
        {
            Status:      "posted",
            CommentID:   987654321,
            CommentURL:  "https://github.com/owner/repo/pull/42#discussion_r987654321",
            BodyPreview: "🔒 **Gitleaks Exclusion Added**...",
        },
        {
            Status:      "posted",
            CommentID:   987654322,
            CommentURL:  "https://github.com/owner/repo/pull/42#discussion_r987654322",
            BodyPreview: "✅ **Gitleaks Exclusion Removed**...",
        },
        {
            Status:      "skipped_duplicate",
            BodyPreview: "🔒 **Gitleaks Exclusion Added**...",
        },
    },
}

// JSON output:
// {
//   "posted": 2,
//   "skipped_duplicates": 1,
//   "errors": 0,
//   "results": [...]
// }
```

## Concurrency Considerations

The Go implementation can leverage goroutines for concurrent operations:

```go
// Concurrent comment posting
func PostCommentsC oncurrently(comments []*GeneratedComment, client github.Client) (*github.ActionOutput, error) {
    var wg sync.WaitGroup
    resultChan := make(chan github.CommentResult, len(comments))

    // Limit concurrency to avoid rate limits
    semaphore := make(chan struct{}, 5) // Max 5 concurrent requests

    for _, comment := range comments {
        wg.Add(1)
        go func(c *GeneratedComment) {
            defer wg.Done()

            semaphore <- struct{}{} // Acquire
            defer func() { <-semaphore }() // Release

            result := postComment(client, c)
            resultChan <- result
        }(comment)
    }

    go func() {
        wg.Wait()
        close(resultChan)
    }()

    // Collect results
    var results []github.CommentResult
    for result := range resultChan {
        results = append(results, result)
    }

    return aggregateResults(results), nil
}
```

## Testing Fixtures

```go
// Package: tests/fixtures

var SampleDiffOutput = `diff --git a/.gitleaksignore b/.gitleaksignore
index abc123..def456 100644
--- a/.gitleaksignore
+++ b/.gitleaksignore
@@ -5,0 +6,2 @@
+database/credentials.json:23
+config/*.env
@@ -10,1 +12,0 @@
-old-secrets.yml`

var SampleConfig = &config.Config{
    GitHubToken: "test-token",
    PRNumber:    42,
    Repository:  "owner/repo",
    CommitSHA:   "abc123def456",
    BaseRef:     "main",
    HeadRef:     "feature/update-ignore",
    Workspace:   "/github/workspace",
}
```
