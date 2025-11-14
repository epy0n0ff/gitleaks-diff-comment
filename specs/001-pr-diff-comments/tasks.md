# Tasks: Automated PR Diff Comment Explanations

**Input**: Design documents from `/specs/001-pr-diff-comments/`
**Prerequisites**: plan.md (Go/Docker custom action), spec.md (.gitleaksignore-focused user stories), research.md, data-model.md, contracts/

**Tests**: Tests are NOT explicitly requested in the specification, so test tasks are NOT included. Focus on implementation only.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

Custom GitHub Action structure:
- Root: `action.yml`, `Dockerfile`, `go.mod`
- Source: `cmd/pr-diff-comment/`, `internal/{diff,comment,github,config}/`
- Tests: `tests/fixtures/`, `tests/integration/`
- Docs: `README.md`, `DEVELOPMENT.md`

---

## Phase 1: Setup (Project Initialization)

**Purpose**: Initialize Go project structure and Docker configuration

- [X] T001 Initialize Go module with `go mod init` in repository root (go.mod, go.sum)
- [X] T002 [P] Create directory structure: cmd/pr-diff-comment/, internal/{diff,comment,github,config}/, tests/{fixtures,integration}/
- [X] T003 [P] Create action.yml with action metadata (name, description, inputs, runs configuration)
- [X] T004 [P] Create Dockerfile with multi-stage build (golang:1.24-alpine builder + alpine:3.22 runtime)
- [X] T005 [P] Create .gitignore for Go project (binaries, vendor/, coverage files)
- [X] T006 [P] Create README.md with action usage instructions and examples
- [X] T007 [P] Create DEVELOPMENT.md with local development setup and testing guide

---

## Phase 2: Foundational (Core Infrastructure)

**Purpose**: Core packages and interfaces that ALL user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T008 Create Config struct and parsing logic in internal/config/config.go (parse action inputs from environment)
- [X] T009 [P] Create DiffChange and OperationType types in internal/diff/types.go
- [X] T010 [P] Create GitleaksEntry struct in internal/diff/types.go
- [X] T011 [P] Create GitHub Client interface in internal/github/client.go
- [X] T012 [P] Create CommentData and GeneratedComment structs in internal/comment/types.go
- [X] T013 [P] Create GitHub API types (PostCommentRequest, PostCommentResponse, ExistingComment, CommentResult, ActionOutput) in internal/github/types.go
- [X] T014 Add go-github dependency: `go get github.com/google/go-github/v57@latest`
- [X] T015 [P] Add oauth2 dependency: `go get golang.org/x/oauth2@latest`
- [X] T016 Create markdown comment templates (templates/addition.md, templates/deletion.md) with emoji indicators
- [X] T017 Implement ClientImpl NewClient function in internal/github/client.go (GitHub API client initialization with oauth2)

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - Automated Context for .gitleaksignore Changes (Priority: P1) 🎯 MVP

**Goal**: Parse .gitleaksignore diffs and post line-level comments with file links

**Independent Test**: Create a PR that modifies .gitleaksignore, verify automated comments appear inline on specific changed lines

### Implementation for User Story 1

- [X] T018 [P] [US1] Implement ParseGitleaksDiff function in internal/diff/parser.go (parse git diff output for .gitleaksignore)
- [X] T019 [P] [US1] Implement ParseGitleaksEntry function in internal/diff/parser.go (extract file pattern and line number from entry)
- [X] T020 [P] [US1] Implement FileLink method on GitleaksEntry in internal/diff/types.go (generate GitHub file URLs)
- [X] T021 [US1] Implement template rendering in internal/comment/generator.go (use text/template with embedded templates)
- [X] T022 [US1] Implement NewGeneratedComment function in internal/comment/generator.go (create comment from DiffChange)
- [X] T023 [US1] Implement CreateReviewComment method in internal/github/comments.go (post line-level review comments via go-github)
- [X] T024 [US1] Implement ListReviewComments method in internal/github/comments.go (fetch existing comments for deduplication)
- [X] T025 [US1] Implement deduplication logic in internal/github/comments.go (check if identical comment exists before posting)
- [X] T026 [US1] Create main.go entry point in cmd/pr-diff-comment/main.go (parse config, call diff parser, generate comments, post via GitHub client)
- [X] T027 [US1] Add error handling and logging throughout the pipeline (config validation, diff parsing errors, API failures)
- [X] T028 [US1] Create sample-diff.txt and sample-gitleaksignore fixtures in tests/fixtures/ for testing
- [X] T029 [US1] Update action.yml with required inputs (github-token, pr-number) and environment variables mapping

**Checkpoint**: At this point, User Story 1 should be fully functional - basic .gitleaksignore commenting works

---

## Phase 4: User Story 2 - Handling Different Entry Types (Priority: P2)

**Goal**: Handle specific file:line entries vs wildcard patterns with appropriate comment formatting and links

