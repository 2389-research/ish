# Admin UI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a web UI at `/admin` for viewing and managing fake Google API data (messages, events, contacts).

**Architecture:** Server-rendered HTML with Go templates, htmx for partial updates, Tailwind CSS for styling. All served from the existing ISH Go server with a new `internal/admin` package.

**Tech Stack:** Go html/template, htmx (CDN), Tailwind CSS (CDN), chi router

---

## Task 1: Template Infrastructure

**Files:**
- Create: `internal/admin/templates.go`
- Create: `internal/admin/templates/layout.html`

**Step 1: Create template loader**

Create `internal/admin/templates.go`:
```go
// ABOUTME: Template loading and rendering for admin UI.
// ABOUTME: Embeds HTML templates and provides render helpers.

package admin

import (
	"embed"
	"html/template"
	"io"
)

//go:embed templates/*
var templateFS embed.FS

var templates *template.Template

func init() {
	templates = template.Must(template.ParseFS(templateFS, "templates/*.html", "templates/**/*.html"))
}

func render(w io.Writer, name string, data any) error {
	return templates.ExecuteTemplate(w, name, data)
}
```

**Step 2: Create base layout**

Create `internal/admin/templates/layout.html`:
```html
{{define "layout"}}
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>ISH Admin</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>
</head>
<body class="bg-gray-100 min-h-screen">
    <nav class="bg-white shadow-sm border-b">
        <div class="max-w-7xl mx-auto px-4 py-3">
            <div class="flex items-center justify-between">
                <a href="/admin/" class="text-xl font-bold text-gray-800">ISH Admin</a>
                <div class="flex gap-4">
                    <a href="/admin/gmail" class="text-gray-600 hover:text-gray-900">Gmail</a>
                    <a href="/admin/calendar" class="text-gray-600 hover:text-gray-900">Calendar</a>
                    <a href="/admin/people" class="text-gray-600 hover:text-gray-900">People</a>
                </div>
            </div>
        </div>
    </nav>
    <main class="max-w-7xl mx-auto px-4 py-8">
        {{template "content" .}}
    </main>
</body>
</html>
{{end}}
```

**Step 3: Verify templates compile**

Run:
```bash
cd /Users/harper/Public/src/2389/ish && go build ./internal/admin/...
```

Expected: No errors (package compiles)

**Step 4: Commit**

```bash
git add internal/admin/
git commit -m "feat(admin): add template infrastructure with layout"
```

---

## Task 2: Dashboard Page

**Files:**
- Create: `internal/admin/handlers.go`
- Create: `internal/admin/templates/dashboard.html`
- Modify: `cmd/ish/main.go`

**Step 1: Create dashboard template**

Create `internal/admin/templates/dashboard.html`:
```html
{{define "content"}}
<div class="space-y-6">
    <h1 class="text-2xl font-bold text-gray-900">Dashboard</h1>

    <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
        <a href="/admin/gmail" class="block p-6 bg-white rounded-lg shadow hover:shadow-md transition">
            <h2 class="text-lg font-semibold text-gray-700">Gmail</h2>
            <p class="text-3xl font-bold text-blue-600">{{.MessageCount}}</p>
            <p class="text-sm text-gray-500">messages in {{.ThreadCount}} threads</p>
        </a>

        <a href="/admin/calendar" class="block p-6 bg-white rounded-lg shadow hover:shadow-md transition">
            <h2 class="text-lg font-semibold text-gray-700">Calendar</h2>
            <p class="text-3xl font-bold text-green-600">{{.EventCount}}</p>
            <p class="text-sm text-gray-500">events</p>
        </a>

        <a href="/admin/people" class="block p-6 bg-white rounded-lg shadow hover:shadow-md transition">
            <h2 class="text-lg font-semibold text-gray-700">People</h2>
            <p class="text-3xl font-bold text-purple-600">{{.PeopleCount}}</p>
            <p class="text-sm text-gray-500">contacts</p>
        </a>
    </div>
</div>
{{end}}
```

**Step 2: Create handlers**

Create `internal/admin/handlers.go`:
```go
// ABOUTME: HTTP handlers for admin UI pages.
// ABOUTME: Serves dashboard and CRUD pages for Gmail, Calendar, People.

package admin

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/2389/ish/internal/store"
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
	})
}

func (h *Handlers) dashboard(w http.ResponseWriter, r *http.Request) {
	counts, err := h.store.GetCounts()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	render(w, "layout", map[string]any{
		"MessageCount": counts.Messages,
		"ThreadCount":  counts.Threads,
		"EventCount":   counts.Events,
		"PeopleCount":  counts.People,
	})
}
```

