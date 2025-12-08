// ABOUTME: Google plugin for ISH.
// ABOUTME: Provides Gmail, Calendar, People, and Tasks APIs.

package google

import (
	"context"
	"database/sql"

	"github.com/2389/ish/plugins/core"
	"github.com/go-chi/chi/v5"
)

func init() {
	core.Register(&GooglePlugin{})
}

type GooglePlugin struct {
	store *GoogleStore
}

func (p *GooglePlugin) Name() string {
	return "google"
}

func (p *GooglePlugin) Health() core.HealthStatus {
	return core.HealthStatus{
		Status:  "healthy",
		Message: "Google plugin operational",
	}
}

func (p *GooglePlugin) RegisterRoutes(r chi.Router) {
	p.registerGmailRoutes(r)
	p.registerCalendarRoutes(r)
	p.registerPeopleRoutes(r)
	p.registerTasksRoutes(r)
}

func (p *GooglePlugin) RegisterAuth(r chi.Router) {
	// OAuth endpoints will be added later
}

func (p *GooglePlugin) Schema() core.PluginSchema {
	return getGoogleSchema()
}

// Seed implementation is in seed.go

func (p *GooglePlugin) ValidateToken(token string) bool {
	// Token validation will be implemented later
	return true
}

// SetDB initializes the Google plugin's database store
func (p *GooglePlugin) SetDB(db *sql.DB) error {
	store, err := NewGoogleStore(db)
	if err != nil {
		return err
	}
	p.store = store
	return nil
}

// ListResources implements core.DataProvider to expose data to admin UI
func (p *GooglePlugin) ListResources(ctx context.Context, slug string, opts core.ListOptions) ([]map[string]interface{}, error) {
	switch slug {
	case "messages":
		messages, err := p.store.ListAllGmailMessages()
		if err != nil {
			return nil, err
		}
		return convertMessagesToMaps(messages), nil
	case "events":
		events, err := p.store.ListAllCalendarEvents()
		if err != nil {
			return nil, err
		}
		return convertEventsToMaps(events), nil
	case "contacts":
		people, err := p.store.ListAllPeople()
		if err != nil {
			return nil, err
		}
		return convertPeopleToMaps(people), nil
	case "tasks":
		tasks, err := p.store.ListAllTasks()
		if err != nil {
			return nil, err
		}
		return convertTasksToMaps(tasks), nil
	default:
		return nil, nil
	}
}

// GetResource implements core.DataProvider
func (p *GooglePlugin) GetResource(ctx context.Context, slug string, id string) (map[string]interface{}, error) {
	// Not implemented yet
	return nil, nil
}

// Conversion helpers
func convertMessagesToMaps(messages []GmailMessageView) []map[string]interface{} {
	result := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		result[i] = map[string]interface{}{
			"id":      msg.ID,
			"subject": msg.Subject,
			"snippet": msg.Snippet,
		}
	}
	return result
}

func convertEventsToMaps(events []CalendarEvent) []map[string]interface{} {
	result := make([]map[string]interface{}, len(events))
	for i, evt := range events {
		result[i] = map[string]interface{}{
			"id":          evt.ID,
			"summary":     evt.Summary,
			"description": evt.Description,
			"start":       evt.StartTime,
			"end":         evt.EndTime,
			"location":    evt.Location,
		}
	}
	return result
}

func convertPeopleToMaps(people []PersonView) []map[string]interface{} {
	result := make([]map[string]interface{}, len(people))
	for i, person := range people {
		result[i] = map[string]interface{}{
			"id":      person.ResourceName,
			"name":    person.DisplayName,
			"email":   person.Email,
			"phone":   "", // Not in PersonView
			"company": "", // Not in PersonView
		}
	}
	return result
}

func convertTasksToMaps(tasks []*Task) []map[string]interface{} {
	result := make([]map[string]interface{}, len(tasks))
	for i, task := range tasks {
		result[i] = map[string]interface{}{
			"id":     task.ID,
			"title":  task.Title,
			"notes":  task.Notes,
			"due":    task.Due,
			"status": task.Status,
		}
	}
	return result
}
