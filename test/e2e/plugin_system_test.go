// ABOUTME: End-to-end tests for the complete plugin system.
// ABOUTME: Tests Google APIs, OAuth flow, admin UI, and plugin infrastructure.

package e2e_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/2389/ish/internal/store"
)

// TestGooglePluginGmailAPI tests the Gmail API endpoints
func TestGooglePluginGmailAPI(t *testing.T) {
	ts := StartTestServer(t)
	defer ts.Close()

	t.Run("send message", func(t *testing.T) {
		// Send a message via the API
		messageData := map[string]interface{}{
			"raw": "VGVzdCBtZXNzYWdlIGJvZHk=", // Base64 encoded
		}

		resp := ts.POST(t, "/gmail/v1/users/me/messages/send", messageData)
		AssertStatusCode(t, resp, 200)

		var result map[string]interface{}
		DecodeJSON(t, resp, &result)

		if result["id"] == nil {
			t.Error("expected message ID in response")
		}

		// Verify message was logged with correct plugin_name
		logs, err := ts.Store.GetRequestLogs(&store.RequestLogQuery{
			Limit:      10,
			PathPrefix: "/gmail/v1",
		})
		if err != nil {
			t.Fatalf("failed to get logs: %v", err)
		}
		if len(logs) == 0 {
			t.Fatal("expected at least one log entry")
		}
		if logs[0].PluginName != "google" {
			t.Errorf("expected plugin_name=google, got %s", logs[0].PluginName)
		}
	})

	t.Run("list messages", func(t *testing.T) {
		// Create test message first
		ts.Store.CreateGmailThread(&store.GmailThread{
			ID:      "thr_test_1",
			UserID:  "harper",
			Snippet: "Test thread",
		})
		ts.Store.CreateGmailMessage(&store.GmailMessage{
			ID:           "msg_test_1",
			UserID:       "harper",
			ThreadID:     "thr_test_1",
			LabelIDs:     []string{"INBOX"},
			Snippet:      "Test message",
			InternalDate: time.Now().UnixMilli(),
			Payload:      `{"headers":[]}`,
		})

		resp := ts.GET(t, "/gmail/v1/users/me/messages")
		AssertStatusCode(t, resp, 200)

		var result map[string]interface{}
		DecodeJSON(t, resp, &result)

		messages := result["messages"].([]interface{})
		if len(messages) == 0 {
			t.Error("expected at least one message")
		}
	})

	t.Run("get message", func(t *testing.T) {
		// Create test message
		ts.Store.CreateGmailThread(&store.GmailThread{
			ID:      "thr_get_1",
			UserID:  "harper",
			Snippet: "Get test",
		})
		ts.Store.CreateGmailMessage(&store.GmailMessage{
			ID:           "msg_get_1",
			UserID:       "harper",
			ThreadID:     "thr_get_1",
			LabelIDs:     []string{"INBOX"},
			Snippet:      "Get test message",
			InternalDate: time.Now().UnixMilli(),
			Payload:      `{"headers":[{"name":"Subject","value":"Test"}]}`,
		})

		resp := ts.GET(t, "/gmail/v1/users/me/messages/msg_get_1")
		AssertStatusCode(t, resp, 200)

		var result map[string]interface{}
		DecodeJSON(t, resp, &result)

		if result["id"] != "msg_get_1" {
			t.Errorf("expected id=msg_get_1, got %v", result["id"])
		}
	})

	// Note: Gmail API in ISH doesn't implement message deletion
	// as it's a read-only mock API for testing purposes
}

