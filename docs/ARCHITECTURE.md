# ISH Architecture

This document describes the architecture of ISH, the fake Google API emulator for local development.

## Overview

ISH is built around a **plugin architecture** that makes it easy to add new mock APIs. Each API (Gmail, Calendar, Stripe, etc.) is implemented as a plugin that registers itself at startup and provides routes, schemas, and data generation.

```
┌─────────────────────────────────────────────────────────────┐
│                         ISH Server                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐                 │
│  │  Google  │  │  OAuth   │  │  Your    │  ...more        │
│  │  Plugin  │  │  Plugin  │  │  Plugin  │  plugins        │
│  └─────┬────┘  └─────┬────┘  └─────┬────┘                 │
│        │             │             │                       │
│        └─────────────┴─────────────┘                       │
│                      │                                     │
│              ┌───────▼────────┐                            │
│              │ Plugin Registry │                            │
│              └───────┬────────┘                            │
│                      │                                     │
│        ┌─────────────┴─────────────┐                      │
│        │                           │                       │
│   ┌────▼────┐              ┌───────▼────────┐             │
│   │  HTTP   │              │  Admin UI      │             │
│   │ Router  │              │ (Schema-based) │             │
│   └────┬────┘              └───────┬────────┘             │
│        │                           │                       │
│        └───────────┬───────────────┘                      │
│                    │                                       │
│            ┌───────▼────────┐                              │
│            │ Middleware      │                              │
│            │ - Auth          │                              │
│            │ - Logging       │                              │
│            └───────┬────────┘                              │
│                    │                                       │
│            ┌───────▼────────┐                              │
│            │  SQLite Store   │                              │
│            └─────────────────┘                              │
└─────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Plugin System

**Location**: `plugins/core/`

The plugin system provides the foundation for all functionality in ISH.

#### Plugin Interface

```go
type Plugin interface {
    Name() string
    Health() HealthStatus
    RegisterRoutes(r chi.Router)
    RegisterAuth(r chi.Router)
    Schema() PluginSchema
    Seed(ctx context.Context, size string) (SeedData, error)
    ValidateToken(token string) bool
}
```

Every plugin must implement this interface. See [Plugin Development Guide](./plugins/DEVELOPMENT.md) for details.

#### Plugin Registry

**File**: `plugins/core/registry.go`

The registry is a global map of plugin name to plugin instance:

```go
var registry = make(map[string]Plugin)

func Register(p Plugin) {
    registry[p.Name()] = p
}

func All() []Plugin {
    // Returns all registered plugins
}
```

Plugins register themselves in `init()` functions:

```go
func init() {
    core.Register(&GooglePlugin{})
}
```

This happens before `main()` runs, so by the time the server starts, all plugins are registered.

#### Plugin Lifecycle

1. **Import**: Plugin package imported with `_` blank import in `cmd/ish/main.go`
2. **Init**: Plugin's `init()` calls `core.Register()`
3. **Bootstrap**: Server calls `SetStore()` on plugins that need database access
4. **Registration**: Server calls `RegisterAuth()` and `RegisterRoutes()` on each plugin
5. **Runtime**: Plugins handle requests via their registered routes

### 2. HTTP Server

**Location**: `cmd/ish/main.go`

The server uses [chi](https://github.com/go-chi/chi) for HTTP routing.

#### Middleware Stack

Requests flow through middleware in this order:

1. **Logger** (`middleware.Logger`): Logs all requests
2. **Recoverer** (`middleware.Recoverer`): Catches panics
3. **Logging Middleware** (`logging.Middleware`): Logs to database with plugin attribution
4. **Auth Middleware** (`auth.Middleware`): Extracts user from token

#### Request Flow

```
HTTP Request
    │
    ▼
┌───────────────┐
│ Logger        │
└───────┬───────┘
        ▼
┌───────────────┐
│ Recoverer     │
└───────┬───────┘
        ▼
┌───────────────┐
│ Logging       │ ──► Store request log
└───────┬───────┘
        ▼
┌───────────────┐
│ Auth          │ ──► Extract user_id
└───────┬───────┘
        ▼
┌───────────────┐
│ Plugin Route  │
│ Handler       │
└───────┬───────┘
        ▼
    Response
