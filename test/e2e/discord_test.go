// ABOUTME: E2E integration tests for Discord webhook plugin
// ABOUTME: Tests full webhook flow including create, send, edit, delete

package e2e_test

import (
	"encoding/json"
	"testing"
)

func TestDiscordWebhookFlow(t *testing.T) {
	srv := StartTestServer(t)
	defer srv.Close()

	webhookID := "test_webhook_123"
	webhookToken := "test_token_abc"
	baseURL := "/api/webhooks/" + webhookID + "/" + webhookToken

	// Step 1: Execute webhook with wait=true to get message back
	executeReq := map[string]interface{}{
		"content":    "Hello from E2E test!",
		"username":   "E2E Bot",
		"avatar_url": "https://example.com/avatar.png",
	}

	resp := srv.POST(t, baseURL+"?wait=true", executeReq)
	AssertStatusCode(t, resp, 200)

	var msg map[string]interface{}
	DecodeJSON(t, resp, &msg)

	messageID, ok := msg["ID"].(string)
	if !ok {
		t.Fatal("Message ID not returned")
	}
	if msg["Content"] != "Hello from E2E test!" {
		t.Fatalf("Expected content 'Hello from E2E test!', got '%v'", msg["Content"])
	}

	// Step 2: Get the webhook details
	resp = srv.GET(t, baseURL)
	AssertStatusCode(t, resp, 200)

	var webhook map[string]interface{}
	DecodeJSON(t, resp, &webhook)

	if webhook["ID"] != webhookID {
		t.Fatalf("Expected webhook ID '%s', got '%v'", webhookID, webhook["ID"])
	}
	if webhook["Token"] != webhookToken {
		t.Fatalf("Expected webhook token '%s', got '%v'", webhookToken, webhook["Token"])
	}
	if webhook["Name"] != "Incoming Webhook" {
		t.Fatalf("Expected default name, got '%v'", webhook["Name"])
	}

	// Step 3: Modify webhook
	modifyReq := map[string]interface{}{
		"name":   "E2E Test Webhook",
		"avatar": "https://example.com/new-avatar.png",
	}

	resp = srv.PATCH(t, baseURL, modifyReq)
	AssertStatusCode(t, resp, 200)

	var modifiedWebhook map[string]interface{}
	DecodeJSON(t, resp, &modifiedWebhook)

	if modifiedWebhook["Name"] != "E2E Test Webhook" {
		t.Fatalf("Expected modified name, got '%v'", modifiedWebhook["Name"])
	}

	// Step 4: Get the message
	resp = srv.GET(t, baseURL+"/messages/"+messageID)
	AssertStatusCode(t, resp, 200)

	var getMessage map[string]interface{}
	DecodeJSON(t, resp, &getMessage)

	if getMessage["Content"] != "Hello from E2E test!" {
		t.Fatalf("Message content mismatch, got '%v'", getMessage["Content"])
	}

	// Step 5: Edit the message
	editReq := map[string]interface{}{
		"content": "Updated E2E content!",
		"embeds": []map[string]interface{}{
			{
				"title":       "Test Embed",
				"description": "This is an E2E test embed",
				"color":       5814783,
			},
		},
	}

	resp = srv.PATCH(t, baseURL+"/messages/"+messageID, editReq)
	AssertStatusCode(t, resp, 200)

	var editedMsg map[string]interface{}
	DecodeJSON(t, resp, &editedMsg)

	if editedMsg["Content"] != "Updated E2E content!" {
		t.Fatalf("Expected updated content, got '%v'", editedMsg["Content"])
	}
	if editedMsg["EditedAt"] == nil {
		t.Fatal("EditedAt should be set after edit")
	}

	// Step 6: Delete the message
	resp = srv.DELETE(t, baseURL+"/messages/"+messageID)
	AssertStatusCode(t, resp, 204)

	// Step 7: Verify message is gone
	resp = srv.GET(t, baseURL+"/messages/"+messageID)
	AssertStatusCode(t, resp, 404)

	// Step 8: Delete the webhook
	resp = srv.DELETE(t, baseURL)
	AssertStatusCode(t, resp, 204)

	// Step 9: Verify webhook is gone
	resp = srv.GET(t, baseURL)
	AssertStatusCode(t, resp, 404)
}

func TestDiscordMultipleMessages(t *testing.T) {
	srv := StartTestServer(t)
	defer srv.Close()

	webhookID := "test_webhook_multi"
	webhookToken := "test_token_multi"
	baseURL := "/api/webhooks/" + webhookID + "/" + webhookToken

	// Send multiple messages
	messages := []string{
		"First message",
		"Second message",
		"Third message",
	}

	var messageIDs []string
	for _, content := range messages {
		req := map[string]interface{}{
			"content": content,
		}
		resp := srv.POST(t, baseURL+"?wait=true", req)
		AssertStatusCode(t, resp, 200)

		var msg map[string]interface{}
		DecodeJSON(t, resp, &msg)
		messageIDs = append(messageIDs, msg["ID"].(string))
	}

	// Verify all messages exist
	for i, msgID := range messageIDs {
		resp := srv.GET(t, baseURL+"/messages/"+msgID)
		AssertStatusCode(t, resp, 200)

		var msg map[string]interface{}
		DecodeJSON(t, resp, &msg)

		if msg["Content"] != messages[i] {
			t.Fatalf("Message %d content mismatch", i)
		}
	}
}

