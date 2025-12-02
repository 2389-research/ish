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
