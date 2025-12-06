// ABOUTME: Unit tests for Twilio API HTTP handlers
// ABOUTME: Tests SMS and Voice API endpoints

package twilio

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestPlugin(t *testing.T) (*TwilioPlugin, *sql.DB) {
	db := setupTestDB(t)
	plugin := &TwilioPlugin{}
	if err := plugin.SetDB(db); err != nil {
		t.Fatalf("Failed to set DB: %v", err)
	}
	return plugin, db
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

func TestSendMessage(t *testing.T) {
	plugin, db := setupTestPlugin(t)
	defer db.Close()

	// Create account first
	account, _ := plugin.store.GetOrCreateAccount("AC123")

	// Prepare request
	form := url.Values{}
	form.Set("To", "+15551234567")
	form.Set("From", "+15559876543")
	form.Set("Body", "Test message")

	req := httptest.NewRequest("POST", "/2010-04-01/Accounts/AC123/Messages.json", bytes.NewBufferString(form.Encode()))
	req.Header.Set("Authorization", basicAuth("AC123", account.AuthToken))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler := plugin.requireAuth(plugin.sendMessage)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !strings.HasPrefix(response["sid"].(string), "SM") {
		t.Fatalf("Expected SID to start with SM, got %s", response["sid"])
	}

	if response["to"] != "+15551234567" {
		t.Fatalf("Expected to +15551234567, got %s", response["to"])
	}

	if response["status"] != "queued" {
		t.Fatalf("Expected status queued, got %s", response["status"])
	}
}

func TestGetMessage(t *testing.T) {
	plugin, db := setupTestPlugin(t)
	defer db.Close()

	account, _ := plugin.store.GetOrCreateAccount("AC123")

	// Create a message first
	msg, _ := plugin.store.CreateMessage("AC123", "+15551234567", "+15559876543", "Test message")

	// Get the message
	req := httptest.NewRequest("GET", "/2010-04-01/Accounts/AC123/Messages/"+msg.Sid+".json", nil)
	req.Header.Set("Authorization", basicAuth("AC123", account.AuthToken))

	rr := httptest.NewRecorder()

	// Need to set URL params manually for chi.URLParam to work
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("MessageSid", msg.Sid)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler := plugin.requireAuth(plugin.getMessage)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["sid"] != msg.Sid {
		t.Fatalf("Expected sid %s, got %s", msg.Sid, response["sid"])
	}
}

func TestListMessages(t *testing.T) {
	plugin, db := setupTestPlugin(t)
	defer db.Close()

	account, _ := plugin.store.GetOrCreateAccount("AC456")

	// Create some test messages
	plugin.store.CreateMessage("AC456", "+15551111111", "+15552222222", "Message 1")
	plugin.store.CreateMessage("AC456", "+15551111111", "+15553333333", "Message 2")
	plugin.store.CreateMessage("AC456", "+15551111111", "+15554444444", "Message 3")

	req := httptest.NewRequest("GET", "/2010-04-01/Accounts/AC456/Messages.json", nil)
	req.Header.Set("Authorization", basicAuth("AC456", account.AuthToken))

	rr := httptest.NewRecorder()
	handler := plugin.requireAuth(plugin.listMessages)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&response)

	messages := response["messages"].([]interface{})
	if len(messages) != 3 {
		t.Fatalf("Expected 3 messages, got %d", len(messages))
	}
}

func TestSendMessageMissingBody(t *testing.T) {
	plugin, db := setupTestPlugin(t)
	defer db.Close()

	account, _ := plugin.store.GetOrCreateAccount("AC789")

	form := url.Values{}
	form.Set("To", "+15551234567")
	form.Set("From", "+15559876543")
	// No Body set

	req := httptest.NewRequest("POST", "/2010-04-01/Accounts/AC789/Messages.json", bytes.NewBufferString(form.Encode()))
	req.Header.Set("Authorization", basicAuth("AC789", account.AuthToken))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler := plugin.requireAuth(plugin.sendMessage)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", rr.Code)
	}
}

