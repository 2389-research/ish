// ABOUTME: Core SQLite store for ISH server.
// ABOUTME: Handles database initialization, migrations, and connection management for request logging.

package store

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

// Migration version constants
const (
	MigrationV1 = 1 // Initial schema with request_logs table
	MigrationV2 = 2 // Add performance indexes for aggregation and filtering queries
)

// CurrentSchemaVersion is the target version for the database schema
const CurrentSchemaVersion = MigrationV2

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Verify connection works
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pooling
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(0) // Connections don't expire

	// Enable foreign keys and WAL mode
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA busy_timeout = 5000",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, err
		}
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

// migrate runs all pending migrations
func (s *Store) migrate() error {
	// Create schema_migrations table if it doesn't exist
	if err := s.createMigrationsTable(); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current schema version
	currentVersion, err := s.getCurrentMigrationVersion()
	if err != nil {
		return fmt.Errorf("failed to get current migration version: %w", err)
	}

	log.Printf("Database schema version: %d, target version: %d", currentVersion, CurrentSchemaVersion)

	// Run migrations in order
	if currentVersion < MigrationV1 {
		if err := s.migrateV1(); err != nil {
			return fmt.Errorf("migration v1 failed: %w", err)
		}
	}

	if currentVersion < MigrationV2 {
		if err := s.migrateV2(); err != nil {
			return fmt.Errorf("migration v2 failed: %w", err)
		}
	}

	return nil
}

// createMigrationsTable creates the schema_migrations tracking table
func (s *Store) createMigrationsTable() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			description TEXT
		)
	`)
	return err
}

// getCurrentMigrationVersion retrieves the current schema version
func (s *Store) getCurrentMigrationVersion() (int, error) {
	var version int
	err := s.db.QueryRow(`
		SELECT COALESCE(MAX(version), 0) FROM schema_migrations
	`).Scan(&version)
	if err != nil {
		return 0, err
	}
	return version, nil
}

// recordMigration records a completed migration
func (s *Store) recordMigration(version int, description string) error {
	_, err := s.db.Exec(`
		INSERT INTO schema_migrations (version, description)
		VALUES (?, ?)
	`, version, description)
	return err
}

// migrateV1 creates the initial request_logs table and indexes
func (s *Store) migrateV1() error {
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
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	if err := s.recordMigration(MigrationV1, "Create request_logs table and indexes"); err != nil {
		return err
	}

	log.Printf("Applied migration v%d: Create request_logs table and indexes", MigrationV1)
	return nil
}

// migrateV2 adds performance-optimized composite indexes for aggregation and filtering queries
func (s *Store) migrateV2() error {
	indexes := []string{
		// Composite index for GetTopEndpoints query (GROUP BY path, ORDER BY COUNT)
		// Optimizes aggregation queries by allowing SQLite to use index to count occurrences per path
		"CREATE INDEX IF NOT EXISTS idx_request_logs_path_count ON request_logs(path, status_code)",

		// Composite index for GetPluginRequestCount and GetPluginErrorRate queries
		// Optimizes filtering by plugin_name AND timestamp range queries used in metrics
		"CREATE INDEX IF NOT EXISTS idx_request_logs_plugin_timestamp ON request_logs(plugin_name, timestamp DESC)",

		// Composite index for GetRequestLogs multi-column filtering
		// Optimizes queries filtering by plugin_name AND method AND status_code together
		"CREATE INDEX IF NOT EXISTS idx_request_logs_plugin_method_status ON request_logs(plugin_name, method, status_code)",

		// Index for user_id filtering and uniqueness checks in GetRequestLogStats
		// WHERE clause filters to non-empty user IDs to keep index smaller
		"CREATE INDEX IF NOT EXISTS idx_request_logs_user_id ON request_logs(user_id) WHERE user_id != ''",

		// Index for date-based filtering in GetRequestLogStats
		// Supports timestamp-based queries with status code filtering
		"CREATE INDEX IF NOT EXISTS idx_request_logs_timestamp_status ON request_logs(timestamp DESC, status_code)",
	}

	for _, indexSQL := range indexes {
		if _, err := s.db.Exec(indexSQL); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	if err := s.recordMigration(MigrationV2, "Add composite indexes for aggregation and filtering queries"); err != nil {
		return err
	}

	log.Printf("Applied migration v%d: Add composite indexes for query optimization", MigrationV2)
	return nil
}
