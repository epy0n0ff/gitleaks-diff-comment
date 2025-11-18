package github

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/google/go-github/v57/github"
	"golang.org/x/oauth2"
)

// Client defines the interface for GitHub API operations
type Client interface {
	// CreateReviewComment posts a line-level review comment on a PR
	CreateReviewComment(ctx context.Context, req *PostCommentRequest) (*PostCommentResponse, error)

	// UpdateReviewComment updates an existing review comment
	UpdateReviewComment(ctx context.Context, req *UpdateCommentRequest) (*PostCommentResponse, error)

	// ListReviewComments fetches all review comments for a PR
	ListReviewComments(ctx context.Context) ([]*ExistingComment, error)

	// CreateIssueComment posts a PR-level comment (fallback)
	CreateIssueComment(ctx context.Context, body string) (*PostCommentResponse, error)

	// CheckRateLimit returns remaining API calls
	CheckRateLimit(ctx context.Context) (int, error)

	// ListPRComments fetches all issue comments for a PR
	ListPRComments(ctx context.Context) ([]*github.IssueComment, error)

	// DeleteComment deletes a comment by ID
	DeleteComment(ctx context.Context, commentID int64) error
}

// ClientImpl is the concrete implementation using go-github
type ClientImpl struct {
	client   *github.Client
	owner    string
	repo     string
	prNumber int
}

// NewClient creates a new GitHub API client
func NewClient(token, owner, repo string, prNumber int, ghHost string) (Client, error) {
	if token == "" {
		return nil, errors.New("GitHub token is required")
	}
	if owner == "" {
		return nil, errors.New("owner is required")
	}
	if repo == "" {
		return nil, errors.New("repo is required")
	}
	if prNumber <= 0 {
		return nil, errors.New("PR number must be positive")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	// Create GitHub client (enterprise or default)
	var ghClient *github.Client
	var err error

	if ghHost != "" {
		// GitHub Enterprise Server
		baseURL := "https://" + ghHost
		uploadURL := "https://" + ghHost

		ghClient, err = github.NewClient(tc).WithEnterpriseURLs(baseURL, uploadURL)
		if err != nil {
			return nil, fmt.Errorf("failed to create GitHub Enterprise client for %s: %w", ghHost, err)
		}
	} else {
		// GitHub.com (default)
		ghClient = github.NewClient(tc)
	}

	return &ClientImpl{
		client:   ghClient,
		owner:    owner,
		repo:     repo,
		prNumber: prNumber,
	}, nil
}

// isAuthError checks if an error is related to authentication
func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "401") ||
		strings.Contains(errMsg, "unauthorized") ||
		strings.Contains(errMsg, "authentication") ||
		strings.Contains(errMsg, "bad credentials")
}

// isNetworkError checks if an error is related to network connectivity
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	// Check for network-related errors
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "no such host") ||
		strings.Contains(errMsg, "timeout") ||
		strings.Contains(errMsg, "network")
}

// enhanceError adds context to errors based on error type
func enhanceError(err error, ghHost string) error {
	if err == nil {
		return nil
	}

	if isAuthError(err) {
		if ghHost != "" {
			return fmt.Errorf("authentication failed for GitHub Enterprise Server at %s\n"+
				"  → Action: Verify token has required permissions (repo, pull_requests)\n"+
				"  → Check: Token is valid for enterprise instance\n"+
				"  → Original error: %w", ghHost, err)
		}
		return fmt.Errorf("authentication failed for GitHub.com\n"+
			"  → Action: Verify token has required permissions (repo, pull_requests)\n"+
			"  → Check: Token is valid and not expired\n"+
			"  → Original error: %w", err)
	}

	if isNetworkError(err) {
		if ghHost != "" {
			return fmt.Errorf("cannot connect to GitHub Enterprise Server at %s\n"+
				"  → Action: Verify hostname is correct and server is reachable\n"+
				"  → Check: Network connectivity, firewall rules, DNS resolution\n"+
				"  → Original error: %w", ghHost, err)
		}
		return fmt.Errorf("cannot connect to GitHub.com\n"+
			"  → Action: Check network connectivity\n"+
			"  → Original error: %w", err)
	}

	// Return original error with minimal context
	return err
}

