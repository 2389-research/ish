// ABOUTME: Database layer for GitHub plugin
// ABOUTME: Manages all github_* tables and CRUD operations

package github

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
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

type PullRequest struct {
	IssueID               int64
	HeadRepoID            int64
	HeadRef               string
	BaseRepoID            int64
	BaseRef               string
	Merged                bool
	Mergeable             bool
	Rebaseable            bool
	MergeCommitSHA        string
	MergedAt              *time.Time
	MergedByID            *int64
	Draft                 bool
	ReviewCommentsCount   int
	CommitsCount          int
	Additions             int
	Deletions             int
	ChangedFiles          int
}

type Comment struct {
	ID        int64
	IssueID   int64
	UserID    int64
	Body      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Review struct {
	ID            int64
	PullRequestID int64
	UserID        int64
	State         string
	Body          string
	CommitSHA     string
	SubmittedAt   *time.Time
	DismissedAt   *time.Time
}

type Webhook struct {
	ID          int64
	RepoID      int64
	URL         string
	ContentType string
	Secret      string
	Events      string
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type WebhookDelivery struct {
	ID           int64
	WebhookID    int64
	EventType    string
	Payload      string
	DeliveredAt  time.Time
	StatusCode   int
	ErrorMessage string
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
		fmt.Fprintf(os.Stderr, "github: warning: failed to update token last_used_at: %v\n", err)
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

// GetUserByID gets a user by ID
func (s *GitHubStore) GetUserByID(id int64) (*User, error) {
	var user User
	var name, email, avatarURL sql.NullString
	err := s.db.QueryRow(`
		SELECT id, login, name, email, avatar_url, type, created_at, updated_at
		FROM github_users WHERE id = ?
	`, id).Scan(&user.ID, &user.Login, &name, &email, &avatarURL, &user.Type, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, err
	}

	if name.Valid {
		user.Name = name.String
	}
	if email.Valid {
		user.Email = email.String
	}
	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}

	return &user, nil
}

// CreateRepository creates a new repository
func (s *GitHubStore) CreateRepository(ownerID int64, name, description string, private bool) (*Repository, error) {
	// Get owner login
	var ownerLogin string
	err := s.db.QueryRow(`SELECT login FROM github_users WHERE id = ?`, ownerID).Scan(&ownerLogin)
	if err != nil {
		return nil, err
	}
	fullName := fmt.Sprintf("%s/%s", ownerLogin, name)

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
	var description sql.NullString
	var pushedAt sql.NullTime

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
		repo.PushedAt = &pushedAt.Time
	}

	return &repo, nil
}

// GetRepositoryByFullName gets a repository by full name (owner/repo)
func (s *GitHubStore) GetRepositoryByFullName(fullName string) (*Repository, error) {
	var repo Repository
	var description sql.NullString
	var pushedAt sql.NullTime

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
		repo.PushedAt = &pushedAt.Time
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
		var description sql.NullString
		var pushedAt sql.NullTime

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
			repo.PushedAt = &pushedAt.Time
		}

		repos = append(repos, &repo)
	}

	return repos, rows.Err()
}

// CreateIssue creates a new issue with auto-incrementing number per repo
// Uses a transaction to prevent race conditions in number assignment
func (s *GitHubStore) CreateIssue(repoID, userID int64, title, body string, isPR bool) (*Issue, error) {
	// Retry loop to handle race condition in issue number assignment
	// The UNIQUE(repo_id, number) constraint will catch duplicate numbers
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		issue, err := s.createIssueAttempt(repoID, userID, title, body, isPR)
		if err != nil {
			// Check if it's a unique constraint violation (race condition)
			// SQLite error message contains "UNIQUE constraint failed"
			if attempt < maxRetries-1 && (err.Error() == "UNIQUE constraint failed: github_issues.repo_id, github_issues.number" ||
				err.Error() == "constraint failed") {
				// Retry with new number
				continue
			}
			return nil, err
		}
		return issue, nil
	}
	return nil, fmt.Errorf("failed to create issue after %d attempts", maxRetries)
}

