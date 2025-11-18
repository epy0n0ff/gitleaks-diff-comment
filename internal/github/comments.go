package github

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/epy0n0ff/gitleaks-diff-comment/internal/comment"
	"github.com/google/go-github/v57/github"
)

// IsBotComment checks if a comment was created by the gitleaks-diff-comment bot
// It uses two identification methods:
// 1. Primary: Check for invisible HTML marker in comment body
// 2. Fallback: Check if comment author is "github-actions[bot]"
func IsBotComment(comment *github.IssueComment) bool {
	if comment == nil {
		return false
	}

	body := comment.GetBody()

	// Primary: Check for invisible marker
	// All bot comments include: <!-- gitleaks-diff-comment: ... -->
	if strings.Contains(body, "<!-- gitleaks-diff-comment:") {
		return true
	}

	// Fallback: Check comment author
	// Handles old comments that may not have the marker
	if comment.GetUser().GetLogin() == "github-actions[bot]" {
		return true
	}

	return false
}

// FilterBotComments separates bot comments from human comments
// Returns only comments that were created by the gitleaks-diff-comment bot
func FilterBotComments(comments []*github.IssueComment) []*github.IssueComment {
	var botComments []*github.IssueComment

	for _, comment := range comments {
		if IsBotComment(comment) {
			botComments = append(botComments, comment)
		}
	}

	return botComments
}

// IsBotReviewComment checks if a review comment was created by the gitleaks-diff-comment bot
// Review comments are the diff comments posted on specific lines of code
func IsBotReviewComment(comment *github.PullRequestComment) bool {
	if comment == nil {
		return false
	}

	body := comment.GetBody()

	// Primary: Check for invisible marker
	// All bot comments include: <!-- gitleaks-diff-comment: ... -->
	if strings.Contains(body, "<!-- gitleaks-diff-comment:") {
		return true
	}

	// Fallback: Check comment author
	// Handles old comments that may not have the marker
	if comment.GetUser().GetLogin() == "github-actions[bot]" {
		return true
	}

	return false
}

// FilterBotReviewComments separates bot review comments from human review comments
// Returns only review comments that were created by the gitleaks-diff-comment bot
func FilterBotReviewComments(comments []*github.PullRequestComment) []*github.PullRequestComment {
	var botComments []*github.PullRequestComment

	for _, comment := range comments {
		if IsBotReviewComment(comment) {
			botComments = append(botComments, comment)
		}
	}

	return botComments
}

// PostComments posts multiple comments concurrently with rate limiting and deduplication
func PostComments(ctx context.Context, client Client, comments []*comment.GeneratedComment, commentMode string, debug bool) (*ActionOutput, error) {
	// Fetch existing comments for deduplication
	existingComments, err := client.ListReviewComments(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list existing comments: %w", err)
	}

	// Check rate limit before starting
	// Note: Respects rate limits from both GitHub.com and GitHub Enterprise Server
	remaining, err := client.CheckRateLimit(ctx)
	if err != nil {
		log.Printf("Warning: failed to check rate limit: %v", err)
	} else if debug {
		log.Printf("GitHub API rate limit remaining: %d calls", remaining)
	}

	if debug {
		log.Printf("Comment mode: %s", commentMode)
	}

	// Post comments concurrently with semaphore
	results := postCommentsConcurrently(ctx, client, comments, existingComments, commentMode, debug)

	// Aggregate results
	output := &ActionOutput{
		Results: results,
	}

	for _, result := range results {
		switch result.Status {
		case "posted":
			output.Posted++
		case "updated":
			output.Posted++ // Count updates as posted
		case "skipped_duplicate":
			output.SkippedDuplicates++
		case "error":
			output.Errors++
		}
	}

	if debug {
		log.Printf("Summary: Posted=%d, Skipped=%d, Errors=%d", output.Posted, output.SkippedDuplicates, output.Errors)
	}

	return output, nil
}

