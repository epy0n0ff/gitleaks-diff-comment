# Development Guide

This guide covers local development, testing, and contributing to the PR Gitleaks Comment Generator action.

## Prerequisites

- Go 1.21 or later
- Docker
- Git
- GitHub account with a test repository

## Project Structure

```
.
├── action.yml              # Action metadata
├── Dockerfile             # Multi-stage Docker build
├── go.mod                 # Go module definition
├── cmd/
│   └── gitleaks-diff-comment/
│       └── main.go        # Entry point
├── internal/
│   ├── config/           # Configuration parsing
│   ├── diff/             # Diff parsing logic
│   ├── comment/          # Comment generation
│   └── github/           # GitHub API client
└── tests/
    ├── fixtures/         # Test data
    └── integration/      # Integration tests
```

## Local Development Setup

### 1. Clone the Repository

```bash
git clone https://github.com/your-org/pr-gitleaks-commenter.git
cd pr-gitleaks-commenter
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Build the Binary

```bash
go build -o gitleaks-diff-comment ./cmd/gitleaks-diff-comment
```

### 4. Run Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test ./internal/diff/...
```

### 5. Run Linter

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run
```

## Testing Locally

### Option 1: Run Binary Directly

```bash
# Set required environment variables
export INPUT_GITHUB-TOKEN="your_github_token"
export INPUT_PR-NUMBER="42"
export GITHUB_REPOSITORY="owner/repo"
export GITHUB_SHA="commit_sha"
export GITHUB_BASE_REF="main"
export GITHUB_HEAD_REF="feature/update-ignore"
export GITHUB_WORKSPACE="/path/to/repo"

# Run the binary
./gitleaks-diff-comment
```

### Option 2: Test with Docker

```bash
# Build the Docker image
docker build -t gitleaks-diff-comment .

# Run the container
docker run --rm \
  -e INPUT_GITHUB-TOKEN="$GITHUB_TOKEN" \
  -e INPUT_PR-NUMBER="42" \
  -e GITHUB_REPOSITORY="owner/repo" \
  -e GITHUB_SHA="$(git rev-parse HEAD)" \
  -e GITHUB_BASE_REF="main" \
  -e GITHUB_HEAD_REF="$(git rev-parse --abbrev-ref HEAD)" \
  -e GITHUB_WORKSPACE="/github/workspace" \
  -v "$(pwd):/github/workspace" \
  -w /github/workspace \
  gitleaks-diff-comment
```

### Option 3: Test with act

[act](https://github.com/nektos/act) allows you to run GitHub Actions locally.

```bash
# Install act
brew install act  # macOS
# or download from https://github.com/nektos/act

# Create a test event file
cat > tests/fixtures/pr-event.json <<EOF
{
  "pull_request": {
    "number": 42
  }
}
EOF

# Run the action locally
act pull_request \
  --secret GITHUB_TOKEN="$GITHUB_TOKEN" \
  --eventpath tests/fixtures/pr-event.json
```

## Development Workflow

### 1. Create a Feature Branch

```bash
git checkout -b feature/your-feature-name
```

### 2. Make Changes

Edit the relevant files in `internal/` or `cmd/`.

### 3. Add Tests

Create or update test files (e.g., `internal/diff/parser_test.go`):

```go
func TestParseDiff(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    []DiffChange
        wantErr bool
    }{
        {
            name:  "addition",
            input: "+config/secrets.yml:42",
            want: []DiffChange{
                {
                    Operation: OperationAddition,
                    Content:   "config/secrets.yml:42",
                },
            },
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseDiff(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ParseDiff() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            // Add assertions
        })
    }
}
```

### 4. Run Tests and Linter

```bash
go test ./...
golangci-lint run
```

### 5. Build Docker Image

```bash
docker build -t gitleaks-diff-comment .
```

### 6. Commit Changes

```bash
git add .
git commit -m "feat: add your feature description"
```

### 7. Push and Create PR

```bash
git push origin feature/your-feature-name
```

## Debugging

### Enable Debug Logging

Set the `debug` input to `true` in your workflow:

```yaml
- name: Comment on .gitleaksignore changes
  uses: ./
  with:
    github-token: ${{ secrets.GITHUB_TOKEN }}
    pr-number: ${{ github.event.pull_request.number }}
    debug: true
```

### View Action Logs

GitHub Actions logs are available in the Actions tab of your repository.

### Debug Locally with Delve

```bash
# Install Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Run with debugger
dlv debug ./cmd/gitleaks-diff-comment -- <args>
```

## Testing with Real PRs

### 1. Create a Test Repository

Create a test repository with a `.gitleaksignore` file.

### 2. Add the Action

Copy the action to `.github/actions/pr-gitleaks-comment/` in your test repository.

### 3. Create a Workflow

```yaml
# .github/workflows/test-pr-comment.yml
name: Test PR Comment

on:
  pull_request:
    paths:
      - '.gitleaksignore'

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: ./.github/actions/pr-gitleaks-comment
        with:
          github-token: ${{ secrets.GITHUB_TOKEN }}
          pr-number: ${{ github.event.pull_request.number }}
```

### 4. Create a Test PR

```bash
git checkout -b test/gitleaks-change
echo "config/test.yml:42" >> .gitleaksignore
git add .gitleaksignore
git commit -m "test: add file to gitleaksignore"
git push origin test/gitleaks-change
```

Create a PR and watch the action run!

## Code Quality Standards

- **Test Coverage**: Aim for >80% coverage
- **Linting**: All code must pass `golangci-lint run`
- **Documentation**: Add GoDoc comments for exported functions
- **Error Handling**: Always return errors, never panic
- **Logging**: Use structured logging with appropriate levels

## Troubleshooting

### "command not found: go"

Install Go from https://go.dev/dl/

### "permission denied" when running Docker

Add your user to the docker group:
```bash
sudo usermod -aG docker $USER
```

### GitHub API rate limit errors

Use a personal access token with higher rate limits, or implement caching.

## Performance Optimization

- Use goroutines for concurrent API requests (max 5 concurrent)
- Implement efficient string processing with `strings.Builder`
- Minimize allocations in hot paths
- Cache parsed diff results when possible

## Release Process

1. Update version in `action.yml`
2. Create a git tag: `git tag v1.0.0`
3. Push tag: `git push origin v1.0.0`
4. GitHub will automatically build and publish the action

## Getting Help

- Open an issue on GitHub
- Check existing issues for similar problems
- Review the specification in `specs/001-gitleaks-diff-comments/`

## Contributing

We welcome contributions! Please:

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

Thank you for contributing to PR Gitleaks Comment Generator!
