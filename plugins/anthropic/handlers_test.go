// ABOUTME: HTTP handler tests for Anthropic plugin scenario CRUD and messages endpoints
// ABOUTME: Tests create, list, delete scenarios, plugin metadata, and SSE message streaming

package anthropic

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

	var response Scenario
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if response.ID == "" {
		t.Fatal("Scenario ID should be generated")
	}
	if response.Pattern != "hello" {
		t.Fatalf("Expected pattern 'hello', got '%s'", response.Pattern)
	}
	if response.ResponseType != "text" {
		t.Fatalf("Expected response_type 'text', got '%s'", response.ResponseType)
	}
	if response.ResponseText != "Hi there!" {
		t.Fatalf("Expected response_text 'Hi there!', got '%s'", response.ResponseText)
	}
	if response.Priority != 10 {
		t.Fatalf("Expected priority 10, got %d", response.Priority)
	}
}

func TestListScenariosHandler(t *testing.T) {
	plugin := setupTestPlugin(t)

	// Seed 2 scenarios directly via store
	plugin.store.CreateScenario(&Scenario{
		Pattern:      "hello",
		ResponseType: "text",
		ResponseText: "Hi!",
		Priority:     10,
	})
	plugin.store.CreateScenario(&Scenario{
		Pattern:      "",
		ResponseType: "text",
		ResponseText: "Default",
		Priority:     0,
	})

	req := newRequestWithParams("GET", "/v1/scenarios", nil, nil)
	w := httptest.NewRecorder()
	plugin.listScenarios(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var scenarios []*Scenario
	if err := json.NewDecoder(w.Body).Decode(&scenarios); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(scenarios) != 2 {
		t.Fatalf("Expected 2 scenarios, got %d", len(scenarios))
	}
}

func TestDeleteScenarioHandler(t *testing.T) {
	plugin := setupTestPlugin(t)

	// Create a scenario
	scenario := &Scenario{
		Pattern:      "to-delete",
		ResponseType: "text",
		ResponseText: "gone",
		Priority:     5,
	}
	plugin.store.CreateScenario(scenario)

	req := newRequestWithParams("DELETE", "/v1/scenarios/"+scenario.ID, nil, map[string]string{
		"id": scenario.ID,
	})
	w := httptest.NewRecorder()
	plugin.deleteScenarioHandler(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("Expected 204, got %d: %s", w.Code, w.Body.String())
	}

	// Verify it's gone
	_, err := plugin.store.GetScenario(scenario.ID)
	if err != sql.ErrNoRows {
		t.Fatal("Expected scenario to be deleted")
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
		t.Fatalf("Expected status 'healthy', got '%s'", health.Status)
	}
}

func sendMessage(t *testing.T, plugin *AnthropicPlugin, body map[string]interface{}) *httptest.ResponseRecorder {
	t.Helper()
	bodyJSON, _ := json.Marshal(body)
	req := newRequestWithParams("POST", "/v1/messages", bodyJSON, nil)
	w := httptest.NewRecorder()
	plugin.handleMessages(w, req)
	return w
}

func TestHandleMessagesTextResponse(t *testing.T) {
	plugin := setupTestPlugin(t)

	// Seed a "hello" scenario
	plugin.store.CreateScenario(&Scenario{
		Pattern:      "hello",
		ResponseType: "text",
		ResponseText: "Hi there!",
		Priority:     10,
	})

	w := sendMessage(t, plugin, map[string]interface{}{
		"model":  "claude-sonnet-4-20250514",
		"stream": true,
		"messages": []map[string]interface{}{
			{"role": "user", "content": "hello world"},
		},
	})

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Fatalf("Expected Content-Type text/event-stream, got %s", contentType)
	}

	body := w.Body.String()

	// Verify SSE events are present
	if !strings.Contains(body, "event: message_start") {
		t.Fatal("Missing message_start event")
	}
	if !strings.Contains(body, "event: content_block_start") {
		t.Fatal("Missing content_block_start event")
	}
	if !strings.Contains(body, "event: content_block_delta") {
		t.Fatal("Missing content_block_delta event")
	}
	if !strings.Contains(body, "Hi there!") {
		t.Fatal("Response should contain scenario text")
	}
	if !strings.Contains(body, "event: content_block_stop") {
		t.Fatal("Missing content_block_stop event")
	}
	if !strings.Contains(body, "event: message_delta") {
		t.Fatal("Missing message_delta event")
	}
	if !strings.Contains(body, `"stop_reason":"end_turn"`) {
		t.Fatal("Missing end_turn stop reason")
	}
	if !strings.Contains(body, "event: message_stop") {
		t.Fatal("Missing message_stop event")
	}
}

