// ABOUTME: Twilio SMS and Voice API plugin for ISH
// ABOUTME: Simulates Twilio Messaging and Voice APIs with webhook callbacks

package twilio

import (
	"context"
	"database/sql"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/2389/ish/plugins/core"
	"github.com/go-chi/chi/v5"
)

func init() {
	core.Register(&TwilioPlugin{})
}

type TwilioPlugin struct {
	store *TwilioStore
}

func (p *TwilioPlugin) Name() string {
	return "twilio"
}

func (p *TwilioPlugin) Health() core.HealthStatus {
	return core.HealthStatus{
		Status:  "healthy",
		Message: "Twilio plugin operational",
	}
}

func (p *TwilioPlugin) RegisterRoutes(r chi.Router) {
	// SMS API
	r.Route("/2010-04-01/Accounts/{AccountSid}/Messages.json", func(r chi.Router) {
		r.Post("/", p.requireAuth(p.sendMessage))
		r.Get("/", p.requireAuth(p.listMessages))
	})
	r.Get("/2010-04-01/Accounts/{AccountSid}/Messages/{MessageSid}.json", p.requireAuth(p.getMessage))

	// Voice API
	r.Route("/2010-04-01/Accounts/{AccountSid}/Calls.json", func(r chi.Router) {
		r.Post("/", p.requireAuth(p.initiateCall))
		r.Get("/", p.requireAuth(p.listCalls))
	})
	r.Get("/2010-04-01/Accounts/{AccountSid}/Calls/{CallSid}.json", p.requireAuth(p.getCall))

	// Phone Numbers API
	r.Get("/2010-04-01/Accounts/{AccountSid}/IncomingPhoneNumbers.json", p.requireAuth(p.listPhoneNumbers))
}

func (p *TwilioPlugin) RegisterAuth(r chi.Router) {
	// Twilio uses HTTP Basic Auth, handled per-request
}


// extractBasicAuth extracts username and password from HTTP Basic Auth header
func extractBasicAuth(authHeader string) (username, password string, ok bool) {
	const prefix = "Basic "
	if !strings.HasPrefix(authHeader, prefix) {
		return "", "", false
	}

	decoded, err := base64.StdEncoding.DecodeString(authHeader[len(prefix):])
	if err != nil {
		return "", "", false
	}

	credentials := strings.SplitN(string(decoded), ":", 2)
	if len(credentials) != 2 {
		return "", "", false
	}

	return credentials[0], credentials[1], true
}

// requireAuth middleware validates HTTP Basic Auth (Account SID + Auth Token)
func (p *TwilioPlugin) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"code": 20003, "message": "Authenticate"}`, http.StatusUnauthorized)
			return
		}

		accountSid, authToken, ok := extractBasicAuth(authHeader)
		if !ok {
			http.Error(w, `{"code": 20003, "message": "Authenticate"}`, http.StatusUnauthorized)
			return
		}

		// Auto-create account if it doesn't exist
		if _, err := p.store.GetOrCreateAccount(accountSid); err != nil {
			http.Error(w, `{"code": 20005, "message": "Internal server error"}`, http.StatusInternalServerError)
			return
		}

		// Validate credentials
		if !p.store.ValidateAccount(accountSid, authToken) {
			http.Error(w, `{"code": 20003, "message": "Authenticate"}`, http.StatusUnauthorized)
			return
		}

		// Store account SID in context for handlers
		ctx := r.Context()
		ctx = context.WithValue(ctx, "account_sid", accountSid)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func (p *TwilioPlugin) ValidateToken(token string) bool {
	// Token validation happens per-request via HTTP Basic Auth
	// This method is not used for Twilio auth pattern
	return false
}

func (p *TwilioPlugin) SetDB(db *sql.DB) error {
	store, err := NewTwilioStore(db)
	if err != nil {
		return err
	}
	p.store = store

	// Start webhook worker
	go p.StartWebhookWorker(context.Background())

	return nil
}
