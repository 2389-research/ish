// ABOUTME: Database layer for Twilio plugin
// ABOUTME: Manages accounts, phone numbers, messages, calls, and webhook queue

package twilio

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"time"
)

type TwilioStore struct {
	db *sql.DB
}

type Account struct {
	AccountSid   string
	AuthToken    string
	FriendlyName string
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func NewTwilioStore(db *sql.DB) (*TwilioStore, error) {
	store := &TwilioStore{db: db}
	if err := store.initTables(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *TwilioStore) initTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS twilio_accounts (
			account_sid TEXT PRIMARY KEY,
			auth_token TEXT NOT NULL,
			friendly_name TEXT,
			status TEXT DEFAULT 'active',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS twilio_phone_numbers (
			sid TEXT PRIMARY KEY,
			account_sid TEXT NOT NULL,
			phone_number TEXT NOT NULL,
			friendly_name TEXT,
			voice_url TEXT,
			voice_method TEXT DEFAULT 'POST',
			sms_url TEXT,
			sms_method TEXT DEFAULT 'POST',
			status_callback TEXT,
			status_callback_method TEXT DEFAULT 'POST',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (account_sid) REFERENCES twilio_accounts(account_sid)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_phone_numbers_account ON twilio_phone_numbers(account_sid)`,

		`CREATE TABLE IF NOT EXISTS twilio_messages (
			sid TEXT PRIMARY KEY,
			account_sid TEXT NOT NULL,
			from_number TEXT NOT NULL,
			to_number TEXT NOT NULL,
			body TEXT,
			status TEXT DEFAULT 'queued',
			direction TEXT,
			date_created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			date_sent TIMESTAMP,
			date_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			num_segments INTEGER DEFAULT 1,
			price REAL,
			price_unit TEXT DEFAULT 'USD',
			FOREIGN KEY (account_sid) REFERENCES twilio_accounts(account_sid)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_account ON twilio_messages(account_sid)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_status ON twilio_messages(status)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_date ON twilio_messages(date_created)`,

		`CREATE TABLE IF NOT EXISTS twilio_calls (
			sid TEXT PRIMARY KEY,
			account_sid TEXT NOT NULL,
			from_number TEXT NOT NULL,
			to_number TEXT NOT NULL,
			status TEXT DEFAULT 'initiated',
			direction TEXT,
			duration INTEGER,
			date_created TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			date_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			answered_by TEXT,
			FOREIGN KEY (account_sid) REFERENCES twilio_accounts(account_sid)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_calls_account ON twilio_calls(account_sid)`,
		`CREATE INDEX IF NOT EXISTS idx_calls_status ON twilio_calls(status)`,

		`CREATE TABLE IF NOT EXISTS twilio_webhook_configs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			account_sid TEXT NOT NULL,
			resource_type TEXT NOT NULL,
			event_type TEXT NOT NULL,
			url TEXT NOT NULL,
			method TEXT DEFAULT 'POST',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (account_sid) REFERENCES twilio_accounts(account_sid)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_webhook_configs_account ON twilio_webhook_configs(account_sid)`,

		`CREATE TABLE IF NOT EXISTS twilio_webhook_queue (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			resource_sid TEXT NOT NULL,
			webhook_url TEXT NOT NULL,
			payload TEXT NOT NULL,
			scheduled_at TIMESTAMP NOT NULL,
			delivered_at TIMESTAMP,
			status TEXT DEFAULT 'pending',
			attempts INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_webhook_queue_schedule ON twilio_webhook_queue(scheduled_at, status)`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func generateAuthToken() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *TwilioStore) GetOrCreateAccount(accountSid string) (*Account, error) {
	// Try to get existing account
	var account Account
	var friendlyName sql.NullString
	err := s.db.QueryRow(`
		SELECT account_sid, auth_token, friendly_name, status, created_at, updated_at
		FROM twilio_accounts
		WHERE account_sid = ?
	`, accountSid).Scan(
		&account.AccountSid,
		&account.AuthToken,
		&friendlyName,
		&account.Status,
		&account.CreatedAt,
		&account.UpdatedAt,
	)

	if err == nil {
		if friendlyName.Valid {
			account.FriendlyName = friendlyName.String
		}
		return &account, nil
	}

	// Account doesn't exist, create it
	authToken, err := generateAuthToken()
	if err != nil {
		return nil, err
	}

	_, err = s.db.Exec(`
		INSERT INTO twilio_accounts (account_sid, auth_token)
		VALUES (?, ?)
	`, accountSid, authToken)
	if err != nil {
		return nil, err
	}

	// Fetch the newly created account
	err = s.db.QueryRow(`
		SELECT account_sid, auth_token, friendly_name, status, created_at, updated_at
		FROM twilio_accounts
		WHERE account_sid = ?
	`, accountSid).Scan(
		&account.AccountSid,
		&account.AuthToken,
		&friendlyName,
		&account.Status,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if friendlyName.Valid {
		account.FriendlyName = friendlyName.String
	}

	return &account, nil
}

func (s *TwilioStore) ValidateAccount(accountSid, authToken string) bool {
	var storedToken string
	err := s.db.QueryRow(`
		SELECT auth_token FROM twilio_accounts
		WHERE account_sid = ? AND status = 'active'
	`, accountSid).Scan(&storedToken)

	if err != nil {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(storedToken), []byte(authToken)) == 1
}

type Message struct {
	Sid         string
	AccountSid  string
	FromNumber  string
	ToNumber    string
	Body        string
	Status      string
	Direction   string
	DateCreated time.Time
	DateSent    *time.Time
	DateUpdated time.Time
	NumSegments int
	Price       float64
	PriceUnit   string
}

func generateSID(prefix string) (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(bytes), nil
}

func (s *TwilioStore) CreateMessage(accountSid, from, to, body string) (*Message, error) {
	sid, err := generateSID("SM")
	if err != nil {
		return nil, err
	}

	// Calculate segments (160 chars per segment)
	numSegments := (len(body) + 159) / 160
	if numSegments == 0 {
		numSegments = 1
	}

	_, err = s.db.Exec(`
		INSERT INTO twilio_messages (sid, account_sid, from_number, to_number, body, status, direction, num_segments, price, price_unit)
		VALUES (?, ?, ?, ?, ?, 'queued', 'outbound-api', ?, ?, 'USD')
	`, sid, accountSid, from, to, body, numSegments, float64(numSegments)*0.0075)

	if err != nil {
		return nil, err
	}

	return s.GetMessage(sid)
}

func (s *TwilioStore) GetMessage(sid string) (*Message, error) {
	var msg Message
	var dateSent sql.NullTime

	err := s.db.QueryRow(`
		SELECT sid, account_sid, from_number, to_number, body, status, direction,
		       date_created, date_sent, date_updated, num_segments, price, price_unit
		FROM twilio_messages
		WHERE sid = ?
	`, sid).Scan(
		&msg.Sid, &msg.AccountSid, &msg.FromNumber, &msg.ToNumber, &msg.Body,
		&msg.Status, &msg.Direction, &msg.DateCreated, &dateSent, &msg.DateUpdated,
		&msg.NumSegments, &msg.Price, &msg.PriceUnit,
	)

	if err != nil {
		return nil, err
	}

	if dateSent.Valid {
		msg.DateSent = &dateSent.Time
	}

	return &msg, nil
}

func (s *TwilioStore) UpdateMessageStatus(sid, status string) error {
	now := time.Now()
	_, err := s.db.Exec(`
		UPDATE twilio_messages
		SET status = ?, date_updated = ?, date_sent = CASE WHEN ? IN ('sent', 'delivered') AND date_sent IS NULL THEN ? ELSE date_sent END
		WHERE sid = ?
	`, status, now, status, now, sid)
	return err
}

func (s *TwilioStore) ListMessages(accountSid string, limit int) ([]Message, error) {
	rows, err := s.db.Query(`
		SELECT sid, account_sid, from_number, to_number, body, status, direction,
		       date_created, date_sent, date_updated, num_segments, price, price_unit
		FROM twilio_messages
		WHERE account_sid = ?
		ORDER BY date_created DESC
		LIMIT ?
	`, accountSid, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var dateSent sql.NullTime

		err := rows.Scan(
			&msg.Sid, &msg.AccountSid, &msg.FromNumber, &msg.ToNumber, &msg.Body,
			&msg.Status, &msg.Direction, &msg.DateCreated, &dateSent, &msg.DateUpdated,
			&msg.NumSegments, &msg.Price, &msg.PriceUnit,
		)
		if err != nil {
			return nil, err
		}

		if dateSent.Valid {
			msg.DateSent = &dateSent.Time
		}

		messages = append(messages, msg)
	}

	return messages, nil
}
