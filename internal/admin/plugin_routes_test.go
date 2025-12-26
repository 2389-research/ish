// ABOUTME: Tests for plugin admin routes that wire schema renderer to HTTP handlers.
// ABOUTME: Verifies list, create, detail, edit views render properly from plugin schemas.

package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/2389/ish/plugins/core"
	"github.com/go-chi/chi/v5"
)

// Mock plugin for testing
type mockPlugin struct {
	name   string
	schema core.PluginSchema
}

func (m *mockPlugin) Name() string                                             { return m.name }
func (m *mockPlugin) Health() core.HealthStatus                                { return core.HealthStatus{Status: "healthy"} }
func (m *mockPlugin) RegisterRoutes(r chi.Router)                              {}
func (m *mockPlugin) RegisterAuth(r chi.Router)                                {}
func (m *mockPlugin) Schema() core.PluginSchema                                { return m.schema }
func (m *mockPlugin) Seed(ctx context.Context, size string) (core.SeedData, error) {
	return core.SeedData{}, nil
}
func (m *mockPlugin) ValidateToken(token string) bool { return true }

// DataProvider implementation for JSON endpoint tests
func (m *mockPlugin) ListResources(ctx context.Context, slug string, opts core.ListOptions) ([]map[string]interface{}, error) {
	return []map[string]interface{}{
		{"id": "1", "subject": "Test Message 1", "from": "test1@example.com"},
		{"id": "2", "subject": "Test Message 2", "from": "test2@example.com"},
	}, nil
}

func (m *mockPlugin) GetResource(ctx context.Context, slug, id string) (map[string]interface{}, error) {
	return map[string]interface{}{
		"id":      id,
		"subject": "Test Message",
		"from":    "test@example.com",
		"body":    "Test body content",
	}, nil
}

var testPluginRegistered = false

func setupTestPlugin() {
	if testPluginRegistered {
		return
	}

	// Register a test plugin
	testSchema := core.PluginSchema{
		Resources: []core.ResourceSchema{
			{
				Name:        "Messages",
				Slug:        "messages",
				ListColumns: []string{"subject", "from"},
				Fields: []core.FieldSchema{
					{Name: "id", Type: "string", Display: "ID", Required: false, Editable: false},
					{Name: "subject", Type: "string", Display: "Subject", Required: true, Editable: true},
					{Name: "from", Type: "email", Display: "From", Required: true, Editable: true},
					{Name: "body", Type: "text", Display: "Body", Required: false, Editable: true},
				},
				Actions: []core.ActionSchema{
					{Name: "delete", HTTPMethod: "DELETE", Endpoint: "/api/messages/{id}", Confirm: true},
				},
			},
		},
	}

	core.Register(&mockPlugin{
		name:   "testplugin",
		schema: testSchema,
	})
	testPluginRegistered = true
}