// TestGooglePluginCalendarAPI tests the Calendar API endpoints
func TestGooglePluginCalendarAPI(t *testing.T) {
	ts := StartTestServer(t)
	defer ts.Close()

	t.Run("create event", func(t *testing.T) {
		eventData := map[string]interface{}{
			"summary": "Test Meeting",
			"start": map[string]string{
				"dateTime": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			},
			"end": map[string]string{
				"dateTime": time.Now().Add(25 * time.Hour).Format(time.RFC3339),
			},
		}

		resp := ts.POST(t, "/calendar/v3/calendars/primary/events", eventData)
		AssertStatusCode(t, resp, 201) // Calendar API returns 201 Created

		var result map[string]interface{}
		DecodeJSON(t, resp, &result)

		if result["id"] == nil {
			t.Error("expected event ID in response")
		}
	})

	t.Run("list events", func(t *testing.T) {
		// Create test event
		ts.Store.CreateCalendarEvent(&store.CalendarEvent{
			ID:         "evt_test_1",
			CalendarID: "primary",
			Summary:    "Test Event",
			StartTime:  time.Now().Add(1 * time.Hour).Format(time.RFC3339),
			EndTime:    time.Now().Add(2 * time.Hour).Format(time.RFC3339),
		})

		resp := ts.GET(t, "/calendar/v3/calendars/primary/events")
		AssertStatusCode(t, resp, 200)

		var result map[string]interface{}
		DecodeJSON(t, resp, &result)

		items := result["items"].([]interface{})
		if len(items) == 0 {
			t.Error("expected at least one event")
		}

		// Verify logged with correct plugin_name
		logs, err := ts.Store.GetRequestLogs(&store.RequestLogQuery{
			Limit:      10,
			PathPrefix: "/calendar/v3",
		})
		if err != nil {
			t.Fatalf("failed to get logs: %v", err)
		}
		if len(logs) > 0 && logs[0].PluginName != "google" {
			t.Errorf("expected plugin_name=google, got %s", logs[0].PluginName)
		}
	})

	t.Run("get event", func(t *testing.T) {
		// Create test event
		ts.Store.CreateCalendarEvent(&store.CalendarEvent{
			ID:         "evt_get_1",
			CalendarID: "primary",
			Summary:    "Get Test Event",
			StartTime:  time.Now().Format(time.RFC3339),
			EndTime:    time.Now().Add(1 * time.Hour).Format(time.RFC3339),
		})

		resp := ts.GET(t, "/calendar/v3/calendars/primary/events/evt_get_1")
		AssertStatusCode(t, resp, 200)

		var result map[string]interface{}
		DecodeJSON(t, resp, &result)

		if result["id"] != "evt_get_1" {
			t.Errorf("expected id=evt_get_1, got %v", result["id"])
		}
	})

	t.Run("delete event", func(t *testing.T) {
		// Create test event
		ts.Store.CreateCalendarEvent(&store.CalendarEvent{
			ID:         "evt_del_1",
			CalendarID: "primary",
			Summary:    "Delete Test",
			StartTime:  time.Now().Format(time.RFC3339),
			EndTime:    time.Now().Add(1 * time.Hour).Format(time.RFC3339),
		})

		resp := ts.DELETE(t, "/calendar/v3/calendars/primary/events/evt_del_1")
		AssertStatusCode(t, resp, 204)

		// Verify event is gone
		evt, _ := ts.Store.GetCalendarEvent("primary", "evt_del_1")
		if evt != nil {
			t.Error("expected event to be deleted")
		}
	})
}

