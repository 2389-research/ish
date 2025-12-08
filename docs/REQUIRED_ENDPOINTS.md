# Required API Endpoints for ish

This document lists ALL Google API endpoints that need to be implemented in ish to support both `pagen-agent` (Jeff) and `gsuite-mcp` clients.

## Status Legend
- ✅ Implemented
- 🟡 Partially Implemented
- ❌ Not Implemented

---

## Gmail API (`/gmail/v1`)

### Read Operations

| Endpoint | Method | Status | Used By | Notes |
|----------|--------|--------|---------|-------|
| `/users/me/profile` | GET | ✅ | Both | User profile |
| `/users/me/messages` | GET | ✅ | Both | List messages with query support |
| `/users/me/messages/{id}` | GET | ✅ | Both | Get message (full & metadata formats) |
| `/users/me/history` | GET | ✅ | Jeff | Sync changes |
| `/users/me/messages/{messageId}/attachments/{attachmentId}` | GET | ✅ | Jeff | Get attachment |

### Write Operations

| Endpoint | Method | Status | Used By | Priority |
|----------|--------|--------|---------|----------|
| `/users/me/messages/send` | POST | ❌ | Both | **CRITICAL** |
| `/users/me/drafts` | POST | ❌ | gsuite-mcp | High |
| `/users/me/drafts/{id}` | PUT | ❌ | gsuite-mcp | Medium |
| `/users/me/drafts/{id}` | DELETE | ❌ | gsuite-mcp | Medium |
| `/users/me/messages/{id}/modify` | POST | ❌ | Both | High (labels) |
| `/users/me/messages/{id}/trash` | POST | ❌ | Jeff | Medium |
| `/users/me/messages/{id}` | DELETE | ❌ | Jeff | Medium |

---

## Calendar API (`/calendar/v3`)

### Read Operations

| Endpoint | Method | Status | Used By | Notes |
|----------|--------|--------|---------|-------|
| `/calendars/primary/events` | GET | ✅ | Both | List events with time filters |
| `/calendars/primary/events/{eventId}` | GET | 🟡 | gsuite-mcp | Needs implementation |

### Write Operations

| Endpoint | Method | Status | Used By | Priority |
|----------|--------|--------|---------|----------|
| `/calendars/primary/events` | POST | ❌ | Both | **CRITICAL** |
| `/calendars/primary/events/{eventId}` | PUT/PATCH | ❌ | gsuite-mcp | High |
| `/calendars/primary/events/{eventId}` | DELETE | ❌ | gsuite-mcp | High |

---

## Tasks API (`/tasks/v1`)

**Note:** Tasks API is only used by `pagen-agent` (Jeff), not by `gsuite-mcp`.

| Endpoint | Method | Status | Used By | Priority |
|----------|--------|--------|---------|----------|
| `/lists/@default/tasks` | GET | ❌ | Jeff | **CRITICAL** |
| `/lists/@default/tasks` | POST | ❌ | Jeff | **CRITICAL** |
| `/lists/@default/tasks/{taskId}` | GET | ❌ | Jeff | High |
| `/lists/@default/tasks/{taskId}` | PUT/PATCH | ❌ | Jeff | High |
| `/lists/@default/tasks/{taskId}` | DELETE | ❌ | Jeff | Medium |

---

## People API (`/people/v1`)

**Note:** People API is only used by `gsuite-mcp`, not by `pagen-agent`.

### Read Operations

| Endpoint | Method | Status | Used By | Notes |
|----------|--------|--------|---------|-------|
| `/people/me/connections` | GET | 🟡 | gsuite-mcp | Needs personFields support |
| `/people:searchContacts` | GET | 🟡 | gsuite-mcp | Needs query & readMask |
| `/people/{resourceName}` | GET | 🟡 | gsuite-mcp | Needs personFields support |

### Write Operations

| Endpoint | Method | Status | Used By | Priority |
|----------|--------|--------|---------|----------|
| `/people:createContact` | POST | ❌ | gsuite-mcp | High |
| `/people/{resourceName}:updateContact` | PATCH | ❌ | gsuite-mcp | Medium |
| `/people/{resourceName}:deleteContact` | DELETE | ❌ | gsuite-mcp | Medium |

---

## Implementation Priority

