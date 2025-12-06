// ABOUTME: Core plugin interface for ISH plugin system.
// ABOUTME: Defines contract that all plugins must implement.

package core

import (
	"context"

	"github.com/go-chi/chi/v5"
)

// Plugin defines the interface that all ISH plugins must implement
type Plugin interface {
	// Metadata
	Name() string
	Health() HealthStatus

	// HTTP Routes
	RegisterRoutes(r chi.Router)
	RegisterAuth(r chi.Router)

	// Admin UI
	Schema() PluginSchema

	// Data Generation
	Seed(ctx context.Context, size string) (SeedData, error)

	// Token Validation
	ValidateToken(token string) bool
}

// HealthStatus represents plugin health
type HealthStatus struct {
	Status  string // "healthy", "degraded", "unavailable"
	Message string
}

// SeedData represents data generation results
type SeedData struct {
	Summary string         // Human-readable summary
	Records map[string]int // Resource counts: {"messages": 50}
}
