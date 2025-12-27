// ABOUTME: Tests for data persistence in Google plugin.
// ABOUTME: Verifies that POST data is persisted and returned on subsequent GET requests.

package google

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/2389/ish/internal/auth"
	"github.com/go-chi/chi/v5"
	_ "github.com/mattn/go-sqlite3"
)

// setupTestPlugin creates a GooglePlugin with an in-memory SQLite database
func setupTestPlugin(t *testing.T) *GooglePlugin {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}

	p := &GooglePlugin{}
	if err := p.SetDB(db); err != nil {
		t.Fatalf("failed to set database: %v", err)
	}

	return p
}

func TestTasksPersistence(t *testing.T) {
	p := setupTestPlugin(t)
	r := chi.NewRouter()
	r.Use(auth.Middleware)
	p.RegisterRoutes(r)

	// Step 1: Create a task via POST
	taskPayload := `{"title": "Test Task from Persistence Test", "notes": "This should persist"}`
	postReq := httptest.NewRequest("POST", "/tasks/v1/lists/@default/tasks", strings.NewReader(taskPayload))
	postReq.Header.Set("Content-Type", "application/json")
	postReq.Header.Set("Authorization", "Bearer test-token")
	postW := httptest.NewRecorder()

	r.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusCreated {
		t.Fatalf("POST task got status %d, want %d. Body: %s", postW.Code, http.StatusCreated, postW.Body.String())
	}

	var createdTask map[string]interface{}
	if err := json.NewDecoder(postW.Body).Decode(&createdTask); err != nil {
		t.Fatalf("failed to decode POST response: %v", err)
	}

	taskID, ok := createdTask["id"].(string)
	if !ok || taskID == "" {
		t.Fatalf("POST response missing task ID")
	}

	t.Logf("Created task with ID: %s", taskID)

	// Step 2: List tasks via GET and verify the created task is present
	getReq := httptest.NewRequest("GET", "/tasks/v1/lists/@default/tasks", nil)
	getReq.Header.Set("Authorization", "Bearer test-token")
	getW := httptest.NewRecorder()

	r.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("GET tasks got status %d, want %d. Body: %s", getW.Code, http.StatusOK, getW.Body.String())
	}

	var listResp map[string]interface{}
	if err := json.NewDecoder(getW.Body).Decode(&listResp); err != nil {
		t.Fatalf("failed to decode GET response: %v", err)
	}

	items, ok := listResp["items"].([]interface{})
	if !ok {
		t.Fatalf("GET response missing items array")
	}

	if len(items) == 0 {
		t.Fatalf("GET returned empty items array - task was not persisted! Response: %s", getW.Body.String())
	}

	// Verify our task is in the list
	found := false
	for _, item := range items {
		task, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if task["id"] == taskID {
			found = true
			if task["title"] != "Test Task from Persistence Test" {
				t.Errorf("task title mismatch: got %v, want 'Test Task from Persistence Test'", task["title"])
			}
			break
		}
	}

	if !found {
		t.Errorf("created task %s not found in GET response. Items: %v", taskID, items)
	}
}

