package config

import (
	"strings"
	"testing"
)

// TestValidate_ValidGHHost tests Config.Validate() with valid gh-host values
func TestValidate_ValidGHHost(t *testing.T) {
	tests := []struct {
		name   string
		ghHost string
	}{
		{
			name:   "empty gh-host (GitHub.com)",
			ghHost: "",
		},
		{
			name:   "simple hostname",
			ghHost: "github.company.com",
		},
		{
			name:   "hostname with subdomain",
			ghHost: "github.enterprise.internal",
		},
		{
			name:   "hostname with port",
			ghHost: "github.company.com:8443",
		},
		{
			name:   "IP address",
			ghHost: "10.0.1.50",
		},
		{
			name:   "IP address with port",
			ghHost: "10.0.1.50:8443",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				GitHubToken: "test-token",
				PRNumber:    123,
				Repository:  "owner/repo",
				CommitSHA:   "abc123",
				CommentMode: "override",
				GHHost:      tt.ghHost,
			}

			err := cfg.Validate()
			if err != nil {
				t.Errorf("Validate() with ghHost=%q failed: %v", tt.ghHost, err)
			}
		})
	}
}

// TestValidate_GHHostWithProtocol tests Config.Validate() rejecting gh-host with protocol
func TestValidate_GHHostWithProtocol(t *testing.T) {
	tests := []struct {
		name      string
		ghHost    string
		wantError string
	}{
		{
			name:      "https protocol",
			ghHost:    "https://github.company.com",
			wantError: "gh-host must not include protocol",
		},
		{
			name:      "http protocol",
			ghHost:    "http://github.company.com",
			wantError: "gh-host must not include protocol",
		},
		{
			name:      "https with port",
			ghHost:    "https://github.company.com:8443",
			wantError: "gh-host must not include protocol",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				GitHubToken: "test-token",
				PRNumber:    123,
				Repository:  "owner/repo",
				CommitSHA:   "abc123",
				CommentMode: "override",
				GHHost:      tt.ghHost,
			}

			err := cfg.Validate()
			if err == nil {
				t.Fatalf("Validate() with ghHost=%q expected error, got nil", tt.ghHost)
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("Validate() error = %v, want error containing %q", err, tt.wantError)
			}

			// Verify error message includes corrected hostname
			expectedHost := strings.Split(tt.ghHost, "://")[1]
			if !strings.Contains(err.Error(), expectedHost) {
				t.Errorf("Validate() error should suggest %q, got: %v", expectedHost, err)
			}
		})
	}
}

// TestValidate_GHHostWithPath tests Config.Validate() rejecting gh-host with path
func TestValidate_GHHostWithPath(t *testing.T) {
	tests := []struct {
		name      string
		ghHost    string
		wantError string
	}{
		{
			name:      "path /api/v3",
			ghHost:    "github.company.com/api/v3",
			wantError: "gh-host must not include path",
		},
		{
			name:      "path /api",
			ghHost:    "github.company.com/api",
			wantError: "gh-host must not include path",
		},
		{
			name:      "trailing slash",
			ghHost:    "github.company.com/",
			wantError: "gh-host must not include path",
		},
		{
			name:      "path with port",
			ghHost:    "github.company.com:8443/api/v3",
			wantError: "gh-host must not include path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				GitHubToken: "test-token",
				PRNumber:    123,
				Repository:  "owner/repo",
				CommitSHA:   "abc123",
				CommentMode: "override",
				GHHost:      tt.ghHost,
			}

			err := cfg.Validate()
			if err == nil {
				t.Fatalf("Validate() with ghHost=%q expected error, got nil", tt.ghHost)
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("Validate() error = %v, want error containing %q", err, tt.wantError)
			}

			// Verify error message includes corrected hostname
			expectedHost := strings.Split(tt.ghHost, "/")[0]
			if !strings.Contains(err.Error(), expectedHost) {
				t.Errorf("Validate() error should suggest %q, got: %v", expectedHost, err)
			}
		})
	}
}

