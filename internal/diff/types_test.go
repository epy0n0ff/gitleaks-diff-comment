package diff

import (
	"testing"
)

func TestParseGitleaksEntry_WithLineNumber(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantPattern   string
		wantLineNum   int
		wantIsPattern bool
		wantErr       bool
	}{
		{
			name:          "file with line number",
			input:         "config/secrets.yml:42",
			wantPattern:   "config/secrets.yml",
			wantLineNum:   42,
			wantIsPattern: false,
			wantErr:       false,
		},
		{
			name:          "wildcard pattern",
			input:         "*.env",
			wantPattern:   "*.env",
			wantLineNum:   0,
			wantIsPattern: true,
			wantErr:       false,
		},
		{
			name:          "directory wildcard",
			input:         "config/*.json",
			wantPattern:   "config/*.json",
			wantLineNum:   0,
			wantIsPattern: true,
			wantErr:       false,
		},
		{
			name:          "simple file path",
			input:         "database/credentials.json",
			wantPattern:   "database/credentials.json",
			wantLineNum:   0,
			wantIsPattern: false,
			wantErr:       false,
		},
		{
			name:          "empty line",
			input:         "",
			wantPattern:   "",
			wantLineNum:   0,
			wantIsPattern: false,
			wantErr:       true,
		},
		{
			name:          "comment line",
			input:         "# This is a comment",
			wantPattern:   "",
			wantLineNum:   0,
			wantIsPattern: false,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := ParseGitleaksEntry(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseGitleaksEntry() expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseGitleaksEntry() unexpected error: %v", err)
				return
			}

			if entry.FilePattern != tt.wantPattern {
				t.Errorf("FilePattern = %v, want %v", entry.FilePattern, tt.wantPattern)
			}

			if entry.LineNumber != tt.wantLineNum {
				t.Errorf("LineNumber = %v, want %v", entry.LineNumber, tt.wantLineNum)
			}

			if entry.IsPattern != tt.wantIsPattern {
				t.Errorf("IsPattern = %v, want %v", entry.IsPattern, tt.wantIsPattern)
			}
		})
	}
}

func TestGitleaksEntry_HasLineNumber(t *testing.T) {
	tests := []struct {
		name     string
		entry    GitleaksEntry
		expected bool
	}{
		{
			name:     "entry with line number",
			entry:    GitleaksEntry{LineNumber: 42},
			expected: true,
		},
		{
			name:     "entry without line number",
			entry:    GitleaksEntry{LineNumber: 0},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.entry.HasLineNumber()
			if result != tt.expected {
				t.Errorf("HasLineNumber() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGitleaksEntry_FileLink(t *testing.T) {
	tests := []struct {
		name      string
		entry     GitleaksEntry
		repo      string
		commitSHA string
		ghHost    string
		expected  string
	}{
		{
			name: "specific file - GitHub.com",
			entry: GitleaksEntry{
				FilePattern: "config/secrets.yml",
				IsPattern:   false,
			},
			repo:      "owner/repo",
			commitSHA: "abc123",
			ghHost:    "",
			expected:  "https://github.com/owner/repo/blob/abc123/config/secrets.yml",
		},
		{
			name: "wildcard pattern - GitHub.com",
			entry: GitleaksEntry{
				FilePattern: "config/*.env",
				IsPattern:   true,
			},
			repo:      "owner/repo",
			commitSHA: "abc123",
			ghHost:    "",
			expected:  "https://github.com/owner/repo/blob/abc123/config",
		},
		{
			name: "root wildcard pattern - GitHub.com",
			entry: GitleaksEntry{
				FilePattern: "*.env",
				IsPattern:   true,
			},
			repo:      "owner/repo",
			commitSHA: "abc123",
			ghHost:    "",
			expected:  "https://github.com/owner/repo/blob/abc123/",
		},
		{
			name: "specific file - Enterprise Server",
			entry: GitleaksEntry{
				FilePattern: "database/credentials.json",
				IsPattern:   false,
			},
			repo:      "owner/repo",
			commitSHA: "def456",
			ghHost:    "github.company.com",
			expected:  "https://github.company.com/owner/repo/blob/def456/database/credentials.json",
		},
		{
			name: "wildcard pattern - Enterprise Server with port",
			entry: GitleaksEntry{
				FilePattern: "secrets/*.key",
				IsPattern:   true,
			},
			repo:      "owner/repo",
			commitSHA: "def456",
			ghHost:    "github.company.com:8443",
			expected:  "https://github.company.com:8443/owner/repo/blob/def456/secrets",
		},
		{
			name: "file with line number - Enterprise Server",
			entry: GitleaksEntry{
				FilePattern: "config/app.yml",
				LineNumber:  123,
				IsPattern:   false,
			},
			repo:      "owner/repo",
			commitSHA: "def456",
			ghHost:    "github.internal",
			expected:  "https://github.internal/owner/repo/blob/def456/config/app.yml#L123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.entry.FileLink(tt.repo, tt.commitSHA, tt.ghHost)
			if result != tt.expected {
				t.Errorf("FileLink() = %v, want %v", result, tt.expected)
			}
		})
	}
}
