// ABOUTME: Calendar API handlers for Google plugin.
// ABOUTME: Implements Calendar v3 API endpoints.

package google

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/2389/ish/internal/store"
	"github.com/go-chi/chi/v5"
)

func (p *GooglePlugin) registerCalendarRoutes(r chi.Router) {
	// Standard Google Calendar API v3 routes
	r.Route("/calendar/v3/calendars/{calendarId}", func(r chi.Router) {
		r.Get("/events", p.listEvents)
		r.Post("/events", p.createEvent)
		r.Get("/events/{eventId}", p.getEvent)
		r.Put("/events/{eventId}", p.updateEvent)
		r.Patch("/events/{eventId}", p.updateEvent)
		r.Delete("/events/{eventId}", p.deleteEvent)
	})

	// Alias routes without /calendar/v3/ prefix (some clients strip this)
	r.Route("/calendars/{calendarId}", func(r chi.Router) {
		r.Get("/events", p.listEvents)
		r.Post("/events", p.createEvent)
		r.Get("/events/{eventId}", p.getEvent)
		r.Put("/events/{eventId}", p.updateEvent)
		r.Patch("/events/{eventId}", p.updateEvent)
		r.Delete("/events/{eventId}", p.deleteEvent)
	})
}

func (p *GooglePlugin) listEvents(w http.ResponseWriter, r *http.Request) {
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
	syncToken := r.URL.Query().Get("syncToken")

	var events []store.CalendarEvent
	var nextToken string
	var nextSyncToken string
	var err error

	if syncToken != "" {
		// Incremental sync - get events since last sync
		events, nextSyncToken, err = p.store.ListCalendarEventsSince(calendarID, syncToken, maxResults)
	} else {
		// Full sync
		events, nextToken, err = p.store.ListCalendarEvents(calendarID, maxResults, pageToken, timeMin, timeMax)
		if err == nil && nextToken == "" {
			// Generate sync token for subsequent incremental syncs
			nextSyncToken, _ = p.store.GetCalendarSyncToken(calendarID)
		}
	}

	if err != nil {
		writeError(w, 500, "Internal error", "INTERNAL")
		return
	}

	// Convert to response format
	items := make([]map[string]any, len(events))
	for i, e := range events {
		var attendees []any
		json.Unmarshal([]byte(e.Attendees), &attendees)

		item := map[string]any{
			"id":          e.ID,
			"summary":     e.Summary,
			"description": e.Description,
			"start":       map[string]string{"dateTime": e.StartTime},
			"end":         map[string]string{"dateTime": e.EndTime},
			"attendees":   attendees,
		}

		// Include optional fields if present
		if e.Location != "" {
			item["location"] = e.Location
		}
		if e.OrganizerEmail != "" || e.OrganizerName != "" {
			item["organizer"] = map[string]any{
				"email":       e.OrganizerEmail,
				"displayName": e.OrganizerName,
			}
		}
		if e.Recurrence != "" {
			var recurrence []string
			json.Unmarshal([]byte(e.Recurrence), &recurrence)
			if len(recurrence) > 0 {
				item["recurrence"] = recurrence
			}
		}
		if e.UpdatedAt != "" {
			item["updated"] = e.UpdatedAt
		}

		items[i] = item
	}

	resp := map[string]any{
		"kind":  "calendar#events",
		"items": items,
	}
	if nextToken != "" {
		resp["nextPageToken"] = nextToken
	}
	if nextSyncToken != "" {
		resp["nextSyncToken"] = nextSyncToken
	}

	writeJSON(w, resp)
}

func (p *GooglePlugin) getEvent(w http.ResponseWriter, r *http.Request) {
	calendarID := chi.URLParam(r, "calendarId")
	eventID := chi.URLParam(r, "eventId")

	evt, err := p.store.GetCalendarEvent(calendarID, eventID)
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

	// Include optional fields if present
	if evt.Location != "" {
		resp["location"] = evt.Location
	}
	if evt.OrganizerEmail != "" || evt.OrganizerName != "" {
		resp["organizer"] = map[string]any{
			"email":       evt.OrganizerEmail,
			"displayName": evt.OrganizerName,
		}
	}
	if evt.Recurrence != "" {
		var recurrence []string
		json.Unmarshal([]byte(evt.Recurrence), &recurrence)
		if len(recurrence) > 0 {
			resp["recurrence"] = recurrence
		}
	}
	if evt.UpdatedAt != "" {
		resp["updated"] = evt.UpdatedAt
	}

	writeJSON(w, resp)
}

