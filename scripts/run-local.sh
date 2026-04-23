#!/usr/bin/env bash

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="${ROOT_DIR}/.tmp"
SERVER_LOG="${TMP_DIR}/run-local-server.log"
CLIENT_LOG="${TMP_DIR}/run-local-client.log"
LOG_ARCHIVES="${RUN_LOCAL_LOG_ARCHIVES:-3}"
# Hard cap on any single log file written by this script. Per-process RLIMIT_FSIZE
# (`ulimit -f`) protects against runaway logging that previously filled the disk.
LOG_MAX_KB="${RUN_LOCAL_LOG_MAX_KB:-524288}"  # 512 MiB default
# Real stderr/stdout preserved so diagnostics never get redirected into a log file
# (which caused a self-referential tail loop that grew to 712 GB). Writes via >&3/>&4
# always go to the terminal that launched the script, even from a redirected subshell.
exec 3>&2 4>&1

GRPC_PORT="${TERMINALS_GRPC_PORT:-}"
CONTROL_WS_PORT="${TERMINALS_CONTROL_WS_PORT:-}"
CONTROL_TCP_PORT="${TERMINALS_CONTROL_TCP_PORT:-}"
ADMIN_PORT="${TERMINALS_ADMIN_HTTP_PORT:-}"
PHOTO_PORT="${TERMINALS_PHOTO_FRAME_HTTP_PORT:-}"
CLIENT_WEB_PORT="${TERMINALS_CLIENT_WEB_PORT:-}"
CLIENT_WEB_HOST="${TERMINALS_CLIENT_WEB_HOST:-0.0.0.0}"
CLIENT_DEVICE="web-server"
SKIP_BOOTSTRAP="false"
TEST_MODE="${RUN_LOCAL_TEST_MODE:-false}"
CLIENT_STARTUP_DELAY_SECONDS="${RUN_LOCAL_CLIENT_STARTUP_DELAY_SECONDS:-5}"
OPEN_BROWSER="${RUN_LOCAL_OPEN_BROWSER:-true}"

export PATH="${ROOT_DIR}/.bin:${ROOT_DIR}/.sdk/flutter/bin:${PATH}"

SERVER_PID=""
CLIENT_PID=""
BROWSER_OPENER_PID=""
CLIENT_FOREGROUND="false"
HAS_ERROR="false"
RESERVED_PORTS=()

usage() {
  cat <<'EOF'
Usage: ./scripts/run-local.sh [--platform web-server|macos|ios|android|linux|windows] [--skip-bootstrap]

Options:
  --platform <device>  Flutter device to run locally (default: web-server)
  --client <device>    Back-compat alias for --platform
  --skip-bootstrap     Skip dependency bootstrap checks/install
  -h, --help           Show this help
EOF
}

report_log_path() {
  local file="$1"
  local label="$2"
  if [[ -f "${file}" ]]; then
    echo "--- ${label}: ${file} ---" >&3
  else
    echo "--- ${label} missing: ${file} ---" >&3
  fi
}

rotate_log() {
  # Rename file → file.1 → ... → file.${LOG_ARCHIVES}. Older archives are dropped.
  local file="$1"
  if [[ ! -e "${file}" ]]; then
    return 0
  fi
  local i="${LOG_ARCHIVES}"
  if [[ -e "${file}.${i}" ]]; then
    rm -f -- "${file}.${i}"
  fi
  while (( i > 1 )); do
    local prev=$((i - 1))
    if [[ -e "${file}.${prev}" ]]; then
      mv -f -- "${file}.${prev}" "${file}.${i}"
    fi
    i=$((i - 1))
  done
  mv -f -- "${file}" "${file}.1"
}

fail() {
  local message="$1"
  HAS_ERROR="true"
  echo "ERROR: ${message}" >&3
  report_log_path "${SERVER_LOG}" "server log"
  report_log_path "${CLIENT_LOG}" "client log"
  exit 1
}

