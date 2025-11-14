# Research: Automated PR Diff Comment Explanations (Docker Custom Action)

**Feature**: 001-pr-diff-comments
**Date**: 2025-11-13
**Purpose**: Resolve technical unknowns and establish implementation patterns for Go/Docker custom action

## Research Questions

### 1. Docker-based Custom GitHub Action Architecture

**Question**: What is the best structure for a Docker-based custom GitHub Action in Go?

**Decision**: Multi-stage Docker build with action.yml interface

**Rationale**:
- Multi-stage builds reduce final image size (Go builder stage + Alpine runtime stage)
- action.yml defines inputs/outputs following GitHub Actions metadata schema v1.0
- Entrypoint script handles environment variable mapping from action inputs
- Go binary compiled statically for Alpine Linux compatibility

**Implementation Pattern**:
```yaml
# action.yml
name: 'PR Gitleaks Comment Generator'
description: 'Automatically comment on .gitleaksignore changes in PRs'
inputs:
  github-token:
    description: 'GitHub token for API authentication'
    required: true
  pr-number:
    description: 'Pull request number'
    required: true
runs:
  using: 'docker'
  image: 'Dockerfile'
```

```dockerfile
# Dockerfile (multi-stage)
FROM golang:1.24-alpine AS builder
WORKDIR /build
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o pr-diff-comment ./cmd/pr-diff-comment

FROM alpine:3.22
RUN apk add --no-cache git ca-certificates
COPY --from=builder /build/pr-diff-comment /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/pr-diff-comment"]
```

**Alternatives Considered**:
- JavaScript action: Requires Node.js runtime, less type-safe than Go
- Composite action: Would need external dependencies, less portable
- Pre-built Docker image: Requires separate registry, complicates distribution

### 2. Go GitHub API Client Selection

**Question**: Which Go library should be used for GitHub API interactions?

**Decision**: Use `github.com/google/go-github/v57` (official Google library)

**Rationale**:
- Most widely used Go GitHub client (14k+ stars)
- Comprehensive API coverage including PR review comments
- Well-maintained with regular updates
- Type-safe request/response structs
- Built-in pagination and rate limit handling
- Official OAuth2 integration via `golang.org/x/oauth2`

**Implementation Pattern**:
```go
import (
    "context"
    "github.com/google/go-github/v57/github"
    "golang.org/x/oauth2"
)

func NewGitHubClient(token string) *github.Client {
    ctx := context.Background()
    ts := oauth2.StaticTokenSource(
        &oauth2.Token{AccessToken: token},
    )
    tc := oauth2.NewClient(ctx, ts)
    return github.NewClient(tc)
}

// Post review comment
client.PullRequests.CreateComment(ctx, owner, repo, prNum, &github.PullRequestComment{
    Body:     github.String(commentBody),
    CommitID: github.String(commitSHA),
    Path:     github.String(".gitleaksignore"),
    Position: github.Int(diffPosition),
})
```

**Alternatives Considered**:
- Direct REST API calls: More code, error-prone, no type safety
- github.com/bradleyfalzon/ghinstallation: For GitHub Apps, not needed for PAT auth
- Custom API wrapper: Unnecessary reinvention of the wheel

### 3. Diff Parsing Strategy in Go

**Question**: How to parse git diff output in Go to extract .gitleaksignore changes?

**Decision**: Use `os/exec` to run git commands, parse diff with regex and line-by-line processing

**Rationale**:
- Git CLI is available in Docker image (installed via apk)
- Go's `bufio.Scanner` efficiently processes diff line-by-line
- Regex patterns can extract hunk headers (@@ -x,y +a,b @@)
- Standard library sufficient, no external diff parsing dependencies needed

**Implementation Pattern**:
```go
func ParseGitleaksDiff(baseBranch, headRef string) ([]DiffChange, error) {
    cmd := exec.Command("git", "diff", baseBranch+"..."+headRef, "--", ".gitleaksignore")
    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }

    var changes []DiffChange
    scanner := bufio.NewScanner(bytes.NewReader(output))
    lineNum := 0
    position := 0

    hunkRegex := regexp.MustCompile(`^@@ -(\d+),?\d* \+(\d+),?\d* @@`)

    for scanner.Scan() {
        line := scanner.Text()
        position++

        if matches := hunkRegex.FindStringSubmatch(line); matches != nil {
            lineNum, _ = strconv.Atoi(matches[2])
            continue
        }

        if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
            changes = append(changes, DiffChange{
                Operation:   "addition",
                LineNumber:  lineNum,
                Content:     strings.TrimPrefix(line, "+"),
                Position:    position,
            })
            lineNum++
        } else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
            changes = append(changes, DiffChange{
                Operation:  "deletion",
                Content:    strings.TrimPrefix(line, "-"),
                Position:   position,
            })
        } else if !strings.HasPrefix(line, "\\") {
            lineNum++
        }
    }

    return changes, scanner.Err()
}
```

