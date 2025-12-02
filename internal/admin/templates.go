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

var templates *template.Template

func init() {
	templates = template.Must(template.ParseFS(templateFS, "templates/*.html", "templates/**/*.html"))
}

func render(w io.Writer, name string, data any) error {
	return templates.ExecuteTemplate(w, name, data)
}
