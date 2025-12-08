// ABOUTME: SendGrid Mail API v3 plugin for ISH
// ABOUTME: Simulates SendGrid's email sending and management APIs

package sendgrid

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/2389/ish/plugins/core"
	"github.com/go-chi/chi/v5"
)

type contextKey string

const accountKey contextKey = "sendgrid_account"

func init() {
	core.Register(&SendGridPlugin{})
}

// setAccountInContext stores the account in the request context
func setAccountInContext(ctx context.Context, account *Account) context.Context {
	return context.WithValue(ctx, accountKey, account)
}

// getAccountFromContext retrieves the account from the request context
func getAccountFromContext(ctx context.Context) (*Account, bool) {
	account, ok := ctx.Value(accountKey).(*Account)
	return account, ok
}

type SendGridPlugin struct {
	store *SendGridStore
}

func (p *SendGridPlugin) Name() string {
	return "sendgrid"
}

func (p *SendGridPlugin) Health() core.HealthStatus {
	return core.HealthStatus{
		Status:  "healthy",
		Message: "SendGrid plugin operational",
	}
}

func (p *SendGridPlugin) RegisterRoutes(r chi.Router) {
	// Mail Send API v3
	r.Post("/v3/mail/send", p.requireAuth(p.sendMail))

	// Mail Settings API
	r.Get("/v3/mail/settings", p.requireAuth(p.getMailSettings))

	// Messages API (for retrieving sent messages)
	r.Get("/v3/messages", p.requireAuth(p.listMessages))
	r.Get("/v3/messages/{message_id}", p.requireAuth(p.getMessage))

	// Suppression Management (bounces, blocks, spam reports)
	r.Get("/v3/suppression/bounces", p.requireAuth(p.listBounces))
	r.Delete("/v3/suppression/bounces/{email}", p.requireAuth(p.deleteBounce))

	r.Get("/v3/suppression/blocks", p.requireAuth(p.listBlocks))
	r.Delete("/v3/suppression/blocks/{email}", p.requireAuth(p.deleteBlock))

	r.Get("/v3/suppression/spam_reports", p.requireAuth(p.listSpamReports))
	r.Delete("/v3/suppression/spam_reports/{email}", p.requireAuth(p.deleteSpamReport))
}

func (p *SendGridPlugin) RegisterAuth(r chi.Router) {
	// SendGrid uses API Key authentication via Bearer token
}

// extractAPIKey extracts the API key from the Authorization header
func extractAPIKey(authHeader string) (string, bool) {
	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return "", false
	}
	return authHeader[len(prefix):], true
}

// requireAuth middleware validates SendGrid API Key
func (p *SendGridPlugin) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeError(w, http.StatusUnauthorized, "authorization required", "")
			return
		}

		apiKey, ok := extractAPIKey(authHeader)
		if !ok {
			writeError(w, http.StatusUnauthorized, "invalid authorization header", "")
			return
		}

		// Validate API key and get account
		account, err := p.store.ValidateAPIKey(apiKey)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid API key", "")
			return
		}

		// Store account in context for handlers
		ctx := r.Context()
		ctx = setAccountInContext(ctx, account)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// writeError writes a SendGrid-style JSON error response
func writeError(w http.ResponseWriter, status int, message, field string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	errors := []map[string]interface{}{
		{
			"message": message,
			"field":   field,
		},
	}

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"errors": errors,
	}); err != nil {
		log.Printf("SendGrid: Failed to encode error response: %v", err)
	}
}

func (p *SendGridPlugin) ValidateToken(token string) bool {
	_, err := p.store.ValidateAPIKey(token)
	return err == nil
}

func (p *SendGridPlugin) SetDB(db *sql.DB) error {
	store, err := NewSendGridStore(db)
	if err != nil {
		return err
	}
	p.store = store
	return nil
}

