# Anthropic API Mock Plugin Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add an `anthropic` plugin to ISH that mocks the Anthropic `/v1/messages` endpoint with streaming SSE responses and configurable scenarios.

**Architecture:** Standard ISH plugin pattern — DB-backed store with scenarios table, SSE streaming via `http.Flusher`, scenario CRUD endpoints for test setup. Consumers seed their own scenarios; `Seed()` provides generic defaults.

**Tech Stack:** Go, chi/v5, SQLite3, SSE (text/event-stream)

---

### Task 1: Store Layer — Tables and Scenario CRUD

**Files:**
- Create: `plugins/anthropic/store.go`

**Step 1: Write the failing test**

Create `plugins/anthropic/store_test.go`:

```go
// ABOUTME: Database store tests for Anthropic plugin
// ABOUTME: Tests scenario and message CRUD operations

package anthropic

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestStore(t *testing.T) *AnthropicStore {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	store, err := NewAnthropicStore(db)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	return store
}

func TestCreateAndGetScenario(t *testing.T) {
	store := setupTestStore(t)

	scenario := &Scenario{
		Pattern:      "hello",
		ResponseType: "text",
		ResponseText: "Hello! How can I help you today?",
		Priority:     10,
	}
	err := store.CreateScenario(scenario)
	if err != nil {
		t.Fatalf("Failed to create scenario: %v", err)
	}
	if scenario.ID == "" {
		t.Fatal("Scenario ID should be generated")
	}

	got, err := store.GetScenario(scenario.ID)
	if err != nil {
		t.Fatalf("Failed to get scenario: %v", err)
	}
	if got.Pattern != "hello" {
		t.Fatalf("Expected pattern 'hello', got '%s'", got.Pattern)
	}
	if got.ResponseText != "Hello! How can I help you today?" {
		t.Fatalf("Unexpected response text: %s", got.ResponseText)
	}
}

func TestMatchScenario(t *testing.T) {
	store := setupTestStore(t)

	// Create scenarios with different priorities
	store.CreateScenario(&Scenario{
		Pattern:      "",
		ResponseType: "text",
		ResponseText: "I received your message.",
		Priority:     0,
	})
	store.CreateScenario(&Scenario{
		Pattern:      "hello",
		ResponseType: "text",
		ResponseText: "Hello! How can I help you today?",
		Priority:     10,
	})

	// Should match "hello" scenario (higher priority)
	matched, err := store.MatchScenario("hello there")
	if err != nil {
		t.Fatalf("Failed to match scenario: %v", err)
	}
	if matched.ResponseText != "Hello! How can I help you today?" {
		t.Fatalf("Expected hello scenario, got: %s", matched.ResponseText)
	}

	// Should match catch-all for unknown input
	matched, err = store.MatchScenario("something random")
	if err != nil {
		t.Fatalf("Failed to match scenario: %v", err)
	}
	if matched.ResponseText != "I received your message." {
		t.Fatalf("Expected catch-all, got: %s", matched.ResponseText)
	}
}

func TestListScenarios(t *testing.T) {
	store := setupTestStore(t)

	store.CreateScenario(&Scenario{
		Pattern:      "hello",
		ResponseType: "text",
		ResponseText: "Hi!",
		Priority:     10,
	})
	store.CreateScenario(&Scenario{
		Pattern:      "bye",
		ResponseType: "text",
		ResponseText: "Goodbye!",
		Priority:     5,
	})

	scenarios, err := store.ListScenarios()
	if err != nil {
		t.Fatalf("Failed to list scenarios: %v", err)
	}
	if len(scenarios) != 2 {
		t.Fatalf("Expected 2 scenarios, got %d", len(scenarios))
	}
	// Should be ordered by priority DESC
	if scenarios[0].Pattern != "hello" {
		t.Fatal("Expected highest priority first")
	}
}

func TestDeleteScenario(t *testing.T) {
	store := setupTestStore(t)

	scenario := &Scenario{
		Pattern:      "hello",
		ResponseType: "text",
		ResponseText: "Hi!",
		Priority:     10,
	}
	store.CreateScenario(scenario)

	err := store.DeleteScenario(scenario.ID)
	if err != nil {
		t.Fatalf("Failed to delete scenario: %v", err)
	}

	_, err = store.GetScenario(scenario.ID)
	if err == nil {
		t.Fatal("Scenario should be deleted")
	}
}

func TestCreateAndListMessages(t *testing.T) {
	store := setupTestStore(t)

	msg := &Message{
		RequestBody:  `{"model":"claude-sonnet-4-20250514","messages":[{"role":"user","content":"hello"}]}`,
		ResponseType: "text",
		ScenarioID:   "scenario-123",
	}
	err := store.CreateMessage(msg)
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}
	if msg.ID == "" {
		t.Fatal("Message ID should be generated")
	}

	messages, err := store.ListMessages(10, 0)
	if err != nil {
		t.Fatalf("Failed to list messages: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(messages))
	}
}

func TestResetStore(t *testing.T) {
	store := setupTestStore(t)

	store.CreateScenario(&Scenario{
		Pattern:      "hello",
		ResponseType: "text",
		ResponseText: "Hi!",
		Priority:     10,
	})
	store.CreateMessage(&Message{
		RequestBody:  "{}",
		ResponseType: "text",
	})

	err := store.Reset()
	if err != nil {
		t.Fatalf("Failed to reset: %v", err)
	}

	scenarios, _ := store.ListScenarios()
	if len(scenarios) != 0 {
		t.Fatal("Scenarios should be cleared after reset")
	}
	messages, _ := store.ListMessages(10, 0)
	if len(messages) != 0 {
		t.Fatal("Messages should be cleared after reset")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/dylanr/work/2389/ish && go test ./plugins/anthropic/ -v`
Expected: FAIL — package doesn't exist yet

**Step 3: Write minimal implementation**

