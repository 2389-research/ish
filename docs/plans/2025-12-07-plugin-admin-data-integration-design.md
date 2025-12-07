# Plugin Admin Data Integration Design

**Date:** 2025-12-07
**Status:** Approved for Implementation
**Author:** Claude Code

## Problem Statement

The plugin admin routes at `/admin/plugins/{plugin}/{resource}` currently render UI but show empty data. Each plugin has its own store with real data, but there's no mechanism to expose this data to the admin UI.

## Solution Overview

Add an optional `DataProvider` interface that plugins can implement to expose their data to the admin UI in a standardized way.

## Architecture

### Core Interface

New file: `plugins/core/data_provider.go`

```go
package core

import "context"

// DataProvider is an optional interface that plugins can implement
// to expose their data to the admin UI
type DataProvider interface {
    Plugin
    ListResources(ctx context.Context, resourceSlug string, opts ListOptions) ([]map[string]interface{}, error)
    GetResource(ctx context.Context, resourceSlug string, id string) (map[string]interface{}, error)
}

type ListOptions struct {
    Limit  int
    Offset int
}
```

**Design decisions:**
- **Optional interface** - Plugins opt-in to admin support
- **Generic return type** - `map[string]interface{}` matches schema field names
- **Context-aware** - Supports cancellation and timeouts
- **Resource-based** - Uses schema resource slugs ("messages", "calls", etc.)

### Store Layer Changes

Each plugin store needs new methods to fetch data across all accounts:

**SendGrid:**
```go
func (s *SendGridStore) ListAllMessages(limit, offset int) ([]*Message, error)
func (s *SendGridStore) ListAllSuppressions(limit, offset int) ([]*Suppression, error)
```

**Twilio:**
```go
func (s *TwilioStore) ListAllMessages(limit, offset int) ([]*Message, error)
func (s *TwilioStore) ListAllCalls(limit, offset int) ([]*Call, error)
func (s *TwilioStore) ListAllAccounts(limit, offset int) ([]*Account, error)
func (s *TwilioStore) ListAllPhoneNumbers(limit, offset int) ([]*PhoneNumber, error)
```

**GitHub:**
```go
func (s *GitHubStore) ListAllRepositories(limit, offset int) ([]*Repository, error)
func (s *GitHubStore) ListAllIssues(limit, offset int) ([]*Issue, error)
func (s *GitHubStore) ListAllPullRequests(limit, offset int) ([]*PullRequest, error)
func (s *GitHubStore) ListAllWebhooks(limit, offset int) ([]*Webhook, error)
```

**Discord:**
```go
func (s *DiscordStore) ListAllGuilds(limit, offset int) ([]*Guild, error)
func (s *DiscordStore) ListAllChannels(limit, offset int) ([]*Channel, error)
func (s *DiscordStore) ListAllMessages(limit, offset int) ([]*Message, error)
```

**OAuth:**
```go
func (s *OAuthStore) ListAllTokens(limit, offset int) ([]*Token, error)
func (s *OAuthStore) ListAllGrants(limit, offset int) ([]*Grant, error)
```

### Plugin Implementation Pattern

Each plugin implements `DataProvider` with a switch on resource slug:

```go
func (p *SendGridPlugin) ListResources(ctx context.Context, slug string, opts core.ListOptions) ([]map[string]interface{}, error) {
    switch slug {
    case "messages":
        messages, err := p.store.ListAllMessages(opts.Limit, opts.Offset)
        if err != nil {
            return nil, err
        }
        return convertMessagesToMaps(messages), nil
    case "suppressions":
        suppressions, err := p.store.ListAllSuppressions(opts.Limit, opts.Offset)
        if err != nil {
            return nil, err
        }
        return convertSuppressionsToMaps(suppressions), nil
    default:
        return nil, fmt.Errorf("unknown resource: %s", slug)
    }
}

func (p *SendGridPlugin) GetResource(ctx context.Context, slug string, id string) (map[string]interface{}, error) {
    switch slug {
    case "messages":
        message, err := p.store.GetMessage(id)
        if err != nil {
            return nil, err
        }
        return convertMessageToMap(message), nil
    // ... similar for other resources
    }
}
```

Helper conversion functions transform structs to maps matching schema fields.

### Admin UI Integration

Update `internal/admin/plugin_routes.go`:

**Before:**
```go
// For now, render with empty data (no backend data fetching yet)
resources := []map[string]interface{}{}
```

