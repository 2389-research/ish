# GitHub Plugin Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a GitHub REST API v3 emulator plugin that simulates repositories, issues, pull requests, reviews, comments, and webhooks with database-only storage.

**Architecture:** DatabasePlugin following ISH's established pattern (Discord/Twilio). Auto-accept authentication with GitHub tokens (ghp_*, gho_*, ghs_*). Synchronous webhook delivery with SSRF protection. All data in SQLite with github_ prefix.

**Tech Stack:** Go 1.21+, SQLite3, chi router, crypto/hmac for webhook signatures

---

## Task 1: Plugin Structure & Store Foundation

**Files:**
- Create: `plugins/github/plugin.go`
- Create: `plugins/github/store.go`
- Create: `plugins/github/store_test.go`

### Step 1: Write the failing test

Create `plugins/github/store_test.go`:

```go
// ABOUTME: Unit tests for GitHub plugin store layer
// ABOUTME: Tests user, token, and repository CRUD operations

package github

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	return db
}

func TestNewGitHubStore(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	store, err := NewGitHubStore(db)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	if store == nil {
		t.Fatal("Store is nil")
	}

	// Verify tables were created
	tables := []string{
		"github_users",
		"github_tokens",
		"github_repositories",
		"github_branches",
		"github_commits",
		"github_issues",
		"github_pull_requests",
		"github_comments",
		"github_reviews",
		"github_review_comments",
		"github_webhooks",
		"github_webhook_deliveries",
	}

	for _, table := range tables {
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
		if err != nil || count != 1 {
			t.Fatalf("Table %s was not created", table)
		}
	}
}

func TestGetOrCreateUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)

	// First call should create
	user, err := store.GetOrCreateUser("alice", "ghp_testtoken123")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	if user.Login != "alice" {
		t.Fatalf("Expected login 'alice', got '%s'", user.Login)
	}
	if user.ID == 0 {
		t.Fatal("User ID should be generated")
	}

	// Second call should retrieve
	user2, err := store.GetOrCreateUser("alice", "ghp_testtoken123")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	if user2.ID != user.ID {
		t.Fatal("Should retrieve same user")
	}
}

func TestValidateToken(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)

	// Create user with token
	user, _ := store.GetOrCreateUser("alice", "ghp_validtoken")

	// Valid token should return user
	validUser, err := store.ValidateToken("ghp_validtoken")
	if err != nil {
		t.Fatalf("Token validation failed: %v", err)
	}
	if validUser.ID != user.ID {
		t.Fatal("ValidateToken returned wrong user")
	}

	// Invalid token should fail
	_, err = store.ValidateToken("ghp_invalidtoken")
	if err != sql.ErrNoRows {
		t.Fatal("Expected sql.ErrNoRows for invalid token")
	}
}
```

### Step 2: Run test to verify it fails

```bash
cd plugins/github
go test -v -run TestNewGitHubStore
```

Expected: FAIL - package github is not in GOROOT

### Step 3: Create plugin.go skeleton

Create `plugins/github/plugin.go`:

```go
// ABOUTME: GitHub REST API v3 emulator plugin for ISH
// ABOUTME: Simulates repos, issues, PRs, reviews, comments, webhooks

package github

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/2389/ish/plugins/core"
	"github.com/go-chi/chi/v5"
)

func init() {
	core.Register(&GitHubPlugin{})
}

type GitHubPlugin struct {
	store *GitHubStore
}

func (p *GitHubPlugin) Name() string {
	return "github"
}

func (p *GitHubPlugin) Health() core.HealthStatus {
	return core.HealthStatus{
		Status:  "healthy",
		Message: "GitHub plugin operational",
	}
}

func (p *GitHubPlugin) RegisterRoutes(r chi.Router) {
	// Will implement in later tasks
}

func (p *GitHubPlugin) RegisterAuth(r chi.Router) {
	// GitHub doesn't use OAuth through ISH
}

func (p *GitHubPlugin) Schema() core.PluginSchema {
	// Will implement in later tasks
	return core.PluginSchema{}
}

func (p *GitHubPlugin) Seed(ctx context.Context, size string) (core.SeedData, error) {
	// Will implement in later tasks
	return core.SeedData{
		Summary: "GitHub plugin seeding not implemented",
		Records: map[string]int{},
	}, nil
}

func (p *GitHubPlugin) ValidateToken(token string) bool {
	user, err := p.store.ValidateToken(token)
	return err == nil && user != nil
}

func (p *GitHubPlugin) SetDB(db *sql.DB) error {
	store, err := NewGitHubStore(db)
	if err != nil {
		return err
	}
	p.store = store
	return nil
}
```

### Step 4: Create store.go with schema

Create `plugins/github/store.go`:

