package config

import (
	"errors"
	"fmt"
	"os"
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

	// Enable debug logging
	Debug bool
}

// ParseFromEnv parses configuration from environment variables
func ParseFromEnv() (*Config, error) {
	cfg := &Config{
		GitHubToken: os.Getenv("INPUT_GITHUB-TOKEN"),
		Repository:  os.Getenv("GITHUB_REPOSITORY"),
		CommitSHA:   os.Getenv("GITHUB_SHA"),
		BaseRef:     os.Getenv("GITHUB_BASE_REF"),
		HeadRef:     os.Getenv("GITHUB_HEAD_REF"),
		Workspace:   os.Getenv("GITHUB_WORKSPACE"),
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

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func Validate(c *Config) error {
	if c.GitHubToken == "" {
		return errors.New("GitHub token is required (INPUT_GITHUB-TOKEN)\n" +
			"  → Action: Set 'github-token' input in your workflow file\n" +
			"  → Example: github-token: ${{ secrets.GITHUB_TOKEN }}")
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
