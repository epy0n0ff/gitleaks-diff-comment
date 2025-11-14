# GitHub Client Interface Contract

**Feature**: `002-github-enterprise-support`
**Component**: `internal/github/client.go`
**Purpose**: Define the contract for GitHub client initialization with enterprise support

---

## Interface: Client

**Location**: `internal/github/client.go`
**Purpose**: Defines GitHub API operations for PR comments

### Methods (Unchanged)

All existing Client interface methods remain unchanged. Enterprise support is transparent to API operations.

```go
type Client interface {
    // CreateReviewComment posts a line-level review comment on a PR
    CreateReviewComment(ctx context.Context, req *PostCommentRequest) (*PostCommentResponse, error)

    // UpdateReviewComment updates an existing review comment
    UpdateReviewComment(ctx context.Context, req *UpdateCommentRequest) (*PostCommentResponse, error)

    // ListReviewComments fetches all review comments for a PR
    ListReviewComments(ctx context.Context) ([]*ExistingComment, error)

    // CreateIssueComment posts a PR-level comment (fallback)
    CreateIssueComment(ctx context.Context, body string) (*PostCommentResponse, error)

    // CheckRateLimit returns remaining API calls
    CheckRateLimit(ctx context.Context) (int, error)
}
```

**Contract Guarantee**: All methods work identically on GitHub.com and GitHub Enterprise Server.

---

## Function: NewClient

**Signature Change**: Add optional `ghHost` parameter

### Current Signature

```go
func NewClient(token, owner, repo string, prNumber int) (Client, error)
```

### New Signature

```go
func NewClient(token, owner, repo string, prNumber int, ghHost string) (Client, error)
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `token` | `string` | Yes | GitHub API authentication token (PAT or installation token) |
| `owner` | `string` | Yes | Repository owner (username or organization) |
| `repo` | `string` | Yes | Repository name |
| `prNumber` | `int` | Yes | Pull request number (must be positive) |
| `ghHost` | `string` | No | GitHub Enterprise Server hostname (empty = GitHub.com) |

### ghHost Parameter Specification

**Type**: `string`
**Default**: `""` (empty string)
**Format**: Hostname or hostname:port

**Behavior**:
- **When empty (`""`)**: Uses GitHub.com public API (`https://api.github.com`)
- **When provided**: Uses GitHub Enterprise Server API (`https://{ghHost}/api/v3/`)

**Expected Input**:
- ✅ `"github.company.com"` → `https://github.company.com/api/v3/`
- ✅ `"github.internal:8443"` → `https://github.internal:8443/api/v3/`
- ✅ `""` → `https://api.github.com` (GitHub.com)
- ❌ `"https://github.company.com"` → Error (includes protocol)
- ❌ `"github.company.com/api/v3"` → Error (includes path)

**Validation**: Assumes `ghHost` has already been validated by `config.Validate()` before being passed to `NewClient`.

### Return Values

**Success**: `(Client, nil)`
- Client ready for API operations
- Configured for either GitHub.com or enterprise endpoint

**Failure**: `(nil, error)`
- Error indicates client initialization failed
- Error message includes context (hostname, reason)

### Error Conditions

| Error Type | Condition | Example Message |
|------------|-----------|-----------------|
| Validation Error | Required parameter missing/invalid | `"GitHub token is required"` |
| Validation Error | Invalid owner/repo/prNumber | `"PR number must be positive"` |
| URL Error | Invalid ghHost format (defensive check) | `"failed to create GitHub Enterprise client for {host}: {error}"` |
| Library Error | go-github WithEnterpriseURLs fails | Wrapped library error |

---

## Implementation Contract

### Client Initialization Logic

```go
func NewClient(token, owner, repo string, prNumber int, ghHost string) (Client, error) {
    // 1. Validate required parameters
    if token == "" {
        return nil, errors.New("GitHub token is required")
    }
    if owner == "" {
        return nil, errors.New("owner is required")
    }
    if repo == "" {
        return nil, errors.New("repo is required")
    }
    if prNumber <= 0 {
        return nil, errors.New("PR number must be positive")
    }

    // 2. Create OAuth2 HTTP client
    ctx := context.Background()
    ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
    tc := oauth2.NewClient(ctx, ts)

    // 3. Create GitHub client (enterprise or default)
    var ghClient *github.Client
    var err error

    if ghHost != "" {
        // GitHub Enterprise Server
        baseURL := "https://" + ghHost
        uploadURL := "https://" + ghHost

        ghClient, err = github.NewClient(tc).WithEnterpriseURLs(baseURL, uploadURL)
        if err != nil {
            return nil, fmt.Errorf("failed to create GitHub Enterprise client for %s: %w", ghHost, err)
        }
    } else {
        // GitHub.com (default)
        ghClient = github.NewClient(tc)
    }

    // 4. Return wrapped client
    return &ClientImpl{
        client:   ghClient,
        owner:    owner,
        repo:     repo,
        prNumber: prNumber,
    }, nil
}
```

### URL Construction Contract

**Responsibility**: Delegated to go-github library's `WithEnterpriseURLs` method

**Input**: `ghHost` (hostname or hostname:port)
**Process**:
1. Prepend `https://` protocol
2. Pass to `WithEnterpriseURLs(baseURL, uploadURL)`
3. Library appends `/api/v3/` and `/api/uploads/` automatically

**Output**: Fully constructed API URLs
- Base API: `https://{ghHost}/api/v3/`
- Upload API: `https://{ghHost}/api/uploads/`

**Edge Cases Handled by Library**:
- Missing trailing slash: Added automatically
- Hostname contains "api.": Path not double-appended
- Port numbers: Preserved in URL

---

## Backward Compatibility Contract

### Guarantees

