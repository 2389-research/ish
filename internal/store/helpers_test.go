// ABOUTME: Tests for SQL LIKE escaping helper function.
// ABOUTME: Tests SQL special character escaping with edge cases.

package store

import (
	"testing"
)

func TestEscapeSQLLike(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no special characters",
			input:    "simple_path",
			expected: "simple\\_path",
		},
		{
			name:     "single percent wildcard",
			input:    "path%with%percent",
			expected: "path\\%with\\%percent",
		},
		{
			name:     "single underscore wildcard",
			input:    "path_with_underscore",
			expected: "path\\_with\\_underscore",
		},
		{
			name:     "backslash escape character",
			input:    "path\\with\\backslash",
			expected: "path\\\\with\\\\backslash",
		},
		{
			name:     "mixed special characters",
			input:    "path%_with\\all_special%",
			expected: "path\\%\\_with\\\\all\\_special\\%",
		},
		{
			name:     "multiple consecutive backslashes",
			input:    "path\\\\double\\\\backslash",
			expected: "path\\\\\\\\double\\\\\\\\backslash",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only special characters",
			input:    "%_%\\",
			expected: "\\%\\_\\%\\\\",
		},
		{
			name:     "real SQL injection attempt",
			input:    "'; DROP TABLE request_logs; --",
			expected: "'; DROP TABLE request\\_logs; --",
		},
		{
			name:     "LIKE wildcard bypass attempt",
			input:    "prefix%",
			expected: "prefix\\%",
		},
		{
			name:     "underscore single char match attempt",
			input:    "path_",
			expected: "path\\_",
		},
		{
			name:     "backslash followed by percent",
			input:    "path\\%",
			expected: "path\\\\\\%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeSQLLike(tt.input)
			if result != tt.expected {
				t.Errorf("escapeSQLLike(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEscapeSQLLikeOrder(t *testing.T) {
	// Test that backslash is escaped first to avoid double-escaping
	input := "test\\%"
	result := escapeSQLLike(input)
	// Should be: test\\ (backslash escaped to \\) + \% (percent escaped to \%)
	expected := "test\\\\\\%"
	if result != expected {
		t.Errorf("escapeSQLLike(%q) = %q, want %q (backslash should be escaped first)", input, result, expected)
	}
}
