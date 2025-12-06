// ABOUTME: Unit tests for Twilio plugin store layer
// ABOUTME: Tests account, message, call, and webhook operations

package twilio

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

func TestNewTwilioStore(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store, err := NewTwilioStore(db)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	if store == nil {
		t.Fatal("Store is nil")
	}

	// Verify all tables were created
	tables := []string{
		"twilio_accounts",
		"twilio_phone_numbers",
		"twilio_messages",
		"twilio_calls",
		"twilio_webhook_configs",
		"twilio_webhook_queue",
	}
	for _, table := range tables {
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err != nil || count != 1 {
			t.Fatalf("table %s was not created", table)
		}
	}
}
