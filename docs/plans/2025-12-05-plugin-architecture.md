# Plugin Architecture Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor ISH from monolithic Google API server into a plugin-based platform where each API (Google, Discord, Stripe, etc.) is a self-contained plugin with its own routes, auth, schema, and seeding.

**Architecture:** Create plugin interface in `plugins/core`, extract existing Google code into `plugins/google`, build schema-driven admin UI renderer that generates HTML from plugin schemas, add unified OAuth token management, update logging to track plugin names.

**Tech Stack:** Go 1.21+, chi router, SQLite, existing HTMX/Tailwind admin UI

---

## Phase 1: Plugin Infrastructure

### Task 1: Create Plugin Interface

**Files:**
- Create: `plugins/core/plugin.go`
- Create: `plugins/core/registry.go`
- Create: `plugins/core/schema.go`

**Step 1: Write plugin interface**

Create `plugins/core/plugin.go`:

```go
// ABOUTME: Core plugin interface for ISH plugin system.
// ABOUTME: Defines contract that all plugins must implement.

package core

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Plugin defines the interface that all ISH plugins must implement
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

// HealthStatus represents plugin health
type HealthStatus struct {
	Status  string // "healthy", "degraded", "unavailable"
	Message string
}

// SeedData represents data generation results
type SeedData struct {
	Summary string         // Human-readable summary
	Records map[string]int // Resource counts: {"messages": 50}
}
```

**Step 2: Write plugin registry**

Create `plugins/core/registry.go`:

```go
// ABOUTME: Plugin registry for registering and retrieving plugins.
// ABOUTME: Plugins register themselves in init() functions.

package core

import (
	"fmt"
	"sync"
)

var (
	registry = make(map[string]Plugin)
	mu       sync.RWMutex
)

// Register adds a plugin to the registry
func Register(p Plugin) {
	mu.Lock()
	defer mu.Unlock()

	name := p.Name()
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("plugin %q already registered", name))
	}
	registry[name] = p
}

// Get retrieves a plugin by name
func Get(name string) (Plugin, bool) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := registry[name]
	return p, ok
}

// All returns all registered plugins
func All() []Plugin {
	mu.RLock()
	defer mu.RUnlock()

	plugins := make([]Plugin, 0, len(registry))
	for _, p := range registry {
		plugins = append(plugins, p)
	}
	return plugins
}

// Names returns all registered plugin names
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
```

**Step 3: Write schema definitions**

Create `plugins/core/schema.go`:

```go
// ABOUTME: Schema definitions for plugin UI generation.
// ABOUTME: Plugins define schemas, ISH core renders the UI.

package core

// PluginSchema defines the admin UI for a plugin
type PluginSchema struct {
	Resources []ResourceSchema
}

// ResourceSchema defines a resource (Messages, Events, etc.)
type ResourceSchema struct {
	Name        string         // "Messages", "Events"
	Slug        string         // "messages", "events" (URL path)
	Fields      []FieldSchema  // What data to show/edit
	Actions     []ActionSchema // Available operations
	ListColumns []string       // Which fields in list view
}

// FieldSchema defines a field in a resource
type FieldSchema struct {
	Name     string // "subject", "from", "date"
	Type     string // "string", "datetime", "email", "text"
	Display  string // "Subject", "From", "Date"
	Required bool
	Editable bool
}

// ActionSchema defines an action on a resource
type ActionSchema struct {
	Name       string // "send", "delete", "archive"
	HTTPMethod string // "POST", "DELETE"
	Endpoint   string // Template: "/messages/{id}/send"
	Confirm    bool   // Show confirmation dialog?
}
```

**Step 4: Build to verify**

Run: `go build -o ish ./cmd/ish`
Expected: SUCCESS (no compilation errors)

**Step 5: Commit**

```bash
git add plugins/core/
git commit -m "feat(plugins): add plugin interface and registry"
```

---

### Task 2: Update Database Schema for Plugins

**Files:**
- Modify: `internal/store/store.go`

**Step 1: Add plugin_name to request_logs**

In `internal/store/store.go`, update the migrations in `New()` function:

Add after existing `request_logs` table creation:

```go
`ALTER TABLE request_logs ADD COLUMN plugin_name TEXT DEFAULT ''`,
```

**Step 2: Create oauth_tokens table**

Add to migrations in `New()` function:

```go
`CREATE TABLE IF NOT EXISTS oauth_tokens (
	token TEXT PRIMARY KEY,
	plugin_name TEXT NOT NULL,
	user_id TEXT,
	scopes TEXT,
	expires_at TIMESTAMP,
	refresh_token TEXT,
	revoked BOOLEAN DEFAULT 0,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)`,
`CREATE INDEX IF NOT EXISTS idx_oauth_tokens_plugin ON oauth_tokens(plugin_name)`,
`CREATE INDEX IF NOT EXISTS idx_oauth_tokens_user ON oauth_tokens(user_id)`,
```

**Step 3: Update RequestLog struct**

In `internal/store/request_logs.go`, add field to `RequestLog`:

```go
type RequestLog struct {
	ID           int64
	Timestamp    time.Time
	PluginName   string  // NEW
	Method       string
	Path         string
	StatusCode   int
	DurationMs   int
	UserID       string
	IPAddress    string
	UserAgent    string
	Error        string
	RequestBody  string
	ResponseBody string
}
```

**Step 4: Update LogRequest to include plugin_name**

In `internal/store/request_logs.go`, update `LogRequest`:

```go
func (s *Store) LogRequest(log *RequestLog) error {
	_, err := s.db.Exec(`
		INSERT INTO request_logs (plugin_name, method, path, status_code, duration_ms, user_id, ip_address, user_agent, error, request_body, response_body)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, log.PluginName, log.Method, log.Path, log.StatusCode, log.DurationMs, log.UserID, log.IPAddress, log.UserAgent, log.Error, log.RequestBody, log.ResponseBody)
	return err
}
```

**Step 5: Update GetRequestLogs to include plugin_name**

In `internal/store/request_logs.go`, update `GetRequestLogs`:

```go
func (s *Store) GetRequestLogs(q *RequestLogQuery) ([]*RequestLog, error) {
	query := `SELECT id, timestamp, COALESCE(plugin_name, ''), method, path, status_code, duration_ms,
	          COALESCE(user_id, ''), COALESCE(ip_address, ''), COALESCE(user_agent, ''), COALESCE(error, ''),
	          COALESCE(request_body, ''), COALESCE(response_body, '')
	          FROM request_logs WHERE 1=1`
	args := []any{}

	if q.Method != "" {
		query += " AND method = ?"
		args = append(args, q.Method)
	}
	if q.PathPrefix != "" {
		query += " AND path LIKE ?"
		args = append(args, q.PathPrefix+"%")
	}
	if q.StatusCode > 0 {
		query += " AND status_code = ?"
		args = append(args, q.StatusCode)
	}
	if q.UserID != "" {
		query += " AND user_id = ?"
		args = append(args, q.UserID)
	}

	query += " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	args = append(args, q.Limit, q.Offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*RequestLog
	for rows.Next() {
		log := &RequestLog{}
		var timestamp string
		if err := rows.Scan(&log.ID, &timestamp, &log.PluginName, &log.Method, &log.Path, &log.StatusCode,
			&log.DurationMs, &log.UserID, &log.IPAddress, &log.UserAgent, &log.Error,
			&log.RequestBody, &log.ResponseBody); err != nil {
			return nil, err
		}
		log.Timestamp, _ = time.Parse("2006-01-02 15:04:05", timestamp)
		logs = append(logs, log)
	}
	return logs, nil
}
```

**Step 6: Build to verify**

Run: `go build -o ish ./cmd/ish`
Expected: SUCCESS

**Step 7: Commit**

```bash
git add internal/store/
git commit -m "feat(store): add plugin_name to request_logs and create oauth_tokens table"
```

---

### Task 3: Create OAuth Token Management

**Files:**
- Create: `internal/store/oauth_tokens.go`

**Step 1: Write token structs and methods**

Create `internal/store/oauth_tokens.go`:

```go
// ABOUTME: OAuth token storage and management.
// ABOUTME: Handles token lifecycle (create, validate, revoke).

