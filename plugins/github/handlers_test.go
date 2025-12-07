// ABOUTME: Tests for GitHub API handlers
// ABOUTME: Tests repository, user, and resource CRUD endpoints

package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestGetAuthenticatedUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Create user
	store.GetOrCreateUser("alice", "ghp_test")

	req := httptest.NewRequest("GET", "/user", nil)
	req.Header.Set("Authorization", "Bearer ghp_test")
	w := httptest.NewRecorder()

	handler := plugin.requireAuth(plugin.getAuthenticatedUser)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["login"] != "alice" {
		t.Fatalf("Expected login 'alice', got %v", resp["login"])
	}
	if resp["type"] != "User" {
		t.Fatalf("Expected type 'User', got %v", resp["type"])
	}
}

func TestCreateRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Create user
	user, _ := store.GetOrCreateUser("alice", "ghp_test")

	body := `{"name": "test-repo", "description": "Test repository", "private": false}`
	req := httptest.NewRequest("POST", "/user/repos", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer ghp_test")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler := plugin.requireAuth(plugin.createUserRepository)
	handler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["name"] != "test-repo" {
		t.Fatalf("Expected name 'test-repo', got %v", resp["name"])
	}
	if resp["full_name"] != "alice/test-repo" {
		t.Fatalf("Expected full_name 'alice/test-repo', got %v", resp["full_name"])
	}
	if resp["description"] != "Test repository" {
		t.Fatalf("Expected description, got %v", resp["description"])
	}

	// Verify in database
	repo, err := store.GetRepository(user.ID, "test-repo")
	if err != nil {
		t.Fatalf("Repo not found in database: %v", err)
	}
	if repo.Name != "test-repo" {
		t.Fatalf("Database name mismatch")
	}
}

func TestListUserRepositories(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Create user and repos
	user, _ := store.GetOrCreateUser("alice", "ghp_test")
	store.CreateRepository(user.ID, "repo1", "", false)
	store.CreateRepository(user.ID, "repo2", "", false)

	req := httptest.NewRequest("GET", "/user/repos", nil)
	req.Header.Set("Authorization", "Bearer ghp_test")
	w := httptest.NewRecorder()

	handler := plugin.requireAuth(plugin.listAuthenticatedUserRepositories)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var repos []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &repos)

	if len(repos) != 2 {
		t.Fatalf("Expected 2 repos, got %d", len(repos))
	}
}

func TestGetRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Create user and repo
	user, _ := store.GetOrCreateUser("alice", "ghp_test")
	store.CreateRepository(user.ID, "my-repo", "Test repo", false)

	req := httptest.NewRequest("GET", "/repos/alice/my-repo", nil)
	req.Header.Set("Authorization", "Bearer ghp_test")
	w := httptest.NewRecorder()

	// Setup chi context for path params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("owner", "alice")
	rctx.URLParams.Add("repo", "my-repo")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := plugin.requireAuth(plugin.getRepository)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["name"] != "my-repo" {
		t.Fatalf("Expected name 'my-repo', got %v", resp["name"])
	}
	if resp["full_name"] != "alice/my-repo" {
		t.Fatalf("Expected full_name 'alice/my-repo', got %v", resp["full_name"])
	}
}

