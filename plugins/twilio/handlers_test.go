// ABOUTME: Unit tests for Twilio API HTTP handlers
// ABOUTME: Tests SMS and Voice API endpoints

package twilio

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

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
