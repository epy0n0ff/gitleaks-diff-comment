# Workflow Interface Contract

**Feature**: `003-clear-comments-command`
**Date**: 2025-11-17
**Purpose**: Define the contract between GitHub Actions workflow and the clear command implementation

## Workflow Trigger

### Input: GitHub Webhook Event

**Event Type**: `issue_comment.created`

**Event Payload** (relevant fields):
```json
{
  "action": "created",
  "issue": {
    "number": 123,
    "pull_request": {
      "url": "https://api.github.com/repos/owner/repo/pulls/123"
    }
  },
  "comment": {
    "id": 987654321,
    "body": "@github-actions /clear please remove old warnings",
    "user": {
      "login": "octocat"
    },
    "created_at": "2025-11-17T12:34:56Z"
  },
  "repository": {
    "name": "repo",
    "owner": {
      "login": "owner"
    },
    "full_name": "owner/repo"
  }
}
```

**Preconditions**:
1. Event is for a pull request (not a standalone issue)
2. Comment body contains `@github-actions` mention
3. Comment body contains `/clear` command (case-insensitive)

**Workflow Filtering**:
```yaml
on:
  issue_comment:
    types: [created]

jobs:
  clear:
    if: |
      github.event.issue.pull_request &&
      contains(github.event.comment.body, '@github-actions') &&
      contains(github.event.comment.body, '/clear')
```

---

## Command Line Interface

### Input: Environment Variables

| Variable | Source | Required | Example | Description |
|----------|--------|----------|---------|-------------|
| `GITHUB_TOKEN` | Workflow secrets | Yes | `ghp_xxx...` | GitHub API authentication token |
| `GITHUB_REPOSITORY` | Workflow context | Yes | `owner/repo` | Repository full name |
| `GITHUB_EVENT_PATH` | GitHub Actions | Yes | `/path/to/event.json` | Path to webhook event payload file |

### Input: Command Arguments

**Execution Pattern**:
```bash
go run ./cmd/gitleaks-diff-comment \
  --command=clear \
  --pr-number=${PR_NUMBER} \
  --comment-id=${COMMENT_ID} \
  --requester=${REQUESTER_LOGIN}
```

**Arguments**:
- `--command=clear`: Command type (required)
- `--pr-number=<int>`: Pull request number (required)
- `--comment-id=<int64>`: Comment ID containing the command (required)
- `--requester=<string>`: GitHub login of user who issued command (required)

**Example**:
```bash
go run ./cmd/gitleaks-diff-comment \
  --command=clear \
  --pr-number=123 \
  --comment-id=987654321 \
  --requester=octocat
```

---

## Output Contract

### Success Case (Exit Code 0)

**Standard Output Format**:
```
::notice::Starting clear command for PR #123 (requested by octocat)
::notice::Found 5 bot comments to delete
::notice::Deleted comment 111111111
::notice::Deleted comment 222222222
::notice::Deleted comment 333333333
::notice::Deleted comment 444444444
::notice::Deleted comment 555555555
::notice::METRICS:{"event_type":"clear_command_executed","timestamp":"2025-11-17T12:34:58Z","pr_number":123,"requested_by":"octocat","comments_cleared":5,"error_count":0,"duration_seconds":2.34,"retry_attempts":0,"success":true}
::notice::✓ Successfully cleared 5 comments in 2.34s
```

**GitHub Actions Step Outputs**:
```yaml
outputs:
  cleared: "5"
  errors: "0"
  duration: "2.34"
  success: "true"
```

**Structured Log Event** (METRICS line):
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

### Error Cases

#### Unauthorized User (Exit Code 1)

**Standard Error**:
```
::error::Permission denied: User 'external-user' does not have required permissions
::error::Required: write, admin, or maintain access to repository
::error::Current permission level: read
::notice::METRICS:{"event_type":"clear_command_executed","timestamp":"2025-11-17T12:34:58Z","pr_number":123,"requested_by":"external-user","comments_cleared":0,"error_count":1,"duration_seconds":0.45,"retry_attempts":0,"success":false}
```

**Exit Code**: 1
**Step Outputs**:
```yaml
outputs:
  cleared: "0"
  errors: "1"
  duration: "0.45"
  success: "false"
```

---

#### Rate Limit Exceeded (Exit Code 1)

