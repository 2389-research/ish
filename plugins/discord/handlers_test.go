// ABOUTME: HTTP handler tests for Discord webhook endpoints
// ABOUTME: Tests executeWebhook, getWebhook, modifyWebhook, message handlers

package discord

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestPlugin(t *testing.T) *DiscordPlugin {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	plugin := &DiscordPlugin{}
	if err := plugin.SetDB(db); err != nil {
		t.Fatalf("Failed to initialize plugin: %v", err)
	}
	return plugin
}

func newRequestWithParams(method, url string, body []byte, params map[string]string) *http.Request {
	req := httptest.NewRequest(method, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	return req.WithContext(ctx)
}

func TestExecuteWebhook(t *testing.T) {
	plugin := setupTestPlugin(t)

	body := map[string]interface{}{
		"content":  "Test message",
		"username": "TestBot",
	}
	bodyJSON, _ := json.Marshal(body)

	req := newRequestWithParams("POST", "/api/webhooks/123/token123?wait=true", bodyJSON, map[string]string{
		"webhookID":    "123",
		"webhookToken": "token123",
	})

	w := httptest.NewRecorder()
	plugin.executeWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var response WebhookMessage
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if response.Content != "Test message" {
		t.Fatalf("Expected content 'Test message', got '%s'", response.Content)
	}
	if response.Username != "TestBot" {
		t.Fatalf("Expected username 'TestBot', got '%s'", response.Username)
	}
	if response.ID == "" {
		t.Fatal("Message ID should be generated")
	}
}

func TestExecuteWebhookNoWait(t *testing.T) {
	plugin := setupTestPlugin(t)

	body := map[string]interface{}{
		"content": "Test message",
	}
	bodyJSON, _ := json.Marshal(body)

	req := newRequestWithParams("POST", "/api/webhooks/123/token123", bodyJSON, map[string]string{
		"webhookID":    "123",
		"webhookToken": "token123",
	})

	w := httptest.NewRecorder()
	plugin.executeWebhook(w, req)

	// Without wait=true, should return 204 No Content
	if w.Code != http.StatusNoContent {
		t.Fatalf("Expected 204, got %d", w.Code)
	}
}

func TestExecuteWebhookWithEmbeds(t *testing.T) {
	plugin := setupTestPlugin(t)

	body := map[string]interface{}{
		"content":  "Message with embeds",
		"username": "EmbedBot",
		"embeds": []map[string]interface{}{
			{
				"title":       "Test Embed",
				"description": "This is a test embed",
				"color":       5814783,
			},
		},
	}
	bodyJSON, _ := json.Marshal(body)

	req := newRequestWithParams("POST", "/api/webhooks/123/token123?wait=true", bodyJSON, map[string]string{
		"webhookID":    "123",
		"webhookToken": "token123",
	})

	w := httptest.NewRecorder()
	plugin.executeWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var response WebhookMessage
	json.NewDecoder(w.Body).Decode(&response)
	if response.Embeds == "" {
		t.Fatal("Embeds should be stored")
	}
}

func TestGetWebhook(t *testing.T) {
	plugin := setupTestPlugin(t)

	// Create webhook first by executing it
	body := map[string]interface{}{"content": "Setup"}
	bodyJSON, _ := json.Marshal(body)
	req := newRequestWithParams("POST", "/api/webhooks/123/token123", bodyJSON, map[string]string{
		"webhookID":    "123",
		"webhookToken": "token123",
	})
	w := httptest.NewRecorder()
	plugin.executeWebhook(w, req)

	// Now get the webhook
	req = newRequestWithParams("GET", "/api/webhooks/123/token123", nil, map[string]string{
		"webhookID":    "123",
		"webhookToken": "token123",
	})

	w = httptest.NewRecorder()
	plugin.getWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var webhook Webhook
	if err := json.NewDecoder(w.Body).Decode(&webhook); err != nil {
		t.Fatalf("Failed to decode webhook: %v", err)
	}
	if webhook.ID != "123" {
		t.Fatalf("Expected ID '123', got '%s'", webhook.ID)
	}
	if webhook.Token != "token123" {
		t.Fatalf("Expected token 'token123', got '%s'", webhook.Token)
	}
	if webhook.Name != "Incoming Webhook" {
		t.Fatalf("Expected default name, got '%s'", webhook.Name)
	}
}

func TestGetWebhookNotFound(t *testing.T) {
	plugin := setupTestPlugin(t)

	req := newRequestWithParams("GET", "/api/webhooks/999/invalid", nil, map[string]string{
		"webhookID":    "999",
		"webhookToken": "invalid",
	})

	w := httptest.NewRecorder()
	plugin.getWebhook(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected 404, got %d", w.Code)
	}
}

func TestModifyWebhook(t *testing.T) {
	plugin := setupTestPlugin(t)

	// Create webhook first
	plugin.store.GetOrCreateWebhook("123", "token123")

	// Modify webhook
	body := map[string]interface{}{
		"name":   "My Custom Webhook",
		"avatar": "https://example.com/avatar.png",
	}
	bodyJSON, _ := json.Marshal(body)

	req := newRequestWithParams("PATCH", "/api/webhooks/123/token123", bodyJSON, map[string]string{
		"webhookID":    "123",
		"webhookToken": "token123",
	})

	w := httptest.NewRecorder()
	plugin.modifyWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var webhook Webhook
	json.NewDecoder(w.Body).Decode(&webhook)
	if webhook.Name != "My Custom Webhook" {
		t.Fatalf("Expected name 'My Custom Webhook', got '%s'", webhook.Name)
	}
	if webhook.Avatar != "https://example.com/avatar.png" {
		t.Fatalf("Expected avatar URL, got '%s'", webhook.Avatar)
	}
}

func TestDeleteWebhook(t *testing.T) {
	plugin := setupTestPlugin(t)

	// Create webhook first
	plugin.store.GetOrCreateWebhook("123", "token123")

	// Delete webhook
	req := newRequestWithParams("DELETE", "/api/webhooks/123/token123", nil, map[string]string{
		"webhookID":    "123",
		"webhookToken": "token123",
	})

	w := httptest.NewRecorder()
	plugin.deleteWebhook(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("Expected 204, got %d", w.Code)
	}

	// Verify webhook is deleted
	_, err := plugin.store.GetWebhook("123", "token123")
	if err != sql.ErrNoRows {
		t.Fatal("Webhook should be deleted")
	}
}

func TestGetWebhookMessage(t *testing.T) {
	plugin := setupTestPlugin(t)

	// Create webhook and message
	webhook, _ := plugin.store.GetOrCreateWebhook("123", "token123")
	msg := &WebhookMessage{
		WebhookID: webhook.ID,
		Content:   "Test message",
		Username:  "TestBot",
	}
	plugin.store.CreateMessage(msg)

	// Get message
	req := newRequestWithParams("GET", "/api/webhooks/123/token123/messages/"+msg.ID, nil, map[string]string{
		"webhookID": "123",
		"messageID": msg.ID,
	})

	w := httptest.NewRecorder()
	plugin.getWebhookMessage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var response WebhookMessage
	json.NewDecoder(w.Body).Decode(&response)
	if response.Content != "Test message" {
		t.Fatalf("Expected content 'Test message', got '%s'", response.Content)
	}
}

func TestEditWebhookMessage(t *testing.T) {
	plugin := setupTestPlugin(t)

	// Create webhook and message
	webhook, _ := plugin.store.GetOrCreateWebhook("123", "token123")
	msg := &WebhookMessage{
		WebhookID: webhook.ID,
		Content:   "Original content",
	}
	plugin.store.CreateMessage(msg)

	// Edit message
	editBody := map[string]interface{}{
		"content": "Updated content",
	}
	bodyJSON, _ := json.Marshal(editBody)

	req := newRequestWithParams("PATCH", "/api/webhooks/123/token123/messages/"+msg.ID, bodyJSON, map[string]string{
		"webhookID": "123",
		"messageID": msg.ID,
	})

	w := httptest.NewRecorder()
	plugin.editWebhookMessage(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var response WebhookMessage
	json.NewDecoder(w.Body).Decode(&response)
	if response.Content != "Updated content" {
		t.Fatalf("Expected updated content, got '%s'", response.Content)
	}
	if response.EditedAt == nil {
		t.Fatal("EditedAt should be set")
	}
}

func TestDeleteWebhookMessage(t *testing.T) {
	plugin := setupTestPlugin(t)

	// Create webhook and message
	webhook, _ := plugin.store.GetOrCreateWebhook("123", "token123")
	msg := &WebhookMessage{
		WebhookID: webhook.ID,
		Content:   "To delete",
	}
	plugin.store.CreateMessage(msg)

	// Delete message
	req := newRequestWithParams("DELETE", "/api/webhooks/123/token123/messages/"+msg.ID, nil, map[string]string{
		"webhookID": "123",
		"messageID": msg.ID,
	})

	w := httptest.NewRecorder()
	plugin.deleteWebhookMessage(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("Expected 204, got %d", w.Code)
	}

	// Verify message is deleted
	_, err := plugin.store.GetMessage(webhook.ID, msg.ID)
	if err != sql.ErrNoRows {
		t.Fatal("Message should be deleted")
	}
}

func TestExecuteWebhookInvalidJSON(t *testing.T) {
	plugin := setupTestPlugin(t)

	req := newRequestWithParams("POST", "/api/webhooks/123/token123", []byte("invalid json"), map[string]string{
		"webhookID":    "123",
		"webhookToken": "token123",
	})

	w := httptest.NewRecorder()
	plugin.executeWebhook(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400, got %d", w.Code)
	}
}

func TestModifyWebhookNotFound(t *testing.T) {
	plugin := setupTestPlugin(t)

	body := map[string]interface{}{
		"name": "New Name",
	}
	bodyJSON, _ := json.Marshal(body)

	req := newRequestWithParams("PATCH", "/api/webhooks/999/invalid", bodyJSON, map[string]string{
		"webhookID":    "999",
		"webhookToken": "invalid",
	})

	w := httptest.NewRecorder()
	plugin.modifyWebhook(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected 404, got %d", w.Code)
	}
}

func TestGetMessageNotFound(t *testing.T) {
	plugin := setupTestPlugin(t)

	req := newRequestWithParams("GET", "/api/webhooks/123/token123/messages/999", nil, map[string]string{
		"webhookID": "123",
		"messageID": "999",
	})

	w := httptest.NewRecorder()
	plugin.getWebhookMessage(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected 404, got %d", w.Code)
	}
}

func TestExecuteWebhookWithThreadID(t *testing.T) {
	plugin := setupTestPlugin(t)

	body := map[string]interface{}{
		"content": "Thread message",
	}
	bodyJSON, _ := json.Marshal(body)

	req := newRequestWithParams("POST", "/api/webhooks/123/token123?wait=true&thread_id=thread_789", bodyJSON, map[string]string{
		"webhookID":    "123",
		"webhookToken": "token123",
	})

	w := httptest.NewRecorder()
	plugin.executeWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}

	var response WebhookMessage
	json.NewDecoder(w.Body).Decode(&response)
	if response.ThreadID != "thread_789" {
		t.Fatalf("Expected thread_id 'thread_789', got '%s'", response.ThreadID)
	}
}
