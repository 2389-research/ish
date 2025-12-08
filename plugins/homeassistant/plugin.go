// ABOUTME: Home Assistant plugin implementation for ISH
// ABOUTME: Provides REST API compatibility for Home Assistant states, services, and events
package homeassistant

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/2389/ish/plugins/core"
	"github.com/go-chi/chi/v5"
)

type contextKey string

const instanceKey contextKey = "homeassistant_instance"

func init() {
	core.Register(&HomeAssistantPlugin{})
}

func setInstanceInContext(ctx context.Context, instance *Instance) context.Context {
	return context.WithValue(ctx, instanceKey, instance)
}

func getInstanceFromContext(ctx context.Context) (*Instance, bool) {
	instance, ok := ctx.Value(instanceKey).(*Instance)
	return instance, ok
}

type HomeAssistantPlugin struct {
	store *Store
}

func (p *HomeAssistantPlugin) Name() string {
	return "homeassistant"
}

func (p *HomeAssistantPlugin) Health() core.HealthStatus {
	return core.HealthStatus{
		Status:  "healthy",
		Message: "Home Assistant plugin operational",
	}
}

func (p *HomeAssistantPlugin) RegisterRoutes(r chi.Router) {
	// Home Assistant API endpoints
	r.Get("/api/", p.requireAuth(p.handleAPIRoot))
	r.Get("/api/states", p.requireAuth(p.handleGetAllStates))
	r.Get("/api/states/{entity_id}", p.requireAuth(p.handleGetState))
	r.Post("/api/states/{entity_id}", p.requireAuth(p.handleSetState))
	r.Post("/api/services/{domain}/{service}", p.requireAuth(p.handleCallService))
	r.Post("/api/events/{event_type}", p.requireAuth(p.handleFireEvent))

	// WebSocket API endpoint
	r.Get("/api/websocket", p.handleWebSocket)
}

func (p *HomeAssistantPlugin) RegisterAuth(r chi.Router) {
	// Home Assistant uses Bearer token authentication
}

func (p *HomeAssistantPlugin) Schema() core.PluginSchema {
	return core.PluginSchema{
		Resources: GetResourceSchemas(),
	}
}

func (p *HomeAssistantPlugin) ValidateToken(token string) bool {
	// Accept any Bearer token for now
	return strings.HasPrefix(token, "token_") || token != ""
}

func (p *HomeAssistantPlugin) SetDB(db *sql.DB) error {
	store, err := NewStore(db)
	if err != nil {
		return err
	}
	p.store = store
	return nil
}

// extractToken extracts the token from the Authorization header
func extractToken(authHeader string) (string, bool) {
	const prefix = "Bearer "
	if !strings.HasPrefix(authHeader, prefix) {
		return "", false
	}
	return authHeader[len(prefix):], true
}

// requireAuth middleware validates Home Assistant token
func (p *HomeAssistantPlugin) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token, ok := extractToken(authHeader)
		if !ok {
			http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
			return
		}

		// Get instance by token
		instance, err := p.store.GetInstanceByToken(token)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx := setInstanceInContext(r.Context(), instance)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// handleAPIRoot returns API info (compatible with Home Assistant /api/)
