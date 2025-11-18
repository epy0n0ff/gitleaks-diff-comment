# Feature Specification: Clear Comments Command

**Feature Branch**: `003-clear-comments-command`
**Created**: 2025-11-17
**Status**: Draft
**Input**: User description: "github-actions botへのメンション付きコメントで、/clearと文字列のコメントがついた場合に、既についているコメントをクリアしたい。"

## Clarifications

### Session 2025-11-17

- Q: When the `/clear` command is executed and GitHub API rate limits are exceeded, how should the system respond? → A: Retry with exponential backoff (up to 3 attempts), then fail gracefully with clear error message
- Q: How should the system deliver feedback about the clear operation results to the user? → A: Only through GitHub Actions logs/workflow output (user must check workflow run)
- Q: When multiple users post `/clear` commands simultaneously on the same PR, how should race conditions be handled? → A: Each request processes independently; if comments already deleted, report 0 cleared

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Clear All Action Comments via Bot Command (Priority: P1)

As a pull request author or maintainer, I want to clear all existing gitleaks-diff-comment bot comments from a PR by mentioning the bot with a `/clear` command, so that I can remove outdated or no-longer-relevant security warnings without manually deleting each comment.

**Why this priority**: This is the core functionality that provides immediate value. Users can clean up their PRs when:
- They've addressed all security concerns and want to remove warning clutter
- They need to regenerate comments after fixing their .gitleaksignore
- Comments are outdated due to PR updates

**Independent Test**: Can be fully tested by creating a PR with existing bot comments, posting a mention comment with `/clear`, and verifying all bot comments are deleted. Delivers standalone value for PR cleanup.

**Acceptance Scenarios**:

1. **Given** a PR has 5 existing comments from gitleaks-diff-comment bot, **When** a user posts "@github-actions /clear", **Then** all 5 bot comments are deleted from the PR
2. **Given** a PR has bot comments and human comments, **When** a user posts "@github-actions /clear", **Then** only bot comments are deleted and human comments remain
3. **Given** a PR has no bot comments, **When** a user posts "@github-actions /clear", **Then** the action completes successfully with a message indicating 0 comments were deleted
4. **Given** a user posts "@github-actions /clear please remove old warnings", **When** the command is processed, **Then** the action recognizes the `/clear` command despite additional text and deletes all bot comments

---

### User Story 2 - Permissions and Authorization (Priority: P2)

As a repository maintainer, I want only authorized users (PR author, repo collaborators with write access, or maintainers) to be able to clear comments, so that random users cannot abuse the clear command to remove security warnings.

**Why this priority**: Essential for security and preventing abuse, but can be implemented after basic functionality works. Without this, malicious actors could hide security warnings.

**Independent Test**: Can be tested by having users with different permission levels attempt the `/clear` command and verifying only authorized users succeed. Delivers security value independently.

**Acceptance Scenarios**:

1. **Given** a PR created by User A, **When** User A posts "@github-actions /clear", **Then** the command succeeds and comments are deleted
2. **Given** a PR created by User A, **When** a collaborator with write access posts "@github-actions /clear", **Then** the command succeeds and comments are deleted
3. **Given** a PR created by User A, **When** an external user with no write access posts "@github-actions /clear", **Then** the command is rejected with a permission denied message
4. **Given** a PR created by User A, **When** a repo maintainer posts "@github-actions /clear", **Then** the command succeeds and comments are deleted

---

### User Story 3 - Confirmation and Feedback (Priority: P3)

As a user who runs the clear command, I want to receive feedback about what was cleared through the GitHub Actions workflow output, so that I can verify the operation succeeded and see how many comments were removed.

**Why this priority**: Nice-to-have for user experience. The primary value is clearing comments; confirmation is supplementary. Can be added after core functionality works.

**Independent Test**: Can be tested by running clear command and verifying the workflow output/logs contain statistics (e.g., "Cleared 3 comments"). Delivers transparency value independently.

**Acceptance Scenarios**:

1. **Given** a PR has 3 bot comments, **When** a user runs "@github-actions /clear", **Then** the workflow output logs "Successfully cleared 3 comments"
2. **Given** a PR has 0 bot comments, **When** a user runs "@github-actions /clear", **Then** the workflow output logs "No comments to clear"
3. **Given** the clear command fails due to API errors, **When** the operation is attempted, **Then** the workflow output logs an error message explaining the failure

---

### Edge Cases

- What happens when a user mentions the bot without the `/clear` command (e.g., just "@github-actions hello")?
  - System should ignore the mention or respond with a help message about available commands
