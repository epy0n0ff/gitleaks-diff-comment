# Implementation Plan: GitHub Enterprise Server Support

**Branch**: `002-github-enterprise-support` | **Date**: 2025-11-14 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/002-github-enterprise-support/spec.md`
**User Guidance**: "actionsのwithの引数でgh-hostみたいな感じで渡されたらそのホストを使うようにしてください。"

## Summary

Add GitHub Enterprise Server support to the gitleaks-diff-comment GitHub Action, enabling enterprise users to use the action with self-hosted GitHub instances by providing a custom API host via the `gh-host` input parameter. When specified, the action will construct the API base URL using the provided host (e.g., `https://{gh-host}/api/v3`). When not specified, it defaults to GitHub.com's public API, ensuring backward compatibility.

**Primary Goal**: Enable enterprise adoption by supporting custom GitHub API endpoints (version 3.14+) without breaking existing GitHub.com workflows.

**Technical Approach**:
- Add optional `gh-host` input parameter to action.yml
- Modify GitHub client initialization to accept and use custom API base URL
- Implement URL validation and error handling for enterprise connectivity
- Preserve all existing functionality when `gh-host` is not provided

## Technical Context

**Language/Version**: Go 1.25
**Primary Dependencies**:
- `github.com/google/go-github/v57` (GitHub API client)
- `golang.org/x/oauth2` (OAuth2 authentication)

**Storage**: N/A (stateless action)
**Testing**: Go standard testing (`go test`), integration tests with mock GitHub API
**Target Platform**: Docker container (alpine:3.22 runtime)
**Project Type**: Single project (GitHub Action with Go CLI backend)
**Performance Goals**:
- API endpoint validation: <2 seconds
- Enterprise API connection: <5 seconds (excluding PR analysis)

**Constraints**:
- Must maintain 100% backward compatibility with existing GitHub.com usage
- Must support GitHub Enterprise Server 3.14+ API conventions
- Must handle both HTTP and HTTPS protocols
- Must provide clear, actionable error messages for configuration issues

**Scale/Scope**:
- Single new input parameter (`gh-host`)
- Modifications to 3-4 existing files (action.yml, config, github client)
- Additional validation logic and error handling
- No new API operations (reuses all existing comment operations)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

**Constitution Status**: Project does not have a ratified constitution (template values present in `.specify/memory/constitution.md`).

**Default Standards Applied**:
- ✅ **Backward Compatibility**: No breaking changes - existing workflows continue to work without modification
- ✅ **Simplicity**: Minimal code changes - single new input parameter, URL construction logic, validation
- ✅ **Testability**: Changes are independently testable with unit tests (URL validation, client initialization) and integration tests (mock enterprise API)
- ✅ **Error Handling**: Clear error messages for common failure scenarios (unreachable host, invalid URL format, authentication failures)
- ✅ **Documentation**: Configuration examples in README for enterprise setup

**No Constitution Violations**: This is an additive feature with no architectural complexity added.

## Project Structure

### Documentation (this feature)

```text
specs/002-github-enterprise-support/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output: go-github enterprise support research
├── data-model.md        # Phase 1 output: Configuration entities
├── quickstart.md        # Phase 1 output: Enterprise setup guide
├── contracts/           # Phase 1 output: Configuration schema
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
# Existing structure (to be modified)
cmd/
└── gitleaks-diff-comment/
    └── main.go                    # [MODIFY] Pass gh-host to config

internal/
├── config/
│   └── config.go                  # [MODIFY] Add GHHost field, parse INPUT_GH-HOST
├── github/
│   └── client.go                  # [MODIFY] Accept baseURL parameter, construct enterprise API URL
├── comment/                       # [NO CHANGES]
│   ├── generator.go
│   ├── templates/
│   └── types.go
└── diff/                          # [NO CHANGES]
    ├── parser.go
    └── types.go

tests/
├── integration/
│   └── enterprise_test.go         # [NEW] Integration tests for enterprise connectivity
└── unit/
    ├── config_test.go             # [MODIFY] Add tests for gh-host parsing
    └── github_client_test.go      # [MODIFY] Add tests for custom base URL

# Root files
action.yml                          # [MODIFY] Add gh-host input
Dockerfile                          # [NO CHANGES]
README.md                           # [MODIFY] Add enterprise configuration examples
```

**Structure Decision**: Single project structure is appropriate. This is a configuration enhancement, not a new service. All changes are within existing packages (config, github) plus test additions. No new architectural components needed.

## Complexity Tracking

> **No Constitution violations to justify** - this feature adds minimal complexity and maintains existing architecture.

---

## Phase 0: Research & Design Decisions

### Research Tasks

1. **RQ-001: go-github Enterprise Support**
   - **Question**: How does `github.com/google/go-github/v57` support custom API base URLs for GitHub Enterprise Server?
   - **Key Points to Investigate**:
     - Does the library have built-in support for custom base URLs?
     - How to properly initialize a client with a custom endpoint?
     - Are there any version-specific considerations for GHES 3.14+?
     - Does it automatically handle `/api/v3` path or must we provide full URL?
   - **Expected Outcome**: Clear pattern for creating enterprise-compatible GitHub clients

