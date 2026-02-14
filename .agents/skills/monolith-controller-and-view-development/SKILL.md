---
name: monolith-controller-and-view-development
description: Use when building HTTP controllers and templates, including REST action scaffolding, view rendering patterns, and route wiring.
---

# Monolith Controller and View Development

## Use this skill when
- Creating request handlers in `app/controllers/`.
- Generating or editing templates in `app/views/`.

## Fast start
- Controller scaffold: `make generator controller widgets index show`
- Full CRUD controller/views: `make generator controller widgets all`

## REST action mapping
- `Index` → `GET /resources`
- `Show` → `GET /resources/{id}`
- `New` → `GET /resources/new`
- `Create` → `POST /resources`
- `Edit` → `GET /resources/{id}/edit`
- `Update` → `PUT/PATCH /resources/{id}`
- `Destroy` → `DELETE /resources/{id}`

## Rendering pattern
Use `views.Render(w, "template_name.html.tmpl", data)`.
Templates are pre-parsed at startup by `views.InitTemplates(...)`.

## Template naming convention
Generators produce files like:
- `app/views/widgets/widgets_index.html.tmpl`
- `app/views/widgets/widgets_show.html.tmpl`
- `app/views/widgets/widgets_new.html.tmpl`
- `app/views/widgets/widgets_edit.html.tmpl`

## Implementation checklist
1. Parse path params with `r.PathValue("id")` for ID routes.
2. Load/save model data via `db.GetDB()` + model helpers.
3. Render template or redirect as appropriate.
4. Register route in `app/routes/routes.go` (generator does this when actions are provided).
5. Add/adjust controller tests.
