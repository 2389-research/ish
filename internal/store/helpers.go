// ABOUTME: SQL helper functions for query construction.
// ABOUTME: Utilities for escaping and handling SQL patterns safely.

package store

import "strings"

// escapeSQLLike escapes SQL LIKE pattern special characters.
// This function prevents SQL injection and unintended wildcards in LIKE queries
// by escaping the special characters %, _, and \ which have special meaning in LIKE patterns.
// The backslash must be escaped first to avoid double-escaping.
func escapeSQLLike(pattern string) string {
	// Escape backslash first to avoid double-escaping
	pattern = strings.ReplaceAll(pattern, "\\", "\\\\")
	// Escape percent wildcard (% matches any sequence in SQL LIKE)
	pattern = strings.ReplaceAll(pattern, "%", "\\%")
	// Escape underscore wildcard (_ matches any single character in SQL LIKE)
	pattern = strings.ReplaceAll(pattern, "_", "\\_")
	return pattern
}
