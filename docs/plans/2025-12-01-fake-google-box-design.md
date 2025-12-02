# Fake Google Box Design

A local emulator for Google APIs (Gmail, Calendar, People) to enable client development without hitting real Google services.

## Decisions

| Decision | Choice |
|----------|--------|
| Language | Go |
| Storage | SQLite |
| Framework | chi router |
| Initial scope | 6 read-only endpoints (2 per API) |
| Auth | `Bearer user:xxx` parsed from header |
| Port | 9000 |

## Project Structure

```
ish/
├── cmd/
│   └── ish/
│       └── main.go          # Entry point, wires everything
├── internal/
│   ├── auth/
│   │   └── middleware.go    # Parse "Bearer user:xxx", extract user
│   ├── store/
│   │   ├── store.go         # SQLite setup, migrations
│   │   ├── gmail.go         # Gmail entity CRUD
│   │   ├── calendar.go      # Calendar entity CRUD
│   │   └── people.go        # People entity CRUD
│   ├── gmail/
│   │   └── handlers.go      # /gmail/v1/... routes
│   ├── calendar/
│   │   └── handlers.go      # /calendar/v3/... routes
│   └── people/
│       └── handlers.go      # /people/v1/... routes
├── fixtures/
│   └── seed.sql             # Default test data
├── docs/
│   └── spec.md              # Original spec
├── go.mod
└── go.sum
```

## SQLite Schema

```sql
-- Users (the fake accounts)
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Gmail: Labels
CREATE TABLE gmail_labels (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    name TEXT NOT NULL,
    type TEXT DEFAULT 'user'
);

-- Gmail: Threads
CREATE TABLE gmail_threads (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    snippet TEXT
);

-- Gmail: Messages
CREATE TABLE gmail_messages (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    thread_id TEXT REFERENCES gmail_threads(id),
    label_ids TEXT,
    snippet TEXT,
    internal_date INTEGER,
    payload TEXT
);

-- Calendar: Calendars
CREATE TABLE calendars (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    summary TEXT
);

-- Calendar: Events
CREATE TABLE calendar_events (
    id TEXT PRIMARY KEY,
    calendar_id TEXT REFERENCES calendars(id),
    summary TEXT,
    description TEXT,
    start_time TEXT,
    end_time TEXT,
    attendees TEXT
);

-- People: Contacts
CREATE TABLE people (
    resource_name TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    data TEXT
);
```

## Auth Middleware

Parses token from `Authorization` header:

```
Authorization: Bearer user:harper  -> user_id = "harper"
Authorization: Bearer harper       -> user_id = "harper"
Authorization: Bearer anything     -> user_id = "default"
```

User ID stored in request context for handler access.

## Initial Endpoints (v1)

| API | Endpoint | Method | Description |
|-----|----------|--------|-------------|
| Gmail | `/gmail/v1/users/{userId}/messages` | GET | List messages |
| Gmail | `/gmail/v1/users/{userId}/messages/{id}` | GET | Get message |
| Calendar | `/calendar/v3/calendars/{calendarId}/events` | GET | List events |
| Calendar | `/calendar/v3/calendars/{calendarId}/events/{id}` | GET | Get event |
| People | `/people/v1/people:searchContacts` | GET | Search contacts |
| People | `/people/v1/{resourceName}` | GET | Get person |

### Query Parameters

- Gmail list: `maxResults`, `pageToken`, `q`
- Calendar list: `maxResults`, `pageToken`, `timeMin`, `timeMax`
- People search: `query`, `pageSize`, `pageToken`

### Response Format

Google-style wrapper:
```json
{
  "messages": [...],
  "nextPageToken": "abc123",
  "resultSizeEstimate": 42
}
```

### Error Format

```json
{
  "error": {
    "code": 404,
    "message": "Message not found",
    "status": "NOT_FOUND"
  }
}
```

## CLI Interface

```bash
ish serve              # Run server on :9000
ish serve --port 9001  # Custom port
ish seed               # Seed default data
ish seed --user harper # Seed specific user
ish reset              # Wipe + reseed
```

## Environment Variables

| Var | Default | Purpose |
|-----|---------|---------|
| `ISH_PORT` | `9000` | Server port |
| `ISH_DB_PATH` | `./ish.db` | SQLite file location |
| `ISH_LOG_LEVEL` | `info` | Logging verbosity |

## Default Seed Data

User `harper` with:
- 5 Gmail messages across 2 threads
- 1 calendar with 3 events
- 3 contacts

## Health Check

`GET /healthz` returns `{"ok": true}`

## Future Enhancements (Not in v1)

- Chaos mode (rate limits, random 500s, latency injection)
- Eventual consistency simulation
- Write operations (POST/PUT/DELETE)
- Record/replay mode for proxying real Google