package store

import (
	"time"
)

// OAuthToken represents an OAuth token
type OAuthToken struct {
	Token        string
	PluginName   string
	UserID       string
	Scopes       string
	ExpiresAt    time.Time
	RefreshToken string
	Revoked      bool
	CreatedAt    time.Time
}

// StoreToken stores a new OAuth token
func (s *Store) StoreToken(token *OAuthToken) error {
	_, err := s.db.Exec(`
		INSERT INTO oauth_tokens (token, plugin_name, user_id, scopes, expires_at, refresh_token, revoked)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, token.Token, token.PluginName, token.UserID, token.Scopes, token.ExpiresAt, token.RefreshToken, token.Revoked)
	return err
}

// GetToken retrieves a token by value
func (s *Store) GetToken(token string) (*OAuthToken, error) {
	t := &OAuthToken{}
	err := s.db.QueryRow(`
		SELECT token, plugin_name, user_id, COALESCE(scopes, ''), expires_at, COALESCE(refresh_token, ''), revoked, created_at
		FROM oauth_tokens WHERE token = ?
	`, token).Scan(&t.Token, &t.PluginName, &t.UserID, &t.Scopes, &t.ExpiresAt, &t.RefreshToken, &t.Revoked, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// RevokeToken marks a token as revoked
func (s *Store) RevokeToken(token string) error {
	_, err := s.db.Exec(`UPDATE oauth_tokens SET revoked = 1 WHERE token = ?`, token)
	return err
}

// ListTokens retrieves all tokens (optionally filtered by plugin)
func (s *Store) ListTokens(pluginName string) ([]*OAuthToken, error) {
	query := `SELECT token, plugin_name, user_id, COALESCE(scopes, ''), expires_at, COALESCE(refresh_token, ''), revoked, created_at
	          FROM oauth_tokens WHERE 1=1`
	args := []any{}

	if pluginName != "" {
		query += " AND plugin_name = ?"
		args = append(args, pluginName)
	}

	query += " ORDER BY created_at DESC LIMIT 100"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*OAuthToken
	for rows.Next() {
		t := &OAuthToken{}
		if err := rows.Scan(&t.Token, &t.PluginName, &t.UserID, &t.Scopes, &t.ExpiresAt, &t.RefreshToken, &t.Revoked, &t.CreatedAt); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, nil
}
```

**Step 2: Build to verify**

Run: `go build -o ish ./cmd/ish`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add internal/store/oauth_tokens.go
git commit -m "feat(store): add OAuth token management"
```

---

## Phase 2: Extract Google Plugin

### Task 4: Create Google Plugin Structure

**Files:**
- Create: `plugins/google/plugin.go`
- Create: `plugins/google/schema.go`

**Step 1: Create plugin.go**

Create `plugins/google/plugin.go`:

```go
// ABOUTME: Google plugin for ISH.
// ABOUTME: Provides Gmail, Calendar, People, and Tasks APIs.

package google

import (
	"context"

	"github.com/2389/ish/internal/store"
	"github.com/2389/ish/plugins/core"
	"github.com/go-chi/chi/v5"
)

func init() {
	core.Register(&GooglePlugin{})
}

type GooglePlugin struct {
	store *store.Store
}

func (p *GooglePlugin) Name() string {
	return "google"
}

func (p *GooglePlugin) Health() core.HealthStatus {
	return core.HealthStatus{
		Status:  "healthy",
		Message: "Google plugin operational",
	}
}

func (p *GooglePlugin) RegisterRoutes(r chi.Router) {
	// TODO: Register Gmail, Calendar, People, Tasks routes
}

func (p *GooglePlugin) RegisterAuth(r chi.Router) {
	// TODO: Register OAuth endpoints
}

func (p *GooglePlugin) Schema() core.PluginSchema {
	return getGoogleSchema()
}

func (p *GooglePlugin) Seed(ctx context.Context, size string) (core.SeedData, error) {
	// TODO: Implement seeding
	return core.SeedData{
		Summary: "Not yet implemented",
		Records: map[string]int{},
	}, nil
}

func (p *GooglePlugin) ValidateToken(token string) bool {
	// TODO: Implement token validation
	return true
}

// SetStore allows injecting the store after construction
func (p *GooglePlugin) SetStore(s *store.Store) {
	p.store = s
}
```

**Step 2: Create schema.go**

Create `plugins/google/schema.go`:

```go
// ABOUTME: Google plugin schema definitions.
// ABOUTME: Defines UI structure for Gmail, Calendar, People, Tasks.

package google

import "github.com/2389/ish/plugins/core"

func getGoogleSchema() core.PluginSchema {
	return core.PluginSchema{
		Resources: []core.ResourceSchema{
			{
				Name:        "Messages",
				Slug:        "messages",
				ListColumns: []string{"subject", "from", "date"},
				Fields: []core.FieldSchema{
					{Name: "id", Type: "string", Display: "ID", Required: false, Editable: false},
					{Name: "subject", Type: "string", Display: "Subject", Required: true, Editable: true},
					{Name: "from", Type: "email", Display: "From", Required: true, Editable: true},
					{Name: "to", Type: "email", Display: "To", Required: true, Editable: true},
					{Name: "body", Type: "text", Display: "Body", Required: false, Editable: true},
					{Name: "date", Type: "datetime", Display: "Date", Required: false, Editable: false},
				},
				Actions: []core.ActionSchema{
					{Name: "delete", HTTPMethod: "DELETE", Endpoint: "/gmail/v1/users/me/messages/{id}", Confirm: true},
				},
			},
			{
				Name:        "Events",
				Slug:        "events",
				ListColumns: []string{"summary", "start", "end"},
				Fields: []core.FieldSchema{
					{Name: "id", Type: "string", Display: "ID", Required: false, Editable: false},
					{Name: "summary", Type: "string", Display: "Summary", Required: true, Editable: true},
					{Name: "description", Type: "text", Display: "Description", Required: false, Editable: true},
					{Name: "start", Type: "datetime", Display: "Start", Required: true, Editable: true},
					{Name: "end", Type: "datetime", Display: "End", Required: true, Editable: true},
					{Name: "location", Type: "string", Display: "Location", Required: false, Editable: true},
				},
				Actions: []core.ActionSchema{
					{Name: "delete", HTTPMethod: "DELETE", Endpoint: "/calendar/v3/calendars/primary/events/{id}", Confirm: true},
				},
			},
			{
				Name:        "Contacts",
				Slug:        "contacts",
				ListColumns: []string{"name", "email", "phone"},
				Fields: []core.FieldSchema{
					{Name: "id", Type: "string", Display: "ID", Required: false, Editable: false},
					{Name: "name", Type: "string", Display: "Name", Required: true, Editable: true},
					{Name: "email", Type: "email", Display: "Email", Required: false, Editable: true},
					{Name: "phone", Type: "string", Display: "Phone", Required: false, Editable: true},
					{Name: "company", Type: "string", Display: "Company", Required: false, Editable: true},
				},
				Actions: []core.ActionSchema{
					{Name: "delete", HTTPMethod: "DELETE", Endpoint: "/people/v1/people/{id}", Confirm: true},
				},
			},
			{
				Name:        "Tasks",
				Slug:        "tasks",
				ListColumns: []string{"title", "due", "status"},
				Fields: []core.FieldSchema{
					{Name: "id", Type: "string", Display: "ID", Required: false, Editable: false},
					{Name: "title", Type: "string", Display: "Title", Required: true, Editable: true},
					{Name: "notes", Type: "text", Display: "Notes", Required: false, Editable: true},
					{Name: "due", Type: "datetime", Display: "Due", Required: false, Editable: true},
					{Name: "status", Type: "string", Display: "Status", Required: true, Editable: true},
				},
				Actions: []core.ActionSchema{
					{Name: "complete", HTTPMethod: "PATCH", Endpoint: "/tasks/v1/lists/@default/tasks/{id}", Confirm: false},
					{Name: "delete", HTTPMethod: "DELETE", Endpoint: "/tasks/v1/lists/@default/tasks/{id}", Confirm: true},
				},
			},
		},
	}
}
```

**Step 3: Build to verify**

Run: `go build -o ish ./cmd/ish`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add plugins/google/
git commit -m "feat(plugins): add Google plugin structure and schema"
```

---

### Task 5: Move Gmail Handlers to Google Plugin

**Files:**
- Create: `plugins/google/gmail.go`
- Modify: `plugins/google/plugin.go`
- Modify: `internal/gmail/handlers.go` (will be deleted later)

**Step 1: Copy Gmail handlers**

Create `plugins/google/gmail.go` by copying from `internal/gmail/handlers.go`:

```go
// ABOUTME: Gmail API handlers for Google plugin.
// ABOUTME: Implements Gmail v1 API endpoints.

package google

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/2389/ish/internal/auth"
	"github.com/2389/ish/internal/store"
	"github.com/go-chi/chi/v5"
)

func (p *GooglePlugin) registerGmailRoutes(r chi.Router) {
	r.Route("/gmail/v1/users/{userId}", func(r chi.Router) {
		r.Get("/profile", p.getProfile)
		r.Get("/messages", p.listMessages)
		r.Post("/messages/send", p.sendMessage)
		r.Get("/messages/{messageId}", p.getMessage)
		r.Get("/messages/{messageId}/attachments/{attachmentId}", p.getAttachment)
		r.Get("/history", p.listHistory)
		r.Post("/messages/{messageId}/modify", p.modifyMessage)
		r.Post("/messages/{messageId}/trash", p.trashMessage)
		r.Delete("/messages/{messageId}", p.deleteMessage)
	})
}

// Copy all handler methods from internal/gmail/handlers.go
// (listMessages, getMessage, sendMessage, etc.)
// Replace h.store with p.store
```

Note: The actual copying will be done in implementation. This is a placeholder showing structure.

**Step 2: Update plugin.go to register Gmail routes**

In `plugins/google/plugin.go`, update `RegisterRoutes`:

```go
func (p *GooglePlugin) RegisterRoutes(r chi.Router) {
	r.Route("/gmail/v1", p.registerGmailRoutes)
	// TODO: Calendar, People, Tasks routes in next tasks
}
```

**Step 3: Build to verify**

Run: `go build -o ish ./cmd/ish`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add plugins/google/gmail.go plugins/google/plugin.go
git commit -m "feat(plugins/google): add Gmail handlers"
```

---

### Task 6: Move Calendar, People, Tasks to Google Plugin

**Files:**
- Create: `plugins/google/calendar.go`
- Create: `plugins/google/people.go`
- Create: `plugins/google/tasks.go`
- Modify: `plugins/google/plugin.go`

**Step 1: Create calendar.go**

Copy from `internal/calendar/handlers.go` to `plugins/google/calendar.go`:

```go
// ABOUTME: Calendar API handlers for Google plugin.
// ABOUTME: Implements Calendar v3 API endpoints.

package google

import (
	"github.com/go-chi/chi/v5"
)

func (p *GooglePlugin) registerCalendarRoutes(r chi.Router) {
	r.Route("/calendar/v3/calendars/{calendarId}", func(r chi.Router) {
		r.Get("/events", p.listEvents)
		r.Post("/events", p.createEvent)
		r.Get("/events/{eventId}", p.getEvent)
		r.Put("/events/{eventId}", p.updateEvent)
		r.Patch("/events/{eventId}", p.updateEvent)
		r.Delete("/events/{eventId}", p.deleteEvent)
	})

	// Alias routes without /calendar/v3/ prefix
	r.Route("/calendars/{calendarId}", func(r chi.Router) {
		r.Get("/events", p.listEvents)
		r.Post("/events", p.createEvent)
		r.Get("/events/{eventId}", p.getEvent)
		r.Put("/events/{eventId}", p.updateEvent)
		r.Patch("/events/{eventId}", p.updateEvent)
		r.Delete("/events/{eventId}", p.deleteEvent)
	})
}

// Copy handler methods from internal/calendar/handlers.go
```

**Step 2: Create people.go**

Copy from `internal/people/handlers.go` to `plugins/google/people.go`:

```go
// ABOUTME: People API handlers for Google plugin.
// ABOUTME: Implements People v1 API endpoints.

package google

import (
	"github.com/go-chi/chi/v5"
)

func (p *GooglePlugin) registerPeopleRoutes(r chi.Router) {
	r.Get("/v1/people/me/connections", p.listConnections)
	r.Get("/v1/people/{resourceId}", p.getPerson)
	r.Get("/v1/people:searchContacts", p.searchContacts)
	r.Get("/people/v1/people:searchContacts", p.searchContacts)
	r.Get("/people/v1/people/{resourceId}", p.getPerson)
	r.Post("/people/v1/people", p.createPerson)
	r.Patch("/people/v1/people/{resourceId}", p.updatePerson)
	r.Delete("/people/v1/people/{resourceId}", p.deletePerson)
}

// Copy handler methods from internal/people/handlers.go
```

**Step 3: Create tasks.go**

Copy from `internal/tasks/handlers.go` to `plugins/google/tasks.go`:

```go
// ABOUTME: Tasks API handlers for Google plugin.
// ABOUTME: Implements Tasks v1 API endpoints.

package google

import (
	"github.com/go-chi/chi/v5"
)

func (p *GooglePlugin) registerTasksRoutes(r chi.Router) {
	r.Route("/tasks/v1", func(r chi.Router) {
		r.Get("/users/@me/lists", p.listTaskLists)
		r.Get("/lists/{tasklist}/tasks", p.listTasks)
		r.Post("/lists/{tasklist}/tasks", p.createTask)
		r.Get("/lists/{tasklist}/tasks/{task}", p.getTask)
		r.Patch("/lists/{tasklist}/tasks/{task}", p.updateTask)
		r.Delete("/lists/{tasklist}/tasks/{task}", p.deleteTask)
	})
}

// Copy handler methods from internal/tasks/handlers.go
```

**Step 4: Update plugin.go**

In `plugins/google/plugin.go`, update `RegisterRoutes`:

```go
func (p *GooglePlugin) RegisterRoutes(r chi.Router) {
	p.registerGmailRoutes(r)
	p.registerCalendarRoutes(r)
	p.registerPeopleRoutes(r)
	p.registerTasksRoutes(r)
}
```

**Step 5: Build to verify**

Run: `go build -o ish ./cmd/ish`
Expected: SUCCESS

**Step 6: Commit**

```bash
git add plugins/google/
git commit -m "feat(plugins/google): add Calendar, People, Tasks handlers"
```

---

## Phase 3: Update Main Server to Use Plugins

### Task 7: Update Main Server

**Files:**
- Modify: `cmd/ish/main.go`
- Create: `internal/logging/plugin.go`

**Step 1: Import Google plugin**

In `cmd/ish/main.go`, add import:

```go
import (
	// ... existing imports ...
	_ "github.com/2389/ish/plugins/google"  // Register Google plugin
	"github.com/2389/ish/plugins/core"
)
```

**Step 2: Update newServer to use plugins**

In `cmd/ish/main.go`, replace the direct handler registration with plugin registration:

Replace:
```go
// API handlers
gmail.NewHandlers(s).RegisterRoutes(r)
calendar.NewHandlers(s).RegisterRoutes(r)
people.NewHandlers(s).RegisterRoutes(r)
tasks.NewHandlers(s).RegisterRoutes(r)
admin.NewHandlers(s).RegisterRoutes(r)
```

With:
```go
// Initialize all plugins with store
for _, plugin := range core.All() {
	if gp, ok := plugin.(*google.GooglePlugin); ok {
		gp.SetStore(s)
	}
	plugin.RegisterAuth(r)
	plugin.RegisterRoutes(r)
}

// Admin UI
admin.NewHandlers(s).RegisterRoutes(r)
```

**Step 3: Add plugin detection to logging middleware**

Create `internal/logging/plugin.go`:

```go
// ABOUTME: Plugin detection for request logging.
// ABOUTME: Determines which plugin a request belongs to based on URL path.

package logging

import "strings"

// GetPluginFromPath determines which plugin handles a given path
func GetPluginFromPath(path string) string {
	// Google APIs
	if strings.HasPrefix(path, "/gmail/") {
		return "google"
	}
	if strings.HasPrefix(path, "/calendar/") || strings.HasPrefix(path, "/calendars/") {
		return "google"
	}
	if strings.HasPrefix(path, "/people/") || strings.HasPrefix(path, "/v1/people") {
		return "google"
	}
	if strings.HasPrefix(path, "/tasks/") {
		return "google"
	}
	if strings.HasPrefix(path, "/oauth/google/") {
		return "google"
	}

	// Unknown
	return "unknown"
}
```

**Step 4: Update logging middleware to use plugin detection**

In `internal/logging/middleware.go`, update to use plugin detection:

```go
func Middleware(s *store.Store) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip logging for health checks and admin UI assets
			if r.URL.Path == "/healthz" || strings.HasPrefix(r.URL.Path, "/admin/") {
				next.ServeHTTP(w, r)
				return
			}

			// Determine plugin
			pluginName := GetPluginFromPath(r.URL.Path)

			// Capture request body (if present)
			var requestBody string
			if r.Body != nil {
				bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, maxBodySize))
				if err == nil {
					requestBody = string(bodyBytes)
					r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				}
			}

			start := time.Now()
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     200,
				body:           &bytes.Buffer{},
			}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start).Milliseconds()
			userID := auth.UserFromContext(r.Context())
			ip := r.RemoteAddr
			if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
				ip = strings.Split(forwarded, ",")[0]
			}

			// Log to database (fire and forget)
			go s.LogRequest(&store.RequestLog{
				PluginName:   pluginName,  // NEW
				Method:       r.Method,
				Path:         r.URL.Path,
				StatusCode:   wrapped.statusCode,
				DurationMs:   int(duration),
				UserID:       userID,
				IPAddress:    ip,
				UserAgent:    r.Header.Get("User-Agent"),
				RequestBody:  requestBody,
				ResponseBody: wrapped.body.String(),
			})
		})
	}
}
```

**Step 5: Build to verify**

Run: `go build -o ish ./cmd/ish`
Expected: SUCCESS

**Step 6: Test server starts**

Run: `./ish serve --port 9001 &`
Run: `curl http://localhost:9001/healthz`
Expected: `{"ok":true}`
Run: `pkill -f "ish serve"`

