# GitHub Plugin Design

**Date**: 2025-12-06
**Status**: Approved
**APIs**: Repositories, Issues, Pull Requests, Comments, Reviews, Webhooks

## Overview

A self-contained GitHub API emulator plugin for ISH that simulates GitHub's REST API v3 with realistic data models and webhook delivery. Mirrors GitHub's actual API as closely as possible while remaining database-driven. Follows ISH's DatabasePlugin interface pattern.

## Architecture

### Plugin Structure
- **Self-contained**: All GitHub logic lives in `plugins/github/`
- **Database-driven**: Uses SQLite tables with `github_` prefix
- **Auto-accept auth**: First API request with any GitHub token auto-creates user
- **Synchronous webhooks**: Fire webhooks immediately when resources change
- **Admin UI integration**: Schema-driven resource management

### Core Components

1. **`plugin.go`**: Interface implementation, route registration, health checks
2. **`store.go`**: Database layer with all tables and queries
3. **`handlers.go`**: HTTP handlers for GitHub API endpoints
4. **`webhooks.go`**: Webhook delivery system with SSRF protection
5. **`schema.go`**: Admin UI resource definitions
6. **`seed.go`**: Test data generation

## API Endpoints

All endpoints implement GitHub REST API v3 format. Routes match real GitHub paths.

### Authentication & Users

```
GET  /user                           # Get authenticated user
GET  /users/{username}               # Get user by username
PATCH /user                          # Update authenticated user
```

### Repositories

```
GET  /user/repos                     # List authenticated user's repos
GET  /users/{username}/repos         # List user's public repos
GET  /orgs/{org}/repos               # List organization repos
GET  /repos/{owner}/{repo}           # Get repository
POST /user/repos                     # Create repo for authenticated user
POST /orgs/{org}/repos               # Create repo in organization
PATCH /repos/{owner}/{repo}          # Update repository
DELETE /repos/{owner}/{repo}         # Delete repository
```

### Issues

```
GET  /repos/{owner}/{repo}/issues           # List issues (excludes PRs)
GET  /repos/{owner}/{repo}/issues/{number}  # Get issue
POST /repos/{owner}/{repo}/issues           # Create issue
PATCH /repos/{owner}/{repo}/issues/{number} # Update issue
```

### Pull Requests

```
GET  /repos/{owner}/{repo}/pulls            # List PRs
GET  /repos/{owner}/{repo}/pulls/{number}   # Get PR
POST /repos/{owner}/{repo}/pulls            # Create PR
PATCH /repos/{owner}/{repo}/pulls/{number}  # Update PR
PUT  /repos/{owner}/{repo}/pulls/{number}/merge  # Merge PR
GET  /repos/{owner}/{repo}/pulls/{number}/files  # Get PR changed files
```

### Comments

```
GET  /repos/{owner}/{repo}/issues/{number}/comments        # List issue comments
POST /repos/{owner}/{repo}/issues/{number}/comments        # Create comment
PATCH /repos/{owner}/{repo}/issues/comments/{comment_id}   # Update comment
DELETE /repos/{owner}/{repo}/issues/comments/{comment_id}  # Delete comment
```

### Reviews

```
GET  /repos/{owner}/{repo}/pulls/{number}/reviews          # List reviews
POST /repos/{owner}/{repo}/pulls/{number}/reviews          # Create review
PUT  /repos/{owner}/{repo}/pulls/{number}/reviews/{id}     # Submit pending review
DELETE /repos/{owner}/{repo}/pulls/{number}/reviews/{id}   # Delete pending review
```

### Review Comments

```
GET  /repos/{owner}/{repo}/pulls/{number}/comments         # List review comments
POST /repos/{owner}/{repo}/pulls/{number}/comments         # Create review comment
PATCH /repos/{owner}/{repo}/pulls/comments/{comment_id}    # Update review comment
DELETE /repos/{owner}/{repo}/pulls/comments/{comment_id}   # Delete review comment
```

### Webhooks