**Independent Test**: Create a PR with both `config/secrets.yml:42` and `*.env` patterns, verify different comment formats

### Implementation for User Story 2

- [X] T030 [P] [US2] Add IsPattern field detection in ParseGitleaksEntry function in internal/diff/parser.go (detect wildcards in patterns)
- [X] T031 [P] [US2] Add HasLineNumber field parsing in ParseGitleaksEntry function in internal/diff/parser.go (extract line number suffix)
- [X] T032 [US2] Update FileLink method in internal/diff/types.go to handle wildcards (link to parent directory for patterns)
- [X] T033 [US2] Update comment templates to include line number mention when present (templates/addition.md, templates/deletion.md)
- [X] T034 [US2] Update comment templates to indicate wildcard pattern matching (templates/addition.md, templates/deletion.md)
- [X] T035 [US2] Add validation in NewGeneratedComment to ensure proper template selection based on entry type in internal/comment/generator.go

**Checkpoint**: Comments now adapt to entry types (specific files vs patterns)

---

## Phase 5: User Story 3 - Large .gitleaksignore Changes (Priority: P2)

**Goal**: Handle PRs with 50+ .gitleaksignore changes efficiently within 2 minutes

**Independent Test**: Create a PR adding 50+ entries to .gitleaksignore, verify all receive comments within 2 minutes

### Implementation for User Story 3

- [X] T036 [P] [US3] Implement concurrent comment posting with goroutines in internal/github/comments.go (use sync.WaitGroup and semaphore pattern)
- [X] T037 [P] [US3] Add rate limit checking in internal/github/client.go (check remaining API calls before posting)
- [X] T038 [US3] Implement exponential backoff retry logic in internal/github/comments.go (1s, 2s, 4s delays for rate limit errors)
- [X] T039 [US3] Add semaphore to limit concurrent API requests to 5 in internal/github/comments.go (prevent overwhelming GitHub API)
- [X] T040 [US3] Add progress logging for large batches in cmd/pr-diff-comment/main.go (log every 10 comments posted)
- [X] T041 [US3] Implement graceful failure after 3 retry attempts in internal/github/comments.go (log error, continue with remaining comments)

**Checkpoint**: System handles high-volume changes efficiently with proper rate limiting

---

## Phase 6: User Story 4 - Workflow Integration and Triggering (Priority: P1)

**Goal**: GitHub Action triggers automatically on PR events with .gitleaksignore changes

**Independent Test**: Open a PR modifying .gitleaksignore, verify action runs automatically within 1 minute

### Implementation for User Story 4

- [X] T042 [US4] Configure action.yml runs section to use Docker image ('docker', 'Dockerfile')
- [X] T043 [US4] Add PR event triggers in action.yml (pull_request types: opened, synchronize, reopened)
- [X] T044 [US4] Add paths filter for .gitleaksignore in action.yml (only trigger when .gitleaksignore changes)
- [X] T045 [US4] Set required permissions in action.yml (pull-requests: write, contents: read)
- [X] T046 [US4] Complete Dockerfile ENTRYPOINT configuration to call /usr/local/bin/pr-diff-comment binary
- [X] T047 [US4] Add environment variable mapping in main.go (INPUT_GITHUB-TOKEN, INPUT_PR-NUMBER, GITHUB_REPOSITORY, etc.)
- [X] T048 [US4] Implement workflow validation in main.go (check if running in GitHub Actions environment)
- [X] T049 [US4] Add build optimization flags to Dockerfile (-ldflags="-w -s" for binary size reduction)

**Checkpoint**: Action is fully integrated and triggers automatically on PR events

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Documentation, error handling improvements, and validation

- [X] T050 [P] Add comprehensive error messages with actionable guidance in internal/config/config.go validation
- [X] T051 [P] Add structured logging with levels (info, warn, error) throughout all packages
- [X] T052 [P] Create test fixtures for edge cases (empty .gitleaksignore, malformed entries, non-existent files) in tests/fixtures/
- [X] T053 [P] Document action inputs and outputs in README.md with examples
- [X] T054 [P] Add troubleshooting section to README.md (common errors, permission issues, rate limits)
- [X] T055 [P] Add local development instructions to DEVELOPMENT.md (docker build, docker run with environment variables)
- [X] T056 [P] Add example workflow YAML in README.md showing how to use the action
- [X] T057 Verify Docker image size is under 50MB (run `docker images` after build)
- [X] T058 Run quickstart.md validation (manual test following quickstart guide)
- [X] T059 Add .github/workflows/ example for testing the action in this repository
- [X] T060 [P] Add Go module tidy and verification: `go mod tidy && go mod verify`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3-6)**: All depend on Foundational phase completion
  - User Story 1 (P1): Core functionality - MVP
  - User Story 2 (P2): Can start after Foundational, enhances US1
  - User Story 3 (P2): Can start after Foundational, enhances US1
  - User Story 4 (P1): Can start after US1 is testable locally
