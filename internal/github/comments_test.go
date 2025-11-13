package github

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/pr-gitleaks-commenter/pr-diff-comment/internal/comment"
	"github.com/pr-gitleaks-commenter/pr-diff-comment/internal/diff"
)

// MockClient is a mock implementation of the GitHub Client interface
type MockClient struct {
	CreateReviewCommentFunc func(ctx context.Context, req *PostCommentRequest) (*PostCommentResponse, error)
	ListReviewCommentsFunc  func(ctx context.Context) ([]*ExistingComment, error)
	CreateIssueCommentFunc  func(ctx context.Context, body string) (*PostCommentResponse, error)
	CheckRateLimitFunc      func(ctx context.Context) (int, error)
}

func (m *MockClient) CreateReviewComment(ctx context.Context, req *PostCommentRequest) (*PostCommentResponse, error) {
	if m.CreateReviewCommentFunc != nil {
		return m.CreateReviewCommentFunc(ctx, req)
	}
	return &PostCommentResponse{ID: 123, HTMLURL: "https://github.com/test"}, nil
}

func (m *MockClient) ListReviewComments(ctx context.Context) ([]*ExistingComment, error) {
	if m.ListReviewCommentsFunc != nil {
		return m.ListReviewCommentsFunc(ctx)
	}
	return []*ExistingComment{}, nil
}

func (m *MockClient) CreateIssueComment(ctx context.Context, body string) (*PostCommentResponse, error) {
	if m.CreateIssueCommentFunc != nil {
		return m.CreateIssueCommentFunc(ctx, body)
	}
	return &PostCommentResponse{ID: 123, HTMLURL: "https://github.com/test"}, nil
}

func (m *MockClient) CheckRateLimit(ctx context.Context) (int, error) {
	if m.CheckRateLimitFunc != nil {
		return m.CheckRateLimitFunc(ctx)
	}
	return 5000, nil
}

func TestPostComments_Concurrency(t *testing.T) {
	// Create 20 test comments
	comments := make([]*comment.GeneratedComment, 20)
	for i := 0; i < 20; i++ {
		comments[i] = &comment.GeneratedComment{
			Body:     fmt.Sprintf("Test comment %d", i),
			Path:     ".gitleaksignore",
			Position: i + 1,
			CommitID: "abc123",
		}
	}

	// Track concurrent API calls
	var mu sync.Mutex
	var maxConcurrent int
	var currentConcurrent int

	mockClient := &MockClient{
		CreateReviewCommentFunc: func(ctx context.Context, req *PostCommentRequest) (*PostCommentResponse, error) {
			mu.Lock()
			currentConcurrent++
			if currentConcurrent > maxConcurrent {
				maxConcurrent = currentConcurrent
			}
			mu.Unlock()

			// Simulate API delay
			time.Sleep(10 * time.Millisecond)

			mu.Lock()
			currentConcurrent--
			mu.Unlock()

			return &PostCommentResponse{
				ID:      int64(123 + currentConcurrent),
				HTMLURL: "https://github.com/test/comment",
			}, nil
		},
		ListReviewCommentsFunc: func(ctx context.Context) ([]*ExistingComment, error) {
			return []*ExistingComment{}, nil
		},
		CheckRateLimitFunc: func(ctx context.Context) (int, error) {
			return 5000, nil
		},
	}

	ctx := context.Background()
	output, err := PostComments(ctx, mockClient, comments, false)

	if err != nil {
		t.Fatalf("PostComments() unexpected error: %v", err)
	}

	// Verify results
	if output.Posted != 20 {
		t.Errorf("Expected 20 posted comments, got %d", output.Posted)
	}

	// Verify concurrency is limited to 5
	if maxConcurrent > 5 {
		t.Errorf("Max concurrent requests exceeded limit: got %d, want <= 5", maxConcurrent)
	}

	if maxConcurrent < 2 {
		t.Errorf("Expected some concurrency, got max %d", maxConcurrent)
	}
}

func TestPostComments_RateLimitRetry(t *testing.T) {
	comments := []*comment.GeneratedComment{
		{
			Body:     "Test comment",
			Path:     ".gitleaksignore",
			Position: 1,
			CommitID: "abc123",
		},
	}

	attemptCount := 0
	mockClient := &MockClient{
		CreateReviewCommentFunc: func(ctx context.Context, req *PostCommentRequest) (*PostCommentResponse, error) {
			attemptCount++
			if attemptCount < 3 {
				// Simulate rate limit error for first 2 attempts
				return nil, errors.New("rate limit exceeded")
			}
			// Succeed on 3rd attempt
			return &PostCommentResponse{ID: 123, HTMLURL: "https://github.com/test"}, nil
		},
		ListReviewCommentsFunc: func(ctx context.Context) ([]*ExistingComment, error) {
			return []*ExistingComment{}, nil
		},
	}

	ctx := context.Background()
	output, err := PostComments(ctx, mockClient, comments, true)

	if err != nil {
		t.Fatalf("PostComments() unexpected error: %v", err)
	}

	// Should succeed after retries
	if output.Posted != 1 {
		t.Errorf("Expected 1 posted comment after retries, got %d", output.Posted)
	}

	if attemptCount != 3 {
		t.Errorf("Expected 3 attempts (2 retries), got %d", attemptCount)
	}
}

