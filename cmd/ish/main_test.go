// ABOUTME: Tests for CLI commands and server wiring.
// ABOUTME: Verifies health check, path validation, and basic server functionality.

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"
)

func TestServer_Healthz(t *testing.T) {
	dbPath := "test_main.db"
	defer os.Remove(dbPath)

	srv, err := newServer(dbPath)
	if err != nil {
		t.Fatalf("newServer() error = %v", err)
	}

	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal() error = %v, response body: %s", err, rr.Body.String())
	}
	if resp["ok"] != true {
		t.Errorf("ok = %v, want true", resp["ok"])
	}
}

func TestValidateAndCleanDBPath_Valid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple relative path",
			input: "ish.db",
		},
		{
			name:  "path with directory",
			input: "./data/ish.db",
		},
		{
			name:  "path with multiple directories",
			input: "./path/to/data/ish.db",
		},
		{
			name:  "absolute path on Unix",
			input: "/tmp/ish.db",
		},
		{
			name:  "path with whitespace trimmed",
			input: "  ish.db  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := validateAndCleanDBPath(tt.input)
			if err != nil {
				t.Errorf("validateAndCleanDBPath(%q) error = %v, want nil", tt.input, err)
			}
			if result == "" {
				t.Errorf("validateAndCleanDBPath(%q) returned empty string", tt.input)
			}
		})
	}
}

func TestValidateAndCleanDBPath_Invalid(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		shouldContain string
	}{
		{
			name:          "empty string",
			input:         "",
			shouldContain: "cannot be empty",
		},
		{
			name:          "current directory dot",
			input:         ".",
			shouldContain: "cannot be empty, '.', or '/'",
		},
		{
			name:          "root directory",
			input:         "/",
			shouldContain: "cannot be empty, '.', or '/'",
		},
		{
			name:          "path traversal with dotdot",
			input:         "../../etc/passwd",
			shouldContain: "cannot contain '..'",
		},
		{
			name:          "dotdot in middle",
			input:         "./data/../../../etc/passwd",
			shouldContain: "cannot contain '..'",
		},
		{
			name:          "git directory blocked",
			input:         ".git/ish.db",
			shouldContain: ".git",
		},
		{
			name:          "svn directory blocked",
			input:         ".svn/ish.db",
			shouldContain: ".svn",
		},
		{
			name:          "node_modules directory blocked",
			input:         "node_modules/ish.db",
			shouldContain: "node_modules",
		},
		{
			name:          "credentials in path blocked",
			input:         "credentials/ish.db",
			shouldContain: "credentials",
		},
		{
			name:          "secret in path blocked",
			input:         "secret/ish.db",
			shouldContain: "secret",
		},
		{
			name:          ".env in path blocked",
			input:         ".env/ish.db",
			shouldContain: ".env",
		},
		{
			name:          "case insensitive bad pattern",
			input:         "CREDENTIALS/ish.db",
			shouldContain: "credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateAndCleanDBPath(tt.input)
			if err == nil {
				t.Errorf("validateAndCleanDBPath(%q) error = nil, want error", tt.input)
			}
			if err != nil && !errContains(err.Error(), tt.shouldContain) {
				t.Errorf("validateAndCleanDBPath(%q) error = %v, should contain %q", tt.input, err, tt.shouldContain)
			}
		})
	}
}

func TestValidateAndCleanDBPath_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	tests := []struct {
		name          string
		input         string
		shouldFail    bool
		shouldContain string
	}{
		{
			name:       "Windows absolute path",
			input:      "C:\\data\\ish.db",
			shouldFail: false,
		},
		{
			name:       "Windows absolute path with UNC",
			input:      "\\\\server\\share\\ish.db",
			shouldFail: false,
		},
		{
			name:          "bare drive letter rejected",
			input:         "C:",
			shouldFail:    true,
			shouldContain: "bare drive letter",
		},
		{
			name:          "bare D drive rejected",
			input:         "D:",
			shouldFail:    true,
			shouldContain: "bare drive letter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateAndCleanDBPath(tt.input)
			if tt.shouldFail && err == nil {
				t.Errorf("validateAndCleanDBPath(%q) error = nil, want error", tt.input)
			}
			if !tt.shouldFail && err != nil {
				t.Errorf("validateAndCleanDBPath(%q) error = %v, want nil", tt.input, err)
			}
			if err != nil && tt.shouldContain != "" && !errContains(err.Error(), tt.shouldContain) {
				t.Errorf("validateAndCleanDBPath(%q) error = %v, should contain %q", tt.input, err, tt.shouldContain)
			}
		})
	}
}

// Helper function to check if error message contains a substring
func errContains(errMsg, substr string) bool {
	return len(errMsg) > 0 && len(substr) > 0 && (errMsg == substr || len(errMsg) > len(substr) && (errMsg[:len(substr)] == substr || errMsg[len(errMsg)-len(substr):] == substr || contains(errMsg, substr)))
}

// Helper to check substring presence
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
