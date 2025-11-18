# Data Model: Clear Comments Command

**Feature**: `003-clear-comments-command`
**Date**: 2025-11-17
**Purpose**: Define data structures, state transitions, and validation rules for the clear command feature

## Overview

This feature introduces command handling infrastructure to the existing gitleaks-diff-comment action. The data model defines structures for:
1. Command detection and parsing
2. User authorization context
3. Comment identification and filtering
4. Operation results and metrics

## Entities

### 1. Command

Represents a user-issued command detected in a PR comment.

**Attributes**:
- `Type` (string): Command type, e.g., "clear"
- `IssueNumber` (int): Pull request number
- `CommentID` (int64): GitHub comment ID containing the command
- `RequestedBy` (string): GitHub login of user who issued command
- `RequestedAt` (time.Time): Timestamp when command was detected
- `Raw` (string): Original comment body text

**Validation Rules**:
- `Type` must be non-empty and match known command pattern (case-insensitive)
- `IssueNumber` must be positive integer
- `CommentID` must be positive integer
- `RequestedBy` must be non-empty string (validated GitHub username)

**State Transitions**:
```
Detected → Validated → Authorized → Executed → Completed
         ↓            ↓             ↓
       Invalid    Unauthorized    Failed
```

**Relationships**:
- Belongs to one Pull Request (via IssueNumber)
- Issued by one User (via RequestedBy)
- Contains zero or one CommandResult

---

### 2. Authorization

Represents the permission check result for a command requester.

**Attributes**:
- `Username` (string): GitHub login being checked
- `PermissionLevel` (string): GitHub permission level (none/read/write/admin/maintain)
- `IsAuthorized` (bool): Whether user can execute command
- `CheckedAt` (time.Time): When permission was verified
- `Reason` (string): Explanation if not authorized

**Validation Rules**:
- `Username` must be non-empty
- `PermissionLevel` must be one of: none, read, write, admin, maintain
- `IsAuthorized` derived from PermissionLevel: true if write/admin/maintain, false otherwise
- `Reason` required if `IsAuthorized` is false

**Authorization Logic**:
```go
allowedLevels := map[string]bool{
    "write":    true,
    "admin":    true,
    "maintain": true,
}
IsAuthorized = allowedLevels[PermissionLevel]
```

**Relationships**:
- Associated with one Command
- Represents permissions for one User in one Repository

---

### 3. BotComment

Represents a comment that was created by the gitleaks-diff-comment bot.

**Attributes**:
- `ID` (int64): GitHub comment ID
- `Body` (string): Comment body text
- `CreatedAt` (time.Time): When comment was created
- `Author` (string): Comment author login
- `HasMarker` (bool): Whether comment contains invisible marker
- `MarkerContent` (string): Extracted marker text if present
- `IsBotComment` (bool): Identification result (true if bot-created)

**Validation Rules**:
- `ID` must be positive integer
- `Body` must be non-empty string
- `Author` must be non-empty string
- `IsBotComment` determined by: HasMarker OR Author == "github-actions[bot]"

**Identification Algorithm**:
```
1. Check Body for "<!-- gitleaks-diff-comment:" substring
   - If found: HasMarker=true, extract marker → IsBotComment=true
2. If not found: Check Author == "github-actions[bot]"
   - If match: IsBotComment=true
3. Otherwise: IsBotComment=false
```

**Relationships**:
- Belongs to one Pull Request
- May be deleted by one ClearOperation

---

### 4. ClearOperation

Represents the execution and results of a clear command.

**Attributes**:
- `CommandID` (string): Unique identifier for this operation (format: "clear-{pr}-{timestamp}")
- `PRNumber` (int): Pull request number
- `RequestedBy` (string): User who initiated the operation
- `StartedAt` (time.Time): Operation start timestamp
- `CompletedAt` (time.Time): Operation completion timestamp (nil if in progress)
- `Status` (string): Operation status (pending/running/completed/failed)
- `CommentsFound` (int): Total bot comments found
- `CommentsDeleted` (int): Successfully deleted comments
- `CommentsFailed` (int): Failed deletion attempts
- `Errors` ([]string): List of error messages encountered
- `RetryCount` (int): Number of retry attempts made
- `Duration` (float64): Total operation time in seconds

**Validation Rules**:
- `CommandID` must be unique and follow format "clear-{pr}-{unixtime}"
- `PRNumber` must be positive integer
- `RequestedBy` must be non-empty string
- `Status` must be one of: pending, running, completed, failed
- `CommentsDeleted` <= `CommentsFound`
- `CommentsFailed` >= 0
- `Duration` = CompletedAt - StartedAt (when completed)

**State Transitions**:
```
pending → running → completed
                 ↓
                failed

Events:
- pending → running: Operation starts, begins fetching comments
- running → completed: All deletions succeeded or gracefully handled
- running → failed: Unrecoverable error (auth, timeout, max retries exceeded)
```

**Success Criteria**:
- Status = "completed"
- CommentsFailed = 0 OR all failures are expected (404 not found - already deleted)
- Duration < 10 seconds (performance goal)

**Relationships**:
- Triggered by one Command
- Deletes zero or more BotComments
- Creates one MetricsEvent

---

### 5. MetricsEvent

Represents metrics data for observability and monitoring.

