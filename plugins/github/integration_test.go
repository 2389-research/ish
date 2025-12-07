// ABOUTME: Integration tests for GitHub plugin end-to-end workflows
// ABOUTME: Tests full issue lifecycle, PR workflow, and webhook delivery

package github

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

// TestFullIssueLifecycle tests the complete lifecycle of an issue from creation to closure
func TestFullIssueLifecycle(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Step 1: Create user + token
	user, err := store.GetOrCreateUser("alice", "ghp_integration_test")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	if user.Login != "alice" {
		t.Fatalf("Expected login 'alice', got '%s'", user.Login)
	}

	// Step 2: Create repository
	repo, err := store.CreateRepository(user.ID, "test-repo", "Integration test repository", false)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	if repo.Name != "test-repo" {
		t.Fatalf("Expected repo name 'test-repo', got '%s'", repo.Name)
	}

	// Step 3: Create issue via API
	issueBody := `{"title": "Bug: Login fails", "body": "Users cannot log in with valid credentials"}`
	req := httptest.NewRequest("POST", "/repos/alice/test-repo/issues", bytes.NewBufferString(issueBody))
	req.Header.Set("Authorization", "Bearer ghp_integration_test")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("owner", "alice")
	rctx.URLParams.Add("repo", "test-repo")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := plugin.requireAuth(plugin.createIssue)
	handler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create issue: %d - %s", w.Code, w.Body.String())
	}

	var issueResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &issueResp)

	if issueResp["number"] != float64(1) {
		t.Fatalf("Expected issue number 1, got %v", issueResp["number"])
	}
	if issueResp["state"] != "open" {
		t.Fatalf("Expected issue state 'open', got %v", issueResp["state"])
	}

	// Step 4: Add comment to issue
	commentBody := `{"body": "I can reproduce this issue. It happens when using special characters in password."}`
	req = httptest.NewRequest("POST", "/repos/alice/test-repo/issues/1/comments", bytes.NewBufferString(commentBody))
	req.Header.Set("Authorization", "Bearer ghp_integration_test")
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("owner", "alice")
	rctx.URLParams.Add("repo", "test-repo")
	rctx.URLParams.Add("number", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler = plugin.requireAuth(plugin.createComment)
	handler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create comment: %d - %s", w.Code, w.Body.String())
	}

	var commentResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &commentResp)

	if commentResp["body"] == nil {
		t.Fatal("Comment should have a body")
	}

	// Step 5: Update issue state to closed
	updateBody := `{"state": "closed", "state_reason": "completed"}`
	req = httptest.NewRequest("PATCH", "/repos/alice/test-repo/issues/1", bytes.NewBufferString(updateBody))
	req.Header.Set("Authorization", "Bearer ghp_integration_test")
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("owner", "alice")
	rctx.URLParams.Add("repo", "test-repo")
	rctx.URLParams.Add("number", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler = plugin.requireAuth(plugin.updateIssue)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to update issue: %d - %s", w.Code, w.Body.String())
	}

	var updateResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &updateResp)

	if updateResp["state"] != "closed" {
		t.Fatalf("Expected issue state 'closed', got %v", updateResp["state"])
	}
	if updateResp["state_reason"] != "completed" {
		t.Fatalf("Expected state_reason 'completed', got %v", updateResp["state_reason"])
	}
	if updateResp["closed_at"] == nil {
		t.Fatal("closed_at should be set")
	}

	// Step 6: Verify all operations succeeded by querying database
	issue, err := store.GetIssueByNumber(repo.ID, 1)
	if err != nil {
		t.Fatalf("Failed to get issue from database: %v", err)
	}
	if issue.State != "closed" {
		t.Fatalf("Database issue state should be 'closed', got '%s'", issue.State)
	}
	if issue.CommentsCount != 1 {
		t.Fatalf("Expected 1 comment, got %d", issue.CommentsCount)
	}
	if issue.ClosedAt == nil {
		t.Fatal("Issue should have ClosedAt timestamp")
	}

	t.Log("Full issue lifecycle completed successfully")
}