```
GET  /repos/{owner}/{repo}/hooks              # List webhooks
POST /repos/{owner}/{repo}/hooks              # Create webhook
GET  /repos/{owner}/{repo}/hooks/{id}         # Get webhook
PATCH /repos/{owner}/{repo}/hooks/{id}        # Update webhook
DELETE /repos/{owner}/{repo}/hooks/{id}       # Delete webhook
POST /repos/{owner}/{repo}/hooks/{id}/tests   # Test webhook (fire test ping)
```

## Database Schema

### `github_users`
User accounts auto-created from token authentication.

```sql
CREATE TABLE github_users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  login TEXT UNIQUE NOT NULL,           -- Username
  name TEXT,
  email TEXT,
  avatar_url TEXT,
  type TEXT DEFAULT 'User',             -- User/Bot/Organization
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_users_login ON github_users(login);
```

### `github_tokens`
Authentication tokens linked to users.

```sql
CREATE TABLE github_tokens (
  token TEXT PRIMARY KEY,               -- ghp_, gho_, ghs_ prefixes
  user_id INTEGER NOT NULL,
  token_type TEXT DEFAULT 'personal',   -- personal/oauth/app
  scopes TEXT,                          -- JSON array of scopes
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  last_used_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES github_users(id)
);
CREATE INDEX idx_tokens_user ON github_tokens(user_id);
```

### `github_repositories`
Repository metadata (no actual .git directories).

```sql
CREATE TABLE github_repositories (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  owner_id INTEGER NOT NULL,
  name TEXT NOT NULL,
  full_name TEXT NOT NULL,              -- owner/repo
  description TEXT,
  private INTEGER DEFAULT 0,            -- BOOLEAN
  default_branch TEXT DEFAULT 'main',
  fork INTEGER DEFAULT 0,
  archived INTEGER DEFAULT 0,
  disabled INTEGER DEFAULT 0,
  stargazers_count INTEGER DEFAULT 0,
  watchers_count INTEGER DEFAULT 0,
  forks_count INTEGER DEFAULT 0,
  open_issues_count INTEGER DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  pushed_at TIMESTAMP,
  FOREIGN KEY (owner_id) REFERENCES github_users(id),
  UNIQUE(owner_id, name)
);
CREATE INDEX idx_repos_owner ON github_repositories(owner_id);
CREATE INDEX idx_repos_full_name ON github_repositories(full_name);
```

### `github_branches`
Branch refs for each repo.

```sql
CREATE TABLE github_branches (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  repo_id INTEGER NOT NULL,
  name TEXT NOT NULL,
  commit_sha TEXT NOT NULL,
  protected INTEGER DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (repo_id) REFERENCES github_repositories(id) ON DELETE CASCADE,
  UNIQUE(repo_id, name)
);
CREATE INDEX idx_branches_repo ON github_branches(repo_id);
```

### `github_commits`
Simulated commit objects (database-only, no .git).

```sql
CREATE TABLE github_commits (
  sha TEXT PRIMARY KEY,                 -- 40 hex characters
  repo_id INTEGER NOT NULL,
  author_login TEXT,
  author_name TEXT NOT NULL,
  author_email TEXT NOT NULL,
  committer_login TEXT,
  message TEXT NOT NULL,
  parent_sha TEXT,                      -- NULL for initial commit
  tree_sha TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (repo_id) REFERENCES github_repositories(id) ON DELETE CASCADE
);
CREATE INDEX idx_commits_repo ON github_commits(repo_id);
CREATE INDEX idx_commits_parent ON github_commits(parent_sha);
```

### `github_issues`
Issues and Pull Requests (PRs are issues with `is_pull_request=1`).