func (p *SendGridPlugin) Schema() core.PluginSchema {
	return core.PluginSchema{
		Resources: []core.ResourceSchema{
			{
				Name:        "Messages",
				Slug:        "messages",
				ListColumns: []string{"id", "from_email", "to_email", "subject", "status", "sent_at"},
				Fields: []core.FieldSchema{
					{Name: "id", Type: "string", Display: "ID", Required: true, Editable: false},
					{Name: "from_email", Type: "string", Display: "From", Required: true, Editable: false},
					{Name: "to_email", Type: "string", Display: "To", Required: true, Editable: false},
					{Name: "subject", Type: "string", Display: "Subject", Required: false, Editable: false},
					{Name: "status", Type: "string", Display: "Status", Required: true, Editable: false},
					{Name: "sent_at", Type: "datetime", Display: "Sent At", Required: false, Editable: false},
				},
				Actions: []core.ActionSchema{},
			},
			{
				Name:        "Suppressions",
				Slug:        "suppressions",
				ListColumns: []string{"id", "email", "type", "reason", "created_at"},
				Fields: []core.FieldSchema{
					{Name: "id", Type: "string", Display: "ID", Required: true, Editable: false},
					{Name: "email", Type: "string", Display: "Email", Required: true, Editable: false},
					{Name: "type", Type: "string", Display: "Type", Required: true, Editable: false},
					{Name: "reason", Type: "text", Display: "Reason", Required: false, Editable: false},
					{Name: "created_at", Type: "datetime", Display: "Created", Required: false, Editable: false},
				},
				Actions: []core.ActionSchema{},
			},
		},
	}
}

// ListResources implements core.DataProvider to expose data to admin UI
func (p *SendGridPlugin) ListResources(ctx context.Context, slug string, opts core.ListOptions) ([]map[string]interface{}, error) {
	switch slug {
	case "messages":
		messages, err := p.store.ListAllMessages(opts.Limit, opts.Offset)
		if err != nil {
			return nil, err
		}
		return convertMessagesToMaps(messages), nil
	case "suppressions":
		suppressions, err := p.store.ListAllSuppressions(opts.Limit, opts.Offset)
		if err != nil {
			return nil, err
		}
		return convertSuppressionsToMaps(suppressions), nil
	default:
		return nil, fmt.Errorf("unknown resource: %s", slug)
	}
}

// GetResource implements core.DataProvider to fetch individual resources
func (p *SendGridPlugin) GetResource(ctx context.Context, slug string, id string) (map[string]interface{}, error) {
	switch slug {
	case "messages":
		message, err := p.store.GetMessage(id)
		if err != nil {
			return nil, err
		}
		return convertMessageToMap(message), nil
	case "suppressions":
		// Suppressions don't have individual GET by ID in current store
		// Would need to add GetSuppression method if needed
		return nil, fmt.Errorf("individual suppression lookup not supported - use ListResources to view all")
	default:
		return nil, fmt.Errorf("unknown resource: %s", slug)
	}
}

// convertMessagesToMaps converts message structs to maps for admin UI
func convertMessagesToMaps(messages []*Message) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(messages))
	for _, msg := range messages {
		result = append(result, convertMessageToMap(msg))
	}
	return result
}

// convertMessageToMap converts a single message to map
func convertMessageToMap(msg *Message) map[string]interface{} {
	return map[string]interface{}{
		"id":         msg.ID,
		"from_email": msg.FromEmail,
		"to_email":   msg.ToEmail,
		"subject":    msg.Subject,
		"status":     msg.Status,
		"sent_at":    msg.SentAt.Format("2006-01-02T15:04:05Z"),
	}
}

// convertSuppressionsToMaps converts suppression structs to maps for admin UI
func convertSuppressionsToMaps(suppressions []*Suppression) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(suppressions))
	for _, supp := range suppressions {
		result = append(result, map[string]interface{}{
			"id":         fmt.Sprintf("%d", supp.ID),
			"email":      supp.Email,
			"type":       supp.Type,
			"reason":     supp.Reason,
			"created_at": supp.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}
	return result
}

// Seed is implemented in seed.go
