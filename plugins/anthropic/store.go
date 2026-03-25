// ABOUTME: Database layer for Anthropic mock API plugin
// ABOUTME: Manages scenarios for pattern-matched responses and message logging

package anthropic

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type AnthropicStore struct {
	db *sql.DB
}

func NewAnthropicStore(db *sql.DB) (*AnthropicStore, error) {
	store := &AnthropicStore{db: db}
	if err := store.initTables(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *AnthropicStore) initTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS anthropic_scenarios (
			id TEXT PRIMARY KEY,
			pattern TEXT,
			response_type TEXT,
			response_text TEXT,
			tool_name TEXT,
			tool_input TEXT,
			priority INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS anthropic_messages (
			id TEXT PRIMARY KEY,
			request_body TEXT,
			response_type TEXT,
			scenario_id TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE INDEX IF NOT EXISTS idx_anthropic_scenarios_priority ON anthropic_scenarios(priority DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_anthropic_messages_created_at ON anthropic_messages(created_at DESC)`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

type Scenario struct {
	ID           string    `json:"id"`
	Pattern      string    `json:"pattern"`
	ResponseType string    `json:"response_type"`
	ResponseText string    `json:"response_text"`
	ToolName     string    `json:"tool_name"`
	ToolInput    string    `json:"tool_input"`
	Priority     int       `json:"priority"`
	CreatedAt    time.Time `json:"created_at"`
}

type Message struct {
	ID           string    `json:"id"`
	RequestBody  string    `json:"request_body"`
	ResponseType string    `json:"response_type"`
	ScenarioID   string    `json:"scenario_id"`
	CreatedAt    time.Time `json:"created_at"`
}

func (s *AnthropicStore) CreateScenario(scenario *Scenario) error {
	scenario.ID = uuid.New().String()
	scenario.CreatedAt = time.Now()

	query := `INSERT INTO anthropic_scenarios (id, pattern, response_type, response_text, tool_name, tool_input, priority, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, scenario.ID, scenario.Pattern, scenario.ResponseType, scenario.ResponseText,
		scenario.ToolName, scenario.ToolInput, scenario.Priority, scenario.CreatedAt)
	return err
}

func (s *AnthropicStore) GetScenario(id string) (*Scenario, error) {
	query := `SELECT id, pattern, response_type, response_text, tool_name, tool_input, priority, created_at
		FROM anthropic_scenarios WHERE id = ?`

	scenario := &Scenario{}
	err := s.db.QueryRow(query, id).Scan(
		&scenario.ID, &scenario.Pattern, &scenario.ResponseType, &scenario.ResponseText,
		&scenario.ToolName, &scenario.ToolInput, &scenario.Priority, &scenario.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return scenario, nil
}

// MatchScenario finds the highest-priority scenario whose pattern is a substring
// of the request body. An empty pattern acts as a catch-all that matches anything.
func (s *AnthropicStore) MatchScenario(requestBody string) (*Scenario, error) {
	query := `SELECT id, pattern, response_type, response_text, tool_name, tool_input, priority, created_at
		FROM anthropic_scenarios
		WHERE pattern = '' OR INSTR(?, pattern) > 0
		ORDER BY priority DESC
		LIMIT 1`

	scenario := &Scenario{}
	err := s.db.QueryRow(query, requestBody).Scan(
		&scenario.ID, &scenario.Pattern, &scenario.ResponseType, &scenario.ResponseText,
		&scenario.ToolName, &scenario.ToolInput, &scenario.Priority, &scenario.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return scenario, nil
}

func (s *AnthropicStore) ListScenarios() ([]*Scenario, error) {
	query := `SELECT id, pattern, response_type, response_text, tool_name, tool_input, priority, created_at
		FROM anthropic_scenarios ORDER BY priority DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scenarios []*Scenario
	for rows.Next() {
		scenario := &Scenario{}
		err := rows.Scan(
			&scenario.ID, &scenario.Pattern, &scenario.ResponseType, &scenario.ResponseText,
			&scenario.ToolName, &scenario.ToolInput, &scenario.Priority, &scenario.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		scenarios = append(scenarios, scenario)
	}
	return scenarios, rows.Err()
}

func (s *AnthropicStore) DeleteScenario(id string) error {
	query := `DELETE FROM anthropic_scenarios WHERE id = ?`
	_, err := s.db.Exec(query, id)
	return err
}

func (s *AnthropicStore) CreateMessage(msg *Message) error {
	msg.ID = uuid.New().String()
	msg.CreatedAt = time.Now()

	query := `INSERT INTO anthropic_messages (id, request_body, response_type, scenario_id, created_at)
		VALUES (?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, msg.ID, msg.RequestBody, msg.ResponseType, msg.ScenarioID, msg.CreatedAt)
	return err
}

func (s *AnthropicStore) ListMessages(limit, offset int) ([]*Message, error) {
	query := `SELECT id, request_body, response_type, scenario_id, created_at
		FROM anthropic_messages ORDER BY created_at DESC LIMIT ? OFFSET ?`

	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		msg := &Message{}
		err := rows.Scan(&msg.ID, &msg.RequestBody, &msg.ResponseType, &msg.ScenarioID, &msg.CreatedAt)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

// Reset clears all data from both tables
func (s *AnthropicStore) Reset() error {
	queries := []string{
		`DELETE FROM anthropic_messages`,
		`DELETE FROM anthropic_scenarios`,
	}
	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}