**Step 3: Add GetCounts to store**

Add to `internal/store/store.go`:
```go
type Counts struct {
	Messages int
	Threads  int
	Events   int
	People   int
}

func (s *Store) GetCounts() (*Counts, error) {
	var c Counts
	s.db.QueryRow("SELECT COUNT(*) FROM gmail_messages").Scan(&c.Messages)
	s.db.QueryRow("SELECT COUNT(*) FROM gmail_threads").Scan(&c.Threads)
	s.db.QueryRow("SELECT COUNT(*) FROM calendar_events").Scan(&c.Events)
	s.db.QueryRow("SELECT COUNT(*) FROM people").Scan(&c.People)
	return &c, nil
}
```

**Step 4: Wire up admin handlers in main.go**

Add to `cmd/ish/main.go` in `newServer` function, after other handlers:
```go
import "github.com/2389/ish/internal/admin"

// In newServer(), after people handlers:
admin.NewHandlers(s).RegisterRoutes(r)
```

**Step 5: Test manually**

Run:
```bash
cd /Users/harper/Public/src/2389/ish && go build -o ish ./cmd/ish && ./ish seed && ./ish serve
```

Open: `http://localhost:9000/admin/`
Expected: Dashboard with counts for messages, events, people

**Step 6: Commit**

```bash
git add internal/admin/ internal/store/store.go cmd/ish/main.go
git commit -m "feat(admin): add dashboard page with data counts"
```

---

## Task 3: Gmail List and Delete

**Files:**
- Create: `internal/admin/templates/gmail/list.html`
- Create: `internal/admin/templates/gmail/row.html`
- Modify: `internal/admin/handlers.go`

**Step 1: Create Gmail list template**

Create `internal/admin/templates/gmail/list.html`:
```html
{{define "content"}}
<div class="space-y-6">
    <div class="flex items-center justify-between">
        <h1 class="text-2xl font-bold text-gray-900">Gmail Messages</h1>
        <a href="/admin/gmail/new" class="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">
            + New Message
        </a>
    </div>

    <div class="bg-white rounded-lg shadow overflow-hidden">
        <table class="min-w-full divide-y divide-gray-200">
            <thead class="bg-gray-50">
                <tr>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">ID</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Subject</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Snippet</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Labels</th>
                    <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>
                </tr>
            </thead>
            <tbody id="message-list" class="bg-white divide-y divide-gray-200">
                {{range .Messages}}
                {{template "gmail-row" .}}
                {{end}}
            </tbody>
        </table>
    </div>
</div>
{{end}}
```

**Step 2: Create Gmail row partial**

Create `internal/admin/templates/gmail/row.html`:
```html
{{define "gmail-row"}}
<tr id="msg-{{.ID}}">
    <td class="px-6 py-4 whitespace-nowrap text-sm font-mono text-gray-900">{{.ID}}</td>
    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{{.Subject}}</td>
    <td class="px-6 py-4 text-sm text-gray-500 max-w-xs truncate">{{.Snippet}}</td>
    <td class="px-6 py-4 whitespace-nowrap text-sm">
        {{range .LabelIDs}}
        <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-800 mr-1">{{.}}</span>
        {{end}}
    </td>
    <td class="px-6 py-4 whitespace-nowrap text-right text-sm">
        <button
            hx-delete="/admin/gmail/{{.ID}}"
            hx-target="#msg-{{.ID}}"
            hx-swap="delete"
            hx-confirm="Delete this message?"
            class="text-red-600 hover:text-red-900">
            Delete
        </button>
    </td>
</tr>
{{end}}
```

**Step 3: Add Gmail handlers**

Add to `internal/admin/handlers.go`:
```go
func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Route("/admin", func(r chi.Router) {
		r.Get("/", h.dashboard)
		r.Get("/gmail", h.gmailList)
		r.Delete("/gmail/{id}", h.gmailDelete)
	})
}

func (h *Handlers) gmailList(w http.ResponseWriter, r *http.Request) {
	messages, err := h.store.ListAllGmailMessages()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	render(w, "layout", map[string]any{
		"Messages": messages,
	})
}

func (h *Handlers) gmailDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.DeleteGmailMessage(id); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(http.StatusOK)
}
```

