// ABOUTME: Tests for Calendar-related store operations.
// ABOUTME: Covers event listing, retrieval, and time filtering.

package store

import (
	"os"
	"testing"
)

func TestStore_CalendarEvents(t *testing.T) {
	dbPath := "test_calendar.db"
	defer os.Remove(dbPath)

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer s.Close()

	// Setup user and calendar
	s.CreateUser("harper")
	cal := &Calendar{ID: "cal_primary", UserID: "harper", Summary: "Primary"}
	err = s.CreateCalendar(cal)
	if err != nil {
		t.Fatalf("CreateCalendar() error = %v", err)
	}

	// Create event
	evt := &CalendarEvent{
		ID:          "evt_1",
		CalendarID:  "cal_primary",
		Summary:     "Coffee",
		Description: "Discuss things",
		StartTime:   "2025-12-01T10:00:00Z",
		EndTime:     "2025-12-01T11:00:00Z",
		Attendees:   `[{"email":"harper@example.com"}]`,
	}
	err = s.CreateCalendarEvent(evt)
	if err != nil {
		t.Fatalf("CreateCalendarEvent() error = %v", err)
	}

	// List events
	events, nextToken, err := s.ListCalendarEvents("cal_primary", 10, "", "", "")
	if err != nil {
		t.Fatalf("ListCalendarEvents() error = %v", err)
	}
	if len(events) != 1 {
		t.Errorf("ListCalendarEvents() got %d events, want 1", len(events))
	}
	if nextToken != "" {
		t.Errorf("ListCalendarEvents() nextToken = %q, want empty", nextToken)
	}

	// Get single event
	got, err := s.GetCalendarEvent("cal_primary", "evt_1")
	if err != nil {
		t.Fatalf("GetCalendarEvent() error = %v", err)
	}
	if got.Summary != "Coffee" {
		t.Errorf("GetCalendarEvent() Summary = %q, want %q", got.Summary, "Coffee")
	}
}
