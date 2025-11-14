# Data Model: GitHub Enterprise Server Support

**Feature**: `002-github-enterprise-support`
**Phase**: 1 - Data Model & Contracts
**Date**: 2025-11-14

## Overview

This document defines the data entities, their relationships, and validation rules for GitHub Enterprise Server support. The feature adds enterprise connectivity through a single new configuration parameter while preserving all existing entities and behaviors.

---

## Entity: Configuration

**Location**: `internal/config/config.go`
**Type**: Struct (existing, modified)

### Purpose

Holds all configuration parsed from GitHub Actions inputs and environment variables. Controls how the action connects to GitHub (either GitHub.com or GitHub Enterprise Server).

### Fields

```go
type Config struct {
    // Existing fields (unchanged)
    GitHubToken string  // GitHub API authentication token
    PRNumber    int     // Pull request number
    Repository  string  // Repository in format "owner/repo"
    CommitSHA   string  // Commit SHA that triggered the action
    BaseRef     string  // Base branch reference (e.g., "main")
    HeadRef     string  // Head branch reference (e.g., "feature/update")
    Workspace   string  // Workspace directory (git repository root)
    CommentMode string  // Comment mode: "override" or "append"
    Debug       bool    // Enable debug logging

    // NEW: Enterprise support
    GHHost      string  // Optional: GitHub Enterprise Server hostname (empty = GitHub.com)
}
```

### Field Specifications

#### GHHost (NEW)

**Type**: `string`
**Required**: No (optional)
**Default**: `""` (empty string = GitHub.com)
**Source**: `INPUT_GH-HOST` environment variable (from action.yml input)

**Format**: Hostname or hostname:port
- ✅ Valid: `github.company.com`
- ✅ Valid: `github.enterprise.internal`
- ✅ Valid: `github.mycompany.com:8443`
- ✅ Valid: `10.0.1.50:8443` (IP address with port)
- ❌ Invalid: `https://github.company.com` (includes protocol)
- ❌ Invalid: `github.company.com/api/v3` (includes path)
- ❌ Invalid: `http://github.company.com` (includes protocol)

**Validation Rules**:
1. If empty: Valid (defaults to GitHub.com)
2. If provided: Must match pattern `^[a-zA-Z0-9]([a-zA-Z0-9.-])*[a-zA-Z0-9](:[0-9]{1,5})?$`
3. Must NOT contain protocol prefix (`http://`, `https://`)
4. Must NOT contain path separator (`/`)
5. Port number (if provided) must be 1-65535
6. Hostname must be valid DNS name or IP address

**Validation Implementation**:
```go
func (c *Config) Validate() error {
    // ... existing validations ...

    // Validate GHHost format
    if c.GHHost != "" {
        if strings.Contains(c.GHHost, "://") {
            return fmt.Errorf("gh-host must not include protocol (http:// or https://)\n"+
                "  → Action: Remove protocol prefix from gh-host\n"+
                "  → Example: gh-host: %s", strings.Split(c.GHHost, "://")[1])
        }

        if strings.Contains(c.GHHost, "/") {
            return fmt.Errorf("gh-host must not include path\n"+
                "  → Action: Remove path from gh-host (e.g., remove /api/v3)\n"+
                "  → Example: gh-host: %s", strings.Split(c.GHHost, "/")[0])
        }

        // Validate hostname format (simplified - could use net.ParseRequestURI)
        // Port validation if present
        if strings.Contains(c.GHHost, ":") {
            parts := strings.Split(c.GHHost, ":")
            if len(parts) != 2 {
                return errors.New("invalid gh-host format with port")
            }
            port, err := strconv.Atoi(parts[1])
            if err != nil || port < 1 || port > 65535 {
                return fmt.Errorf("invalid port in gh-host: %s (must be 1-65535)", parts[1])
            }
        }
    }

    return nil
}
```

### Parsing Logic

**Location**: `internal/config/config.go` → `ParseFromEnv()` function

```go
func ParseFromEnv() (*Config, error) {
    cfg := &Config{
        GitHubToken: os.Getenv("INPUT_GITHUB-TOKEN"),
        Repository:  os.Getenv("GITHUB_REPOSITORY"),
        CommitSHA:   os.Getenv("GITHUB_SHA"),
        BaseRef:     os.Getenv("GITHUB_BASE_REF"),
        HeadRef:     os.Getenv("GITHUB_HEAD_REF"),
        Workspace:   os.Getenv("GITHUB_WORKSPACE"),
        CommentMode: os.Getenv("INPUT_COMMENT-MODE"),
        GHHost:      os.Getenv("INPUT_GH-HOST"),  // NEW
    }

    // Default comment mode
    if cfg.CommentMode == "" {
        cfg.CommentMode = "override"
    }

    // Parse PR number
    prNumStr := os.Getenv("INPUT_PR-NUMBER")
    if prNumStr != "" {
        prNum, err := strconv.Atoi(prNumStr)
        if err != nil {
            return nil, fmt.Errorf("invalid PR number: %w", err)
        }
        cfg.PRNumber = prNum
    }

    // Parse debug flag
    debugStr := os.Getenv("INPUT_DEBUG")
    cfg.Debug = strings.ToLower(debugStr) == "true"

    // Validate configuration
    if err := cfg.Validate(); err != nil {
        return nil, err
    }

    return cfg, nil
}
```

