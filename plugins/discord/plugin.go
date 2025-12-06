// ABOUTME: Discord webhook API plugin for ISH
// ABOUTME: Simulates Discord webhook API v10

package discord

import (
	"context"
	"database/sql"

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
