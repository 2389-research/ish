// ABOUTME: Database layer for Twilio plugin
// ABOUTME: Manages accounts, phone numbers, messages, calls, and webhook queue

package twilio

import (
	"database/sql"
)

type TwilioStore struct {
	db *sql.DB
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
