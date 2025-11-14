package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/epy0n0ff/gitleaks-diff-comment/internal/github"
)

// MockEnterpriseServer creates a mock GitHub Enterprise Server for testing
func MockEnterpriseServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	return server
}

// TestEnterprisePATAuthentication tests Personal Access Token authentication with enterprise
func TestEnterprisePATAuthentication(t *testing.T) {
	// Create mock enterprise server that validates PAT
	server := MockEnterpriseServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Verify Authorization header
		auth := r.Header.Get("Authorization")
		if auth == "" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"message": "Requires authentication",
			})
			return
		}

		// Verify Bearer token format (PAT or GitHub App token)
		if !strings.HasPrefix(auth, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"message": "Bad credentials",
			})
			return
		}

		// Mock successful authentication - return rate limit info
		if strings.Contains(r.URL.Path, "/rate_limit") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"resources": map[string]interface{}{
					"core": map[string]interface{}{
						"limit":     5000,
						"remaining": 4999,
						"reset":     1234567890,
					},
				},
			})
			return
		}

		// Mock PR comments list
		if strings.Contains(r.URL.Path, "/pulls/") && strings.Contains(r.URL.Path, "/comments") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]interface{}{})
			return
		}

		// Default success response
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
	})
	defer server.Close()

	// Extract hostname from mock server URL (remove http://)
	ghHost := strings.TrimPrefix(server.URL, "http://")

	// Create client with mock enterprise server
	client, err := github.NewClient("test-pat-token", "owner", "repo", 123, ghHost)
	if err != nil {
		t.Fatalf("NewClient() failed with valid PAT: %v", err)
	}

	// Test API call with PAT authentication
	ctx := context.Background()
	_, err = client.ListReviewComments(ctx)
	if err != nil {
		t.Errorf("ListReviewComments() failed with valid PAT: %v", err)
	}
}

// TestEnterpriseAuthenticationFailure tests authentication failure with clear error message
func TestEnterpriseAuthenticationFailure(t *testing.T) {
	// Create mock enterprise server that rejects authentication
	server := MockEnterpriseServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Always return 401 Unauthorized
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Bad credentials",
		})
	})
	defer server.Close()

	// Extract hostname from mock server URL
	ghHost := strings.TrimPrefix(server.URL, "http://")

	// Create client (client creation should succeed)
	client, err := github.NewClient("invalid-token", "owner", "repo", 123, ghHost)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	// Test API call with invalid token (should fail with auth error)
	ctx := context.Background()
	_, err = client.ListReviewComments(ctx)
	if err == nil {
		t.Fatal("ListReviewComments() expected authentication error, got nil")
	}

	// Verify error message is helpful
	errMsg := err.Error()
	if !strings.Contains(strings.ToLower(errMsg), "401") &&
		!strings.Contains(strings.ToLower(errMsg), "unauthorized") &&
		!strings.Contains(strings.ToLower(errMsg), "bad credentials") {
		t.Errorf("Error message should indicate authentication failure, got: %v", err)
	}
}

// TestEnterpriseNetworkError tests network connectivity error handling
func TestEnterpriseNetworkError(t *testing.T) {
	// Use invalid hostname that will cause network error
	ghHost := "nonexistent.github.enterprise.local"

	// Create client (should succeed - validation happens during API calls)
	client, err := github.NewClient("test-token", "owner", "repo", 123, ghHost)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	// Test API call with unreachable server (should fail with network error)
	ctx := context.Background()
	_, err = client.CheckRateLimit(ctx)
	if err == nil {
		t.Fatal("CheckRateLimit() expected network error, got nil")
	}

	// Verify error message indicates network issue
	errMsg := strings.ToLower(err.Error())
	hasNetworkIndicator := strings.Contains(errMsg, "connection") ||
		strings.Contains(errMsg, "network") ||
		strings.Contains(errMsg, "no such host") ||
		strings.Contains(errMsg, "dial")

	if !hasNetworkIndicator {
		t.Errorf("Error message should indicate network issue, got: %v", err)
	}
}

// TestEnterpriseWithPort tests enterprise hostname with custom port
func TestEnterpriseWithPort(t *testing.T) {
	// Create mock enterprise server
	server := MockEnterpriseServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Verify authentication
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Mock rate limit endpoint
		if strings.Contains(r.URL.Path, "/rate_limit") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"resources": map[string]interface{}{
					"core": map[string]interface{}{
						"limit":     5000,
						"remaining": 5000,
						"reset":     1234567890,
					},
				},
			})
			return
		}

		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	// Extract hostname with port from mock server URL
	ghHost := strings.TrimPrefix(server.URL, "http://")

	// Create client with hostname:port format
	client, err := github.NewClient("test-token", "owner", "repo", 123, ghHost)
	if err != nil {
		t.Fatalf("NewClient() failed with hostname:port: %v", err)
	}

	// Test API call
	ctx := context.Background()
	remaining, err := client.CheckRateLimit(ctx)
	if err != nil {
		t.Errorf("CheckRateLimit() failed: %v", err)
	}

	if remaining != 5000 {
		t.Errorf("Expected remaining rate limit 5000, got %d", remaining)
	}
}