2. **RQ-002: URL Validation Best Practices**
   - **Question**: What validation should be performed on user-provided GitHub host input?
   - **Key Points to Investigate**:
     - Format validation (hostname vs full URL)
     - Protocol handling (HTTP vs HTTPS)
     - Common enterprise URL patterns (with/without port, path prefixes)
     - Error messages for invalid inputs
   - **Expected Outcome**: Validation logic specification and error message templates

3. **RQ-003: OAuth2 Enterprise Compatibility**
   - **Question**: Does `golang.org/x/oauth2` work with enterprise GitHub instances without modifications?
   - **Key Points to Investigate**:
     - Does oauth2 client respect custom base URLs automatically?
     - Any special configuration needed for enterprise auth endpoints?
     - Token format differences between GitHub.com and GHES?
   - **Expected Outcome**: Confirmation of oauth2 compatibility or required adaptations

4. **RQ-004: GitHub Actions Environment Variables**
   - **Question**: Are there existing GitHub Actions environment variables that indicate enterprise environments?
   - **Key Points to Investigate**:
     - `GITHUB_API_URL` or similar standard variables
     - `GITHUB_SERVER_URL` usage patterns
     - Best practices for detecting vs explicit configuration
   - **Expected Outcome**: Decision on whether to support auto-detection or require explicit configuration

5. **RQ-005: SSL/TLS Certificate Handling**
   - **Question**: How should self-signed or custom enterprise certificates be handled?
   - **Key Points to Investigate**:
     - Default Go TLS verification behavior
     - Options for disabling verification (and security implications)
     - Best practices for enterprise certificate trust
   - **Expected Outcome**: Certificate handling strategy (strict by default, opt-in skip verification?)

### Design Decisions

These will be documented in `research.md` after investigation:

- **DD-001**: GitHub client initialization pattern with custom base URL
- **DD-002**: Input parameter naming (`gh-host` vs `github-host` vs `api-url`)
- **DD-003**: URL construction logic (full URL vs hostname + automatic `/api/v3` append)
- **DD-004**: Validation error message format and guidance
- **DD-005**: SSL certificate verification strategy
- **DD-006**: Backward compatibility testing approach

---

## Phase 1: Data Model & Contracts

### Data Model Changes

**Modified Entity: Configuration** (`internal/config/config.go`)

```go
type Config struct {
    // ... existing fields ...
    GitHubToken string
    PRNumber    int
    Repository  string
    CommitSHA   string
    BaseRef     string
    HeadRef     string
    Workspace   string
    CommentMode string
    Debug       bool

    // NEW: GitHub host for enterprise support
    GHHost      string  // Optional: Custom GitHub host (e.g., "github.company.com")
}
```

**New Entity: API Configuration** (conceptual - may be embedded in client)

```go
type APIConfig struct {
    BaseURL     string  // Full API base URL (e.g., "https://api.github.com" or "https://github.company.com/api/v3")
    Host        string  // Host portion (e.g., "github.company.com")
    Protocol    string  // "https" or "http"
    IsEnterprise bool   // true if using custom host
}
```

### Contract: Action Configuration

**File**: `specs/002-github-enterprise-support/contracts/action-config.yml`

```yaml
# GitHub Action Input Schema
inputs:
  gh-host:
    description: 'GitHub Enterprise Server hostname (e.g., github.company.com). Leave empty for GitHub.com.'
    required: false
    default: ''
    examples:
      - 'github.company.com'
      - 'github.enterprise.internal'
      - 'github.mycompany.com:8443'

validation:
  format: 'hostname or hostname:port'
  protocols: ['https', 'http']
  no_path: true  # Path will be auto-appended (/api/v3)
  no_protocol: true  # Protocol prefix should NOT be included by user

error_messages:
  invalid_format: 'Invalid gh-host format. Expected: "hostname" or "hostname:port" (e.g., "github.company.com" or "github.company.com:8443")'
  unreachable: 'Cannot connect to GitHub Enterprise Server at {host}. Please verify the hostname is correct and the server is reachable.'
  auth_failed: 'Authentication failed for GitHub Enterprise Server at {host}. Please check your token has the required permissions.'
```

### Contract: API Client Interface

**File**: `specs/002-github-enterprise-support/contracts/github-client.go`

```go
// GitHub Client Interface (no changes to operations)
type Client interface {
    CreateReviewComment(ctx context.Context, req *PostCommentRequest) (*PostCommentResponse, error)
    UpdateReviewComment(ctx context.Context, req *UpdateCommentRequest) (*PostCommentResponse, error)
    ListReviewComments(ctx context.Context) ([]*ExistingComment, error)
    CreateIssueComment(ctx context.Context, body string) (*PostCommentResponse, error)
    CheckRateLimit(ctx context.Context) (int, error)
}

// NewClient signature change
// OLD: NewClient(token, owner, repo string, prNumber int) (Client, error)
// NEW: NewClient(token, owner, repo string, prNumber int, baseURL string) (Client, error)
//      - baseURL: Optional custom API base URL (empty string = default to GitHub.com)
//      - Returns error if baseURL is invalid or unreachable
```

