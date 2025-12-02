// ABOUTME: HTTP handlers for admin UI pages.
// ABOUTME: Serves dashboard and CRUD pages for Gmail, Calendar, People.

package admin

import (
	"net/http"
	"strings"

	"github.com/2389/ish/internal/store"
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
		r.Get("/gmail", h.gmailList)
		r.Get("/gmail/new", h.gmailForm)
		r.Post("/gmail", h.gmailCreate)
		r.Delete("/gmail/{id}", h.gmailDelete)
		r.Get("/calendar", h.calendarList)
		r.Get("/calendar/new", h.calendarForm)
		r.Post("/calendar", h.calendarCreate)
		r.Delete("/calendar/{id}", h.calendarDelete)
		r.Get("/people", h.peopleList)
		r.Get("/people/new", h.peopleForm)
		r.Post("/people", h.peopleCreate)
		r.Delete("/people/{id}", h.peopleDelete)
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
