// ABOUTME: Database operations and schema for SendGrid plugin
// ABOUTME: Handles accounts, API keys, messages, and suppressions

package sendgrid

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
)

type Account struct {
	ID        int64
	Email     string
	Name      string
	CreatedAt time.Time
}

type APIKey struct {
	ID         int64
	AccountID  int64
	Key        string
	Name       string
	Scopes     string
	LastUsedAt *time.Time
	CreatedAt  time.Time
}

type Message struct {
	ID          string
	AccountID   int64
	FromEmail   string
	FromName    string
	ToEmail     string
	ToName      string
	Subject     string
	TextContent string
	HTMLContent string
	Status      string
	SentAt      time.Time
}

type Suppression struct {
	ID        int64
	AccountID int64
	Email     string
	Type      string // "bounce", "block", "spam_report"
	Reason    string
	CreatedAt time.Time
}

type SendGridStore struct {
	db *sql.DB
}

func NewSendGridStore(db *sql.DB) (*SendGridStore, error) {
	store := &SendGridStore{db: db}
	if err := store.createTables(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *SendGridStore) createTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS sendgrid_accounts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS sendgrid_api_keys (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		account_id INTEGER NOT NULL,
		key TEXT NOT NULL UNIQUE,
		name TEXT NOT NULL,
		scopes TEXT NOT NULL DEFAULT 'mail.send',
		last_used_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (account_id) REFERENCES sendgrid_accounts(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_sendgrid_api_keys_key ON sendgrid_api_keys(key);
	CREATE INDEX IF NOT EXISTS idx_sendgrid_api_keys_account ON sendgrid_api_keys(account_id);

	CREATE TABLE IF NOT EXISTS sendgrid_messages (
		id TEXT PRIMARY KEY,
		account_id INTEGER NOT NULL,
		from_email TEXT NOT NULL,
		from_name TEXT,
		to_email TEXT NOT NULL,
		to_name TEXT,
		subject TEXT,
		text_content TEXT,
		html_content TEXT,
		status TEXT NOT NULL DEFAULT 'delivered',
		sent_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (account_id) REFERENCES sendgrid_accounts(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_sendgrid_messages_account ON sendgrid_messages(account_id);
	CREATE INDEX IF NOT EXISTS idx_sendgrid_messages_to_email ON sendgrid_messages(to_email);
	CREATE INDEX IF NOT EXISTS idx_sendgrid_messages_sent_at ON sendgrid_messages(sent_at);

	CREATE TABLE IF NOT EXISTS sendgrid_suppressions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		account_id INTEGER NOT NULL,
		email TEXT NOT NULL,
		type TEXT NOT NULL CHECK(type IN ('bounce', 'block', 'spam_report')),
		reason TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (account_id) REFERENCES sendgrid_accounts(id) ON DELETE CASCADE,
		UNIQUE(account_id, email, type)
	);

	CREATE INDEX IF NOT EXISTS idx_sendgrid_suppressions_account ON sendgrid_suppressions(account_id);
	CREATE INDEX IF NOT EXISTS idx_sendgrid_suppressions_email ON sendgrid_suppressions(email);
	CREATE INDEX IF NOT EXISTS idx_sendgrid_suppressions_type ON sendgrid_suppressions(type);
	`

	_, err := s.db.Exec(schema)
	return err
}

// ValidateAPIKey validates an API key and returns the associated account
func (s *SendGridStore) ValidateAPIKey(apiKey string) (*Account, error) {
	var account Account
	var accountID int64

	err := s.db.QueryRow(`
		SELECT account_id FROM sendgrid_api_keys WHERE key = ?
	`, apiKey).Scan(&accountID)

	if err != nil {
		return nil, fmt.Errorf("invalid API key")
	}

	// Get account details
	err = s.db.QueryRow(`
		SELECT id, email, name, created_at FROM sendgrid_accounts WHERE id = ?
	`, accountID).Scan(&account.ID, &account.Email, &account.Name, &account.CreatedAt)

	if err != nil {
		return nil, err
	}

	// Update last_used_at (best-effort, don't fail request if this fails)
	_, err = s.db.Exec(`
		UPDATE sendgrid_api_keys SET last_used_at = CURRENT_TIMESTAMP WHERE key = ?
	`, apiKey)
	if err != nil {
		// Log but don't fail the request
		fmt.Fprintf(os.Stderr, "Warning: failed to update API key last_used_at: %v\n", err)
	}

	return &account, nil
}

// CreateAccount creates a new SendGrid account
func (s *SendGridStore) CreateAccount(email, name string) (*Account, error) {
	result, err := s.db.Exec(`
		INSERT INTO sendgrid_accounts (email, name)
		VALUES (?, ?)
	`, email, name)

	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return s.GetAccount(id)
}

// GetAccount retrieves an account by ID
func (s *SendGridStore) GetAccount(accountID int64) (*Account, error) {
	var account Account
	err := s.db.QueryRow(`
		SELECT id, email, name, created_at
		FROM sendgrid_accounts
		WHERE id = ?
	`, accountID).Scan(&account.ID, &account.Email, &account.Name, &account.CreatedAt)

	if err != nil {
		return nil, err
	}

	return &account, nil
}

// CreateAPIKey creates a new API key for an account
func (s *SendGridStore) CreateAPIKey(accountID int64, name, scopes string) (*APIKey, error) {
	// Generate a SendGrid-style API key (SG. prefix)
	key := fmt.Sprintf("SG.%s", uuid.New().String())

	result, err := s.db.Exec(`
		INSERT INTO sendgrid_api_keys (account_id, key, name, scopes)
		VALUES (?, ?, ?, ?)
	`, accountID, key, name, scopes)

	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return s.GetAPIKey(id)
}

// GetAPIKey retrieves an API key by ID
func (s *SendGridStore) GetAPIKey(keyID int64) (*APIKey, error) {
	var apiKey APIKey
	err := s.db.QueryRow(`
		SELECT id, account_id, key, name, scopes, last_used_at, created_at
		FROM sendgrid_api_keys
		WHERE id = ?
	`, keyID).Scan(&apiKey.ID, &apiKey.AccountID, &apiKey.Key, &apiKey.Name,
		&apiKey.Scopes, &apiKey.LastUsedAt, &apiKey.CreatedAt)

	if err != nil {
		return nil, err
	}

	return &apiKey, nil
}

// CreateMessage creates a new message record
func (s *SendGridStore) CreateMessage(accountID int64, fromEmail, fromName, toEmail, toName, subject, textContent, htmlContent string) (*Message, error) {
	messageID := uuid.New().String()

	_, err := s.db.Exec(`
		INSERT INTO sendgrid_messages (id, account_id, from_email, from_name, to_email, to_name, subject, text_content, html_content, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'delivered')
	`, messageID, accountID, fromEmail, fromName, toEmail, toName, subject, textContent, htmlContent)

	if err != nil {
		return nil, err
	}

	return s.GetMessage(messageID)
}

// GetMessage retrieves a message by ID
func (s *SendGridStore) GetMessage(messageID string) (*Message, error) {
	var msg Message
	err := s.db.QueryRow(`
		SELECT id, account_id, from_email, from_name, to_email, to_name, subject, text_content, html_content, status, sent_at
		FROM sendgrid_messages
		WHERE id = ?
	`, messageID).Scan(&msg.ID, &msg.AccountID, &msg.FromEmail, &msg.FromName,
		&msg.ToEmail, &msg.ToName, &msg.Subject, &msg.TextContent, &msg.HTMLContent,
		&msg.Status, &msg.SentAt)

	if err != nil {
		return nil, err
	}

	return &msg, nil
}

// ListMessages retrieves messages for an account
func (s *SendGridStore) ListMessages(accountID int64, limit, offset int) ([]*Message, error) {
	rows, err := s.db.Query(`
		SELECT id, account_id, from_email, from_name, to_email, to_name, subject, text_content, html_content, status, sent_at
		FROM sendgrid_messages
		WHERE account_id = ?
		ORDER BY sent_at DESC
		LIMIT ? OFFSET ?
	`, accountID, limit, offset)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		var msg Message
		err := rows.Scan(&msg.ID, &msg.AccountID, &msg.FromEmail, &msg.FromName,
			&msg.ToEmail, &msg.ToName, &msg.Subject, &msg.TextContent, &msg.HTMLContent,
			&msg.Status, &msg.SentAt)
		if err != nil {
			return nil, err
		}
		messages = append(messages, &msg)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

// CreateSuppression creates a new suppression entry
func (s *SendGridStore) CreateSuppression(accountID int64, email, suppressionType, reason string) (*Suppression, error) {
	result, err := s.db.Exec(`
		INSERT INTO sendgrid_suppressions (account_id, email, type, reason)
		VALUES (?, ?, ?, ?)
	`, accountID, email, suppressionType, reason)

	if err != nil {
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return s.GetSuppression(id)
}

// GetSuppression retrieves a suppression by ID
func (s *SendGridStore) GetSuppression(suppressionID int64) (*Suppression, error) {
	var supp Suppression
	err := s.db.QueryRow(`
		SELECT id, account_id, email, type, reason, created_at
		FROM sendgrid_suppressions
		WHERE id = ?
	`, suppressionID).Scan(&supp.ID, &supp.AccountID, &supp.Email, &supp.Type,
		&supp.Reason, &supp.CreatedAt)

	if err != nil {
		return nil, err
	}

	return &supp, nil
}

// ListSuppressions retrieves suppressions for an account by type
func (s *SendGridStore) ListSuppressions(accountID int64, suppressionType string) ([]*Suppression, error) {
	rows, err := s.db.Query(`
		SELECT id, account_id, email, type, reason, created_at
		FROM sendgrid_suppressions
		WHERE account_id = ? AND type = ?
		ORDER BY created_at DESC
	`, accountID, suppressionType)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suppressions []*Suppression
	for rows.Next() {
		var supp Suppression
		err := rows.Scan(&supp.ID, &supp.AccountID, &supp.Email, &supp.Type,
			&supp.Reason, &supp.CreatedAt)
		if err != nil {
			return nil, err
		}
		suppressions = append(suppressions, &supp)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return suppressions, nil
}

// DeleteSuppression deletes a suppression by email and type
func (s *SendGridStore) DeleteSuppression(accountID int64, email, suppressionType string) error {
	_, err := s.db.Exec(`
		DELETE FROM sendgrid_suppressions
		WHERE account_id = ? AND email = ? AND type = ?
	`, accountID, email, suppressionType)

	return err
}

// GetProductionAPIKey retrieves the production API key for an account
func (s *SendGridStore) GetProductionAPIKey(accountID int64) (string, error) {
	var apiKey string
	err := s.db.QueryRow(`
		SELECT key FROM sendgrid_api_keys
		WHERE account_id = ? AND name LIKE 'Production%'
		LIMIT 1
	`, accountID).Scan(&apiKey)

	if err != nil {
		return "", err
	}

	return apiKey, nil
}