```

### 3. Schema-Driven Admin UI

**Location**: `internal/admin/`

The admin UI automatically generates interfaces based on plugin schemas.

#### How It Works

1. Plugin defines schema in `Schema()` method
2. Admin UI queries `core.All()` to get all plugins
3. For each plugin, render resources from `plugin.Schema()`
4. Schema describes fields, types, actions
5. UI renders forms, tables, buttons based on schema

#### Schema Structure

```go
type PluginSchema struct {
    Resources []ResourceSchema
}

type ResourceSchema struct {
    Name        string         // "Messages"
    Slug        string         // "messages" (URL)
    Fields      []FieldSchema  // What to show/edit
    Actions     []ActionSchema // What users can do
    ListColumns []string       // Columns in list view
}
```

Example flow:
1. User visits `/admin/messages`
2. Admin routes call `plugins/google.Schema()`
3. Find "Messages" resource in schema
4. Render table with `ListColumns` fields
5. Render action buttons from `Actions`
6. User clicks "delete" → makes DELETE request to endpoint in action

### 4. Request Logging

**Location**: `internal/logging/`

Every API request is logged to the database with plugin attribution.

#### Logging Middleware

```go
func Middleware(store *store.Store) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()

            // Determine which plugin handles this route
            pluginName := detectPlugin(r.URL.Path)

            // Wrap response writer to capture status code
            ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

            // Handle request
            next.ServeHTTP(ww, r)

            // Log to database (async)
            go store.LogRequest(&RequestLog{
                PluginName: pluginName,
                Method:     r.Method,
                Path:       r.URL.Path,
                StatusCode: ww.Status(),
                DurationMs: time.Since(start).Milliseconds(),
            })
        })
    }
}
```

#### Plugin Detection

The logging system detects which plugin handled a request by matching URL path prefixes:

- `/gmail/` → google
- `/calendar/` → google
- `/oauth/` → oauth
- `/v1/customers` → stripe

This is done by checking path patterns against known plugin routes.

### 5. Authentication

**Location**: `internal/auth/`

ISH uses **fake authentication** for local development convenience.

#### Token Format

```
Authorization: Bearer user:USERNAME
Authorization: Bearer oauth:ACCESS_TOKEN
```

#### Auth Middleware

```go
func Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Skip auth for admin routes
        if strings.HasPrefix(r.URL.Path, "/admin") {
            next.ServeHTTP(w, r)
            return
        }

        // Extract token
        auth := r.Header.Get("Authorization")
        token := strings.TrimPrefix(auth, "Bearer ")

        // Parse token type
        if strings.HasPrefix(token, "user:") {
            userID := strings.TrimPrefix(token, "user:")
            ctx := context.WithValue(r.Context(), "user_id", userID)
            next.ServeHTTP(w, r.WithContext(ctx))
            return
        }

        if strings.HasPrefix(token, "oauth:") {
            // Validate OAuth token with plugins
            accessToken := strings.TrimPrefix(token, "oauth:")
            // Check with OAuth plugin...
        }

        // No valid token
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
    })
}
```

### 6. OAuth Simulation

**Location**: `plugins/oauth/`

The OAuth plugin provides a mock OAuth 2.0 provider for testing authentication flows.

#### OAuth Flow

```
1. Client requests authorization
   GET /oauth/{plugin}/authorize?redirect_uri=...&state=...

2. OAuth plugin generates auth code
   Redirects to: {redirect_uri}?code={CODE}&state={STATE}

3. Client exchanges code for token
   POST /oauth/{plugin}/token
   Body: grant_type=authorization_code&code={CODE}

4. OAuth plugin returns tokens
   Response: {
       "access_token": "...",
       "refresh_token": "...",
       "token_type": "Bearer"
   }

5. Client uses access token
   Authorization: Bearer {ACCESS_TOKEN}
```

#### Token Storage

OAuth tokens are stored in the `oauth_tokens` table:

```sql
CREATE TABLE oauth_tokens (
    access_token TEXT PRIMARY KEY,
    refresh_token TEXT,
    plugin_name TEXT NOT NULL,
    user_id TEXT NOT NULL,
    revoked BOOLEAN DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
)
```

### 7. Data Storage

**Location**: `internal/store/`

ISH uses SQLite for data persistence.

#### Store Structure

```go
type Store struct {
    db *sql.DB
}

