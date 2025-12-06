// ABOUTME: Tests for OAuth flow handlers.
// ABOUTME: Verifies authorization, token exchange, and revocation endpoints.

package oauth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestHandleAuthorize(t *testing.T) {
	s, cleanup := createTestStore(t)
	defer cleanup()

	p := &OAuthPlugin{store: s}
	r := chi.NewRouter()
	p.RegisterAuth(r)

	tests := []struct {
		name           string
		redirectURI    string
		state          string
		wantStatusCode int
		wantLocation   bool
	}{
		{
			name:           "valid authorization request",
			redirectURI:    "http://localhost:9001/callback",
			state:          "test_state_123",
			wantStatusCode: http.StatusFound,
			wantLocation:   true,
		},
		{
			name:           "authorization without state",
			redirectURI:    "http://localhost:9001/callback",
			state:          "",
			wantStatusCode: http.StatusFound,
			wantLocation:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build URL with query parameters
			u := "/oauth/google/authorize?redirect_uri=" + url.QueryEscape(tt.redirectURI)
			if tt.state != "" {
				u += "&state=" + url.QueryEscape(tt.state)
			}

			req := httptest.NewRequest("GET", u, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("Status code = %d, want %d", w.Code, tt.wantStatusCode)
			}

			if tt.wantLocation {
				location := w.Header().Get("Location")
				if location == "" {
					t.Error("Expected Location header")
				}

				// Parse the redirect URL
				redirectURL, err := url.Parse(location)
				if err != nil {
					t.Fatalf("Failed to parse redirect URL: %v", err)
				}

				// Check for authorization code
				code := redirectURL.Query().Get("code")
				if code == "" {
					t.Error("Expected 'code' parameter in redirect URL")
				}

				// Check state is preserved
				if tt.state != "" {
					state := redirectURL.Query().Get("state")
					if state != tt.state {
						t.Errorf("State = %q, want %q", state, tt.state)
					}
				}
			}
		})
	}
}

func TestHandleToken(t *testing.T) {
	s, cleanup := createTestStore(t)
	defer cleanup()

	p := &OAuthPlugin{store: s}
	r := chi.NewRouter()
	p.RegisterAuth(r)

	tests := []struct {
		name           string
		grantType      string
		code           string
		wantStatusCode int
		wantToken      bool
	}{
		{
			name:           "valid token exchange",
			grantType:      "authorization_code",
			code:           "valid_code_123",
			wantStatusCode: http.StatusOK,
			wantToken:      true,
		},
		{
			name:           "missing grant type",
			grantType:      "",
			code:           "valid_code_123",
			wantStatusCode: http.StatusBadRequest,
			wantToken:      false,
		},
		{
			name:           "missing code",
			grantType:      "authorization_code",
			code:           "",
			wantStatusCode: http.StatusBadRequest,
			wantToken:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build form data
			form := url.Values{}
			if tt.grantType != "" {
				form.Set("grant_type", tt.grantType)
			}
			if tt.code != "" {
				form.Set("code", tt.code)
			}

			req := httptest.NewRequest("POST", "/oauth/google/token", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("Status code = %d, want %d", w.Code, tt.wantStatusCode)
			}

			if tt.wantToken {
				var resp map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				accessToken, ok := resp["access_token"].(string)
				if !ok || accessToken == "" {
					t.Error("Expected 'access_token' in response")
				}

				refreshToken, ok := resp["refresh_token"].(string)
				if !ok || refreshToken == "" {
					t.Error("Expected 'refresh_token' in response")
				}

				tokenType, ok := resp["token_type"].(string)
				if !ok || tokenType != "Bearer" {
					t.Errorf("token_type = %q, want %q", tokenType, "Bearer")
				}

				expiresIn, ok := resp["expires_in"].(float64)
				if !ok || expiresIn != 3600 {
					t.Errorf("expires_in = %v, want 3600", expiresIn)
				}

				// Verify token was stored in database
				storedToken, err := s.GetToken(accessToken)
				if err != nil {
					t.Errorf("Token not found in database: %v", err)
				}
				if storedToken.PluginName != "google" {
					t.Errorf("Token plugin_name = %q, want %q", storedToken.PluginName, "google")
				}
			}
		})
	}
}