```sql
CREATE TABLE github_issues (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  repo_id INTEGER NOT NULL,
  number INTEGER NOT NULL,              -- Repo-scoped issue number
  title TEXT NOT NULL,
  body TEXT,
  state TEXT DEFAULT 'open',            -- open/closed
  state_reason TEXT,                    -- completed/not_planned/reopened
  user_id INTEGER NOT NULL,             -- Creator
  assignee_ids TEXT,                    -- JSON array of user IDs
  label_ids TEXT,                       -- JSON array of label IDs
  milestone_id INTEGER,
  locked INTEGER DEFAULT 0,
  comments_count INTEGER DEFAULT 0,
  is_pull_request INTEGER DEFAULT 0,   -- BOOLEAN
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  closed_at TIMESTAMP,
  FOREIGN KEY (repo_id) REFERENCES github_repositories(id) ON DELETE CASCADE,
  FOREIGN KEY (user_id) REFERENCES github_users(id),
  UNIQUE(repo_id, number)
);
CREATE INDEX idx_issues_repo ON github_issues(repo_id);
CREATE INDEX idx_issues_state ON github_issues(state);
CREATE INDEX idx_issues_number ON github_issues(repo_id, number);
```

### `github_pull_requests`
PR-specific metadata (extends github_issues).

```sql
CREATE TABLE github_pull_requests (
  issue_id INTEGER PRIMARY KEY,
  head_repo_id INTEGER NOT NULL,
  head_ref TEXT NOT NULL,               -- Source branch
  base_repo_id INTEGER NOT NULL,
  base_ref TEXT NOT NULL,               -- Target branch
  merged INTEGER DEFAULT 0,
  mergeable INTEGER DEFAULT 1,
  rebaseable INTEGER DEFAULT 1,
  merge_commit_sha TEXT,
  merged_at TIMESTAMP,
  merged_by_id INTEGER,
  draft INTEGER DEFAULT 0,
  review_comments_count INTEGER DEFAULT 0,
  commits_count INTEGER DEFAULT 1,
  additions INTEGER DEFAULT 0,
  deletions INTEGER DEFAULT 0,
  changed_files INTEGER DEFAULT 0,
  FOREIGN KEY (issue_id) REFERENCES github_issues(id) ON DELETE CASCADE,
  FOREIGN KEY (head_repo_id) REFERENCES github_repositories(id),
  FOREIGN KEY (base_repo_id) REFERENCES github_repositories(id),
  FOREIGN KEY (merged_by_id) REFERENCES github_users(id)
);
CREATE INDEX idx_prs_head_repo ON github_pull_requests(head_repo_id);
CREATE INDEX idx_prs_base_repo ON github_pull_requests(base_repo_id);
```

### `github_comments`
Comments on issues and pull requests.

```sql
CREATE TABLE github_comments (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  issue_id INTEGER NOT NULL,
  user_id INTEGER NOT NULL,
  body TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (issue_id) REFERENCES github_issues(id) ON DELETE CASCADE,
  FOREIGN KEY (user_id) REFERENCES github_users(id)
);
CREATE INDEX idx_comments_issue ON github_comments(issue_id);
CREATE INDEX idx_comments_created ON github_comments(created_at);
```

### `github_reviews`
Pull request reviews.

```sql
CREATE TABLE github_reviews (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  pull_request_id INTEGER NOT NULL,     -- FK to github_issues
  user_id INTEGER NOT NULL,
  state TEXT NOT NULL,                  -- PENDING/COMMENTED/APPROVED/CHANGES_REQUESTED/DISMISSED
  body TEXT,
  commit_sha TEXT,
  submitted_at TIMESTAMP,
  dismissed_at TIMESTAMP,
  FOREIGN KEY (pull_request_id) REFERENCES github_issues(id) ON DELETE CASCADE,
  FOREIGN KEY (user_id) REFERENCES github_users(id)
);
CREATE INDEX idx_reviews_pr ON github_reviews(pull_request_id);
CREATE INDEX idx_reviews_state ON github_reviews(state);
```

### `github_review_comments`
Inline code review comments on PRs.

