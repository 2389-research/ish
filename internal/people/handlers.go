// ABOUTME: HTTP handlers for People API endpoints.
// ABOUTME: Implements contact search and retrieval with Google-compatible responses.

package people

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/2389/ish/internal/auth"
	"github.com/2389/ish/internal/store"
)

type Handlers struct {
	store *store.Store
}

func NewHandlers(s *store.Store) *Handlers {
	return &Handlers{store: s}
}

func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Get("/v1/people/me/connections", h.listConnections)
	r.Get("/v1/people/{resourceId}", h.getPerson)
	r.Get("/v1/people:searchContacts", h.searchContacts) // Alias without /people/v1/ prefix
	r.Get("/people/v1/people:searchContacts", h.searchContacts)
	r.Get("/people/v1/people/{resourceId}", h.getPerson)
}

func (h *Handlers) listConnections(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromContext(r.Context())

	pageSize := 100
	if ps := r.URL.Query().Get("pageSize"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 {
			pageSize = v
		}
	}
	pageToken := r.URL.Query().Get("pageToken")
	syncToken := r.URL.Query().Get("syncToken")

	// Use the new connections endpoint with sync token support
	people, nextPageToken, nextSyncToken, err := h.store.ListPeopleConnections(userID, pageSize, pageToken, syncToken)
	if err != nil {
		writeError(w, 500, "Internal error", "INTERNAL")
		return
	}

	// Convert to connections format
	connections := make([]map[string]any, len(people))
	for i, p := range people {
		var data map[string]any
		json.Unmarshal([]byte(p.Data), &data)

		person := map[string]any{
			"resourceName": p.ResourceName,
		}
		// Merge data fields
		for k, v := range data {
			person[k] = v
		}
		connections[i] = person
	}

	resp := map[string]any{
		"connections": connections,
		"totalPeople": len(connections),
		"totalItems":  len(connections),
	}
	if nextPageToken != "" {
		resp["nextPageToken"] = nextPageToken
	}
	if nextSyncToken != "" {
		resp["nextSyncToken"] = nextSyncToken
	}

	writeJSON(w, resp)
}

func (h *Handlers) searchContacts(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromContext(r.Context())
	query := r.URL.Query().Get("query")

	pageSize := 10
	if ps := r.URL.Query().Get("pageSize"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 {
			pageSize = v
		}
	}
	pageToken := r.URL.Query().Get("pageToken")

	people, nextToken, err := h.store.SearchPeople(userID, query, pageSize, pageToken)
	if err != nil {
		writeError(w, 500, "Internal error", "INTERNAL")
		return
	}

	// Convert to response format
	results := make([]map[string]any, len(people))
	for i, p := range people {
		var data map[string]any
		json.Unmarshal([]byte(p.Data), &data)

		person := map[string]any{
			"resourceName": p.ResourceName,
		}
		// Merge data fields
		for k, v := range data {
			person[k] = v
		}

		results[i] = map[string]any{
			"person": person,
		}
	}

	resp := map[string]any{
		"results": results,
	}
	if nextToken != "" {
		resp["nextPageToken"] = nextToken
	}

	writeJSON(w, resp)
}

func (h *Handlers) getPerson(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromContext(r.Context())
	resourceID := chi.URLParam(r, "resourceId")
	resourceName := "people/" + resourceID

	// Handle full resource name format
	if strings.HasPrefix(resourceID, "people/") {
		resourceName = resourceID
	}

	person, err := h.store.GetPerson(userID, resourceName)
	if err != nil {
		writeError(w, 404, "Person not found", "NOT_FOUND")
		return
	}

	var data map[string]any
	json.Unmarshal([]byte(person.Data), &data)

	resp := map[string]any{
		"resourceName": person.ResourceName,
	}
	// Merge data fields
	for k, v := range data {
		resp[k] = v
	}

	writeJSON(w, resp)
}

func writeJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, code int, message, status string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
			"status":  status,
		},
	})
}