Create `plugins/anthropic/store.go`:

```go
// ABOUTME: Database layer for Anthropic API mock plugin
// ABOUTME: Manages anthropic_scenarios and anthropic_messages tables

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
			pattern TEXT NOT NULL DEFAULT '',
			response_type TEXT NOT NULL DEFAULT 'text',
			response_text TEXT NOT NULL DEFAULT '',
			tool_name TEXT NOT NULL DEFAULT '',
			tool_input TEXT NOT NULL DEFAULT '{}',
			priority INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS anthropic_messages (
			id TEXT PRIMARY KEY,
			request_body TEXT NOT NULL,
			response_type TEXT NOT NULL DEFAULT '',
			scenario_id TEXT NOT NULL DEFAULT '',
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
	ResponseText string    `json:"response_text,omitempty"`
	ToolName     string    `json:"tool_name,omitempty"`
	ToolInput    string    `json:"tool_input,omitempty"`
	Priority     int       `json:"priority"`
	CreatedAt    time.Time `json:"created_at"`
}

type Message struct {
	ID           string    `json:"id"`
	RequestBody  string    `json:"request_body"`
	ResponseType string    `json:"response_type"`
	ScenarioID   string    `json:"scenario_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

func (s *AnthropicStore) CreateScenario(scenario *Scenario) error {
	if scenario.ID == "" {
		scenario.ID = uuid.New().String()
	}
	scenario.CreatedAt = time.Now()

	query := `INSERT INTO anthropic_scenarios (id, pattern, response_type, response_text, tool_name, tool_input, priority, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, scenario.ID, scenario.Pattern, scenario.ResponseType,
		scenario.ResponseText, scenario.ToolName, scenario.ToolInput, scenario.Priority, scenario.CreatedAt)
	return err
}

func (s *AnthropicStore) GetScenario(id string) (*Scenario, error) {
	query := `SELECT id, pattern, response_type, response_text, tool_name, tool_input, priority, created_at
		FROM anthropic_scenarios WHERE id = ?`
	scenario := &Scenario{}
	err := s.db.QueryRow(query, id).Scan(
		&scenario.ID, &scenario.Pattern, &scenario.ResponseType,
		&scenario.ResponseText, &scenario.ToolName, &scenario.ToolInput,
		&scenario.Priority, &scenario.CreatedAt)
	return scenario, err
}

// MatchScenario finds the highest-priority scenario whose pattern is a substring of the input.
// An empty pattern acts as a catch-all.
func (s *AnthropicStore) MatchScenario(input string) (*Scenario, error) {
	query := `SELECT id, pattern, response_type, response_text, tool_name, tool_input, priority, created_at
		FROM anthropic_scenarios
		WHERE pattern = '' OR instr(lower(?), lower(pattern)) > 0
		ORDER BY priority DESC
		LIMIT 1`
	scenario := &Scenario{}
	err := s.db.QueryRow(query, input).Scan(
		&scenario.ID, &scenario.Pattern, &scenario.ResponseType,
		&scenario.ResponseText, &scenario.ToolName, &scenario.ToolInput,
		&scenario.Priority, &scenario.CreatedAt)
	return scenario, err
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
		err := rows.Scan(&scenario.ID, &scenario.Pattern, &scenario.ResponseType,
			&scenario.ResponseText, &scenario.ToolName, &scenario.ToolInput,
			&scenario.Priority, &scenario.CreatedAt)
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
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
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

func (s *AnthropicStore) Reset() error {
	if _, err := s.db.Exec("DELETE FROM anthropic_messages"); err != nil {
		return err
	}
	if _, err := s.db.Exec("DELETE FROM anthropic_scenarios"); err != nil {
		return err
	}
	return nil
}
```

**Step 4: Run tests to verify they pass**

Run: `cd /Users/dylanr/work/2389/ish && go test ./plugins/anthropic/ -v`
Expected: All tests PASS

**Step 5: Commit**

```bash
git add plugins/anthropic/store.go plugins/anthropic/store_test.go
git commit -m "feat(anthropic): add store layer with scenario and message CRUD"
```

---

### Task 2: SSE Formatting Helpers

**Files:**
- Create: `plugins/anthropic/sse.go`

**Step 1: Write the failing test**

Create `plugins/anthropic/sse_test.go`:

```go
// ABOUTME: Tests for SSE event formatting helpers
// ABOUTME: Verifies correct Anthropic streaming event format

package anthropic

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteMessageStart(t *testing.T) {
	var buf bytes.Buffer
	writeMessageStart(&buf, "msg_123", "claude-sonnet-4-20250514")

	output := buf.String()
	if !strings.Contains(output, "event: message_start") {
		t.Fatal("Missing event: message_start")
	}
	if !strings.Contains(output, `"type":"message_start"`) {
		t.Fatal("Missing type field")
	}
	if !strings.Contains(output, `"id":"msg_123"`) {
		t.Fatal("Missing message id")
	}
	if !strings.Contains(output, `"role":"assistant"`) {
		t.Fatal("Missing role")
	}
}

func TestWriteTextContentBlock(t *testing.T) {
	var buf bytes.Buffer
	writeContentBlockStart(&buf, 0, "text")

	output := buf.String()
	if !strings.Contains(output, "event: content_block_start") {
		t.Fatal("Missing event: content_block_start")
	}
	if !strings.Contains(output, `"type":"text"`) {
		t.Fatal("Missing text type")
	}
}

func TestWriteTextDelta(t *testing.T) {
	var buf bytes.Buffer
	writeTextDelta(&buf, 0, "Hello world")

	output := buf.String()
	if !strings.Contains(output, "event: content_block_delta") {
		t.Fatal("Missing event: content_block_delta")
	}
	if !strings.Contains(output, `"text":"Hello world"`) {
		t.Fatal("Missing text content")
	}
}

func TestWriteToolUseContentBlock(t *testing.T) {
	var buf bytes.Buffer
	writeToolUseBlockStart(&buf, 0, "toolu_123", "get_current_time")

	output := buf.String()
	if !strings.Contains(output, "event: content_block_start") {
		t.Fatal("Missing event: content_block_start")
	}
	if !strings.Contains(output, `"type":"tool_use"`) {
		t.Fatal("Missing tool_use type")
	}
	if !strings.Contains(output, `"name":"get_current_time"`) {
		t.Fatal("Missing tool name")
	}
}

func TestWriteInputJSONDelta(t *testing.T) {
	var buf bytes.Buffer
	writeInputJSONDelta(&buf, 0, `{"key":"value"}`)

	output := buf.String()
	if !strings.Contains(output, `"type":"input_json_delta"`) {
		t.Fatal("Missing input_json_delta type")
	}
	if !strings.Contains(output, `{"key":"value"}`) {
		t.Fatal("Missing partial JSON")
	}
}

func TestWriteMessageStop(t *testing.T) {
	var buf bytes.Buffer
	writeContentBlockStop(&buf, 0)
	writeMessageDelta(&buf, "end_turn")
	writeMessageStop(&buf)

	output := buf.String()
	if !strings.Contains(output, "event: content_block_stop") {
		t.Fatal("Missing content_block_stop")
	}
	if !strings.Contains(output, "event: message_delta") {
		t.Fatal("Missing message_delta")
	}
	if !strings.Contains(output, `"stop_reason":"end_turn"`) {
		t.Fatal("Missing stop_reason")
	}
	if !strings.Contains(output, "event: message_stop") {
		t.Fatal("Missing message_stop")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/dylanr/work/2389/ish && go test ./plugins/anthropic/ -run TestWrite -v`
Expected: FAIL — functions don't exist

**Step 3: Write minimal implementation**

Create `plugins/anthropic/sse.go`:

```go
// ABOUTME: SSE event formatting helpers for Anthropic streaming responses
// ABOUTME: Generates properly formatted Server-Sent Events matching Anthropic's API format

package anthropic

import (
	"encoding/json"
	"fmt"
	"io"
)

func writeSSEEvent(w io.Writer, event string, data interface{}) {
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, jsonData)
}

func writeMessageStart(w io.Writer, msgID string, model string) {
	writeSSEEvent(w, "message_start", map[string]interface{}{
		"type": "message_start",
		"message": map[string]interface{}{
			"id":          msgID,
			"type":        "message",
			"role":        "assistant",
			"content":     []interface{}{},
			"model":       model,
			"stop_reason": nil,
		},
	})
}

func writeContentBlockStart(w io.Writer, index int, blockType string) {
	writeSSEEvent(w, "content_block_start", map[string]interface{}{
		"type":  "content_block_start",
		"index": index,
		"content_block": map[string]interface{}{
			"type": blockType,
			"text": "",
		},
	})
}

func writeToolUseBlockStart(w io.Writer, index int, toolUseID string, toolName string) {
	writeSSEEvent(w, "content_block_start", map[string]interface{}{
		"type":  "content_block_start",
		"index": index,
		"content_block": map[string]interface{}{
			"type":  "tool_use",
			"id":    toolUseID,
			"name":  toolName,
			"input": map[string]interface{}{},
		},
	})
}

func writeTextDelta(w io.Writer, index int, text string) {
	writeSSEEvent(w, "content_block_delta", map[string]interface{}{
		"type":  "content_block_delta",
		"index": index,
		"delta": map[string]interface{}{
			"type": "text_delta",
			"text": text,
		},
	})
}

func writeInputJSONDelta(w io.Writer, index int, partialJSON string) {
	writeSSEEvent(w, "content_block_delta", map[string]interface{}{
		"type":  "content_block_delta",
		"index": index,
		"delta": map[string]interface{}{
			"type":         "input_json_delta",
			"partial_json": partialJSON,
		},
	})
}

func writeContentBlockStop(w io.Writer, index int) {
	writeSSEEvent(w, "content_block_stop", map[string]interface{}{
		"type":  "content_block_stop",
		"index": index,
	})
}

func writeMessageDelta(w io.Writer, stopReason string) {
	writeSSEEvent(w, "message_delta", map[string]interface{}{
		"type": "message_delta",
		"delta": map[string]interface{}{
			"stop_reason": stopReason,
		},
	})
}

func writeMessageStop(w io.Writer) {
	writeSSEEvent(w, "message_stop", map[string]interface{}{
		"type": "message_stop",
	})
}
```

**Step 4: Run tests to verify they pass**

Run: `cd /Users/dylanr/work/2389/ish && go test ./plugins/anthropic/ -run TestWrite -v`
Expected: All tests PASS

**Step 5: Commit**

```bash
git add plugins/anthropic/sse.go plugins/anthropic/sse_test.go
git commit -m "feat(anthropic): add SSE event formatting helpers"
```

---

### Task 3: Plugin Skeleton and Scenario CRUD Handlers

**Files:**
- Create: `plugins/anthropic/plugin.go`
- Create: `plugins/anthropic/scenarios.go`
- Create: `plugins/anthropic/schema.go`
- Create: `plugins/anthropic/seed.go`

**Step 1: Write the failing test**

Create `plugins/anthropic/handlers_test.go`:

```go
// ABOUTME: HTTP handler tests for Anthropic plugin endpoints
// ABOUTME: Tests scenario CRUD and message streaming handlers

package anthropic

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestPlugin(t *testing.T) *AnthropicPlugin {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	plugin := &AnthropicPlugin{}
	if err := plugin.SetDB(db); err != nil {
		t.Fatalf("Failed to initialize plugin: %v", err)
	}
	return plugin
}

func newRequestWithParams(method, url string, body []byte, params map[string]string) *http.Request {
	req := httptest.NewRequest(method, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	return req.WithContext(ctx)
}

func TestCreateScenarioHandler(t *testing.T) {
	plugin := setupTestPlugin(t)

	body := map[string]interface{}{
		"pattern":       "hello",
		"response_type": "text",
		"response_text": "Hi there!",
		"priority":      10,
	}
	bodyJSON, _ := json.Marshal(body)

	req := newRequestWithParams("POST", "/v1/scenarios", bodyJSON, nil)
	w := httptest.NewRecorder()
	plugin.createScenario(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var scenario Scenario
	json.NewDecoder(w.Body).Decode(&scenario)
	if scenario.Pattern != "hello" {
		t.Fatalf("Expected pattern 'hello', got '%s'", scenario.Pattern)
	}
	if scenario.ID == "" {
		t.Fatal("Scenario should have an ID")
	}
}

func TestListScenariosHandler(t *testing.T) {
	plugin := setupTestPlugin(t)

	plugin.store.CreateScenario(&Scenario{
		Pattern: "hello", ResponseType: "text", ResponseText: "Hi!", Priority: 10,
	})
	plugin.store.CreateScenario(&Scenario{
		Pattern: "bye", ResponseType: "text", ResponseText: "Goodbye!", Priority: 5,
	})

	req := newRequestWithParams("GET", "/v1/scenarios", nil, nil)
	w := httptest.NewRecorder()
	plugin.listScenarios(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var scenarios []*Scenario
	json.NewDecoder(w.Body).Decode(&scenarios)
	if len(scenarios) != 2 {
		t.Fatalf("Expected 2 scenarios, got %d", len(scenarios))
	}
}

func TestDeleteScenarioHandler(t *testing.T) {
	plugin := setupTestPlugin(t)

	scenario := &Scenario{
		Pattern: "hello", ResponseType: "text", ResponseText: "Hi!", Priority: 10,
	}
	plugin.store.CreateScenario(scenario)

	req := newRequestWithParams("DELETE", "/v1/scenarios/"+scenario.ID, nil, map[string]string{
		"id": scenario.ID,
	})
	w := httptest.NewRecorder()
	plugin.deleteScenarioHandler(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("Expected 204, got %d", w.Code)
	}
}

func TestPluginName(t *testing.T) {
	plugin := &AnthropicPlugin{}
	if plugin.Name() != "anthropic" {
		t.Fatalf("Expected name 'anthropic', got '%s'", plugin.Name())
	}
}

func TestPluginHealth(t *testing.T) {
	plugin := &AnthropicPlugin{}
	health := plugin.Health()
	if health.Status != "healthy" {
		t.Fatalf("Expected healthy status, got '%s'", health.Status)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/dylanr/work/2389/ish && go test ./plugins/anthropic/ -run "TestCreate|TestList|TestDelete|TestPlugin" -v`
Expected: FAIL — types and functions don't exist

**Step 3: Write implementation**

Create `plugins/anthropic/plugin.go`:

```go
// ABOUTME: Anthropic API mock plugin for ISH
// ABOUTME: Simulates the Anthropic /v1/messages endpoint with streaming SSE responses

package anthropic

import (
	"context"
	"database/sql"

	"github.com/2389/ish/plugins/core"
	"github.com/go-chi/chi/v5"
)

func init() {
	core.Register(&AnthropicPlugin{})
}

type AnthropicPlugin struct {
	store *AnthropicStore
}

func (p *AnthropicPlugin) Name() string {
	return "anthropic"
}

func (p *AnthropicPlugin) Health() core.HealthStatus {
	return core.HealthStatus{
		Status:  "healthy",
		Message: "Anthropic API mock plugin operational",
	}
}

func (p *AnthropicPlugin) RegisterRoutes(r chi.Router) {
	r.Post("/v1/messages", p.handleMessages)

	r.Route("/v1/scenarios", func(r chi.Router) {
		r.Post("/", p.createScenario)
		r.Get("/", p.listScenarios)
		r.Delete("/{id}", p.deleteScenarioHandler)
	})
}

func (p *AnthropicPlugin) RegisterAuth(r chi.Router) {
	// Anthropic uses API keys in headers, no OAuth flow needed
}

func (p *AnthropicPlugin) ValidateToken(token string) bool {
	// Accept any token — this is a mock
	return true
}

func (p *AnthropicPlugin) SetDB(db *sql.DB) error {
	store, err := NewAnthropicStore(db)
	if err != nil {
		return err
	}
	p.store = store
	return nil
}

func (p *AnthropicPlugin) Reset(ctx context.Context) error {
	return p.store.Reset()
}
```

Create `plugins/anthropic/scenarios.go`:

```go
// ABOUTME: HTTP handlers for scenario CRUD endpoints
// ABOUTME: Allows tests to seed custom response scenarios via API

package anthropic

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (p *AnthropicPlugin) createScenario(w http.ResponseWriter, r *http.Request) {
	if p.store == nil {
		writeError(w, 500, "Plugin not initialized")
		return
	}

	var scenario Scenario
	if err := json.NewDecoder(r.Body).Decode(&scenario); err != nil {
		writeError(w, 400, "Invalid request body")
		return
	}

	if err := p.store.CreateScenario(&scenario); err != nil {
		writeError(w, 500, "Failed to create scenario")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(scenario)
}

func (p *AnthropicPlugin) listScenarios(w http.ResponseWriter, r *http.Request) {
	if p.store == nil {
		writeError(w, 500, "Plugin not initialized")
		return
	}

	scenarios, err := p.store.ListScenarios()
	if err != nil {
		writeError(w, 500, "Failed to list scenarios")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scenarios)
}

func (p *AnthropicPlugin) deleteScenarioHandler(w http.ResponseWriter, r *http.Request) {
	if p.store == nil {
		writeError(w, 500, "Plugin not initialized")
		return
	}

	id := chi.URLParam(r, "id")
	if err := p.store.DeleteScenario(id); err != nil {
		writeError(w, 500, "Failed to delete scenario")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"type":    "error",
		"message": message,
	})
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
```

Create `plugins/anthropic/schema.go`:

```go
// ABOUTME: Admin UI schema definitions for Anthropic plugin
// ABOUTME: Defines Scenarios and Messages resources for schema-driven UI

package anthropic

import (
	"context"
	"fmt"

	"github.com/2389/ish/plugins/core"
)

func (p *AnthropicPlugin) Schema() core.PluginSchema {
	return core.PluginSchema{
		Resources: []core.ResourceSchema{
			{
				Name:        "Scenarios",
				Slug:        "scenarios",
				ListColumns: []string{"pattern", "response_type", "priority", "created_at"},
				Fields: []core.FieldSchema{
					{Name: "id", Type: "string", Display: "ID", Required: false, Editable: false},
					{Name: "pattern", Type: "string", Display: "Pattern", Required: true, Editable: true},
					{Name: "response_type", Type: "string", Display: "Response Type", Required: true, Editable: true},
					{Name: "response_text", Type: "text", Display: "Response Text", Required: false, Editable: true},
					{Name: "tool_name", Type: "string", Display: "Tool Name", Required: false, Editable: true},
					{Name: "tool_input", Type: "text", Display: "Tool Input (JSON)", Required: false, Editable: true},
					{Name: "priority", Type: "string", Display: "Priority", Required: true, Editable: true},
					{Name: "created_at", Type: "datetime", Display: "Created", Required: false, Editable: false},
				},
				Actions: []core.ActionSchema{
					{Name: "delete", HTTPMethod: "DELETE", Endpoint: "/v1/scenarios/{id}", Confirm: true},
				},
			},
			{
				Name:        "Messages",
				Slug:        "messages",
				ListColumns: []string{"id", "response_type", "scenario_id", "created_at"},
				Fields: []core.FieldSchema{
					{Name: "id", Type: "string", Display: "ID", Required: false, Editable: false},
					{Name: "request_body", Type: "text", Display: "Request Body", Required: false, Editable: false},
					{Name: "response_type", Type: "string", Display: "Response Type", Required: false, Editable: false},
					{Name: "scenario_id", Type: "string", Display: "Scenario ID", Required: false, Editable: false},
					{Name: "created_at", Type: "datetime", Display: "Created", Required: false, Editable: false},
				},
				Actions: []core.ActionSchema{},
			},
		},
	}
}

// ListResources implements core.DataProvider
func (p *AnthropicPlugin) ListResources(ctx context.Context, slug string, opts core.ListOptions) ([]map[string]interface{}, error) {
	switch slug {
	case "scenarios":
		scenarios, err := p.store.ListScenarios()
		if err != nil {
			return nil, err
		}
		result := make([]map[string]interface{}, 0, len(scenarios))
		for _, s := range scenarios {
			result = append(result, map[string]interface{}{
				"id":            s.ID,
				"pattern":       s.Pattern,
				"response_type": s.ResponseType,
				"response_text": s.ResponseText,
				"tool_name":     s.ToolName,
				"tool_input":    s.ToolInput,
				"priority":      fmt.Sprintf("%d", s.Priority),
				"created_at":    s.CreatedAt.Format("2006-01-02T15:04:05Z"),
			})
		}
		return result, nil
	case "messages":
		messages, err := p.store.ListMessages(opts.Limit, opts.Offset)
		if err != nil {
			return nil, err
		}
		result := make([]map[string]interface{}, 0, len(messages))
		for _, m := range messages {
			result = append(result, map[string]interface{}{
				"id":            m.ID,
				"request_body":  m.RequestBody,
				"response_type": m.ResponseType,
				"scenario_id":   m.ScenarioID,
				"created_at":    m.CreatedAt.Format("2006-01-02T15:04:05Z"),
			})
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unknown resource: %s", slug)
	}
}

// GetResource implements core.DataProvider
func (p *AnthropicPlugin) GetResource(ctx context.Context, slug string, id string) (map[string]interface{}, error) {
	switch slug {
	case "scenarios":
		s, err := p.store.GetScenario(id)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"id":            s.ID,
			"pattern":       s.Pattern,
			"response_type": s.ResponseType,
			"response_text": s.ResponseText,
			"tool_name":     s.ToolName,
			"tool_input":    s.ToolInput,
			"priority":      fmt.Sprintf("%d", s.Priority),
			"created_at":    s.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}, nil
	default:
		return nil, fmt.Errorf("unknown resource or unsupported get: %s", slug)
	}
}
```

Create `plugins/anthropic/seed.go`:

```go
// ABOUTME: Default seed data for Anthropic plugin
// ABOUTME: Creates generic greeting and catch-all scenarios

package anthropic

import (
	"context"
	"fmt"

	"github.com/2389/ish/plugins/core"
)

func (p *AnthropicPlugin) Seed(ctx context.Context, size string) (core.SeedData, error) {
	scenarios := []Scenario{
		{
			Pattern:      "hello",
			ResponseType: "text",
			ResponseText: "Hello! How can I help you today?",
			Priority:     10,
		},
		{
			Pattern:      "",
			ResponseType: "text",
			ResponseText: "I received your message and I'm happy to help.",
			Priority:     0,
		},
	}

	total := 0
	for _, s := range scenarios {
		if err := p.store.CreateScenario(&s); err != nil {
			continue
		}
		total++
	}

	return core.SeedData{
		Summary: fmt.Sprintf("Created %d default scenarios", total),
		Records: map[string]int{
			"scenarios": total,
		},
	}, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `cd /Users/dylanr/work/2389/ish && go test ./plugins/anthropic/ -v`
Expected: All tests PASS

**Step 5: Commit**

```bash
git add plugins/anthropic/plugin.go plugins/anthropic/scenarios.go plugins/anthropic/schema.go plugins/anthropic/seed.go plugins/anthropic/handlers_test.go
git commit -m "feat(anthropic): add plugin skeleton, scenario CRUD handlers, schema, and seed"
```

---

### Task 4: POST /v1/messages Handler with SSE Streaming

**Files:**
- Create: `plugins/anthropic/handlers.go`

**Step 1: Write the failing test**

Add to `plugins/anthropic/handlers_test.go`:

```go
func TestHandleMessagesTextResponse(t *testing.T) {
	plugin := setupTestPlugin(t)

	// Seed a "hello" scenario
	plugin.store.CreateScenario(&Scenario{
		Pattern: "hello", ResponseType: "text", ResponseText: "Hi there!", Priority: 10,
	})

	body := map[string]interface{}{
		"model":  "claude-sonnet-4-20250514",
		"stream": true,
		"messages": []map[string]interface{}{
			{"role": "user", "content": "hello world"},
		},
	}
	bodyJSON, _ := json.Marshal(body)

	req := newRequestWithParams("POST", "/v1/messages", bodyJSON, nil)
	w := httptest.NewRecorder()
	plugin.handleMessages(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	output := w.Body.String()
	if !strings.Contains(output, "event: message_start") {
		t.Fatal("Missing message_start event")
	}
	if !strings.Contains(output, "event: content_block_start") {
		t.Fatal("Missing content_block_start event")
	}
	if !strings.Contains(output, "Hi there!") {
		t.Fatal("Missing response text in delta")
	}
	if !strings.Contains(output, "event: message_stop") {
		t.Fatal("Missing message_stop event")
	}

	// Verify content type
	if ct := w.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Expected text/event-stream, got %s", ct)
	}
}

func TestHandleMessagesToolUseResponse(t *testing.T) {
	plugin := setupTestPlugin(t)

	plugin.store.CreateScenario(&Scenario{
		Pattern:      "what time",
		ResponseType: "tool_use",
		ToolName:     "get_current_time",
		ToolInput:    `{"timezone":"UTC"}`,
		Priority:     10,
	})

	body := map[string]interface{}{
		"model":  "claude-sonnet-4-20250514",
		"stream": true,
		"messages": []map[string]interface{}{
			{"role": "user", "content": "what time is it?"},
		},
	}
	bodyJSON, _ := json.Marshal(body)

	req := newRequestWithParams("POST", "/v1/messages", bodyJSON, nil)
	w := httptest.NewRecorder()
	plugin.handleMessages(w, req)

	output := w.Body.String()
	if !strings.Contains(output, `"type":"tool_use"`) {
		t.Fatal("Missing tool_use content block")
	}
	if !strings.Contains(output, `"name":"get_current_time"`) {
		t.Fatal("Missing tool name")
	}
	if !strings.Contains(output, `input_json_delta`) {
		t.Fatal("Missing input_json_delta")
	}
	// Tool use should have stop_reason "tool_use"
	if !strings.Contains(output, `"stop_reason":"tool_use"`) {
		t.Fatal("Missing stop_reason tool_use")
	}
}

func TestHandleMessagesToolResultFollowUp(t *testing.T) {
	plugin := setupTestPlugin(t)

	body := map[string]interface{}{
		"model":  "claude-sonnet-4-20250514",
		"stream": true,
		"messages": []map[string]interface{}{
			{"role": "user", "content": "what time is it?"},
			{"role": "assistant", "content": []map[string]interface{}{
				{"type": "tool_use", "id": "toolu_123", "name": "get_current_time", "input": map[string]interface{}{}},
			}},
			{"role": "user", "content": []map[string]interface{}{
				{"type": "tool_result", "tool_use_id": "toolu_123", "content": "2026-03-24T10:30:00Z"},
			}},
		},
	}
	bodyJSON, _ := json.Marshal(body)

	req := newRequestWithParams("POST", "/v1/messages", bodyJSON, nil)
	w := httptest.NewRecorder()
	plugin.handleMessages(w, req)

	output := w.Body.String()
	if !strings.Contains(output, "event: message_start") {
		t.Fatal("Missing message_start")
	}
	// Should respond with text summarizing the tool result
	if !strings.Contains(output, "event: content_block_delta") {
		t.Fatal("Missing text delta for tool result follow-up")
	}
	if !strings.Contains(output, `"stop_reason":"end_turn"`) {
		t.Fatal("Tool result follow-up should have end_turn stop reason")
	}
}

func TestHandleMessagesCatchAll(t *testing.T) {
	plugin := setupTestPlugin(t)

	// Seed only a catch-all
	plugin.store.CreateScenario(&Scenario{
		Pattern: "", ResponseType: "text", ResponseText: "I received your message.", Priority: 0,
	})

	body := map[string]interface{}{
		"model":  "claude-sonnet-4-20250514",
		"stream": true,
		"messages": []map[string]interface{}{
			{"role": "user", "content": "something random"},
		},
	}
	bodyJSON, _ := json.Marshal(body)

	req := newRequestWithParams("POST", "/v1/messages", bodyJSON, nil)
	w := httptest.NewRecorder()
	plugin.handleMessages(w, req)

	output := w.Body.String()
	if !strings.Contains(output, "I received your message.") {
		t.Fatal("Expected catch-all response")
	}
}

func TestHandleMessagesLogsRequest(t *testing.T) {
	plugin := setupTestPlugin(t)

	plugin.store.CreateScenario(&Scenario{
		Pattern: "", ResponseType: "text", ResponseText: "Ok.", Priority: 0,
	})

	body := map[string]interface{}{
		"model":  "claude-sonnet-4-20250514",
		"stream": true,
		"messages": []map[string]interface{}{
			{"role": "user", "content": "test message"},
		},
	}
	bodyJSON, _ := json.Marshal(body)

	req := newRequestWithParams("POST", "/v1/messages", bodyJSON, nil)
	w := httptest.NewRecorder()
	plugin.handleMessages(w, req)

	messages, err := plugin.store.ListMessages(10, 0)
	if err != nil {
		t.Fatalf("Failed to list messages: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("Expected 1 logged message, got %d", len(messages))
	}
	if messages[0].ResponseType != "text" {
		t.Fatalf("Expected logged response_type 'text', got '%s'", messages[0].ResponseType)
	}
}

func TestHandleMessagesNoScenarios(t *testing.T) {
	plugin := setupTestPlugin(t)

	body := map[string]interface{}{
		"model":  "claude-sonnet-4-20250514",
		"stream": true,
		"messages": []map[string]interface{}{
			{"role": "user", "content": "hello"},
		},
	}
	bodyJSON, _ := json.Marshal(body)

	req := newRequestWithParams("POST", "/v1/messages", bodyJSON, nil)
	w := httptest.NewRecorder()
	plugin.handleMessages(w, req)

	// Should still return a valid SSE response with a fallback message
	output := w.Body.String()
	if !strings.Contains(output, "event: message_start") {
		t.Fatal("Should still produce valid SSE even with no scenarios")
	}
}
```

Add `"strings"` to the imports in the test file.

**Step 2: Run test to verify it fails**

Run: `cd /Users/dylanr/work/2389/ish && go test ./plugins/anthropic/ -run "TestHandleMessages" -v`
Expected: FAIL — `handleMessages` doesn't exist

**Step 3: Write implementation**

Create `plugins/anthropic/handlers.go`:

```go
// ABOUTME: HTTP handler for POST /v1/messages endpoint
// ABOUTME: Matches input against scenarios and streams SSE responses

package anthropic

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
)

// messagesRequest represents the Anthropic API request body
type messagesRequest struct {
	Model    string           `json:"model"`
	Stream   bool             `json:"stream"`
	Messages []messageContent `json:"messages"`
}

// messageContent represents a single message in the conversation
type messageContent struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

func (p *AnthropicPlugin) handleMessages(w http.ResponseWriter, r *http.Request) {
	if p.store == nil {
		writeError(w, 500, "Plugin not initialized")
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, 400, "Failed to read request body")
		return
	}

	var req messagesRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		writeError(w, 400, "Invalid request body")
		return
	}

	if len(req.Messages) == 0 {
		writeError(w, 400, "Messages array is required")
		return
	}

	model := req.Model
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	// Check if the last message contains a tool result
	lastMsg := req.Messages[len(req.Messages)-1]
	isToolResult := containsToolResult(lastMsg)

	var scenario *Scenario
	var responseType string

	if isToolResult {
		// For tool results, respond with a text summary
		responseType = "text"
		toolResultContent := extractToolResultContent(lastMsg)
		scenario = &Scenario{
			ResponseType: "text",
			ResponseText: fmt.Sprintf("I've completed the tool call. The result was: %s", toolResultContent),
		}
	} else {
		// Extract last user message text and match against scenarios
		userText := extractUserText(lastMsg)
		matched, err := p.store.MatchScenario(userText)
		if err == sql.ErrNoRows || err != nil {
			// No scenario matched — use a hardcoded fallback
			scenario = &Scenario{
				ResponseType: "text",
				ResponseText: "I received your message and I'm processing it.",
			}
			responseType = "text"
		} else {
			scenario = matched
			responseType = matched.ResponseType
		}
	}

	// Log the request
	scenarioID := ""
	if scenario.ID != "" {
		scenarioID = scenario.ID
	}
	p.store.CreateMessage(&Message{
		RequestBody:  string(bodyBytes),
		ResponseType: responseType,
		ScenarioID:   scenarioID,
	})

	// Stream the SSE response
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	msgID := "msg_" + uuid.New().String()[:12]

	writeMessageStart(w, msgID, model)

	switch scenario.ResponseType {
	case "tool_use":
		toolUseID := "toolu_" + uuid.New().String()[:12]
		writeToolUseBlockStart(w, 0, toolUseID, scenario.ToolName)
		toolInput := scenario.ToolInput
		if toolInput == "" {
			toolInput = "{}"
		}
		writeInputJSONDelta(w, 0, toolInput)
		writeContentBlockStop(w, 0)
		writeMessageDelta(w, "tool_use")
	default:
		writeContentBlockStart(w, 0, "text")
		writeTextDelta(w, 0, scenario.ResponseText)
		writeContentBlockStop(w, 0)
		writeMessageDelta(w, "end_turn")
	}

	writeMessageStop(w)

	// Flush if the writer supports it
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

