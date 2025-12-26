// ABOUTME: Plugin admin routes that wire schema renderer to HTTP handlers.
// ABOUTME: Provides generic CRUD UI for any plugin's resources using schemas.

package admin

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"

	"github.com/2389/ish/plugins/core"
	"github.com/go-chi/chi/v5"
)

// PluginHandlers handles plugin admin UI routes
type PluginHandlers struct{}

// RegisterRoutes registers plugin admin routes
func (h *PluginHandlers) RegisterRoutes(r chi.Router) {
	// JSON API endpoints for agents to verify integrations
	r.Get("/admin/plugins/{plugin}/{resource}.json", h.PluginListJSON)
	r.Get("/admin/plugins/{plugin}/{resource}/{id}.json", h.PluginDetailJSON)

	// HTML views
	r.Route("/admin/plugins/{plugin}/{resource}", func(r chi.Router) {
		r.Get("/", h.PluginListView)
		r.Get("/new", h.PluginCreateForm)
		r.Get("/{id}", h.PluginDetailView)
		r.Get("/{id}/edit", h.PluginEditForm)
	})
}

// PluginListView renders a list of resources using the schema renderer
func (h *PluginHandlers) PluginListView(w http.ResponseWriter, r *http.Request) {
	pluginName := chi.URLParam(r, "plugin")
	resourceSlug := chi.URLParam(r, "resource")

	// Get plugin from registry
	plugin, ok := core.Get(pluginName)
	if !ok {
		http.Error(w, "Plugin not found", http.StatusNotFound)
		return
	}

	// Get schema
	schema := plugin.Schema()

	// Find resource schema
	resourceSchema := findResourceSchema(schema, resourceSlug)
	if resourceSchema == nil {
		http.Error(w, "Resource not found", http.StatusNotFound)
		return
	}

	// Check if plugin supports data fetching
	var resources []map[string]interface{}
	if dataProvider, ok := plugin.(core.DataProvider); ok {
		opts := core.ListOptions{Limit: 50, Offset: 0}
		fetchedResources, err := dataProvider.ListResources(r.Context(), resourceSlug, opts)
		if err != nil {
			log.Printf("Error fetching %s data from %s: %v", resourceSlug, pluginName, err)
			resources = []map[string]interface{}{} // fallback to empty
		} else {
			resources = fetchedResources
		}
	} else {
		log.Printf("Plugin %s does not implement DataProvider", pluginName)
		resources = []map[string]interface{}{}
	}

	// Render list view using schema renderer
	listHTML := RenderResourceList(*resourceSchema, resources)

	// Wrap in admin layout
	pageData := pluginListData{
		PluginName:   pluginName,
		ResourceName: resourceSchema.Name,
		ResourceSlug: resourceSlug,
		ListHTML:     template.HTML(listHTML),
	}

	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "plugin-list", pageData)
}

// PluginCreateForm renders a create form using the schema renderer
func (h *PluginHandlers) PluginCreateForm(w http.ResponseWriter, r *http.Request) {
	pluginName := chi.URLParam(r, "plugin")
	resourceSlug := chi.URLParam(r, "resource")

	// Get plugin from registry
	plugin, ok := core.Get(pluginName)
	if !ok {
		http.Error(w, "Plugin not found", http.StatusNotFound)
		return
	}

	// Get schema
	schema := plugin.Schema()

	// Find resource schema
	resourceSchema := findResourceSchema(schema, resourceSlug)
	if resourceSchema == nil {
		http.Error(w, "Resource not found", http.StatusNotFound)
		return
	}

	// Render form with no data (create mode)
	formHTML := RenderResourceForm(*resourceSchema, nil)

	// Wrap in admin layout
	pageData := pluginFormData{
		PluginName:   pluginName,
		ResourceName: resourceSchema.Name,
		ResourceSlug: resourceSlug,
		FormHTML:     template.HTML(formHTML),
	}

	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "plugin-form", pageData)
}

