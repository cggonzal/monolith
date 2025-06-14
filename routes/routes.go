package routes

import (
	"embed"
	"monolith/controllers"
	"monolith/middleware"
	"monolith/ws"
	"net/http"
	"net/http/pprof"
)

func InitServerHandler(staticFiles embed.FS) http.Handler {
	// Create a new ServeMux for routing
	mux := http.NewServeMux()

	// Register all routes
	registerRoutes(mux, staticFiles)

	// Wrap with structured logging middleware
	loggedRouter := middleware.LoggingMiddleware(mux)

	return loggedRouter
}

func registerRoutes(mux *http.ServeMux, staticFiles embed.FS) {
	// Serve static files from embedded filesystem
	staticFileServer := http.FileServer(http.FS(staticFiles))
	mux.Handle("GET /static/", staticFileServer)

	// Public routes
	mux.HandleFunc("GET /login", controllers.AuthCtrl.ShowLoginForm)
	mux.HandleFunc("POST /login", controllers.AuthCtrl.Login)
	mux.HandleFunc("GET /signup", controllers.AuthCtrl.ShowSignupForm)
	mux.HandleFunc("POST /signup", controllers.AuthCtrl.Signup)
	mux.HandleFunc("GET /logout", controllers.AuthCtrl.Logout)

	mux.HandleFunc("GET /dashboard", middleware.RequireLogin(controllers.DashboardCtrl.Show))
	mux.HandleFunc("GET /edit/{id}", middleware.RequireLogin(controllers.ItemCtrl.EditItemHandler))
	mux.HandleFunc("POST /delete/{id}", middleware.RequireLogin(controllers.ItemCtrl.DeleteItemHandler))

	// serve websockets routes at "/ws" endpoint with shared hub
	mux.HandleFunc("GET /ws", middleware.RequireLogin(func(w http.ResponseWriter, r *http.Request) {
		ws.ServeWs(w, r)
	}))

	// pprof routes
	mux.HandleFunc("GET /debug/pprof/", pprof.Index)
	mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
}
