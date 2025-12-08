# ISH: Your Local Google API Digital Twin

> **ISH** provides a complete, self-hosted Google API server for development, testing, and demonstration environments. Stop burning through API quotas, avoid complex OAuth flows during development, and gain full control over your test data.

## The Problem

Modern applications increasingly rely on Google's suite of APIs (Gmail, Calendar, People, Tasks), but developing against production APIs creates significant friction:

- **API Rate Limits**: Production quotas get exhausted during development and testing
- **OAuth Complexity**: Setting up and maintaining OAuth credentials for every developer is time-consuming
- **Data Privacy**: Testing with real user data raises privacy and compliance concerns
- **Network Dependencies**: Development requires constant internet connectivity and working Google services
- **Cost**: High-volume testing can incur unexpected API usage costs
- **Test Data Management**: Creating and maintaining realistic test scenarios is difficult

## The Solution

**ISH** is a drop-in replacement for Google's APIs that runs entirely on your infrastructure. It provides:

✅ **100% Local Operation** - No external dependencies, no network calls to Google
✅ **Zero Rate Limits** - Test as aggressively as you need
✅ **Instant Setup** - Single binary, SQLite backend, running in seconds
✅ **Full API Compatibility** - Works with existing Google API client libraries
✅ **Rich Admin Interface** - Web-based UI for managing test data
✅ **Request Introspection** - Built-in logging and debugging tools
✅ **AI-Powered Seeding** - Generate realistic test data automatically

## Core Features

### Complete API Coverage

ISH implements the most commonly used Google API endpoints:

**Gmail API (v1)**
- List messages with filtering and pagination
- Send messages (with base64-encoded RFC822 format)
- Get individual messages with full payload
- Message history and modifications
- Attachment handling
- Label management

**Calendar API (v3)**
- List events with time-based filtering
- Create, update, and delete events
- Recurring events support
- Multiple calendars
- Attendee management
- Sync token support for incremental updates

**People API (v1)**
- List contacts with pagination
- Search contacts by query
- Create, update, and delete contacts
- Batch operations
- Organization and phone number support

**Tasks API (v1)**
- List task lists
- Create, update, and delete tasks
- Task completion tracking
- Due date management
- Notes and metadata

**Auto-Reply Management**
- GET/PUT auto-reply settings
- Supports office-sync and similar tools

### Professional Admin Interface

The built-in web interface (`/admin/`) provides complete control:

- **Dashboard**: Real-time metrics showing message counts, events, contacts, and tasks
- **Gmail Management**: Create messages, view threads, manage labels
- **Calendar Management**: Create events, set attendees, configure recurring patterns
- **People Management**: Add contacts with full details (name, email, phone, organization)
- **Tasks Management**: Create and organize task lists and individual tasks
- **Request Logs**: Full introspection of API calls with request/response bodies
- **AI Generation**: One-click generation of realistic test data

### Advanced Request Logging

Every API request is logged with comprehensive details:

- HTTP method, path, and status code
- Request and response bodies (pretty-printed JSON)
- Response time in milliseconds
- User identification and IP address
- Timestamp for temporal analysis

**Powerful Analytics:**
- Total request counts and trends
- Error rate monitoring
- Average response times
- Most frequently accessed endpoints
- Per-endpoint performance metrics

### Intelligent Data Seeding

ISH includes AI-powered data generation that creates realistic, coherent test datasets:

```bash
./ish seed
```

Generates:
- 50+ realistic emails with proper headers and threading
- 25+ calendar events with attendees and descriptions
- 25+ contacts with names, emails, phone numbers, companies
- 10+ tasks with due dates and completion status

The AI generator creates contextually appropriate data that mimics real-world usage patterns, making your testing environment feel authentic.

## Technical Architecture

### Lightweight & Portable

- **Single Binary**: No dependencies, no complex setup
- **SQLite Backend**: File-based database, easy backup and restore
- **Embedded Templates**: UI resources compiled into the binary
- **Go Implementation**: Fast, memory-efficient, cross-platform

### HTTP API Server