cleanup() {
  if [[ -n "${BROWSER_OPENER_PID}" ]] && kill -0 "${BROWSER_OPENER_PID}" >/dev/null 2>&1; then
    kill "${BROWSER_OPENER_PID}" >/dev/null 2>&1 || true
    wait "${BROWSER_OPENER_PID}" >/dev/null 2>&1 || true
  fi

  if [[ -n "${CLIENT_PID}" ]] && kill -0 "${CLIENT_PID}" >/dev/null 2>&1; then
    kill "${CLIENT_PID}" >/dev/null 2>&1 || true
    wait "${CLIENT_PID}" >/dev/null 2>&1 || true
  fi

  if [[ -n "${SERVER_PID}" ]] && kill -0 "${SERVER_PID}" >/dev/null 2>&1; then
    kill "${SERVER_PID}" >/dev/null 2>&1 || true
    wait "${SERVER_PID}" >/dev/null 2>&1 || true
  fi
}

on_err() {
  local exit_code="$?"
  local line_no="$1"
  HAS_ERROR="true"
  # Always write to the saved real stderr (fd 3) — never to whatever fd 2 happens
  # to be, because in backgrounded subshells fd 2 is redirected into the server log.
  echo "ERROR: command failed at line ${line_no}: ${BASH_COMMAND}" >&3
  report_log_path "${SERVER_LOG}" "server log"
  report_log_path "${CLIENT_LOG}" "client log"
  exit "${exit_code}"
}

on_exit() {
  local exit_code="$1"
  cleanup
  if [[ "${HAS_ERROR}" == "true" || "${exit_code}" -ne 0 ]]; then
    echo "Exiting with errors. Logs in ${TMP_DIR}." >&3
  fi
}

trap 'on_err ${LINENO}' ERR
trap 'on_exit $?' EXIT

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

extract_browser_url() {
  local file="$1"
  grep -Eo 'http://localhost:[0-9]+' "${file}" | tail -n 1
}

is_port_available() {
  local port="$1"

  if command -v lsof >/dev/null 2>&1; then
    if lsof -nP -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1; then
      return 1
    fi
    return 0
  fi

  if nc -z 127.0.0.1 "${port}" >/dev/null 2>&1; then
    return 1
  fi

  return 0
}

is_port_reserved() {
  local port="$1"
  local reserved_port
  for reserved_port in "${RESERVED_PORTS[@]:-}"; do
    if [[ -z "${reserved_port}" ]]; then
      continue
    fi
    if [[ "${reserved_port}" == "${port}" ]]; then
      return 0
    fi
  done
  return 1
}

reserve_port() {
  local port="$1"
  RESERVED_PORTS+=("${port}")
}

find_available_port() {
  local start_port="$1"
  local max_tries="$2"
  local candidate="${start_port}"
  local i

  for ((i = 0; i < max_tries; i++)); do
    if is_port_available "${candidate}" && ! is_port_reserved "${candidate}"; then
      echo "${candidate}"
      return 0
    fi
    candidate=$((candidate + 1))
  done

  return 1
}

require_available_port() {
  local port="$1"
  local label="$2"
  local env_name="$3"

  if is_port_reserved "${port}"; then
    fail "${label} port ${port} conflicts with another selected local port (set ${env_name} to a unique open port or unset it for auto-selection)"
  fi

  if ! is_port_available "${port}"; then
    fail "${label} port ${port} is already in use (set ${env_name} to an open port or unset it for auto-selection)"
  fi

  reserve_port "${port}"
}