**Step 4: Add store methods**

Add to `internal/store/gmail.go`:
```go
type GmailMessageView struct {
	ID       string
	Subject  string
	Snippet  string
	LabelIDs []string
}

func (s *Store) ListAllGmailMessages() ([]GmailMessageView, error) {
	rows, err := s.db.Query("SELECT id, snippet, label_ids, payload FROM gmail_messages ORDER BY internal_date DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []GmailMessageView
	for rows.Next() {
		var m GmailMessageView
		var labelJSON, payload string
		if err := rows.Scan(&m.ID, &m.Snippet, &labelJSON, &payload); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(labelJSON), &m.LabelIDs)

		// Extract subject from payload
		var p struct {
			Headers []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"headers"`
		}
		json.Unmarshal([]byte(payload), &p)
		for _, h := range p.Headers {
			if h.Name == "Subject" {
				m.Subject = h.Value
				break
			}
		}
		messages = append(messages, m)
	}
	return messages, nil
}

func (s *Store) DeleteGmailMessage(id string) error {
	_, err := s.db.Exec("DELETE FROM gmail_messages WHERE id = ?", id)
	return err
}
```

**Step 5: Test manually**

Run server, go to `http://localhost:9000/admin/gmail`
- Should see message list
- Click delete, confirm, row should disappear

**Step 6: Commit**

```bash
git add internal/admin/ internal/store/gmail.go
git commit -m "feat(admin): add Gmail list and delete"
```

---

## Task 4: Gmail Create

**Files:**
- Create: `internal/admin/templates/gmail/form.html`
- Modify: `internal/admin/handlers.go`
- Modify: `internal/store/gmail.go`

**Step 1: Create Gmail form template**

Create `internal/admin/templates/gmail/form.html`:
```html
{{define "content"}}
<div class="space-y-6">
    <h1 class="text-2xl font-bold text-gray-900">New Message</h1>

    <form hx-post="/admin/gmail" hx-target="#message-list" hx-swap="beforeend" class="bg-white rounded-lg shadow p-6 space-y-4 max-w-2xl">
        <div>
            <label class="block text-sm font-medium text-gray-700">From</label>
            <input type="email" name="from" required class="mt-1 block w-full rounded border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 px-3 py-2 border">
        </div>
        <div>
            <label class="block text-sm font-medium text-gray-700">Subject</label>
            <input type="text" name="subject" required class="mt-1 block w-full rounded border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 px-3 py-2 border">
        </div>
        <div>
            <label class="block text-sm font-medium text-gray-700">Body</label>
            <textarea name="body" rows="4" class="mt-1 block w-full rounded border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 px-3 py-2 border"></textarea>
        </div>
        <div>
            <label class="block text-sm font-medium text-gray-700">Labels (comma-separated)</label>
            <input type="text" name="labels" value="INBOX" class="mt-1 block w-full rounded border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 px-3 py-2 border">
        </div>
        <div class="flex gap-4">
            <button type="submit" class="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">Create</button>
            <a href="/admin/gmail" class="px-4 py-2 bg-gray-200 text-gray-700 rounded hover:bg-gray-300">Cancel</a>
        </div>
    </form>
</div>
{{end}}
```

**Step 2: Add form and create handlers**

Add to `internal/admin/handlers.go` in RegisterRoutes:
```go
r.Get("/gmail/new", h.gmailForm)
r.Post("/gmail", h.gmailCreate)
```

Add handler methods:
```go
func (h *Handlers) gmailForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	render(w, "layout", nil)
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
	render(w, "gmail-row", msg)
}
```

Add import for `strings` at top.

**Step 3: Add store method**

