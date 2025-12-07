// ABOUTME: Optional DataProvider interface for exposing plugin data to admin UI
// ABOUTME: Plugins implement this to enable admin viewing of their resources

package core

import "context"

// DataProvider is an optional interface that plugins can implement
// to expose their data to the admin UI in a standardized way
type DataProvider interface {
	Plugin
	ListResources(ctx context.Context, resourceSlug string, opts ListOptions) ([]map[string]interface{}, error)
	GetResource(ctx context.Context, resourceSlug string, id string) (map[string]interface{}, error)
}

// ListOptions provides pagination and filtering options for listing resources
type ListOptions struct {
	Limit  int
	Offset int
}
