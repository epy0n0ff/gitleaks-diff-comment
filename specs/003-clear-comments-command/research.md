# Research: Clear Comments Command

**Feature**: `003-clear-comments-command`
**Date**: 2025-11-17
**Purpose**: Research technical approaches and best practices for implementing the `/clear` command feature

## Research Questions

Based on Technical Context unknowns and implementation requirements:

1. How to detect and parse bot mention commands in GitHub issue comments?
2. How to verify user permissions (PR author, write access, maintainer) via GitHub API?
3. How to identify bot-created comments using invisible markers or API filtering?
4. How to implement exponential backoff retry for GitHub API rate limits?
5. How to trigger GitHub Actions on issue_comment events?
6. How to track execution metrics in GitHub Actions context?

---

## R1: Bot Mention Command Detection

### Decision
Use regex pattern matching on issue comment body to detect `@github-actions` mention followed by case-insensitive `/clear` command.

### Rationale
- GitHub webhook `issue_comment` event provides comment body as plain text
- Regex allows flexible matching (handles whitespace, additional text after command)
- Case-insensitive matching via `(?i)` flag in Go regex
- Simple, testable, no external dependencies

### Implementation Pattern
```go
// Pseudo-code pattern
commandPattern := regexp.MustCompile(`(?i)@github-actions\s+/clear`)
if commandPattern.MatchString(commentBody) {
    // Process clear command
}
```

### Alternatives Considered
- **String exact match**: Too rigid, doesn't handle variations
  - Rejected: Doesn't meet FR-006 (additional text handling)
- **Full NLP parsing**: Overkill for simple command
  - Rejected: Unnecessary complexity for single command pattern
- **GitHub slash commands API**: Not available for custom actions
  - Rejected: GitHub doesn't provide this for third-party actions

### Edge Cases to Handle
- Multiple mentions in same comment (process first match)
- Command in code blocks (ignore, only process plain text)
- Partial matches like `@github-actions-bot` (must match exact bot name)

---

## R2: User Permission Verification

### Decision
Use GitHub API `repos.GetPermissionLevel` to verify requesting user has write access or higher.

### Rationale
- GitHub API provides permission levels: none, read, write, admin
- Write access indicates collaborator with commit rights
- API automatically handles PR author detection (author always has permission on their PR)
- Maintainers have admin level

### Implementation Pattern
```go
// Using go-github/v57
permission, _, err := client.Repositories.GetPermissionLevel(ctx, owner, repo, username)
if err != nil {
    return fmt.Errorf("failed to check permissions: %w", err)
}

// Accept write, admin, or maintain levels
allowedLevels := map[string]bool{
    "write": true,
    "admin": true,
    "maintain": true,
}

if !allowedLevels[permission.GetPermission()] {
    return ErrUnauthorized
}
```

### Alternatives Considered
- **Check PR author only**: Too restrictive
  - Rejected: Doesn't meet FR-004 (collaborators should have access)
- **Check team membership**: Requires additional API calls
  - Rejected: Permission level API is simpler and sufficient
- **Allow any authenticated user**: Security risk
  - Rejected: Violates security requirements (US2)

### API Endpoint
`GET /repos/{owner}/{repo}/collaborators/{username}/permission`

---

## R3: Bot Comment Identification

### Decision
Use invisible HTML comment markers (`<!-- gitleaks-diff-comment: ... -->`) as primary method, with API author filtering as fallback.

### Rationale
- Existing codebase already adds invisible markers to all bot comments
- Markers provide reliable identification even if bot account changes
- Fallback to API `comment.User.Login` for backward compatibility
- No additional API calls needed (comments already fetched)

### Implementation Pattern
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
```

### Alternatives Considered
- **API author filtering only**: Brittle if bot name changes
  - Rejected: Markers provide more reliable identification
- **Database of comment IDs**: Requires state management
  - Rejected: Action should remain stateless
- **GitHub GraphQL comment metadata**: More complex API
  - Rejected: REST API sufficient, no added benefit

### Edge Cases
- Comments edited by users (marker preserved in body)
- Old comments without markers (fallback handles these)

---

## R4: Exponential Backoff Retry Strategy

### Decision
Implement custom exponential backoff with jitter for GitHub API rate limit retries (max 3 attempts).

### Rationale
- GitHub API returns `X-RateLimit-*` headers indicating rate limit status
- Exponential backoff prevents thundering herd problem
- Jitter prevents synchronized retries from multiple workflows
- 3 attempts balances reliability vs workflow timeout risk

### Implementation Pattern
```go
func retryWithBackoff(operation func() error, maxRetries int) error {
    baseDelay := 2 * time.Second
    maxDelay := 32 * time.Second

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

        // Calculate delay: baseDelay * 2^attempt + jitter
        delay := baseDelay * (1 << uint(attempt))
        if delay > maxDelay {
            delay = maxDelay
        }

        // Add jitter: random 0-50% of delay
        jitter := time.Duration(rand.Int63n(int64(delay / 2)))
        time.Sleep(delay + jitter)
    }

    return fmt.Errorf("unexpected: exhausted retries")
}
```

### Retry Schedule
- Attempt 1: Immediate
- Attempt 2: 2-3s delay (2s + 0-1s jitter)
- Attempt 3: 4-6s delay (4s + 0-2s jitter)
- Total max time: ~9 seconds (within 10s performance goal)

### Alternatives Considered
- **GitHub API client built-in retry**: go-github v57 doesn't provide automatic retry
  - Rejected: Must implement custom
- **Constant backoff**: Doesn't reduce load on API
  - Rejected: Exponential is GitHub best practice
- **Unlimited retries**: Risk workflow timeout
  - Rejected: 3 attempts balanced against 10s goal

### Rate Limit Detection
```go
func isRateLimitError(err error) bool {
    var rateLimitErr *github.RateLimitError
    return errors.As(err, &rateLimitErr)
}
```

---

## R5: GitHub Actions Workflow Trigger

### Decision
Create separate workflow file triggered by `issue_comment` event with type filter for bot mentions.

### Rationale
- `issue_comment` event fires for all PR comments
- Can filter by comment body containing `@github-actions`
- Separate workflow avoids interfering with existing diff comment workflow
- Allows independent scaling and timeout configuration

### Implementation Pattern
```yaml
# .github/workflows/clear-command.yml
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

      - name: Execute clear command
        run: |
          go run ./cmd/gitleaks-diff-comment \
            --command=clear \
            --pr-number=${{ github.event.issue.number }} \
            --comment-id=${{ github.event.comment.id }}
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### Key Workflow Configuration
- **Event**: `issue_comment.types: [created]` (not edited/deleted)
- **Condition**: Must be PR comment + contain bot mention + contain /clear
- **Permissions**: `pull-requests: write` for comment deletion
- **Timeout**: Default 10 minutes (sufficient for 10s operation goal)