func TestPluginListView(t *testing.T) {
	setupTestPlugin()

	h := &PluginHandlers{}
	req := httptest.NewRequest("GET", "/admin/plugins/testplugin/messages", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("plugin", "testplugin")
	rctx.URLParams.Add("resource", "messages")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.PluginListView(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Subject") {
		t.Error("Expected list view to contain 'Subject' column header")
	}
	if !strings.Contains(body, "From") {
		t.Error("Expected list view to contain 'From' column header")
	}
}

func TestPluginCreateForm(t *testing.T) {
	setupTestPlugin()

	h := &PluginHandlers{}
	req := httptest.NewRequest("GET", "/admin/plugins/testplugin/messages/new", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("plugin", "testplugin")
	rctx.URLParams.Add("resource", "messages")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.PluginCreateForm(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Subject") {
		t.Error("Expected form to contain 'Subject' field")
	}
	if !strings.Contains(body, "From") {
		t.Error("Expected form to contain 'From' field")
	}
	if !strings.Contains(body, "<form") {
		t.Error("Expected form element")
	}
}

func TestPluginDetailView(t *testing.T) {
	setupTestPlugin()

	h := &PluginHandlers{}
	req := httptest.NewRequest("GET", "/admin/plugins/testplugin/messages/123", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("plugin", "testplugin")
	rctx.URLParams.Add("resource", "messages")
	rctx.URLParams.Add("id", "123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.PluginDetailView(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "Subject") {
		t.Error("Expected detail view to contain 'Subject' field label")
	}
}

func TestPluginEditForm(t *testing.T) {
	setupTestPlugin()

	h := &PluginHandlers{}
	req := httptest.NewRequest("GET", "/admin/plugins/testplugin/messages/123/edit", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("plugin", "testplugin")
	rctx.URLParams.Add("resource", "messages")
	rctx.URLParams.Add("id", "123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.PluginEditForm(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "<form") {
		t.Error("Expected form element")
	}
}

func TestPluginNotFound(t *testing.T) {
	h := &PluginHandlers{}
	req := httptest.NewRequest("GET", "/admin/plugins/nonexistent/messages", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("plugin", "nonexistent")
	rctx.URLParams.Add("resource", "messages")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.PluginListView(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for nonexistent plugin, got %d", w.Code)
	}
}

func TestResourceNotFound(t *testing.T) {
	setupTestPlugin()

	h := &PluginHandlers{}
	req := httptest.NewRequest("GET", "/admin/plugins/testplugin/nonexistent", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("plugin", "testplugin")
	rctx.URLParams.Add("resource", "nonexistent")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.PluginListView(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for nonexistent resource, got %d", w.Code)
	}
}

func TestURLParameterParsing(t *testing.T) {
	setupTestPlugin()

	h := &PluginHandlers{}
	req := httptest.NewRequest("GET", "/admin/plugins/testplugin/messages", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("plugin", "testplugin")
	rctx.URLParams.Add("resource", "messages")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.PluginListView(w, req)

	if w.Code != http.StatusOK {
		t.Error("Failed to parse URL parameters correctly")
	}
}

// JSON endpoint tests - for agents to verify integration worked

func TestPluginListJSON(t *testing.T) {
	setupTestPlugin()

	h := &PluginHandlers{}
	req := httptest.NewRequest("GET", "/admin/plugins/testplugin/messages.json", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("plugin", "testplugin")
	rctx.URLParams.Add("resource", "messages")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.PluginListJSON(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, `"plugin"`) {
		t.Error("Expected JSON response to contain 'plugin' key")
	}
	if !strings.Contains(body, `"resource"`) {
		t.Error("Expected JSON response to contain 'resource' key")
	}
	if !strings.Contains(body, `"data"`) {
		t.Error("Expected JSON response to contain 'data' key")
	}
}

func TestPluginDetailJSON(t *testing.T) {
	setupTestPlugin()

	h := &PluginHandlers{}
	req := httptest.NewRequest("GET", "/admin/plugins/testplugin/messages/123.json", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("plugin", "testplugin")
	rctx.URLParams.Add("resource", "messages")
	rctx.URLParams.Add("id", "123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.PluginDetailJSON(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, `"plugin"`) {
		t.Error("Expected JSON response to contain 'plugin' key")
	}
	if !strings.Contains(body, `"resource"`) {
		t.Error("Expected JSON response to contain 'resource' key")
	}
	if !strings.Contains(body, `"data"`) {
		t.Error("Expected JSON response to contain 'data' key")
	}
	if !strings.Contains(body, `"id"`) {
		t.Error("Expected JSON response to contain 'id' key")
	}
}

func TestPluginListJSONNotFound(t *testing.T) {
	h := &PluginHandlers{}
	req := httptest.NewRequest("GET", "/admin/plugins/nonexistent/messages.json", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("plugin", "nonexistent")
	rctx.URLParams.Add("resource", "messages")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.PluginListJSON(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for nonexistent plugin, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json even for errors, got %s", contentType)
	}
}

func TestPluginDetailJSONNotFound(t *testing.T) {
	h := &PluginHandlers{}
	req := httptest.NewRequest("GET", "/admin/plugins/nonexistent/messages/123.json", nil)
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("plugin", "nonexistent")
	rctx.URLParams.Add("resource", "messages")
	rctx.URLParams.Add("id", "123")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.PluginDetailJSON(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for nonexistent plugin, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json even for errors, got %s", contentType)
	}
}
