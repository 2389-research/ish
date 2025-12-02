// ABOUTME: Authentication middleware for fake Google API requests.
// ABOUTME: Parses Bearer tokens and extracts user identity for request context.

package auth

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const userContextKey contextKey = "user"

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := extractUser(r.Header.Get("Authorization"))
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserFromContext(ctx context.Context) string {
	user, ok := ctx.Value(userContextKey).(string)
	if !ok || user == "" {
		return "default"
	}
	return user
}

func extractUser(authHeader string) string {
	if authHeader == "" {
		return "default"
	}

	// Remove "Bearer " prefix
	token := strings.TrimPrefix(authHeader, "Bearer ")
	token = strings.TrimSpace(token)

	if token == "" {
		return "default"
	}

	// Check for "user:" prefix
	if strings.HasPrefix(token, "user:") {
		return strings.TrimPrefix(token, "user:")
	}

	return token
}
