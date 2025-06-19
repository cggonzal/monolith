// Package main configures and starts the monolith HTTP server.
package main

import (
	"embed"
	"log/slog"
	"os"

	"monolith/db"
	"monolith/generator"
	"monolith/server_management"
	"monolith/views"
	"monolith/ws"
)

//go:embed static/*
var staticFiles embed.FS

//go:embed views/*.html.tmpl
var templateFiles embed.FS

func main() {
	// Dispatch to generators if requested
	if len(os.Args) > 1 && (os.Args[1] == "generator" || os.Args[1] == "generators") {
		args := os.Args[2:]
		if os.Args[1] == "generators" {
			args = append([]string{"help"}, args...)
		}
		if err := generator.Run(args); err != nil {
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
	views.InitTemplates(templateFiles)

	// initialize the websocket pub/sub
	ws.InitPubSub()

	// start the server!
	server_management.RunServer(staticFiles)
}
