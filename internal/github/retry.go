package github

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/go-github/v57/github"
)

// RetryWithBackoff executes an operation with exponential backoff retry logic
// Returns (retryAttempts, error) where retryAttempts is the number of retries performed
func RetryWithBackoff(operation func() error, maxRetries int) (int, error) {
	baseDelay := 2 * time.Second
	maxDelay := 32 * time.Second
	retryAttempts := 0

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := operation()
		if err == nil {
			return retryAttempts, nil
		}

		// Check if error is retryable (rate limit or temporary network error)
		if !isRateLimitError(err) {
			// Non-retryable error, fail immediately
			return retryAttempts, err
		}

		// Last attempt failed, return error
		if attempt == maxRetries-1 {
			return retryAttempts, fmt.Errorf("max retries (%d) exceeded: %w", maxRetries, err)
		}

		// Calculate delay with exponential backoff
		delay := baseDelay * (1 << uint(attempt))
		if delay > maxDelay {
			delay = maxDelay
		}

		// Add jitter: random 0-50% of delay
		jitter := time.Duration(rand.Int63n(int64(delay / 2)))
		totalDelay := delay + jitter

		retryAttempts++
		time.Sleep(totalDelay)
	}

	return retryAttempts, fmt.Errorf("unexpected: exhausted retries without error")
}

// isRateLimitError checks if an error is a GitHub API rate limit error
func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}

	// Check for RateLimitError type from go-github
	var rateLimitErr *github.RateLimitError
	if errors.As(err, &rateLimitErr) {
		return true
	}

	// Check for AbuseRateLimitError type from go-github
	var abuseRateLimitErr *github.AbuseRateLimitError
	if errors.As(err, &abuseRateLimitErr) {
		return true
	}

	return false
}
