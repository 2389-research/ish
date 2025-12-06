// ABOUTME: Test helpers for E2E testing.
// ABOUTME: Provides utilities for starting test server, making requests, and assertions.

package e2e_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/2389/ish/internal/admin"
	"github.com/2389/ish/internal/auth"
	"github.com/2389/ish/internal/logging"
	"github.com/2389/ish/internal/store"
	"github.com/2389/ish/plugins/core"
	_ "github.com/2389/ish/plugins/google" // Register Google plugin
	_ "github.com/2389/ish/plugins/oauth"  // Register OAuth plugin
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// TestServer wraps a test HTTP server with a store
type TestServer struct {
	Server *httptest.Server
	Store  *store.Store
	DBPath string
}

// StartTestServer creates and starts a test server with all plugins registered
func StartTestServer(t *testing.T) *TestServer {
	t.Helper()

	// Use unique DB path for each test
	dbPath := fmt.Sprintf("test_e2e_%d.db", time.Now().UnixNano())

	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	// Create default user
	if err := s.CreateUser("harper"); err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	// Create default task list
	if err := s.CreateTaskList(&store.TaskList{
		ID:     "@default",
		UserID: "harper",
		Title:  "My Tasks",
	}); err != nil {
		t.Fatalf("failed to create task list: %v", err)
	}

	// Create primary calendar
	if err := s.CreateCalendar(&store.Calendar{
		ID:      "primary",
		UserID:  "harper",
		Summary: "Primary Calendar",
	}); err != nil {
		t.Fatalf("failed to create calendar: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(logging.Middleware(s))
	r.Use(auth.Middleware)

	// Register all plugins
	for _, plugin := range core.All() {
		// Set database for plugins that need it
		if dbPlugin, ok := plugin.(core.DatabasePlugin); ok {
			if err := dbPlugin.SetDB(s.GetDB()); err != nil {
				t.Fatalf("failed to initialize plugin %s: %v", plugin.Name(), err)
			}
		}

		// Register routes
		plugin.RegisterAuth(r)
		plugin.RegisterRoutes(r)
	}

	// Register admin routes
	admin.NewHandlers(s).RegisterRoutes(r)

	srv := httptest.NewServer(r)

	return &TestServer{
		Server: srv,
		Store:  s,
		DBPath: dbPath,
	}
}

// Close shuts down the test server and cleans up
func (ts *TestServer) Close() {
	ts.Server.Close()
	ts.Store.Close()
	os.Remove(ts.DBPath)
}

// GET makes a GET request with authorization
func (ts *TestServer) GET(t *testing.T, path string) *http.Response {
	t.Helper()
	req, err := http.NewRequest("GET", ts.Server.URL+path, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer user:harper")

	resp, err := ts.Server.Client().Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

// POST makes a POST request with JSON body and authorization
func (ts *TestServer) POST(t *testing.T, path string, body interface{}) *http.Response {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal body: %v", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest("POST", ts.Server.URL+path, bodyReader)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer user:harper")
	req.Header.Set("Content-Type", "application/json")

	resp, err := ts.Server.Client().Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

// POSTForm makes a POST request with form data
func (ts *TestServer) POSTForm(t *testing.T, path string, data url.Values) *http.Response {
	t.Helper()
	req, err := http.NewRequest("POST", ts.Server.URL+path, bytes.NewReader([]byte(data.Encode())))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := ts.Server.Client().Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

// PUT makes a PUT request with JSON body and authorization
func (ts *TestServer) PUT(t *testing.T, path string, body interface{}) *http.Response {
	t.Helper()
	jsonBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	req, err := http.NewRequest("PUT", ts.Server.URL+path, bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer user:harper")
	req.Header.Set("Content-Type", "application/json")

	resp, err := ts.Server.Client().Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

// PATCH makes a PATCH request with JSON body and authorization
func (ts *TestServer) PATCH(t *testing.T, path string, body interface{}) *http.Response {
	t.Helper()
	jsonBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	req, err := http.NewRequest("PATCH", ts.Server.URL+path, bytes.NewReader(jsonBody))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer user:harper")
	req.Header.Set("Content-Type", "application/json")

	resp, err := ts.Server.Client().Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

// DELETE makes a DELETE request with authorization
func (ts *TestServer) DELETE(t *testing.T, path string) *http.Response {
	t.Helper()
	req, err := http.NewRequest("DELETE", ts.Server.URL+path, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer user:harper")

	resp, err := ts.Server.Client().Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	return resp
}

// AssertStatusCode checks if response has expected status code
func AssertStatusCode(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("expected status %d, got %d. Body: %s", expected, resp.StatusCode, string(body))
	}
}

// DecodeJSON decodes response body as JSON
func DecodeJSON(t *testing.T, resp *http.Response, v interface{}) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
}

// ReadBody reads and returns the response body
func ReadBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	return string(body)
}