func (p *GooglePlugin) createEvent(w http.ResponseWriter, r *http.Request) {
	calendarID := chi.URLParam(r, "calendarId")

	var req struct {
		Summary     string `json:"summary"`
		Description string `json:"description"`
		Location    string `json:"location"`
		Start       struct {
			DateTime string `json:"dateTime"`
			Date     string `json:"date"`
		} `json:"start"`
		End struct {
			DateTime string `json:"dateTime"`
			Date     string `json:"date"`
		} `json:"end"`
		Attendees []struct {
			Email string `json:"email"`
		} `json:"attendees"`
		Recurrence []string `json:"recurrence"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "Invalid request body", "INVALID_REQUEST")
		return
	}

	// Get start/end times (prefer dateTime over date)
	startTime := req.Start.DateTime
	if startTime == "" {
		startTime = req.Start.Date
	}
	endTime := req.End.DateTime
	if endTime == "" {
		endTime = req.End.Date
	}

	if req.Summary == "" || startTime == "" || endTime == "" {
		writeError(w, 400, "Missing required fields: summary, start, end", "INVALID_REQUEST")
		return
	}

	// Convert attendees to JSON
	attendeesJSON, _ := json.Marshal(req.Attendees)

	// Convert recurrence to JSON
	recurrenceJSON := ""
	if len(req.Recurrence) > 0 {
		bytes, _ := json.Marshal(req.Recurrence)
		recurrenceJSON = string(bytes)
	}

	event, err := p.store.CreateCalendarEvent(&store.CalendarEvent{
		CalendarID:  calendarID,
		Summary:     req.Summary,
		Description: req.Description,
		Location:    req.Location,
		StartTime:   startTime,
		EndTime:     endTime,
		Attendees:   string(attendeesJSON),
		Recurrence:  recurrenceJSON,
	})
	if err != nil {
		writeError(w, 500, "Failed to create event", "INTERNAL")
		return
	}

	// Parse attendees back to array
	var attendees []any
	json.Unmarshal([]byte(event.Attendees), &attendees)

	resp := map[string]any{
		"kind":        "calendar#event",
		"id":          event.ID,
		"summary":     event.Summary,
		"description": event.Description,
		"location":    event.Location,
		"start":       map[string]string{"dateTime": event.StartTime},
		"end":         map[string]string{"dateTime": event.EndTime},
		"attendees":   attendees,
		"status":      "confirmed",
		"htmlLink":    "https://calendar.google.com/calendar/event?eid=" + event.ID,
		"created":     event.UpdatedAt,
		"updated":     event.UpdatedAt,
	}

	if event.Recurrence != "" {
		var recurrence []string
		json.Unmarshal([]byte(event.Recurrence), &recurrence)
		if len(recurrence) > 0 {
			resp["recurrence"] = recurrence
		}
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, resp)
}

func (p *GooglePlugin) updateEvent(w http.ResponseWriter, r *http.Request) {
	calendarID := chi.URLParam(r, "calendarId")
	eventID := chi.URLParam(r, "eventId")

	// First get the existing event
	existing, err := p.store.GetCalendarEvent(calendarID, eventID)
	if err != nil {
		writeError(w, 404, "Event not found", "NOT_FOUND")
		return
	}

	var req struct {
		Summary     *string `json:"summary"`
		Description *string `json:"description"`
		Location    *string `json:"location"`
		Start       *struct {
			DateTime string `json:"dateTime"`
			Date     string `json:"date"`
		} `json:"start"`
		End *struct {
			DateTime string `json:"dateTime"`
			Date     string `json:"date"`
		} `json:"end"`
		Attendees *[]struct {
			Email string `json:"email"`
		} `json:"attendees"`
		Recurrence *[]string `json:"recurrence"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "Invalid request body", "INVALID_REQUEST")
		return
	}

	// Update fields if provided
	if req.Summary != nil {
		existing.Summary = *req.Summary
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.Location != nil {
		existing.Location = *req.Location
	}
	if req.Start != nil {
		if req.Start.DateTime != "" {
			existing.StartTime = req.Start.DateTime
		} else if req.Start.Date != "" {
			existing.StartTime = req.Start.Date
		}
	}
	if req.End != nil {
		if req.End.DateTime != "" {
			existing.EndTime = req.End.DateTime
		} else if req.End.Date != "" {
			existing.EndTime = req.End.Date
		}
	}
	if req.Attendees != nil {
		bytes, _ := json.Marshal(*req.Attendees)
		existing.Attendees = string(bytes)
	}
	if req.Recurrence != nil {
		bytes, _ := json.Marshal(*req.Recurrence)
		existing.Recurrence = string(bytes)
	}

	updated, err := p.store.UpdateCalendarEvent(existing)
	if err != nil {
		writeError(w, 500, "Failed to update event", "INTERNAL")
		return
	}

	// Parse attendees back to array
	var attendees []any
	json.Unmarshal([]byte(updated.Attendees), &attendees)

	resp := map[string]any{
		"kind":        "calendar#event",
		"id":          updated.ID,
		"summary":     updated.Summary,
		"description": updated.Description,
		"location":    updated.Location,
		"start":       map[string]string{"dateTime": updated.StartTime},
		"end":         map[string]string{"dateTime": updated.EndTime},
		"attendees":   attendees,
		"status":      "confirmed",
		"htmlLink":    "https://calendar.google.com/calendar/event?eid=" + updated.ID,
		"updated":     updated.UpdatedAt,
	}

	if updated.Recurrence != "" {
		var recurrence []string
		json.Unmarshal([]byte(updated.Recurrence), &recurrence)
		if len(recurrence) > 0 {
			resp["recurrence"] = recurrence
		}
	}

	writeJSON(w, resp)
}

func (p *GooglePlugin) deleteEvent(w http.ResponseWriter, r *http.Request) {
	calendarID := chi.URLParam(r, "calendarId")
	eventID := chi.URLParam(r, "eventId")

	err := p.store.DeleteCalendarEvent(calendarID, eventID)
	if err != nil {
		writeError(w, 404, "Event not found", "NOT_FOUND")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
