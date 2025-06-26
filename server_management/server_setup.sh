#!/usr/bin/env bash
# This script is meant to be run from the root of the monolith project.
# Boot‑straps a fresh Ubuntu server for zero‑downtime Go deploys. Assumes that the ssh command can find your ssh key by default.
# This script sets up a server with Caddy, systemd socket activation, and a basic Caddyfile.
# Usage: ./server_setup.sh [--no-litestream] user@host example.com [ACCESS_KEY_ID] [SECRET_ACCESS_KEY]

set -xeuo pipefail

NO_LITESTREAM=false
if [[ "${1:-}" == "--no-litestream" ]]; then
    NO_LITESTREAM=true
    shift
fi

REMOTE="$1"                 # e.g. ubuntu@203.0.113.5
APP_NAME="monolith"         # systemd unit prefix and directory name
DOMAIN="$2"                 # domain served by Caddy
ACCESS_KEY_ID="${ACCESS_KEY_ID:-${3:-}}"
SECRET_ACCESS_KEY="${SECRET_ACCESS_KEY:-${4:-}}"
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

# ----- 2b. Install Litestream -------------------------------------------------
if [ "$NO_LITESTREAM" != "true" ] && ! command -v litestream >/dev/null ; then
  echo "+ Installing Litestream"
  curl -fsSL https://github.com/benbjohnson/litestream/releases/latest/download/litestream-linux-amd64.deb -o /tmp/litestream.deb
  sudo dpkg -i /tmp/litestream.deb
  rm /tmp/litestream.deb
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
WorkingDirectory=$APP_DIR
Restart=always
RestartSec=2
TimeoutStopSec=30
KillMode=mixed

[Install]
WantedBy=multi-user.target
UNIT

# ----- 4b. Litestream configuration -----------------------------------------
if [ "$NO_LITESTREAM" != "true" ]; then
  sudo tee /etc/litestream.yml >/dev/null <<CFG
access-key-id: ${ACCESS_KEY_ID}
secret-access-key: ${SECRET_ACCESS_KEY}
dbs:
  - path: $APP_DIR/app.db
    replicas:
      - type: s3
        bucket: ${APP_NAME}-backups
        endpoint: https://nyc3.digitaloceanspaces.com
        region: nyc3
        path: ${APP_NAME}.db
CFG
fi

# ----- 5. Caddyfile --------------------------------------------------------
sudo tee /etc/caddy/Caddyfile >/dev/null <<CFG
$DOMAIN {
	encode zstd gzip
	reverse_proxy 127.0.0.1:$BIN_PORT
}
CFG

# ----- 6. Enable & start everything ---------------------------------------
sudo systemctl daemon-reload
sudo systemctl enable --now $APP_NAME.socket
sudo systemctl restart caddy       # picks up Caddyfile
if [ "$NO_LITESTREAM" != "true" ]; then
  sudo systemctl enable --now litestream.service
fi
echo "✅ Server bootstrap complete."
EOF

echo "You can now deploy your app using the deploy.sh script."
