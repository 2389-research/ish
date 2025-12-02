# ish

A fake Google API emulator for local development. "Google, but made of cardboard and duct tape."

Run Gmail, Calendar, and People APIs locally without hitting real Google services. Point your clients at `http://localhost:9000` instead of `https://www.googleapis.com` and develop offline.

## Quick Start

```bash
# Build
go build -o ish ./cmd/ish

# Seed with test data (static or AI-generated)
./ish seed --db ish.db

# Start the server
./ish serve --db ish.db --port 9000
```

Then configure your client:
```bash
export GOOGLE_API_BASE_URL=http://localhost:9000
```

## Authentication

Authentication is intentionally fake. Use a bearer token in this format:

```
Authorization: Bearer user:USERNAME
```

For example:
```bash
curl -H "Authorization: Bearer user:harper" \
  http://localhost:9000/gmail/v1/users/me/messages
```

## API Endpoints

### Gmail API

| Endpoint | Description |
|----------|-------------|
| `GET /gmail/v1/users/{userId}/profile` | Get user profile with historyId |
| `GET /gmail/v1/users/{userId}/messages` | List messages (supports `q`, `maxResults`, `pageToken`) |
| `GET /gmail/v1/users/{userId}/messages/{id}` | Get message details |
| `GET /gmail/v1/users/{userId}/messages/{id}/attachments/{attachmentId}` | Get attachment |
| `GET /gmail/v1/users/{userId}/history` | List history for incremental sync |

**Query syntax supported:** `is:unread`, `is:starred`, `in:inbox`, `in:sent`, `label:NAME`, `after:YYYY/M/D`

### Calendar API

| Endpoint | Description |
|----------|-------------|
| `GET /calendar/v3/calendars/{calendarId}/events` | List events (supports `timeMin`, `timeMax`, `syncToken`) |
| `GET /calendar/v3/calendars/{calendarId}/events/{eventId}` | Get event details |

### People API

| Endpoint | Description |
|----------|-------------|
| `GET /v1/people/me/connections` | List contacts (supports `syncToken`, `pageToken`) |
| `GET /v1/people/{resourceId}` | Get person details |
| `GET /people/v1/people:searchContacts` | Search contacts |

## Admin UI

Visit `http://localhost:9000/admin` for a web interface to:

- View and manage Gmail messages, Calendar events, and Contacts
- Generate AI-powered realistic test data (requires `ANTHROPIC_API_KEY`)
- See sample curl commands in the Getting Started guide

## Seeding Data

### Static seed data
```bash
./ish seed --db ish.db
```

### AI-generated data (requires Anthropic API key)
```bash
export ANTHROPIC_API_KEY=sk-ant-...
./ish seed --db ish.db --ai
```

Or generate via the admin UI at `/admin`.

## Features

- **Pagination**: `pageToken`, `maxResults` on list endpoints
- **Incremental sync**: `syncToken` for Calendar and People, `historyId` for Gmail
- **Query filtering**: Gmail query syntax (`is:unread`, `after:`, etc.)
- **Time filtering**: Calendar `timeMin`/`timeMax`
- **SQLite storage**: Inspectable database file

## Client Integration

Abstract your Google client behind an interface, then swap the base URL:

```typescript
const baseUrl = process.env.GOOGLE_API_BASE_URL || 'https://www.googleapis.com';

const response = await fetch(`${baseUrl}/gmail/v1/users/me/messages`, {
  headers: { Authorization: `Bearer user:${userId}` }
});
```

- Production: `GOOGLE_API_BASE_URL=https://www.googleapis.com`
- Development: `GOOGLE_API_BASE_URL=http://localhost:9000`

## License

MIT
