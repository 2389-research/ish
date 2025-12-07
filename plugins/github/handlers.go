// ABOUTME: HTTP handlers for GitHub REST API v3 endpoints
// ABOUTME: Implements user, repository, issue, PR, review, webhook handlers

package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// getAuthenticatedUser handles GET /user
func (p *GitHubPlugin) getAuthenticatedUser(w http.ResponseWriter, r *http.Request) {
	user, ok := getUserFromContext(r)
	if !ok {
		writeError(w, http.StatusInternalServerError, "authentication context invalid")
		return
	}

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
	user, ok := getUserFromContext(r)
	if !ok {
		writeError(w, http.StatusInternalServerError, "authentication context invalid")
		return
	}

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
	user, ok := getUserFromContext(r)
	if !ok {
		writeError(w, http.StatusInternalServerError, "authentication context invalid")
		return
	}

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

// createIssue handles POST /repos/{owner}/{repo}/issues
func (p *GitHubPlugin) createIssue(w http.ResponseWriter, r *http.Request) {
	user, ok := getUserFromContext(r)
	if !ok {
		writeError(w, http.StatusInternalServerError, "authentication context invalid")
		return
	}
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")

	var req struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}

	// Get repository
	fullName := owner + "/" + repoName
	repo, err := p.store.GetRepositoryByFullName(fullName)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	issue, err := p.store.CreateIssue(repo.ID, user.ID, req.Title, req.Body, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create issue")
		return
	}

	response := issueToResponse(issue, user, repo)

	// Fire webhooks for issues event
	webhookPayload := map[string]interface{}{
		"action": "opened",
		"issue":  response,
		"repository": map[string]interface{}{
			"id":        repo.ID,
			"name":      repo.Name,
			"full_name": repo.FullName,
		},
	}
	go p.fireWebhooksForEvent(repo.ID, "issues", webhookPayload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// listIssues handles GET /repos/{owner}/{repo}/issues
func (p *GitHubPlugin) listIssues(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")

	// Get repository
	fullName := owner + "/" + repoName
	repo, err := p.store.GetRepositoryByFullName(fullName)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	state := r.URL.Query().Get("state") // open, closed, all

	issues, err := p.store.ListIssues(repo.ID, state, false)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list issues")
		return
	}

	var response []map[string]interface{}
	for _, issue := range issues {
		issueUser, _ := p.store.GetUserByID(issue.UserID)
		response = append(response, issueToResponse(issue, issueUser, repo))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getIssue handles GET /repos/{owner}/{repo}/issues/{number}
func (p *GitHubPlugin) getIssue(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")
	number := chi.URLParam(r, "number")

	// Get repository
	fullName := owner + "/" + repoName
	repo, err := p.store.GetRepositoryByFullName(fullName)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	// Parse number
	var issueNum int
	if _, err := fmt.Sscanf(number, "%d", &issueNum); err != nil {
		writeError(w, http.StatusBadRequest, "invalid issue number")
		return
	}

	issue, err := p.store.GetIssueByNumber(repo.ID, issueNum)
	if err != nil {
		writeError(w, http.StatusNotFound, "issue not found")
		return
	}

	issueUser, _ := p.store.GetUserByID(issue.UserID)
	response := issueToResponse(issue, issueUser, repo)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// updateIssue handles PATCH /repos/{owner}/{repo}/issues/{number}
func (p *GitHubPlugin) updateIssue(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")
	number := chi.URLParam(r, "number")

	var req struct {
		Title       *string `json:"title"`
		Body        *string `json:"body"`
		State       *string `json:"state"`
		StateReason *string `json:"state_reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get repository
	fullName := owner + "/" + repoName
	repo, err := p.store.GetRepositoryByFullName(fullName)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	// Parse number
	var issueNum int
	if _, err := fmt.Sscanf(number, "%d", &issueNum); err != nil {
		writeError(w, http.StatusBadRequest, "invalid issue number")
		return
	}

	issue, err := p.store.GetIssueByNumber(repo.ID, issueNum)
	if err != nil {
		writeError(w, http.StatusNotFound, "issue not found")
		return
	}

	// Update fields
	if req.Title != nil {
		issue.Title = *req.Title
	}
	if req.Body != nil {
		issue.Body = *req.Body
	}
	if req.State != nil {
		issue.State = *req.State
		if *req.State == "closed" && issue.ClosedAt == nil {
			now := time.Now()
			issue.ClosedAt = &now
		} else if *req.State == "open" {
			issue.ClosedAt = nil
		}
	}
	if req.StateReason != nil {
		issue.StateReason = *req.StateReason
	}

	if err := p.store.UpdateIssue(issue); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update issue")
		return
	}

	issueUser, _ := p.store.GetUserByID(issue.UserID)
	response := issueToResponse(issue, issueUser, repo)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// issueToResponse converts Issue to GitHub API response format
func issueToResponse(issue *Issue, user *User, repo *Repository) map[string]interface{} {
	response := map[string]interface{}{
		"id":             issue.ID,
		"number":         issue.Number,
		"title":          issue.Title,
		"body":           issue.Body,
		"state":          issue.State,
		"locked":         issue.Locked,
		"comments":       issue.CommentsCount,
		"created_at":     issue.CreatedAt.Format(time.RFC3339),
		"updated_at":     issue.UpdatedAt.Format(time.RFC3339),
		"repository_url": fmt.Sprintf("/repos/%s", repo.FullName),
	}

	// Handle nil user gracefully (user might have been deleted)
	if user != nil {
		response["user"] = map[string]interface{}{
			"login": user.Login,
			"id":    user.ID,
			"type":  user.Type,
		}
	} else {
		response["user"] = map[string]interface{}{
			"login": "[deleted]",
			"id":    0,
			"type":  "User",
		}
	}

	if issue.StateReason != "" {
		response["state_reason"] = issue.StateReason
	}

	if issue.ClosedAt != nil {
		response["closed_at"] = issue.ClosedAt.Format(time.RFC3339)
	}

	return response
}

// createPullRequest handles POST /repos/{owner}/{repo}/pulls
func (p *GitHubPlugin) createPullRequest(w http.ResponseWriter, r *http.Request) {
	user, ok := getUserFromContext(r)
	if !ok {
		writeError(w, http.StatusInternalServerError, "authentication context invalid")
		return
	}
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")

	var req struct {
		Title string `json:"title"`
		Body  string `json:"body"`
		Head  string `json:"head"`
		Base  string `json:"base"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if req.Head == "" {
		writeError(w, http.StatusBadRequest, "head is required")
		return
	}
	if req.Base == "" {
		writeError(w, http.StatusBadRequest, "base is required")
		return
	}

	// Get repository
	fullName := owner + "/" + repoName
	repo, err := p.store.GetRepositoryByFullName(fullName)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	issue, pr, err := p.store.CreatePullRequest(repo.ID, user.ID, req.Title, req.Body, req.Head, req.Base)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create pull request")
		return
	}

	response := pullRequestToResponse(issue, pr, user, repo)

	// Fire webhooks for pull_request event
	webhookPayload := map[string]interface{}{
		"action":       "opened",
		"pull_request": response,
		"repository": map[string]interface{}{
			"id":        repo.ID,
			"name":      repo.Name,
			"full_name": repo.FullName,
		},
	}
	go p.fireWebhooksForEvent(repo.ID, "pull_request", webhookPayload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// listPullRequests handles GET /repos/{owner}/{repo}/pulls
func (p *GitHubPlugin) listPullRequests(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")

	// Get repository
	fullName := owner + "/" + repoName
	repo, err := p.store.GetRepositoryByFullName(fullName)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	state := r.URL.Query().Get("state") // open, closed, all

	issues, err := p.store.ListPullRequests(repo.ID, state)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list pull requests")
		return
	}

	var response []map[string]interface{}
	for _, issue := range issues {
		issueUser, _ := p.store.GetUserByID(issue.UserID)
		_, pr, err := p.store.GetPullRequest(repo.ID, int(issue.Number))
		if err != nil {
			continue
		}
		response = append(response, pullRequestToResponse(issue, pr, issueUser, repo))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getPullRequest handles GET /repos/{owner}/{repo}/pulls/{number}
func (p *GitHubPlugin) getPullRequest(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")
	number := chi.URLParam(r, "number")

	// Get repository
	fullName := owner + "/" + repoName
	repo, err := p.store.GetRepositoryByFullName(fullName)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	// Parse number
	var prNum int
	if _, err := fmt.Sscanf(number, "%d", &prNum); err != nil {
		writeError(w, http.StatusBadRequest, "invalid pull request number")
		return
	}

	issue, pr, err := p.store.GetPullRequest(repo.ID, prNum)
	if err != nil {
		writeError(w, http.StatusNotFound, "pull request not found")
		return
	}

	issueUser, _ := p.store.GetUserByID(issue.UserID)
	response := pullRequestToResponse(issue, pr, issueUser, repo)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// mergePullRequest handles PUT /repos/{owner}/{repo}/pulls/{number}/merge
func (p *GitHubPlugin) mergePullRequest(w http.ResponseWriter, r *http.Request) {
	user, ok := getUserFromContext(r)
	if !ok {
		writeError(w, http.StatusInternalServerError, "authentication context invalid")
		return
	}
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")
	number := chi.URLParam(r, "number")

	// Get repository
	fullName := owner + "/" + repoName
	repo, err := p.store.GetRepositoryByFullName(fullName)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	// Parse number
	var prNum int
	if _, err := fmt.Sscanf(number, "%d", &prNum); err != nil {
		writeError(w, http.StatusBadRequest, "invalid pull request number")
		return
	}

	issue, pr, err := p.store.GetPullRequest(repo.ID, prNum)
	if err != nil {
		writeError(w, http.StatusNotFound, "pull request not found")
		return
	}

	if pr.Merged {
		writeError(w, http.StatusMethodNotAllowed, "pull request already merged")
		return
	}

	if err := p.store.MergePullRequest(issue.ID, user.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to merge pull request")
		return
	}

	// Reload PR to get updated data
	issue, pr, _ = p.store.GetPullRequest(repo.ID, prNum)
	issueUser, _ := p.store.GetUserByID(issue.UserID)
	response := pullRequestToResponse(issue, pr, issueUser, repo)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// pullRequestToResponse converts Issue + PullRequest to GitHub API response format
func pullRequestToResponse(issue *Issue, pr *PullRequest, user *User, repo *Repository) map[string]interface{} {
	response := map[string]interface{}{
		"id":         issue.ID,
		"number":     issue.Number,
		"title":      issue.Title,
		"body":       issue.Body,
		"state":      issue.State,
		"locked":     issue.Locked,
		"created_at": issue.CreatedAt.Format(time.RFC3339),
		"updated_at": issue.UpdatedAt.Format(time.RFC3339),
		"user": map[string]interface{}{
			"login": user.Login,
			"id":    user.ID,
			"type":  user.Type,
		},
		"head": map[string]interface{}{
			"ref": pr.HeadRef,
			"repo": map[string]interface{}{
				"id":        repo.ID,
				"name":      repo.Name,
				"full_name": repo.FullName,
			},
		},
		"base": map[string]interface{}{
			"ref": pr.BaseRef,
			"repo": map[string]interface{}{
				"id":        repo.ID,
				"name":      repo.Name,
				"full_name": repo.FullName,
			},
		},
		"merged":     pr.Merged,
		"mergeable":  pr.Mergeable,
		"rebaseable": pr.Rebaseable,
		"draft":      pr.Draft,
	}

	if issue.StateReason != "" {
		response["state_reason"] = issue.StateReason
	}

	if issue.ClosedAt != nil {
		response["closed_at"] = issue.ClosedAt.Format(time.RFC3339)
	}

	if pr.MergedAt != nil {
		response["merged_at"] = pr.MergedAt.Format(time.RFC3339)
	}

	if pr.MergedByID != nil {
		response["merged_by"] = map[string]interface{}{
			"id": *pr.MergedByID,
		}
	}

	if pr.MergeCommitSHA != "" {
		response["merge_commit_sha"] = pr.MergeCommitSHA
	}

	return response
}

// createComment handles POST /repos/{owner}/{repo}/issues/{number}/comments
func (p *GitHubPlugin) createComment(w http.ResponseWriter, r *http.Request) {
	user, ok := getUserFromContext(r)
	if !ok {
		writeError(w, http.StatusInternalServerError, "authentication context invalid")
		return
	}
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")
	number := chi.URLParam(r, "number")

	var req struct {
		Body string `json:"body"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Body == "" {
		writeError(w, http.StatusBadRequest, "body is required")
		return
	}

	// Get repository
	fullName := owner + "/" + repoName
	repo, err := p.store.GetRepositoryByFullName(fullName)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	// Parse number
	var issueNum int
	if _, err := fmt.Sscanf(number, "%d", &issueNum); err != nil {
		writeError(w, http.StatusBadRequest, "invalid issue number")
		return
	}

	// Get issue to verify it exists
	issue, err := p.store.GetIssueByNumber(repo.ID, issueNum)
	if err != nil {
		writeError(w, http.StatusNotFound, "issue not found")
		return
	}

	// Create comment
	comment, err := p.store.CreateComment(issue.ID, user.ID, req.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create comment")
		return
	}

	response := commentToResponse(comment, user)

	// Fire webhooks for issue_comment event
	issueResponse := issueToResponse(issue, user, repo)
	webhookPayload := map[string]interface{}{
		"action":  "created",
		"comment": response,
		"issue":   issueResponse,
		"repository": map[string]interface{}{
			"id":        repo.ID,
			"name":      repo.Name,
			"full_name": repo.FullName,
		},
	}
	go p.fireWebhooksForEvent(repo.ID, "issue_comment", webhookPayload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// listComments handles GET /repos/{owner}/{repo}/issues/{number}/comments
func (p *GitHubPlugin) listComments(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")
	number := chi.URLParam(r, "number")

	// Get repository
	fullName := owner + "/" + repoName
	repo, err := p.store.GetRepositoryByFullName(fullName)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	// Parse number
	var issueNum int
	if _, err := fmt.Sscanf(number, "%d", &issueNum); err != nil {
		writeError(w, http.StatusBadRequest, "invalid issue number")
		return
	}

	// Get issue to verify it exists
	issue, err := p.store.GetIssueByNumber(repo.ID, issueNum)
	if err != nil {
		writeError(w, http.StatusNotFound, "issue not found")
		return
	}

	// List comments
	comments, err := p.store.ListComments(issue.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list comments")
		return
	}

	var response []map[string]interface{}
	for _, comment := range comments {
		commentUser, _ := p.store.GetUserByID(comment.UserID)
		response = append(response, commentToResponse(comment, commentUser))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// updateComment handles PATCH /repos/{owner}/{repo}/issues/comments/{comment_id}
func (p *GitHubPlugin) updateComment(w http.ResponseWriter, r *http.Request) {
	commentID := chi.URLParam(r, "comment_id")

	var req struct {
		Body string `json:"body"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Body == "" {
		writeError(w, http.StatusBadRequest, "body is required")
		return
	}

	// Parse comment ID
	var id int64
	if _, err := fmt.Sscanf(commentID, "%d", &id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid comment id")
		return
	}

	// Get comment
	comment, err := p.store.GetComment(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "comment not found")
		return
	}

	// Update comment
	comment.Body = req.Body
	if err := p.store.UpdateComment(comment); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update comment")
		return
	}

	// Get user for response
	commentUser, _ := p.store.GetUserByID(comment.UserID)
	response := commentToResponse(comment, commentUser)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// deleteComment handles DELETE /repos/{owner}/{repo}/issues/comments/{comment_id}
func (p *GitHubPlugin) deleteComment(w http.ResponseWriter, r *http.Request) {
	commentID := chi.URLParam(r, "comment_id")

	// Parse comment ID
	var id int64
	if _, err := fmt.Sscanf(commentID, "%d", &id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid comment id")
		return
	}

	// Verify comment exists
	_, err := p.store.GetComment(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "comment not found")
		return
	}

	// Delete comment
	if err := p.store.DeleteComment(id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete comment")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// commentToResponse converts Comment to GitHub API response format
func commentToResponse(comment *Comment, user *User) map[string]interface{} {
	response := map[string]interface{}{
		"id":         comment.ID,
		"body":       comment.Body,
		"created_at": comment.CreatedAt.Format(time.RFC3339),
		"updated_at": comment.UpdatedAt.Format(time.RFC3339),
	}

	if user != nil {
		response["user"] = map[string]interface{}{
			"login": user.Login,
			"id":    user.ID,
			"type":  user.Type,
		}
	}

	return response
}

// createReview handles POST /repos/{owner}/{repo}/pulls/{number}/reviews
func (p *GitHubPlugin) createReview(w http.ResponseWriter, r *http.Request) {
	user, ok := getUserFromContext(r)
	if !ok {
		writeError(w, http.StatusInternalServerError, "authentication context invalid")
		return
	}
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")
	number := chi.URLParam(r, "number")

	var req struct {
		State string `json:"state"`
		Body  string `json:"body"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.State == "" {
		writeError(w, http.StatusBadRequest, "state is required")
		return
	}

	// Validate state
	validStates := map[string]bool{
		"PENDING":            true,
		"COMMENTED":          true,
		"APPROVED":           true,
		"CHANGES_REQUESTED":  true,
	}
	if !validStates[req.State] {
		writeError(w, http.StatusBadRequest, "invalid state")
		return
	}

	// Get repository
	fullName := owner + "/" + repoName
	repo, err := p.store.GetRepositoryByFullName(fullName)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	// Parse number
	var prNum int
	if _, err := fmt.Sscanf(number, "%d", &prNum); err != nil {
		writeError(w, http.StatusBadRequest, "invalid pull request number")
		return
	}

	// Get PR to verify it exists
	issue, _, err := p.store.GetPullRequest(repo.ID, prNum)
	if err != nil {
		writeError(w, http.StatusNotFound, "pull request not found")
		return
	}

	// Create review
	review, err := p.store.CreateReview(issue.ID, user.ID, req.State, req.Body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create review")
		return
	}

	response := reviewToResponse(review, user)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// listReviews handles GET /repos/{owner}/{repo}/pulls/{number}/reviews
func (p *GitHubPlugin) listReviews(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")
	number := chi.URLParam(r, "number")

	// Get repository
	fullName := owner + "/" + repoName
	repo, err := p.store.GetRepositoryByFullName(fullName)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	// Parse number
	var prNum int
	if _, err := fmt.Sscanf(number, "%d", &prNum); err != nil {
		writeError(w, http.StatusBadRequest, "invalid pull request number")
		return
	}

	// Get PR to verify it exists
	issue, _, err := p.store.GetPullRequest(repo.ID, prNum)
	if err != nil {
		writeError(w, http.StatusNotFound, "pull request not found")
		return
	}

	// List reviews
	reviews, err := p.store.ListReviews(issue.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list reviews")
		return
	}

	var response []map[string]interface{}
	for _, review := range reviews {
		reviewUser, _ := p.store.GetUserByID(review.UserID)
		response = append(response, reviewToResponse(review, reviewUser))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// submitReview handles PUT /repos/{owner}/{repo}/pulls/{number}/reviews/{id}
func (p *GitHubPlugin) submitReview(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")
	number := chi.URLParam(r, "number")
	reviewID := chi.URLParam(r, "id")

	// Get repository
	fullName := owner + "/" + repoName
	repo, err := p.store.GetRepositoryByFullName(fullName)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	// Parse number
	var prNum int
	if _, err := fmt.Sscanf(number, "%d", &prNum); err != nil {
		writeError(w, http.StatusBadRequest, "invalid pull request number")
		return
	}

	// Verify PR exists
	_, _, err = p.store.GetPullRequest(repo.ID, prNum)
	if err != nil {
		writeError(w, http.StatusNotFound, "pull request not found")
		return
	}

	// Parse review ID
	var id int64
	if _, err := fmt.Sscanf(reviewID, "%d", &id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid review id")
		return
	}

	// Verify review exists
	review, err := p.store.GetReview(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "review not found")
		return
	}

	// Submit the review
	if err := p.store.SubmitReview(id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to submit review")
		return
	}

	// Reload review to get updated data
	review, _ = p.store.GetReview(id)
	reviewUser, _ := p.store.GetUserByID(review.UserID)
	response := reviewToResponse(review, reviewUser)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// dismissReview handles DELETE /repos/{owner}/{repo}/pulls/{number}/reviews/{id}
func (p *GitHubPlugin) dismissReview(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")
	number := chi.URLParam(r, "number")
	reviewID := chi.URLParam(r, "id")

	// Get repository
	fullName := owner + "/" + repoName
	repo, err := p.store.GetRepositoryByFullName(fullName)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	// Parse number
	var prNum int
	if _, err := fmt.Sscanf(number, "%d", &prNum); err != nil {
		writeError(w, http.StatusBadRequest, "invalid pull request number")
		return
	}

	// Verify PR exists
	_, _, err = p.store.GetPullRequest(repo.ID, prNum)
	if err != nil {
		writeError(w, http.StatusNotFound, "pull request not found")
		return
	}

	// Parse review ID
	var id int64
	if _, err := fmt.Sscanf(reviewID, "%d", &id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid review id")
		return
	}

	// Verify review exists
	review, err := p.store.GetReview(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "review not found")
		return
	}

	// Dismiss the review
	if err := p.store.DismissReview(id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to dismiss review")
		return
	}

	// Reload review to get updated data
	review, _ = p.store.GetReview(id)
	reviewUser, _ := p.store.GetUserByID(review.UserID)
	response := reviewToResponse(review, reviewUser)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// reviewToResponse converts Review to GitHub API response format
func reviewToResponse(review *Review, user *User) map[string]interface{} {
	response := map[string]interface{}{
		"id":         review.ID,
		"state":      review.State,
		"body":       review.Body,
		"commit_id":  review.CommitSHA,
	}

	if user != nil {
		response["user"] = map[string]interface{}{
			"login": user.Login,
			"id":    user.ID,
			"type":  user.Type,
		}
	}

	if review.SubmittedAt != nil {
		response["submitted_at"] = review.SubmittedAt.Format(time.RFC3339)
	}

	if review.DismissedAt != nil {
		response["dismissed_at"] = review.DismissedAt.Format(time.RFC3339)
	}

	return response
}

// fireWebhooksForEvent finds active webhooks for an event and fires them
// Includes panic recovery to prevent goroutine crashes from affecting the server
func (p *GitHubPlugin) fireWebhooksForEvent(repoID int64, eventType string, payload interface{}) {
	// Add panic recovery to prevent goroutine crashes
	defer func() {
		if r := recover(); r != nil {
			// In production, this should be logged
			// For now, we silently recover to prevent server crashes
			_ = r
		}
	}()

	webhooks, err := p.store.GetActiveWebhooksForEvent(repoID, eventType)
	if err != nil {
		// Log error but don't fail the request
		return
	}

	// Fire webhooks synchronously for now
	for _, webhook := range webhooks {
		payloadBytes, _ := json.Marshal(payload)
		err := fireWebhook(webhook, eventType, payload)

		statusCode := 200
		errorMsg := ""
		if err != nil {
			statusCode = 500
			errorMsg = err.Error()
		}

		// Log delivery
		p.store.CreateWebhookDelivery(webhook.ID, eventType, string(payloadBytes), statusCode, errorMsg)
	}
}

// createWebhook handles POST /repos/{owner}/{repo}/hooks
func (p *GitHubPlugin) createWebhook(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")

	var req struct {
		Config struct {
			URL         string `json:"url"`
			ContentType string `json:"content_type"`
			Secret      string `json:"secret"`
		} `json:"config"`
		Events []string `json:"events"`
		Active *bool    `json:"active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Config.URL == "" {
		writeError(w, http.StatusBadRequest, "url is required")
		return
	}

	// Get repository
	fullName := owner + "/" + repoName
	repo, err := p.store.GetRepositoryByFullName(fullName)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	// Default content type
	contentType := req.Config.ContentType
	if contentType == "" {
		contentType = "json"
	}

	// Create webhook
	webhook, err := p.store.CreateWebhook(repo.ID, req.Config.URL, contentType, req.Config.Secret, req.Events)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	response := webhookToResponse(webhook)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// listWebhooks handles GET /repos/{owner}/{repo}/hooks
func (p *GitHubPlugin) listWebhooks(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")

	// Get repository
	fullName := owner + "/" + repoName
	repo, err := p.store.GetRepositoryByFullName(fullName)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	webhooks, err := p.store.ListWebhooks(repo.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list webhooks")
		return
	}

	var response []map[string]interface{}
	for _, webhook := range webhooks {
		response = append(response, webhookToResponse(webhook))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getWebhook handles GET /repos/{owner}/{repo}/hooks/{id}
func (p *GitHubPlugin) getWebhook(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")
	hookID := chi.URLParam(r, "id")

	// Get repository
	fullName := owner + "/" + repoName
	repo, err := p.store.GetRepositoryByFullName(fullName)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	// Parse hook ID
	var id int64
	if _, err := fmt.Sscanf(hookID, "%d", &id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid hook id")
		return
	}

	webhook, err := p.store.GetWebhook(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "webhook not found")
		return
	}

	// Verify webhook belongs to this repo
	if webhook.RepoID != repo.ID {
		writeError(w, http.StatusNotFound, "webhook not found")
		return
	}

	response := webhookToResponse(webhook)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// updateWebhook handles PATCH /repos/{owner}/{repo}/hooks/{id}
func (p *GitHubPlugin) updateWebhook(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")
	hookID := chi.URLParam(r, "id")

	var req struct {
		Config *struct {
			URL         string `json:"url"`
			ContentType string `json:"content_type"`
			Secret      string `json:"secret"`
		} `json:"config"`
		Events []string `json:"events"`
		Active *bool    `json:"active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Get repository
	fullName := owner + "/" + repoName
	repo, err := p.store.GetRepositoryByFullName(fullName)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	// Parse hook ID
	var id int64
	if _, err := fmt.Sscanf(hookID, "%d", &id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid hook id")
		return
	}

	webhook, err := p.store.GetWebhook(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "webhook not found")
		return
	}

	// Verify webhook belongs to this repo
	if webhook.RepoID != repo.ID {
		writeError(w, http.StatusNotFound, "webhook not found")
		return
	}

	// Update fields
	if req.Config != nil {
		if req.Config.URL != "" {
			webhook.URL = req.Config.URL
		}
		if req.Config.ContentType != "" {
			webhook.ContentType = req.Config.ContentType
		}
		if req.Config.Secret != "" {
			webhook.Secret = req.Config.Secret
		}
	}
	if len(req.Events) > 0 {
		eventsStr := ""
		for i, event := range req.Events {
			if i > 0 {
				eventsStr += ","
			}
			eventsStr += event
		}
		webhook.Events = eventsStr
	}
	if req.Active != nil {
		webhook.Active = *req.Active
	}

	if err := p.store.UpdateWebhook(webhook); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	response := webhookToResponse(webhook)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// deleteWebhook handles DELETE /repos/{owner}/{repo}/hooks/{id}
func (p *GitHubPlugin) deleteWebhook(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")
	hookID := chi.URLParam(r, "id")

	// Get repository
	fullName := owner + "/" + repoName
	repo, err := p.store.GetRepositoryByFullName(fullName)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	// Parse hook ID
	var id int64
	if _, err := fmt.Sscanf(hookID, "%d", &id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid hook id")
		return
	}

	webhook, err := p.store.GetWebhook(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "webhook not found")
		return
	}

	// Verify webhook belongs to this repo
	if webhook.RepoID != repo.ID {
		writeError(w, http.StatusNotFound, "webhook not found")
		return
	}

	if err := p.store.DeleteWebhook(id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete webhook")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// testWebhook handles POST /repos/{owner}/{repo}/hooks/{id}/tests
func (p *GitHubPlugin) testWebhook(w http.ResponseWriter, r *http.Request) {
	owner := chi.URLParam(r, "owner")
	repoName := chi.URLParam(r, "repo")
	hookID := chi.URLParam(r, "id")

	// Get repository
	fullName := owner + "/" + repoName
	repo, err := p.store.GetRepositoryByFullName(fullName)
	if err != nil {
		writeError(w, http.StatusNotFound, "repository not found")
		return
	}

	// Parse hook ID
	var id int64
	if _, err := fmt.Sscanf(hookID, "%d", &id); err != nil {
		writeError(w, http.StatusBadRequest, "invalid hook id")
		return
	}

	webhook, err := p.store.GetWebhook(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "webhook not found")
		return
	}

	// Verify webhook belongs to this repo
	if webhook.RepoID != repo.ID {
		writeError(w, http.StatusNotFound, "webhook not found")
		return
	}

	// Fire test ping event
	payload := map[string]interface{}{
		"zen":    "Design for failure.",
		"hook_id": webhook.ID,
		"repository": map[string]interface{}{
			"id":        repo.ID,
			"name":      repo.Name,
			"full_name": repo.FullName,
		},
	}

	payloadBytes, _ := json.Marshal(payload)
	err = fireWebhook(webhook, "ping", payload)

	statusCode := 200
	errorMsg := ""
	if err != nil {
		statusCode = 500
		errorMsg = err.Error()
	}

	// Log delivery
	p.store.CreateWebhookDelivery(webhook.ID, "ping", string(payloadBytes), statusCode, errorMsg)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "webhook delivery failed: "+err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// webhookToResponse converts Webhook to GitHub API response format
func webhookToResponse(webhook *Webhook) map[string]interface{} {
	events := []string{}
	if webhook.Events != "" {
		events = strings.Split(webhook.Events, ",")
	}

	response := map[string]interface{}{
		"id":     webhook.ID,
		"type":   "Repository",
		"active": webhook.Active,
		"events": events,
		"config": map[string]interface{}{
			"url":          webhook.URL,
			"content_type": webhook.ContentType,
		},
		"created_at": webhook.CreatedAt.Format(time.RFC3339),
		"updated_at": webhook.UpdatedAt.Format(time.RFC3339),
	}

	return response
}
