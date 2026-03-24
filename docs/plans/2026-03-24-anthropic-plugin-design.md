# Anthropic API Mock Plugin Design

## Overview

Add an `anthropic` plugin to ISH that mocks the Anthropic `/v1/messages` endpoint with streaming SSE responses. The plugin is generic and reusable — consumers seed their own scenarios via API to define how the mock responds to specific inputs.

## Motivation

Jeff needs integration tests that exercise the full TUI without hitting the real Anthropic API. Other projects may also need a local Anthropic mock with custom tool scenarios.

## Data Model

### `anthropic_scenarios` table

Maps input patterns to canned responses.

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT PK | UUID |
| `pattern` | TEXT | Substring to match in the last user message (empty = catch-all) |
| `response_type` | TEXT | `"text"` or `"tool_use"` |
| `response_text` | TEXT | Text to stream back (for text responses) |
| `tool_name` | TEXT | Tool name (for tool_use responses) |
| `tool_input` | TEXT | JSON string of tool params (for tool_use responses) |
| `priority` | INTEGER | Higher = checked first |
| `created_at` | TIMESTAMP | When the scenario was created |

### `anthropic_messages` table

Logs each request for test verification.

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT PK | UUID |
| `request_body` | TEXT | Full JSON request body |
| `response_type` | TEXT | What kind of response was sent |
| `scenario_id` | TEXT | Which scenario matched (nullable) |
| `created_at` | TIMESTAMP | When the request was received |

## Routes

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/messages` | Main Anthropic API endpoint (streaming SSE) |
| `POST` | `/v1/scenarios` | Add a custom scenario (test setup) |
| `GET` | `/v1/scenarios` | List current scenarios |
| `DELETE` | `/v1/scenarios/{id}` | Remove a scenario |

## SSE Streaming Flow

When `POST /v1/messages` is called with `"stream": true`:

1. Parse request body, extract the last user message text
2. Check if the last message has `role: "tool"` → respond with a text summary of the tool result
3. Otherwise, match against scenarios by pattern (highest priority first, substring match)
4. Stream the matched response as Anthropic SSE events:

### Text response events:
```
event: message_start
data: {"type":"message_start","message":{"id":"msg_xxx","type":"message","role":"assistant","content":[],"model":"claude-sonnet-4-20250514","stop_reason":null}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello! How can I help?"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"}}

event: message_stop
data: {"type":"message_stop"}
```

### Tool use response events:
```
event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_xxx","name":"get_current_time","input":{}}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{}"}}

event: content_block_stop
data: {"type":"content_block_stop","index":0}
```

## File Structure

```
plugins/anthropic/
├── plugin.go      # Plugin struct, interface impl, route registration
├── store.go       # DB tables, scenario and message CRUD
├── handlers.go    # POST /v1/messages handler with SSE streaming
├── scenarios.go   # Scenario CRUD handlers (POST/GET/DELETE /v1/scenarios)
├── sse.go         # SSE event formatting helpers
├── schema.go      # Admin UI schema
├── seed.go        # Generic default scenarios
```

## Seed Data

`Seed()` creates generic defaults only:

| Pattern | Type | Response |
|---------|------|----------|
| `"hello"` | text | `"Hello! How can I help you today?"` |
| (empty — catch-all) | text | Echoes back acknowledgment of the user's input |

Consumers (like Jeff) add their own tool-specific scenarios via `POST /v1/scenarios` during test setup.

## Tool Result Handling

When the conversation history contains a message with `role: "tool"`, the mock responds with a text message summarizing what "happened" — e.g., `"I've completed the tool call. The result was: <tool result content>"`.

## Integration

- Add `_ "github.com/2389/ish/plugins/anthropic"` import to `cmd/ish/main.go`
- Implements `Plugin`, `DatabasePlugin`, `ResettablePlugin`, `DataProvider` interfaces
- Token validation accepts any bearer token (like Discord's approach)
