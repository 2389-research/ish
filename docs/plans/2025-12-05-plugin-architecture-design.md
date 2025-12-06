# ISH Plugin Architecture Design

**Date:** 2025-12-05
**Status:** Approved
**Author:** Claude Code (with Doctor Biz)

## Overview

Transform ISH from a monolithic Google API server into a plugin-based platform for simulating any HTTP API. Plugins are Go modules that implement a standard interface and register themselves at compile time, following the Hashicorp Terraform provider pattern.

## Goals

1. **Zero-change consumption** - Apps only change base URL, nothing else
2. **Extensible platform** - Support Google, Discord, Stripe, Twilio, GitHub, any HTTP API
3. **Schema-driven UI** - No HTML in plugins, just data definitions
4. **Shared infrastructure** - Single SQLite DB, unified logging, common OAuth simulation
5. **External plugins** - Community can create plugins as separate Go modules
6. **Built-in quality** - ISH team maintains core plugins (Google, OAuth provider)

## Core Principles

- **Simple and brutal** - ISH is a test double, not a production replacement
- **Good enough for testing** - Don't over-engineer the UI or features
- **Plugin isolation** - Each plugin owns its routes, auth, schema, and data
- **Shared observability** - All plugins use common logging and token management

## Architecture

### Plugin Interface

```go
package core

type Plugin interface {
    // Metadata
    Name() string                          // "google", "discord", "stripe"
    Health() HealthStatus                  // Plugin health check

    // HTTP Routes
    RegisterRoutes(r chi.Router)           // API endpoints
    RegisterAuth(r chi.Router)             // OAuth/auth flows

    // Admin UI
    Schema() PluginSchema                  // UI definition (no HTML)

    // Data Generation
    Seed(ctx context.Context, size string) (SeedData, error)

    // Token Validation
    ValidateToken(token string) bool       // Check if token is valid for this plugin
}

type HealthStatus struct {
    Status  string    // "healthy", "degraded", "unavailable"
    Message string
}
```

### Plugin Schema (UI Generation)

Plugins define data structures, ISH core renders the UI:

```go
type PluginSchema struct {
    Resources []ResourceSchema
}

type ResourceSchema struct {
    Name        string              // "Messages", "Events", "Contacts"
    Slug        string              // "messages", "events" (URL path)
    Fields      []FieldSchema       // What data to show/edit
    Actions     []ActionSchema      // Available operations
    ListColumns []string            // Which fields in list view
}

type FieldSchema struct {
    Name        string              // "subject", "from", "date"
    Type        string              // "string", "datetime", "email", "text"
    Display     string              // "Subject", "From", "Date"
    Required    bool
    Editable    bool
}

type ActionSchema struct {
    Name        string              // "send", "delete", "archive"
    HTTPMethod  string              // "POST", "DELETE"
    Endpoint    string              // Template: "/messages/{id}/send"
    Confirm     bool                // Show confirmation dialog?
}
```

**ISH Core Renders:**
- List view: Table with columns from `ListColumns`
- Create form: Inputs from `Fields` where `Editable: true`
- Detail view: Display all `Fields`
- Action buttons: Buttons from `Actions`

**No HTML in plugins** - Just schema definitions, ISH handles rendering.

### Directory Structure

```
ish/
├── cmd/ish/                    # Main binary
├── plugins/
│   ├── core/                   # Plugin interface & registry
│   │   ├── plugin.go           # Plugin interface
│   │   ├── registry.go         # Plugin registration
│   │   └── schema.go           # Schema definitions
│   ├── google/                 # Built-in: Google APIs
│   │   ├── plugin.go           # Google plugin implementation
│   │   ├── gmail.go            # Gmail handlers
│   │   ├── calendar.go         # Calendar handlers
│   │   ├── people.go           # People handlers
│   │   ├── tasks.go            # Tasks handlers
│   │   └── schema.go           # UI schema definitions
│   └── oauth/                  # Built-in: OAuth provider simulation
│       ├── plugin.go           # OAuth plugin
│       └── tokens.go           # Token lifecycle
├── internal/
│   ├── admin/                  # Schema-driven admin UI renderer
│   ├── logging/                # Unified request logging
│   └── store/                  # Database layer
└── docs/
    └── plugins/                # Plugin development guide
```

