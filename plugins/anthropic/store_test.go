// ABOUTME: Unit tests for Anthropic plugin store layer
// ABOUTME: Tests scenario and message CRUD operations with in-memory SQLite

package anthropic

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	return db
}

func TestCreateAndGetScenario(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, err := NewAnthropicStore(db)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	scenario := &Scenario{
		Pattern:      "hello",
		ResponseType: "text",
		ResponseText: "Hi there!",
		Priority:     10,
	}
	err = store.CreateScenario(scenario)
	if err != nil {
		t.Fatalf("Failed to create scenario: %v", err)
	}
	if scenario.ID == "" {
		t.Fatal("Scenario ID should be generated")
	}
	if scenario.CreatedAt.IsZero() {
		t.Fatal("CreatedAt should be set")
	}

	// Retrieve the scenario
	retrieved, err := store.GetScenario(scenario.ID)
	if err != nil {
		t.Fatalf("Failed to get scenario: %v", err)
	}
	if retrieved.Pattern != "hello" {
		t.Fatalf("Expected pattern 'hello', got '%s'", retrieved.Pattern)
	}
	if retrieved.ResponseType != "text" {
		t.Fatalf("Expected response_type 'text', got '%s'", retrieved.ResponseType)
	}
	if retrieved.ResponseText != "Hi there!" {
		t.Fatalf("Expected response_text 'Hi there!', got '%s'", retrieved.ResponseText)
	}
	if retrieved.Priority != 10 {
		t.Fatalf("Expected priority 10, got %d", retrieved.Priority)
	}

	// Get non-existent scenario
	_, err = store.GetScenario("nonexistent")
	if err != sql.ErrNoRows {
		t.Fatal("Expected sql.ErrNoRows for non-existent scenario")
	}
}

func TestMatchScenario(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewAnthropicStore(db)

	// Create a catch-all scenario (empty pattern) with low priority
	catchAll := &Scenario{
		Pattern:      "",
		ResponseType: "text",
		ResponseText: "Default response",
		Priority:     1,
	}
	store.CreateScenario(catchAll)

	// Create a specific scenario with higher priority
	specific := &Scenario{
		Pattern:      "weather",
		ResponseType: "text",
		ResponseText: "It's sunny!",
		Priority:     10,
	}
	store.CreateScenario(specific)

	// Create another specific scenario with medium priority
	medium := &Scenario{
		Pattern:      "weather",
		ResponseType: "tool_use",
		ToolName:     "get_weather",
		ToolInput:    `{"location":"here"}`,
		Priority:     5,
	}
	store.CreateScenario(medium)

	// Match "weather" should return the highest priority match
	matched, err := store.MatchScenario("What's the weather like?")
	if err != nil {
		t.Fatalf("Failed to match scenario: %v", err)
	}
	if matched.ID != specific.ID {
		t.Fatalf("Expected highest priority scenario (ID %s), got %s", specific.ID, matched.ID)
	}

	// Match something that only the catch-all matches
	matched, err = store.MatchScenario("random question")
	if err != nil {
		t.Fatalf("Failed to match catch-all scenario: %v", err)
	}
	if matched.ID != catchAll.ID {
		t.Fatalf("Expected catch-all scenario (ID %s), got %s", catchAll.ID, matched.ID)
	}

	// Verify tool_use fields
	toolMatch, err := store.GetScenario(medium.ID)
	if err != nil {
		t.Fatalf("Failed to get tool scenario: %v", err)
	}
	if toolMatch.ToolName != "get_weather" {
		t.Fatalf("Expected tool_name 'get_weather', got '%s'", toolMatch.ToolName)
	}
	if toolMatch.ToolInput != `{"location":"here"}` {
		t.Fatalf("Expected tool_input, got '%s'", toolMatch.ToolInput)
	}
}