```go
// ABOUTME: Database layer for GitHub plugin
// ABOUTME: Manages all github_* tables and CRUD operations

package github

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

type GitHubStore struct {
	db *sql.DB
}

type User struct {
	ID        int64
	Login     string
	Name      string
	Email     string
	AvatarURL string
	Type      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Token struct {
	Token     string
	UserID    int64
	TokenType string
	Scopes    string
	CreatedAt time.Time
	LastUsedAt time.Time
}

type Repository struct {
	ID              int64
	OwnerID         int64
	Name            string
	FullName        string
	Description     string
	Private         bool
	DefaultBranch   string
	Fork            bool
	Archived        bool
	Disabled        bool
	StargazersCount int
	WatchersCount   int
	ForksCount      int
	OpenIssuesCount int
	CreatedAt       time.Time
	UpdatedAt       time.Time
	PushedAt        *time.Time
}

func NewGitHubStore(db *sql.DB) (*GitHubStore, error) {
	store := &GitHubStore{db: db}
	if err := store.initTables(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *GitHubStore) initTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS github_users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			login TEXT UNIQUE NOT NULL,
			name TEXT,
			email TEXT,
			avatar_url TEXT,
			type TEXT DEFAULT 'User',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_users_login ON github_users(login)`,

		`CREATE TABLE IF NOT EXISTS github_tokens (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			token_type TEXT DEFAULT 'personal',
			scopes TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_used_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES github_users(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tokens_user ON github_tokens(user_id)`,

		`CREATE TABLE IF NOT EXISTS github_repositories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			owner_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			full_name TEXT NOT NULL,
			description TEXT,
			private INTEGER DEFAULT 0,
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
		)`,
		`CREATE INDEX IF NOT EXISTS idx_repos_owner ON github_repositories(owner_id)`,
		`CREATE INDEX IF NOT EXISTS idx_repos_full_name ON github_repositories(full_name)`,

		`CREATE TABLE IF NOT EXISTS github_branches (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repo_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			commit_sha TEXT NOT NULL,
			protected INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (repo_id) REFERENCES github_repositories(id) ON DELETE CASCADE,
			UNIQUE(repo_id, name)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_branches_repo ON github_branches(repo_id)`,

		`CREATE TABLE IF NOT EXISTS github_commits (
			sha TEXT PRIMARY KEY,
			repo_id INTEGER NOT NULL,
			author_login TEXT,
			author_name TEXT NOT NULL,
			author_email TEXT NOT NULL,
			committer_login TEXT,
			message TEXT NOT NULL,
			parent_sha TEXT,
			tree_sha TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (repo_id) REFERENCES github_repositories(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_commits_repo ON github_commits(repo_id)`,
		`CREATE INDEX IF NOT EXISTS idx_commits_parent ON github_commits(parent_sha)`,

		`CREATE TABLE IF NOT EXISTS github_issues (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repo_id INTEGER NOT NULL,
			number INTEGER NOT NULL,
			title TEXT NOT NULL,
			body TEXT,
			state TEXT DEFAULT 'open',
			state_reason TEXT,
			user_id INTEGER NOT NULL,
			assignee_ids TEXT,
			label_ids TEXT,
			milestone_id INTEGER,
			locked INTEGER DEFAULT 0,
			comments_count INTEGER DEFAULT 0,
			is_pull_request INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			closed_at TIMESTAMP,
			FOREIGN KEY (repo_id) REFERENCES github_repositories(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES github_users(id),
			UNIQUE(repo_id, number)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_issues_repo ON github_issues(repo_id)`,
		`CREATE INDEX IF NOT EXISTS idx_issues_state ON github_issues(state)`,
		`CREATE INDEX IF NOT EXISTS idx_issues_number ON github_issues(repo_id, number)`,

		`CREATE TABLE IF NOT EXISTS github_pull_requests (
			issue_id INTEGER PRIMARY KEY,
			head_repo_id INTEGER NOT NULL,
			head_ref TEXT NOT NULL,
			base_repo_id INTEGER NOT NULL,
			base_ref TEXT NOT NULL,
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
		)`,
		`CREATE INDEX IF NOT EXISTS idx_prs_head_repo ON github_pull_requests(head_repo_id)`,
		`CREATE INDEX IF NOT EXISTS idx_prs_base_repo ON github_pull_requests(base_repo_id)`,

		`CREATE TABLE IF NOT EXISTS github_comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			issue_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			body TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (issue_id) REFERENCES github_issues(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES github_users(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_comments_issue ON github_comments(issue_id)`,
		`CREATE INDEX IF NOT EXISTS idx_comments_created ON github_comments(created_at)`,

		`CREATE TABLE IF NOT EXISTS github_reviews (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pull_request_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			state TEXT NOT NULL,
			body TEXT,
			commit_sha TEXT,
			submitted_at TIMESTAMP,
			dismissed_at TIMESTAMP,
			FOREIGN KEY (pull_request_id) REFERENCES github_issues(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES github_users(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_reviews_pr ON github_reviews(pull_request_id)`,
		`CREATE INDEX IF NOT EXISTS idx_reviews_state ON github_reviews(state)`,

		`CREATE TABLE IF NOT EXISTS github_review_comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			pull_request_id INTEGER NOT NULL,
			review_id INTEGER,
			user_id INTEGER NOT NULL,
			body TEXT NOT NULL,
			path TEXT NOT NULL,
			position INTEGER,
			commit_sha TEXT NOT NULL,
			in_reply_to_id INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (pull_request_id) REFERENCES github_issues(id) ON DELETE CASCADE,
			FOREIGN KEY (review_id) REFERENCES github_reviews(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES github_users(id),
			FOREIGN KEY (in_reply_to_id) REFERENCES github_review_comments(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_review_comments_pr ON github_review_comments(pull_request_id)`,
		`CREATE INDEX IF NOT EXISTS idx_review_comments_review ON github_review_comments(review_id)`,

		`CREATE TABLE IF NOT EXISTS github_webhooks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			repo_id INTEGER NOT NULL,
			url TEXT NOT NULL,
			content_type TEXT DEFAULT 'json',
			secret TEXT,
			events TEXT NOT NULL,
			active INTEGER DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (repo_id) REFERENCES github_repositories(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_webhooks_repo ON github_webhooks(repo_id)`,
		`CREATE INDEX IF NOT EXISTS idx_webhooks_active ON github_webhooks(active)`,

		`CREATE TABLE IF NOT EXISTS github_webhook_deliveries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			webhook_id INTEGER NOT NULL,
			event_type TEXT NOT NULL,
			payload TEXT NOT NULL,
			delivered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			status_code INTEGER,
			error_message TEXT,
			FOREIGN KEY (webhook_id) REFERENCES github_webhooks(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_deliveries_webhook ON github_webhook_deliveries(webhook_id)`,
		`CREATE INDEX IF NOT EXISTS idx_deliveries_delivered ON github_webhook_deliveries(delivered_at DESC)`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("failed to create tables: %w", err)
		}
	}
	return nil
}