**Step 7: Commit**

```bash
git add cmd/ish/main.go internal/logging/
git commit -m "feat: integrate plugin system into main server"
```

---

## Phase 4: Schema-Driven Admin UI

### Task 8: Create Schema Renderer

**Files:**
- Create: `internal/admin/schema_renderer.go`

**Step 1: Write schema renderer**

Create `internal/admin/schema_renderer.go`:

```go
// ABOUTME: Schema-driven UI renderer for plugins.
// ABOUTME: Generates HTML from plugin schemas without requiring HTML in plugins.

package admin

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"

	"github.com/2389/ish/plugins/core"
)

// RenderResourceList renders a list view from schema
func RenderResourceList(w http.ResponseWriter, plugin core.Plugin, resource core.ResourceSchema, data []map[string]any) {
	tmpl := `
{{define "content"}}
<div class="space-y-6">
    <div class="flex justify-between items-center">
        <h1 class="text-2xl font-bold text-gray-900">{{.ResourceName}}</h1>
        <a href="/admin/plugins/{{.PluginName}}/{{.ResourceSlug}}/new" class="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700">
            Create New
        </a>
    </div>

    <div class="bg-white rounded-lg shadow overflow-hidden">
        <table class="min-w-full divide-y divide-gray-200">
            <thead class="bg-gray-50">
                <tr>
                    {{range .Columns}}
                    <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">{{.}}</th>
                    {{end}}
                    <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Actions</th>
                </tr>
            </thead>
            <tbody class="bg-white divide-y divide-gray-200">
                {{range .Data}}
                <tr class="hover:bg-gray-50">
                    {{range $.ColumnKeys}}
                    <td class="px-4 py-3 text-sm text-gray-900">{{index $ .}}</td>
                    {{end}}
                    <td class="px-4 py-3 text-sm">
                        <a href="/admin/plugins/{{$.PluginName}}/{{$.ResourceSlug}}/{{.id}}" class="text-blue-600 hover:text-blue-800">View</a>
                    </td>
                </tr>
                {{end}}
            </tbody>
        </table>
    </div>
