# Plugin Development Guide

This guide explains how to create plugins for ISH (the fake Google API emulator).

## Overview

ISH uses a plugin architecture where each API or service is implemented as a plugin. Plugins register themselves at startup, provide HTTP routes, admin UI schemas, data seeding, and OAuth integration.

## Plugin Interface

Every plugin must implement the `core.Plugin` interface defined in `plugins/core/plugin.go`:

```go
type Plugin interface {
    // Metadata
    Name() string
    Health() HealthStatus

    // HTTP Routes
    RegisterRoutes(r chi.Router)
    RegisterAuth(r chi.Router)

    // Admin UI
    Schema() PluginSchema

    // Data Generation
    Seed(ctx context.Context, size string) (SeedData, error)

    // Token Validation
    ValidateToken(token string) bool
}
```

## Plugin Storage

Plugins that need database access implement the `DatabasePlugin` interface:

```go
type DatabasePlugin interface {
    Plugin
    SetDB(db *sql.DB) error
}
```

This interface allows plugins to receive a raw `*sql.DB` connection and manage their own database tables independently. Plugins are responsible for creating and managing their own schemas.

### Best Practices

**1. Own Your Tables** - Create `plugins/yourplugin/store.go` with all database code

Each plugin should have its own store file that encapsulates all database operations. This keeps the plugin self-contained and makes it easy to extract into an external module if needed.

**2. Table Naming** - Use `{plugin}_{resource}` pattern (e.g., `discord_webhooks`, `google_messages`)

This prevents table name collisions between plugins and makes it clear which plugin owns which tables.

**3. Initialize Tables** - Create tables in `NewYourStore()` constructor

Tables should be created when the store is initialized, typically in an `initTables()` method:

```go
func NewDiscordStore(db *sql.DB) (*DiscordStore, error) {
    store := &DiscordStore{db: db}
    if err := store.initTables(); err != nil {
        return nil, err
    }
    return store, nil
}

func (s *DiscordStore) initTables() error {
    queries := []string{
        `CREATE TABLE IF NOT EXISTS discord_webhooks (...)`,
        `CREATE TABLE IF NOT EXISTS discord_messages (...)`,
    }
    for _, query := range queries {
        if _, err := s.db.Exec(query); err != nil {
            return err
        }
    }
    return nil
}
```

**4. Migrations** - Handle schema changes in `initTables()`

Use `CREATE TABLE IF NOT EXISTS` and `ALTER TABLE IF NOT EXISTS` to handle schema evolution gracefully.

**5. Self-Contained** - Keep all database logic within the plugin

Don't add plugin-specific code to `internal/store/`. The core store should only handle cross-cutting concerns like request logging.

### Example Structure

See `plugins/discord/store.go` for a reference implementation showing:
- Table initialization in `NewDiscordStore()`
- Struct definitions for database models
- CRUD methods (Create, Read, Update, Delete)
- Helper functions for ID generation
- Proper error handling and null handling

### Method Descriptions

#### `Name() string`

Returns the unique identifier for this plugin (e.g., "google", "oauth", "stripe").

This name is used for:
- Plugin registration and lookup
- Request logging (`plugin_name` field)
- OAuth flow routing (`/oauth/{plugin}/authorize`)

#### `Health() HealthStatus`

Returns the current health status of the plugin.

```go
type HealthStatus struct {
    Status  string // "healthy", "degraded", "unavailable"
    Message string
}
```

Example:
```go
func (p *MyPlugin) Health() core.HealthStatus {
    return core.HealthStatus{
        Status:  "healthy",
        Message: "My plugin operational",
    }
}
```

#### `RegisterRoutes(r chi.Router)`

Registers API endpoints for this plugin. Use chi router methods to define routes.

Example:
```go
func (p *GooglePlugin) RegisterRoutes(r chi.Router) {
    // Gmail routes
    r.Get("/gmail/v1/users/{userId}/messages", p.handleListMessages)
    r.Get("/gmail/v1/users/{userId}/messages/{id}", p.handleGetMessage)

    // Calendar routes
    r.Get("/calendar/v3/calendars/{calendarId}/events", p.handleListEvents)
    r.Post("/calendar/v3/calendars/{calendarId}/events", p.handleCreateEvent)
}
```

