package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/epy0n0ff/gitleaks-diff-comment/internal/comment"
	"github.com/epy0n0ff/gitleaks-diff-comment/internal/config"
	"github.com/epy0n0ff/gitleaks-diff-comment/internal/diff"
	"github.com/epy0n0ff/gitleaks-diff-comment/internal/github"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	// Validate we're running in GitHub Actions environment
	if os.Getenv("GITHUB_ACTIONS") != "true" {
		log.Println("Warning: Not running in GitHub Actions environment")
		log.Println("This action is designed to run as a GitHub Action")
	}

	// Parse configuration from environment
	cfg, err := config.ParseFromEnv()
	if err != nil {
		return fmt.Errorf("failed to parse configuration: %w", err)
	}

	if cfg.Debug {
		log.Println("Debug mode enabled")
		log.Printf("Configuration: PR=%d, Repo=%s, Commit=%s", cfg.PRNumber, cfg.Repository, cfg.CommitSHA)
	}

	// Change to workspace directory if specified
	if cfg.Workspace != "" {
		if err := os.Chdir(cfg.Workspace); err != nil {
			return fmt.Errorf("failed to change to workspace directory: %w", err)
		}
		if cfg.Debug {
			log.Printf("Changed to workspace: %s", cfg.Workspace)
		}
	}

	// Parse diff to find .gitleaksignore changes
	if cfg.Debug {
		log.Println("Parsing .gitleaksignore diff...")
	}

	changes, err := diff.ParseGitleaksDiff(cfg.BaseRef, cfg.HeadRef)
	if err != nil {
		return fmt.Errorf("failed to parse diff: %w", err)
	}

	if len(changes) == 0 {
		log.Println("No changes found in .gitleaksignore")
		outputResult(&github.ActionOutput{})
		return nil
	}

	if cfg.Debug {
		log.Printf("Found %d changes in .gitleaksignore", len(changes))
	}

	// Generate comments for each change
	var comments []*comment.GeneratedComment
	for _, change := range changes {
		comm, err := comment.NewGeneratedComment(&change, cfg.Repository, cfg.CommitSHA)
		if err != nil {
			log.Printf("Warning: failed to generate comment for change at position %d: %v", change.Position, err)
			continue
		}
		comments = append(comments, comm)
	}

	if len(comments) == 0 {
		log.Println("No valid comments generated")
		outputResult(&github.ActionOutput{})
		return nil
	}

	if cfg.Debug {
		log.Printf("Generated %d comments", len(comments))
	}

	// Create GitHub API client
	client, err := github.NewClient(cfg.GitHubToken, cfg.Owner(), cfg.Repo(), cfg.PRNumber)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Post comments
	ctx := context.Background()
	output, err := github.PostComments(ctx, client, comments, cfg.Debug)
	if err != nil {
		return fmt.Errorf("failed to post comments: %w", err)
	}

	// Output results
	outputResult(output)

	// Print summary
	log.Printf("✓ Posted: %d comments", output.Posted)
	log.Printf("⊘ Skipped: %d duplicates", output.SkippedDuplicates)
	if output.Errors > 0 {
		log.Printf("✗ Errors: %d", output.Errors)
	}

	// Exit with error if there were errors
	if output.Errors > 0 {
		return fmt.Errorf("completed with %d errors", output.Errors)
	}

	return nil
}

// outputResult outputs the action results in GitHub Actions format
func outputResult(output *github.ActionOutput) {
	// Output for GitHub Actions
	fmt.Printf("::set-output name=posted::%d\n", output.Posted)
	fmt.Printf("::set-output name=skipped_duplicates::%d\n", output.SkippedDuplicates)
	fmt.Printf("::set-output name=errors::%d\n", output.Errors)

	// Also output JSON for debugging
	jsonOutput, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Printf("Warning: failed to marshal output as JSON: %v", err)
		return
	}
	fmt.Printf("\nResults:\n%s\n", string(jsonOutput))
}
