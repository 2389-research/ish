// ABOUTME: HTTP handlers for SendGrid Mail API v3 endpoints
// ABOUTME: Implements mail sending, message retrieval, and suppression management

package sendgrid

import (
	"encoding/json"
	"log"
	"net/http"
	"net/mail"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// SendMailRequest represents the v3/mail/send request body
type SendMailRequest struct {
	Personalizations []Personalization `json:"personalizations"`
	From             EmailAddress      `json:"from"`
	Subject          string            `json:"subject"`
	Content          []Content         `json:"content"`
}

type Personalization struct {
	To      []EmailAddress `json:"to"`
	Cc      []EmailAddress `json:"cc,omitempty"`
	Bcc     []EmailAddress `json:"bcc,omitempty"`
	Subject string         `json:"subject,omitempty"`
}

type EmailAddress struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type Content struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// sendMail handles POST /v3/mail/send
func (p *SendGridPlugin) sendMail(w http.ResponseWriter, r *http.Request) {
	account, ok := getAccountFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required", "")
		return
	}

	var req SendMailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", "")
		return
	}

	// Validate request
	if len(req.Personalizations) == 0 {
		writeError(w, http.StatusBadRequest, "personalizations is required", "personalizations")
		return
	}

	if len(req.Personalizations[0].To) == 0 {
		writeError(w, http.StatusBadRequest, "at least one 'to' email is required", "personalizations.to")
		return
	}

	if req.From.Email == "" {
		writeError(w, http.StatusBadRequest, "from email is required", "from.email")
		return
	}

	// Validate from email format
	if _, err := mail.ParseAddress(req.From.Email); err != nil {
		writeError(w, http.StatusBadRequest, "invalid from email address", "from.email")
		return
	}

	// Validate to email format
	if _, err := mail.ParseAddress(req.Personalizations[0].To[0].Email); err != nil {
		writeError(w, http.StatusBadRequest, "invalid to email address", "personalizations.to.email")
		return
	}

	// Extract content
	var textContent, htmlContent string
	for _, content := range req.Content {
		if content.Type == "text/plain" {
			textContent = content.Value
		} else if content.Type == "text/html" {
			htmlContent = content.Value
		}
	}

	// Use personalization subject if provided, otherwise use top-level subject
	subject := req.Subject
	if req.Personalizations[0].Subject != "" {
		subject = req.Personalizations[0].Subject
	}

	// For simplicity, send to the first recipient
	// Real SendGrid sends to all recipients
	toEmail := req.Personalizations[0].To[0].Email
	toName := req.Personalizations[0].To[0].Name

	// Create message record
	message, err := p.store.CreateMessage(
		account.ID,
		req.From.Email,
		req.From.Name,
		toEmail,
		toName,
		subject,
		textContent,
		htmlContent,
	)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to send message", "")
		return
	}

	// SendGrid returns 202 Accepted
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Message-Id", message.ID)
	w.WriteHeader(http.StatusAccepted)
}

