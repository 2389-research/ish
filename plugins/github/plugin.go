// ABOUTME: GitHub REST API v3 emulator plugin for ISH
// ABOUTME: Simulates repos, issues, PRs, reviews, comments, webhooks

package github

import (
	"context"
	"database/sql"
	"encoding/json"
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

// getUserFromContext safely extracts the authenticated user from request context
// Returns the user and true if found, or nil and false if not present or wrong type
func getUserFromContext(r *http.Request) (*User, bool) {
	val := r.Context().Value(userContextKey)
	if val == nil {
		return nil, false
	}
	user, ok := val.(*User)
	return user, ok
}

// writeError writes a GitHub-style JSON error response
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
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
	// User endpoints
	r.Get("/user", p.requireAuth(p.getAuthenticatedUser))
	r.Patch("/user", p.requireAuth(p.updateAuthenticatedUser))
	r.Get("/users/{username}", p.requireAuth(p.getUser))

	// Repository endpoints
	r.Get("/user/repos", p.requireAuth(p.listAuthenticatedUserRepositories))
	r.Post("/user/repos", p.requireAuth(p.createUserRepository))
	r.Get("/repos/{owner}/{repo}", p.requireAuth(p.getRepository))
	r.Patch("/repos/{owner}/{repo}", p.requireAuth(p.updateRepository))
	r.Delete("/repos/{owner}/{repo}", p.requireAuth(p.deleteRepository))

	// Issue endpoints
	r.Get("/repos/{owner}/{repo}/issues", p.requireAuth(p.listIssues))
	r.Post("/repos/{owner}/{repo}/issues", p.requireAuth(p.createIssue))
	r.Get("/repos/{owner}/{repo}/issues/{number}", p.requireAuth(p.getIssue))
	r.Patch("/repos/{owner}/{repo}/issues/{number}", p.requireAuth(p.updateIssue))

	// Pull Request endpoints
	r.Post("/repos/{owner}/{repo}/pulls", p.requireAuth(p.createPullRequest))
	r.Get("/repos/{owner}/{repo}/pulls", p.requireAuth(p.listPullRequests))
	r.Get("/repos/{owner}/{repo}/pulls/{number}", p.requireAuth(p.getPullRequest))
	r.Put("/repos/{owner}/{repo}/pulls/{number}/merge", p.requireAuth(p.mergePullRequest))

	// Comment endpoints
	r.Post("/repos/{owner}/{repo}/issues/{number}/comments", p.requireAuth(p.createComment))
	r.Get("/repos/{owner}/{repo}/issues/{number}/comments", p.requireAuth(p.listComments))
	r.Patch("/repos/{owner}/{repo}/issues/comments/{comment_id}", p.requireAuth(p.updateComment))
	r.Delete("/repos/{owner}/{repo}/issues/comments/{comment_id}", p.requireAuth(p.deleteComment))

	// Review endpoints
	r.Post("/repos/{owner}/{repo}/pulls/{number}/reviews", p.requireAuth(p.createReview))
	r.Get("/repos/{owner}/{repo}/pulls/{number}/reviews", p.requireAuth(p.listReviews))
	r.Put("/repos/{owner}/{repo}/pulls/{number}/reviews/{id}", p.requireAuth(p.submitReview))
	r.Delete("/repos/{owner}/{repo}/pulls/{number}/reviews/{id}", p.requireAuth(p.dismissReview))

	// Webhook endpoints
	r.Post("/repos/{owner}/{repo}/hooks", p.requireAuth(p.createWebhook))
	r.Get("/repos/{owner}/{repo}/hooks", p.requireAuth(p.listWebhooks))
	r.Get("/repos/{owner}/{repo}/hooks/{id}", p.requireAuth(p.getWebhook))
	r.Patch("/repos/{owner}/{repo}/hooks/{id}", p.requireAuth(p.updateWebhook))
	r.Delete("/repos/{owner}/{repo}/hooks/{id}", p.requireAuth(p.deleteWebhook))
	r.Post("/repos/{owner}/{repo}/hooks/{id}/tests", p.requireAuth(p.testWebhook))
}

// Placeholder handlers for routes not yet implemented
func (p *GitHubPlugin) updateAuthenticatedUser(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}

func (p *GitHubPlugin) getUser(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}

func (p *GitHubPlugin) updateRepository(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}

func (p *GitHubPlugin) deleteRepository(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotImplemented, "not implemented")
}

