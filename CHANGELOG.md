# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
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