### State Transitions

```
User Input (action.yml) → Environment Variable (INPUT_GH-HOST)
  ↓
ParseFromEnv() → Config.GHHost
  ↓
Validate() → Validation Errors or Success
  ↓
Config passed to NewClient(token, owner, repo, prNumber, ghHost)
  ↓
GitHub Client initialized with enterprise or default URL
```

### Relationships

- **Config → GitHub Client**: Config.GHHost is passed to `github.NewClient()` to determine API base URL
- **Config → Validation**: Config.Validate() checks GHHost format before client creation
- **Config → Error Messages**: Invalid GHHost produces user-friendly error with correction guidance

---

## Entity: API Configuration

**Location**: Conceptual (embedded in GitHub client creation logic)
**Type**: Derived (not stored as struct, computed during client initialization)

### Purpose

Represents the computed API endpoint configuration used by the GitHub client. Derived from Config.GHHost during client initialization.

### Computed Values

```go
// Conceptual representation (not actual struct in codebase)
type APIConfig struct {
    BaseURL      string  // Full API base URL (e.g., "https://api.github.com" or "https://github.company.com/api/v3/")
    UploadURL    string  // Full upload API URL (e.g., "https://uploads.github.com" or "https://github.company.com/api/uploads/")
    Host         string  // Original hostname (e.g., "github.company.com")
    IsEnterprise bool    // true if using custom host
}
```

### Derivation Logic

**Location**: `internal/github/client.go` → `NewClient()` function

```go
func NewClient(token, owner, repo string, prNumber int, ghHost string) (Client, error) {
    // ... validation ...

    ctx := context.Background()
    ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
    tc := oauth2.NewClient(ctx, ts)

    var ghClient *github.Client
    var err error

    if ghHost != "" {
        // Enterprise: Compute API URLs
        baseURL := "https://" + ghHost
        uploadURL := "https://" + ghHost

        // WithEnterpriseURLs automatically appends /api/v3/ and /api/uploads/
        ghClient, err = github.NewClient(tc).WithEnterpriseURLs(baseURL, uploadURL)
        if err != nil {
            return nil, fmt.Errorf("failed to create GitHub Enterprise client for %s: %w", ghHost, err)
        }
    } else {
        // Default: GitHub.com
        ghClient = github.NewClient(tc)
    }

    return &ClientImpl{
        client:   ghClient,
        owner:    owner,
        repo:     repo,
        prNumber: prNumber,
    }, nil
}
```

### Transformation Examples

| Input (GHHost) | BaseURL Computed | Final API URL (by go-github) |
|----------------|------------------|------------------------------|
| `""` (empty) | N/A (default client) | `https://api.github.com` |
| `github.company.com` | `https://github.company.com` | `https://github.company.com/api/v3/` |
| `github.internal:8443` | `https://github.internal:8443` | `https://github.internal:8443/api/v3/` |
| `api.github.com` | `https://api.github.com` | `https://api.github.com/` (no /api/v3 due to "api." detection) |

---

## Validation Rules Summary

### Configuration Validation

| Rule ID | Field | Validation | Error Message |
|---------|-------|------------|---------------|
| VR-001 | GHHost | Must not contain `://` | "gh-host must not include protocol (http:// or https://)" |
| VR-002 | GHHost | Must not contain `/` | "gh-host must not include path" |
| VR-003 | GHHost | Port must be 1-65535 if provided | "invalid port in gh-host: {port} (must be 1-65535)" |
| VR-004 | GHHost | Empty string is valid (defaults to GitHub.com) | N/A |
| VR-005 | GitHubToken | Must not be empty (existing) | "GitHub token is required (INPUT_GITHUB-TOKEN)" |
| VR-006 | PRNumber | Must be positive (existing) | "PR number must be positive (INPUT_PR-NUMBER)" |

### Client Initialization Validation

| Rule ID | Validation | Error Source | Error Message |
|---------|------------|--------------|---------------|
| CR-001 | WithEnterpriseURLs succeeds | go-github library | "failed to create GitHub Enterprise client for {host}: {error}" |
| CR-002 | OAuth2 client creation | oauth2 library | Auto-handled by library |
| CR-003 | TLS certificate validation | Go TLS library | "x509: certificate signed by unknown authority" or similar |