```sql
CREATE TABLE github_review_comments (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  pull_request_id INTEGER NOT NULL,
  review_id INTEGER,
  user_id INTEGER NOT NULL,
  body TEXT NOT NULL,
  path TEXT NOT NULL,                   -- File path
  position INTEGER,                     -- Line number
  commit_sha TEXT NOT NULL,
  in_reply_to_id INTEGER,               -- Thread replies
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (pull_request_id) REFERENCES github_issues(id) ON DELETE CASCADE,
  FOREIGN KEY (review_id) REFERENCES github_reviews(id) ON DELETE CASCADE,
  FOREIGN KEY (user_id) REFERENCES github_users(id),
  FOREIGN KEY (in_reply_to_id) REFERENCES github_review_comments(id)
);
CREATE INDEX idx_review_comments_pr ON github_review_comments(pull_request_id);
CREATE INDEX idx_review_comments_review ON github_review_comments(review_id);
```

### `github_webhooks`
Webhook configurations per repository.

```sql
CREATE TABLE github_webhooks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  repo_id INTEGER NOT NULL,
  url TEXT NOT NULL,
  content_type TEXT DEFAULT 'json',    -- json/form
  secret TEXT,                          -- For HMAC signatures
  events TEXT NOT NULL,                 -- JSON array: ["push", "issues", "pull_request"]
  active INTEGER DEFAULT 1,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (repo_id) REFERENCES github_repositories(id) ON DELETE CASCADE
);
CREATE INDEX idx_webhooks_repo ON github_webhooks(repo_id);
CREATE INDEX idx_webhooks_active ON github_webhooks(active);
```

### `github_webhook_deliveries`
Delivery history for debugging webhooks.

```sql
CREATE TABLE github_webhook_deliveries (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  webhook_id INTEGER NOT NULL,
  event_type TEXT NOT NULL,             -- issues, pull_request, etc.
  payload TEXT NOT NULL,                -- Full JSON payload
  delivered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  status_code INTEGER,
  error_message TEXT,
  FOREIGN KEY (webhook_id) REFERENCES github_webhooks(id) ON DELETE CASCADE
);
CREATE INDEX idx_deliveries_webhook ON github_webhook_deliveries(webhook_id);
CREATE INDEX idx_deliveries_delivered ON github_webhook_deliveries(delivered_at DESC);
```

## Authentication

**GitHub-style tokens** with auto-accept pattern:

- Token formats: `ghp_` (personal), `gho_` (OAuth), `ghs_` (app/server)
- Accept both `Authorization: Bearer <token>` and `Authorization: token <token>` headers
- First request with any token auto-creates user account
- Token is stored in `github_tokens` table
- Username extracted from token or generated (`user_<random>`)
- Subsequent requests validate token exists in database

Example:
```bash
curl -H "Authorization: Bearer ghp_1234567890abcdef" \
  http://localhost:9000/user
```

Response includes user object with login, id, and profile data.

## Webhook Implementation

### Event Triggering

Webhooks fire **synchronously** when resources change. When an API handler modifies data, it:

1. Commits the database transaction
2. Queries all active webhooks for the repo where `events` array contains the event type
3. Constructs the GitHub-formatted JSON payload
4. Sends HTTP POST to each webhook URL
5. Logs delivery to `github_webhook_deliveries`

### Event Types

Common events that trigger webhooks:

- **`issues`** - Issue opened, edited, closed, reopened, assigned, labeled
- **`issue_comment`** - Comment created, edited, deleted on issues
- **`pull_request`** - PR opened, edited, closed, reopened, merged
- **`pull_request_review`** - Review submitted, edited, dismissed
- **`pull_request_review_comment`** - Review comment created, edited, deleted

### Payload Format

Webhooks POST JSON matching GitHub's exact schema:

```json
{
  "action": "opened",
  "issue": {
    "id": 1,
    "number": 42,
    "title": "Bug in authentication",
    "state": "open",
    "user": { "login": "alice", "id": 1 },
    "body": "Detailed description...",
    "created_at": "2025-12-06T10:00:00Z",
    "updated_at": "2025-12-06T10:00:00Z"
  },
  "repository": {
    "id": 5,
    "name": "backend",
    "full_name": "alice/backend",
    "owner": { "login": "alice", "id": 1 },
    "private": false
  },
  "sender": {
    "login": "alice",
    "id": 1,
    "type": "User"
  }
}
```

