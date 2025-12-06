// ABOUTME: Tests for admin HTTP handlers.
// ABOUTME: Verifies dashboard and CRUD pages render correctly.

package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/2389/ish/internal/store"
	"github.com/2389/ish/plugins/core"
	"github.com/go-chi/chi/v5"
)

// Mock plugin for dashboard testing
type dashboardMockPlugin struct {
	name   string
	health core.HealthStatus
	schema core.PluginSchema
}

func (m *dashboardMockPlugin) Name() string                                             { return m.name }
func (m *dashboardMockPlugin) Health() core.HealthStatus                                { return m.health }
func (m *dashboardMockPlugin) RegisterRoutes(r chi.Router)                              {}
func (m *dashboardMockPlugin) RegisterAuth(r chi.Router)                                {}
func (m *dashboardMockPlugin) Schema() core.PluginSchema                                { return m.schema }
func (m *dashboardMockPlugin) Seed(ctx context.Context, size string) (core.SeedData, error) {
	return core.SeedData{}, nil
}
func (m *dashboardMockPlugin) ValidateToken(token string) bool { return true }

var dashboardPluginsRegistered = false

func setupDashboardPlugins() {
	if dashboardPluginsRegistered {
		return
	}

	// Register test plugins with different health statuses
	core.Register(&dashboardMockPlugin{
		name: "google",
		health: core.HealthStatus{
			Status:  "healthy",
			Message: "All systems operational",
		},
		schema: core.PluginSchema{
			Resources: []core.ResourceSchema{
				{Name: "Messages", Slug: "messages"},
				{Name: "Events", Slug: "events"},
				{Name: "People", Slug: "people"},
			},
		},
	})

	core.Register(&dashboardMockPlugin{
		name: "tasks",
		health: core.HealthStatus{
			Status:  "degraded",
			Message: "Slow response times",
		},
		schema: core.PluginSchema{
			Resources: []core.ResourceSchema{
				{Name: "Tasks", Slug: "tasks"},
			},
		},
	})

	core.Register(&dashboardMockPlugin{
		name: "calendar",
		health: core.HealthStatus{
			Status:  "unavailable",
			Message: "Service down",
		},
		schema: core.PluginSchema{
			Resources: []core.ResourceSchema{
				{Name: "Calendars", Slug: "calendars"},
			},
		},
	})

	dashboardPluginsRegistered = true
}

func TestDashboardWithPlugins(t *testing.T) {
	setupDashboardPlugins()

	// Create test database
	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}
	defer s.Close()

	// Insert test request logs for plugins
	yesterday := time.Now().Add(-24 * time.Hour)
	testLogs := []struct {
		plugin string
		status int
	}{
		{"google", 200},
		{"google", 200},
		{"google", 404},
		{"google", 500},
		{"google", 200},
		{"tasks", 200},
		{"tasks", 500},
		{"calendar", 503},
	}

	for _, log := range testLogs {
		err := s.LogRequest(&store.RequestLog{
			PluginName: log.plugin,
			Method:     "GET",
			Path:       "/test",
			StatusCode: log.status,
			DurationMs: 10,
			Timestamp:  yesterday.Add(1 * time.Hour),
		})
		if err != nil {
			t.Fatalf("Failed to insert test log: %v", err)
		}
	}

	// Create handler
	h := NewHandlers(s)

	// Make request to dashboard
	req := httptest.NewRequest("GET", "/admin/", nil)
	w := httptest.NewRecorder()

	h.dashboard(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()

	// Verify plugin names appear
	if !strings.Contains(body, "google") {
		t.Error("Expected dashboard to contain google plugin")
	}
	if !strings.Contains(body, "tasks") {
		t.Error("Expected dashboard to contain tasks plugin")
	}
	if !strings.Contains(body, "calendar") {
		t.Error("Expected dashboard to contain calendar plugin")
	}

	// Verify health statuses appear (capitalized in the UI)
	if !strings.Contains(body, "Healthy") {
		t.Error("Expected dashboard to show Healthy status")
	}
	if !strings.Contains(body, "Degraded") {
		t.Error("Expected dashboard to show Degraded status")
	}
	if !strings.Contains(body, "Unavailable") {
		t.Error("Expected dashboard to show Unavailable status")
	}

	// Verify request counts appear (google should have 5 requests)
	if !strings.Contains(body, "5") {
		t.Error("Expected dashboard to show request count")
	}

	// Verify error rates appear (google has 2 errors / 5 requests = 40%)
	if !strings.Contains(body, "40") {
		t.Error("Expected dashboard to show error rate")
	}

	// Verify resource links appear
	if !strings.Contains(body, "/admin/plugins/google/messages") {
		t.Error("Expected dashboard to contain link to google messages")
	}
	if !strings.Contains(body, "/admin/plugins/tasks/tasks") {
		t.Error("Expected dashboard to contain link to tasks")
	}
}

