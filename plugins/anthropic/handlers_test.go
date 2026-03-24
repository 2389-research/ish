// ABOUTME: HTTP handler tests for Anthropic plugin scenario CRUD endpoints
// ABOUTME: Tests create, list, delete scenarios and plugin metadata methods

package anthropic

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestPlugin(t *testing.T) *AnthropicPlugin {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	plugin := &AnthropicPlugin{}
	if err := plugin.SetDB(db); err != nil {
		t.Fatalf("Failed to initialize plugin: %v", err)
	}
	return plugin
}

func newRequestWithParams(method, url string, body []byte, params map[string]string) *http.Request {
	req := httptest.NewRequest(method, url, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	return req.WithContext(ctx)
}

func TestCreateScenarioHandler(t *testing.T) {
	plugin := setupTestPlugin(t)

	body := map[string]interface{}{
		"pattern":       "hello",
		"response_type": "text",
		"response_text": "Hi there!",
		"priority":      10,
	}
	bodyJSON, _ := json.Marshal(body)

	req := newRequestWithParams("POST", "/v1/scenarios", bodyJSON, nil)
	w := httptest.NewRecorder()
	plugin.createScenario(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var response Scenario
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if response.ID == "" {
		t.Fatal("Scenario ID should be generated")
	}
	if response.Pattern != "hello" {
		t.Fatalf("Expected pattern 'hello', got '%s'", response.Pattern)
	}
	if response.ResponseType != "text" {
		t.Fatalf("Expected response_type 'text', got '%s'", response.ResponseType)
	}
	if response.ResponseText != "Hi there!" {
		t.Fatalf("Expected response_text 'Hi there!', got '%s'", response.ResponseText)
	}
	if response.Priority != 10 {
		t.Fatalf("Expected priority 10, got %d", response.Priority)
	}
}

func TestListScenariosHandler(t *testing.T) {
	plugin := setupTestPlugin(t)

	// Seed 2 scenarios directly via store
	plugin.store.CreateScenario(&Scenario{
		Pattern:      "hello",
		ResponseType: "text",
		ResponseText: "Hi!",
		Priority:     10,
	})
	plugin.store.CreateScenario(&Scenario{
		Pattern:      "",
		ResponseType: "text",
		ResponseText: "Default",
		Priority:     0,
	})

	req := newRequestWithParams("GET", "/v1/scenarios", nil, nil)
	w := httptest.NewRecorder()
	plugin.listScenarios(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var scenarios []*Scenario
	if err := json.NewDecoder(w.Body).Decode(&scenarios); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(scenarios) != 2 {
		t.Fatalf("Expected 2 scenarios, got %d", len(scenarios))
	}
}

func TestDeleteScenarioHandler(t *testing.T) {
	plugin := setupTestPlugin(t)

	// Create a scenario
	scenario := &Scenario{
		Pattern:      "to-delete",
		ResponseType: "text",
		ResponseText: "gone",
		Priority:     5,
	}
	plugin.store.CreateScenario(scenario)

	req := newRequestWithParams("DELETE", "/v1/scenarios/"+scenario.ID, nil, map[string]string{
		"id": scenario.ID,
	})
	w := httptest.NewRecorder()
	plugin.deleteScenarioHandler(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("Expected 204, got %d: %s", w.Code, w.Body.String())
	}

	// Verify it's gone
	_, err := plugin.store.GetScenario(scenario.ID)
	if err != sql.ErrNoRows {
		t.Fatal("Expected scenario to be deleted")
	}
}

func TestPluginName(t *testing.T) {
	plugin := &AnthropicPlugin{}
	if plugin.Name() != "anthropic" {
		t.Fatalf("Expected name 'anthropic', got '%s'", plugin.Name())
	}
}

func TestPluginHealth(t *testing.T) {
	plugin := &AnthropicPlugin{}
	health := plugin.Health()
	if health.Status != "healthy" {
		t.Fatalf("Expected status 'healthy', got '%s'", health.Status)
	}
}