Add to `internal/store/gmail.go`:
```go
import (
	"encoding/base64"
	"fmt"
	"time"
)

func (s *Store) CreateGmailMessageFromForm(userID, from, subject, body string, labels []string) (*GmailMessageView, error) {
	id := fmt.Sprintf("msg_%d", time.Now().UnixNano())
	threadID := fmt.Sprintf("thr_%d", time.Now().UnixNano())

	// Create thread first
	s.db.Exec("INSERT INTO gmail_threads (id, user_id, snippet) VALUES (?, ?, ?)",
		threadID, userID, truncate(body, 100))

	// Build payload
	payload := fmt.Sprintf(`{"headers":[{"name":"From","value":"%s"},{"name":"Subject","value":"%s"}],"body":{"data":"%s"}}`,
		from, subject, base64.StdEncoding.EncodeToString([]byte(body)))

	labelJSON, _ := json.Marshal(labels)

	_, err := s.db.Exec(
		"INSERT INTO gmail_messages (id, user_id, thread_id, label_ids, snippet, internal_date, payload) VALUES (?, ?, ?, ?, ?, ?, ?)",
		id, userID, threadID, string(labelJSON), truncate(body, 100), time.Now().UnixMilli(), payload,
	)
	if err != nil {
		return nil, err
	}

	return &GmailMessageView{
		ID:       id,
		Subject:  subject,
		Snippet:  truncate(body, 100),
		LabelIDs: labels,
	}, nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
```

**Step 4: Test manually**

Go to `/admin/gmail/new`, fill form, submit. New row should appear.

**Step 5: Commit**

```bash
git add internal/admin/ internal/store/gmail.go
git commit -m "feat(admin): add Gmail create form"
```

---

## Task 5: Calendar List, Create, Delete

**Files:**
- Create: `internal/admin/templates/calendar/list.html`
- Create: `internal/admin/templates/calendar/row.html`
- Create: `internal/admin/templates/calendar/form.html`
- Modify: `internal/admin/handlers.go`
- Modify: `internal/store/calendar.go`

**Step 1: Create Calendar templates**

Create `internal/admin/templates/calendar/list.html`:
```html
{{define "content"}}
<div class="space-y-6">
    <div class="flex items-center justify-between">
        <h1 class="text-2xl font-bold text-gray-900">Calendar Events</h1>
        <a href="/admin/calendar/new" class="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700">
            + New Event
        </a>
    </div>

    <div class="bg-white rounded-lg shadow overflow-hidden">
        <table class="min-w-full divide-y divide-gray-200">
            <thead class="bg-gray-50">
                <tr>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">ID</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Summary</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Start</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">End</th>
                    <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>
                </tr>
            </thead>
            <tbody id="event-list" class="bg-white divide-y divide-gray-200">
                {{range .Events}}
                {{template "calendar-row" .}}
                {{end}}
            </tbody>
        </table>
    </div>
</div>
{{end}}
```

Create `internal/admin/templates/calendar/row.html`:
```html
{{define "calendar-row"}}
<tr id="evt-{{.ID}}">
    <td class="px-6 py-4 whitespace-nowrap text-sm font-mono text-gray-900">{{.ID}}</td>
    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{{.Summary}}</td>
    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{.StartTime}}</td>
    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{.EndTime}}</td>
    <td class="px-6 py-4 whitespace-nowrap text-right text-sm">
        <button
            hx-delete="/admin/calendar/{{.ID}}"
            hx-target="#evt-{{.ID}}"
            hx-swap="delete"
            hx-confirm="Delete this event?"
            class="text-red-600 hover:text-red-900">
            Delete
        </button>
    </td>
</tr>
{{end}}
```

Create `internal/admin/templates/calendar/form.html`:
```html
{{define "content"}}
<div class="space-y-6">
    <h1 class="text-2xl font-bold text-gray-900">New Event</h1>

    <form hx-post="/admin/calendar" hx-target="#event-list" hx-swap="beforeend" class="bg-white rounded-lg shadow p-6 space-y-4 max-w-2xl">
        <div>
            <label class="block text-sm font-medium text-gray-700">Summary</label>
            <input type="text" name="summary" required class="mt-1 block w-full rounded border-gray-300 shadow-sm px-3 py-2 border">
        </div>
        <div>
            <label class="block text-sm font-medium text-gray-700">Description</label>
            <textarea name="description" rows="2" class="mt-1 block w-full rounded border-gray-300 shadow-sm px-3 py-2 border"></textarea>
        </div>
        <div class="grid grid-cols-2 gap-4">
            <div>
                <label class="block text-sm font-medium text-gray-700">Start</label>
                <input type="datetime-local" name="start" required class="mt-1 block w-full rounded border-gray-300 shadow-sm px-3 py-2 border">
            </div>
            <div>
                <label class="block text-sm font-medium text-gray-700">End</label>
                <input type="datetime-local" name="end" required class="mt-1 block w-full rounded border-gray-300 shadow-sm px-3 py-2 border">
            </div>
        </div>
        <div class="flex gap-4">
            <button type="submit" class="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700">Create</button>
            <a href="/admin/calendar" class="px-4 py-2 bg-gray-200 text-gray-700 rounded hover:bg-gray-300">Cancel</a>
        </div>
    </form>
</div>
{{end}}
```

