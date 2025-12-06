// ABOUTME: Twilio SMS and Voice API plugin for ISH
// ABOUTME: Simulates Twilio Messaging and Voice APIs with webhook callbacks

package twilio

import (
	"context"
	"database/sql"

	"github.com/2389/ish/plugins/core"
	"github.com/go-chi/chi/v5"
)

func init() {
	core.Register(&TwilioPlugin{})
}

type TwilioPlugin struct {
	store *TwilioStore
}

func (p *TwilioPlugin) Name() string {
	return "twilio"
}

func (p *TwilioPlugin) Health() core.HealthStatus {
	return core.HealthStatus{
		Status:  "healthy",
		Message: "Twilio plugin operational",
	}
}

func (p *TwilioPlugin) RegisterRoutes(r chi.Router) {
	// Routes will be added in Task 3
}

func (p *TwilioPlugin) RegisterAuth(r chi.Router) {
	// Twilio uses HTTP Basic Auth, handled per-request
}

func (p *TwilioPlugin) Schema() core.PluginSchema {
	// Schema will be added in Task 5
	return core.PluginSchema{}
}

func (p *TwilioPlugin) Seed(ctx context.Context, size string) (core.SeedData, error) {
	// Seeding will be added in Task 6
	return core.SeedData{}, nil
}

func (p *TwilioPlugin) ValidateToken(token string) bool {
	// Token validation will be added in Task 2
	return false
}

func (p *TwilioPlugin) SetDB(db *sql.DB) error {
	store, err := NewTwilioStore(db)
	if err != nil {
		return err
	}
	p.store = store
	return nil
}