func (p *GitHubPlugin) RegisterAuth(r chi.Router) {
	// GitHub doesn't use OAuth through ISH
}

func (p *GitHubPlugin) Schema() core.PluginSchema {
	return core.PluginSchema{
		Resources: []core.ResourceSchema{
			{
				Name:        "Users",
				Slug:        "users",
				ListColumns: []string{"id", "login", "name", "type", "repos_count"},
				Fields: []core.FieldSchema{
					{Name: "id", Type: "string", Display: "ID", Required: true, Editable: false},
					{Name: "login", Type: "string", Display: "Login", Required: true, Editable: false},
					{Name: "name", Type: "string", Display: "Name", Required: false, Editable: false},
					{Name: "email", Type: "string", Display: "Email", Required: false, Editable: false},
					{Name: "avatar_url", Type: "string", Display: "Avatar URL", Required: false, Editable: false},
					{Name: "type", Type: "string", Display: "Type", Required: false, Editable: false},
					{Name: "repos_count", Type: "string", Display: "Repositories", Required: false, Editable: false},
					{Name: "created_at", Type: "datetime", Display: "Created", Required: false, Editable: false},
				},
				Actions: []core.ActionSchema{},
			},
			{
				Name:        "Repositories",
				Slug:        "repositories",
				ListColumns: []string{"id", "full_name", "owner", "private", "stars", "issues_count"},
				Fields: []core.FieldSchema{
					{Name: "id", Type: "string", Display: "ID", Required: true, Editable: false},
					{Name: "name", Type: "string", Display: "Name", Required: true, Editable: false},
					{Name: "full_name", Type: "string", Display: "Full Name", Required: true, Editable: false},
					{Name: "owner", Type: "string", Display: "Owner", Required: true, Editable: false},
					{Name: "description", Type: "text", Display: "Description", Required: false, Editable: false},
					{Name: "private", Type: "string", Display: "Private", Required: false, Editable: false},
					{Name: "default_branch", Type: "string", Display: "Default Branch", Required: false, Editable: false},
					{Name: "stars", Type: "string", Display: "Stars", Required: false, Editable: false},
					{Name: "forks", Type: "string", Display: "Forks", Required: false, Editable: false},
					{Name: "issues_count", Type: "string", Display: "Open Issues", Required: false, Editable: false},
					{Name: "created_at", Type: "datetime", Display: "Created", Required: false, Editable: false},
				},
				Actions: []core.ActionSchema{},
			},
			{
				Name:        "Issues",
				Slug:        "issues",
				ListColumns: []string{"id", "repo", "number", "title", "state", "created_at"},
				Fields: []core.FieldSchema{
					{Name: "id", Type: "string", Display: "ID", Required: true, Editable: false},
					{Name: "repo", Type: "string", Display: "Repository", Required: true, Editable: false},
					{Name: "number", Type: "string", Display: "Number", Required: true, Editable: false},
					{Name: "title", Type: "string", Display: "Title", Required: true, Editable: false},
					{Name: "body", Type: "text", Display: "Body", Required: false, Editable: false},
					{Name: "state", Type: "string", Display: "State", Required: true, Editable: false},
					{Name: "state_reason", Type: "string", Display: "State Reason", Required: false, Editable: false},
					{Name: "user", Type: "string", Display: "Author", Required: false, Editable: false},
					{Name: "comments_count", Type: "string", Display: "Comments", Required: false, Editable: false},
					{Name: "created_at", Type: "datetime", Display: "Created", Required: false, Editable: false},
					{Name: "closed_at", Type: "datetime", Display: "Closed", Required: false, Editable: false},
				},
				Actions: []core.ActionSchema{},
			},
			{
				Name:        "Pull Requests",
				Slug:        "pull_requests",
				ListColumns: []string{"id", "repo", "number", "title", "state", "merged"},
				Fields: []core.FieldSchema{
					{Name: "id", Type: "string", Display: "ID", Required: true, Editable: false},
					{Name: "repo", Type: "string", Display: "Repository", Required: true, Editable: false},
					{Name: "number", Type: "string", Display: "Number", Required: true, Editable: false},
					{Name: "title", Type: "string", Display: "Title", Required: true, Editable: false},
					{Name: "body", Type: "text", Display: "Body", Required: false, Editable: false},
					{Name: "state", Type: "string", Display: "State", Required: true, Editable: false},
					{Name: "merged", Type: "string", Display: "Merged", Required: false, Editable: false},
					{Name: "draft", Type: "string", Display: "Draft", Required: false, Editable: false},
					{Name: "head_ref", Type: "string", Display: "Head Branch", Required: false, Editable: false},
					{Name: "base_ref", Type: "string", Display: "Base Branch", Required: false, Editable: false},
					{Name: "user", Type: "string", Display: "Author", Required: false, Editable: false},
					{Name: "merged_at", Type: "datetime", Display: "Merged", Required: false, Editable: false},
					{Name: "created_at", Type: "datetime", Display: "Created", Required: false, Editable: false},
				},
				Actions: []core.ActionSchema{},
			},
			{
				Name:        "Webhooks",
				Slug:        "webhooks",
				ListColumns: []string{"id", "repo", "url", "events", "active"},
				Fields: []core.FieldSchema{
					{Name: "id", Type: "string", Display: "ID", Required: true, Editable: false},
					{Name: "repo", Type: "string", Display: "Repository", Required: true, Editable: false},
					{Name: "url", Type: "string", Display: "URL", Required: true, Editable: false},
					{Name: "content_type", Type: "string", Display: "Content Type", Required: false, Editable: false},
					{Name: "events", Type: "text", Display: "Events", Required: false, Editable: false},
					{Name: "active", Type: "string", Display: "Active", Required: false, Editable: false},
					{Name: "created_at", Type: "datetime", Display: "Created", Required: false, Editable: false},
					{Name: "updated_at", Type: "datetime", Display: "Updated", Required: false, Editable: false},
				},
				Actions: []core.ActionSchema{},
			},
		},
	}
}