func TestDiscordWebhookWithoutWait(t *testing.T) {
	srv := StartTestServer(t)
	defer srv.Close()

	webhookID := "test_webhook_nowait"
	webhookToken := "test_token_nowait"
	baseURL := "/api/webhooks/" + webhookID + "/" + webhookToken

	// Execute webhook without wait parameter
	req := map[string]interface{}{
		"content": "No wait message",
	}

	resp := srv.POST(t, baseURL, req)
	// Should return 204 No Content when wait is not true
	AssertStatusCode(t, resp, 204)
}

func TestDiscordWebhookWithEmbeds(t *testing.T) {
	srv := StartTestServer(t)
	defer srv.Close()

	webhookID := "test_webhook_embeds"
	webhookToken := "test_token_embeds"
	baseURL := "/api/webhooks/" + webhookID + "/" + webhookToken

	// Send message with embeds and components
	req := map[string]interface{}{
		"content":  "Message with rich content",
		"username": "Rich Bot",
		"embeds": []map[string]interface{}{
			{
				"title":       "Embed Title",
				"description": "Embed description with lots of content",
				"color":       3447003,
				"fields": []map[string]interface{}{
					{
						"name":   "Field 1",
						"value":  "Value 1",
						"inline": true,
					},
					{
						"name":   "Field 2",
						"value":  "Value 2",
						"inline": true,
					},
				},
			},
		},
		"components": []map[string]interface{}{
			{
				"type": 1,
				"components": []map[string]interface{}{
					{
						"type":  2,
						"label": "Click Me",
						"style": 1,
					},
				},
			},
		},
	}

	resp := srv.POST(t, baseURL+"?wait=true", req)
	AssertStatusCode(t, resp, 200)

	var msg map[string]interface{}
	DecodeJSON(t, resp, &msg)

	if msg["Content"] != "Message with rich content" {
		t.Fatalf("Content mismatch, got '%v'", msg["Content"])
	}
	if msg["Username"] != "Rich Bot" {
		t.Fatalf("Username mismatch, got '%v'", msg["Username"])
	}

	// Verify embeds were stored
	embedsStr, ok := msg["Embeds"].(string)
	if !ok || embedsStr == "" {
		t.Fatal("Embeds should be stored as JSON string")
	}

	var embeds []map[string]interface{}
	if err := json.Unmarshal([]byte(embedsStr), &embeds); err != nil {
		t.Fatalf("Failed to parse embeds JSON: %v", err)
	}
	if len(embeds) != 1 {
		t.Fatalf("Expected 1 embed, got %d", len(embeds))
	}
	if embeds[0]["title"] != "Embed Title" {
		t.Fatalf("Embed title mismatch, got '%v'", embeds[0]["title"])
	}
}

func TestDiscordWebhookWithThreadID(t *testing.T) {
	srv := StartTestServer(t)
	defer srv.Close()

	webhookID := "test_webhook_thread"
	webhookToken := "test_token_thread"
	threadID := "thread_12345"
	baseURL := "/api/webhooks/" + webhookID + "/" + webhookToken

	// Send message to thread
	req := map[string]interface{}{
		"content": "Thread message",
	}

	resp := srv.POST(t, baseURL+"?wait=true&thread_id="+threadID, req)
	AssertStatusCode(t, resp, 200)

	var msg map[string]interface{}
	DecodeJSON(t, resp, &msg)

	if msg["ThreadID"] != threadID {
		t.Fatalf("Expected thread_id '%s', got '%v'", threadID, msg["ThreadID"])
	}
}

func TestDiscordErrorHandling(t *testing.T) {
	srv := StartTestServer(t)
	defer srv.Close()

	webhookID := "test_webhook_errors"
	webhookToken := "test_token_errors"
	baseURL := "/api/webhooks/" + webhookID + "/" + webhookToken

	// Test invalid JSON
	resp := srv.POST(t, baseURL, "invalid json")
	AssertStatusCode(t, resp, 400)

	// Test get non-existent message
	resp = srv.GET(t, baseURL+"/messages/999999")
	AssertStatusCode(t, resp, 404)

	// Test edit non-existent message
	editReq := map[string]interface{}{
		"content": "Updated",
	}
	resp = srv.PATCH(t, baseURL+"/messages/999999", editReq)
	AssertStatusCode(t, resp, 404)
}
