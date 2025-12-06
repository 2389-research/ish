// ABOUTME: Tests for seed functionality
// ABOUTME: Verifies seed data generation works correctly

package twilio

import (
	"context"
	"testing"
)

func TestSeed(t *testing.T) {
	plugin, db := setupTestPlugin(t)
	defer db.Close()

	// Test small seed
	seedData, err := plugin.Seed(context.Background(), "small")
	if err != nil {
		t.Fatalf("Seed failed: %v", err)
	}

	// Verify summary
	if seedData.Summary == "" {
		t.Fatal("Seed summary should not be empty")
	}

	// Verify record counts
	if seedData.Records["accounts"] != 1 {
		t.Fatalf("Expected 1 account, got %d", seedData.Records["accounts"])
	}
	if seedData.Records["phone_numbers"] != 3 {
		t.Fatalf("Expected 3 phone numbers, got %d", seedData.Records["phone_numbers"])
	}
	if seedData.Records["messages"] != 10 {
		t.Fatalf("Expected 10 messages, got %d", seedData.Records["messages"])
	}
	if seedData.Records["calls"] != 5 {
		t.Fatalf("Expected 5 calls, got %d", seedData.Records["calls"])
	}

	// Verify actual DB counts
	var count int

	db.QueryRow("SELECT COUNT(*) FROM twilio_accounts").Scan(&count)
	if count != 1 {
		t.Fatalf("Expected 1 account in DB, got %d", count)
	}

	db.QueryRow("SELECT COUNT(*) FROM twilio_phone_numbers").Scan(&count)
	if count != 3 {
		t.Fatalf("Expected 3 phone numbers in DB, got %d", count)
	}

	db.QueryRow("SELECT COUNT(*) FROM twilio_messages").Scan(&count)
	if count != 10 {
		t.Fatalf("Expected 10 messages in DB, got %d", count)
	}

	db.QueryRow("SELECT COUNT(*) FROM twilio_calls").Scan(&count)
	if count != 5 {
		t.Fatalf("Expected 5 calls in DB, got %d", count)
	}

	// Verify phone numbers have correct format
	var phoneNumber string
	db.QueryRow("SELECT phone_number FROM twilio_phone_numbers LIMIT 1").Scan(&phoneNumber)
	if phoneNumber[:5] != "+1555" {
		t.Fatalf("Expected phone number to start with +1555, got %s", phoneNumber)
	}

	// Verify messages have varied statuses
	var status string
	var hasQueued, hasSent, hasDelivered bool
	rows, _ := db.Query("SELECT status FROM twilio_messages")
	for rows.Next() {
		rows.Scan(&status)
		if status == "queued" {
			hasQueued = true
		} else if status == "sent" {
			hasSent = true
		} else if status == "delivered" {
			hasDelivered = true
		}
	}
	rows.Close()

	if !hasQueued && !hasSent && !hasDelivered {
		t.Fatal("Messages should have varied statuses")
	}

	// Verify calls have varied statuses
	var hasInitiated, hasInProgress, hasCompleted bool
	rows, _ = db.Query("SELECT status FROM twilio_calls")
	for rows.Next() {
		rows.Scan(&status)
		if status == "initiated" {
			hasInitiated = true
		} else if status == "in-progress" {
			hasInProgress = true
		} else if status == "completed" {
			hasCompleted = true
		}
	}
	rows.Close()

	if !hasInitiated && !hasInProgress && !hasCompleted {
		t.Fatal("Calls should have varied statuses")
	}
}

func TestSeedSizes(t *testing.T) {
	tests := []struct {
		size         string
		accounts     int
		phoneNumbers int
		messages     int
		calls        int
	}{
		{"small", 1, 3, 10, 5},
		{"medium", 3, 10, 50, 20},
		{"large", 10, 30, 200, 100},
		{"", 1, 3, 10, 5}, // default to small
	}

	for _, tt := range tests {
		t.Run(tt.size, func(t *testing.T) {
			plugin, db := setupTestPlugin(t)
			defer db.Close()

			seedData, err := plugin.Seed(context.Background(), tt.size)
			if err != nil {
				t.Fatalf("Seed failed: %v", err)
			}

			if seedData.Records["accounts"] != tt.accounts {
				t.Errorf("Expected %d accounts, got %d", tt.accounts, seedData.Records["accounts"])
			}
			if seedData.Records["phone_numbers"] != tt.phoneNumbers {
				t.Errorf("Expected %d phone numbers, got %d", tt.phoneNumbers, seedData.Records["phone_numbers"])
			}
			if seedData.Records["messages"] != tt.messages {
				t.Errorf("Expected %d messages, got %d", tt.messages, seedData.Records["messages"])
			}
			if seedData.Records["calls"] != tt.calls {
				t.Errorf("Expected %d calls, got %d", tt.calls, seedData.Records["calls"])
			}
		})
	}
}
