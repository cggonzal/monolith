#!/usr/bin/env bash
# Builds & deploys the latest Go binary with zero downtime
# Usage: ./deploy.sh user@host

set -euo pipefail

REMOTE="$1"                 # e.g. ubuntu@203.0.113.5
APP_NAME="monolith"
APP_DIR="/opt/$APP_NAME"

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
ssh "$REMOTE" bash -s <<EOF
set -euo pipefail
sudo ln -sfn $RELEASE_DIR $APP_DIR/current
sudo systemctl restart $APP_NAME.service   # listener stays up via socket activation
sudo systemctl reload caddy               # only needed if Caddyfile changed
echo "✅ Deployed release $TS"
EOF

rm "$BIN"

echo "✅ Deployed release $TS"
echo "✅ Deployment complete!"