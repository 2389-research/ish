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
		{"simple token", "Bearer harper", "harper"},
		{"no header", "", "default"},
		{"empty bearer", "Bearer ", "default"},
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
