// ABOUTME: Google OAuth mock handlers for token exchange and validation.
// ABOUTME: Enables jeff and other clients to authenticate against ish mock server.

package google

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Google OAuth scopes used for Gmail, Calendar, and Tasks
const googleOAuthScopes = "https://www.googleapis.com/auth/gmail.modify https://www.googleapis.com/auth/calendar https://www.googleapis.com/auth/tasks"

// registerOAuthRoutes registers Google OAuth mock endpoints
func (p *GooglePlugin) registerOAuthRoutes(r chi.Router) {
	r.Post("/oauth2/token", p.handleOAuthToken)
	r.Get("/oauth2/v3/tokeninfo", p.handleTokenInfo)
	r.Get("/oauth2/v3/userinfo", p.handleUserInfo)
}

// handleOAuthToken handles POST /oauth2/token
// Accepts OAuth token requests and returns valid-looking tokens for mock auth
func (p *GooglePlugin) handleOAuthToken(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	grantType := r.FormValue("grant_type")
	if grantType == "" {
		writeJSONError(w, http.StatusBadRequest, "Missing grant_type")
		return
	}

	var accessToken, refreshToken string

	switch grantType {
	case "refresh_token":
		rt := r.FormValue("refresh_token")
		if rt == "" {
			writeJSONError(w, http.StatusBadRequest, "Missing refresh_token")
			return
		}
		accessToken = generateMockToken("access")
		refreshToken = generateMockToken("refresh")

	case "authorization_code":
		code := r.FormValue("code")
		if code == "" {
			writeJSONError(w, http.StatusBadRequest, "Missing code")
			return
		}
		accessToken = generateMockToken("access")
		refreshToken = generateMockToken("refresh")

	default:
		writeJSONError(w, http.StatusBadRequest, "Unsupported grant_type: "+grantType)
		return
	}

	response := map[string]interface{}{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    3600,
		"refresh_token": refreshToken,
		"scope":         googleOAuthScopes,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleTokenInfo handles GET /oauth2/v3/tokeninfo?access_token=xxx
// Validates tokens and returns user info for mock auth
func (p *GooglePlugin) handleTokenInfo(w http.ResponseWriter, r *http.Request) {
	accessToken := r.URL.Query().Get("access_token")
	if accessToken == "" {
		writeJSONError(w, http.StatusBadRequest, "Missing access_token parameter")
		return
	}

	// For mock purposes, accept any token and return valid info
	response := map[string]interface{}{
		"email":          "eval-user@example.com",
		"email_verified": "true",
		"expires_in":     "3600",
		"scope":          googleOAuthScopes,
		"access_type":    "offline",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleUserInfo handles GET /oauth2/v3/userinfo
// Returns user profile info for mock auth
func (p *GooglePlugin) handleUserInfo(w http.ResponseWriter, r *http.Request) {
	// For mock purposes, return a valid user regardless of token
	response := map[string]interface{}{
		"email":          "eval-user@example.com",
		"email_verified": true,
		"name":           "Eval User",
		"given_name":     "Eval",
		"family_name":    "User",
		"picture":        "https://example.com/avatar.png",
		"sub":            "eval-user-12345",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// generateMockToken generates a random mock token with ish prefix
func generateMockToken(prefix string) string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("failed to generate random token: %v", err))
	}
	return fmt.Sprintf("ish-mock-%s-%s", prefix, hex.EncodeToString(b))
}

// writeJSONError writes an error response in JSON format
func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error":             "invalid_request",
		"error_description": message,
	})
}
