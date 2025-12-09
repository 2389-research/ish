# ISH - Intelligent Server Hub

**A complete API testing platform for local development.** Mock any API service locally—Google, GitHub, Twilio, Discord, SendGrid, and more—without hitting production servers or burning through API quotas.

## What ISH Can Do

- 🔌 **Mock 7+ Popular APIs**: Google (Gmail, Calendar, Contacts, Tasks), GitHub, Twilio, Discord, SendGrid, Home Assistant, OAuth 2.0
- 🔐 **Realistic Authentication**: OAuth 2.0 authorization flows, token refresh/revocation, or simple bearer tokens
- 💾 **Persistent SQLite Storage**: All data stored locally in an inspectable database
- 🎨 **Auto-Generated Admin UI**: Web interface to view resources across all plugins and browse request logs
- 🤖 **AI-Powered Test Data**: Uses OpenAI to generate realistic emails, contacts, and events (falls back to static data if no API key)
- 📊 **Request Logging**: Track all API calls with timestamps and plugin attribution
- 📧 **Auto-Reply Simulation**: Automatically generates realistic replies to sent emails (enable with ISH_AUTO_REPLY=true)
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
./ish seed

# 3. Start the server
./ish serve --port 9000

# 4. Test it out!
curl -H "Authorization: Bearer user:me" \
  http://localhost:9000/gmail/v1/users/me/messages
```

**Database Location:** ISH automatically determines the best database location using this priority:
1. `--db` flag (highest priority, overrides all defaults)
2. `ISH_DB_PATH` environment variable
3. `./ish.db` if it already exists (backwards compatibility)
4. Platform-specific default:
   - **Linux/macOS**: `~/.local/share/ish/ish.db` (XDG Base Directory spec)
   - **Windows**: `%LOCALAPPDATA%\ish\ish.db` (typically `C:\Users\You\AppData\Local\ish\`)

**View everything in your browser:** Visit `http://localhost:9000/admin`

### Using with Your App

Point your client at ISH instead of the real API:

```typescript
// Development: use ISH locally
const baseUrl = process.env.API_BASE_URL || 'http://localhost:9000';

// Production: real API
// export API_BASE_URL=https://www.googleapis.com
```

## SDK Integration Examples

ISH works seamlessly with official API client libraries. Just point them at your local ISH server instead of production endpoints.

### Google APIs (Node.js)

```javascript
const { google } = require('googleapis');

// Configure Gmail client to use ISH
const gmail = google.gmail({
  version: 'v1',
  auth: 'user:harper',  // Simple auth for testing
  // Point to ISH instead of production
  rootUrl: process.env.GOOGLE_API_BASE_URL || 'http://localhost:9000'
});

// Now use the SDK normally
const messages = await gmail.users.messages.list({
  userId: 'me',
  maxResults: 10,
  q: 'is:unread'
});

// Calendar works the same way
const calendar = google.calendar({
  version: 'v3',
  auth: 'user:harper',
  rootUrl: process.env.GOOGLE_API_BASE_URL || 'http://localhost:9000'
});

const events = await calendar.events.list({
  calendarId: 'primary',
  timeMin: new Date().toISOString(),
  maxResults: 10
});
```

### Google APIs (Python)

```python
from googleapiclient.discovery import build
import os

# Point to ISH for local development
base_url = os.getenv('GOOGLE_API_BASE_URL', 'http://localhost:9000')

# Build Gmail client
gmail = build(
    'gmail', 'v1',
    developerKey='user:harper',  # Simple auth
    static_discovery=False,
    # Override the discovery URL to use ISH
    discoveryServiceUrl=f'{base_url}/discovery/v1/apis/{{api}}/{{apiVersion}}/rest'
)

# Use the SDK normally
messages = gmail.users().messages().list(userId='me', maxResults=10).execute()

# Calendar example
calendar = build('calendar', 'v3', developerKey='user:harper', static_discovery=False)
events = calendar.events().list(calendarId='primary', maxResults=10).execute()
```

### GitHub (Octokit - Node.js)

```javascript
const { Octokit } = require('@octokit/rest');

const octokit = new Octokit({
  auth: 'ghp_testtoken123',
  // Point to ISH instead of api.github.com
  baseUrl: process.env.GITHUB_API_BASE_URL || 'http://localhost:9000'
});

// Use GitHub SDK normally
const { data: repos } = await octokit.repos.listForAuthenticatedUser();
const { data: issues } = await octokit.issues.listForRepo({
  owner: 'facebook',
  repo: 'react'
});

// Create issues, PRs, comments, etc.
await octokit.issues.create({
  owner: 'facebook',
  repo: 'react',
  title: 'Test issue from ISH',
  body: 'This is a test'
});
```

