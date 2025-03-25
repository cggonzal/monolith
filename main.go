package main

import (
	"context"
	"embed"
	"log"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"monolith/db"
	"monolith/handlers"
	"monolith/middleware"
	"monolith/ws"
)

//go:embed static/*
var staticFiles embed.FS

//go:embed templates/*
var templateFiles embed.FS

func main() {
	// Configure global structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Initialize database
	db.Connect()

	// initialize templates
	handlers.InitTemplates(templateFiles)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("Starting server", "address", ":"+port)

	mux := http.NewServeMux()

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

	// serve websockets routes at "/ws" endpoint
	mux.HandleFunc("GET /ws", middleware.RequireLogin(ws.ServeWs))

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

	// Block waiting for SIGTERM so the app can have a zero downtime deploy when it
	// is running as a systemd service and we do a systemctl restart <service_name>

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		for sig := range sigChan { // block until a signal is received
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP:
				// Graceful shutdown
				slog.Info("Received termination signal. Shutting down...")
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if err := server.Shutdown(ctx); err != nil {
					slog.Error("Error during graceful shutdown: " + err.Error())
				}
				os.Exit(0) // Exit the process to allow systemd to start a new instance
			}
		}
	}()

	slog.Info("Server running", "address", ":"+port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
