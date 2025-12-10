// ABOUTME: Tests for core SQLite store initialization and schema migrations.
// ABOUTME: Verifies database setup and request_logs table creation.

package store

import (
	"os"
	"testing"
	"time"
)

func TestNewStore_CreatesDatabase(t *testing.T) {
	dbPath := "test_ish.db"
	defer os.Remove(dbPath)

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer s.Close()

	// Verify request_logs table exists
	var name string
	err = s.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='request_logs'").Scan(&name)
	if err != nil {
		t.Errorf("request_logs table not found: %v", err)
	}
}

func TestStore_GetDB(t *testing.T) {
	dbPath := "test_ish.db"
	defer os.Remove(dbPath)

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer s.Close()

	// Verify GetDB returns a valid database connection
	db := s.GetDB()
	if db == nil {
		t.Error("GetDB() returned nil")
	}

	// Verify we can query the database
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM request_logs").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 request logs, got %d", count)
	}
}

func TestMigrationVersionTracking(t *testing.T) {
	s := setupTestDB(t)
	defer s.Close()

	// Verify schema_migrations table exists
	var tableName string
	err := s.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&tableName)
	if err != nil {
		t.Fatalf("schema_migrations table not found: %v", err)
	}

	// Verify current migration version is V2
	version, err := s.getCurrentMigrationVersion()
	if err != nil {
		t.Fatalf("Failed to get current migration version: %v", err)
	}
	if version != CurrentSchemaVersion {
		t.Errorf("Expected current schema version %d, got %d", CurrentSchemaVersion, version)
	}

	// Verify migration records exist
	var count int
	err = s.db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query schema_migrations: %v", err)
	}
	if count == 0 {
		t.Error("Expected migration records in schema_migrations table")
	}
}

func TestMigrationRecords(t *testing.T) {
	s := setupTestDB(t)
	defer s.Close()

	// Query migration records
	rows, err := s.db.Query("SELECT version, description FROM schema_migrations ORDER BY version")
	if err != nil {
		t.Fatalf("Failed to query schema_migrations: %v", err)
	}
	defer rows.Close()

	var versions []int
	var descriptions []string

	for rows.Next() {
		var version int
		var description string
		if err := rows.Scan(&version, &description); err != nil {
			t.Fatalf("Failed to scan migration record: %v", err)
		}
		versions = append(versions, version)
		descriptions = append(descriptions, description)
	}

	// Verify V1 exists
	if len(versions) == 0 {
		t.Error("Expected at least one migration record")
		return
	}

	if versions[0] != MigrationV1 {
		t.Errorf("Expected first migration to be V%d, got V%d", MigrationV1, versions[0])
	}

	// Verify V2 exists
	if len(versions) < 2 {
		t.Errorf("Expected at least 2 migration records, got %d", len(versions))
		return
	}

	if versions[1] != MigrationV2 {
		t.Errorf("Expected second migration to be V%d, got V%d", MigrationV2, versions[1])
	}
}

func TestParseTimestamp_MultipleFormats(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "SQLite default format without milliseconds",
			input:   "2024-12-09 14:30:45",
			wantErr: false,
		},
		{
			name:    "SQLite format with milliseconds",
			input:   "2024-12-09 14:30:45.123",
			wantErr: false,
		},
		{
			name:    "RFC3339 format",
			input:   "2024-12-09T14:30:45Z",
			wantErr: false,
		},
		{
			name:    "RFC3339 with timezone offset",
			input:   "2024-12-09T14:30:45-05:00",
			wantErr: false,
		},
		{
			name:    "RFC3339 with nanoseconds",
			input:   "2024-12-09T14:30:45.123456789Z",
			wantErr: false,
		},
		{
			name:    "Invalid format",
			input:   "invalid-timestamp",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTimestamp(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseTimestamp(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.IsZero() {
				t.Errorf("parseTimestamp(%q) returned zero time", tt.input)
			}
		})
	}
}

func TestParseTimestamp_TimezoneAwareness(t *testing.T) {
	// Test that parsing handles timezone-aware timestamps correctly
	tests := []struct {
		name        string
		input       string
		shouldBeUTC bool
	}{
		{
			name:        "UTC timestamp",
			input:       "2024-12-09T14:30:45Z",
			shouldBeUTC: true,
		},
		{
			name:        "Non-UTC timezone",
			input:       "2024-12-09T14:30:45-05:00",
			shouldBeUTC: false, // Should preserve timezone info
		},
		{
			name:        "Default SQLite format (no tz info, defaults to UTC)",
			input:       "2024-12-09 14:30:45",
			shouldBeUTC: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTimestamp(tt.input)
			if err != nil {
				t.Fatalf("parseTimestamp(%q) failed: %v", tt.input, err)
			}

			if tt.shouldBeUTC {
				if got.Location() != time.UTC {
					t.Errorf("Expected UTC location, got %v", got.Location())
				}
			} else {
				// Non-UTC timestamps should preserve their timezone info
				if got.Location().String() == "UTC" {
					t.Errorf("Expected non-UTC location, got UTC")
				}
			}
		})
	}
}

func TestRequestLogRoundtrip_TimezonePreservation(t *testing.T) {
	s := setupTestDB(t)
	defer s.Close()

	// Create a timestamp with specific timezone
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("Failed to load timezone: %v", err)
	}

	now := time.Now().In(loc)

	// Insert a request log
	log := &RequestLog{
		Timestamp:  now,
		PluginName: "test",
		Method:     "GET",
		Path:       "/test",
		StatusCode: 200,
		DurationMs: 100,
	}

	if err := s.LogRequest(log); err != nil {
		t.Fatalf("LogRequest failed: %v", err)
	}

	// Retrieve the log and verify timestamp
	retrieved, err := s.GetRecentRequests("test", 1)
	if err != nil {
		t.Fatalf("GetRecentRequests failed: %v", err)
	}

	if len(retrieved) == 0 {
		t.Fatal("No logs retrieved")
	}

	// The retrieved timestamp should be the same moment in time
	retrievedTime := retrieved[0].Timestamp
	if !now.Equal(retrievedTime) && now.Unix() != retrievedTime.Unix() {
		// Allow for small time differences due to database storage precision
		timeDiff := now.Unix() - retrievedTime.Unix()
		if timeDiff < -1 || timeDiff > 1 {
			t.Errorf("Timestamp mismatch: expected %v, got %v (diff: %d seconds)", now, retrievedTime, timeDiff)
		}
	}
}
