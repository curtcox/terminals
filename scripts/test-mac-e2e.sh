#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="${ROOT_DIR}/.tmp/mac-e2e"
mkdir -p "${TMP_DIR}"

export PATH="${ROOT_DIR}/.bin:${ROOT_DIR}/.sdk/flutter/bin:${PATH}"

BASE_PORT=$((54000 + (RANDOM % 500)))
GRPC_PORT="${TERMINALS_E2E_GRPC_PORT:-$BASE_PORT}"
ADMIN_PORT="${TERMINALS_E2E_ADMIN_PORT:-$((BASE_PORT + 1))}"
PHOTO_PORT="${TERMINALS_E2E_PHOTO_PORT:-$((BASE_PORT + 2))}"
MDNS_SUFFIX="${RANDOM}${RANDOM}"
MDNS_SERVICE_BASE="_terminals-e2e-${MDNS_SUFFIX}._tcp.local"
MDNS_SERVICE_SERVER="${MDNS_SERVICE_BASE}."
MDNS_NAME="HomeServerE2E-${MDNS_SUFFIX}"

SERVER_PID=""
CLIENT_PID=""

fail() {
  local message="$1"
  echo "FAIL: ${message}" >&2
  exit 1
}

cleanup() {
  if [[ -n "${CLIENT_PID}" ]] && kill -0 "${CLIENT_PID}" >/dev/null 2>&1; then
    kill "${CLIENT_PID}" >/dev/null 2>&1 || true
    wait "${CLIENT_PID}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${SERVER_PID}" ]] && kill -0 "${SERVER_PID}" >/dev/null 2>&1; then
    kill "${SERVER_PID}" >/dev/null 2>&1 || true
    wait "${SERVER_PID}" >/dev/null 2>&1 || true
  fi
  return 0
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

print_log_tail() {
  local file="$1"
  local label="$2"
  if [[ -f "${file}" ]]; then
    echo "--- ${label} (tail) ---"
    tail -n 80 "${file}" || true
    echo "--- end ${label} ---"
  else
    echo "--- ${label} missing: ${file} ---"
  fi
}

require_macos_client_prereqs() {
  if ! command -v xcodebuild >/dev/null 2>&1; then
    fail "xcodebuild is not available. Install Xcode and Command Line Tools."
  fi
  if ! command -v pod >/dev/null 2>&1; then
    fail "CocoaPods is not installed (pod command not found). Install with: brew install cocoapods"
  fi
}

start_server() {
  local server_log="$1"
  (
    cd "${ROOT_DIR}/terminal_server"
    TERMINALS_GRPC_PORT="${GRPC_PORT}" \
    TERMINALS_ADMIN_HTTP_PORT="${ADMIN_PORT}" \
    TERMINALS_PHOTO_FRAME_HTTP_PORT="${PHOTO_PORT}" \
    TERMINALS_MDNS_SERVICE="${MDNS_SERVICE_SERVER}" \
    TERMINALS_MDNS_NAME="${MDNS_NAME}" \
    go run ./cmd/server
  ) >"${server_log}" 2>&1 &
  SERVER_PID=$!
}

start_client() {
  local client_log="$1"
  shift
  (
    cd "${ROOT_DIR}/terminal_client"
    flutter run -d macos "$@"
  ) >"${client_log}" 2>&1 &
  CLIENT_PID=$!
}

stop_processes() {
  cleanup
  SERVER_PID=""
  CLIENT_PID=""
}

test_server_can_be_launched() {
  local server_log="${TMP_DIR}/server-launch.log"
  : >"${server_log}"

  start_server "${server_log}"

  if ! wait_for_log "${server_log}" "terminal server starting at" "30"; then
    print_log_tail "${server_log}" "server-launch.log"
    fail "server did not emit startup log markers"
  fi
  if ! wait_for_log "${server_log}" "control service ready" "30"; then
    print_log_tail "${server_log}" "server-launch.log"
    fail "server did not reach ready state"
  fi

  stop_processes
  echo "PASS: server can be launched"
}

test_client_can_be_launched() {
  local client_log="${TMP_DIR}/client-launch.log"
  : >"${client_log}"

  require_macos_client_prereqs

  start_client "${client_log}" \
    --dart-define=TERMINALS_E2E_EMIT_EVENTS=true

  if ! wait_for_log "${client_log}" "E2E_EVENT: client_started" "300"; then
    print_log_tail "${client_log}" "client-launch.log"
    fail "client did not launch cleanly (missing E2E_EVENT: client_started)"
  fi

  stop_processes
  echo "PASS: client can be launched"
}

test_client_detects_and_connects_to_server() {
  local server_log="${TMP_DIR}/connect-server.log"
  local client_log="${TMP_DIR}/connect-client.log"
  : >"${server_log}"
  : >"${client_log}"

  require_macos_client_prereqs

  start_server "${server_log}"
  if ! wait_for_log "${server_log}" "control service ready" "30"; then
    print_log_tail "${server_log}" "connect-server.log"
    fail "server did not reach ready state in connect test"
  fi

  start_client "${client_log}" \
    --dart-define=TERMINALS_E2E_EMIT_EVENTS=true \
    --dart-define=TERMINALS_E2E_AUTO_SCAN_CONNECT=true \
    --dart-define=TERMINALS_MDNS_SERVICE_TYPE="${MDNS_SERVICE_BASE}"

  if ! wait_for_log "${client_log}" "E2E_EVENT: discovered_servers=[1-9][0-9]*" "300"; then
    print_log_tail "${client_log}" "connect-client.log"
    print_log_tail "${server_log}" "connect-server.log"
    fail "client did not discover the running server"
  fi
  if ! wait_for_log "${client_log}" "E2E_EVENT: register_ack" "300"; then
    print_log_tail "${client_log}" "connect-client.log"
    print_log_tail "${server_log}" "connect-server.log"
    fail "client discovered server but did not complete register/connect"
  fi

  stop_processes
  echo "PASS: client detected and connected to running server"
}

main() {
  echo "Running macOS end-to-end tests..."
  echo "Using grpc=${GRPC_PORT} admin=${ADMIN_PORT} photo=${PHOTO_PORT} service=${MDNS_SERVICE_BASE}"
  test_server_can_be_launched
  test_client_can_be_launched
  test_client_detects_and_connects_to_server
  echo "All macOS E2E tests passed."
}

main "$@"
