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

### Basic Setup

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
```

### Inputs

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `github-token` | Yes | - | GitHub token for API authentication |
| `pr-number` | Yes | - | Pull request number |
| `debug` | No | `false` | Enable debug logging |

### Outputs

| Output | Description |
|--------|-------------|
| `posted` | Number of comments posted |
| `skipped_duplicates` | Number of duplicate comments skipped |
| `errors` | Number of errors encountered |

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

### Getting Help

1. Check the [DEVELOPMENT.md](./DEVELOPMENT.md) for local testing
2. Review action logs in the Actions tab
3. Enable debug mode for detailed information
4. Open an issue with logs and workflow configuration

## Contributing

See [DEVELOPMENT.md](./DEVELOPMENT.md) for local development instructions.

## License

MIT License - See LICENSE file for details
