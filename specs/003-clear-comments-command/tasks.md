# Tasks: Clear Comments Command

**Feature**: `003-clear-comments-command`
**Input**: Design documents from `/specs/003-clear-comments-command/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Not explicitly requested in the specification. Test tasks are included as optional checkpoints for validation but can be implemented after core functionality if preferred.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

This is a single Go project with the following structure:
- Repository root: `/workspace`
- Source code: `/workspace/cmd/`, `/workspace/internal/`
- Tests: `/workspace/tests/` (to be created), `/workspace/internal/*/` (existing test files)
- Workflow: `/workspace/.github/workflows/`
- Documentation: `/workspace/README.md`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare development environment and create shared command infrastructure

- [x] T001 Create internal/commands package directory structure at /workspace/internal/commands/
- [x] T002 [P] Create tests/unit/commands directory for command unit tests at /workspace/tests/unit/commands/
- [x] T003 [P] Create tests/integration directory if not exists at /workspace/tests/integration/
- [x] T004 [P] Create tests/fixtures directory for test data at /workspace/tests/fixtures/

---

## Phase 2: Foundational (Core Command Infrastructure)

**Purpose**: Build command detection and routing infrastructure needed by all user stories

- [x] T005 Create Command struct with fields (Type, IssueNumber, CommentID, RequestedBy, Raw) in /workspace/internal/commands/types.go
- [x] T006 [P] Implement DetectCommand function with case-insensitive regex pattern matching in /workspace/internal/commands/detector.go
- [x] T007 [P] Add unit tests for DetectCommand covering case variations and edge cases in /workspace/tests/unit/commands/detector_test.go
- [x] T008 Update Config struct to include Command field in /workspace/internal/config/config.go
- [x] T009 Add command-line flag parsing (--command, --pr-number, --comment-id, --requester) in /workspace/cmd/gitleaks-diff-comment/main.go
- [x] T010 Implement command routing logic to dispatch to clear handler in /workspace/cmd/gitleaks-diff-comment/main.go

---

## Phase 3: User Story 1 - Clear All Action Comments (P1)

**Purpose**: Implement core comment clearing functionality

**Why this priority**: This is the MVP - enables users to clean up bot comments via /clear command

**Independent Test**: Create PR with bot comments, run /clear command, verify all bot comments deleted and human comments preserved

### Comment Identification

- [x] T011 [P] [US1] Create IsBotComment function to check for invisible marker in /workspace/internal/github/comments.go
- [x] T012 [P] [US1] Add fallback bot author detection (github-actions[bot]) in IsBotComment function in /workspace/internal/github/comments.go
- [x] T013 [P] [US1] Implement FilterBotComments function to separate bot comments from human comments in /workspace/internal/github/comments.go
- [x] T014 [P] [US1] Add unit tests for IsBotComment and FilterBotComments in /workspace/tests/unit/commands/comments_test.go

### Comment Deletion

- [x] T015 [US1] Add DeleteComment method to GitHub client in /workspace/internal/github/client.go
- [x] T016 [US1] Implement ListPRComments method to fetch all comments for a PR in /workspace/internal/github/client.go
- [x] T017 [US1] Create ClearOperation struct to track execution state in /workspace/internal/commands/clear.go
- [x] T018 [US1] Implement ClearCommand.Execute method with comment fetching, filtering, and deletion loop in /workspace/internal/commands/clear.go
- [x] T019 [US1] Add error handling for 404 Not Found (comment already deleted) in /workspace/internal/commands/clear.go
- [ ] T020 [P] [US1] Add unit tests for ClearCommand.Execute with mocked GitHub client in /workspace/tests/unit/commands/clear_test.go

### Integration

- [x] T021 [US1] Create GitHub Actions workflow file for issue_comment event trigger at /workspace/.github/workflows/clear-command.yml
- [x] T022 [US1] Add workflow job with permissions (pull-requests: write, issues: write) in /workspace/.github/workflows/clear-command.yml
- [x] T023 [US1] Add workflow step to execute clear command with environment variables in /workspace/.github/workflows/clear-command.yml
- [ ] T024 [P] [US1] Create integration test that simulates full clear flow in /workspace/tests/integration/clear_command_test.go

---

## Phase 4: User Story 2 - Permissions and Authorization (P2)

**Purpose**: Add authorization checks to prevent unauthorized comment clearing

**Why this priority**: Essential for security, prevents abuse

**Independent Test**: Test with users at different permission levels, verify only authorized users can clear comments

### Permission Verification

- [x] T025 [P] [US2] Create Authorization struct with fields (Username, PermissionLevel, IsAuthorized, Reason) in /workspace/internal/commands/types.go
- [x] T026 [US2] Implement CheckUserPermission method using GitHub API GetPermissionLevel in /workspace/internal/github/client.go
- [x] T027 [US2] Add permission level validation logic (allow write/admin/maintain) in /workspace/internal/github/client.go
- [ ] T028 [P] [US2] Add unit tests for CheckUserPermission covering all permission levels in /workspace/tests/unit/commands/authorization_test.go

### Integration with Clear Command

- [x] T029 [US2] Add permission check at start of ClearCommand.Execute in /workspace/internal/commands/clear.go
- [x] T030 [US2] Create ErrUnauthorized error type with descriptive message in /workspace/internal/commands/errors.go
- [x] T031 [US2] Update main.go to handle ErrUnauthorized and output appropriate error message in /workspace/cmd/gitleaks-diff-comment/main.go
- [ ] T032 [P] [US2] Add integration test for unauthorized user attempting clear command in /workspace/tests/integration/authorization_test.go

---

## Phase 5: User Story 3 - Confirmation and Feedback (P3)

**Purpose**: Add workflow output logging and metrics for transparency

**Why this priority**: Improves user experience but not critical for functionality

**Independent Test**: Run clear command and verify workflow logs contain operation statistics

### Metrics and Logging

- [x] T033 [P] [US3] Create MetricsEvent struct matching contract schema in /workspace/internal/commands/metrics.go
- [x] T034 [P] [US3] Implement logMetrics function to output structured JSON in /workspace/internal/commands/metrics.go
- [x] T035 [US3] Add operation timing tracking (StartedAt, CompletedAt, Duration) to ClearOperation in /workspace/internal/commands/clear.go
- [x] T036 [US3] Add counter tracking (CommentsFound, CommentsDeleted, CommentsFailed) to ClearOperation in /workspace/internal/commands/clear.go
- [x] T037 [US3] Call logMetrics at end of ClearCommand.Execute with final counts in /workspace/internal/commands/clear.go

### Workflow Output

- [x] T038 [US3] Add ::notice:: log statements for operation progress in /workspace/internal/commands/clear.go
- [x] T039 [US3] Add ::warning:: log statements for retry attempts in /workspace/internal/commands/clear.go
- [x] T040 [US3] Add ::error:: log statements for fatal errors in /workspace/internal/commands/clear.go
- [x] T041 [US3] Update workflow to capture and display output summary in /workspace/.github/workflows/clear-command.yml
- [ ] T042 [P] [US3] Add integration test that verifies metrics output format in /workspace/tests/integration/metrics_test.go

---

## Phase 6: Retry Logic and Error Handling

**Purpose**: Implement exponential backoff retry for rate limits and robust error handling

- [ ] T043 [P] Create retry.go with retryWithBackoff function implementing exponential backoff in /workspace/internal/github/retry.go
- [ ] T044 [P] Implement isRateLimitError helper function to detect rate limit errors in /workspace/internal/github/retry.go
- [ ] T045 [P] Add jitter calculation to retry delays (0-50% of base delay) in /workspace/internal/github/retry.go
- [ ] T046 Update ClearCommand to use retryWithBackoff for comment deletions in /workspace/internal/commands/clear.go
- [ ] T047 Add retry attempt counter and logging for each retry in /workspace/internal/commands/clear.go
- [ ] T048 [P] Add unit tests for retry logic with mocked rate limit errors in /workspace/tests/unit/commands/retry_test.go
- [ ] T049 [P] Add integration test for rate limit scenario with mocked GitHub API in /workspace/tests/integration/rate_limit_test.go

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, edge case handling, and final validation

- [ ] T050 [P] Add command detection for non-clear mentions with help message logic in /workspace/internal/commands/detector.go
- [ ] T051 [P] Add validation for invalid command IDs and PR numbers in /workspace/internal/config/config.go
- [ ] T052 [P] Update README.md with /clear command usage instructions and examples
- [ ] T053 [P] Add troubleshooting section to README.md covering common errors (permissions, rate limits, etc.)
- [ ] T054 Run all unit tests and verify 100% pass (go test ./internal/... ./cmd/...)
- [ ] T055 Run all integration tests and verify pass (go test ./tests/integration/...)
- [ ] T056 [P] Test workflow manually on test repository with real PR and bot comments
- [ ] T057 [P] Verify concurrent clear command execution handling (multiple users posting /clear simultaneously)
- [ ] T058 [P] Review all error messages for clarity and actionability per data-model.md specifications
- [ ] T059 Update CHANGELOG.md with /clear command feature description and usage examples

---

## Dependencies & Execution Order

### User Story Dependencies

```
Setup (Phase 1) → Foundational (Phase 2) → All User Stories can proceed in parallel

User Story 1 (P1) ─┐
                   ├→ Phase 6 (Retry Logic) → Phase 7 (Polish)
User Story 2 (P2) ─┤
                   │
User Story 3 (P3) ─┘
```

### Critical Path

1. **Setup** (T001-T004): Directory structure - must complete first
2. **Foundational** (T005-T010): Command infrastructure - blocking for all stories
3. **User Story 1** (T011-T024): Core functionality - MVP completion
4. **User Story 2** (T025-T032): Authorization - independent of US1, can start after Foundational
5. **User Story 3** (T033-T042): Metrics - independent of US1/US2, can start after Foundational
6. **Retry Logic** (T043-T049): Depends on US1 (uses ClearCommand)
7. **Polish** (T050-T059): Depends on all previous phases

### Parallel Execution Opportunities

**Phase 1 (Setup)**: All tasks (T001-T004) can run in parallel - different directories

**Phase 2 (Foundational)**:
- Parallel: T006, T007 (detector + tests in different files)
- Sequential: T005 → T008 → T009 → T010 (data dependencies)

**Phase 3 (US1)**:
- Parallel: T011, T012, T013, T014 (comment identification + tests)
- Parallel: T015, T016 (GitHub client methods in same file but independent)
- Sequential: T017 → T018 → T019 (ClearCommand dependencies)
- Parallel: T020, T024 (tests after implementation)
- Parallel: T021, T022, T023 (workflow file sections)

**Phase 4 (US2)**:
- Parallel: T025, T026, T027, T028 (authorization logic + tests)
- Sequential: T029 → T030 → T031 (integration into clear command)
- Parallel: T032 (test after integration)

**Phase 5 (US3)**:
- Parallel: T033, T034 (metrics struct + logging)
- Sequential: T035 → T036 → T037 (timing and counters in ClearOperation)
- Parallel: T038, T039, T040 (logging statements)
- Sequential: T041 (workflow update after logging complete)
- Parallel: T042 (test after implementation)

**Phase 6 (Retry)**:
- Parallel: T043, T044, T045, T048 (retry logic + tests)
- Sequential: T046 → T047 (integration into clear command)
- Parallel: T049 (integration test)

**Phase 7 (Polish)**:
- Parallel: T050, T051, T052, T053, T058, T059 (independent updates)
- Sequential: T054, T055, T056, T057 (testing sequence)

---

## Implementation Strategy

### MVP (Minimum Viable Product)

**Scope**: User Story 1 only (T001-T024)

**Delivers**:
- Command detection for /clear
- Bot comment identification and deletion
- Basic workflow integration
- Core functionality without auth or metrics

**Time Estimate**: 1-2 days

**Validation**: Manual test on PR with bot comments, verify deletion works

### Incremental Delivery

**Sprint 1** (MVP):
- Phase 1, 2, 3 (Setup + Foundational + US1)
- Deliverable: Working /clear command (no auth)

**Sprint 2** (Security):
- Phase 4 (US2)
- Deliverable: Permission checks prevent unauthorized clearing

**Sprint 3** (Observability):
- Phase 5, 6 (US3 + Retry Logic)
- Deliverable: Metrics logging and rate limit handling

**Sprint 4** (Polish):
- Phase 7
- Deliverable: Production-ready with documentation

### Testing Strategy

**Unit Tests** (included in each phase):
- Command detection: T007
- Comment identification: T014
- Clear command logic: T020
- Authorization: T028
- Retry logic: T048

**Integration Tests** (validation checkpoints):
- Full clear flow: T024
- Authorization scenarios: T032
- Metrics output: T042
- Rate limit handling: T049

**Manual Testing** (final validation):
- Real PR test: T056
- Concurrent execution: T057

### Task Count Summary

- **Total Tasks**: 59
- **Phase 1 (Setup)**: 4 tasks
- **Phase 2 (Foundational)**: 6 tasks
- **Phase 3 (US1)**: 14 tasks
- **Phase 4 (US2)**: 8 tasks
- **Phase 5 (US3)**: 10 tasks
- **Phase 6 (Retry)**: 7 tasks
- **Phase 7 (Polish)**: 10 tasks

**Parallel Opportunities**: 28 tasks marked [P] can run concurrently

**MVP Task Count**: 24 tasks (Phases 1, 2, 3)

---

## Next Steps

1. Begin with Phase 1 (Setup) to create directory structure
2. Complete Phase 2 (Foundational) for command infrastructure
3. Implement Phase 3 (US1) for MVP
4. Optional: Add Phases 4-7 for production readiness
5. Run tests throughout (unit tests included in each phase)
6. Final validation with Phase 7 (manual testing and polish)

## Success Criteria

Each user story must be independently testable:

- **US1 Success**: User can post @github-actions /clear and bot comments are deleted
- **US2 Success**: Unauthorized users are rejected with permission error
- **US3 Success**: Workflow logs show operation statistics and metrics

Each phase builds incrementally on previous phases while maintaining independent story value.