func TestCreateIssue(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Create user and repo
	user, _ := store.GetOrCreateUser("alice", "ghp_test")
	repo, _ := store.CreateRepository(user.ID, "test-repo", "", false)

	body := `{"title": "Bug in login", "body": "Login fails with invalid credentials"}`
	req := httptest.NewRequest("POST", "/repos/alice/test-repo/issues", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer ghp_test")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Setup chi context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("owner", "alice")
	rctx.URLParams.Add("repo", "test-repo")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := plugin.requireAuth(plugin.createIssue)
	handler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["number"] != float64(1) {
		t.Fatalf("Expected number 1, got %v", resp["number"])
	}
	if resp["title"] != "Bug in login" {
		t.Fatalf("Expected title, got %v", resp["title"])
	}
	if resp["state"] != "open" {
		t.Fatalf("Expected state 'open', got %v", resp["state"])
	}

	// Verify in database
	issue, err := store.GetIssueByNumber(repo.ID, 1)
	if err != nil {
		t.Fatalf("Issue not found: %v", err)
	}
	if issue.Title != "Bug in login" {
		t.Fatal("Title mismatch in database")
	}
}

func TestListIssues(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Create user, repo, and issues
	user, _ := store.GetOrCreateUser("alice", "ghp_test")
	repo, _ := store.CreateRepository(user.ID, "test-repo", "", false)
	store.CreateIssue(repo.ID, user.ID, "Issue 1", "Body 1", false)
	store.CreateIssue(repo.ID, user.ID, "Issue 2", "Body 2", false)

	req := httptest.NewRequest("GET", "/repos/alice/test-repo/issues", nil)
	req.Header.Set("Authorization", "Bearer ghp_test")
	w := httptest.NewRecorder()

	// Setup chi context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("owner", "alice")
	rctx.URLParams.Add("repo", "test-repo")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := plugin.requireAuth(plugin.listIssues)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var issues []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &issues)

	if len(issues) != 2 {
		t.Fatalf("Expected 2 issues, got %d", len(issues))
	}
}

