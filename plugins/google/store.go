// ABOUTME: Database layer for Google plugin (Gmail, Calendar, People, Tasks)
// ABOUTME: Owns all Google-related tables and queries

package google

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// GoogleStore handles all database operations for the Google plugin
type GoogleStore struct {
	db *sql.DB
}

// NewGoogleStore creates a new GoogleStore with the given database
func NewGoogleStore(db *sql.DB) (*GoogleStore, error) {
	store := &GoogleStore{db: db}
	if err := store.initTables(); err != nil {
		return nil, err
	}
	return store, nil
}

// initTables creates all Google plugin tables if they don't exist
func (s *GoogleStore) initTables() error {
	queries := []string{
		// Gmail tables
		`CREATE TABLE IF NOT EXISTS gmail_messages (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			thread_id TEXT,
			label_ids TEXT,
			snippet TEXT,
			internal_date INTEGER,
			payload TEXT,
			history_id INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_gmail_messages_user_id ON gmail_messages(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_gmail_messages_thread_id ON gmail_messages(thread_id)`,
		`CREATE INDEX IF NOT EXISTS idx_gmail_messages_internal_date ON gmail_messages(internal_date)`,

		`CREATE TABLE IF NOT EXISTS gmail_threads (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			snippet TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_gmail_threads_user_id ON gmail_threads(user_id)`,

		`CREATE TABLE IF NOT EXISTS gmail_attachments (
			id TEXT PRIMARY KEY,
			message_id TEXT NOT NULL,
			filename TEXT,
			mime_type TEXT,
			size INTEGER,
			data TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_gmail_attachments_message_id ON gmail_attachments(message_id)`,

		// Calendar tables
		`CREATE TABLE IF NOT EXISTS calendars (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			summary TEXT,
			sync_token TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_calendars_user_id ON calendars(user_id)`,

		`CREATE TABLE IF NOT EXISTS calendar_events (
			id TEXT PRIMARY KEY,
			calendar_id TEXT NOT NULL,
			summary TEXT,
			description TEXT,
			start_time TEXT,
			end_time TEXT,
			attendees TEXT,
			location TEXT,
			organizer_email TEXT,
			organizer_name TEXT,
			recurrence TEXT,
			updated_at TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_calendar_events_calendar_id ON calendar_events(calendar_id)`,
		`CREATE INDEX IF NOT EXISTS idx_calendar_events_start_time ON calendar_events(start_time)`,

		// People tables
		`CREATE TABLE IF NOT EXISTS people (
			resource_name TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			data TEXT,
			updated_at TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_people_user_id ON people(user_id)`,

		`CREATE TABLE IF NOT EXISTS sync_tokens (
			id TEXT PRIMARY KEY,
			resource_type TEXT NOT NULL,
			user_id TEXT NOT NULL,
			token TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sync_tokens_resource_user ON sync_tokens(resource_type, user_id)`,

		// Tasks tables
		`CREATE TABLE IF NOT EXISTS task_lists (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			title TEXT,
			updated_at TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_task_lists_user_id ON task_lists(user_id)`,

		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			list_id TEXT NOT NULL,
			title TEXT,
			notes TEXT,
			due TEXT,
			status TEXT DEFAULT 'needsAction',
			completed TEXT,
			updated_at TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_list_id ON tasks(list_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status)`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

// Gmail types and methods

type GmailThread struct {
	ID      string
	UserID  string
	Snippet string
}

type GmailMessage struct {
	ID           string
	UserID       string
	ThreadID     string
	LabelIDs     []string
	Snippet      string
	InternalDate int64
	Payload      string
}

type GmailMessageView struct {
	ID       string
	Subject  string
	Snippet  string
	LabelIDs []string
}

type GmailMessageDetail struct {
	ID           string
	ThreadID     string
	Subject      string
	From         string
	To           string
	Date         string
	Body         string
	Snippet      string
	LabelIDs     []string
	InternalDate int64
}

type GmailProfile struct {
	EmailAddress  string
	MessagesTotal int
	ThreadsTotal  int
	HistoryID     int64
}

type GmailHistoryEntry struct {
	ID            int64
	MessagesAdded []GmailMessage
}

type GmailAttachment struct {
	ID        string
	MessageID string
	Filename  string
	MimeType  string
	Size      int
	Data      string
}

func (s *GoogleStore) CreateGmailThread(t *GmailThread) error {
	_, err := s.db.Exec(
		"INSERT INTO gmail_threads (id, user_id, snippet) VALUES (?, ?, ?)",
		t.ID, t.UserID, t.Snippet,
	)
	return err
}

func (s *GoogleStore) CreateGmailMessage(m *GmailMessage) error {
	labelJSON, _ := json.Marshal(m.LabelIDs)
	_, err := s.db.Exec(
		"INSERT INTO gmail_messages (id, user_id, thread_id, label_ids, snippet, internal_date, payload) VALUES (?, ?, ?, ?, ?, ?, ?)",
		m.ID, m.UserID, m.ThreadID, string(labelJSON), m.Snippet, m.InternalDate, m.Payload,
	)
	return err
}

// GmailQueryFilters holds parsed Gmail query filters.
type GmailQueryFilters struct {
	Labels    []string
	AfterDate int64 // Unix milliseconds
}

// parseGmailQuery extracts label filters and date filters from Gmail query syntax.
// Supports: is:unread, is:read, in:inbox, in:sent, label:NAME, after:YYYY/M/D
func parseGmailQuery(query string) GmailQueryFilters {
	var filters GmailQueryFilters
	parts := strings.Fields(query)

	for _, part := range parts {
		partLower := strings.ToLower(part)
		switch {
		case partLower == "is:unread":
			filters.Labels = append(filters.Labels, "UNREAD")
		case partLower == "is:read":
			// For is:read, we'd need to exclude UNREAD - skip for now
		case partLower == "in:inbox":
			filters.Labels = append(filters.Labels, "INBOX")
		case partLower == "in:sent":
			filters.Labels = append(filters.Labels, "SENT")
		case partLower == "in:trash":
			filters.Labels = append(filters.Labels, "TRASH")
		case partLower == "in:spam":
			filters.Labels = append(filters.Labels, "SPAM")
		case partLower == "is:starred":
			filters.Labels = append(filters.Labels, "STARRED")
		case partLower == "is:important":
			filters.Labels = append(filters.Labels, "IMPORTANT")
		case strings.HasPrefix(partLower, "label:"):
			filters.Labels = append(filters.Labels, strings.ToUpper(strings.TrimPrefix(partLower, "label:")))
		case strings.HasPrefix(partLower, "after:"):
			// Parse date in format YYYY/M/D or YYYY/MM/DD
			dateStr := strings.TrimPrefix(partLower, "after:")
			if t, err := time.Parse("2006/1/2", dateStr); err == nil {
				filters.AfterDate = t.UnixMilli()
			} else if t, err := time.Parse("2006/01/02", dateStr); err == nil {
				filters.AfterDate = t.UnixMilli()
			}
		case strings.HasPrefix(partLower, "before:"):
			// Could add before: support if needed
		}
	}

	return filters
}

func (s *GoogleStore) ListGmailMessages(userID string, maxResults int, pageToken string, query string) ([]GmailMessage, string, error) {
	offset := 0
	if pageToken != "" {
		decoded, err := base64.StdEncoding.DecodeString(pageToken)
		if err == nil {
			offset, _ = strconv.Atoi(string(decoded))
		}
	}

	sqlQuery := "SELECT id, user_id, thread_id, label_ids, snippet, internal_date, payload FROM gmail_messages WHERE user_id = ?"
	args := []any{userID}

	// Parse Gmail query syntax
	if query != "" {
		filters := parseGmailQuery(query)
		for _, label := range filters.Labels {
			// Use JSON contains check for label_ids array
			sqlQuery += " AND label_ids LIKE ?"
			args = append(args, "%\""+label+"\"%")
		}
		if filters.AfterDate > 0 {
			sqlQuery += " AND internal_date >= ?"
			args = append(args, filters.AfterDate)
		}
	}

	sqlQuery += " ORDER BY internal_date DESC LIMIT ? OFFSET ?"
	args = append(args, maxResults+1, offset) // +1 to check if there's more

	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var messages []GmailMessage
	for rows.Next() {
		var m GmailMessage
		var labelJSON string
		err := rows.Scan(&m.ID, &m.UserID, &m.ThreadID, &labelJSON, &m.Snippet, &m.InternalDate, &m.Payload)
		if err != nil {
			return nil, "", err
		}
		json.Unmarshal([]byte(labelJSON), &m.LabelIDs)
		messages = append(messages, m)
	}

	var nextToken string
	if len(messages) > maxResults {
		messages = messages[:maxResults]
		nextToken = base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset + maxResults)))
	}

	return messages, nextToken, nil
}

func (s *GoogleStore) GetGmailMessage(userID, messageID string) (*GmailMessage, error) {
	var m GmailMessage
	var labelJSON string
	err := s.db.QueryRow(
		"SELECT id, user_id, thread_id, label_ids, snippet, internal_date, payload FROM gmail_messages WHERE user_id = ? AND id = ?",
		userID, messageID,
	).Scan(&m.ID, &m.UserID, &m.ThreadID, &labelJSON, &m.Snippet, &m.InternalDate, &m.Payload)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("message not found")
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(labelJSON), &m.LabelIDs)
	return &m, nil
}

func (s *GoogleStore) GetGmailMessageDetail(userID, messageID string) (*GmailMessageDetail, error) {
	m, err := s.GetGmailMessage(userID, messageID)
	if err != nil {
		return nil, err
	}

	detail := &GmailMessageDetail{
		ID:           m.ID,
		ThreadID:     m.ThreadID,
		Snippet:      m.Snippet,
		LabelIDs:     m.LabelIDs,
		InternalDate: m.InternalDate,
	}

	// Parse payload for headers and body
	var p struct {
		Headers []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"headers"`
		Body struct {
			Data string `json:"data"`
		} `json:"body"`
		Parts []struct {
			MimeType string `json:"mimeType"`
			Body     struct {
				Data string `json:"data"`
			} `json:"body"`
		} `json:"parts"`
	}
	json.Unmarshal([]byte(m.Payload), &p)

	for _, h := range p.Headers {
		switch h.Name {
		case "Subject":
			detail.Subject = h.Value
		case "From":
			detail.From = h.Value
		case "To":
			detail.To = h.Value
		case "Date":
			detail.Date = h.Value
		}
	}

	// Get body from either direct body or first text/plain part
	if p.Body.Data != "" {
		if decoded, err := base64.URLEncoding.DecodeString(p.Body.Data); err == nil {
			detail.Body = string(decoded)
		}
	} else {
		for _, part := range p.Parts {
			if part.MimeType == "text/plain" && part.Body.Data != "" {
				if decoded, err := base64.URLEncoding.DecodeString(part.Body.Data); err == nil {
					detail.Body = string(decoded)
					break
				}
			}
		}
	}

	return detail, nil
}

