# Discord Webhook Plugin + Store Refactor Design

**Date:** 2025-12-06
**Status:** Approved
**Author:** Claude Code (with Doctor Biz)

## Overview

Add a Discord webhook API plugin to ISH while refactoring all plugins to be truly self-contained with their own storage layers. This establishes the proper pattern for plugin architecture where each plugin owns its complete data layer.

## Goals

1. **Self-Contained Plugins** - Every plugin owns its storage, migrations, and data access
2. **Discord Webhook Support** - Full Discord webhook API v10 simulation for testing
3. **Reference Implementation** - Discord plugin demonstrates ideal plugin architecture
4. **Clean Core** - Core `internal/store/` only handles DB connection and request logging

## Architecture Changes

### Current State (Before)

```
internal/store/
├── store.go                 # DB connection + all plugin data
├── gmail.go                 # Gmail-specific queries
├── calendar.go              # Calendar-specific queries
├── people.go                # People-specific queries
├── tasks.go                 # Tasks-specific queries
└── oauth_tokens.go          # OAuth token management

plugins/google/
├── plugin.go                # Receives store from main
├── gmail.go                 # Calls s.store methods
├── calendar.go
└── tasks.go

plugins/oauth/
└── plugin.go                # Receives store from main
```

**Problem:** Plugins depend on shared `internal/store/`. Can't be extracted to external modules.

### Target State (After)

```
internal/store/
├── store.go                 # DB connection ONLY
└── request_logs.go          # Shared request logging

plugins/google/
├── plugin.go                # Initializes own DB
├── store.go                 # Owns: gmail, calendar, people, tasks tables
├── gmail.go
├── calendar.go
└── tasks.go

plugins/oauth/
├── plugin.go                # Initializes own DB
├── store.go                 # Owns: oauth_tokens table
└── handlers.go

plugins/discord/             # NEW - Reference implementation
├── plugin.go                # Fully self-contained
├── store.go                 # Owns: discord_webhooks, discord_webhook_messages
├── handlers.go              # Discord webhook API
└── schema.go                # Admin UI definitions
```

**Benefit:** Each plugin is a complete, self-contained unit that could be moved to external Go module.

## Discord Webhook Plugin Specification

### API Endpoints (Discord API v10)

**Webhook Management:**
- `POST /api/webhooks/{webhook.id}/{webhook.token}` - Execute webhook
  - Query params: `?wait=true` (return message), `?thread_id={id}`
  - Body: content, username, avatar_url, embeds, components, files
- `GET /api/webhooks/{webhook.id}/{webhook.token}` - Get webhook info
- `PATCH /api/webhooks/{webhook.id}/{webhook.token}` - Modify webhook
- `DELETE /api/webhooks/{webhook.id}/{webhook.token}` - Delete webhook

**Message Management:**
- `GET /api/webhooks/{webhook.id}/{webhook.token}/messages/{message.id}` - Get message
- `PATCH /api/webhooks/{webhook.id}/{webhook.token}/messages/{message.id}` - Edit message
- `DELETE /api/webhooks/{webhook.id}/{webhook.token}/messages/{message.id}` - Delete message

### Database Schema

**Table: `discord_webhooks`**
```sql
CREATE TABLE discord_webhooks (
    id TEXT PRIMARY KEY,              -- Discord snowflake ID
    token TEXT NOT NULL,              -- Webhook token
    type INTEGER DEFAULT 1,           -- Webhook type (1 = Incoming)
    name TEXT,                        -- Webhook name
    avatar TEXT,                      -- Avatar hash
    channel_id TEXT,                  -- Simulated channel ID
    guild_id TEXT,                    -- Simulated guild ID
    application_id TEXT,              -- Application ID
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,             -- Soft delete
    UNIQUE(id, token)
);
```

**Table: `discord_webhook_messages`**
```sql
CREATE TABLE discord_webhook_messages (
    id TEXT PRIMARY KEY,              -- Message snowflake ID
    webhook_id TEXT NOT NULL,         -- References discord_webhooks.id
    content TEXT,                     -- Message text content
    username TEXT,                    -- Override webhook username
    avatar_url TEXT,                  -- Override webhook avatar
    embeds TEXT,                      -- JSON array of embeds
    components TEXT,                  -- JSON array of components
    attachments TEXT,                 -- JSON array of attachments
    thread_id TEXT,                   -- Thread ID if sent to thread
    flags INTEGER,                    -- Message flags
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    edited_at TIMESTAMP,              -- Last edit timestamp
    deleted_at TIMESTAMP,             -- Soft delete
    FOREIGN KEY (webhook_id) REFERENCES discord_webhooks(id) ON DELETE CASCADE
);

CREATE INDEX idx_webhook_messages_webhook_id ON discord_webhook_messages(webhook_id);
CREATE INDEX idx_webhook_messages_created_at ON discord_webhook_messages(created_at DESC);
```