### Delivery Headers

Each webhook includes GitHub's standard headers:

- `X-GitHub-Event: issues` (event type)
- `X-GitHub-Delivery: <uuid>` (unique delivery ID)
- `X-Hub-Signature-256: sha256=<hmac>` (HMAC of payload with webhook secret)
- `Content-Type: application/json`
- `User-Agent: GitHub-Hookshot/ISH`

### Security (SSRF Protection)

URL validation prevents Server-Side Request Forgery attacks:

- **Block localhost**: `localhost`, `127.0.0.1`, `::1`
- **Block private IPs**: `10.0.0.0/8`, `192.168.0.0/16`, `172.16.0.0/12`
- **Block link-local**: `169.254.0.0/16`, `fe80::/10`
- **Require valid schemes**: Only `http://` or `https://`
- **Parse validation**: URL must parse correctly
- **Applied at creation**: Webhook creation fails if URL is invalid

### Core Functions

**`FireWebhook(repoID int, eventType string, payload interface{})`**
- Queries all active webhooks for repo where events array contains eventType
- Constructs JSON payload with action, resource, repository, sender
- Generates delivery UUID and HMAC signature
- Sends HTTP POST with GitHub headers
- Logs delivery to github_webhook_deliveries

**`ValidateWebhookURL(url string) error`**
- Parses URL and extracts hostname
- Checks against private IP ranges and localhost
- Returns error if URL is unsafe

**`GenerateHMAC(payload []byte, secret string) string`**
- Computes HMAC-SHA256 of payload using webhook secret
- Returns `sha256=<hex>` format for X-Hub-Signature-256 header

## Admin UI

Schema-driven resources auto-generated at `/admin/github/`:

### Users (`/admin/github/users`)
- **List view**: login, name, type, created_at
- **Create**: Generate user with custom login, auto-generate token
- **Detail**: Show user's repos, recent activity
- **Actions**: "Generate New Token", "View Tokens"

### Repositories (`/admin/github/repositories`)
- **List view**: full_name, owner, private, stars, forks, open_issues
- **Create**: Create repo for user/org with name, description, private flag
- **Edit**: Update description, default_branch, archived status
- **Delete**: Soft delete (cascades to issues, PRs, webhooks)
- **Actions**: "View Issues", "View PRs", "View Webhooks"

### Issues (`/admin/github/issues`)
- **List view**: repo, number, title, state, assignees, labels, created_at
- **Create**: Create issue with title, body, assignees, labels
- **Edit**: Update title, body, state, state_reason
- **Detail**: Show comments, timeline, linked PRs
- **Filters**: By repo, state, assignee, label

### Pull Requests (`/admin/github/pull_requests`)
- **List view**: repo, number, title, state, mergeable, merged, created_at
- **Create**: Create PR with head/base branches, title, body
- **Edit**: Update title, body, state
- **Detail**: Show files changed, commits, reviews, comments
- **Actions**: "Merge PR", "Close PR", "Reopen PR"
- **Filters**: By repo, state, merged status

### Webhooks (`/admin/github/webhooks`)
- **List view**: repo, url, events, active, created_at
- **Create**: Add webhook with URL, events, secret, content_type
- **Edit**: Update URL, events, active status
- **Delete**: Remove webhook
- **Actions**: "Test Webhook" (fire ping event), "View Deliveries"

### Webhook Deliveries (`/admin/github/webhook_deliveries`)
- **List view**: webhook, event_type, status_code, delivered_at
- **Detail**: Full payload, headers, response
- **Read-only**: No create/edit/delete
- **Filters**: By webhook, event_type, status_code, date range

## Seeding Strategy

`Seed(size string)` creates realistic GitHub project scenarios:

