// ABOUTME: Test helper functions for OAuth plugin tests
// ABOUTME: Provides shared test database setup

package oauth

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// createTestStore creates a temporary test database with OAuthStore
func createTestStore(t *testing.T) (*OAuthStore, func()) {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	store, err := NewOAuthStore(db)
	if err != nil {
		t.Fatalf("Failed to create OAuthStore: %v", err)
	}

	return store, func() { db.Close() }
}