**Attributes**:
- `EventType` (string): Always "clear_command_executed"
- `Timestamp` (time.Time): Event timestamp (ISO 8601 format)
- `PRNumber` (int): Pull request number
- `RequestedBy` (string): User who executed command
- `CommentsCleared` (int): Number of comments successfully deleted
- `ErrorCount` (int): Number of errors encountered
- `DurationSeconds` (float64): Total operation time
- `RetryAttempts` (int): Number of retries performed
- `Success` (bool): Whether operation completed successfully

**Validation Rules**:
- `EventType` must equal "clear_command_executed"
- `Timestamp` in UTC timezone
- `PRNumber` must be positive integer
- `CommentsCleared` >= 0
- `ErrorCount` >= 0
- `DurationSeconds` > 0
- `RetryAttempts` >= 0 and <= 3
- `Success` = true if ErrorCount == 0 AND operation completed

**Serialization Format** (JSON):
```json
{
  "event_type": "clear_command_executed",
  "timestamp": "2025-11-17T12:34:56Z",
  "pr_number": 123,
  "requested_by": "octocat",
  "comments_cleared": 5,
  "error_count": 0,
  "duration_seconds": 2.34,
  "retry_attempts": 0,
  "success": true
}
```

**Relationships**:
- Generated by one ClearOperation
- Consumed by external monitoring systems via workflow logs

---

## Data Flow

### Clear Command Execution Flow

```
1. Webhook Event (issue_comment.created)
   ↓
2. Parse Comment → Create Command entity
   ↓
3. Validate Command (syntax, PR existence)
   ↓
4. Check Authorization → Create Authorization entity
   ↓
5. Fetch PR Comments → Filter to BotComment entities
   ↓
6. Create ClearOperation (status: pending → running)
   ↓
7. For each BotComment:
   a. Delete via GitHub API
   b. Retry with exponential backoff if rate limited
   c. Track success/failure
   ↓
8. Update ClearOperation (status: completed/failed)
   ↓
9. Generate MetricsEvent
   ↓
10. Log results to workflow output
```

### Error Handling State Transitions

```
Rate Limit Error:
  running → (retry with backoff) → running
  After 3 retries: running → failed

Permission Error:
  Unauthorized → (immediate failure) → failed

Not Found Error (404):
  running → (continue, increment CommentsFailed) → running
  Note: Comment already deleted, not a critical error

Network Error:
  running → (retry with backoff) → running
  After 3 retries: running → failed
```

---

## Constraints and Invariants

### System Constraints

1. **Concurrency**: Multiple ClearOperations can run simultaneously on same PR
   - Each operates on snapshot of comments at query time
   - No distributed locking required
   - Race conditions acceptable (idempotent deletion)

2. **Retry Limits**:
   - Maximum 3 retry attempts per comment deletion
   - Total operation timeout: 10 seconds (SC-001)
   - Exponential backoff: 2s, 4s, 8s base delays

3. **Scale Limits**:
   - Maximum 100 bot comments per PR (SC-005)
   - Comments fetched in single API call (GitHub returns up to 100 per page)

### Data Invariants

1. **Command Uniqueness**: Each Command.CommentID is unique (one command per comment)

2. **Authorization Consistency**: Authorization.IsAuthorized must match permission level logic

3. **Operation Totals**:
   ```
   ClearOperation.CommentsFound = CommentsDeleted + CommentsFailed
   ```

4. **Metrics Accuracy**:
   ```
   MetricsEvent.CommentsCleared == ClearOperation.CommentsDeleted
   MetricsEvent.ErrorCount == ClearOperation.CommentsFailed
   MetricsEvent.Success == (ClearOperation.Status == "completed")
   ```

5. **Status Progression**: ClearOperation.Status only moves forward in state machine (no rollbacks)

---

## Validation Rules Summary

| Entity | Key Validation | Error Message |
|--------|----------------|---------------|
| Command | Type matches `/clear` (case-insensitive) | "Invalid command type: expected /clear" |
| Command | IssueNumber > 0 | "Invalid PR number: must be positive" |
| Command | RequestedBy non-empty | "Missing requesting user" |
| Authorization | PermissionLevel in allowed set | "Unknown permission level" |
| Authorization | IsAuthorized true for write+ | "User lacks required permissions (write, admin, or maintain)" |
| BotComment | ID > 0 | "Invalid comment ID" |
| BotComment | IsBotComment correctly identified | N/A (soft failure, log warning) |
| ClearOperation | CommentsDeleted <= CommentsFound | "Deleted count exceeds found count" |
| ClearOperation | Status in valid set | "Invalid operation status" |
| MetricsEvent | EventType == "clear_command_executed" | "Invalid event type" |
| MetricsEvent | DurationSeconds > 0 | "Invalid duration" |

---

## Database / Storage

**Storage Type**: None (stateless operation)

All entities exist only in-memory during workflow execution:
- Command parsed from webhook payload
- Authorization fetched from GitHub API
- BotComments fetched from GitHub API
- ClearOperation tracks in-memory state
- MetricsEvent written to workflow logs (stdout)

No persistent storage required. This maintains the stateless nature of the GitHub Action.

---

## Next Steps

1. Define API contracts for workflow inputs and GitHub API interactions
2. Create quickstart guide demonstrating data flow
3. Generate implementation tasks for each entity and operation