func TestUpdateIssueState(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Create user, repo, and issue
	user, _ := store.GetOrCreateUser("alice", "ghp_test")
	repo, _ := store.CreateRepository(user.ID, "test-repo", "", false)
	issue, _ := store.CreateIssue(repo.ID, user.ID, "Test issue", "Body", false)

	body := `{"state": "closed", "state_reason": "completed"}`
	req := httptest.NewRequest("PATCH", "/repos/alice/test-repo/issues/1", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer ghp_test")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Setup chi context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("owner", "alice")
	rctx.URLParams.Add("repo", "test-repo")
	rctx.URLParams.Add("number", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := plugin.requireAuth(plugin.updateIssue)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify in database
	updated, _ := store.GetIssueByNumber(repo.ID, int(issue.Number))
	if updated.State != "closed" {
		t.Fatalf("Expected state 'closed', got '%s'", updated.State)
	}
	if updated.StateReason != "completed" {
		t.Fatalf("Expected state_reason 'completed', got '%s'", updated.StateReason)
	}
	if updated.ClosedAt == nil {
		t.Fatal("ClosedAt should be set")
	}
}

func TestCreatePullRequest(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Create user and repo
	user, _ := store.GetOrCreateUser("alice", "ghp_test")
	repo, _ := store.CreateRepository(user.ID, "test-repo", "", false)

	body := `{"title": "Add feature X", "body": "This PR adds feature X", "head": "feature-x", "base": "main"}`
	req := httptest.NewRequest("POST", "/repos/alice/test-repo/pulls", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer ghp_test")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Setup chi context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("owner", "alice")
	rctx.URLParams.Add("repo", "test-repo")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := plugin.requireAuth(plugin.createPullRequest)
	handler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["number"] != float64(1) {
		t.Fatalf("Expected number 1, got %v", resp["number"])
	}
	if resp["title"] != "Add feature X" {
		t.Fatalf("Expected title, got %v", resp["title"])
	}
	if resp["state"] != "open" {
		t.Fatalf("Expected state 'open', got %v", resp["state"])
	}

	// Check PR-specific fields
	head := resp["head"].(map[string]interface{})
	if head["ref"] != "feature-x" {
		t.Fatalf("Expected head ref 'feature-x', got %v", head["ref"])
	}
	base := resp["base"].(map[string]interface{})
	if base["ref"] != "main" {
		t.Fatalf("Expected base ref 'main', got %v", base["ref"])
	}
	if resp["merged"] != false {
		t.Fatalf("Expected merged false, got %v", resp["merged"])
	}

	// Verify in database - should create both issue and PR record
	issue, err := store.GetIssueByNumber(repo.ID, 1)
	if err != nil {
		t.Fatalf("Issue not found: %v", err)
	}
	if !issue.IsPullRequest {
		t.Fatal("Issue should be marked as pull request")
	}

	_, pr, err := store.GetPullRequest(repo.ID, 1)
	if err != nil {
		t.Fatalf("PR not found: %v", err)
	}
	if pr.HeadRef != "feature-x" {
		t.Fatalf("Expected head ref 'feature-x', got '%s'", pr.HeadRef)
	}
	if pr.BaseRef != "main" {
		t.Fatalf("Expected base ref 'main', got '%s'", pr.BaseRef)
	}
}

func TestListPullRequests(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Create user, repo, and PRs
	user, _ := store.GetOrCreateUser("alice", "ghp_test")
	repo, _ := store.CreateRepository(user.ID, "test-repo", "", false)

	// Create a regular issue and two PRs
	store.CreateIssue(repo.ID, user.ID, "Regular issue", "Not a PR", false)
	store.CreatePullRequest(repo.ID, user.ID, "PR 1", "Body 1", "feature-1", "main")
	store.CreatePullRequest(repo.ID, user.ID, "PR 2", "Body 2", "feature-2", "main")

	req := httptest.NewRequest("GET", "/repos/alice/test-repo/pulls", nil)
	req.Header.Set("Authorization", "Bearer ghp_test")
	w := httptest.NewRecorder()

	// Setup chi context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("owner", "alice")
	rctx.URLParams.Add("repo", "test-repo")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := plugin.requireAuth(plugin.listPullRequests)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var prs []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &prs)

	// Should only return the 2 PRs, not the regular issue
	if len(prs) != 2 {
		t.Fatalf("Expected 2 PRs, got %d", len(prs))
	}

	// Verify numbers are 2 and 3 (issue #1 was the regular issue)
	if prs[1]["number"] != float64(2) {
		t.Fatalf("Expected PR number 2, got %v", prs[1]["number"])
	}
	if prs[0]["number"] != float64(3) {
		t.Fatalf("Expected PR number 3, got %v", prs[0]["number"])
	}
}

func TestGetPullRequest(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Create user, repo, and PR
	user, _ := store.GetOrCreateUser("alice", "ghp_test")
	repo, _ := store.CreateRepository(user.ID, "test-repo", "", false)
	store.CreatePullRequest(repo.ID, user.ID, "Test PR", "Body", "feature", "main")

	req := httptest.NewRequest("GET", "/repos/alice/test-repo/pulls/1", nil)
	req.Header.Set("Authorization", "Bearer ghp_test")
	w := httptest.NewRecorder()

	// Setup chi context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("owner", "alice")
	rctx.URLParams.Add("repo", "test-repo")
	rctx.URLParams.Add("number", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := plugin.requireAuth(plugin.getPullRequest)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["number"] != float64(1) {
		t.Fatalf("Expected number 1, got %v", resp["number"])
	}
	if resp["title"] != "Test PR" {
		t.Fatalf("Expected title 'Test PR', got %v", resp["title"])
	}

	head := resp["head"].(map[string]interface{})
	if head["ref"] != "feature" {
		t.Fatalf("Expected head ref 'feature', got %v", head["ref"])
	}
}

func TestMergePullRequest(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Create user, repo, and PR
	user, _ := store.GetOrCreateUser("alice", "ghp_test")
	repo, _ := store.CreateRepository(user.ID, "test-repo", "", false)
	issue, _, _ := store.CreatePullRequest(repo.ID, user.ID, "Test PR", "Body", "feature", "main")

	body := `{"commit_message": "Merge pull request #1"}`
	req := httptest.NewRequest("PUT", "/repos/alice/test-repo/pulls/1/merge", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer ghp_test")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Setup chi context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("owner", "alice")
	rctx.URLParams.Add("repo", "test-repo")
	rctx.URLParams.Add("number", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := plugin.requireAuth(plugin.mergePullRequest)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["merged"] != true {
		t.Fatalf("Expected merged true, got %v", resp["merged"])
	}

	// Verify in database
	updatedIssue, pr, _ := store.GetPullRequest(repo.ID, int(issue.Number))

	// PR should be marked as merged
	if !pr.Merged {
		t.Fatal("PR should be marked as merged")
	}
	if pr.MergedAt == nil {
		t.Fatal("MergedAt should be set")
	}
	if pr.MergedByID == nil || *pr.MergedByID != user.ID {
		t.Fatalf("MergedByID should be set to %d, got %v", user.ID, pr.MergedByID)
	}

	// Issue should be closed
	if updatedIssue.State != "closed" {
		t.Fatalf("Issue state should be 'closed', got '%s'", updatedIssue.State)
	}
	if updatedIssue.ClosedAt == nil {
		t.Fatal("ClosedAt should be set")
	}
}

func TestCreateComment(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Create user, repo, and issue
	user, _ := store.GetOrCreateUser("alice", "ghp_test")
	repo, _ := store.CreateRepository(user.ID, "test-repo", "", false)
	issue, _ := store.CreateIssue(repo.ID, user.ID, "Test Issue", "Body", false)

	// Verify initial comments_count is 0
	if issue.CommentsCount != 0 {
		t.Fatalf("Expected comments_count 0, got %d", issue.CommentsCount)
	}

	body := `{"body": "This is a test comment"}`
	req := httptest.NewRequest("POST", "/repos/alice/test-repo/issues/1/comments", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer ghp_test")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Setup chi context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("owner", "alice")
	rctx.URLParams.Add("repo", "test-repo")
	rctx.URLParams.Add("number", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := plugin.requireAuth(plugin.createComment)
	handler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["body"] != "This is a test comment" {
		t.Fatalf("Expected body 'This is a test comment', got %v", resp["body"])
	}
	if resp["user"].(map[string]interface{})["login"] != "alice" {
		t.Fatalf("Expected user login 'alice', got %v", resp["user"])
	}

	// Verify comments_count was incremented
	updatedIssue, _ := store.GetIssueByNumber(repo.ID, int(issue.Number))
	if updatedIssue.CommentsCount != 1 {
		t.Fatalf("Expected comments_count 1, got %d", updatedIssue.CommentsCount)
	}
}

func TestListComments(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Create user, repo, issue, and comments
	user, _ := store.GetOrCreateUser("alice", "ghp_test")
	repo, _ := store.CreateRepository(user.ID, "test-repo", "", false)
	issue, _ := store.CreateIssue(repo.ID, user.ID, "Test Issue", "Body", false)

	// Create 3 comments
	store.CreateComment(issue.ID, user.ID, "First comment")
	store.CreateComment(issue.ID, user.ID, "Second comment")
	store.CreateComment(issue.ID, user.ID, "Third comment")

	req := httptest.NewRequest("GET", "/repos/alice/test-repo/issues/1/comments", nil)
	req.Header.Set("Authorization", "Bearer ghp_test")
	w := httptest.NewRecorder()

	// Setup chi context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("owner", "alice")
	rctx.URLParams.Add("repo", "test-repo")
	rctx.URLParams.Add("number", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := plugin.requireAuth(plugin.listComments)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp) != 3 {
		t.Fatalf("Expected 3 comments, got %d", len(resp))
	}

	// Verify comments are in order
	if resp[0]["body"] != "First comment" {
		t.Fatalf("Expected first comment body 'First comment', got %v", resp[0]["body"])
	}
	if resp[1]["body"] != "Second comment" {
		t.Fatalf("Expected second comment body 'Second comment', got %v", resp[1]["body"])
	}
	if resp[2]["body"] != "Third comment" {
		t.Fatalf("Expected third comment body 'Third comment', got %v", resp[2]["body"])
	}
}

func TestUpdateComment(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Create user, repo, issue, and comment
	user, _ := store.GetOrCreateUser("alice", "ghp_test")
	repo, _ := store.CreateRepository(user.ID, "test-repo", "", false)
	issue, _ := store.CreateIssue(repo.ID, user.ID, "Test Issue", "Body", false)
	comment, _ := store.CreateComment(issue.ID, user.ID, "Original comment")

	body := `{"body": "Updated comment"}`
	req := httptest.NewRequest("PATCH", "/repos/alice/test-repo/issues/comments/1", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer ghp_test")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Setup chi context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("owner", "alice")
	rctx.URLParams.Add("repo", "test-repo")
	rctx.URLParams.Add("comment_id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := plugin.requireAuth(plugin.updateComment)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["body"] != "Updated comment" {
		t.Fatalf("Expected body 'Updated comment', got %v", resp["body"])
	}

	// Verify in database
	updatedComment, _ := store.GetComment(comment.ID)
	if updatedComment.Body != "Updated comment" {
		t.Fatalf("Expected database body 'Updated comment', got %s", updatedComment.Body)
	}
}

func TestDeleteComment(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Create user, repo, issue, and comment
	user, _ := store.GetOrCreateUser("alice", "ghp_test")
	repo, _ := store.CreateRepository(user.ID, "test-repo", "", false)
	issue, _ := store.CreateIssue(repo.ID, user.ID, "Test Issue", "Body", false)
	comment, _ := store.CreateComment(issue.ID, user.ID, "Comment to delete")

	req := httptest.NewRequest("DELETE", "/repos/alice/test-repo/issues/comments/1", nil)
	req.Header.Set("Authorization", "Bearer ghp_test")
	w := httptest.NewRecorder()

	// Setup chi context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("owner", "alice")
	rctx.URLParams.Add("repo", "test-repo")
	rctx.URLParams.Add("comment_id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := plugin.requireAuth(plugin.deleteComment)
	handler(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("Expected 204, got %d: %s", w.Code, w.Body.String())
	}

	// Verify comment was deleted from database
	_, err := store.GetComment(comment.ID)
	if err == nil {
		t.Fatal("Expected error getting deleted comment")
	}
}

func TestCreateReview(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Create user, repo, and PR
	user, _ := store.GetOrCreateUser("alice", "ghp_test")
	repo, _ := store.CreateRepository(user.ID, "test-repo", "", false)
	issue, _, _ := store.CreatePullRequest(repo.ID, user.ID, "Test PR", "Body", "feature", "main")

	// Test creating PENDING review
	body := `{"state": "PENDING", "body": "Reviewing this PR"}`
	req := httptest.NewRequest("POST", "/repos/alice/test-repo/pulls/1/reviews", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer ghp_test")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Setup chi context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("owner", "alice")
	rctx.URLParams.Add("repo", "test-repo")
	rctx.URLParams.Add("number", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := plugin.requireAuth(plugin.createReview)
	handler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["state"] != "PENDING" {
		t.Fatalf("Expected state 'PENDING', got %v", resp["state"])
	}
	if resp["body"] != "Reviewing this PR" {
		t.Fatalf("Expected body 'Reviewing this PR', got %v", resp["body"])
	}
	if resp["user"].(map[string]interface{})["login"] != "alice" {
		t.Fatalf("Expected user 'alice', got %v", resp["user"])
	}
	// PENDING review should not have submitted_at
	if _, exists := resp["submitted_at"]; exists {
		t.Fatal("PENDING review should not have submitted_at")
	}

	// Verify in database
	reviews, _ := store.ListReviews(issue.ID)
	if len(reviews) != 1 {
		t.Fatalf("Expected 1 review in database, got %d", len(reviews))
	}
	if reviews[0].State != "PENDING" {
		t.Fatalf("Expected state 'PENDING', got '%s'", reviews[0].State)
	}
}

func TestCreateReviewWithDifferentStates(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Create user, repo, and PR
	user, _ := store.GetOrCreateUser("alice", "ghp_test")
	repo, _ := store.CreateRepository(user.ID, "test-repo", "", false)
	issue, _, _ := store.CreatePullRequest(repo.ID, user.ID, "Test PR", "Body", "feature", "main")

	states := []string{"APPROVED", "CHANGES_REQUESTED", "COMMENTED"}

	for i, state := range states {
		body := fmt.Sprintf(`{"state": "%s", "body": "Review %d"}`, state, i+1)
		req := httptest.NewRequest("POST", "/repos/alice/test-repo/pulls/1/reviews", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer ghp_test")
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		// Setup chi context
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("owner", "alice")
		rctx.URLParams.Add("repo", "test-repo")
		rctx.URLParams.Add("number", "1")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		handler := plugin.requireAuth(plugin.createReview)
		handler(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("Expected 201 for state %s, got %d: %s", state, w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)

		if resp["state"] != state {
			t.Fatalf("Expected state '%s', got %v", state, resp["state"])
		}
	}

	// Verify all reviews in database
	reviews, _ := store.ListReviews(issue.ID)
	if len(reviews) != 3 {
		t.Fatalf("Expected 3 reviews in database, got %d", len(reviews))
	}
}

func TestListReviews(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Create user, repo, PR, and reviews
	user, _ := store.GetOrCreateUser("alice", "ghp_test")
	repo, _ := store.CreateRepository(user.ID, "test-repo", "", false)
	issue, _, _ := store.CreatePullRequest(repo.ID, user.ID, "Test PR", "Body", "feature", "main")

	store.CreateReview(issue.ID, user.ID, "PENDING", "First review")
	store.CreateReview(issue.ID, user.ID, "APPROVED", "LGTM")
	store.CreateReview(issue.ID, user.ID, "CHANGES_REQUESTED", "Needs work")

	req := httptest.NewRequest("GET", "/repos/alice/test-repo/pulls/1/reviews", nil)
	req.Header.Set("Authorization", "Bearer ghp_test")
	w := httptest.NewRecorder()

	// Setup chi context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("owner", "alice")
	rctx.URLParams.Add("repo", "test-repo")
	rctx.URLParams.Add("number", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := plugin.requireAuth(plugin.listReviews)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if len(resp) != 3 {
		t.Fatalf("Expected 3 reviews, got %d", len(resp))
	}

	// Verify review states
	states := []string{"PENDING", "APPROVED", "CHANGES_REQUESTED"}
	for i, review := range resp {
		if review["state"] != states[i] {
			t.Fatalf("Expected state '%s' at index %d, got %v", states[i], i, review["state"])
		}
	}
}

func TestSubmitReview(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Create user, repo, PR, and PENDING review
	user, _ := store.GetOrCreateUser("alice", "ghp_test")
	repo, _ := store.CreateRepository(user.ID, "test-repo", "", false)
	issue, _, _ := store.CreatePullRequest(repo.ID, user.ID, "Test PR", "Body", "feature", "main")
	review, _ := store.CreateReview(issue.ID, user.ID, "PENDING", "Review body")

	// Verify review is PENDING and has no submitted_at
	if review.SubmittedAt != nil {
		t.Fatal("PENDING review should not have submitted_at")
	}

	body := `{}`
	req := httptest.NewRequest("PUT", "/repos/alice/test-repo/pulls/1/reviews/1", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer ghp_test")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Setup chi context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("owner", "alice")
	rctx.URLParams.Add("repo", "test-repo")
	rctx.URLParams.Add("number", "1")
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := plugin.requireAuth(plugin.submitReview)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	// Submitted review should have submitted_at
	if _, exists := resp["submitted_at"]; !exists {
		t.Fatal("Submitted review should have submitted_at")
	}

	// Verify in database
	updated, _ := store.GetReview(review.ID)
	if updated.SubmittedAt == nil {
		t.Fatal("Review should have submitted_at after submit")
	}
}

func TestDismissReview(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Create user, repo, PR, and APPROVED review
	user, _ := store.GetOrCreateUser("alice", "ghp_test")
	repo, _ := store.CreateRepository(user.ID, "test-repo", "", false)
	issue, _, _ := store.CreatePullRequest(repo.ID, user.ID, "Test PR", "Body", "feature", "main")
	review, _ := store.CreateReview(issue.ID, user.ID, "APPROVED", "LGTM")

	// Verify review is APPROVED
	if review.State != "APPROVED" {
		t.Fatalf("Expected state 'APPROVED', got '%s'", review.State)
	}
	if review.DismissedAt != nil {
		t.Fatal("Review should not have dismissed_at initially")
	}

	body := `{}`
	req := httptest.NewRequest("DELETE", "/repos/alice/test-repo/pulls/1/reviews/1", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer ghp_test")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Setup chi context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("owner", "alice")
	rctx.URLParams.Add("repo", "test-repo")
	rctx.URLParams.Add("number", "1")
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := plugin.requireAuth(plugin.dismissReview)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	// Dismissed review should have DISMISSED state
	if resp["state"] != "DISMISSED" {
		t.Fatalf("Expected state 'DISMISSED', got %v", resp["state"])
	}
	if _, exists := resp["dismissed_at"]; !exists {
		t.Fatal("Dismissed review should have dismissed_at")
	}

	// Verify in database
	updated, _ := store.GetReview(review.ID)
	if updated.State != "DISMISSED" {
		t.Fatalf("Expected state 'DISMISSED', got '%s'", updated.State)
	}
	if updated.DismissedAt == nil {
		t.Fatal("Review should have dismissed_at after dismiss")
	}
}
