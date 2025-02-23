package handlers

import (
	"embed"
	"html/template"
	"net/http"
	"strings"

	"crudapp/session"
)

//go:embed templates/*
var templateFiles embed.FS

// Template cache (loads all HTML templates from embedded filesystem)
var tmpl = template.Must(template.ParseFS(templateFiles, "templates/*.html"))

// Home renders the main page
func Home(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "index.html", nil)
}

// ShowLoginForm renders the login page
func ShowLoginForm(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "login.html", nil)
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
	tmpl.ExecuteTemplate(w, "dashboard.html", nil)
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
	tmpl.ExecuteTemplate(w, "edit.html", data)
}

// DeleteItemHandler simulates deleting an item
func DeleteItemHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/delete/")
	http.Redirect(w, r, "/?deleted="+id, http.StatusSeeOther)
}
