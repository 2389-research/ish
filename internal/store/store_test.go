// ABOUTME: Tests for core SQLite store initialization and schema migrations.
// ABOUTME: Verifies database setup and request_logs table creation.

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

	// Verify request_logs table exists
	var name string
	err = s.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='request_logs'").Scan(&name)
	if err != nil {
		t.Errorf("request_logs table not found: %v", err)
	}
}

func TestStore_GetDB(t *testing.T) {
	dbPath := "test_ish.db"
	defer os.Remove(dbPath)

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer s.Close()

	// Verify GetDB returns a valid database connection
	db := s.GetDB()
	if db == nil {
		t.Error("GetDB() returned nil")
	}

	// Verify we can query the database
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM request_logs").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 request logs, got %d", count)
	}
}