**Standard Error** (after 3 retry attempts):
```
::warning::Rate limit exceeded, retrying in 2.5s (attempt 1/3)
::warning::Rate limit exceeded, retrying in 4.8s (attempt 2/3)
::warning::Rate limit exceeded, retrying in 8.3s (attempt 3/3)
::error::Rate limit exceeded after 3 retry attempts
::error::GitHub API rate limit: 0 requests remaining, resets at 2025-11-17T13:00:00Z
::error::Please wait for rate limit to reset and try again
::notice::METRICS:{"event_type":"clear_command_executed","timestamp":"2025-11-17T12:35:10Z","pr_number":123,"requested_by":"octocat","comments_cleared":2,"error_count":3,"duration_seconds":15.6,"retry_attempts":3,"success":false}
```

**Exit Code**: 1
**Note**: Some comments may have been deleted before rate limit hit

---

#### No Comments Found (Exit Code 0)

**Standard Output**:
```
::notice::Starting clear command for PR #123 (requested by octocat)
::notice::No bot comments found to delete
::notice::METRICS:{"event_type":"clear_command_executed","timestamp":"2025-11-17T12:34:57Z","pr_number":123,"requested_by":"octocat","comments_cleared":0,"error_count":0,"duration_seconds":0.89,"retry_attempts":0,"success":true}
::notice::✓ Completed successfully (0 comments cleared) in 0.89s
```

**Exit Code**: 0 (not an error condition)

---

#### Invalid Command Format (Exit Code 1)

**Standard Error**:
```
::error::Invalid command: expected /clear (case-insensitive)
::error::Received command: '/cleer'
::error::Valid commands: /clear, /CLEAR, /Clear (case-insensitive)
```

**Exit Code**: 1

---

## GitHub API Interactions

### Required API Calls

#### 1. Check User Permissions

**Endpoint**: `GET /repos/{owner}/{repo}/collaborators/{username}/permission`

**Request**:
```http
GET /repos/owner/repo/collaborators/octocat/permission
Authorization: Bearer ghp_xxx...
```

**Response** (Success - Authorized):
```json
{
  "permission": "write",
  "user": {
    "login": "octocat"
  }
}
```

**Response** (Success - Unauthorized):
```json
{
  "permission": "read",
  "user": {
    "login": "external-user"
  }
}
```

**Error Handling**:
- 404 Not Found: User is not a collaborator (treat as unauthorized)
- 403 Forbidden: Insufficient token permissions
- 401 Unauthorized: Invalid token

---

#### 2. List PR Comments

**Endpoint**: `GET /repos/{owner}/{repo}/issues/{issue_number}/comments`

**Request**:
```http
GET /repos/owner/repo/issues/123/comments?per_page=100
Authorization: Bearer ghp_xxx...
```

**Response**:
```json
[
  {
    "id": 111111111,
    "body": "<!-- gitleaks-diff-comment: .gitleaksignore:64:RIGHT -->\n🔒 **Gitleaks Exclusion Added**...",
    "user": {
      "login": "github-actions[bot]"
    },
    "created_at": "2025-11-16T10:00:00Z"
  },
  {
    "id": 222222222,
    "body": "This looks good to me!",
    "user": {
      "login": "octocat"
    },
    "created_at": "2025-11-16T11:00:00Z"
  },
  {
    "id": 333333333,
    "body": "<!-- gitleaks-diff-comment: .gitleaksignore:65:RIGHT -->\n🔒 **Gitleaks Exclusion Added**...",
    "user": {
      "login": "github-actions[bot]"
    },
    "created_at": "2025-11-16T12:00:00Z"
  }
]
```

**Filtering Logic**:
- Comment 111111111: Bot comment (has marker) → DELETE
- Comment 222222222: Human comment (no marker, different author) → PRESERVE
- Comment 333333333: Bot comment (has marker) → DELETE

**Pagination**:
- Default: 30 comments per page
- Maximum: 100 comments per page
- For > 100 comments: Multiple API calls required (out of scope for MVP, SC-005 limits to 100)

---

#### 3. Delete Comment

**Endpoint**: `DELETE /repos/{owner}/{repo}/issues/comments/{comment_id}`

**Request**:
```http
DELETE /repos/owner/repo/issues/comments/111111111
Authorization: Bearer ghp_xxx...
```

**Response** (Success):
```http
HTTP/1.1 204 No Content
```

**Error Responses**:

**404 Not Found** (comment already deleted):
```json
{
  "message": "Not Found",
  "documentation_url": "https://docs.github.com/rest/issues/comments#delete-an-issue-comment"
}
```
*Handling*: Log as warning, continue (not a critical error)

**403 Rate Limit**:
```json
{
  "message": "API rate limit exceeded",
  "documentation_url": "https://docs.github.com/rest/overview/resources-in-the-rest-api#rate-limiting"
}
```
*Handling*: Retry with exponential backoff (up to 3 attempts)