#### `RegisterAuth(r chi.Router)`

Registers OAuth/authentication endpoints for this plugin. Usually empty for API plugins since the OAuth plugin handles authentication.

The OAuth plugin uses this to register flow endpoints:
```go
func (p *OAuthPlugin) RegisterAuth(r chi.Router) {
    r.Get("/oauth/{plugin}/authorize", p.handleAuthorize)
    r.Post("/oauth/{plugin}/token", p.handleToken)
    r.Post("/oauth/{plugin}/revoke", p.handleRevoke)
}
```

#### `Schema() PluginSchema`

Defines the admin UI structure. Returns a schema describing resources, fields, and actions.

See [Schema-Driven UI](#schema-driven-ui) section for details.

#### `Seed(ctx context.Context, size string) (SeedData, error)`

Generates test data for this plugin. Called when running `./ish seed` or via the admin UI.

Parameters:
- `ctx`: Context for cancellation
- `size`: Data size hint (e.g., "small", "medium", "large")

Returns:
```go
type SeedData struct {
    Summary string         // Human-readable summary
    Records map[string]int // Resource counts: {"messages": 50}
}
```

Example:
```go
func (p *StripePlugin) Seed(ctx context.Context, size string) (core.SeedData, error) {
    // Create test customers
    for i := 0; i < 10; i++ {
        p.store.CreateCustomer(&Customer{...})
    }

    // Create test charges
    for i := 0; i < 25; i++ {
        p.store.CreateCharge(&Charge{...})
    }

    return core.SeedData{
        Summary: "Created 10 customers and 25 charges",
        Records: map[string]int{
            "customers": 10,
            "charges": 25,
        },
    }, nil
}
```

#### `ValidateToken(token string) bool`

Validates an OAuth access token for this plugin. Returns true if the token is valid and not revoked.

Example:
```go
func (p *OAuthPlugin) ValidateToken(token string) bool {
    t, err := p.store.GetToken(token)
    if err != nil {
        return false
    }
    return !t.Revoked
}
```

## Plugin Registration

Plugins register themselves in their `init()` function using `core.Register()`:

```go
package myplugin

import "github.com/2389/ish/plugins/core"

func init() {
    core.Register(&MyPlugin{})
}
```

This happens automatically when the package is imported:

```go
// cmd/ish/main.go
import (
    _ "github.com/2389/ish/plugins/google"  // Registers Google plugin
    _ "github.com/2389/ish/plugins/oauth"   // Registers OAuth plugin
    _ "github.com/2389/ish/plugins/stripe"  // Registers your plugin
)
```

## Schema-Driven UI

Plugins define their admin UI declaratively using schemas. The ISH admin panel automatically renders forms, lists, and actions based on these schemas.

### ResourceSchema

Each resource (e.g., Messages, Events, Customers) is defined with:

```go
type ResourceSchema struct {
    Name        string         // Display name: "Messages", "Customers"
    Slug        string         // URL path: "messages", "customers"
    Fields      []FieldSchema  // Field definitions
    Actions     []ActionSchema // Available operations
    ListColumns []string       // Which fields to show in list view
}
```

### FieldSchema

Fields define the data structure:

```go
type FieldSchema struct {
    Name     string // Field name: "email", "amount", "status"
    Type     string // Field type: "string", "email", "datetime", "text"
    Display  string // Display label: "Email Address"
    Required bool   // Is this field required?
    Editable bool   // Can users edit this field?
}
```

Supported field types:
- `string`: Short text input
- `text`: Multi-line text area
- `email`: Email input with validation
- `datetime`: Date/time picker
- `number`: Numeric input
- `boolean`: Checkbox

### ActionSchema

Actions define operations on resources:

```go
type ActionSchema struct {
    Name       string // Action name: "delete", "refund", "send"
    HTTPMethod string // "POST", "DELETE", "PATCH"
    Endpoint   string // Template: "/customers/{id}/refund"
    Confirm    bool   // Show confirmation dialog?
}
```

### Example Schema

```go
func getStripeSchema() core.PluginSchema {
    return core.PluginSchema{
        Resources: []core.ResourceSchema{
            {
                Name:        "Customers",
                Slug:        "customers",
                ListColumns: []string{"email", "name", "created"},
                Fields: []core.FieldSchema{
                    {Name: "id", Type: "string", Display: "ID", Required: false, Editable: false},
                    {Name: "email", Type: "email", Display: "Email", Required: true, Editable: true},
                    {Name: "name", Type: "string", Display: "Name", Required: true, Editable: true},
                    {Name: "created", Type: "datetime", Display: "Created", Required: false, Editable: false},
                },
                Actions: []core.ActionSchema{
                    {Name: "delete", HTTPMethod: "DELETE", Endpoint: "/v1/customers/{id}", Confirm: true},
                },
            },
            {
                Name:        "Charges",
                Slug:        "charges",
                ListColumns: []string{"amount", "customer", "status"},
                Fields: []core.FieldSchema{
                    {Name: "id", Type: "string", Display: "ID", Required: false, Editable: false},
                    {Name: "amount", Type: "number", Display: "Amount", Required: true, Editable: true},
                    {Name: "currency", Type: "string", Display: "Currency", Required: true, Editable: true},
                    {Name: "customer", Type: "string", Display: "Customer ID", Required: true, Editable: true},
                    {Name: "status", Type: "string", Display: "Status", Required: false, Editable: false},
                },
                Actions: []core.ActionSchema{
                    {Name: "refund", HTTPMethod: "POST", Endpoint: "/v1/charges/{id}/refund", Confirm: true},
                },
            },
        },
    }
}
```

## Creating a Plugin

### Step 1: Create Plugin Package

Create a new package in `plugins/`:

```bash
mkdir -p plugins/myplugin
```

### Step 2: Define the Plugin Type

Create `plugins/myplugin/plugin.go`:

```go
package myplugin

import (
    "context"
    "github.com/2389/ish/internal/store"
    "github.com/2389/ish/plugins/core"
    "github.com/go-chi/chi/v5"
)

func init() {
    core.Register(&MyPlugin{})
}

type MyPlugin struct {
    store *store.Store
}

func (p *MyPlugin) Name() string {
    return "myplugin"
}

func (p *MyPlugin) Health() core.HealthStatus {
    return core.HealthStatus{
        Status:  "healthy",
        Message: "My plugin operational",
    }
}

func (p *MyPlugin) RegisterRoutes(r chi.Router) {
    // Define your API routes
    r.Get("/v1/myresource", p.handleList)
    r.Post("/v1/myresource", p.handleCreate)
    r.Get("/v1/myresource/{id}", p.handleGet)
    r.Delete("/v1/myresource/{id}", p.handleDelete)
}

func (p *MyPlugin) RegisterAuth(r chi.Router) {
    // Usually empty - OAuth plugin handles auth
}

func (p *MyPlugin) Schema() core.PluginSchema {
    return getMyPluginSchema()
}

func (p *MyPlugin) Seed(ctx context.Context, size string) (core.SeedData, error) {
    // Generate test data
    return core.SeedData{
        Summary: "Created test data",
        Records: map[string]int{},
    }, nil
}

func (p *MyPlugin) ValidateToken(token string) bool {
    // Validate OAuth tokens if needed
    return true
}

// SetStore allows injecting the store after construction
func (p *MyPlugin) SetStore(s *store.Store) {
    p.store = s
}
```

### Step 3: Implement Handlers

Create `plugins/myplugin/handlers.go`:

```go
package myplugin

import (
    "encoding/json"
    "net/http"
    "github.com/go-chi/chi/v5"
)

func (p *MyPlugin) handleList(w http.ResponseWriter, r *http.Request) {
    // Get authenticated user from context
    userID := r.Context().Value("user_id").(string)

    // Fetch data from store
    items, err := p.store.GetMyItems(userID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    // Return JSON response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]any{
        "items": items,
    })
}

func (p *MyPlugin) handleCreate(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("user_id").(string)

    var req struct {
        Name string `json:"name"`
        Data string `json:"data"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    item, err := p.store.CreateMyItem(userID, req.Name, req.Data)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(item)
}

