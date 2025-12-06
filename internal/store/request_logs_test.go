// ABOUTME: Tests for request log storage operations.
// ABOUTME: Tests plugin metrics calculations and request log queries.

package store

import (
	"testing"
	"time"
)

func TestGetPluginRequestCount(t *testing.T) {
	s := setupTestDB(t)
	defer s.Close()

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	twoDaysAgo := now.Add(-48 * time.Hour)

	// Insert test data - mix of plugins and times
	testLogs := []*RequestLog{
		{PluginName: "google", Method: "GET", Path: "/gmail/messages", StatusCode: 200, DurationMs: 10, Timestamp: now},
		{PluginName: "google", Method: "GET", Path: "/gmail/messages", StatusCode: 200, DurationMs: 15, Timestamp: yesterday.Add(1 * time.Hour)},
		{PluginName: "google", Method: "POST", Path: "/calendar/events", StatusCode: 201, DurationMs: 20, Timestamp: yesterday.Add(2 * time.Hour)},
		{PluginName: "google", Method: "GET", Path: "/people", StatusCode: 200, DurationMs: 5, Timestamp: twoDaysAgo},
		{PluginName: "tasks", Method: "GET", Path: "/tasks", StatusCode: 200, DurationMs: 8, Timestamp: yesterday.Add(3 * time.Hour)},
		{PluginName: "tasks", Method: "POST", Path: "/tasks", StatusCode: 201, DurationMs: 12, Timestamp: now.Add(-1 * time.Hour)},
	}

	for _, log := range testLogs {
		// Insert with specific timestamp
		_, err := s.db.Exec(`
			INSERT INTO request_logs (plugin_name, method, path, status_code, duration_ms, timestamp)
			VALUES (?, ?, ?, ?, ?, ?)
		`, log.PluginName, log.Method, log.Path, log.StatusCode, log.DurationMs, log.Timestamp)
		if err != nil {
			t.Fatalf("Failed to insert test log: %v", err)
		}
	}

	// Test counting requests in last 24 hours for "google"
	count, err := s.GetPluginRequestCount("google", yesterday)
	if err != nil {
		t.Fatalf("GetPluginRequestCount failed: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected 3 requests for google in last 24h, got %d", count)
	}

	// Test counting requests in last 24 hours for "tasks"
	count, err = s.GetPluginRequestCount("tasks", yesterday)
	if err != nil {
		t.Fatalf("GetPluginRequestCount failed: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 requests for tasks in last 24h, got %d", count)
	}

	// Test non-existent plugin
	count, err = s.GetPluginRequestCount("nonexistent", yesterday)
	if err != nil {
		t.Fatalf("GetPluginRequestCount failed: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 requests for nonexistent plugin, got %d", count)
	}
}

func TestGetPluginErrorRate(t *testing.T) {
	s := setupTestDB(t)
	defer s.Close()

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	// Insert test data with various status codes
	testLogs := []*RequestLog{
		{PluginName: "google", Method: "GET", Path: "/gmail/messages", StatusCode: 200, DurationMs: 10, Timestamp: yesterday.Add(1 * time.Hour)},
		{PluginName: "google", Method: "GET", Path: "/gmail/messages", StatusCode: 200, DurationMs: 15, Timestamp: yesterday.Add(2 * time.Hour)},
		{PluginName: "google", Method: "POST", Path: "/calendar/events", StatusCode: 400, DurationMs: 20, Timestamp: yesterday.Add(3 * time.Hour)},
		{PluginName: "google", Method: "GET", Path: "/people", StatusCode: 404, DurationMs: 5, Timestamp: yesterday.Add(4 * time.Hour)},
		{PluginName: "google", Method: "DELETE", Path: "/gmail/messages/1", StatusCode: 500, DurationMs: 30, Timestamp: yesterday.Add(5 * time.Hour)},
		{PluginName: "tasks", Method: "GET", Path: "/tasks", StatusCode: 200, DurationMs: 8, Timestamp: yesterday.Add(6 * time.Hour)},
		{PluginName: "tasks", Method: "POST", Path: "/tasks", StatusCode: 201, DurationMs: 12, Timestamp: yesterday.Add(7 * time.Hour)},
		{PluginName: "tasks", Method: "GET", Path: "/tasks/1", StatusCode: 500, DurationMs: 15, Timestamp: yesterday.Add(8 * time.Hour)},
	}

	for _, log := range testLogs {
		_, err := s.db.Exec(`
			INSERT INTO request_logs (plugin_name, method, path, status_code, duration_ms, timestamp)
			VALUES (?, ?, ?, ?, ?, ?)
		`, log.PluginName, log.Method, log.Path, log.StatusCode, log.DurationMs, log.Timestamp)
		if err != nil {
			t.Fatalf("Failed to insert test log: %v", err)
		}
	}

	// Test error rate for "google" (3 errors out of 5 requests = 60%)
	rate, err := s.GetPluginErrorRate("google", yesterday)
	if err != nil {
		t.Fatalf("GetPluginErrorRate failed: %v", err)
	}
	if rate != 60.0 {
		t.Errorf("Expected 60.0%% error rate for google, got %.2f%%", rate)
	}

	// Test error rate for "tasks" (1 error out of 3 requests = 33.33%)
	rate, err = s.GetPluginErrorRate("tasks", yesterday)
	if err != nil {
		t.Fatalf("GetPluginErrorRate failed: %v", err)
	}
	if rate < 33.0 || rate > 34.0 {
		t.Errorf("Expected ~33.33%% error rate for tasks, got %.2f%%", rate)
	}

	// Test plugin with no requests
	rate, err = s.GetPluginErrorRate("nonexistent", yesterday)
	if err != nil {
		t.Fatalf("GetPluginErrorRate failed: %v", err)
	}
	if rate != 0.0 {
		t.Errorf("Expected 0.0%% error rate for nonexistent plugin, got %.2f%%", rate)
	}
}

