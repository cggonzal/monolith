// Package main configures and starts the monolith HTTP server.
package main

import (
	"embed"
	"log/slog"
	"os"

	"monolith/config"
	"monolith/db"
	"monolith/handlers"
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

	// start the server!
	server_management.RunServer(staticFiles)
}
