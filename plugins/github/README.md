# GitHub Plugin

A comprehensive GitHub REST API v3 emulator for ISH that simulates repositories, issues, pull requests, reviews, comments, and webhooks.

## Overview

The GitHub plugin provides a complete digital twin of GitHub's REST API, allowing applications to develop and test against a local GitHub-compatible API without requiring actual GitHub infrastructure. This is particularly useful for:

- Developing GitHub integrations and applications locally
- Testing webhook delivery without exposing endpoints to the internet
- Creating reproducible test environments for CI/CD pipelines
- Prototyping GitHub Apps and OAuth applications

## Features

The plugin implements the following GitHub API functionality:

### User Management
- Get authenticated user information
- Get user by username
- Update user profile

### Repository Management
- Create repositories
- List user repositories
- Get repository details
- Update repository settings
- Delete repositories

### Issue Management
- Create issues
- List issues (with filtering)
- Get issue details
- Update issue state and metadata
- Close/reopen issues

### Pull Request Management
- Create pull requests
- List pull requests
- Get PR details
- Merge pull requests
- PR state management

### Comments
- Create issue/PR comments
- List comments
- Update comments
- Delete comments

### Reviews
- Create PR reviews (PENDING, APPROVED, CHANGES_REQUESTED, COMMENTED)
- List reviews
- Submit pending reviews
- Dismiss reviews

### Webhooks
- Create webhooks with SSRF protection
- List webhooks
- Get webhook details
- Update webhooks
- Delete webhooks
- Test webhook delivery
- Automatic webhook firing on events
- HMAC signature validation (X-Hub-Signature-256)

## Authentication

The GitHub plugin uses GitHub-style personal access tokens for authentication. All API requests must include an `Authorization` header with a valid token.

### Token Formats

The plugin supports GitHub's standard token prefixes:
- `ghp_*` - Personal access tokens
- `gho_*` - OAuth access tokens
- `ghs_*` - Server-to-server tokens

### Authentication Header

Include the token in one of these formats:

```bash
Authorization: Bearer ghp_abc123xyz
```

or

```bash
Authorization: token ghp_abc123xyz
```

### Creating Tokens

Tokens are automatically created when users are added via seeding or the admin interface. You can also create tokens programmatically through the store layer.

## API Endpoints

All endpoints are prefixed with the plugin mount point (typically `/` on port 9000).

### Users

#### Get Authenticated User
```bash
GET /user
Authorization: Bearer ghp_abc123
```

Response:
```json
{
  "login": "alice",
  "id": 1,
  "type": "User",
  "name": "Alice Smith",
  "email": "alice@example.com",
  "avatar_url": "https://avatars.githubusercontent.com/u/1",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

#### Get User by Username
```bash
GET /users/{username}
Authorization: Bearer ghp_abc123
```

### Repositories

#### Create Repository
```bash
POST /user/repos
Authorization: Bearer ghp_abc123
Content-Type: application/json

