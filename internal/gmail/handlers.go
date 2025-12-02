// ABOUTME: HTTP handlers for Gmail API endpoints.
// ABOUTME: Implements message listing and retrieval with Google-compatible responses.

package gmail

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/2389/ish/internal/auth"
	"github.com/2389/ish/internal/store"
)

type Handlers struct {
	store *store.Store
}

func NewHandlers(s *store.Store) *Handlers {
	return &Handlers{store: s}
}

func (h *Handlers) RegisterRoutes(r chi.Router) {
	r.Route("/gmail/v1/users/{userId}", func(r chi.Router) {
		r.Get("/profile", h.getProfile)
		r.Get("/messages", h.listMessages)
		r.Get("/messages/{messageId}", h.getMessage)
		r.Get("/messages/{messageId}/attachments/{attachmentId}", h.getAttachment)
		r.Get("/history", h.listHistory)
	})
}

func (h *Handlers) listMessages(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	if userID == "me" {
		userID = auth.UserFromContext(r.Context())
	}

	maxResults := 100
	if mr := r.URL.Query().Get("maxResults"); mr != "" {
		if v, err := strconv.Atoi(mr); err == nil && v > 0 {
			maxResults = v
		}
	}
	pageToken := r.URL.Query().Get("pageToken")
	query := r.URL.Query().Get("q")

	messages, nextToken, err := h.store.ListGmailMessages(userID, maxResults, pageToken, query)
	if err != nil {
		writeError(w, 500, "Internal error", "INTERNAL")
		return
	}

	// Convert to response format
	msgList := make([]map[string]any, len(messages))
	for i, m := range messages {
		msgList[i] = map[string]any{
			"id":       m.ID,
			"threadId": m.ThreadID,
		}
	}

	resp := map[string]any{
		"messages":           msgList,
		"resultSizeEstimate": len(messages),
	}
	if nextToken != "" {
		resp["nextPageToken"] = nextToken
	}

	writeJSON(w, resp)
}

func (h *Handlers) getMessage(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	if userID == "me" {
		userID = auth.UserFromContext(r.Context())
	}
	messageID := chi.URLParam(r, "messageId")

	msg, err := h.store.GetGmailMessage(userID, messageID)
	if err != nil {
		writeError(w, 404, "Message not found", "NOT_FOUND")
		return
	}

	// Parse payload JSON
	var payload map[string]any
	json.Unmarshal([]byte(msg.Payload), &payload)

	resp := map[string]any{
		"id":           msg.ID,
		"threadId":     msg.ThreadID,
		"labelIds":     msg.LabelIDs,
		"snippet":      msg.Snippet,
		"internalDate": strconv.FormatInt(msg.InternalDate, 10),
		"payload":      payload,
	}

	writeJSON(w, resp)
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

func (h *Handlers) getProfile(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	if userID == "me" {
		userID = auth.UserFromContext(r.Context())
	}

	profile, err := h.store.GetGmailProfile(userID)
	if err != nil {
		writeError(w, 500, "Internal error", "INTERNAL")
		return
	}

	resp := map[string]any{
		"emailAddress":  profile.EmailAddress,
		"messagesTotal": profile.MessagesTotal,
		"threadsTotal":  profile.ThreadsTotal,
		"historyId":     strconv.FormatInt(profile.HistoryID, 10),
	}

	writeJSON(w, resp)
}

func (h *Handlers) getAttachment(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	if userID == "me" {
		userID = auth.UserFromContext(r.Context())
	}
	messageID := chi.URLParam(r, "messageId")
	attachmentID := chi.URLParam(r, "attachmentId")

	attachment, err := h.store.GetGmailAttachment(userID, messageID, attachmentID)
	if err != nil {
		writeError(w, 404, "Attachment not found", "NOT_FOUND")
		return
	}

	resp := map[string]any{
		"attachmentId": attachment.ID,
		"size":         attachment.Size,
		"data":         attachment.Data,
	}

	writeJSON(w, resp)
}

func (h *Handlers) listHistory(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	if userID == "me" {
		userID = auth.UserFromContext(r.Context())
	}

	startHistoryID := int64(1)
	if sid := r.URL.Query().Get("startHistoryId"); sid != "" {
		if v, err := strconv.ParseInt(sid, 10, 64); err == nil {
			startHistoryID = v
		}
	}

	maxResults := 100
	if mr := r.URL.Query().Get("maxResults"); mr != "" {
		if v, err := strconv.Atoi(mr); err == nil && v > 0 {
			maxResults = v
		}
	}
	pageToken := r.URL.Query().Get("pageToken")

	entries, historyID, nextToken, err := h.store.ListGmailHistory(userID, startHistoryID, maxResults, pageToken)
	if err != nil {
		writeError(w, 500, "Internal error", "INTERNAL")
		return
	}

	// Convert entries to response format
	historyList := make([]map[string]any, len(entries))
	for i, entry := range entries {
		messagesAdded := make([]map[string]any, len(entry.MessagesAdded))
		for j, msg := range entry.MessagesAdded {
			messagesAdded[j] = map[string]any{
				"message": map[string]any{
					"id":       msg.ID,
					"threadId": msg.ThreadID,
					"labelIds": msg.LabelIDs,
				},
			}
		}
		historyList[i] = map[string]any{
			"id":            strconv.FormatInt(entry.ID, 10),
			"messagesAdded": messagesAdded,
		}
	}

	resp := map[string]any{
		"history":   historyList,
		"historyId": strconv.FormatInt(historyID, 10),
	}
	if nextToken != "" {
		resp["nextPageToken"] = nextToken
	}

	writeJSON(w, resp)
}
