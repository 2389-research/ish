// ABOUTME: HTTP handlers for admin UI pages.
// ABOUTME: Serves dashboard and CRUD pages for Gmail, Calendar, People.

package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/2389/ish/internal/store"
	"github.com/2389/ish/plugins/core"
	"github.com/go-chi/chi/v5"
)

type Handlers struct {
	store *store.Store
}

func NewHandlers(s *store.Store) *Handlers {
	return &Handlers{store: s}
}

func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Route("/admin", func(r chi.Router) {
		r.Get("/", h.dashboard)
		r.Get("/guide", h.guide)

		// Redirect old Google routes to new plugin routes
		r.Get("/gmail", h.redirectToPluginRoute("/admin/plugins/google/messages"))
		r.Get("/gmail/new", h.redirectToPluginRoute("/admin/plugins/google/messages/new"))
		r.Get("/gmail/{id}", h.redirectGmailView)
		r.Get("/calendar", h.redirectToPluginRoute("/admin/plugins/google/events"))
		r.Get("/calendar/new", h.redirectToPluginRoute("/admin/plugins/google/events/new"))
		r.Get("/calendar/{id}", h.redirectCalendarView)
		r.Get("/people", h.redirectToPluginRoute("/admin/plugins/google/contacts"))
		r.Get("/people/new", h.redirectToPluginRoute("/admin/plugins/google/contacts/new"))
		r.Get("/people/{id}", h.redirectPeopleView)
		r.Get("/tasks", h.redirectToPluginRoute("/admin/plugins/google/tasks"))
		r.Get("/tasks/new", h.redirectToPluginRoute("/admin/plugins/google/tasks/new"))
		r.Get("/tasks/{id}", h.redirectTasksView)

		r.Get("/logs", h.logsList)
	})

	// Register plugin admin routes
	pluginHandlers := &PluginHandlers{}
	pluginHandlers.RegisterRoutes(r)
}

func (h *Handlers) dashboard(w http.ResponseWriter, r *http.Request) {
	// Get all plugins and their data
	plugins := getPluginDashboardData(h.store)

	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "dashboard", map[string]any{
		"MessageCount": 0, // Stats aggregation not implemented - view resources directly
		"ThreadCount":  0, // Stats aggregation not implemented - view resources directly
		"EventCount":   0, // Stats aggregation not implemented - view resources directly
		"PeopleCount":  0, // Stats aggregation not implemented - view resources directly
		"Tasks":        0, // Stats aggregation not implemented - view resources directly
		"Requests":     0, // Stats aggregation not implemented - view resources directly
		"Plugins":      plugins,
	})
}

func (h *Handlers) guide(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "guide", nil)
}

func (h *Handlers) gmailList(w http.ResponseWriter, r *http.Request) {
	// TODO: Access Google plugin store directly
	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "gmail-list", map[string]any{
		"Messages": []any{},
	})
}

func (h *Handlers) gmailView(w http.ResponseWriter, r *http.Request) {
	// TODO: Access Google plugin store directly
	http.Error(w, "Admin UI is view-only - use API endpoints for detailed operations", http.StatusNotImplemented)
}

func (h *Handlers) gmailDelete(w http.ResponseWriter, r *http.Request) {
	// TODO: Access Google plugin store directly
	http.Error(w, "Admin UI is view-only - use API endpoints for detailed operations", http.StatusNotImplemented)
}

func (h *Handlers) gmailForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "gmail-form", nil)
}

func (h *Handlers) gmailCreate(w http.ResponseWriter, r *http.Request) {
	// TODO: Access Google plugin store directly
	http.Error(w, "Admin UI is view-only - use API endpoints for detailed operations", http.StatusNotImplemented)
}

func (h *Handlers) calendarList(w http.ResponseWriter, r *http.Request) {
	// TODO: Access Google plugin store directly
	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "calendar-list", map[string]any{"Events": []any{}})
}

func (h *Handlers) calendarForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "calendar-form", nil)
}

