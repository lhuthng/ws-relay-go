#!/usr/bin/env bash

set -euo pipefail

APP_NAME="${APP_NAME:-ws-relay-go}"
REMOTE_PATH="${REMOTE_PATH:-/usr/local/bin/ws-relay-go}"
SERVICE_NAME="${SERVICE_NAME:-ws-relay-go}"
SERVICE_USER="${SERVICE_USER:-$USER}"
SERVICE_PORT="${SERVICE_PORT:-5001}"
ENV_FILE="/etc/default/${SERVICE_NAME}"
UNIT_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
STATE_DIR="/var/lib/${SERVICE_NAME}"

sudo install -d -m 0755 "$(dirname "${REMOTE_PATH}")"
sudo install -d -m 0755 "${STATE_DIR}"

if id -u "${SERVICE_USER}" >/dev/null 2>&1; then
  sudo chown "${SERVICE_USER}:${SERVICE_USER}" "${STATE_DIR}"
fi

sudo install -m 0755 "/tmp/${APP_NAME}" "${REMOTE_PATH}"

if ! sudo test -f "${ENV_FILE}"; then
  printf 'PORT=5001\nMAX_ROOM_SIZE=4\n' | sudo tee "${ENV_FILE}" >/dev/null
fi

tmp_env_file="$(mktemp)"
trap 'rm -f "${tmp_env_file}"' EXIT

sudo cat "${ENV_FILE}" > "${tmp_env_file}" || true

if grep -q '^PORT=' "${tmp_env_file}"; then
  sed -i.bak "s/^PORT=.*/PORT=${SERVICE_PORT}/" "${tmp_env_file}"
else
  printf '\nPORT=%s\n' "${SERVICE_PORT}" >> "${tmp_env_file}"
fi

if ! grep -q '^MAX_ROOM_SIZE=' "${tmp_env_file}"; then
  printf 'MAX_ROOM_SIZE=4\n' >> "${tmp_env_file}"
fi

sudo install -m 0644 "${tmp_env_file}" "${ENV_FILE}"

cat <<EOF | sudo tee "${UNIT_FILE}" >/dev/null
[Unit]
Description=${SERVICE_NAME} websocket relay server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=${SERVICE_USER}
Group=${SERVICE_USER}
WorkingDirectory=${STATE_DIR}
EnvironmentFile=-${ENV_FILE}
ExecStart=${REMOTE_PATH}
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable "${SERVICE_NAME}"
sudo systemctl restart "${SERVICE_NAME}"
sudo systemctl --no-pager --full status "${SERVICE_NAME}"
