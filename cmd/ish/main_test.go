// ABOUTME: Tests for CLI commands and server wiring.
// ABOUTME: Verifies health check and basic server functionality.

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestServer_Healthz(t *testing.T) {
	dbPath := "test_main.db"
	defer os.Remove(dbPath)

	srv, err := newServer(dbPath)
	if err != nil {
		t.Fatalf("newServer() error = %v", err)
	}

	req := httptest.NewRequest("GET", "/healthz", nil)
	rr := httptest.NewRecorder()

	srv.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["ok"] != true {
		t.Errorf("ok = %v, want true", resp["ok"])
	}
}
