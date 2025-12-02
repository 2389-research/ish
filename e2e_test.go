// ABOUTME: End-to-end integration tests for the ISH server.
// ABOUTME: Verifies full request/response flow across all APIs.

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/2389/ish/internal/auth"
	"github.com/2389/ish/internal/calendar"
	"github.com/2389/ish/internal/gmail"
	"github.com/2389/ish/internal/people"
	"github.com/2389/ish/internal/store"
)

func setupTestServer(t *testing.T) (*httptest.Server, func()) {
	dbPath := "test_e2e.db"

	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("store.New() error = %v", err)
	}

	// Seed test data
	s.CreateUser("harper")
	s.CreateGmailThread(&store.GmailThread{ID: "thr_1", UserID: "harper", Snippet: "Test thread"})
	s.CreateGmailMessage(&store.GmailMessage{ID: "msg_1", UserID: "harper", ThreadID: "thr_1", LabelIDs: []string{"INBOX"}, Snippet: "Test message", InternalDate: 1733000000000, Payload: `{"headers":[]}`})
	s.CreateCalendar(&store.Calendar{ID: "primary", UserID: "harper", Summary: "Primary"})
	s.CreateCalendarEvent(&store.CalendarEvent{ID: "evt_1", CalendarID: "primary", Summary: "Test Event", StartTime: "2025-12-01T10:00:00Z", EndTime: "2025-12-01T11:00:00Z"})
	s.CreatePerson(&store.Person{ResourceName: "people/c1", UserID: "harper", Data: `{"names":[{"displayName":"Test Person"}]}`})

	r := chi.NewRouter()
	r.Use(auth.Middleware)
	gmail.NewHandlers(s).RegisterRoutes(r)
	calendar.NewHandlers(s).RegisterRoutes(r)
	people.NewHandlers(s).RegisterRoutes(r)

	srv := httptest.NewServer(r)

	cleanup := func() {
		srv.Close()
		s.Close()
		os.Remove(dbPath)
	}

	return srv, cleanup
}

func TestE2E_GmailFlow(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client := srv.Client()

	// List messages
	req, _ := http.NewRequest("GET", srv.URL+"/gmail/v1/users/me/messages", nil)
	req.Header.Set("Authorization", "Bearer user:harper")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("list messages error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("list status = %d, want 200", resp.StatusCode)
	}

	var listResp map[string]any
	json.NewDecoder(resp.Body).Decode(&listResp)
	msgs := listResp["messages"].([]any)
	if len(msgs) != 1 {
		t.Errorf("messages count = %d, want 1", len(msgs))
	}

	// Get single message
	req, _ = http.NewRequest("GET", srv.URL+"/gmail/v1/users/me/messages/msg_1", nil)
	req.Header.Set("Authorization", "Bearer user:harper")
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("get message error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("get status = %d, want 200", resp.StatusCode)
	}
}

func TestE2E_CalendarFlow(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client := srv.Client()

	// List events
	req, _ := http.NewRequest("GET", srv.URL+"/calendar/v3/calendars/primary/events", nil)
	req.Header.Set("Authorization", "Bearer user:harper")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("list events error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("list status = %d, want 200", resp.StatusCode)
	}

	var listResp map[string]any
	json.NewDecoder(resp.Body).Decode(&listResp)
	items := listResp["items"].([]any)
	if len(items) != 1 {
		t.Errorf("items count = %d, want 1", len(items))
	}

	// Get single event
	req, _ = http.NewRequest("GET", srv.URL+"/calendar/v3/calendars/primary/events/evt_1", nil)
	req.Header.Set("Authorization", "Bearer user:harper")
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("get event error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("get status = %d, want 200", resp.StatusCode)
	}
}

func TestE2E_PeopleFlow(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	client := srv.Client()

	// Search contacts
	req, _ := http.NewRequest("GET", srv.URL+"/people/v1/people:searchContacts?query=Test", nil)
	req.Header.Set("Authorization", "Bearer user:harper")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("search contacts error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("search status = %d, want 200", resp.StatusCode)
	}

	var searchResp map[string]any
	json.NewDecoder(resp.Body).Decode(&searchResp)
	results := searchResp["results"].([]any)
	if len(results) != 1 {
		t.Errorf("results count = %d, want 1", len(results))
	}

	// Get single person
	req, _ = http.NewRequest("GET", srv.URL+"/people/v1/people/c1", nil)
	req.Header.Set("Authorization", "Bearer user:harper")
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("get person error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("get status = %d, want 200", resp.StatusCode)
	}
}