func TestListScenarios(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewAnthropicStore(db)

	// Create scenarios with different priorities
	store.CreateScenario(&Scenario{Pattern: "low", ResponseType: "text", ResponseText: "low", Priority: 1})
	store.CreateScenario(&Scenario{Pattern: "high", ResponseType: "text", ResponseText: "high", Priority: 100})
	store.CreateScenario(&Scenario{Pattern: "mid", ResponseType: "text", ResponseText: "mid", Priority: 50})

	scenarios, err := store.ListScenarios()
	if err != nil {
		t.Fatalf("Failed to list scenarios: %v", err)
	}
	if len(scenarios) != 3 {
		t.Fatalf("Expected 3 scenarios, got %d", len(scenarios))
	}

	// Should be ordered by priority DESC
	if scenarios[0].Priority != 100 {
		t.Fatalf("Expected first scenario priority 100, got %d", scenarios[0].Priority)
	}
	if scenarios[1].Priority != 50 {
		t.Fatalf("Expected second scenario priority 50, got %d", scenarios[1].Priority)
	}
	if scenarios[2].Priority != 1 {
		t.Fatalf("Expected third scenario priority 1, got %d", scenarios[2].Priority)
	}
}

func TestDeleteScenario(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewAnthropicStore(db)

	scenario := &Scenario{
		Pattern:      "to-delete",
		ResponseType: "text",
		ResponseText: "gone",
		Priority:     5,
	}
	store.CreateScenario(scenario)

	// Verify it exists
	_, err := store.GetScenario(scenario.ID)
	if err != nil {
		t.Fatalf("Scenario should exist before delete: %v", err)
	}

	// Delete it
	err = store.DeleteScenario(scenario.ID)
	if err != nil {
		t.Fatalf("Failed to delete scenario: %v", err)
	}

	// Verify it's gone
	_, err = store.GetScenario(scenario.ID)
	if err != sql.ErrNoRows {
		t.Fatal("Expected sql.ErrNoRows after deleting scenario")
	}
}

func TestCreateAndListMessages(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewAnthropicStore(db)

	// Create some messages
	for i := 0; i < 5; i++ {
		msg := &Message{
			RequestBody:  `{"messages":[{"role":"user","content":"hello"}]}`,
			ResponseType: "text",
			ScenarioID:   "scenario-1",
		}
		err := store.CreateMessage(msg)
		if err != nil {
			t.Fatalf("Failed to create message: %v", err)
		}
		if msg.ID == "" {
			t.Fatal("Message ID should be generated")
		}
		if msg.CreatedAt.IsZero() {
			t.Fatal("CreatedAt should be set")
		}
	}

	// List all messages
	messages, err := store.ListMessages(10, 0)
	if err != nil {
		t.Fatalf("Failed to list messages: %v", err)
	}
	if len(messages) != 5 {
		t.Fatalf("Expected 5 messages, got %d", len(messages))
	}

	// Test limit
	messages, err = store.ListMessages(3, 0)
	if err != nil {
		t.Fatalf("Failed to list messages with limit: %v", err)
	}
	if len(messages) != 3 {
		t.Fatalf("Expected 3 messages (limit), got %d", len(messages))
	}

	// Test offset
	messages, err = store.ListMessages(10, 3)
	if err != nil {
		t.Fatalf("Failed to list messages with offset: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages (offset 3 of 5), got %d", len(messages))
	}
}

func TestResetStore(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewAnthropicStore(db)

	// Populate both tables
	store.CreateScenario(&Scenario{Pattern: "test", ResponseType: "text", ResponseText: "hi", Priority: 1})
	store.CreateMessage(&Message{RequestBody: "{}", ResponseType: "text"})

	// Verify data exists
	scenarios, _ := store.ListScenarios()
	if len(scenarios) != 1 {
		t.Fatal("Expected 1 scenario before reset")
	}
	messages, _ := store.ListMessages(10, 0)
	if len(messages) != 1 {
		t.Fatal("Expected 1 message before reset")
	}

	// Reset
	err := store.Reset()
	if err != nil {
		t.Fatalf("Failed to reset store: %v", err)
	}

	// Verify both tables are empty
	scenarios, _ = store.ListScenarios()
	if len(scenarios) != 0 {
		t.Fatalf("Expected 0 scenarios after reset, got %d", len(scenarios))
	}
	messages, _ = store.ListMessages(10, 0)
	if len(messages) != 0 {
		t.Fatalf("Expected 0 messages after reset, got %d", len(messages))
	}
}
