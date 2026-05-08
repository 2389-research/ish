// ABOUTME: E2E integration tests for the Anthropic Messages API plugin.
// ABOUTME: Tests scenario CRUD plus streaming and non-streaming text/tool_use responses.

package e2e_test

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAnthropicScenarioCRUD(t *testing.T) {
	ts := StartTestServer(t)
	defer ts.Close()

	// Step 1: Create a scenario
	scenarioReq := map[string]interface{}{
		"pattern":       "hello",
		"response_type": "text",
		"response_text": "Hi from ISH!",
		"priority":      10,
	}

	resp := ts.POST(t, "/v1/scenarios", scenarioReq)
	AssertStatusCode(t, resp, 201)

	var created map[string]interface{}
	DecodeJSON(t, resp, &created)

	scenarioID, ok := created["id"].(string)
	if !ok || scenarioID == "" {
		t.Fatal("expected scenario ID in response")
	}

	// Step 2: List scenarios and verify at least 1 exists
	resp = ts.GET(t, "/v1/scenarios")
	AssertStatusCode(t, resp, 200)

	var scenarios []map[string]interface{}
	DecodeJSON(t, resp, &scenarios)

	if len(scenarios) < 1 {
		t.Fatalf("expected at least 1 scenario, got %d", len(scenarios))
	}

	// Step 3: Delete the scenario
	resp = ts.DELETE(t, "/v1/scenarios/"+scenarioID)
	AssertStatusCode(t, resp, 204)
}

func TestAnthropicStreamingTextResponse(t *testing.T) {
	ts := StartTestServer(t)
	defer ts.Close()

	// Seed a scenario that matches "greetings"
	scenarioReq := map[string]interface{}{
		"pattern":       "greetings",
		"response_type": "text",
		"response_text": "Hello, friend!",
		"priority":      10,
	}

	resp := ts.POST(t, "/v1/scenarios", scenarioReq)
	AssertStatusCode(t, resp, 201)

	// Send a messages request that should match the scenario
	messagesReq := map[string]interface{}{
		"model":  "claude-sonnet-4-20250514",
		"stream": true,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": "greetings friend",
			},
		},
	}

	resp = ts.POST(t, "/v1/messages", messagesReq)
	AssertStatusCode(t, resp, 200)

	// Verify Content-Type is text/event-stream
	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/event-stream" {
		t.Fatalf("expected Content-Type text/event-stream, got %s", contentType)
	}

	body := ReadBody(t, resp)

	// Verify SSE event structure
	if !strings.Contains(body, "event: message_start") {
		t.Error("expected body to contain 'event: message_start'")
	}
	if !strings.Contains(body, "Hello, friend!") {
		t.Error("expected body to contain response text 'Hello, friend!'")
	}
	if !strings.Contains(body, "event: message_stop") {
		t.Error("expected body to contain 'event: message_stop'")
	}
}

func TestAnthropicStreamingToolUseResponse(t *testing.T) {
	ts := StartTestServer(t)
	defer ts.Close()

	// Seed a tool_use scenario
	scenarioReq := map[string]interface{}{
		"pattern":       "what time",
		"response_type": "tool_use",
		"tool_name":     "get_current_time",
		"tool_input":    "{}",
		"priority":      10,
	}

	resp := ts.POST(t, "/v1/scenarios", scenarioReq)
	AssertStatusCode(t, resp, 201)

	// Send a messages request that should match the tool_use scenario
	messagesReq := map[string]interface{}{
		"model":  "claude-sonnet-4-20250514",
		"stream": true,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": "what time is it?",
			},
		},
	}

	resp = ts.POST(t, "/v1/messages", messagesReq)
	AssertStatusCode(t, resp, 200)

	body := ReadBody(t, resp)

	// Verify tool_use response structure
	if !strings.Contains(body, `"type":"tool_use"`) {
		t.Error("expected body to contain '\"type\":\"tool_use\"'")
	}
	if !strings.Contains(body, "get_current_time") {
		t.Error("expected body to contain tool name 'get_current_time'")
	}
	if !strings.Contains(body, `"stop_reason":"tool_use"`) {
		t.Error("expected body to contain '\"stop_reason\":\"tool_use\"'")
	}
}