- What happens when multiple users post `/clear` simultaneously on the same PR?
  - Each workflow execution processes independently, querying and deleting comments that exist at the time of execution. If comments are already deleted by a concurrent request, the workflow reports 0 comments cleared
- What happens when the bot loses permissions mid-operation?
  - System should fail gracefully with an error message and log the permission issue
- What happens when a user tries to clear comments while the action is currently running and posting new comments?
  - System should either queue the clear operation or inform the user to retry after the current action completes
- What happens when comment IDs are invalid or comments were already manually deleted?
  - System should skip invalid IDs and continue processing remaining comments, reporting final count
- What happens when GitHub API rate limits are exceeded during the clear operation?
  - System should retry with exponential backoff (up to 3 attempts), then fail gracefully with a clear error message explaining the rate limit and suggesting the user retry later

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST detect when a PR comment mentions the github-actions bot with the `/clear` command
- **FR-002**: System MUST identify all comments on the PR that were created by the gitleaks-diff-comment bot (using the invisible marker or API author filtering)
- **FR-003**: System MUST delete all identified bot comments from the PR when the `/clear` command is authorized
- **FR-004**: System MUST verify the requesting user has appropriate permissions (PR author, write collaborator, or maintainer) before executing the clear command
- **FR-005**: System MUST preserve all non-bot comments (human comments) when clearing bot comments
- **FR-006**: System MUST handle the `/clear` command regardless of additional text in the comment (e.g., "@github-actions /clear please remove old warnings" should work)
- **FR-007**: System MUST provide feedback through GitHub Actions workflow output/logs indicating how many comments were cleared
- **FR-008**: System MUST handle cases where no bot comments exist gracefully without errors
- **FR-009**: System MUST log the clear operation including who requested it, when, and how many comments were deleted
- **FR-010**: System MUST use the existing GitHub token permissions (pull_requests:write) to delete comments
- **FR-011**: System MUST retry comment deletion operations with exponential backoff (up to 3 attempts) when GitHub API rate limits are exceeded, then fail with a descriptive error message if all retries are exhausted

### Key Entities

- **Clear Command Request**: Represents a user's request to clear comments
  - Attributes: requesting user, PR number, timestamp, comment ID containing the command
  - Relationships: Associated with a specific pull request and user
- **Bot Comment**: Represents a comment created by the gitleaks-diff-comment action
  - Attributes: comment ID, PR number, creation timestamp, invisible marker
  - Relationships: Part of a pull request, created by the bot user
- **Authorization Context**: Represents the permission level of the requesting user
  - Attributes: user login, permission level (author/write/admin), verification status
  - Relationships: Associated with the repository and specific user

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can successfully clear all bot comments from a PR in under 10 seconds by posting a single mention comment with `/clear`
- **SC-002**: System correctly identifies and deletes only bot comments, preserving 100% of human comments in all tested scenarios
- **SC-003**: 100% of unauthorized clear attempts (from users without write access) are rejected with appropriate error messages
- **SC-004**: System provides clear feedback in workflow output immediately after operation completion indicating the number of comments cleared
- **SC-005**: Clear operation succeeds for PRs with up to 100 bot comments without timeout or performance degradation

## Assumptions

- The action will run in response to PR comment events (issue_comment webhook)
- The GitHub token used by the action has sufficient permissions to delete comments (pull_requests:write)
- Comments created by the bot contain the invisible marker (`<!-- gitleaks-diff-comment: ... -->`) for identification
- The GitHub API provides user permission information for authorization checks
- The action will be triggered when the bot is mentioned in a comment (filtered by comment body containing bot mention)

## Scope

### In Scope

- Detecting mentions of the github-actions bot with `/clear` command
- Identifying all comments created by the gitleaks-diff-comment bot on the current PR
- Deleting identified bot comments when authorized
- Permission verification for the requesting user
- Providing feedback on operation results
- Handling edge cases (no comments, permission errors, API failures)

### Out of Scope

- Clearing comments from other bots or actions
- Clearing comments across multiple PRs simultaneously
- Undo functionality to restore deleted comments
- Granular comment selection (e.g., "clear only addition comments")
- Scheduling automatic comment cleanup
- Comment archiving or backup before deletion
- Integration with other comment management tools

## Dependencies

- Existing GitHub Actions workflow infrastructure
- GitHub API access for comment retrieval and deletion
- GitHub webhook for issue_comment events
- User permission API or equivalent for authorization checks
- Existing gitleaks-diff-comment action codebase