func TestPostComments_MaxRetriesExceeded(t *testing.T) {
	comments := []*comment.GeneratedComment{
		{
			Body:     "Test comment",
			Path:     ".gitleaksignore",
			Position: 1,
			CommitID: "abc123",
		},
	}

	attemptCount := 0
	mockClient := &MockClient{
		CreateReviewCommentFunc: func(ctx context.Context, req *PostCommentRequest) (*PostCommentResponse, error) {
			attemptCount++
			// Always fail with rate limit
			return nil, errors.New("rate limit exceeded")
		},
		ListReviewCommentsFunc: func(ctx context.Context) ([]*ExistingComment, error) {
			return []*ExistingComment{}, nil
		},
	}

	ctx := context.Background()
	output, err := PostComments(ctx, mockClient, comments, false)

	if err != nil {
		t.Fatalf("PostComments() unexpected error: %v", err)
	}

	// Should record error after max retries
	if output.Errors != 1 {
		t.Errorf("Expected 1 error after max retries, got %d", output.Errors)
	}

	// Should have attempted 4 times (1 initial + 3 retries)
	if attemptCount != 4 {
		t.Errorf("Expected 4 attempts (1 initial + 3 retries), got %d", attemptCount)
	}
}

func TestPostComments_Deduplication(t *testing.T) {
	comments := []*comment.GeneratedComment{
		{
			Body:     "Test comment",
			Path:     ".gitleaksignore",
			Position: 1,
			CommitID: "abc123",
		},
	}

	mockClient := &MockClient{
		CreateReviewCommentFunc: func(ctx context.Context, req *PostCommentRequest) (*PostCommentResponse, error) {
			t.Error("CreateReviewComment should not be called for duplicate")
			return nil, errors.New("should not be called")
		},
		ListReviewCommentsFunc: func(ctx context.Context) ([]*ExistingComment, error) {
			// Return existing comment that matches
			return []*ExistingComment{
				{
					ID:       999,
					Body:     "Test comment",
					Path:     ".gitleaksignore",
					Position: 1,
				},
			}, nil
		},
	}

	ctx := context.Background()
	output, err := PostComments(ctx, mockClient, comments, false)

	if err != nil {
		t.Fatalf("PostComments() unexpected error: %v", err)
	}

	// Should skip duplicate
	if output.SkippedDuplicates != 1 {
		t.Errorf("Expected 1 skipped duplicate, got %d", output.SkippedDuplicates)
	}

	if output.Posted != 0 {
		t.Errorf("Expected 0 posted comments, got %d", output.Posted)
	}
}

func TestIsDuplicate(t *testing.T) {
	tests := []struct {
		name            string
		newComment      *comment.GeneratedComment
		existingComment *ExistingComment
		expected        bool
	}{
		{
			name: "exact match",
			newComment: &comment.GeneratedComment{
				Body:     "Test comment",
				Path:     ".gitleaksignore",
				Position: 1,
			},
			existingComment: &ExistingComment{
				Body:     "Test comment",
				Path:     ".gitleaksignore",
				Position: 1,
			},
			expected: true,
		},
		{
			name: "different position",
			newComment: &comment.GeneratedComment{
				Body:     "Test comment",
				Path:     ".gitleaksignore",
				Position: 1,
			},
			existingComment: &ExistingComment{
				Body:     "Test comment",
				Path:     ".gitleaksignore",
				Position: 2,
			},
			expected: false,
		},
		{
			name: "different body",
			newComment: &comment.GeneratedComment{
				Body:     "Test comment A",
				Path:     ".gitleaksignore",
				Position: 1,
			},
			existingComment: &ExistingComment{
				Body:     "Test comment B",
				Path:     ".gitleaksignore",
				Position: 1,
			},
			expected: false,
		},
		{
			name: "whitespace normalized match",
			newComment: &comment.GeneratedComment{
				Body:     "Test   comment\n\nwith spaces",
				Path:     ".gitleaksignore",
				Position: 1,
			},
			existingComment: &ExistingComment{
				Body:     "Test comment with spaces",
				Path:     ".gitleaksignore",
				Position: 1,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			existing := []*ExistingComment{tt.existingComment}
			result := isDuplicate(tt.newComment, existing)
			if result != tt.expected {
				t.Errorf("isDuplicate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNormalizeWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "multiple spaces",
			input:    "Test   comment   here",
			expected: "Test comment here",
		},
		{
			name:     "newlines",
			input:    "Test\ncomment\n\nhere",
			expected: "Test comment here",
		},
		{
			name:     "tabs and mixed whitespace",
			input:    "Test\t  comment\n\there",
			expected: "Test comment here",
		},
		{
			name:     "leading and trailing whitespace",
			input:    "  Test comment  ",
			expected: "Test comment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeWhitespace(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeWhitespace() = %q, want %q", result, tt.expected)
			}
		})
	}
}
