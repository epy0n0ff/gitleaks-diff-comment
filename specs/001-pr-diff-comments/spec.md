# Feature Specification: Automated PR Diff Comment Explanations

**Feature Branch**: `001-pr-diff-comments`
**Created**: 2025-11-13
**Status**: Draft
**Input**: User description: "PRが作成されたときにGithubActionsでdiffで差分があったファイルについての補足説明をコメントで追加したい"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Automated Context for Changed Files (Priority: P1)

When a developer creates a pull request, the system automatically analyzes the changed files and adds explanatory comments to help reviewers understand what was modified and why those changes matter.

**Why this priority**: This is the core value proposition - providing immediate context to reviewers without manual effort. Reviewers can understand changes faster, leading to quicker reviews and fewer back-and-forth questions.

**Independent Test**: Can be fully tested by creating a PR with file changes and verifying that automated comments appear on the PR with explanations of what changed in each file.

**Acceptance Scenarios**:

1. **Given** a new pull request is created with 3 modified files, **When** the GitHub Action workflow runs, **Then** comments are added to the PR explaining what changed in each of the 3 files
2. **Given** a pull request with new files added, **When** the automation runs, **Then** comments explain the purpose of the newly added files
3. **Given** a pull request with deleted files, **When** the automation runs, **Then** comments explain what files were removed and their previous purpose

---

### User Story 2 - Handling Different File Types (Priority: P2)

The system provides contextually appropriate explanations based on the file type - code files get different analysis than configuration files or documentation.

**Why this priority**: Different file types need different kinds of explanations. Code changes need logic descriptions, config changes need impact explanations, and docs need content summaries.

**Independent Test**: Create a PR with mixed file types (source code, config files, markdown) and verify each gets appropriate contextual explanations.

**Acceptance Scenarios**:

1. **Given** a PR containing both source code and configuration file changes, **When** comments are generated, **Then** code files receive logic-focused explanations and config files receive impact-focused explanations
2. **Given** a PR with only documentation changes, **When** comments are generated, **Then** explanations summarize content changes without technical implementation details
3. **Given** a PR with binary files or images, **When** comments are generated, **Then** the system indicates file type changes without attempting to analyze content

---

### User Story 3 - Large Pull Request Handling (Priority: P2)

For pull requests with many changed files, the system prioritizes which files get detailed explanations to avoid overwhelming reviewers with too many comments.

**Why this priority**: Large PRs can have dozens of changed files. Commenting on every file could create noise. Prioritization ensures important changes get attention.

**Independent Test**: Create a PR with 20+ changed files and verify that the most significant changes receive detailed comments while minor changes are summarized.

**Acceptance Scenarios**:

1. **Given** a PR with more than 10 changed files, **When** comments are generated, **Then** files with the most lines changed receive detailed explanations
2. **Given** a PR with many small formatting changes and few logic changes, **When** comments are generated, **Then** logic changes are prioritized over formatting changes
3. **Given** a PR exceeding a comment threshold, **When** the system generates comments, **Then** a single summary comment is posted instead of individual file comments

---

### User Story 4 - Workflow Integration and Triggering (Priority: P1)

The GitHub Action triggers automatically when a pull request is opened or updated, requiring no manual intervention from developers.

**Why this priority**: Automation must be seamless. If developers need to manually trigger it, adoption will fail. This is essential for the feature to provide value consistently.

**Independent Test**: Open a new PR and verify the GitHub Action runs automatically without any manual triggers. Update the PR and verify it runs again.

**Acceptance Scenarios**:

1. **Given** a new pull request is opened, **When** the PR is created, **Then** the GitHub Action workflow triggers automatically within 1 minute
2. **Given** an existing pull request receives new commits, **When** the commits are pushed, **Then** the workflow triggers again and updates comments
3. **Given** a draft pull request is converted to ready for review, **When** the status changes, **Then** the workflow triggers to add comments

---

### Edge Cases

