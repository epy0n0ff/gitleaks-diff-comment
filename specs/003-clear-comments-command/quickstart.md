# Quick Start: Clear Comments Command

**Feature**: `003-clear-comments-command`
**Date**: 2025-11-17
**Audience**: Developers implementing the feature and users testing it

## For Users

### Basic Usage

1. **Trigger the command** by commenting on a pull request:
   ```
   @github-actions /clear
   ```

2. **View the results** in the GitHub Actions workflow logs:
   - Go to the PR's "Checks" tab
   - Click on "Clear Comments Command" workflow
   - View the log output showing how many comments were cleared

### Command Variations

All these commands work (case-insensitive):
```
@github-actions /clear
@github-actions /CLEAR
@github-actions /Clear
@github-actions /clear please remove old comments
```

### Requirements

- You must be one of:
  - The PR author
  - A repository collaborator with write access
  - A repository maintainer/admin

- The PR must have at least one bot comment to clear

### What Gets Deleted

✅ **Deleted**:
- Comments created by `github-actions[bot]`
- Comments containing the invisible marker `<!-- gitleaks-diff-comment: ... -->`

❌ **Preserved**:
- All human-written comments
- Comments from other bots
- Comments without the gitleaks-diff-comment marker

### Expected Results

**Success Case**:
```
✓ Successfully cleared 5 comments in 2.34s
```

**No Comments Case**:
```
✓ Completed successfully (0 comments cleared) in 0.89s
```

**Permission Denied**:
```
✗ Permission denied: User 'username' does not have required permissions
Required: write, admin, or maintain access to repository
```

---

## For Developers

### Implementation Overview

```
1. Webhook (issue_comment.created)
   ↓
2. Workflow (.github/workflows/clear-command.yml)
   ↓
3. Command Detection (internal/commands/detector.go)
   ↓
4. Permission Check (internal/github/client.go)
   ↓
5. Comment Fetching + Filtering (internal/github/comments.go)
   ↓
6. Comment Deletion with Retry (internal/commands/clear.go)
   ↓
7. Metrics Logging + Workflow Output
```

### Key Components

#### 1. Workflow Definition

**File**: `.github/workflows/clear-command.yml`

```yaml
name: Clear Comments Command

on:
  issue_comment:
    types: [created]

jobs:
  clear:
    if: |
      github.event.issue.pull_request &&
      contains(github.event.comment.body, '@github-actions') &&
      contains(github.event.comment.body, '/clear')
    runs-on: ubuntu-latest
    permissions:
      pull-requests: write
      issues: write
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.25'

      - name: Execute clear command
        run: |
          go run ./cmd/gitleaks-diff-comment \
            --command=clear \
            --pr-number=${{ github.event.issue.number }} \
            --comment-id=${{ github.event.comment.id }} \
            --requester=${{ github.event.comment.user.login }}
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GITHUB_REPOSITORY: ${{ github.repository }}
```

#### 2. Command Detection

**File**: `internal/commands/detector.go`

```go
package commands

import (
    "regexp"
    "strings"
)

var commandPattern = regexp.MustCompile(`(?i)@github-actions\s+/(clear)`)

type Command struct {
    Type         string
    IssueNumber  int
    CommentID    int64
    RequestedBy  string
    Raw          string
}

func DetectCommand(commentBody string) (string, bool) {
    matches := commandPattern.FindStringSubmatch(commentBody)
    if len(matches) < 2 {
        return "", false
    }
    return strings.ToLower(matches[1]), true
}
```

#### 3. Permission Verification

**File**: `internal/github/client.go` (extend existing)

```go
func (c *Client) CheckUserPermission(ctx context.Context, username string) (bool, error) {
    permission, _, err := c.client.Repositories.GetPermissionLevel(
        ctx,
        c.owner,
        c.repo,
        username,
    )
    if err != nil {
        return false, fmt.Errorf("permission check failed: %w", err)
    }

    allowedLevels := map[string]bool{
        "write":    true,
        "admin":    true,
        "maintain": true,
    }

    return allowedLevels[permission.GetPermission()], nil
}
```

#### 4. Comment Identification

**File**: `internal/github/comments.go` (new)

