// ABOUTME: Template loading and rendering for admin UI.
// ABOUTME: Embeds HTML templates and provides render helpers.

package admin

import (
	"embed"
	"html/template"
	"io"
)

//go:embed templates/*
var templateFS embed.FS

var (
	layoutTmpl   *template.Template
	pageTmpls    map[string]*template.Template
	partialTmpls *template.Template
)

// partialPaths defines all row templates used for htmx partial rendering
var partialPaths = []string{
	"templates/gmail/row.html",
	"templates/calendar/row.html",
	"templates/people/row.html",
	"templates/tasks/row.html",
}

// pageDefinitions maps page names to their template files
func getPageDefinitions() map[string]string {
	return map[string]string{
		"dashboard":     "templates/dashboard.html",
		"guide":         "templates/guide.html",
		"gmail-list":    "templates/gmail/list.html",
		"gmail-form":    "templates/gmail/form.html",
		"gmail-view":    "templates/gmail/view.html",
		"calendar-list": "templates/calendar/list.html",
		"calendar-form": "templates/calendar/form.html",
		"calendar-view": "templates/calendar/view.html",
		"people-list":   "templates/people/list.html",
		"people-form":   "templates/people/form.html",
		"people-view":   "templates/people/view.html",
		"tasks-list":    "templates/tasks/list.html",
		"tasks-form":    "templates/tasks/form.html",
		"tasks-view":    "templates/tasks/view.html",
		"logs-list":     "templates/logs/list.html",
		"plugin-list":   "templates/plugins/list.html",
		"plugin-form":   "templates/plugins/form.html",
		"plugin-detail": "templates/plugins/detail.html",
	}
}

// parsePartialTemplates creates a template bundle with all row templates for htmx rendering
func parsePartialTemplates() *template.Template {
	return template.Must(template.ParseFS(templateFS, partialPaths...))
}

// parsePageTemplates creates a map of page templates, each with layout and partials
func parsePageTemplates() map[string]*template.Template {
	templates := make(map[string]*template.Template)
	pages := getPageDefinitions()

	for name, path := range pages {
		// Clone layout and add page template
		tmpl := template.Must(layoutTmpl.Clone())
		tmpl = template.Must(tmpl.ParseFS(templateFS, path))

		// Add all partial templates to support htmx rendering
		tmpl = template.Must(tmpl.ParseFS(templateFS, partialPaths...))

		templates[name] = tmpl
	}

	return templates
}

func init() {
	// Parse layout as base
	layoutTmpl = template.Must(template.ParseFS(templateFS, "templates/layout.html"))

	// Parse partials (row templates for htmx)
	partialTmpls = parsePartialTemplates()

	// Parse each page template with its own copy of layout
	pageTmpls = parsePageTemplates()
}

func renderPage(w io.Writer, page string, data any) error {
	tmpl, ok := pageTmpls[page]
	if !ok {
		return nil
	}
	return tmpl.ExecuteTemplate(w, "layout", data)
}

func renderPartial(w io.Writer, name string, data any) error {
	return partialTmpls.ExecuteTemplate(w, name, data)
}
