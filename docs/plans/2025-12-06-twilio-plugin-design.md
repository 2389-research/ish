# Twilio Plugin Design

**Date**: 2025-12-06
**Status**: Approved
**APIs**: SMS (Messaging) and Voice (Calls)

## Overview

A self-contained Twilio API emulator plugin for ISH that simulates SMS messaging and voice calls with realistic webhook callbacks. Follows ISH's DatabasePlugin interface pattern, matching the architecture of Discord, Google, and OAuth plugins.

## Architecture

### Plugin Structure
- **Self-contained**: All Twilio logic lives in `plugins/twilio/`
- **Database-driven**: Uses SQLite tables with twilio_ prefix
- **Auto-accept auth**: First API request auto-creates account (like Discord webhooks)
- **Async webhooks**: Background goroutines deliver status callbacks with realistic delays
- **Admin UI integration**: Schema-driven resource management

### Core Components

1. **`plugin.go`**: Interface implementation, route registration, health checks
2. **`store.go`**: Database layer with all tables and queries
3. **`handlers.go`**: HTTP handlers for Twilio API endpoints
4. **`webhooks.go`**: Async webhook delivery system
5. **`schema.go`**: Admin UI resource definitions

## API Endpoints

### SMS API (Twilio Messaging API 2010-04-01)

```
POST /2010-04-01/Accounts/{AccountSid}/Messages.json
  - Send SMS message
  - Body params: To, From, Body
  - Returns: Message resource with sid, status=queued
  - Triggers: Async webhook progression (queued → sent → delivered)

GET /2010-04-01/Accounts/{AccountSid}/Messages.json
  - List messages with pagination
  - Query params: PageSize, PageToken, To, From, DateSent
  - Returns: Paginated message list

GET /2010-04-01/Accounts/{AccountSid}/Messages/{MessageSid}.json
  - Get single message details
  - Returns: Full message resource with current status
```

### Voice API (Twilio Voice API 2010-04-01)

```
POST /2010-04-01/Accounts/{AccountSid}/Calls.json
  - Initiate outbound call
  - Body params: To, From, Url (TwiML)
  - Returns: Call resource with sid, status=initiated
  - Triggers: Async status progression (initiated → ringing → in-progress → completed)

GET /2010-04-01/Accounts/{AccountSid}/Calls.json
  - List calls with pagination
  - Query params: PageSize, PageToken, To, From, Status
  - Returns: Paginated call list

GET /2010-04-01/Accounts/{AccountSid}/Calls/{CallSid}.json
  - Get single call details
  - Returns: Full call resource with duration
```

### Phone Numbers API

```
GET /2010-04-01/Accounts/{AccountSid}/IncomingPhoneNumbers.json
  - List account's phone numbers
  - Returns: Phone number resources with webhook configs
  - Note: Numbers are auto-seeded, not dynamically purchased
```

## Database Schema

### `twilio_accounts`
Auto-created on first API request with any Account SID.

