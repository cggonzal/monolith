# CLAUDE.md

This file provides Claude Code-specific repository instructions.

## Local skills location
Repository-local skills live in `.agents/skills/*/SKILL.md`.

## Required behavior for Claude Code in this repo
For every task:
1. Discover all skill files matching `.agents/skills/*/SKILL.md`.
2. Treat those skills as available even if the user does not explicitly mention them.
3. Select the minimal set of relevant skill(s) based on task intent.
4. Prefer `monolith-project-overview` first for orientation, then task-specific skills.

## Available local skills
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

## Precedence
When repository-local skill instructions conflict with generic guidance, prefer repository-local skills.
