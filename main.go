package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("Starting server", "address", ":"+port)

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

	server := &http.Server{
		Addr:    ":" + port,
		Handler: loggedRouter,
	}

	// Block waiting for SIGHUP so the app can have a zero downtime deploy when it
	// is running as a systemd service and we do a systemctl reload <service_name>

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		for sig := range sigChan { // block until a signal is received
			switch sig {
			case syscall.SIGHUP:
				// Graceful restart
				slog.Info("Received SIGHUP. Graceful restarting...")
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if err := server.Shutdown(ctx); err != nil {
					slog.Error("Error during graceful restart:", err)
				}
				// The process will exit, and systemd will start a new instance
			case syscall.SIGINT, syscall.SIGTERM:
				// Graceful shutdown
				slog.Info("Received termination signal. Shutting down...")
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if err := server.Shutdown(ctx); err != nil {
					slog.Error("Error during graceful shutdown:", err)
				}
				os.Exit(0)
			}
		}
	}()

	slog.Info("Server running", "address", ":"+port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
