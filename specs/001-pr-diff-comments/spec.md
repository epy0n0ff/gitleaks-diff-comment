# Feature Specification: Automated PR Diff Comment Explanations

**Feature Branch**: `001-pr-diff-comments`
**Created**: 2025-11-13
**Status**: Draft
**Input**: User description: "PRが作成されたときにGithubActionsでdiffで差分があったファイルについての補足説明をコメントで追加したい"

## Clarifications

### Session 2025-11-13

- Q: Feature Scope - Should the system comment on all changed files or only specific files? → A: Only `.gitleaksignore` files are analyzed and commented on
- Q: Comment Content Structure - What format should the comments use? → A: Security-focused with emoji indicators (🔒 for additions, ✅ for deletions) and file links
- Q: Non-existent File Handling - What if a .gitleaksignore entry references a file that doesn't exist? → A: Post comment with link anyway, GitHub will show 404 page
- Q: Rate Limit Recovery Strategy - How should the system handle GitHub API rate limits? → A: Retry with exponential backoff (1s, 2s, 4s), fail gracefully after 3 attempts
- Q: Force-Push Comment Handling - What happens to comments when PR is force-pushed? → A: Keep existing comments, add new comments only for new changes

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Automated Context for .gitleaksignore Changes (Priority: P1)

When a developer creates a pull request that modifies `.gitleaksignore`, the system automatically analyzes the changes and adds explanatory comments on the specific diff lines where files are being added or removed from the gitleaks ignore list. Each comment provides context about which file is being excluded from security scanning and links to that file.

**Why this priority**: This is the core value proposition - providing immediate security context when developers modify what files are excluded from secret scanning. Reviewers can quickly understand the security implications of .gitleaksignore changes without manually inspecting each line.

**Independent Test**: Can be fully tested by creating a PR that modifies .gitleaksignore and verifying that automated comments appear inline on the specific lines that were added or removed.

**Acceptance Scenarios**:

1. **Given** a new pull request adds 3 file patterns to `.gitleaksignore`, **When** the GitHub Action workflow runs, **Then** line-level comments are added on each of the 3 added lines explaining which files are now excluded from scanning
2. **Given** a pull request removes file patterns from `.gitleaksignore`, **When** the automation runs, **Then** line-level comments on the removed lines explain that these files will now be scanned
3. **Given** a pull request both adds and removes entries in `.gitleaksignore`, **When** the automation runs, **Then** appropriate comments are posted for both additions and deletions

---

### User Story 2 - Handling Different Entry Types in .gitleaksignore (Priority: P2)

The system generates appropriate comments based on the type of entry in `.gitleaksignore` - file paths with line numbers get different treatment than wildcard patterns, and comments include working links to the referenced files in the repository.

**Why this priority**: .gitleaksignore entries can be specific file:line references (e.g., `config/secrets.yml:42`) or wildcard patterns (e.g., `*.env`). Comments should adapt to provide the most relevant context and links for each type.

**Independent Test**: Create a PR that adds both specific file:line entries and wildcard patterns to .gitleaksignore, verify each type receives appropriately formatted comments with correct file links.

**Acceptance Scenarios**:

1. **Given** a PR adds `config/secrets.yml:42` to `.gitleaksignore`, **When** comments are generated, **Then** the comment includes a link to the specific file and mentions the line number being ignored
2. **Given** a PR adds `*.env` pattern to `.gitleaksignore`, **When** comments are generated, **Then** the comment indicates this is a wildcard pattern that matches multiple files
3. **Given** a PR removes a wildcard pattern from `.gitleaksignore`, **When** comments are generated, **Then** the comment explains all matching files will now be scanned

---

### User Story 3 - Large .gitleaksignore Changes (Priority: P2)

For pull requests with many lines added or removed in `.gitleaksignore`, the system posts individual line-level comments for each change without artificial limits.

**Why this priority**: Even when .gitleaksignore has many changes (e.g., bulk adding/removing patterns), each entry has security implications that reviewers need to understand. Every change should be commented on.

**Independent Test**: Create a PR that adds or removes 20+ entries in .gitleaksignore and verify that each line receives an individual comment.

**Acceptance Scenarios**:

1. **Given** a PR adds 50 entries to `.gitleaksignore`, **When** comments are generated, **Then** all 50 additions receive individual line-level comments
2. **Given** a PR removes 30 entries from `.gitleaksignore`, **When** comments are generated, **Then** all 30 deletions receive individual line-level comments explaining re-activation of scanning
3. **Given** a PR with 100+ changes to `.gitleaksignore`, **When** the workflow runs, **Then** it completes within 2 minutes despite the volume

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