- What happens when a PR has no file changes (empty PR)?
- How does the system handle PRs with hundreds of changed files?
- What if the GitHub Action encounters rate limits from the GitHub API?
- How are merge conflicts in changed files handled?
- What happens if a file is renamed without content changes?
- How does the system handle very large files (>1000 lines changed)?
- What if the PR is created by a bot or automated system?
- How are comments updated if the PR is force-pushed?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST trigger automatically when a pull request is opened
- **FR-002**: System MUST trigger automatically when a pull request is updated with new commits
- **FR-003**: System MUST identify all files that have been modified, added, or deleted in the PR diff
- **FR-004**: System MUST generate explanatory text describing what changed in each file
- **FR-005**: System MUST post generated explanations as comments on the pull request
- **FR-006**: System MUST handle PRs with unlimited changed files without artificial limits
- **FR-007**: System MUST differentiate between file types (source code, configuration, documentation, etc.) when generating explanations
- **FR-008**: System MUST avoid posting duplicate comments if the workflow runs multiple times
- **FR-009**: System MUST complete processing and post comments within a reasonable timeframe (target: within 2 minutes of PR creation)
- **FR-010**: System MUST handle errors gracefully without blocking the PR workflow
- **FR-011**: System MUST identify file additions, deletions, and modifications separately
- **FR-012**: System MUST use GitHub API authentication to post comments securely
- **FR-013**: System MUST respect GitHub API rate limits and handle throttling
- **FR-014**: System MUST provide meaningful explanations, not just file names or line counts

### Key Entities

- **Pull Request**: The GitHub pull request that triggers the workflow, contains metadata about the changes, author, and target branch
- **Changed File**: Individual file within the PR that has been added, modified, or deleted, includes diff information and file metadata
- **Diff Content**: The actual line-by-line changes within each file, used to generate contextual explanations
- **Generated Comment**: The explanatory text created for each changed file, posted to the PR as a review comment or general comment
- **Workflow Execution**: A single run of the GitHub Action, tracks processing status and results for one PR event

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Reviewers can understand the purpose of changed files within 30 seconds of viewing the PR
- **SC-002**: 90% of PRs receive automated comments within 2 minutes of creation
- **SC-003**: System successfully processes PRs regardless of the number of changed files (tested with PRs containing 100+ files)
- **SC-004**: Automated comments reduce reviewer questions about "what changed in this file" by 60%
- **SC-005**: 85% of generated explanations are rated as helpful by reviewers (if feedback mechanism exists)
- **SC-006**: Zero false negatives - all changed files in scope are commented on
- **SC-007**: Workflow execution time scales linearly with number of changed files
- **SC-008**: System maintains 99% uptime during PR creation events

## Assumptions *(optional)*

- The repository uses GitHub as the source control platform
- Developers have appropriate permissions to configure GitHub Actions
- The repository already has basic CI/CD workflows configured
- Comments will be in English (or the same language as code comments in the repository)
- The GitHub Action has network access to call GitHub APIs
- Repository maintainers want automated comments and won't consider them spam
- File explanations don't need to be persisted beyond the PR lifecycle
- Standard GitHub API rate limits are sufficient for the expected PR volume

## Dependencies *(optional)*

- GitHub Actions must be enabled for the repository
- Repository must have a valid GitHub token/PAT for API access with PR comment permissions
- Workflow must have access to PR diff information via GitHub context
- System will use rule-based text generation (no external AI services required)

## Constraints *(optional)*

- Must work within GitHub Actions execution time limits (maximum 6 hours, but target under 2 minutes)
- Must stay within GitHub API rate limits (5000 requests per hour for authenticated requests)
- Comments must be concise enough to be useful but not overwhelming (suggested: 100-300 characters per file)
- Cannot modify files or PR content, only add comments
- Must not expose sensitive information in comments (credentials, secrets, internal URLs)

## Out of Scope *(optional)*

- Analyzing code quality or suggesting improvements (this is about explaining changes, not critiquing them)
- Providing automated code review or approval/rejection decisions
- Generating comments for issues (only pull requests)
- Interactive conversation or responding to reviewer questions
- Integrating with external project management tools
- Providing multilingual explanations based on reviewer preferences
- Analyzing performance impacts of changes
- Tracking historical patterns across multiple PRs