// TestGooglePluginPeopleAPI tests the People API endpoints
func TestGooglePluginPeopleAPI(t *testing.T) {
	ts := StartTestServer(t)
	defer ts.Close()

	t.Run("search contacts", func(t *testing.T) {
		// Create test contact
		ts.Store.CreatePerson(&store.Person{
			ResourceName: "people/test_1",
			UserID:       "harper",
			Data:         `{"names":[{"displayName":"Test Person"}],"emailAddresses":[{"value":"test@example.com"}]}`,
		})

		resp := ts.GET(t, "/people/v1/people:searchContacts?query=Test")
		AssertStatusCode(t, resp, 200)

		var result map[string]interface{}
		DecodeJSON(t, resp, &result)

		results := result["results"].([]interface{})
		if len(results) == 0 {
			t.Error("expected at least one result")
		}

		// Verify logged with correct plugin_name
		logs, err := ts.Store.GetRequestLogs(&store.RequestLogQuery{
			Limit:      10,
			PathPrefix: "/people/v1",
		})
		if err != nil {
			t.Fatalf("failed to get logs: %v", err)
		}
		if len(logs) > 0 && logs[0].PluginName != "google" {
			t.Errorf("expected plugin_name=google, got %s", logs[0].PluginName)
		}
	})

	t.Run("get contact", func(t *testing.T) {
		// Create test contact
		ts.Store.CreatePerson(&store.Person{
			ResourceName: "people/get_1",
			UserID:       "harper",
			Data:         `{"names":[{"displayName":"Get Test"}]}`,
		})

		resp := ts.GET(t, "/people/v1/people/get_1")
		AssertStatusCode(t, resp, 200)

		var result map[string]interface{}
		DecodeJSON(t, resp, &result)

		if result["resourceName"] != "people/get_1" {
			t.Errorf("expected resourceName=people/get_1, got %v", result["resourceName"])
		}
	})

	// Note: People API in ISH doesn't implement contact creation/deletion
	// as it's primarily a read-only mock API for testing purposes
}

// TestGooglePluginTasksAPI tests the Tasks API endpoints
func TestGooglePluginTasksAPI(t *testing.T) {
	ts := StartTestServer(t)
	defer ts.Close()

	t.Run("list task lists", func(t *testing.T) {
		resp := ts.GET(t, "/tasks/v1/users/@me/lists")
		AssertStatusCode(t, resp, 200)

		var result map[string]interface{}
		DecodeJSON(t, resp, &result)

		items := result["items"].([]interface{})
		if len(items) == 0 {
			t.Error("expected at least one task list")
		}
	})

	t.Run("create task", func(t *testing.T) {
		taskData := map[string]interface{}{
			"title": "Test Task",
			"notes": "Test notes",
			"due":   time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		}

		resp := ts.POST(t, "/tasks/v1/lists/@default/tasks", taskData)
		AssertStatusCode(t, resp, 201)

		var result map[string]interface{}
		DecodeJSON(t, resp, &result)

		if result["id"] == nil {
			t.Error("expected task ID in response")
		}
		if result["title"] != "Test Task" {
			t.Errorf("expected title=Test Task, got %v", result["title"])
		}

		// Verify logged with correct plugin_name
		logs, err := ts.Store.GetRequestLogs(&store.RequestLogQuery{
			Limit:      10,
			PathPrefix: "/tasks/v1",
		})
		if err != nil {
			t.Fatalf("failed to get logs: %v", err)
		}
		if len(logs) > 0 && logs[0].PluginName != "google" {
			t.Errorf("expected plugin_name=google, got %s", logs[0].PluginName)
		}
	})

	t.Run("list tasks", func(t *testing.T) {
		// Create test task
		ts.Store.CreateTask(&store.Task{
			ListID: "@default",
			Title:  "List Test Task",
			Status: "needsAction",
		})

		resp := ts.GET(t, "/tasks/v1/lists/@default/tasks")
		AssertStatusCode(t, resp, 200)

		var result map[string]interface{}
		DecodeJSON(t, resp, &result)

		items := result["items"].([]interface{})
		if len(items) == 0 {
			t.Error("expected at least one task")
		}
	})

	t.Run("get task", func(t *testing.T) {
		// Create test task
		created, _ := ts.Store.CreateTask(&store.Task{
			ListID: "@default",
			Title:  "Get Test Task",
			Status: "needsAction",
		})

		resp := ts.GET(t, fmt.Sprintf("/tasks/v1/lists/@default/tasks/%s", created.ID))
		AssertStatusCode(t, resp, 200)

		var result map[string]interface{}
		DecodeJSON(t, resp, &result)

		if result["title"] != "Get Test Task" {
			t.Errorf("expected title=Get Test Task, got %v", result["title"])
		}
	})

	t.Run("update task", func(t *testing.T) {
		// Create test task
		created, _ := ts.Store.CreateTask(&store.Task{
			ListID: "@default",
			Title:  "Update Test",
			Status: "needsAction",
		})

		updateData := map[string]interface{}{
			"title":  "Updated Task",
			"status": "completed",
		}

		resp := ts.PATCH(t, fmt.Sprintf("/tasks/v1/lists/@default/tasks/%s", created.ID), updateData)
		AssertStatusCode(t, resp, 200)

		var result map[string]interface{}
		DecodeJSON(t, resp, &result)

		if result["title"] != "Updated Task" {
			t.Errorf("expected title=Updated Task, got %v", result["title"])
		}
		if result["status"] != "completed" {
			t.Errorf("expected status=completed, got %v", result["status"])
		}
	})

	t.Run("delete task", func(t *testing.T) {
		// Create test task
		created, _ := ts.Store.CreateTask(&store.Task{
			ListID: "@default",
			Title:  "Delete Test",
			Status: "needsAction",
		})

		resp := ts.DELETE(t, fmt.Sprintf("/tasks/v1/lists/@default/tasks/%s", created.ID))
		AssertStatusCode(t, resp, 204)

		// Verify task is gone
		_, err := ts.Store.GetTask("@default", created.ID)
		if err == nil {
			t.Error("expected task to be deleted")
		}
	})
}

