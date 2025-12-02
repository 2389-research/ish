// ABOUTME: Tests for People API HTTP handlers.
// ABOUTME: Verifies contact search and retrieval endpoints.

package people

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/2389/ish/internal/auth"
	"github.com/2389/ish/internal/store"
)

func TestHandlers_SearchContacts(t *testing.T) {
	dbPath := "test_people_handlers.db"
	defer os.Remove(dbPath)

	s, _ := store.New(dbPath)
	defer s.Close()

	// Seed data
	s.CreateUser("harper")
	s.CreatePerson(&store.Person{
		ResourceName: "people/c123",
		UserID:       "harper",
		Data:         `{"names":[{"displayName":"Alice Smith"}],"emailAddresses":[{"value":"alice@example.com"}]}`,
	})

	h := NewHandlers(s)
	r := chi.NewRouter()
	r.Use(auth.Middleware)
	h.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/people/v1/people:searchContacts?query=alice", nil)
	req.Header.Set("Authorization", "Bearer user:harper")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)
	results, ok := resp["results"].([]any)
	if !ok || len(results) != 1 {
		t.Errorf("results count = %d, want 1", len(results))
	}
}

func TestHandlers_GetPerson(t *testing.T) {
	dbPath := "test_people_handlers2.db"
	defer os.Remove(dbPath)

	s, _ := store.New(dbPath)
	defer s.Close()

	// Seed data
	s.CreateUser("harper")
	s.CreatePerson(&store.Person{
		ResourceName: "people/c123",
		UserID:       "harper",
		Data:         `{"names":[{"displayName":"Alice Smith"}],"emailAddresses":[{"value":"alice@example.com"}]}`,
	})

	h := NewHandlers(s)
	r := chi.NewRouter()
	r.Use(auth.Middleware)
	h.RegisterRoutes(r)

	req := httptest.NewRequest("GET", "/people/v1/people/c123", nil)
	req.Header.Set("Authorization", "Bearer user:harper")
	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["resourceName"] != "people/c123" {
		t.Errorf("resourceName = %v, want people/c123", resp["resourceName"])
	}
}
