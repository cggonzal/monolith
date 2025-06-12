// Package main configures and starts the monolith HTTP server.
package main

import (
	"context"
	"embed"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"monolith/config"
	"monolith/db"
	"monolith/handlers"
	"monolith/routes"
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

	// initialize the websocket pub/sub
	ws.InitPubSub()

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

	server := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		Handler:           routes.InitServerHandler(staticFiles),
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
