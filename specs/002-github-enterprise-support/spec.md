# Feature Specification: GitHub Enterprise Server Support

**Feature Branch**: `002-github-enterprise-support`
**Created**: 2025-11-14
**Status**: Draft
**Input**: User description: "セルフホストのGithubEnterpriseServerにも対応する"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Enterprise User Can Use Action on Self-Hosted GitHub (Priority: P1)

An organization using GitHub Enterprise Server (self-hosted) wants to use the gitleaks-diff-comment action in their PR workflows. The action should connect to their enterprise instance URL and work identically to how it works on GitHub.com.

**Why this priority**: This is the core requirement - without this, the feature doesn't exist. Enterprise customers cannot use the action at all without custom API endpoint support.

**Independent Test**: Can be fully tested by configuring the action with a GitHub Enterprise Server URL and verifying it successfully posts comments on PRs in that environment. Delivers immediate value by enabling enterprise adoption.

**Acceptance Scenarios**:

1. **Given** a GitHub Enterprise Server instance at `https://github.company.com`, **When** a user configures the action with `github-api-url: https://github.company.com/api/v3`, **Then** the action connects to the enterprise API endpoint instead of GitHub.com
2. **Given** an enterprise instance with valid authentication, **When** the action runs on a PR with .gitleaksignore changes, **Then** comments are posted successfully to the enterprise PR
3. **Given** no custom API URL is provided, **When** the action runs on GitHub.com, **Then** it defaults to the public GitHub API and works as before (backward compatibility)

---

### User Story 2 - Support Multiple Enterprise Authentication Methods (Priority: P2)

Organizations using GitHub Enterprise Server may have different authentication configurations (personal access tokens, GitHub Apps, OAuth Apps). The action should work with standard GitHub authentication methods supported by enterprise instances.

**Why this priority**: While core connectivity (P1) gets enterprises running, different auth methods are often mandated by security policies. This enables broader enterprise adoption but isn't blocking for initial usage.

**Independent Test**: Can be tested by authenticating with different token types (PAT, GitHub App installation token) against an enterprise instance and verifying successful API operations.

**Acceptance Scenarios**:

1. **Given** a GitHub Enterprise Server with personal access token authentication, **When** the action uses a PAT with appropriate scopes, **Then** all API operations succeed
2. **Given** a GitHub Enterprise Server with GitHub App authentication, **When** the action uses an installation token, **Then** all API operations succeed
3. **Given** insufficient token permissions, **When** the action attempts API operations, **Then** a clear error message indicates the missing permissions

---

### User Story 3 - Validate Enterprise Instance Connectivity (Priority: P2)

Before attempting to post comments, users want to know if their enterprise instance is reachable and properly configured, to avoid cryptic connection failures during PR workflows.

**Why this priority**: Improves troubleshooting experience but isn't required for basic functionality. Users can still use the feature without explicit validation, though errors will be less clear.

**Independent Test**: Can be tested by providing invalid or unreachable enterprise URLs and verifying clear error messages guide users to fix configuration issues.

**Acceptance Scenarios**:

1. **Given** an unreachable enterprise API URL, **When** the action initializes, **Then** it fails fast with a clear error message indicating the API endpoint cannot be reached
2. **Given** an invalid API URL format (e.g., missing `/api/v3`), **When** the action attempts connection, **Then** it provides guidance on correct URL format
3. **Given** a reachable enterprise instance, **When** the action checks connectivity, **Then** it proceeds with normal operation

---

### User Story 4 - Support Enterprise-Specific Rate Limits (Priority: P3)

GitHub Enterprise Server instances may have different rate limits than GitHub.com. The action should respect enterprise-specific rate limit headers and adjust retry behavior accordingly.

**Why this priority**: Nice to have for optimal performance in high-volume scenarios, but the existing retry logic will work (just less efficiently). Not blocking for adoption.

**Independent Test**: Can be tested by monitoring rate limit headers from an enterprise instance and verifying the action respects custom limits without exceeding quotas.

**Acceptance Scenarios**:

1. **Given** an enterprise instance with custom rate limits, **When** the action checks rate limits, **Then** it reads and respects the enterprise-specific limits from API headers
2. **Given** approaching enterprise rate limits, **When** posting multiple comments, **Then** the action throttles appropriately to avoid exceeding limits
3. **Given** different rate limits than GitHub.com defaults, **When** debug logging is enabled, **Then** the action logs the detected rate limits for troubleshooting

---

### Edge Cases