**Step 2: Add Calendar handlers**

Add routes in RegisterRoutes:
```go
r.Get("/calendar", h.calendarList)
r.Get("/calendar/new", h.calendarForm)
r.Post("/calendar", h.calendarCreate)
r.Delete("/calendar/{id}", h.calendarDelete)
```

Add handler methods:
```go
func (h *Handlers) calendarList(w http.ResponseWriter, r *http.Request) {
	events, err := h.store.ListAllCalendarEvents()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	render(w, "layout", map[string]any{"Events": events})
}

func (h *Handlers) calendarForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	render(w, "layout", nil)
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
	render(w, "calendar-row", evt)
}

func (h *Handlers) calendarDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.DeleteCalendarEvent(id); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(http.StatusOK)
}
```

**Step 3: Add store methods**

Add to `internal/store/calendar.go`:
```go
func (s *Store) ListAllCalendarEvents() ([]CalendarEvent, error) {
	rows, err := s.db.Query("SELECT id, calendar_id, summary, description, start_time, end_time, attendees FROM calendar_events ORDER BY start_time")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []CalendarEvent
	for rows.Next() {
		var e CalendarEvent
		if err := rows.Scan(&e.ID, &e.CalendarID, &e.Summary, &e.Description, &e.StartTime, &e.EndTime, &e.Attendees); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

func (s *Store) CreateCalendarEventFromForm(summary, description, start, end string) (*CalendarEvent, error) {
	id := fmt.Sprintf("evt_%d", time.Now().UnixNano())

	// Convert datetime-local format to ISO 8601
	startTime := start + ":00Z"
	endTime := end + ":00Z"

	_, err := s.db.Exec(
		"INSERT INTO calendar_events (id, calendar_id, summary, description, start_time, end_time, attendees) VALUES (?, ?, ?, ?, ?, ?, ?)",
		id, "primary", summary, description, startTime, endTime, "[]",
	)
	if err != nil {
		return nil, err
	}

	return &CalendarEvent{
		ID:          id,
		CalendarID:  "primary",
		Summary:     summary,
		Description: description,
		StartTime:   startTime,
		EndTime:     endTime,
		Attendees:   "[]",
	}, nil
}

func (s *Store) DeleteCalendarEvent(id string) error {
	_, err := s.db.Exec("DELETE FROM calendar_events WHERE id = ?", id)
	return err
}
```

Add imports for `fmt` and `time`.

**Step 4: Commit**

```bash
git add internal/admin/ internal/store/calendar.go
git commit -m "feat(admin): add Calendar list, create, delete"
```

---

## Task 6: People List, Create, Delete

**Files:**
- Create: `internal/admin/templates/people/list.html`
- Create: `internal/admin/templates/people/row.html`
- Create: `internal/admin/templates/people/form.html`
- Modify: `internal/admin/handlers.go`
- Modify: `internal/store/people.go`

**Step 1: Create People templates**

Create `internal/admin/templates/people/list.html`:
```html
{{define "content"}}
<div class="space-y-6">
    <div class="flex items-center justify-between">
        <h1 class="text-2xl font-bold text-gray-900">People / Contacts</h1>
        <a href="/admin/people/new" class="px-4 py-2 bg-purple-600 text-white rounded hover:bg-purple-700">
            + New Contact
        </a>
    </div>

    <div class="bg-white rounded-lg shadow overflow-hidden">
        <table class="min-w-full divide-y divide-gray-200">
            <thead class="bg-gray-50">
                <tr>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Resource Name</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Name</th>
                    <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">Email</th>
                    <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">Actions</th>
                </tr>
            </thead>
            <tbody id="people-list" class="bg-white divide-y divide-gray-200">
                {{range .People}}
                {{template "people-row" .}}
                {{end}}
            </tbody>
        </table>
    </div>
</div>
{{end}}
```