{
  "name": "my-repo",
  "description": "A test repository",
  "private": false
}
```

Response:
```json
{
  "id": 1,
  "name": "my-repo",
  "full_name": "alice/my-repo",
  "description": "A test repository",
  "private": false,
  "owner": {
    "login": "alice",
    "id": 1,
    "type": "User"
  },
  "default_branch": "main",
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

#### List User Repositories
```bash
GET /user/repos
Authorization: Bearer ghp_abc123
```

#### Get Repository
```bash
GET /repos/{owner}/{repo}
Authorization: Bearer ghp_abc123
```

### Issues

#### Create Issue
```bash
POST /repos/{owner}/{repo}/issues
Authorization: Bearer ghp_abc123
Content-Type: application/json

{
  "title": "Bug: Login fails",
  "body": "Users cannot log in with valid credentials"
}
```

Response:
```json
{
  "number": 1,
  "title": "Bug: Login fails",
  "body": "Users cannot log in with valid credentials",
  "state": "open",
  "user": {
    "login": "alice",
    "id": 1
  },
  "comments": 0,
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

#### List Issues
```bash
GET /repos/{owner}/{repo}/issues
Authorization: Bearer ghp_abc123
```

#### Get Issue
```bash
GET /repos/{owner}/{repo}/issues/{number}
Authorization: Bearer ghp_abc123
```

#### Update Issue
```bash
PATCH /repos/{owner}/{repo}/issues/{number}
Authorization: Bearer ghp_abc123
Content-Type: application/json

{
  "state": "closed",
  "state_reason": "completed"
}
```

### Pull Requests

#### Create Pull Request
```bash
POST /repos/{owner}/{repo}/pulls
Authorization: Bearer ghp_abc123
Content-Type: application/json

{
  "title": "Add feature X",
  "body": "This PR adds feature X",
  "head": "feature-x",
  "base": "main"
}
```

Response:
```json
{
  "number": 1,
  "title": "Add feature X",
  "body": "This PR adds feature X",
  "state": "open",
  "merged": false,
  "draft": false,
  "head": {
    "ref": "feature-x",
    "sha": "abc123"
  },
  "base": {
    "ref": "main",
    "sha": "def456"
  },
  "user": {
    "login": "alice",
    "id": 1
  },
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-15T10:30:00Z"
}
```

#### List Pull Requests
```bash
GET /repos/{owner}/{repo}/pulls
Authorization: Bearer ghp_abc123
```

#### Get Pull Request
```bash
GET /repos/{owner}/{repo}/pulls/{number}
Authorization: Bearer ghp_abc123
```

#### Merge Pull Request
```bash
PUT /repos/{owner}/{repo}/pulls/{number}/merge
Authorization: Bearer ghp_abc123
Content-Type: application/json

{
  "commit_message": "Merge pull request #1"
}
```

### Comments

#### Create Comment
```bash
POST /repos/{owner}/{repo}/issues/{number}/comments
Authorization: Bearer ghp_abc123
Content-Type: application/json

{
  "body": "This looks good!"
}
```

#### List Comments
```bash
GET /repos/{owner}/{repo}/issues/{number}/comments
Authorization: Bearer ghp_abc123
```

#### Update Comment
```bash
PATCH /repos/{owner}/{repo}/issues/comments/{comment_id}
Authorization: Bearer ghp_abc123
Content-Type: application/json

{
  "body": "Updated comment text"
}
```

#### Delete Comment
```bash
DELETE /repos/{owner}/{repo}/issues/comments/{comment_id}
Authorization: Bearer ghp_abc123
```

### Reviews

#### Create Review
```bash
POST /repos/{owner}/{repo}/pulls/{number}/reviews
Authorization: Bearer ghp_abc123
Content-Type: application/json

{
  "state": "APPROVED",
  "body": "Looks good to me!"
}
```

Valid states:
- `PENDING` - Review is saved but not submitted
- `APPROVED` - Approve the pull request
- `CHANGES_REQUESTED` - Request changes
- `COMMENTED` - General comment without approval/rejection

#### List Reviews
```bash
GET /repos/{owner}/{repo}/pulls/{number}/reviews
Authorization: Bearer ghp_abc123
```

#### Submit Review
```bash
PUT /repos/{owner}/{repo}/pulls/{number}/reviews/{id}
Authorization: Bearer ghp_abc123
Content-Type: application/json

{}
```

Submits a PENDING review, making it visible and setting the `submitted_at` timestamp.

#### Dismiss Review
```bash
DELETE /repos/{owner}/{repo}/pulls/{number}/reviews/{id}
Authorization: Bearer ghp_abc123
```

Changes review state to DISMISSED and sets `dismissed_at` timestamp.

## Webhooks

Webhooks allow you to receive HTTP POST notifications when specific events occur.

### Creating Webhooks

```bash
POST /repos/{owner}/{repo}/hooks
Authorization: Bearer ghp_abc123
Content-Type: application/json

{
  "config": {
    "url": "https://example.com/webhook",
    "content_type": "json",
    "secret": "my-secret-key"
  },
  "events": ["issues", "pull_request"]
}
```

#### SSRF Protection

The webhook system includes protection against Server-Side Request Forgery (SSRF) attacks. The following URLs are blocked:

- `localhost`, `127.0.0.1`, `::1`
- Private IP ranges: `10.*`, `192.168.*`, `172.16-31.*`
- Link-local addresses: `169.254.*`
- Only `http` and `https` schemes are allowed

### Webhook Events

Supported events:
- `issues` - Issue created, updated, closed
- `pull_request` - PR created, updated, merged
- `issue_comment` - Comment created, updated, deleted
- `pull_request_review` - Review created, submitted, dismissed

### Webhook Payloads

Webhooks are delivered as HTTP POST requests with the following headers:

```
Content-Type: application/json
X-GitHub-Event: issues
X-GitHub-Delivery: {webhook-id}
X-Hub-Signature-256: sha256={hmac-signature}
```

Example payload for an `issues` event:

```json
{
  "action": "opened",
  "issue": {
    "number": 1,
    "title": "Bug found",
    "body": "Description",
    "state": "open",
    "user": {
      "login": "alice",
      "id": 1
    }
  },
  "repository": {
    "id": 1,
    "name": "my-repo",
    "full_name": "alice/my-repo",
    "owner": {
      "login": "alice"
    }
  },
  "sender": {
    "login": "alice",
    "id": 1
  }
}
```

### Webhook Signatures

Webhooks include an HMAC-SHA256 signature in the `X-Hub-Signature-256` header when a secret is configured. This allows you to verify that the webhook came from your ISH instance.

To verify:

```python
import hmac
import hashlib

def verify_signature(payload, signature, secret):
    expected = 'sha256=' + hmac.new(
        secret.encode(),
        payload.encode(),
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(expected, signature)
```

### Managing Webhooks

#### List Webhooks
```bash
GET /repos/{owner}/{repo}/hooks
Authorization: Bearer ghp_abc123
```

#### Get Webhook
```bash
GET /repos/{owner}/{repo}/hooks/{id}
Authorization: Bearer ghp_abc123
```

#### Update Webhook
```bash
PATCH /repos/{owner}/{repo}/hooks/{id}
Authorization: Bearer ghp_abc123
Content-Type: application/json

{
  "config": {
    "url": "https://new-url.com/webhook"
  },
  "events": ["issues"]
}
```

#### Delete Webhook
```bash
DELETE /repos/{owner}/{repo}/hooks/{id}
Authorization: Bearer ghp_abc123
```

#### Test Webhook
```bash
POST /repos/{owner}/{repo}/hooks/{id}/tests
Authorization: Bearer ghp_abc123
```

Triggers a test webhook delivery with a ping event.

## Admin UI

The GitHub plugin provides admin resources for managing data through the ISH admin interface.

Available resources:
- **Users** - View and manage GitHub users
- **Repositories** - View and manage repositories
- **Issues** - View all issues across repositories
- **Pull Requests** - View all pull requests
- **Webhooks** - View and manage webhooks

Access the admin interface at `http://localhost:8000/admin` (or your configured admin port).

### Admin Resource Features

Each resource provides:
- List view with key columns
- Detailed record view
- Timestamps for created/updated dates
- Relationships between resources

## Seeding

The GitHub plugin includes an AI-powered seeding system that generates realistic test data.

### Using the Seed Endpoint

```bash
POST http://localhost:8000/admin/seed
Content-Type: application/json

{
  "plugin": "github",
  "count": 10
}
```

This will generate:
- Multiple users with realistic usernames and profiles
- Repositories with descriptions and metadata
- Issues with titles, bodies, and comments
- Pull requests with reviews
- Webhooks (to allowed test URLs only)

### Seed Data Characteristics

The seeding system generates:
- **Diverse usernames**: Mix of real names, tech terms, and creative handles
- **Realistic repositories**: Various types (libraries, apps, tools) with appropriate descriptions
- **Contextual issues**: Bug reports, feature requests, questions with appropriate labels
- **Pull requests**: With realistic branch names, descriptions, and review states
- **Active discussions**: Multiple comments per issue/PR with threaded conversations

### Manual Seeding

You can also seed data programmatically:

```go
plugin := &GitHubPlugin{store: store}
count, err := plugin.Seed(10)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Created %d records\n", count)
```

## Examples

### Complete Workflow Example

Here's a complete example of creating a repository, opening an issue, commenting, and closing it:

```bash
# Set your token
TOKEN="ghp_abc123xyz"
BASE_URL="http://localhost:9000"

# Create a repository
curl -X POST "$BASE_URL/user/repos" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-project",
    "description": "My awesome project",
    "private": false
  }'

# Create an issue
curl -X POST "$BASE_URL/repos/alice/my-project/issues" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Bug: Button not working",
    "body": "The submit button does not respond to clicks"
  }'

# Add a comment
curl -X POST "$BASE_URL/repos/alice/my-project/issues/1/comments" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "body": "I can reproduce this in Chrome. Works fine in Firefox."
  }'

# Close the issue
curl -X PATCH "$BASE_URL/repos/alice/my-project/issues/1" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "state": "closed",
    "state_reason": "completed"
  }'
```

### Pull Request Workflow

```bash
TOKEN="ghp_abc123xyz"
BASE_URL="http://localhost:9000"

# Create a pull request
curl -X POST "$BASE_URL/repos/alice/my-project/pulls" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Add user authentication",
    "body": "This PR implements OAuth2 authentication",
    "head": "feature/auth",
    "base": "main"
  }'

# Add a review
curl -X POST "$BASE_URL/repos/alice/my-project/pulls/1/reviews" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "state": "APPROVED",
    "body": "Great work! All tests passing."
  }'

# Merge the PR
curl -X PUT "$BASE_URL/repos/alice/my-project/pulls/1/merge" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "commit_message": "Merge pull request #1 from feature/auth"
  }'
```

### Webhook Setup

```bash
TOKEN="ghp_abc123xyz"
BASE_URL="http://localhost:9000"

# Create a webhook
curl -X POST "$BASE_URL/repos/alice/my-project/hooks" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "config": {
      "url": "https://example.com/webhook",
      "content_type": "json",
      "secret": "super-secret-key"
    },
    "events": ["issues", "pull_request"]
  }'

# Test the webhook
curl -X POST "$BASE_URL/repos/alice/my-project/hooks/1/tests" \
  -H "Authorization: Bearer $TOKEN"

# List all webhooks
curl "$BASE_URL/repos/alice/my-project/hooks" \
  -H "Authorization: Bearer $TOKEN"
```

## Database Schema

The plugin uses the following SQLite tables:

- `github_users` - User accounts
- `github_tokens` - Authentication tokens
- `github_repositories` - Repository metadata
- `github_branches` - Branch information
- `github_commits` - Commit history
- `github_issues` - Issues (including PRs)
- `github_pull_requests` - PR-specific data
- `github_comments` - Issue and PR comments
- `github_reviews` - PR reviews
- `github_review_comments` - Review-specific comments
- `github_webhooks` - Webhook configurations
- `github_webhook_deliveries` - Webhook delivery logs

All tables include appropriate indexes for query performance and foreign key constraints for data integrity.

## Testing

The plugin includes comprehensive test coverage:

- **Unit tests** - Individual function and method tests
- **Handler tests** - HTTP endpoint tests with mocked requests
- **Integration tests** - End-to-end workflow tests
- **Webhook tests** - Delivery and signature validation

Run tests:

```bash
cd plugins/github
go test -v
```

Run specific test:

```bash
go test -v -run TestFullIssueLifecycle
```

## Development

### Adding New Endpoints

1. Define the handler in `handlers.go`
2. Add the route in `plugin.go` `RegisterRoutes()`
3. Implement store methods in `store.go`
4. Add tests in appropriate `*_test.go` files
5. Update this README

### Database Migrations

The plugin automatically creates tables on initialization. For schema changes:

1. Update the schema in `store.go`
2. Increment the schema version
3. Add migration logic in `NewGitHubStore()`
4. Test with existing databases

## Compatibility

The plugin aims for compatibility with GitHub REST API v3. Notable differences:

- Simplified OAuth flow (tokens are pre-created)
- No actual Git operations (branches/commits are simulated)
- Webhook delivery is synchronous (no retry queue)
- Limited pagination support
- Simplified search (exact matches only)

## License

Part of the ISH project. See main project LICENSE file.