func (h *Handlers) calendarCreate(w http.ResponseWriter, r *http.Request) {
	// TODO: Access Google plugin store directly
	http.Error(w, "Admin UI is view-only - use API endpoints for detailed operations", http.StatusNotImplemented)
}

func (h *Handlers) calendarView(w http.ResponseWriter, r *http.Request) {
	// TODO: Access Google plugin store directly
	http.Error(w, "Admin UI is view-only - use API endpoints for detailed operations", http.StatusNotImplemented)
}

func (h *Handlers) calendarDelete(w http.ResponseWriter, r *http.Request) {
	// TODO: Access Google plugin store directly
	http.Error(w, "Admin UI is view-only - use API endpoints for detailed operations", http.StatusNotImplemented)
}

func (h *Handlers) peopleList(w http.ResponseWriter, r *http.Request) {
	// TODO: Access Google plugin store directly
	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "people-list", map[string]any{"People": []any{}})
}

func (h *Handlers) peopleForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "people-form", nil)
}

func (h *Handlers) peopleCreate(w http.ResponseWriter, r *http.Request) {
	// TODO: Access Google plugin store directly
	http.Error(w, "Admin UI is view-only - use API endpoints for detailed operations", http.StatusNotImplemented)
}

func (h *Handlers) peopleView(w http.ResponseWriter, r *http.Request) {
	// TODO: Access Google plugin store directly
	http.Error(w, "Admin UI is view-only - use API endpoints for detailed operations", http.StatusNotImplemented)
}

func (h *Handlers) peopleDelete(w http.ResponseWriter, r *http.Request) {
	// TODO: Access Google plugin store directly
	http.Error(w, "Admin UI is view-only - use API endpoints for detailed operations", http.StatusNotImplemented)
}

func (h *Handlers) gmailGenerate(w http.ResponseWriter, r *http.Request) {
	// TODO: Access Google plugin store directly
	http.Error(w, "Admin UI is view-only - use API endpoints for detailed operations", http.StatusNotImplemented)
}

func (h *Handlers) calendarGenerate(w http.ResponseWriter, r *http.Request) {
	// TODO: Access Google plugin store directly
	http.Error(w, "Admin UI is view-only - use API endpoints for detailed operations", http.StatusNotImplemented)
}

func (h *Handlers) peopleGenerate(w http.ResponseWriter, r *http.Request) {
	// TODO: Access Google plugin store directly
	http.Error(w, "Admin UI is view-only - use API endpoints for detailed operations", http.StatusNotImplemented)
}

func (h *Handlers) tasksList(w http.ResponseWriter, r *http.Request) {
	// TODO: Access Google plugin store directly
	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "tasks-list", map[string]any{"Tasks": []any{}})
}

func (h *Handlers) tasksForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "tasks-form", nil)
}

func (h *Handlers) tasksCreate(w http.ResponseWriter, r *http.Request) {
	// TODO: Access Google plugin store directly
	http.Error(w, "Admin UI is view-only - use API endpoints for detailed operations", http.StatusNotImplemented)
}

func (h *Handlers) tasksView(w http.ResponseWriter, r *http.Request) {
	// TODO: Access Google plugin store directly
	http.Error(w, "Admin UI is view-only - use API endpoints for detailed operations", http.StatusNotImplemented)
}

func (h *Handlers) tasksDelete(w http.ResponseWriter, r *http.Request) {
	// TODO: Access Google plugin store directly
	http.Error(w, "Admin UI is view-only - use API endpoints for detailed operations", http.StatusNotImplemented)
}

func (h *Handlers) tasksGenerate(w http.ResponseWriter, r *http.Request) {
	// TODO: Access Google plugin store directly
	http.Error(w, "Admin UI is view-only - use API endpoints for detailed operations", http.StatusNotImplemented)
}

