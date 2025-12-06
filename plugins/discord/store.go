// ABOUTME: Database layer for Discord webhook plugin
// ABOUTME: Manages discord_webhooks and discord_webhook_messages tables

package discord

import (
	"database/sql"
	"time"
)

type DiscordStore struct {
	db *sql.DB
}

func NewDiscordStore(db *sql.DB) (*DiscordStore, error) {
	store := &DiscordStore{db: db}
	if err := store.initTables(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *DiscordStore) initTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS discord_webhooks (
			id TEXT PRIMARY KEY,
			token TEXT NOT NULL,
			type INTEGER DEFAULT 1,
			name TEXT,
			avatar TEXT,
			channel_id TEXT,
			guild_id TEXT,
			application_id TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			deleted_at TIMESTAMP,
			UNIQUE(id, token)
		)`,

		`CREATE TABLE IF NOT EXISTS discord_webhook_messages (
			id TEXT PRIMARY KEY,
			webhook_id TEXT NOT NULL,
			content TEXT,
			username TEXT,
			avatar_url TEXT,
			embeds TEXT,
			components TEXT,
			attachments TEXT,
			thread_id TEXT,
			flags INTEGER,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			edited_at TIMESTAMP,
			deleted_at TIMESTAMP,
			FOREIGN KEY (webhook_id) REFERENCES discord_webhooks(id) ON DELETE CASCADE
		)`,

		`CREATE INDEX IF NOT EXISTS idx_webhook_messages_webhook_id ON discord_webhook_messages(webhook_id)`,
		`CREATE INDEX IF NOT EXISTS idx_webhook_messages_created_at ON discord_webhook_messages(created_at DESC)`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

type Webhook struct {
	ID            string
	Token         string
	Type          int
	Name          string
	Avatar        string
	ChannelID     string
	GuildID       string
	ApplicationID string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     *time.Time
}

type WebhookMessage struct {
	ID          string
	WebhookID   string
	Content     string
	Username    string
	AvatarURL   string
	Embeds      string
	Components  string
	Attachments string
	ThreadID    string
	Flags       int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	EditedAt    *time.Time
	DeletedAt   *time.Time
}
