// ABOUTME: People API handlers for Google plugin.
// ABOUTME: Implements People v1 API endpoints.

package google

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/2389/ish/internal/auth"
	"github.com/go-chi/chi/v5"
)

func (p *GooglePlugin) registerPeopleRoutes(r chi.Router) {
	// Register People API v1 routes under both /v1 and /people/v1 prefixes
	// for compatibility with different Google API client configurations
	registerPeopleV1Routes := func(r chi.Router) {
		r.Get("/people/me/connections", p.listConnections)
		r.Get("/people/{resourceId}", p.getPerson)
		r.Patch("/people/{resourceId}:updateContact", p.updateContact)
		r.Delete("/people/{resourceId}:deleteContact", p.deleteContact)
		r.Get("/people:searchContacts", p.searchContacts)
		r.Post("/people:createContact", p.createContact)
	}

	r.Route("/v1", registerPeopleV1Routes)
	r.Route("/people/v1", registerPeopleV1Routes)
}

func (p *GooglePlugin) listConnections(w http.ResponseWriter, r *http.Request) {
	if p.store == nil {
		writeError(w, 500, "Plugin not initialized", "INTERNAL")
		return
	}

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
		if err := json.Unmarshal([]byte(person.Data), &data); err != nil {
			log.Printf("Failed to unmarshal person data: %v", err)
			data = map[string]any{}
		}

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
	if p.store == nil {
		writeError(w, 500, "Plugin not initialized", "INTERNAL")
		return
	}

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
		if err := json.Unmarshal([]byte(person.Data), &data); err != nil {
			log.Printf("Failed to unmarshal person data: %v", err)
			data = map[string]any{}
		}

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
	if p.store == nil {
		writeError(w, 500, "Plugin not initialized", "INTERNAL")
		return
	}

	userID := auth.UserFromContext(r.Context())
	resourceID := urlParam(r, "resourceId")
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
	if err := json.Unmarshal([]byte(person.Data), &data); err != nil {
		log.Printf("Failed to unmarshal person data: %v", err)
		data = map[string]any{}
	}

	resp := map[string]any{
		"resourceName": person.ResourceName,
	}
	// Merge data fields
	for k, v := range data {
		resp[k] = v
	}

	writeJSON(w, resp)
}

func (p *GooglePlugin) createContact(w http.ResponseWriter, r *http.Request) {
	if p.store == nil {
		writeError(w, 500, "Plugin not initialized", "INTERNAL")
		return
	}

	userID := auth.UserFromContext(r.Context())

	// Parse request body
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "Invalid request body", "INVALID_ARGUMENT")
		return
	}

	// Extract name and email from the request
	var name, email string
	if names, ok := req["names"].([]any); ok && len(names) > 0 {
		if nameObj, ok := names[0].(map[string]any); ok {
			if displayName, ok := nameObj["displayName"].(string); ok {
				name = displayName
			}
		}
	}
	if emailAddresses, ok := req["emailAddresses"].([]any); ok && len(emailAddresses) > 0 {
		if emailObj, ok := emailAddresses[0].(map[string]any); ok {
			if value, ok := emailObj["value"].(string); ok {
				email = value
			}
		}
	}

	// Validate that at least one field is provided
	if name == "" && email == "" {
		writeError(w, 400, "Contact must have at least a name or email", "INVALID_ARGUMENT")
		return
	}

	// Create the contact
	person, err := p.store.CreatePersonFromForm(userID, name, email)
	if err != nil {
		writeError(w, 500, "Failed to create contact", "INTERNAL")
		return
	}

	// Return the created contact
	w.WriteHeader(http.StatusCreated)
	resp := map[string]any{
		"resourceName": person.ResourceName,
		"names": []map[string]string{
			{"displayName": person.DisplayName},
		},
		"emailAddresses": []map[string]string{
			{"value": person.Email},
		},
	}

	writeJSON(w, resp)
}

func (p *GooglePlugin) updateContact(w http.ResponseWriter, r *http.Request) {
	if p.store == nil {
		writeError(w, 500, "Plugin not initialized", "INTERNAL")
		return
	}

	userID := auth.UserFromContext(r.Context())
	resourceID := urlParam(r, "resourceId")
	resourceName := "people/" + resourceID

	// Handle full resource name format
	if strings.HasPrefix(resourceID, "people/") {
		resourceName = resourceID
	}

	// Parse request body
	var req map[string]any
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "Invalid request body", "INVALID_ARGUMENT")
		return
	}

	// Update the contact
	person, err := p.store.UpdatePerson(userID, resourceName, req)
	if err != nil {
		if err.Error() == "person not found" {
			writeError(w, 404, "Contact not found", "NOT_FOUND")
		} else {
			writeError(w, 500, "Failed to update contact", "INTERNAL")
		}
		return
	}

	// Parse the updated data for response
	var data map[string]any
	if err := json.Unmarshal([]byte(person.Data), &data); err != nil {
		log.Printf("Failed to unmarshal person data: %v", err)
		data = map[string]any{}
	}

	resp := map[string]any{
		"resourceName": person.ResourceName,
	}
	// Merge data fields
	for k, v := range data {
		resp[k] = v
	}

	writeJSON(w, resp)
}

func (p *GooglePlugin) deleteContact(w http.ResponseWriter, r *http.Request) {
	if p.store == nil {
		writeError(w, 500, "Plugin not initialized", "INTERNAL")
		return
	}

	userID := auth.UserFromContext(r.Context())
	resourceID := urlParam(r, "resourceId")
	resourceName := "people/" + resourceID

	// Handle full resource name format
	if strings.HasPrefix(resourceID, "people/") {
		resourceName = resourceID
	}

	// Delete the contact
	err := p.store.DeletePerson(userID, resourceName)
	if err != nil {
		if err.Error() == "person not found" {
			writeError(w, 404, "Contact not found", "NOT_FOUND")
		} else {
			writeError(w, 500, "Failed to delete contact", "INTERNAL")
		}
		return
	}

	// Return 204 No Content on success
	w.WriteHeader(http.StatusNoContent)
}