resolve_ports() {
  RESERVED_PORTS=()

  if [[ -n "${GRPC_PORT}" ]]; then
    require_available_port "${GRPC_PORT}" "gRPC" "TERMINALS_GRPC_PORT"
  else
    GRPC_PORT="$(find_available_port 50051 200 || true)"
    if [[ -z "${GRPC_PORT}" ]]; then
      fail "unable to find open port for gRPC starting at 50051"
    fi
    reserve_port "${GRPC_PORT}"
  fi

  if [[ -n "${ADMIN_PORT}" ]]; then
    require_available_port "${ADMIN_PORT}" "admin" "TERMINALS_ADMIN_HTTP_PORT"
  else
    ADMIN_PORT="$(find_available_port 50053 200 || true)"
    if [[ -z "${ADMIN_PORT}" ]]; then
      fail "unable to find open port for admin HTTP starting at 50053"
    fi
    reserve_port "${ADMIN_PORT}"
  fi

  if [[ -n "${CONTROL_WS_PORT}" ]]; then
    require_available_port "${CONTROL_WS_PORT}" "control websocket" "TERMINALS_CONTROL_WS_PORT"
  else
    CONTROL_WS_PORT="$(find_available_port 50054 200 || true)"
    if [[ -z "${CONTROL_WS_PORT}" ]]; then
      fail "unable to find open port for control websocket starting at 50054"
    fi
    reserve_port "${CONTROL_WS_PORT}"
  fi

  if [[ -n "${PHOTO_PORT}" ]]; then
    require_available_port "${PHOTO_PORT}" "photo frame" "TERMINALS_PHOTO_FRAME_HTTP_PORT"
  else
    PHOTO_PORT="$(find_available_port 50052 200 || true)"
    if [[ -z "${PHOTO_PORT}" ]]; then
      fail "unable to find open port for photo frame HTTP starting at 50052"
    fi
    reserve_port "${PHOTO_PORT}"
  fi

  if [[ -n "${CONTROL_TCP_PORT}" ]]; then
    require_available_port "${CONTROL_TCP_PORT}" "control TCP" "TERMINALS_CONTROL_TCP_PORT"
  else
    CONTROL_TCP_PORT="$(find_available_port 50055 200 || true)"
    if [[ -z "${CONTROL_TCP_PORT}" ]]; then
      fail "unable to find open port for control TCP starting at 50055"
    fi
    reserve_port "${CONTROL_TCP_PORT}"
  fi

  if [[ "${CLIENT_DEVICE}" == "web-server" ]]; then
    if [[ -n "${CLIENT_WEB_PORT}" ]]; then
      require_available_port "${CLIENT_WEB_PORT}" "web client" "TERMINALS_CLIENT_WEB_PORT"
    else
      CLIENT_WEB_PORT="$(find_available_port 60739 200 || true)"
      if [[ -z "${CLIENT_WEB_PORT}" ]]; then
        fail "unable to find open port for web client starting at 60739"
      fi
    fi
  fi
}

require_cmd() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    fail "required command not found: ${name}"
  fi
}

parse_args() {
  while [[ "$#" -gt 0 ]]; do
    case "$1" in
      --platform)
        shift
        if [[ "$#" -eq 0 ]]; then
          usage
          fail "missing value for --platform"
        fi
        CLIENT_DEVICE="$1"
        ;;
      --client)
        shift
        if [[ "$#" -eq 0 ]]; then
          usage
          fail "missing value for --client"
        fi
        CLIENT_DEVICE="$1"
        ;;
      --skip-bootstrap)
        SKIP_BOOTSTRAP="true"
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        usage
        fail "unknown argument: $1"
        ;;
    esac
    shift
  done

  case "${CLIENT_DEVICE}" in
    web-server|macos|ios|android|linux|windows)
      ;;
    *)
      fail "unsupported client device '${CLIENT_DEVICE}'. Use one of: web-server, macos, ios, android, linux, windows."
      ;;
  esac
}

bootstrap() {
  require_cmd go
  require_cmd flutter
  require_cmd nc

  if [[ "${CLIENT_DEVICE}" == "web-server" ]]; then
    if [[ "${SKIP_BOOTSTRAP}" == "true" && ! -d "${ROOT_DIR}/terminal_client/web" ]]; then
      fail "web support is not configured (missing terminal_client/web). Re-run without --skip-bootstrap or run: (cd terminal_client && flutter create . --platforms=web)"
    fi
  fi

  if [[ "${SKIP_BOOTSTRAP}" == "true" ]]; then
    echo "Skipping bootstrap checks/install (--skip-bootstrap)."
    return 0
  fi

  echo "Bootstrapping dependencies..."
  (
    cd "${ROOT_DIR}/terminal_server"
    go mod download
  )
  (
    cd "${ROOT_DIR}/terminal_client"
    flutter pub get
  )

  if [[ "${CLIENT_DEVICE}" == "web-server" ]]; then
    (
      cd "${ROOT_DIR}/terminal_client"
      flutter config --enable-web
      if [[ ! -d "web" ]]; then
        echo "Enabling Flutter web platform support in terminal_client..."
        flutter create . --platforms=web
      fi
    )
  fi

  if [[ "${CLIENT_DEVICE}" == "macos" ]]; then
    require_cmd xcodebuild
    require_cmd pod
    (
      cd "${ROOT_DIR}/terminal_client/macos"
      env -u DART_DEFINES pod install
    )
  fi
}

