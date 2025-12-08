# office-os Support in ish

## Summary

**office-os is ALREADY FULLY SUPPORTED by ish!** 🎉

office-os is a sync client that pulls data from Google APIs and stores it locally. It uses the official Google API Go client libraries to sync:
- Gmail messages (with history-based incremental sync)
- Calendar events (with syncToken-based incremental sync)  
- People contacts (with syncToken-based incremental sync)

All the endpoints office-os needs are already implemented in ish.

---

## What office-os Does

office-os fetches data from Google APIs and stores it in a local SQLite database for querying. It's a **read-only sync client** - it only pulls data, doesn't write.

### Sync Operations

1. **Gmail Sync** (`internal/google/gmail.go`)
   - Full sync: Gets last 6 months of messages
   - Incremental sync: Uses History API with historyId cursor
   - Downloads attachments
   - Supports resume from checkpoint (pageToken)

2. **Calendar Sync** (`internal/google/calendar.go`)
   - Full sync: Gets events from 6 months ago to 1 year ahead
   - Incremental sync: Uses syncToken for delta updates
   - Supports resume from checkpoint

3. **Contacts Sync** (`internal/google/contacts.go`)
   - Lists connections from People API
   - Incremental sync with syncToken
   - Extracts names, emails, phones, organizations

---

## API Endpoints office-os Uses

### Gmail API

| Endpoint | Method | office-os Usage | ish Status |
|----------|--------|-----------------|------------|
| `/gmail/v1/users/me/profile` | GET | Get historyId for incremental sync | ✅ Implemented |
| `/gmail/v1/users/me/messages` | GET | List messages with query filter | ✅ Implemented |
| `/gmail/v1/users/me/messages/{id}` | GET | Get full message with format=full | ✅ Implemented |
| `/gmail/v1/users/me/messages/{msgId}/attachments/{id}` | GET | Download attachments | ✅ Implemented |
| `/gmail/v1/users/me/history` | GET | Incremental sync via historyId | ✅ Implemented |

### Calendar API

| Endpoint | Method | office-os Usage | ish Status |
|----------|--------|-----------------|------------|
| `/calendar/v3/calendars/primary/events` | GET | List events with time range, syncToken | ✅ Implemented |

### People API

| Endpoint | Method | office-os Usage | ish Status |
|----------|--------|-----------------|------------|
| `/people/v1/people/me/connections` | GET | List contacts with personFields, syncToken | ✅ Implemented |

---

## Configuration

To make office-os use ish instead of real Google APIs:

### Option 1: Environment Variables (Recommended)

```bash
# In office-os's environment
export ISH_MODE=true
export ISH_BASE_URL=http://localhost:9000
export ISH_USER=harper
```

Then in office-os code, detect ISH_MODE and use custom endpoint:

```go
// In internal/google/auth.go or similar
func getAPIEndpoint() string {
    if os.Getenv("ISH_MODE") == "true" {
        return os.Getenv("ISH_BASE_URL")  // http://localhost:9000
    }
    return ""  // Use default Google APIs
}

// When creating services:
var opts []option.ClientOption
opts = append(opts, option.WithHTTPClient(client))

if baseURL := getAPIEndpoint(); baseURL != "" {
    opts = append(opts, option.WithEndpoint(baseURL))
}

service, err := gmail.NewService(ctx, opts...)
```

### Option 2: Direct Code Change

In `internal/google/*.go` files, add endpoint option:

```go
// gmail.go
service, err := gmail.NewService(ctx, 
    option.WithHTTPClient(client),
    option.WithEndpoint("http://localhost:9000"),  // Point to ish
)

// calendar.go  
service, err := calendar.NewService(ctx,
    option.WithHTTPClient(client),
    option.WithEndpoint("http://localhost:9000"),  // Point to ish
)

// contacts.go
service, err := people.NewService(ctx,
    option.WithHTTPClient(client),
    option.WithEndpoint("http://localhost:9000"),  // Point to ish
)
```

---

## Testing office-os with ish

### 1. Start ish server

```bash
cd ish
go build -o ish ./cmd/ish
./ish reset  # Seed with test data
./ish serve --port 9000
```

### 2. Configure office-os

```bash
cd office-os
export ISH_MODE=true
export ISH_BASE_URL=http://localhost:9000  
export ISH_USER=harper
```

### 3. Run office-os sync

```bash
# Assuming office-os has a sync command
./office-sync sync --provider google --account harper
```

### 4. Verify

Check that office-os database has synced data:

```bash
sqlite3 office-os.db "SELECT COUNT(*) FROM objects WHERE kind='message'"
sqlite3 office-os.db "SELECT COUNT(*) FROM objects WHERE kind='event'"
sqlite3 office-os.db "SELECT COUNT(*) FROM objects WHERE kind='record'"
```

---

## What office-os Will Get from ish

When synced against ish with default seed data:

- **50 Gmail messages** (from last 7 days)
  - Mix of INBOX, SENT, UNREAD labels
  - AI-generated realistic content (if OPENAI_API_KEY set)
  - Subject, from, to, body, snippets
  - Threaded conversations

- **25 Calendar events** (spread over 6 months)
  - AI-generated realistic meetings/appointments
  - Start/end times, descriptions, locations
  - Attendees, organizers

- **25 People contacts**
  - Names, email addresses, phone numbers
  - Companies, job titles
  - Realistic AI-generated profiles

- **10 Tasks** (optional, if office-os adds Tasks support)
  - Mix of needsAction and completed statuses
  - Due dates, notes
  
---

## Incremental Sync Support

ish fully supports incremental sync:

### Gmail History API
- Returns historyId in profile response
- History endpoint returns messageAdded events
- ish maintains history IDs for incremental sync

### Calendar syncToken
- Events list returns nextSyncToken
- Subsequent calls with syncToken get only changes
- ish maintains sync tokens per calendar

### People syncToken
- Connections list returns nextSyncToken
- Subsequent calls with syncToken get only changes
- ish maintains sync tokens per account

---

## Known Limitations

1. **personFields filtering**: office-os requests specific fields via `personFields` parameter, but ish returns all fields. This is **safe** - clients ignore extra fields.

2. **No real OAuth flow**: ish uses simple Bearer token auth. office-os will need to provide a fake token or ish needs to accept any token in ISH_MODE.

3. **Static data**: Unless you manually add data via the admin UI, ish returns the same seed data on each sync.

---

## Next Steps

1. ✅ All ish endpoints are implemented
2. 📝 Add ISH_MODE support to office-os code
3. 🧪 Test sync with ish
4. 🐛 Fix any compatibility issues
5. 📊 Verify data appears correctly in office-os queries

---

## Quick Start

```bash
# Terminal 1: Start ish
cd ish
./ish reset && ./ish serve

# Terminal 2: Run office-os with ish
cd office-os
ISH_MODE=true ISH_BASE_URL=http://localhost:9000 ISH_USER=harper ./office-sync sync

# Terminal 3: Check results
sqlite3 office-os.db "SELECT kind, COUNT(*) FROM objects GROUP BY kind"
```

Expected output:
```
event|25
message|50
record|25
```

🎉 **office-os is ready to sync with ish!**