Create `internal/admin/templates/people/row.html`:
```html
{{define "people-row"}}
<tr id="person-{{.ID}}">
    <td class="px-6 py-4 whitespace-nowrap text-sm font-mono text-gray-900">{{.ResourceName}}</td>
    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{{.DisplayName}}</td>
    <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{.Email}}</td>
    <td class="px-6 py-4 whitespace-nowrap text-right text-sm">
        <button
            hx-delete="/admin/people/{{.ID}}"
            hx-target="#person-{{.ID}}"
            hx-swap="delete"
            hx-confirm="Delete this contact?"
            class="text-red-600 hover:text-red-900">
            Delete
        </button>
    </td>
</tr>
{{end}}
```

Create `internal/admin/templates/people/form.html`:
```html
{{define "content"}}
<div class="space-y-6">
    <h1 class="text-2xl font-bold text-gray-900">New Contact</h1>

    <form hx-post="/admin/people" hx-target="#people-list" hx-swap="beforeend" class="bg-white rounded-lg shadow p-6 space-y-4 max-w-2xl">
        <div>
            <label class="block text-sm font-medium text-gray-700">Display Name</label>
            <input type="text" name="name" required class="mt-1 block w-full rounded border-gray-300 shadow-sm px-3 py-2 border">
        </div>
        <div>
            <label class="block text-sm font-medium text-gray-700">Email</label>
            <input type="email" name="email" required class="mt-1 block w-full rounded border-gray-300 shadow-sm px-3 py-2 border">
        </div>
        <div class="flex gap-4">
            <button type="submit" class="px-4 py-2 bg-purple-600 text-white rounded hover:bg-purple-700">Create</button>
            <a href="/admin/people" class="px-4 py-2 bg-gray-200 text-gray-700 rounded hover:bg-gray-300">Cancel</a>
        </div>
    </form>
</div>
{{end}}
```

**Step 2: Add People handlers**

Add routes:
```go
r.Get("/people", h.peopleList)
r.Get("/people/new", h.peopleForm)
r.Post("/people", h.peopleCreate)
r.Delete("/people/{id}", h.peopleDelete)
```

Add handler methods:
```go
func (h *Handlers) peopleList(w http.ResponseWriter, r *http.Request) {
	people, err := h.store.ListAllPeople()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	render(w, "layout", map[string]any{"People": people})
}

func (h *Handlers) peopleForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	render(w, "layout", nil)
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
	render(w, "people-row", p)
}

func (h *Handlers) peopleDelete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.store.DeletePerson(id); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(http.StatusOK)
}
```

**Step 3: Add store methods**

Add to `internal/store/people.go`:
```go
type PersonView struct {
	ID           string
	ResourceName string
	DisplayName  string
	Email        string
}

func (s *Store) ListAllPeople() ([]PersonView, error) {
	rows, err := s.db.Query("SELECT resource_name, data FROM people ORDER BY resource_name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var people []PersonView
	for rows.Next() {
		var p PersonView
		var data string
		if err := rows.Scan(&p.ResourceName, &data); err != nil {
			return nil, err
		}

		// Extract ID from resource_name (people/c123 -> c123)
		p.ID = strings.TrimPrefix(p.ResourceName, "people/")

		// Parse data JSON
		var d struct {
			Names          []struct{ DisplayName string } `json:"names"`
			EmailAddresses []struct{ Value string }       `json:"emailAddresses"`
		}
		json.Unmarshal([]byte(data), &d)
		if len(d.Names) > 0 {
			p.DisplayName = d.Names[0].DisplayName
		}
		if len(d.EmailAddresses) > 0 {
			p.Email = d.EmailAddresses[0].Value
		}
		people = append(people, p)
	}
	return people, nil
}

func (s *Store) CreatePersonFromForm(userID, name, email string) (*PersonView, error) {
	id := fmt.Sprintf("c%d", time.Now().UnixNano())
	resourceName := "people/" + id

	data := fmt.Sprintf(`{"names":[{"displayName":"%s"}],"emailAddresses":[{"value":"%s"}]}`, name, email)

	_, err := s.db.Exec(
		"INSERT INTO people (resource_name, user_id, data) VALUES (?, ?, ?)",
		resourceName, userID, data,
	)
	if err != nil {
		return nil, err
	}

	return &PersonView{
		ID:           id,
		ResourceName: resourceName,
		DisplayName:  name,
		Email:        email,
	}, nil
}

func (s *Store) DeletePerson(id string) error {
	resourceName := "people/" + id
	_, err := s.db.Exec("DELETE FROM people WHERE resource_name = ?", resourceName)
	return err
}
```

Add imports for `strings`, `fmt`, `time`.

