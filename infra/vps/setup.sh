#!/usr/bin/env bash
# One-time VPS setup for the Go API (Ubuntu, run as the deploy user, which needs sudo). Safe to re-run:
# it never overwrites an existing /etc/binarysust-api.env and does not start the service.
#
#   scp -r infra/vps <user>@<host>:/tmp/binarysust-vps && ssh <user>@<host> bash /tmp/binarysust-vps/setup.sh
#
# Afterwards: fill /etc/binarysust-api.env, set up Cloudflare Tunnel, run the deploy-api workflow
# (docs/DEPLOY.md step 3).
set -euo pipefail

here=$(cd "$(dirname "$0")" && pwd)
app_dir=/opt/binarysust-api
env_file=/etc/binarysust-api.env
deploy_user=$(id -un)

sudo install -d -o "$deploy_user" -g "$(id -gn)" -m 755 "$app_dir" "$app_dir/releases"

if sudo test -e "$env_file"; then
  echo "kept existing $env_file"
else
  sudo install -o root -g root -m 600 "$here/binarysust-api.env.example" "$env_file"
  echo "created $env_file: fill it in with: sudo nano $env_file"
fi

sudo install -o root -g root -m 644 "$here/binarysust-api.service" /etc/systemd/system/binarysust-api.service
sudo systemctl daemon-reload
sudo systemctl enable binarysust-api >/dev/null
echo "installed and enabled binarysust-api.service (starts on the first deploy)"

if command -v cloudflared >/dev/null; then
  echo "cloudflared: $(cloudflared --version)"
else
  echo "cloudflared is not installed yet (docs/DEPLOY.md step 3)"
fi
