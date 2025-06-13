package controllers

import (
	"monolith/templates"
	"net/http"
	"strings"
)

type ItemController struct{}

var ItemCtrl = &ItemController{}

// EditItemHandler handles displaying an edit form (simulated)
func (ic *ItemController) EditItemHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/edit/")
	data := struct {
		ID   string
		Name string
	}{
		ID:   id,
		Name: "Example Item " + id,
	}
	templates.ExecuteTemplate(w, "edit.html.tmpl", data)
}

// DeleteItemHandler simulates deleting an item
func (ic *ItemController) DeleteItemHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/delete/")
	http.Redirect(w, r, "/?deleted="+id, http.StatusSeeOther)
}
