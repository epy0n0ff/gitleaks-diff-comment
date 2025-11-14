# Research: GitHub Enterprise Server Support

**Date**: 2025-11-14
**Feature**: GitHub Enterprise Server Support (`002-github-enterprise-support`)
**Phase**: 0 - Research & Design Decisions

## Overview

This document captures research findings for adding GitHub Enterprise Server support to the gitleaks-diff-comment action. The goal is to enable users to specify a custom GitHub host via the `gh-host` input parameter, which the action will use to connect to self-hosted GitHub instances.

---

## RQ-001: go-github Enterprise Support

### Question
How does `github.com/google/go-github/v57` support custom API base URLs for GitHub Enterprise Server?

### Findings

**Built-in Support: YES** ✅

The go-github v57 library has comprehensive built-in support for GitHub Enterprise Server through the `WithEnterpriseURLs` method.

**Two Approaches**:
1. **Modern (Recommended)**: `github.NewClient(httpClient).WithEnterpriseURLs(baseURL, uploadURL)`
2. **Legacy (Deprecated)**: `github.NewEnterpriseClient(baseURL, uploadURL, httpClient)` - wrapper around modern approach

**Key Features**:
- Automatic path suffix handling: Adds `/api/v3/` if not present
- Automatic upload URL handling: Adds `/api/uploads/` if not present
- Smart detection: Won't double-append if hostname contains "api." or ".api."
- Trailing slash handling: Automatically adds if missing

### Code Pattern

```go
import (
    "github.com/google/go-github/v57/github"
    "golang.org/x/oauth2"
)

func NewClient(token string, ghHost string) (*github.Client, error) {
    ctx := context.Background()
    ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
    tc := oauth2.NewClient(ctx, ts)

    if ghHost != "" {
        // Enterprise: user provides "github.company.com"
        baseURL := "https://" + ghHost
        uploadURL := "https://" + ghHost

        // Library automatically adds /api/v3/ and /api/uploads/
        return github.NewClient(tc).WithEnterpriseURLs(baseURL, uploadURL)
    }

    // Default: GitHub.com
    return github.NewClient(tc), nil
}
```

### Critical Finding: URL Format

**From go-github Issue #958**: The most common mistake is omitting `/api/v3/` suffix, causing "HTTP 406 Not Acceptable" errors.

**Solution**: Let `WithEnterpriseURLs` handle path construction automatically.

**User Input Format**:
- ✅ Accept: `github.company.com` or `github.company.com:8443`
- ❌ Reject: `https://github.company.com` (protocol prefix)
- ❌ Reject: `github.company.com/api/v3` (path suffix)

**Library Behavior**:
- Input: `https://github.company.com` → Output: `https://github.company.com/api/v3/`
- Input: `https://github.company.com/api/v3` → Output: `https://github.company.com/api/v3/`
- Input: `https://api.github.com` → Output: `https://api.github.com/` (no /api/v3 added due to "api." detection)

### Decision: DD-001

**Use `WithEnterpriseURLs` method** with automatic path handling.

**Implementation**:
1. Accept hostname-only input from user (`gh-host: github.company.com`)
2. Prepend `https://` protocol
3. Pass to `WithEnterpriseURLs` - library handles `/api/v3/` path
4. No manual URL path manipulation

**Rationale**: Leverages library's tested logic, avoids edge cases (ports, subdomains, trailing slashes).

---

## RQ-002: URL Validation Best Practices

### Question
What validation should be performed on user-provided GitHub host input?

### Findings

**Validation Requirements**:

1. **Format Validation**
   - Accept: hostname or hostname:port format
   - Reject: Full URLs with protocol (`https://`)
   - Reject: Paths included (`/api/v3`)
   - Pattern: `^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*(:[0-9]{1,5})?$`

2. **Protocol Handling**
   - Default to HTTPS (append `https://` prefix)
   - Support HTTP via explicit configuration (future: `gh-insecure` flag)
   - Reject mixed protocols in same input

3. **Common Patterns**
   - Simple hostname: `github.company.com`
   - With subdomain: `github.enterprise.internal`
   - With port: `github.company.com:8443`
   - Internal network: `github.local`

4. **Error Messages**
   - Invalid format: "Invalid gh-host format. Expected: 'hostname' or 'hostname:port' (e.g., 'github.company.com' or 'github.company.com:8443'). Do not include 'https://' prefix or '/api/v3' path."
   - Unreachable: "Cannot connect to GitHub Enterprise Server at {host}. Verify hostname is correct and server is reachable from Actions runner."
   - Authentication failure: "Authentication failed for {host}. Check token permissions: repo (read), pull_requests (write)."

### Decision: DD-002

**Input Parameter Naming: `gh-host`** (as requested by user)

**Alternatives Considered**:
- `github-host` - more explicit but verbose
- `api-url` - too technical, implies full URL
- `enterprise-url` - confusing (GitHub Enterprise Cloud uses different URL)