// GetOrCreateUser retrieves or creates a user (auto-accept pattern)
func (s *GitHubStore) GetOrCreateUser(login, token string) (*User, error) {
	// Try to get existing user
	var user User
	err := s.db.QueryRow(`
		SELECT id, login, name, email, avatar_url, type, created_at, updated_at
		FROM github_users WHERE login = ?
	`, login).Scan(&user.ID, &user.Login, &user.Name, &user.Email, &user.AvatarURL, &user.Type, &user.CreatedAt, &user.UpdatedAt)

	if err == nil {
		// User exists, create token if not exists
		_, err = s.db.Exec(`
			INSERT OR IGNORE INTO github_tokens (token, user_id, token_type, created_at, last_used_at)
			VALUES (?, ?, 'personal', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, token, user.ID)
		if err != nil {
			return nil, err
		}
		return &user, nil
	}

	if err != sql.ErrNoRows {
		return nil, err
	}

	// Create new user
	result, err := s.db.Exec(`
		INSERT INTO github_users (login, type, created_at, updated_at)
		VALUES (?, 'User', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, login)
	if err != nil {
		return nil, err
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	// Create token
	_, err = s.db.Exec(`
		INSERT INTO github_tokens (token, user_id, token_type, created_at, last_used_at)
		VALUES (?, ?, 'personal', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, token, userID)
	if err != nil {
		return nil, err
	}

	user = User{
		ID:        userID,
		Login:     login,
		Type:      "User",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return &user, nil
}

// ValidateToken checks if token exists and returns associated user
func (s *GitHubStore) ValidateToken(token string) (*User, error) {
	var user User
	err := s.db.QueryRow(`
		SELECT u.id, u.login, u.name, u.email, u.avatar_url, u.type, u.created_at, u.updated_at
		FROM github_users u
		JOIN github_tokens t ON u.id = t.user_id
		WHERE t.token = ?
	`, token).Scan(&user.ID, &user.Login, &user.Name, &user.Email, &user.AvatarURL, &user.Type, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, err
	}

	// Update last_used_at
	s.db.Exec(`UPDATE github_tokens SET last_used_at = CURRENT_TIMESTAMP WHERE token = ?`, token)

	return &user, nil
}

// generateToken creates a GitHub-style token with prefix
func generateToken(prefix string) (string, error) {
	bytes := make([]byte, 20)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return prefix + "_" + hex.EncodeToString(bytes), nil
}
```

### Step 5: Run tests to verify they pass

```bash
cd plugins/github
go test -v
```

Expected: PASS (all 3 tests)

### Step 6: Commit

```bash
git add plugins/github/
git commit -m "feat(github): add plugin structure and store foundation

- Implement GitHubPlugin with DatabasePlugin interface
- Create store with 12 tables (users, tokens, repos, issues, PRs, reviews, webhooks)
- Add auto-accept user creation pattern
- Add token validation with last_used_at tracking
- Tests for NewGitHubStore, GetOrCreateUser, ValidateToken

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 2: Authentication Middleware

**Files:**
- Modify: `plugins/github/plugin.go`
- Create: `plugins/github/auth_test.go`

### Step 1: Write the failing test

Create `plugins/github/auth_test.go`:

```go
// ABOUTME: Tests for GitHub authentication middleware
// ABOUTME: Tests token extraction and validation

package github

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractToken(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
		hasToken bool
	}{
		{"Bearer format", "Bearer ghp_test123", "ghp_test123", true},
		{"Token format", "token ghp_test123", "ghp_test123", true},
		{"No auth header", "", "", false},
		{"Invalid format", "Basic abc123", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			token, ok := extractToken(req)
			if ok != tt.hasToken {
				t.Fatalf("Expected hasToken=%v, got %v", tt.hasToken, ok)
			}
			if token != tt.expected {
				t.Fatalf("Expected token '%s', got '%s'", tt.expected, token)
			}
		})
	}
}

func TestRequireAuth(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)

	plugin := &GitHubPlugin{store: store}

	// Create user with token
	store.GetOrCreateUser("alice", "ghp_valid")

	handler := plugin.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*User)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(user.Login))
	})

	t.Run("Valid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer ghp_valid")
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d", w.Code)
		}
		if w.Body.String() != "alice" {
			t.Fatalf("Expected 'alice', got '%s'", w.Body.String())
		}
	})

	t.Run("Missing token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("Expected 401, got %d", w.Code)
		}
	})

	t.Run("Invalid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer ghp_invalid")
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("Expected 401, got %d", w.Code)
		}
	})
}
```

### Step 2: Run test to verify it fails

```bash
cd plugins/github
go test -v -run TestExtractToken
```

Expected: FAIL - undefined: extractToken

### Step 3: Implement authentication middleware

Modify `plugins/github/plugin.go`, add after the imports:

```go
type contextKey string

const userContextKey contextKey = "github_user"

// extractToken extracts GitHub token from Authorization header
// Supports both "Bearer <token>" and "token <token>" formats
func extractToken(r *http.Request) (string, bool) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", false
	}

	// Try "Bearer <token>" format
	if len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:], true
	}

	// Try "token <token>" format (GitHub's alternative)
	if len(auth) > 6 && auth[:6] == "token " {
		return auth[6:], true
	}

	return "", false
}

// requireAuth middleware validates GitHub token and adds user to context
func (p *GitHubPlugin) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := extractToken(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "requires authentication")
			return
		}

		user, err := p.store.ValidateToken(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "bad credentials")
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// writeError writes a GitHub-style JSON error response
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"message": "%s"}`, message)
}
```

Add import at the top:
```go
import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/2389/ish/plugins/core"
	"github.com/go-chi/chi/v5"
)
```

### Step 4: Run tests to verify they pass

```bash
cd plugins/github
go test -v
```

Expected: PASS (all tests including new auth tests)

### Step 5: Commit

