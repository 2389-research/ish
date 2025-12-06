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
	// Tables will be migrated from internal/store in next tasks
	return nil
}