1. **Signature Change is Additive**
   - New parameter (`ghHost`) added at end
   - Callers can continue using positional arguments by passing `""`
   - Example: `NewClient(token, owner, repo, prNumber, "")` = old behavior

2. **Default Behavior Unchanged**
   - When `ghHost == ""`, behavior identical to previous version
   - All existing tests pass without modification
   - GitHub.com workflows work unchanged

3. **No Breaking Changes to Interface**
   - `Client` interface methods unchanged
   - Return types unchanged
   - Error behavior unchanged (except new ghHost-related errors)

### Migration Path

**Phase 1** (Current): All callers pass empty string
```go
// Existing call site (to be updated)
client, err := github.NewClient(token, owner, repo, prNumber)

// Updated call site (temporary - pass empty string)
client, err := github.NewClient(token, owner, repo, prNumber, "")
```

**Phase 2** (After implementation): Pass config.GHHost
```go
// Final call site (with enterprise support)
client, err := github.NewClient(token, owner, repo, prNumber, cfg.GHHost)
```

---

## Testing Contract

### Unit Tests Required

1. **Default GitHub.com Behavior**
   ```go
   func TestNewClient_GitHubCom(t *testing.T) {
       client, err := NewClient("token", "owner", "repo", 1, "")
       // Verify: Client created successfully
       // Verify: Uses default GitHub.com API
   }
   ```

2. **Enterprise Client Creation**
   ```go
   func TestNewClient_Enterprise(t *testing.T) {
       client, err := NewClient("token", "owner", "repo", 1, "github.company.com")
       // Verify: Client created successfully
       // Verify: Uses enterprise API URL
   }
   ```

3. **Enterprise with Port**
   ```go
   func TestNewClient_EnterpriseWithPort(t *testing.T) {
       client, err := NewClient("token", "owner", "repo", 1, "github.company.com:8443")
       // Verify: Client created successfully
       // Verify: Port preserved in API URL
   }
   ```

4. **Validation Errors**
   ```go
   func TestNewClient_ValidationErrors(t *testing.T) {
       // Test: Empty token
       _, err := NewClient("", "owner", "repo", 1, "")
       // Expect: Error "GitHub token is required"

       // Test: Invalid PR number
       _, err = NewClient("token", "owner", "repo", -1, "")
       // Expect: Error "PR number must be positive"
   }
   ```

### Integration Tests Required

1. **End-to-End Enterprise Connection**
   - Mock GitHub Enterprise Server API
   - Create client with custom ghHost
   - Execute API operation (e.g., list comments)
   - Verify request sent to correct enterprise URL

2. **Rate Limit Check (Enterprise)**
   - Mock enterprise rate limit endpoint
   - Verify rate limit check uses enterprise URL
   - Verify rate limit headers parsed correctly

3. **Comment Operations (Enterprise)**
   - Mock enterprise PR comment endpoints
   - Test create, update, list operations
   - Verify all operations use enterprise URL

---

## Performance Contract

### Initialization Time

**Target**: Client initialization completes in <1 second

**Breakdown**:
- OAuth2 client creation: <100ms
- go-github client creation: <100ms
- WithEnterpriseURLs URL validation: <50ms
- Total: <250ms (well below 1 second target)

### API Call Performance

**Target**: First API call completes in <5 seconds (per SC-002)

**Factors**:
- Network latency to enterprise server
- TLS handshake
- API request/response time

**Not Controlled by Client**: Network and server performance are external factors.

---

## Security Contract

### TLS Certificate Validation

**Default Behavior**: Strict certificate validation (Go default)
- Valid certificates: Connection succeeds
- Self-signed certificates: Connection fails with clear error
- Expired certificates: Connection fails with clear error

**Error Messages**:
- `x509: certificate signed by unknown authority` → Self-signed certificate
- `x509: certificate has expired` → Expired certificate
- `x509: certificate is valid for X, not Y` → Hostname mismatch

**User Responsibility**: Ensure enterprise certificate is trusted by Actions runner.

### Token Security

- Tokens passed via environment variable (GitHub Actions secret)
- Tokens never logged or exposed in error messages
- OAuth2 library handles secure token injection in requests

---

## Monitoring and Debugging

### Debug Logging

When `cfg.Debug == true`:
- Log ghHost value being used
- Log computed API base URL (without sensitive data)
- Log client initialization success/failure

Example:
```
GitHub Enterprise Server: github.company.com
API Base URL: https://github.company.com/api/v3/
Client initialized successfully
```

### Error Diagnostics

All errors include context:
- Hostname being accessed (if enterprise)
- Operation that failed
- Underlying error message

Example:
```
failed to create GitHub Enterprise client for github.company.com: invalid URL escape "%zzz"
```

---

## Version Compatibility

### go-github Library

**Minimum Version**: v57.0.0 (current)
**Method Used**: `WithEnterpriseURLs(baseURL, uploadURL string) (*Client, error)`
**Status**: Stable, recommended approach as of go-github v50+

### GitHub Enterprise Server

**Minimum Version**: 3.14+ (per requirements)
**API Version**: GitHub REST API v3
**Compatibility**: Standard API v3 endpoints, no version-specific code

---

## Summary

### Contract Guarantees

1. ✅ **Backward Compatible**: Empty `ghHost` maintains existing behavior
2. ✅ **Enterprise Support**: Non-empty `ghHost` enables enterprise connectivity
3. ✅ **Transparent to Callers**: Client interface unchanged, enterprise support is configuration
4. ✅ **Error Clarity**: All errors provide context and actionable guidance
5. ✅ **Security**: TLS validation enforced, tokens handled securely
6. ✅ **Performance**: Client initialization <1s, API calls inherit network characteristics

### Breaking Changes

**None** - This is an additive change. New parameter added, default behavior preserved.
