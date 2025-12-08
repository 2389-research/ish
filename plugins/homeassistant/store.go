// ABOUTME: Home Assistant plugin database store layer
// ABOUTME: Manages instances, entities, states, and service calls in SQLite
package homeassistant

import (
	"database/sql"
	"fmt"
	"time"
)

// Store handles Home Assistant data persistence
type Store struct {
	db *sql.DB
}

// NewStore creates a new Home Assistant store
func NewStore(db *sql.DB) (*Store, error) {
	store := &Store{db: db}
	if err := store.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}
	return store, nil
}

// Instance represents a Home Assistant instance
type Instance struct {
	ID        int64     `json:"id"`
	URL       string    `json:"url"`
	Token     string    `json:"token"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Entity represents a Home Assistant entity
type Entity struct {
	ID         int64     `json:"id"`
	InstanceID int64     `json:"instance_id"`
	EntityID   string    `json:"entity_id"`
	FriendlyName string  `json:"friendly_name"`
	Domain     string    `json:"domain"`
	Platform   string    `json:"platform"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// State represents the current state of an entity
type State struct {
	ID         int64     `json:"id"`
	InstanceID int64     `json:"instance_id"`
	EntityID   string    `json:"entity_id"`
	State      string    `json:"state"`
	Attributes string    `json:"attributes"` // JSON blob
	LastChanged time.Time `json:"last_changed"`
	LastUpdated time.Time `json:"last_updated"`
	CreatedAt  time.Time `json:"created_at"`
}

// ServiceCall represents a call to a Home Assistant service
type ServiceCall struct {
	ID         int64     `json:"id"`
	InstanceID int64     `json:"instance_id"`
	Domain     string    `json:"domain"`
	Service    string    `json:"service"`
	ServiceData string   `json:"service_data"` // JSON blob
	EntityID   string    `json:"entity_id"`
	Status     string    `json:"status"` // success, failed, pending
	CalledAt   time.Time `json:"called_at"`
	CreatedAt  time.Time `json:"created_at"`
}

func (s *Store) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS homeassistant_instances (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		url TEXT NOT NULL,
		token TEXT NOT NULL,
		name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS homeassistant_entities (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		instance_id INTEGER NOT NULL,
		entity_id TEXT NOT NULL,
		friendly_name TEXT,
		domain TEXT NOT NULL,
		platform TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (instance_id) REFERENCES homeassistant_instances(id),
		UNIQUE(instance_id, entity_id)
	);

	CREATE TABLE IF NOT EXISTS homeassistant_states (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		instance_id INTEGER NOT NULL,
		entity_id TEXT NOT NULL,
		state TEXT NOT NULL,
		attributes TEXT, -- JSON
		last_changed DATETIME NOT NULL,
		last_updated DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (instance_id) REFERENCES homeassistant_instances(id)
	);

	CREATE TABLE IF NOT EXISTS homeassistant_service_calls (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		instance_id INTEGER NOT NULL,
		domain TEXT NOT NULL,
		service TEXT NOT NULL,
		service_data TEXT, -- JSON
		entity_id TEXT,
		status TEXT NOT NULL DEFAULT 'pending',
		called_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (instance_id) REFERENCES homeassistant_instances(id)
	);

	CREATE INDEX IF NOT EXISTS idx_entities_instance ON homeassistant_entities(instance_id);
	CREATE INDEX IF NOT EXISTS idx_states_instance ON homeassistant_states(instance_id);
	CREATE INDEX IF NOT EXISTS idx_states_entity ON homeassistant_states(entity_id);
	CREATE INDEX IF NOT EXISTS idx_service_calls_instance ON homeassistant_service_calls(instance_id);
	`

	_, err := s.db.Exec(schema)
	return err
}

// CreateInstance creates a new Home Assistant instance
func (s *Store) CreateInstance(url, token, name string) (*Instance, error) {
	now := time.Now()
	result, err := s.db.Exec(`
		INSERT INTO homeassistant_instances (url, token, name, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, url, token, name, now, now)
	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &Instance{
		ID:        id,
		URL:       url,
		Token:     token,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// GetOrCreateInstance gets or creates an instance by URL
func (s *Store) GetOrCreateInstance(url, token, name string) (*Instance, error) {
	var instance Instance
	err := s.db.QueryRow(`
		SELECT id, url, token, name, created_at, updated_at
		FROM homeassistant_instances
		WHERE url = ?
	`, url).Scan(&instance.ID, &instance.URL, &instance.Token, &instance.Name, &instance.CreatedAt, &instance.UpdatedAt)

	if err == sql.ErrNoRows {
		return s.CreateInstance(url, token, name)
	}
	if err != nil {
		return nil, err
	}

	return &instance, nil
}

// CreateOrUpdateEntity creates or updates an entity
func (s *Store) CreateOrUpdateEntity(instanceID int64, entityID, friendlyName, domain, platform string) error {
	now := time.Now()
	_, err := s.db.Exec(`
		INSERT INTO homeassistant_entities (instance_id, entity_id, friendly_name, domain, platform, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(instance_id, entity_id) DO UPDATE SET
			friendly_name = excluded.friendly_name,
			domain = excluded.domain,
			platform = excluded.platform,
			updated_at = excluded.updated_at
	`, instanceID, entityID, friendlyName, domain, platform, now, now)
	return err
}

// RecordState records a state for an entity
func (s *Store) RecordState(instanceID int64, entityID, state, attributes string, lastChanged, lastUpdated time.Time) error {
	_, err := s.db.Exec(`
		INSERT INTO homeassistant_states (instance_id, entity_id, state, attributes, last_changed, last_updated, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, instanceID, entityID, state, attributes, lastChanged, lastUpdated, time.Now())
	return err
}

// RecordServiceCall records a service call
func (s *Store) RecordServiceCall(instanceID int64, domain, service, serviceData, entityID, status string, calledAt time.Time) error {
	_, err := s.db.Exec(`
		INSERT INTO homeassistant_service_calls (instance_id, domain, service, service_data, entity_id, status, called_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, instanceID, domain, service, serviceData, entityID, status, calledAt, time.Now())
	return err
}

// ListAllInstances retrieves all instances for admin view
func (s *Store) ListAllInstances(limit, offset int) ([]Instance, error) {
	rows, err := s.db.Query(`
		SELECT id, url, token, name, created_at, updated_at
		FROM homeassistant_instances
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instances []Instance
	for rows.Next() {
		var inst Instance
		err := rows.Scan(&inst.ID, &inst.URL, &inst.Token, &inst.Name, &inst.CreatedAt, &inst.UpdatedAt)
		if err != nil {
			return nil, err
		}
		instances = append(instances, inst)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return instances, nil
}

// ListAllEntities retrieves all entities for admin view
func (s *Store) ListAllEntities(limit, offset int) ([]Entity, error) {
	rows, err := s.db.Query(`
		SELECT id, instance_id, entity_id, friendly_name, domain, platform, created_at, updated_at
		FROM homeassistant_entities
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entities []Entity
	for rows.Next() {
		var ent Entity
		var friendlyName, platform sql.NullString
		err := rows.Scan(&ent.ID, &ent.InstanceID, &ent.EntityID, &friendlyName, &ent.Domain, &platform, &ent.CreatedAt, &ent.UpdatedAt)
		if err != nil {
			return nil, err
		}
		if friendlyName.Valid {
			ent.FriendlyName = friendlyName.String
		}
		if platform.Valid {
			ent.Platform = platform.String
		}
		entities = append(entities, ent)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return entities, nil
}

// ListAllStates retrieves all states for admin view
func (s *Store) ListAllStates(limit, offset int) ([]State, error) {
	rows, err := s.db.Query(`
		SELECT id, instance_id, entity_id, state, attributes, last_changed, last_updated, created_at
		FROM homeassistant_states
		ORDER BY last_updated DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var states []State
	for rows.Next() {
		var st State
		var attributes sql.NullString
		err := rows.Scan(&st.ID, &st.InstanceID, &st.EntityID, &st.State, &attributes, &st.LastChanged, &st.LastUpdated, &st.CreatedAt)
		if err != nil {
			return nil, err
		}
		if attributes.Valid {
			st.Attributes = attributes.String
		}
		states = append(states, st)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return states, nil
}

// ListAllServiceCalls retrieves all service calls for admin view
func (s *Store) ListAllServiceCalls(limit, offset int) ([]ServiceCall, error) {
	rows, err := s.db.Query(`
		SELECT id, instance_id, domain, service, service_data, entity_id, status, called_at, created_at
		FROM homeassistant_service_calls
		ORDER BY called_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var calls []ServiceCall
	for rows.Next() {
		var call ServiceCall
		var serviceData, entityID sql.NullString
		err := rows.Scan(&call.ID, &call.InstanceID, &call.Domain, &call.Service, &serviceData, &entityID, &call.Status, &call.CalledAt, &call.CreatedAt)
		if err != nil {
			return nil, err
		}
		if serviceData.Valid {
			call.ServiceData = serviceData.String
		}
		if entityID.Valid {
			call.EntityID = entityID.String
		}
		calls = append(calls, call)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return calls, nil
}