---

## Phase 2: Implementation Tasks

**Note**: Detailed tasks will be generated by `/speckit.tasks` command. This section provides high-level task categories:

### Task Categories

1. **Configuration Enhancement** (FR-001, FR-002, FR-003)
   - Add `gh-host` input to action.yml
   - Add `GHHost` field to Config struct
   - Parse `INPUT_GH-HOST` environment variable
   - Implement URL validation logic
   - Add configuration tests

2. **GitHub Client Modification** (FR-004, FR-005, FR-006)
   - Modify `NewClient` to accept optional base URL
   - Implement URL construction logic (host → full API URL)
   - Update client initialization in main.go
   - Add client creation tests

3. **Error Handling** (FR-008)
   - Implement connectivity validation
   - Add clear error messages for common failures
   - Distinguish authentication vs network vs configuration errors
   - Add error handling tests

4. **SSL/TLS Support** (FR-007, FR-012)
   - Configure TLS certificate verification
   - Support HTTP and HTTPS protocols
   - Add certificate validation tests

5. **Backward Compatibility** (FR-009, SC-006)
   - Ensure empty `gh-host` defaults to GitHub.com
   - Regression testing for GitHub.com workflows
   - Update existing tests to verify no breaking changes

6. **Documentation** (SC-001, SC-007)
   - Add enterprise setup guide to README
   - Provide workflow configuration examples
   - Document common troubleshooting scenarios
   - Create quickstart.md for enterprise users

7. **Rate Limit Support** (FR-010)
   - Verify existing rate limit logic works with enterprise
   - Add debug logging for enterprise rate limits
   - Add rate limit tests

---

## Success Validation

### Validation Criteria (from Success Criteria)

- ✅ **SC-001**: Enterprise users can add `gh-host: github.company.com` to workflow and action works without code changes
- ✅ **SC-002**: Connection to enterprise API completes in <5 seconds
- ✅ **SC-003**: All existing operations (post, update, list comments) work identically on enterprise
- ✅ **SC-004**: Invalid `gh-host` values produce clear error messages within 2 seconds
- ✅ **SC-005**: Supports GHES 3.14+ (verified via API version detection)
- ✅ **SC-006**: Existing GitHub.com workflows work unchanged (regression tests pass)
- ✅ **SC-007**: Enterprise setup documented and completable in <10 minutes

### Test Strategy

**Unit Tests**:
- Configuration parsing with/without `gh-host`
- URL validation for valid/invalid formats
- API base URL construction
- Error message generation

**Integration Tests**:
- Mock enterprise API server
- End-to-end workflow with custom host
- Authentication with enterprise token
- Rate limit header parsing

**Regression Tests**:
- All existing tests pass with empty `gh-host` (GitHub.com default)
- No changes to comment posting behavior
- Backward compatibility verification

---

## Risk Assessment

### Technical Risks

| Risk | Impact | Mitigation |
|------|--------|-----------|
| go-github library doesn't support custom base URLs cleanly | High | Research Phase 0 will confirm; fallback to HTTP client customization if needed |
| Enterprise API differs from GitHub.com in subtle ways | Medium | Target GHES 3.14+ ensures modern API compatibility; comprehensive integration testing |
| SSL certificate issues with self-signed certs | Medium | Clear documentation on certificate requirements; consider optional verification skip (with warnings) |
| URL validation too strict/lenient | Low | Iterative testing with common enterprise URL patterns; clear error messages guide users |

### Operational Risks

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Users provide incorrect `gh-host` format | Low | Validation with helpful error messages and examples in documentation |
| Enterprise instance unreachable from Actions runner | Medium | Clear error messages; documentation covers network/firewall requirements |
| Token permissions insufficient for enterprise | Low | Error messages indicate missing permissions; documentation lists required scopes |

---

## Dependencies & Blockers

### External Dependencies

1. **go-github library v57** - Must support custom base URL configuration
   - **Status**: To be confirmed in Phase 0 research
   - **Blocker Risk**: Low (library is designed for enterprise support)

2. **GitHub Enterprise Server 3.14+** - Must implement standard API v3
   - **Status**: Specified in requirements
   - **Blocker Risk**: Low (GHES follows GitHub API standards)

### Internal Dependencies

1. **Existing config package** - Foundation for new `gh-host` parameter
   - **Status**: Ready (well-structured config.go)
   - **Blocker Risk**: None

2. **Existing GitHub client** - Must be modifiable to accept custom base URL
   - **Status**: Ready (clean client abstraction in client.go)
   - **Blocker Risk**: None

---

## Next Steps

1. ✅ **Phase 0 Complete**: After executing `/speckit.plan`, proceed with research tasks (RQ-001 through RQ-005)
2. ⏳ **Phase 1 Pending**: Generate detailed data-model.md and contracts based on research findings
3. ⏳ **Phase 2 Pending**: Execute `/speckit.tasks` to generate implementation task list

**Current Phase**: Ready to begin Phase 0 Research
