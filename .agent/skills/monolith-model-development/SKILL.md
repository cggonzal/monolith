---
name: monolith-model-development
description: Use when creating or modifying GORM models in app/models, including generator usage, CRUD helper patterns, hooks, and migration registration.
---

# Monolith Model Development

## Use this skill when
- Adding a new persisted domain entity.
- Updating schema and model helpers.

## Default approach
1. Scaffold with `make generator model <Name> field:type...` when possible.
2. Confirm model is added to `db/db.go` AutoMigrate list.
3. Add/adjust helper functions in `app/models/<name>.go`.
4. Add tests under `app/models/*_test.go` and run `go test ./...`.

## Expected model conventions
- Embed `gorm.Model`.
- Include `IsActive bool` soft-delete style flag.
- Provide CRUD helpers (`CreateX`, `GetXByID`, `GetAllXs`, `UpdateX`, `DeleteX`).
- Keep validation in hooks (`BeforeSave`/`AfterSave`) or explicit helper functions.

## If you need full REST scaffolding
Use `make generator resource <singular> field:type...`.
This will create model + controller + routes + views in one pass.

## Migration notes
- Auto migration occurs during `db.InitDB()` on startup.
- Adding a model without registering it in AutoMigrate means no schema creation.
