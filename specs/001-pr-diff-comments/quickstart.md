# Quickstart Guide: PR Gitleaks Comment Automation

**Feature**: 001-pr-diff-comments
**Target Audience**: Developers setting up the workflow for the first time
**Time to Complete**: ~15 minutes

## Overview

This workflow automatically comments on pull requests when `.gitleaksignore` files are modified, providing context about which files are being added or removed from gitleaks security scanning.

## Prerequisites

- GitHub repository with pull requests enabled
- GitHub Actions enabled for the repository
- Write permissions to configure workflows
- Basic understanding of GitHub Actions

## Quick Setup

### 1. Create Workflow File

Create `.github/workflows/pr-gitleaks-comments.yml`:

```yaml
name: PR Gitleaks Comment Generator

on:
  pull_request:
    types: [opened, synchronize, reopened]
    paths:
      - '.gitleaksignore'

permissions:
  pull-requests: write
  contents: read

jobs:
  comment-on-gitleaks-changes:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout repository
        uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Full history for diff

      - name: Setup environment
        run: |
          echo "Setting up environment variables"
          echo "PR_NUMBER=${{ github.event.pull_request.number }}" >> $GITHUB_ENV
          echo "BASE_REF=origin/${{ github.event.pull_request.base.ref }}" >> $GITHUB_ENV

      - name: Parse .gitleaksignore diff
        id: parse
        run: |
          bash scripts/pr-diff-comment/parse-gitleaks-diff.sh > /tmp/changes.json
          echo "changes=$(cat /tmp/changes.json)" >> $GITHUB_OUTPUT

      - name: Generate comments
        id: generate
        run: |
          echo '${{ steps.parse.outputs.changes }}' | \
            bash scripts/pr-diff-comment/generate-comment.sh > /tmp/comments.json
          echo "comments=$(cat /tmp/comments.json)" >> $GITHUB_OUTPUT

      - name: Post comments to PR
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        run: |
          echo '${{ steps.generate.outputs.comments }}' | \
            bash scripts/pr-diff-comment/post-comment.sh
```

### 2. Create Script Directory

```bash
mkdir -p scripts/pr-diff-comment
```

### 3. Create Supporting Scripts

The workflow requires three shell scripts (see contracts/ for detailed specifications):

- `scripts/pr-diff-comment/parse-gitleaks-diff.sh` - Extracts .gitleaksignore changes
- `scripts/pr-diff-comment/generate-comment.sh` - Creates comment text
- `scripts/pr-diff-comment/post-comment.sh` - Posts to GitHub API

### 4. Commit and Push

```bash
git add .github/workflows/pr-gitleaks-comments.yml
git add scripts/pr-diff-comment/
git commit -m "feat: add PR gitleaks comment automation"
git push
```

## Testing the Setup

### Test 1: Create a Test PR

1. Create a new branch:
   ```bash
   git checkout -b test/gitleaks-workflow
   ```

2. Modify `.gitleaksignore`:
   ```bash
   echo "config/test-secrets.yml" >> .gitleaksignore
   git add .gitleaksignore
   git commit -m "test: add test file to gitleaks ignore"
   git push -u origin test/gitleaks-workflow
   ```

3. Create a PR on GitHub

4. Verify the workflow runs:
   - Go to Actions tab
   - Check "PR Gitleaks Comment Generator" workflow
   - Verify it completes successfully

5. Check the PR for automated comments:
   - Navigate to Files Changed tab
   - Look for inline comments on `.gitleaksignore` changes

### Test 2: Verify Deduplication

1. Push another commit to the same PR:
   ```bash
   echo "# comment" >> README.md
   git add README.md
   git commit -m "test: trigger workflow again"
   git push
   ```

2. Verify workflow runs but doesn't post duplicate comments

### Test 3: Test Removal

1. Remove an entry from `.gitleaksignore`:
   ```bash
   sed -i '/test-secrets/d' .gitleaksignore
   git add .gitleaksignore
   git commit -m "test: remove from gitleaks ignore"
   git push
   ```

2. Verify comment indicates removal (✅ icon, different message)

## Expected Behavior

### When .gitleaksignore is Modified

**For Added Lines:**
```
🔒 Gitleaks Exclusion Added

This PR adds `config/secrets.yml` to the gitleaks ignore list.

📄 [View file](https://github.com/owner/repo/blob/sha/config/secrets.yml)

⚠️ Security Note: Files in `.gitleaksignore` will be excluded from secret scanning.
Please ensure this exclusion is intentional and documented.
```

