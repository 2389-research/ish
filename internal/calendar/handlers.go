// ABOUTME: HTTP handlers for Calendar API endpoints.
// ABOUTME: Implements event listing and retrieval with Google-compatible responses.

package calendar

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/2389/ish/internal/store"
)

type Handlers struct {
	store *store.Store
}

func NewHandlers(s *store.Store) *Handlers {
	return &Handlers{store: s}
}

func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Route("/calendar/v3/calendars/{calendarId}", func(r chi.Router) {
		r.Get("/events", h.listEvents)
		r.Get("/events/{eventId}", h.getEvent)
	})
}

func (h *Handlers) listEvents(w http.ResponseWriter, r *http.Request) {
	calendarID := chi.URLParam(r, "calendarId")

	maxResults := 250
	if mr := r.URL.Query().Get("maxResults"); mr != "" {
		if v, err := strconv.Atoi(mr); err == nil && v > 0 {
			maxResults = v
		}
	}
	pageToken := r.URL.Query().Get("pageToken")
	timeMin := r.URL.Query().Get("timeMin")
	timeMax := r.URL.Query().Get("timeMax")

	events, nextToken, err := h.store.ListCalendarEvents(calendarID, maxResults, pageToken, timeMin, timeMax)
	if err != nil {
		writeError(w, 500, "Internal error", "INTERNAL")
		return
	}

	// Convert to response format
	items := make([]map[string]any, len(events))
	for i, e := range events {
		var attendees []any
		json.Unmarshal([]byte(e.Attendees), &attendees)

		items[i] = map[string]any{
			"id":          e.ID,
			"summary":     e.Summary,
			"description": e.Description,
			"start":       map[string]string{"dateTime": e.StartTime},
			"end":         map[string]string{"dateTime": e.EndTime},
			"attendees":   attendees,
		}
	}

	resp := map[string]any{
		"kind":  "calendar#events",
		"items": items,
	}
	if nextToken != "" {
		resp["nextPageToken"] = nextToken
	}

	writeJSON(w, resp)
}

func (h *Handlers) getEvent(w http.ResponseWriter, r *http.Request) {
	calendarID := chi.URLParam(r, "calendarId")
	eventID := chi.URLParam(r, "eventId")

	evt, err := h.store.GetCalendarEvent(calendarID, eventID)
	if err != nil {
		writeError(w, 404, "Event not found", "NOT_FOUND")
		return
	}

	var attendees []any
	json.Unmarshal([]byte(evt.Attendees), &attendees)

	resp := map[string]any{
		"kind":        "calendar#event",
		"id":          evt.ID,
		"summary":     evt.Summary,
		"description": evt.Description,
		"start":       map[string]string{"dateTime": evt.StartTime},
		"end":         map[string]string{"dateTime": evt.EndTime},
		"attendees":   attendees,
	}

	writeJSON(w, resp)
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, code int, message, status string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
			"status":  status,
		},
	})
}
