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
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	// Restrict to one connection so all queries share the same in-memory database.
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

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

// assertNonStreamMessageEnvelope checks the common JSON envelope fields
// returned by stream:false responses.
func assertNonStreamMessageEnvelope(t *testing.T, w *httptest.ResponseRecorder, expectedModel string) map[string]interface{} {
	t.Helper()

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("Expected Content-Type application/json, got %q", contentType)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	if id, _ := resp["id"].(string); !strings.HasPrefix(id, "msg_") {
		t.Fatalf("Expected id to start with 'msg_', got %v", resp["id"])
	}
	if resp["type"] != "message" {
		t.Fatalf("Expected type 'message', got %v", resp["type"])
	}
	if resp["role"] != "assistant" {
		t.Fatalf("Expected role 'assistant', got %v", resp["role"])
	}
	if resp["model"] != expectedModel {
		t.Fatalf("Expected model %q, got %v", expectedModel, resp["model"])
	}
	if _, ok := resp["usage"].(map[string]interface{}); !ok {
		t.Fatalf("Expected usage object, got %v", resp["usage"])
	}

	return resp
}

func TestHandleMessagesNonStreamText(t *testing.T) {
	plugin := setupTestPlugin(t)

	plugin.store.CreateScenario(&Scenario{
		Pattern:      "hello",
		ResponseType: "text",
		ResponseText: "Hi there!",
		Priority:     10,
	})

	w := sendMessage(t, plugin, map[string]interface{}{
		"model":  "claude-sonnet-4-20250514",
		"stream": false,
		"messages": []map[string]interface{}{
			{"role": "user", "content": "hello world"},
		},
	})

	resp := assertNonStreamMessageEnvelope(t, w, "claude-sonnet-4-20250514")

	if resp["stop_reason"] != "end_turn" {
		t.Fatalf("Expected stop_reason 'end_turn', got %v", resp["stop_reason"])
	}

	content, ok := resp["content"].([]interface{})
	if !ok || len(content) != 1 {
		t.Fatalf("Expected content array of length 1, got %v", resp["content"])
	}
	block := content[0].(map[string]interface{})
	if block["type"] != "text" {
		t.Fatalf("Expected content[0].type 'text', got %v", block["type"])
	}
	if block["text"] != "Hi there!" {
		t.Fatalf("Expected scenario text, got %v", block["text"])
	}
}

func TestHandleMessagesNonStreamDefaultsToJSON(t *testing.T) {
	// Real Anthropic treats a missing `stream` field as false. Verify the
	// mock matches that behavior.
	plugin := setupTestPlugin(t)

	plugin.store.CreateScenario(&Scenario{
		Pattern:      "hello",
		ResponseType: "text",
		ResponseText: "Hi there!",
		Priority:     10,
	})

	w := sendMessage(t, plugin, map[string]interface{}{
		"model": "claude-sonnet-4-20250514",
		"messages": []map[string]interface{}{
			{"role": "user", "content": "hello world"},
		},
	})

	assertNonStreamMessageEnvelope(t, w, "claude-sonnet-4-20250514")
}

func TestHandleMessagesNonStreamToolUse(t *testing.T) {
	plugin := setupTestPlugin(t)

	plugin.store.CreateScenario(&Scenario{
		Pattern:      "weather",
		ResponseType: "tool_use",
		ToolName:     "get_weather",
		ToolInput:    `{"location":"San Francisco"}`,
		Priority:     10,
	})

	w := sendMessage(t, plugin, map[string]interface{}{
		"model":  "claude-sonnet-4-20250514",
		"stream": false,
		"messages": []map[string]interface{}{
			{"role": "user", "content": "what's the weather"},
		},
	})

	resp := assertNonStreamMessageEnvelope(t, w, "claude-sonnet-4-20250514")

	if resp["stop_reason"] != "tool_use" {
		t.Fatalf("Expected stop_reason 'tool_use', got %v", resp["stop_reason"])
	}

	content, ok := resp["content"].([]interface{})
	if !ok || len(content) != 1 {
		t.Fatalf("Expected content array of length 1, got %v", resp["content"])
	}
	block := content[0].(map[string]interface{})
	if block["type"] != "tool_use" {
		t.Fatalf("Expected content[0].type 'tool_use', got %v", block["type"])
	}
	if id, _ := block["id"].(string); !strings.HasPrefix(id, "toolu_") {
		t.Fatalf("Expected tool_use id to start with 'toolu_', got %v", block["id"])
	}
	if block["name"] != "get_weather" {
		t.Fatalf("Expected tool name 'get_weather', got %v", block["name"])
	}

	input, ok := block["input"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected input to be a JSON object, got %v (type %T)", block["input"], block["input"])
	}
	if input["location"] != "San Francisco" {
		t.Fatalf("Expected input.location 'San Francisco', got %v", input["location"])
	}
}

func TestHandleMessagesNonStreamToolResultFollowUp(t *testing.T) {
	plugin := setupTestPlugin(t)

	w := sendMessage(t, plugin, map[string]interface{}{
		"model":  "claude-sonnet-4-20250514",
		"stream": false,
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

	resp := assertNonStreamMessageEnvelope(t, w, "claude-sonnet-4-20250514")
	if resp["stop_reason"] != "end_turn" {
		t.Fatalf("Expected stop_reason 'end_turn', got %v", resp["stop_reason"])
	}

	content := resp["content"].([]interface{})
	block := content[0].(map[string]interface{})
	if !strings.Contains(block["text"].(string), "72 degrees and sunny") {
		t.Fatalf("Expected tool result text in response, got %v", block["text"])
	}
}

func TestHandleMessagesNonStreamFallback(t *testing.T) {
	plugin := setupTestPlugin(t)

	// No scenarios seeded; non-streaming should still produce a valid Message.
	w := sendMessage(t, plugin, map[string]interface{}{
		"model":  "claude-sonnet-4-20250514",
		"stream": false,
		"messages": []map[string]interface{}{
			{"role": "user", "content": "anything"},
		},
	})

	resp := assertNonStreamMessageEnvelope(t, w, "claude-sonnet-4-20250514")
	if resp["stop_reason"] != "end_turn" {
		t.Fatalf("Expected stop_reason 'end_turn', got %v", resp["stop_reason"])
	}
	content := resp["content"].([]interface{})
	if len(content) != 1 {
		t.Fatalf("Expected one content block, got %d", len(content))
	}
}