**External Plugins (community):**
```
github.com/acme/ish-stripe-plugin/
github.com/acme/ish-discord-plugin/
github.com/acme/ish-twilio-plugin/
```

### Plugin Registration

Plugins auto-register in `init()`:

```go
// plugins/google/plugin.go
package google

import "github.com/2389/ish/plugins/core"

func init() {
    core.Register(&GooglePlugin{})
}

type GooglePlugin struct {
    store *store.Store
}

func (p *GooglePlugin) Name() string {
    return "google"
}

func (p *GooglePlugin) RegisterRoutes(r chi.Router) {
    r.Route("/gmail/v1", p.registerGmailRoutes)
    r.Route("/calendar/v3", p.registerCalendarRoutes)
    r.Route("/tasks/v1", p.registerTasksRoutes)
}

// ... implement other Plugin interface methods
```

**Main binary imports plugins:**
```go
// cmd/ish/main.go
import (
    _ "github.com/2389/ish/plugins/google"
    _ "github.com/2389/ish/plugins/oauth"
    _ "github.com/acme/ish-stripe-plugin"  // External plugin
)
```

Plugins compile into single binary, no runtime loading complexity.

## Shared Infrastructure

### Database Schema

```sql
-- Core tables (ISH manages)
CREATE TABLE request_logs (
    id INTEGER PRIMARY KEY,
    plugin_name TEXT NOT NULL,          -- "google", "discord"
    timestamp TIMESTAMP,
    method TEXT,
    path TEXT,                          -- Original path: "/gmail/v1/users/me/messages"
    status_code INTEGER,
    duration_ms INTEGER,
    user_id TEXT,
    request_body TEXT,
    response_body TEXT
);

CREATE TABLE oauth_tokens (
    token TEXT PRIMARY KEY,
    plugin_name TEXT NOT NULL,
    user_id TEXT,
    scopes TEXT,
    expires_at TIMESTAMP,
    refresh_token TEXT,
    revoked BOOLEAN DEFAULT 0
);

-- Plugin tables (plugins manage)
-- Convention: {plugin}_{resource}
-- Examples:
--   google_messages
--   google_events
--   discord_messages
--   stripe_payments
```

**Table Naming Convention:** `{plugin}_{resource}`

**Benefits:**
- Single SQLite file for entire system
- Unified request logs filterable by plugin
- Cross-plugin analytics
- Simple backup/restore

### Request Logging

All plugins use the same logging middleware:

```go
func LoggingMiddleware(store *store.Store) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Determine which plugin this request belongs to
            plugin := getPluginFromPath(r.URL.Path)

            // Capture request/response
            start := time.Now()
            wrapped := &responseWriter{ResponseWriter: w, body: &bytes.Buffer{}}
            next.ServeHTTP(wrapped, r)
            duration := time.Since(start)

            // Log to database
            store.LogRequest(&RequestLog{
                PluginName:   plugin,
                Path:         r.URL.Path,
                Method:       r.Method,
                StatusCode:   wrapped.statusCode,
                DurationMs:   duration.Milliseconds(),
                RequestBody:  readBody(r),
                ResponseBody: wrapped.body.String(),
            })
        })
    }
}
```

**Admin UI filtering:**
- View all requests across all plugins
- Filter by plugin name
- Filter by URL path
- Filter by status code
- Filter by time range

### OAuth Simulation

**Token Lifecycle:**

1. **Auto-accept any token** - First time ISH sees a token, it stores it
2. **Track everything** - All tokens stored in database
3. **Revocable** - Admin can revoke any token to test error handling
4. **Manual expiration** - Tokens don't expire unless admin sets it

**Token Validation:**
```go
func ValidateToken(token string, plugin string) bool {
    // First time seeing this token? Store it
    if !tokenExists(token) {
        storeToken(token, plugin, user: "default", revoked: false)
    }

    // Check if revoked
    return !isRevoked(token)
}
```

**OAuth Flow (Mock with Real Behavior):**