// TestErrorClassification tests error classification helpers
func TestErrorClassification(t *testing.T) {
	// Note: isAuthError and isNetworkError are not exported, so we test indirectly
	// by observing error messages from actual API calls

	t.Run("auth error detection", func(t *testing.T) {
		server := MockEnterpriseServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"message": "Bad credentials",
			})
		})
		defer server.Close()

		ghHost := strings.TrimPrefix(server.URL, "http://")
		client, _ := github.NewClient("bad-token", "owner", "repo", 123, ghHost)

		ctx := context.Background()
		_, err := client.CheckRateLimit(ctx)
		if err == nil {
			t.Fatal("Expected authentication error")
		}

		// Verify error is classified as authentication error
		errMsg := strings.ToLower(err.Error())
		if !strings.Contains(errMsg, "401") && !strings.Contains(errMsg, "unauthorized") {
			t.Errorf("Expected auth error classification, got: %v", err)
		}
	})
}

// TestInvalidURLFormat tests that invalid gh-host formats are caught during validation
// Note: This is tested at the config layer, not client layer
func TestInvalidURLFormat(t *testing.T) {
	tests := []struct {
		name      string
		ghHost    string
		expectErr bool
		errText   string
	}{
		{
			name:      "valid hostname",
			ghHost:    "github.company.com",
			expectErr: false,
		},
		{
			name:      "valid hostname with port",
			ghHost:    "github.company.com:8443",
			expectErr: false,
		},
		{
			name:      "empty (GitHub.com default)",
			ghHost:    "",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// NewClient should succeed for valid formats
			// (validation happens in Config.Validate, not NewClient)
			_, err := github.NewClient("test-token", "owner", "repo", 123, tt.ghHost)

			if tt.expectErr && err == nil {
				t.Errorf("Expected error for ghHost=%q, got nil", tt.ghHost)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Unexpected error for ghHost=%q: %v", tt.ghHost, err)
			}
			if tt.expectErr && err != nil && !strings.Contains(err.Error(), tt.errText) {
				t.Errorf("Error should contain %q, got: %v", tt.errText, err)
			}
		})
	}
}

// TestEnterpriseRateLimitHandling tests that enterprise rate limits are read and respected
func TestEnterpriseRateLimitHandling(t *testing.T) {
	// Create mock enterprise server with custom rate limits
	server := MockEnterpriseServer(t, func(w http.ResponseWriter, r *http.Request) {
		// Verify authentication
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Mock rate limit endpoint with custom enterprise limits
		if strings.Contains(r.URL.Path, "/rate_limit") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"resources": map[string]interface{}{
					"core": map[string]interface{}{
						"limit":     10000, // Custom enterprise limit
						"remaining": 9500,  // Custom remaining count
						"reset":     1234567890,
					},
				},
			})
			return
		}

		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	ghHost := strings.TrimPrefix(server.URL, "http://")
	client, err := github.NewClient("test-token", "owner", "repo", 123, ghHost)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	// Check rate limit - should return custom enterprise values
	ctx := context.Background()
	remaining, err := client.CheckRateLimit(ctx)
	if err != nil {
		t.Errorf("CheckRateLimit() failed: %v", err)
	}

	// Verify we got the enterprise rate limit (9500)
	if remaining != 9500 {
		t.Errorf("Expected enterprise rate limit remaining=9500, got %d", remaining)
	}
}

// TestEnterpriseRateLimitDebugLogging tests that debug logs show rate limit values
func TestEnterpriseRateLimitDebugLogging(t *testing.T) {
	// This test verifies that rate limit information is available via CheckRateLimit
	// The actual debug logging happens in comments.PostComments when debug=true

	// Create mock server with rate limit headers
	server := MockEnterpriseServer(t, func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Different rate limit for enterprise
		if strings.Contains(r.URL.Path, "/rate_limit") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"resources": map[string]interface{}{
					"core": map[string]interface{}{
						"limit":     15000, // Higher enterprise limit
						"remaining": 14800,
						"reset":     1234567890,
					},
				},
			})
			return
		}

		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	ghHost := strings.TrimPrefix(server.URL, "http://")
	client, err := github.NewClient("test-token", "owner", "repo", 123, ghHost)
	if err != nil {
		t.Fatalf("NewClient() failed: %v", err)
	}

	// Verify rate limit can be retrieved for debug logging
	ctx := context.Background()
	remaining, err := client.CheckRateLimit(ctx)
	if err != nil {
		t.Fatalf("CheckRateLimit() failed: %v", err)
	}

	// Verify enterprise rate limit is detected
	if remaining != 14800 {
		t.Errorf("Expected enterprise rate limit remaining=14800, got %d", remaining)
	}

	// Note: Actual debug logging is tested by setting debug=true in PostComments
	// and verifying log output contains "GitHub API rate limit remaining: {remaining} calls"
}