```sql
CREATE TABLE twilio_accounts (
  account_sid TEXT PRIMARY KEY,  -- AC + 32 hex chars
  auth_token TEXT NOT NULL,      -- Random token for HTTP Basic Auth
  friendly_name TEXT,
  status TEXT DEFAULT 'active',  -- active/suspended
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### `twilio_phone_numbers`
Virtual phone inventory with webhook configurations.

```sql
CREATE TABLE twilio_phone_numbers (
  sid TEXT PRIMARY KEY,              -- PN + 32 hex chars
  account_sid TEXT NOT NULL,
  phone_number TEXT NOT NULL,        -- E.164 format (+1234567890)
  friendly_name TEXT,
  voice_url TEXT,                    -- TwiML webhook for incoming calls
  voice_method TEXT DEFAULT 'POST',  -- POST/GET
  sms_url TEXT,                      -- Webhook for incoming SMS
  sms_method TEXT DEFAULT 'POST',
  status_callback TEXT,              -- Status update webhook
  status_callback_method TEXT DEFAULT 'POST',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (account_sid) REFERENCES twilio_accounts(account_sid)
);
CREATE INDEX idx_phone_numbers_account ON twilio_phone_numbers(account_sid);
```

### `twilio_messages`
SMS message records with status tracking.

```sql
CREATE TABLE twilio_messages (
  sid TEXT PRIMARY KEY,          -- SM + 32 hex chars
  account_sid TEXT NOT NULL,
  from_number TEXT NOT NULL,     -- E.164 phone number
  to_number TEXT NOT NULL,
  body TEXT,
  status TEXT DEFAULT 'queued',  -- queued/sent/delivered/failed
  direction TEXT,                -- inbound/outbound-api
  date_created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  date_sent TIMESTAMP,
  date_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  num_segments INTEGER DEFAULT 1, -- 1 per 160 chars
  price REAL,
  price_unit TEXT DEFAULT 'USD',
  FOREIGN KEY (account_sid) REFERENCES twilio_accounts(account_sid)
);
CREATE INDEX idx_messages_account ON twilio_messages(account_sid);
CREATE INDEX idx_messages_status ON twilio_messages(status);
CREATE INDEX idx_messages_date ON twilio_messages(date_created);
```

### `twilio_calls`
Voice call records with duration tracking.

```sql
CREATE TABLE twilio_calls (
  sid TEXT PRIMARY KEY,              -- CA + 32 hex chars
  account_sid TEXT NOT NULL,
  from_number TEXT NOT NULL,
  to_number TEXT NOT NULL,
  status TEXT DEFAULT 'initiated',   -- initiated/ringing/in-progress/completed/failed
  direction TEXT,                    -- inbound/outbound-api
  duration INTEGER,                  -- Seconds (NULL until completed)
  date_created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  date_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  answered_by TEXT,                  -- human/machine (for AMD)
  FOREIGN KEY (account_sid) REFERENCES twilio_accounts(account_sid)
);
CREATE INDEX idx_calls_account ON twilio_calls(account_sid);
CREATE INDEX idx_calls_status ON twilio_calls(status);
```

### `twilio_webhook_configs`
User-registered webhook URLs for status callbacks.

```sql
CREATE TABLE twilio_webhook_configs (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  account_sid TEXT NOT NULL,
  resource_type TEXT NOT NULL,   -- message/call
  event_type TEXT NOT NULL,      -- status_change
  url TEXT NOT NULL,
  method TEXT DEFAULT 'POST',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (account_sid) REFERENCES twilio_accounts(account_sid)
);
CREATE INDEX idx_webhook_configs_account ON twilio_webhook_configs(account_sid);
```

### `twilio_webhook_queue`
Pending webhook deliveries with scheduling.

```sql
CREATE TABLE twilio_webhook_queue (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  resource_sid TEXT NOT NULL,        -- SM* or CA* being updated
  webhook_url TEXT NOT NULL,
  payload TEXT NOT NULL,             -- JSON body
  scheduled_at TIMESTAMP NOT NULL,   -- Delivery time (now + delay)
  delivered_at TIMESTAMP,            -- NULL until sent
  status TEXT DEFAULT 'pending',     -- pending/delivered/failed
  attempts INTEGER DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_webhook_queue_schedule ON twilio_webhook_queue(scheduled_at, status);
```

## Webhook Implementation

### SMS Lifecycle (Automatic Progression)

1. **queued** (immediate): Message created, webhook fired instantly
2. **sent** (+100ms): Message left ISH, `MessageStatus=sent` webhook
3. **delivered** (+500ms): Final state, `MessageStatus=delivered` webhook

### Voice Call Lifecycle

1. **initiated** (immediate): Call created
2. **ringing** (+200ms): Destination ringing
3. **in-progress** (+800ms): Call answered, duration tracking starts
4. **completed** (+random 5-30s): Call ended, final duration set

### Core Functions

**`StartWebhookWorker()`**
- Polls `twilio_webhook_queue` every 100ms
- Delivers webhooks where `scheduled_at <= NOW()`
- Updates `delivered_at` and `status`
- Retries failed deliveries up to 3 attempts

**`QueueMessageWebhook(messageSid, status, delay)`**
- Inserts webhook into queue with `scheduled_at = NOW() + delay`
- Looks up phone number's `sms_url` and `status_callback` configs
- Constructs Twilio-formatted form-encoded payload
- All message fields included in payload

**`QueueCallWebhook(callSid, status, delay)`**
- Same pattern for calls
- Uses `voice_url` and `status_callback` from phone number config
- Includes `CallDuration` field for completed calls

### Webhook Payload Format

Form-encoded POST matching Twilio's format:

```
MessageSid=SM123...
MessageStatus=delivered
From=+15551234567
To=+15559876543
Body=Hello+World
AccountSid=AC123...
```

Webhooks fire for **every status transition**, not just final state. This matches production Twilio behavior where apps track message/call progress through multiple callbacks.

## Authentication

**HTTP Basic Auth** with auto-accept pattern:

- Username: Account SID (AC...)
- Password: Auth Token
- First request with any Account SID auto-creates the account
- Auth token is randomly generated and returned in response
- Subsequent requests validate token from database

Example:
```bash
curl -X POST https://localhost:9000/2010-04-01/Accounts/AC123/Messages.json \
  -u AC123:auto_generated_token \
  -d "To=+15551234567" \
  -d "From=+15559876543" \
  -d "Body=Hello World"
```

## Admin UI

Schema-driven resources auto-generated at `/admin/twilio/`:

### Accounts (`/admin/twilio/accounts`)
- **List view**: account_sid, friendly_name, status, created_at
- **Create**: Generates new AC sid + auth_token, optional friendly_name
- **Delete**: Soft delete (status=suspended), cascades to messages/calls
- **Actions**: "Copy Auth Token", "Reset Token"

### Messages (`/admin/twilio/messages`)
- **List view**: sid, from→to, body preview, status, date_created
- **Create**: Send new SMS (auto-queues webhooks)
- **Detail**: Full message with all status transitions
- **Filters**: By account, status, direction, date range

### Calls (`/admin/twilio/calls`)
- **List view**: sid, from→to, status, duration, date_created
- **Create**: Initiate call (triggers async status progression)
- **Detail**: Full call record with TwiML execution log
- **Filters**: By account, status, direction

### Phone Numbers (`/admin/twilio/phone_numbers`)
- **List view**: phone_number, friendly_name, account owner
- **Create**: Add virtual number with webhook configs
- **Edit**: Update voice_url, sms_url, status callbacks
- **Delete**: Soft delete (prevents new messages/calls)

## Seeding Strategy

`Seed(size string)` creates realistic test data:

### Small
- 1 account
- 3 phone numbers
- 10 messages (mix of queued/sent/delivered)
- 5 calls (mix of statuses)

### Medium
- 3 accounts
- 10 phone numbers
- 50 messages
- 20 calls

### Large
- 10 accounts
- 30 phone numbers
- 200 messages
- 100 calls

### Data Characteristics
- Phone numbers use realistic US formats (+1-555-xxx-xxxx)
- Messages include common SMS patterns (verification codes, notifications, conversations)
- Calls have varied durations (30s-600s)
- Half of messages/calls are delivered/completed, rest show in-progress states
- Representative distribution of statuses for webhook testing

### AI Seeding (with `--ai` flag)
- Generates contextual SMS conversations
- Realistic caller IDs and business names
- Business scenarios: appointment reminders, order confirmations, support tickets
- Natural message timing and threading

## SID Generation

Twilio-style SIDs with appropriate prefixes:

- **Account**: `AC` + 32 hex chars
- **Message**: `SM` + 32 hex chars
- **Call**: `CA` + 32 hex chars
- **Phone Number**: `PN` + 32 hex chars

SIDs are globally unique across the plugin and generated using crypto/rand for realistic entropy.

## Testing Strategy

### Unit Tests
- SID generation uniqueness
- Webhook payload formatting
- Status transition logic
- Auth token validation

### Integration Tests
- Full SMS send → webhook delivery flow
- Call lifecycle with duration tracking
- Pagination with PageToken
- Account auto-creation on first request

### End-to-End Tests
- Send SMS via API, verify webhook received
- Initiate call, verify status progression
- List messages with filters
- Admin UI CRUD operations

## Future Enhancements

Not in initial scope, but possible additions:

- **TwiML execution**: Parse and simulate TwiML responses for calls
- **MMS support**: Image/video attachments for messages
- **Transcription**: Mock call transcripts
- **Phone number provisioning**: Dynamic number purchase API
- **Subaccounts**: Hierarchical account structure
- **WhatsApp API**: WhatsApp messaging endpoints
- **Programmable Video**: Video room management
