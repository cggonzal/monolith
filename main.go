// Package main configures and starts the monolith HTTP server.
package main

import (
	"embed"
	"flag"
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
	guides := flag.Bool("guides", false, "Run guides server")
	g := flag.Bool("g", false, "Run guides server (shorthand)")
	flag.Parse()

	if *guides || *g {
		server_management.RunGuidesServer()
		return
	}

	// Dispatch to generator if requested
	args := flag.Args()
	if len(args) > 0 && args[0] == "generator" {
		genArgs := args[1:]
		if err := generator.Run(genArgs); err != nil {
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