// containsToolResult checks if a message contains a tool_result content block
func containsToolResult(msg messageContent) bool {
	if msg.Role != "user" {
		return false
	}
	// Content can be a string or an array of content blocks
	var blocks []struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		return false
	}
	for _, b := range blocks {
		if b.Type == "tool_result" {
			return true
		}
	}
	return false
}

// extractToolResultContent pulls the content string from a tool_result block
func extractToolResultContent(msg messageContent) string {
	var blocks []struct {
		Type    string `json:"type"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		return "unknown"
	}
	for _, b := range blocks {
		if b.Type == "tool_result" {
			return b.Content
		}
	}
	return "unknown"
}

// extractUserText gets the text content from the last user message.
// Content can be a plain string or an array of content blocks.
func extractUserText(msg messageContent) string {
	// Try as plain string first
	var text string
	if err := json.Unmarshal(msg.Content, &text); err == nil {
		return text
	}

	// Try as array of content blocks
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(msg.Content, &blocks); err == nil {
		for _, b := range blocks {
			if b.Type == "text" {
				return b.Text
			}
		}
	}

	return ""
}
```

**Step 4: Run tests to verify they pass**

Run: `cd /Users/dylanr/work/2389/ish && go test ./plugins/anthropic/ -v`
Expected: All tests PASS

**Step 5: Commit**

```bash
git add plugins/anthropic/handlers.go plugins/anthropic/handlers_test.go
git commit -m "feat(anthropic): add POST /v1/messages handler with SSE streaming"
```

---

### Task 5: Register Plugin in main.go

**Files:**
- Modify: `cmd/ish/main.go:31` (add import)

**Step 1: Add the import**

Add after the existing plugin imports in `cmd/ish/main.go`:

```go
_ "github.com/2389/ish/plugins/anthropic" // Register Anthropic plugin
```

**Step 2: Run full test suite**

Run: `cd /Users/dylanr/work/2389/ish && go test ./... -v`
Expected: All tests PASS (including existing tests)

**Step 3: Run the server manually to verify**

Run: `cd /Users/dylanr/work/2389/ish && go build ./cmd/ish/ && ./ish serve --port 8080 &`

Verify: `curl -s localhost:8080/admin/health | jq '.plugins[] | select(.name == "anthropic")'`
Expected: Shows anthropic plugin with healthy status

Clean up: Kill the background server

**Step 4: Commit**

```bash
git add cmd/ish/main.go
git commit -m "feat(anthropic): register plugin in main.go"
```

---

### Task 6: End-to-End Smoke Test

**Files:**
- Create: `test/e2e/anthropic_test.go`

**Step 1: Write the E2E test**

```go
// ABOUTME: End-to-end tests for Anthropic API mock plugin
// ABOUTME: Tests scenario seeding and SSE streaming via HTTP

package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestAnthropicScenarioCRUD(t *testing.T) {
	ts := StartTestServer(t)
	defer ts.Close()

	// Create a scenario
	scenario := map[string]interface{}{
		"pattern":       "hello",
		"response_type": "text",
		"response_text": "Hi from ISH!",
		"priority":      10,
	}
	scenarioJSON, _ := json.Marshal(scenario)
	resp := ts.Post(t, "/v1/scenarios", string(scenarioJSON))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", resp.StatusCode)
	}

	var created map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()
	scenarioID := created["id"].(string)

	// List scenarios
	resp = ts.Get(t, "/v1/scenarios")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
	var scenarios []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&scenarios)
	resp.Body.Close()
	if len(scenarios) < 1 {
		t.Fatal("Expected at least 1 scenario")
	}

	// Delete scenario
	resp = ts.Delete(t, "/v1/scenarios/"+scenarioID)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("Expected 204, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestAnthropicStreamingTextResponse(t *testing.T) {
	ts := StartTestServer(t)
	defer ts.Close()

	// Seed a scenario
	scenario := map[string]interface{}{
		"pattern":       "greetings",
		"response_type": "text",
		"response_text": "Hello from the mock!",
		"priority":      10,
	}
	scenarioJSON, _ := json.Marshal(scenario)
	resp := ts.Post(t, "/v1/scenarios", string(scenarioJSON))
	resp.Body.Close()

	// Send a message
	body := map[string]interface{}{
		"model":  "claude-sonnet-4-20250514",
		"stream": true,
		"messages": []map[string]interface{}{
			{"role": "user", "content": "greetings friend"},
		},
	}
	bodyJSON, _ := json.Marshal(body)
	resp = ts.Post(t, "/v1/messages", string(bodyJSON))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
	defer resp.Body.Close()

	// Verify SSE format
	buf := new(strings.Builder)
	buf.ReadFrom(resp.Body)
	output := buf.String()

	if !strings.Contains(output, "event: message_start") {
		t.Fatal("Missing message_start")
	}
	if !strings.Contains(output, "Hello from the mock!") {
		t.Fatal("Missing response text")
	}
	if !strings.Contains(output, "event: message_stop") {
		t.Fatal("Missing message_stop")
	}

	// Verify content type is SSE
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Expected text/event-stream, got %s", ct)
	}
}

func TestAnthropicStreamingToolUseResponse(t *testing.T) {
	ts := StartTestServer(t)
	defer ts.Close()

	// Seed a tool_use scenario
	scenario := map[string]interface{}{
		"pattern":       "what time",
		"response_type": "tool_use",
		"tool_name":     "get_current_time",
		"tool_input":    `{"timezone":"UTC"}`,
		"priority":      10,
	}
	scenarioJSON, _ := json.Marshal(scenario)
	resp := ts.Post(t, "/v1/scenarios", string(scenarioJSON))
	resp.Body.Close()

	body := map[string]interface{}{
		"model":  "claude-sonnet-4-20250514",
		"stream": true,
		"messages": []map[string]interface{}{
			{"role": "user", "content": "what time is it?"},
		},
	}
	bodyJSON, _ := json.Marshal(body)
	resp = ts.Post(t, "/v1/messages", string(bodyJSON))
	defer resp.Body.Close()

	buf := new(strings.Builder)
	buf.ReadFrom(resp.Body)
	output := buf.String()

	if !strings.Contains(output, `"type":"tool_use"`) {
		t.Fatal("Missing tool_use block")
	}
	if !strings.Contains(output, `"name":"get_current_time"`) {
		t.Fatal("Missing tool name")
	}
	if !strings.Contains(output, `"stop_reason":"tool_use"`) {
		t.Fatal("Missing tool_use stop_reason")
	}
}
```

Note: This test depends on the `StartTestServer` helper at `test/e2e/helpers.go` and may need adjustments based on its exact API (check for `Post`, `Get`, `Delete` methods).

**Step 2: Run E2E tests**

Run: `cd /Users/dylanr/work/2389/ish && go test ./test/e2e/ -run TestAnthropic -v`
Expected: All tests PASS

**Step 3: Run full test suite**

Run: `cd /Users/dylanr/work/2389/ish && go test ./... -v`
Expected: All tests PASS

**Step 4: Commit**

```bash
git add test/e2e/anthropic_test.go
git commit -m "test(anthropic): add end-to-end tests for scenario CRUD and SSE streaming"
```