// TestTasksPersistenceWithDifferentTokens verifies that Tasks work across token changes
// Since Tasks uses list_id (from URL) not user_id, this should always work
func TestTasksPersistenceWithDifferentTokens(t *testing.T) {
	p := setupTestPlugin(t)
	r := chi.NewRouter()
	r.Use(auth.Middleware)
	p.RegisterRoutes(r)

	// Step 1: Create a task with token A
	taskPayload := `{"title": "Task with Token A", "notes": "Should be visible with Token B"}`
	postReq := httptest.NewRequest("POST", "/tasks/v1/lists/@default/tasks", strings.NewReader(taskPayload))
	postReq.Header.Set("Content-Type", "application/json")
	postReq.Header.Set("Authorization", "Bearer ish-mock-access-token-12345")
	postW := httptest.NewRecorder()

	r.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusCreated {
		t.Fatalf("POST task got status %d, want %d. Body: %s", postW.Code, http.StatusCreated, postW.Body.String())
	}

	var createdTask map[string]interface{}
	if err := json.NewDecoder(postW.Body).Decode(&createdTask); err != nil {
		t.Fatalf("failed to decode POST response: %v", err)
	}

	taskID, ok := createdTask["id"].(string)
	if !ok || taskID == "" {
		t.Fatalf("POST response missing task ID")
	}

	t.Logf("Created task with ID: %s", taskID)

	// Step 2: List tasks with DIFFERENT token B
	getReq := httptest.NewRequest("GET", "/tasks/v1/lists/@default/tasks", nil)
	getReq.Header.Set("Authorization", "Bearer ish-mock-access-token-67890")
	getW := httptest.NewRecorder()

	r.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("GET tasks got status %d, want %d. Body: %s", getW.Code, http.StatusOK, getW.Body.String())
	}

	var listResp map[string]interface{}
	if err := json.NewDecoder(getW.Body).Decode(&listResp); err != nil {
		t.Fatalf("failed to decode GET response: %v", err)
	}

	items, ok := listResp["items"].([]interface{})
	if !ok {
		t.Fatalf("GET response missing items array")
	}

	if len(items) == 0 {
		t.Fatalf("GET returned empty items array - task was not persisted!")
	}

	// Verify our task is in the list
	found := false
	for _, item := range items {
		task, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if task["id"] == taskID {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("created task %s not found in GET response. Items: %v", taskID, items)
	}
}

func TestGmailPersistence(t *testing.T) {
	p := setupTestPlugin(t)
	r := chi.NewRouter()
	r.Use(auth.Middleware)
	p.RegisterRoutes(r)

	// Step 1: Send a message via POST
	// Gmail expects base64-encoded RFC 2822 message
	rawEmail := "From: test@example.com\r\nTo: recipient@example.com\r\nSubject: Test Email\r\n\r\nThis is a test email body."
	encodedEmail := base64.URLEncoding.EncodeToString([]byte(rawEmail))
	messagePayload := `{"raw": "` + encodedEmail + `"}`

	postReq := httptest.NewRequest("POST", "/gmail/v1/users/me/messages/send", strings.NewReader(messagePayload))
	postReq.Header.Set("Content-Type", "application/json")
	postReq.Header.Set("Authorization", "Bearer test-token")
	postW := httptest.NewRecorder()

	r.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusOK {
		t.Fatalf("POST message got status %d, want %d. Body: %s", postW.Code, http.StatusOK, postW.Body.String())
	}

	var sentMessage map[string]interface{}
	if err := json.NewDecoder(postW.Body).Decode(&sentMessage); err != nil {
		t.Fatalf("failed to decode POST response: %v", err)
	}

	messageID, ok := sentMessage["id"].(string)
	if !ok || messageID == "" {
		t.Fatalf("POST response missing message ID")
	}

	t.Logf("Sent message with ID: %s", messageID)

	// Step 2: List messages via GET and verify the sent message is present
	getReq := httptest.NewRequest("GET", "/gmail/v1/users/me/messages", nil)
	getReq.Header.Set("Authorization", "Bearer test-token")
	getW := httptest.NewRecorder()

	r.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("GET messages got status %d, want %d. Body: %s", getW.Code, http.StatusOK, getW.Body.String())
	}

	var listResp map[string]interface{}
	if err := json.NewDecoder(getW.Body).Decode(&listResp); err != nil {
		t.Fatalf("failed to decode GET response: %v", err)
	}

	messages, ok := listResp["messages"].([]interface{})
	if !ok {
		t.Fatalf("GET response missing messages array")
	}

	if len(messages) == 0 {
		t.Fatalf("GET returned empty messages array - message was not persisted! Response: %s", getW.Body.String())
	}

	// Verify our message is in the list
	found := false
	for _, item := range messages {
		msg, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if msg["id"] == messageID {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("sent message %s not found in GET response. Messages: %v", messageID, messages)
	}
}

