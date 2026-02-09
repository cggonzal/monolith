---
name: monolith-command-reference
description: Use when an agent needs authoritative make/go command usage in this repository, including test, run, guides, deploy, server setup, and generator invocations.
---

# Monolith Command Reference

## Use this skill when
- You need exact commands for development and automation.
- You are creating or updating CI/dev workflows.
- You need generator command syntax.

## Primary make targets
- `make` → default dev target (`air` hot reload)
- `make run` → `go run main.go`
- `make guides` → guides server (`go run main.go --guides`)
- `make build` → compile binary `monolith`
- `make test` / `make testv`
- `make clean` (test cache)
- `make doc` (godoc on `:6060`)

## Generator commands
- `make generator model NAME field:type ...`
- `make generator controller NAME [actions...]`
- `make generator resource NAME field:type ...`
- `make generator authentication`
- `make generator job NAME`
- `make generator admin`
- `make generator help [command]`
- Alias: `make g ...`

## Server management commands
- `make server-setup <user@host> <domain>`
- `make deploy <user@host>`

## Common debugging routine
1. `make test`
2. `go test ./... -run <Name>` for targeted test
3. `make run` and verify endpoint behavior
4. If scaffolding is involved, run generator + `go test ./...` again
