package main

import (
	"log"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os"

	"crudapp/db"
	"crudapp/handlers"
	"crudapp/middleware"
)

func main() {
	// Initialize database
	db.Connect()

	// Configure global structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("Starting server", "address", ":8080")

	mux := http.NewServeMux()

	// Serve static files (keeping /static/ in URL)
	mux.Handle("GET /static/", http.FileServer(http.Dir(".")))

	// OAuth routes
	mux.HandleFunc("GET /auth/google", handlers.HandleGoogleLogin)
	mux.HandleFunc("GET /auth/google/callback", handlers.HandleGoogleCallback)

	// Public routes
	mux.HandleFunc("GET /", handlers.Home)
	mux.HandleFunc("GET /login", handlers.ShowLoginForm)
	mux.HandleFunc("GET /logout", handlers.Logout)

	// Protected routes
	mux.HandleFunc("GET /dashboard", middleware.RequireLogin(handlers.Dashboard))
	mux.HandleFunc("GET /edit/{id}", middleware.RequireLogin(handlers.EditItemHandler))
	mux.HandleFunc("POST /delete/{id}", middleware.RequireLogin(handlers.DeleteItemHandler))

	// pprof routes
	mux.HandleFunc("GET /debug/pprof/", pprof.Index)
	mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)

	// Wrap with structured logging middleware
	loggedRouter := middleware.LoggingMiddleware(mux)

	slog.Info("Server running", "address", ":8080")
	log.Fatal(http.ListenAndServe(":8080", loggedRouter))
}
