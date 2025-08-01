package controllers

import (
	"log/slog"
	"monolith/db"
	"net/http"
)

// HelloController handles the /hello route
// It executes a simple database query and then writes Hello World.
type HelloController struct{}

// HelloCtrl is the shared instance used by routes.
var HelloCtrl = &HelloController{}

// Greet queries the database and responds with a greeting.
func (hc *HelloController) Greet(w http.ResponseWriter, r *http.Request) {
	if d := db.GetDB(); d != nil {
		if err := d.Exec("SELECT 1").Error; err != nil {
			slog.Error("db error", "error", err)
		}
	}
	_, _ = w.Write([]byte("Hello World"))
}