**Step 4: Commit**

```bash
git add internal/admin/ internal/store/people.go
git commit -m "feat(admin): add People list, create, delete"
```

---

## Task 7: Scenario Test for Admin UI

**Files:**
- Create: `.scratch/test_admin_ui_scenario.sh`

**Step 1: Create admin UI scenario test**

Create `.scratch/test_admin_ui_scenario.sh`:
```bash
#!/bin/bash
set -e

echo "=== Admin UI Scenario Test ==="

cd /Users/harper/Public/src/2389/ish
go build -o ish ./cmd/ish

export ISH_DB_PATH="/tmp/ish_admin_scenario_$$.db"
export ISH_PORT="19877"

cleanup() {
    kill $SERVER_PID 2>/dev/null || true
    rm -f "$ISH_DB_PATH" ish
}
trap cleanup EXIT

./ish seed --db "$ISH_DB_PATH"
./ish serve --port "$ISH_PORT" --db "$ISH_DB_PATH" &
SERVER_PID=$!

for i in {1..30}; do
    curl -s "http://localhost:$ISH_PORT/healthz" > /dev/null 2>&1 && break
    sleep 0.1
done

echo "Testing dashboard..."
DASHBOARD=$(curl -s "http://localhost:$ISH_PORT/admin/")
echo "$DASHBOARD" | grep -q "ISH Admin" || { echo "✗ Dashboard missing header"; exit 1; }
echo "$DASHBOARD" | grep -q "Gmail" || { echo "✗ Dashboard missing Gmail link"; exit 1; }
echo "✓ Dashboard loads"

echo "Testing Gmail list..."
GMAIL=$(curl -s "http://localhost:$ISH_PORT/admin/gmail")
echo "$GMAIL" | grep -q "msg_" || { echo "✗ Gmail list empty"; exit 1; }
echo "✓ Gmail list loads"

echo "Testing Calendar list..."
CAL=$(curl -s "http://localhost:$ISH_PORT/admin/calendar")
echo "$CAL" | grep -q "evt_" || { echo "✗ Calendar list empty"; exit 1; }
echo "✓ Calendar list loads"

echo "Testing People list..."
PEOPLE=$(curl -s "http://localhost:$ISH_PORT/admin/people")
echo "$PEOPLE" | grep -q "people/" || { echo "✗ People list empty"; exit 1; }
echo "✓ People list loads"

echo "Testing Gmail create..."
NEW_MSG=$(curl -s -X POST "http://localhost:$ISH_PORT/admin/gmail" \
    -d "from=test@example.com" \
    -d "subject=Test Message" \
    -d "body=Hello World" \
    -d "labels=INBOX")
echo "$NEW_MSG" | grep -q "Test Message" || { echo "✗ Gmail create failed"; exit 1; }
echo "✓ Gmail create works"

echo "Testing Calendar create..."
NEW_EVT=$(curl -s -X POST "http://localhost:$ISH_PORT/admin/calendar" \
    -d "summary=Test Event" \
    -d "description=Test" \
    -d "start=2025-12-15T10:00" \
    -d "end=2025-12-15T11:00")
echo "$NEW_EVT" | grep -q "Test Event" || { echo "✗ Calendar create failed"; exit 1; }
echo "✓ Calendar create works"

echo "Testing People create..."
NEW_PERSON=$(curl -s -X POST "http://localhost:$ISH_PORT/admin/people" \
    -d "name=Test Person" \
    -d "email=testperson@example.com")
echo "$NEW_PERSON" | grep -q "Test Person" || { echo "✗ People create failed"; exit 1; }
echo "✓ People create works"

echo ""
echo "=========================================="
echo "✓ ALL ADMIN UI SCENARIOS PASSED"
echo "=========================================="
```

**Step 2: Make executable and test**

```bash
chmod +x .scratch/test_admin_ui_scenario.sh
./.scratch/test_admin_ui_scenario.sh
```

**Step 3: Commit (just verify it works, don't commit .scratch)**

No commit needed - .scratch is gitignored.

---

## Summary

| Task | Description |
|------|-------------|
| 1 | Template infrastructure + layout |
| 2 | Dashboard page with counts |
| 3 | Gmail list and delete |
| 4 | Gmail create form |
| 5 | Calendar list, create, delete |
| 6 | People list, create, delete |
| 7 | Scenario test |

Total: 7 tasks, ~6 commits
