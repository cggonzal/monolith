package controllers

import (
	"monolith/views"
	"net/http"
)

type IndexController struct{}

var IndexCtrl = &IndexController{}

// ShowIndex renders the index page
func (ic *IndexController) ShowIndex(w http.ResponseWriter, r *http.Request) {
	// Render the index page using the views package
	views.Render(w, "index.html.tmpl", nil)
}