func TestHandleMessagesToolUseResponse(t *testing.T) {
	plugin := setupTestPlugin(t)

	// Seed a tool_use scenario
	plugin.store.CreateScenario(&Scenario{
		Pattern:      "weather",
		ResponseType: "tool_use",
		ToolName:     "get_weather",
		ToolInput:    `{"location":"San Francisco"}`,
		Priority:     10,
	})

	w := sendMessage(t, plugin, map[string]interface{}{
		"model":  "claude-sonnet-4-20250514",
		"stream": true,
		"messages": []map[string]interface{}{
			{"role": "user", "content": "what's the weather"},
		},
	})

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()

	if !strings.Contains(body, "get_weather") {
		t.Fatal("Response should contain tool name")
	}
	if !strings.Contains(body, `"stop_reason":"tool_use"`) {
		t.Fatal("Missing tool_use stop reason")
	}
	if !strings.Contains(body, "San Francisco") {
		t.Fatal("Response should contain tool input")
	}
}

func TestHandleMessagesToolResultFollowUp(t *testing.T) {
	plugin := setupTestPlugin(t)

	w := sendMessage(t, plugin, map[string]interface{}{
		"model":  "claude-sonnet-4-20250514",
		"stream": true,
		"messages": []map[string]interface{}{
			{"role": "user", "content": "what's the weather"},
			{"role": "assistant", "content": []map[string]interface{}{
				{"type": "tool_use", "id": "toolu_123", "name": "get_weather", "input": map[string]interface{}{"location": "SF"}},
			}},
			{"role": "user", "content": []map[string]interface{}{
				{"type": "tool_result", "tool_use_id": "toolu_123", "content": "72 degrees and sunny"},
			}},
		},
	})

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()

	// Should respond with text summary of tool result
	if !strings.Contains(body, "event: message_start") {
		t.Fatal("Missing message_start event")
	}
	if !strings.Contains(body, `"stop_reason":"end_turn"`) {
		t.Fatal("Missing end_turn stop reason for tool result follow-up")
	}
}

func TestHandleMessagesCatchAll(t *testing.T) {
	plugin := setupTestPlugin(t)

	// Seed only a catch-all scenario (empty pattern)
	plugin.store.CreateScenario(&Scenario{
		Pattern:      "",
		ResponseType: "text",
		ResponseText: "I can help with that.",
		Priority:     0,
	})

	w := sendMessage(t, plugin, map[string]interface{}{
		"model":  "claude-sonnet-4-20250514",
		"stream": true,
		"messages": []map[string]interface{}{
			{"role": "user", "content": "something completely random xyz123"},
		},
	})

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()
	if !strings.Contains(body, "I can help with that.") {
		t.Fatal("Response should contain catch-all text")
	}
}

func TestHandleMessagesLogsRequest(t *testing.T) {
	plugin := setupTestPlugin(t)

	// Seed a scenario so we get a match
	plugin.store.CreateScenario(&Scenario{
		Pattern:      "log-test",
		ResponseType: "text",
		ResponseText: "Logged!",
		Priority:     10,
	})

	w := sendMessage(t, plugin, map[string]interface{}{
		"model":  "claude-sonnet-4-20250514",
		"stream": true,
		"messages": []map[string]interface{}{
			{"role": "user", "content": "log-test message"},
		},
	})

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the request was logged
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

	// No scenarios seeded at all
	w := sendMessage(t, plugin, map[string]interface{}{
		"model":  "claude-sonnet-4-20250514",
		"stream": true,
		"messages": []map[string]interface{}{
			{"role": "user", "content": "anything"},
		},
	})

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	body := w.Body.String()

	// Should still produce valid SSE with fallback text
	if !strings.Contains(body, "event: message_start") {
		t.Fatal("Missing message_start event in fallback response")
	}
	if !strings.Contains(body, "event: message_stop") {
		t.Fatal("Missing message_stop event in fallback response")
	}
	if !strings.Contains(body, `"stop_reason":"end_turn"`) {
		t.Fatal("Missing end_turn stop reason in fallback response")
	}
}