// TestGmailPersistenceWithDifferentTokens reproduces the Docker issue where
// different access tokens are used for POST and GET requests
func TestGmailPersistenceWithDifferentTokens(t *testing.T) {
	p := setupTestPlugin(t)
	r := chi.NewRouter()
	r.Use(auth.Middleware)
	p.RegisterRoutes(r)

	// Step 1: Send a message with token A
	rawEmail := "From: test@example.com\r\nTo: recipient@example.com\r\nSubject: Test Email\r\n\r\nThis is a test email body."
	encodedEmail := base64.URLEncoding.EncodeToString([]byte(rawEmail))
	messagePayload := `{"raw": "` + encodedEmail + `"}`

	postReq := httptest.NewRequest("POST", "/gmail/v1/users/me/messages/send", strings.NewReader(messagePayload))
	postReq.Header.Set("Content-Type", "application/json")
	// Simulate token from OAuth - each token refresh generates a new token
	postReq.Header.Set("Authorization", "Bearer ish-mock-access-token-12345")
	postW := httptest.NewRecorder()

	r.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusOK {
		t.Fatalf("POST message got status %d, want %d. Body: %s", postW.Code, http.StatusOK, postW.Body.String())
	}

	var sentMessage map[string]interface{}
	if err := json.NewDecoder(postW.Body).Decode(&sentMessage); err != nil {
		t.Fatalf("failed to decode POST response: %v", err)
	}

	messageID, ok := sentMessage["id"].(string)
	if !ok || messageID == "" {
		t.Fatalf("POST response missing message ID")
	}

	t.Logf("Sent message with ID: %s", messageID)

	// Step 2: List messages with DIFFERENT token B (simulating token refresh)
	getReq := httptest.NewRequest("GET", "/gmail/v1/users/me/messages", nil)
	// Different token than POST - simulates token refresh
	getReq.Header.Set("Authorization", "Bearer ish-mock-access-token-67890")
	getW := httptest.NewRecorder()

	r.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("GET messages got status %d, want %d. Body: %s", getW.Code, http.StatusOK, getW.Body.String())
	}

	var listResp map[string]interface{}
	if err := json.NewDecoder(getW.Body).Decode(&listResp); err != nil {
		t.Fatalf("failed to decode GET response: %v", err)
	}

	messages, ok := listResp["messages"].([]interface{})
	if !ok {
		t.Fatalf("GET response missing messages array")
	}

	// This test expects to FAIL with current implementation!
	// The tokens are used as user IDs, so different tokens = different users = no data found
	if len(messages) == 0 {
		t.Fatalf("BUG REPRODUCED: GET returned empty messages array when using different token. " +
			"This is because the token is used as user_id, so different tokens = different users.")
	}

	// Verify our message is in the list
	found := false
	for _, item := range messages {
		msg, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if msg["id"] == messageID {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("sent message %s not found in GET response. Messages: %v", messageID, messages)
	}
}

