---
name: monolith-websocket-pubsub
description: Use when implementing real-time functionality using the built-in websocket pub/sub layer (`/ws`), including subscribe/message protocol and channel design.
---

# Monolith WebSocket Pub/Sub

## Use this skill when
- Building chat/live updates.
- Debugging channel subscriptions or message broadcast behavior.

## Entry points
- Route: `GET /ws` in `app/routes/routes.go`
- Implementation: `ws/ws.go`
- Persistent message model: `app/models/ws.go`

## Client protocol
Send JSON commands:
- Subscribe: `{"command":"subscribe","identifier":"ChatChannel"}`
- Publish: `{"command":"message","identifier":"ChatChannel","data":"Hello"}`

## Server behavior
- Hub tracks channel subscriptions.
- `message` commands are persisted then broadcast to subscribed clients.
- Client loop handles register/unregister and ping/pong lifecycle.

## Extension workflow
1. Pick channel naming strategy (room, user, domain event).
2. Add frontend websocket client logic in `static/js/application.js` or app-specific JS.
3. Optionally enforce auth checks before allowing subscribe/publish.
4. Add tests in `ws/ws_test.go` style for command handling.