</div>
{{end}}
`

	t := template.Must(templates.Clone())
	t = template.Must(t.Parse(tmpl))

	buf := &bytes.Buffer{}
	err := t.Execute(buf, map[string]any{
		"PluginName":   plugin.Name(),
		"ResourceName": resource.Name,
		"ResourceSlug": resource.Slug,
		"Columns":      resource.ListColumns,
		"ColumnKeys":   resource.ListColumns,
		"Data":         data,
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write(buf.Bytes())
}

// RenderResourceForm renders a create/edit form from schema
func RenderResourceForm(w http.ResponseWriter, plugin core.Plugin, resource core.ResourceSchema, data map[string]any) {
	tmpl := `
{{define "content"}}
<div class="space-y-6">
    <h1 class="text-2xl font-bold text-gray-900">{{if .IsEdit}}Edit{{else}}Create{{end}} {{.ResourceName}}</h1>

    <form hx-post="/admin/plugins/{{.PluginName}}/{{.ResourceSlug}}" class="bg-white rounded-lg shadow p-6 space-y-4">
        {{range .Fields}}
        {{if .Editable}}
        <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">{{.Display}}{{if .Required}} *{{end}}</label>
            {{if eq .Type "text"}}
            <textarea name="{{.Name}}" rows="4" class="w-full border border-gray-300 rounded px-3 py-2" {{if .Required}}required{{end}}>{{index $.Data .Name}}</textarea>
            {{else if eq .Type "datetime"}}
            <input type="datetime-local" name="{{.Name}}" class="w-full border border-gray-300 rounded px-3 py-2" value="{{index $.Data .Name}}" {{if .Required}}required{{end}}>
            {{else}}
            <input type="{{.Type}}" name="{{.Name}}" class="w-full border border-gray-300 rounded px-3 py-2" value="{{index $.Data .Name}}" {{if .Required}}required{{end}}>
            {{end}}
        </div>
        {{end}}
        {{end}}

        <div class="flex gap-4">
            <button type="submit" class="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700">
                {{if .IsEdit}}Update{{else}}Create{{end}}
            </button>
            <a href="/admin/plugins/{{.PluginName}}/{{.ResourceSlug}}" class="bg-gray-200 text-gray-800 px-4 py-2 rounded hover:bg-gray-300">
                Cancel
            </a>
        </div>
    </form>