// postCommentsConcurrently posts comments with controlled concurrency
func postCommentsConcurrently(ctx context.Context, client Client, comments []*comment.GeneratedComment, existingComments []*ExistingComment, commentMode string, debug bool) []CommentResult {
	var wg sync.WaitGroup
	resultChan := make(chan CommentResult, len(comments))

	// Limit concurrency to 5 to avoid overwhelming GitHub API
	semaphore := make(chan struct{}, 5)

	for i, c := range comments {
		wg.Add(1)
		go func(idx int, comm *comment.GeneratedComment) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Check for existing comment with same content
			existingComment := findExistingComment(comm, existingComments)

			if commentMode == "override" && existingComment != nil {
				// Check if line number has changed
				if existingComment.Line != comm.Line {
					// Line has shifted - delete old comment and post new one at correct line
					if debug {
						log.Printf("[%d/%d] Line shifted (%d → %d), replacing comment", idx+1, len(comments), existingComment.Line, comm.Line)
					}
					// Delete old comment (best effort, ignore errors)
					_ = client.DeleteReviewComment(ctx, existingComment.ID)
					// Post new comment at correct line
					result := postCommentWithRetry(ctx, client, comm, debug, idx+1, len(comments))
					resultChan <- result
					return
				}

				// Same line - update existing comment body
				if debug {
					log.Printf("[%d/%d] Updating existing comment at line %d (%s)", idx+1, len(comments), comm.Line, comm.Side)
				}
				result := updateCommentWithRetry(ctx, client, comm, existingComment.ID, debug, idx+1, len(comments))
				resultChan <- result
				return
			}

			if commentMode == "append" && existingComment != nil {
				// Append mode: skip if duplicate exists
				if isDuplicateContent(comm, existingComment) {
					if debug {
						log.Printf("[%d/%d] Skipping duplicate comment at line %d (%s)", idx+1, len(comments), comm.Line, comm.Side)
					}
					resultChan <- CommentResult{
						Status:      "skipped_duplicate",
						BodyPreview: comm.GetBodyPreview(),
					}
					return
				}
			}

			// Post new comment with retry logic
			result := postCommentWithRetry(ctx, client, comm, debug, idx+1, len(comments))
			resultChan <- result
		}(i, c)
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results with progress logging for large batches
	var results []CommentResult
	totalComments := len(comments)
	processedCount := 0
	postedCount := 0

	for result := range resultChan {
		results = append(results, result)
		processedCount++

		if result.Status == "posted" {
			postedCount++
		}

		// Log progress every 10 comments for large batches (20+ comments)
		if totalComments >= 20 && processedCount%10 == 0 {
			log.Printf("Progress: %d/%d comments processed, %d posted", processedCount, totalComments, postedCount)
		}
	}

	// Final progress log for large batches
	if totalComments >= 20 {
		log.Printf("Completed: %d/%d comments processed, %d posted", processedCount, totalComments, postedCount)
	}

	return results
}

// postCommentWithRetry posts a comment with exponential backoff retry
func postCommentWithRetry(ctx context.Context, client Client, comm *comment.GeneratedComment, debug bool, idx, total int) CommentResult {
	req := &PostCommentRequest{
		Body:     comm.Body,
		CommitID: comm.CommitID,
		Path:     comm.Path,
		Line:     comm.Line,
		Side:     comm.Side,
		Position: comm.Position, // Kept for backwards compatibility
	}

	maxRetries := 3
	delays := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			if debug {
				log.Printf("[%d/%d] Retry attempt %d after %v", idx, total, attempt, delays[attempt-1])
			}
			time.Sleep(delays[attempt-1])
		}

		resp, err := client.CreateReviewComment(ctx, req)
		if err != nil {
			// Check if this is a rate limit error
			if strings.Contains(err.Error(), "rate limit") || strings.Contains(err.Error(), "abuse") {
				if attempt < maxRetries {
					log.Printf("[%d/%d] Rate limit hit, retrying...", idx, total)
					continue
				}
			}

			// Final failure
			if debug {
				log.Printf("[%d/%d] Failed to post comment: %v", idx, total, err)
			}
			return CommentResult{
				Status:      "error",
				Error:       err.Error(),
				BodyPreview: comm.GetBodyPreview(),
			}
		}

		// Success
		if debug {
			log.Printf("[%d/%d] Posted comment at line %d (%s): %s", idx, total, comm.Line, comm.Side, resp.HTMLURL)
		}
		return CommentResult{
			Status:      "posted",
			CommentID:   resp.ID,
			CommentURL:  resp.HTMLURL,
			BodyPreview: comm.GetBodyPreview(),
		}
	}

	// Should not reach here, but handle gracefully
	return CommentResult{
		Status:      "error",
		Error:       "max retries exceeded",
		BodyPreview: comm.GetBodyPreview(),
	}
}

