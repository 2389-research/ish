// ABOUTME: Admin UI schema definitions for Anthropic plugin
// ABOUTME: Defines Scenarios and Messages resources for schema-driven UI

package anthropic

import (
	"context"
	"fmt"

	"github.com/2389/ish/plugins/core"
)

// Schema returns the admin UI schema for the Anthropic plugin
func (p *AnthropicPlugin) Schema() core.PluginSchema {
	return core.PluginSchema{
		Resources: []core.ResourceSchema{
			{
				Name:        "Scenarios",
				Slug:        "scenarios",
				ListColumns: []string{"id", "pattern", "response_type", "priority", "created_at"},
				Fields: []core.FieldSchema{
					{Name: "id", Type: "string", Display: "ID", Required: true, Editable: false},
					{Name: "pattern", Type: "string", Display: "Pattern", Required: false, Editable: true},
					{Name: "response_type", Type: "string", Display: "Response Type", Required: true, Editable: true},
					{Name: "response_text", Type: "text", Display: "Response Text", Required: false, Editable: true},
					{Name: "tool_name", Type: "string", Display: "Tool Name", Required: false, Editable: true},
					{Name: "tool_input", Type: "text", Display: "Tool Input (JSON)", Required: false, Editable: true},
					{Name: "priority", Type: "number", Display: "Priority", Required: true, Editable: true},
					{Name: "created_at", Type: "datetime", Display: "Created", Required: false, Editable: false},
				},
				Actions: []core.ActionSchema{
					{
						Name:       "delete",
						HTTPMethod: "DELETE",
						Endpoint:   "/v1/scenarios/{id}",
						Confirm:    true,
					},
				},
			},
			{
				Name:        "Messages",
				Slug:        "messages",
				ListColumns: []string{"id", "response_type", "scenario_id", "created_at"},
				Fields: []core.FieldSchema{
					{Name: "id", Type: "string", Display: "ID", Required: true, Editable: false},
					{Name: "request_body", Type: "text", Display: "Request Body", Required: false, Editable: false},
					{Name: "response_type", Type: "string", Display: "Response Type", Required: false, Editable: false},
					{Name: "scenario_id", Type: "string", Display: "Scenario ID", Required: false, Editable: false},
					{Name: "created_at", Type: "datetime", Display: "Created", Required: false, Editable: false},
				},
			},
		},
	}
}

// ListResources implements core.DataProvider to expose data to admin UI
func (p *AnthropicPlugin) ListResources(ctx context.Context, slug string, opts core.ListOptions) ([]map[string]interface{}, error) {
	switch slug {
	case "scenarios":
		scenarios, err := p.store.ListScenarios()
		if err != nil {
			return nil, err
		}
		return convertScenariosToMaps(scenarios), nil
	case "messages":
		messages, err := p.store.ListMessages(opts.Limit, opts.Offset)
		if err != nil {
			return nil, err
		}
		return convertMessagesToMaps(messages), nil
	default:
		return nil, fmt.Errorf("unknown resource: %s", slug)
	}
}

// GetResource implements core.DataProvider to fetch individual resources
func (p *AnthropicPlugin) GetResource(ctx context.Context, slug string, id string) (map[string]interface{}, error) {
	switch slug {
	case "scenarios":
		scenario, err := p.store.GetScenario(id)
		if err != nil {
			return nil, err
		}
		return convertScenarioToMap(scenario), nil
	default:
		return nil, fmt.Errorf("unknown resource: %s", slug)
	}
}

// convertScenariosToMaps converts scenario structs to maps for admin UI
func convertScenariosToMaps(scenarios []*Scenario) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(scenarios))
	for _, s := range scenarios {
		result = append(result, convertScenarioToMap(s))
	}
	return result
}

// convertScenarioToMap converts a single scenario to map
func convertScenarioToMap(s *Scenario) map[string]interface{} {
	m := map[string]interface{}{
		"id":            s.ID,
		"pattern":       s.Pattern,
		"response_type": s.ResponseType,
		"response_text": s.ResponseText,
		"priority":      s.Priority,
		"created_at":    s.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if s.ToolName != "" {
		m["tool_name"] = s.ToolName
	}
	if s.ToolInput != "" {
		m["tool_input"] = s.ToolInput
	}
	return m
}

// convertMessagesToMaps converts message structs to maps for admin UI
func convertMessagesToMaps(messages []*Message) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(messages))
	for _, msg := range messages {
		result = append(result, convertMessageToMap(msg))
	}
	return result
}

// convertMessageToMap converts a single message to map
func convertMessageToMap(msg *Message) map[string]interface{} {
	return map[string]interface{}{
		"id":            msg.ID,
		"request_body":  msg.RequestBody,
		"response_type": msg.ResponseType,
		"scenario_id":   msg.ScenarioID,
		"created_at":    msg.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
