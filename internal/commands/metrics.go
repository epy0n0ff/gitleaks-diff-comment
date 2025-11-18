package commands

import (
	"encoding/json"
	"fmt"
	"time"
)

// MetricsEvent represents structured metrics data for observability
type MetricsEvent struct {
	// EventType is always "clear_command_executed"
	EventType string `json:"event_type"`

	// Timestamp is the event timestamp in ISO 8601 UTC format
	Timestamp string `json:"timestamp"`

	// PRNumber is the pull request number
	PRNumber int `json:"pr_number"`

	// RequestedBy is the GitHub username who executed the command
	RequestedBy string `json:"requested_by"`

	// CommentsCleared is the number of comments successfully deleted
	CommentsCleared int `json:"comments_cleared"`

	// ErrorCount is the number of errors encountered
	ErrorCount int `json:"error_count"`

	// DurationSeconds is the total operation time
	DurationSeconds float64 `json:"duration_seconds"`

	// RetryAttempts is the number of retries performed
	RetryAttempts int `json:"retry_attempts"`

	// Success indicates whether operation completed successfully
	Success bool `json:"success"`
}

// NewMetricsEvent creates a MetricsEvent from a ClearOperation
func NewMetricsEvent(op *ClearOperation) *MetricsEvent {
	return &MetricsEvent{
		EventType:       "clear_command_executed",
		Timestamp:       op.CompletedAt.UTC().Format(time.RFC3339),
		PRNumber:        op.PRNumber,
		RequestedBy:     op.RequestedBy,
		CommentsCleared: op.CommentsDeleted,
		ErrorCount:      op.CommentsFailed,
		DurationSeconds: op.Duration,
		RetryAttempts:   op.RetryCount,
		Success:         op.Status == "completed" && op.CommentsFailed == 0,
	}
}

// logMetrics outputs structured JSON metrics to stdout for external monitoring systems
// Format: ::notice::METRICS:{json}
func logMetrics(event *MetricsEvent) error {
	jsonBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	fmt.Printf("::notice::METRICS:%s\n", string(jsonBytes))
	return nil
}
