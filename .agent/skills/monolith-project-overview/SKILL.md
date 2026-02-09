---
name: monolith-project-overview
description: Use for quick onboarding to this Monolith Go codebase, understanding startup flow, directory layout, and where to implement changes.
---

# Monolith Project Overview

## Use this skill when
- You need to get oriented in the codebase quickly.
- You are deciding where to place new code.
- You need to trace request lifecycle from `main.go` to controllers.

## High-signal map
- Entry point: `main.go`
- App code: `app/` (`controllers`, `models`, `views`, `routes`, `middleware`, `jobs`, `services`, `session`, `config`)
- Database bootstrapping: `db/db.go`
- Realtime pub/sub: `ws/`
- Scaffolding/generators: `generator/generator.go`
- Static assets: `static/`
- Deployment/runtime scripts: `server_management/`
- Human docs: `README.md` and `guides/*.html`

## Boot sequence (critical)
1. `config.InitConfig()` loads env defaults.
2. `session.InitSession()` initializes cookie store.
3. `db.InitDB()` opens DB + automigrates models.
4. `jobs.InitJobQueue()` starts worker queue.
5. `views.InitTemplates(...)` parses templates.
6. `ws.InitPubSub()` starts WebSocket hub.
7. `server_management.RunServer(...)` starts HTTP server.

## First commands to run
- `make run` (start app)
- `make test` (full tests)
- `make generator help` (list generator capabilities)
- `make guides` (serve built-in guides)

## Placement rules
- HTTP handlers: `app/controllers/`
- Shared request wrappers: `app/middleware/`
- Persistence/domain logic: `app/models/`
- Async work: `app/jobs/`
- External integrations: `app/services/`
- Route wiring: `app/routes/routes.go`
- Templates: `app/views/**/*.html.tmpl`
- Static files: `static/{css,js,img}`
