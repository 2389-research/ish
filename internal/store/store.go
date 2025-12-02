// ABOUTME: SQLite store for fake Google API data.
// ABOUTME: Handles database initialization, migrations, and connection management.

package store

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS gmail_labels (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id),
		name TEXT NOT NULL,
		type TEXT DEFAULT 'user'
	);

	CREATE TABLE IF NOT EXISTS gmail_threads (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id),
		snippet TEXT
	);

	CREATE TABLE IF NOT EXISTS gmail_messages (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id),
		thread_id TEXT REFERENCES gmail_threads(id),
		label_ids TEXT,
		snippet TEXT,
		internal_date INTEGER,
		payload TEXT
	);

	CREATE TABLE IF NOT EXISTS calendars (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id),
		summary TEXT
	);

	CREATE TABLE IF NOT EXISTS calendar_events (
		id TEXT PRIMARY KEY,
		calendar_id TEXT REFERENCES calendars(id),
		summary TEXT,
		description TEXT,
		start_time TEXT,
		end_time TEXT,
		attendees TEXT
	);

	CREATE TABLE IF NOT EXISTS people (
		resource_name TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id),
		data TEXT
	);
	`
	_, err := s.db.Exec(schema)
	return err
}