</div>
{{end}}
`

	t := template.Must(templates.Clone())
	t = template.Must(t.Parse(tmpl))

	buf := &bytes.Buffer{}
	err := t.Execute(buf, map[string]any{
		"PluginName":   plugin.Name(),
		"ResourceName": resource.Name,
		"ResourceSlug": resource.Slug,
		"Fields":       resource.Fields,
		"Data":         data,
		"IsEdit":       len(data) > 0,
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write(buf.Bytes())
}
```

**Step 2: Build to verify**

Run: `go build -o ish ./cmd/ish`
Expected: SUCCESS

**Step 3: Commit**

```bash
git add internal/admin/schema_renderer.go
git commit -m "feat(admin): add schema-driven UI renderer"
```

---

### Task 9: Wire Schema Renderer to Admin Routes

**Files:**
- Create: `internal/admin/plugin_routes.go`
- Modify: `internal/admin/handlers.go`

**Step 1: Create plugin routes handler**

Create `internal/admin/plugin_routes.go`:

```go
// ABOUTME: Admin routes for plugin resources.
// ABOUTME: Dynamically generates CRUD routes based on plugin schemas.

package admin

import (
	"fmt"
	"net/http"

	"github.com/2389/ish/plugins/core"
	"github.com/go-chi/chi/v5"
)

// RegisterPluginRoutes registers admin routes for all plugins
func RegisterPluginRoutes(r chi.Router, plugins []core.Plugin) {
	for _, plugin := range plugins {
		schema := plugin.Schema()
		for _, resource := range schema.Resources {
			pluginName := plugin.Name()
			resourceSlug := resource.Slug

			// List
			r.Get(fmt.Sprintf("/admin/plugins/%s/%s", pluginName, resourceSlug), func(w http.ResponseWriter, r *http.Request) {
				// TODO: Fetch data from plugin/store
				data := []map[string]any{}
				RenderResourceList(w, plugin, resource, data)
			})

			// Create form
			r.Get(fmt.Sprintf("/admin/plugins/%s/%s/new", pluginName, resourceSlug), func(w http.ResponseWriter, r *http.Request) {
				RenderResourceForm(w, plugin, resource, map[string]any{})
			})

			// View/Edit form
			r.Get(fmt.Sprintf("/admin/plugins/%s/%s/{id}", pluginName, resourceSlug), func(w http.ResponseWriter, r *http.Request) {
				id := chi.URLParam(r, "id")
				// TODO: Fetch data for this ID
				data := map[string]any{"id": id}
				RenderResourceForm(w, plugin, resource, data)
			})

			// Create/Update
			r.Post(fmt.Sprintf("/admin/plugins/%s/%s", pluginName, resourceSlug), func(w http.ResponseWriter, r *http.Request) {
				// TODO: Handle create/update
				http.Redirect(w, r, fmt.Sprintf("/admin/plugins/%s/%s", pluginName, resourceSlug), http.StatusSeeOther)
			})

			// Delete
			r.Delete(fmt.Sprintf("/admin/plugins/%s/%s/{id}", pluginName, resourceSlug), func(w http.ResponseWriter, r *http.Request) {
				// TODO: Handle delete
				w.WriteHeader(http.StatusOK)
			})
		}
	}
}
```

**Step 2: Update admin handlers to register plugin routes**

In `internal/admin/handlers.go`, update `RegisterRoutes`:

```go
func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Route("/admin", func(r chi.Router) {
		// Existing routes
		r.Get("/", h.dashboard)
		r.Get("/guide", h.guide)
		r.Get("/logs", h.logsList)
		r.Get("/gmail", h.gmailList)
		r.Get("/calendar", h.calendarList)
		r.Get("/people", h.peopleList)
		r.Get("/tasks", h.tasksList)
		// ... other existing routes ...

		// NEW: Plugin routes
		RegisterPluginRoutes(r, core.All())
	})
}
```

**Step 3: Build to verify**

Run: `go build -o ish ./cmd/ish`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add internal/admin/
git commit -m "feat(admin): wire schema renderer to admin routes"
```

---

## Phase 5: OAuth Plugin

### Task 10: Create OAuth Plugin

**Files:**
- Create: `plugins/oauth/plugin.go`
- Create: `plugins/oauth/handlers.go`

**Step 1: Create oauth plugin**

Create `plugins/oauth/plugin.go`:

```go
// ABOUTME: OAuth provider simulation plugin.
// ABOUTME: Provides mock OAuth endpoints for testing token flows.

package oauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/2389/ish/internal/store"
	"github.com/2389/ish/plugins/core"
	"github.com/go-chi/chi/v5"
)

func init() {
	core.Register(&OAuthPlugin{})
}

type OAuthPlugin struct {
	store *store.Store
}

func (p *OAuthPlugin) Name() string {
	return "oauth"
}

func (p *OAuthPlugin) Health() core.HealthStatus {
	return core.HealthStatus{
		Status:  "healthy",
		Message: "OAuth plugin operational",
	}
}

func (p *OAuthPlugin) RegisterRoutes(r chi.Router) {
	// OAuth provider doesn't have API routes
}

func (p *OAuthPlugin) RegisterAuth(r chi.Router) {
	r.Get("/oauth/{provider}/authorize", p.handleAuthorize)
	r.Post("/oauth/{provider}/token", p.handleToken)
	r.Post("/oauth/{provider}/revoke", p.handleRevoke)
}

