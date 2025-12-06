// ABOUTME: HTTP handlers for GitHub REST API v3 endpoints
// ABOUTME: Implements user, repository, issue, PR, review, webhook handlers

package github

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// getAuthenticatedUser handles GET /user
func (p *GitHubPlugin) getAuthenticatedUser(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userContextKey).(*User)

	response := map[string]interface{}{
		"login":      user.Login,
		"id":         user.ID,
		"type":       user.Type,
		"name":       user.Name,
		"email":      user.Email,
		"avatar_url": user.AvatarURL,
		"created_at": user.CreatedAt.Format(time.RFC3339),
		"updated_at": user.UpdatedAt.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// createUserRepository handles POST /user/repos
func (p *GitHubPlugin) createUserRepository(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userContextKey).(*User)

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Private     bool   `json:"private"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	repo, err := p.store.CreateRepository(user.ID, req.Name, req.Description, req.Private)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create repository")
		return
	}

	response := repositoryToResponse(repo, user)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// listAuthenticatedUserRepositories handles GET /user/repos
func (p *GitHubPlugin) listAuthenticatedUserRepositories(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value(userContextKey).(*User)

	repos, err := p.store.ListUserRepositories(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list repositories")
		return
	}

	var response []map[string]interface{}
	for _, repo := range repos {
		response = append(response, repositoryToResponse(repo, user))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getRepository handles GET /repos/{owner}/{repo}
func (p *GitHubPlugin) getRepository(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")

	fullName := owner + "/" + repoName
	repo, err := p.store.GetRepositoryByFullName(fullName)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	// Get owner user
	ownerUser, err := p.store.GetUserByID(repo.OwnerID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get owner")
		return
	}

	response := repositoryToResponse(repo, ownerUser)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// repositoryToResponse converts Repository to GitHub API response format
func repositoryToResponse(repo *Repository, owner *User) map[string]interface{} {
	response := map[string]interface{}{
		"id":                repo.ID,
		"name":              repo.Name,
		"full_name":         repo.FullName,
		"description":       repo.Description,
		"private":           repo.Private,
		"fork":              repo.Fork,
		"archived":          repo.Archived,
		"disabled":          repo.Disabled,
		"default_branch":    repo.DefaultBranch,
		"stargazers_count":  repo.StargazersCount,
		"watchers_count":    repo.WatchersCount,
		"forks_count":       repo.ForksCount,
		"open_issues_count": repo.OpenIssuesCount,
		"created_at":        repo.CreatedAt.Format(time.RFC3339),
		"updated_at":        repo.UpdatedAt.Format(time.RFC3339),
		"owner": map[string]interface{}{
			"login": owner.Login,
			"id":    owner.ID,
			"type":  owner.Type,
		},
	}

	if repo.PushedAt != nil {
		response["pushed_at"] = repo.PushedAt.Format(time.RFC3339)
	}

	return response
}
