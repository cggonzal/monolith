/*
Package controllers contains the HTTP route controllers and template initialization
used by the application.
*/
package controllers

import (
	"embed"
	"html/template"
	"monolith/session"
	"net/http"
	"strings"
)

// Template cache. InitTemplates() fills this with all HTML templates from embedded filesystem
var tmpl *template.Template

// parse all templates and store them in the template cache
func InitTemplates(templateFiles embed.FS) {
	tmpl = template.Must(template.ParseFS(templateFiles, "templates/*.html.tmpl"))
}

// ShowLoginForm renders the login page
func ShowLoginForm(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "login.html.tmpl", nil)
}

// Logout clears the session and redirects to home
func Logout(w http.ResponseWriter, r *http.Request) {
	session.Logout(w, r)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Dashboard renders a protected page for logged-in users
func Dashboard(w http.ResponseWriter, r *http.Request) {
	if !session.IsLoggedIn(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	tmpl.ExecuteTemplate(w, "dashboard.html.tmpl", nil)
}

// EditItemHandler handles displaying an edit form (simulated)
func EditItemHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/edit/")
	data := struct {
		ID   string
		Name string
	}{
		ID:   id,
		Name: "Example Item " + id,
	}
	tmpl.ExecuteTemplate(w, "edit.html.tmpl", data)
}

// DeleteItemHandler simulates deleting an item
func DeleteItemHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/delete/")
	http.Redirect(w, r, "/?deleted="+id, http.StatusSeeOther)
}