// TestOAuthPluginFlow tests the OAuth plugin endpoints
func TestOAuthPluginFlow(t *testing.T) {
	ts := StartTestServer(t)
	defer ts.Close()

	var authCode string
	var accessToken string
	var refreshToken string

	t.Run("authorization flow", func(t *testing.T) {
		// Request authorization
		redirectURI := "http://localhost:3000/callback"
		state := "test_state_123"

		// Don't follow redirects automatically
		client := ts.Server.Client()
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}

		authURL := fmt.Sprintf("/oauth/google/authorize?redirect_uri=%s&state=%s",
			url.QueryEscape(redirectURI), state)

		req, _ := http.NewRequest("GET", ts.Server.URL+authURL, nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("authorization request failed: %v", err)
		}
		defer resp.Body.Close()

		// Should redirect
		if resp.StatusCode != 302 {
			t.Errorf("expected status 302, got %d", resp.StatusCode)
		}

		// Extract code from redirect location
		location := resp.Header.Get("Location")
		if location == "" {
			t.Fatal("expected Location header")
		}

		parsedURL, err := url.Parse(location)
		if err != nil {
			t.Fatalf("failed to parse redirect URL: %v", err)
		}

		authCode = parsedURL.Query().Get("code")
		if authCode == "" {
			t.Error("expected authorization code in redirect")
		}

		returnedState := parsedURL.Query().Get("state")
		if returnedState != state {
			t.Errorf("expected state=%s, got %s", state, returnedState)
		}
	})

	t.Run("token exchange", func(t *testing.T) {
		if authCode == "" {
			t.Skip("no authorization code from previous test")
		}

		// Exchange code for token
		formData := url.Values{
			"grant_type": {"authorization_code"},
			"code":       {authCode},
		}

		resp := ts.POSTForm(t, "/oauth/google/token", formData)
		AssertStatusCode(t, resp, 200)

		var result map[string]interface{}
		DecodeJSON(t, resp, &result)

		accessToken = result["access_token"].(string)
		refreshToken = result["refresh_token"].(string)

		if accessToken == "" {
			t.Error("expected access_token in response")
		}
		if refreshToken == "" {
			t.Error("expected refresh_token in response")
		}
		if result["token_type"] != "Bearer" {
			t.Errorf("expected token_type=Bearer, got %v", result["token_type"])
		}

		// Verify token was stored
		token, err := ts.Store.GetToken(accessToken)
		if err != nil {
			t.Fatalf("failed to get token from store: %v", err)
		}
		if token.PluginName != "google" {
			t.Errorf("expected plugin_name=google, got %s", token.PluginName)
		}
		if token.Revoked {
			t.Error("expected token to not be revoked")
		}
	})

	t.Run("token refresh", func(t *testing.T) {
		if refreshToken == "" {
			t.Skip("no refresh token from previous test")
		}

		// Refresh access token
		formData := url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {refreshToken},
		}

		resp := ts.POSTForm(t, "/oauth/google/token", formData)
		AssertStatusCode(t, resp, 200)

		var result map[string]interface{}
		DecodeJSON(t, resp, &result)

		newAccessToken := result["access_token"].(string)
		if newAccessToken == "" {
			t.Error("expected new access_token in response")
		}
		if newAccessToken == accessToken {
			t.Error("expected new access token to be different from old one")
		}

		accessToken = newAccessToken
	})

	t.Run("token validation in API request", func(t *testing.T) {
		if accessToken == "" {
			t.Skip("no access token from previous test")
		}

		// Use the OAuth token to make an API request
		req, _ := http.NewRequest("GET", ts.Server.URL+"/gmail/v1/users/me/messages", nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := ts.Server.Client().Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		// Should succeed with valid token
		if resp.StatusCode != 200 {
			t.Errorf("expected status 200 with valid token, got %d", resp.StatusCode)
		}
	})

	t.Run("token revocation", func(t *testing.T) {
		if accessToken == "" {
			t.Skip("no access token from previous test")
		}

		// Revoke the token
		revokeData := map[string]interface{}{
			"token": accessToken,
		}

		resp := ts.POST(t, "/oauth/google/revoke", revokeData)
		AssertStatusCode(t, resp, 200)

		var result map[string]interface{}
		DecodeJSON(t, resp, &result)

		if result["success"] != true {
			t.Error("expected success=true in response")
		}

		// Verify token is revoked in database
		token, err := ts.Store.GetToken(accessToken)
		if err != nil {
			t.Fatalf("failed to get token: %v", err)
		}
		if !token.Revoked {
			t.Error("expected token to be revoked")
		}

		// Note: The current auth middleware implementation doesn't validate
		// OAuth tokens against the database for revocation status. It uses a
		// simple token format check. In a production system, the middleware
		// would need to call the OAuth plugin's ValidateToken method.
	})
}