func New(dbPath string) (*Store, error) {
    db, err := sql.Open("sqlite3", dbPath)
    // Initialize schema
    // Return store
}
```

#### Database Schema

Each plugin can define its own tables. Common patterns:

**User-scoped data**:
```sql
CREATE TABLE gmail_messages (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    thread_id TEXT,
    snippet TEXT,
    -- ... more fields
)
```

**Plugin-specific data**:
```sql
CREATE TABLE stripe_customers (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL,
    name TEXT,
    created INTEGER NOT NULL
)
```

**System tables**:
```sql
CREATE TABLE request_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    plugin_name TEXT,
    method TEXT,
    path TEXT,
    status_code INTEGER,
    duration_ms INTEGER,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
)
```

### 8. Data Seeding

**Location**: `internal/seed/`

ISH can generate realistic test data using AI or static templates.

#### Seed Generator

```go
type Generator struct {
    userID string
    useAI  bool
}

func (g *Generator) Generate(ctx context.Context, emailCount, eventCount, contactCount int) (*SeedData, error) {
    if g.useAI {
        // Call Anthropic API to generate realistic data
        return g.generateWithAI(ctx, emailCount, eventCount, contactCount)
    }

    // Use static templates
    return g.generateStatic(emailCount, eventCount, contactCount)
}
```

#### Plugin Seeding

Each plugin implements `Seed()` to generate its own test data:

```go
func (p *GooglePlugin) Seed(ctx context.Context, size string) (core.SeedData, error) {
    // Create messages, events, contacts, tasks
    return core.SeedData{
        Summary: "Created 50 messages, 25 events, 25 contacts",
        Records: map[string]int{
            "messages": 50,
            "events":   25,
            "contacts": 25,
        },
    }, nil
}
```

Seed data can be generated:
- Via CLI: `./ish seed --db ish.db`
- Via admin UI: Click "Generate Data" button

## Built-in Plugins

### Google Plugin

**Location**: `plugins/google/`

Provides mock implementations of Google APIs:

- **Gmail API** (`/gmail/v1/*`): Messages, threads, labels, attachments
- **Calendar API** (`/calendar/v3/*`): Events, calendars
- **People API** (`/people/v1/*`): Contacts, connections
- **Tasks API** (`/tasks/v1/*`): Task lists, tasks

Each API is implemented in its own file:
- `gmail.go`: Gmail routes and handlers
- `calendar.go`: Calendar routes and handlers
- `people.go`: People routes and handlers
- `tasks.go`: Tasks routes and handlers

### OAuth Plugin

**Location**: `plugins/oauth/`

Provides OAuth 2.0 simulation for all plugins:

- Authorization flow (`GET /oauth/{plugin}/authorize`)
- Token exchange (`POST /oauth/{plugin}/token`)
- Token refresh (`POST /oauth/{plugin}/token` with `grant_type=refresh_token`)
- Token revocation (`POST /oauth/{plugin}/revoke`)

The OAuth plugin is generic - it works for any plugin by using the `{plugin}` parameter in routes.

## Request Flow Example

Let's trace a request to list Gmail messages:

```
1. Client sends request:
   GET /gmail/v1/users/me/messages
   Authorization: Bearer user:harper

2. Request enters middleware stack:
   Logger → Recoverer → Logging → Auth

3. Logging middleware:
   - Detects plugin: "google" (from /gmail/ prefix)
   - Stores plugin_name for later logging

4. Auth middleware:
   - Extracts token: "user:harper"
   - Parses user_id: "harper"
   - Adds to context: ctx.Value("user_id") = "harper"

5. Router matches route:
   - Path: /gmail/v1/users/{userId}/messages
   - Handler: googlePlugin.handleListMessages

6. Handler executes:
   - Gets user_id from context
   - Queries store.ListGmailMessages(user_id)
   - Returns JSON response

7. Response flows back through middleware:
   - Logging middleware captures status code
   - Async goroutine writes to request_logs table

8. Client receives response:
   {
     "messages": [...],
     "nextPageToken": "..."
   }
```

## Database Design

### Core Tables

**users**: User accounts
```sql
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
)
```

**request_logs**: API request logging
```sql
CREATE TABLE request_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    plugin_name TEXT,
    method TEXT,
    path TEXT,
    status_code INTEGER,
    duration_ms INTEGER,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
)
```

### Google Plugin Tables

**gmail_messages**: Email messages
```sql
CREATE TABLE gmail_messages (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    thread_id TEXT,
    label_ids TEXT,
    snippet TEXT,
    internal_date INTEGER,
    payload TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id)
)
```

**calendar_events**: Calendar events
```sql
CREATE TABLE calendar_events (
    id TEXT PRIMARY KEY,
    calendar_id TEXT NOT NULL,
    summary TEXT,
    description TEXT,
    start_time TEXT,
    end_time TEXT,
    attendees TEXT,
    FOREIGN KEY (calendar_id) REFERENCES calendars(id)
)
```

**people**: Contacts
```sql
CREATE TABLE people (
    resource_name TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    data TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id)
)
```

**tasks**: Tasks
```sql
CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    list_id TEXT NOT NULL,
    title TEXT NOT NULL,
    notes TEXT,
    due TEXT,
    status TEXT,
    completed TEXT,
    updated_at TEXT,
    FOREIGN KEY (list_id) REFERENCES task_lists(id)
)
```

### OAuth Plugin Tables

**oauth_tokens**: OAuth access tokens
```sql
CREATE TABLE oauth_tokens (
    access_token TEXT PRIMARY KEY,
    refresh_token TEXT,
    plugin_name TEXT NOT NULL,
    user_id TEXT NOT NULL,
    revoked BOOLEAN DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
)
```

## Extension Points

### Adding a New Plugin

1. Create package in `plugins/yourplugin/`
2. Implement `core.Plugin` interface
3. Call `core.Register()` in `init()`
4. Import in `cmd/ish/main.go`
5. Add database tables if needed
6. Write tests

See [Plugin Development Guide](./plugins/DEVELOPMENT.md) for details.

### Adding Middleware

Add middleware in `cmd/ish/main.go` before plugin route registration:

```go
r.Use(middleware.Logger)
r.Use(middleware.Recoverer)
r.Use(logging.Middleware(s))
r.Use(auth.Middleware)
r.Use(yourMiddleware)  // Add here
```

### Custom Admin Pages

The admin UI uses templates in `internal/admin/templates.go`. To add custom pages:

1. Add route in `internal/admin/handlers.go`
2. Create template function
3. Render with data from store

## Design Decisions

### Why Plugins?

**Extensibility**: Adding a new API (Stripe, Twilio, etc.) doesn't require modifying core code. Just drop in a new plugin.

**Modularity**: Each API is self-contained. Google plugin can be tested independently of OAuth plugin.

**Discoverability**: Plugins auto-register. Admin UI automatically shows all plugin resources.

### Why Schema-Driven UI?

**Less Code**: Plugins declare what they need, ISH renders it. No manual HTML/forms per plugin.

**Consistency**: All resources have the same UI patterns.

**Flexibility**: Plugins can define custom field types, actions, and validation.

### Why SQLite?

**Simplicity**: Single file, no separate database server.

**Inspectable**: Use `sqlite3 ish.db` to query data directly.

**Portable**: Database file can be committed to version control for reproducible tests.

**Fast Enough**: For local development, SQLite is plenty fast.

### Why Fake Auth?

**Convenience**: No need to manage real OAuth flows in development.

**Flexibility**: Switch users with `Bearer user:alice` vs `Bearer user:bob`.

**Testing**: Easy to test multi-user scenarios.

## Performance Considerations

### Request Logging

Logging happens asynchronously in a goroutine to avoid blocking request handling:

```go
go store.LogRequest(&RequestLog{...})
```

### Database Connections

SQLite uses a single connection. For higher concurrency, consider connection pooling or switching to PostgreSQL.

### Schema Rendering

Schemas are static and cached. Rendering happens once per plugin at startup.

## Security Note

**ISH is for local development only.** Do not expose it to the internet. It has:

- Fake authentication (no password verification)
- No rate limiting
- No input sanitization beyond basic validation
- No HTTPS support

Use ISH behind a firewall for testing your applications locally.

## Future Enhancements

Potential improvements:

1. **Hot Reload**: Reload plugins without restarting server
2. **Plugin Marketplace**: Share plugins across teams
3. **Webhooks**: Simulate webhook delivery
4. **Event System**: Plugins can publish/subscribe to events
5. **GraphQL Support**: Auto-generate GraphQL schemas from plugins
6. **Remote Plugins**: Load plugins over HTTP
7. **Plugin Dependencies**: Plugins can depend on other plugins

## Summary

ISH's plugin architecture makes it:

- **Easy to extend**: Add new APIs as plugins
- **Self-documenting**: Schemas describe structure
- **Observable**: Request logs show what happened
- **Testable**: Each plugin tested independently
- **Maintainable**: Clear separation of concerns

The key insight: **treating each API as a plugin** enables a clean, modular architecture where functionality can be added without touching core code.
