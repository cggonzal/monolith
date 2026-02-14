# Monolith Agent Skills

These skills follow the open skills layout (`.agents/skills/<skill>/SKILL.md`) so AI agents can load only the workflow they need.

The repository root `AGENTS.md` is configured to auto-discover these skills on every task, so users do not need to explicitly ask agents to use them.

Claude Code compatibility: see `CLAUDE.md` at repo root for identical auto-discovery instructions.

## Skills
- `monolith-project-overview`
- `monolith-command-reference`
- `monolith-generator-workflows`
- `monolith-model-development`
- `monolith-controller-and-view-development`
- `monolith-job-development`
- `monolith-websocket-pubsub`
- `monolith-auth-and-sessions`
- `monolith-admin-dashboard`
- `monolith-routing-and-middleware`
- `monolith-static-assets`
- `monolith-database-and-services`
- `monolith-testing-and-operations`

Tip: load `monolith-project-overview` first, then feature-specific skills.