// TestGmailWithAnyTokens verifies that ALL tokens map to the same user for mock server
func TestGmailWithAnyTokens(t *testing.T) {
	p := setupTestPlugin(t)
	r := chi.NewRouter()
	r.Use(auth.Middleware)
	p.RegisterRoutes(r)

	// Step 1: Send with a Google-style access token (ya29.*)
	rawEmail := "From: jeff@example.com\r\nTo: recipient@example.com\r\nSubject: Test\r\n\r\nTest."
	encodedEmail := base64.URLEncoding.EncodeToString([]byte(rawEmail))
	messagePayload := `{"raw": "` + encodedEmail + `"}`

	postReq := httptest.NewRequest("POST", "/gmail/v1/users/me/messages/send", strings.NewReader(messagePayload))
	postReq.Header.Set("Content-Type", "application/json")
	// Token that might come from Google OAuth (not ish-mock prefix)
	postReq.Header.Set("Authorization", "Bearer ya29.a0AfH6SMC...")
	postW := httptest.NewRecorder()

	r.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusOK {
		t.Fatalf("POST got status %d. Body: %s", postW.Code, postW.Body.String())
	}

	var sentMessage map[string]interface{}
	json.NewDecoder(postW.Body).Decode(&sentMessage)
	messageID := sentMessage["id"].(string)
	t.Logf("Sent message with ID: %s using ya29 token", messageID)

	// Step 2: List with a DIFFERENT token (any format) - should still find the message
	getReq := httptest.NewRequest("GET", "/gmail/v1/users/me/messages", nil)
	getReq.Header.Set("Authorization", "Bearer totally-different-token-format")
	getW := httptest.NewRecorder()

	r.ServeHTTP(getW, getReq)

	var listResp map[string]interface{}
	json.NewDecoder(getW.Body).Decode(&listResp)
	messages, _ := listResp["messages"].([]interface{})

	if len(messages) == 0 {
		t.Fatalf("BUG: Different tokens should still find messages in mock server mode")
	}

	// Verify the message is found
	found := false
	for _, item := range messages {
		msg := item.(map[string]interface{})
		if msg["id"] == messageID {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("message %s not found with different token format", messageID)
	}
}

// TestGmailWithRandomMockTokens simulates the Docker scenario where different
// random ish-mock tokens are used for POST and GET
func TestGmailWithRandomMockTokens(t *testing.T) {
	p := setupTestPlugin(t)
	r := chi.NewRouter()
	r.Use(auth.Middleware)
	p.RegisterRoutes(r)

	// Step 1: Send a message with a random ish-mock token (simulating jeff)
	rawEmail := "From: jeff@example.com\r\nTo: recipient@example.com\r\nSubject: Docker Test\r\n\r\nTest from Docker."
	encodedEmail := base64.URLEncoding.EncodeToString([]byte(rawEmail))
	messagePayload := `{"raw": "` + encodedEmail + `"}`

	postReq := httptest.NewRequest("POST", "/gmail/v1/users/me/messages/send", strings.NewReader(messagePayload))
	postReq.Header.Set("Content-Type", "application/json")
	// Token like what OAuth mock generates
	postReq.Header.Set("Authorization", "Bearer ish-mock-access-abc123def456")
	postW := httptest.NewRecorder()

	r.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusOK {
		t.Fatalf("POST message got status %d, want %d. Body: %s", postW.Code, http.StatusOK, postW.Body.String())
	}

	var sentMessage map[string]interface{}
	if err := json.NewDecoder(postW.Body).Decode(&sentMessage); err != nil {
		t.Fatalf("failed to decode POST response: %v", err)
	}

	messageID := sentMessage["id"].(string)
	t.Logf("jeff sent message with ID: %s", messageID)

	// Step 2: List messages with a DIFFERENT random ish-mock token (simulating eval harness)
	getReq := httptest.NewRequest("GET", "/gmail/v1/users/me/messages", nil)
	// Different random token, simulating eval harness
	getReq.Header.Set("Authorization", "Bearer ish-mock-access-xyz789uvw321")
	getW := httptest.NewRecorder()

	r.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("GET messages got status %d, want %d. Body: %s", getW.Code, http.StatusOK, getW.Body.String())
	}

	var listResp map[string]interface{}
	if err := json.NewDecoder(getW.Body).Decode(&listResp); err != nil {
		t.Fatalf("failed to decode GET response: %v", err)
	}

	messages, ok := listResp["messages"].([]interface{})
	if !ok {
		t.Fatalf("GET response missing messages array")
	}

	if len(messages) == 0 {
		t.Fatalf("BUG: Different ish-mock tokens returned empty messages. Both should map to same user!")
	}

	// Verify the message is found
	found := false
	for _, item := range messages {
		msg := item.(map[string]interface{})
		if msg["id"] == messageID {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("message %s not found with different ish-mock token", messageID)
	}
}

// TestTasksURLEncodingPersistence verifies that @default and %40default work the same
func TestTasksURLEncodingPersistence(t *testing.T) {
	p := setupTestPlugin(t)
	r := chi.NewRouter()
	r.Use(auth.Middleware)
	p.RegisterRoutes(r)

	// Step 1: Create a task using @default (unencoded)
	taskPayload := `{"title": "URL Encoding Test Task"}`
	postReq := httptest.NewRequest("POST", "/tasks/v1/lists/@default/tasks", strings.NewReader(taskPayload))
	postReq.Header.Set("Content-Type", "application/json")
	postReq.Header.Set("Authorization", "Bearer test-token")
	postW := httptest.NewRecorder()

	r.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusCreated {
		t.Fatalf("POST task got status %d, want %d. Body: %s", postW.Code, http.StatusCreated, postW.Body.String())
	}

	var createdTask map[string]interface{}
	if err := json.NewDecoder(postW.Body).Decode(&createdTask); err != nil {
		t.Fatalf("failed to decode POST response: %v", err)
	}

	taskID := createdTask["id"].(string)
	t.Logf("Created task with ID: %s using @default", taskID)

	// Step 2: List tasks using %40default (URL-encoded @)
	getReq := httptest.NewRequest("GET", "/tasks/v1/lists/%40default/tasks", nil)
	getReq.Header.Set("Authorization", "Bearer test-token")
	getW := httptest.NewRecorder()

	r.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("GET tasks got status %d, want %d. Body: %s", getW.Code, http.StatusOK, getW.Body.String())
	}

	var listResp map[string]interface{}
	if err := json.NewDecoder(getW.Body).Decode(&listResp); err != nil {
		t.Fatalf("failed to decode GET response: %v", err)
	}

	items, ok := listResp["items"].([]interface{})
	if !ok {
		t.Fatalf("GET response missing items array")
	}

	if len(items) == 0 {
		t.Fatalf("BUG: GET with %%40default returned empty when task was created with @default. URL encoding not normalized!")
	}

	// Verify the task is found
	found := false
	for _, item := range items {
		task := item.(map[string]interface{})
		if task["id"] == taskID {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("task %s not found when querying with %%40default", taskID)
	}
}

func TestCalendarPersistence(t *testing.T) {
	// Calendar should work - this test confirms our baseline
	p := setupTestPlugin(t)
	r := chi.NewRouter()
	r.Use(auth.Middleware)
	p.RegisterRoutes(r)

	// Step 1: Create an event via POST
	eventPayload := `{
		"summary": "Test Event from Persistence Test",
		"start": {"dateTime": "2025-01-15T10:00:00Z"},
		"end": {"dateTime": "2025-01-15T11:00:00Z"}
	}`
	postReq := httptest.NewRequest("POST", "/calendar/v3/calendars/primary/events", strings.NewReader(eventPayload))
	postReq.Header.Set("Content-Type", "application/json")
	postReq.Header.Set("Authorization", "Bearer test-token")
	postW := httptest.NewRecorder()

	r.ServeHTTP(postW, postReq)

	if postW.Code != http.StatusCreated {
		t.Fatalf("POST event got status %d, want %d. Body: %s", postW.Code, http.StatusCreated, postW.Body.String())
	}

	var createdEvent map[string]interface{}
	if err := json.NewDecoder(postW.Body).Decode(&createdEvent); err != nil {
		t.Fatalf("failed to decode POST response: %v", err)
	}

	eventID, ok := createdEvent["id"].(string)
	if !ok || eventID == "" {
		t.Fatalf("POST response missing event ID")
	}

	t.Logf("Created event with ID: %s", eventID)

	// Step 2: List events via GET and verify the created event is present
	getReq := httptest.NewRequest("GET", "/calendar/v3/calendars/primary/events", nil)
	getReq.Header.Set("Authorization", "Bearer test-token")
	getW := httptest.NewRecorder()

	r.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("GET events got status %d, want %d. Body: %s", getW.Code, http.StatusOK, getW.Body.String())
	}

	var listResp map[string]interface{}
	if err := json.NewDecoder(getW.Body).Decode(&listResp); err != nil {
		t.Fatalf("failed to decode GET response: %v", err)
	}

	items, ok := listResp["items"].([]interface{})
	if !ok {
		t.Fatalf("GET response missing items array")
	}

	if len(items) == 0 {
		t.Fatalf("GET returned empty items array - event was not persisted! Response: %s", getW.Body.String())
	}

	// Verify our event is in the list
	found := false
	for _, item := range items {
		event, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if event["id"] == eventID {
			found = true
			if event["summary"] != "Test Event from Persistence Test" {
				t.Errorf("event summary mismatch: got %v, want 'Test Event from Persistence Test'", event["summary"])
			}
			break
		}
	}

	if !found {
		t.Errorf("created event %s not found in GET response. Items: %v", eventID, items)
	}
}
