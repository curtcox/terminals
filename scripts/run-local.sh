#!/usr/bin/env bash

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP_DIR="${ROOT_DIR}/.tmp"
SERVER_LOG="${TMP_DIR}/run-local-server.log"
CLIENT_LOG="${TMP_DIR}/run-local-client.log"
CLIENT_DIAG_LOG="${TMP_DIR}/run-local-client-diagnostics.log"
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
CONTROL_HTTP_PORT="${TERMINALS_CONTROL_HTTP_PORT:-}"
ADMIN_PORT="${TERMINALS_ADMIN_HTTP_PORT:-}"
PHOTO_PORT="${TERMINALS_PHOTO_FRAME_HTTP_PORT:-}"
CLIENT_WEB_PORT="${TERMINALS_CLIENT_WEB_PORT:-}"
CLIENT_WEB_HOST="${TERMINALS_CLIENT_WEB_HOST:-0.0.0.0}"
BUILD_SHA="${TERMINALS_BUILD_SHA:-}"
BUILD_DATE="${TERMINALS_BUILD_DATE:-}"
CLIENT_DEVICE="web-server"
SKIP_BOOTSTRAP="false"
TEST_MODE="${RUN_LOCAL_TEST_MODE:-false}"
CLIENT_STARTUP_DELAY_SECONDS="${RUN_LOCAL_CLIENT_STARTUP_DELAY_SECONDS:-5}"
OPEN_BROWSER="${RUN_LOCAL_OPEN_BROWSER:-true}"

export PATH="${ROOT_DIR}/.bin:${ROOT_DIR}/.sdk/flutter/bin:${PATH}"
export FLUTTER_SUPPRESS_ANALYTICS="${FLUTTER_SUPPRESS_ANALYTICS:-true}"
export DART_SUPPRESS_ANALYTICS="${DART_SUPPRESS_ANALYTICS:-true}"
export COCOAPODS_DISABLE_STATS="${COCOAPODS_DISABLE_STATS:-true}"

SERVER_PID=""
CLIENT_PID=""
BROWSER_OPENER_PID=""
CLIENT_FOREGROUND="false"
HAS_ERROR="false"
RESERVED_PORTS=()
CLIENT_RESTART_ATTEMPTS=0
MAX_CLIENT_RESTART_ATTEMPTS="${RUN_LOCAL_CLIENT_RESTART_ATTEMPTS:-3}"
CLIENT_CONTROL_HOST="127.0.0.1"
CLIENT_AUTO_CONNECT_ON_STARTUP="true"
CLIENT_DART_DEFINE_ARGS=()

# shellcheck source=lib/run-local-impl-core.sh
source "${ROOT_DIR}/scripts/lib/run-local-impl-core.sh"
# shellcheck source=lib/run-local-impl-run.sh
source "${ROOT_DIR}/scripts/lib/run-local-impl-run.sh"

trap 'on_err ${LINENO}' ERR
trap 'on_exit $?' EXIT

main() {
  mkdir -p "${TMP_DIR}"
  # Rotate, don't truncate — start_server/start_client will rotate again just
  # before each run, but rotating here too means a failed parse_args still
  # preserves the previous run's logs as .1.
  rotate_log "${SERVER_LOG}"
  rotate_log "${CLIENT_LOG}"
  parse_args "$@"
  resolve_ports
  resolve_build_metadata
  configure_client_dart_defines
  echo "Using ports: grpc=${GRPC_PORT} control_ws=${CONTROL_WS_PORT} admin=${ADMIN_PORT} photo=${PHOTO_PORT} control_tcp=${CONTROL_TCP_PORT} control_http=${CONTROL_HTTP_PORT}"
  echo "Using build metadata: sha=${BUILD_SHA} date=${BUILD_DATE}"
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
