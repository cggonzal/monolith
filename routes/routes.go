package routes

import (
	"embed"
	"monolith/handlers"
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

	// OAuth routes
	mux.HandleFunc("GET /auth/google", handlers.HandleGoogleLogin)
	mux.HandleFunc("GET /auth/google/callback", handlers.HandleGoogleCallback)

	// Public routes
	mux.HandleFunc("GET /login", handlers.ShowLoginForm)
	mux.HandleFunc("GET /logout", handlers.Logout)

	// Protected routes
	mux.HandleFunc("GET /dashboard", middleware.RequireLogin(handlers.Dashboard))
	mux.HandleFunc("GET /edit/{id}", middleware.RequireLogin(handlers.EditItemHandler))
	mux.HandleFunc("POST /delete/{id}", middleware.RequireLogin(handlers.DeleteItemHandler))

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