### Phase 1: Critical Write Operations (Blocking)
1. ✅ **Gmail Send** - `POST /gmail/v1/users/me/messages/send`
2. ✅ **Calendar Create** - `POST /calendar/v3/calendars/primary/events`
3. ✅ **Tasks List** - `GET /tasks/v1/lists/@default/tasks`
4. ✅ **Tasks Create** - `POST /tasks/v1/lists/@default/tasks`

### Phase 2: High Priority CRUD
1. ✅ Calendar Update/Delete
2. ✅ Tasks Update/Complete
3. ✅ Gmail Modify Labels
4. ✅ People Create Contact

### Phase 3: Completeness
1. ✅ Gmail Drafts
2. ✅ Tasks Delete
3. ✅ People Update/Delete
4. ✅ Gmail Trash/Delete

---

## Additional Requirements

### Auto-Reply Feature (Enhancement)
When Gmail send endpoint is called:
- Generate AI-powered reply using OpenAI
- Create reply message in database after 2-30 second delay
- Thread properly with original message
- Mark as `INBOX`, `UNREAD`

### Query Parameters Support

**Gmail List Messages:**
- `q` - Query string (e.g., "is:unread in:inbox")
- `maxResults` - Limit results
- `pageToken` - Pagination
- `labelIds` - Filter by labels

**Calendar List Events:**
- `timeMin` / `timeMax` - RFC3339 date range
- `maxResults` - Limit results
- `singleEvents` - Expand recurring events
- `orderBy` - Sort order (startTime)

**Tasks List:**
- `showCompleted` - Include completed tasks
- `maxResults` - Limit results

**People Connections:**
- `personFields` - Comma-separated fields to return
- `pageSize` - Limit results
- `syncToken` - Incremental sync

**People Search:**
- `query` - Search query
- `readMask` - Comma-separated fields to return
- `pageSize` - Limit results

---

## Response Format Compatibility

All responses must match Google API format exactly:

### Gmail Message
```json
{
  "id": "msg_123",
  "threadId": "thr_456",
  "labelIds": ["INBOX", "UNREAD"],
  "snippet": "Preview text...",
  "internalDate": "1234567890000",
  "payload": {
    "headers": [
      {"name": "From", "value": "alice@example.com"},
      {"name": "Subject", "value": "Hello"}
    ],
    "body": {"data": "base64-encoded-body"}
  }
}
```

### Calendar Event
```json
{
  "id": "evt_123",
  "summary": "Meeting",
  "description": "Discuss...",
  "start": {"dateTime": "2025-12-10T10:00:00-07:00"},
  "end": {"dateTime": "2025-12-10T11:00:00-07:00"},
  "attendees": [{"email": "alice@example.com"}],
  "htmlLink": "http://..."
}
```

### Task
```json
{
  "id": "task_123",
  "title": "Buy groceries",
  "notes": "Milk, eggs, bread",
  "due": "2025-12-10T00:00:00Z",
  "status": "needsAction",
  "completed": null
}
```

### Person (Contact)
```json
{
  "resourceName": "people/c123",
  "names": [{"displayName": "Alice Smith"}],
  "emailAddresses": [{"value": "alice@example.com"}],
  "phoneNumbers": [{"value": "+1-555-0100"}]
}
```

---

## Testing Matrix

Each endpoint should be tested with:
1. ✅ Valid request returns 200 OK
2. ✅ Invalid ID returns 404 Not Found
3. ✅ Missing required fields returns 400 Bad Request
4. ✅ Pagination works correctly
5. ✅ Query filtering works correctly

---

## Environment Variables

```bash
# Ish server configuration
ISH_PORT=9000
ISH_DB_PATH=./ish.db

# Auto-reply feature (optional)
ISH_AUTO_REPLY=true
ISH_OPENAI_KEY=sk-...
ISH_REPLY_DELAY_MIN=2
ISH_REPLY_DELAY_MAX=30
```

---

## Client Configuration

### pagen-agent (Jeff)
```bash
ISH_MODE=true
ISH_BASE_URL=http://localhost:9000
ISH_USER=harper
```

### gsuite-mcp
```bash
ISH_MODE=true
ISH_BASE_URL=http://localhost:9000
ISH_USER=testuser
```