- **Polish (Phase 7)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories - **THIS IS MVP**
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - Enhances US1 comment formatting
- **User Story 3 (P2)**: Can start after Foundational (Phase 2) - Enhances US1 performance
- **User Story 4 (P1)**: Should start after US1 is working - Integrates US1 into GitHub Actions

**Recommended Order**: Setup → Foundational → US1 (test locally) → US4 (deploy) → US2 → US3 → Polish

### Within Each User Story

- Core parsing (diff, gitleaks entry) before comment generation
- Comment generation before GitHub API posting
- API client implementation before comment posting
- Deduplication before posting
- Error handling and retry logic after basic functionality works

### Parallel Opportunities

- **Phase 1 Setup**: All tasks (T002-T007) can run in parallel except T001 (go mod init)
- **Phase 2 Foundational**: Type definitions (T009-T013, T016) can run in parallel, dependencies (T014-T015) can run in parallel
- **Phase 3 US1**: Parsing (T018-T020) can start in parallel, templates and client work (T021-T024) can run in parallel
- **Phase 4 US2**: Template updates (T033-T034) can run in parallel with parser enhancements (T030-T031)
- **Phase 5 US3**: Concurrency (T036), rate limiting (T037), and retry logic (T038) can be developed in parallel
- **Phase 6 US4**: action.yml updates (T042-T045) and Dockerfile optimization (T049) can run in parallel
- **Phase 7 Polish**: Most documentation tasks (T050-T056) can run in parallel

---

## Parallel Example: User Story 1

```bash
# Launch parsing tasks together:
Task: "Implement ParseGitleaksDiff function in internal/diff/parser.go"
Task: "Implement ParseGitleaksEntry function in internal/diff/parser.go"
Task: "Implement FileLink method on GitleaksEntry in internal/diff/types.go"

# Launch GitHub client and comment generation together:
Task: "Implement template rendering in internal/comment/generator.go"
Task: "Implement CreateReviewComment method in internal/github/comments.go"
Task: "Implement ListReviewComments method in internal/github/comments.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 + User Story 4)

1. Complete Phase 1: Setup → Go project structure ready
2. Complete Phase 2: Foundational → Core types and interfaces defined
3. Complete Phase 3: User Story 1 → Basic commenting works locally
4. **Test locally with sample fixtures**
5. Complete Phase 6: User Story 4 → Deploy as GitHub Action
6. **STOP and VALIDATE**: Test in real PR
7. Deploy/demo MVP

**MVP Scope**: User Story 1 (core .gitleaksignore commenting) + User Story 4 (GitHub Actions integration)

### Incremental Delivery

1. **Foundation**: Setup + Foundational → ~7 tasks
2. **MVP**: Add US1 + US4 → ~23 tasks → Test independently → Deploy (working action!)
3. **Enhancements**: Add US2 → ~6 tasks → Better comment formatting
4. **Performance**: Add US3 → ~6 tasks → Handles large changes
5. **Polish**: Add Phase 7 → ~11 tasks → Production-ready

Each increment adds value without breaking previous functionality.

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together (~7 tasks)
2. Once Foundational is done:
   - Developer A: User Story 1 (core commenting)
   - Developer B: User Story 4 (GitHub Actions integration)
   - Developer C: Comment templates and fixtures
3. After MVP (US1+US4):
   - Developer A: User Story 2 (entry type handling)
   - Developer B: User Story 3 (performance/rate limiting)
   - Developer C: Phase 7 (documentation and polish)

---

## Notes

- **[P] tasks**: Different files, no dependencies - safe to parallelize
- **[Story] label**: Maps task to specific user story for traceability
- **Tests**: Not included (not requested in spec) - focus on implementation
- **MVP**: User Story 1 + User Story 4 provides core value
- **Go conventions**: Use standard Go project layout (cmd/, internal/)
- **Docker**: Multi-stage build keeps image size under 50MB
- **Error handling**: Graceful failures, don't block PR workflow
- **Rate limiting**: Exponential backoff, max 5 concurrent requests
- **Deduplication**: Check existing comments before posting
- **Commit strategy**: Commit after each task or logical group (e.g., complete a package)
- **Testing strategy**: Manual testing with fixtures and real PRs (no automated tests required)

---

## Task Count Summary

- **Phase 1 (Setup)**: 7 tasks
- **Phase 2 (Foundational)**: 10 tasks
- **Phase 3 (US1 - MVP Core)**: 12 tasks
- **Phase 4 (US2 - Entry Types)**: 6 tasks
- **Phase 5 (US3 - Performance)**: 6 tasks
- **Phase 6 (US4 - Integration)**: 8 tasks
- **Phase 7 (Polish)**: 11 tasks

**Total**: 60 tasks

**MVP Scope** (US1 + US4): 37 tasks (Setup + Foundational + US1 + US4)

**Parallel Opportunities**: ~25 tasks marked [P] can run in parallel within their phases
