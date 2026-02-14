# AGENTS.md

This repository includes local, project-specific agent skills under `.agents/skills`.

## Auto-load local skills (always)
For **every task** in this repository, agents should:
1. Discover all skill files matching `.agents/skills/*/SKILL.md`.
2. Treat those skills as available without the user needing to explicitly mention them.
3. Select and use any matching skill(s) based on the task intent.
4. Prefer `monolith-project-overview` first for orientation, then task-specific skills.

## Available local skills
- `monolith-project-overview`: onboarding, architecture map, where to implement changes.
- `monolith-command-reference`: authoritative `make`/`go` command usage.
- `monolith-generator-workflows`: generator commands and expected scaffold mutations.
- `monolith-model-development`: GORM model patterns and migration registration.
- `monolith-controller-and-view-development`: controllers, templates, REST action wiring.
- `monolith-job-development`: async jobs, queue registration, payload handling.
- `monolith-websocket-pubsub`: `/ws` realtime protocol and pub/sub behavior.
- `monolith-auth-and-sessions`: login/signup/logout and cookie session flows.
- `monolith-admin-dashboard`: admin dashboard, admin middleware, pprof endpoints.
- `monolith-routing-and-middleware`: route registration and middleware ordering.
- `monolith-static-assets`: CSS/JS/image assets and template integration.
- `monolith-database-and-services`: DB init/migrations and service integrations.
- `monolith-testing-and-operations`: test strategy, run/build/deploy operational commands.

## Skill precedence
- If both system/global skills and local project skills apply, use the smallest set that fully covers the task.
- When local project instructions conflict with generic guidance, prefer local project skills for repository-specific behavior.
