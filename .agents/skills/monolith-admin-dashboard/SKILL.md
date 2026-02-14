---
name: monolith-admin-dashboard
description: Use when generating or extending the built-in admin dashboard, admin-only middleware, and pprof debug endpoints.
---

# Monolith Admin Dashboard

## Use this skill when
- Enabling operational/admin UI.
- Adding admin-only diagnostics and controls.

## Bootstrap command
`make generator admin`

## What gets scaffolded
- Admin controller (`/admin` dashboard)
- Admin template
- `RequireAdmin` middleware helpers/tests
- Route wiring for admin dashboard and `/debug/pprof/*`
- Auto-runs authentication scaffold if `User` model does not exist

## Expected route shape
- `GET /admin` and `POST /admin` guarded by admin middleware
- `GET /debug/pprof/...` endpoints guarded by admin middleware

## Extension workflow
1. Add cards/metrics/actions in admin template.
2. Keep heavy operations behind POST actions.
3. Ensure admin middleware checks real session user role.
4. Add tests for unauthorized and authorized paths.