```
1. App redirects user to:
   https://ish.example.com/oauth/google/authorize?client_id=...&scope=...

2. ISH shows simple auth page (auto-approves in dev mode)

3. ISH redirects back with auth code:
   https://app.example.com/callback?code=fake_auth_code_12345

4. App exchanges code for token:
   POST https://ish.example.com/oauth/google/token
   { "code": "fake_auth_code_12345", ... }

5. ISH returns token response:
   {
     "access_token": "fake_access_token_67890",
     "refresh_token": "fake_refresh_token_abcde",
     "expires_in": 3600,
     "token_type": "Bearer"
   }

6. App uses token:
   GET https://ish.example.com/gmail/v1/users/me/messages
   Authorization: Bearer fake_access_token_67890
```

**Plugin OAuth Registration:**
```go
func (p *GooglePlugin) RegisterAuth(r chi.Router) {
    r.Get("/oauth/google/authorize", p.handleAuthorize)
    r.Post("/oauth/google/token", p.handleToken)
    r.Post("/oauth/google/revoke", p.handleRevoke)
}
```

Each plugin owns its OAuth endpoints completely.

## Data Seeding

Plugins choose between programmatic or LLM-based seeding:

### Programmatic (Fast, Deterministic)

```go
func (p *StripePlugin) Seed(ctx context.Context, size string) (SeedData, error) {
    count := sizeToCount(size)  // "small" = 10, "medium" = 50, "large" = 200

    for i := 0; i < count; i++ {
        p.store.CreatePayment(&Payment{
            ID:     fmt.Sprintf("pay_%d", i),
            Amount: rand.Intn(10000),
            Status: "succeeded",
        })
    }

    return SeedData{
        Summary: fmt.Sprintf("Created %d payments", count),
        Records: map[string]int{"payments": count},
    }, nil
}
```

### LLM-Based (Realistic Content)

```go
func (p *GooglePlugin) Seed(ctx context.Context, size string) (SeedData, error) {
    prompt := fmt.Sprintf(`
Generate %s realistic Gmail messages for a software developer.
Include subjects, from addresses, dates, and body content.
Return as JSON array.
`, size)

    data := callLLM(ctx, prompt)
    messages := parseMessages(data)

    for _, msg := range messages {
        p.store.CreateMessage(msg)
    }

    return SeedData{
        Summary: fmt.Sprintf("Created %d messages", len(messages)),
        Records: map[string]int{"messages": len(messages)},
    }, nil
}
```

**CLI Usage:**
```bash
./ish seed --plugin google --size medium
./ish seed --plugin stripe --size small
./ish seed --all --size large
```

## Admin UI

### Structure

```
/admin/
  /                          # Dashboard (all plugins overview)
  /logs                      # Unified logs (filterable by plugin)
  /tokens                    # OAuth token management
  /plugins/google/messages   # Google messages (schema-driven)
  /plugins/google/events     # Google events (schema-driven)
  /plugins/discord/channels  # Discord channels (schema-driven)
```

### Schema-Driven Rendering

**Example: Google Messages**

Plugin defines schema:
```go
func (p *GooglePlugin) Schema() PluginSchema {
    return PluginSchema{
        Resources: []ResourceSchema{
            {
                Name: "Messages",
                Slug: "messages",
                ListColumns: []string{"subject", "from", "date"},
                Fields: []FieldSchema{
                    {Name: "subject", Type: "string", Display: "Subject", Required: true, Editable: true},
                    {Name: "from", Type: "email", Display: "From", Required: true, Editable: true},
                    {Name: "to", Type: "email", Display: "To", Required: true, Editable: true},
                    {Name: "body", Type: "text", Display: "Body", Editable: true},
                    {Name: "date", Type: "datetime", Display: "Date", Editable: false},
                },
                Actions: []ActionSchema{
                    {Name: "send", HTTPMethod: "POST", Endpoint: "/gmail/v1/users/me/messages/send"},
                    {Name: "delete", HTTPMethod: "DELETE", Endpoint: "/gmail/v1/users/me/messages/{id}", Confirm: true},
                },
            },
        },
    }
}
```

ISH core automatically generates:
- **List view:** Table with Subject, From, Date columns
- **Create form:** Inputs for subject, from, to, body
- **Detail view:** Display all fields
- **Action buttons:** Send and Delete buttons

**No HTML in plugins!**

### Dashboard

