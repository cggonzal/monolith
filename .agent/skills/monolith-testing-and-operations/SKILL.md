---
name: monolith-testing-and-operations
description: Use when validating changes with tests, running guides/docs servers, and handling operational tasks such as server setup and deployment.
---

# Monolith Testing and Operations

## Use this skill when
- Running regression checks.
- Preparing deployment-related changes.

## Test commands
- Full suite: `make test`
- Verbose suite: `make testv`
- Package-scope: `go test ./app/...`, `go test ./ws/...`, etc.

## Operational commands
- Start dev app: `make run` (or `make` for `air` hot reload)
- Serve guides: `make guides`
- Build binary: `make build`
- Deploy: `make deploy <user@host>`
- Initial server setup: `make server-setup <user@host> <domain>`

## Deployment artifacts
- Deployment scripts/config under `server_management/`.
- Caddy config included for reverse proxy + retry buffering behavior.

## Validation checklist before shipping
1. `make test`
2. Smoke test key endpoints (`/`, `/ws`, generated routes).
3. If middleware/auth changes, test anonymous and authenticated behavior.
4. If deployment code changed, review scripts and service restart assumptions.
