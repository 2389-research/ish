// ABOUTME: OAuth plugin for ISH mock OAuth provider.
// ABOUTME: Provides OAuth flow simulation for testing token flows.

package oauth

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/2389/ish/plugins/core"
	"github.com/go-chi/chi/v5"
)

func init() {
	core.Register(&OAuthPlugin{})
}

type OAuthPlugin struct {
	store *OAuthStore
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

	// Check if token is expired
	if !t.ExpiresAt.IsZero() && t.ExpiresAt.Before(time.Now()) {
		return false
	}

	return true
}

// SetDB initializes the OAuth plugin's database store
func (p *OAuthPlugin) SetDB(db *sql.DB) error {
	store, err := NewOAuthStore(db)
	if err != nil {
		return err
	}
	p.store = store
	return nil
}

// ListResources implements core.DataProvider to expose data to admin UI
func (p *OAuthPlugin) ListResources(ctx context.Context, slug string, opts core.ListOptions) ([]map[string]interface{}, error) {
	switch slug {
	case "tokens":
		tokens, err := p.store.ListAllTokens(opts.Limit, opts.Offset)
		if err != nil {
			return nil, err
		}
		return convertTokensToMaps(tokens), nil
	default:
		return nil, fmt.Errorf("unknown resource: %s", slug)
	}
}

// GetResource implements core.DataProvider to fetch individual resources
func (p *OAuthPlugin) GetResource(ctx context.Context, slug string, id string) (map[string]interface{}, error) {
	switch slug {
	case "tokens":
		token, err := p.store.GetToken(id)
		if err != nil {
			return nil, err
		}
		return convertTokenToMap(token), nil
	default:
		return nil, fmt.Errorf("unknown resource: %s", slug)
	}
}

// convertTokensToMaps converts token structs to maps for admin UI
func convertTokensToMaps(tokens []*OAuthToken) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(tokens))
	for _, token := range tokens {
		result = append(result, convertTokenToMap(token))
	}
	return result
}

// convertTokenToMap converts a single token to map
func convertTokenToMap(token *OAuthToken) map[string]interface{} {
	m := map[string]interface{}{
		"token":       token.Token,
		"plugin_name": token.PluginName,
		"user_id":     token.UserID,
		"scopes":      token.Scopes,
		"revoked":     token.Revoked,
		"created_at":  token.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	// Handle nullable expires_at field
	if !token.ExpiresAt.IsZero() {
		m["expires_at"] = token.ExpiresAt.Format("2006-01-02T15:04:05Z")
	} else {
		m["expires_at"] = ""
	}

	return m
}
