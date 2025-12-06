// ABOUTME: Gmail API handlers for Google plugin.
// ABOUTME: Implements Gmail v1 API endpoints.

package google

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/2389/ish/internal/auth"
	"github.com/2389/ish/internal/autoreply"
	"github.com/go-chi/chi/v5"
)

// googleStoreAdapter adapts GoogleStore to autoreply.GmailMessageSender interface
type googleStoreAdapter struct {
	store *GoogleStore
}

func (a *googleStoreAdapter) SendGmailMessage(userID, from, to, subject, body string) (any, error) {
	return a.store.SendGmailMessage(userID, from, to, subject, body)
}

func (p *GooglePlugin) registerGmailRoutes(r chi.Router) {
	r.Route("/gmail/v1/users/{userId}", func(r chi.Router) {
		r.Get("/profile", p.getProfile)
		r.Get("/messages", p.listMessages)
		r.Post("/messages/send", p.sendMessage)
		r.Get("/messages/{messageId}", p.getMessage)
		r.Get("/messages/{messageId}/attachments/{attachmentId}", p.getAttachment)
		r.Get("/history", p.listHistory)
	})
}

func (p *GooglePlugin) listMessages(w http.ResponseWriter, r *http.Request) {
	if p.store == nil {
		writeError(w, 500, "Plugin not initialized", "INTERNAL")
		return
	}

	userID := chi.URLParam(r, "userId")
	if userID == "me" {
		userID = auth.UserFromContext(r.Context())
	}

	maxResults := 100
	if mr := r.URL.Query().Get("maxResults"); mr != "" {
		if v, err := strconv.Atoi(mr); err == nil && v > 0 {
			maxResults = v
		}
	}
	pageToken := r.URL.Query().Get("pageToken")
	query := r.URL.Query().Get("q")

	messages, nextToken, err := p.store.ListGmailMessages(userID, maxResults, pageToken, query)
	if err != nil {
		writeError(w, 500, "Internal error", "INTERNAL")
		return
	}

	// Convert to response format
	msgList := make([]map[string]any, len(messages))
	for i, m := range messages {
		msgList[i] = map[string]any{
			"id":       m.ID,
			"threadId": m.ThreadID,
		}
	}

	resp := map[string]any{
		"messages":           msgList,
		"resultSizeEstimate": len(messages),
	}
	if nextToken != "" {
		resp["nextPageToken"] = nextToken
	}

	writeJSON(w, resp)
}

func (p *GooglePlugin) getMessage(w http.ResponseWriter, r *http.Request) {
	if p.store == nil {
		writeError(w, 500, "Plugin not initialized", "INTERNAL")
		return
	}

	userID := chi.URLParam(r, "userId")
	if userID == "me" {
		userID = auth.UserFromContext(r.Context())
	}
	messageID := chi.URLParam(r, "messageId")

	msg, err := p.store.GetGmailMessage(userID, messageID)
	if err != nil {
		writeError(w, 404, "Message not found", "NOT_FOUND")
		return
	}

	// Parse payload JSON
	var payload map[string]any
	if err := json.Unmarshal([]byte(msg.Payload), &payload); err != nil {
		writeError(w, 500, "Failed to parse message payload", "INTERNAL")
		return
	}

	resp := map[string]any{
		"id":           msg.ID,
		"threadId":     msg.ThreadID,
		"labelIds":     msg.LabelIDs,
		"snippet":      msg.Snippet,
		"internalDate": strconv.FormatInt(msg.InternalDate, 10),
		"payload":      payload,
	}

	writeJSON(w, resp)
}

func (p *GooglePlugin) getProfile(w http.ResponseWriter, r *http.Request) {
	if p.store == nil {
		writeError(w, 500, "Plugin not initialized", "INTERNAL")
		return
	}

	userID := chi.URLParam(r, "userId")
	if userID == "me" {
		userID = auth.UserFromContext(r.Context())
	}

	profile, err := p.store.GetGmailProfile(userID)
	if err != nil {
		writeError(w, 500, "Internal error", "INTERNAL")
		return
	}

	resp := map[string]any{
		"emailAddress":  profile.EmailAddress,
		"messagesTotal": profile.MessagesTotal,
		"threadsTotal":  profile.ThreadsTotal,
		"historyId":     strconv.FormatInt(profile.HistoryID, 10),
	}

	writeJSON(w, resp)
}

func (p *GooglePlugin) getAttachment(w http.ResponseWriter, r *http.Request) {
	if p.store == nil {
		writeError(w, 500, "Plugin not initialized", "INTERNAL")
		return
	}

	userID := chi.URLParam(r, "userId")
	if userID == "me" {
		userID = auth.UserFromContext(r.Context())
	}
	messageID := chi.URLParam(r, "messageId")
	attachmentID := chi.URLParam(r, "attachmentId")

	attachment, err := p.store.GetGmailAttachment(userID, messageID, attachmentID)
	if err != nil {
		writeError(w, 404, "Attachment not found", "NOT_FOUND")
		return
	}

	resp := map[string]any{
		"attachmentId": attachment.ID,
		"size":         attachment.Size,
		"data":         attachment.Data,
	}

	writeJSON(w, resp)
}