// findExistingComment finds an existing comment at the same location using the marker
func findExistingComment(newComment *comment.GeneratedComment, existingComments []*ExistingComment) *ExistingComment {
	// Extract marker from new comment body
	marker := extractMarker(newComment.Body)
	if marker == "" {
		return nil
	}

	// Find comment with matching marker
	for _, existing := range existingComments {
		if extractMarker(existing.Body) == marker {
			return existing
		}
	}

	return nil
}

// extractMarker extracts the marker from comment body
// Marker format: <!-- gitleaks-diff-comment: {path}:{content}:{side} -->
// Content is the actual gitleaks pattern (e.g., "secret.txt" or "*.env")
func extractMarker(body string) string {
	start := strings.Index(body, "<!-- gitleaks-diff-comment: ")
	if start == -1 {
		return ""
	}
	end := strings.Index(body[start:], " -->")
	if end == -1 {
		return ""
	}
	return body[start : start+end+4] // Include " -->"
}

// isDuplicateContent checks if comment content is duplicate (for append mode)
func isDuplicateContent(newComment *comment.GeneratedComment, existingComment *ExistingComment) bool {
	// Normalize whitespace for comparison
	existingBody := normalizeWhitespace(existingComment.Body)
	newBody := normalizeWhitespace(newComment.Body)
	return existingBody == newBody
}

// updateCommentWithRetry updates a comment with exponential backoff retry
func updateCommentWithRetry(ctx context.Context, client Client, comm *comment.GeneratedComment, commentID int64, debug bool, idx, total int) CommentResult {
	req := &UpdateCommentRequest{
		CommentID: commentID,
		Body:      comm.Body,
	}

	maxRetries := 3
	delays := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			if debug {
				log.Printf("[%d/%d] Retry attempt %d after %v", idx, total, attempt, delays[attempt-1])
			}
			time.Sleep(delays[attempt-1])
		}

		resp, err := client.UpdateReviewComment(ctx, req)
		if err != nil {
			// Check if this is a rate limit error
			if strings.Contains(err.Error(), "rate limit") || strings.Contains(err.Error(), "abuse") {
				if attempt < maxRetries {
					log.Printf("[%d/%d] Rate limit hit, retrying...", idx, total)
					continue
				}
			}

			// Final failure
			if debug {
				log.Printf("[%d/%d] Failed to update comment: %v", idx, total, err)
			}
			return CommentResult{
				Status:      "error",
				Error:       err.Error(),
				BodyPreview: comm.GetBodyPreview(),
			}
		}

		// Success
		if debug {
			log.Printf("[%d/%d] Updated comment at line %d (%s): %s", idx, total, comm.Line, comm.Side, resp.HTMLURL)
		}
		return CommentResult{
			Status:      "updated",
			CommentID:   resp.ID,
			CommentURL:  resp.HTMLURL,
			BodyPreview: comm.GetBodyPreview(),
		}
	}

	// Should not reach here, but handle gracefully
	return CommentResult{
		Status:      "error",
		Error:       "max retries exceeded",
		BodyPreview: comm.GetBodyPreview(),
	}
}

// normalizeWhitespace normalizes whitespace for comparison
func normalizeWhitespace(s string) string {
	// Replace multiple whitespace with single space
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}
