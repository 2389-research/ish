// ABOUTME: Tests for GitHub authentication middleware
// ABOUTME: Tests token extraction and validation

package github

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractToken(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected string
		hasToken bool
	}{
		{"Bearer format", "Bearer ghp_test123", "ghp_test123", true},
		{"Token format", "token ghp_test123", "ghp_test123", true},
		{"No auth header", "", "", false},
		{"Invalid format", "Basic abc123", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			token, ok := extractToken(req)
			if ok != tt.hasToken {
				t.Fatalf("Expected hasToken=%v, got %v", tt.hasToken, ok)
			}
			if token != tt.expected {
				t.Fatalf("Expected token '%s', got '%s'", tt.expected, token)
			}
		})
	}
}

func TestRequireAuth(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	store, _ := NewGitHubStore(db)

	plugin := &GitHubPlugin{store: store}

	// Create user with token
	store.GetOrCreateUser("alice", "ghp_valid")

	handler := plugin.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(userContextKey).(*User)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(user.Login))
	})

	t.Run("Valid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer ghp_valid")
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d", w.Code)
		}
		if w.Body.String() != "alice" {
			t.Fatalf("Expected 'alice', got '%s'", w.Body.String())
		}
	})

	t.Run("Missing token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("Expected 401, got %d", w.Code)
		}
	})

	t.Run("Invalid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Authorization", "Bearer ghp_invalid")
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("Expected 401, got %d", w.Code)
		}
	})
}
