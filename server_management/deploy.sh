#!/usr/bin/env bash
# Build & deploy the latest Go binary with zero‑downtime rollout.
# If PRUNE=true it also deletes all but the newest $KEEP releases.
#
# Usage examples:
#   ./deploy.sh ubuntu@203.0.113.5         # deploy, no pruning
#   PRUNE=true ./deploy.sh ubuntu@203.0.113.5   # deploy + prune
#
# Environment variables you may override:
#   KEEP   – how many past releases to keep (default 5)
#   PRUNE  – "true" enables pruning, anything else disables it

set -xeuo pipefail

REMOTE="$1"                 # e.g. ubuntu@203.0.113.5
APP_NAME="monolith"
APP_DIR="/opt/$APP_NAME"

KEEP="${KEEP:-5}"
PRUNE="${PRUNE:-false}"

TS=$(date +%Y%m%d%H%M%S)
RELEASE_DIR="$APP_DIR/releases/$TS"
BIN="$APP_NAME"             # assumes your main package builds to this name

echo "▶ Building $BIN..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-s -w' -o "$BIN" .

echo "▶ Creating release directory on remote..."
ssh "$REMOTE" "sudo mkdir -p $RELEASE_DIR && sudo chown \$(whoami): $RELEASE_DIR"

echo "▶ Copying binary..."
scp "$BIN" "$REMOTE:$RELEASE_DIR/"

echo "▶ Updating symlink & restarting service..."
ssh "$REMOTE" bash -s -- "$APP_DIR" "$APP_NAME" "$RELEASE_DIR" "$KEEP" "$PRUNE" <<'EOSH'
set -xeuo pipefail
APP_DIR="$1"; APP_NAME="$2"; RELEASE_DIR="$3"; KEEP="$4"; PRUNE="$5"

# 1) Atomic symlink swap
sudo ln -sfn "$RELEASE_DIR" "$APP_DIR/current"

# 2) Zero‑downtime restart
sudo systemctl restart "$APP_NAME.service"
sudo systemctl reload caddy    # only if Caddyfile changed

# 3) Optional pruning
if [[ "$PRUNE" == "true" ]]; then
  cd "$APP_DIR/releases"
  sudo ls -1dt */ | tail -n +$((KEEP+1)) | sudo xargs -r rm -rf --
  echo "🧹 Pruned old releases (kept $KEEP)."
else
  echo "ℹ️  Skipped pruning (PRUNE=$PRUNE)."
fi

echo "✅ Deployed release $(basename "$RELEASE_DIR")"
EOSH

rm "$BIN"

echo "✅ Deployment complete!"