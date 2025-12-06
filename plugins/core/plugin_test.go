// ABOUTME: Tests for plugin interface contracts and mock plugin implementation.
// ABOUTME: Validates that mock plugin correctly implements all Plugin interface methods.

package core

import (
	"context"
	"testing"

	"github.com/go-chi/chi/v5"
)

// fullMockPlugin implements the Plugin interface with more comprehensive behavior
type fullMockPlugin struct {
	name          string
	healthStatus  HealthStatus
	tokenValid    bool
	routesCount   int
	authCount     int
	schema        PluginSchema
	seedResult    SeedData
	seedError     error
}

func (m *fullMockPlugin) Name() string {
	return m.name
}

func (m *fullMockPlugin) Health() HealthStatus {
	return m.healthStatus
}

func (m *fullMockPlugin) RegisterRoutes(r chi.Router) {
	m.routesCount++
}

func (m *fullMockPlugin) RegisterAuth(r chi.Router) {
	m.authCount++
}

func (m *fullMockPlugin) Schema() PluginSchema {
	return m.schema
}

func (m *fullMockPlugin) Seed(ctx context.Context, size string) (SeedData, error) {
	return m.seedResult, m.seedError
}

func (m *fullMockPlugin) ValidateToken(token string) bool {
	return m.tokenValid
}

func TestPluginInterfaceName(t *testing.T) {
	plugin := &fullMockPlugin{name: "test-plugin"}

	if plugin.Name() != "test-plugin" {
		t.Errorf("expected Name() to return 'test-plugin', got %q", plugin.Name())
	}
}

func TestPluginInterfaceHealth(t *testing.T) {
	tests := []struct {
		name     string
		status   HealthStatus
		expected string
	}{
		{
			name:     "healthy status",
			status:   HealthStatus{Status: "healthy", Message: "All systems operational"},
			expected: "healthy",
		},
		{
			name:     "degraded status",
			status:   HealthStatus{Status: "degraded", Message: "Partial outage"},
			expected: "degraded",
		},
		{
			name:     "unavailable status",
			status:   HealthStatus{Status: "unavailable", Message: "Service down"},
			expected: "unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := &fullMockPlugin{healthStatus: tt.status}
			health := plugin.Health()

			if health.Status != tt.expected {
				t.Errorf("expected status %q, got %q", tt.expected, health.Status)
			}
		})
	}
}

func TestPluginInterfaceRegisterRoutes(t *testing.T) {
	plugin := &fullMockPlugin{}
	router := chi.NewRouter()

	plugin.RegisterRoutes(router)

	if plugin.routesCount != 1 {
		t.Errorf("expected RegisterRoutes to be called once, was called %d times", plugin.routesCount)
	}
}

func TestPluginInterfaceRegisterAuth(t *testing.T) {
	plugin := &fullMockPlugin{}
	router := chi.NewRouter()

	plugin.RegisterAuth(router)

	if plugin.authCount != 1 {
		t.Errorf("expected RegisterAuth to be called once, was called %d times", plugin.authCount)
	}
}

func TestPluginInterfaceSchema(t *testing.T) {
	schema := PluginSchema{
		Resources: []ResourceSchema{
			{
				Name: "Messages",
				Slug: "messages",
				Fields: []FieldSchema{
					{
						Name:     "subject",
						Type:     "string",
						Display:  "Subject",
						Required: true,
						Editable: true,
					},
				},
				Actions: []ActionSchema{
					{
						Name:       "send",
						HTTPMethod: "POST",
						Endpoint:   "/messages/{id}/send",
						Confirm:    true,
					},
				},
				ListColumns: []string{"subject", "from", "date"},
			},
		},
	}

	plugin := &fullMockPlugin{schema: schema}
	result := plugin.Schema()

	if len(result.Resources) != 1 {
		t.Errorf("expected 1 resource in schema, got %d", len(result.Resources))
	}

	if result.Resources[0].Name != "Messages" {
		t.Errorf("expected resource name 'Messages', got %q", result.Resources[0].Name)
	}
}

func TestPluginInterfaceSeed(t *testing.T) {
	tests := []struct {
		name       string
		size       string
		seedData   SeedData
		seedError  error
		wantErr    bool
	}{
		{
			name: "successful seed",
			size: "small",
			seedData: SeedData{
				Summary: "Generated 10 messages",
				Records: map[string]int{"messages": 10},
			},
			seedError: nil,
			wantErr:   false,
		},
		{
			name:      "seed with error",
			size:      "large",
			seedData:  SeedData{},
			seedError: context.Canceled,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := &fullMockPlugin{
				seedResult: tt.seedData,
				seedError:  tt.seedError,
			}

			ctx := context.Background()
			result, err := plugin.Seed(ctx, tt.size)

			if (err != nil) != tt.wantErr {
				t.Errorf("Seed() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result.Summary != tt.seedData.Summary {
				t.Errorf("expected summary %q, got %q", tt.seedData.Summary, result.Summary)
			}
		})
	}
}

func TestPluginInterfaceValidateToken(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		tokenValid bool
		expected   bool
	}{
		{
			name:       "valid token",
			token:      "valid-token",
			tokenValid: true,
			expected:   true,
		},
		{
			name:       "invalid token",
			token:      "invalid-token",
			tokenValid: false,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := &fullMockPlugin{tokenValid: tt.tokenValid}
			result := plugin.ValidateToken(tt.token)

			if result != tt.expected {
				t.Errorf("ValidateToken() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestHealthStatusStruct(t *testing.T) {
	health := HealthStatus{
		Status:  "healthy",
		Message: "All systems go",
	}

	if health.Status != "healthy" {
		t.Errorf("expected Status 'healthy', got %q", health.Status)
	}

	if health.Message != "All systems go" {
		t.Errorf("expected Message 'All systems go', got %q", health.Message)
	}
}

func TestSeedDataStruct(t *testing.T) {
	seedData := SeedData{
		Summary: "Generated test data",
		Records: map[string]int{
			"messages": 50,
			"events":   25,
		},
	}

	if seedData.Summary != "Generated test data" {
		t.Errorf("expected Summary 'Generated test data', got %q", seedData.Summary)
	}

	if seedData.Records["messages"] != 50 {
		t.Errorf("expected 50 messages, got %d", seedData.Records["messages"])
	}

	if seedData.Records["events"] != 25 {
		t.Errorf("expected 25 events, got %d", seedData.Records["events"])
	}
}

func TestPluginInterfaceImplementation(t *testing.T) {
	// This test verifies that our mock implements the Plugin interface
	var _ Plugin = &fullMockPlugin{}
	var _ Plugin = &mockPlugin{}
}