func TestHandleTokenRefresh(t *testing.T) {
	s, cleanup := createTestStore(t)
	defer cleanup()

	p := &OAuthPlugin{store: s}
	r := chi.NewRouter()
	p.RegisterAuth(r)

	// First, get a token with refresh token
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", "initial_code")

	req := httptest.NewRequest("POST", "/oauth/google/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var initialResp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&initialResp)
	refreshToken := initialResp["refresh_token"].(string)

	// Now use refresh token to get a new access token
	refreshForm := url.Values{}
	refreshForm.Set("grant_type", "refresh_token")
	refreshForm.Set("refresh_token", refreshToken)

	refreshReq := httptest.NewRequest("POST", "/oauth/google/token", strings.NewReader(refreshForm.Encode()))
	refreshReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	refreshW := httptest.NewRecorder()
	r.ServeHTTP(refreshW, refreshReq)

	if refreshW.Code != http.StatusOK {
		t.Errorf("Refresh token exchange status = %d, want %d", refreshW.Code, http.StatusOK)
	}

	var refreshResp map[string]interface{}
	if err := json.NewDecoder(refreshW.Body).Decode(&refreshResp); err != nil {
		t.Fatalf("Failed to decode refresh response: %v", err)
	}

	newAccessToken, ok := refreshResp["access_token"].(string)
	if !ok || newAccessToken == "" {
		t.Error("Expected new 'access_token' in refresh response")
	}

	// Verify new token is stored
	storedToken, err := s.GetToken(newAccessToken)
	if err != nil {
		t.Errorf("New token not found in database: %v", err)
	}
	if storedToken.PluginName != "google" {
		t.Errorf("New token plugin_name = %q, want %q", storedToken.PluginName, "google")
	}
}

func TestHandleRevoke(t *testing.T) {
	s, cleanup := createTestStore(t)
	defer cleanup()

	p := &OAuthPlugin{store: s}
	r := chi.NewRouter()
	p.RegisterAuth(r)

	// First, create a token to revoke
	token := &OAuthToken{
		Token:      "token_to_revoke",
		PluginName: "google",
		UserID:     "test_user",
		Scopes:     "email profile",
		Revoked:    false,
	}
	if err := s.StoreToken(token); err != nil {
		t.Fatalf("Failed to store token: %v", err)
	}

	tests := []struct {
		name           string
		token          string
		wantStatusCode int
	}{
		{
			name:           "revoke existing token",
			token:          "token_to_revoke",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "revoke non-existent token (should succeed)",
			token:          "non_existent_token",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "missing token parameter",
			token:          "",
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{}
			if tt.token != "" {
				form.Set("token", tt.token)
			}

			req := httptest.NewRequest("POST", "/oauth/google/revoke", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("Status code = %d, want %d", w.Code, tt.wantStatusCode)
			}

			if tt.wantStatusCode == http.StatusOK && tt.token == "token_to_revoke" {
				// Verify token was actually revoked
				storedToken, err := s.GetToken(tt.token)
				if err != nil {
					t.Fatalf("Failed to get token: %v", err)
				}
				if !storedToken.Revoked {
					t.Error("Token should be marked as revoked")
				}
			}
		})
	}
}

func TestHandleRevokeJSON(t *testing.T) {
	s, cleanup := createTestStore(t)
	defer cleanup()

	p := &OAuthPlugin{store: s}
	r := chi.NewRouter()
	p.RegisterAuth(r)

	// Create a token to revoke
	token := &OAuthToken{
		Token:      "json_revoke_token",
		PluginName: "google",
		UserID:     "test_user",
		Scopes:     "email profile",
		Revoked:    false,
	}
	if err := s.StoreToken(token); err != nil {
		t.Fatalf("Failed to store token: %v", err)
	}

	// Test JSON body
	body := map[string]string{"token": "json_revoke_token"}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/oauth/google/revoke", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
	}

	// Verify token was revoked
	storedToken, err := s.GetToken("json_revoke_token")
	if err != nil {
		t.Fatalf("Failed to get token: %v", err)
	}
	if !storedToken.Revoked {
		t.Error("Token should be marked as revoked")
	}
}
