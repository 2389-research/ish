// ABOUTME: HTTP handler for POST /v1/messages with SSE streaming and JSON non-streaming transports
// ABOUTME: Matches user messages against scenarios and returns Anthropic-format responses

package anthropic

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// messagesRequest represents the incoming Anthropic Messages API request body.
type messagesRequest struct {
	Model    string           `json:"model"`
	Stream   bool             `json:"stream"`
	Messages []messageContent `json:"messages"`
}

// messageContent represents a single message in the conversation.
type messageContent struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

// handleMessages handles POST /v1/messages. It returns SSE events when
// req.Stream is true and a single JSON Message body otherwise (matching real
// Anthropic, where stream defaults to false).
func (p *AnthropicPlugin) handleMessages(w http.ResponseWriter, r *http.Request) {
	if p.store == nil {
		writeError(w, http.StatusInternalServerError, "Plugin not initialized")
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}

	var req messagesRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(req.Messages) == 0 {
		writeError(w, http.StatusBadRequest, "Messages array must not be empty")
		return
	}

	if req.Model == "" {
		req.Model = "claude-sonnet-4-20250514"
	}

	lastMsg := req.Messages[len(req.Messages)-1]

	// If last message contains a tool result, respond with a text summary
	if containsToolResult(lastMsg) {
		toolContent := extractToolResultContent(lastMsg)
		responseText := "Based on the tool result: " + toolContent
		if req.Stream {
			p.streamTextResponse(w, req.Model, responseText, string(bodyBytes), "")
		} else {
			p.respondTextNonStream(w, req.Model, responseText, string(bodyBytes), "")
		}
		return
	}

	// Extract user text and match against scenarios
	userText := extractUserText(lastMsg)

	scenario, err := p.store.MatchScenario(userText)
	if err != nil {
		if err == sql.ErrNoRows {
			// No scenarios match; use hardcoded fallback
			fallbackText := "I understand your request. How can I help you?"
			if req.Stream {
				p.streamTextResponse(w, req.Model, fallbackText, string(bodyBytes), "")
			} else {
				p.respondTextNonStream(w, req.Model, fallbackText, string(bodyBytes), "")
			}
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to match scenario")
		return
	}

	if scenario.ResponseType == "tool_use" {
		if req.Stream {
			p.streamToolUseResponse(w, req.Model, scenario, string(bodyBytes))
		} else {
			p.respondToolUseNonStream(w, req.Model, scenario, string(bodyBytes))
		}
	} else {
		if req.Stream {
			p.streamTextResponse(w, req.Model, scenario.ResponseText, string(bodyBytes), scenario.ID)
		} else {
			p.respondTextNonStream(w, req.Model, scenario.ResponseText, string(bodyBytes), scenario.ID)
		}
	}
}

// streamTextResponse streams a text response as SSE events and logs the request.
func (p *AnthropicPlugin) streamTextResponse(w http.ResponseWriter, model, text, requestBody, scenarioID string) {
	p.store.CreateMessage(&Message{
		RequestBody:  requestBody,
		ResponseType: "text",
		ScenarioID:   scenarioID,
	})

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	msgID := "msg_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:12]

	writeMessageStart(w, msgID, model)
	writeContentBlockStart(w, 0, "text")
	writeTextDelta(w, 0, text)
	writeContentBlockStop(w, 0)
	writeMessageDelta(w, "end_turn")
	writeMessageStop(w)

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// streamToolUseResponse streams a tool_use response as SSE events and logs the request.
func (p *AnthropicPlugin) streamToolUseResponse(w http.ResponseWriter, model string, scenario *Scenario, requestBody string) {
	p.store.CreateMessage(&Message{
		RequestBody:  requestBody,
		ResponseType: "tool_use",
		ScenarioID:   scenario.ID,
	})

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	msgID := "msg_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:12]
	toolUseID := "toolu_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:12]

	writeMessageStart(w, msgID, model)
	writeToolUseBlockStart(w, 0, toolUseID, scenario.ToolName)
	writeInputJSONDelta(w, 0, scenario.ToolInput)
	writeContentBlockStop(w, 0)
	writeMessageDelta(w, "tool_use")
	writeMessageStop(w)

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// respondTextNonStream returns a single JSON Message body with a text content
// block. Mirrors streamTextResponse but for stream:false clients.
func (p *AnthropicPlugin) respondTextNonStream(w http.ResponseWriter, model, text, requestBody, scenarioID string) {
	p.store.CreateMessage(&Message{
		RequestBody:  requestBody,
		ResponseType: "text",
		ScenarioID:   scenarioID,
	})

	msgID := "msg_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:12]

	body := map[string]interface{}{
		"id":   msgID,
		"type": "message",
		"role": "assistant",
		"content": []map[string]interface{}{
			{"type": "text", "text": text},
		},
		"model":         model,
		"stop_reason":   "end_turn",
		"stop_sequence": nil,
		"usage": map[string]interface{}{
			"input_tokens":  0,
			"output_tokens": 0,
		},
	}
	writeJSON(w, body)
}

// respondToolUseNonStream returns a single JSON Message body whose content
// array holds a tool_use block. Mirrors streamToolUseResponse but for
// stream:false clients.
func (p *AnthropicPlugin) respondToolUseNonStream(w http.ResponseWriter, model string, scenario *Scenario, requestBody string) {
	p.store.CreateMessage(&Message{
		RequestBody:  requestBody,
		ResponseType: "tool_use",
		ScenarioID:   scenario.ID,
	})

	msgID := "msg_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:12]
	toolUseID := "toolu_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:12]

	// Scenario.ToolInput is stored as a JSON string; emit it as a JSON object
	// rather than a string so the response matches real Anthropic.
	var toolInput interface{} = map[string]interface{}{}
	if scenario.ToolInput != "" {
		var parsed interface{}
		if err := json.Unmarshal([]byte(scenario.ToolInput), &parsed); err == nil {
			toolInput = parsed
		}
	}

	body := map[string]interface{}{
		"id":   msgID,
		"type": "message",
		"role": "assistant",
		"content": []map[string]interface{}{
			{
				"type":  "tool_use",
				"id":    toolUseID,
				"name":  scenario.ToolName,
				"input": toolInput,
			},
		},
		"model":         model,
		"stop_reason":   "tool_use",
		"stop_sequence": nil,
		"usage": map[string]interface{}{
			"input_tokens":  0,
			"output_tokens": 0,
		},
	}
	writeJSON(w, body)
}

// containsToolResult checks if the message is a user message containing a tool_result block.
func containsToolResult(msg messageContent) bool {
	if msg.Role != "user" {
		return false
	}

	var blocks []map[string]interface{}
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		return false
	}

	for _, block := range blocks {
		if block["type"] == "tool_result" {
			return true
		}
	}
	return false
}

// extractToolResultContent pulls the content string from the first tool_result block.
func extractToolResultContent(msg messageContent) string {
	var blocks []map[string]interface{}
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		return ""
	}

	for _, block := range blocks {
		if block["type"] == "tool_result" {
			if content, ok := block["content"].(string); ok {
				return content
			}
		}
	}
	return ""
}

// extractUserText gets the text from the last user message. It handles both
// plain string content and array-of-blocks content formats.
func extractUserText(msg messageContent) string {
	// Try string content first
	var text string
	if err := json.Unmarshal(msg.Content, &text); err == nil {
		return text
	}

	// Try array-of-blocks content
	var blocks []map[string]interface{}
	if err := json.Unmarshal(msg.Content, &blocks); err == nil {
		for _, block := range blocks {
			if block["type"] == "text" {
				if t, ok := block["text"].(string); ok {
					return t
				}
			}
		}
	}

	return ""
}
