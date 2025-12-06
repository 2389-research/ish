// ABOUTME: Integration tests for Twilio plugin
// ABOUTME: Tests end-to-end flows including webhook delivery

package twilio

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestFullSMSFlow(t *testing.T) {
	plugin, db := setupTestPlugin(t)
	defer db.Close()

	// Create account with phone number and webhook config
	account, _ := plugin.store.GetOrCreateAccount("ACTEST")
	phoneNumber, _ := plugin.store.CreatePhoneNumber("ACTEST", "+15551234567", "Test Phone")

	// Update phone number with status callback
	_, err := db.Exec(`
		UPDATE twilio_phone_numbers
		SET status_callback = 'http://example.com/webhook'
		WHERE sid = ?
	`, phoneNumber.Sid)
	if err != nil {
		t.Fatalf("Failed to set webhook: %v", err)
	}

	// Send message
	form := url.Values{}
	form.Set("To", "+15559876543")
	form.Set("From", "+15551234567")
	form.Set("Body", "Integration test")

	req := httptest.NewRequest("POST", "/2010-04-01/Accounts/ACTEST/Messages.json",
		bytes.NewBufferString(form.Encode()))
	req.Header.Set("Authorization", basicAuth("ACTEST", account.AuthToken))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler := plugin.requireAuth(plugin.sendMessage)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&response)
	messageSid := response["sid"].(string)

	// Wait for lifecycle to complete (sent at 100ms, delivered at 600ms)
	time.Sleep(1000 * time.Millisecond)

	// Verify message status has progressed (at least to sent, ideally delivered)
	msg, err := plugin.store.GetMessage(messageSid)
	if err != nil {
		t.Fatalf("Failed to get message: %v", err)
	}
	if msg.Status != "sent" && msg.Status != "delivered" {
		t.Fatalf("Expected status sent or delivered, got %s", msg.Status)
	}

	// Verify webhooks were queued (at least 2 for queued, sent; may have 3 with delivered)
	var webhookCount int
	db.QueryRow("SELECT COUNT(*) FROM twilio_webhook_queue WHERE resource_sid = ?", messageSid).Scan(&webhookCount)
	if webhookCount < 2 {
		t.Fatalf("Expected at least 2 webhooks queued, got %d", webhookCount)
	}
}

func TestFullCallFlow(t *testing.T) {
	plugin, db := setupTestPlugin(t)
	defer db.Close()

	account, _ := plugin.store.GetOrCreateAccount("ACTEST2")
	phoneNumber, _ := plugin.store.CreatePhoneNumber("ACTEST2", "+15551111111", "Test Phone")

	db.Exec(`
		UPDATE twilio_phone_numbers
		SET status_callback = 'http://example.com/call-webhook'
		WHERE sid = ?
	`, phoneNumber.Sid)

	// Initiate call
	form := url.Values{}
	form.Set("To", "+15559999999")
	form.Set("From", "+15551111111")
	form.Set("Url", "http://example.com/twiml")

	req := httptest.NewRequest("POST", "/2010-04-01/Accounts/ACTEST2/Calls.json",
		bytes.NewBufferString(form.Encode()))
	req.Header.Set("Authorization", basicAuth("ACTEST2", account.AuthToken))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler := plugin.requireAuth(plugin.initiateCall)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&response)
	callSid := response["sid"].(string)

	// Wait for ringing status (occurs at 200ms)
	time.Sleep(400 * time.Millisecond)

	call, err := plugin.store.GetCall(callSid)
	if err != nil {
		t.Fatalf("Failed to get call: %v", err)
	}
	if call.Status != "ringing" {
		t.Fatalf("Expected status ringing, got %s", call.Status)
	}

	// Wait for in-progress (occurs at 1000ms)
	time.Sleep(1100 * time.Millisecond)

	call, err = plugin.store.GetCall(callSid)
	if err != nil {
		t.Fatalf("Failed to get call: %v", err)
	}
	if call.Status != "in-progress" {
		t.Fatalf("Expected status in-progress, got %s", call.Status)
	}

	// Verify webhooks were queued (at least 3 for initiated, ringing, in-progress)
	var webhookCount int
	db.QueryRow("SELECT COUNT(*) FROM twilio_webhook_queue WHERE resource_sid = ?", callSid).Scan(&webhookCount)
	if webhookCount < 3 {
		t.Fatalf("Expected at least 3 webhooks queued, got %d", webhookCount)
	}
}

func TestAuthValidation(t *testing.T) {
	plugin, db := setupTestPlugin(t)
	defer db.Close()

	// Create account
	account, _ := plugin.store.GetOrCreateAccount("ACAUTH")

	// Test 1: Valid auth
	req := httptest.NewRequest("GET", "/2010-04-01/Accounts/ACAUTH/Messages.json", nil)
	req.Header.Set("Authorization", basicAuth("ACAUTH", account.AuthToken))

	rr := httptest.NewRecorder()
	handler := plugin.requireAuth(plugin.listMessages)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected 200 with valid auth, got %d", rr.Code)
	}

	// Test 2: Invalid token
	req = httptest.NewRequest("GET", "/2010-04-01/Accounts/ACAUTH/Messages.json", nil)
	req.Header.Set("Authorization", basicAuth("ACAUTH", "wrongtoken"))

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 with invalid token, got %d", rr.Code)
	}

	// Test 3: Invalid account (but will auto-create, then fail validation)
	req = httptest.NewRequest("GET", "/2010-04-01/Accounts/ACNONEXISTENT/Messages.json", nil)
	req.Header.Set("Authorization", basicAuth("ACNONEXISTENT", "anytoken"))

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 with invalid account token, got %d", rr.Code)
	}

	// Test 4: Missing auth
	req = httptest.NewRequest("GET", "/2010-04-01/Accounts/ACAUTH/Messages.json", nil)

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 with missing auth, got %d", rr.Code)
	}
}