// getMailSettings handles GET /v3/mail/settings
func (p *SendGridPlugin) getMailSettings(w http.ResponseWriter, r *http.Request) {
	account, ok := getAccountFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required", "")
		return
	}

	// Return basic settings
	settings := map[string]interface{}{
		"account_id": account.ID,
		"settings": map[string]interface{}{
			"footer": map[string]interface{}{
				"enabled": false,
			},
			"sandbox_mode": map[string]interface{}{
				"enabled": false,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(settings); err != nil {
		log.Printf("SendGrid: Failed to encode settings response: %v", err)
	}
}

// listMessages handles GET /v3/messages
func (p *SendGridPlugin) listMessages(w http.ResponseWriter, r *http.Request) {
	account, ok := getAccountFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required", "")
		return
	}

	// Get pagination parameters with DoS protection
	limit := 50
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 1000 {
			limit = l
		}
	}

	offset := 0
	if offsetParam := r.URL.Query().Get("offset"); offsetParam != "" {
		if o, err := strconv.Atoi(offsetParam); err == nil && o >= 0 {
			offset = o
		}
	}

	messages, err := p.store.ListMessages(account.ID, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list messages", "")
		return
	}

	// Convert to response format
	response := make([]map[string]interface{}, 0, len(messages))
	for _, msg := range messages {
		response = append(response, map[string]interface{}{
			"msg_id":     msg.ID,
			"from_email": msg.FromEmail,
			"to_email":   msg.ToEmail,
			"subject":    msg.Subject,
			"status":     msg.Status,
			"sent_at":    msg.SentAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"messages": response,
	}); err != nil {
		log.Printf("SendGrid: Failed to encode messages response: %v", err)
	}
}

// getMessage handles GET /v3/messages/{message_id}
func (p *SendGridPlugin) getMessage(w http.ResponseWriter, r *http.Request) {
	account, ok := getAccountFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required", "")
		return
	}

	messageID := chi.URLParam(r, "message_id")

	message, err := p.store.GetMessage(messageID)
	if err != nil {
		writeError(w, http.StatusNotFound, "message not found", "message_id")
		return
	}

	// Verify ownership
	if message.AccountID != account.ID {
		writeError(w, http.StatusNotFound, "message not found", "message_id")
		return
	}

	response := map[string]interface{}{
		"msg_id":       message.ID,
		"from_email":   message.FromEmail,
		"from_name":    message.FromName,
		"to_email":     message.ToEmail,
		"to_name":      message.ToName,
		"subject":      message.Subject,
		"text_content": message.TextContent,
		"html_content": message.HTMLContent,
		"status":       message.Status,
		"sent_at":      message.SentAt.Format("2006-01-02T15:04:05Z"),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("SendGrid: Failed to encode message response: %v", err)
	}
}

// listBounces handles GET /v3/suppression/bounces
func (p *SendGridPlugin) listBounces(w http.ResponseWriter, r *http.Request) {
	p.listSuppressionsByType(w, r, "bounce")
}

// listBlocks handles GET /v3/suppression/blocks
func (p *SendGridPlugin) listBlocks(w http.ResponseWriter, r *http.Request) {
	p.listSuppressionsByType(w, r, "block")
}

// listSpamReports handles GET /v3/suppression/spam_reports
func (p *SendGridPlugin) listSpamReports(w http.ResponseWriter, r *http.Request) {
	p.listSuppressionsByType(w, r, "spam_report")
}

// listSuppressionsByType is a helper to list suppressions by type
func (p *SendGridPlugin) listSuppressionsByType(w http.ResponseWriter, r *http.Request, suppressionType string) {
	account, ok := getAccountFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required", "")
		return
	}

	suppressions, err := p.store.ListSuppressions(account.ID, suppressionType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list suppressions", "")
		return
	}

	// Convert to response format
	response := make([]map[string]interface{}, 0, len(suppressions))
	for _, supp := range suppressions {
		response = append(response, map[string]interface{}{
			"email":      supp.Email,
			"reason":     supp.Reason,
			"created":    supp.CreatedAt.Unix(),
			"created_at": supp.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("SendGrid: Failed to encode suppressions response: %v", err)
	}
}

// deleteBounce handles DELETE /v3/suppression/bounces/{email}
func (p *SendGridPlugin) deleteBounce(w http.ResponseWriter, r *http.Request) {
	p.deleteSuppressionByType(w, r, "bounce")
}

// deleteBlock handles DELETE /v3/suppression/blocks/{email}
func (p *SendGridPlugin) deleteBlock(w http.ResponseWriter, r *http.Request) {
	p.deleteSuppressionByType(w, r, "block")
}

// deleteSpamReport handles DELETE /v3/suppression/spam_reports/{email}
func (p *SendGridPlugin) deleteSpamReport(w http.ResponseWriter, r *http.Request) {
	p.deleteSuppressionByType(w, r, "spam_report")
}

// deleteSuppressionByType is a helper to delete suppressions by type
func (p *SendGridPlugin) deleteSuppressionByType(w http.ResponseWriter, r *http.Request, suppressionType string) {
	account, ok := getAccountFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authentication required", "")
		return
	}

	email := chi.URLParam(r, "email")

	err := p.store.DeleteSuppression(account.ID, email, suppressionType)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete suppression", "")
		return
	}

	// Return 204 No Content on success
	w.WriteHeader(http.StatusNoContent)
}
