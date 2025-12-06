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
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		history_id INTEGER DEFAULT 1
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
		payload TEXT,
		history_id INTEGER DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS gmail_attachments (
		id TEXT PRIMARY KEY,
		message_id TEXT NOT NULL REFERENCES gmail_messages(id),
		filename TEXT,
		mime_type TEXT,
		size INTEGER,
		data TEXT
	);

	CREATE TABLE IF NOT EXISTS calendars (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id),
		summary TEXT,
		sync_token TEXT
	);

	CREATE TABLE IF NOT EXISTS calendar_events (
		id TEXT PRIMARY KEY,
		calendar_id TEXT REFERENCES calendars(id),
		summary TEXT,
		description TEXT,
		start_time TEXT,
		end_time TEXT,
		attendees TEXT,
		location TEXT,
		organizer_email TEXT,
		organizer_name TEXT,
		recurrence TEXT,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS people (
		resource_name TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id),
		data TEXT,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS sync_tokens (
		id TEXT PRIMARY KEY,
		resource_type TEXT NOT NULL,
		user_id TEXT NOT NULL,
		token TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS task_lists (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL REFERENCES users(id),
		title TEXT NOT NULL,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS tasks (
		id TEXT PRIMARY KEY,
		list_id TEXT NOT NULL REFERENCES task_lists(id),
		title TEXT NOT NULL,
		notes TEXT,
		due TEXT,
		status TEXT DEFAULT 'needsAction',
		completed TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS request_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		method TEXT NOT NULL,
		path TEXT NOT NULL,
		status_code INTEGER,
		duration_ms INTEGER,
		user_id TEXT,
		ip_address TEXT,
		user_agent TEXT,
		request_body TEXT,
		response_body TEXT,
		error TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_request_logs_timestamp ON request_logs(timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_request_logs_path ON request_logs(path);
	CREATE INDEX IF NOT EXISTS idx_request_logs_status ON request_logs(status_code);
	`
	_, err := s.db.Exec(schema)
	if err != nil {
		return err
	}

	// Add columns to existing tables if they don't exist (for migrations)
	migrations := []string{
		"ALTER TABLE users ADD COLUMN history_id INTEGER DEFAULT 1",
		"ALTER TABLE gmail_messages ADD COLUMN history_id INTEGER DEFAULT 0",
		"ALTER TABLE calendars ADD COLUMN sync_token TEXT",
		"ALTER TABLE calendar_events ADD COLUMN location TEXT",
		"ALTER TABLE calendar_events ADD COLUMN organizer_email TEXT",
		"ALTER TABLE calendar_events ADD COLUMN organizer_name TEXT",
		"ALTER TABLE calendar_events ADD COLUMN recurrence TEXT",
		"ALTER TABLE calendar_events ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP",
		"ALTER TABLE people ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP",
		"ALTER TABLE request_logs ADD COLUMN request_body TEXT",
		"ALTER TABLE request_logs ADD COLUMN response_body TEXT",
		"ALTER TABLE request_logs ADD COLUMN plugin_name TEXT DEFAULT ''",
	}
	for _, m := range migrations {
		s.db.Exec(m) // Ignore errors for already existing columns
	}

	// Create oauth_tokens table
	oauthSchema := `
	CREATE TABLE IF NOT EXISTS oauth_tokens (
		token TEXT PRIMARY KEY,
		plugin_name TEXT NOT NULL,
		user_id TEXT,
		scopes TEXT,
		expires_at TIMESTAMP,
		refresh_token TEXT,
		revoked BOOLEAN DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_oauth_tokens_plugin ON oauth_tokens(plugin_name);
	CREATE INDEX IF NOT EXISTS idx_oauth_tokens_user ON oauth_tokens(user_id);
	`
	_, err = s.db.Exec(oauthSchema)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) CreateUser(id string) error {
	_, err := s.db.Exec("INSERT OR IGNORE INTO users (id) VALUES (?)", id)
	return err
}

func (s *Store) UserExists(id string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users WHERE id = ?", id).Scan(&count)
	return count > 0, err
}

type Counts struct {
	Messages int
	Threads  int
	Events   int
	People   int
	Tasks    int
	Requests int
}

func (s *Store) GetCounts() (*Counts, error) {
	var c Counts
	s.db.QueryRow("SELECT COUNT(*) FROM gmail_messages").Scan(&c.Messages)
	s.db.QueryRow("SELECT COUNT(*) FROM gmail_threads").Scan(&c.Threads)
	s.db.QueryRow("SELECT COUNT(*) FROM calendar_events").Scan(&c.Events)
	s.db.QueryRow("SELECT COUNT(*) FROM people").Scan(&c.People)
	s.db.QueryRow("SELECT COUNT(*) FROM tasks").Scan(&c.Tasks)
	s.db.QueryRow("SELECT COUNT(*) FROM request_logs").Scan(&c.Requests)
	return &c, nil
}
