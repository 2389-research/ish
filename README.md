# ish

A fake Google API emulator for local development. "Google, but made of cardboard and duct tape."

Run Gmail, Calendar, People, and Tasks APIs locally without hitting real Google services. Point your clients at `http://localhost:9000` instead of `https://www.googleapis.com` and develop offline.

Built with a **plugin architecture** for easy extension. Add your own mock APIs (Stripe, Twilio, AWS, etc.) by implementing the simple Plugin interface.

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

ISH supports two authentication modes:

### 1. Simple Bearer Tokens (for quick testing)

Use a bearer token in this format:

```
Authorization: Bearer user:USERNAME
```

For example:
```bash
curl -H "Authorization: Bearer user:harper" \
  http://localhost:9000/gmail/v1/users/me/messages
```

### 2. OAuth 2.0 Flows (for realistic testing)

ISH includes a mock OAuth 2.0 provider that simulates the complete authorization code flow:

```bash
# 1. Direct user to authorize endpoint
GET /oauth/{plugin}/authorize?redirect_uri=YOUR_CALLBACK&state=random123

# 2. ISH auto-approves and redirects with code
# Redirect: YOUR_CALLBACK?code=AUTHZ_CODE&state=random123

# 3. Exchange authorization code for access token
POST /oauth/{plugin}/token
  grant_type=authorization_code&code=AUTHZ_CODE&redirect_uri=YOUR_CALLBACK

# Response: {"access_token": "token_...", "refresh_token": "refresh_..."}

# 4. Use access token for API requests
curl -H "Authorization: Bearer token_..." \
  http://localhost:9000/gmail/v1/users/me/messages

# 5. Refresh when expired
POST /oauth/{plugin}/token
  grant_type=refresh_token&refresh_token=refresh_...
```

Supported OAuth plugins: `google`, `github`, and any other plugin that implements OAuth.

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

### Tasks API

| Endpoint | Description |
|----------|-------------|
| `GET /tasks/v1/users/@me/lists` | List task lists |
| `GET /tasks/v1/lists/{listId}/tasks` | List tasks in a list |
| `POST /tasks/v1/lists/{listId}/tasks` | Create a new task |
| `GET /tasks/v1/lists/{listId}/tasks/{taskId}` | Get task details |
| `PATCH /tasks/v1/lists/{listId}/tasks/{taskId}` | Update a task |
| `DELETE /tasks/v1/lists/{listId}/tasks/{taskId}` | Delete a task |

## Plugin System

ISH uses a **plugin architecture** where each API is implemented as a plugin. This makes the system:

- **Extensible**: Add new APIs without modifying core code
- **Modular**: Each plugin is self-contained with its own routes, schema, and data
- **Testable**: Plugins can be tested independently
- **Discoverable**: Plugins auto-register and appear in admin UI

### Built-in Plugins

- **Google Plugin**: Gmail, Calendar, People, and Tasks APIs
- **OAuth Plugin**: Mock OAuth 2.0 provider for testing authentication flows (authorization code, refresh tokens, revocation)
- **Discord Plugin**: Discord webhook API v10 for testing webhook integrations
- **Twilio Plugin**: Twilio SMS and Voice APIs with async webhook callbacks
- **GitHub Plugin**: GitHub REST API (users, repos, issues, PRs, comments, reviews, webhooks)
- **Home Assistant Plugin**: Home Assistant REST API for smart home integrations
- **SendGrid Plugin**: SendGrid email API for transactional email testing

### Example Usage

**Discord Webhooks:**
```bash
# Send a webhook message
curl -X POST http://localhost:9000/api/webhooks/YOUR_ID/YOUR_TOKEN \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Hello from ISH!",
    "username": "My Bot",
    "embeds": [{
      "title": "Test Embed",
      "description": "This is a test",
      "color": 5814783
    }]
  }'

# Get webhook info
curl http://localhost:9000/api/webhooks/YOUR_ID/YOUR_TOKEN

# Edit a message
curl -X PATCH http://localhost:9000/api/webhooks/YOUR_ID/YOUR_TOKEN/messages/MESSAGE_ID \
  -H "Content-Type: application/json" \
  -d '{"content": "Updated content"}'

# Delete a message
curl -X DELETE http://localhost:9000/api/webhooks/YOUR_ID/YOUR_TOKEN/messages/MESSAGE_ID
```

### Creating Your Own Plugin

You can create plugins for any API you want to mock:

```go
type MyPlugin struct {
    store *store.Store
}

func (p *MyPlugin) Name() string { return "myplugin" }

func (p *MyPlugin) RegisterRoutes(r chi.Router) {
    r.Get("/v1/myresource", p.handleList)
    r.Post("/v1/myresource", p.handleCreate)
}

func (p *MyPlugin) Schema() core.PluginSchema {
    // Define admin UI structure
}

func (p *MyPlugin) Seed(ctx context.Context, size string) (core.SeedData, error) {
    // Generate test data
}
```

See the [Plugin Development Guide](./docs/plugins/DEVELOPMENT.md) for complete documentation and the [Stripe Plugin Example](./docs/plugins/example-stripe-plugin.md) for a working example.

## Admin UI

Visit `http://localhost:9000/admin` for a web interface to:

- View and manage resources from all plugins (Messages, Events, Contacts, Tasks)
- Browse request logs with plugin attribution
- Generate AI-powered realistic test data (requires `ANTHROPIC_API_KEY`)
- See sample curl commands in the Getting Started guide

The admin UI is **schema-driven**: plugins define their data structure, and ISH automatically generates forms, lists, and actions.

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

- **Plugin Architecture**: Extensible system for adding new mock APIs
- **Schema-Driven UI**: Auto-generated admin interface from plugin schemas
- **Request Logging**: Track all API calls with plugin attribution
- **OAuth Simulation**: Mock OAuth 2.0 flows for testing authentication
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

## Documentation

- [Plugin Development Guide](./docs/plugins/DEVELOPMENT.md) - Learn how to create plugins
- [Stripe Plugin Example](./docs/plugins/example-stripe-plugin.md) - Complete working example
- [Architecture Overview](./docs/ARCHITECTURE.md) - System design and plugin infrastructure
- [API Specification](./docs/spec.md) - Google API compatibility details

## License

MIT
