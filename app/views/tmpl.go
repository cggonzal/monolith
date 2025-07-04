// Package views provides functionality to parse and cache HTML templates from an embedded filesystem.
package views

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"strings"
)

// Template cache. InitTemplates() fills this with all HTML templates from embedded filesystem
// cached templates keyed by filename
var templates map[string]*template.Template

// parse all templates and store them in the template cache
func InitTemplates(templateFiles embed.FS) {
	templates = make(map[string]*template.Template)
	fs.WalkDir(templateFiles, "app/views", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".html.tmpl") {
			return nil
		}
		if d.Name() == "base.html.tmpl" {
			return nil
		}
		t := template.Must(template.ParseFS(templateFiles, "app/views/base.html.tmpl", path))
		templates[d.Name()] = t
		return nil
	})
}

// Render executes a named template with the provided data and writes the output to the given writer.
func Render(wr io.Writer, name string, data interface{}) error {
	t, ok := templates[name]
	if !ok {
		return fmt.Errorf("template %s not found", name)
	}
	return t.ExecuteTemplate(wr, "base.html.tmpl", data)
}
