// ABOUTME: SSE formatting helpers for Anthropic streaming API responses
// ABOUTME: Writes Server-Sent Events matching Anthropic's message streaming protocol

package anthropic

import (
	"encoding/json"
	"fmt"
	"io"
)

// writeSSEEvent writes a single SSE event with the given event name and JSON-marshaled data.
func writeSSEEvent(w io.Writer, event string, data interface{}) {
	jsonBytes, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, jsonBytes)
}

// writeMessageStart writes a message_start SSE event with the initial message envelope.
func writeMessageStart(w io.Writer, msgID, model string) {
	writeSSEEvent(w, "message_start", map[string]interface{}{
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

// writeContentBlockStart writes a content_block_start SSE event for a text block.
func writeContentBlockStart(w io.Writer, index int, blockType string) {
	writeSSEEvent(w, "content_block_start", map[string]interface{}{
		"index": index,
		"content_block": map[string]interface{}{
			"type": blockType,
			"text": "",
		},
	})
}

// writeToolUseBlockStart writes a content_block_start SSE event for a tool_use block.
func writeToolUseBlockStart(w io.Writer, index int, toolUseID, toolName string) {
	writeSSEEvent(w, "content_block_start", map[string]interface{}{
		"index": index,
		"content_block": map[string]interface{}{
			"type":  "tool_use",
			"id":    toolUseID,
			"name":  toolName,
			"input": map[string]interface{}{},
		},
	})
}

// writeTextDelta writes a content_block_delta SSE event with a text_delta payload.
func writeTextDelta(w io.Writer, index int, text string) {
	writeSSEEvent(w, "content_block_delta", map[string]interface{}{
		"index": index,
		"delta": map[string]interface{}{
			"type": "text_delta",
			"text": text,
		},
	})
}

// writeInputJSONDelta writes a content_block_delta SSE event with an input_json_delta payload.
func writeInputJSONDelta(w io.Writer, index int, partialJSON string) {
	writeSSEEvent(w, "content_block_delta", map[string]interface{}{
		"index": index,
		"delta": map[string]interface{}{
			"type":         "input_json_delta",
			"partial_json": partialJSON,
		},
	})
}

// writeContentBlockStop writes a content_block_stop SSE event.
func writeContentBlockStop(w io.Writer, index int) {
	writeSSEEvent(w, "content_block_stop", map[string]interface{}{
		"index": index,
	})
}

// writeMessageDelta writes a message_delta SSE event with the stop reason.
func writeMessageDelta(w io.Writer, stopReason string) {
	writeSSEEvent(w, "message_delta", map[string]interface{}{
		"delta": map[string]interface{}{
			"stop_reason": stopReason,
		},
	})
}

// writeMessageStop writes a message_stop SSE event.
func writeMessageStop(w io.Writer) {
	writeSSEEvent(w, "message_stop", map[string]interface{}{})
}
