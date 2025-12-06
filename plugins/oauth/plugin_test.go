// ABOUTME: Tests for OAuth plugin interface implementation.
// ABOUTME: Verifies plugin registration, health checks, and metadata.

package oauth

import (
	"context"
	"testing"

	"github.com/2389/ish/internal/store"
	"github.com/2389/ish/plugins/core"
	"github.com/go-chi/chi/v5"
)

func TestOAuthPluginImplementsInterface(t *testing.T) {
	var _ core.Plugin = (*OAuthPlugin)(nil)
}

func TestOAuthPluginName(t *testing.T) {
	p := &OAuthPlugin{}
	if got := p.Name(); got != "oauth" {
		t.Errorf("Name() = %q, want %q", got, "oauth")
	}
}

func TestOAuthPluginHealth(t *testing.T) {
	p := &OAuthPlugin{}
	health := p.Health()

	if health.Status != "healthy" {
		t.Errorf("Health().Status = %q, want %q", health.Status, "healthy")
	}

	if health.Message == "" {
		t.Error("Health().Message should not be empty")
	}
}

func TestOAuthPluginRegisterAuth(t *testing.T) {
	p := &OAuthPlugin{}
	r := chi.NewRouter()

	// Should not panic
	p.RegisterAuth(r)
}

func TestOAuthPluginRegisterRoutes(t *testing.T) {
	p := &OAuthPlugin{}
	r := chi.NewRouter()

	// OAuth plugin has no API routes, only auth routes
	p.RegisterRoutes(r)
}

func TestOAuthPluginSchema(t *testing.T) {
	p := &OAuthPlugin{}
	schema := p.Schema()

	if len(schema.Resources) == 0 {
		t.Error("Schema should have at least one resource")
	}

	// Find the Tokens resource
	var tokensResource *core.ResourceSchema
	for i := range schema.Resources {
		if schema.Resources[i].Slug == "tokens" {
			tokensResource = &schema.Resources[i]
			break
		}
	}

	if tokensResource == nil {
		t.Fatal("Schema should have a 'tokens' resource")
	}

	if tokensResource.Name != "OAuth Tokens" {
		t.Errorf("Tokens resource Name = %q, want %q", tokensResource.Name, "OAuth Tokens")
	}

	// Check for expected fields
	expectedFields := []string{"token", "plugin_name", "user_id", "scopes", "expires_at", "revoked", "created_at"}
	if len(tokensResource.Fields) != len(expectedFields) {
		t.Errorf("Expected %d fields, got %d", len(expectedFields), len(tokensResource.Fields))
	}

	// Check for revoke action
	hasRevokeAction := false
	for _, action := range tokensResource.Actions {
		if action.Name == "revoke" {
			hasRevokeAction = true
			if action.HTTPMethod != "POST" {
				t.Errorf("Revoke action HTTPMethod = %q, want %q", action.HTTPMethod, "POST")
			}
			if !action.Confirm {
				t.Error("Revoke action should require confirmation")
			}
		}
	}
	if !hasRevokeAction {
		t.Error("Schema should have a 'revoke' action")
	}

	// Check list columns
	expectedColumns := []string{"token", "plugin_name", "user_id", "revoked", "created_at"}
	if len(tokensResource.ListColumns) != len(expectedColumns) {
		t.Errorf("Expected %d list columns, got %d", len(expectedColumns), len(tokensResource.ListColumns))
	}
}

func TestOAuthPluginSeed(t *testing.T) {
	p := &OAuthPlugin{}
	ctx := context.Background()

	// OAuth plugin should not seed data
	data, err := p.Seed(ctx, "small")
	if err != nil {
		t.Errorf("Seed() error = %v, want nil", err)
	}

	if data.Summary == "" {
		t.Error("Seed() should return a summary message")
	}
}

func TestOAuthPluginValidateToken(t *testing.T) {
	// Create a test database
	s, cleanup := createTestStore(t)
	defer cleanup()

	p := &OAuthPlugin{store: s}

	// Test non-existent token
	if p.ValidateToken("invalid_token") {
		t.Error("ValidateToken() should return false for non-existent token")
	}

	// Store a valid token
	token := &store.OAuthToken{
		Token:      "valid_token_123",
		PluginName: "google",
		UserID:     "test_user",
		Scopes:     "email profile",
		Revoked:    false,
	}
	if err := s.StoreToken(token); err != nil {
		t.Fatalf("Failed to store token: %v", err)
	}

	// Test valid token
	if !p.ValidateToken("valid_token_123") {
		t.Error("ValidateToken() should return true for valid token")
	}

	// Revoke the token
	if err := s.RevokeToken("valid_token_123"); err != nil {
		t.Fatalf("Failed to revoke token: %v", err)
	}

	// Test revoked token
	if p.ValidateToken("valid_token_123") {
		t.Error("ValidateToken() should return false for revoked token")
	}
}

func TestOAuthPluginSetStore(t *testing.T) {
	s, cleanup := createTestStore(t)
	defer cleanup()

	p := &OAuthPlugin{}
	p.SetStore(s)

	if p.store != s {
		t.Error("SetStore() should set the store field")
	}
}

// createTestStore creates a temporary test database
func createTestStore(t *testing.T) (*store.Store, func()) {
	t.Helper()
	tmpDB := t.TempDir() + "/test.db"
	s, err := store.New(tmpDB)
	if err != nil {
		t.Fatalf("Failed to create test store: %v", err)
	}
	return s, func() { s.Close() }
}
