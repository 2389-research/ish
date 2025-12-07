// ABOUTME: Plugin admin routes that wire schema renderer to HTTP handlers.
// ABOUTME: Provides generic CRUD UI for any plugin's resources using schemas.

package admin

import (
	"context"
	"fmt"
	"html"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/2389/ish/plugins/core"
	"github.com/go-chi/chi/v5"
)

// PluginHandlers handles plugin admin UI routes
type PluginHandlers struct{}

// RegisterRoutes registers plugin admin routes
func (h *PluginHandlers) RegisterRoutes(r chi.Router) {
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
		ListHTML:     template.HTML(listHTML),
	}

	w.Header().Set("Content-Type", "text/html")
	renderPluginPage(w, "plugin-list", pageData)
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
		FormHTML:     template.HTML(formHTML),
	}

	w.Header().Set("Content-Type", "text/html")
	renderPluginPage(w, "plugin-form", pageData)
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
		DetailHTML:   template.HTML(detailHTML),
	}

	w.Header().Set("Content-Type", "text/html")
	renderPluginPage(w, "plugin-detail", pageData)
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
		FormHTML:     template.HTML(formHTML),
	}

	w.Header().Set("Content-Type", "text/html")
	renderPluginPage(w, "plugin-form", pageData)
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
	ListHTML     template.HTML
}

type pluginFormData struct {
	PluginName   string
	ResourceName string
	FormHTML     template.HTML
}

type pluginDetailData struct {
	PluginName   string
	ResourceName string
	DetailHTML   template.HTML
}

func renderPluginPage(w http.ResponseWriter, templateName string, data interface{}) error {
	// For now, render a simple HTML wrapper
	// This will be replaced with proper template rendering later
	switch templateName {
	case "plugin-list":
		d := data.(pluginListData)
		htmlStr := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>%s - %s</title></head>
<body>
<h1>%s - %s</h1>
%s
</body>
</html>`, html.EscapeString(d.PluginName), html.EscapeString(d.ResourceName),
			html.EscapeString(d.PluginName), html.EscapeString(d.ResourceName), d.ListHTML)
		w.Write([]byte(htmlStr))

	case "plugin-form":
		d := data.(pluginFormData)
		htmlStr := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>%s - %s</title></head>
<body>
<h1>%s - %s</h1>
%s
</body>
</html>`, html.EscapeString(d.PluginName), html.EscapeString(d.ResourceName),
			html.EscapeString(d.PluginName), html.EscapeString(d.ResourceName), d.FormHTML)
		w.Write([]byte(htmlStr))

	case "plugin-detail":
		d := data.(pluginDetailData)
		htmlStr := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><title>%s - %s</title></head>
<body>
<h1>%s - %s</h1>
%s
</body>
</html>`, html.EscapeString(d.PluginName), html.EscapeString(d.ResourceName),
			html.EscapeString(d.PluginName), html.EscapeString(d.ResourceName), d.DetailHTML)
		w.Write([]byte(htmlStr))
	}
	return nil
}

// Helper to capitalize first letter
func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
