// ABOUTME: Google API Discovery Service implementation for ISH.
// ABOUTME: Returns discovery documents that describe the Gmail, Calendar, and People APIs.

package google

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (p *GooglePlugin) registerDiscoveryRoutes(r chi.Router) {
	r.Get("/discovery/v1/apis/{api}/{version}/rest", p.getDiscoveryDoc)
}

func (p *GooglePlugin) getDiscoveryDoc(w http.ResponseWriter, r *http.Request) {
	api := urlParam(r, "api")
	version := urlParam(r, "version")

	var doc interface{}
	switch api {
	case "gmail":
		if version == "v1" {
			doc = getGmailDiscoveryDoc(r)
		}
	case "calendar":
		if version == "v3" {
			doc = getCalendarDiscoveryDoc(r)
		}
	case "people":
		if version == "v1" {
			doc = getPeopleDiscoveryDoc(r)
		}
	}

	if doc == nil {
		http.Error(w, "API not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(doc)
}

func getGmailDiscoveryDoc(r *http.Request) map[string]interface{} {
	baseURL := getBaseURL(r)
	return map[string]interface{}{
		"kind":              "discovery#restDescription",
		"discoveryVersion":  "v1",
		"id":                "gmail:v1",
		"name":              "gmail",
		"version":           "v1",
		"title":             "Gmail API",
		"description":       "ISH mock implementation of the Gmail API",
		"protocol":          "rest",
		"baseUrl":           baseURL + "/gmail/v1/",
		"basePath":          "/gmail/v1/",
		"rootUrl":           baseURL + "/",
		"servicePath":       "gmail/v1/",
		"batchPath":         "batch/gmail/v1",
		"documentationLink": "https://developers.google.com/gmail/api/",
		"auth": map[string]interface{}{
			"oauth2": map[string]interface{}{
				"scopes": map[string]interface{}{
					"https://www.googleapis.com/auth/gmail.readonly": map[string]interface{}{
						"description": "View your email messages and settings",
					},
					"https://www.googleapis.com/auth/gmail.send": map[string]interface{}{
						"description": "Send email on your behalf",
					},
					"https://www.googleapis.com/auth/gmail.modify": map[string]interface{}{
						"description": "View and modify but not delete your email",
					},
				},
			},
		},
		"resources": map[string]interface{}{
			"users": map[string]interface{}{
				"resources": map[string]interface{}{
					"messages": map[string]interface{}{
						"methods": map[string]interface{}{
							"list": map[string]interface{}{
								"id":          "gmail.users.messages.list",
								"path":        "users/{userId}/messages",
								"httpMethod":  "GET",
								"description": "Lists the messages in the user's mailbox",
								"parameters": map[string]interface{}{
									"userId": map[string]interface{}{
										"type":     "string",
										"required": true,
										"location": "path",
									},
									"maxResults": map[string]interface{}{
										"type":     "integer",
										"location": "query",
									},
									"pageToken": map[string]interface{}{
										"type":     "string",
										"location": "query",
									},
									"q": map[string]interface{}{
										"type":     "string",
										"location": "query",
									},
								},
							},
							"get": map[string]interface{}{
								"id":          "gmail.users.messages.get",
								"path":        "users/{userId}/messages/{id}",
								"httpMethod":  "GET",
								"description": "Gets the specified message",
								"parameters": map[string]interface{}{
									"userId": map[string]interface{}{
										"type":     "string",
										"required": true,
										"location": "path",
									},
									"id": map[string]interface{}{
										"type":     "string",
										"required": true,
										"location": "path",
									},
								},
							},
							"send": map[string]interface{}{
								"id":          "gmail.users.messages.send",
								"path":        "users/{userId}/messages/send",
								"httpMethod":  "POST",
								"description": "Sends the specified message",
								"parameters": map[string]interface{}{
									"userId": map[string]interface{}{
										"type":     "string",
										"required": true,
										"location": "path",
									},
								},
							},
							"delete": map[string]interface{}{
								"id":          "gmail.users.messages.delete",
								"path":        "users/{userId}/messages/{id}",
								"httpMethod":  "DELETE",
								"description": "Deletes the specified message",
								"parameters": map[string]interface{}{
									"userId": map[string]interface{}{
										"type":     "string",
										"required": true,
										"location": "path",
									},
									"id": map[string]interface{}{
										"type":     "string",
										"required": true,
										"location": "path",
									},
								},
							},
							"trash": map[string]interface{}{
								"id":          "gmail.users.messages.trash",
								"path":        "users/{userId}/messages/{id}/trash",
								"httpMethod":  "POST",
								"description": "Moves the specified message to the trash",
								"parameters": map[string]interface{}{
									"userId": map[string]interface{}{
										"type":     "string",
										"required": true,
										"location": "path",
									},
									"id": map[string]interface{}{
										"type":     "string",
										"required": true,
										"location": "path",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func getCalendarDiscoveryDoc(r *http.Request) map[string]interface{} {
	baseURL := getBaseURL(r)
	return map[string]interface{}{
		"kind":              "discovery#restDescription",
		"discoveryVersion":  "v1",
		"id":                "calendar:v3",
		"name":              "calendar",
		"version":           "v3",
		"title":             "Calendar API",
		"description":       "ISH mock implementation of the Google Calendar API",
		"protocol":          "rest",
		"baseUrl":           baseURL + "/calendar/v3/",
		"basePath":          "/calendar/v3/",
		"rootUrl":           baseURL + "/",
		"servicePath":       "calendar/v3/",
		"documentationLink": "https://developers.google.com/calendar/",
		"resources": map[string]interface{}{
			"events": map[string]interface{}{
				"methods": map[string]interface{}{
					"list": map[string]interface{}{
						"id":          "calendar.events.list",
						"path":        "calendars/{calendarId}/events",
						"httpMethod":  "GET",
						"description": "Returns events on the specified calendar",
						"parameters": map[string]interface{}{
							"calendarId": map[string]interface{}{
								"type":     "string",
								"required": true,
								"location": "path",
							},
						},
					},
					"insert": map[string]interface{}{
						"id":          "calendar.events.insert",
						"path":        "calendars/{calendarId}/events",
						"httpMethod":  "POST",
						"description": "Creates an event",
						"parameters": map[string]interface{}{
							"calendarId": map[string]interface{}{
								"type":     "string",
								"required": true,
								"location": "path",
							},
						},
					},
					"get": map[string]interface{}{
						"id":          "calendar.events.get",
						"path":        "calendars/{calendarId}/events/{eventId}",
						"httpMethod":  "GET",
						"description": "Returns an event",
						"parameters": map[string]interface{}{
							"calendarId": map[string]interface{}{
								"type":     "string",
								"required": true,
								"location": "path",
							},
							"eventId": map[string]interface{}{
								"type":     "string",
								"required": true,
								"location": "path",
							},
						},
					},
					"update": map[string]interface{}{
						"id":          "calendar.events.update",
						"path":        "calendars/{calendarId}/events/{eventId}",
						"httpMethod":  "PUT",
						"description": "Updates an event",
						"parameters": map[string]interface{}{
							"calendarId": map[string]interface{}{
								"type":     "string",
								"required": true,
								"location": "path",
							},
							"eventId": map[string]interface{}{
								"type":     "string",
								"required": true,
								"location": "path",
							},
						},
					},
					"patch": map[string]interface{}{
						"id":          "calendar.events.patch",
						"path":        "calendars/{calendarId}/events/{eventId}",
						"httpMethod":  "PATCH",
						"description": "Updates an event. This method supports patch semantics",
						"parameters": map[string]interface{}{
							"calendarId": map[string]interface{}{
								"type":     "string",
								"required": true,
								"location": "path",
							},
							"eventId": map[string]interface{}{
								"type":     "string",
								"required": true,
								"location": "path",
							},
						},
					},
					"delete": map[string]interface{}{
						"id":          "calendar.events.delete",
						"path":        "calendars/{calendarId}/events/{eventId}",
						"httpMethod":  "DELETE",
						"description": "Deletes an event",
						"parameters": map[string]interface{}{
							"calendarId": map[string]interface{}{
								"type":     "string",
								"required": true,
								"location": "path",
							},
							"eventId": map[string]interface{}{
								"type":     "string",
								"required": true,
								"location": "path",
							},
						},
					},
				},
			},
		},
	}
}

func getPeopleDiscoveryDoc(r *http.Request) map[string]interface{} {
	baseURL := getBaseURL(r)
	return map[string]interface{}{
		"kind":              "discovery#restDescription",
		"discoveryVersion":  "v1",
		"id":                "people:v1",
		"name":              "people",
		"version":           "v1",
		"title":             "People API",
		"description":       "ISH mock implementation of the Google People API",
		"protocol":          "rest",
		"baseUrl":           baseURL + "/people/v1/",
		"basePath":          "/people/v1/",
		"rootUrl":           baseURL + "/",
		"servicePath":       "people/v1/",
		"documentationLink": "https://developers.google.com/people/",
		"resources": map[string]interface{}{
			"people": map[string]interface{}{
				"methods": map[string]interface{}{
					"get": map[string]interface{}{
						"id":          "people.people.get",
						"path":        "people/{resourceName}",
						"httpMethod":  "GET",
						"description": "Provides information about a person",
						"parameters": map[string]interface{}{
							"resourceName": map[string]interface{}{
								"type":     "string",
								"required": true,
								"location": "path",
							},
						},
					},
					"searchContacts": map[string]interface{}{
						"id":          "people.people.searchContacts",
						"path":        "people:searchContacts",
						"httpMethod":  "GET",
						"description": "Provides a list of contacts in the authenticated user's grouped contacts",
						"parameters": map[string]interface{}{
							"query": map[string]interface{}{
								"type":     "string",
								"location": "query",
							},
							"pageSize": map[string]interface{}{
								"type":     "integer",
								"location": "query",
							},
						},
					},
					"createContact": map[string]interface{}{
						"id":          "people.people.createContact",
						"path":        "people:createContact",
						"httpMethod":  "POST",
						"description": "Create a new contact",
					},
					"updateContact": map[string]interface{}{
						"id":          "people.people.updateContact",
						"path":        "people/{resourceName}:updateContact",
						"httpMethod":  "PATCH",
						"description": "Update an existing contact",
						"parameters": map[string]interface{}{
							"resourceName": map[string]interface{}{
								"type":     "string",
								"required": true,
								"location": "path",
							},
						},
					},
					"deleteContact": map[string]interface{}{
						"id":          "people.people.deleteContact",
						"path":        "people/{resourceName}:deleteContact",
						"httpMethod":  "DELETE",
						"description": "Delete a contact",
						"parameters": map[string]interface{}{
							"resourceName": map[string]interface{}{
								"type":     "string",
								"required": true,
								"location": "path",
							},
						},
					},
				},
			},
			"people.connections": map[string]interface{}{
				"methods": map[string]interface{}{
					"list": map[string]interface{}{
						"id":          "people.people.connections.list",
						"path":        "people/me/connections",
						"httpMethod":  "GET",
						"description": "Provides a list of the authenticated user's contacts",
						"parameters": map[string]interface{}{
							"pageSize": map[string]interface{}{
								"type":     "integer",
								"location": "query",
							},
							"pageToken": map[string]interface{}{
								"type":     "string",
								"location": "query",
							},
						},
					},
				},
			},
		},
	}
}

func getBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	host := r.Host
	return scheme + "://" + host
}