// TestFullPullRequestWorkflow tests the complete PR workflow from creation to merge
func TestFullPullRequestWorkflow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Step 1: Create user + repo
	user, err := store.GetOrCreateUser("bob", "ghp_pr_test")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	repo, err := store.CreateRepository(user.ID, "pr-repo", "Repository for PR testing", false)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Step 2: Create pull request
	prBody := `{"title": "Add new feature", "body": "This PR adds a new authentication feature", "head": "feature/auth", "base": "main"}`
	req := httptest.NewRequest("POST", "/repos/bob/pr-repo/pulls", bytes.NewBufferString(prBody))
	req.Header.Set("Authorization", "Bearer ghp_pr_test")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("owner", "bob")
	rctx.URLParams.Add("repo", "pr-repo")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := plugin.requireAuth(plugin.createPullRequest)
	handler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create PR: %d - %s", w.Code, w.Body.String())
	}

	var prResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &prResp)

	if prResp["number"] != float64(1) {
		t.Fatalf("Expected PR number 1, got %v", prResp["number"])
	}
	if prResp["state"] != "open" {
		t.Fatalf("Expected PR state 'open', got %v", prResp["state"])
	}
	if prResp["merged"] != false {
		t.Fatalf("Expected merged false, got %v", prResp["merged"])
	}

	// Step 3: Add review (APPROVED)
	reviewBody := `{"state": "APPROVED", "body": "Looks good to me! All tests passing."}`
	req = httptest.NewRequest("POST", "/repos/bob/pr-repo/pulls/1/reviews", bytes.NewBufferString(reviewBody))
	req.Header.Set("Authorization", "Bearer ghp_pr_test")
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("owner", "bob")
	rctx.URLParams.Add("repo", "pr-repo")
	rctx.URLParams.Add("number", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler = plugin.requireAuth(plugin.createReview)
	handler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create review: %d - %s", w.Code, w.Body.String())
	}

	var reviewResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &reviewResp)

	if reviewResp["state"] != "APPROVED" {
		t.Fatalf("Expected review state 'APPROVED', got %v", reviewResp["state"])
	}

	// Step 4: Add comment
	commentBody := `{"body": "Thanks for the review! I addressed all the feedback."}`
	req = httptest.NewRequest("POST", "/repos/bob/pr-repo/issues/1/comments", bytes.NewBufferString(commentBody))
	req.Header.Set("Authorization", "Bearer ghp_pr_test")
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("owner", "bob")
	rctx.URLParams.Add("repo", "pr-repo")
	rctx.URLParams.Add("number", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler = plugin.requireAuth(plugin.createComment)
	handler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create comment: %d - %s", w.Code, w.Body.String())
	}

	// Step 5: Merge PR
	mergeBody := `{"commit_message": "Merge pull request #1 from feature/auth"}`
	req = httptest.NewRequest("PUT", "/repos/bob/pr-repo/pulls/1/merge", bytes.NewBufferString(mergeBody))
	req.Header.Set("Authorization", "Bearer ghp_pr_test")
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("owner", "bob")
	rctx.URLParams.Add("repo", "pr-repo")
	rctx.URLParams.Add("number", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler = plugin.requireAuth(plugin.mergePullRequest)
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to merge PR: %d - %s", w.Code, w.Body.String())
	}

	var mergeResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &mergeResp)

	if mergeResp["merged"] != true {
		t.Fatalf("Expected merged true, got %v", mergeResp["merged"])
	}

	// Step 6: Verify PR is merged and closed
	issue, pr, err := store.GetPullRequest(repo.ID, 1)
	if err != nil {
		t.Fatalf("Failed to get PR from database: %v", err)
	}

	// Verify issue is closed
	if issue.State != "closed" {
		t.Fatalf("Expected issue state 'closed', got '%s'", issue.State)
	}
	if issue.ClosedAt == nil {
		t.Fatal("Issue should have ClosedAt timestamp")
	}

	// Verify PR is merged
	if !pr.Merged {
		t.Fatal("PR should be marked as merged")
	}
	if pr.MergedAt == nil {
		t.Fatal("PR should have MergedAt timestamp")
	}
	if pr.MergedByID == nil {
		t.Fatal("PR should have MergedByID set")
	}
	if *pr.MergedByID != user.ID {
		t.Fatalf("Expected MergedByID %d, got %d", user.ID, *pr.MergedByID)
	}

	// Verify comment count
	if issue.CommentsCount != 1 {
		t.Fatalf("Expected 1 comment, got %d", issue.CommentsCount)
	}

	t.Log("Full PR workflow completed successfully")
}