func TestDashboardPluginHealthColors(t *testing.T) {
	setupDashboardPlugins()

	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}
	defer s.Close()

	h := NewHandlers(s)

	req := httptest.NewRequest("GET", "/admin/", nil)
	w := httptest.NewRecorder()

	h.dashboard(w, req)

	body := w.Body.String()

	// Verify color coding classes are present
	// Healthy = green
	if !strings.Contains(body, "bg-green") || !strings.Contains(body, "text-green") {
		t.Error("Expected dashboard to use green color for healthy status")
	}

	// Degraded = yellow
	if !strings.Contains(body, "bg-yellow") || !strings.Contains(body, "text-yellow") {
		t.Error("Expected dashboard to use yellow color for degraded status")
	}

	// Unavailable = red
	if !strings.Contains(body, "bg-red") || !strings.Contains(body, "text-red") {
		t.Error("Expected dashboard to use red color for unavailable status")
	}
}

func TestDashboardRecentRequests(t *testing.T) {
	setupDashboardPlugins()

	s, err := store.New(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}
	defer s.Close()

	// Insert test request logs with specific paths
	now := time.Now()
	testLogs := []struct {
		plugin string
		method string
		path   string
		status int
		when   time.Time
	}{
		{"google", "GET", "/gmail/messages", 200, now.Add(-1 * time.Minute)},
		{"google", "POST", "/calendar/events", 201, now.Add(-2 * time.Minute)},
		{"google", "GET", "/people/1", 404, now.Add(-3 * time.Minute)},
		{"google", "DELETE", "/gmail/messages/1", 500, now.Add(-4 * time.Minute)},
		{"google", "GET", "/gmail/threads", 200, now.Add(-5 * time.Minute)},
		{"google", "PUT", "/calendar/events/1", 200, now.Add(-6 * time.Minute)},
	}

	for _, log := range testLogs {
		err := s.LogRequest(&store.RequestLog{
			PluginName: log.plugin,
			Method:     log.method,
			Path:       log.path,
			StatusCode: log.status,
			DurationMs: 10,
			Timestamp:  log.when,
		})
		if err != nil {
			t.Fatalf("Failed to insert test log: %v", err)
		}
	}

	h := NewHandlers(s)
	req := httptest.NewRequest("GET", "/admin/", nil)
	w := httptest.NewRecorder()

	h.dashboard(w, req)

	body := w.Body.String()

	// Verify recent request paths appear (should show top 5)
	if !strings.Contains(body, "/gmail/messages") {
		t.Error("Expected dashboard to show recent request path")
	}
	if !strings.Contains(body, "POST") || !strings.Contains(body, "GET") || !strings.Contains(body, "DELETE") {
		t.Error("Expected dashboard to show HTTP methods")
	}

	// Verify status codes appear
	if !strings.Contains(body, "200") {
		t.Error("Expected dashboard to show status code 200")
	}
	if !strings.Contains(body, "404") {
		t.Error("Expected dashboard to show status code 404")
	}
}
