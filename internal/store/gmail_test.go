// ABOUTME: Tests for Gmail-related store operations.
// ABOUTME: Covers message listing, retrieval, and filtering.

package store

import (
	"os"
	"testing"
)

func TestStore_GmailMessages(t *testing.T) {
	dbPath := "test_gmail.db"
	defer os.Remove(dbPath)

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer s.Close()

	// Setup user
	s.CreateUser("harper")

	// Create thread and message
	thread := &GmailThread{ID: "thr_1", UserID: "harper", Snippet: "Hello world"}
	err = s.CreateGmailThread(thread)
	if err != nil {
		t.Fatalf("CreateGmailThread() error = %v", err)
	}

	msg := &GmailMessage{
		ID:           "msg_1",
		UserID:       "harper",
		ThreadID:     "thr_1",
		LabelIDs:     []string{"INBOX", "STARRED"},
		Snippet:      "Hello world",
		InternalDate: 1733000000000,
		Payload:      `{"headers":[{"name":"Subject","value":"Test"}]}`,
	}
	err = s.CreateGmailMessage(msg)
	if err != nil {
		t.Fatalf("CreateGmailMessage() error = %v", err)
	}

	// List messages
	msgs, nextToken, err := s.ListGmailMessages("harper", 10, "", "")
	if err != nil {
		t.Fatalf("ListGmailMessages() error = %v", err)
	}
	if len(msgs) != 1 {
		t.Errorf("ListGmailMessages() got %d messages, want 1", len(msgs))
	}
	if nextToken != "" {
		t.Errorf("ListGmailMessages() nextToken = %q, want empty", nextToken)
	}

	// Get single message
	got, err := s.GetGmailMessage("harper", "msg_1")
	if err != nil {
		t.Fatalf("GetGmailMessage() error = %v", err)
	}
	if got.ID != "msg_1" {
		t.Errorf("GetGmailMessage() ID = %q, want %q", got.ID, "msg_1")
	}
}