**Alternatives Considered**:
- Third-party diff library (go-diff): Additional dependency for simple parsing
- GitHub API diff endpoint: Requires API call, rate-limited, more complex
- Pure regex parsing: Less robust for complex diff scenarios

### 4. Comment Template Generation

**Question**: How to generate markdown comment templates in Go?

**Decision**: Use Go's `text/template` package with embedded templates

**Rationale**:
- Standard library, no external dependencies
- Template inheritance and composition
- Safe HTML/markdown escaping
- Can embed templates using `//go:embed` directive (Go 1.16+)
- Easy to test with different data inputs

**Implementation Pattern**:
```go
import (
    _ "embed"
    "text/template"
)

//go:embed templates/addition.md
var additionTemplate string

//go:embed templates/deletion.md
var deletionTemplate string

type CommentData struct {
    FilePattern string
    FileLink    string
    Operation   string
}

func GenerateComment(change DiffChange, repo, sha string) (string, error) {
    tmpl := additionTemplate
    if change.Operation == "deletion" {
        tmpl = deletionTemplate
    }

    t, err := template.New("comment").Parse(tmpl)
    if err != nil {
        return "", err
    }

    data := CommentData{
        FilePattern: extractFilePattern(change.Content),
        FileLink:    buildFileLink(repo, sha, change.Content),
        Operation:   change.Operation,
    }

    var buf bytes.Buffer
    if err := t.Execute(&buf, data); err != nil {
        return "", err
    }

    return buf.String(), nil
}
```

**Alternatives Considered**:
- String concatenation: Less maintainable, harder to modify templates
- External template files: Complicates Docker image, harder to distribute
- Third-party templating (Mustache, Handlebars): Unnecessary complexity

### 5. Testing Strategy for Go Custom Action

**Question**: How to test a Go-based GitHub Action locally and in CI?

**Decision**: Multi-layered testing with mocks for GitHub API

**Rationale**:
- Unit tests for each package using Go's testing framework
- Table-driven tests for different diff scenarios
- Mocks for GitHub API using interfaces
- Integration tests using `act` for full action testing
- Fixtures for sample diffs and API responses

**Implementation Pattern**:
```go
// Define interface for mocking
type GitHubClient interface {
    CreateComment(ctx context.Context, owner, repo string, number int, comment *github.PullRequestComment) (*github.PullRequestComment, *github.Response, error)
    ListComments(ctx context.Context, owner, repo string, number int, opts *github.PullRequestListCommentsOptions) ([]*github.PullRequestComment, *github.Response, error)
}

// Mock implementation for tests
type MockGitHubClient struct {
    CreateCommentFunc func(...) (...)
    ListCommentsFunc  func(...) (...)
}

// Unit test
func TestGenerateComment(t *testing.T) {
    tests := []struct {
        name    string
        change  DiffChange
        want    string
        wantErr bool
    }{
        {
            name: "addition comment",
            change: DiffChange{
                Operation: "addition",
                Content:   "config/secrets.yml:42",
            },
            want:    "🔒 **Gitleaks Exclusion Added**",
            wantErr: false,
        },
        // More test cases...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := GenerateComment(tt.change, "owner/repo", "abc123")
            if (err != nil) != tt.wantErr {
                t.Errorf("GenerateComment() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !strings.Contains(got, tt.want) {
                t.Errorf("GenerateComment() = %v, want to contain %v", got, tt.want)
            }
        })
    }
}
```

**Alternatives Considered**:
- Only integration tests: Slow, hard to cover edge cases
- No mocking: Requires real GitHub API, brittle tests
- External test framework (Ginkgo): Standard library sufficient

### 6. Docker Image Optimization

**Question**: How to minimize Docker image size for faster action startup?

**Decision**: Multi-stage build with Alpine Linux and static binary compilation

**Rationale**:
- Alpine base image is ~5MB vs Ubuntu's ~70MB
- Static Go binary (CGO_ENABLED=0) has no runtime dependencies
- Only install necessary Alpine packages (git, ca-certificates)
- Multi-stage build discards build dependencies
- Target image size: <50MB

**Implementation Pattern**:
```dockerfile
# Stage 1: Build
FROM golang:1.24-alpine AS builder
WORKDIR /build

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Build static binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o pr-diff-comment \
    ./cmd/pr-diff-comment

# Stage 2: Runtime
FROM alpine:3.22
RUN apk add --no-cache git ca-certificates

# Copy only the binary
COPY --from=builder /build/pr-diff-comment /usr/local/bin/

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/pr-diff-comment"]
```

**Optimization Flags**:
- `-ldflags="-w -s"`: Strip debug info and symbol table (~30% size reduction)
- `CGO_ENABLED=0`: Static binary, no libc dependency
- `GOARCH=amd64`: Target most common GitHub Actions runners