**After:**
```go
// Check if plugin supports data fetching
var resources []map[string]interface{}
if dataProvider, ok := plugin.(core.DataProvider); ok {
    opts := core.ListOptions{Limit: 50, Offset: 0}
    resources, err = dataProvider.ListResources(r.Context(), resourceSlug, opts)
    if err != nil {
        log.Printf("Error fetching %s data from %s: %v", resourceSlug, pluginName, err)
        resources = []map[string]interface{}{} // fallback to empty
    }
} else {
    log.Printf("Plugin %s does not implement DataProvider", pluginName)
}
```

Similar updates for `PluginDetailView` and `PluginEditForm`.

## Error Handling

**Plugin doesn't implement DataProvider:**
- Show empty list with message: "Admin UI not yet available for this plugin"

**DataProvider returns error:**
- Log error with context
- Show empty list with message: "Error loading data"

**Database errors:**
- Return error from store methods
- Handled by plugin layer, logged by admin routes

**Context cancellation:**
- Respect context deadlines
- Return early if cancelled

**Edge cases:**
- Empty database → "No resources found" message
- Large datasets → enforce max limit of 1000 records
- Invalid resource slug → return 404 "Resource not found"
- Malformed data → log warning, skip record, continue

## Testing Strategy

Create end-to-end scenario tests for each plugin:

**Test pattern:**
```bash
#!/bin/bash
# .scratch/test_{plugin}_admin.sh

set -e

# Setup
rm -f test_admin.db
./ish seed -d test_admin.db --size medium

# Start server
./ish serve -d test_admin.db -p 19999 &
SERVER_PID=$!
sleep 2

# Test: Fetch admin page
response=$(curl -s http://localhost:19999/admin/plugins/{plugin}/{resource})

# Verify: Contains actual data
echo "$response" | grep -q "{expected_column}" || (echo "FAIL"; exit 1)
echo "$response" | grep -q "{expected_data}" || (echo "FAIL"; exit 1)

# Cleanup
kill $SERVER_PID
rm -f test_admin.db

echo "✓ {Plugin} admin scenario test passed"
```

**Tests created:**
- `.scratch/test_sendgrid_admin.sh`
- `.scratch/test_twilio_admin.sh`
- `.scratch/test_github_admin.sh`
- `.scratch/test_discord_admin.sh`
- `.scratch/test_oauth_admin.sh`

**Tests verify:**
1. Admin page renders successfully
2. Column headers match schema
3. Actual seeded data appears in HTML
4. Multiple resources per plugin work

## Implementation Plan

1. **Core interface** - Create `plugins/core/data_provider.go`
2. **SendGrid** - Add store methods + implement DataProvider
3. **Twilio** - Add store methods + implement DataProvider
4. **GitHub** - Add store methods + implement DataProvider
5. **Discord** - Add store methods + implement DataProvider
6. **OAuth** - Add store methods + implement DataProvider
7. **Admin integration** - Update `plugin_routes.go`
8. **Scenario tests** - Create test scripts for all plugins
9. **Validation** - Run all scenario tests, verify data appears

## Rollback Plan

The `DataProvider` interface is optional. If issues arise:
- Plugins without DataProvider show empty data (existing behavior)
- No breaking changes to existing functionality
- Can disable specific plugin implementations if needed

## Success Criteria

- ✅ All 5 plugins implement DataProvider interface
- ✅ All admin pages show real data from stores
- ✅ All 5 scenario tests pass
- ✅ No regressions in existing plugin functionality
- ✅ Error cases handled gracefully

## Files Modified

**New:**
- `plugins/core/data_provider.go`
- `.scratch/test_sendgrid_admin.sh`
- `.scratch/test_twilio_admin.sh`
- `.scratch/test_github_admin.sh`
- `.scratch/test_discord_admin.sh`
- `.scratch/test_oauth_admin.sh`

**Modified:**
- `plugins/sendgrid/store.go` (new ListAll methods)
- `plugins/sendgrid/plugin.go` (implement DataProvider)
- `plugins/twilio/store.go`
- `plugins/twilio/plugin.go`
- `plugins/github/store.go`
- `plugins/github/plugin.go`
- `plugins/discord/store.go`
- `plugins/discord/plugin.go`
- `plugins/oauth/store.go`
- `plugins/oauth/plugin.go`
- `internal/admin/plugin_routes.go` (call DataProvider)

## Future Enhancements

- Pagination controls in admin UI
- Filtering and search
- Sorting by columns
- Create/edit/delete operations
- Bulk actions