// TestValidate_GHHostPortValidation tests Config.Validate() with port number validation
func TestValidate_GHHostPortValidation(t *testing.T) {
	tests := []struct {
		name      string
		ghHost    string
		wantError bool
		errorText string
	}{
		{
			name:      "valid port 8443",
			ghHost:    "github.company.com:8443",
			wantError: false,
		},
		{
			name:      "valid port 443",
			ghHost:    "github.company.com:443",
			wantError: false,
		},
		{
			name:      "valid port 80",
			ghHost:    "github.company.com:80",
			wantError: false,
		},
		{
			name:      "valid port 1",
			ghHost:    "github.company.com:1",
			wantError: false,
		},
		{
			name:      "valid port 65535",
			ghHost:    "github.company.com:65535",
			wantError: false,
		},
		{
			name:      "invalid port 0",
			ghHost:    "github.company.com:0",
			wantError: true,
			errorText: "invalid port in gh-host",
		},
		{
			name:      "invalid port 65536",
			ghHost:    "github.company.com:65536",
			wantError: true,
			errorText: "invalid port in gh-host",
		},
		{
			name:      "invalid port 99999",
			ghHost:    "github.company.com:99999",
			wantError: true,
			errorText: "invalid port in gh-host",
		},
		{
			name:      "invalid port negative",
			ghHost:    "github.company.com:-1",
			wantError: true,
			errorText: "invalid port in gh-host",
		},
		{
			name:      "invalid port non-numeric",
			ghHost:    "github.company.com:abc",
			wantError: true,
			errorText: "invalid port in gh-host",
		},
		{
			name:      "multiple colons",
			ghHost:    "github.company.com:8443:extra",
			wantError: true,
			errorText: "invalid gh-host format with port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				GitHubToken: "test-token",
				PRNumber:    123,
				Repository:  "owner/repo",
				CommitSHA:   "abc123",
				CommentMode: "override",
				GHHost:      tt.ghHost,
			}

			err := cfg.Validate()
			if tt.wantError {
				if err == nil {
					t.Fatalf("Validate() with ghHost=%q expected error, got nil", tt.ghHost)
				}
				if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errorText)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() with ghHost=%q failed: %v", tt.ghHost, err)
				}
			}
		})
	}
}

// TestValidate_RequiredFields tests Config.Validate() with required fields
func TestValidate_RequiredFields(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError string
	}{
		{
			name: "missing token",
			config: &Config{
				GitHubToken: "",
				PRNumber:    123,
				Repository:  "owner/repo",
				CommitSHA:   "abc123",
				CommentMode: "override",
			},
			wantError: "GitHub token is required",
		},
		{
			name: "missing PR number",
			config: &Config{
				GitHubToken: "token",
				PRNumber:    0,
				Repository:  "owner/repo",
				CommitSHA:   "abc123",
				CommentMode: "override",
			},
			wantError: "PR number must be positive",
		},
		{
			name: "missing repository",
			config: &Config{
				GitHubToken: "token",
				PRNumber:    123,
				Repository:  "",
				CommitSHA:   "abc123",
				CommentMode: "override",
			},
			wantError: "repository is required",
		},
		{
			name: "invalid repository format",
			config: &Config{
				GitHubToken: "token",
				PRNumber:    123,
				Repository:  "invalid-format",
				CommitSHA:   "abc123",
				CommentMode: "override",
			},
			wantError: "repository must be in format owner/repo",
		},
		{
			name: "missing commit SHA",
			config: &Config{
				GitHubToken: "token",
				PRNumber:    123,
				Repository:  "owner/repo",
				CommitSHA:   "",
				CommentMode: "override",
			},
			wantError: "commit SHA is required",
		},
		{
			name: "invalid comment mode",
			config: &Config{
				GitHubToken: "token",
				PRNumber:    123,
				Repository:  "owner/repo",
				CommitSHA:   "abc123",
				CommentMode: "invalid",
			},
			wantError: "comment-mode must be 'override' or 'append'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if err == nil {
				t.Fatalf("Validate() expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("Validate() error = %v, want error containing %q", err, tt.wantError)
			}
		})
	}
}
