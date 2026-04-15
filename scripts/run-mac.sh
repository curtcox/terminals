#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SERVER_LOG_DIR="${ROOT_DIR}/.tmp"
SERVER_LOG="${SERVER_LOG_DIR}/run-mac-server.log"
CLIENT_LOG="${SERVER_LOG_DIR}/run-mac-client.log"
GRPC_PORT="${TERMINALS_GRPC_PORT:-50051}"
ADMIN_PORT="${TERMINALS_ADMIN_HTTP_PORT:-50053}"
PHOTO_PORT="${TERMINALS_PHOTO_FRAME_HTTP_PORT:-50052}"

export PATH="${ROOT_DIR}/.bin:${ROOT_DIR}/.sdk/flutter/bin:${PATH}"

mkdir -p "${SERVER_LOG_DIR}"

SERVER_PID=""
CLIENT_PID=""

cleanup() {
  local exit_code=$?
  if [[ -n "${CLIENT_PID}" ]] && kill -0 "${CLIENT_PID}" >/dev/null 2>&1; then
    kill "${CLIENT_PID}" >/dev/null 2>&1 || true
    wait "${CLIENT_PID}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${SERVER_PID}" ]] && kill -0 "${SERVER_PID}" >/dev/null 2>&1; then
    kill "${SERVER_PID}" >/dev/null 2>&1 || true
    wait "${SERVER_PID}" >/dev/null 2>&1 || true
  fi
  exit "${exit_code}"
}

trap cleanup EXIT INT TERM

wait_for_port() {
  local host="$1"
  local port="$2"
  local timeout_seconds="$3"
  local deadline=$((SECONDS + timeout_seconds))

  while (( SECONDS < deadline )); do
    if nc -z "${host}" "${port}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done

  return 1
}

wait_for_log() {
  local file="$1"
  local regex="$2"
  local timeout_seconds="$3"
  local deadline=$((SECONDS + timeout_seconds))

  while (( SECONDS < deadline )); do
    if [[ -f "${file}" ]] && grep -Eq "${regex}" "${file}"; then
      return 0
    fi
    sleep 1
  done

  return 1
}

echo "Starting server..."
(
  cd "${ROOT_DIR}/terminal_server"
  TERMINALS_GRPC_PORT="${GRPC_PORT}" \
  TERMINALS_ADMIN_HTTP_PORT="${ADMIN_PORT}" \
  TERMINALS_PHOTO_FRAME_HTTP_PORT="${PHOTO_PORT}" \
  go run ./cmd/server
) >"${SERVER_LOG}" 2>&1 &
SERVER_PID=$!

if ! wait_for_log "${SERVER_LOG}" "control service ready" "30"; then
  echo "Server did not report ready state within 30s"
  echo "Server log: ${SERVER_LOG}"
  exit 1
fi

echo "Server ready on 127.0.0.1:${GRPC_PORT}"
echo "Starting macOS client (interactive)..."
(
  cd "${ROOT_DIR}/terminal_client"
  flutter run -d macos
) >"${CLIENT_LOG}" 2>&1 &
CLIENT_PID=$!

echo "Server log: ${SERVER_LOG}"
echo "Client log: ${CLIENT_LOG}"
echo "Press Ctrl+C to stop both."

wait "${CLIENT_PID}"