**Alternatives Considered**:
- Scratch image: No shell for debugging, breaks git commands
- Distroless: Similar size to Alpine, less familiar
- Ubuntu base: 10x larger image, slower pull times

## Technology Stack Summary

| Component | Technology | Version | Purpose |
|-----------|-----------|---------|---------|
| Language | Go | 1.21+ | Core implementation |
| GitHub API | google/go-github | v57 | API client |
| Authentication | golang.org/x/oauth2 | Latest | Token auth |
| Container | Docker | Multi-stage | Packaging |
| Base Image | Alpine Linux | 3.22 | Runtime environment |
| Testing | Go testing | Standard lib | Unit tests |
| Mocking | Interfaces | Native Go | Test doubles |
| Templates | text/template | Standard lib | Comment generation |
| CLI | flag package | Standard lib | Argument parsing |

## Implementation Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Docker image build failures | Low | High | Multi-stage build tested locally, CI build verification |
| Go dependencies conflicts | Low | Medium | Use Go modules with version pinning, `go mod tidy` |
| GitHub API rate limits | Medium | High | Implement rate limit checking, exponential backoff |
| Action input parsing errors | Low | Medium | Validate all inputs at startup, fail fast with clear errors |
| Git command failures | Low | High | Check git availability, validate repo state before operations |
| Large Docker image | Low | Medium | Multi-stage build ensures <50MB target |

## Performance Considerations

**Expected Action Duration**:
- Docker image pull (first time): ~10-15 seconds
- Docker image pull (cached): <1 second
- Go binary startup: <100ms
- Diff parsing: <1 second (even for large files)
- Comment generation: <100ms per change
- API posting: ~1-2 seconds per comment
- **Total**: <30 seconds for typical PRs (<10 changes)

**Optimization Strategies**:
- Static binary compilation for instant startup
- Concurrent API requests using goroutines
- Efficient string processing with strings.Builder
- Minimal allocations in hot paths
- Early exit if no .gitleaksignore changes

## Security Considerations

1. **Token Security**: Never log GITHUB_TOKEN, use oauth2 client for secure transmission
2. **Input Validation**: Sanitize all action inputs before use
3. **Command Injection**: Use `exec.Command` with separate arguments, never shell expansion
4. **Dependency Security**: Regular `go get -u` updates, scan with `govulncheck`
5. **Container Security**: Alpine base for smaller attack surface, no unnecessary packages
6. **Rate Limiting**: Respect GitHub API limits to avoid account restrictions

## Best Practices Applied

1. **Idempotency**: Deduplication ensures safe re-runs
2. **Fail-Fast**: Validate inputs and environment at startup
3. **Observability**: Structured logging with log levels (info, warn, error)
4. **Error Handling**: All errors returned and handled, no panics
5. **Testing**: >80% code coverage target, table-driven tests
6. **Documentation**: GoDoc comments on all exported functions
7. **Code Quality**: `golangci-lint` with strict configuration
8. **Dependency Management**: Minimal dependencies, all via Go modules

## Development Workflow

### Local Development

```bash
# Build
go build -o pr-diff-comment ./cmd/pr-diff-comment

# Run unit tests
go test ./...

# Run with coverage
go test -cover -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Lint
golangci-lint run

# Build Docker image
docker build -t pr-diff-comment .

# Test Docker action locally
docker run --rm \
  -e INPUT_GITHUB-TOKEN=$GITHUB_TOKEN \
  -e INPUT_PR-NUMBER=42 \
  -e GITHUB_REPOSITORY=owner/repo \
  -v $(pwd):/github/workspace \
  pr-diff-comment
```

### Integration Testing with act

```bash
# Install act
brew install act  # macOS
# or download from https://github.com/nektos/act

# Run action locally
act pull_request \
  --secret GITHUB_TOKEN=$GITHUB_TOKEN \
  --eventpath tests/fixtures/pr-event.json
```

## Go Module Dependencies

```go
// go.mod
module github.com/your-org/pr-gitleaks-commenter

go 1.21

require (
    github.com/google/go-github/v57 v57.0.0
    golang.org/x/oauth2 v0.15.0
)

// Test dependencies
require (
    github.com/stretchr/testify v1.8.4
)
```

## Action Distribution

Custom actions can be distributed in three ways:

1. **Same repository** (chosen for this feature):
   ```yaml
   steps:
     - uses: ./.github/actions/pr-gitleaks-comment
   ```

2. **Separate repository**:
   ```yaml
   steps:
     - uses: your-org/pr-gitleaks-comment@v1
   ```

3. **GitHub Marketplace** (future consideration):
   - Requires public repository
   - Additional metadata in action.yml
   - Release tags for versioning

## Next Steps Post-Research

With all technical decisions made:
1. Create data model with Go struct definitions
2. Define contracts with Go interfaces and action.yml
3. Generate quickstart guide for action usage
4. Update agent context with Go/Docker stack
