// ABOUTME: HTTP handlers for Anthropic scenario CRUD endpoints
// ABOUTME: Manages scenario creation, listing, and deletion for pattern-matched responses

package anthropic

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// createScenario handles POST /v1/scenarios
func (p *AnthropicPlugin) createScenario(w http.ResponseWriter, r *http.Request) {
	if p.store == nil {
		writeError(w, http.StatusInternalServerError, "Plugin not initialized")
		return
	}

	var scenario Scenario
	if err := json.NewDecoder(r.Body).Decode(&scenario); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if scenario.ResponseType != "text" && scenario.ResponseType != "tool_use" {
		writeError(w, http.StatusBadRequest, "response_type must be 'text' or 'tool_use'")
		return
	}
	if scenario.ResponseType == "text" && scenario.ResponseText == "" {
		writeError(w, http.StatusBadRequest, "response_text is required for text scenarios")
		return
	}
	if scenario.ResponseType == "tool_use" && scenario.ToolName == "" {
		writeError(w, http.StatusBadRequest, "tool_name is required for tool_use scenarios")
		return
	}

	if err := p.store.CreateScenario(&scenario); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create scenario")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(scenario)
}

// listScenarios handles GET /v1/scenarios
func (p *AnthropicPlugin) listScenarios(w http.ResponseWriter, r *http.Request) {
	if p.store == nil {
		writeError(w, http.StatusInternalServerError, "Plugin not initialized")
		return
	}

	scenarios, err := p.store.ListScenarios()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list scenarios")
		return
	}

	writeJSON(w, scenarios)
}

// deleteScenarioHandler handles DELETE /v1/scenarios/{id}
func (p *AnthropicPlugin) deleteScenarioHandler(w http.ResponseWriter, r *http.Request) {
	if p.store == nil {
		writeError(w, http.StatusInternalServerError, "Plugin not initialized")
		return
	}

	id := chi.URLParam(r, "id")

	if err := p.store.DeleteScenario(id); err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete scenario")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"type":    "error",
		"message": message,
	})
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
