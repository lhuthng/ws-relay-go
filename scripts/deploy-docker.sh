#!/usr/bin/env bash

set -euo pipefail

APP_NAME="${APP_NAME:-ws-relay-go}"
IMAGE_REF="${IMAGE_REF:?IMAGE_REF is required}"
SERVICE_PORT="${SERVICE_PORT:?SERVICE_PORT is required}"
CONTAINER_PORT="${CONTAINER_PORT:-5001}"
MAX_ROOM_SIZE="${MAX_ROOM_SIZE:-4}"

if ! command -v docker >/dev/null 2>&1; then
  if command -v apt-get >/dev/null 2>&1; then
    sudo apt-get update
    sudo apt-get install -y docker.io
    sudo systemctl enable --now docker
    sudo usermod -aG docker "${USER}" || true
  else
    echo "docker is not installed and this script only knows how to bootstrap it on apt-based systems" >&2
    exit 1
  fi
fi

sudo docker pull "${IMAGE_REF}"
sudo docker rm -f "${APP_NAME}" >/dev/null 2>&1 || true
sudo docker run -d \
  --name "${APP_NAME}" \
  --restart unless-stopped \
  -p "${SERVICE_PORT}:${CONTAINER_PORT}" \
  -e PORT="${CONTAINER_PORT}" \
  -e MAX_ROOM_SIZE="${MAX_ROOM_SIZE}" \
  "${IMAGE_REF}"

sudo docker ps --filter "name=${APP_NAME}"
