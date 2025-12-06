// ABOUTME: People API handlers for Google plugin.
// ABOUTME: Implements People v1 API endpoints.

package google

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/2389/ish/internal/auth"
	"github.com/go-chi/chi/v5"
)

func (p *GooglePlugin) registerPeopleRoutes(r chi.Router) {
	r.Get("/v1/people/me/connections", p.listConnections)
	r.Get("/v1/people/{resourceId}", p.getPerson)
	r.Get("/v1/people:searchContacts", p.searchContacts)
	r.Get("/people/v1/people:searchContacts", p.searchContacts)
	r.Get("/people/v1/people/{resourceId}", p.getPerson)
}

func (p *GooglePlugin) listConnections(w http.ResponseWriter, r *http.Request) {
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
	people, nextPageToken, nextSyncToken, err := p.store.ListPeopleConnections(userID, pageSize, pageToken, syncToken)
	if err != nil {
		writeError(w, 500, "Internal error", "INTERNAL")
		return
	}

	// Convert to connections format
	connections := make([]map[string]any, len(people))
	for i, person := range people {
		var data map[string]any
		json.Unmarshal([]byte(person.Data), &data)

		personMap := map[string]any{
			"resourceName": person.ResourceName,
		}
		// Merge data fields
		for k, v := range data {
			personMap[k] = v
		}
		connections[i] = personMap
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

func (p *GooglePlugin) searchContacts(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromContext(r.Context())
	query := r.URL.Query().Get("query")

	pageSize := 10
	if ps := r.URL.Query().Get("pageSize"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 {
			pageSize = v
		}
	}
	pageToken := r.URL.Query().Get("pageToken")

	people, nextToken, err := p.store.SearchPeople(userID, query, pageSize, pageToken)
	if err != nil {
		writeError(w, 500, "Internal error", "INTERNAL")
		return
	}

	// Convert to response format
	results := make([]map[string]any, len(people))
	for i, person := range people {
		var data map[string]any
		json.Unmarshal([]byte(person.Data), &data)

		personMap := map[string]any{
			"resourceName": person.ResourceName,
		}
		// Merge data fields
		for k, v := range data {
			personMap[k] = v
		}

		results[i] = map[string]any{
			"person": personMap,
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

func (p *GooglePlugin) getPerson(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromContext(r.Context())
	resourceID := chi.URLParam(r, "resourceId")
	resourceName := "people/" + resourceID

	// Handle full resource name format
	if strings.HasPrefix(resourceID, "people/") {
		resourceName = resourceID
	}

	person, err := p.store.GetPerson(userID, resourceName)
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
