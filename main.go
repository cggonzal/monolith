// Package main configures and starts the monolith HTTP server.
package main

import (
	"embed"
	"log/slog"
	"os"

	"monolith/db"
	"monolith/generator"
	"monolith/handlers"
	"monolith/server_management"
	"monolith/ws"
)

//go:embed static/*
var staticFiles embed.FS

//go:embed templates/*
var templateFiles embed.FS

func main() {
	// Dispatch to generators if requested
	if len(os.Args) > 1 && os.Args[1] == "generator" {
		if err := generator.Run(os.Args[2:]); err != nil {
			slog.Error("generator failed", "error", err)
		}
		return
	}

	// Configure global structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// Initialize database
	db.Connect()

	// initialize templates
	handlers.InitTemplates(templateFiles)

	// initialize the websocket pub/sub
	ws.InitPubSub()

	// start the server!
	server_management.RunServer(staticFiles)
}