// PluginDetailView renders a detail view using the schema renderer
func (h *PluginHandlers) PluginDetailView(w http.ResponseWriter, r *http.Request) {
	pluginName := chi.URLParam(r, "plugin")
	resourceSlug := chi.URLParam(r, "resource")
	id := chi.URLParam(r, "id")

	// Get plugin from registry
	plugin, ok := core.Get(pluginName)
	if !ok {
		http.Error(w, "Plugin not found", http.StatusNotFound)
		return
	}

	// Get schema
	schema := plugin.Schema()

	// Find resource schema
	resourceSchema := findResourceSchema(schema, resourceSlug)
	if resourceSchema == nil {
		http.Error(w, "Resource not found", http.StatusNotFound)
		return
	}

	// Check if plugin supports data fetching
	var data map[string]interface{}
	if dataProvider, ok := plugin.(core.DataProvider); ok {
		fetchedData, err := dataProvider.GetResource(context.Background(), resourceSlug, id)
		if err != nil {
			log.Printf("Error fetching %s/%s from %s: %v", resourceSlug, id, pluginName, err)
			// Fallback to mock data
			data = map[string]interface{}{
				"id":      id,
				"subject": "Test Subject",
				"from":    "test@example.com",
				"body":    "Test body content",
			}
		} else {
			data = fetchedData
		}
	} else {
		log.Printf("Plugin %s does not implement DataProvider", pluginName)
		data = map[string]interface{}{
			"id":      id,
			"subject": "Test Subject",
			"from":    "test@example.com",
			"body":    "Test body content",
		}
	}

	// Render detail view using schema renderer
	detailHTML := RenderResourceDetail(*resourceSchema, data)

	// Wrap in admin layout
	pageData := pluginDetailData{
		PluginName:   pluginName,
		ResourceName: resourceSchema.Name,
		ResourceSlug: resourceSlug,
		ResourceID:   id,
		DetailHTML:   template.HTML(detailHTML),
	}

	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "plugin-detail", pageData)
}

// PluginEditForm renders an edit form using the schema renderer
func (h *PluginHandlers) PluginEditForm(w http.ResponseWriter, r *http.Request) {
	pluginName := chi.URLParam(r, "plugin")
	resourceSlug := chi.URLParam(r, "resource")
	id := chi.URLParam(r, "id")

	// Get plugin from registry
	plugin, ok := core.Get(pluginName)
	if !ok {
		http.Error(w, "Plugin not found", http.StatusNotFound)
		return
	}

	// Get schema
	schema := plugin.Schema()

	// Find resource schema
	resourceSchema := findResourceSchema(schema, resourceSlug)
	if resourceSchema == nil {
		http.Error(w, "Resource not found", http.StatusNotFound)
		return
	}

	// Check if plugin supports data fetching
	var data map[string]interface{}
	if dataProvider, ok := plugin.(core.DataProvider); ok {
		fetchedData, err := dataProvider.GetResource(context.Background(), resourceSlug, id)
		if err != nil {
			log.Printf("Error fetching %s/%s from %s: %v", resourceSlug, id, pluginName, err)
			// Fallback to mock data
			data = map[string]interface{}{
				"id":      id,
				"subject": "Test Subject",
				"from":    "test@example.com",
				"body":    "Test body content",
			}
		} else {
			data = fetchedData
		}
	} else {
		log.Printf("Plugin %s does not implement DataProvider", pluginName)
		data = map[string]interface{}{
			"id":      id,
			"subject": "Test Subject",
			"from":    "test@example.com",
			"body":    "Test body content",
		}
	}

	// Render form with data (edit mode)
	formHTML := RenderResourceForm(*resourceSchema, data)

	// Wrap in admin layout
	pageData := pluginFormData{
		PluginName:   pluginName,
		ResourceName: resourceSchema.Name,
		ResourceSlug: resourceSlug,
		FormHTML:     template.HTML(formHTML),
	}

	w.Header().Set("Content-Type", "text/html")
	renderPage(w, "plugin-form", pageData)
}

