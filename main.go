// Package main configures and starts the monolith HTTP server.
package main

import (
	"embed"
	"log/slog"
	"os"

	"monolith/app/config"
	"monolith/app/jobs"
	"monolith/app/session"
	"monolith/app/views"
	"monolith/db"
	"monolith/generator"
	"monolith/server_management"
	"monolith/ws"
)

//go:embed static/*
var staticFiles embed.FS

//go:embed app/views/**
var templateFiles embed.FS

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--guides" {
		server_management.RunGuidesServer()
		return
	}

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

	// initialize configuration
	config.InitConfig()

	// initialize session
	session.InitSession()

	// initialize database
	db.InitDB()

	// initialize job queue, must come after initializing the database
	jobs.InitJobQueue()

	// initialize templates
	views.InitTemplates(templateFiles)

	// initialize the websocket pub/sub
	ws.InitPubSub()

	// start the server!
	server_management.RunServer(staticFiles)
}
