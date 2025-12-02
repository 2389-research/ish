// ABOUTME: Tests for Gmail API HTTP handlers.
// ABOUTME: Verifies message listing and retrieval endpoints.

package gmail

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/2389/ish/internal/auth"
	"github.com/2389/ish/internal/store"
)

func TestHandlers_ListMessages(t *testing.T) {
	dbPath := "test_gmail_handlers.db"
	defer os.Remove(dbPath)

	s, _ := store.New(dbPath)
	defer s.Close()

	// Seed data
	s.CreateUser("harper")
	s.CreateGmailThread(&store.GmailThread{ID: "thr_1", UserID: "harper", Snippet: "Test"})
	s.CreateGmailMessage(&store.GmailMessage{
		ID: "msg_1", UserID: "harper", ThreadID: "thr_1",
		LabelIDs: []string{"INBOX"}, Snippet: "Hello", InternalDate: 1733000000000,
	})

	h := NewHandlers(s)
	r := chi.NewRouter()
	r.Use(auth.Middleware)
	h.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/gmail/v1/users/me/messages", nil)
	req.Header.Set("Authorization", "Bearer user:harper")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)
	msgs, ok := resp["messages"].([]any)
	if !ok || len(msgs) != 1 {
		t.Errorf("messages count = %d, want 1", len(msgs))
	}
}

func TestHandlers_GetMessage(t *testing.T) {
	dbPath := "test_gmail_handlers2.db"
	defer os.Remove(dbPath)

	s, _ := store.New(dbPath)
	defer s.Close()

	// Seed data
	s.CreateUser("harper")
	s.CreateGmailThread(&store.GmailThread{ID: "thr_1", UserID: "harper", Snippet: "Test"})
	s.CreateGmailMessage(&store.GmailMessage{
		ID: "msg_1", UserID: "harper", ThreadID: "thr_1",
		LabelIDs: []string{"INBOX"}, Snippet: "Hello", InternalDate: 1733000000000,
		Payload: `{"headers":[{"name":"Subject","value":"Test"}]}`,
	})

	h := NewHandlers(s)
	r := chi.NewRouter()
	r.Use(auth.Middleware)
	h.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/gmail/v1/users/me/messages/msg_1", nil)
	req.Header.Set("Authorization", "Bearer user:harper")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["id"] != "msg_1" {
		t.Errorf("id = %v, want msg_1", resp["id"])
	}
}