- What happens when a PR has no `.gitleaksignore` changes (workflow should not trigger)?
- What if `.gitleaksignore` is deleted entirely in the PR?
- What if `.gitleaksignore` is created for the first time in the PR?
- How does the system handle merge conflicts in `.gitleaksignore`?
- Rate limit handling: System retries with exponential backoff (1s, 2s, 4s) up to 3 attempts, then fails gracefully
- Entry references non-existent file: Comments are posted with links anyway (GitHub shows 404 gracefully)
- What if `.gitleaksignore` has 100+ changes in a single PR?
- What if the PR is created by a bot or automated system?
- Force-push handling: Existing comments are preserved, deduplication logic prevents re-posting identical comments
- What if `.gitleaksignore` contains malformed entries (invalid patterns)?
- What happens if the same line in `.gitleaksignore` is modified (deletion + addition at same location)?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST trigger automatically when a pull request is opened that modifies `.gitleaksignore`
- **FR-002**: System MUST trigger automatically when a pull request is updated with new commits that modify `.gitleaksignore`
- **FR-003**: System MUST identify only `.gitleaksignore` file changes in the PR diff (ignore all other files)
- **FR-004**: System MUST parse each added or removed line in `.gitleaksignore` to extract the file pattern or path
- **FR-005**: System MUST generate line-level review comments for each added or removed line in `.gitleaksignore`
- **FR-006**: System MUST include a link to the referenced file in each comment (based on the pattern in .gitleaksignore)
- **FR-007**: System MUST differentiate between additions (files being excluded) and deletions (files being re-scanned) in comment text using emoji indicators (🔒 for additions, ✅ for deletions)
- **FR-008**: System MUST format addition comments as: "🔒 **Gitleaks Exclusion Added** - `<file_pattern>` will be excluded from secret scanning. [View file](<link>)" with appropriate security warning
- **FR-009**: System MUST format deletion comments as: "✅ **Gitleaks Exclusion Removed** - `<file_pattern>` will now be scanned by gitleaks. [View file](<link>)"
- **FR-010**: System MUST handle `.gitleaksignore` entries with line numbers (e.g., `path/file.yml:42`) and link to the specific file
- **FR-011**: System MUST handle wildcard patterns (e.g., `*.env`, `config/*`) and indicate pattern matching in comments
- **FR-012**: System MUST generate file links for all entries without validating file existence (GitHub will handle 404s gracefully if files don't exist)
- **FR-013**: System MUST post comments as PR review comments at the specific diff positions
- **FR-014**: System MUST avoid posting duplicate comments if the workflow runs multiple times (including after force-push events)
- **FR-014a**: System MUST preserve existing comments when PR is force-pushed and only add new comments for newly introduced changes
- **FR-015**: System MUST complete processing and post comments within 2 minutes regardless of number of .gitleaksignore changes
- **FR-016**: System MUST handle errors gracefully without blocking the PR workflow
- **FR-017**: System MUST use GitHub API authentication to post comments securely
- **FR-018**: System MUST implement exponential backoff retry logic (1 second, 2 seconds, 4 seconds) when encountering API rate limits
- **FR-019**: System MUST fail gracefully after 3 retry attempts if rate limits persist, logging the failure without blocking the PR

### Key Entities

- **Pull Request**: The GitHub pull request that triggers the workflow when `.gitleaksignore` is modified
- **Gitleaksignore Diff**: The specific changes to the `.gitleaksignore` file, parsed line-by-line for additions and deletions
- **Gitleaks Entry**: A single line in `.gitleaksignore` representing a file path or pattern to be excluded from secret scanning (may include optional line number like `path:42`)
- **Generated Comment**: The explanatory text created for each added or removed line in `.gitleaksignore`, posted as a line-level PR review comment with file link
- **File Link**: A URL pointing to the file referenced in a gitleaks entry, constructed from the repository path and commit SHA

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Reviewers can understand which files are being excluded/re-scanned within 10 seconds of viewing the `.gitleaksignore` diff
- **SC-002**: 90% of PRs with `.gitleaksignore` changes receive automated comments within 2 minutes of creation
- **SC-003**: System successfully processes `.gitleaksignore` changes regardless of the number of lines modified (tested with 100+ line changes)
- **SC-004**: Every added or removed line in `.gitleaksignore` receives exactly one line-level comment with a file link
- **SC-005**: 85% of generated comments are rated as helpful by reviewers for understanding security implications
- **SC-006**: Zero false negatives - all added/removed lines in `.gitleaksignore` are commented on
- **SC-007**: Workflow execution time scales linearly with number of `.gitleaksignore` line changes
- **SC-008**: System maintains 99% uptime during PR creation events involving `.gitleaksignore`
- **SC-009**: File links in comments are valid and point to correct files/directories 95% of the time

## Assumptions *(optional)*

- The repository uses GitHub as the source control platform
- The repository uses gitleaks for secret scanning
- Developers have appropriate permissions to configure GitHub Actions
- Comments will be in English
- The GitHub Action has network access to call GitHub APIs
- Repository maintainers want security-focused automated comments on `.gitleaksignore` changes
- Comments don't need to be persisted beyond the PR lifecycle
- Standard GitHub API rate limits are sufficient for the expected PR volume
- `.gitleaksignore` follows standard gitleaks format (file paths or patterns, optionally with `:line_number` suffix)

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

- Commenting on any files other than `.gitleaksignore`
- Validating whether the files referenced in `.gitleaksignore` actually contain secrets
- Running gitleaks scans or secret detection
- Analyzing code quality or suggesting improvements
- Providing automated code review or approval/rejection decisions
- Generating comments for issues (only pull requests)
- Interactive conversation or responding to reviewer questions
- Integrating with external project management tools
- Providing multilingual explanations
- Tracking historical patterns across multiple PRs
- Automatically approving or rejecting PRs based on `.gitleaksignore` changes