**For Removed Lines:**
```
✅ Gitleaks Exclusion Removed

This PR removes `old-config/*.env` from the gitleaks ignore list.

📄 [Previously ignored file](https://github.com/owner/repo/blob/sha/old-config)

ℹ️ Note: This file will now be scanned by gitleaks in future runs.
```

### When .gitleaksignore is NOT Modified

The workflow will not trigger at all (configured via `paths` filter).

## Troubleshooting

### Workflow Doesn't Trigger

**Symptoms**: PR created but workflow doesn't run

**Causes & Solutions**:

1. `.gitleaksignore` not modified
   - Solution: Ensure your PR actually changes `.gitleaksignore`

2. Workflow file not in default branch
   - Solution: Merge workflow file to main/master first

3. Actions disabled for repository
   - Solution: Enable in Settings → Actions → General

### Comments Not Posted

**Symptoms**: Workflow runs successfully but no comments appear

**Possible Causes**:

1. **Duplicate comments detected**
   - Check workflow logs for "skipped_duplicates" count
   - Solution: This is expected behavior

2. **Permission issues**
   - Check workflow logs for 403/401 errors
   - Solution: Verify `pull-requests: write` permission in workflow YAML

3. **GitHub token expired**
   - Rare, but check if `GITHUB_TOKEN` is valid
   - Solution: Default token should work automatically

### Workflow Fails

**Check workflow logs for specific errors:**

1. **Exit code 2**: Missing environment variables
   - Solution: Verify all `${{ github.* }}` context variables are available

2. **Exit code 3**: Git operations failed
   - Solution: Check `fetch-depth: 0` in checkout step

3. **Exit code 4**: API rate limit
   - Solution: Add rate limit handling or reduce frequency

### Script Not Found

**Error**: `scripts/pr-diff-comment/parse-gitleaks-diff.sh: No such file or directory`

**Solution**: Ensure all three scripts exist and have execute permissions:
```bash
chmod +x scripts/pr-diff-comment/*.sh
git add scripts/pr-diff-comment/*.sh
git commit -m "fix: add execute permissions to scripts"
```

## Configuration Options

### Adjust Comment Threshold

To prevent noise on large changes, modify the parse script to limit comments:

```bash
# In parse-gitleaks-diff.sh
MAX_COMMENTS=10  # Limit to 10 comments per PR
```

### Customize Comment Templates

Edit comment templates in `generate-comment.sh`:

```bash
# Find the template section and modify text
ADDITION_TEMPLATE="🔒 **Custom Message**\n\n..."
```

### Change Deduplication Behavior

Disable deduplication (not recommended):

```bash
# In post-comment.sh call
bash scripts/pr-diff-comment/post-comment.sh --no-check-duplicates
```

## Performance

**Typical Execution Times:**
- Small PR (1-5 changes): ~10-15 seconds
- Medium PR (6-20 changes): ~20-30 seconds
- Large PR (20+ changes): ~40-60 seconds

**Resource Usage:**
- CPU: Minimal (shell script processing)
- Memory: <50MB
- Network: ~1-5 API calls per run

## Security Considerations

1. **GitHub Token**: Uses repository's automatic token (read-only by default)
2. **Secrets**: Never logs file contents, only paths
3. **Injection**: All variables quoted in shell scripts
4. **Rate Limits**: Implements backoff to respect API limits

## Next Steps

After successful setup:

1. **Monitor initial PRs** to ensure comments are helpful
2. **Adjust templates** based on team feedback
3. **Document in README** that this automation exists
4. **Train team** on why `.gitleaksignore` changes get commented

## Support

For issues or questions:

1. Check workflow logs in Actions tab
2. Review script contracts in `specs/001-pr-diff-comments/contracts/`
3. Consult data model in `specs/001-pr-diff-comments/data-model.md`
4. Open issue in repository with workflow logs attached

## Advanced Usage

### Running Scripts Locally

Test scripts without creating PRs:

```bash
# Set environment variables
export BASE_REF="origin/main"
export HEAD_REF="HEAD"
export GITHUB_REPOSITORY="owner/repo"
export GITHUB_SHA=$(git rev-parse HEAD)
export PR_NUMBER=123
export GITHUB_TOKEN="your-token"

# Run pipeline
./scripts/pr-diff-comment/parse-gitleaks-diff.sh | \
  ./scripts/pr-diff-comment/generate-comment.sh | \
  ./scripts/pr-diff-comment/post-comment.sh --dry-run
```

### Integration with Other Workflows

Combine with other security workflows:

```yaml
jobs:
  gitleaks-scan:
    runs-on: ubuntu-latest
    steps:
      - uses: gitleaks/gitleaks-action@v2

  comment-on-changes:
    needs: gitleaks-scan
    runs-on: ubuntu-latest
    steps:
      # ... (pr-gitleaks-comments workflow)
```

### Testing with act

Test workflow locally using [act](https://github.com/nektos/act):

```bash
# Install act
brew install act  # macOS
# or
curl https://raw.githubusercontent.com/nektos/act/master/install.sh | sudo bash

# Run workflow
act pull_request -j comment-on-gitleaks-changes \
  --secret GITHUB_TOKEN=your-token \
  --eventpath test-event.json
```
