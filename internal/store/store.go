// ABOUTME: Core SQLite store for ISH server.
// ABOUTME: Handles database initialization, migrations, and connection management for request logging.

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

// GetDB returns the underlying database connection for plugins
func (s *Store) GetDB() *sql.DB {
	return s.db
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS request_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		plugin_name TEXT DEFAULT '',
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
	CREATE INDEX IF NOT EXISTS idx_request_logs_plugin ON request_logs(plugin_name);
	`
	_, err := s.db.Exec(schema)
	return err
}
