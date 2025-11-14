# Quick Start: GitHub Enterprise Server Support

**Feature**: `002-github-enterprise-support`
**Audience**: Enterprise users deploying gitleaks-diff-comment on self-hosted GitHub

---

## Overview

This guide helps you configure the gitleaks-diff-comment action to work with your GitHub Enterprise Server (GHES) instance. The setup takes less than 10 minutes.

**What You'll Need**:
- GitHub Enterprise Server 3.14 or higher
- GitHub Actions enabled on your GHES instance
- A GitHub token with `repo` and `pull_requests:write` permissions
- Network access from Actions runners to your GHES instance

---

## Quick Setup (5 Steps)

### Step 1: Verify Prerequisites

Check your GitHub Enterprise Server version:
```bash
curl https://github.company.com/api/v3/meta | jq '.installed_version'
```

Expected output: `"3.14.0"` or higher

### Step 2: Create GitHub Token

1. Navigate to your GHES instance: `https://github.company.com`
2. Go to **Settings** → **Developer settings** → **Personal access tokens**
3. Click **Generate new token**
4. Select scopes:
   - ✅ `repo` (Full control of private repositories)
   - ✅ `workflow` (Update GitHub Action workflows)
5. Click **Generate token** and copy it

**Security Note**: Store this token as a GitHub Actions secret, never commit it to your repository.

### Step 3: Add Token to Repository Secrets

1. Go to your repository: `https://github.company.com/{owner}/{repo}`
2. Navigate to **Settings** → **Secrets and variables** → **Actions**
3. Click **New repository secret**
4. Name: `GITHUB_TOKEN` (or custom name)
5. Value: Paste the token from Step 2
6. Click **Add secret**

### Step 4: Configure Workflow

Create or update `.github/workflows/gitleaks-comment.yml`:

```yaml
name: Gitleaks Comment

on:
  pull_request:
    types: [opened, synchronize]

jobs:
  comment-on-gitleaks-changes:
    runs-on: ubuntu-latest

    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0  # Required for diff analysis

      - name: Comment on .gitleaksignore changes
        uses: epy0n0ff/gitleaks-diff-comment@v1
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          pr-number: ${{ github.event.pull_request.number }}
          gh-host: github.company.com  # 👈 Add this line for enterprise
```

**Key Change**: Add `gh-host: github.company.com` (replace with your GHES hostname)

### Step 5: Test the Configuration

1. Create a test PR that modifies `.gitleaksignore`
2. The action should run automatically
3. Check the Actions tab for logs: `https://github.company.com/{owner}/{repo}/actions`
4. Verify comments appear on the PR

**Expected Behavior**: Comments posted at the exact lines changed in `.gitleaksignore`

---

## Common Scenarios

### Scenario 1: Enterprise with Custom Port

If your GHES instance uses a non-standard port (e.g., 8443):

```yaml
- uses: epy0n0ff/gitleaks-diff-comment@v1
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    pr-number: ${{ github.event.pull_request.number }}
    gh-host: github.company.com:8443  # 👈 Include port number
```

### Scenario 2: Internal Hostname

If your GHES uses an internal hostname:

```yaml
- uses: epy0n0ff/gitleaks-diff-comment@v1
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    pr-number: ${{ github.event.pull_request.number }}
    gh-host: github.internal  # 👈 Internal DNS name
```

### Scenario 3: IP Address (Testing Only)

For testing or private networks:

```yaml
- uses: epy0n0ff/gitleaks-diff-comment@v1
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    pr-number: ${{ github.event.pull_request.number }}
    gh-host: 10.0.1.50:8443  # 👈 IP address with port
```

**Not Recommended for Production**: Use DNS hostnames for production deployments.

### Scenario 4: Override Comment Mode

Change default behavior (override existing comments vs append new ones):

```yaml
- uses: epy0n0ff/gitleaks-diff-comment@v1
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    pr-number: ${{ github.event.pull_request.number }}
    gh-host: github.company.com
    comment-mode: append  # 👈 "override" (default) or "append"
```

### Scenario 5: Enable Debug Logging

For troubleshooting connectivity issues:

```yaml
- uses: epy0n0ff/gitleaks-diff-comment@v1
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    pr-number: ${{ github.event.pull_request.number }}
    gh-host: github.company.com
    debug: true  # 👈 Enable detailed logging
```

---

## Troubleshooting

### Issue: "Cannot connect to GitHub Enterprise Server"

**Error Message**:
```
Error: Cannot connect to GitHub Enterprise Server at github.company.com
→ Action: Verify hostname is correct and server is reachable
```

**Solutions**:
1. **Verify hostname**: `ping github.company.com` from runner machine
2. **Check DNS**: Ensure hostname resolves correctly
3. **Verify firewall**: Ensure runners can reach GHES on port 443 (or custom port)
4. **Test API endpoint**: `curl https://github.company.com/api/v3/meta`

### Issue: "x509: certificate signed by unknown authority"

