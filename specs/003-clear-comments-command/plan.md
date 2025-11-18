# Implementation Plan: Clear Comments Command

**Branch**: `003-clear-comments-command` | **Date**: 2025-11-17 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/003-clear-comments-command/spec.md`

## Summary

This feature adds a `/clear` command that allows authorized users to remove all bot-created comments from a pull request by mentioning `@github-actions /clear` in a PR comment. The system will:
- Detect case-insensitive `/clear` commands in issue_comment events
- Verify user permissions (PR author, write collaborators, maintainers)
- Identify and delete only bot comments (preserve human comments)
- Provide feedback through workflow logs
- Handle rate limits with exponential backoff retry
- Track execution metrics for monitoring

## Technical Context

**Language/Version**: Go 1.25 (matches existing codebase)
**Primary Dependencies**:
- github.com/google/go-github/v57 (GitHub API client)
- golang.org/x/oauth2 (authentication)

**Storage**: N/A (stateless GitHub Action)
**Testing**: Go testing framework (`go test`), existing test structure in `/tests/`
**Target Platform**: Docker container (GitHub Actions runtime)
**Project Type**: Single Go project (GitHub Action)
**Performance Goals**:
- Clear operation completes in <10 seconds (SC-001)
- Support up to 100 comments without degradation (SC-005)

**Constraints**:
- Must work within GitHub Actions workflow runtime
- Limited to pull_requests:write token permissions
- Subject to GitHub API rate limits

**Scale/Scope**:
- Single PR scope (up to 100 bot comments per PR)
- Concurrent executions handled independently
- Execution count metric tracking

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Status**: No constitution file found or constitution is template-only. Proceeding with standard best practices:
- ✅ Extend existing codebase (no new project complexity)
- ✅ Test-driven approach (unit + integration tests)
- ✅ Observability through logging and metrics (FR-009, FR-012)
- ✅ Clear contracts (webhook input → comment deletion + workflow output)

## Project Structure

### Documentation (this feature)

```text
specs/003-clear-comments-command/
├── spec.md              # Feature specification (complete)
├── plan.md              # This file
├── research.md          # Phase 0 output (pending)
├── data-model.md        # Phase 1 output (pending)
├── quickstart.md        # Phase 1 output (pending)
├── contracts/           # Phase 1 output (pending)
└── tasks.md             # Phase 2 output (/speckit.tasks - not created by plan)
```

### Source Code (repository root)

```text
cmd/
└── gitleaks-diff-comment/
    └── main.go          # Entry point - add command routing

internal/
├── config/
│   └── config.go        # Add command field to Config struct
├── comment/
│   ├── generator.go     # Existing comment generation
│   ├── types.go         # Existing comment types
│   └── templates/       # Existing templates
├── diff/
│   ├── parser.go        # Existing diff parsing
│   └── types.go         # Existing diff types
├── github/
│   ├── client.go        # Add ClearComments method
│   └── comments.go      # Add comment deletion + filtering logic
└── commands/            # NEW: Command handling
    ├── clear.go         # Clear command implementation
    └── detector.go      # Command detection logic

tests/
├── integration/
│   └── clear_command_test.go  # NEW: Clear command integration tests
├── unit/
│   └── commands/              # NEW: Unit tests for command logic
└── fixtures/
    └── clear_test_data.json   # NEW: Test fixtures

.github/workflows/
└── clear-command.yml    # NEW: Workflow for issue_comment events
```

**Structure Decision**: Extending existing single Go project structure. Adding new `internal/commands/` package for command handling logic separate from existing comment generation logic. This maintains clear separation of concerns while integrating with existing GitHub API client and configuration infrastructure.

## Complexity Tracking

No constitutional violations. This feature extends the existing GitHub Action with minimal complexity:
- Reuses existing GitHub API client infrastructure
- Follows existing patterns (config, internal packages, testing)
- No new external dependencies required
- Single command addition to existing action
