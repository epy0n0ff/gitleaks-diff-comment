# PR Gitleaks Comment Generator

Automatically add explanatory comments to GitHub pull requests when `.gitleaksignore` files are modified. This GitHub Action comments on the specific diff lines where files are added/removed from the gitleaks ignore list, providing context about which files are being excluded from security scanning.

## Features

- 🔒 Automatic comments on `.gitleaksignore` additions with security warnings
- ✅ Clear notifications when files are removed from ignore list
- 🔗 Direct links to referenced files in the repository
- 🚀 Fast processing with concurrent API requests
- 🔄 Intelligent deduplication to avoid duplicate comments
- ⚡ Exponential backoff retry logic for API rate limits

## Usage

### Combined Workflow (Recommended)

Create a single workflow file `.github/workflows/gitleaks-comment.yml` with both comment and clear jobs:

```yaml
name: Gitleaks Diff Comment

on:
  pull_request:
    types: [opened, synchronize, reopened]
    paths:
      - '.gitleaksignore'
  issue_comment:
    types: [created]

permissions:
  pull-requests: write
  contents: read
  issues: write

jobs:
  # Post diff comments when .gitleaksignore changes
  comment:
    if: github.event_name == 'pull_request'
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: ./
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          pr-number: ${{ github.event.pull_request.number }}
          commit-sha: ${{ github.event.pull_request.head.sha }}

  # Clear bot comments with /clear command
  clear:
    if: |
      github.event_name == 'issue_comment' &&
      github.event.issue.pull_request &&
      contains(github.event.comment.body, '@github-actions') &&
      contains(github.event.comment.body, '/clear')
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: ./
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          pr-number: ${{ github.event.issue.number }}
          command: clear
          comment-id: ${{ github.event.comment.id }}
          requester: ${{ github.event.comment.user.login }}
```

### Basic Setup (Comment Only)

Create a workflow file (e.g., `.github/workflows/pr-gitleaks-comment.yml`):

```yaml
name: PR Gitleaks Comment

on:
  pull_request:
    types: [opened, synchronize, reopened]
    paths:
      - '.gitleaksignore'

permissions:
  pull-requests: write
  contents: read

jobs:
  comment:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Comment on .gitleaksignore changes
        uses: ./
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          pr-number: ${{ github.event.pull_request.number }}
          commit-sha: ${{ github.event.pull_request.head.sha }}
```

### Inputs

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `github-token` | Yes | - | GitHub token for API authentication (requires `repo` and `pull_requests:write` scopes) |
| `pr-number` | Yes | - | Pull request number |
| `commit-sha` | No | Auto-detected | Commit SHA to attach comments to. Defaults to PR HEAD commit via `git rev-parse HEAD`. Recommended: `${{ github.event.pull_request.head.sha }}` |
| `comment-mode` | No | `override` | Comment mode: `override` (update existing) or `append` (always create new) |
| `gh-host` | No | `''` | GitHub Enterprise Server hostname (e.g., `github.company.com`). Leave empty for GitHub.com |
| `debug` | No | `false` | Enable debug logging |

### Outputs

| Output | Description |
|--------|-------------|
| `posted` | Number of comments posted |
| `skipped_duplicates` | Number of duplicate comments skipped |
| `errors` | Number of errors encountered |

### Clear Comments Command

You can clear all bot-generated comments from a PR by posting a comment with the `/clear` command:

```
@github-actions /clear
```

**Setup Options**:

