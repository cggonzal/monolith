/*
Package server_management provides helpers for running the HTTP server used by
the monolith.
*/
package server_management

import (
	"context"
	"embed"
	"fmt"
	"log"
	"log/slog"
	"monolith/app/config"
	"monolith/app/routes"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func RunServer(staticFiles embed.FS) {
	addr := "127.0.0.1:" + config.PORT
	slog.Info("Starting server", "address", addr)

	server := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		Handler:           routes.InitServerHandler(staticFiles),
	}

	// Graceful shutdown on SIGTERM/SIGINT.
	idleConnsClosed := make(chan struct{})
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			slog.Error("HTTP shutdown", "error", err)
		}
		close(idleConnsClosed)
	}()

	slog.Info("serving HTTP")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Serve: %v", err)
	}

	<-idleConnsClosed
	slog.Info("goodbye")
}

func RunGuidesServer() {
	addr := ":9000"
	server := &http.Server{
		Addr:              addr,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
		Handler:           http.FileServer(http.Dir("guides")),
	}

	fmt.Println("Serving guides at http://localhost:9000")
	fmt.Println("Open http://localhost:9000 in your browser to view the guides.")

	idleConnsClosed := make(chan struct{})
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			slog.Error("guides shutdown", "error", err)
		}
		close(idleConnsClosed)
	}()

	slog.Info("serving guides HTTP", "address", "http://localhost:9000")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Serve guides: %v", err)
	}

	<-idleConnsClosed
	slog.Info("guides server stopped")
}
