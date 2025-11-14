# Tasks: GitHub Enterprise Server Support

**Feature**: `002-github-enterprise-support`
**Input**: Design documents from `/specs/002-github-enterprise-support/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Not explicitly requested in the specification. Test tasks are included as optional checkpoints for validation but can be implemented after core functionality if preferred.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

This is a single Go project with the following structure:
- Repository root: `/workspace`
- Source code: `/workspace/cmd/`, `/workspace/internal/`
- Tests: `/workspace/tests/` (to be created), `/workspace/internal/*/` (existing test files)
- Action config: `/workspace/action.yml`
- Documentation: `/workspace/README.md`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Prepare development environment and ensure existing tests pass

- [x] T001 Verify Go 1.25 environment and dependencies in go.mod
- [x] T002 [P] Run existing tests to establish baseline (go test ./...)
- [x] T003 [P] Create tests/integration/ directory for new enterprise integration tests
- [x] T004 [P] Review contracts/action-input.yml and contracts/github-client.md for implementation requirements

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [x] T005 Add gh-host input to action.yml with description and default empty string
- [x] T006 Add GHHost field to Config struct in internal/config/config.go
- [x] T007 Parse INPUT_GH-HOST environment variable in ParseFromEnv() in internal/config/config.go
- [x] T008 Add GHHost validation logic in Config.Validate() in internal/config/config.go (reject protocol, reject path, validate port)
- [x] T009 Update NewClient signature to accept ghHost parameter in internal/github/client.go
- [x] T010 Update main.go in cmd/gitleaks-diff-comment/main.go to pass cfg.GHHost to github.NewClient()

**Checkpoint**: Foundation ready - configuration parsing and client initialization infrastructure complete. User story implementation can now begin.

---

## Phase 3: User Story 1 - Enterprise User Can Use Action on Self-Hosted GitHub (Priority: P1) 🎯 MVP

**Goal**: Enable users to configure gh-host parameter and successfully connect to GitHub Enterprise Server instances, with full backward compatibility for GitHub.com users.

**Independent Test**:
1. Set gh-host to a test GHES hostname in workflow YAML
2. Run action on PR with .gitleaksignore changes
3. Verify comments are posted successfully to enterprise PR
4. Without gh-host, verify action still works on GitHub.com (regression test)

**Acceptance Criteria** (from spec.md):
- AC1: Action connects to enterprise API endpoint when gh-host is provided
- AC2: Comments posted successfully to enterprise PR
- AC3: Empty gh-host defaults to GitHub.com (backward compatibility)

### Implementation for User Story 1

- [x] T011 [US1] Implement enterprise URL construction logic in NewClient() in internal/github/client.go (prepend https://, call WithEnterpriseURLs)
- [x] T012 [US1] Add error handling for WithEnterpriseURLs failure in internal/github/client.go
- [x] T013 [US1] Add debug logging for gh-host value and computed API URL in NewClient() in internal/github/client.go
- [x] T014 [US1] Update existing unit tests for NewClient in internal/github/client_test.go to pass empty string for ghHost
- [x] T015 [P] [US1] Add unit test for NewClient with GitHub.com (empty gh-host) in internal/github/client_test.go
- [x] T016 [P] [US1] Add unit test for NewClient with enterprise hostname in internal/github/client_test.go
- [x] T017 [P] [US1] Add unit test for NewClient with enterprise hostname and port in internal/github/client_test.go
- [x] T018 [P] [US1] Add unit test for Config.Validate() with valid gh-host in internal/config/config_test.go
- [x] T019 [P] [US1] Add unit test for Config.Validate() rejecting gh-host with protocol in internal/config/config_test.go
- [x] T020 [P] [US1] Add unit test for Config.Validate() rejecting gh-host with path in internal/config/config_test.go
- [x] T021 [P] [US1] Add unit test for Config.Validate() with port number validation in internal/config/config_test.go

**Checkpoint**: User Story 1 MVP complete. At this point:
- ✅ gh-host parameter added to action.yml
- ✅ Configuration parsing and validation working
- ✅ GitHub client can connect to enterprise or GitHub.com
- ✅ Backward compatibility maintained (empty gh-host = GitHub.com)
- ✅ Unit tests cover configuration and client initialization

**Manual Test**: Deploy to test GHES instance and verify comment posting works

---

## Phase 4: User Story 2 - Support Multiple Enterprise Authentication Methods (Priority: P2)

**Goal**: Ensure action works with different token types (PAT, GitHub App tokens) and provides clear error messages for authentication failures.

**Independent Test**:
1. Use Personal Access Token with enterprise instance
2. Use GitHub App installation token with enterprise instance
3. Use token with insufficient permissions and verify error message

**Acceptance Criteria** (from spec.md):
- AC1: PAT authentication succeeds with enterprise
- AC2: GitHub App token authentication succeeds with enterprise
- AC3: Clear error messages for insufficient permissions

### Implementation for User Story 2

- [x] T022 [US2] Add authentication error detection logic in internal/github/client.go (distinguish auth vs network errors)
- [x] T023 [US2] Enhance error messages in NewClient to indicate authentication vs configuration issues in internal/github/client.go
- [x] T024 [US2] Add test helper function for mock enterprise API in tests/integration/enterprise_test.go
- [x] T025 [P] [US2] Add integration test for PAT authentication in tests/integration/enterprise_test.go
- [x] T026 [P] [US2] Add integration test for authentication failure with clear error message in tests/integration/enterprise_test.go
- [x] T027 [US2] Update Config.Validate() error messages to reference required token scopes in internal/config/config.go

**Checkpoint**: User Story 2 complete. At this point:
- ✅ Action works with different authentication methods
- ✅ Clear error messages distinguish auth failures from other errors
- ✅ Integration tests verify auth handling
- ✅ User Story 1 functionality still works independently

---

## Phase 5: User Story 3 - Validate Enterprise Instance Connectivity (Priority: P2)

**Goal**: Provide fast, clear error messages when enterprise configuration is invalid or instance is unreachable, improving troubleshooting experience.

**Independent Test**:
1. Provide invalid hostname format and verify error message with guidance
2. Provide unreachable hostname and verify clear connectivity error
3. Provide valid reachable hostname and verify action proceeds

**Acceptance Criteria** (from spec.md):
- AC1: Unreachable hosts fail fast with clear error message
- AC2: Invalid URL formats provide guidance on correct format
- AC3: Reachable instances proceed with normal operation

### Implementation for User Story 3

- [x] T028 [US3] Add URL format validation helper function in internal/config/config.go (check for protocol prefix, path separator)
- [x] T029 [US3] Enhance validation error messages with examples and guidance in Config.Validate() in internal/config/config.go
- [x] T030 [US3] Add network connectivity error handling in NewClient in internal/github/client.go
- [x] T031 [US3] Add timeout configuration for enterprise connectivity checks in internal/github/client.go (target <2 seconds per SC-004)
- [x] T032 [P] [US3] Add unit test for Config.Validate() error message format in internal/config/config_test.go
- [x] T033 [P] [US3] Add integration test for unreachable hostname error in tests/integration/enterprise_test.go
- [x] T034 [P] [US3] Add integration test for invalid URL format error in tests/integration/enterprise_test.go

**Checkpoint**: User Story 3 complete. At this point:
- ✅ Invalid configurations produce clear, actionable error messages
- ✅ Network errors are detected and reported clearly
- ✅ Validation happens quickly (<2 seconds per SC-004)
- ✅ Previous user stories still work independently

---

## Phase 6: User Story 4 - Support Enterprise-Specific Rate Limits (Priority: P3)

**Goal**: Respect custom rate limits from enterprise instances and log them when debug mode is enabled.

**Independent Test**:
1. Mock enterprise API with custom rate limit headers
2. Verify action reads and respects enterprise rate limits
3. Enable debug mode and verify rate limits are logged

**Acceptance Criteria** (from spec.md):
- AC1: Action reads enterprise rate limit headers
- AC2: Throttling respects enterprise-specific limits
- AC3: Debug logging shows detected rate limits

### Implementation for User Story 4

- [x] T035 [US4] Review existing rate limit handling in internal/github/comments.go (verify it uses rate limit headers)
- [x] T036 [US4] Add debug logging for rate limit values in CheckRateLimit in internal/github/client.go
- [x] T037 [US4] Add debug logging for rate limit detection when action initializes in cmd/gitleaks-diff-comment/main.go
- [x] T038 [P] [US4] Add integration test for custom rate limit handling in tests/integration/enterprise_test.go
- [x] T039 [P] [US4] Add integration test verifying debug logs show rate limit values in tests/integration/enterprise_test.go

**Checkpoint**: User Story 4 complete. At this point:
- ✅ Enterprise rate limits are respected
- ✅ Debug logging provides rate limit visibility
- ✅ All four user stories work independently
- ✅ Feature is fully functional per specification

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, testing validation, and final quality checks

- [ ] T040 [P] Add enterprise configuration section to README.md with gh-host examples
- [ ] T041 [P] Add troubleshooting section to README.md with common error messages and solutions
- [ ] T042 [P] Copy quickstart.md content to README.md enterprise section or link to quickstart guide
- [ ] T043 Run all unit tests (go test ./internal/...) and verify 100% pass
- [ ] T044 Run all integration tests (go test ./tests/integration/...) and verify pass
- [ ] T045 [P] Run go vet and golint to ensure code quality
- [ ] T046 [P] Verify Docker build succeeds (docker build -t test .)
- [ ] T047 Test gh-host with real GHES instance if available (manual validation)
- [ ] T048 Run regression tests with empty gh-host to verify GitHub.com functionality unchanged
- [ ] T049 [P] Update CHANGELOG or release notes with enterprise support feature
- [ ] T050 Review all error messages for clarity and actionability per data-model.md error states

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phases 3-6)**: All depend on Foundational phase completion
  - User stories CAN proceed in parallel if team capacity allows
  - OR sequentially in priority order: US1 (P1) → US2 (P2) → US3 (P2) → US4 (P3)
- **Polish (Phase 7)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - MVP requirement, no dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - Builds on US1's client initialization but tests independently
- **User Story 3 (P2)**: Can start after Foundational (Phase 2) - Enhances US1's validation, tests independently
- **User Story 4 (P3)**: Can start after Foundational (Phase 2) - Uses existing rate limit code, tests independently

**Note**: User Stories 2, 3, and 4 enhance User Story 1 but are independently testable and deployable.

### Within Each User Story

- Core implementation tasks before tests (pragmatic approach - implement then validate)
- Unit tests marked [P] can run in parallel (different test files)
- Integration tests depend on implementation being complete
- Story complete and validated before moving to next priority

### Parallel Opportunities

**Phase 1 (Setup)**:
- T002, T003, T004 can all run in parallel

**Phase 2 (Foundational)**:
- T005 (action.yml) independent of Go code changes
- T006-T008 (config.go) can be done together
- T009-T010 (client.go + main.go) depend on T006-T008 being complete

**Phase 3 (User Story 1)**:
- T015-T021 (all unit tests) can run in parallel once implementation (T011-T014) is complete
- Test writing can happen in parallel with final debugging

**Phase 4 (User Story 2)**:
- T025-T026 (integration tests) can run in parallel once test infrastructure (T022-T024) is ready

**Phase 5 (User Story 3)**:
- T032-T034 (all tests) can run in parallel once implementation (T028-T031) is complete

**Phase 6 (User Story 4)**:
- T038-T039 (integration tests) can run in parallel once implementation (T035-T037) is complete

**Phase 7 (Polish)**:
- T040-T042 (documentation) can run in parallel
- T045-T046 (linting and build) can run in parallel
- T049-T050 (release notes and review) can run in parallel

---

## Parallel Example: User Story 1

```bash
# After T011-T014 complete, launch all unit tests together:
Task T015: "Unit test for NewClient with GitHub.com (empty gh-host)"
Task T016: "Unit test for NewClient with enterprise hostname"
Task T017: "Unit test for NewClient with enterprise hostname and port"
Task T018: "Unit test for Config.Validate() with valid gh-host"
Task T019: "Unit test for Config.Validate() rejecting gh-host with protocol"
Task T020: "Unit test for Config.Validate() rejecting gh-host with path"
Task T021: "Unit test for Config.Validate() with port number validation"

# All 7 test tasks (T015-T021) can run in parallel as they modify different test files/functions
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

**Fastest path to working enterprise support:**

1. ✅ Complete Phase 1: Setup (4 tasks, ~30 minutes)
2. ✅ Complete Phase 2: Foundational (6 tasks, ~2-3 hours)
   - This adds gh-host parameter and basic client initialization
3. ✅ Complete Phase 3: User Story 1 (11 tasks, ~3-4 hours)
   - Core enterprise connectivity
   - Backward compatibility
   - Basic validation
4. **STOP and VALIDATE**:
   - Run unit tests (T015-T021)
   - Test with real GHES instance if available
   - Verify GitHub.com still works (regression)
5. **MVP READY**: Can deploy and use with enterprise instances

**Total MVP Time**: ~6-8 hours for experienced Go developer

### Incremental Delivery (Recommended)

**Deliver value progressively:**

1. **Foundation** (Phases 1-2): ~3-4 hours
   - gh-host parameter exists but not fully functional

2. **MVP: Core Enterprise Support** (+ Phase 3): ~3-4 hours
   - Enterprise connectivity working
   - Backward compatible
   - Deploy and gather feedback

3. **Better Auth Handling** (+ Phase 4): ~2-3 hours
   - Clearer error messages for auth issues
   - Better security troubleshooting

4. **Improved UX** (+ Phase 5): ~2-3 hours
   - Faster, clearer error messages
   - Better validation guidance

5. **Production Ready** (+ Phase 6-7): ~2-3 hours
   - Rate limit handling
   - Documentation
   - Final polish

**Total Time**: ~12-15 hours spread across multiple deployments

### Parallel Team Strategy

**If you have 2-3 developers:**

1. **Week 1**: Everyone works on Foundation (Phases 1-2)
   - Pair programming or code reviews
   - Ensures solid base for parallel work

2. **Week 2**: Split work after Foundation complete
   - **Developer A**: User Story 1 (MVP) - Priority 1
   - **Developer B**: User Story 2 + 3 (Auth + Validation) - Priority 2
   - **Developer C**: User Story 4 + Documentation - Priority 3

3. **Week 3**: Integration and polish
   - Merge all user stories
   - Cross-test each other's work
   - Complete Phase 7 polish together

**Benefit**: Can complete entire feature in 2-3 weeks instead of 4-5 weeks sequential

---

## Testing Strategy

### Unit Tests (Go standard testing)

**Coverage targets per user story:**
- US1: Configuration parsing, URL validation, client initialization
- US2: Authentication error detection and messaging
- US3: Validation logic and error message formatting
- US4: Rate limit logging (if testable at unit level)

**Command**: `go test ./internal/config ./internal/github -v`

### Integration Tests (Mock HTTP Server)

**Test scenarios:**
- Mock GitHub Enterprise Server API endpoints
- Test full workflow: parse config → create client → make API call
- Verify error handling for network failures, auth failures
- Verify rate limit header parsing

**Command**: `go test ./tests/integration/... -v`

**Location**: `tests/integration/enterprise_test.go`

### Regression Tests

**Critical checks:**
- All existing tests pass with empty gh-host
- GitHub.com workflows unchanged
- No breaking changes to Client interface
- Error messages still clear for GitHub.com users

**Command**: `go test ./... -v` (run entire test suite)

### Manual Validation

**If GHES instance available:**
1. Deploy action to test repository on GHES
2. Create PR modifying .gitleaksignore
3. Verify comments posted successfully
4. Test invalid configurations and verify error messages
5. Compare behavior to GitHub.com version

**If no GHES instance:**
- Unit and integration tests provide good coverage
- Mock server simulates enterprise API
- Ready for first production deployment

---

## Success Validation Checklist

Track progress against success criteria from spec.md:

- [ ] **SC-001**: Users can configure gh-host without code modifications (verify with real workflow YAML)
- [ ] **SC-002**: Connection time <5 seconds (measure with debug logging)
- [ ] **SC-003**: All operations work identically on enterprise (regression tests pass)
- [ ] **SC-004**: Configuration errors detected <2 seconds with clear messages (unit tests verify)
- [ ] **SC-005**: Supports GHES 3.14+ (documentation states minimum version)
- [ ] **SC-006**: Zero breaking changes for GitHub.com users (regression tests pass)
- [ ] **SC-007**: Setup time <10 minutes (quickstart.md validated)

---

## Task Completion Tracking

**Total Tasks**: 50

**By Phase**:
- Phase 1 (Setup): 4 tasks
- Phase 2 (Foundational): 6 tasks
- Phase 3 (User Story 1 - P1): 11 tasks
- Phase 4 (User Story 2 - P2): 6 tasks
- Phase 5 (User Story 3 - P2): 7 tasks
- Phase 6 (User Story 4 - P3): 5 tasks
- Phase 7 (Polish): 11 tasks

**By User Story**:
- User Story 1: 11 tasks (MVP critical)
- User Story 2: 6 tasks (auth improvements)
- User Story 3: 7 tasks (validation improvements)
- User Story 4: 5 tasks (rate limit support)
- Infrastructure/Polish: 21 tasks (setup + foundational + polish)

**Parallel Opportunities**: 29 tasks marked [P] can run in parallel within their phase

**Estimated Time**:
- MVP (Phases 1-3): 6-8 hours
- Full Feature (All Phases): 12-15 hours
- With Parallel Team: 8-10 hours

---

## Notes

- All tasks include exact file paths for implementation
- [P] tasks target different files or independent functionality
- Each user story is independently completable and testable
- Tests are included but can be implemented after core functionality if preferred
- Commit after each logical group of tasks (e.g., after each phase checkpoint)
- Stop at any checkpoint to validate story independence
- Priority: Focus on User Story 1 (P1) first for MVP, then add P2/P3 stories incrementally