func (p *OAuthPlugin) Schema() core.PluginSchema {
	return core.PluginSchema{
		Resources: []core.ResourceSchema{
			{
				Name:        "Tokens",
				Slug:        "tokens",
				ListColumns: []string{"token", "plugin", "user", "created"},
				Fields: []core.FieldSchema{
					{Name: "token", Type: "string", Display: "Token", Required: false, Editable: false},
					{Name: "plugin_name", Type: "string", Display: "Plugin", Required: false, Editable: false},
					{Name: "user_id", Type: "string", Display: "User", Required: false, Editable: false},
					{Name: "scopes", Type: "string", Display: "Scopes", Required: false, Editable: false},
					{Name: "revoked", Type: "string", Display: "Revoked", Required: false, Editable: false},
					{Name: "created_at", Type: "datetime", Display: "Created", Required: false, Editable: false},
				},
				Actions: []core.ActionSchema{
					{Name: "revoke", HTTPMethod: "POST", Endpoint: "/oauth/{provider}/revoke", Confirm: true},
				},
			},
		},
	}
}

func (p *OAuthPlugin) Seed(ctx context.Context, size string) (core.SeedData, error) {
	return core.SeedData{
		Summary: "OAuth plugin does not seed data",
		Records: map[string]int{},
	}, nil
}

func (p *OAuthPlugin) ValidateToken(token string) bool {
	t, err := p.store.GetToken(token)
	if err != nil {
		return false
	}
	return !t.Revoked
}

func (p *OAuthPlugin) SetStore(s *store.Store) {
	p.store = s
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
```

**Step 2: Create OAuth handlers**

Create `plugins/oauth/handlers.go`:

```go
// ABOUTME: OAuth flow handlers.
// ABOUTME: Implements authorize, token exchange, and revoke endpoints.

package oauth

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/2389/ish/internal/store"
	"github.com/go-chi/chi/v5"
)