- What happens when the enterprise API URL has a custom path prefix (e.g., `/api/github/v3`)?
- How does the system handle enterprise instances with self-signed SSL certificates?
- What happens when an enterprise instance is running an older GitHub Enterprise Server version with slightly different API behavior?
- How does the action behave when the enterprise instance is temporarily unreachable (network issues)?
- What happens if the enterprise instance doesn't support the line-based comment API (older versions)?
- How does the action handle enterprise instances behind VPNs or with IP allowlists?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST accept an optional GitHub API base URL configuration parameter
- **FR-002**: System MUST default to `https://api.github.com` when no custom API URL is provided (backward compatibility)
- **FR-003**: System MUST validate the provided API URL format before attempting connections
- **FR-004**: System MUST support GitHub Enterprise Server API endpoints following the `/api/v3` path convention
- **FR-005**: System MUST authenticate using the provided GitHub token against the configured API endpoint
- **FR-006**: System MUST support all existing GitHub API operations (create comment, update comment, list comments, check rate limits) against enterprise instances
- **FR-007**: System MUST handle SSL/TLS certificate verification for enterprise instances
- **FR-008**: System MUST provide clear error messages when enterprise connectivity fails, distinguishing between authentication, network, and configuration errors
- **FR-009**: System MUST preserve all existing functionality when used with GitHub.com (no breaking changes)
- **FR-010**: System MUST respect rate limit headers returned by enterprise instances
- **FR-011**: System MUST work with GitHub Enterprise Server versions 3.14 and above
- **FR-012**: System MUST handle both HTTP and HTTPS enterprise endpoints

### Key Entities

- **Enterprise API Endpoint**: The base URL for GitHub Enterprise Server API (e.g., `https://github.company.com/api/v3`), including protocol, hostname, port, and path
- **API Configuration**: Settings that determine how the action connects to GitHub (API URL, token, SSL verification options)
- **Enterprise Rate Limits**: Custom rate limiting rules specific to the enterprise instance, potentially different from GitHub.com defaults

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can successfully configure and use the action with GitHub Enterprise Server instances without code modifications
- **SC-002**: The action connects to and operates with enterprise instances in under 5 seconds (excluding PR analysis time)
- **SC-003**: 100% of existing functionality (comment posting, updating, deduplication) works identically on enterprise instances
- **SC-004**: Configuration errors for enterprise URLs are detected within 2 seconds with clear, actionable error messages
- **SC-005**: The action supports at least 95% of GitHub Enterprise Server installations (versions 3.14+)
- **SC-006**: Zero breaking changes for existing GitHub.com users (backward compatibility maintained)
- **SC-007**: Enterprise users can complete initial setup and configuration in under 10 minutes with documentation

## Assumptions *(optional)*

- GitHub Enterprise Server instances follow standard GitHub API v3 conventions
- Enterprise instances are accessible via HTTPS with valid SSL certificates (or users can disable verification)
- Authentication mechanisms (tokens, OAuth) work the same on enterprise as on GitHub.com
- The go-github library supports custom API base URLs for enterprise connectivity
- Enterprise administrators have already granted necessary token permissions for the action to operate

## Dependencies *(optional)*

- **go-github library**: Must support custom base URL configuration for enterprise endpoints
- **OAuth2 library**: Must work with enterprise authentication endpoints
- **GitHub Enterprise Server API**: Enterprise instances must implement the standard GitHub API v3 specifications
- **Network connectivity**: Enterprise instances must be network-accessible from where the action runs (consider firewall rules, VPNs)

## Out of Scope *(optional)*

- Support for GitHub Enterprise Server versions older than 3.14
- Custom authentication methods specific to enterprise deployments (SAML, LDAP) - these are handled by GitHub's token generation
- Migration tools for converting GitHub.com workflows to enterprise
- On-premises runner-specific optimizations
- Support for multiple enterprise instances in a single workflow
- Automatic discovery of enterprise API endpoints
- Enterprise-specific features not available in the public GitHub API

## Future Considerations *(optional)*

- Auto-detection of GitHub Enterprise Server vs GitHub.com based on environment
- Support for GitHub Enterprise Cloud (which uses different API endpoints than self-hosted)
- Caching of enterprise instance metadata to reduce API calls
- Support for enterprise instances with custom API paths beyond `/api/v3`
- Integration with GitHub App authentication for enhanced security in enterprise environments
- Metrics and monitoring specific to enterprise deployments (connection success rates, latency)
