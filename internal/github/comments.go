package github

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/epy0n0ff/gitleaks-diff-comment/internal/comment"
)

// PostComments posts multiple comments concurrently with rate limiting and deduplication
func PostComments(ctx context.Context, client Client, comments []*comment.GeneratedComment, debug bool) (*ActionOutput, error) {
	// Fetch existing comments for deduplication
	existingComments, err := client.ListReviewComments(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list existing comments: %w", err)
	}

	// Check rate limit before starting
	remaining, err := client.CheckRateLimit(ctx)
	if err != nil {
		log.Printf("Warning: failed to check rate limit: %v", err)
	} else if debug {
		log.Printf("GitHub API rate limit remaining: %d", remaining)
	}

	// Post comments concurrently with semaphore
	results := postCommentsConcurrently(ctx, client, comments, existingComments, debug)

	// Aggregate results
	output := &ActionOutput{
		Results: results,
	}

	for _, result := range results {
		switch result.Status {
		case "posted":
			output.Posted++
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
func postCommentsConcurrently(ctx context.Context, client Client, comments []*comment.GeneratedComment, existingComments []*ExistingComment, debug bool) []CommentResult {
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

			// Check for duplicates
			if isDuplicate(comm, existingComments) {
				if debug {
					log.Printf("[%d/%d] Skipping duplicate comment at line %d (%s)", idx+1, len(comments), comm.Line, comm.Side)
				}
				resultChan <- CommentResult{
					Status:      "skipped_duplicate",
					BodyPreview: comm.GetBodyPreview(),
				}
				return
			}

			// Post comment with retry logic
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

// isDuplicate checks if a comment already exists
func isDuplicate(newComment *comment.GeneratedComment, existingComments []*ExistingComment) bool {
	for _, existing := range existingComments {
		// Check if path and position match
		if existing.Path == newComment.Path && existing.Position == newComment.Position {
			// Check if body is similar (normalize whitespace)
			existingBody := normalizeWhitespace(existing.Body)
			newBody := normalizeWhitespace(newComment.Body)

			if existingBody == newBody {
				return true
			}
		}
	}
	return false
}

// normalizeWhitespace normalizes whitespace for comparison
func normalizeWhitespace(s string) string {
	// Replace multiple whitespace with single space
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}