func TestInitiateCall(t *testing.T) {
	plugin, db := setupTestPlugin(t)
	defer db.Close()

	account, _ := plugin.store.GetOrCreateAccount("AC456")

	form := url.Values{}
	form.Set("To", "+15551234567")
	form.Set("From", "+15559876543")
	form.Set("Url", "http://example.com/twiml")

	req := httptest.NewRequest("POST", "/2010-04-01/Accounts/AC456/Calls.json", bytes.NewBufferString(form.Encode()))
	req.Header.Set("Authorization", basicAuth("AC456", account.AuthToken))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler := plugin.requireAuth(plugin.initiateCall)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&response)

	if !strings.HasPrefix(response["sid"].(string), "CA") {
		t.Fatalf("Expected SID to start with CA, got %s", response["sid"])
	}

	if response["status"] != "initiated" {
		t.Fatalf("Expected status initiated, got %s", response["status"])
	}
}

func TestInvalidPhoneNumbers(t *testing.T) {
	plugin, db := setupTestPlugin(t)
	defer db.Close()

	account, _ := plugin.store.GetOrCreateAccount("AC999")

	tests := []struct {
		name string
		to   string
		from string
	}{
		{"missing plus", "15551234567", "+15559876543"},
		{"starts with zero", "+05551234567", "+15559876543"},
		{"too short", "+1", "+15559876543"},
		{"invalid characters", "+1-555-123-4567", "+15559876543"},
		{"invalid from", "+15551234567", "5559876543"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := url.Values{}
			form.Set("To", tt.to)
			form.Set("From", tt.from)
			form.Set("Body", "Test")

			req := httptest.NewRequest("POST", "/2010-04-01/Accounts/AC999/Messages.json", bytes.NewBufferString(form.Encode()))
			req.Header.Set("Authorization", basicAuth("AC999", account.AuthToken))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			rr := httptest.NewRecorder()
			handler := plugin.requireAuth(plugin.sendMessage)
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400 for %s, got %d", tt.name, rr.Code)
			}
		})
	}
}

func TestMessageBodyTooLong(t *testing.T) {
	plugin, db := setupTestPlugin(t)
	defer db.Close()

	account, _ := plugin.store.GetOrCreateAccount("AC888")

	// Create a body longer than 1600 characters
	longBody := strings.Repeat("a", 1601)

	form := url.Values{}
	form.Set("To", "+15551234567")
	form.Set("From", "+15559876543")
	form.Set("Body", longBody)

	req := httptest.NewRequest("POST", "/2010-04-01/Accounts/AC888/Messages.json", bytes.NewBufferString(form.Encode()))
	req.Header.Set("Authorization", basicAuth("AC888", account.AuthToken))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	rr := httptest.NewRecorder()
	handler := plugin.requireAuth(plugin.sendMessage)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400 for long body, got %d", rr.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&response)

	if !strings.Contains(response["message"].(string), "maximum length") {
		t.Errorf("Expected error message about maximum length, got %s", response["message"])
	}
}

func TestValidPhoneNumbers(t *testing.T) {
	tests := []struct {
		phone string
		valid bool
	}{
		{"+15551234567", true},
		{"+442012345678", true},
		{"+33123456789", true},
		{"+123456789012345", true}, // max length (15 digits)
		{"15551234567", false},      // missing +
		{"+05551234567", false},     // starts with 0
		{"+1", false},               // too short
		{"+1234567890123456", false}, // too long (16 digits)
	}

	for _, tt := range tests {
		t.Run(tt.phone, func(t *testing.T) {
			result := validatePhoneNumber(tt.phone)
			if result != tt.valid {
				t.Errorf("validatePhoneNumber(%s) = %v, expected %v", tt.phone, result, tt.valid)
			}
		})
	}
}
