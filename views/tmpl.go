// Package views provides functionality to parse and cache HTML templates from an embedded filesystem.
package views

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"strings"
)

// Template cache. InitTemplates() fills this with all HTML templates from embedded filesystem
// cached templates keyed by filename
var templates map[string]*template.Template

// parse all templates and store them in the template cache
func InitTemplates(templateFiles embed.FS) {
	templates = make(map[string]*template.Template)
	entries, err := templateFiles.ReadDir("views")
	if err != nil {
		panic(err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".html.tmpl") {
			continue
		}

		// skip the base layout itself
		if e.Name() == "base.html.tmpl" {
			continue
		}

		file := filepath.Join("views", e.Name())
		t := template.Must(template.ParseFS(templateFiles, "views/base.html.tmpl", file))
		templates[e.Name()] = t
	}
}

// Render executes a named template with the provided data and writes the output to the given writer.
func Render(wr io.Writer, name string, data interface{}) error {
	t, ok := templates[name]
	if !ok {
		return fmt.Errorf("template %s not found", name)
	}
	return t.ExecuteTemplate(wr, "base.html.tmpl", data)
}
