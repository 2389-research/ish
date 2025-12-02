# ISH v2 Roadmap

Future enhancements planned for ISH (fake Google API server).

## Chaos Mode

Environment variables to inject failures for testing client resilience:

| Variable | Example | Effect |
|----------|---------|--------|
| `ISH_RATE_LIMIT_PERCENT` | `10` | 10% of requests return HTTP 429 |
| `ISH_RANDOM_500_PERCENT` | `2` | 2% of requests return HTTP 500 |
| `ISH_LATENCY_MS` | `100` | Add 100ms artificial latency |
| `ISH_LATENCY_JITTER_MS` | `50` | Add 0-50ms random jitter |

### Implementation Notes

- Middleware-based, applied before handlers
- Configurable per-endpoint or global
- Log when chaos is triggered for debugging

## Eventual Consistency Simulation

Simulate Google's distributed system behavior:

- Insert operations succeed immediately
- Inserted items don't appear in list queries for N milliseconds
- Configurable via `ISH_CONSISTENCY_DELAY_MS`

### Use Cases

- Test polling/retry logic in clients
- Verify UI handles "just created but not visible" states
- Stress test eventual consistency assumptions

## Write Operations

### Gmail
- `POST /gmail/v1/users/{userId}/messages/send` - Store message locally
- `POST /gmail/v1/users/{userId}/messages/{id}/modify` - Update labels
- `DELETE /gmail/v1/users/{userId}/messages/{id}` - Soft delete

### Calendar
- `POST /calendar/v3/calendars/{calendarId}/events` - Create event
- `PUT /calendar/v3/calendars/{calendarId}/events/{eventId}` - Update event
- `DELETE /calendar/v3/calendars/{calendarId}/events/{eventId}` - Delete event

### People
- `POST /people/v1/people:createContact` - Create contact
- `PATCH /people/v1/{resourceName}:updateContact` - Update contact
- `DELETE /people/v1/{resourceName}:deleteContact` - Delete contact

## Record/Replay Mode

Proxy mode for capturing real Google API traffic:

### Record Mode
```bash
ISH_MODE=record ISH_UPSTREAM=https://www.googleapis.com ish serve
```

- Forward requests to real Google
- Store request/response pairs with timestamps
- Save to `recordings/` directory as JSON

### Replay Mode
```bash
ISH_MODE=replay ISH_RECORDING_DIR=./recordings ish serve
```

- Match incoming requests to recorded responses
- Deterministic testing with real data shapes
- Useful for CI/CD without Google credentials

## Batch Operations

Support Google's batch endpoint pattern:

- `POST /batch` - Multiple operations in single request
- Multipart request body
- Individual status codes per operation

## Watch/Push Notifications

Webhook support for change notifications:

- `POST /gmail/v1/users/{userId}/watch` - Start watching
- `POST /gmail/v1/users/{userId}/stop` - Stop watching
- Configurable webhook delivery to localhost endpoints

## Labels Management

Full Gmail labels API:

- `GET /gmail/v1/users/{userId}/labels` - List labels
- `POST /gmail/v1/users/{userId}/labels` - Create label
- `PATCH /gmail/v1/users/{userId}/labels/{id}` - Update label
- `DELETE /gmail/v1/users/{userId}/labels/{id}` - Delete label

## ACL Support

Calendar sharing and permissions:

- `GET /calendar/v3/calendars/{calendarId}/acl` - List ACLs
- `POST /calendar/v3/calendars/{calendarId}/acl` - Add ACL
- `DELETE /calendar/v3/calendars/{calendarId}/acl/{ruleId}` - Remove ACL

## Metrics & Observability

- Prometheus metrics endpoint at `/metrics`
- Request counts, latencies, error rates
- Per-endpoint and per-user breakdowns