Shows overview of all plugins:
- Plugin name and health status
- Request count (last 24h)
- Error rate
- Most recent requests
- Quick links to plugin resources

## Request Routing

### HTTP Flow

```
1. Request arrives: GET /gmail/v1/users/me/messages
2. Logging middleware captures request (before routing)
3. Router matches plugin routes (google plugin registered /gmail/*)
4. Auth middleware validates token
5. Plugin handler processes request
6. Response returned
7. Logging middleware captures response body
```

### Main Server Wiring

```go
func newServer(dbPath string) (http.Handler, error) {
    store, err := store.New(dbPath)
    if err != nil {
        return nil, err
    }

    r := chi.NewRouter()

    // Global middleware
    r.Use(middleware.Logger)
    r.Use(middleware.Recoverer)
    r.Use(logging.Middleware(store))
    r.Use(auth.Middleware(store))

    // Health check
    r.Get("/healthz", healthCheck)

    // Each plugin registers its routes
    for _, plugin := range registry.All() {
        plugin.RegisterAuth(r)
        plugin.RegisterRoutes(r)
    }

    // Admin UI (schema-driven)
    admin.RegisterRoutes(r, registry.All())

    return r, nil
}
```

## Migration Strategy

### Phase 1: Create Plugin Infrastructure

1. Define `plugins/core` package with interfaces
2. Create plugin registry
3. Build schema-driven admin renderer
4. Update logging to include `plugin_name`
5. Create OAuth token management

### Phase 2: Extract Google Plugin

1. Create `plugins/google` package
2. Move existing Gmail/Calendar/People/Tasks code
3. Implement Plugin interface
4. Define schemas for each resource
5. Register in `init()`
6. Test that everything still works

### Phase 3: Add OAuth Plugin

1. Create `plugins/oauth` package
2. Implement OAuth provider simulation
3. Token lifecycle management
4. Admin UI for token management

### Phase 4: Documentation & Examples

1. Write plugin development guide
2. Create example external plugin
3. Update product documentation
4. Migration guide for existing deployments

## API Compatibility

**Consuming apps change ONLY the base URL:**

**Before:**
```javascript
const service = google.gmail({
  version: 'v1',
  baseURL: 'https://gmail.googleapis.com',
  auth: oauthClient
});
```

**After:**
```javascript
const service = google.gmail({
  version: 'v1',
  baseURL: 'https://ish.example.com',  // <- ONLY CHANGE
  auth: oauthClient
});
```

All other code stays identical - no changes to request/response handling.

## Future Extensibility

### Community Plugins

External plugins follow the same pattern:

```go
// github.com/acme/ish-discord-plugin
package discord

import "github.com/2389/ish/plugins/core"

func init() {
    core.Register(&DiscordPlugin{})
}

type DiscordPlugin struct {}

func (p *DiscordPlugin) Name() string { return "discord" }

func (p *DiscordPlugin) RegisterRoutes(r chi.Router) {
    r.Route("/api/v10", p.registerDiscordRoutes)
}

// ... implement Plugin interface
```

**Usage:**
```go
// cmd/ish/main.go
import _ "github.com/acme/ish-discord-plugin"
```

Build:
```bash
go build -o ish ./cmd/ish
```

### Plugin Marketplace

Future enhancement: Plugin discovery and installation

```bash
./ish plugin install github.com/acme/ish-stripe-plugin
./ish plugin list
./ish plugin remove stripe
```

## Non-Goals

- **Production replacement** - ISH remains a test double
- **Perfect UI** - Schema-driven UI is "good enough", not beautiful
- **Event orchestration** - Plugins simulate events/webhooks, but ISH doesn't orchestrate between plugins
- **Runtime plugin loading** - Compile-time only for simplicity

## Success Criteria

1. **Consuming apps work with URL change only** - No code changes
2. **Community can create plugins** - Clear interface, good docs
3. **ISH team maintains Google plugin** - High quality built-in support
4. **Admin UI auto-generates from schema** - No HTML in plugins
5. **Unified observability** - All plugins visible in logs/dashboard
6. **OAuth simulation works** - Token lifecycle testing enabled

## Open Questions

None - Design validated with Doctor Biz.

## Appendix: Example Plugin

See `docs/plugins/example-stripe-plugin.md` for complete working example.