```bash
git add plugins/github/
git commit -m "feat(github): add authentication middleware

- Implement extractToken supporting Bearer and token formats
- Add requireAuth middleware with user context
- Add writeError helper for GitHub-style JSON errors
- Tests for token extraction and auth middleware

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 3: Repository API Handlers

**Files:**
- Create: `plugins/github/handlers.go`
- Create: `plugins/github/handlers_test.go`
- Modify: `plugins/github/plugin.go`
- Modify: `plugins/github/store.go`

### Step 1: Write the failing test

Create `plugins/github/handlers_test.go`:

```go
// ABOUTME: Tests for GitHub API handlers
// ABOUTME: Tests repository, user, and resource CRUD endpoints

package github

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestGetAuthenticatedUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Create user
	store.GetOrCreateUser("alice", "ghp_test")

	req := httptest.NewRequest("GET", "/user", nil)
	req.Header.Set("Authorization", "Bearer ghp_test")
	w := httptest.NewRecorder()

	handler := plugin.requireAuth(plugin.getAuthenticatedUser)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["login"] != "alice" {
		t.Fatalf("Expected login 'alice', got %v", resp["login"])
	}
	if resp["type"] != "User" {
		t.Fatalf("Expected type 'User', got %v", resp["type"])
	}
}

func TestCreateRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Create user
	user, _ := store.GetOrCreateUser("alice", "ghp_test")

	body := `{"name": "test-repo", "description": "Test repository", "private": false}`
	req := httptest.NewRequest("POST", "/user/repos", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer ghp_test")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := plugin.requireAuth(plugin.createUserRepository)
	handler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["name"] != "test-repo" {
		t.Fatalf("Expected name 'test-repo', got %v", resp["name"])
	}
	if resp["full_name"] != "alice/test-repo" {
		t.Fatalf("Expected full_name 'alice/test-repo', got %v", resp["full_name"])
	}
	if resp["description"] != "Test repository" {
		t.Fatalf("Expected description, got %v", resp["description"])
	}

	// Verify in database
	repo, err := store.GetRepository(user.ID, "test-repo")
	if err != nil {
		t.Fatalf("Repo not found in database: %v", err)
	}
	if repo.Name != "test-repo" {
		t.Fatalf("Database name mismatch")
	}
}

func TestListUserRepositories(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Create user and repos
	user, _ := store.GetOrCreateUser("alice", "ghp_test")
	store.CreateRepository(user.ID, "repo1", "", false)
	store.CreateRepository(user.ID, "repo2", "", false)

	req := httptest.NewRequest("GET", "/user/repos", nil)
	req.Header.Set("Authorization", "Bearer ghp_test")
	w := httptest.NewRecorder()

	handler := plugin.requireAuth(plugin.listAuthenticatedUserRepositories)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var repos []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &repos)

	if len(repos) != 2 {
		t.Fatalf("Expected 2 repos, got %d", len(repos))
	}
}

func TestGetRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Create user and repo
	user, _ := store.GetOrCreateUser("alice", "ghp_test")
	store.CreateRepository(user.ID, "my-repo", "Test repo", false)

	req := httptest.NewRequest("GET", "/repos/alice/my-repo", nil)
	req.Header.Set("Authorization", "Bearer ghp_test")
	w := httptest.NewRecorder()

	// Setup chi context for path params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("owner", "alice")
	rctx.URLParams.Add("repo", "my-repo")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := plugin.requireAuth(plugin.getRepository)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["name"] != "my-repo" {
		t.Fatalf("Expected name 'my-repo', got %v", resp["name"])
	}
	if resp["full_name"] != "alice/my-repo" {
		t.Fatalf("Expected full_name 'alice/my-repo', got %v", resp["full_name"])
	}
}
```

### Step 2: Run test to verify it fails

```bash
cd plugins/github
go test -v -run TestGetAuthenticatedUser
```

Expected: FAIL - undefined: getAuthenticatedUser

### Step 3: Add repository methods to store

Modify `plugins/github/store.go`, add these methods at the end:

```go
// CreateRepository creates a new repository
func (s *GitHubStore) CreateRepository(ownerID int64, name, description string, private bool) (*Repository, error) {
	fullName := fmt.Sprintf("%d/%s", ownerID, name) // Will update with actual owner login later

	// Get owner login
	var ownerLogin string
	err := s.db.QueryRow(`SELECT login FROM github_users WHERE id = ?`, ownerID).Scan(&ownerLogin)
	if err != nil {
		return nil, err
	}
	fullName = fmt.Sprintf("%s/%s", ownerLogin, name)

	now := time.Now()
	result, err := s.db.Exec(`
		INSERT INTO github_repositories (owner_id, name, full_name, description, private, default_branch, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'main', ?, ?)
	`, ownerID, name, fullName, description, private, now, now)

	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &Repository{
		ID:            id,
		OwnerID:       ownerID,
		Name:          name,
		FullName:      fullName,
		Description:   description,
		Private:       private,
		DefaultBranch: "main",
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

// GetRepository gets a repository by owner ID and name
func (s *GitHubStore) GetRepository(ownerID int64, name string) (*Repository, error) {
	var repo Repository
	var description, pushedAt sql.NullString
	var pushedAtTime *time.Time

	err := s.db.QueryRow(`
		SELECT id, owner_id, name, full_name, description, private, default_branch, fork, archived, disabled,
			stargazers_count, watchers_count, forks_count, open_issues_count,
			created_at, updated_at, pushed_at
		FROM github_repositories
		WHERE owner_id = ? AND name = ?
	`, ownerID, name).Scan(
		&repo.ID, &repo.OwnerID, &repo.Name, &repo.FullName, &description, &repo.Private,
		&repo.DefaultBranch, &repo.Fork, &repo.Archived, &repo.Disabled,
		&repo.StargazersCount, &repo.WatchersCount, &repo.ForksCount, &repo.OpenIssuesCount,
		&repo.CreatedAt, &repo.UpdatedAt, &pushedAt,
	)

	if err != nil {
		return nil, err
	}

	if description.Valid {
		repo.Description = description.String
	}
	if pushedAt.Valid {
		t, _ := time.Parse(time.RFC3339, pushedAt.String)
		pushedAtTime = &t
		repo.PushedAt = pushedAtTime
	}

	return &repo, nil
}

// GetRepositoryByFullName gets a repository by full name (owner/repo)
func (s *GitHubStore) GetRepositoryByFullName(fullName string) (*Repository, error) {
	var repo Repository
	var description, pushedAt sql.NullString
	var pushedAtTime *time.Time

	err := s.db.QueryRow(`
		SELECT id, owner_id, name, full_name, description, private, default_branch, fork, archived, disabled,
			stargazers_count, watchers_count, forks_count, open_issues_count,
			created_at, updated_at, pushed_at
		FROM github_repositories
		WHERE full_name = ?
	`, fullName).Scan(
		&repo.ID, &repo.OwnerID, &repo.Name, &repo.FullName, &description, &repo.Private,
		&repo.DefaultBranch, &repo.Fork, &repo.Archived, &repo.Disabled,
		&repo.StargazersCount, &repo.WatchersCount, &repo.ForksCount, &repo.OpenIssuesCount,
		&repo.CreatedAt, &repo.UpdatedAt, &pushedAt,
	)

	if err != nil {
		return nil, err
	}

	if description.Valid {
		repo.Description = description.String
	}
	if pushedAt.Valid {
		t, _ := time.Parse(time.RFC3339, pushedAt.String)
		pushedAtTime = &t
		repo.PushedAt = pushedAtTime
	}

	return &repo, nil
}

// ListUserRepositories lists all repositories for a user
func (s *GitHubStore) ListUserRepositories(ownerID int64) ([]*Repository, error) {
	rows, err := s.db.Query(`
		SELECT id, owner_id, name, full_name, description, private, default_branch, fork, archived, disabled,
			stargazers_count, watchers_count, forks_count, open_issues_count,
			created_at, updated_at, pushed_at
		FROM github_repositories
		WHERE owner_id = ?
		ORDER BY created_at DESC
	`, ownerID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []*Repository
	for rows.Next() {
		var repo Repository
		var description, pushedAt sql.NullString
		var pushedAtTime *time.Time

		err := rows.Scan(
			&repo.ID, &repo.OwnerID, &repo.Name, &repo.FullName, &description, &repo.Private,
			&repo.DefaultBranch, &repo.Fork, &repo.Archived, &repo.Disabled,
			&repo.StargazersCount, &repo.WatchersCount, &repo.ForksCount, &repo.OpenIssuesCount,
			&repo.CreatedAt, &repo.UpdatedAt, &pushedAt,
		)
		if err != nil {
			return nil, err
		}

		if description.Valid {
			repo.Description = description.String
		}
		if pushedAt.Valid {
			t, _ := time.Parse(time.RFC3339, pushedAt.String)
			pushedAtTime = &t
			repo.PushedAt = pushedAtTime
		}

		repos = append(repos, &repo)
	}

	return repos, rows.Err()
}
```

### Step 4: Create handlers.go

Create `plugins/github/handlers.go`:

```go
// ABOUTME: HTTP handlers for GitHub REST API v3 endpoints
// ABOUTME: Implements user, repository, issue, PR, review, webhook handlers

package github

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// getAuthenticatedUser handles GET /user
func (p *GitHubPlugin) getAuthenticatedUser(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userContextKey).(*User)

	response := map[string]interface{}{
		"login":      user.Login,
		"id":         user.ID,
		"type":       user.Type,
		"name":       user.Name,
		"email":      user.Email,
		"avatar_url": user.AvatarURL,
		"created_at": user.CreatedAt.Format(time.RFC3339),
		"updated_at": user.UpdatedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// createUserRepository handles POST /user/repos
func (p *GitHubPlugin) createUserRepository(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userContextKey).(*User)

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Private     bool   `json:"private"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	repo, err := p.store.CreateRepository(user.ID, req.Name, req.Description, req.Private)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create repository")
		return
	}

	response := repositoryToResponse(repo, user)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// listAuthenticatedUserRepositories handles GET /user/repos
func (p *GitHubPlugin) listAuthenticatedUserRepositories(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userContextKey).(*User)

	repos, err := p.store.ListUserRepositories(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list repositories")
		return
	}

	var response []map[string]interface{}
	for _, repo := range repos {
		response = append(response, repositoryToResponse(repo, user))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getRepository handles GET /repos/{owner}/{repo}
func (p *GitHubPlugin) getRepository(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")

	fullName := owner + "/" + repoName
	repo, err := p.store.GetRepositoryByFullName(fullName)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	// Get owner user
	ownerUser, err := p.store.GetUserByID(repo.OwnerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get owner")
		return
	}

	response := repositoryToResponse(repo, ownerUser)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// repositoryToResponse converts Repository to GitHub API response format
func repositoryToResponse(repo *Repository, owner *User) map[string]interface{} {
	response := map[string]interface{}{
		"id":                repo.ID,
		"name":              repo.Name,
		"full_name":         repo.FullName,
		"description":       repo.Description,
		"private":           repo.Private,
		"fork":              repo.Fork,
		"archived":          repo.Archived,
		"disabled":          repo.Disabled,
		"default_branch":    repo.DefaultBranch,
		"stargazers_count":  repo.StargazersCount,
		"watchers_count":    repo.WatchersCount,
		"forks_count":       repo.ForksCount,
		"open_issues_count": repo.OpenIssuesCount,
		"created_at":        repo.CreatedAt.Format(time.RFC3339),
		"updated_at":        repo.UpdatedAt.Format(time.RFC3339),
		"owner": map[string]interface{}{
			"login": owner.Login,
			"id":    owner.ID,
			"type":  owner.Type,
		},
	}

	if repo.PushedAt != nil {
		response["pushed_at"] = repo.PushedAt.Format(time.RFC3339)
	}

	return response
}
```

### Step 5: Add GetUserByID to store

Modify `plugins/github/store.go`, add this method:

```go
// GetUserByID gets a user by ID
func (s *GitHubStore) GetUserByID(id int64) (*User, error) {
	var user User
	err := s.db.QueryRow(`
		SELECT id, login, name, email, avatar_url, type, created_at, updated_at
		FROM github_users WHERE id = ?
	`, id).Scan(&user.ID, &user.Login, &user.Name, &user.Email, &user.AvatarURL, &user.Type, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, err
	}
	return &user, nil
}
```

### Step 6: Register routes in plugin

Modify `plugins/github/plugin.go`, update `RegisterRoutes`:

```go
func (p *GitHubPlugin) RegisterRoutes(r chi.Router) {
	// User endpoints
	r.Get("/user", p.requireAuth(p.getAuthenticatedUser))
	r.Patch("/user", p.requireAuth(p.updateAuthenticatedUser))
	r.Get("/users/{username}", p.requireAuth(p.getUser))

	// Repository endpoints
	r.Get("/user/repos", p.requireAuth(p.listAuthenticatedUserRepositories))
	r.Post("/user/repos", p.requireAuth(p.createUserRepository))
	r.Get("/repos/{owner}/{repo}", p.requireAuth(p.getRepository))
	r.Patch("/repos/{owner}/{repo}", p.requireAuth(p.updateRepository))
	r.Delete("/repos/{owner}/{repo}", p.requireAuth(p.deleteRepository))

	// More routes will be added in later tasks
}
```

Add placeholder handlers for now:

```go
func (p *GitHubPlugin) updateAuthenticatedUser(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}

func (p *GitHubPlugin) getUser(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}

func (p *GitHubPlugin) updateRepository(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}

func (p *GitHubPlugin) deleteRepository(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}
```

### Step 7: Run tests to verify they pass

```bash
cd plugins/github
go test -v
```

Expected: PASS (all tests)

### Step 8: Commit

```bash
git add plugins/github/
git commit -m "feat(github): implement repository API handlers

- Add GET /user (authenticated user)
- Add POST /user/repos (create repository)
- Add GET /user/repos (list user repositories)
- Add GET /repos/{owner}/{repo} (get repository)
- Add store methods: CreateRepository, GetRepository, ListUserRepositories, GetUserByID
- Tests for all repository endpoints

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 4: Issues API Handlers

**Files:**
- Modify: `plugins/github/handlers.go`
- Modify: `plugins/github/store.go`
- Modify: `plugins/github/handlers_test.go`
- Modify: `plugins/github/plugin.go`

### Step 1: Write the failing test

Add to `plugins/github/handlers_test.go`:

```go
func TestCreateIssue(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Create user and repo
	user, _ := store.GetOrCreateUser("alice", "ghp_test")
	repo, _ := store.CreateRepository(user.ID, "test-repo", "", false)

	body := `{"title": "Bug in login", "body": "Login fails with invalid credentials"}`
	req := httptest.NewRequest("POST", "/repos/alice/test-repo/issues", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer ghp_test")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Setup chi context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("owner", "alice")
	rctx.URLParams.Add("repo", "test-repo")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := plugin.requireAuth(plugin.createIssue)
	handler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["number"] != float64(1) {
		t.Fatalf("Expected number 1, got %v", resp["number"])
	}
	if resp["title"] != "Bug in login" {
		t.Fatalf("Expected title, got %v", resp["title"])
	}
	if resp["state"] != "open" {
		t.Fatalf("Expected state 'open', got %v", resp["state"])
	}

	// Verify in database
	issue, err := store.GetIssueByNumber(repo.ID, 1)
	if err != nil {
		t.Fatalf("Issue not found: %v", err)
	}
	if issue.Title != "Bug in login" {
		t.Fatal("Title mismatch in database")
	}
}

func TestListIssues(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Create user, repo, and issues
	user, _ := store.GetOrCreateUser("alice", "ghp_test")
	repo, _ := store.CreateRepository(user.ID, "test-repo", "", false)
	store.CreateIssue(repo.ID, user.ID, "Issue 1", "Body 1", false)
	store.CreateIssue(repo.ID, user.ID, "Issue 2", "Body 2", false)

	req := httptest.NewRequest("GET", "/repos/alice/test-repo/issues", nil)
	req.Header.Set("Authorization", "Bearer ghp_test")
	w := httptest.NewRecorder()

	// Setup chi context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("owner", "alice")
	rctx.URLParams.Add("repo", "test-repo")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := plugin.requireAuth(plugin.listIssues)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var issues []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &issues)

	if len(issues) != 2 {
		t.Fatalf("Expected 2 issues, got %d", len(issues))
	}
}

func TestUpdateIssueState(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Create user, repo, and issue
	user, _ := store.GetOrCreateUser("alice", "ghp_test")
	repo, _ := store.CreateRepository(user.ID, "test-repo", "", false)
	issue, _ := store.CreateIssue(repo.ID, user.ID, "Test issue", "Body", false)

	body := `{"state": "closed", "state_reason": "completed"}`
	req := httptest.NewRequest("PATCH", "/repos/alice/test-repo/issues/1", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer ghp_test")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Setup chi context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("owner", "alice")
	rctx.URLParams.Add("repo", "test-repo")
	rctx.URLParams.Add("number", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := plugin.requireAuth(plugin.updateIssue)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify in database
	updated, _ := store.GetIssueByNumber(repo.ID, int(issue.Number))
	if updated.State != "closed" {
		t.Fatalf("Expected state 'closed', got '%s'", updated.State)
	}
	if updated.StateReason != "completed" {
		t.Fatalf("Expected state_reason 'completed', got '%s'", updated.StateReason)
	}
	if updated.ClosedAt == nil {
		t.Fatal("ClosedAt should be set")
	}
}
```

### Step 2: Run test to verify it fails

```bash
cd plugins/github
go test -v -run TestCreateIssue
```

Expected: FAIL - undefined: createIssue

### Step 3: Add Issue type and store methods

Add to `plugins/github/store.go`:

```go
type Issue struct {
	ID            int64
	RepoID        int64
	Number        int64
	Title         string
	Body          string
	State         string
	StateReason   string
	UserID        int64
	AssigneeIDs   string
	LabelIDs      string
	MilestoneID   *int64
	Locked        bool
	CommentsCount int
	IsPullRequest bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
	ClosedAt      *time.Time
}

// CreateIssue creates a new issue with auto-incrementing number per repo
func (s *GitHubStore) CreateIssue(repoID, userID int64, title, body string, isPR bool) (*Issue, error) {
	// Get next issue number for this repo
	var maxNumber sql.NullInt64
	err := s.db.QueryRow(`SELECT MAX(number) FROM github_issues WHERE repo_id = ?`, repoID).Scan(&maxNumber)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	number := int64(1)
	if maxNumber.Valid {
		number = maxNumber.Int64 + 1
	}

	now := time.Now()
	isPRInt := 0
	if isPR {
		isPRInt = 1
	}

	result, err := s.db.Exec(`
		INSERT INTO github_issues (repo_id, number, title, body, state, user_id, is_pull_request, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'open', ?, ?, ?, ?)
	`, repoID, number, title, body, userID, isPRInt, now, now)

	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &Issue{
		ID:            id,
		RepoID:        repoID,
		Number:        number,
		Title:         title,
		Body:          body,
		State:         "open",
		UserID:        userID,
		IsPullRequest: isPR,
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

// GetIssueByNumber gets an issue by repo ID and number
func (s *GitHubStore) GetIssueByNumber(repoID int64, number int) (*Issue, error) {
	var issue Issue
	var body, stateReason, assigneeIDs, labelIDs sql.NullString
	var milestoneID sql.NullInt64
	var closedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, repo_id, number, title, body, state, state_reason, user_id, assignee_ids, label_ids, milestone_id,
			locked, comments_count, is_pull_request, created_at, updated_at, closed_at
		FROM github_issues
		WHERE repo_id = ? AND number = ?
	`, repoID, number).Scan(
		&issue.ID, &issue.RepoID, &issue.Number, &issue.Title, &body, &issue.State, &stateReason,
		&issue.UserID, &assigneeIDs, &labelIDs, &milestoneID, &issue.Locked, &issue.CommentsCount,
		&issue.IsPullRequest, &issue.CreatedAt, &issue.UpdatedAt, &closedAt,
	)

	if err != nil {
		return nil, err
	}

	if body.Valid {
		issue.Body = body.String
	}
	if stateReason.Valid {
		issue.StateReason = stateReason.String
	}
	if assigneeIDs.Valid {
		issue.AssigneeIDs = assigneeIDs.String
	}
	if labelIDs.Valid {
		issue.LabelIDs = labelIDs.String
	}
	if milestoneID.Valid {
		id := milestoneID.Int64
		issue.MilestoneID = &id
	}
	if closedAt.Valid {
		issue.ClosedAt = &closedAt.Time
	}

	return &issue, nil
}

// ListIssues lists issues for a repository (excludes PRs by default)
func (s *GitHubStore) ListIssues(repoID int64, state string, includePRs bool) ([]*Issue, error) {
	query := `
		SELECT id, repo_id, number, title, body, state, state_reason, user_id, assignee_ids, label_ids, milestone_id,
			locked, comments_count, is_pull_request, created_at, updated_at, closed_at
		FROM github_issues
		WHERE repo_id = ?
	`

	args := []interface{}{repoID}

	if state != "" {
		query += " AND state = ?"
		args = append(args, state)
	}

	if !includePRs {
		query += " AND is_pull_request = 0"
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var issues []*Issue
	for rows.Next() {
		var issue Issue
		var body, stateReason, assigneeIDs, labelIDs sql.NullString
		var milestoneID sql.NullInt64
		var closedAt sql.NullTime

		err := rows.Scan(
			&issue.ID, &issue.RepoID, &issue.Number, &issue.Title, &body, &issue.State, &stateReason,
			&issue.UserID, &assigneeIDs, &labelIDs, &milestoneID, &issue.Locked, &issue.CommentsCount,
			&issue.IsPullRequest, &issue.CreatedAt, &issue.UpdatedAt, &closedAt,
		)
		if err != nil {
			return nil, err
		}

		if body.Valid {
			issue.Body = body.String
		}
		if stateReason.Valid {
			issue.StateReason = stateReason.String
		}
		if assigneeIDs.Valid {
			issue.AssigneeIDs = assigneeIDs.String
		}
		if labelIDs.Valid {
			issue.LabelIDs = labelIDs.String
		}
		if milestoneID.Valid {
			id := milestoneID.Int64
			issue.MilestoneID = &id
		}
		if closedAt.Valid {
			issue.ClosedAt = &closedAt.Time
		}

		issues = append(issues, &issue)
	}

	return issues, rows.Err()
}

// UpdateIssue updates an issue
func (s *GitHubStore) UpdateIssue(issue *Issue) error {
	now := time.Now()
	issue.UpdatedAt = now

	_, err := s.db.Exec(`
		UPDATE github_issues
		SET title = ?, body = ?, state = ?, state_reason = ?, updated_at = ?, closed_at = ?
		WHERE id = ?
	`, issue.Title, issue.Body, issue.State, issue.StateReason, issue.UpdatedAt, issue.ClosedAt, issue.ID)

	return err
}
```

### Step 4: Add issue handlers

Add to `plugins/github/handlers.go`:

```go
// createIssue handles POST /repos/{owner}/{repo}/issues
func (p *GitHubPlugin) createIssue(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userContextKey).(*User)
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")

	var req struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	// Get repository
	fullName := owner + "/" + repoName
	repo, err := p.store.GetRepositoryByFullName(fullName)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	issue, err := p.store.CreateIssue(repo.ID, user.ID, req.Title, req.Body, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create issue")
		return
	}

	response := issueToResponse(issue, user, repo)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// listIssues handles GET /repos/{owner}/{repo}/issues
func (p *GitHubPlugin) listIssues(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")

	// Get repository
	fullName := owner + "/" + repoName
	repo, err := p.store.GetRepositoryByFullName(fullName)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	state := r.URL.Query().Get("state") // open, closed, all

	issues, err := p.store.ListIssues(repo.ID, state, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list issues")
		return
	}

	var response []map[string]interface{}
	for _, issue := range issues {
		issueUser, _ := p.store.GetUserByID(issue.UserID)
		response = append(response, issueToResponse(issue, issueUser, repo))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getIssue handles GET /repos/{owner}/{repo}/issues/{number}
func (p *GitHubPlugin) getIssue(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")
	number := chi.URLParam(r, "number")

	// Get repository
	fullName := owner + "/" + repoName
	repo, err := p.store.GetRepositoryByFullName(fullName)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	// Parse number
	var issueNum int
	fmt.Sscanf(number, "%d", &issueNum)

	issue, err := p.store.GetIssueByNumber(repo.ID, issueNum)
	if err != nil {
		writeError(w, http.StatusNotFound, "issue not found")
		return
	}

	issueUser, _ := p.store.GetUserByID(issue.UserID)
	response := issueToResponse(issue, issueUser, repo)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// updateIssue handles PATCH /repos/{owner}/{repo}/issues/{number}
func (p *GitHubPlugin) updateIssue(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")
	number := chi.URLParam(r, "number")

	var req struct {
		Title       *string `json:"title"`
		Body        *string `json:"body"`
		State       *string `json:"state"`
		StateReason *string `json:"state_reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get repository
	fullName := owner + "/" + repoName
	repo, err := p.store.GetRepositoryByFullName(fullName)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	// Parse number
	var issueNum int
	fmt.Sscanf(number, "%d", &issueNum)

	issue, err := p.store.GetIssueByNumber(repo.ID, issueNum)
	if err != nil {
		writeError(w, http.StatusNotFound, "issue not found")
		return
	}

	// Update fields
	if req.Title != nil {
		issue.Title = *req.Title
	}
	if req.Body != nil {
		issue.Body = *req.Body
	}
	if req.State != nil {
		issue.State = *req.State
		if *req.State == "closed" && issue.ClosedAt == nil {
			now := time.Now()
			issue.ClosedAt = &now
		} else if *req.State == "open" {
			issue.ClosedAt = nil
		}
	}
	if req.StateReason != nil {
		issue.StateReason = *req.StateReason
	}

	if err := p.store.UpdateIssue(issue); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update issue")
		return
	}

	issueUser, _ := p.store.GetUserByID(issue.UserID)
	response := issueToResponse(issue, issueUser, repo)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// issueToResponse converts Issue to GitHub API response format
func issueToResponse(issue *Issue, user *User, repo *Repository) map[string]interface{} {
	response := map[string]interface{}{
		"id":             issue.ID,
		"number":         issue.Number,
		"title":          issue.Title,
		"body":           issue.Body,
		"state":          issue.State,
		"locked":         issue.Locked,
		"comments":       issue.CommentsCount,
		"created_at":     issue.CreatedAt.Format(time.RFC3339),
		"updated_at":     issue.UpdatedAt.Format(time.RFC3339),
		"user": map[string]interface{}{
			"login": user.Login,
			"id":    user.ID,
			"type":  user.Type,
		},
		"repository_url": fmt.Sprintf("/repos/%s", repo.FullName),
	}

	if issue.StateReason != "" {
		response["state_reason"] = issue.StateReason
	}

	if issue.ClosedAt != nil {
		response["closed_at"] = issue.ClosedAt.Format(time.RFC3339)
	}

	return response
}
```

### Step 5: Register issue routes

Modify `plugins/github/plugin.go`, update `RegisterRoutes` to add:

```go
	// Issue endpoints
	r.Get("/repos/{owner}/{repo}/issues", p.requireAuth(p.listIssues))
	r.Post("/repos/{owner}/{repo}/issues", p.requireAuth(p.createIssue))
	r.Get("/repos/{owner}/{repo}/issues/{number}", p.requireAuth(p.getIssue))
	r.Patch("/repos/{owner}/{repo}/issues/{number}", p.requireAuth(p.updateIssue))
```

### Step 6: Run tests to verify they pass

```bash
cd plugins/github
go test -v
```

Expected: PASS (all tests)

### Step 7: Commit

```bash
git add plugins/github/
git commit -m "feat(github): implement Issues API handlers

- Add POST /repos/{owner}/{repo}/issues (create issue)
- Add GET /repos/{owner}/{repo}/issues (list issues)
- Add GET /repos/{owner}/{repo}/issues/{number} (get issue)
- Add PATCH /repos/{owner}/{repo}/issues/{number} (update issue)
- Auto-incrementing issue numbers per repository
- State management (open/closed) with closed_at timestamp
- Tests for create, list, get, update operations

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Remaining Tasks Summary

**Task 5**: Pull Requests API (similar to Issues, extends github_issues table)
**Task 6**: Comments API (issue comments, PR comments)
**Task 7**: Reviews API (PR reviews with states)
**Task 8**: Webhook System (delivery, SSRF protection, HMAC signatures)
**Task 9**: Admin UI Schema
**Task 10**: Seeding Strategy
**Task 11**: Register Plugin in main
**Task 12**: Integration Tests
**Task 13**: Documentation

Due to length constraints, I'll save this plan now and you can request the remaining tasks if needed. The pattern for Tasks 5-13 follows the same TDD structure shown above.

---

## Execution Handoff

Plan saved to `docs/plans/2025-12-06-github-plugin-implementation.md`.

**Two execution options:**

**1. Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

**Which approach?**
