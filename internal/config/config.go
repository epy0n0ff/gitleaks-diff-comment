package config

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Config holds all configuration parsed from action inputs and environment
type Config struct {
	// GitHub API token for authentication
	GitHubToken string

	// Pull request number
	PRNumber int

	// Repository in format "owner/repo"
	Repository string

	// Commit SHA that triggered the action
	CommitSHA string

	// Base branch reference (e.g., "main")
	BaseRef string

	// Head branch reference (e.g., "feature/update-ignore")
	HeadRef string

	// Workspace directory (git repository root)
	Workspace string

	// Comment mode: "override" or "append"
	CommentMode string

	// Enable debug logging
	Debug bool

	// GitHub Enterprise Server hostname (empty = GitHub.com)
	GHHost string

	// Command is the command to execute (e.g., "clear" or empty for normal mode)
	Command string

	// CommentID is the comment ID that triggered the command
	CommentID int64

	// Requester is the GitHub username who requested the command
	Requester string
}

// ParseFromEnv parses configuration from environment variables
func ParseFromEnv() (*Config, error) {
	cfg := &Config{
		GitHubToken: os.Getenv("INPUT_GITHUB-TOKEN"),
		Repository:  os.Getenv("GITHUB_REPOSITORY"),
		CommitSHA:   getCommitSHA(),
		BaseRef:     os.Getenv("GITHUB_BASE_REF"),
		HeadRef:     os.Getenv("GITHUB_HEAD_REF"),
		Workspace:   os.Getenv("GITHUB_WORKSPACE"),
		CommentMode: os.Getenv("INPUT_COMMENT-MODE"),
		GHHost:      os.Getenv("INPUT_GH-HOST"),
	}

	// Default comment mode to "override" if not specified
	if cfg.CommentMode == "" {
		cfg.CommentMode = "override"
	}

	// Parse PR number
	prNumStr := os.Getenv("INPUT_PR-NUMBER")
	if prNumStr != "" {
		prNum, err := strconv.Atoi(prNumStr)
		if err != nil {
			return nil, fmt.Errorf("invalid PR number: %w", err)
		}
		cfg.PRNumber = prNum
	}

	// Parse debug flag
	debugStr := os.Getenv("INPUT_DEBUG")
	cfg.Debug = strings.ToLower(debugStr) == "true"

	// Parse command-related fields (optional, for command mode)
	cfg.Command = os.Getenv("INPUT_COMMAND")
	cfg.Requester = os.Getenv("INPUT_REQUESTER")

	// Parse comment ID (optional, for command mode)
	commentIDStr := os.Getenv("INPUT_COMMENT-ID")
	if commentIDStr != "" {
		commentID, err := strconv.ParseInt(commentIDStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid comment ID: %w", err)
		}
		cfg.CommentID = commentID
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.GitHubToken == "" {
		return errors.New("GitHub token is required (INPUT_GITHUB-TOKEN)\n" +
			"  → Action: Set 'github-token' input in your workflow file\n" +
			"  → Example: github-token: ${{ secrets.GITHUB_TOKEN }}\n" +
			"  → Required scopes: repo (read), pull_requests (write)")
	}
	if c.PRNumber <= 0 {
		return errors.New("PR number must be positive (INPUT_PR-NUMBER)\n" +
			"  → Action: Set 'pr-number' input in your workflow file\n" +
			"  → Example: pr-number: ${{ github.event.pull_request.number }}")
	}
	if c.Repository == "" {
		return errors.New("repository is required (GITHUB_REPOSITORY)\n" +
			"  → Action: This is automatically set by GitHub Actions\n" +
			"  → Ensure the action is running in a GitHub Actions workflow")
	}
	if !strings.Contains(c.Repository, "/") {
		return fmt.Errorf("repository must be in format owner/repo, got: %s\n"+
			"  → Action: Check GITHUB_REPOSITORY environment variable\n"+
			"  → Expected format: owner/repository-name", c.Repository)
	}
	if c.CommitSHA == "" {
		return errors.New("commit SHA is required (GITHUB_SHA)\n" +
			"  → Action: This is automatically set by GitHub Actions\n" +
			"  → Ensure the action is running in a GitHub Actions workflow")
	}
	if c.CommentMode != "override" && c.CommentMode != "append" {
		return fmt.Errorf("comment-mode must be 'override' or 'append', got: %s\n"+
			"  → Action: Set 'comment-mode' input to either 'override' or 'append'\n"+
			"  → Example: comment-mode: override", c.CommentMode)
	}

	// Validate GHHost format (GitHub Enterprise Server hostname)
	if c.GHHost != "" {
		// Reject protocol prefix (http:// or https://)
		if strings.Contains(c.GHHost, "://") {
			hostWithoutProtocol := strings.Split(c.GHHost, "://")[1]
			return fmt.Errorf("gh-host must not include protocol (http:// or https://)\n"+
				"  → Action: Remove protocol prefix from gh-host\n"+
				"  → Example: gh-host: %s", hostWithoutProtocol)
		}

		// Reject path separator (/)
		if strings.Contains(c.GHHost, "/") {
			hostWithoutPath := strings.Split(c.GHHost, "/")[0]
			return fmt.Errorf("gh-host must not include path\n"+
				"  → Action: Remove path from gh-host (e.g., remove /api/v3)\n"+
				"  → Example: gh-host: %s", hostWithoutPath)
		}

		// Validate port number if present
		if strings.Contains(c.GHHost, ":") {
			parts := strings.Split(c.GHHost, ":")
			if len(parts) != 2 {
				return errors.New("invalid gh-host format with port\n" +
					"  → Action: Use format hostname:port\n" +
					"  → Example: gh-host: github.company.com:8443")
			}
			port, err := strconv.Atoi(parts[1])
			if err != nil || port < 1 || port > 65535 {
				return fmt.Errorf("invalid port in gh-host: %s (must be 1-65535)\n"+
					"  → Action: Use valid port number\n"+
					"  → Example: gh-host: github.company.com:8443", parts[1])
			}
		}
	}

	return nil
}

// Owner returns the repository owner from Repository field
func (c *Config) Owner() string {
	parts := strings.Split(c.Repository, "/")
	if len(parts) != 2 {
		return ""
	}
	return parts[0]
}

// Repo returns the repository name from Repository field
func (c *Config) Repo() string {
	parts := strings.Split(c.Repository, "/")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

// IsCommandMode returns true if the action is running in command mode
func (c *Config) IsCommandMode() bool {
	return c.Command != ""
}

// getCommitSHA gets the commit SHA to use for PR comments
// Priority: INPUT_COMMIT-SHA > git rev-parse HEAD > GITHUB_SHA
func getCommitSHA() string {
	// First, check if user provided commit-sha input
	if commitSHA := os.Getenv("INPUT_COMMIT-SHA"); commitSHA != "" {
		return commitSHA
	}

	// Try to get actual HEAD commit from git
	// This is the most reliable method in PR context
	cmd := exec.Command("git", "rev-parse", "HEAD")
	if output, err := cmd.Output(); err == nil {
		headCommit := strings.TrimSpace(string(output))
		if headCommit != "" {
			return headCommit
		}
	}

	// Fallback to GITHUB_SHA (may not be PR HEAD in some contexts)
	return os.Getenv("GITHUB_SHA")
}