func TestAnthropicNonStreamingTextResponse(t *testing.T) {
	ts := StartTestServer(t)
	defer ts.Close()

	scenarioReq := map[string]interface{}{
		"pattern":       "greetings",
		"response_type": "text",
		"response_text": "Hello, friend!",
		"priority":      10,
	}

	resp := ts.POST(t, "/v1/scenarios", scenarioReq)
	AssertStatusCode(t, resp, 201)

	messagesReq := map[string]interface{}{
		"model":  "claude-sonnet-4-20250514",
		"stream": false,
		"messages": []map[string]interface{}{
			{"role": "user", "content": "greetings friend"},
		},
	}

	resp = ts.POST(t, "/v1/messages", messagesReq)
	AssertStatusCode(t, resp, 200)

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("expected Content-Type application/json, got %s", contentType)
	}

	var msg map[string]interface{}
	DecodeJSON(t, resp, &msg)

	if msg["type"] != "message" {
		t.Errorf("expected type 'message', got %v", msg["type"])
	}
	if msg["role"] != "assistant" {
		t.Errorf("expected role 'assistant', got %v", msg["role"])
	}
	if msg["stop_reason"] != "end_turn" {
		t.Errorf("expected stop_reason 'end_turn', got %v", msg["stop_reason"])
	}

	content, ok := msg["content"].([]interface{})
	if !ok || len(content) != 1 {
		t.Fatalf("expected content array of length 1, got %v", msg["content"])
	}
	block := content[0].(map[string]interface{})
	if block["type"] != "text" {
		t.Errorf("expected content[0].type 'text', got %v", block["type"])
	}
	if block["text"] != "Hello, friend!" {
		t.Errorf("expected response text 'Hello, friend!', got %v", block["text"])
	}
}

func TestAnthropicNonStreamingToolUseResponse(t *testing.T) {
	ts := StartTestServer(t)
	defer ts.Close()

	scenarioReq := map[string]interface{}{
		"pattern":       "what time",
		"response_type": "tool_use",
		"tool_name":     "get_current_time",
		"tool_input":    `{"timezone":"UTC"}`,
		"priority":      10,
	}

	resp := ts.POST(t, "/v1/scenarios", scenarioReq)
	AssertStatusCode(t, resp, 201)

	messagesReq := map[string]interface{}{
		"model":  "claude-sonnet-4-20250514",
		"stream": false,
		"messages": []map[string]interface{}{
			{"role": "user", "content": "what time is it?"},
		},
	}

	resp = ts.POST(t, "/v1/messages", messagesReq)
	AssertStatusCode(t, resp, 200)

	var msg map[string]interface{}
	DecodeJSON(t, resp, &msg)

	if msg["stop_reason"] != "tool_use" {
		t.Errorf("expected stop_reason 'tool_use', got %v", msg["stop_reason"])
	}

	content, ok := msg["content"].([]interface{})
	if !ok || len(content) != 1 {
		t.Fatalf("expected content array of length 1, got %v", msg["content"])
	}
	block := content[0].(map[string]interface{})
	if block["type"] != "tool_use" {
		t.Errorf("expected content[0].type 'tool_use', got %v", block["type"])
	}
	if block["name"] != "get_current_time" {
		t.Errorf("expected tool name 'get_current_time', got %v", block["name"])
	}

	input, ok := block["input"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected input to be a JSON object, got %v (type %T)", block["input"], block["input"])
	}
	if input["timezone"] != "UTC" {
		t.Errorf("expected input.timezone 'UTC', got %v", input["timezone"])
	}
}

// TestAnthropicDefaultsToNonStreaming verifies a request without a `stream`
// field returns JSON, matching real Anthropic where stream defaults to false.
func TestAnthropicDefaultsToNonStreaming(t *testing.T) {
	ts := StartTestServer(t)
	defer ts.Close()

	scenarioReq := map[string]interface{}{
		"pattern":       "hi",
		"response_type": "text",
		"response_text": "Hi!",
		"priority":      10,
	}
	resp := ts.POST(t, "/v1/scenarios", scenarioReq)
	AssertStatusCode(t, resp, 201)

	messagesReq := map[string]interface{}{
		"model": "claude-sonnet-4-20250514",
		"messages": []map[string]interface{}{
			{"role": "user", "content": "hi"},
		},
	}

	resp = ts.POST(t, "/v1/messages", messagesReq)
	AssertStatusCode(t, resp, 200)

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("expected Content-Type application/json by default, got %s", contentType)
	}

	body := ReadBody(t, resp)

	// Should be a complete JSON object, not an SSE stream.
	if strings.Contains(body, "event: message_start") {
		t.Fatal("expected JSON body, got SSE stream")
	}
	var msg map[string]interface{}
	if err := json.Unmarshal([]byte(body), &msg); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if msg["type"] != "message" {
		t.Errorf("expected type 'message', got %v", msg["type"])
	}
}