---

## Data Flow Diagram

```
┌─────────────────────┐
│  GitHub Actions     │
│  Workflow YAML      │
│                     │
│  with:              │
│    gh-host: X       │
└──────────┬──────────┘
           │
           ↓ (GitHub Actions runtime sets INPUT_GH-HOST)
┌─────────────────────┐
│  Environment        │
│  Variables          │
│                     │
│  INPUT_GH-HOST=X    │
└──────────┬──────────┘
           │
           ↓
┌─────────────────────┐
│  config.ParseFromEnv│
│                     │
│  Read INPUT_GH-HOST │
│  Store in Config    │
└──────────┬──────────┘
           │
           ↓
┌─────────────────────┐
│  config.Validate    │
│                     │
│  Check GHHost format│
│  Reject invalid     │
└──────────┬──────────┘
           │
           ↓ (if valid)
┌─────────────────────┐
│  github.NewClient   │
│                     │
│  ghHost="" → default│
│  ghHost="X" → https│
│              ://X   │
└──────────┬──────────┘
           │
           ↓
┌─────────────────────┐
│  go-github library  │
│  WithEnterpriseURLs │
│                     │
│  Append /api/v3/    │
│  Create HTTP client │
└──────────┬──────────┘
           │
           ↓
┌─────────────────────┐
│  GitHub API Client  │
│                     │
│  Ready for API calls│
└─────────────────────┘
```

---

## Error States

### Configuration Errors

**Trigger**: Invalid GHHost format
**Detection**: `config.Validate()` during `ParseFromEnv()`
**Example**:
```
Error: gh-host must not include protocol (http:// or https://)
  → Action: Remove protocol prefix from gh-host
  → Example: gh-host: github.company.com
```

**Recovery**: User fixes workflow YAML, re-runs action

### Client Initialization Errors

**Trigger**: Invalid URL passed to `WithEnterpriseURLs`
**Detection**: go-github library during client creation
**Example**:
```
Error: failed to create GitHub Enterprise client for github.company.com: invalid URL
```

**Recovery**: Should not occur if validation is correct (defensive check)

### Runtime Errors

**Trigger**: Network unreachable, certificate invalid, authentication failed
**Detection**: During first API call
**Examples**:
- `dial tcp: lookup github.company.com: no such host` → DNS resolution failed
- `x509: certificate signed by unknown authority` → SSL certificate issue
- `401 Unauthorized` → Token invalid or insufficient permissions

**Recovery**: User fixes network/certificate/token configuration

---

## Testing Requirements

### Unit Tests

1. **Config Parsing**
   - Parse GHHost from environment variable
   - Default to empty string when INPUT_GH-HOST not set

2. **Config Validation**
   - Reject GHHost with protocol prefix
   - Reject GHHost with path
   - Accept valid hostname
   - Accept hostname with port
   - Accept empty GHHost (GitHub.com)
   - Reject invalid port numbers

3. **Client Initialization**
   - Create default client when GHHost is empty
   - Create enterprise client when GHHost is provided
   - Pass correct baseURL to WithEnterpriseURLs

### Integration Tests

1. **End-to-End Configuration**
   - Set INPUT_GH-HOST environment variable
   - Parse configuration
   - Initialize client
   - Verify client uses correct API endpoint (mock)

2. **Error Handling**
   - Invalid GHHost formats produce clear errors
   - Network errors produce clear errors
   - Certificate errors produce clear errors

---

## Backward Compatibility

### Existing Behavior Preserved

| Scenario | Behavior | Test Coverage |
|----------|----------|---------------|
| GHHost not provided | Defaults to GitHub.com (empty string) | Regression tests |
| GHHost empty string | Same as not provided | Unit tests |
| All existing inputs | Work unchanged | Regression tests |
| Existing API operations | Work identically | Integration tests |

### Migration Path

**No migration needed** - existing workflows continue to work without modification.

**Opt-in enhancement** - enterprise users add single `gh-host` parameter.

---

## Summary

### Changes to Existing Entities

- **Config struct**: Add `GHHost` field (optional, default empty)
- **Config.Validate()**: Add GHHost format validation
- **Config.ParseFromEnv()**: Parse INPUT_GH-HOST environment variable
- **github.NewClient()**: Accept `ghHost` parameter, use WithEnterpriseURLs if provided

### New Entities

None - all changes are modifications to existing entities.

### Validation Guarantees

- GHHost format validated before client creation
- Clear error messages guide users to fix configuration
- Invalid configurations fail fast (before attempting API calls)
- Enterprise URL construction delegated to tested library (go-github)

### Data Integrity

- No data persistence required (stateless action)
- Configuration validated on every run
- No state carried between action executions