1. **Recommended**: Use the [Combined Workflow](#combined-workflow-recommended) shown above
2. **Alternative**: Create a separate `.github/workflows/clear-command.yml` on your default branch (main):

```yaml
name: Clear Comments Command

on:
  issue_comment:
    types: [created]

jobs:
  clear:
    if: |
      github.event.issue.pull_request &&
      contains(github.event.comment.body, '@github-actions') &&
      contains(github.event.comment.body, '/clear')
    runs-on: ubuntu-latest
    permissions:
      pull-requests: write
      issues: write
    steps:
      - uses: actions/checkout@v4
      - uses: ./
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          pr-number: ${{ github.event.issue.number }}
          command: clear
          comment-id: ${{ github.event.comment.id }}
          requester: ${{ github.event.comment.user.login }}
```

**Requirements**:
- **IMPORTANT**: The workflow file must exist on your default branch (main) for `issue_comment` events to work
- User must have write, admin, or maintain access to the repository
- Only deletes comments created by this action (identified by invisible markers)
- Preserves all human-written comments

**Usage examples**:
- `@github-actions /clear` - Basic usage
- `@github-actions /CLEAR` - Case-insensitive
- `@github-actions /clear please remove old comments` - Additional text allowed

## Example Comments

### Addition Comment

When a file is added to `.gitleaksignore`:

> 🔒 **Gitleaks Exclusion Added**
>
> `database/credentials.json:23` will be excluded from secret scanning.
>
> [View file](https://github.com/owner/repo/blob/abc123/database/credentials.json)
>
> ⚠️ **Security Note**: This file will no longer be scanned by gitleaks. Ensure this exclusion is intentional.

### Deletion Comment

When a file is removed from `.gitleaksignore`:

> ✅ **Gitleaks Exclusion Removed**
>
> `old-config/*.env` will now be scanned by gitleaks.
>
> [View file](https://github.com/owner/repo/blob/abc123/old-config)

## How It Works

1. **Trigger**: Action runs when a PR is opened/updated with `.gitleaksignore` changes
2. **Parse Diff**: Extracts additions and deletions from the diff
3. **Generate Comments**: Creates contextual comments with file links
4. **Post Comments**: Posts line-level review comments via GitHub API
5. **Deduplicate**: Checks existing comments to avoid duplicates

## Requirements

- GitHub Actions enabled for the repository
- `GITHUB_TOKEN` with `pull-requests: write` permission
- `.gitleaksignore` file in the repository

## Rate Limiting

The action implements exponential backoff retry logic:
- Initial delay: 1 second
- Second retry: 2 seconds
- Third retry: 4 seconds
- Maximum retries: 3 attempts

If rate limits persist, the action fails gracefully without blocking the PR workflow.

## Security

- Never logs the GitHub token
- Validates all inputs at startup
- Uses secure OAuth2 authentication
- No command injection vulnerabilities
- Minimal Alpine-based Docker image

## Troubleshooting

### Common Issues

#### Action not triggering

**Symptom**: Workflow doesn't run when .gitleaksignore is modified

**Solutions**:
- Verify the workflow file is in `.github/workflows/` directory
- Check that `paths` filter includes `.gitleaksignore`
- Ensure PR events are configured: `[opened, synchronize, reopened]`
- Verify workflow file has correct YAML syntax

#### Permission denied errors

**Symptom**: Error: "Resource not accessible by integration"

**Solutions**:
- Add required permissions to workflow:
  ```yaml
  permissions:
    pull-requests: write
    contents: read
  ```
- Verify repository settings allow GitHub Actions to create comments
- Check that `GITHUB_TOKEN` has sufficient permissions

#### Rate limit errors

**Symptom**: "rate limit exceeded" in action logs

**Solutions**:
- Action automatically retries with exponential backoff (1s, 2s, 4s)
- For large PRs (50+ changes), the action processes them in batches
- If rate limits persist, reduce PR size or wait for rate limit reset
- Monitor: The action logs rate limit status when debug mode is enabled

#### Comments not appearing

**Symptom**: Action runs successfully but no comments visible

**Solutions**:
- Check that .gitleaksignore actually changed in the PR diff
- Verify comments aren't being deduplicated (check action logs)
- Ensure `fetch-depth: 0` is set in checkout step for full git history
- Confirm PR is not in draft mode (comments may be hidden)

#### Invalid configuration errors

**Symptom**: "GitHub token is required" or similar validation errors

**Solutions**:
- Ensure `github-token` input is set: `${{ secrets.GITHUB_TOKEN }}`
- Verify `pr-number` input: `${{ github.event.pull_request.number }}`
- Check that action is running in pull_request event context
- Review error message for specific guidance on missing configuration

#### Clear command permission denied

**Symptom**: `/clear` command responds with "Permission denied: User does not have required permissions"

**Solutions**:
- Verify you have write, admin, or maintain access to the repository
- Check repository collaborator settings
- PR authors automatically have permission on their own PRs
- External contributors with read-only access cannot use this command

#### Clear command not triggering

**Symptom**: Posting `@github-actions /clear` comment doesn't trigger the workflow

**Solutions**:
- **MOST COMMON**: Ensure clear-command.yml workflow file exists on the **default branch (main/master)**
  - `issue_comment` events only trigger workflows from the default branch
  - Feature branch workflows will NOT trigger on issue comments
  - Solution: Merge the workflow file to main branch first
- Ensure clear-command.yml workflow file exists in `.github/workflows/`
- Verify workflow has correct permissions: `pull-requests: write`, `issues: write`
- Check workflow `if` condition includes all required checks
- Command is case-insensitive: `/clear`, `/CLEAR`, `/Clear` all work
- Must mention `@github-actions` before the command

#### No comments cleared

**Symptom**: Clear command runs but reports 0 comments cleared

**Solutions**:
- Verify bot comments exist on the PR (check for comments with gitleaks exclusion markers)
- Only comments created by this action are deleted (identified by invisible markers)
- Human comments are always preserved
- Check workflow logs for "Found N bot comments to delete" message

#### Rate limit errors during clear

**Symptom**: "Rate limit exceeded after 3 retries" in clear command logs

**Solutions**:
- Action automatically retries with exponential backoff (2s, 4s, 8s)
- Maximum 3 retry attempts per comment deletion
- If many comments (50+), rate limits may be hit
- Wait for rate limit reset (check X-RateLimit-Reset header in logs)
- Re-run the command after rate limit resets

#### Docker build failures

**Symptom**: Action fails during Docker build step

**Solutions**:
- Verify go.mod and go.sum are present and valid
- Check that all source files are committed
- Ensure Dockerfile is not corrupted
- Review build logs for specific Go compilation errors

### Debug Mode

Enable debug logging for detailed troubleshooting:

```yaml
- uses: ./
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    pr-number: ${{ github.event.pull_request.number }}
    debug: 'true'  # Enable debug logging
```

Debug mode provides:
- Configuration details
- Rate limit status
- Per-comment processing logs
- Retry attempt information
- Progress updates for large batches

## GitHub Enterprise Server Support

This action fully supports GitHub Enterprise Server (GHES) 3.14+ installations. Configure your enterprise instance using the `gh-host` parameter.

### Enterprise Setup

Add the `gh-host` input to your workflow:

```yaml
- name: Comment on .gitleaksignore changes
  uses: epy0n0ff/gitleaks-diff-comment@v1
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    pr-number: ${{ github.event.pull_request.number }}
    commit-sha: ${{ github.event.pull_request.head.sha }}
    gh-host: github.company.com  # Your enterprise hostname
```

**Important**: Provide only the hostname (optionally with port). Do not include `https://` or path components.

### Examples

```yaml
# Simple hostname
gh-host: github.company.com

# Hostname with custom port
gh-host: github.company.com:8443

# Internal hostname
gh-host: github.internal

# GitHub.com (default) - leave empty or omit
gh-host: ''
```

### Troubleshooting

**Cannot connect to GitHub Enterprise Server**
```
Error: cannot connect to GitHub Enterprise Server at github.company.com
  → Action: Verify hostname is correct and server is reachable
  → Check: Network connectivity, firewall rules, DNS resolution
```

**Solutions**:
- Verify hostname: `ping github.company.com` from runner
- Check DNS resolution
- Ensure runner can reach GHES on port 443 (or custom port)
- Test API endpoint: `curl https://github.company.com/api/v3/meta`

**Certificate signed by unknown authority**
```
Error: x509: certificate signed by unknown authority
  → Action: Install GitHub Enterprise Server certificate in runner trust store
```

**Solutions**:
- Use valid SSL certificate from trusted CA (recommended)
- Install self-signed certificate on runners (testing only)
- See [quickstart guide](./specs/002-github-enterprise-support/quickstart.md) for certificate setup

**Authentication failed**
```
Error: authentication failed for GitHub Enterprise Server at github.company.com
  → Action: Verify token has required permissions (repo, pull_requests)
  → Check: Token is valid for enterprise instance
```

**Solutions**:
- Verify token scopes in GHES settings
- Check token expiration date
- Test token: `curl -H "Authorization: Bearer TOKEN" https://github.company.com/api/v3/user`
- Regenerate token if needed

**Invalid gh-host format**
```
Error: gh-host must not include protocol (http:// or https://)
  → Action: Remove protocol prefix from gh-host
  → Example: gh-host: github.company.com
```

**Solution**: Remove `https://` or `http://` and any path like `/api/v3`

### Enterprise Features

- ✅ Automatic rate limit detection from enterprise instance
- ✅ Support for Personal Access Tokens and GitHub App tokens
- ✅ Custom port numbers (e.g., `:8443`)
- ✅ Internal hostnames and IP addresses
- ✅ Full backward compatibility with GitHub.com

For detailed setup instructions, see the [Enterprise Quick Start Guide](./specs/002-github-enterprise-support/quickstart.md).

### Getting Help

1. Check the [DEVELOPMENT.md](./DEVELOPMENT.md) for local testing
2. Review action logs in the Actions tab
3. Enable debug mode for detailed information
4. Open an issue with logs and workflow configuration

## Contributing

See [DEVELOPMENT.md](./DEVELOPMENT.md) for local development instructions.

## License

MIT License - See LICENSE file for details
