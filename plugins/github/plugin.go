// ABOUTME: GitHub REST API v3 emulator plugin for ISH
// ABOUTME: Simulates repos, issues, PRs, reviews, comments, webhooks

package github

import (
	"context"
	"database/sql"

	"github.com/2389/ish/plugins/core"
	"github.com/go-chi/chi/v5"
)

func init() {
	core.Register(&GitHubPlugin{})
}

type GitHubPlugin struct {
	store *GitHubStore
}

func (p *GitHubPlugin) Name() string {
	return "github"
}

func (p *GitHubPlugin) Health() core.HealthStatus {
	return core.HealthStatus{
		Status:  "healthy",
		Message: "GitHub plugin operational",
	}
}

func (p *GitHubPlugin) RegisterRoutes(r chi.Router) {
	// Will implement in later tasks
}

func (p *GitHubPlugin) RegisterAuth(r chi.Router) {
	// GitHub doesn't use OAuth through ISH
}

func (p *GitHubPlugin) Schema() core.PluginSchema {
	// Will implement in later tasks
	return core.PluginSchema{}
}

func (p *GitHubPlugin) Seed(ctx context.Context, size string) (core.SeedData, error) {
	// Will implement in later tasks
	return core.SeedData{
		Summary: "GitHub plugin seeding not implemented",
		Records: map[string]int{},
	}, nil
}

func (p *GitHubPlugin) ValidateToken(token string) bool {
	user, err := p.store.ValidateToken(token)
	return err == nil && user != nil
}

func (p *GitHubPlugin) SetDB(db *sql.DB) error {
	store, err := NewGitHubStore(db)
	if err != nil {
		return err
	}
	p.store = store
	return nil
}