// CreateReviewComment posts a line-level review comment on a PR
func (c *ClientImpl) CreateReviewComment(ctx context.Context, req *PostCommentRequest) (*PostCommentResponse, error) {
	comment := &github.PullRequestComment{
		Body:     github.String(req.Body),
		CommitID: github.String(req.CommitID),
		Path:     github.String(req.Path),
	}

	// Use Line-based API (recommended) instead of deprecated Position-based API
	if req.Line > 0 && req.Side != "" {
		comment.Line = github.Int(req.Line)
		comment.Side = github.String(req.Side)
	} else {
		// Fallback to Position for backwards compatibility
		comment.Position = github.Int(req.Position)
	}

	created, _, err := c.client.PullRequests.CreateComment(ctx, c.owner, c.repo, c.prNumber, comment)
	if err != nil {
		return nil, err
	}

	return &PostCommentResponse{
		ID:        created.GetID(),
		HTMLURL:   created.GetHTMLURL(),
		CreatedAt: created.GetCreatedAt().Time,
	}, nil
}

// UpdateReviewComment updates an existing review comment
func (c *ClientImpl) UpdateReviewComment(ctx context.Context, req *UpdateCommentRequest) (*PostCommentResponse, error) {
	comment := &github.PullRequestComment{
		Body: github.String(req.Body),
	}

	updated, _, err := c.client.PullRequests.EditComment(ctx, c.owner, c.repo, req.CommentID, comment)
	if err != nil {
		return nil, err
	}

	return &PostCommentResponse{
		ID:        updated.GetID(),
		HTMLURL:   updated.GetHTMLURL(),
		CreatedAt: updated.GetCreatedAt().Time,
	}, nil
}

// ListReviewComments fetches all review comments for a PR
func (c *ClientImpl) ListReviewComments(ctx context.Context) ([]*ExistingComment, error) {
	opts := &github.PullRequestListCommentsOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var allComments []*ExistingComment
	for {
		comments, resp, err := c.client.PullRequests.ListComments(ctx, c.owner, c.repo, c.prNumber, opts)
		if err != nil {
			return nil, err
		}

		for _, comment := range comments {
			allComments = append(allComments, &ExistingComment{
				ID:       comment.GetID(),
				Body:     comment.GetBody(),
				Path:     comment.GetPath(),
				Position: comment.GetPosition(),
				Line:     comment.GetLine(),
				Side:     comment.GetSide(),
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allComments, nil
}

// CreateIssueComment posts a PR-level comment (fallback)
func (c *ClientImpl) CreateIssueComment(ctx context.Context, body string) (*PostCommentResponse, error) {
	comment := &github.IssueComment{
		Body: github.String(body),
	}

	created, _, err := c.client.Issues.CreateComment(ctx, c.owner, c.repo, c.prNumber, comment)
	if err != nil {
		return nil, err
	}

	return &PostCommentResponse{
		ID:        created.GetID(),
		HTMLURL:   created.GetHTMLURL(),
		CreatedAt: created.GetCreatedAt().Time,
	}, nil
}

// CheckRateLimit returns remaining API calls
// Note: Automatically reads rate limit headers from any GitHub instance (including enterprise)
func (c *ClientImpl) CheckRateLimit(ctx context.Context) (int, error) {
	rate, _, err := c.client.RateLimit.Get(ctx)
	if err != nil {
		return 0, err
	}

	// The go-github library automatically parses X-RateLimit-* headers
	// from both GitHub.com and GitHub Enterprise Server responses
	return rate.Core.Remaining, nil
}

// ListPRComments fetches all issue comments for a pull request
// PR comments are actually issue comments in the GitHub API
func (c *ClientImpl) ListPRComments(ctx context.Context) ([]*github.IssueComment, error) {
	opts := &github.IssueListCommentsOptions{
		ListOptions: github.ListOptions{
			PerPage: 100, // Maximum allowed per page
		},
	}

	var allComments []*github.IssueComment

	for {
		comments, resp, err := c.client.Issues.ListComments(ctx, c.owner, c.repo, c.prNumber, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to list comments: %w", err)
		}

		allComments = append(allComments, comments...)

		// Check if there are more pages
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allComments, nil
}

// DeleteComment deletes an issue comment by ID
// Handles 404 errors gracefully (comment already deleted)
func (c *ClientImpl) DeleteComment(ctx context.Context, commentID int64) error {
	_, err := c.client.Issues.DeleteComment(ctx, c.owner, c.repo, commentID)
	if err != nil {
		// Check if it's a 404 (comment already deleted)
		if strings.Contains(err.Error(), "404") {
			// Not an error - comment is already gone
			return nil
		}
		return fmt.Errorf("failed to delete comment %d: %w", commentID, err)
	}
	return nil
}