### Small
- 2 users
- 3 repositories
- 5 issues (3 open, 2 closed)
- 2 pull requests (1 open, 1 merged)
- 10 comments
- 3 reviews
- 5 commits per repo

### Medium
- 5 users
- 10 repositories
- 30 issues (20 open, 10 closed)
- 15 pull requests (8 open, 5 merged, 2 closed)
- 50 comments
- 20 reviews
- 5 webhooks
- 15 commits per repo

### Large
- 20 users
- 30 repositories
- 100 issues
- 50 pull requests
- 200 comments
- 80 reviews
- 15 webhooks
- 30 commits per repo

### Data Characteristics

**Repositories**:
- Common naming: `frontend`, `backend`, `api`, `mobile-app`, `docs`, `cli`
- Mix of public/private
- Realistic star/fork counts
- Default branch: `main` or `master`

**Issues**:
- Bug reports: "Fix authentication timeout", "Memory leak in worker process"
- Features: "Add dark mode support", "Implement rate limiting"
- Questions: "How to configure CORS?", "Best practices for caching"
- Mixed states: open, closed (completed), closed (not_planned)

**Pull Requests**:
- Descriptive branch names: `fix/auth-bug`, `feat/dark-mode`, `refactor/api-cleanup`
- Reference issues in body: "Fixes #42"
- Mix of states: open, merged, closed without merge
- Realistic changed file counts, additions/deletions

**Reviews**:
- State distribution: 50% APPROVED, 30% CHANGES_REQUESTED, 20% COMMENTED
- Realistic review bodies: "LGTM!", "Looks good but please add tests", "Consider refactoring this method"

**Comments**:
- Conversation threads with timestamps showing progression
- Technical discussions: "We could use Redis for this", "What about performance?"
- Resolution confirmations: "Fixed in latest commit", "Thanks, merged!"

**Commits**:
- Realistic SHAs (40 hex characters)
- Conventional commit messages: "feat: add user authentication", "fix: resolve memory leak"
- Author attribution with name and email
- Parent commit linkage for history

### AI Seeding (with `--ai` flag)

Uses Claude API to generate coherent project narratives:

- **Project theme**: "Building a task management SaaS"
- **Related issues/PRs**: Issues reference each other, PRs close specific issues
- **Natural conversations**: Comment threads show realistic technical discussions
- **Code review feedback**: Reviews include specific technical suggestions
- **Commit history**: Logical progression of features and bug fixes

Example AI-generated scenario:
1. Issue #1: "Add user authentication system"
2. PR #1: "feat: implement JWT authentication" (closes #1)
3. Review on PR #1: "Looks great but we should add rate limiting to prevent brute force"
4. Issue #2: "Add rate limiting to login endpoint"
5. PR #2: "feat: add rate limiting with Redis" (closes #2)
6. Comments showing discussion about Redis vs in-memory cache

## Testing Strategy

### Unit Tests
- Token generation and validation
- Webhook payload formatting
- HMAC signature generation
- URL validation (SSRF protection)
- Issue number incrementing per repo

### Integration Tests
- Full issue lifecycle: create → comment → close
- PR workflow: create → review → merge
- Webhook delivery: create issue → webhook fires → delivery logged
- Pagination with Link headers
- Authentication with multiple token types

### End-to-End Tests
- Create repo via API, verify in database
- Create issue, verify webhook fired
- Merge PR, verify issue auto-closed (if referenced)
- List resources with pagination
- Admin UI CRUD operations

## Future Enhancements

Not in initial scope, but possible additions:

- **Labels & Milestones**: Tag issues/PRs, group by milestone
- **Projects**: Classic project boards with columns and cards
- **Actions**: Workflow runs, artifacts, job logs
- **Git operations**: Actual file contents, tree objects, diffs
- **Search**: Full-text search across issues, PRs, code
- **Notifications**: User notification feed for mentions, assignments
- **Team permissions**: Org membership, team-based access control
- **Status checks**: Commit status API for CI/CD integrations
- **Releases**: Release tags, assets, changelogs
- **Gists**: Standalone code snippets with comments