func (s *GoogleStore) ListAllGmailMessages() ([]GmailMessageView, error) {
	rows, err := s.db.Query("SELECT id, snippet, label_ids, payload FROM gmail_messages ORDER BY internal_date DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []GmailMessageView
	for rows.Next() {
		var m GmailMessageView
		var labelJSON, payload string
		if err := rows.Scan(&m.ID, &m.Snippet, &labelJSON, &payload); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(labelJSON), &m.LabelIDs)

		// Extract subject from payload
		var p struct {
			Headers []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"headers"`
		}
		json.Unmarshal([]byte(payload), &p)
		for _, h := range p.Headers {
			if h.Name == "Subject" {
				m.Subject = h.Value
				break
			}
		}
		messages = append(messages, m)
	}
	return messages, nil
}

func (s *GoogleStore) DeleteGmailMessage(id string) error {
	_, err := s.db.Exec("DELETE FROM gmail_messages WHERE id = ?", id)
	return err
}

func (s *GoogleStore) CreateGmailMessageFromForm(userID, from, subject, body string, labels []string) (*GmailMessageView, error) {
	id := fmt.Sprintf("msg_%d", time.Now().UnixNano())
	threadID := fmt.Sprintf("thr_%d", time.Now().UnixNano())

	// Create thread first
	s.db.Exec("INSERT INTO gmail_threads (id, user_id, snippet) VALUES (?, ?, ?)",
		threadID, userID, truncate(body, 100))

	// Build payload using json.Marshal to properly escape special characters
	payloadData := map[string]any{
		"headers": []map[string]string{
			{"name": "From", "value": from},
			{"name": "Subject", "value": subject},
		},
		"body": map[string]string{
			"data": base64.URLEncoding.EncodeToString([]byte(body)),
		},
	}
	payloadBytes, _ := json.Marshal(payloadData)

	labelJSON, _ := json.Marshal(labels)

	_, err := s.db.Exec(
		"INSERT INTO gmail_messages (id, user_id, thread_id, label_ids, snippet, internal_date, payload) VALUES (?, ?, ?, ?, ?, ?, ?)",
		id, userID, threadID, string(labelJSON), truncate(body, 100), time.Now().UnixMilli(), string(payloadBytes),
	)
	if err != nil {
		return nil, err
	}

	return &GmailMessageView{
		ID:       id,
		Subject:  subject,
		Snippet:  truncate(body, 100),
		LabelIDs: labels,
	}, nil
}

// SendGmailMessage creates a sent message and returns it
func (s *GoogleStore) SendGmailMessage(userID, from, to, subject, body string) (*GmailMessage, error) {
	id := fmt.Sprintf("msg_%d", time.Now().UnixNano())
	threadID := fmt.Sprintf("thr_%d", time.Now().UnixNano())

	snippet := truncate(body, 100)

	// Create thread first
	s.db.Exec("INSERT INTO gmail_threads (id, user_id, snippet) VALUES (?, ?, ?)",
		threadID, userID, snippet)

	// Build payload with all headers
	payloadData := map[string]any{
		"headers": []map[string]string{
			{"name": "From", "value": from},
			{"name": "To", "value": to},
			{"name": "Subject", "value": subject},
		},
		"body": map[string]string{
			"data": base64.URLEncoding.EncodeToString([]byte(body)),
		},
	}
	payloadBytes, _ := json.Marshal(payloadData)

	// Messages that are sent get the SENT label
	labels := []string{"SENT"}
	labelJSON, _ := json.Marshal(labels)

	_, err := s.db.Exec(
		"INSERT INTO gmail_messages (id, user_id, thread_id, label_ids, snippet, internal_date, payload) VALUES (?, ?, ?, ?, ?, ?, ?)",
		id, userID, threadID, string(labelJSON), snippet, time.Now().UnixMilli(), string(payloadBytes),
	)
	if err != nil {
		return nil, err
	}

	return &GmailMessage{
		ID:           id,
		UserID:       userID,
		ThreadID:     threadID,
		LabelIDs:     labels,
		Snippet:      snippet,
		InternalDate: time.Now().UnixMilli(),
		Payload:      string(payloadBytes),
	}, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// GetGmailProfile returns the user's Gmail profile with current historyId.
func (s *GoogleStore) GetGmailProfile(userID string) (*GmailProfile, error) {
	var historyID int64
	err := s.db.QueryRow("SELECT COALESCE(history_id, 1) FROM users WHERE id = ?", userID).Scan(&historyID)
	if err != nil {
		historyID = 1
	}

	var msgCount, threadCount int
	s.db.QueryRow("SELECT COUNT(*) FROM gmail_messages WHERE user_id = ?", userID).Scan(&msgCount)
	s.db.QueryRow("SELECT COUNT(*) FROM gmail_threads WHERE user_id = ?", userID).Scan(&threadCount)

	return &GmailProfile{
		EmailAddress:  userID + "@example.com",
		MessagesTotal: msgCount,
		ThreadsTotal:  threadCount,
		HistoryID:     historyID,
	}, nil
}

// ListGmailHistory returns messages added since the given historyId.
func (s *GoogleStore) ListGmailHistory(userID string, startHistoryID int64, maxResults int, pageToken string) ([]GmailHistoryEntry, int64, string, error) {
	offset := 0
	if pageToken != "" {
		decoded, err := base64.StdEncoding.DecodeString(pageToken)
		if err == nil {
			offset, _ = strconv.Atoi(string(decoded))
		}
	}

	// Get current history ID
	var currentHistoryID int64
	s.db.QueryRow("SELECT COALESCE(history_id, 1) FROM users WHERE id = ?", userID).Scan(&currentHistoryID)

	// Get messages with history_id >= startHistoryID
	sqlQuery := `SELECT id, user_id, thread_id, label_ids, snippet, internal_date, payload, COALESCE(history_id, 0)
				 FROM gmail_messages WHERE user_id = ? AND COALESCE(history_id, 0) >= ?
				 ORDER BY history_id ASC LIMIT ? OFFSET ?`

	rows, err := s.db.Query(sqlQuery, userID, startHistoryID, maxResults+1, offset)
	if err != nil {
		return nil, 0, "", err
	}
	defer rows.Close()

	var messages []GmailMessage
	for rows.Next() {
		var m GmailMessage
		var labelJSON string
		var historyID int64
		err := rows.Scan(&m.ID, &m.UserID, &m.ThreadID, &labelJSON, &m.Snippet, &m.InternalDate, &m.Payload, &historyID)
		if err != nil {
			return nil, 0, "", err
		}
		json.Unmarshal([]byte(labelJSON), &m.LabelIDs)
		messages = append(messages, m)
	}

	var nextToken string
	if len(messages) > maxResults {
		messages = messages[:maxResults]
		nextToken = base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset + maxResults)))
	}

	// Group into history entries (one entry per message for simplicity)
	var entries []GmailHistoryEntry
	for i, msg := range messages {
		entries = append(entries, GmailHistoryEntry{
			ID:            startHistoryID + int64(i),
			MessagesAdded: []GmailMessage{msg},
		})
	}

	return entries, currentHistoryID, nextToken, nil
}

// IncrementHistoryID increments the user's history ID and returns the new value.
func (s *GoogleStore) IncrementHistoryID(userID string) (int64, error) {
	_, err := s.db.Exec("UPDATE users SET history_id = COALESCE(history_id, 0) + 1 WHERE id = ?", userID)
	if err != nil {
		return 0, err
	}
	var newID int64
	err = s.db.QueryRow("SELECT history_id FROM users WHERE id = ?", userID).Scan(&newID)
	return newID, err
}

// GetGmailAttachment retrieves an attachment by message and attachment ID.
func (s *GoogleStore) GetGmailAttachment(userID, messageID, attachmentID string) (*GmailAttachment, error) {
	var a GmailAttachment
	err := s.db.QueryRow(
		"SELECT id, message_id, filename, mime_type, size, data FROM gmail_attachments WHERE id = ? AND message_id = ?",
		attachmentID, messageID,
	).Scan(&a.ID, &a.MessageID, &a.Filename, &a.MimeType, &a.Size, &a.Data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("attachment not found")
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// CreateGmailAttachment creates a new attachment.
func (s *GoogleStore) CreateGmailAttachment(a *GmailAttachment) error {
	_, err := s.db.Exec(
		"INSERT INTO gmail_attachments (id, message_id, filename, mime_type, size, data) VALUES (?, ?, ?, ?, ?, ?)",
		a.ID, a.MessageID, a.Filename, a.MimeType, a.Size, a.Data,
	)
	return err
}

// Calendar types and methods

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

func (s *GoogleStore) CreateCalendar(c *Calendar) error {
	_, err := s.db.Exec(
		"INSERT INTO calendars (id, user_id, summary) VALUES (?, ?, ?)",
		c.ID, c.UserID, c.Summary,
	)
	return err
}

func (s *GoogleStore) CreateCalendarEvent(e *CalendarEvent) (*CalendarEvent, error) {
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

func (s *GoogleStore) ListCalendarEvents(calendarID string, maxResults int, pageToken string, timeMin string, timeMax string) ([]CalendarEvent, string, error) {
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

func (s *GoogleStore) GetCalendarEvent(calendarID, eventID string) (*CalendarEvent, error) {
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

func (s *GoogleStore) ListAllCalendarEvents() ([]CalendarEvent, error) {
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

func (s *GoogleStore) CreateCalendarEventFromForm(summary, description, start, end string) (*CalendarEvent, error) {
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
func (s *GoogleStore) UpdateCalendarEvent(e *CalendarEvent) (*CalendarEvent, error) {
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
func (s *GoogleStore) DeleteCalendarEvent(calendarID, eventID string) error {
	_, err := s.db.Exec("DELETE FROM calendar_events WHERE calendar_id = ? AND id = ?", calendarID, eventID)
	return err
}

// GetCalendarSyncToken returns the current sync token for a calendar.
func (s *GoogleStore) GetCalendarSyncToken(calendarID string) (string, error) {
	var token string
	err := s.db.QueryRow("SELECT COALESCE(sync_token, '') FROM calendars WHERE id = ?", calendarID).Scan(&token)
	if err != nil || token == "" {
		// If calendar doesn't exist or no sync token, generate a new token
		token = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	}
	return token, nil
}

// UpdateCalendarSyncToken updates the sync token for a calendar.
func (s *GoogleStore) UpdateCalendarSyncToken(calendarID string) (string, error) {
	token := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	_, err := s.db.Exec("UPDATE calendars SET sync_token = ? WHERE id = ?", token, calendarID)
	if err != nil {
		return "", err
	}
	return token, nil
}

// ListCalendarEventsSince returns events updated since the given sync token.
func (s *GoogleStore) ListCalendarEventsSince(calendarID string, syncToken string, maxResults int) ([]CalendarEvent, string, error) {
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

// People types and methods

type Person struct {
	ResourceName string
	UserID       string
	Data         string
}

type PersonView struct {
	ID           string
	ResourceName string
	DisplayName  string
	Email        string
}

func (s *GoogleStore) CreatePerson(p *Person) error {
	_, err := s.db.Exec(
		"INSERT INTO people (resource_name, user_id, data) VALUES (?, ?, ?)",
		p.ResourceName, p.UserID, p.Data,
	)
	return err
}

func (s *GoogleStore) SearchPeople(userID string, query string, pageSize int, pageToken string) ([]Person, string, error) {
	offset := 0
	if pageToken != "" {
		decoded, err := base64.StdEncoding.DecodeString(pageToken)
		if err == nil {
			offset, _ = strconv.Atoi(string(decoded))
		}
	}

	sqlQuery := "SELECT resource_name, user_id, data FROM people WHERE user_id = ?"
	args := []any{userID}

	if query != "" {
		sqlQuery += " AND data LIKE ? ESCAPE '\\'"
		escaped := strings.ReplaceAll(query, "%", "\\%")
		escaped = strings.ReplaceAll(escaped, "_", "\\_")
		args = append(args, "%"+escaped+"%")
	}

	sqlQuery += " ORDER BY resource_name ASC LIMIT ? OFFSET ?"
	args = append(args, pageSize+1, offset)

	rows, err := s.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var people []Person
	for rows.Next() {
		var p Person
		err := rows.Scan(&p.ResourceName, &p.UserID, &p.Data)
		if err != nil {
			return nil, "", err
		}
		people = append(people, p)
	}

	var nextToken string
	if len(people) > pageSize {
		people = people[:pageSize]
		nextToken = base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset + pageSize)))
	}

	return people, nextToken, nil
}

func (s *GoogleStore) GetPerson(userID, resourceName string) (*Person, error) {
	var p Person
	err := s.db.QueryRow(
		"SELECT resource_name, user_id, data FROM people WHERE user_id = ? AND resource_name = ?",
		userID, resourceName,
	).Scan(&p.ResourceName, &p.UserID, &p.Data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("person not found")
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *GoogleStore) ListAllPeople() ([]PersonView, error) {
	rows, err := s.db.Query("SELECT resource_name, data FROM people ORDER BY resource_name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var people []PersonView
	for rows.Next() {
		var p PersonView
		var data string
		if err := rows.Scan(&p.ResourceName, &data); err != nil {
			return nil, err
		}

		// Extract ID from resource_name (people/c123 -> c123)
		p.ID = strings.TrimPrefix(p.ResourceName, "people/")

		// Parse data JSON
		var d struct {
			Names          []struct{ DisplayName string } `json:"names"`
			EmailAddresses []struct{ Value string }       `json:"emailAddresses"`
		}
		if err := json.Unmarshal([]byte(data), &d); err != nil {
			return nil, err
		}
		if len(d.Names) > 0 {
			p.DisplayName = d.Names[0].DisplayName
		}
		if len(d.EmailAddresses) > 0 {
			p.Email = d.EmailAddresses[0].Value
		}
		people = append(people, p)
	}
	return people, nil
}

func (s *GoogleStore) GetPersonView(userID, resourceName string) (*PersonView, error) {
	p, err := s.GetPerson(userID, resourceName)
	if err != nil {
		return nil, err
	}

	view := &PersonView{
		ID:           strings.TrimPrefix(p.ResourceName, "people/"),
		ResourceName: p.ResourceName,
	}

	var d struct {
		Names          []struct{ DisplayName string } `json:"names"`
		EmailAddresses []struct{ Value string }       `json:"emailAddresses"`
	}
	if err := json.Unmarshal([]byte(p.Data), &d); err != nil {
		return nil, err
	}
	if len(d.Names) > 0 {
		view.DisplayName = d.Names[0].DisplayName
	}
	if len(d.EmailAddresses) > 0 {
		view.Email = d.EmailAddresses[0].Value
	}

	return view, nil
}

func (s *GoogleStore) CreatePersonFromForm(userID, name, email string) (*PersonView, error) {
	id := fmt.Sprintf("c%d", time.Now().UnixNano())
	resourceName := "people/" + id

	// Build data using json.Marshal to properly escape special characters
	personData := map[string]any{
		"names":          []map[string]string{{"displayName": name}},
		"emailAddresses": []map[string]string{{"value": email}},
	}
	dataBytes, _ := json.Marshal(personData)

	_, err := s.db.Exec(
		"INSERT INTO people (resource_name, user_id, data) VALUES (?, ?, ?)",
		resourceName, userID, string(dataBytes),
	)
	if err != nil {
		return nil, err
	}

	return &PersonView{
		ID:           id,
		ResourceName: resourceName,
		DisplayName:  name,
		Email:        email,
	}, nil
}

func (s *GoogleStore) DeletePerson(id string) error {
	resourceName := "people/" + id
	_, err := s.db.Exec("DELETE FROM people WHERE resource_name = ?", resourceName)
	return err
}

// GetPeopleSyncToken returns the current sync token for a user's contacts.
func (s *GoogleStore) GetPeopleSyncToken(userID string) (string, error) {
	var token string
	err := s.db.QueryRow("SELECT token FROM sync_tokens WHERE resource_type = 'people' AND user_id = ?", userID).Scan(&token)
	if err == sql.ErrNoRows {
		// Generate new sync token if none exists
		token = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
		s.db.Exec("INSERT INTO sync_tokens (id, resource_type, user_id, token) VALUES (?, 'people', ?, ?)",
			fmt.Sprintf("sync_people_%s", userID), userID, token)
	} else if err != nil {
		return "", err
	}
	return token, nil
}

// UpdatePeopleSyncToken updates the sync token for a user's contacts.
func (s *GoogleStore) UpdatePeopleSyncToken(userID string) (string, error) {
	token := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	_, err := s.db.Exec("UPDATE sync_tokens SET token = ? WHERE resource_type = 'people' AND user_id = ?", token, userID)
	if err != nil {
		return "", err
	}
	return token, nil
}

// ListPeopleSince returns people updated since the given sync token.
func (s *GoogleStore) ListPeopleSince(userID string, syncToken string, pageSize int) ([]Person, string, error) {
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

	// Use COALESCE to handle NULL updated_at values (treat them as epoch 0)
	sqlQuery := `SELECT resource_name, user_id, data FROM people
		WHERE user_id = ? AND COALESCE(updated_at, '1970-01-01T00:00:00Z') > ? ORDER BY updated_at ASC LIMIT ?`

	rows, err := s.db.Query(sqlQuery, userID, sinceTimestamp, pageSize)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var people []Person
	for rows.Next() {
		var p Person
		if err := rows.Scan(&p.ResourceName, &p.UserID, &p.Data); err != nil {
			return nil, "", err
		}
		people = append(people, p)
	}

	// Generate new sync token based on current time
	newToken := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	return people, newToken, nil
}

// ListPeopleConnections lists people for the connections endpoint with sync token support.
func (s *GoogleStore) ListPeopleConnections(userID string, pageSize int, pageToken string, syncToken string) ([]Person, string, string, error) {
	if syncToken != "" {
		// Incremental sync
		people, newSyncToken, err := s.ListPeopleSince(userID, syncToken, pageSize)
		return people, "", newSyncToken, err
	}

	// Full sync with pagination
	offset := 0
	if pageToken != "" {
		decoded, err := base64.StdEncoding.DecodeString(pageToken)
		if err == nil {
			offset, _ = strconv.Atoi(string(decoded))
		}
	}

	sqlQuery := "SELECT resource_name, user_id, data FROM people WHERE user_id = ? ORDER BY resource_name ASC LIMIT ? OFFSET ?"

	rows, err := s.db.Query(sqlQuery, userID, pageSize+1, offset)
	if err != nil {
		return nil, "", "", err
	}
	defer rows.Close()

	var people []Person
	for rows.Next() {
		var p Person
		if err := rows.Scan(&p.ResourceName, &p.UserID, &p.Data); err != nil {
			return nil, "", "", err
		}
		people = append(people, p)
	}

	var nextPageToken string
	if len(people) > pageSize {
		people = people[:pageSize]
		nextPageToken = base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset + pageSize)))
	}

	// If no more pages, generate sync token
	var nextSyncToken string
	if nextPageToken == "" {
		nextSyncToken, _ = s.GetPeopleSyncToken(userID)
	}

	return people, nextPageToken, nextSyncToken, nil
}

// Tasks types and methods

type TaskList struct {
	ID        string
	UserID    string
	Title     string
	UpdatedAt string
}

type Task struct {
	ID        string
	ListID    string
	Title     string
	Notes     string
	Due       string
	Status    string
	Completed string
	UpdatedAt string
}

// CreateTaskList creates a new task list
func (s *GoogleStore) CreateTaskList(tl *TaskList) error {
	if tl.ID == "" {
		tl.ID = fmt.Sprintf("tasklist_%d", time.Now().UnixNano())
	}
	tl.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	_, err := s.db.Exec(
		"INSERT INTO task_lists (id, user_id, title, updated_at) VALUES (?, ?, ?, ?)",
		tl.ID, tl.UserID, tl.Title, tl.UpdatedAt,
	)
	return err
}

// GetTaskList retrieves a task list by ID
func (s *GoogleStore) GetTaskList(listID string) (*TaskList, error) {
	var tl TaskList
	err := s.db.QueryRow(
		"SELECT id, user_id, title, COALESCE(updated_at, '') FROM task_lists WHERE id = ?",
		listID,
	).Scan(&tl.ID, &tl.UserID, &tl.Title, &tl.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task list not found")
	}
	return &tl, err
}

// CreateTask creates a new task
func (s *GoogleStore) CreateTask(t *Task) (*Task, error) {
	if t.ID == "" {
		t.ID = fmt.Sprintf("task_%d", time.Now().UnixNano())
	}
	t.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	_, err := s.db.Exec(
		`INSERT INTO tasks (id, list_id, title, notes, due, status, completed, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.ListID, t.Title, t.Notes, t.Due, t.Status, t.Completed, t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// GetTask retrieves a task by list ID and task ID
func (s *GoogleStore) GetTask(listID, taskID string) (*Task, error) {
	var t Task
	err := s.db.QueryRow(
		`SELECT id, list_id, title, COALESCE(notes, ''), COALESCE(due, ''), status,
		 COALESCE(completed, ''), COALESCE(updated_at, '') FROM tasks
		 WHERE list_id = ? AND id = ?`,
		listID, taskID,
	).Scan(&t.ID, &t.ListID, &t.Title, &t.Notes, &t.Due, &t.Status, &t.Completed, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("task not found")
	}
	return &t, err
}

// ListTasks lists tasks in a task list
func (s *GoogleStore) ListTasks(listID string, showCompleted bool, maxResults int64) ([]*Task, error) {
	query := `SELECT id, list_id, title, COALESCE(notes, ''), COALESCE(due, ''), status,
			  COALESCE(completed, ''), COALESCE(updated_at, '') FROM tasks
			  WHERE list_id = ?`
	args := []any{listID}

	if !showCompleted {
		query += " AND status != 'completed'"
	}

	query += " ORDER BY updated_at DESC LIMIT ?"
	args = append(args, maxResults)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.ListID, &t.Title, &t.Notes, &t.Due, &t.Status, &t.Completed, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, &t)
	}
	return tasks, nil
}

// UpdateTask updates an existing task
func (s *GoogleStore) UpdateTask(t *Task) (*Task, error) {
	t.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	_, err := s.db.Exec(
		`UPDATE tasks SET title = ?, notes = ?, due = ?, status = ?, completed = ?, updated_at = ?
		 WHERE list_id = ? AND id = ?`,
		t.Title, t.Notes, t.Due, t.Status, t.Completed, t.UpdatedAt, t.ListID, t.ID,
	)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// DeleteTask deletes a task
func (s *GoogleStore) DeleteTask(listID, taskID string) error {
	_, err := s.db.Exec("DELETE FROM tasks WHERE list_id = ? AND id = ?", listID, taskID)
	return err
}

// ListAllTasks lists all tasks for admin UI
func (s *GoogleStore) ListAllTasks() ([]*Task, error) {
	query := `SELECT id, list_id, title, COALESCE(notes, ''), COALESCE(due, ''), status,
			  COALESCE(completed, ''), COALESCE(updated_at, '') FROM tasks
			  ORDER BY updated_at DESC LIMIT 100`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.ListID, &t.Title, &t.Notes, &t.Due, &t.Status, &t.Completed, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, &t)
	}
	return tasks, nil
}

// CreateTaskFromForm creates a task from admin form input
func (s *GoogleStore) CreateTaskFromForm(title, notes, due, status string) (*Task, error) {
	task := &Task{
		ListID: "@default",
		Title:  title,
		Notes:  notes,
		Due:    due,
		Status: status,
	}
	return s.CreateTask(task)
}