**Error Message**:
```
Error: x509: certificate signed by unknown authority
→ Action: Install GitHub Enterprise Server certificate in runner trust store
```

**Solutions**:

**Option A: Use Valid Certificate (Recommended)**
1. Obtain valid SSL certificate from trusted CA
2. Install on GHES instance
3. No runner configuration needed

**Option B: Install Self-Signed Certificate on Runner**
```bash
# Download certificate
openssl s_client -showcerts -connect github.company.com:443 </dev/null 2>/dev/null | \
  openssl x509 -outform PEM > github-enterprise.crt

# Install certificate (Ubuntu/Debian)
sudo cp github-enterprise.crt /usr/local/share/ca-certificates/
sudo update-ca-certificates

# Restart runner
sudo systemctl restart actions.runner.*
```

**Option C: Docker-based Runner**
```dockerfile
# Add to Dockerfile for custom runner image
COPY github-enterprise.crt /usr/local/share/ca-certificates/
RUN update-ca-certificates
```

### Issue: "Authentication failed"

**Error Message**:
```
Error: Authentication failed for github.company.com
→ Action: Check token has required permissions
```

**Solutions**:
1. **Verify token scopes**: Go to GHES → Settings → Developer settings → Personal access tokens
   - Ensure `repo` and `workflow` scopes are enabled
2. **Check token expiration**: Tokens may have expiration dates
3. **Test token manually**:
   ```bash
   curl -H "Authorization: Bearer YOUR_TOKEN" \
        https://github.company.com/api/v3/user
   ```
4. **Regenerate token**: If expired, create new token and update secret

### Issue: "Invalid gh-host format"

**Error Message**:
```
Error: Invalid gh-host format 'https://github.company.com'
→ Action: Remove 'https://' prefix
→ Example: gh-host: github.company.com
```

**Solution**: Remove protocol prefix from `gh-host` value

**Wrong**:
```yaml
gh-host: https://github.company.com  # ❌ Includes https://
gh-host: github.company.com/api/v3   # ❌ Includes path
```

**Correct**:
```yaml
gh-host: github.company.com          # ✅ Hostname only
gh-host: github.company.com:8443     # ✅ Hostname with port
```

### Issue: "Rate limit exceeded"

**Error Message**:
```
Error: API rate limit exceeded for github.company.com
```

**Solutions**:
1. **Check rate limits**: Different GHES instances have different rate limits
2. **Enable debug logging**: See actual limit values
   ```yaml
   debug: true
   ```
3. **Contact GHES admin**: May need to adjust rate limit settings
4. **Reduce comment frequency**: Use `comment-mode: override` to update existing comments instead of creating new ones

### Issue: Action runs but no comments appear

**Possible Causes**:
1. **No changes to .gitleaksignore**: Action only comments when file is modified
2. **Comments filtered as duplicates**: Check if `comment-mode: override` is updating existing comments
3. **Wrong PR number**: Verify `pr-number` is correct
4. **Permissions issue**: Token may lack `pull_requests:write` permission

**Debug Steps**:
1. Enable debug logging: `debug: true`
2. Check action logs in Actions tab
3. Look for "Skipped: X duplicates" messages
4. Verify .gitleaksignore was actually changed in the PR diff

---

## Network Requirements

### Required Connectivity

Actions runners must be able to reach your GHES instance on these ports:

| Port | Protocol | Purpose |
|------|----------|---------|
| 443 | HTTPS | API access (default) |
| 8443 | HTTPS | API access (custom port, if configured) |
| 22 | SSH | Git operations (if using SSH) |

### Firewall Rules

Ensure your firewall allows outbound HTTPS from runners to GHES:

```
Source: Actions runner network
Destination: GHES instance
Port: 443 (or custom HTTPS port)
Protocol: TCP
```

### DNS Requirements

Runner must be able to resolve your GHES hostname:

```bash
# Test DNS resolution
nslookup github.company.com

# Test HTTPS connectivity
curl -I https://github.company.com
```

---

## Certificate Requirements

### Production (Recommended)

Use certificates signed by a trusted Certificate Authority (CA):

