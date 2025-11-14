package github

import (
	"context"
	"errors"
	"fmt"

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
func (c *ClientImpl) CheckRateLimit(ctx context.Context) (int, error) {
	rate, _, err := c.client.RateLimit.Get(ctx)
	if err != nil {
		return 0, err
	}

	return rate.Core.Remaining, nil
}
