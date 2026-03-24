// ABOUTME: Unit tests for Anthropic SSE streaming format helpers
// ABOUTME: Verifies correct event formatting for message, content block, and delta events

package anthropic

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// parseSSEEvent splits a raw SSE block into event name and parsed JSON data.
func parseSSEEvent(raw string) (event string, data map[string]interface{}, err error) {
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		if strings.HasPrefix(line, "event: ") {
			event = strings.TrimPrefix(line, "event: ")
		}
		if strings.HasPrefix(line, "data: ") {
			jsonStr := strings.TrimPrefix(line, "data: ")
			err = json.Unmarshal([]byte(jsonStr), &data)
		}
	}
	return
}

func TestWriteMessageStart(t *testing.T) {
	var buf bytes.Buffer
	writeMessageStart(&buf, "msg_123", "claude-3-opus-20240229")

	event, data, err := parseSSEEvent(buf.String())
	if err != nil {
		t.Fatalf("failed to parse SSE event: %v", err)
	}

	if event != "message_start" {
		t.Errorf("expected event 'message_start', got %q", event)
	}

	msg, ok := data["message"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected message object in data, got %v", data)
	}

	if msg["id"] != "msg_123" {
		t.Errorf("expected id 'msg_123', got %v", msg["id"])
	}
	if msg["type"] != "message" {
		t.Errorf("expected type 'message', got %v", msg["type"])
	}
	if msg["role"] != "assistant" {
		t.Errorf("expected role 'assistant', got %v", msg["role"])
	}
	if msg["model"] != "claude-3-opus-20240229" {
		t.Errorf("expected model 'claude-3-opus-20240229', got %v", msg["model"])
	}
	if msg["stop_reason"] != nil {
		t.Errorf("expected stop_reason null, got %v", msg["stop_reason"])
	}

	content, ok := msg["content"].([]interface{})
	if !ok || len(content) != 0 {
		t.Errorf("expected empty content array, got %v", msg["content"])
	}
}

func TestWriteTextContentBlock(t *testing.T) {
	var buf bytes.Buffer
	writeContentBlockStart(&buf, 0, "text")

	event, data, err := parseSSEEvent(buf.String())
	if err != nil {
		t.Fatalf("failed to parse SSE event: %v", err)
	}

	if event != "content_block_start" {
		t.Errorf("expected event 'content_block_start', got %q", event)
	}

	idx, ok := data["index"].(float64)
	if !ok || int(idx) != 0 {
		t.Errorf("expected index 0, got %v", data["index"])
	}

	cb, ok := data["content_block"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected content_block object, got %v", data)
	}
	if cb["type"] != "text" {
		t.Errorf("expected type 'text', got %v", cb["type"])
	}
	if cb["text"] != "" {
		t.Errorf("expected empty text, got %v", cb["text"])
	}
}

func TestWriteTextDelta(t *testing.T) {
	var buf bytes.Buffer
	writeTextDelta(&buf, 0, "Hello, world!")

	event, data, err := parseSSEEvent(buf.String())
	if err != nil {
		t.Fatalf("failed to parse SSE event: %v", err)
	}

	if event != "content_block_delta" {
		t.Errorf("expected event 'content_block_delta', got %q", event)
	}

	idx, ok := data["index"].(float64)
	if !ok || int(idx) != 0 {
		t.Errorf("expected index 0, got %v", data["index"])
	}

	delta, ok := data["delta"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected delta object, got %v", data)
	}
	if delta["type"] != "text_delta" {
		t.Errorf("expected type 'text_delta', got %v", delta["type"])
	}
	if delta["text"] != "Hello, world!" {
		t.Errorf("expected text 'Hello, world!', got %v", delta["text"])
	}
}

func TestWriteToolUseContentBlock(t *testing.T) {
	var buf bytes.Buffer
	writeToolUseBlockStart(&buf, 1, "toolu_abc123", "get_weather")

	event, data, err := parseSSEEvent(buf.String())
	if err != nil {
		t.Fatalf("failed to parse SSE event: %v", err)
	}

	if event != "content_block_start" {
		t.Errorf("expected event 'content_block_start', got %q", event)
	}

	idx, ok := data["index"].(float64)
	if !ok || int(idx) != 1 {
		t.Errorf("expected index 1, got %v", data["index"])
	}

	cb, ok := data["content_block"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected content_block object, got %v", data)
	}
	if cb["type"] != "tool_use" {
		t.Errorf("expected type 'tool_use', got %v", cb["type"])
	}
	if cb["id"] != "toolu_abc123" {
		t.Errorf("expected id 'toolu_abc123', got %v", cb["id"])
	}
	if cb["name"] != "get_weather" {
		t.Errorf("expected name 'get_weather', got %v", cb["name"])
	}
}

func TestWriteInputJSONDelta(t *testing.T) {
	var buf bytes.Buffer
	writeInputJSONDelta(&buf, 1, `{"location": "San`)

	event, data, err := parseSSEEvent(buf.String())
	if err != nil {
		t.Fatalf("failed to parse SSE event: %v", err)
	}

	if event != "content_block_delta" {
		t.Errorf("expected event 'content_block_delta', got %q", event)
	}

	idx, ok := data["index"].(float64)
	if !ok || int(idx) != 1 {
		t.Errorf("expected index 1, got %v", data["index"])
	}

	delta, ok := data["delta"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected delta object, got %v", data)
	}
	if delta["type"] != "input_json_delta" {
		t.Errorf("expected type 'input_json_delta', got %v", delta["type"])
	}
	if delta["partial_json"] != `{"location": "San` {
		t.Errorf("expected partial_json, got %v", delta["partial_json"])
	}
}

func TestWriteMessageStop(t *testing.T) {
	var buf bytes.Buffer

	// Write the full stop sequence: content_block_stop, message_delta, message_stop
	writeContentBlockStop(&buf, 0)
	writeMessageDelta(&buf, "end_turn")
	writeMessageStop(&buf)

	raw := buf.String()
	// Split into individual SSE events (separated by double newlines)
	events := strings.Split(strings.TrimRight(raw, "\n"), "\n\n")

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d: %v", len(events), events)
	}

	// Check content_block_stop
	ev0, data0, err := parseSSEEvent(events[0])
	if err != nil {
		t.Fatalf("failed to parse event 0: %v", err)
	}
	if ev0 != "content_block_stop" {
		t.Errorf("expected 'content_block_stop', got %q", ev0)
	}
	idx, ok := data0["index"].(float64)
	if !ok || int(idx) != 0 {
		t.Errorf("expected index 0, got %v", data0["index"])
	}

	// Check message_delta
	ev1, data1, err := parseSSEEvent(events[1])
	if err != nil {
		t.Fatalf("failed to parse event 1: %v", err)
	}
	if ev1 != "message_delta" {
		t.Errorf("expected 'message_delta', got %q", ev1)
	}
	delta, ok := data1["delta"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected delta object, got %v", data1)
	}
	if delta["stop_reason"] != "end_turn" {
		t.Errorf("expected stop_reason 'end_turn', got %v", delta["stop_reason"])
	}

	// Check message_stop
	ev2, _, err := parseSSEEvent(events[2])
	if err != nil {
		t.Fatalf("failed to parse event 2: %v", err)
	}
	if ev2 != "message_stop" {
		t.Errorf("expected 'message_stop', got %q", ev2)
	}
}