// Seed is implemented in seed.go

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

// ListResources implements core.DataProvider to expose data to admin UI
func (p *GitHubPlugin) ListResources(ctx context.Context, slug string, opts core.ListOptions) ([]map[string]interface{}, error) {
	switch slug {
	case "repositories":
		repos, err := p.store.ListAllRepositories(opts.Limit, opts.Offset)
		if err != nil {
			return nil, err
		}
		return convertRepositoriesToMaps(repos), nil
	case "issues":
		issues, err := p.store.ListAllIssues(opts.Limit, opts.Offset)
		if err != nil {
			return nil, err
		}
		return convertIssuesToMaps(p.store, issues), nil
	case "pull_requests":
		prs, err := p.store.ListAllPullRequests(opts.Limit, opts.Offset)
		if err != nil {
			return nil, err
		}
		return convertPullRequestsToMaps(p.store, prs), nil
	case "webhooks":
		webhooks, err := p.store.ListAllWebhooks(opts.Limit, opts.Offset)
		if err != nil {
			return nil, err
		}
		return convertWebhooksToMaps(p.store, webhooks), nil
	default:
		return nil, fmt.Errorf("unknown resource: %s", slug)
	}
}

// GetResource implements core.DataProvider to fetch individual resources
func (p *GitHubPlugin) GetResource(ctx context.Context, slug string, id string) (map[string]interface{}, error) {
	switch slug {
	case "repositories":
		// Parse ID as int64
		var repoID int64
		_, err := fmt.Sscanf(id, "%d", &repoID)
		if err != nil {
			return nil, fmt.Errorf("invalid repository ID: %s", id)
		}
		// GetRepositoryByID would need to be added to store if needed
		return nil, fmt.Errorf("get repository by ID not implemented")
	case "issues":
		var issueID int64
		_, err := fmt.Sscanf(id, "%d", &issueID)
		if err != nil {
			return nil, fmt.Errorf("invalid issue ID: %s", id)
		}
		// GetIssueByID would need to be added to store if needed
		return nil, fmt.Errorf("get issue by ID not implemented")
	case "pull_requests":
		var prID int64
		_, err := fmt.Sscanf(id, "%d", &prID)
		if err != nil {
			return nil, fmt.Errorf("invalid pull request ID: %s", id)
		}
		// GetPullRequestByID would need to be added to store if needed
		return nil, fmt.Errorf("get pull request by ID not implemented")
	case "webhooks":
		var webhookID int64
		_, err := fmt.Sscanf(id, "%d", &webhookID)
		if err != nil {
			return nil, fmt.Errorf("invalid webhook ID: %s", id)
		}
		webhook, err := p.store.GetWebhook(webhookID)
		if err != nil {
			return nil, err
		}
		return convertWebhookToMap(p.store, *webhook), nil
	default:
		return nil, fmt.Errorf("unknown resource: %s", slug)
	}
}