**403 Forbidden** (insufficient permissions):
```json
{
  "message": "Forbidden",
  "documentation_url": "https://docs.github.com/rest/issues/comments#delete-an-issue-comment"
}
```
*Handling*: Fatal error, exit immediately

---

## Retry Strategy Contract

### Retry Conditions

**Retry-able Errors**:
1. Rate limit exceeded (403 with rate limit headers)
2. Temporary network errors (5xx server errors)
3. Timeout errors

**Non-Retry-able Errors**:
1. Authentication errors (401, 403 permission denied)
2. Not found errors (404)
3. Validation errors (400, 422)

### Retry Algorithm

**Parameters**:
- Initial delay: 2 seconds
- Backoff multiplier: 2x
- Maximum delay: 32 seconds
- Jitter: 0-50% of calculated delay
- Maximum attempts: 3

**Delay Calculation**:
```
attempt 1: immediate
attempt 2: 2s + jitter(0-1s) = 2-3s
attempt 3: 4s + jitter(0-2s) = 4-6s

Total maximum time: ~9 seconds (within 10s SC-001 goal)
```

**Pseudo-code**:
```python
for attempt in range(1, 4):
    result = delete_comment(comment_id)

    if result.success:
        return success

    if not is_retryable(result.error):
        return error

    if attempt == 3:
        return error  # Max retries exceeded

    delay = min(2 * (2 ** (attempt - 1)), 32)
    jitter = random(0, delay / 2)
    sleep(delay + jitter)

return error
```

---

## Token Permissions Required

**Minimum Required Permissions**:
```yaml
permissions:
  pull-requests: write  # For deleting PR review comments
  issues: write         # For deleting issue comments (PR comments are issue comments)
```

**Token Scopes** (for PAT):
- `repo` scope (includes pull requests and issues access)

**Not Required**:
- `contents: write` (no code changes)
- `actions: write` (no workflow modifications)
- `packages: write` (no package operations)

---

## Performance Contract

**Service Level Objectives** (from Success Criteria):

| Metric | Target | Source |
|--------|--------|--------|
| Operation completion time | < 10 seconds | SC-001 |
| Comments supported | Up to 100 | SC-005 |
| Human comment preservation | 100% | SC-002 |
| Unauthorized rejection rate | 100% | SC-003 |
| Feedback timing | Immediate (< 1s after completion) | SC-004 |

**Typical Performance**:
- Permission check: ~200ms
- List comments: ~300ms
- Delete 1 comment: ~200ms
- Delete 10 comments: ~2s (sequential, with ~200ms each)
- Total for 10 comments: ~2.5s

**Edge Case Performance**:
- 100 comments (max): ~20s sequential deletion
  - Solution: Concurrent deletion (future optimization)
  - MVP: Sequential deletion acceptable for <= 100 comments

---

## Monitoring and Observability

### Log Levels

**::notice::**: Informational, operation progress
```
::notice::Starting clear command for PR #123
::notice::Found 5 bot comments to delete
::notice::Deleted comment 111111111
```

**::warning::**: Recoverable errors, retry attempts
```
::warning::Rate limit exceeded, retrying in 2.5s (attempt 1/3)
::warning::Comment 999999999 not found (may have been deleted manually)
```

**::error::**: Fatal errors, operation failures
```
::error::Permission denied: User lacks required permissions
::error::Rate limit exceeded after 3 retry attempts
```

### Metrics Event Schema

**Event Type**: `clear_command_executed`

**Required Fields**:
```typescript
interface MetricsEvent {
  event_type: "clear_command_executed";  // Constant
  timestamp: string;                      // ISO 8601 UTC
  pr_number: number;                      // Positive integer
  requested_by: string;                   // GitHub login
  comments_cleared: number;               // >= 0
  error_count: number;                    // >= 0
  duration_seconds: number;               // > 0
  retry_attempts: number;                 // 0-3
  success: boolean;                       // true if error_count == 0
}
```

**Example**:
```json
{
  "event_type": "clear_command_executed",
  "timestamp": "2025-11-17T12:34:58.123Z",
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

## Contract Versioning

**Version**: 1.0.0
**Status**: Initial implementation
**Breaking Changes**: N/A (first version)

**Future Considerations**:
- Concurrent deletion support (performance optimization)
- Selective comment clearing (e.g., "clear only additions")
- Comment archiving before deletion
- Undo functionality

**Backward Compatibility**: This is the initial implementation, no backward compatibility concerns.