start_server() {
  rotate_log "${SERVER_LOG}"
  : >"${SERVER_LOG}"
  echo "Starting server..."
  (
    # Disarm ERR/errtrace inside the redirected subshell. Otherwise the inherited
    # trap would write diagnostics to fd 2 — which is the server log here — and a
    # tail-of-self loop filled the disk with 712 GB of duplicated lines.
    trap - ERR
    set +E
    # Hard cap on any single write stream (RLIMIT_FSIZE). Runaway output gets a
    # SIGXFSZ instead of eating the disk.
    ulimit -f "${LOG_MAX_KB}" 2>/dev/null || true
    cd "${ROOT_DIR}/terminal_server"
    TERMINALS_GRPC_PORT="${GRPC_PORT}" \
    TERMINALS_CONTROL_WS_PORT="${CONTROL_WS_PORT}" \
    TERMINALS_CONTROL_TCP_PORT="${CONTROL_TCP_PORT}" \
    TERMINALS_ADMIN_HTTP_PORT="${ADMIN_PORT}" \
    TERMINALS_PHOTO_FRAME_HTTP_PORT="${PHOTO_PORT}" \
    go run ./cmd/server
  ) >"${SERVER_LOG}" 2>&1 &
  SERVER_PID=$!

  local deadline=$((SECONDS + 45))
  while (( SECONDS < deadline )); do
    if [[ -f "${SERVER_LOG}" ]] && grep -Eq "control service ready" "${SERVER_LOG}"; then
      echo "Server ready on 127.0.0.1:${GRPC_PORT}"
      return 0
    fi
    if ! kill -0 "${SERVER_PID}" >/dev/null 2>&1; then
      wait "${SERVER_PID}" || true
      fail "server exited before reaching ready state"
    fi
    sleep 1
  done

  if ! wait_for_log "${SERVER_LOG}" "control service ready" "1"; then
    fail "server did not reach ready state within timeout"
  fi

  echo "Server ready on 127.0.0.1:${GRPC_PORT}"
}

