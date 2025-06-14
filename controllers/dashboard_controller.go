package controllers

import (
	"monolith/session"
	"monolith/views"
	"net/http"
)

type DashboardController struct{}

var DashboardCtrl = &DashboardController{}

// DashboardShow renders a protected page for logged-in users
func (dc *DashboardController) Show(w http.ResponseWriter, r *http.Request) {
	if !session.IsLoggedIn(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	views.Render(w, "dashboard.html.tmpl", nil)
}
