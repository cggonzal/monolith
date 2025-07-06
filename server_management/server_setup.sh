#!/usr/bin/env bash
# This script is meant to be run from the root of the monolith project.
# Boot‑straps a fresh Ubuntu server for zero‑downtime Go deploys. Assumes that the ssh command can find your ssh key by default.
# This script sets up a server with Caddy and systemd socket activation.
# It uploads the Caddyfile from server_management/Caddyfile in this repo.
# Usage: ./server_setup.sh user@host

set -xeuo pipefail

REMOTE="$1"                 # e.g. ubuntu@203.0.113.5
APP_NAME="monolith"         # systemd unit prefix and directory name
APP_DIR="/opt/$APP_NAME"    # where releases/ and current -> releaseX live
BIN_PORT="9000"             # must match systemd socket + Caddy reverse_proxy

ssh "$REMOTE" bash -s <<EOF
set -xeuo pipefail
# ----- 1. Base packages ----------------------------------------------------
sudo apt-get update -qq
sudo DEBIAN_FRONTEND=noninteractive apt-get install -y \
     curl tzdata git ca-certificates

# ----- 2. Install Caddy ----------------------------------------------------
if ! command -v caddy >/dev/null ; then
  echo "+ Installing Caddy"
  sudo apt-get install -y debian-keyring debian-archive-keyring apt-transport-https
  curl -fsSL https://dl.cloudsmith.io/public/caddy/stable/gpg.key | sudo tee /usr/share/keyrings/caddy-stable-archive-keyring.asc
  echo "deb [signed-by=/usr/share/keyrings/caddy-stable-archive-keyring.asc] \
        https://dl.cloudsmith.io/public/caddy/stable/deb/ubuntu \$(lsb_release -cs) main" \
        | sudo tee /etc/apt/sources.list.d/caddy-stable.list
  sudo apt-get update -qq
  sudo apt-get install -y caddy
fi

# ----- 3. Create app directories ------------------------------------------
sudo mkdir -p $APP_DIR/releases
sudo chown -R \$(whoami): \$(dirname $APP_DIR)

# ----- 4. systemd socket + service ----------------------------------------
sudo tee /etc/systemd/system/$APP_NAME.socket >/dev/null <<UNIT
[Unit]
Description=$APP_NAME listener (socket activation)

[Socket]
ListenStream=127.0.0.1:$BIN_PORT
NoDelay=true

[Install]
WantedBy=sockets.target
UNIT

sudo tee /etc/systemd/system/$APP_NAME.service >/dev/null <<UNIT
[Unit]
Description=$APP_NAME service (Go static binary)
Requires=$APP_NAME.socket
After=network.target

[Service]
Type=notify
ExecStart=$APP_DIR/current/$APP_NAME -listen-fd
Restart=always
RestartSec=2
TimeoutStopSec=30
KillMode=mixed

[Install]
WantedBy=multi-user.target
UNIT

# ----- 5. Enable services -------------------------------------------------
sudo systemctl daemon-reload
sudo systemctl enable --now $APP_NAME.socket
echo "✅ Base server setup complete."
EOF

echo "▶ Uploading Caddyfile..."
scp "$(dirname "$0")/Caddyfile" "$REMOTE:/tmp/Caddyfile"
ssh "$REMOTE" "sudo mv /tmp/Caddyfile /etc/caddy/Caddyfile"
ssh "$REMOTE" "sudo systemctl restart caddy"

echo "You can now deploy your app using the deploy.sh script."
