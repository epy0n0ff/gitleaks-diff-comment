# Stage 1: Build
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Cache dependencies
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o pr-diff-comment \
    ./cmd/pr-diff-comment

# Stage 2: Runtime
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache git ca-certificates

# Copy binary from builder
COPY --from=builder /build/pr-diff-comment /usr/local/bin/pr-diff-comment

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/pr-diff-comment"]