```go
func IsBotComment(comment *github.IssueComment) bool {
    body := comment.GetBody()

    // Primary: Check for invisible marker
    if strings.Contains(body, "<!-- gitleaks-diff-comment:") {
        return true
    }

    // Fallback: Check comment author
    if comment.GetUser().GetLogin() == "github-actions[bot]" {
        return true
    }

    return false
}

func FilterBotComments(comments []*github.IssueComment) []*github.IssueComment {
    var botComments []*github.IssueComment
    for _, comment := range comments {
        if IsBotComment(comment) {
            botComments = append(botComments, comment)
        }
    }
    return botComments
}
```

#### 5. Comment Deletion with Retry

**File**: `internal/commands/clear.go` (new)

```go
func (c *ClearCommand) Execute(ctx context.Context) error {
    // 1. Check permissions
    authorized, err := c.client.CheckUserPermission(ctx, c.RequestedBy)
    if err != nil {
        return fmt.Errorf("permission check failed: %w", err)
    }
    if !authorized {
        return ErrUnauthorized
    }

    // 2. Fetch and filter comments
    comments, err := c.client.ListPRComments(ctx, c.PRNumber)
    if err != nil {
        return fmt.Errorf("failed to fetch comments: %w", err)
    }

    botComments := FilterBotComments(comments)
    log.Printf("Found %d bot comments to delete", len(botComments))

    // 3. Delete each comment with retry
    deleted := 0
    errors := 0
    for _, comment := range botComments {
        err := c.deleteWithRetry(ctx, comment.GetID())
        if err != nil {
            log.Printf("Error deleting comment %d: %v", comment.GetID(), err)
            errors++
        } else {
            deleted++
        }
    }

    // 4. Log metrics
    c.logMetrics(deleted, errors)

    return nil
}

func (c *ClearCommand) deleteWithRetry(ctx context.Context, commentID int64) error {
    return retryWithBackoff(func() error {
        return c.client.DeleteComment(ctx, commentID)
    }, 3)
}
```

#### 6. Retry Logic

**File**: `internal/github/retry.go` (new)

```go
func retryWithBackoff(operation func() error, maxRetries int) error {
    baseDelay := 2 * time.Second

    for attempt := 0; attempt < maxRetries; attempt++ {
        err := operation()
        if err == nil {
            return nil
        }

        if !isRateLimitError(err) {
            return err // Don't retry non-rate-limit errors
        }

        if attempt == maxRetries-1 {
            return fmt.Errorf("max retries exceeded: %w", err)
        }

        delay := baseDelay * (1 << uint(attempt))
        jitter := time.Duration(rand.Int63n(int64(delay / 2)))
        time.Sleep(delay + jitter)
    }

    return nil
}

func isRateLimitError(err error) bool {
    var rateLimitErr *github.RateLimitError
    return errors.As(err, &rateLimitErr)
}
```

### Data Flow Example

**Scenario**: User posts `@github-actions /clear` on PR #123 with 3 bot comments

```
Step 1: Webhook Event
  ↓
  {
    "action": "created",
    "issue": { "number": 123, "pull_request": {...} },
    "comment": {
      "id": 987654321,
      "body": "@github-actions /clear",
      "user": { "login": "octocat" }
    }
  }

Step 2: Workflow Triggered
  ↓
  if: github.event.issue.pull_request &&
      contains('@github-actions') &&
      contains('/clear')
  ✓ Passes, job starts

Step 3: Command Detection
  ↓
  DetectCommand("@github-actions /clear")
  → Returns ("clear", true)

Step 4: Permission Check
  ↓
  CheckUserPermission(ctx, "octocat")
  → GET /repos/owner/repo/collaborators/octocat/permission
  → Response: {"permission": "write"}
  → Returns (true, nil)

Step 5: Fetch Comments
  ↓
  ListPRComments(ctx, 123)
  → GET /repos/owner/repo/issues/123/comments
  → Returns [comment1, comment2, comment3, comment4, comment5]

Step 6: Filter Bot Comments
  ↓
  FilterBotComments([...])
  → comment1: Has marker → Include
  → comment2: No marker, human author → Exclude
  → comment3: Has marker → Include
  → comment4: No marker, human author → Exclude
  → comment5: No marker, bot author → Include
  → Returns [comment1, comment3, comment5]

Step 7: Delete Comments
  ↓
  For comment1 (ID: 111111111):
    DELETE /repos/owner/repo/issues/comments/111111111
    → 204 No Content ✓

  For comment3 (ID: 333333333):
    DELETE /repos/owner/repo/issues/comments/333333333
    → 204 No Content ✓

  For comment5 (ID: 555555555):
    DELETE /repos/owner/repo/issues/comments/555555555
    → 204 No Content ✓

Step 8: Log Results
  ↓
  ::notice::✓ Successfully cleared 3 comments in 1.23s
  ::notice::METRICS:{"event_type":"clear_command_executed",...}
```

