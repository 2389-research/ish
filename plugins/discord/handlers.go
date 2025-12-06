// ABOUTME: HTTP handlers for Discord webhook API endpoints
// ABOUTME: Implements Discord API v10 webhook specification

package discord

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// executeWebhook handles POST /api/webhooks/{webhook.id}/{webhook.token}
func (p *DiscordPlugin) executeWebhook(w http.ResponseWriter, r *http.Request) {
	if p.store == nil {
		writeError(w, 500, "Plugin not initialized")
		return
	}

	webhookID := chi.URLParam(r, "webhookID")
	webhookToken := chi.URLParam(r, "webhookToken")
	wait := r.URL.Query().Get("wait") == "true"
	threadID := r.URL.Query().Get("thread_id")

	// Get or create webhook (auto-accept pattern)
	webhook, err := p.store.GetOrCreateWebhook(webhookID, webhookToken)
	if err != nil {
		writeError(w, 500, "Failed to get webhook")
		return
	}

	// Parse request body
	var req struct {
		Content     string                   `json:"content"`
		Username    string                   `json:"username"`
		AvatarURL   string                   `json:"avatar_url"`
		Embeds      []map[string]interface{} `json:"embeds"`
		Components  []map[string]interface{} `json:"components"`
		Attachments []map[string]interface{} `json:"attachments"`
		Flags       int                      `json:"flags"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "Invalid request body")
		return
	}

	// Create message
	msg := &WebhookMessage{
		WebhookID: webhook.ID,
		Content:   req.Content,
		Username:  req.Username,
		AvatarURL: req.AvatarURL,
		ThreadID:  threadID,
		Flags:     req.Flags,
	}

	// Marshal JSON fields
	if len(req.Embeds) > 0 {
		embedsJSON, _ := json.Marshal(req.Embeds)
		msg.Embeds = string(embedsJSON)
	}
	if len(req.Components) > 0 {
		componentsJSON, _ := json.Marshal(req.Components)
		msg.Components = string(componentsJSON)
	}
	if len(req.Attachments) > 0 {
		attachmentsJSON, _ := json.Marshal(req.Attachments)
		msg.Attachments = string(attachmentsJSON)
	}

	if err := p.store.CreateMessage(msg); err != nil {
		writeError(w, 500, "Failed to create message")
		return
	}

	// Return message if wait=true, otherwise 204
	if wait {
		writeJSON(w, msg)
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode response", 500)
	}
}

func writeError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": message,
		"code":    code,
	})
}

// getWebhook handles GET /api/webhooks/{webhook.id}/{webhook.token}
func (p *DiscordPlugin) getWebhook(w http.ResponseWriter, r *http.Request) {
	if p.store == nil {
		writeError(w, 500, "Plugin not initialized")
		return
	}

	webhookID := chi.URLParam(r, "webhookID")
	webhookToken := chi.URLParam(r, "webhookToken")

	webhook, err := p.store.GetWebhook(webhookID, webhookToken)
	if err != nil {
		writeError(w, 404, "Webhook not found")
		return
	}

	writeJSON(w, webhook)
}

// modifyWebhook handles PATCH /api/webhooks/{webhook.id}/{webhook.token}
func (p *DiscordPlugin) modifyWebhook(w http.ResponseWriter, r *http.Request) {
	if p.store == nil {
		writeError(w, 500, "Plugin not initialized")
		return
	}

	webhookID := chi.URLParam(r, "webhookID")
	webhookToken := chi.URLParam(r, "webhookToken")

	webhook, err := p.store.GetWebhook(webhookID, webhookToken)
	if err != nil {
		writeError(w, 404, "Webhook not found")
		return
	}

	var req struct {
		Name   string `json:"name"`
		Avatar string `json:"avatar"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "Invalid request body")
		return
	}

	if req.Name != "" {
		webhook.Name = req.Name
	}
	if req.Avatar != "" {
		webhook.Avatar = req.Avatar
	}

	if err := p.store.UpdateWebhook(webhook); err != nil {
		writeError(w, 500, "Failed to update webhook")
		return
	}

	writeJSON(w, webhook)
}

// deleteWebhook handles DELETE /api/webhooks/{webhook.id}/{webhook.token}
func (p *DiscordPlugin) deleteWebhook(w http.ResponseWriter, r *http.Request) {
	if p.store == nil {
		writeError(w, 500, "Plugin not initialized")
		return
	}

	webhookID := chi.URLParam(r, "webhookID")
	webhookToken := chi.URLParam(r, "webhookToken")

	if err := p.store.DeleteWebhook(webhookID, webhookToken); err != nil {
		writeError(w, 500, "Failed to delete webhook")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// getWebhookMessage handles GET /api/webhooks/{webhook.id}/{webhook.token}/messages/{message.id}
func (p *DiscordPlugin) getWebhookMessage(w http.ResponseWriter, r *http.Request) {
	if p.store == nil {
		writeError(w, 500, "Plugin not initialized")
		return
	}

	webhookID := chi.URLParam(r, "webhookID")
	messageID := chi.URLParam(r, "messageID")

	msg, err := p.store.GetMessage(webhookID, messageID)
	if err != nil {
		writeError(w, 404, "Message not found")
		return
	}

	writeJSON(w, msg)
}

// editWebhookMessage handles PATCH /api/webhooks/{webhook.id}/{webhook.token}/messages/{message.id}
func (p *DiscordPlugin) editWebhookMessage(w http.ResponseWriter, r *http.Request) {
	if p.store == nil {
		writeError(w, 500, "Plugin not initialized")
		return
	}

	webhookID := chi.URLParam(r, "webhookID")
	messageID := chi.URLParam(r, "messageID")

	msg, err := p.store.GetMessage(webhookID, messageID)
	if err != nil {
		writeError(w, 404, "Message not found")
		return
	}

	var req struct {
		Content     *string                  `json:"content"`
		Embeds      []map[string]interface{} `json:"embeds"`
		Components  []map[string]interface{} `json:"components"`
		Attachments []map[string]interface{} `json:"attachments"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "Invalid request body")
		return
	}

	if req.Content != nil {
		msg.Content = *req.Content
	}
	if len(req.Embeds) > 0 {
		embedsJSON, _ := json.Marshal(req.Embeds)
		msg.Embeds = string(embedsJSON)
	}
	if len(req.Components) > 0 {
		componentsJSON, _ := json.Marshal(req.Components)
		msg.Components = string(componentsJSON)
	}
	if len(req.Attachments) > 0 {
		attachmentsJSON, _ := json.Marshal(req.Attachments)
		msg.Attachments = string(attachmentsJSON)
	}

	if err := p.store.UpdateMessage(msg); err != nil {
		writeError(w, 500, "Failed to update message")
		return
	}

	writeJSON(w, msg)
}

// deleteWebhookMessage handles DELETE /api/webhooks/{webhook.id}/{webhook.token}/messages/{message.id}
func (p *DiscordPlugin) deleteWebhookMessage(w http.ResponseWriter, r *http.Request) {
	if p.store == nil {
		writeError(w, 500, "Plugin not initialized")
		return
	}

	webhookID := chi.URLParam(r, "webhookID")
	messageID := chi.URLParam(r, "messageID")

	if err := p.store.DeleteMessage(webhookID, messageID); err != nil {
		writeError(w, 500, "Failed to delete message")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