### Twilio (Node.js)

```javascript
const twilio = require('twilio');

const client = twilio(
  'test_account_sid',
  'test_auth_token',
  {
    // Point to ISH instead of api.twilio.com
    accountSid: 'test_account_sid',
    edge: undefined,
    region: undefined,
    // Override the hostname
    lazyLoading: false,
    httpClient: {
      request: (opts) => {
        opts.uri = opts.uri.replace('https://api.twilio.com', 'http://localhost:9000');
        return require('https').request(opts);
      }
    }
  }
);

// Send SMS via ISH
const message = await client.messages.create({
  body: 'Hello from ISH!',
  from: '+15551234567',
  to: '+15559876543'
});
```

### SendGrid (Node.js)

```javascript
const sgMail = require('@sendgrid/mail');

// ISH doesn't need a real API key
sgMail.setApiKey('SG.test_key_from_ish');

// Override the API host to point to ISH
sgMail.setClient(require('@sendgrid/client'));
sgMail.client.setDefaultRequest('baseUrl', 'http://localhost:9000');

// Send email via ISH
await sgMail.send({
  to: 'test@example.com',
  from: 'sender@example.com',
  subject: 'Testing ISH',
  text: 'This email goes to ISH, not SendGrid'
});
```

### Discord Webhooks (Node.js)

```javascript
const { WebhookClient } = require('discord.js');

// Point Discord webhook client to ISH
const webhook = new WebhookClient({
  url: 'http://localhost:9000/api/webhooks/YOUR_ID/YOUR_TOKEN'
});

// Send messages via ISH
await webhook.send({
  content: 'Hello from ISH!',
  username: 'ISH Bot',
  embeds: [{
    title: 'Test Embed',
    description: 'This webhook goes to ISH, not Discord',
    color: 0x5865F2
  }]
});
```

### Key Takeaway

**Most SDKs let you override the base URL or endpoint.** Check your SDK's documentation for:
- `baseUrl`, `rootUrl`, `endpoint` configuration options
- Environment variables like `GOOGLE_API_BASE_URL`
- Custom HTTP client/transport configuration

Switch between ISH (dev) and production by changing one environment variable!

## Authentication

ISH supports two authentication modes:

### 1. Simple Bearer Tokens (for quick testing)

Use a bearer token in this format:

```
Authorization: Bearer user:USERNAME
```

For example:
```bash
curl -H "Authorization: Bearer user:me" \
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

**Note:** For API testing, use simple bearer tokens (`user:USERNAME`) instead of OAuth tokens. OAuth flow is implemented for testing authorization code exchange and token management, but tokens are not currently validated for API authentication.

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

### Health Check

| Endpoint | Description |
|----------|-------------|
| `GET /healthz` | Health check endpoint (returns `{"ok": true}`) |

Useful for Docker health checks, Kubernetes readiness probes, and monitoring.

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
- See sample curl commands in the Getting Started guide

The admin UI is **schema-driven**: plugins define their data structure, and ISH automatically generates forms, lists, and actions.

## Seeding Data

ISH uses **AI-generated data by default** when `OPENAI_API_KEY` is set. Falls back to static test data if no API key is provided.

```bash
# Seed with AI-generated data (default if OPENAI_API_KEY is set)
export OPENAI_API_KEY=sk-...
./ish seed

# Or reset and reseed (clears existing data first)
./ish reset
```

**Note:** If you need to reseed, use `./ish reset` to clear existing data first, as the seed command is not idempotent and will fail on duplicate entries.

## Environment Variables

| Variable | Purpose | Default |
|----------|---------|---------|
| `OPENAI_API_KEY` | Enable AI-generated seed data | (none - uses static data) |
| `OPENAI_MODEL` | OpenAI model for generation | `gpt-4o-mini` |
| `ISH_AUTO_REPLY` | Enable automatic email replies | `false` |
| `ISH_REPLY_DELAY_MIN` | Min seconds before auto-reply | `2` |
| `ISH_REPLY_DELAY_MAX` | Max seconds before auto-reply | `30` |
| `ISH_PORT` | Server port | `9000` |
| `ISH_DB_PATH` | Database location | (see Database Location section) |

## Documentation

- [Plugin Development Guide](./docs/plugins/DEVELOPMENT.md) - Learn how to create plugins
- [Stripe Plugin Example](./docs/plugins/example-stripe-plugin.md) - Complete working example
- [Architecture Overview](./docs/ARCHITECTURE.md) - System design and plugin infrastructure
- [API Specification](./docs/spec.md) - Google API compatibility details

## License

MIT
