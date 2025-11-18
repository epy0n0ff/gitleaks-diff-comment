# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- **Critical: Fixed commit SHA detection for PR comments** - Resolved `422 Validation Failed` error on GitHub Enterprise Server
  - Issue: Comments were failing with "pull_request_review_thread.end_commit_oid is not part of the pull request"
  - Root cause: Using `GITHUB_SHA` environment variable which may not point to PR HEAD commit
  - Solution: Added `commit-sha` input parameter with auto-detection via `git rev-parse HEAD`
  - Recommended usage: `commit-sha: ${{ github.event.pull_request.head.sha }}`
  - Backward compatible: Auto-detects correct commit if not specified
  - Affects: Both GitHub.com and GitHub Enterprise Server installations
- **Fixed file links in PR comments for Enterprise Server** - File links now use correct enterprise hostname
  - Issue: File links in comments always pointed to github.com even on enterprise servers
  - Solution: FileLink method now uses gh-host parameter to generate enterprise URLs
  - Impact: Users can now click links to view files on their enterprise instance
  - Works with custom ports (e.g., `github.company.com:8443`)

### Added
- **/clear command for comment management** - Clear all bot comments from a PR with a simple command
  - Post `@github-actions /clear` in any PR comment to remove all bot-generated comments
  - Case-insensitive command detection (`/clear`, `/CLEAR`, `/Clear` all work)
  - **Authorization**: Only users with write, admin, or maintain access can use this command
  - **Safety**: Preserves all human-written comments, only deletes bot comments with markers
  - **Retry logic**: Exponential backoff (2s, 4s, 8s) handles rate limits automatically
  - **Observability**: Structured JSON metrics logged for monitoring and debugging
  - **Workflow**: Separate `clear-command.yml` workflow triggered by issue_comment events
  - **Performance**: Completes in <10 seconds for typical PRs (≤100 comments)
  - **Example**: User posts `/clear` → Bot checks permissions → Identifies bot comments → Deletes with retry → Logs metrics
- Command detection infrastructure with regex pattern matching
- Permission verification using GitHub API GetPermissionLevel
- Metrics event logging with structured JSON format (event_type, timestamp, counts, duration)
- GitHub Actions annotations (::notice::, ::warning::, ::error::) for visibility
- Job summary in workflow showing operation results
- **GitHub Enterprise Server support** - Action now supports GitHub Enterprise Server (GHES) 3.14+ installations
  - New `gh-host` input parameter for enterprise hostname configuration (e.g., `github.company.com`)
  - Support for custom ports (e.g., `github.company.com:8443`)
  - Support for internal hostnames and IP addresses
  - Automatic rate limit detection from enterprise instances
  - Full backward compatibility with GitHub.com (default behavior unchanged)
- Enhanced error messages with context-specific troubleshooting guidance
  - Authentication errors now show token validation steps
  - Network errors provide connectivity troubleshooting guidance
  - Configuration errors include format correction examples
- Comprehensive enterprise documentation
  - Setup instructions in README.md
  - Troubleshooting guide for common errors
  - Quickstart guide for enterprise users
- Integration tests for enterprise authentication scenarios
  - Personal Access Token (PAT) authentication validation
  - Network error handling
  - Rate limit handling for enterprise instances
  - Custom port support validation

### Changed
- GitHub API client now uses `WithEnterpriseURLs` for enterprise instances
- Rate limit checking respects enterprise-specific limits
- Error handling enhanced with `isAuthError` and `isNetworkError` helpers

### Technical Details
- Requires go-github v57 (includes `WithEnterpriseURLs` support)
- Uses OAuth2 Bearer token authentication (compatible with PAT and GitHub App tokens)
- Supports GHES API version 3.14+
- Docker image: golang:1.25-alpine

## [1.0.0] - Initial Release

### Added
- Automatic comments on `.gitleaksignore` additions with security warnings
- Clear notifications when files are removed from ignore list
- Direct links to referenced files in the repository
- Fast processing with concurrent API requests
- Intelligent deduplication to avoid duplicate comments
- Exponential backoff retry logic for API rate limits
- Support for `override` and `append` comment modes
- Debug logging mode for troubleshooting
