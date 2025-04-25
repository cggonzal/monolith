package main

import (
	"context"
	"embed"
	"log"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"monolith/config"
	"monolith/db"
	"monolith/handlers"
	"monolith/middleware"
	"monolith/server_management"
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

	// get the port from the environment variable or default to 9000
	if config.PORT == "" {
		config.PORT = "9000"
	}

	// Grab the listener from systemd (fall back to a normal port if run
	// without socket activation — handy for local dev).
	listeners, err := server_management.SdListeners()
	var ln net.Listener
	if err == nil && len(listeners) > 0 {
		ln = listeners[0]
		log.Printf("using systemd listener on %s", ln.Addr())
	} else {
		ln, err = net.Listen("tcp", "127.0.0.1:"+config.PORT)
		if err != nil {
			log.Fatalf("listen: %v", err)
		}
		log.Printf("socket activation unavailable, listening on %s", ln.Addr())
	}

	slog.Info("Starting server", "address", ":"+config.PORT)

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
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		Handler:           loggedRouter,
	}

	// Tell systemd we’re ready **before** we start accepting traffic.
	go server_management.SdNotifyReady()

	// Graceful shutdown on SIGTERM/SIGINT.
	idleConnsClosed := make(chan struct{})
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("HTTP shutdown: %v", err)
		}
		close(idleConnsClosed)
	}()

	log.Printf("serving HTTP")
	if err := server.Serve(ln); err != http.ErrServerClosed {
		log.Fatalf("Serve: %v", err)
	}

	<-idleConnsClosed
	log.Printf("goodbye")
}
