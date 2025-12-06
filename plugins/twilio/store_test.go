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

func TestGetOrCreateAccount(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewTwilioStore(db)

	// First call should create
	account, err := store.GetOrCreateAccount("AC123")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}
	if account.AccountSid != "AC123" {
		t.Fatalf("Expected SID 'AC123', got '%s'", account.AccountSid)
	}
	if account.AuthToken == "" {
		t.Fatal("Auth token should be generated")
	}

	originalToken := account.AuthToken

	// Second call should return same account
	account2, err := store.GetOrCreateAccount("AC123")
	if err != nil {
		t.Fatalf("Failed to get account: %v", err)
	}
	if account2.AuthToken != originalToken {
		t.Fatal("Auth token should not change on second call")
	}
}

func TestValidateAccount(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewTwilioStore(db)

	account, _ := store.GetOrCreateAccount("AC456")

	// Valid credentials
	if !store.ValidateAccount("AC456", account.AuthToken) {
		t.Fatal("Valid credentials should pass")
	}

	// Invalid token
	if store.ValidateAccount("AC456", "wrongtoken") {
		t.Fatal("Invalid token should fail")
	}

	// Invalid account
	if store.ValidateAccount("ACnonexistent", "anytoken") {
		t.Fatal("Nonexistent account should fail")
	}
}