start_client() {
  rotate_log "${CLIENT_LOG}"
  : >"${CLIENT_LOG}"
  echo "Starting local client (${CLIENT_DEVICE})..."

  if [[ "${CLIENT_DEVICE}" == "web-server" && "${TEST_MODE}" != "true" ]]; then
    CLIENT_FOREGROUND="true"
    local browser_url="http://localhost:${CLIENT_WEB_PORT}"
    echo "Browser client URL: ${browser_url}"
    if [[ "${OPEN_BROWSER}" == "true" ]]; then
      (
        local deadline=$((SECONDS + 120))
        while (( SECONDS < deadline )); do
          if [[ -f "${CLIENT_LOG}" ]] && grep -Eq 'lib/main.dart is being served at http://[^ ]+:[0-9]+' "${CLIENT_LOG}"; then
            if command -v open >/dev/null 2>&1; then
              open "${browser_url}" >/dev/null 2>&1 || true
            elif command -v xdg-open >/dev/null 2>&1; then
              xdg-open "${browser_url}" >/dev/null 2>&1 || true
            fi
            exit 0
          fi
          sleep 1
        done
        exit 0
      ) &
      BROWSER_OPENER_PID=$!
    fi

    set +e
    (
      trap - ERR
      set +E
      ulimit -f "${LOG_MAX_KB}" 2>/dev/null || true
      cd "${ROOT_DIR}/terminal_client"
      flutter run \
        -d web-server \
        --web-port="${CLIENT_WEB_PORT}" \
        --web-hostname="${CLIENT_WEB_HOST}" \
        --dart-define=TERMINALS_CONTROL_WS_PORT="${CONTROL_WS_PORT}" \
        --dart-define=TERMINALS_GRPC_PORT="${GRPC_PORT}"
    ) 2>&1 | tee "${CLIENT_LOG}"
    local client_status=${PIPESTATUS[0]}
    set -e
    if [[ "${client_status}" -ne 0 ]]; then
      fail "client exited with status ${client_status}"
    fi
    return 0
  fi

  (
    trap - ERR
    set +E
    ulimit -f "${LOG_MAX_KB}" 2>/dev/null || true
    cd "${ROOT_DIR}/terminal_client"
    if [[ "${CLIENT_DEVICE}" == "web-server" ]]; then
      flutter run \
        -d web-server \
        --web-port="${CLIENT_WEB_PORT}" \
        --web-hostname="${CLIENT_WEB_HOST}" \
        --dart-define=TERMINALS_CONTROL_WS_PORT="${CONTROL_WS_PORT}" \
        --dart-define=TERMINALS_GRPC_PORT="${GRPC_PORT}"
    else
      flutter run -d "${CLIENT_DEVICE}"
    fi
  ) >"${CLIENT_LOG}" 2>&1 &
  CLIENT_PID=$!

  sleep "${CLIENT_STARTUP_DELAY_SECONDS}"
  if ! kill -0 "${CLIENT_PID}" >/dev/null 2>&1; then
    wait "${CLIENT_PID}" || true
    fail "client exited immediately after launch"
  fi

  if [[ "${CLIENT_DEVICE}" == "web-server" ]]; then
    if ! wait_for_log "${CLIENT_LOG}" 'lib/main.dart is being served at http://localhost:[0-9]+' "120"; then
      fail "web client did not report browser URL within timeout"
    fi
    local browser_url
    browser_url="$(extract_browser_url "${CLIENT_LOG}" || true)"
    if [[ -z "${browser_url}" ]]; then
      fail "web client started but browser URL could not be determined"
    fi
    echo "Browser client URL: ${browser_url}"
  fi
}

monitor_processes() {
  echo "Server log: ${SERVER_LOG}"
  echo "Client log: ${CLIENT_LOG}"

  if [[ "${TEST_MODE}" == "true" ]]; then
    if ! kill -0 "${SERVER_PID}" >/dev/null 2>&1; then
      wait "${SERVER_PID}" || true
      fail "server exited unexpectedly"
    fi
    if ! kill -0 "${CLIENT_PID}" >/dev/null 2>&1; then
      wait "${CLIENT_PID}" || true
      fail "client exited unexpectedly"
    fi
    echo "Test mode: startup checks passed."
    return 0
  fi

  echo "Press Ctrl+C to stop both processes."

  while true; do
    if ! kill -0 "${SERVER_PID}" >/dev/null 2>&1; then
      wait "${SERVER_PID}" || true
      fail "server exited unexpectedly"
    fi

    if ! kill -0 "${CLIENT_PID}" >/dev/null 2>&1; then
      wait "${CLIENT_PID}" || true
      fail "client exited unexpectedly"
    fi

    sleep 1
  done
}

main() {
  mkdir -p "${TMP_DIR}"
  # Rotate, don't truncate — start_server/start_client will rotate again just
  # before each run, but rotating here too means a failed parse_args still
  # preserves the previous run's logs as .1.
  rotate_log "${SERVER_LOG}"
  rotate_log "${CLIENT_LOG}"
  parse_args "$@"
  resolve_ports
  echo "Using ports: grpc=${GRPC_PORT} control_ws=${CONTROL_WS_PORT} admin=${ADMIN_PORT} photo=${PHOTO_PORT} control_tcp=${CONTROL_TCP_PORT}"
  if [[ "${CLIENT_DEVICE}" == "web-server" ]]; then
    echo "Using web client endpoint: http://localhost:${CLIENT_WEB_PORT}"
  fi
  bootstrap
  start_server
  start_client
  if [[ "${CLIENT_FOREGROUND}" == "true" ]]; then
    return 0
  fi
  monitor_processes
}

main "$@"
