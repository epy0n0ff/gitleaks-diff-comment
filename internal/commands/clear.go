package commands

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/epy0n0ff/gitleaks-diff-comment/internal/github"
)

// ClearOperation tracks the execution state of a clear command
type ClearOperation struct {
	// CommandID is a unique identifier for this operation
	CommandID string

	// PRNumber is the pull request number
	PRNumber int

	// RequestedBy is the user who initiated the operation
	RequestedBy string

	// StartedAt is the operation start timestamp
	StartedAt time.Time

	// CompletedAt is the operation completion timestamp (nil if in progress)
	CompletedAt time.Time

	// Status is the operation status (pending/running/completed/failed)
	Status string

	// CommentsFound is the total bot comments found
	CommentsFound int

	// CommentsDeleted is the number of successfully deleted comments
	CommentsDeleted int

	// CommentsFailed is the number of failed deletion attempts
	CommentsFailed int

	// Errors is a list of error messages encountered
	Errors []string

	// RetryCount is the number of retry attempts made
	RetryCount int

	// Duration is the total operation time in seconds
	Duration float64
}

// ClearCommand handles the execution of a /clear command
type ClearCommand struct {
	// PRNumber is the pull request to clear comments from
	PRNumber int

	// RequestedBy is the GitHub username who requested the command
	RequestedBy string

	// CommentID is the comment ID that triggered this command
	CommentID int64

	// Client is the GitHub API client
	Client github.Client

	// Operation tracks execution state
	Operation *ClearOperation
}

// NewClearCommand creates a new clear command instance
func NewClearCommand(prNumber int, requestedBy string, commentID int64, client github.Client) *ClearCommand {
	return &ClearCommand{
		PRNumber:    prNumber,
		RequestedBy: requestedBy,
		CommentID:   commentID,
		Client:      client,
		Operation: &ClearOperation{
			CommandID:   fmt.Sprintf("clear-%d-%d", prNumber, time.Now().Unix()),
			PRNumber:    prNumber,
			RequestedBy: requestedBy,
			StartedAt:   time.Now(),
			Status:      "pending",
		},
	}
}

// Execute runs the clear command
// 1. Check user permissions
// 2. Fetch all PR comments
// 3. Filter to bot comments only
// 4. Delete each bot comment
// 5. Track results and errors
func (c *ClearCommand) Execute(ctx context.Context) error {
	c.Operation.Status = "running"
	log.Printf("::notice::Starting clear command for PR #%d (requested by %s)", c.PRNumber, c.RequestedBy)

	// Check user permissions
	authorized, permissionLevel, err := c.Client.CheckUserPermission(ctx, c.RequestedBy)
	if err != nil {
		c.Operation.Status = "failed"
		c.Operation.Errors = append(c.Operation.Errors, err.Error())
		return fmt.Errorf("failed to check permissions: %w", err)
	}

	if !authorized {
		c.Operation.Status = "failed"
		errUnauth := NewErrUnauthorized(c.RequestedBy, permissionLevel)
		c.Operation.Errors = append(c.Operation.Errors, errUnauth.Error())
		c.finalize()
		return errUnauth
	}

	log.Printf("::notice::Permission check passed: %s has %s access", c.RequestedBy, permissionLevel)

	// Fetch all review comments (diff comments) for the PR
	// These are the comments posted on specific lines of code
	reviewComments, err := c.Client.ListPRReviewComments(ctx)
	if err != nil {
		c.Operation.Status = "failed"
		c.Operation.Errors = append(c.Operation.Errors, err.Error())
		c.finalize()
		c.logMetricsOnError()
		log.Printf("::error::Failed to fetch review comments: %v", err)
		return fmt.Errorf("failed to fetch review comments: %w", err)
	}

	// Filter to bot comments only
	botComments := github.FilterBotReviewComments(reviewComments)
	c.Operation.CommentsFound = len(botComments)

	log.Printf("::notice::Found %d bot review comments to delete", len(botComments))

	if len(botComments) == 0 {
		c.Operation.Status = "completed"
		c.finalize()
		c.logMetricsOnCompletion()
		log.Println("::notice::No bot comments found to delete")
		return nil
	}

	// Delete each bot comment with retry logic
	for _, comment := range botComments {
		commentID := comment.GetID()

		// Use retry with backoff for rate limit handling
		retries, err := c.deleteCommentWithRetry(ctx, commentID)

		// Track total retry attempts
		c.Operation.RetryCount += retries

		if err != nil {
			// Log error but continue with other comments
			errMsg := fmt.Sprintf("Failed to delete comment %d after %d retries: %v", commentID, retries, err)
			log.Printf("::warning::%s", errMsg)
			c.Operation.Errors = append(c.Operation.Errors, errMsg)
			c.Operation.CommentsFailed++
		} else {
			if retries > 0 {
				log.Printf("::notice::Deleted comment %d (after %d retries)", commentID, retries)
			} else {
				log.Printf("::notice::Deleted comment %d", commentID)
			}
			c.Operation.CommentsDeleted++
		}
	}

	// Finalize operation
	if c.Operation.CommentsFailed > 0 {
		c.Operation.Status = "completed" // Partial success is still completion
	} else {
		c.Operation.Status = "completed"
	}

	c.finalize()

	// Log metrics
	c.logMetricsOnCompletion()

	// Report results
	if c.Operation.CommentsFailed > 0 {
		log.Printf("::notice::✓ Cleared %d comments with %d failures in %.2fs",
			c.Operation.CommentsDeleted, c.Operation.CommentsFailed, c.Operation.Duration)
		return fmt.Errorf("completed with %d failures", c.Operation.CommentsFailed)
	}

	log.Printf("::notice::✓ Successfully cleared %d comments in %.2fs",
		c.Operation.CommentsDeleted, c.Operation.Duration)

	return nil
}

// finalize completes the operation and calculates duration
func (c *ClearCommand) finalize() {
	c.Operation.CompletedAt = time.Now()
	c.Operation.Duration = c.Operation.CompletedAt.Sub(c.Operation.StartedAt).Seconds()
}

// logMetricsOnCompletion logs metrics for successful or partially successful operations
func (c *ClearCommand) logMetricsOnCompletion() {
	event := NewMetricsEvent(c.Operation)
	if err := logMetrics(event); err != nil {
		log.Printf("::warning::Failed to log metrics: %v", err)
	}
}

// logMetricsOnError logs metrics for failed operations
func (c *ClearCommand) logMetricsOnError() {
	event := NewMetricsEvent(c.Operation)
	event.Success = false
	if err := logMetrics(event); err != nil {
		log.Printf("::warning::Failed to log metrics: %v", err)
	}
}

// deleteCommentWithRetry deletes a review comment with exponential backoff retry
// Returns (retryAttempts, error)
func (c *ClearCommand) deleteCommentWithRetry(ctx context.Context, commentID int64) (int, error) {
	maxRetries := 3

	retries, err := github.RetryWithBackoff(func() error {
		return c.Client.DeleteReviewComment(ctx, commentID)
	}, maxRetries)

	// Log retry attempts if any occurred
	if retries > 0 {
		if err != nil {
			log.Printf("::warning::Rate limit encountered for comment %d, failed after %d retries", commentID, retries)
		} else {
			log.Printf("::notice::Rate limit encountered for comment %d, succeeded after %d retries", commentID, retries)
		}
	}

	return retries, err
}
