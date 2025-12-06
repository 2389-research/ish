// ABOUTME: Database layer for Google plugin (Gmail, Calendar, People, Tasks)
// ABOUTME: Owns all Google-related tables and queries

package google

import (
	"database/sql"
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
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}
