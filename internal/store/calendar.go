// ABOUTME: Calendar-related store operations for calendars and events.
// ABOUTME: Handles CRUD and listing with time range filtering.

package store

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"strconv"
)

type Calendar struct {
	ID      string
	UserID  string
	Summary string
}

type CalendarEvent struct {
	ID          string
	CalendarID  string
	Summary     string
	Description string
	StartTime   string
	EndTime     string
	Attendees   string
}

func (s *Store) CreateCalendar(c *Calendar) error {
	_, err := s.db.Exec(
		"INSERT INTO calendars (id, user_id, summary) VALUES (?, ?, ?)",
		c.ID, c.UserID, c.Summary,
	)
	return err
}

func (s *Store) CreateCalendarEvent(e *CalendarEvent) error {
	_, err := s.db.Exec(
		"INSERT INTO calendar_events (id, calendar_id, summary, description, start_time, end_time, attendees) VALUES (?, ?, ?, ?, ?, ?, ?)",
		e.ID, e.CalendarID, e.Summary, e.Description, e.StartTime, e.EndTime, e.Attendees,
	)
	return err
}

func (s *Store) ListCalendarEvents(calendarID string, maxResults int, pageToken string, timeMin string, timeMax string) ([]CalendarEvent, string, error) {
	offset := 0
	if pageToken != "" {
		decoded, err := base64.StdEncoding.DecodeString(pageToken)
		if err == nil {
			offset, _ = strconv.Atoi(string(decoded))
		}
	}

	sqlQuery := "SELECT id, calendar_id, summary, description, start_time, end_time, attendees FROM calendar_events WHERE calendar_id = ?"
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
		err := rows.Scan(&e.ID, &e.CalendarID, &e.Summary, &e.Description, &e.StartTime, &e.EndTime, &e.Attendees)
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
		"SELECT id, calendar_id, summary, description, start_time, end_time, attendees FROM calendar_events WHERE calendar_id = ? AND id = ?",
		calendarID, eventID,
	).Scan(&e.ID, &e.CalendarID, &e.Summary, &e.Description, &e.StartTime, &e.EndTime, &e.Attendees)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("event not found")
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}
