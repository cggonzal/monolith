---
name: monolith-routing-and-middleware
description: Use when editing route registration, middleware ordering, CSRF behavior, and request pipeline behavior for HTTP handlers.
---

# Monolith Routing and Middleware

## Use this skill when
- Adding new HTTP endpoints.
- Inserting request/response middleware.
- Troubleshooting request flow issues.

## Routing fundamentals
- Main router is `http.ServeMux` in `app/routes/routes.go`.
- Routes are registered in `registerRoutes`.
- Pattern format uses method + path (e.g., `"GET /widgets/{id}"`).

## Middleware fundamentals
- Middleware registration order lives in `app/middleware/registration.go`.
- Current baseline includes logging and CSRF middleware.
- Middleware wraps mux globally through `InitServerHandler`.

## Adding middleware safely
1. Implement `func(http.Handler) http.Handler` in `app/middleware/`.
2. Add it to registration slice in desired order.
3. Ensure it is side-effect free and deterministic.
4. Add tests similar to existing middleware tests.

## CSRF behavior
`CSRFMiddleware` leverages Go's cross-origin protections; keep it in the stack unless you intentionally disable protection for a specific reason.
