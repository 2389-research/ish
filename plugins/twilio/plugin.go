// ABOUTME: Twilio SMS and Voice API plugin for ISH
// ABOUTME: Simulates Twilio Messaging and Voice APIs with webhook callbacks

package twilio

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/2389/ish/plugins/core"
	"github.com/go-chi/chi/v5"
)

// contextKey is a type for context keys to avoid collisions
type contextKey string

const accountSidKey contextKey = "account_sid"

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
		ctx = context.WithValue(ctx, accountSidKey, accountSid)
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

// ListResources implements core.DataProvider to expose data to admin UI
func (p *TwilioPlugin) ListResources(ctx context.Context, slug string, opts core.ListOptions) ([]map[string]interface{}, error) {
	switch slug {
	case "accounts":
		accounts, err := p.store.ListAllAccounts(opts.Limit, opts.Offset)
		if err != nil {
			return nil, err
		}
		return convertAccountsToMaps(accounts), nil
	case "messages":
		messages, err := p.store.ListAllMessages(opts.Limit, opts.Offset)
		if err != nil {
			return nil, err
		}
		return convertMessagesToMaps(messages), nil
	case "calls":
		calls, err := p.store.ListAllCalls(opts.Limit, opts.Offset)
		if err != nil {
			return nil, err
		}
		return convertCallsToMaps(calls), nil
	case "phone_numbers":
		phoneNumbers, err := p.store.ListAllPhoneNumbers(opts.Limit, opts.Offset)
		if err != nil {
			return nil, err
		}
		return convertPhoneNumbersToMaps(phoneNumbers), nil
	default:
		return nil, fmt.Errorf("unknown resource: %s", slug)
	}
}

// GetResource implements core.DataProvider to fetch individual resources
func (p *TwilioPlugin) GetResource(ctx context.Context, slug string, id string) (map[string]interface{}, error) {
	switch slug {
	case "accounts":
		account, err := p.store.GetOrCreateAccount(id)
		if err != nil {
			return nil, err
		}
		return convertAccountToMap(*account), nil
	case "messages":
		message, err := p.store.GetMessage(id)
		if err != nil {
			return nil, err
		}
		return convertMessageToMap(*message), nil
	case "calls":
		call, err := p.store.GetCall(id)
		if err != nil {
			return nil, err
		}
		return convertCallToMap(*call), nil
	case "phone_numbers":
		phoneNumber, err := p.store.GetPhoneNumber(id)
		if err != nil {
			return nil, err
		}
		return convertPhoneNumberToMap(*phoneNumber), nil
	default:
		return nil, fmt.Errorf("unknown resource: %s", slug)
	}
}

// Helper conversion functions

func convertAccountsToMaps(accounts []Account) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(accounts))
	for _, acct := range accounts {
		result = append(result, convertAccountToMap(acct))
	}
	return result
}

func convertAccountToMap(acct Account) map[string]interface{} {
	return map[string]interface{}{
		"account_sid":   acct.AccountSid,
		"auth_token":    acct.AuthToken,
		"friendly_name": acct.FriendlyName,
		"status":        acct.Status,
		"created_at":    acct.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func convertMessagesToMaps(messages []Message) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(messages))
	for _, msg := range messages {
		result = append(result, convertMessageToMap(msg))
	}
	return result
}

func convertMessageToMap(msg Message) map[string]interface{} {
	m := map[string]interface{}{
		"sid":          msg.Sid,
		"account_sid":  msg.AccountSid,
		"from_number":  msg.FromNumber,
		"to_number":    msg.ToNumber,
		"body":         msg.Body,
		"status":       msg.Status,
		"direction":    msg.Direction,
		"date_created": msg.DateCreated.Format("2006-01-02T15:04:05Z"),
	}
	if msg.DateSent != nil {
		m["date_sent"] = msg.DateSent.Format("2006-01-02T15:04:05Z")
	}
	return m
}

func convertCallsToMaps(calls []Call) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(calls))
	for _, call := range calls {
		result = append(result, convertCallToMap(call))
	}
	return result
}

func convertCallToMap(call Call) map[string]interface{} {
	m := map[string]interface{}{
		"sid":          call.Sid,
		"account_sid":  call.AccountSid,
		"from_number":  call.FromNumber,
		"to_number":    call.ToNumber,
		"status":       call.Status,
		"date_created": call.DateCreated.Format("2006-01-02T15:04:05Z"),
	}
	if call.Duration != nil {
		m["duration"] = fmt.Sprintf("%d", *call.Duration)
	}
	return m
}

func convertPhoneNumbersToMaps(phoneNumbers []PhoneNumber) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(phoneNumbers))
	for _, pn := range phoneNumbers {
		result = append(result, convertPhoneNumberToMap(pn))
	}
	return result
}

func convertPhoneNumberToMap(pn PhoneNumber) map[string]interface{} {
	return map[string]interface{}{
		"sid":           pn.Sid,
		"account_sid":   pn.AccountSid,
		"phone_number":  pn.PhoneNumber,
		"friendly_name": pn.FriendlyName,
		"voice_url":     pn.VoiceURL,
		"sms_url":       pn.SmsURL,
		"created_at":    pn.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
