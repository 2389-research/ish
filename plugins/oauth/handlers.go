// ABOUTME: OAuth flow handlers for authorization, token exchange, and revocation.
// ABOUTME: Implements mock OAuth provider for testing.

package oauth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/2389/ish/internal/store"
	"github.com/go-chi/chi/v5"
)

// handleAuthorize handles GET /oauth/{plugin}/authorize
// Auto-approves and redirects with authorization code
func (p *OAuthPlugin) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	_ = chi.URLParam(r, "plugin") // Plugin name is in URL but not needed for mock flow
	redirectURI := r.URL.Query().Get("redirect_uri")
	state := r.URL.Query().Get("state")

	// Generate authorization code
	code := generateRandomToken("code")

	// Build redirect URL with code
	u, err := url.Parse(redirectURI)
	if err != nil {
		http.Error(w, "Invalid redirect_uri", http.StatusBadRequest)
		return
	}

	q := u.Query()
	q.Set("code", code)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()

	// Auto-approve: redirect immediately
	http.Redirect(w, r, u.String(), http.StatusFound)
}

// handleToken handles POST /oauth/{plugin}/token
// Exchanges authorization code or refresh token for access token
func (p *OAuthPlugin) handleToken(w http.ResponseWriter, r *http.Request) {
	pluginName := chi.URLParam(r, "plugin")

	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	grantType := r.FormValue("grant_type")
	if grantType == "" {
		http.Error(w, "Missing grant_type", http.StatusBadRequest)
		return
	}

	var accessToken, refreshToken string

	switch grantType {
	case "authorization_code":
		code := r.FormValue("code")
		if code == "" {
			http.Error(w, "Missing code", http.StatusBadRequest)
			return
		}

		// Generate tokens
		accessToken = generateRandomToken("token")
		refreshToken = generateRandomToken("refresh")

	case "refresh_token":
		rt := r.FormValue("refresh_token")
		if rt == "" {
			http.Error(w, "Missing refresh_token", http.StatusBadRequest)
			return
		}

		// Generate new access token, keep same refresh token
		accessToken = generateRandomToken("token")
		refreshToken = rt

	default:
		http.Error(w, "Unsupported grant_type", http.StatusBadRequest)
		return
	}

	// Store token in database
	token := &store.OAuthToken{
		Token:        accessToken,
		PluginName:   pluginName,
		UserID:       "auto_user", // Could extract from code/state in real impl
		Scopes:       "email profile openid",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
		RefreshToken: refreshToken,
		Revoked:      false,
	}

	if err := p.store.StoreToken(token); err != nil {
		http.Error(w, "Failed to store token", http.StatusInternalServerError)
		return
	}

	// Return OAuth token response
	response := map[string]interface{}{
		"access_token":  accessToken,
		"token_type":    "Bearer",
		"expires_in":    3600,
		"refresh_token": refreshToken,
		"scope":         "email profile openid",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleRevoke handles POST /oauth/{plugin}/revoke
// Revokes a token
func (p *OAuthPlugin) handleRevoke(w http.ResponseWriter, r *http.Request) {
	var token string

	// Support both form data and JSON
	contentType := r.Header.Get("Content-Type")
	if contentType == "application/json" {
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		token = body["token"]
	} else {
		// Parse form data
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}
		token = r.FormValue("token")
	}

	if token == "" {
		http.Error(w, "Missing token parameter", http.StatusBadRequest)
		return
	}

	// Revoke the token (ignore errors if token doesn't exist)
	p.store.RevokeToken(token)

	// Return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// generateRandomToken generates a random token with a prefix
func generateRandomToken(prefix string) string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(b))
}