func (p *GooglePlugin) listHistory(w http.ResponseWriter, r *http.Request) {
	if p.store == nil {
		writeError(w, 500, "Plugin not initialized", "INTERNAL")
		return
	}

	userID := chi.URLParam(r, "userId")
	if userID == "me" {
		userID = auth.UserFromContext(r.Context())
	}

	startHistoryID := int64(1)
	if sid := r.URL.Query().Get("startHistoryId"); sid != "" {
		if v, err := strconv.ParseInt(sid, 10, 64); err == nil {
			startHistoryID = v
		}
	}

	maxResults := 100
	if mr := r.URL.Query().Get("maxResults"); mr != "" {
		if v, err := strconv.Atoi(mr); err == nil && v > 0 {
			maxResults = v
		}
	}
	pageToken := r.URL.Query().Get("pageToken")

	entries, historyID, nextToken, err := p.store.ListGmailHistory(userID, startHistoryID, maxResults, pageToken)
	if err != nil {
		writeError(w, 500, "Internal error", "INTERNAL")
		return
	}

	// Convert entries to response format
	historyList := make([]map[string]any, len(entries))
	for i, entry := range entries {
		messagesAdded := make([]map[string]any, len(entry.MessagesAdded))
		for j, msg := range entry.MessagesAdded {
			messagesAdded[j] = map[string]any{
				"message": map[string]any{
					"id":       msg.ID,
					"threadId": msg.ThreadID,
					"labelIds": msg.LabelIDs,
				},
			}
		}
		historyList[i] = map[string]any{
			"id":            strconv.FormatInt(entry.ID, 10),
			"messagesAdded": messagesAdded,
		}
	}

	resp := map[string]any{
		"history":   historyList,
		"historyId": strconv.FormatInt(historyID, 10),
	}
	if nextToken != "" {
		resp["nextPageToken"] = nextToken
	}

	writeJSON(w, resp)
}

func (p *GooglePlugin) sendMessage(w http.ResponseWriter, r *http.Request) {
	if p.store == nil {
		writeError(w, 500, "Plugin not initialized", "INTERNAL")
		return
	}

	userID := chi.URLParam(r, "userId")
	if userID == "me" {
		userID = auth.UserFromContext(r.Context())
	}

	var req struct {
		Raw string `json:"raw"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "Invalid request body", "INVALID_REQUEST")
		return
	}

	// Decode base64 email
	decoded, err := base64.URLEncoding.DecodeString(req.Raw)
	if err != nil {
		// Try URL-safe variant
		decoded, err = base64.RawURLEncoding.DecodeString(req.Raw)
		if err != nil {
			writeError(w, 400, "Invalid base64 encoding", "INVALID_REQUEST")
			return
		}
	}

	// Parse email headers
	headers, body := parseEmail(string(decoded))

	to := headers["To"]
	subject := headers["Subject"]
	from := headers["From"]
	if from == "" {
		from = userID + "@example.com"
	}

	// Create message with SENT label
	msg, err := p.store.SendGmailMessage(userID, from, to, subject, body)
	if err != nil {
		writeError(w, 500, "Failed to send message", "INTERNAL")
		return
	}

	// Trigger auto-reply (runs in background)
	autoReply := autoreply.New(&googleStoreAdapter{store: p.store})
	autoReply.GenerateReply(userID, from, to, subject, body, msg.ThreadID)

	resp := map[string]any{
		"id":       msg.ID,
		"threadId": msg.ThreadID,
		"labelIds": msg.LabelIDs,
	}

	writeJSON(w, resp)
}

// parseEmail parses an RFC 2822 email message into headers and body
func parseEmail(email string) (map[string]string, string) {
	headers := make(map[string]string)

	// Split headers and body by empty line
	parts := strings.SplitN(email, "\r\n\r\n", 2)
	if len(parts) == 1 {
		parts = strings.SplitN(email, "\n\n", 2)
	}

	headerLines := strings.Split(parts[0], "\n")
	var body string
	if len(parts) > 1 {
		body = parts[1]
	}

	// Parse headers
	for _, line := range headerLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Handle multi-line headers
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue // Skip continuation lines for simplicity
		}

		colonIdx := strings.Index(line, ":")
		if colonIdx > 0 {
			name := strings.TrimSpace(line[:colonIdx])
			value := strings.TrimSpace(line[colonIdx+1:])
			headers[name] = value
		}
	}

	return headers, body
}

// writeJSON writes a JSON response to the ResponseWriter.
// Note: If JSON encoding fails, an error is logged but the function continues.
// Since headers are already sent at this point, we cannot return a different error status.
// The client will receive a partial/malformed JSON response which they should handle.
func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
		// Note: Cannot change status code or send error response - headers already committed
	}
}

// writeError writes a Google API-style error response.
// Note: If JSON encoding fails after WriteHeader is called, an error is logged but no
// recovery is possible since headers and status code are already sent to the client.
// The client will receive a partial/malformed error response.
func writeError(w http.ResponseWriter, code int, message, status string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
			"status":  status,
		},
	}); err != nil {
		log.Printf("Failed to encode error response: %v", err)
		// Note: Cannot recover - status code and headers already sent
	}
}