// Helper conversion functions

func convertRepositoriesToMaps(repos []Repository) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(repos))
	for _, repo := range repos {
		result = append(result, convertRepositoryToMap(repo))
	}
	return result
}

func convertRepositoryToMap(repo Repository) map[string]interface{} {
	m := map[string]interface{}{
		"id":           fmt.Sprintf("%d", repo.ID),
		"name":         repo.Name,
		"full_name":    repo.FullName,
		"owner":        fmt.Sprintf("%d", repo.OwnerID),
		"description":  repo.Description,
		"private":      fmt.Sprintf("%t", repo.Private),
		"default_branch": repo.DefaultBranch,
		"stars":        fmt.Sprintf("%d", repo.StargazersCount),
		"forks":        fmt.Sprintf("%d", repo.ForksCount),
		"issues_count": fmt.Sprintf("%d", repo.OpenIssuesCount),
		"created_at":   repo.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	return m
}

func convertIssuesToMaps(store *GitHubStore, issues []Issue) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(issues))
	for _, issue := range issues {
		result = append(result, convertIssueToMap(store, issue))
	}
	return result
}

func convertIssueToMap(store *GitHubStore, issue Issue) map[string]interface{} {
	// Get repository name
	repoName := fmt.Sprintf("%d", issue.RepoID)
	// Get user login
	userName := fmt.Sprintf("%d", issue.UserID)

	m := map[string]interface{}{
		"id":             fmt.Sprintf("%d", issue.ID),
		"repo":           repoName,
		"number":         fmt.Sprintf("%d", issue.Number),
		"title":          issue.Title,
		"body":           issue.Body,
		"state":          issue.State,
		"state_reason":   issue.StateReason,
		"user":           userName,
		"comments_count": fmt.Sprintf("%d", issue.CommentsCount),
		"created_at":     issue.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if issue.ClosedAt != nil {
		m["closed_at"] = issue.ClosedAt.Format("2006-01-02T15:04:05Z")
	}

	return m
}

func convertPullRequestsToMaps(store *GitHubStore, prs []PullRequest) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(prs))
	for _, pr := range prs {
		result = append(result, convertPullRequestToMap(store, pr))
	}
	return result
}

func convertPullRequestToMap(store *GitHubStore, pr PullRequest) map[string]interface{} {
	// Get the issue details for this PR
	issueID := pr.IssueID
	repoName := fmt.Sprintf("%d", pr.BaseRepoID)

	m := map[string]interface{}{
		"id":       fmt.Sprintf("%d", issueID),
		"repo":     repoName,
		"number":   fmt.Sprintf("%d", issueID), // Issue ID as placeholder for number
		"title":    "",                         // Would need to join with issues table
		"body":     "",                         // Would need to join with issues table
		"state":    "",                         // Would need to join with issues table
		"merged":   fmt.Sprintf("%t", pr.Merged),
		"draft":    fmt.Sprintf("%t", pr.Draft),
		"head_ref": pr.HeadRef,
		"base_ref": pr.BaseRef,
		"user":     "",                         // Would need to join with issues table
	}

	if pr.MergedAt != nil {
		m["merged_at"] = pr.MergedAt.Format("2006-01-02T15:04:05Z")
	}

	// Note: This is a simplified implementation
	// A full implementation would join with github_issues table to get title, body, state, user, number, created_at

	return m
}

func convertWebhooksToMaps(store *GitHubStore, webhooks []Webhook) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(webhooks))
	for _, webhook := range webhooks {
		result = append(result, convertWebhookToMap(store, webhook))
	}
	return result
}

func convertWebhookToMap(store *GitHubStore, webhook Webhook) map[string]interface{} {
	repoName := fmt.Sprintf("%d", webhook.RepoID)

	return map[string]interface{}{
		"id":           fmt.Sprintf("%d", webhook.ID),
		"repo":         repoName,
		"url":          webhook.URL,
		"content_type": webhook.ContentType,
		"events":       webhook.Events,
		"active":       fmt.Sprintf("%t", webhook.Active),
		"created_at":   webhook.CreatedAt.Format("2006-01-02T15:04:05Z"),
		"updated_at":   webhook.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