// TestWebhookDelivery tests webhook creation and delivery logging
func TestWebhookDelivery(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)
	plugin := &GitHubPlugin{store: store}

	// Step 1: Create repo + webhook
	user, err := store.GetOrCreateUser("charlie", "ghp_webhook_test")
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	_, err = store.CreateRepository(user.ID, "webhook-repo", "Repository with webhooks", false)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create webhook via API with external URL (SSRF protection allows this)
	// Using example.com which is a real, resolvable domain
	webhookBody := `{
		"config": {
			"url": "https://example.com/github",
			"content_type": "json",
			"secret": "test-secret-123"
		},
		"events": ["issues", "pull_request"]
	}`

	req := httptest.NewRequest("POST", "/repos/charlie/webhook-repo/hooks", bytes.NewBufferString(webhookBody))
	req.Header.Set("Authorization", "Bearer ghp_webhook_test")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("owner", "charlie")
	rctx.URLParams.Add("repo", "webhook-repo")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := plugin.requireAuth(plugin.createWebhook)
	handler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create webhook: %d - %s", w.Code, w.Body.String())
	}

	var webhookResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &webhookResp)

	webhookID := int64(webhookResp["id"].(float64))

	// Verify webhook was created with correct configuration
	if webhookResp["id"] == nil {
		t.Fatal("Webhook should have an ID")
	}

	config := webhookResp["config"].(map[string]interface{})
	if config["url"] != "https://example.com/github" {
		t.Fatalf("Expected URL 'https://example.com/github', got %v", config["url"])
	}
	if config["content_type"] != "json" {
		t.Fatalf("Expected content_type 'json', got %v", config["content_type"])
	}

	events := webhookResp["events"].([]interface{})
	if len(events) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(events))
	}

	// Step 2: Test webhook delivery would be triggered (we'll verify the database structure)
	// Create issue to test webhook triggering mechanism
	issueBody := `{"title": "Test webhook issue", "body": "This tests webhook infrastructure"}`
	req = httptest.NewRequest("POST", "/repos/charlie/webhook-repo/issues", bytes.NewBufferString(issueBody))
	req.Header.Set("Authorization", "Bearer ghp_webhook_test")
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	rctx = chi.NewRouteContext()
	rctx.URLParams.Add("owner", "charlie")
	rctx.URLParams.Add("repo", "webhook-repo")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler = plugin.requireAuth(plugin.createIssue)
	handler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Failed to create issue: %d - %s", w.Code, w.Body.String())
	}

	// Step 3: Wait for async webhook delivery to complete (with polling)
	// Poll for webhook delivery for up to 1 second
	var deliveryCount int
	maxAttempts := 20
	for attempt := 0; attempt < maxAttempts; attempt++ {
		err = db.QueryRow("SELECT COUNT(*) FROM github_webhook_deliveries WHERE webhook_id = ?", webhookID).Scan(&deliveryCount)
		if err != nil {
			t.Fatalf("Failed to query webhook deliveries: %v", err)
		}
		if deliveryCount == 1 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if deliveryCount != 1 {
		t.Fatalf("Expected 1 webhook delivery logged, got %d (waited %d ms)", deliveryCount, maxAttempts*50)
	}

	// Step 4: Verify delivery contains event information
	var eventType, payloadJSON string
	var statusCode int
	err = db.QueryRow(`
		SELECT event_type, payload, status_code
		FROM github_webhook_deliveries
		WHERE webhook_id = ?
		LIMIT 1
	`, webhookID).Scan(&eventType, &payloadJSON, &statusCode)

	if err != nil {
		t.Fatalf("Failed to get webhook delivery details: %v", err)
	}

	if eventType != "issues" {
		t.Fatalf("Expected event type 'issues', got '%s'", eventType)
	}

	// Parse and verify payload contains issue data
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		t.Fatalf("Failed to parse webhook payload: %v", err)
	}

	if payload["action"] == nil {
		t.Fatal("Webhook payload should contain 'action' field")
	}

	if payload["issue"] == nil {
		t.Fatal("Webhook payload should contain 'issue' field")
	}

	issueData := payload["issue"].(map[string]interface{})
	if issueData["title"] != "Test webhook issue" {
		t.Fatalf("Expected issue title 'Test webhook issue', got %v", issueData["title"])
	}

	if payload["repository"] == nil {
		t.Fatal("Webhook payload should contain 'repository' field")
	}

	repoData := payload["repository"].(map[string]interface{})
	if repoData["name"] != "webhook-repo" {
		t.Fatalf("Expected repo name 'webhook-repo', got %v", repoData["name"])
	}

	t.Log("Webhook delivery test completed successfully")
}