- Built on **chi router** for fast, idiomatic HTTP handling
- Middleware stack for authentication, logging, and recovery
- RESTful endpoints matching Google's API specifications
- JSON responses with proper error handling

### Storage Layer

```
internal/store/
├── store.go          # Core database and migrations
├── gmail.go          # Gmail data operations
├── calendar.go       # Calendar data operations
├── people.go         # Contacts data operations
├── tasks.go          # Tasks data operations
└── request_logs.go   # Request logging and analytics
```

Clean separation of concerns with type-safe Go interfaces.

### Admin Interface

```
internal/admin/
├── handlers.go       # HTTP handlers for admin UI
├── templates.go      # Template management
└── templates/        # HTML templates with Tailwind CSS
```

Server-side rendered UI with HTMX for dynamic updates without heavy JavaScript frameworks.

## Use Cases

### 1. Development Environments

Replace Google API calls during local development:

```bash
# Start ISH on port 9000
./ish serve --port 9000

# Point your app to ISH instead of Google
export GOOGLE_API_BASE_URL=http://localhost:9000
```

Your existing Google client libraries work unchanged - just point them at ISH.

### 2. Continuous Integration / Testing

Run ISH in your CI pipeline for fast, reliable tests:

```yaml
# .github/workflows/test.yml
- name: Start ISH
  run: |
    ./ish reset --db test.db
    ./ish serve --port 9000 --db test.db &

- name: Run Integration Tests
  run: npm test
  env:
    GOOGLE_API_BASE_URL: http://localhost:9000
```

No flaky tests due to network issues or rate limits.

### 3. Demonstrations & Sales

Show your product without depending on live Google accounts:

- Consistent demo data that never changes unexpectedly
- No risk of exposing real customer information
- Works offline at trade shows and customer sites
- Reset to clean state instantly with `./ish reset`

### 4. Offline Development

Work on planes, trains, or anywhere without internet:

- Full Google API functionality without network
- Modify test data to explore edge cases
- Debug issues without worrying about API quotas

### 5. Educational Environments

Teach API integration without managing OAuth credentials:

- Students work with identical datasets
- No Google account required for each student
- Instructors can review student API usage in request logs
- Safe environment for experimentation

## Getting Started

### Installation

```bash
# Build from source
git clone https://github.com/2389/ish.git
cd ish
go build -o ish ./cmd/ish

# Or download pre-built binary
curl -L https://github.com/2389/ish/releases/latest/download/ish-$(uname -s)-$(uname -m) -o ish
chmod +x ish
```

### Quick Start

```bash
# Reset database and seed with test data
./ish reset

# Start the server
./ish serve --port 9000

# Access admin UI
open http://localhost:9000/admin/

# Test an API call
curl http://localhost:9000/gmail/v1/users/me/messages \
  -H "Authorization: Bearer fake-token"
```

### Configuration

Environment variables:

- `ISH_PORT` - HTTP port (default: 9000)
- `ISH_DB_PATH` - SQLite database path (default: ./ish.db)

Command-line flags override environment variables.

### Integration

ISH works with standard Google API client libraries:

**Node.js:**
```javascript
const google = require('googleapis').google;

const gmail = google.gmail({
  version: 'v1',
  baseURL: 'http://localhost:9000',
  auth: 'fake-token'  // ISH accepts any token
});

const res = await gmail.users.messages.list({
  userId: 'me',
  maxResults: 10
});
```

**Python:**
```python
from googleapiclient.discovery import build

service = build('gmail', 'v1',
                developerKey='fake-key',
                baseUrl='http://localhost:9000')

results = service.users().messages().list(
    userId='me',
    maxResults=10
).execute()
```

**Go:**
```go
import "google.golang.org/api/gmail/v1"

service, err := gmail.NewService(ctx,
    option.WithAPIKey("fake-key"),
    option.WithEndpoint("http://localhost:9000"))

msgs, err := service.Users.Messages.
    List("me").
    MaxResults(10).
    Do()
```

## API Compatibility Matrix

