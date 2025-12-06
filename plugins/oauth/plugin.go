// ABOUTME: OAuth plugin for ISH mock OAuth provider.
// ABOUTME: Provides OAuth flow simulation for testing token flows.

package oauth

import (
	"context"
	"database/sql"

	"github.com/2389/ish/internal/store"
	"github.com/2389/ish/plugins/core"
	"github.com/go-chi/chi/v5"
)

func init() {
	core.Register(&OAuthPlugin{})
}

type OAuthPlugin struct {
	store *store.Store
}

func (p *OAuthPlugin) Name() string {
	return "oauth"
}

func (p *OAuthPlugin) Health() core.HealthStatus {
	return core.HealthStatus{
		Status:  "healthy",
		Message: "OAuth plugin operational",
	}
}

func (p *OAuthPlugin) RegisterRoutes(r chi.Router) {
	// OAuth plugin has no API routes, only auth routes
}

func (p *OAuthPlugin) RegisterAuth(r chi.Router) {
	// Register OAuth flow endpoints for all plugins
	r.Get("/oauth/{plugin}/authorize", p.handleAuthorize)
	r.Post("/oauth/{plugin}/token", p.handleToken)
	r.Post("/oauth/{plugin}/revoke", p.handleRevoke)
}

func (p *OAuthPlugin) Schema() core.PluginSchema {
	return getOAuthSchema()
}

func (p *OAuthPlugin) Seed(ctx context.Context, size string) (core.SeedData, error) {
	return core.SeedData{
		Summary: "OAuth tokens are created on-demand during authentication",
		Records: map[string]int{},
	}, nil
}

func (p *OAuthPlugin) ValidateToken(token string) bool {
	if p.store == nil {
		return false
	}

	t, err := p.store.GetToken(token)
	if err != nil {
		// Token doesn't exist
		if err == sql.ErrNoRows {
			return false
		}
		// Database error - log it but return false for safety
		return false
	}

	// Token exists but is revoked
	if t.Revoked {
		return false
	}

	return true
}

// SetStore allows injecting the store after construction
func (p *OAuthPlugin) SetStore(s *store.Store) {
	p.store = s
}
