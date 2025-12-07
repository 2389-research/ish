// ABOUTME: Discord webhook API plugin for ISH
// ABOUTME: Simulates Discord webhook API v10

package discord

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/2389/ish/plugins/core"
	"github.com/go-chi/chi/v5"
)

func init() {
	core.Register(&DiscordPlugin{})
}

type DiscordPlugin struct {
	store *DiscordStore
}

func (p *DiscordPlugin) Name() string {
	return "discord"
}

func (p *DiscordPlugin) Health() core.HealthStatus {
	return core.HealthStatus{
		Status:  "healthy",
		Message: "Discord webhook plugin operational",
	}
}

func (p *DiscordPlugin) RegisterRoutes(r chi.Router) {
	r.Route("/api/webhooks/{webhookID}/{webhookToken}", func(r chi.Router) {
		// Webhook endpoints
		r.Post("/", p.executeWebhook)
		r.Get("/", p.getWebhook)
		r.Patch("/", p.modifyWebhook)
		r.Delete("/", p.deleteWebhook)

		// Message endpoints
		r.Get("/messages/{messageID}", p.getWebhookMessage)
		r.Patch("/messages/{messageID}", p.editWebhookMessage)
		r.Delete("/messages/{messageID}", p.deleteWebhookMessage)
	})
}

func (p *DiscordPlugin) RegisterAuth(r chi.Router) {
	// Discord webhooks don't use OAuth
}

func (p *DiscordPlugin) Seed(ctx context.Context, size string) (core.SeedData, error) {
	return core.SeedData{
		Summary: "Discord plugin seeding not implemented",
		Records: map[string]int{},
	}, nil
}

func (p *DiscordPlugin) ValidateToken(token string) bool {
	// Discord uses webhook tokens in URL, validation happens in handlers
	return true
}

func (p *DiscordPlugin) SetDB(db *sql.DB) error {
	store, err := NewDiscordStore(db)
	if err != nil {
		return err
	}
	p.store = store
	return nil
}

// ListResources implements core.DataProvider to expose data to admin UI
func (p *DiscordPlugin) ListResources(ctx context.Context, slug string, opts core.ListOptions) ([]map[string]interface{}, error) {
	switch slug {
	case "webhooks":
		webhooks, err := p.store.ListAllWebhooks(opts.Limit, opts.Offset)
		if err != nil {
			return nil, err
		}
		return convertWebhooksToMaps(webhooks), nil
	case "messages":
		messages, err := p.store.ListAllMessages(opts.Limit, opts.Offset)
		if err != nil {
			return nil, err
		}
		return convertMessagesToMaps(messages), nil
	default:
		return nil, fmt.Errorf("unknown resource: %s", slug)
	}
}

// GetResource implements core.DataProvider to fetch individual resources
func (p *DiscordPlugin) GetResource(ctx context.Context, slug string, id string) (map[string]interface{}, error) {
	switch slug {
	case "webhooks":
		// Webhooks need both ID and token - not supported via this interface
		return nil, fmt.Errorf("get webhook by ID requires token parameter")
	case "messages":
		// Messages need both webhook_id and message_id - not supported via this interface
		return nil, fmt.Errorf("get message by ID requires webhook_id parameter")
	default:
		return nil, fmt.Errorf("unknown resource: %s", slug)
	}
}

// convertWebhooksToMaps converts webhook structs to maps for admin UI
func convertWebhooksToMaps(webhooks []*Webhook) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(webhooks))
	for _, webhook := range webhooks {
		result = append(result, convertWebhookToMap(webhook))
	}
	return result
}

// convertWebhookToMap converts a single webhook to map
func convertWebhookToMap(webhook *Webhook) map[string]interface{} {
	m := map[string]interface{}{
		"id":         webhook.ID,
		"token":      webhook.Token,
		"type":       fmt.Sprintf("%d", webhook.Type),
		"name":       webhook.Name,
		"channel_id": webhook.ChannelID,
		"guild_id":   webhook.GuildID,
		"created_at": webhook.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if webhook.Avatar != "" {
		m["avatar"] = webhook.Avatar
	}
	if webhook.ApplicationID != "" {
		m["application_id"] = webhook.ApplicationID
	}
	return m
}

// convertMessagesToMaps converts message structs to maps for admin UI
func convertMessagesToMaps(messages []*WebhookMessage) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(messages))
	for _, msg := range messages {
		result = append(result, convertMessageToMap(msg))
	}
	return result
}

// convertMessageToMap converts a single message to map
func convertMessageToMap(msg *WebhookMessage) map[string]interface{} {
	m := map[string]interface{}{
		"id":         msg.ID,
		"webhook_id": msg.WebhookID,
		"content":    msg.Content,
		"username":   msg.Username,
		"created_at": msg.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if msg.AvatarURL != "" {
		m["avatar_url"] = msg.AvatarURL
	}
	if msg.Embeds != "" {
		m["embeds"] = msg.Embeds
	}
	if msg.Components != "" {
		m["components"] = msg.Components
	}
	if msg.Attachments != "" {
		m["attachments"] = msg.Attachments
	}
	if msg.ThreadID != "" {
		m["thread_id"] = msg.ThreadID
	}
	if msg.EditedAt != nil {
		m["edited_at"] = msg.EditedAt.Format("2006-01-02T15:04:05Z")
	}
	return m
}