**Rationale**: Short, clear, follows GitHub Actions naming conventions (`github-token`, `github-server-url`).

### Decision: DD-003

**URL Construction: Hostname + Automatic Protocol/Path**

**Approach**:
1. User provides: `github.company.com`
2. Validation: Check hostname format
3. Construction: `https://` + hostname → `https://github.company.com`
4. Library: Adds `/api/v3/` → `https://github.company.com/api/v3/`

**Rationale**:
- Simple user experience (just hostname)
- Leverages library's tested path handling
- Clear separation: validation (config layer) vs construction (client layer)

### Decision: DD-004

**Validation Error Messages: Actionable Guidance**

**Format**:
```
Error: [Problem Statement]
→ Action: [What user should do]
→ Example: [Correct format example]
```

**Examples**:
```
Error: Invalid gh-host format 'https://github.company.com'
→ Action: Provide hostname only, without 'https://' prefix
→ Example: gh-host: github.company.com
```

**Rationale**: Errors should guide users to fix configuration quickly (<2 seconds per SC-004).

---

## RQ-003: OAuth2 Enterprise Compatibility

### Question
Does `golang.org/x/oauth2` work with enterprise GitHub instances without modifications?

### Findings

**YES** ✅ - OAuth2 works seamlessly with GitHub Enterprise.

**How It Works**:
1. `oauth2.NewClient(ctx, tokenSource)` creates HTTP client with token injection
2. Token is added to `Authorization: Bearer {token}` header for all requests
3. Once GitHub client is initialized with `WithEnterpriseURLs`, all API calls automatically use custom base URL
4. No special OAuth2 configuration needed for enterprise endpoints

**Token Compatibility**:
- Personal Access Tokens (PAT): Work identically on GHES and GitHub.com
- GitHub App Installation Tokens: Compatible with GHES 3.14+
- OAuth App Tokens: Supported (though not primary use case for Actions)

**Required Token Scopes** (same for GitHub.com and GHES):
- `repo` or `public_repo` (read access)
- `pull_requests:write` (comment posting)

**Authentication Flow**:
```
User provides token → oauth2.StaticTokenSource → oauth2.NewClient
→ github.NewClient(httpClient) → WithEnterpriseURLs
→ All API calls use enterprise URL with token header
```

### Decision: DD-005

**No OAuth2 Modifications Needed**

**Implementation**: Use existing OAuth2 setup unchanged. Enterprise URL configuration is orthogonal to authentication.

**Rationale**: OAuth2 is protocol-level, works with any HTTPS endpoint.

---

## RQ-004: GitHub Actions Environment Variables

### Question
Are there existing GitHub Actions environment variables that indicate enterprise environments?

### Findings

**Standard GitHub Actions Variables**:

1. **`GITHUB_SERVER_URL`** (Available in all Actions runners)
   - GitHub.com: `https://github.com`
   - GHES: `https://github.company.com` (custom domain)
   - Set automatically by Actions runtime

2. **`GITHUB_API_URL`** (Available in all Actions runners)
   - GitHub.com: `https://api.github.com`
   - GHES: `https://github.company.com/api/v3` (note: includes path!)
   - Set automatically by Actions runtime

3. **`GITHUB_REPOSITORY`** (Available in all Actions runners)
   - Format: `owner/repo` (same on GitHub.com and GHES)

### Auto-Detection Possibility

**Option A: Use `GITHUB_API_URL` directly**
```yaml
# User doesn't specify gh-host, we detect from environment
- uses: actions/gitleaks-diff-comment@v1
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    # gh-host auto-detected from GITHUB_API_URL
```

**Option B: Require explicit `gh-host` parameter**
```yaml
# User must explicitly provide gh-host
- uses: actions/gitleaks-diff-comment@v1
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    gh-host: github.company.com  # Required for GHES
```

### Decision: DD-006

**Use Explicit `gh-host` Parameter (No Auto-Detection)**

**Rationale**:
1. **Clarity**: Explicit configuration makes enterprise usage obvious in workflow files
2. **Simplicity**: No environment variable parsing logic, fewer edge cases
3. **Debugging**: Users can see exactly what they configured
4. **Flexibility**: Allows testing against different hosts without changing environment
5. **Backward Compatibility**: Empty `gh-host` = GitHub.com (clear default behavior)

**Future Enhancement**: Could add auto-detection later if user feedback shows it's valuable.

**Alternative Rejected**: Auto-detection from `GITHUB_API_URL` has complexity:
- URL includes `/api/v3` path - need to strip it
- Ambiguous whether user wants override or detection
- Harder to test (environment-dependent behavior)

---

## RQ-005: SSL/TLS Certificate Handling

### Question
How should self-signed or custom enterprise certificates be handled?

### Findings

