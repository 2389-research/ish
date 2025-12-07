# SendGrid Plugin

A digital twin implementation of the SendGrid Mail API v3 for ISH. This plugin simulates SendGrid's email sending, message retrieval, and suppression management APIs.

## Features

- **Mail Send API**: Send emails via the v3/mail/send endpoint
- **Messages API**: Retrieve sent message details and history
- **Suppression Management**: Manage bounces, blocks, and spam reports
- **API Key Authentication**: Bearer token-based authentication
- **Multiple Accounts**: Support for multiple SendGrid accounts

## API Endpoints

### Mail Send

```bash
POST /v3/mail/send
Authorization: Bearer SG.xxxx
Content-Type: application/json

{
  "personalizations": [
    {
      "to": [{"email": "recipient@example.com", "name": "Recipient"}],
      "subject": "Test Email"
    }
  ],
  "from": {"email": "sender@example.com", "name": "Sender"},
  "subject": "Test Email",
  "content": [
    {"type": "text/plain", "value": "This is a test email"},
    {"type": "text/html", "value": "<p>This is a test email</p>"}
  ]
}
```

**Response**: 202 Accepted with `X-Message-Id` header

### Messages

```bash
# List messages
GET /v3/messages
Authorization: Bearer SG.xxxx

# Get specific message
GET /v3/messages/{message_id}
Authorization: Bearer SG.xxxx
```

### Suppressions

```bash
# List bounces
GET /v3/suppression/bounces
Authorization: Bearer SG.xxxx

# Delete bounce
DELETE /v3/suppression/bounces/{email}
Authorization: Bearer SG.xxxx

# List blocks
GET /v3/suppression/blocks
Authorization: Bearer SG.xxxx

# Delete block
DELETE /v3/suppression/blocks/{email}
Authorization: Bearer SG.xxxx

# List spam reports
GET /v3/suppression/spam_reports
Authorization: Bearer SG.xxxx

# Delete spam report
DELETE /v3/suppression/spam_reports/{email}
Authorization: Bearer SG.xxxx
```

## Database Schema

### Tables

- **sendgrid_accounts**: Account management
- **sendgrid_api_keys**: API key storage and validation
- **sendgrid_messages**: Sent message records
- **sendgrid_suppressions**: Bounce, block, and spam report tracking

## Testing

Run the integration tests:

```bash
cd plugins/sendgrid
go test -v
```

## Test Data

The plugin includes seed data generation. When running `./ish seed`, test accounts, API keys, messages, and suppressions are created automatically.

### Sample API Keys

After seeding, API keys are printed to the console in the format:

```
=== SendGrid Test API Keys ===
Account 1 (harper@example.com): SG.xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
Account 2 (alice@startup.io): SG.xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
Account 3 (bob@bigcorp.com): SG.xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
```

## Example Usage

```bash
# Start ISH server
./ish serve

# Send a test email (use API key from seed output)
curl -X POST http://localhost:8080/v3/mail/send \
  -H "Authorization: Bearer SG.xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "personalizations": [{
      "to": [{"email": "test@example.com"}]
    }],
    "from": {"email": "noreply@example.com"},
    "subject": "Test Email",
    "content": [{"type": "text/plain", "value": "Hello World"}]
  }'

# List sent messages
curl http://localhost:8080/v3/messages \
  -H "Authorization: Bearer SG.xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"

# List bounces
curl http://localhost:8080/v3/suppression/bounces \
  -H "Authorization: Bearer SG.xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
```

## Architecture

### Authentication

The plugin uses Bearer token authentication via the `Authorization` header. API keys follow the SendGrid format: `SG.{uuid}`.

### Message Storage

All sent messages are stored in the database with:
- Unique message ID (UUID)
- From/To addresses and names
- Subject and content (text/HTML)
- Status (always "delivered" in simulation)
- Timestamp

### Suppression Management

Suppressions are categorized by type:
- **bounce**: Email delivery failures
- **block**: Unsubscribed or blocked recipients
- **spam_report**: Recipients who marked email as spam

Each suppression includes the email address, type, reason, and creation timestamp.

## Differences from Real SendGrid

1. **Simplified Personalization**: Only the first recipient is processed
2. **No Email Delivery**: Messages are stored but not actually sent
3. **Limited Scopes**: API key scopes are stored but not enforced
4. **No Rate Limiting**: No request throttling implemented
5. **No Webhooks**: Event webhook callbacks not supported
6. **Always Delivered**: All messages report "delivered" status

## Implementation Files

- `plugin.go`: Main plugin registration and routing
- `store.go`: Database operations and schema
- `handlers.go`: HTTP endpoint handlers
- `seed.go`: Test data generation
- `integration_test.go`: Comprehensive integration tests
