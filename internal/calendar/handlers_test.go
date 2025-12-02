// ABOUTME: Tests for Calendar API HTTP handlers.
// ABOUTME: Verifies event listing and retrieval endpoints.

package calendar

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

func TestHandlers_ListEvents(t *testing.T) {
	dbPath := "test_calendar_handlers.db"
	defer os.Remove(dbPath)

	s, _ := store.New(dbPath)
	defer s.Close()

	// Seed data
	s.CreateUser("harper")
	s.CreateCalendar(&store.Calendar{ID: "primary", UserID: "harper", Summary: "Primary"})
	s.CreateCalendarEvent(&store.CalendarEvent{
		ID: "evt_1", CalendarID: "primary", Summary: "Coffee",
		StartTime: "2025-12-01T10:00:00Z", EndTime: "2025-12-01T11:00:00Z",
	})

	h := NewHandlers(s)
	r := chi.NewRouter()
	r.Use(auth.Middleware)
	h.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/calendar/v3/calendars/primary/events", nil)
	req.Header.Set("Authorization", "Bearer user:harper")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)
	items, ok := resp["items"].([]any)
	if !ok || len(items) != 1 {
		t.Errorf("items count = %d, want 1", len(items))
	}
}

func TestHandlers_GetEvent(t *testing.T) {
	dbPath := "test_calendar_handlers2.db"
	defer os.Remove(dbPath)

	s, _ := store.New(dbPath)
	defer s.Close()

	// Seed data
	s.CreateUser("harper")
	s.CreateCalendar(&store.Calendar{ID: "primary", UserID: "harper", Summary: "Primary"})
	s.CreateCalendarEvent(&store.CalendarEvent{
		ID: "evt_1", CalendarID: "primary", Summary: "Coffee",
		Description: "Discuss things",
		StartTime: "2025-12-01T10:00:00Z", EndTime: "2025-12-01T11:00:00Z",
		Attendees: `[{"email":"harper@example.com"}]`,
	})

	h := NewHandlers(s)
	r := chi.NewRouter()
	r.Use(auth.Middleware)
	h.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/calendar/v3/calendars/primary/events/evt_1", nil)
	req.Header.Set("Authorization", "Bearer user:harper")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["summary"] != "Coffee" {
		t.Errorf("summary = %v, want Coffee", resp["summary"])
	}
}