**Default Go TLS Behavior**:
- Go's HTTP client validates SSL certificates against system trust store
- Self-signed certificates will fail with: `x509: certificate signed by unknown authority`
- Invalid hostname certificates fail with: `x509: certificate is valid for X, not Y`

**Enterprise Certificate Scenarios**:

1. **Valid Enterprise Certificate** (Signed by trusted CA)
   - ✅ Works out of box - no configuration needed
   - Most enterprise environments use internal CAs properly configured

2. **Self-Signed Certificate**
   - ❌ Fails by default
   - Requires: Certificate added to runner's trust store OR verification disabled

3. **Internal CA Certificate**
   - ✅ Works if CA cert is in runner's trust store
   - GHES admins typically configure this system-wide

**Options for Handling**:

**Option A: Strict Validation (Recommended)**
- Always validate certificates (default Go behavior)
- Document certificate requirements in README
- User must configure runner environment to trust certificates

**Option B: Opt-In Skip Verification**
- Add `gh-insecure: true` parameter to disable verification
- Show security warning in logs when enabled
- Only for development/testing environments

**Option C: Custom CA Bundle**
- Add `gh-ca-cert` parameter for custom CA certificate path
- More secure than disabling verification
- More complex implementation

### Decision: DD-007

**Start with Strict Validation (Option A)**

**Implementation Phase 1**:
- Use default Go HTTPS client (certificate validation enabled)
- Document in README: "Ensure GitHub Enterprise Server certificate is trusted by runner"
- Clear error messages for certificate failures

**Future Enhancement (Phase 2)**:
- Add optional `gh-insecure` flag if user demand exists
- Include prominent security warning
- Log warning on every request when enabled

**Rationale**:
1. **Security First**: Disabling certificate validation is dangerous (MitM attacks)
2. **Simplicity**: No extra configuration logic in MVP
3. **Enterprise Best Practice**: Proper certificate management is standard
4. **Clear Errors**: Certificate failures produce clear error messages from Go

**Documentation Required**:
```markdown
## Certificate Requirements

GitHub Enterprise Server must use a valid SSL/TLS certificate trusted by the Actions runner.

For self-signed certificates:
1. Add certificate to runner's trust store
2. Restart runner service
3. Verify with: `curl https://github.company.com/api/v3`

For internal CA certificates:
1. Install CA certificate on runner host
2. Update system certificate store
3. Verify with: `openssl s_client -connect github.company.com:443`
```

---

## Summary of Design Decisions

| ID | Decision | Choice | Rationale |
|----|----------|--------|-----------|
| DD-001 | GitHub Client Initialization | Use `WithEnterpriseURLs` method | Leverages library's tested path handling, avoids edge cases |
| DD-002 | Input Parameter Naming | `gh-host` | Short, clear, follows GitHub Actions conventions |
| DD-003 | URL Construction | Hostname + auto protocol/path | Simple UX, leverages library capabilities |
| DD-004 | Error Messages | Actionable guidance format | Helps users fix issues quickly (<2s per SC-004) |
| DD-005 | OAuth2 Configuration | No modifications needed | Protocol-level, works with any HTTPS endpoint |
| DD-006 | Enterprise Detection | Explicit parameter (no auto-detect) | Clarity, simplicity, better debugging |
| DD-007 | Certificate Handling | Strict validation (Phase 1) | Security first, standard enterprise practice |

---

## Implementation Readiness

### Resolved Questions

✅ **RQ-001**: go-github supports enterprise via `WithEnterpriseURLs` - clear implementation pattern identified
✅ **RQ-002**: Validation approach defined - hostname format with clear error messages
✅ **RQ-003**: OAuth2 works without modifications - existing auth code reusable
✅ **RQ-004**: Auto-detection possible but explicit config preferred - simpler and clearer
✅ **RQ-005**: Certificate handling strategy defined - strict validation with clear documentation

### No Blockers Identified

All research questions resolved with clear implementation paths. Ready to proceed to Phase 1: Data Model & Contracts.

### Technical Risk Assessment

| Risk | Severity | Mitigation |
|------|----------|-----------|
| go-github enterprise support issues | Low | Library has mature enterprise support, well-documented |
| URL format edge cases | Low | Leveraging library's path handling, comprehensive validation |
| Certificate trust issues | Medium | Clear documentation, standard enterprise practice |
| Backward compatibility breaks | Low | Empty gh-host maintains current behavior, regression tests verify |

**Overall Risk**: **LOW** - All technical unknowns resolved, implementation path is clear.

---

## Next Phase: Phase 1

Ready to generate:
1. **data-model.md** - Configuration entity details, validation rules
2. **contracts/** - Action input schema, API client interface
3. **quickstart.md** - Enterprise setup guide with examples
4. **Update agent context** - Add enterprise configuration to development context

**Phase 1 Prerequisites**: All met ✅