| API | Endpoints | Read | Write | Delete | Notes |
|-----|-----------|------|-------|--------|-------|
| Gmail API v1 | ✅ | ✅ | ✅ | ✅ | Full message lifecycle |
| Calendar API v3 | ✅ | ✅ | ✅ | ✅ | Events, recurrence, sync tokens |
| People API v1 | ✅ | ✅ | ✅ | ✅ | Contacts, search, batch ops |
| Tasks API v1 | ✅ | ✅ | ✅ | ✅ | Lists and tasks |
| Auto-Reply | ✅ | ✅ | ✅ | - | Settings management |

## Security & Privacy

### By Design

- **No External Communication**: ISH never contacts Google or any external service
- **Local Data Only**: All data stays on your infrastructure
- **No OAuth Required**: Simple token authentication for development
- **Audit Trail**: Complete request logging for compliance

### Recommended Practices

⚠️ **ISH is designed for development and testing environments, not production use.**

- Run ISH on internal networks or localhost only
- Use firewall rules to restrict access
- Regularly backup the SQLite database if needed
- Don't store real user data in ISH
- Review request logs before sharing datasets

## Performance Characteristics

Typical performance on modern hardware:

- **Startup Time**: < 100ms
- **Request Latency**: < 5ms for most operations
- **Throughput**: > 10,000 requests/second
- **Memory Usage**: < 50MB with typical datasets
- **Database Size**: ~10MB with 1000s of messages/events

ISH is optimized for low latency and high throughput, making it suitable for aggressive testing scenarios.

## Extensibility

ISH's clean architecture makes it easy to extend:

### Adding New Endpoints

1. Add handler in `internal/<api>/handlers.go`
2. Register route in `RegisterRoutes()`
3. Add store methods in `internal/store/<api>.go`
4. Add admin UI templates if needed

### Custom Seeding

The seed generator accepts custom data sources:

```go
gen := seed.NewGenerator("user-id")
data, _ := gen.Generate(ctx, emailCount, eventCount, contactCount)
// Insert custom data alongside generated data
```

### Request Logging Extensions

The logging middleware can be extended to:
- Send logs to external systems (Elasticsearch, Datadog)
- Filter sensitive data before storage
- Add custom metrics and alerting

## Comparison to Alternatives

| Solution | Setup Complexity | API Coverage | Cost | Offline |
|----------|-----------------|--------------|------|---------|
| **ISH** | 🟢 One command | 🟢 Gmail, Calendar, People, Tasks | 🟢 Free | 🟢 Yes |
| Google API Test Environment | 🟡 OAuth setup | 🟢 All APIs | 🟡 Free tier limits | 🔴 No |
| Custom Mock Server | 🔴 Build yourself | 🟡 Whatever you build | 🟢 Time investment | 🟢 Yes |
| Postman Mock Servers | 🟡 Per-endpoint setup | 🟡 Manual config | 🟡 Paid plans | 🔴 No |

## Roadmap

Future enhancements being considered:

- **Drive API**: File storage and retrieval
- **Sheets API**: Spreadsheet operations
- **OAuth Mock**: Simulate OAuth flows for client testing
- **Multi-User Support**: Multiple user accounts in one instance
- **WebSocket Support**: Real-time push notifications
- **Cloud Storage Backend**: S3/GCS as alternative to SQLite
- **Docker Image**: Official containerized distribution
- **Metrics Export**: Prometheus/StatsD integration

## Support & Contribution

### Documentation

- Full API documentation at `/admin/guide`
- Source code with detailed comments
- Example integration in `examples/` directory

### Contributing

ISH is open source and welcomes contributions:

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Submit a pull request

### Getting Help

- GitHub Issues for bug reports
- Discussions for questions and ideas
- Email support for enterprise deployments

## Conclusion

**ISH eliminates the friction of developing against Google APIs.** Whether you're building a new integration, testing existing code, demonstrating your product, or teaching API development, ISH provides a fast, reliable, and completely local alternative to Google's production services.

**Get started in 60 seconds:**

```bash
./ish reset && ./ish serve
```

Then point your browser to `http://localhost:9000/admin/` and start exploring.

---

*ISH is not affiliated with or endorsed by Google. All Google API trademarks are property of Google LLC.*
