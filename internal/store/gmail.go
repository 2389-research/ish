// ABOUTME: Gmail-related store operations for messages and threads.
// ABOUTME: Handles CRUD and listing with pagination support.

package store

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
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

	if query != "" {
		sqlQuery += " AND snippet LIKE ?"
		args = append(args, "%"+query+"%")
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
