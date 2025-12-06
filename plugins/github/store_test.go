// ABOUTME: Unit tests for GitHub plugin store layer
// ABOUTME: Tests user, token, and repository CRUD operations

package github

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	return db
}

func TestNewGitHubStore(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store, err := NewGitHubStore(db)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	if store == nil {
		t.Fatal("Store is nil")
	}

	// Verify tables were created
	tables := []string{
		"github_users",
		"github_tokens",
		"github_repositories",
		"github_branches",
		"github_commits",
		"github_issues",
		"github_pull_requests",
		"github_comments",
		"github_reviews",
		"github_review_comments",
		"github_webhooks",
		"github_webhook_deliveries",
	}

	for _, table := range tables {
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err != nil || count != 1 {
			t.Fatalf("Table %s was not created", table)
		}
	}
}

func TestGetOrCreateUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)

	// First call should create
	user, err := store.GetOrCreateUser("alice", "ghp_testtoken123")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	if user.Login != "alice" {
		t.Fatalf("Expected login 'alice', got '%s'", user.Login)
	}
	if user.ID == 0 {
		t.Fatal("User ID should be generated")
	}

	// Second call should retrieve
	user2, err := store.GetOrCreateUser("alice", "ghp_testtoken123")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	if user2.ID != user.ID {
		t.Fatal("Should retrieve same user")
	}
}

func TestValidateToken(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)

	// Create user with token
	user, _ := store.GetOrCreateUser("alice", "ghp_validtoken")

	// Valid token should return user
	validUser, err := store.ValidateToken("ghp_validtoken")
	if err != nil {
		t.Fatalf("Token validation failed: %v", err)
	}
	if validUser.ID != user.ID {
		t.Fatal("ValidateToken returned wrong user")
	}

	// Invalid token should fail
	_, err = store.ValidateToken("ghp_invalidtoken")
	if err != sql.ErrNoRows {
		t.Fatal("Expected sql.ErrNoRows for invalid token")
	}
}
