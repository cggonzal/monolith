package controllers

import (
	"monolith/app/views"
	"net/http"
)

type ErrorController struct{}

var ErrorCtrl = &ErrorController{}

// Show404 renders the 404 page with a 404 status code.
func (ec *ErrorController) Show404(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	views.Render(w, "404.html.tmpl", nil)
}