func (p *OAuthPlugin) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	redirectURI := r.URL.Query().Get("redirect_uri")
	state := r.URL.Query().Get("state")

	// Auto-approve and redirect with auth code
	code := generateToken()

	// Build redirect URL
	redirectURL := redirectURI + "?code=" + code + "&state=" + state
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (p *OAuthPlugin) handleToken(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	grantType := r.FormValue("grant_type")

	var accessToken, refreshToken string

	if grantType == "authorization_code" {
		// Exchange code for token
		accessToken = generateToken()
		refreshToken = generateToken()

		// Store token
		p.store.StoreToken(&store.OAuthToken{
			Token:        accessToken,
			PluginName:   provider,
			UserID:       "default",
			Scopes:       r.FormValue("scope"),
			ExpiresAt:    time.Now().Add(1 * time.Hour),
			RefreshToken: refreshToken,
			Revoked:      false,
		})
	} else if grantType == "refresh_token" {
		// Refresh token
		oldRefresh := r.FormValue("refresh_token")
		// TODO: Validate old refresh token

		accessToken = generateToken()
		refreshToken = generateToken()

		p.store.StoreToken(&store.OAuthToken{
			Token:        accessToken,
			PluginName:   provider,
			UserID:       "default",
			Scopes:       r.FormValue("scope"),
			ExpiresAt:    time.Now().Add(1 * time.Hour),
			RefreshToken: refreshToken,
			Revoked:      false,
		})
	}

	resp := map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    3600,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (p *OAuthPlugin) handleRevoke(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("token")
	p.store.RevokeToken(token)
	w.WriteHeader(http.StatusOK)
}
```

**Step 3: Import OAuth plugin in main**

In `cmd/ish/main.go`, add import:

```go
import (
	_ "github.com/2389/ish/plugins/oauth"  // Register OAuth plugin
)
```

**Step 4: Update plugin initialization to set store**

In `cmd/ish/main.go`, update `newServer`:

```go
// Initialize all plugins with store
for _, plugin := range core.All() {
	switch p := plugin.(type) {
	case *google.GooglePlugin:
		p.SetStore(s)
	case *oauth.OAuthPlugin:
		p.SetStore(s)
	}
	plugin.RegisterAuth(r)
	plugin.RegisterRoutes(r)
}
```

**Step 5: Build to verify**

Run: `go build -o ish ./cmd/ish`
Expected: SUCCESS

**Step 6: Commit**

```bash
git add plugins/oauth/ cmd/ish/main.go
git commit -m "feat(plugins): add OAuth provider simulation plugin"
```

---

## Phase 6: Testing & Documentation

### Task 11: Update Admin Dashboard

**Files:**
- Modify: `internal/admin/templates/dashboard.html`
- Modify: `internal/admin/handlers.go`

**Step 1: Update dashboard template to show plugins**

In `internal/admin/templates/dashboard.html`, add plugins section:

```html
<!-- Add after existing cards -->
<div class="mt-8">
    <h2 class="text-xl font-bold text-gray-900 mb-4">Plugins</h2>
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {{range .Plugins}}
        <div class="bg-white p-6 rounded-lg shadow">
            <div class="flex justify-between items-start mb-4">
                <h3 class="text-lg font-semibold text-gray-900">{{.Name}}</h3>
                <span class="px-2 py-1 text-xs rounded {{if eq .Health.Status "healthy"}}bg-green-100 text-green-800{{else}}bg-red-100 text-red-800{{end}}">
                    {{.Health.Status}}
                </span>
            </div>
            <p class="text-sm text-gray-600 mb-4">{{.Health.Message}}</p>
            <div class="space-y-2">
                {{range .Resources}}
                <a href="/admin/plugins/{{$.Name}}/{{.Slug}}" class="block text-sm text-blue-600 hover:text-blue-800">
                    → {{.Name}}
                </a>
                {{end}}
            </div>
        </div>
        {{end}}
    </div>
</div>
```

**Step 2: Update dashboard handler to pass plugins**

In `internal/admin/handlers.go`, update `dashboard`:

```go
func (h *Handlers) dashboard(w http.ResponseWriter, r *http.Request) {
	counts, _ := h.store.GetCounts()

	// Get plugins with their schemas
	var pluginData []map[string]any
	for _, plugin := range core.All() {
		schema := plugin.Schema()
		pluginData = append(pluginData, map[string]any{
			"Name":      plugin.Name(),
			"Health":    plugin.Health(),
			"Resources": schema.Resources,
		})
	}

	renderPage(w, "dashboard", map[string]any{
		"Messages":  counts.Messages,
		"Events":    counts.Events,
		"People":    counts.People,
		"Tasks":     counts.Tasks,
		"Requests":  counts.Requests,
		"Plugins":   pluginData,
	})
}
```

**Step 3: Build to verify**

Run: `go build -o ish ./cmd/ish`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add internal/admin/
git commit -m "feat(admin): add plugins section to dashboard"
```

---

### Task 12: Update Logs to Filter by Plugin

**Files:**
- Modify: `internal/admin/templates/logs/list.html`
- Modify: `internal/admin/handlers.go`
- Modify: `internal/store/request_logs.go`

**Step 1: Add plugin filter to RequestLogQuery**

In `internal/store/request_logs.go`, update `RequestLogQuery`:

```go
type RequestLogQuery struct {
	Limit      int
	Offset     int
	Method     string
	PathPrefix string
	StatusCode int
	UserID     string
	PluginName string  // NEW
}
```

**Step 2: Update GetRequestLogs to filter by plugin**

In `internal/store/request_logs.go`, update `GetRequestLogs`:

```go
func (s *Store) GetRequestLogs(q *RequestLogQuery) ([]*RequestLog, error) {
	query := `SELECT id, timestamp, COALESCE(plugin_name, ''), method, path, status_code, duration_ms,
	          COALESCE(user_id, ''), COALESCE(ip_address, ''), COALESCE(user_agent, ''), COALESCE(error, ''),
	          COALESCE(request_body, ''), COALESCE(response_body, '')
	          FROM request_logs WHERE 1=1`
	args := []any{}

	if q.Method != "" {
		query += " AND method = ?"
		args = append(args, q.Method)
	}
	if q.PathPrefix != "" {
		query += " AND path LIKE ?"
		args = append(args, q.PathPrefix+"%")
	}
	if q.StatusCode > 0 {
		query += " AND status_code = ?"
		args = append(args, q.StatusCode)
	}
	if q.UserID != "" {
		query += " AND user_id = ?"
		args = append(args, q.UserID)
	}
	if q.PluginName != "" {  // NEW
		query += " AND plugin_name = ?"
		args = append(args, q.PluginName)
	}

	query += " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	args = append(args, q.Limit, q.Offset)

	// ... rest of function unchanged
}
```

**Step 3: Add plugin filter UI to logs template**

In `internal/admin/templates/logs/list.html`, add plugin filter:

```html
<!-- Add after the title, before stats cards -->
<div class="bg-white rounded-lg shadow p-4 mb-6">
    <form method="get" class="flex gap-4">
        <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Plugin</label>
            <select name="plugin" class="border border-gray-300 rounded px-3 py-2">
                <option value="">All Plugins</option>
                {{range .Plugins}}
                <option value="{{.}}" {{if eq . $.SelectedPlugin}}selected{{end}}>{{.}}</option>
                {{end}}
            </select>
        </div>
        <div class="flex items-end">
            <button type="submit" class="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700">
                Filter
            </button>
        </div>
    </form>
</div>
```

**Step 4: Update logsList handler to pass plugins and selected**

In `internal/admin/handlers.go`, update `logsList`:

```go
func (h *Handlers) logsList(w http.ResponseWriter, r *http.Request) {
	method := r.URL.Query().Get("method")
	pathPrefix := r.URL.Query().Get("path")
	pluginName := r.URL.Query().Get("plugin")  // NEW
	statusCode := 0
	if sc := r.URL.Query().Get("status"); sc != "" {
		fmt.Sscanf(sc, "%d", &statusCode)
	}

	logs, err := h.store.GetRequestLogs(&store.RequestLogQuery{
		Limit:      100,
		Offset:     0,
		Method:     method,
		PathPrefix: pathPrefix,
		StatusCode: statusCode,
		PluginName: pluginName,  // NEW
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	for _, log := range logs {
		log.RequestBody = prettyJSON(log.RequestBody)
		log.ResponseBody = prettyJSON(log.ResponseBody)
	}

	stats, _ := h.store.GetRequestLogStats()
	topEndpoints, _ := h.store.GetTopEndpoints(10)

	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "logs-list", map[string]any{
		"Logs":           logs,
		"Stats":          stats,
		"TopEndpoints":   topEndpoints,
		"Plugins":        core.Names(),      // NEW
		"SelectedPlugin": pluginName,        // NEW
	})
}
```

**Step 5: Add plugin column to logs table**

In `internal/admin/templates/logs/list.html`, add plugin column:

```html
<thead class="bg-gray-50">
    <tr>
        <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Timestamp</th>
        <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Plugin</th>  <!-- NEW -->
        <th class="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Method</th>
        <!-- ... rest of columns ... -->
    </tr>
</thead>
<tbody class="bg-white">
    {{range .Logs}}
    <tr class="border-b border-gray-200 hover:bg-gray-50">
        <td class="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
            {{.Timestamp.Format "15:04:05"}}
        </td>
        <td class="px-4 py-3 whitespace-nowrap text-sm">  <!-- NEW -->
            <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-purple-100 text-purple-800">
                {{.PluginName}}
            </span>
        </td>
        <!-- ... rest of row ... -->
    </tr>
    {{end}}
</tbody>
```

**Step 6: Build to verify**

Run: `go build -o ish ./cmd/ish`
Expected: SUCCESS

**Step 7: Commit**

```bash
git add internal/store/ internal/admin/
git commit -m "feat(admin): add plugin filter to request logs"
```

---

### Task 13: End-to-End Testing

**Files:**
- None (testing only)

**Step 1: Reset and seed database**

Run: `./ish reset`
Expected: Database reset message

**Step 2: Start server**

Run: `./ish serve --port 9001 > /tmp/ish-test.log 2>&1 &`
Expected: Server starts in background

**Step 3: Test Gmail API**

Run: `curl -s http://localhost:9001/gmail/v1/users/me/messages -H "Authorization: Bearer test-token-123" | jq .messages[0].id`
Expected: Returns message ID

**Step 4: Test Calendar API**

Run: `curl -s http://localhost:9001/calendar/v3/calendars/primary/events -H "Authorization: Bearer test-token-123" | jq .items[0].summary`
Expected: Returns event summary

**Step 5: Test admin dashboard**

Run: `curl -s http://localhost:9001/admin/ | grep -c "Plugins"`
Expected: 1 (plugin section exists)

**Step 6: Test logs show plugin names**

Run: `curl -s http://localhost:9001/admin/logs | grep -c "google"`
Expected: >0 (plugin name appears)

**Step 7: Test OAuth token endpoint**

Run: `curl -s -X POST http://localhost:9001/oauth/google/token -d "grant_type=authorization_code&code=test" | jq .access_token`
Expected: Returns token value

**Step 8: Stop server**

Run: `pkill -f "ish serve"`
Expected: Server stops

**Step 9: Verify all tests passed**

All tests should pass. If any fail, investigate and fix before proceeding.

**Step 10: Commit test results**

```bash
echo "All end-to-end tests passed" >> docs/plans/test-results.txt
git add docs/plans/test-results.txt
git commit -m "test: verify plugin architecture end-to-end"
```

---

### Task 14: Update Documentation

**Files:**
- Modify: `README.md`
- Create: `docs/plugins/README.md`

**Step 1: Update main README**

In `README.md`, update architecture section:

```markdown
## Architecture

ISH uses a **plugin-based architecture** where each API (Google, Discord, Stripe, etc.) is a self-contained plugin:

- **Plugins** implement a standard interface and register at compile time
- **Schema-driven UI** - Plugins define data structures, ISH generates admin UI
- **Unified logging** - All plugin requests logged with plugin identification
- **OAuth simulation** - Mock OAuth provider for testing token flows

### Built-in Plugins

- **Google** - Gmail, Calendar, People, Tasks APIs
- **OAuth** - OAuth provider simulation for token lifecycle testing

### Plugin Development

See [docs/plugins/README.md](docs/plugins/README.md) for creating custom plugins.
```

**Step 2: Create plugin development guide**

Create `docs/plugins/README.md`:

```markdown
# Plugin Development Guide

## Overview

ISH plugins are Go packages that implement the `Plugin` interface. Plugins register themselves at compile time and provide:

- HTTP routes for API endpoints
- OAuth/auth endpoints
- Schema definitions for admin UI
- Data seeding functions

## Creating a Plugin

### 1. Implement the Plugin Interface

```go
package myplugin

import (
    "context"
    "github.com/2389/ish/plugins/core"
    "github.com/go-chi/chi/v5"
)

func init() {
    core.Register(&MyPlugin{})
}

type MyPlugin struct {}

func (p *MyPlugin) Name() string {
    return "myplugin"
}

func (p *MyPlugin) Health() core.HealthStatus {
    return core.HealthStatus{Status: "healthy", Message: "OK"}
}

func (p *MyPlugin) RegisterRoutes(r chi.Router) {
    r.Get("/api/v1/resource", p.handleResource)
}

func (p *MyPlugin) RegisterAuth(r chi.Router) {
    r.Get("/oauth/myplugin/authorize", p.handleAuthorize)
    r.Post("/oauth/myplugin/token", p.handleToken)
}

func (p *MyPlugin) Schema() core.PluginSchema {
    return core.PluginSchema{
        Resources: []core.ResourceSchema{
            {
                Name: "Items",
                Slug: "items",
                ListColumns: []string{"name", "created"},
                Fields: []core.FieldSchema{
                    {Name: "name", Type: "string", Display: "Name", Required: true, Editable: true},
                    {Name: "created", Type: "datetime", Display: "Created", Editable: false},
                },
                Actions: []core.ActionSchema{
                    {Name: "delete", HTTPMethod: "DELETE", Endpoint: "/api/v1/items/{id}", Confirm: true},
                },
            },
        },
    }
}

func (p *MyPlugin) Seed(ctx context.Context, size string) (core.SeedData, error) {
    // Generate test data
    return core.SeedData{
        Summary: "Created 10 items",
        Records: map[string]int{"items": 10},
    }, nil
}

func (p *MyPlugin) ValidateToken(token string) bool {
    // Validate OAuth token
    return true
}
```

### 2. Create Database Tables

Use naming convention: `{plugin}_{resource}`

```go
CREATE TABLE myplugin_items (
    id TEXT PRIMARY KEY,
    name TEXT,
    created TIMESTAMP
)
```

### 3. Import in Main Binary

```go
// cmd/ish/main.go
import _ "github.com/yourname/ish-myplugin"
```

### 4. Build

```bash
go build -o ish ./cmd/ish
```

## Schema-Driven UI

The admin UI is auto-generated from your plugin's schema. No HTML required!

### Field Types

- `string` - Text input
- `email` - Email input
- `datetime` - Datetime picker
- `text` - Textarea

### Actions

Define actions users can take on resources:

```go
Actions: []core.ActionSchema{
    {Name: "send", HTTPMethod: "POST", Endpoint: "/api/v1/items/{id}/send", Confirm: false},
    {Name: "delete", HTTPMethod: "DELETE", Endpoint: "/api/v1/items/{id}", Confirm: true},
}
```

## Examples

See `plugins/google` for a complete example.

## Best Practices

1. **Table naming**: Use `{plugin}_{resource}` pattern
2. **Schema simplicity**: Keep schemas simple, ISH is for testing not production
3. **OAuth**: Implement auth endpoints even if tokens are fake
4. **Seeding**: Support small/medium/large sizes
5. **Health checks**: Report actual plugin status
```

**Step 3: Build to verify**

Run: `go build -o ish ./cmd/ish`
Expected: SUCCESS

**Step 4: Commit**

```bash
git add README.md docs/plugins/
git commit -m "docs: update README and add plugin development guide"
```

---

## Phase 7: Cleanup

### Task 15: Remove Old Internal Packages

**Files:**
- Delete: `internal/gmail/`
- Delete: `internal/calendar/`
- Delete: `internal/people/`
- Delete: `internal/tasks/`
- Delete: `internal/autoreply/`

**Step 1: Remove old packages**

Run:
```bash
rm -rf internal/gmail
rm -rf internal/calendar
rm -rf internal/people
rm -rf internal/tasks
rm -rf internal/autoreply
```

**Step 2: Build to verify no dependencies**

Run: `go build -o ish ./cmd/ish`
Expected: SUCCESS (no missing imports)

**Step 3: Commit**

```bash
git add -A
git commit -m "refactor: remove old internal API packages (moved to plugins)"
```

---

### Task 16: Final Integration Test

**Files:**
- None (testing only)

**Step 1: Full reset**

Run: `./ish reset`
Expected: Clean database

**Step 2: Start server**

Run: `./ish serve --port 9001 &`

**Step 3: Test all APIs work**

Run:
```bash
curl -s http://localhost:9001/gmail/v1/users/me/messages -H "Authorization: Bearer test" | jq -e '.messages'
curl -s http://localhost:9001/calendar/v3/calendars/primary/events -H "Authorization: Bearer test" | jq -e '.items'
curl -s http://localhost:9001/people/v1/people/me/connections -H "Authorization: Bearer test" | jq -e '.connections'
curl -s http://localhost:9001/tasks/v1/users/@me/lists -H "Authorization: Bearer test" | jq -e '.items'
```

Expected: All return data

**Step 4: Test OAuth flow**

Run:
```bash
curl -s -X POST http://localhost:9001/oauth/google/token -d "grant_type=authorization_code&code=test" | jq -e '.access_token'
```

Expected: Returns token

**Step 5: Test admin UI renders**

Run:
```bash
curl -s http://localhost:9001/admin/ | grep -c "google"
curl -s http://localhost:9001/admin/plugins/google/messages | grep -c "Messages"
```

Expected: Both return >0

**Step 6: Test logging tracks plugins**

Run:
```bash
curl -s http://localhost:9001/admin/logs | grep -c "plugin_name"
```

Expected: >0

**Step 7: Stop server**

Run: `pkill -f "ish serve"`

**Step 8: All tests pass - commit**

```bash
git tag v2.0.0-plugin-architecture
git commit --allow-empty -m "test: verify full plugin architecture integration"
```

---

## Completion Checklist

- [ ] Task 1: Plugin interface created
- [ ] Task 2: Database schema updated
- [ ] Task 3: OAuth token management
- [ ] Task 4: Google plugin structure
- [ ] Task 5: Gmail handlers moved
- [ ] Task 6: Calendar/People/Tasks moved
- [ ] Task 7: Main server uses plugins
- [ ] Task 8: Schema renderer created
- [ ] Task 9: Schema renderer wired
- [ ] Task 10: OAuth plugin created
- [ ] Task 11: Dashboard shows plugins
- [ ] Task 12: Logs filter by plugin
- [ ] Task 13: End-to-end tests pass
- [ ] Task 14: Documentation updated
- [ ] Task 15: Old packages removed
- [ ] Task 16: Final integration test

---

## Rollback Plan

If issues arise:

```bash
git checkout main
cd ..
rm -rf .worktrees/plugin-architecture
git worktree prune
```

## Post-Implementation

After completing all tasks:

1. Merge feature branch to main
2. Update product-description.md with plugin info
3. Create example external plugin
4. Announce plugin system to community

---

**End of Implementation Plan**
