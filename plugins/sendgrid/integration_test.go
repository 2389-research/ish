// ABOUTME: Integration tests for SendGrid plugin
// ABOUTME: Tests mail sending, message retrieval, and suppression management

package sendgrid

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) (*sql.DB, *SendGridPlugin) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}

	plugin := &SendGridPlugin{}
	if err := plugin.SetDB(db); err != nil {
		t.Fatalf("Failed to initialize plugin: %v", err)
	}

	return db, plugin
}

func TestSendMail(t *testing.T) {
	db, plugin := setupTestDB(t)
	defer db.Close()

	// Create test account and API key
	account, err := plugin.store.CreateAccount("test@example.com", "Test User")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	apiKey, err := plugin.store.CreateAPIKey(account.ID, "Test Key", "mail.send")
	if err != nil {
		t.Fatalf("Failed to create API key: %v", err)
	}

	// Test sending a mail
	requestBody := SendMailRequest{
		Personalizations: []Personalization{
			{
				To: []EmailAddress{
					{Email: "recipient@example.com", Name: "Recipient"},
				},
				Subject: "Test Email",
			},
		},
		From: EmailAddress{
			Email: "sender@example.com",
			Name:  "Sender",
		},
		Subject: "Test Email",
		Content: []Content{
			{Type: "text/plain", Value: "This is a test email"},
			{Type: "text/html", Value: "<p>This is a test email</p>"},
		},
	}

	bodyBytes, _ := json.Marshal(requestBody)
	req := httptest.NewRequest(http.MethodPost, "/v3/mail/send", bytes.NewReader(bodyBytes))
	req.Header.Set("Authorization", "Bearer "+apiKey.Key)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	router := chi.NewRouter()
	plugin.RegisterRoutes(router)
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Errorf("Expected status %d, got %d: %s", http.StatusAccepted, rr.Code, rr.Body.String())
	}

	// Verify message was created
	messageID := rr.Header().Get("X-Message-Id")
	if messageID == "" {
		t.Error("Expected X-Message-Id header")
	}

	message, err := plugin.store.GetMessage(messageID)
	if err != nil {
		t.Fatalf("Failed to get message: %v", err)
	}

	if message.ToEmail != "recipient@example.com" {
		t.Errorf("Expected to_email 'recipient@example.com', got '%s'", message.ToEmail)
	}

	if message.Subject != "Test Email" {
		t.Errorf("Expected subject 'Test Email', got '%s'", message.Subject)
	}

	if message.Status != "delivered" {
		t.Errorf("Expected status 'delivered', got '%s'", message.Status)
	}
}

func TestSendMailValidation(t *testing.T) {
	db, plugin := setupTestDB(t)
	defer db.Close()

	// Create test account and API key
	account, err := plugin.store.CreateAccount("test@example.com", "Test User")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	apiKey, err := plugin.store.CreateAPIKey(account.ID, "Test Key", "mail.send")
	if err != nil {
		t.Fatalf("Failed to create API key: %v", err)
	}

	tests := []struct {
		name           string
		requestBody    SendMailRequest
		expectedStatus int
		expectedError  string
	}{
		{
			name: "missing personalizations",
			requestBody: SendMailRequest{
				From: EmailAddress{Email: "sender@example.com"},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "personalizations is required",
		},
		{
			name: "missing to email",
			requestBody: SendMailRequest{
				Personalizations: []Personalization{
					{To: []EmailAddress{}},
				},
				From: EmailAddress{Email: "sender@example.com"},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "at least one 'to' email is required",
		},
		{
			name: "missing from email",
			requestBody: SendMailRequest{
				Personalizations: []Personalization{
					{To: []EmailAddress{{Email: "recipient@example.com"}}},
				},
				From: EmailAddress{},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "from email is required",
		},
	}

	router := chi.NewRouter()
	plugin.RegisterRoutes(router)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/v3/mail/send", bytes.NewReader(bodyBytes))
			req.Header.Set("Authorization", "Bearer "+apiKey.Key)
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			var response map[string]interface{}
			json.NewDecoder(rr.Body).Decode(&response)

			errors, ok := response["errors"].([]interface{})
			if !ok || len(errors) == 0 {
				t.Fatal("Expected errors in response")
			}

			firstError := errors[0].(map[string]interface{})
			if firstError["message"] != tt.expectedError {
				t.Errorf("Expected error '%s', got '%s'", tt.expectedError, firstError["message"])
			}
		})
	}
}

