// ABOUTME: Google plugin for ISH.
// ABOUTME: Provides Gmail, Calendar, People, and Tasks APIs.

package google

import (
	"context"

	"github.com/2389/ish/internal/store"
	"github.com/2389/ish/plugins/core"
	"github.com/go-chi/chi/v5"
)

func init() {
	core.Register(&GooglePlugin{})
}

type GooglePlugin struct {
	store *store.Store
}

func (p *GooglePlugin) Name() string {
	return "google"
}

func (p *GooglePlugin) Health() core.HealthStatus {
	return core.HealthStatus{
		Status:  "healthy",
		Message: "Google plugin operational",
	}
}

func (p *GooglePlugin) RegisterRoutes(r chi.Router) {
	p.registerGmailRoutes(r)
	// Calendar, People, Tasks routes will be added in Task 6
}

func (p *GooglePlugin) RegisterAuth(r chi.Router) {
	// OAuth endpoints will be added later
}

func (p *GooglePlugin) Schema() core.PluginSchema {
	return getGoogleSchema()
}

func (p *GooglePlugin) Seed(ctx context.Context, size string) (core.SeedData, error) {
	// Seeding will be implemented later
	return core.SeedData{
		Summary: "Not yet implemented",
		Records: map[string]int{},
	}, nil
}

func (p *GooglePlugin) ValidateToken(token string) bool {
	// Token validation will be implemented later
	return true
}

// SetStore allows injecting the store after construction
func (p *GooglePlugin) SetStore(s *store.Store) {
	p.store = s
}
