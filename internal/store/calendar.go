// ABOUTME: Calendar-related store operations for calendars and events.
// ABOUTME: Handles CRUD and listing with time range filtering.

package store

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"
)

type Calendar struct {
	ID      string
	UserID  string
	Summary string
}

type CalendarEvent struct {
	ID             string
	CalendarID     string
	Summary        string
	Description    string
	StartTime      string
	EndTime        string
	Attendees      string
	Location       string
	OrganizerEmail string
	OrganizerName  string
	Recurrence     string
	UpdatedAt      string
}

func (s *Store) CreateCalendar(c *Calendar) error {
	_, err := s.db.Exec(
		"INSERT INTO calendars (id, user_id, summary) VALUES (?, ?, ?)",
		c.ID, c.UserID, c.Summary,
	)
	return err
}

func (s *Store) CreateCalendarEvent(e *CalendarEvent) (*CalendarEvent, error) {
	// Generate ID if not provided
	if e.ID == "" {
		e.ID = fmt.Sprintf("evt_%d", time.Now().UnixNano())
	}

	// Set updated_at timestamp
	e.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	_, err := s.db.Exec(
		`INSERT INTO calendar_events (id, calendar_id, summary, description, start_time, end_time, attendees, location, recurrence, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.CalendarID, e.Summary, e.Description, e.StartTime, e.EndTime, e.Attendees, e.Location, e.Recurrence, e.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (s *Store) ListCalendarEvents(calendarID string, maxResults int, pageToken string, timeMin string, timeMax string) ([]CalendarEvent, string, error) {
	offset := 0
	if pageToken != "" {
		decoded, err := base64.StdEncoding.DecodeString(pageToken)
		if err == nil {
			offset, _ = strconv.Atoi(string(decoded))
		}
	}

	sqlQuery := `SELECT id, calendar_id, summary, description, start_time, end_time, attendees,
		COALESCE(location, ''), COALESCE(organizer_email, ''), COALESCE(organizer_name, ''),
		COALESCE(recurrence, ''), COALESCE(updated_at, '') FROM calendar_events WHERE calendar_id = ?`
	args := []any{calendarID}

	if timeMin != "" {
		sqlQuery += " AND start_time >= ?"
		args = append(args, timeMin)
	}
	if timeMax != "" {
		sqlQuery += " AND start_time <= ?"
		args = append(args, timeMax)
	}

	sqlQuery += " ORDER BY start_time ASC LIMIT ? OFFSET ?"
	args = append(args, maxResults+1, offset)

	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var events []CalendarEvent
	for rows.Next() {
		var e CalendarEvent
		err := rows.Scan(&e.ID, &e.CalendarID, &e.Summary, &e.Description, &e.StartTime, &e.EndTime, &e.Attendees,
			&e.Location, &e.OrganizerEmail, &e.OrganizerName, &e.Recurrence, &e.UpdatedAt)
		if err != nil {
			return nil, "", err
		}
		events = append(events, e)
	}

	var nextToken string
	if len(events) > maxResults {
		events = events[:maxResults]
		nextToken = base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset + maxResults)))
	}

	return events, nextToken, nil
}

func (s *Store) GetCalendarEvent(calendarID, eventID string) (*CalendarEvent, error) {
	var e CalendarEvent
	err := s.db.QueryRow(
		`SELECT id, calendar_id, summary, description, start_time, end_time, attendees,
		COALESCE(location, ''), COALESCE(organizer_email, ''), COALESCE(organizer_name, ''),
		COALESCE(recurrence, ''), COALESCE(updated_at, '') FROM calendar_events WHERE calendar_id = ? AND id = ?`,
		calendarID, eventID,
	).Scan(&e.ID, &e.CalendarID, &e.Summary, &e.Description, &e.StartTime, &e.EndTime, &e.Attendees,
		&e.Location, &e.OrganizerEmail, &e.OrganizerName, &e.Recurrence, &e.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("event not found")
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (s *Store) ListAllCalendarEvents() ([]CalendarEvent, error) {
	rows, err := s.db.Query(`SELECT id, calendar_id, summary, description, start_time, end_time, attendees,
		COALESCE(location, ''), COALESCE(organizer_email, ''), COALESCE(organizer_name, ''),
		COALESCE(recurrence, ''), COALESCE(updated_at, '') FROM calendar_events ORDER BY start_time`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []CalendarEvent
	for rows.Next() {
		var e CalendarEvent
		if err := rows.Scan(&e.ID, &e.CalendarID, &e.Summary, &e.Description, &e.StartTime, &e.EndTime, &e.Attendees,
			&e.Location, &e.OrganizerEmail, &e.OrganizerName, &e.Recurrence, &e.UpdatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

func (s *Store) CreateCalendarEventFromForm(summary, description, start, end string) (*CalendarEvent, error) {
	id := fmt.Sprintf("evt_%d", time.Now().UnixNano())

	// Convert datetime-local format to ISO 8601
	startTime := start + ":00Z"
	endTime := end + ":00Z"

	_, err := s.db.Exec(
		"INSERT INTO calendar_events (id, calendar_id, summary, description, start_time, end_time, attendees) VALUES (?, ?, ?, ?, ?, ?, ?)",
		id, "primary", summary, description, startTime, endTime, "[]",
	)
	if err != nil {
		return nil, err
	}

	return &CalendarEvent{
		ID:          id,
		CalendarID:  "primary",
		Summary:     summary,
		Description: description,
		StartTime:   startTime,
		EndTime:     endTime,
		Attendees:   "[]",
	}, nil
}

// UpdateCalendarEvent updates an existing calendar event
func (s *Store) UpdateCalendarEvent(e *CalendarEvent) (*CalendarEvent, error) {
	// Update timestamp
	e.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	_, err := s.db.Exec(
		`UPDATE calendar_events SET summary = ?, description = ?, start_time = ?, end_time = ?,
		 attendees = ?, location = ?, recurrence = ?, updated_at = ?
		 WHERE calendar_id = ? AND id = ?`,
		e.Summary, e.Description, e.StartTime, e.EndTime, e.Attendees, e.Location, e.Recurrence, e.UpdatedAt,
		e.CalendarID, e.ID,
	)
	if err != nil {
		return nil, err
	}
	return e, nil
}

// DeleteCalendarEvent deletes a calendar event
func (s *Store) DeleteCalendarEvent(calendarID, eventID string) error {
	_, err := s.db.Exec("DELETE FROM calendar_events WHERE calendar_id = ? AND id = ?", calendarID, eventID)
	return err
}

// GetCalendarSyncToken returns the current sync token for a calendar.
func (s *Store) GetCalendarSyncToken(calendarID string) (string, error) {
	var token string
	err := s.db.QueryRow("SELECT COALESCE(sync_token, '') FROM calendars WHERE id = ?", calendarID).Scan(&token)
	if err != nil || token == "" {
		// If calendar doesn't exist or no sync token, generate a new token
		token = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	}
	return token, nil
}

// UpdateCalendarSyncToken updates the sync token for a calendar.
func (s *Store) UpdateCalendarSyncToken(calendarID string) (string, error) {
	token := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	_, err := s.db.Exec("UPDATE calendars SET sync_token = ? WHERE id = ?", token, calendarID)
	if err != nil {
		return "", err
	}
	return token, nil
}

// ListCalendarEventsSince returns events updated since the given sync token.
func (s *Store) ListCalendarEventsSince(calendarID string, syncToken string, maxResults int) ([]CalendarEvent, string, error) {
	// Decode the sync token to get a timestamp
	var sinceTime int64
	if syncToken != "" {
		decoded, err := base64.StdEncoding.DecodeString(syncToken)
		if err == nil {
			sinceTime, _ = strconv.ParseInt(string(decoded), 10, 64)
		}
	}

	// Convert nanoseconds to timestamp for comparison
	sinceTimestamp := time.Unix(0, sinceTime).Format(time.RFC3339)

	// Use COALESCE in WHERE clause to handle NULL updated_at values (treat them as epoch 0)
	sqlQuery := `SELECT id, calendar_id, summary, description, start_time, end_time, attendees,
		COALESCE(location, ''), COALESCE(organizer_email, ''), COALESCE(organizer_name, ''),
		COALESCE(recurrence, ''), COALESCE(updated_at, '') FROM calendar_events
		WHERE calendar_id = ? AND COALESCE(updated_at, '1970-01-01T00:00:00Z') > ? ORDER BY updated_at ASC LIMIT ?`

	rows, err := s.db.Query(sqlQuery, calendarID, sinceTimestamp, maxResults)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var events []CalendarEvent
	for rows.Next() {
		var e CalendarEvent
		if err := rows.Scan(&e.ID, &e.CalendarID, &e.Summary, &e.Description, &e.StartTime, &e.EndTime, &e.Attendees,
			&e.Location, &e.OrganizerEmail, &e.OrganizerName, &e.Recurrence, &e.UpdatedAt); err != nil {
			return nil, "", err
		}
		events = append(events, e)
	}

	// Generate new sync token based on current time
	newToken := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	return events, newToken, nil
}
