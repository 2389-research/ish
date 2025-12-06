// ABOUTME: GitHub REST API v3 emulator plugin for ISH
// ABOUTME: Simulates repos, issues, PRs, reviews, comments, webhooks

package github

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/2389/ish/plugins/core"
	"github.com/go-chi/chi/v5"
)

func init() {
	core.Register(&GitHubPlugin{})
}

type contextKey string

const userContextKey contextKey = "github_user"

// extractToken extracts GitHub token from Authorization header
// Supports both "Bearer <token>" and "token <token>" formats
func extractToken(r *http.Request) (string, bool) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", false
	}

	// Try "Bearer <token>" format
	if len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:], true
	}

	// Try "token <token>" format (GitHub's alternative)
	if len(auth) > 6 && auth[:6] == "token " {
		return auth[6:], true
	}

	return "", false
}

// requireAuth middleware validates GitHub token and adds user to context
func (p *GitHubPlugin) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := extractToken(r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "requires authentication")
			return
		}

		user, err := p.store.ValidateToken(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "bad credentials")
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// writeError writes a GitHub-style JSON error response
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"message": "%s"}`, message)
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
	if p.store == nil {
		return false
	}
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
