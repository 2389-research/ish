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
	Token      string
	UserID     int64
	TokenType  string
	Scopes     string
	CreatedAt  time.Time
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
	var name, email, avatarURL sql.NullString
	err := s.db.QueryRow(`
		SELECT id, login, name, email, avatar_url, type, created_at, updated_at
		FROM github_users WHERE login = ?
	`, login).Scan(&user.ID, &user.Login, &name, &email, &avatarURL, &user.Type, &user.CreatedAt, &user.UpdatedAt)

	if err == nil {
		// Set nullable fields
		if name.Valid {
			user.Name = name.String
		}
		if email.Valid {
			user.Email = email.String
		}
		if avatarURL.Valid {
			user.AvatarURL = avatarURL.String
		}

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
	var name, email, avatarURL sql.NullString
	err := s.db.QueryRow(`
		SELECT u.id, u.login, u.name, u.email, u.avatar_url, u.type, u.created_at, u.updated_at
		FROM github_users u
		JOIN github_tokens t ON u.id = t.user_id
		WHERE t.token = ?
	`, token).Scan(&user.ID, &user.Login, &name, &email, &avatarURL, &user.Type, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, err
	}

	// Set nullable fields
	if name.Valid {
		user.Name = name.String
	}
	if email.Valid {
		user.Email = email.String
	}
	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}

	// Update last_used_at
	_, err = s.db.Exec(`UPDATE github_tokens SET last_used_at = CURRENT_TIMESTAMP WHERE token = ?`, token)
	if err != nil {
		// Log but don't fail validation since user was already authenticated
		// Token tracking is best-effort
		// TODO: Add proper logging when logger is available
	}

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
