// Package templates provides functionality to parse and cache HTML templates from an embedded filesystem.
package templates

import (
	"embed"
	"html/template"
	"io"
)

// Template cache. InitTemplates() fills this with all HTML templates from embedded filesystem
var tmpl *template.Template

// parse all templates and store them in the template cache
func InitTemplates(templateFiles embed.FS) {
	tmpl = template.Must(template.ParseFS(templateFiles, "templates/*.html.tmpl"))
}

// ExecuteTemplate executes a named template with the provided data and writes the output to the given writer.
func ExecuteTemplate(wr io.Writer, name string, data interface{}) error {
	return tmpl.ExecuteTemplate(wr, name, data)
}
