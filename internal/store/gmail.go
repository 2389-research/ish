// ABOUTME: Gmail-related store operations for messages and threads.
// ABOUTME: Handles CRUD and listing with pagination support.

package store

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

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

func (s *Store) CreateGmailThread(t *GmailThread) error {
	_, err := s.db.Exec(
		"INSERT INTO gmail_threads (id, user_id, snippet) VALUES (?, ?, ?)",
		t.ID, t.UserID, t.Snippet,
	)
	return err
}

func (s *Store) CreateGmailMessage(m *GmailMessage) error {
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

func (s *Store) ListGmailMessages(userID string, maxResults int, pageToken string, query string) ([]GmailMessage, string, error) {
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

func (s *Store) GetGmailMessage(userID, messageID string) (*GmailMessage, error) {
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

func (s *Store) GetGmailMessageDetail(userID, messageID string) (*GmailMessageDetail, error) {
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

func (s *Store) ListAllGmailMessages() ([]GmailMessageView, error) {
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

func (s *Store) DeleteGmailMessage(id string) error {
	_, err := s.db.Exec("DELETE FROM gmail_messages WHERE id = ?", id)
	return err
}

func (s *Store) CreateGmailMessageFromForm(userID, from, subject, body string, labels []string) (*GmailMessageView, error) {
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
func (s *Store) SendGmailMessage(userID, from, to, subject, body string) (*GmailMessage, error) {
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

// GmailProfile represents a user's Gmail profile.
type GmailProfile struct {
	EmailAddress  string
	MessagesTotal int
	ThreadsTotal  int
	HistoryID     int64
}

// GetGmailProfile returns the user's Gmail profile with current historyId.
func (s *Store) GetGmailProfile(userID string) (*GmailProfile, error) {
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

// GmailHistoryEntry represents a history entry for incremental sync.
type GmailHistoryEntry struct {
	ID            int64
	MessagesAdded []GmailMessage
}

// ListGmailHistory returns messages added since the given historyId.
func (s *Store) ListGmailHistory(userID string, startHistoryID int64, maxResults int, pageToken string) ([]GmailHistoryEntry, int64, string, error) {
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
func (s *Store) IncrementHistoryID(userID string) (int64, error) {
	_, err := s.db.Exec("UPDATE users SET history_id = COALESCE(history_id, 0) + 1 WHERE id = ?", userID)
	if err != nil {
		return 0, err
	}
	var newID int64
	err = s.db.QueryRow("SELECT history_id FROM users WHERE id = ?", userID).Scan(&newID)
	return newID, err
}

// GmailAttachment represents an email attachment.
type GmailAttachment struct {
	ID        string
	MessageID string
	Filename  string
	MimeType  string
	Size      int
	Data      string
}

// GetGmailAttachment retrieves an attachment by message and attachment ID.
func (s *Store) GetGmailAttachment(userID, messageID, attachmentID string) (*GmailAttachment, error) {
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
func (s *Store) CreateGmailAttachment(a *GmailAttachment) error {
	_, err := s.db.Exec(
		"INSERT INTO gmail_attachments (id, message_id, filename, mime_type, size, data) VALUES (?, ?, ?, ?, ?, ?)",
		a.ID, a.MessageID, a.Filename, a.MimeType, a.Size, a.Data,
	)
	return err
}
