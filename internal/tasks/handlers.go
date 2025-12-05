// ABOUTME: HTTP handlers for Tasks API endpoints.
// ABOUTME: Implements task list and task operations with Google-compatible responses.

package tasks

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

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
	r.Route("/tasks/v1/lists/{tasklist}", func(r chi.Router) {
		r.Get("/tasks", h.listTasks)
		r.Post("/tasks", h.createTask)
		r.Get("/tasks/{task}", h.getTask)
		r.Put("/tasks/{task}", h.updateTask)
		r.Patch("/tasks/{task}", h.updateTask) // PATCH also calls updateTask
		r.Delete("/tasks/{task}", h.deleteTask)
	})
}

func (h *Handlers) listTasks(w http.ResponseWriter, r *http.Request) {
	listID := chi.URLParam(r, "tasklist")

	showCompleted := true
	if sc := r.URL.Query().Get("showCompleted"); sc == "false" {
		showCompleted = false
	}

	maxResults := int64(100)
	if mr := r.URL.Query().Get("maxResults"); mr != "" {
		if v, err := strconv.ParseInt(mr, 10, 64); err == nil && v > 0 {
			maxResults = v
		}
	}

	tasks, err := h.store.ListTasks(listID, showCompleted, maxResults)
	if err != nil {
		writeError(w, 500, "Internal error", "INTERNAL")
		return
	}

	// Convert to response format
	items := make([]map[string]any, len(tasks))
	for i, t := range tasks {
		item := map[string]any{
			"kind":    "tasks#task",
			"id":      t.ID,
			"title":   t.Title,
			"updated": t.UpdatedAt,
			"status":  t.Status,
		}

		if t.Notes != "" {
			item["notes"] = t.Notes
		}
		if t.Due != "" {
			item["due"] = t.Due
		}
		if t.Completed != "" {
			item["completed"] = t.Completed
		}

		items[i] = item
	}

	resp := map[string]any{
		"kind":  "tasks#tasks",
		"items": items,
	}

	writeJSON(w, resp)
}

func (h *Handlers) createTask(w http.ResponseWriter, r *http.Request) {
	listID := chi.URLParam(r, "tasklist")

	var req struct {
		Title string `json:"title"`
		Notes string `json:"notes"`
		Due   string `json:"due"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "Invalid request body", "INVALID_REQUEST")
		return
	}

	if req.Title == "" {
		writeError(w, 400, "Missing required field: title", "INVALID_REQUEST")
		return
	}

	task := &store.Task{
		ListID: listID,
		Title:  req.Title,
		Notes:  req.Notes,
		Due:    req.Due,
		Status: "needsAction",
	}

	created, err := h.store.CreateTask(task)
	if err != nil {
		writeError(w, 500, "Failed to create task", "INTERNAL")
		return
	}

	resp := map[string]any{
		"kind":    "tasks#task",
		"id":      created.ID,
		"title":   created.Title,
		"updated": created.UpdatedAt,
		"status":  created.Status,
	}

	if created.Notes != "" {
		resp["notes"] = created.Notes
	}
	if created.Due != "" {
		resp["due"] = created.Due
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, resp)
}

func (h *Handlers) getTask(w http.ResponseWriter, r *http.Request) {
	listID := chi.URLParam(r, "tasklist")
	taskID := chi.URLParam(r, "task")

	task, err := h.store.GetTask(listID, taskID)
	if err != nil {
		writeError(w, 404, "Task not found", "NOT_FOUND")
		return
	}

	resp := map[string]any{
		"kind":    "tasks#task",
		"id":      task.ID,
		"title":   task.Title,
		"updated": task.UpdatedAt,
		"status":  task.Status,
	}

	if task.Notes != "" {
		resp["notes"] = task.Notes
	}
	if task.Due != "" {
		resp["due"] = task.Due
	}
	if task.Completed != "" {
		resp["completed"] = task.Completed
	}

	writeJSON(w, resp)
}

func (h *Handlers) updateTask(w http.ResponseWriter, r *http.Request) {
	listID := chi.URLParam(r, "tasklist")
	taskID := chi.URLParam(r, "task")

	// Get existing task
	existing, err := h.store.GetTask(listID, taskID)
	if err != nil {
		writeError(w, 404, "Task not found", "NOT_FOUND")
		return
	}

	var req struct {
		Title  *string `json:"title"`
		Notes  *string `json:"notes"`
		Due    *string `json:"due"`
		Status *string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "Invalid request body", "INVALID_REQUEST")
		return
	}

	// Update fields if provided
	if req.Title != nil {
		existing.Title = *req.Title
	}
	if req.Notes != nil {
		existing.Notes = *req.Notes
	}
	if req.Due != nil {
		existing.Due = *req.Due
	}
	if req.Status != nil {
		existing.Status = *req.Status
		// If marking as completed, set completed timestamp
		if *req.Status == "completed" && existing.Completed == "" {
			existing.Completed = time.Now().UTC().Format(time.RFC3339)
		} else if *req.Status != "completed" {
			existing.Completed = ""
		}
	}

	updated, err := h.store.UpdateTask(existing)
	if err != nil {
		writeError(w, 500, "Failed to update task", "INTERNAL")
		return
	}

	resp := map[string]any{
		"kind":    "tasks#task",
		"id":      updated.ID,
		"title":   updated.Title,
		"updated": updated.UpdatedAt,
		"status":  updated.Status,
	}

	if updated.Notes != "" {
		resp["notes"] = updated.Notes
	}
	if updated.Due != "" {
		resp["due"] = updated.Due
	}
	if updated.Completed != "" {
		resp["completed"] = updated.Completed
	}

	writeJSON(w, resp)
}

func (h *Handlers) deleteTask(w http.ResponseWriter, r *http.Request) {
	listID := chi.URLParam(r, "tasklist")
	taskID := chi.URLParam(r, "task")

	err := h.store.DeleteTask(listID, taskID)
	if err != nil {
		writeError(w, 404, "Task not found", "NOT_FOUND")
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