---

## Testing

### Unit Tests

**Test Command Detection**:
```go
func TestDetectCommand(t *testing.T) {
    tests := []struct {
        input    string
        expected string
        found    bool
    }{
        {"@github-actions /clear", "clear", true},
        {"@github-actions /CLEAR", "clear", true},
        {"@github-actions /clear please", "clear", true},
        {"@github-actions hello", "", false},
        {"/clear without mention", "", false},
    }

    for _, tt := range tests {
        cmd, found := DetectCommand(tt.input)
        assert.Equal(t, tt.expected, cmd)
        assert.Equal(t, tt.found, found)
    }
}
```

**Test Bot Comment Identification**:
```go
func TestIsBotComment(t *testing.T) {
    tests := []struct {
        name     string
        comment  *github.IssueComment
        expected bool
    }{
        {
            name: "with marker",
            comment: &github.IssueComment{
                Body: github.String("<!-- gitleaks-diff-comment: ... --> content"),
            },
            expected: true,
        },
        {
            name: "bot author",
            comment: &github.IssueComment{
                Body: github.String("no marker"),
                User: &github.User{Login: github.String("github-actions[bot]")},
            },
            expected: true,
        },
        {
            name: "human comment",
            comment: &github.IssueComment{
                Body: github.String("LGTM"),
                User: &github.User{Login: github.String("octocat")},
            },
            expected: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := IsBotComment(tt.comment)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Integration Tests

**Test Full Clear Flow**:
```go
func TestClearCommandIntegration(t *testing.T) {
    // Setup: Create test PR with bot comments
    client := NewTestClient(t)
    pr := client.CreateTestPR()
    client.AddBotComment(pr, "comment 1")
    client.AddBotComment(pr, "comment 2")
    client.AddHumanComment(pr, "human comment")

    // Execute: Run clear command
    cmd := &ClearCommand{
        PRNumber:    pr.Number,
        RequestedBy: "test-user",
        client:      client,
    }
    err := cmd.Execute(context.Background())

    // Verify
    assert.NoError(t, err)
    comments := client.ListPRComments(pr.Number)
    assert.Equal(t, 1, len(comments)) // Only human comment remains
    assert.Equal(t, "human comment", comments[0].GetBody())
}
```

---

## Troubleshooting

### Common Issues

**Issue**: "Permission denied" error
**Solution**: Verify user has write, admin, or maintain access to the repository

**Issue**: Command not detected
**Solution**: Ensure comment includes `@github-actions` and `/clear` (case-insensitive)

**Issue**: Rate limit exceeded
**Solution**: Wait for rate limit to reset (check `X-RateLimit-Reset` header in logs)

**Issue**: No comments cleared but expected some
**Solution**: Check if comments have the invisible marker or bot author

### Debug Mode

Enable detailed logging:
```yaml
- name: Execute clear command
  run: |
    go run ./cmd/gitleaks-diff-comment \
      --command=clear \
      --pr-number=${{ github.event.issue.number }} \
      --comment-id=${{ github.event.comment.id }} \
      --requester=${{ github.event.comment.user.login }} \
      --debug
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    DEBUG: "true"
```

### Metrics Parsing

Extract metrics from workflow logs:
```bash
# Get workflow logs
gh run view <run-id> --log > workflow.log

# Extract metrics events
grep "METRICS:" workflow.log | sed 's/.*METRICS://' | jq .
```

Example output:
```json
{
  "event_type": "clear_command_executed",
  "timestamp": "2025-11-17T12:34:58Z",
  "pr_number": 123,
  "requested_by": "octocat",
  "comments_cleared": 5,
  "error_count": 0,
  "duration_seconds": 2.34,
  "retry_attempts": 0,
  "success": true
}
```

---

## Next Steps

1. Implement command detection (internal/commands/detector.go)
2. Implement permission checking (internal/github/client.go)
3. Implement comment filtering (internal/github/comments.go)
4. Implement clear command logic (internal/commands/clear.go)
5. Implement retry logic (internal/github/retry.go)
6. Create workflow file (.github/workflows/clear-command.yml)
7. Write unit tests (tests/unit/commands/)
8. Write integration tests (tests/integration/clear_command_test.go)
9. Test manually on a real PR
10. Document user-facing behavior in README.md