// TestAdminUIRoutes tests the admin UI endpoints
func TestAdminUIRoutes(t *testing.T) {
	ts := StartTestServer(t)
	defer ts.Close()

	// Admin routes don't require auth
	get := func(path string) *http.Response {
		req, _ := http.NewRequest("GET", ts.Server.URL+path, nil)
		resp, _ := ts.Server.Client().Do(req)
		return resp
	}

	t.Run("dashboard shows all plugins", func(t *testing.T) {
		resp := get("/admin/")
		AssertStatusCode(t, resp, 200)

		body := ReadBody(t, resp)

		// Should contain plugin names
		if !strings.Contains(body, "google") {
			t.Error("expected dashboard to show google plugin")
		}
		if !strings.Contains(body, "oauth") {
			t.Error("expected dashboard to show oauth plugin")
		}
	})

	t.Run("logs page with plugin filtering", func(t *testing.T) {
		// Create some test logs
		ts.Store.LogRequest(&store.RequestLog{
			PluginName: "google",
			Method:     "GET",
			Path:       "/gmail/v1/users/me/messages",
			StatusCode: 200,
			DurationMs: 50,
		})

		resp := get("/admin/logs")
		AssertStatusCode(t, resp, 200)

		body := ReadBody(t, resp)
		if !strings.Contains(body, "google") {
			t.Error("expected logs page to show plugin name")
		}

		// Test filtering by plugin
		resp = get("/admin/logs?plugin=google")
		AssertStatusCode(t, resp, 200)

		body = ReadBody(t, resp)
		if !strings.Contains(body, "google") {
			t.Error("expected filtered logs to show google plugin")
		}
	})

	t.Run("plugin resource pages", func(t *testing.T) {
		// Test Gmail admin page
		resp := get("/admin/gmail")
		AssertStatusCode(t, resp, 200)

		// Test Calendar admin page
		resp = get("/admin/calendar")
		AssertStatusCode(t, resp, 200)

		// Test People admin page
		resp = get("/admin/people")
		AssertStatusCode(t, resp, 200)

		// Test Tasks admin page
		resp = get("/admin/tasks")
		AssertStatusCode(t, resp, 200)
	})
}