func (p *MyPlugin) handleGet(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    userID := r.Context().Value("user_id").(string)

    item, err := p.store.GetMyItem(userID, id)
    if err != nil {
        http.Error(w, "Not found", http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(item)
}

func (p *MyPlugin) handleDelete(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    userID := r.Context().Value("user_id").(string)

    if err := p.store.DeleteMyItem(userID, id); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}
```

### Step 4: Define Schema

Create `plugins/myplugin/schema.go`:

```go
package myplugin

import "github.com/2389/ish/plugins/core"

func getMyPluginSchema() core.PluginSchema {
    return core.PluginSchema{
        Resources: []core.ResourceSchema{
            {
                Name:        "My Resources",
                Slug:        "myresources",
                ListColumns: []string{"name", "data", "created"},
                Fields: []core.FieldSchema{
                    {Name: "id", Type: "string", Display: "ID", Required: false, Editable: false},
                    {Name: "name", Type: "string", Display: "Name", Required: true, Editable: true},
                    {Name: "data", Type: "text", Display: "Data", Required: false, Editable: true},
                    {Name: "created", Type: "datetime", Display: "Created", Required: false, Editable: false},
                },
                Actions: []core.ActionSchema{
                    {Name: "delete", HTTPMethod: "DELETE", Endpoint: "/v1/myresource/{id}", Confirm: true},
                },
            },
        },
    }
}
```

### Step 5: Register Plugin

Import the plugin in `cmd/ish/main.go`:

```go
import (
    _ "github.com/2389/ish/plugins/google"
    _ "github.com/2389/ish/plugins/oauth"
    _ "github.com/2389/ish/plugins/myplugin"  // Add your plugin
)
```

### Step 6: Add Database Tables

If your plugin needs database tables, add migrations in `internal/store/store.go`:

```go
func (s *Store) initSchema() error {
    // ... existing tables ...

    // Add your plugin's tables
    _, err = s.db.Exec(`
        CREATE TABLE IF NOT EXISTS my_items (
            id TEXT PRIMARY KEY,
            user_id TEXT NOT NULL,
            name TEXT NOT NULL,
            data TEXT,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        )
    `)
    if err != nil {
        return err
    }

    return nil
}
```

## Testing Plugins

Create `plugins/myplugin/plugin_test.go`:

```go
package myplugin_test

import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/2389/ish/internal/store"
    "github.com/2389/ish/plugins/myplugin"
    "github.com/go-chi/chi/v5"
)

func TestMyPlugin(t *testing.T) {
    // Create test store
    s, err := store.New(":memory:")
    if err != nil {
        t.Fatal(err)
    }
    defer s.Close()

    // Create plugin
    p := &myplugin.MyPlugin{}
    p.SetStore(s)

    // Create test router
    r := chi.NewRouter()
    p.RegisterRoutes(r)

    t.Run("list items", func(t *testing.T) {
        req := httptest.NewRequest("GET", "/v1/myresource", nil)
        w := httptest.NewRecorder()

        r.ServeHTTP(w, req)

        if w.Code != http.StatusOK {
            t.Errorf("expected status 200, got %d", w.Code)
        }
    })

    t.Run("create item", func(t *testing.T) {
        req := httptest.NewRequest("POST", "/v1/myresource",
            strings.NewReader(`{"name":"test","data":"test data"}`))
        req.Header.Set("Content-Type", "application/json")
        w := httptest.NewRecorder()

        r.ServeHTTP(w, req)

        if w.Code != http.StatusCreated {
            t.Errorf("expected status 201, got %d", w.Code)
        }
    })
}
```

## Best Practices

### 1. Use Dependency Injection

Don't access the store directly in `init()`. Instead, provide a `SetStore()` method:

```go
type MyPlugin struct {
    store *store.Store
}

func (p *MyPlugin) SetStore(s *store.Store) {
    p.store = s
}
```

The main server calls this after plugin registration:

```go
for _, plugin := range core.All() {
    if sp, ok := plugin.(interface{ SetStore(*store.Store) }); ok {
        sp.SetStore(s)
    }
}
```

### 2. Extract User from Context

Authentication middleware adds the user ID to the request context:

```go
userID := r.Context().Value("user_id").(string)
```

Always scope data to the authenticated user.

### 3. Return Standard JSON Responses

Follow Google API conventions:

```go
// List responses
{
    "items": [...],
    "nextPageToken": "...",
}

// Single resource
{
    "id": "...",
    "name": "...",
    ...
}

// Errors
{
    "error": {
        "message": "Resource not found",
        "code": 404
    }
}
```

### 4. Handle Pagination

For list endpoints, support pagination parameters:

```go
func (p *MyPlugin) handleList(w http.ResponseWriter, r *http.Request) {
    pageToken := r.URL.Query().Get("pageToken")
    maxResults := 100
    if mr := r.URL.Query().Get("maxResults"); mr != "" {
        maxResults, _ = strconv.Atoi(mr)
    }

    items, nextToken, err := p.store.ListItems(userID, pageToken, maxResults)
    // ...
}
```

### 5. Support Incremental Sync

Implement sync tokens for efficient data synchronization:

```go
{
    "items": [...],
    "syncToken": "abc123"  // Include in next request to get only changes
}
```

### 6. Write Comprehensive Tests

Test all CRUD operations, error cases, and edge cases:

- List empty collection
- Create with valid/invalid data
- Get existing/non-existent resource
- Update with partial data
- Delete and verify deletion
- Pagination edge cases
- Authentication failures

### 7. Document Your API

Add comments to exported types and functions:

```go
// MyPlugin provides a mock implementation of the MyService API
// for local development and testing.
type MyPlugin struct {
    store *store.Store
}

// handleCreate creates a new resource.
// POST /v1/myresource
// Request body: {"name": "...", "data": "..."}
func (p *MyPlugin) handleCreate(w http.ResponseWriter, r *http.Request) {
    // ...
}
```

## Example Plugins

See these plugins for reference implementations:

- **Google Plugin** (`plugins/google/`): Comprehensive example with Gmail, Calendar, People, and Tasks APIs
- **OAuth Plugin** (`plugins/oauth/`): Token management and OAuth flow
- **Stripe Plugin Example** (`docs/plugins/example-stripe-plugin.md`): Complete example showing payment processing

## Request Logging

All requests are automatically logged with plugin attribution. The logging middleware captures:

- Plugin name
- HTTP method and path
- Status code
- Response time
- Timestamp

Logs are stored in the database and viewable in the admin UI at `/admin/logs`.

## OAuth Integration

If your plugin needs OAuth authentication:

1. The OAuth plugin handles the flow (`/oauth/{plugin}/authorize`, `/oauth/{plugin}/token`)
2. Tokens are stored with `plugin_name` field
3. Implement `ValidateToken()` to check token validity
4. Middleware validates tokens before routing to your handlers

See `plugins/oauth/` for the complete implementation.

## Troubleshooting

### Plugin Not Registered

Ensure you:
1. Called `core.Register()` in `init()`
2. Imported the plugin package in `cmd/ish/main.go` with `_` blank import
3. Used the correct plugin name

### Routes Not Working

Check:
1. Route registration in `RegisterRoutes()`
2. Path patterns match your API design
3. HTTP methods are correct
4. Authentication middleware is allowing requests

### Database Errors

Verify:
1. Tables are created in `store.initSchema()`
2. Store is injected via `SetStore()`
3. SQL queries are correct
4. Foreign key constraints are satisfied

## Further Reading

- [Example Stripe Plugin](./example-stripe-plugin.md) - Complete working example
- [Architecture Documentation](../ARCHITECTURE.md) - System design overview
- [Chi Router Documentation](https://github.com/go-chi/chi) - HTTP routing
