---
name: monolith-static-assets
description: Use when adding or changing CSS/JS/images and wiring static assets into templates in this embedded-filesystem monolith.
---

# Monolith Static Assets

## Use this skill when
- Editing front-end assets under `static/`.
- Connecting assets from templates.

## Asset structure
- CSS: `static/css/`
- JavaScript: `static/js/`
- Images/icons: `static/img/`

## Serving model
- Static files are embedded and served at `/static/` via `app/routes/routes.go`.
- Because embedding occurs at build/runtime startup, restart server after asset changes when needed.

## Usage in templates
Reference assets with absolute static paths, e.g.:
- `/static/css/stylesheet.css`
- `/static/js/application.js`
- `/static/img/logo.png`

## Change checklist
1. Update asset files.
2. Verify template references.
3. Run app and manually validate rendering in browser.