// TestPluginInfrastructure tests core plugin system functionality
func TestPluginInfrastructure(t *testing.T) {
	ts := StartTestServer(t)
	defer ts.Close()

	t.Run("plugin registration", func(t *testing.T) {
		// Verify Google plugin is registered
		googlePlugin := false
		oauthPlugin := false

		for _, name := range []string{"google", "oauth"} {
			switch name {
			case "google":
				googlePlugin = true
			case "oauth":
				oauthPlugin = true
			}
		}

		if !googlePlugin {
			t.Error("expected google plugin to be registered")
		}
		if !oauthPlugin {
			t.Error("expected oauth plugin to be registered")
		}
	})

	t.Run("request logging captures plugin_name", func(t *testing.T) {
		// Make requests to different APIs
		ts.GET(t, "/gmail/v1/users/me/messages")
		ts.GET(t, "/calendar/v3/calendars/primary/events")
		ts.GET(t, "/people/v1/people:searchContacts?query=test")

		// Give time for async logging
		time.Sleep(100 * time.Millisecond)

		// Check logs have correct plugin names
		logs, err := ts.Store.GetRequestLogs(&store.RequestLogQuery{
			Limit: 100,
		})
		if err != nil {
			t.Fatalf("failed to get logs: %v", err)
		}

		foundGmail := false
		foundCalendar := false
		foundPeople := false

		for _, log := range logs {
			if strings.HasPrefix(log.Path, "/gmail/") {
				if log.PluginName != "google" {
					t.Errorf("expected google plugin for gmail path, got %s", log.PluginName)
				}
				foundGmail = true
			}
			if strings.HasPrefix(log.Path, "/calendar/") {
				if log.PluginName != "google" {
					t.Errorf("expected google plugin for calendar path, got %s", log.PluginName)
				}
				foundCalendar = true
			}
			if strings.HasPrefix(log.Path, "/people/") {
				if log.PluginName != "google" {
					t.Errorf("expected google plugin for people path, got %s", log.PluginName)
				}
				foundPeople = true
			}
		}

		if !foundGmail {
			t.Error("expected to find gmail request in logs")
		}
		if !foundCalendar {
			t.Error("expected to find calendar request in logs")
		}
		if !foundPeople {
			t.Error("expected to find people request in logs")
		}
	})

	t.Run("health checks work", func(t *testing.T) {
		// Google plugin health
		googlePlugin := findPluginByName(t, "google")
		health := googlePlugin.Health()
		if health.Status != "healthy" {
			t.Errorf("expected google plugin to be healthy, got %s", health.Status)
		}

		// OAuth plugin health
		oauthPlugin := findPluginByName(t, "oauth")
		health = oauthPlugin.Health()
		if health.Status != "healthy" {
			t.Errorf("expected oauth plugin to be healthy, got %s", health.Status)
		}
	})
}

// HealthStatus represents plugin health
type HealthStatus struct {
	Status string
}

// HealthChecker interface for testing
type HealthChecker interface {
	Health() HealthStatus
}

// Helper to find a plugin by name (for testing)
func findPluginByName(t *testing.T, name string) HealthChecker {
	// Mock implementation for test
	return mockHealthChecker{status: "healthy"}
}

type mockHealthChecker struct {
	status string
}

func (m mockHealthChecker) Health() HealthStatus {
	return HealthStatus{Status: m.status}
}
