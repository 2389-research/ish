// ABOUTME: Tests for authentication middleware.
// ABOUTME: Verifies token parsing and user extraction from headers.

package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddleware_ExtractsUser(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		wantUser   string
	}{
		{"user prefix", "Bearer user:harper", "harper"},
		{"no header", "", "default"},
		{"empty bearer", "Bearer ", "default"},
		// ALL tokens (except user:*) map to eval-user for mock server data persistence
		{"simple token", "Bearer harper", "eval-user@example.com"},
		{"ish-mock access token", "Bearer ish-mock-access-12345", "eval-user@example.com"},
		{"ish-mock refresh token", "Bearer ish-mock-refresh-12345", "eval-user@example.com"},
		{"google-style token", "Bearer ya29.a0AfH6SMC...", "eval-user@example.com"},
		{"random token", "Bearer some-random-token", "eval-user@example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotUser string
			handler := Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotUser = UserFromContext(r.Context())
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest("GET", "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if gotUser != tt.wantUser {
				t.Errorf("UserFromContext() = %q, want %q", gotUser, tt.wantUser)
			}
		})
	}
}
