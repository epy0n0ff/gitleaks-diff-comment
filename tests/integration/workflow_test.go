package integration

import (
	"os"
	"testing"

	"github.com/epy0n0ff/gitleaks-diff-comment/internal/config"
)

// TestGitHubActionsEnvironment validates that required environment variables
// are correctly mapped for GitHub Actions
func TestGitHubActionsEnvironment(t *testing.T) {
	// Set up test environment variables as they would appear in GitHub Actions
	testEnvVars := map[string]string{
		"INPUT_GITHUB-TOKEN": "test-token-123",
		"INPUT_PR-NUMBER":    "42",
		"GITHUB_REPOSITORY":  "owner/test-repo",
		"GITHUB_SHA":         "abc123def456",
		"GITHUB_BASE_REF":    "main",
		"GITHUB_HEAD_REF":    "feature/test",
		"GITHUB_WORKSPACE":   "/github/workspace",
		"INPUT_DEBUG":        "false",
	}

	// Backup original environment
	originalEnv := make(map[string]string)
	for key := range testEnvVars {
		originalEnv[key] = os.Getenv(key)
	}

	// Set test environment variables
	for key, value := range testEnvVars {
		os.Setenv(key, value)
	}

	// Restore original environment after test
	defer func() {
		for key, originalValue := range originalEnv {
			if originalValue == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, originalValue)
			}
		}
	}()

	// Parse configuration
	cfg, err := config.ParseFromEnv()
	if err != nil {
		t.Fatalf("ParseFromEnv() unexpected error: %v", err)
	}

	// Validate parsed values
	if cfg.GitHubToken != "test-token-123" {
		t.Errorf("GitHubToken = %v, want %v", cfg.GitHubToken, "test-token-123")
	}

	if cfg.PRNumber != 42 {
		t.Errorf("PRNumber = %v, want %v", cfg.PRNumber, 42)
	}

	if cfg.Repository != "owner/test-repo" {
		t.Errorf("Repository = %v, want %v", cfg.Repository, "owner/test-repo")
	}

	if cfg.Owner() != "owner" {
		t.Errorf("Owner() = %v, want %v", cfg.Owner(), "owner")
	}

	if cfg.Repo() != "test-repo" {
		t.Errorf("Repo() = %v, want %v", cfg.Repo(), "test-repo")
	}

	if cfg.CommitSHA != "abc123def456" {
		t.Errorf("CommitSHA = %v, want %v", cfg.CommitSHA, "abc123def456")
	}

	if cfg.BaseRef != "main" {
		t.Errorf("BaseRef = %v, want %v", cfg.BaseRef, "main")
	}

	if cfg.HeadRef != "feature/test" {
		t.Errorf("HeadRef = %v, want %v", cfg.HeadRef, "feature/test")
	}

	if cfg.Debug != false {
		t.Errorf("Debug = %v, want %v", cfg.Debug, false)
	}
}

// TestRequiredEnvironmentVariables validates that missing required variables cause errors
func TestRequiredEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name       string
		envVars    map[string]string
		shouldFail bool
		errorMsg   string
	}{
		{
			name: "missing github token",
			envVars: map[string]string{
				"INPUT_PR-NUMBER":   "42",
				"GITHUB_REPOSITORY": "owner/repo",
				"GITHUB_SHA":        "abc123",
			},
			shouldFail: true,
			errorMsg:   "GitHub token",
		},
		{
			name: "missing pr number",
			envVars: map[string]string{
				"INPUT_GITHUB-TOKEN": "token",
				"GITHUB_REPOSITORY":  "owner/repo",
				"GITHUB_SHA":         "abc123",
			},
			shouldFail: true,
			errorMsg:   "PR number",
		},
		{
			name: "invalid pr number",
			envVars: map[string]string{
				"INPUT_GITHUB-TOKEN": "token",
				"INPUT_PR-NUMBER":    "invalid",
				"GITHUB_REPOSITORY":  "owner/repo",
				"GITHUB_SHA":         "abc123",
			},
			shouldFail: true,
			errorMsg:   "invalid PR number",
		},
		{
			name: "missing repository",
			envVars: map[string]string{
				"INPUT_GITHUB-TOKEN": "token",
				"INPUT_PR-NUMBER":    "42",
				"GITHUB_SHA":         "abc123",
			},
			shouldFail: true,
			errorMsg:   "repository",
		},
		{
			name: "invalid repository format",
			envVars: map[string]string{
				"INPUT_GITHUB-TOKEN": "token",
				"INPUT_PR-NUMBER":    "42",
				"GITHUB_REPOSITORY":  "invalid-format",
				"GITHUB_SHA":         "abc123",
			},
			shouldFail: true,
			errorMsg:   "owner/repo",
		},
		{
			name: "missing commit sha",
			envVars: map[string]string{
				"INPUT_GITHUB-TOKEN": "token",
				"INPUT_PR-NUMBER":    "42",
				"GITHUB_REPOSITORY":  "owner/repo",
			},
			shouldFail: true,
			errorMsg:   "commit SHA",
		},
		{
			name: "all required variables present",
			envVars: map[string]string{
				"INPUT_GITHUB-TOKEN": "token",
				"INPUT_PR-NUMBER":    "42",
				"GITHUB_REPOSITORY":  "owner/repo",
				"GITHUB_SHA":         "abc123",
			},
			shouldFail: false,
			errorMsg:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all environment variables
			os.Clearenv()

			// Set test environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Try to parse configuration
			_, err := config.ParseFromEnv()

			if tt.shouldFail {
				if err == nil {
					t.Errorf("Expected error containing %q, but got nil", tt.errorMsg)
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

// TestDebugFlagParsing validates debug flag parsing from environment
func TestDebugFlagParsing(t *testing.T) {
	tests := []struct {
		name        string
		debugValue  string
		expectedVal bool
	}{
		{"debug true", "true", true},
		{"debug false", "false", false},
		{"debug empty", "", false},
		{"debug TRUE", "TRUE", true},
		{"debug False", "False", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set required variables
			os.Setenv("INPUT_GITHUB-TOKEN", "token")
			os.Setenv("INPUT_PR-NUMBER", "42")
			os.Setenv("GITHUB_REPOSITORY", "owner/repo")
			os.Setenv("GITHUB_SHA", "abc123")
			os.Setenv("INPUT_DEBUG", tt.debugValue)

			cfg, err := config.ParseFromEnv()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if cfg.Debug != tt.expectedVal {
				t.Errorf("Debug = %v, want %v", cfg.Debug, tt.expectedVal)
			}
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
