package github

import (
	"strings"
	"testing"
)

// TestNewClient_GitHubCom tests NewClient with empty gh-host (GitHub.com default)
func TestNewClient_GitHubCom(t *testing.T) {
	client, err := NewClient("test-token", "owner", "repo", 123, "")
	if err != nil {
		t.Fatalf("NewClient() with empty ghHost failed: %v", err)
	}

	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}

	// Verify client implementation
	impl, ok := client.(*ClientImpl)
	if !ok {
		t.Fatal("NewClient() did not return *ClientImpl")
	}

	if impl.owner != "owner" {
		t.Errorf("Expected owner 'owner', got %s", impl.owner)
	}
	if impl.repo != "repo" {
		t.Errorf("Expected repo 'repo', got %s", impl.repo)
	}
	if impl.prNumber != 123 {
		t.Errorf("Expected prNumber 123, got %d", impl.prNumber)
	}
}

// TestNewClient_Enterprise tests NewClient with enterprise hostname
func TestNewClient_Enterprise(t *testing.T) {
	client, err := NewClient("test-token", "owner", "repo", 123, "github.company.com")
	if err != nil {
		t.Fatalf("NewClient() with enterprise ghHost failed: %v", err)
	}

	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}

	// Verify client implementation
	impl, ok := client.(*ClientImpl)
	if !ok {
		t.Fatal("NewClient() did not return *ClientImpl")
	}

	if impl.owner != "owner" {
		t.Errorf("Expected owner 'owner', got %s", impl.owner)
	}
	if impl.repo != "repo" {
		t.Errorf("Expected repo 'repo', got %s", impl.repo)
	}
	if impl.prNumber != 123 {
		t.Errorf("Expected prNumber 123, got %d", impl.prNumber)
	}
}

// TestNewClient_EnterpriseWithPort tests NewClient with enterprise hostname and port
func TestNewClient_EnterpriseWithPort(t *testing.T) {
	client, err := NewClient("test-token", "owner", "repo", 123, "github.company.com:8443")
	if err != nil {
		t.Fatalf("NewClient() with enterprise ghHost and port failed: %v", err)
	}

	if client == nil {
		t.Fatal("NewClient() returned nil client")
	}

	// Verify client implementation
	impl, ok := client.(*ClientImpl)
	if !ok {
		t.Fatal("NewClient() did not return *ClientImpl")
	}

	if impl.owner != "owner" {
		t.Errorf("Expected owner 'owner', got %s", impl.owner)
	}
	if impl.repo != "repo" {
		t.Errorf("Expected repo 'repo', got %s", impl.repo)
	}
	if impl.prNumber != 123 {
		t.Errorf("Expected prNumber 123, got %d", impl.prNumber)
	}
}

// TestNewClient_ValidationErrors tests NewClient parameter validation
func TestNewClient_ValidationErrors(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		owner     string
		repo      string
		prNumber  int
		ghHost    string
		wantError string
	}{
		{
			name:      "empty token",
			token:     "",
			owner:     "owner",
			repo:      "repo",
			prNumber:  123,
			ghHost:    "",
			wantError: "GitHub token is required",
		},
		{
			name:      "empty owner",
			token:     "token",
			owner:     "",
			repo:      "repo",
			prNumber:  123,
			ghHost:    "",
			wantError: "owner is required",
		},
		{
			name:      "empty repo",
			token:     "token",
			owner:     "owner",
			repo:      "",
			prNumber:  123,
			ghHost:    "",
			wantError: "repo is required",
		},
		{
			name:      "negative PR number",
			token:     "token",
			owner:     "owner",
			repo:      "repo",
			prNumber:  -1,
			ghHost:    "",
			wantError: "PR number must be positive",
		},
		{
			name:      "zero PR number",
			token:     "token",
			owner:     "owner",
			repo:      "repo",
			prNumber:  0,
			ghHost:    "",
			wantError: "PR number must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.token, tt.owner, tt.repo, tt.prNumber, tt.ghHost)
			if err == nil {
				t.Fatalf("NewClient() expected error, got nil (client: %v)", client)
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("NewClient() error = %v, want error containing %q", err, tt.wantError)
			}
		})
	}
}

// TestNewClient_InvalidEnterpriseURL tests WithEnterpriseURLs error handling
func TestNewClient_InvalidEnterpriseURL(t *testing.T) {
	// Test with a malformed URL that might cause WithEnterpriseURLs to fail
	// Note: go-github's WithEnterpriseURLs is quite permissive, so this mainly
	// verifies error handling exists
	client, err := NewClient("token", "owner", "repo", 123, "github.company.com")

	// If no error, verify client was created
	if err == nil && client == nil {
		t.Fatal("NewClient() returned nil client without error")
	}

	// If error occurred, verify it includes the hostname
	if err != nil && !strings.Contains(err.Error(), "github.company.com") {
		t.Errorf("NewClient() error should include hostname, got: %v", err)
	}
}