func TestListMessages(t *testing.T) {
	db, plugin := setupTestDB(t)
	defer db.Close()

	// Create test account and API key
	account, err := plugin.store.CreateAccount("test@example.com", "Test User")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	apiKey, err := plugin.store.CreateAPIKey(account.ID, "Test Key", "messages.read")
	if err != nil {
		t.Fatalf("Failed to create API key: %v", err)
	}

	// Create test messages
	for i := 0; i < 3; i++ {
		_, err := plugin.store.CreateMessage(
			account.ID,
			"sender@example.com",
			"Sender",
			"recipient@example.com",
			"Recipient",
			"Test Subject",
			"Test content",
			"<p>Test content</p>",
		)
		if err != nil {
			t.Fatalf("Failed to create message: %v", err)
		}
	}

	// Test listing messages
	req := httptest.NewRequest(http.MethodGet, "/v3/messages", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey.Key)

	rr := httptest.NewRecorder()

	router := chi.NewRouter()
	plugin.RegisterRoutes(router)
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var response map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&response)

	messages, ok := response["messages"].([]interface{})
	if !ok {
		t.Fatal("Expected messages array in response")
	}

	if len(messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(messages))
	}
}

func TestGetMessage(t *testing.T) {
	db, plugin := setupTestDB(t)
	defer db.Close()

	// Create test account and API key
	account, err := plugin.store.CreateAccount("test@example.com", "Test User")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	apiKey, err := plugin.store.CreateAPIKey(account.ID, "Test Key", "messages.read")
	if err != nil {
		t.Fatalf("Failed to create API key: %v", err)
	}

	// Create test message
	message, err := plugin.store.CreateMessage(
		account.ID,
		"sender@example.com",
		"Sender",
		"recipient@example.com",
		"Recipient",
		"Test Subject",
		"Test content",
		"<p>Test content</p>",
	)
	if err != nil {
		t.Fatalf("Failed to create message: %v", err)
	}

	// Test getting message
	req := httptest.NewRequest(http.MethodGet, "/v3/messages/"+message.ID, nil)
	req.Header.Set("Authorization", "Bearer "+apiKey.Key)

	rr := httptest.NewRecorder()

	router := chi.NewRouter()
	plugin.RegisterRoutes(router)
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d: %s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var response map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&response)

	if response["msg_id"] != message.ID {
		t.Errorf("Expected msg_id '%s', got '%s'", message.ID, response["msg_id"])
	}

	if response["subject"] != "Test Subject" {
		t.Errorf("Expected subject 'Test Subject', got '%s'", response["subject"])
	}
}

func TestSuppressions(t *testing.T) {
	db, plugin := setupTestDB(t)
	defer db.Close()

	// Create test account and API key
	account, err := plugin.store.CreateAccount("test@example.com", "Test User")
	if err != nil {
		t.Fatalf("Failed to create account: %v", err)
	}

	apiKey, err := plugin.store.CreateAPIKey(account.ID, "Test Key", "suppressions.manage")
	if err != nil {
		t.Fatalf("Failed to create API key: %v", err)
	}

	router := chi.NewRouter()
	plugin.RegisterRoutes(router)

	// Create test suppressions
	suppressions := []struct {
		email  string
		stype  string
		reason string
	}{
		{"bounce1@test.com", "bounce", "550 User not found"},
		{"bounce2@test.com", "bounce", "550 Mailbox full"},
		{"block1@test.com", "block", "User unsubscribed"},
		{"spam1@test.com", "spam_report", "Marked as spam"},
	}

	for _, supp := range suppressions {
		_, err := plugin.store.CreateSuppression(account.ID, supp.email, supp.stype, supp.reason)
		if err != nil {
			t.Fatalf("Failed to create suppression: %v", err)
		}
	}

	// Test listing bounces
	req := httptest.NewRequest(http.MethodGet, "/v3/suppression/bounces", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey.Key)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var bounces []interface{}
	json.NewDecoder(rr.Body).Decode(&bounces)

	if len(bounces) != 2 {
		t.Errorf("Expected 2 bounces, got %d", len(bounces))
	}

	// Test listing blocks
	req = httptest.NewRequest(http.MethodGet, "/v3/suppression/blocks", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey.Key)

	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var blocks []interface{}
	json.NewDecoder(rr.Body).Decode(&blocks)

	if len(blocks) != 1 {
		t.Errorf("Expected 1 block, got %d", len(blocks))
	}

	// Test deleting a bounce
	req = httptest.NewRequest(http.MethodDelete, "/v3/suppression/bounces/bounce1@test.com", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey.Key)

	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d", http.StatusNoContent, rr.Code)
	}

	// Verify deletion
	req = httptest.NewRequest(http.MethodGet, "/v3/suppression/bounces", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey.Key)

	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	json.NewDecoder(rr.Body).Decode(&bounces)

	if len(bounces) != 1 {
		t.Errorf("Expected 1 bounce after deletion, got %d", len(bounces))
	}
}

func TestAuthentication(t *testing.T) {
	db, plugin := setupTestDB(t)
	defer db.Close()

	router := chi.NewRouter()
	plugin.RegisterRoutes(router)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "missing auth header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid auth format",
			authHeader:     "InvalidFormat",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "invalid API key",
			authHeader:     "Bearer invalid_key_12345",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v3/messages", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}