func (p *HomeAssistantPlugin) handleAPIRoot(w http.ResponseWriter, r *http.Request) {
	instance, ok := getInstanceFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	response := map[string]interface{}{
		"message":  "API running.",
		"instance": instance.Name,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// handleGetAllStates returns all entity states
func (p *HomeAssistantPlugin) handleGetAllStates(w http.ResponseWriter, r *http.Request) {
	instance, ok := getInstanceFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	states, err := p.store.ListAllStates(100, 0)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Filter to this instance
	var instanceStates []State
	for _, state := range states {
		if state.InstanceID == instance.ID {
			instanceStates = append(instanceStates, state)
		}
	}

	// Convert to Home Assistant format
	response := make([]map[string]interface{}, 0)
	for _, state := range instanceStates {
		var attributes map[string]interface{}
		if state.Attributes != "" {
			json.Unmarshal([]byte(state.Attributes), &attributes)
		}

		response = append(response, map[string]interface{}{
			"entity_id":    state.EntityID,
			"state":        state.State,
			"attributes":   attributes,
			"last_changed": state.LastChanged.Format(time.RFC3339),
			"last_updated": state.LastUpdated.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding states response: %v", err)
	}
}

// handleGetState returns a single entity state
func (p *HomeAssistantPlugin) handleGetState(w http.ResponseWriter, r *http.Request) {
	instance, ok := getInstanceFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	entityID := chi.URLParam(r, "entity_id")

	// Get latest state for this entity
	var state State
	err := p.store.db.QueryRow(`
		SELECT id, instance_id, entity_id, state, attributes, last_changed, last_updated, created_at
		FROM homeassistant_states
		WHERE instance_id = ? AND entity_id = ?
		ORDER BY last_updated DESC
		LIMIT 1
	`, instance.ID, entityID).Scan(&state.ID, &state.InstanceID, &state.EntityID, &state.State, &state.Attributes, &state.LastChanged, &state.LastUpdated, &state.CreatedAt)

	if err == sql.ErrNoRows {
		http.Error(w, "Entity not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var attributes map[string]interface{}
	if state.Attributes != "" {
		json.Unmarshal([]byte(state.Attributes), &attributes)
	}

	response := map[string]interface{}{
		"entity_id":    state.EntityID,
		"state":        state.State,
		"attributes":   attributes,
		"last_changed": state.LastChanged.Format(time.RFC3339),
		"last_updated": state.LastUpdated.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding state response: %v", err)
	}
}

// handleSetState sets or updates an entity state
func (p *HomeAssistantPlugin) handleSetState(w http.ResponseWriter, r *http.Request) {
	instance, ok := getInstanceFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	entityID := chi.URLParam(r, "entity_id")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var req struct {
		State      string                 `json:"state"`
		Attributes map[string]interface{} `json:"attributes"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Convert attributes to JSON
	attributesJSON, err := json.Marshal(req.Attributes)
	if err != nil {
		http.Error(w, "Invalid attributes", http.StatusBadRequest)
		return
	}

	// Extract domain from entity_id (e.g., "light.living_room" -> "light")
	parts := strings.SplitN(entityID, ".", 2)
	domain := "unknown"
	if len(parts) == 2 {
		domain = parts[0]
	}

	// Create or update entity
	err = p.store.CreateOrUpdateEntity(instance.ID, entityID, "", domain, "")
	if err != nil {
		log.Printf("Error creating/updating entity: %v", err)
	}

	// Record state
	now := time.Now()
	err = p.store.RecordState(instance.ID, entityID, req.State, string(attributesJSON), now, now)
	if err != nil {
		http.Error(w, "Failed to record state", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"entity_id":    entityID,
		"state":        req.State,
		"attributes":   req.Attributes,
		"last_changed": now.Format(time.RFC3339),
		"last_updated": now.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding set state response: %v", err)
	}
}

// handleCallService calls a Home Assistant service
func (p *HomeAssistantPlugin) handleCallService(w http.ResponseWriter, r *http.Request) {
	instance, ok := getInstanceFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	domain := chi.URLParam(r, "domain")
	service := chi.URLParam(r, "service")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var req struct {
		EntityID    string                 `json:"entity_id"`
		ServiceData map[string]interface{} `json:"service_data"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Convert service data to JSON
	serviceDataJSON, err := json.Marshal(req.ServiceData)
	if err != nil {
		http.Error(w, "Invalid service data", http.StatusBadRequest)
		return
	}

	// Record service call
	now := time.Now()
	err = p.store.RecordServiceCall(instance.ID, domain, service, string(serviceDataJSON), req.EntityID, "success", now)
	if err != nil {
		log.Printf("Error recording service call: %v", err)
		http.Error(w, "Failed to record service call", http.StatusInternalServerError)
		return
	}

	response := []map[string]interface{}{
		{
			"entity_id": req.EntityID,
			"success":   true,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding service call response: %v", err)
	}
}

// handleFireEvent fires a Home Assistant event
func (p *HomeAssistantPlugin) handleFireEvent(w http.ResponseWriter, r *http.Request) {
	instance, ok := getInstanceFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	eventType := chi.URLParam(r, "event_type")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	var eventData map[string]interface{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &eventData); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
	}

	// For now, just log events (could add events table later)
	log.Printf("Event fired on instance %s: %s with data: %v", instance.Name, eventType, eventData)

	response := map[string]interface{}{
		"message": "Event " + eventType + " fired.",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding event response: %v", err)
	}
}

// ListResources implements core.DataProvider to expose data to admin UI
func (p *HomeAssistantPlugin) ListResources(ctx context.Context, slug string, opts core.ListOptions) ([]map[string]interface{}, error) {
	switch slug {
	case "instances":
		instances, err := p.store.ListAllInstances(opts.Limit, opts.Offset)
		if err != nil {
			return nil, err
		}
		return convertInstancesToMaps(instances), nil
	case "entities":
		entities, err := p.store.ListAllEntities(opts.Limit, opts.Offset)
		if err != nil {
			return nil, err
		}
		return convertEntitiesToMaps(entities), nil
	case "states":
		states, err := p.store.ListAllStates(opts.Limit, opts.Offset)
		if err != nil {
			return nil, err
		}
		return convertStatesToMaps(states), nil
	case "service_calls":
		calls, err := p.store.ListAllServiceCalls(opts.Limit, opts.Offset)
		if err != nil {
			return nil, err
		}
		return convertServiceCallsToMaps(calls), nil
	default:
		return nil, fmt.Errorf("unknown resource: %s", slug)
	}
}

// GetResource implements core.DataProvider
func (p *HomeAssistantPlugin) GetResource(ctx context.Context, slug string, id string) (map[string]interface{}, error) {
	// For now, return error - can implement later if needed
	return nil, fmt.Errorf("GetResource not implemented for homeassistant")
}

// Conversion helpers
func convertInstancesToMaps(instances []Instance) []map[string]interface{} {
	result := make([]map[string]interface{}, len(instances))
	for i, inst := range instances {
		result[i] = convertInstanceToMap(inst)
	}
	return result
}

func convertInstanceToMap(inst Instance) map[string]interface{} {
	return map[string]interface{}{
		"id":         inst.ID,
		"name":       inst.Name,
		"url":        inst.URL,
		"created_at": inst.CreatedAt.Format("2006-01-02T15:04:05Z"),
		"updated_at": inst.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func convertEntitiesToMaps(entities []Entity) []map[string]interface{} {
	result := make([]map[string]interface{}, len(entities))
	for i, ent := range entities {
		result[i] = convertEntityToMap(ent)
	}
	return result
}

func convertEntityToMap(ent Entity) map[string]interface{} {
	return map[string]interface{}{
		"id":            ent.ID,
		"instance_id":   ent.InstanceID,
		"entity_id":     ent.EntityID,
		"friendly_name": ent.FriendlyName,
		"domain":        ent.Domain,
		"platform":      ent.Platform,
		"created_at":    ent.CreatedAt.Format("2006-01-02T15:04:05Z"),
		"updated_at":    ent.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func convertStatesToMaps(states []State) []map[string]interface{} {
	result := make([]map[string]interface{}, len(states))
	for i, state := range states {
		result[i] = convertStateToMap(state)
	}
	return result
}

func convertStateToMap(state State) map[string]interface{} {
	return map[string]interface{}{
		"id":           state.ID,
		"instance_id":  state.InstanceID,
		"entity_id":    state.EntityID,
		"state":        state.State,
		"attributes":   state.Attributes,
		"last_changed": state.LastChanged.Format("2006-01-02T15:04:05Z"),
		"last_updated": state.LastUpdated.Format("2006-01-02T15:04:05Z"),
		"created_at":   state.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func convertServiceCallsToMaps(calls []ServiceCall) []map[string]interface{} {
	result := make([]map[string]interface{}, len(calls))
	for i, call := range calls {
		result[i] = convertServiceCallToMap(call)
	}
	return result
}

func convertServiceCallToMap(call ServiceCall) map[string]interface{} {
	return map[string]interface{}{
		"id":           call.ID,
		"instance_id":  call.InstanceID,
		"domain":       call.Domain,
		"service":      call.Service,
		"entity_id":    call.EntityID,
		"service_data": call.ServiceData,
		"status":       call.Status,
		"called_at":    call.CalledAt.Format("2006-01-02T15:04:05Z"),
		"created_at":   call.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
