// ABOUTME: Tests for Google OAuth mock handlers.
// ABOUTME: Verifies token exchange, tokeninfo, and userinfo endpoints work correctly.

package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestOAuthTokenEndpoint(t *testing.T) {
	p := &GooglePlugin{}
	r := chi.NewRouter()
	p.RegisterAuth(r)

	tests := []struct {
		name           string
		grantType      string
		refreshToken   string
		wantStatus     int
		wantTokenType  string
		wantHasAccess  bool
		wantHasRefresh bool
	}{
		{
			name:           "refresh_token grant returns tokens",
			grantType:      "refresh_token",
			refreshToken:   "test-refresh-token",
			wantStatus:     http.StatusOK,
			wantTokenType:  "Bearer",
			wantHasAccess:  true,
			wantHasRefresh: true,
		},
		{
			name:           "authorization_code grant returns tokens",
			grantType:      "authorization_code",
			refreshToken:   "",
			wantStatus:     http.StatusOK,
			wantTokenType:  "Bearer",
			wantHasAccess:  true,
			wantHasRefresh: true,
		},
		{
			name:       "missing grant_type returns error",
			grantType:  "",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{}
			if tt.grantType != "" {
				form.Set("grant_type", tt.grantType)
			}
			if tt.refreshToken != "" {
				form.Set("refresh_token", tt.refreshToken)
			}
			if tt.grantType == "authorization_code" {
				form.Set("code", "test-auth-code")
			}

			req := httptest.NewRequest("POST", "/oauth2/token", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var resp map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if tokenType, ok := resp["token_type"].(string); !ok || tokenType != tt.wantTokenType {
					t.Errorf("got token_type %v, want %s", resp["token_type"], tt.wantTokenType)
				}

				if tt.wantHasAccess {
					if _, ok := resp["access_token"].(string); !ok {
						t.Error("response missing access_token")
					}
				}

				if tt.wantHasRefresh {
					if _, ok := resp["refresh_token"].(string); !ok {
						t.Error("response missing refresh_token")
					}
				}

				if _, ok := resp["expires_in"].(float64); !ok {
					t.Error("response missing expires_in")
				}

				if _, ok := resp["scope"].(string); !ok {
					t.Error("response missing scope")
				}
			}
		})
	}
}

func TestOAuthTokenInfoEndpoint(t *testing.T) {
	p := &GooglePlugin{}
	r := chi.NewRouter()
	p.RegisterAuth(r)

	tests := []struct {
		name        string
		accessToken string
		wantStatus  int
		wantEmail   bool
	}{
		{
			name:        "valid token returns info",
			accessToken: "ish-mock-token-12345",
			wantStatus:  http.StatusOK,
			wantEmail:   true,
		},
		{
			name:        "missing token returns error",
			accessToken: "",
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/oauth2/v3/tokeninfo"
			if tt.accessToken != "" {
				url += "?access_token=" + tt.accessToken
			}

			req := httptest.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var resp map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if tt.wantEmail {
					if _, ok := resp["email"].(string); !ok {
						t.Error("response missing email")
					}
				}

				if _, ok := resp["expires_in"].(string); !ok {
					t.Error("response missing expires_in")
				}

				if _, ok := resp["scope"].(string); !ok {
					t.Error("response missing scope")
				}
			}
		})
	}
}

func TestOAuthUserInfoEndpoint(t *testing.T) {
	p := &GooglePlugin{}
	r := chi.NewRouter()
	p.RegisterAuth(r)

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{
			name:       "with bearer token returns user info",
			authHeader: "Bearer ish-mock-token-12345",
			wantStatus: http.StatusOK,
		},
		{
			name:       "without auth header returns user info (permissive mode)",
			authHeader: "",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/oauth2/v3/userinfo", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var resp map[string]interface{}
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if _, ok := resp["email"].(string); !ok {
					t.Error("response missing email")
				}

				if _, ok := resp["name"].(string); !ok {
					t.Error("response missing name")
				}
			}
		})
	}
}
