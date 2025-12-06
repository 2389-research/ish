// ABOUTME: Discord webhook API plugin for ISH
// ABOUTME: Simulates Discord webhook API v10

package discord

import (
	"context"
	"database/sql"
	"net/http"

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
	r.Route("/api/webhooks", func(r chi.Router) {
		// Endpoints will be added in next tasks
	})
}

func (p *DiscordPlugin) RegisterAuth(r chi.Router) {
	// Discord webhooks don't use OAuth
}

func (p *DiscordPlugin) Schema() core.PluginSchema {
	// Schema will be added in later task
	return core.PluginSchema{Resources: []core.ResourceSchema{}}
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