// createIssueAttempt performs a single attempt to create an issue
func (s *GitHubStore) createIssueAttempt(repoID, userID int64, title, body string, isPR bool) (*Issue, error) {
	// Start transaction for atomic number assignment
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() // Will be no-op if tx.Commit() succeeds

	// Get next issue number for this repo (within transaction)
	var maxNumber sql.NullInt64
	err = tx.QueryRow(`SELECT MAX(number) FROM github_issues WHERE repo_id = ?`, repoID).Scan(&maxNumber)
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

	// Insert issue with the calculated number
	result, err := tx.Exec(`
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

	// Commit transaction
	if err := tx.Commit(); err != nil {
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

// CreatePullRequest creates a new pull request (issue + PR record) atomically
// Uses a transaction to ensure both the issue and PR are created together
func (s *GitHubStore) CreatePullRequest(repoID, userID int64, title, body, headRef, baseRef string) (*Issue, *PullRequest, error) {
	// Start transaction for atomic PR+Issue creation
	tx, err := s.db.Begin()
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback() // Will be no-op if tx.Commit() succeeds

	// Get next issue number for this repo (within transaction)
	var maxNumber sql.NullInt64
	err = tx.QueryRow(`SELECT MAX(number) FROM github_issues WHERE repo_id = ?`, repoID).Scan(&maxNumber)
	if err != nil && err != sql.ErrNoRows {
		return nil, nil, err
	}

	number := int64(1)
	if maxNumber.Valid {
		number = maxNumber.Int64 + 1
	}

	now := time.Now()

	// Create the issue with is_pull_request=1
	result, err := tx.Exec(`
		INSERT INTO github_issues (repo_id, number, title, body, state, user_id, is_pull_request, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'open', ?, 1, ?, ?)
	`, repoID, number, title, body, userID, now, now)

	if err != nil {
		return nil, nil, err
	}

	issueID, err := result.LastInsertId()
	if err != nil {
		return nil, nil, err
	}

	// Create the PR record
	_, err = tx.Exec(`
		INSERT INTO github_pull_requests (issue_id, head_repo_id, head_ref, base_repo_id, base_ref, merged, mergeable, rebaseable)
		VALUES (?, ?, ?, ?, ?, 0, 1, 1)
	`, issueID, repoID, headRef, repoID, baseRef)

	if err != nil {
		return nil, nil, err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}

	issue := &Issue{
		ID:            issueID,
		RepoID:        repoID,
		Number:        number,
		Title:         title,
		Body:          body,
		State:         "open",
		UserID:        userID,
		IsPullRequest: true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	pr := &PullRequest{
		IssueID:    issueID,
		HeadRepoID: repoID,
		HeadRef:    headRef,
		BaseRepoID: repoID,
		BaseRef:    baseRef,
		Merged:     false,
		Mergeable:  true,
		Rebaseable: true,
	}

	return issue, pr, nil
}

// GetPullRequest gets a pull request by repo ID and number
func (s *GitHubStore) GetPullRequest(repoID int64, number int) (*Issue, *PullRequest, error) {
	// Get the issue first
	issue, err := s.GetIssueByNumber(repoID, number)
	if err != nil {
		return nil, nil, err
	}

	if !issue.IsPullRequest {
		return nil, nil, fmt.Errorf("issue #%d is not a pull request", number)
	}

	// Get the PR data
	var pr PullRequest
	var mergeCommitSHA sql.NullString
	var mergedAt sql.NullTime
	var mergedByID sql.NullInt64

	err = s.db.QueryRow(`
		SELECT issue_id, head_repo_id, head_ref, base_repo_id, base_ref, merged, mergeable, rebaseable,
			merge_commit_sha, merged_at, merged_by_id, draft, review_comments_count, commits_count,
			additions, deletions, changed_files
		FROM github_pull_requests
		WHERE issue_id = ?
	`, issue.ID).Scan(
		&pr.IssueID, &pr.HeadRepoID, &pr.HeadRef, &pr.BaseRepoID, &pr.BaseRef, &pr.Merged, &pr.Mergeable, &pr.Rebaseable,
		&mergeCommitSHA, &mergedAt, &mergedByID, &pr.Draft, &pr.ReviewCommentsCount, &pr.CommitsCount,
		&pr.Additions, &pr.Deletions, &pr.ChangedFiles,
	)

	if err != nil {
		return nil, nil, err
	}

	if mergeCommitSHA.Valid {
		pr.MergeCommitSHA = mergeCommitSHA.String
	}
	if mergedAt.Valid {
		pr.MergedAt = &mergedAt.Time
	}
	if mergedByID.Valid {
		id := mergedByID.Int64
		pr.MergedByID = &id
	}

	return issue, &pr, nil
}

// ListPullRequests lists pull requests for a repository
func (s *GitHubStore) ListPullRequests(repoID int64, state string) ([]*Issue, error) {
	// List issues where is_pull_request=1
	return s.ListIssues(repoID, state, true)
}

// MergePullRequest marks a PR as merged and closes the issue
func (s *GitHubStore) MergePullRequest(issueID, mergedByID int64) error {
	now := time.Now()

	// Update the PR record
	_, err := s.db.Exec(`
		UPDATE github_pull_requests
		SET merged = 1, merged_at = ?, merged_by_id = ?
		WHERE issue_id = ?
	`, now, mergedByID, issueID)

	if err != nil {
		return err
	}

	// Close the issue
	_, err = s.db.Exec(`
		UPDATE github_issues
		SET state = 'closed', closed_at = ?, updated_at = ?
		WHERE id = ?
	`, now, now, issueID)

	return err
}

// CreateComment creates a new comment and increments the issue's comments_count
// Uses a transaction to ensure atomicity
func (s *GitHubStore) CreateComment(issueID, userID int64, body string) (*Comment, error) {
	// Start transaction for atomic insert + count update
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() // Will be no-op if tx.Commit() succeeds

	now := time.Now()

	// Insert the comment
	result, err := tx.Exec(`
		INSERT INTO github_comments (issue_id, user_id, body, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, issueID, userID, body, now, now)

	if err != nil {
		return nil, err
	}

	commentID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	// Increment issue's comments_count
	_, err = tx.Exec(`
		UPDATE github_issues
		SET comments_count = comments_count + 1
		WHERE id = ?
	`, issueID)

	if err != nil {
		return nil, err
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &Comment{
		ID:        commentID,
		IssueID:   issueID,
		UserID:    userID,
		Body:      body,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// GetComment gets a comment by ID
func (s *GitHubStore) GetComment(commentID int64) (*Comment, error) {
	var comment Comment

	err := s.db.QueryRow(`
		SELECT id, issue_id, user_id, body, created_at, updated_at
		FROM github_comments
		WHERE id = ?
	`, commentID).Scan(&comment.ID, &comment.IssueID, &comment.UserID, &comment.Body, &comment.CreatedAt, &comment.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return &comment, nil
}

// ListComments lists all comments for an issue/PR
func (s *GitHubStore) ListComments(issueID int64) ([]*Comment, error) {
	rows, err := s.db.Query(`
		SELECT id, issue_id, user_id, body, created_at, updated_at
		FROM github_comments
		WHERE issue_id = ?
		ORDER BY created_at ASC
	`, issueID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*Comment
	for rows.Next() {
		var comment Comment
		err := rows.Scan(&comment.ID, &comment.IssueID, &comment.UserID, &comment.Body, &comment.CreatedAt, &comment.UpdatedAt)
		if err != nil {
			return nil, err
		}
		comments = append(comments, &comment)
	}

	return comments, rows.Err()
}

// UpdateComment updates a comment's body and updated_at timestamp
func (s *GitHubStore) UpdateComment(comment *Comment) error {
	now := time.Now()
	comment.UpdatedAt = now

	_, err := s.db.Exec(`
		UPDATE github_comments
		SET body = ?, updated_at = ?
		WHERE id = ?
	`, comment.Body, comment.UpdatedAt, comment.ID)

	return err
}

// DeleteComment deletes a comment (hard delete) and decrements the issue's comment count
// Uses a transaction to ensure atomicity
func (s *GitHubStore) DeleteComment(commentID int64) error {
	// Start transaction for atomic delete + count update
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // Will be no-op if tx.Commit() succeeds

	// Get the issue_id for this comment
	var issueID int64
	err = tx.QueryRow(`
		SELECT issue_id FROM github_comments WHERE id = ?
	`, commentID).Scan(&issueID)

	if err != nil {
		return err
	}

	// Delete the comment
	_, err = tx.Exec(`
		DELETE FROM github_comments
		WHERE id = ?
	`, commentID)

	if err != nil {
		return err
	}

	// Decrement the issue's comments_count
	_, err = tx.Exec(`
		UPDATE github_issues
		SET comments_count = CASE
			WHEN comments_count > 0 THEN comments_count - 1
			ELSE 0
		END
		WHERE id = ?
	`, issueID)

	if err != nil {
		return err
	}

	return tx.Commit()
}

// generateCommitSHA creates a fake 40-character hex SHA for reviews
func generateCommitSHA() (string, error) {
	bytes := make([]byte, 20)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreateReview creates a new review for a pull request
func (s *GitHubStore) CreateReview(pullRequestID, userID int64, state, body string) (*Review, error) {
	// Generate a fake commit SHA
	commitSHA, err := generateCommitSHA()
	if err != nil {
		return nil, err
	}

	result, err := s.db.Exec(`
		INSERT INTO github_reviews (pull_request_id, user_id, state, body, commit_sha)
		VALUES (?, ?, ?, ?, ?)
	`, pullRequestID, userID, state, body, commitSHA)

	if err != nil {
		return nil, err
	}

	reviewID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &Review{
		ID:            reviewID,
		PullRequestID: pullRequestID,
		UserID:        userID,
		State:         state,
		Body:          body,
		CommitSHA:     commitSHA,
	}, nil
}

// GetReview gets a review by ID
func (s *GitHubStore) GetReview(reviewID int64) (*Review, error) {
	var review Review
	var body sql.NullString
	var submittedAt, dismissedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, pull_request_id, user_id, state, body, commit_sha, submitted_at, dismissed_at
		FROM github_reviews
		WHERE id = ?
	`, reviewID).Scan(&review.ID, &review.PullRequestID, &review.UserID, &review.State, &body, &review.CommitSHA, &submittedAt, &dismissedAt)

	if err != nil {
		return nil, err
	}

	if body.Valid {
		review.Body = body.String
	}
	if submittedAt.Valid {
		review.SubmittedAt = &submittedAt.Time
	}
	if dismissedAt.Valid {
		review.DismissedAt = &dismissedAt.Time
	}

	return &review, nil
}

// ListReviews lists all reviews for a pull request
func (s *GitHubStore) ListReviews(pullRequestID int64) ([]*Review, error) {
	rows, err := s.db.Query(`
		SELECT id, pull_request_id, user_id, state, body, commit_sha, submitted_at, dismissed_at
		FROM github_reviews
		WHERE pull_request_id = ?
		ORDER BY id ASC
	`, pullRequestID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []*Review
	for rows.Next() {
		var review Review
		var body sql.NullString
		var submittedAt, dismissedAt sql.NullTime

		err := rows.Scan(&review.ID, &review.PullRequestID, &review.UserID, &review.State, &body, &review.CommitSHA, &submittedAt, &dismissedAt)
		if err != nil {
			return nil, err
		}

		if body.Valid {
			review.Body = body.String
		}
		if submittedAt.Valid {
			review.SubmittedAt = &submittedAt.Time
		}
		if dismissedAt.Valid {
			review.DismissedAt = &dismissedAt.Time
		}

		reviews = append(reviews, &review)
	}

	return reviews, rows.Err()
}

// SubmitReview sets the submitted_at timestamp for a review
func (s *GitHubStore) SubmitReview(reviewID int64) error {
	now := time.Now()
	_, err := s.db.Exec(`
		UPDATE github_reviews
		SET submitted_at = ?
		WHERE id = ?
	`, now, reviewID)

	return err
}

// DismissReview sets the dismissed_at timestamp and changes state to DISMISSED
func (s *GitHubStore) DismissReview(reviewID int64) error {
	now := time.Now()
	_, err := s.db.Exec(`
		UPDATE github_reviews
		SET state = 'DISMISSED', dismissed_at = ?
		WHERE id = ?
	`, now, reviewID)

	return err
}

// CreateWebhook creates a new webhook for a repository
func (s *GitHubStore) CreateWebhook(repoID int64, url, contentType, secret string, events []string) (*Webhook, error) {
	// Validate the webhook URL for SSRF protection
	if err := validateWebhookURL(url); err != nil {
		return nil, err
	}

	now := time.Now()
	eventsStr := ""
	if len(events) > 0 {
		// Join events with commas
		for i, event := range events {
			if i > 0 {
				eventsStr += ","
			}
			eventsStr += event
		}
	}

	result, err := s.db.Exec(`
		INSERT INTO github_webhooks (repo_id, url, content_type, secret, events, active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 1, ?, ?)
	`, repoID, url, contentType, secret, eventsStr, now, now)

	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &Webhook{
		ID:          id,
		RepoID:      repoID,
		URL:         url,
		ContentType: contentType,
		Secret:      secret,
		Events:      eventsStr,
		Active:      true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// ListWebhooks lists all webhooks for a repository
func (s *GitHubStore) ListWebhooks(repoID int64) ([]*Webhook, error) {
	rows, err := s.db.Query(`
		SELECT id, repo_id, url, content_type, secret, events, active, created_at, updated_at
		FROM github_webhooks
		WHERE repo_id = ?
		ORDER BY id ASC
	`, repoID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []*Webhook
	for rows.Next() {
		var webhook Webhook
		var secret sql.NullString

		err := rows.Scan(
			&webhook.ID, &webhook.RepoID, &webhook.URL, &webhook.ContentType,
			&secret, &webhook.Events, &webhook.Active, &webhook.CreatedAt, &webhook.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if secret.Valid {
			webhook.Secret = secret.String
		}

		webhooks = append(webhooks, &webhook)
	}

	return webhooks, rows.Err()
}

// GetWebhook gets a webhook by ID
func (s *GitHubStore) GetWebhook(webhookID int64) (*Webhook, error) {
	var webhook Webhook
	var secret sql.NullString

	err := s.db.QueryRow(`
		SELECT id, repo_id, url, content_type, secret, events, active, created_at, updated_at
		FROM github_webhooks
		WHERE id = ?
	`, webhookID).Scan(
		&webhook.ID, &webhook.RepoID, &webhook.URL, &webhook.ContentType,
		&secret, &webhook.Events, &webhook.Active, &webhook.CreatedAt, &webhook.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if secret.Valid {
		webhook.Secret = secret.String
	}

	return &webhook, nil
}

// UpdateWebhook updates a webhook
func (s *GitHubStore) UpdateWebhook(webhook *Webhook) error {
	// Validate the webhook URL for SSRF protection
	if err := validateWebhookURL(webhook.URL); err != nil {
		return err
	}

	now := time.Now()
	webhook.UpdatedAt = now

	_, err := s.db.Exec(`
		UPDATE github_webhooks
		SET url = ?, content_type = ?, secret = ?, events = ?, active = ?, updated_at = ?
		WHERE id = ?
	`, webhook.URL, webhook.ContentType, webhook.Secret, webhook.Events, webhook.Active, webhook.UpdatedAt, webhook.ID)

	return err
}

// DeleteWebhook deletes a webhook
func (s *GitHubStore) DeleteWebhook(webhookID int64) error {
	_, err := s.db.Exec(`DELETE FROM github_webhooks WHERE id = ?`, webhookID)
	return err
}

// CreateWebhookDelivery logs a webhook delivery attempt
func (s *GitHubStore) CreateWebhookDelivery(webhookID int64, eventType, payload string, statusCode int, errorMsg string) error {
	_, err := s.db.Exec(`
		INSERT INTO github_webhook_deliveries (webhook_id, event_type, payload, delivered_at, status_code, error_message)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, ?, ?)
	`, webhookID, eventType, payload, statusCode, errorMsg)

	return err
}

// eventMatches checks if an event type matches any event in a comma-separated list
// Returns true if the eventType exactly matches one of the events in the list
func eventMatches(eventsList, eventType string) bool {
	// Split by comma and check for exact matches
	events := strings.Split(eventsList, ",")
	for _, event := range events {
		// Trim whitespace and compare
		if strings.TrimSpace(event) == eventType {
			return true
		}
	}
	return false
}

// GetActiveWebhooksForEvent gets all active webhooks for a repo that subscribe to an event
func (s *GitHubStore) GetActiveWebhooksForEvent(repoID int64, eventType string) ([]*Webhook, error) {
	rows, err := s.db.Query(`
		SELECT id, repo_id, url, content_type, secret, events, active, created_at, updated_at
		FROM github_webhooks
		WHERE repo_id = ? AND active = 1
	`, repoID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []*Webhook
	for rows.Next() {
		var webhook Webhook
		var secret sql.NullString

		err := rows.Scan(
			&webhook.ID, &webhook.RepoID, &webhook.URL, &webhook.ContentType,
			&secret, &webhook.Events, &webhook.Active, &webhook.CreatedAt, &webhook.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if secret.Valid {
			webhook.Secret = secret.String
		}

		// Check if this webhook subscribes to this event type
		// Events are stored as comma-separated list
		// Empty Events means subscribe to all events
		if webhook.Events == "" || eventMatches(webhook.Events, eventType) {
			webhooks = append(webhooks, &webhook)
		}
	}

	return webhooks, rows.Err()
}

// ListAllRepositories retrieves repositories across all accounts for admin view
func (s *GitHubStore) ListAllRepositories(limit, offset int) ([]Repository, error) {
	rows, err := s.db.Query(`
		SELECT id, owner_id, name, full_name, description, private, default_branch, fork, archived, disabled,
			stargazers_count, watchers_count, forks_count, open_issues_count,
			created_at, updated_at, pushed_at
		FROM github_repositories
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []Repository
	for rows.Next() {
		var repo Repository
		var description sql.NullString
		var pushedAt sql.NullTime

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
			repo.PushedAt = &pushedAt.Time
		}

		repos = append(repos, repo)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return repos, nil
}

// ListAllIssues retrieves issues across all repositories for admin view
func (s *GitHubStore) ListAllIssues(limit, offset int) ([]Issue, error) {
	rows, err := s.db.Query(`
		SELECT id, repo_id, number, title, body, state, state_reason, user_id, assignee_ids, label_ids, milestone_id,
			locked, comments_count, is_pull_request, created_at, updated_at, closed_at
		FROM github_issues
		WHERE is_pull_request = 0
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var issues []Issue
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

		issues = append(issues, issue)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return issues, nil
}

// ListAllPullRequests retrieves pull requests across all repositories for admin view
func (s *GitHubStore) ListAllPullRequests(limit, offset int) ([]PullRequest, error) {
	rows, err := s.db.Query(`
		SELECT pr.issue_id, pr.head_repo_id, pr.head_ref, pr.base_repo_id, pr.base_ref, pr.merged, pr.mergeable, pr.rebaseable,
			pr.merge_commit_sha, pr.merged_at, pr.merged_by_id, pr.draft, pr.review_comments_count, pr.commits_count,
			pr.additions, pr.deletions, pr.changed_files
		FROM github_pull_requests pr
		JOIN github_issues i ON pr.issue_id = i.id
		ORDER BY i.created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []PullRequest
	for rows.Next() {
		var pr PullRequest
		var mergeCommitSHA sql.NullString
		var mergedAt sql.NullTime
		var mergedByID sql.NullInt64

		err := rows.Scan(
			&pr.IssueID, &pr.HeadRepoID, &pr.HeadRef, &pr.BaseRepoID, &pr.BaseRef, &pr.Merged, &pr.Mergeable, &pr.Rebaseable,
			&mergeCommitSHA, &mergedAt, &mergedByID, &pr.Draft, &pr.ReviewCommentsCount, &pr.CommitsCount,
			&pr.Additions, &pr.Deletions, &pr.ChangedFiles,
		)
		if err != nil {
			return nil, err
		}

		if mergeCommitSHA.Valid {
			pr.MergeCommitSHA = mergeCommitSHA.String
		}
		if mergedAt.Valid {
			pr.MergedAt = &mergedAt.Time
		}
		if mergedByID.Valid {
			id := mergedByID.Int64
			pr.MergedByID = &id
		}

		prs = append(prs, pr)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return prs, nil
}

// ListAllWebhooks retrieves webhooks across all repositories for admin view
func (s *GitHubStore) ListAllWebhooks(limit, offset int) ([]Webhook, error) {
	rows, err := s.db.Query(`
		SELECT id, repo_id, url, content_type, secret, events, active, created_at, updated_at
		FROM github_webhooks
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []Webhook
	for rows.Next() {
		var webhook Webhook
		var secret sql.NullString

		err := rows.Scan(
			&webhook.ID, &webhook.RepoID, &webhook.URL, &webhook.ContentType,
			&secret, &webhook.Events, &webhook.Active, &webhook.CreatedAt, &webhook.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if secret.Valid {
			webhook.Secret = secret.String
		}

		webhooks = append(webhooks, webhook)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return webhooks, nil
}
