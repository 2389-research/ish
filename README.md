# ISH - Intelligent Server Hub

**A complete API testing platform for local development.** Mock any API service locally—Google, GitHub, Twilio, Discord, SendGrid, and more—without hitting production servers or burning through API quotas.

## What ISH Can Do

- 🔌 **Mock 7+ Popular APIs**: Google (Gmail, Calendar, Contacts, Tasks), GitHub, Twilio, Discord, SendGrid, Home Assistant, OAuth 2.0
- 🔐 **Realistic Authentication**: OAuth 2.0 authorization flows, token refresh/revocation, or simple bearer tokens
- 💾 **Persistent SQLite Storage**: All data stored locally in an inspectable database
- 🎨 **Auto-Generated Admin UI**: Web interface to view/manage all resources across plugins
- 🤖 **AI-Powered Test Data**: Generate realistic emails, contacts, events using Claude AI
- 📊 **Request Logging**: Track all API calls with timestamps and plugin attribution
- 🔍 **Full Query Support**: Gmail search syntax, Calendar time filtering, pagination, incremental sync
- 🚀 **Plugin Architecture**: Add your own mock APIs by implementing a simple interface
- 📡 **Webhook Simulation**: Async webhook delivery for Twilio, SSRF-protected GitHub webhooks

## Available Plugins

| Plugin | What It Mocks | Key Features |
|--------|---------------|--------------|
| **Google** | Gmail, Calendar, Contacts, Tasks | Query syntax, pagination, history/sync tokens, attachments |
| **OAuth** | OAuth 2.0 provider | Authorization code flow, refresh tokens, revocation |
| **GitHub** | REST API v3 | Repos, issues, PRs, comments, reviews, webhooks (SSRF-protected) |
| **Twilio** | SMS & Voice APIs | Async webhook delivery, phone numbers, calls, messages |
| **Discord** | Webhook API v10 | Execute webhooks, edit/delete messages, embeds, components |
| **SendGrid** | Email API v3 | Send emails, manage API keys, suppressions |
| **Home Assistant** | REST API | Entities, states, service calls, token auth |

**Total**: 7 plugins, 50+ API endpoints, production-quality test data

## Quick Start

```bash
# 1. Build ISH
go build -o ish ./cmd/ish

# 2. Seed with test data (creates realistic emails, contacts, repos, etc.)
./ish seed --db ish.db

# 3. Start the server
./ish serve --db ish.db --port 9000

# 4. Test it out!
curl -H "Authorization: Bearer user:harper" \
  http://localhost:9000/gmail/v1/users/me/messages
```

**Want AI-generated data?** Set `ANTHROPIC_API_KEY` and run `./ish seed --db ish.db --ai`

**View everything in your browser:** Visit `http://localhost:9000/admin`

### Using with Your App

Point your client at ISH instead of the real API:

```typescript
// Development: use ISH locally
const baseUrl = process.env.API_BASE_URL || 'http://localhost:9000';

// Production: real API
// export API_BASE_URL=https://www.googleapis.com
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

See [Available Plugins](#available-plugins) above for the complete list of 7 built-in plugins.

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

## Documentation

- [Plugin Development Guide](./docs/plugins/DEVELOPMENT.md) - Learn how to create plugins
- [Stripe Plugin Example](./docs/plugins/example-stripe-plugin.md) - Complete working example
- [Architecture Overview](./docs/ARCHITECTURE.md) - System design and plugin infrastructure
- [API Specification](./docs/spec.md) - Google API compatibility details

## License

MIT
