// ABOUTME: Tests for GitHub API handlers
// ABOUTME: Tests repository, user, and resource CRUD endpoints

package github

import (
	"bytes"
	"context"
	"encoding/json"
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
