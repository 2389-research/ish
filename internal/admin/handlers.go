// ABOUTME: HTTP handlers for admin UI pages.
// ABOUTME: Serves dashboard and CRUD pages for Gmail, Calendar, People.

package admin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/2389/ish/internal/seed"
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
		r.Get("/gmail", h.gmailList)
		r.Get("/gmail/new", h.gmailForm)
		r.Get("/gmail/{id}", h.gmailView)
		r.Post("/gmail", h.gmailCreate)
		r.Post("/gmail/generate", h.gmailGenerate)
		r.Delete("/gmail/{id}", h.gmailDelete)
		r.Get("/calendar", h.calendarList)
		r.Get("/calendar/new", h.calendarForm)
		r.Get("/calendar/{id}", h.calendarView)
		r.Post("/calendar", h.calendarCreate)
		r.Post("/calendar/generate", h.calendarGenerate)
		r.Delete("/calendar/{id}", h.calendarDelete)
		r.Get("/people", h.peopleList)
		r.Get("/people/new", h.peopleForm)
		r.Get("/people/{id}", h.peopleView)
		r.Post("/people", h.peopleCreate)
		r.Post("/people/generate", h.peopleGenerate)
		r.Delete("/people/{id}", h.peopleDelete)
		r.Get("/tasks", h.tasksList)
		r.Get("/tasks/new", h.tasksForm)
		r.Get("/tasks/{id}", h.tasksView)
		r.Post("/tasks", h.tasksCreate)
		r.Post("/tasks/generate", h.tasksGenerate)
		r.Delete("/tasks/{id}", h.tasksDelete)
		r.Get("/logs", h.logsList)
	})

	// Register plugin admin routes
	pluginHandlers := &PluginHandlers{}
	pluginHandlers.RegisterRoutes(r)
}

func (h *Handlers) dashboard(w http.ResponseWriter, r *http.Request) {
	counts, err := h.store.GetCounts()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Get all plugins and their data
	plugins := getPluginDashboardData(h.store)

	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "dashboard", map[string]any{
		"MessageCount": counts.Messages,
		"ThreadCount":  counts.Threads,
		"EventCount":   counts.Events,
		"PeopleCount":  counts.People,
		"Tasks":        counts.Tasks,
		"Requests":     counts.Requests,
		"Plugins":      plugins,
	})
}

func (h *Handlers) guide(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "guide", nil)
}

func (h *Handlers) gmailList(w http.ResponseWriter, r *http.Request) {
	messages, err := h.store.ListAllGmailMessages()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "gmail-list", map[string]any{
		"Messages": messages,
	})
}

func (h *Handlers) gmailView(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	msg, err := h.store.GetGmailMessageDetail("harper", id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "gmail-view", msg)
}

func (h *Handlers) gmailDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.DeleteGmailMessage(id); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) gmailForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "gmail-form", nil)
}

func (h *Handlers) gmailCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	from := r.FormValue("from")
	subject := r.FormValue("subject")
	body := r.FormValue("body")
	labelsStr := r.FormValue("labels")

	labels := strings.Split(labelsStr, ",")
	for i := range labels {
		labels[i] = strings.TrimSpace(labels[i])
	}

	msg, err := h.store.CreateGmailMessageFromForm("harper", from, subject, body, labels)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Return just the row for htmx to append
	w.Header().Set("Content-Type", "text/html")
	renderPartial(w, "gmail-row", msg)
}

func (h *Handlers) calendarList(w http.ResponseWriter, r *http.Request) {
	events, err := h.store.ListAllCalendarEvents()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "calendar-list", map[string]any{"Events": events})
}

func (h *Handlers) calendarForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "calendar-form", nil)
}

func (h *Handlers) calendarCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	evt, err := h.store.CreateCalendarEventFromForm(
		r.FormValue("summary"),
		r.FormValue("description"),
		r.FormValue("start"),
		r.FormValue("end"),
	)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	renderPartial(w, "calendar-row", evt)
}

func (h *Handlers) calendarView(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	evt, err := h.store.GetCalendarEvent("primary", id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if evt == nil {
		http.Error(w, "event not found", 404)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "calendar-view", evt)
}

func (h *Handlers) calendarDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.DeleteCalendarEvent("primary", id); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) peopleList(w http.ResponseWriter, r *http.Request) {
	people, err := h.store.ListAllPeople()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "people-list", map[string]any{"People": people})
}

func (h *Handlers) peopleForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "people-form", nil)
}

func (h *Handlers) peopleCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	p, err := h.store.CreatePersonFromForm("harper", r.FormValue("name"), r.FormValue("email"))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	renderPartial(w, "people-row", p)
}

func (h *Handlers) peopleView(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := h.store.GetPersonView("harper", "people/"+id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "people-view", p)
}

