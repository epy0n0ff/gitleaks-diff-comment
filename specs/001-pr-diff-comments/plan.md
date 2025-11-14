# Implementation Plan: Automated PR Diff Comment Explanations

**Branch**: `001-pr-diff-comments` | **Date**: 2025-11-13 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-pr-diff-comments/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

**Primary Requirement**: Automatically add explanatory comments to GitHub pull requests when `.gitleaksignore` files are modified. The system will comment on the specific diff lines where files are added/removed from the gitleaks ignore list, providing context about which files are being excluded from security scanning.

**Technical Approach**: Custom GitHub Action implemented in Go, packaged as a Docker container. The action parses `.gitleaksignore` diffs, generates contextual comments with file links, and posts them via GitHub API using line-level review comments.

## Technical Context

**Language/Version**: Go 1.25+
**Primary Dependencies**:
- github.com/google/go-github/v57 (GitHub API client)
- golang.org/x/oauth2 (GitHub authentication)
- Docker (multi-stage build for container packaging)
**Storage**: N/A (stateless action, no persistent storage)
**Testing**: Go testing framework (testing package), table-driven tests, testify/assert for assertions
**Target Platform**: GitHub Actions runners (Docker container, linux/amd64 and linux/arm64)
**Project Type**: Single custom action (Docker-based GitHub Action)
**Performance Goals**: Process and comment on PRs within 2 minutes regardless of file count
**Constraints**:
- Must stay within GitHub API rate limits (5000 req/hour authenticated)
- Must work within GitHub Actions timeout (6 hours max, target <2 min)
- Comments only on `.gitleaksignore` file changes (not all changed files)
- Docker image must be under 500MB for reasonable pull times
**Scale/Scope**: Support PRs with unlimited file changes, but only analyze `.gitleaksignore` modifications

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Initial Status**: ✅ PASSED (No project constitution defined - using default best practices)

The constitution file contains only template placeholders with no specific principles defined. This plan follows GitHub Actions and Go best practices:
- Single-purpose action with clear inputs/outputs
- Idempotent operations (can be re-run safely)
- Error handling with appropriate exit codes
- Minimal external dependencies
- Clear documentation in action.yml

**Post-Design Re-evaluation**: ✅ PASSED

Design artifacts reviewed with Go/Docker architecture:
- **research.md**: Docker action patterns, Go GitHub API usage, multi-stage builds
- **data-model.md**: Go struct definitions with JSON tags
- **contracts/**: Action inputs/outputs, Go package interfaces
- **quickstart.md**: Custom action usage in workflows

No constitution violations detected. The design maintains simplicity:
- Single Go binary in Docker container
- Standard GitHub Action interface (action.yml)
- Testable Go packages with clear responsibilities
- No unnecessary frameworks beyond GitHub API client

## Project Structure

### Documentation (this feature)

```text
specs/001-pr-diff-comments/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
# Custom GitHub Action structure
action.yml               # Action metadata and interface
Dockerfile              # Multi-stage Docker build
entrypoint.sh           # Docker entrypoint script

# Go source code
cmd/
└── pr-diff-comment/
    └── main.go         # Entry point, CLI argument parsing

internal/
├── diff/
│   ├── parser.go       # Parse git diff for .gitleaksignore
│   ├── parser_test.go
│   └── types.go        # DiffChange, GitleaksEntry structs
├── comment/
│   ├── generator.go    # Generate comment text with templates
│   ├── generator_test.go
│   └── templates.go    # Markdown templates for additions/deletions
├── github/
│   ├── client.go       # GitHub API client wrapper
│   ├── client_test.go
│   ├── comments.go     # Post review comments, handle deduplication
│   └── types.go        # API request/response types
└── config/
    ├── config.go       # Parse action inputs from environment
    └── config_test.go

# Go module files
go.mod
go.sum

# Testing
tests/
├── fixtures/
│   ├── sample-diff.txt
│   └── sample-gitleaksignore
└── integration/
    └── action_test.go

# Documentation
README.md               # Action usage documentation
DEVELOPMENT.md          # Local development and testing guide
```

**Structure Decision**: Custom GitHub Action with Docker container. Go code follows standard Go project layout with `cmd/` for entry points and `internal/` for private packages. Docker multi-stage build compiles Go binary and packages in minimal Alpine image.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

N/A - No complexity violations. This is a straightforward custom action with minimal dependencies.
