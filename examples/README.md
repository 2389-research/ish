# ISH API Integration Examples

This directory contains Python examples demonstrating how to integrate with ISH's fake API endpoints. These examples show real-world usage patterns for each supported plugin.

## Prerequisites

```bash
# Install required dependencies
pip install requests

# Or use uv (recommended)
uv pip install requests
```

## Quick Start

1. **Start ISH server with seeded data:**

```bash
# Seed the database
./ish seed

# Start the server
./ish serve
```

2. **Run any example:**

```bash
# Make scripts executable
chmod +x examples/*.py

# Run an example
python examples/google_gmail.py
```

## Available Examples

### Google APIs

#### Gmail (`google_gmail.py`)
Demonstrates Gmail API integration including:
- List messages in mailbox
- Search messages by query
- Get detailed message information
- Send new emails
- Move messages to trash

**Usage:**
```bash
python examples/google_gmail.py
```

**Key features:**
- Message listing with pagination
- Query-based search
- Full message details with headers
- Error handling examples

---

#### Calendar (`google_calendar.py`)
Shows how to manage calendar events:
- List upcoming events
- Get event details
- Create new events
- Update existing events
- Delete events

**Usage:**
```bash
python examples/google_calendar.py
```

**Key features:**
- Time-based filtering
- Event creation with start/end times
- Update event properties
- Full CRUD operations

---

#### Tasks (`google_tasks.py`)
Task management integration:
- List tasks (completed and pending)
- Get task details
- Create new tasks with due dates
- Update tasks
- Mark tasks as completed
- Delete tasks

**Usage:**
```bash
python examples/google_tasks.py
```

**Key features:**
- Task filtering by status
- Due date management
- Batch task creation
- Completion tracking

---

### Communication APIs

#### SendGrid (`sendgrid.py`)
Email sending via SendGrid API:
- Send plain text emails
- Send HTML emails
- Manage suppression lists
- Add/remove email suppressions
- Batch email sending

**Usage:**
```bash
python examples/sendgrid.py
```

**Key features:**
- Multiple content types (text/HTML)
- Suppression list management
- Batch sending capabilities
- Error handling for suppressions

---

#### Twilio (`twilio.py`)
SMS and voice call integration:
- Send SMS messages
- List sent/received messages
- Make outbound calls
- List call history
- Filter messages by recipient

**Usage:**
```bash
python examples/twilio.py
```

**Key features:**
- SMS verification codes
- Appointment reminders
- 2FA code sending
- Call management
- Message filtering

---

### Developer Tools

#### GitHub (`github.py`)
Repository and issue management:
- List user repositories
- Get repository details
- List/create issues
- Create pull requests
- Add comments to issues/PRs
- List open/closed issues

**Usage:**
```bash
python examples/github.py
```

**Key features:**
- Full repository information
- Issue labeling
- PR creation with markdown
- Comment management

---

### Smart Home

#### Home Assistant (`homeassistant.py`)
Smart home device control:
- Get all entity states
- Control lights (on/off, brightness)
- Manage thermostats
- Control media players
- Run automation scenarios
- Check security sensors

**Usage:**
```bash
python examples/homeassistant.py
```

**Key features:**
- Multi-domain entity control
- Automation scenarios (morning routine, night mode)
- State management
- Service calls with parameters

**Note:** Home Assistant requires valid access tokens. Run `./ish seed homeassistant` to see available test tokens.

---

## Configuration

Each example can be configured with custom base URL and authentication:

```python
# Google APIs
client = ISHGmailClient(
    base_url="http://localhost:9000",
    user_id="me"
)

# SendGrid
client = ISHSendGridClient(
    base_url="http://localhost:9000",
    api_key="SG.your-test-key"
)

# Twilio
client = ISHTwilioClient(
    base_url="http://localhost:9000",
    account_sid="AC_test_account",
    auth_token="test_token"
)

# GitHub
client = ISHGitHubClient(
    base_url="http://localhost:9000",
    token="gh_test_token"
)

# Home Assistant
client = ISHHomeAssistantClient(
    base_url="http://localhost:9000",
    token="token_home_main"
)
```

## Testing Against Real ISH Server

1. **Start ISH on custom port:**

```bash
./ish serve --port 8080
```

2. **Update example to use custom port:**

```python
client = ISHGmailClient(base_url="http://localhost:8080")
```

3. **Run the example:**

```bash
python examples/google_gmail.py
```

## Getting Test Data

Each plugin can be seeded individually:

```bash
# Seed all plugins
./ish seed

# Seed specific plugin
./ish seed google
./ish seed sendgrid
./ish seed twilio
./ish seed github
./ish seed homeassistant
```

After seeding, check the output for test tokens and credentials.

## Common Patterns

### Error Handling

All examples demonstrate proper error handling:

```python
try:
    messages = client.list_messages()
    # Process messages...
except requests.exceptions.HTTPError as e:
    print(f"HTTP Error: {e.response.status_code}")
    print(f"Message: {e.response.text}")
except Exception as e:
    print(f"Unexpected error: {e}")
```

### Pagination

Examples show how to handle pagination:

```python
# Limit results
messages = client.list_messages(max_results=10)

# Page through results
for page in range(5):
    messages = client.list_messages(
        max_results=20,
        page_token=next_page_token
    )
```

### Batch Operations

Several examples demonstrate batch operations:

```python
# Send multiple emails
recipients = ["user1@example.com", "user2@example.com"]
for recipient in recipients:
    client.send_mail(
        to_email=recipient,
        from_email="noreply@app.com",
        subject="Batch Message",
        content="..."
    )
```

## API Documentation

For complete API documentation, visit the ISH admin UI:

```bash
# Start ISH
./ish serve

# Open in browser
open http://localhost:9000/admin/
```

The admin UI provides:
- Live API endpoint documentation
- Real-time request/response logs
- Plugin status and health checks
- Seeded data inspection

## Troubleshooting

### Connection Refused

```
Error: Connection refused on localhost:9000
```

**Solution:** Make sure ISH server is running:
```bash
./ish serve
```

### Authentication Errors

```
HTTP Error: 401 Unauthorized
```

**Solution:** Check your API token/credentials. For seeded data, run:
```bash
./ish seed <plugin-name>
```

And note the test credentials printed in the output.

### No Data Found

```
Found 0 messages
```

**Solution:** Seed the database first:
```bash
./ish seed
```

## Advanced Usage

### Custom Client Implementation

You can create your own client based on these examples:

```python
import requests

class MyCustomClient:
    def __init__(self, base_url="http://localhost:9000"):
        self.base_url = base_url
        self.headers = {"Authorization": "Bearer my-token"}

    def custom_method(self):
        url = f"{self.base_url}/api/endpoint"
        response = requests.get(url, headers=self.headers)
        response.raise_for_status()
        return response.json()
```

### Integration Testing

Use these examples as a foundation for integration tests:

```python
import unittest
from examples.google_gmail import ISHGmailClient

class TestGmailIntegration(unittest.TestCase):
    def setUp(self):
        self.client = ISHGmailClient(base_url="http://localhost:9000")

    def test_list_messages(self):
        messages = self.client.list_messages()
        self.assertIsInstance(messages, dict)
        self.assertIn("messages", messages)
```

## Contributing

To add new examples:

1. Create a new Python file in `examples/`
2. Follow the existing pattern:
   - Client class with clear method names
   - Comprehensive `main()` demonstrating all features
   - Error handling examples
   - Clear section headers with separators
3. Update this README with example documentation
4. Test against running ISH server

## License

These examples are part of the ISH project.