func TestGetRecentRequests(t *testing.T) {
	s := setupTestDB(t)
	defer s.Close()

	now := time.Now()

	// Insert test data in reverse chronological order (we want to test ordering)
	testLogs := []*RequestLog{
		{PluginName: "google", Method: "GET", Path: "/gmail/messages/5", StatusCode: 200, DurationMs: 10, Timestamp: now.Add(-1 * time.Minute)},
		{PluginName: "google", Method: "POST", Path: "/calendar/events", StatusCode: 201, DurationMs: 15, Timestamp: now.Add(-2 * time.Minute)},
		{PluginName: "google", Method: "GET", Path: "/people", StatusCode: 404, DurationMs: 20, Timestamp: now.Add(-3 * time.Minute)},
		{PluginName: "tasks", Method: "GET", Path: "/tasks", StatusCode: 200, DurationMs: 8, Timestamp: now.Add(-4 * time.Minute)},
		{PluginName: "google", Method: "DELETE", Path: "/gmail/messages/1", StatusCode: 500, DurationMs: 30, Timestamp: now.Add(-5 * time.Minute)},
		{PluginName: "google", Method: "GET", Path: "/gmail/messages/4", StatusCode: 200, DurationMs: 12, Timestamp: now.Add(-6 * time.Minute)},
		{PluginName: "google", Method: "GET", Path: "/gmail/messages/3", StatusCode: 200, DurationMs: 11, Timestamp: now.Add(-7 * time.Minute)},
	}

	for _, log := range testLogs {
		_, err := s.db.Exec(`
			INSERT INTO request_logs (plugin_name, method, path, status_code, duration_ms, timestamp)
			VALUES (?, ?, ?, ?, ?, ?)
		`, log.PluginName, log.Method, log.Path, log.StatusCode, log.DurationMs, log.Timestamp)
		if err != nil {
			t.Fatalf("Failed to insert test log: %v", err)
		}
	}

	// Test getting recent requests for "google"
	logs, err := s.GetRecentRequests("google", 3)
	if err != nil {
		t.Fatalf("GetRecentRequests failed: %v", err)
	}
	if len(logs) != 3 {
		t.Errorf("Expected 3 recent requests, got %d", len(logs))
	}

	// Verify order (most recent first)
	if len(logs) > 0 && logs[0].Path != "/gmail/messages/5" {
		t.Errorf("Expected most recent request to be /gmail/messages/5, got %s", logs[0].Path)
	}

	// Test limiting to 5 requests
	logs, err = s.GetRecentRequests("google", 5)
	if err != nil {
		t.Fatalf("GetRecentRequests failed: %v", err)
	}
	if len(logs) != 5 {
		t.Errorf("Expected 5 recent requests, got %d", len(logs))
	}

	// Test plugin with no requests
	logs, err = s.GetRecentRequests("nonexistent", 5)
	if err != nil {
		t.Fatalf("GetRecentRequests failed: %v", err)
	}
	if len(logs) != 0 {
		t.Errorf("Expected 0 requests for nonexistent plugin, got %d", len(logs))
	}
}

// Helper to setup a test database
func setupTestDB(t *testing.T) *Store {
	s, err := New(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	return s
}
