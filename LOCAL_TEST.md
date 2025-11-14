# Local Testing Guide

Since Go is not installed in this environment, use Docker to build and test.

## Build Docker Image

```bash
docker build -t gitleaks-diff-comment:test .
```

## Test Locally with Docker

### Option 1: Test with mock environment variables

```bash
docker run --rm \
  -e INPUT_GITHUB-TOKEN="test_token" \
  -e INPUT_PR-NUMBER="1" \
  -e GITHUB_REPOSITORY="owner/repo" \
  -e GITHUB_SHA="abc123" \
  -e GITHUB_BASE_REF="main" \
  -e GITHUB_HEAD_REF="feature/test" \
  -e GITHUB_WORKSPACE="/workspace" \
  -e GITHUB_ACTIONS="true" \
  -e INPUT_DEBUG="true" \
  -v "$(pwd):/workspace" \
  -w /workspace \
  gitleaks-diff-comment:test
```

### Option 2: Test the diff parser directly

Create a test script and run it inside the container:

```bash
# Create test script
cat > test_diff.sh <<'EOF'
#!/bin/sh
cd /workspace
git config --global --add safe.directory /workspace
git diff HEAD~1..HEAD -- .gitleaksignore
EOF

chmod +x test_diff.sh

# Run in container
docker run --rm \
  -v "$(pwd):/workspace" \
  -w /workspace \
  gitleaks-diff-comment:test \
  /bin/sh /workspace/test_diff.sh
```

## Quick Build Test

Just test if the code compiles:

```bash
docker build --target builder -t gitleaks-diff-comment:build-test .
```

If this succeeds, the Go code compiles correctly.

## GitHub Actions Local Testing with act

If you have [act](https://github.com/nektos/act) installed:

```bash
# Install act (if not installed)
# On macOS: brew install act
# On Linux: see https://github.com/nektos/act#installation

# Run the action locally
act pull_request \
  --secret GITHUB_TOKEN="your_token" \
  -e tests/pr-event.json
```
