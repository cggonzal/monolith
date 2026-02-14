---
name: monolith-generator-workflows
description: Use when adding app features via the built-in generator (model/controller/resource/authentication/job/admin), including expected file mutations and post-generation validation.
---

# Monolith Generator Workflows

## Use this skill when
- You need to scaffold features quickly and safely.
- You need to know all supported generator subcommands.

## Generator surface
`make generator <command> [args]`

Supported commands:
- `model`
- `controller`
- `resource`
- `authentication`
- `job`
- `admin`

## Behavior summary
- `model`: creates model + model test; updates `db/db.go` AutoMigrate.
- `controller`: creates controller + controller test + views; optionally injects routes.
- `resource`: combines `model` + pluralized controller with full CRUD actions.
- `authentication`: scaffolds user model, session helpers, auth middleware/controller/templates/routes.
- `job`: creates job file + test, updates job enum and queue registration.
- `admin`: scaffolds dashboard/middleware/routes, and auto-runs auth generation if `User` model is missing.

## Safe scaffolding checklist
1. Run generator command.
2. Inspect modified files for route collisions/import order.
3. Fill TODO placeholders in generated handlers/jobs.
4. Run `gofmt` if needed and `go test ./...`.
5. Start app and manually hit generated routes.

## Practical patterns
- Greenfield CRUD: `make generator resource post title:string body:string`
- Existing model + UI: `make generator controller posts index show new create edit update destroy`
- Async task: `make generator job SendDigest`
- Auth bootstrap: `make generator authentication`
- Admin/profiling: `make generator admin`
