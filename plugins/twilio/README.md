# Twilio Plugin

Mock Twilio SMS and Voice APIs for local development testing.

## Features

- **SMS API**: Send messages, list messages, get message details
- **Voice API**: Initiate calls, list calls, get call details
- **Phone Numbers**: List configured phone numbers
- **Auto-accept Auth**: HTTP Basic Auth with account auto-creation
- **Async Webhooks**: Realistic status callback timing
- **Admin UI**: Schema-driven resource management

## Authentication

Use HTTP Basic Auth with Account SID as username and Auth Token as password:

```bash
curl -u AC123:token123 http://localhost:9000/2010-04-01/Accounts/AC123/Messages.json
```

Accounts are auto-created on first request. The auth token is randomly generated and returned.

## SMS Example

```bash
# Send SMS
curl -X POST "http://localhost:9000/2010-04-01/Accounts/AC123/Messages.json" \
  -u "AC123:token123" \
  -d "To=+15551234567" \
  -d "From=+15559876543" \
  -d "Body=Hello World"

# List messages
curl "http://localhost:9000/2010-04-01/Accounts/AC123/Messages.json" \
  -u "AC123:token123"

# Get message details
curl "http://localhost:9000/2010-04-01/Accounts/AC123/Messages/SM456.json" \
  -u "AC123:token123"
```

## Voice Example

```bash
# Initiate call
curl -X POST "http://localhost:9000/2010-04-01/Accounts/AC123/Calls.json" \
  -u "AC123:token123" \
  -d "To=+15551234567" \
  -d "From=+15559876543" \
  -d "Url=http://example.com/twiml"

# List calls
curl "http://localhost:9000/2010-04-01/Accounts/AC123/Calls.json" \
  -u "AC123:token123"
```

## Webhook Callbacks

Configure `status_callback` on phone numbers to receive async status updates:

```sql
UPDATE twilio_phone_numbers
SET status_callback = 'http://localhost:3000/webhook'
WHERE phone_number = '+15559876543';
```

### SMS Status Progression

- `queued` (immediate)
- `sent` (+100ms)
- `delivered` (+500ms)

### Call Status Progression

- `initiated` (immediate)
- `ringing` (+200ms)
- `in-progress` (+800ms)
- `completed` (+5-30s, random duration)

## Admin UI

Visit `http://localhost:9000/admin/twilio` to manage:

- Accounts (view SIDs, auth tokens)
- Messages (send, view status)
- Calls (initiate, track duration)
- Phone Numbers (configure webhooks)

## Testing

```bash
cd plugins/twilio
go test -v
```

Includes unit tests, handler tests, and integration tests for full SMS/Voice flows.
