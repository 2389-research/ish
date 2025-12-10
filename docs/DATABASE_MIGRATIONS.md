# Database Migrations Guide

This document describes the database migration system used in the ISH project.

## Overview

The ISH database uses a versioned migration system to track schema changes. All migrations are applied automatically when the application starts, ensuring the database is always in sync with the code.

### Migration Components

1. **schema_migrations Table**: Tracks which migrations have been applied
2. **Migration Constants**: Define migration versions in the code
3. **Migration Functions**: Implement the actual schema changes
4. **Version Tracking**: Ensures idempotent migration application

## Current Schema Versions

### Migration V1: Initial Schema (Baseline)
- Creates `request_logs` table with all fields for HTTP request logging
- Creates basic single-column indexes for common query patterns:
  - `idx_request_logs_timestamp`: Descending timestamp for ordering
  - `idx_request_logs_path`: Path filtering for endpoint analysis
  - `idx_request_logs_status`: Status code filtering for error tracking
  - `idx_request_logs_plugin`: Plugin name filtering for plugin-specific queries

### Migration V2: Performance Indexes (Current)
- Adds composite indexes optimized for common query patterns
- `idx_request_logs_path_count`: Optimizes `GetTopEndpoints` aggregation
- `idx_request_logs_plugin_timestamp`: Optimizes plugin metrics (count, error rate)
- `idx_request_logs_plugin_method_status`: Optimizes multi-column filtering
- `idx_request_logs_user_id`: Partial index for user ID filtering (non-empty only)
- `idx_request_logs_timestamp_status`: Date-based and status code filtering

## Migration Process

### Automatic Migration

When the application starts:

1. `New()` function initializes the database connection
2. Calls `migrate()` to run pending migrations
3. `migrate()` checks the current version in `schema_migrations`
4. Runs each migration function in order if version is behind target
5. Records successful migrations in the table

### Migration Execution Flow

```
New(dbPath)
  └─> db.Exec(pragmas)
  └─> migrate()
      ├─> createMigrationsTable()  // Creates tracking table if needed
      ├─> getCurrentMigrationVersion()  // Checks current version
      ├─> for each pending migration:
      │   └─> migrateV*()  // Apply migration
      │       └─> recordMigration()  // Record in table
      └─> return
```

## Time Zone Handling

The migration system works in conjunction with timezone-aware timestamp parsing.

### Timestamp Storage

SQLite stores timestamps as TEXT using the format: `YYYY-MM-DD HH:MM:SS.fff`

### Timestamp Parsing

The `parseTimestamp()` function supports multiple formats:
- SQLite default: `2006-01-02 15:04:05` or with milliseconds `2006-01-02 15:04:05.000`
- RFC3339: `2006-01-02T15:04:05Z` or with timezone offset `2006-01-02T15:04:05-05:00`
- RFC3339 with nanoseconds: `2006-01-02T15:04:05.999999999Z`

### Timezone Behavior

- **RFC3339 with timezone**: Preserves the timezone offset from the format
- **RFC3339 UTC**: Parses as UTC
- **Default SQLite format**: Assumes UTC (no timezone in string)
- **Local time detection**: If `time.Local` is detected, converts to UTC

## Adding New Migrations

### Step 1: Create Migration Function

Add a new constant and migration function to `store.go`:

```go
const (
    MigrationV1 = 1  // Initial schema
    MigrationV2 = 2  // Performance indexes
    MigrationV3 = 3  // Your new migration
)

const CurrentSchemaVersion = MigrationV3

func (s *Store) migrateV3() error {
    // Your migration SQL
    _, err := s.db.Exec(`
        ALTER TABLE request_logs ADD COLUMN new_column TEXT;
    `)
    if err != nil {
        return err
    }

    if err := s.recordMigration(MigrationV3, "Add new_column to request_logs"); err != nil {
        return err
    }

    log.Printf("Applied migration v%d: Add new_column to request_logs", MigrationV3)
    return nil
}
```

### Step 2: Add to Migration Chain

Update the `migrate()` function:

```go
func (s *Store) migrate() error {
    // ... existing code ...

    if currentVersion < MigrationV3 {
        if err := s.migrateV3(); err != nil {
            return fmt.Errorf("migration v3 failed: %w", err)
        }
    }

    return nil
}
```

### Step 3: Test the Migration

```go
func TestMigrationV3(t *testing.T) {
    s := setupTestDB(t)
    defer s.Close()

    // Verify the new column exists
    var columnName string
    err := s.db.QueryRow(`
        SELECT name FROM pragma_table_info('request_logs')
        WHERE name = 'new_column'
    `).Scan(&columnName)

    if err != nil {
        t.Errorf("new_column not found: %v", err)
    }
}
```

## Backwards Compatibility

### Key Principles

1. **Additive Only**: Migrations only add tables, columns, or indexes. Never remove or rename existing structures without a deprecation period.

2. **Default Values**: When adding new columns, provide sensible defaults to avoid breaking existing code:
   ```go
   ALTER TABLE request_logs ADD COLUMN new_field TEXT DEFAULT '';
   ```

3. **Index Safety**: Composite indexes are transparent to queries; they only improve performance.

4. **Version Idempotency**: Each migration can be run multiple times safely:
   - Use `CREATE TABLE IF NOT EXISTS`
   - Use `CREATE INDEX IF NOT EXISTS`
   - Record version only after successful completion

### Migration Verification

Query the migrations table to verify state:

```sql
-- View all applied migrations
SELECT version, applied_at, description FROM schema_migrations ORDER BY version;

-- Check current schema version
SELECT MAX(version) as current_version FROM schema_migrations;

-- View migration history
SELECT * FROM schema_migrations WHERE applied_at > datetime('now', '-1 day');
```

## Troubleshooting

### Migration Failed to Apply

If a migration fails during application startup:

1. Check the error log for the specific failure message
2. Verify the database file isn't corrupted: `sqlite3 ish.db "PRAGMA integrity_check;"`
3. Check if the migration was partially applied: `SELECT * FROM schema_migrations;`
4. If partially applied, manually fix or remove the bad record

### Rollback (Manual Process)

SQLite doesn't support transactions for DDL in the same way as other databases. If a migration needs to be undone:

1. Stop the application
2. Manually reverse the migration SQL
3. Delete the migration record: `DELETE FROM schema_migrations WHERE version = X;`
4. Restart the application

### Downgrading Schema

To downgrade to a previous version:

1. Manually reverse migrations in reverse order
2. Delete records from `schema_migrations` for versions being removed
3. Update `CurrentSchemaVersion` constant in code
4. Rebuild and restart application

## Performance Considerations

### Index Strategy

- **V1 Basic Indexes**: Good for single-column filtering
- **V2 Composite Indexes**: Optimized for common query patterns in metrics and analytics
- **Partial Indexes**: `idx_request_logs_user_id` uses `WHERE user_id != ''` to keep index smaller

### Query Optimization

The V2 indexes are designed to support:
- Aggregations: `GetTopEndpoints()` with `GROUP BY path`
- Time-range queries: `GetPluginRequestCount()` with `timestamp >= ?`
- Multi-column filtering: Combined plugin, method, and status filters
- Statistical queries: Error rate and duration calculations

## Related Code

- `internal/store/store.go`: Migration definitions and execution
- `internal/store/request_logs.go`: Timestamp parsing and log operations
- `internal/store/store_test.go`: Migration and timestamp tests
- `internal/store/request_logs_test.go`: Request log operation tests