func (h *Handlers) peopleDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.DeletePerson(id); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) gmailGenerate(w http.ResponseWriter, r *http.Request) {
	gen := seed.NewGenerator("harper")
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	email, err := gen.GenerateSingleEmail(ctx)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Create the email in store
	threadID := fmt.Sprintf("thr_%d", time.Now().UnixNano())
	msgID := fmt.Sprintf("msg_%d", time.Now().UnixNano())

	snippet := email.Body
	if len(snippet) > 100 {
		snippet = snippet[:100] + "..."
	}

	h.store.CreateGmailThread(&store.GmailThread{
		ID:      threadID,
		UserID:  "harper",
		Snippet: snippet,
	})

	bodyEncoded := base64.StdEncoding.EncodeToString([]byte(email.Body))
	payload := fmt.Sprintf(`{"headers":[{"name":"From","value":"%s"},{"name":"To","value":"%s"},{"name":"Subject","value":"%s"}],"body":{"data":"%s"}}`,
		email.From, email.To, email.Subject, bodyEncoded)

	h.store.CreateGmailMessage(&store.GmailMessage{
		ID:           msgID,
		UserID:       "harper",
		ThreadID:     threadID,
		LabelIDs:     email.Labels,
		Snippet:      snippet,
		InternalDate: time.Now().UnixMilli(),
		Payload:      payload,
	})

	w.Header().Set("Content-Type", "text/html")
	renderPartial(w, "gmail-row", &store.GmailMessageView{
		ID:       msgID,
		Subject:  email.Subject,
		Snippet:  snippet,
		LabelIDs: email.Labels,
	})
}

func (h *Handlers) calendarGenerate(w http.ResponseWriter, r *http.Request) {
	gen := seed.NewGenerator("harper")
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	event, err := gen.GenerateSingleEvent(ctx)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	// Build attendees JSON
	attendees := make([]map[string]string, len(event.Attendees))
	for i, email := range event.Attendees {
		attendees[i] = map[string]string{"email": email}
	}
	attendeesJSON, _ := json.Marshal(attendees)

	evtID := fmt.Sprintf("evt_%d", time.Now().UnixNano())
	evt := &store.CalendarEvent{
		ID:          evtID,
		CalendarID:  "primary",
		Summary:     event.Summary,
		Description: event.Description,
		StartTime:   event.StartTime,
		EndTime:     event.EndTime,
		Attendees:   string(attendeesJSON),
	}

	evt, err = h.store.CreateCalendarEvent(evt)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	renderPartial(w, "calendar-row", evt)
}

func (h *Handlers) peopleGenerate(w http.ResponseWriter, r *http.Request) {
	gen := seed.NewGenerator("harper")
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	contact, err := gen.GenerateSingleContact(ctx)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	personData := map[string]any{
		"names": []map[string]string{
			{"displayName": contact.Name},
		},
		"emailAddresses": []map[string]string{
			{"value": contact.Email},
		},
	}
	if contact.Phone != "" {
		personData["phoneNumbers"] = []map[string]string{
			{"value": contact.Phone},
		}
	}
	if contact.Company != "" {
		personData["organizations"] = []map[string]string{
			{"name": contact.Company},
		}
	}
	dataJSON, _ := json.Marshal(personData)

	id := fmt.Sprintf("c%d", time.Now().UnixNano())
	resourceName := "people/" + id
	if err := h.store.CreatePerson(&store.Person{
		ResourceName: resourceName,
		UserID:       "harper",
		Data:         string(dataJSON),
	}); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	renderPartial(w, "people-row", &store.PersonView{
		ID:           id,
		ResourceName: resourceName,
		DisplayName:  contact.Name,
		Email:        contact.Email,
	})
}

func (h *Handlers) tasksList(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.store.ListAllTasks()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "tasks-list", map[string]any{"Tasks": tasks})
}

func (h *Handlers) tasksForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "tasks-form", nil)
}

func (h *Handlers) tasksCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	task, err := h.store.CreateTaskFromForm(
		r.FormValue("title"),
		r.FormValue("notes"),
		r.FormValue("due"),
		r.FormValue("status"),
	)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	renderPartial(w, "tasks-row", task)
}

func (h *Handlers) tasksView(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	task, err := h.store.GetTask("@default", id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "tasks-view", task)
}

func (h *Handlers) tasksDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.DeleteTask("@default", id); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) tasksGenerate(w http.ResponseWriter, r *http.Request) {
	// Generate a task using static data (no AI generation for now)
	titles := []string{
		"Review project proposal",
		"Prepare quarterly report",
		"Schedule team meeting",
		"Update documentation",
		"Fix production bug",
		"Refactor authentication logic",
		"Write unit tests",
		"Deploy to staging environment",
	}
	notes := []string{
		"High priority - needs review by end of week",
		"Include metrics from last quarter",
		"Discuss upcoming sprint planning",
		"Update API documentation with new endpoints",
		"User reported login issues",
		"Improve code maintainability",
		"Increase test coverage to 80%",
		"Test all new features before production deploy",
	}

	idx := int(time.Now().UnixNano()) % len(titles)

	task, err := h.store.CreateTaskFromForm(
		titles[idx],
		notes[idx],
		time.Now().Add(7*24*time.Hour).Format(time.RFC3339),
		"needsAction",
	)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	renderPartial(w, "tasks-row", task)
}

func (h *Handlers) logsList(w http.ResponseWriter, r *http.Request) {
	// Get filter parameters
	method := r.URL.Query().Get("method")
	pathPrefix := r.URL.Query().Get("path")
	statusCode := 0
	if sc := r.URL.Query().Get("status"); sc != "" {
		fmt.Sscanf(sc, "%d", &statusCode)
	}

	logs, err := h.store.GetRequestLogs(&store.RequestLogQuery{
		Limit:      100,
		Offset:     0,
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

	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "logs-list", map[string]any{
		"Logs":         logs,
		"Stats":        stats,
		"TopEndpoints": topEndpoints,
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
