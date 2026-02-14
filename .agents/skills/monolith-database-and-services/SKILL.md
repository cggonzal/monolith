---
name: monolith-database-and-services
description: Use when working on persistence setup, migrations, and service integrations (like email) that depend on app config and models.
---

# Monolith Database and Services

## Use this skill when
- Modifying DB initialization or model migration behavior.
- Implementing external integrations in `app/services/`.

## Database flow
- Initialized by `db.InitDB()`.
- Uses GORM with SQLite driver by default.
- AutoMigrate runs for registered models at startup.

## Model registration rule
Any new model must be included in `db/db.go` AutoMigrate call.
The model generator handles this automatically.

## Services pattern
- Service code belongs under `app/services/` (e.g., `app/services/email`).
- Keep HTTP controllers thin; push external API logic into services.
- Source environment settings from `app/config/config.go`.

## Email specifics
- Uses `MAILGUN_DOMAIN`, `MAILGUN_API_KEY`, and `MAILGUN_API_BASE`.
- Missing mail variables are warned by config init.
