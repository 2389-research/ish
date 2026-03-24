// ABOUTME: Anthropic Messages API plugin for ISH
// ABOUTME: Simulates Anthropic Messages API with scenario-based responses

package anthropic

import (
	"context"
	"database/sql"

	"github.com/2389/ish/plugins/core"
	"github.com/go-chi/chi/v5"
)

func init() {
	core.Register(&AnthropicPlugin{})
}

type AnthropicPlugin struct {
	store *AnthropicStore
}

func (p *AnthropicPlugin) Name() string {
	return "anthropic"
}

func (p *AnthropicPlugin) Health() core.HealthStatus {
	return core.HealthStatus{
		Status:  "healthy",
		Message: "Anthropic Messages API plugin operational",
	}
}

func (p *AnthropicPlugin) RegisterRoutes(r chi.Router) {
	r.Post("/v1/messages", p.handleMessages)
	r.Post("/v1/scenarios", p.createScenario)
	r.Get("/v1/scenarios", p.listScenarios)
	r.Delete("/v1/scenarios/{id}", p.deleteScenarioHandler)
}

func (p *AnthropicPlugin) RegisterAuth(r chi.Router) {
	// Anthropic uses API keys, no OAuth flow needed
}

func (p *AnthropicPlugin) ValidateToken(token string) bool {
	return true
}

func (p *AnthropicPlugin) SetDB(db *sql.DB) error {
	store, err := NewAnthropicStore(db)
	if err != nil {
		return err
	}
	p.store = store
	return nil
}

// Reset implements core.ResettablePlugin to clear all plugin data
func (p *AnthropicPlugin) Reset(ctx context.Context) error {
	return p.store.Reset()
}