func (h *Handlers) logsList(w http.ResponseWriter, r *http.Request) {
	// Get filter parameters
	pluginName := r.URL.Query().Get("plugin")
	method := r.URL.Query().Get("method")
	pathPrefix := r.URL.Query().Get("path")
	statusCode := 0
	if sc := r.URL.Query().Get("status"); sc != "" {
		fmt.Sscanf(sc, "%d", &statusCode)
	}

	logs, err := h.store.GetRequestLogs(&store.RequestLogQuery{
		Limit:      100,
		Offset:     0,
		PluginName: pluginName,
		Method:     method,
		PathPrefix: pathPrefix,
		StatusCode: statusCode,
	})
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Pretty-print JSON in request/response bodies
	for _, log := range logs {
		log.RequestBody = prettyJSON(log.RequestBody)
		log.ResponseBody = prettyJSON(log.ResponseBody)
	}

	stats, err := h.store.GetRequestLogStats()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	topEndpoints, err := h.store.GetTopEndpoints(10)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Get all plugin names for the dropdown
	pluginNames := core.Names()

	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "logs-list", map[string]any{
		"Logs":           logs,
		"Stats":          stats,
		"TopEndpoints":   topEndpoints,
		"PluginNames":    pluginNames,
		"SelectedPlugin": pluginName,
	})
}

// prettyJSON formats JSON with indentation, or returns original string if not valid JSON
func prettyJSON(s string) string {
	if s == "" {
		return s
	}
	var obj any
	if err := json.Unmarshal([]byte(s), &obj); err != nil {
		return s // Not valid JSON, return as-is
	}
	formatted, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return s
	}
	return string(formatted)
}

// PluginDashboardData represents data for a plugin on the dashboard
type PluginDashboardData struct {
	Name           string
	Health         core.HealthStatus
	RequestCount   int
	ErrorRate      float64
	RecentRequests []*store.RequestLog
	Resources      []PluginResourceLink
}

// PluginResourceLink represents a quick link to a plugin resource
type PluginResourceLink struct {
	Name string
	Slug string
	URL  string
}

// getPluginDashboardData fetches dashboard data for all plugins
func getPluginDashboardData(s *store.Store) []PluginDashboardData {
	yesterday := time.Now().Add(-24 * time.Hour)
	var pluginData []PluginDashboardData

	for _, plugin := range core.All() {
		name := plugin.Name()
		health := plugin.Health()

		// Get request count in last 24 hours
		requestCount, _ := s.GetPluginRequestCount(name, yesterday)

		// Get error rate in last 24 hours
		errorRate, _ := s.GetPluginErrorRate(name, yesterday)

		// Get 5 most recent requests
		recentRequests, _ := s.GetRecentRequests(name, 5)

		// Get resource links from schema
		schema := plugin.Schema()
		var resources []PluginResourceLink
		for _, res := range schema.Resources {
			resources = append(resources, PluginResourceLink{
				Name: res.Name,
				Slug: res.Slug,
				URL:  fmt.Sprintf("/admin/plugins/%s/%s", name, res.Slug),
			})
		}

		pluginData = append(pluginData, PluginDashboardData{
			Name:           name,
			Health:         health,
			RequestCount:   requestCount,
			ErrorRate:      errorRate,
			RecentRequests: recentRequests,
			Resources:      resources,
		})
	}

	return pluginData
}

// Redirect helper functions for old Google routes
func (h *Handlers) redirectToPluginRoute(newPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, newPath, http.StatusMovedPermanently)
	}
}

func (h *Handlers) redirectGmailView(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	http.Redirect(w, r, "/admin/plugins/google/messages/"+id, http.StatusMovedPermanently)
}

func (h *Handlers) redirectCalendarView(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	http.Redirect(w, r, "/admin/plugins/google/events/"+id, http.StatusMovedPermanently)
}

func (h *Handlers) redirectPeopleView(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	http.Redirect(w, r, "/admin/plugins/google/contacts/"+id, http.StatusMovedPermanently)
}

func (h *Handlers) redirectTasksView(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	http.Redirect(w, r, "/admin/plugins/google/tasks/"+id, http.StatusMovedPermanently)
}