### Auto-Accept Pattern

Following OAuth plugin pattern:
1. First `POST /api/webhooks/{id}/{token}` auto-creates webhook if doesn't exist
2. Auto-generated: name="Incoming Webhook", random channel_id/guild_id
3. All subsequent requests use stored webhook
4. Admin can view/delete webhooks via UI

### Admin UI Schema

**Resource 1: Webhooks**
- List: id, name, token (truncated), channel_id, created_at
- Detail: Full webhook object
- Actions: Delete, View Messages

**Resource 2: Messages**
- List: id, webhook_id, content (truncated), username, created_at
- Detail: Full message with formatted embeds
- Actions: Delete, View Raw JSON
- Filter: By webhook_id

## Implementation Plan

### Phase 1: Refactor Existing Plugins

**Task 1.1: Extract Google Plugin Store**
- Create `plugins/google/store.go`
- Move Gmail, Calendar, People, Tasks queries from `internal/store/`
- Add table initialization methods
- Update plugin to call `plugin.InitDB(db)`

**Task 1.2: Extract OAuth Plugin Store**
- Create `plugins/oauth/store.go`
- Move `oauth_tokens` table management
- Update plugin initialization

**Task 1.3: Simplify Core Store**
- Remove plugin-specific code from `internal/store/`
- Keep only: DB connection, request logging
- Add `GetDB() *sql.DB` for plugins
- Update tests

**Task 1.4: Update Main Server**
- Pass `*sql.DB` to plugins via `SetDB()` method
- Remove plugin-specific store dependencies

### Phase 2: Discord Plugin

**Task 2.1: Plugin Structure**
- Create `plugins/discord/plugin.go`
- Implement Plugin interface
- Auto-register via `init()`

**Task 2.2: Storage Layer**
- Create `plugins/discord/store.go`
- Define tables and migrations
- Implement CRUD operations

**Task 2.3: HTTP Handlers**
- Create `plugins/discord/handlers.go`
- Implement all webhook endpoints
- Match Discord API v10 format exactly

**Task 2.4: Admin Schema**
- Create `plugins/discord/schema.go`
- Define Webhooks and Messages resources
- Wire to schema renderer

**Task 2.5: Registration**
- Add import to `cmd/ish/main.go`
- Update plugin initialization

### Phase 3: Testing & Documentation

**Task 3.1: Unit Tests**
- Store layer tests
- Handler tests for all endpoints
- Schema validation tests

**Task 3.2: Integration Tests**
- Full webhook flow (create, send, edit, delete)
- Admin UI rendering tests
- Error handling tests

**Task 3.3: Documentation**
- Update README with Discord plugin
- Document self-contained plugin pattern
- Add Discord API examples

## Plugin Interface Changes

No changes to `Plugin` interface, but add helper:

```go
// Plugins that need database access implement this
type DatabasePlugin interface {
    Plugin
    SetDB(db *sql.DB)
}
```

Main server checks for this interface and calls `SetDB()` if present.

## Migration Path

1. **Phase 1 must complete before Phase 2** - Can't add Discord until refactor done
2. **All tests must pass after Phase 1** - Refactor is zero-regression
3. **Discord serves as validation** - Proves pattern works for new plugins

## Success Criteria

- ✅ All existing tests pass after refactor
- ✅ Google/OAuth plugins function identically
- ✅ Discord webhook sends work with real Discord libraries
- ✅ Plugins can theoretically be moved to external repos
- ✅ `internal/store/` has no plugin-specific code
- ✅ Admin UI shows Discord webhooks and messages correctly

## Non-Goals

- Slack/GitHub webhook compatibility endpoints (Discord-native only)
- Rate limiting simulation (keep simple)
- Thread/forum channel simulation (just store thread_id)
- Webhook avatar validation (accept any URL)

## References

- Discord API Spec: https://github.com/discord/discord-api-spec
- Discord Webhook Docs: https://discord.com/developers/docs/resources/webhook
- ISH Plugin Architecture: `docs/ARCHITECTURE.md`
- Plugin Development Guide: `docs/plugins/DEVELOPMENT.md`
