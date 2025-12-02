// ABOUTME: Tests for SQLite store initialization and schema migrations.
// ABOUTME: Verifies database setup and table creation.

package store

import (
	"os"
	"testing"
)

func TestNewStore_CreatesDatabase(t *testing.T) {
	dbPath := "test_ish.db"
	defer os.Remove(dbPath)

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer s.Close()

	// Verify tables exist
	tables := []string{"users", "gmail_labels", "gmail_threads", "gmail_messages", "calendars", "calendar_events", "people"}
	for _, table := range tables {
		var name string
		err := s.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Errorf("table %s not found: %v", table, err)
		}
	}
}

func TestStore_CreateAndGetUser(t *testing.T) {
	dbPath := "test_ish.db"
	defer os.Remove(dbPath)

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer s.Close()

	// Create user
	err = s.CreateUser("harper")
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	// Get user
	exists, err := s.UserExists("harper")
	if err != nil {
		t.Fatalf("UserExists() error = %v", err)
	}
	if !exists {
		t.Error("UserExists() = false, want true")
	}

	// Non-existent user
	exists, err = s.UserExists("nobody")
	if err != nil {
		t.Fatalf("UserExists() error = %v", err)
	}
	if exists {
		t.Error("UserExists() = true for non-existent user")
	}
}
