// ABOUTME: Database layer for Discord webhook plugin
// ABOUTME: Manages discord_webhooks and discord_webhook_messages tables

package discord

import (
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"fmt"
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

// generateSnowflake creates a Discord-like snowflake ID (simplified)
func generateSnowflake() string {
	var n uint64
	binary.Read(rand.Reader, binary.BigEndian, &n)
	return fmt.Sprintf("%d", n)
}

// GetOrCreateWebhook retrieves or creates a webhook (auto-accept pattern)
func (s *DiscordStore) GetOrCreateWebhook(id, token string) (*Webhook, error) {
	// Try to get existing
	webhook, err := s.GetWebhook(id, token)
	if err == nil {
		return webhook, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// Create new webhook
	webhook = &Webhook{
		ID:        id,
		Token:     token,
		Type:      1,
		Name:      "Incoming Webhook",
		ChannelID: generateSnowflake(),
		GuildID:   generateSnowflake(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	query := `INSERT INTO discord_webhooks (id, token, type, name, channel_id, guild_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err = s.db.Exec(query, webhook.ID, webhook.Token, webhook.Type, webhook.Name, webhook.ChannelID, webhook.GuildID, webhook.CreatedAt, webhook.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return webhook, nil
}

func (s *DiscordStore) GetWebhook(id, token string) (*Webhook, error) {
	query := `SELECT id, token, type, name, avatar, channel_id, guild_id, application_id, created_at, updated_at, deleted_at
		FROM discord_webhooks WHERE id = ? AND token = ? AND deleted_at IS NULL`

	webhook := &Webhook{}
	var deletedAt sql.NullTime
	err := s.db.QueryRow(query, id, token).Scan(
		&webhook.ID, &webhook.Token, &webhook.Type, &webhook.Name, &webhook.Avatar,
		&webhook.ChannelID, &webhook.GuildID, &webhook.ApplicationID,
		&webhook.CreatedAt, &webhook.UpdatedAt, &deletedAt,
	)
	if err != nil {
		return nil, err
	}
	if deletedAt.Valid {
		webhook.DeletedAt = &deletedAt.Time
	}
	return webhook, nil
}

func (s *DiscordStore) UpdateWebhook(webhook *Webhook) error {
	query := `UPDATE discord_webhooks SET name = ?, avatar = ?, updated_at = ? WHERE id = ? AND token = ?`
	webhook.UpdatedAt = time.Now()
	_, err := s.db.Exec(query, webhook.Name, webhook.Avatar, webhook.UpdatedAt, webhook.ID, webhook.Token)
	return err
}

func (s *DiscordStore) DeleteWebhook(id, token string) error {
	query := `UPDATE discord_webhooks SET deleted_at = ? WHERE id = ? AND token = ?`
	_, err := s.db.Exec(query, time.Now(), id, token)
	return err
}