1. **Public CA** (e.g., Let's Encrypt, DigiCert)
   - ✅ Works out of the box
   - ✅ No runner configuration needed
   - ✅ Automatically trusted by all systems

2. **Internal CA**
   - ✅ Works if CA cert is in system trust store
   - ⚠️ Requires CA cert installation on runners
   - ✅ More secure than self-signed

### Testing/Development

Self-signed certificates can be used for testing:

1. **Generate self-signed certificate**:
   ```bash
   openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes \
     -subj "/CN=github.company.com"
   ```

2. **Install on GHES instance**

3. **Install on runners** (see Troubleshooting section above)

**Security Warning**: Self-signed certificates do not prevent man-in-the-middle attacks. Use only in isolated development environments.

---

## Advanced Configuration

### Using GitHub App Authentication

For enhanced security, use GitHub App installation tokens:

```yaml
- name: Generate GitHub App token
  id: generate-token
  uses: tibdex/github-app-token@v1
  with:
    app_id: ${{ secrets.APP_ID }}
    private_key: ${{ secrets.APP_PRIVATE_KEY }}

- name: Comment on .gitleaksignore changes
  uses: epy0n0ff/gitleaks-diff-comment@v1
  with:
    github-token: ${{ steps.generate-token.outputs.token }}
    pr-number: ${{ github.event.pull_request.number }}
    gh-host: github.company.com
```

### Multiple Runners on Different Networks

If you have runners in multiple networks (DMZ, internal, etc.):

```yaml
jobs:
  comment-internal:
    runs-on: [self-hosted, internal]  # Runner with internal network access
    steps:
      - uses: epy0n0ff/gitleaks-diff-comment@v1
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          pr-number: ${{ github.event.pull_request.number }}
          gh-host: github.internal  # Internal hostname
```

### Conditional Enterprise Configuration

Use different hostnames based on environment:

```yaml
- uses: epy0n0ff/gitleaks-diff-comment@v1
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    pr-number: ${{ github.event.pull_request.number }}
    gh-host: ${{ secrets.GHES_HOSTNAME }}  # Store hostname in secret
```

---

## Migration from GitHub.com

### Before (GitHub.com)

```yaml
- uses: epy0n0ff/gitleaks-diff-comment@v1
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    pr-number: ${{ github.event.pull_request.number }}
```

### After (GitHub Enterprise Server)

```yaml
- uses: epy0n0ff/gitleaks-diff-comment@v1
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    pr-number: ${{ github.event.pull_request.number }}
    gh-host: github.company.com  # 👈 Only change needed
```

**Migration Time**: <5 minutes per workflow

**Breaking Changes**: None - just add one parameter

---

## Testing Your Configuration

### Test Checklist

- [ ] GHES version is 3.14 or higher
- [ ] GitHub Actions is enabled on GHES
- [ ] Token created with correct permissions
- [ ] Token added to repository secrets
- [ ] Workflow file updated with `gh-host` parameter
- [ ] Runner can reach GHES hostname (ping/curl)
- [ ] Certificate is valid or trusted
- [ ] Test PR created with .gitleaksignore changes
- [ ] Action runs successfully
- [ ] Comments appear on test PR

### Validation Commands

```bash
# 1. Check GHES version
curl https://github.company.com/api/v3/meta | jq '.installed_version'

# 2. Test API connectivity
curl -H "Authorization: Bearer YOUR_TOKEN" \
     https://github.company.com/api/v3/user

# 3. Test certificate
openssl s_client -connect github.company.com:443 -showcerts

# 4. Test DNS resolution
nslookup github.company.com

# 5. Test rate limits
curl -H "Authorization: Bearer YOUR_TOKEN" \
     https://github.company.com/api/v3/rate_limit
```

---

## Getting Help

### Debug Logs

Enable debug logging to see detailed execution information:

```yaml
- uses: epy0n0ff/gitleaks-diff-comment@v1
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    pr-number: ${{ github.event.pull_request.number }}
    gh-host: github.company.com
    debug: true  # 👈 Enable debug mode
```

Debug logs include:
- GitHub Enterprise Server hostname
- Computed API base URL
- Client initialization status
- Rate limit information
- Comment posting progress

### Support Resources

- **Documentation**: [README.md](../../../README.md)
- **Issue Tracker**: [GitHub Issues](https://github.com/epy0n0ff/gitleaks-diff-comment/issues)
- **Specification**: [spec.md](./spec.md) - Full feature specification
- **API Contract**: [contracts/github-client.md](./contracts/github-client.md)

### Common Questions

**Q: Do I need to modify my GHES configuration?**
A: No, the action works with standard GHES installations. No server-side changes needed.

**Q: Can I use the same workflow for both GitHub.com and GHES?**
A: Yes, but you'll need different `gh-host` values. Consider using repository variables or secrets.

**Q: Does this support GitHub Enterprise Cloud?**
A: GitHub Enterprise Cloud uses the same API as GitHub.com, so no `gh-host` is needed.

**Q: What's the performance impact?**
A: Minimal - client initialization adds <1 second. Network latency to your GHES instance is the primary factor.

**Q: Can I use HTTP instead of HTTPS?**
A: Not in the current version. HTTPS is enforced for security. Port 80 connections will fail.

---

## Next Steps

After successful setup:

1. **Monitor action execution**: Check Actions tab regularly
2. **Review comments**: Ensure they're helpful for your team
3. **Adjust comment mode**: Try `override` vs `append` based on preferences
4. **Enable for all repositories**: Roll out to other repos once validated
5. **Provide feedback**: Report issues or suggestions

**Estimated Total Setup Time**: 10 minutes

**Success Criteria**: Comments appear on PRs modifying .gitleaksignore within your GHES instance.