// JSON API handlers for agents to verify integrations

// PluginListJSON returns resource list as JSON for agents to verify integrations
func (h *PluginHandlers) PluginListJSON(w http.ResponseWriter, r *http.Request) {
	pluginName := chi.URLParam(r, "plugin")
	resourceSlug := chi.URLParam(r, "resource")

	w.Header().Set("Content-Type", "application/json")

	// Get plugin from registry
	plugin, ok := core.Get(pluginName)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error":  "Plugin not found",
			"plugin": pluginName,
		})
		return
	}

	// Get schema
	schema := plugin.Schema()

	// Find resource schema
	resourceSchema := findResourceSchema(schema, resourceSlug)
	if resourceSchema == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error":    "Resource not found",
			"plugin":   pluginName,
			"resource": resourceSlug,
		})
		return
	}

	// Check if plugin supports data fetching
	var resources []map[string]interface{}
	if dataProvider, ok := plugin.(core.DataProvider); ok {
		opts := core.ListOptions{Limit: 50, Offset: 0}
		fetchedResources, err := dataProvider.ListResources(r.Context(), resourceSlug, opts)
		if err != nil {
			log.Printf("Error fetching %s data from %s: %v", resourceSlug, pluginName, err)
			resources = []map[string]interface{}{}
		} else {
			resources = fetchedResources
		}
	} else {
		resources = []map[string]interface{}{}
	}

	json.NewEncoder(w).Encode(map[string]any{
		"plugin":   pluginName,
		"resource": resourceSlug,
		"count":    len(resources),
		"data":     resources,
	})
}

// PluginDetailJSON returns a single resource as JSON for agents to verify integrations
func (h *PluginHandlers) PluginDetailJSON(w http.ResponseWriter, r *http.Request) {
	pluginName := chi.URLParam(r, "plugin")
	resourceSlug := chi.URLParam(r, "resource")
	id := chi.URLParam(r, "id")

	w.Header().Set("Content-Type", "application/json")

	// Get plugin from registry
	plugin, ok := core.Get(pluginName)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error":  "Plugin not found",
			"plugin": pluginName,
		})
		return
	}

	// Get schema
	schema := plugin.Schema()

	// Find resource schema
	resourceSchema := findResourceSchema(schema, resourceSlug)
	if resourceSchema == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error":    "Resource not found",
			"plugin":   pluginName,
			"resource": resourceSlug,
		})
		return
	}

	// Check if plugin supports data fetching
	var data map[string]interface{}
	if dataProvider, ok := plugin.(core.DataProvider); ok {
		fetchedData, err := dataProvider.GetResource(context.Background(), resourceSlug, id)
		if err != nil {
			log.Printf("Error fetching %s/%s from %s: %v", resourceSlug, id, pluginName, err)
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]any{
				"error":    "Resource item not found",
				"plugin":   pluginName,
				"resource": resourceSlug,
				"id":       id,
			})
			return
		}
		data = fetchedData
	} else {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error":    "Plugin does not support data fetching",
			"plugin":   pluginName,
			"resource": resourceSlug,
			"id":       id,
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]any{
		"plugin":   pluginName,
		"resource": resourceSlug,
		"id":       id,
		"data":     data,
	})
}

// Helper functions

func findResourceSchema(schema core.PluginSchema, slug string) *core.ResourceSchema {
	for _, res := range schema.Resources {
		if res.Slug == slug {
			return &res
		}
	}
	return nil
}

type pluginListData struct {
	PluginName   string
	ResourceName string
	ResourceSlug string
	ListHTML     template.HTML
}

type pluginFormData struct {
	PluginName   string
	ResourceName string
	ResourceSlug string
	FormHTML     template.HTML
}

type pluginDetailData struct {
	PluginName   string
	ResourceName string
	ResourceSlug string
	ResourceID   string
	DetailHTML   template.HTML
}