### Alternatives Considered
- **Modify existing workflow**: Complicates existing logic
  - Rejected: Separate workflows maintain clarity
- **Use workflow_dispatch**: Requires manual trigger
  - Rejected: Should be automatic on comment creation
- **Use GitHub App webhook**: Requires external infrastructure
  - Rejected: GitHub Actions sufficient, simpler deployment

---

## R6: Execution Metrics Tracking

### Decision
Use GitHub Actions workflow outputs and logs for execution count tracking. Emit structured log events for external monitoring systems to consume.

### Rationale
- GitHub Actions provides job summary and step outputs natively
- Structured logging (JSON format) allows external log aggregation
- No additional infrastructure required
- Aligns with clarification (workflow output feedback)

### Implementation Pattern
```go
type MetricsEvent struct {
    EventType   string    `json:"event_type"`
    Timestamp   time.Time `json:"timestamp"`
    PRNumber    int       `json:"pr_number"`
    RequestedBy string    `json:"requested_by"`
    Cleared     int       `json:"comments_cleared"`
    Errors      int       `json:"errors"`
    Duration    float64   `json:"duration_seconds"`
}

func logMetrics(event MetricsEvent) {
    jsonBytes, _ := json.Marshal(event)
    fmt.Printf("::notice::METRICS:%s\n", string(jsonBytes))
}
```

### Metrics Collected
- **Execution count**: Number of times command executed (FR-012)
- **Comments cleared**: Total comments deleted per execution
- **Error count**: Failed deletion operations
- **Duration**: Time to complete operation (SC-001 validation)
- **Requesting user**: Who executed command (FR-009)

### GitHub Actions Integration
```yaml
- name: Clear comments
  id: clear
  run: go run ./cmd/gitleaks-diff-comment --command=clear

- name: Report metrics
  run: |
    echo "cleared=${{ steps.clear.outputs.cleared }}" >> $GITHUB_STEP_SUMMARY
    echo "duration=${{ steps.clear.outputs.duration }}s" >> $GITHUB_STEP_SUMMARY
```

### Alternatives Considered
- **External metrics service (Datadog, CloudWatch)**: Requires API keys, additional cost
  - Rejected: Workflow logs sufficient for MVP
- **GitHub API usage analytics**: Only tracks API call counts, not feature usage
  - Rejected: Need feature-specific metrics
- **Custom metrics endpoint**: Requires external infrastructure
  - Rejected: Over-engineering for execution count tracking

### Consumption Pattern
External systems can:
1. Fetch workflow run logs via GitHub API
2. Parse `METRICS:` JSON events from logs
3. Aggregate execution counts, duration percentiles, error rates
4. Alert on anomalies (high error rate, slow duration)

---

## Summary of Decisions

| Question | Decision | Key Benefit |
|----------|----------|-------------|
| Command detection | Regex pattern matching with case-insensitive flag | Simple, flexible, testable |
| Permission verification | GitHub API `GetPermissionLevel` | Leverages existing API, handles all roles |
| Bot comment identification | Invisible HTML markers + API author fallback | Reliable, backward compatible |
| Rate limit retry | Exponential backoff with jitter (3 attempts) | Respects API limits, prevents cascades |
| Workflow trigger | Separate `issue_comment` workflow | Clean separation, independent scaling |
| Metrics tracking | Workflow outputs + structured logging | No infrastructure, consumable by external systems |

## Technology Choices Confirmed

- **Go 1.25**: Matches existing codebase, no migration needed
- **go-github/v57**: Already in use, provides all required APIs
- **GitHub Actions**: Native platform, no external hosting needed
- **Go standard library**: Testing, regex, time - no additional dependencies

## Next Steps

1. **Phase 1**: Create data model defining command structures and state
2. **Phase 1**: Define API contracts for workflow inputs/outputs
3. **Phase 1**: Write quickstart guide for users and developers
4. **Phase 2**: Generate implementation tasks based on research findings
