#!/usr/bin/env bash
# scripts/ui-inspect-run.sh — start/stop the Flutter web + macOS clients for UI
# inspection. Runs the two flavors of run-local.sh sequentially (not in parallel)
# so they cannot race on the shared .tmp/run-local-*.log rotation. Records PIDs
# and URLs so the inspection tooling and the `stop` subcommand have a
# deterministic handle on the processes instead of relying on `pkill` or
# process-listing, which can be restricted in sandboxed environments.

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE_DIR="${ROOT_DIR}/.tmp/ui-inspect"
WEB_LOG="${STATE_DIR}/web.log"
MACOS_LOG="${STATE_DIR}/macos.log"
STATE_FILE="${STATE_DIR}/state"

READY_TIMEOUT="${UI_INSPECT_READY_TIMEOUT:-240}"
SKIP_WEB="${UI_INSPECT_SKIP_WEB:-}"
SKIP_MACOS="${UI_INSPECT_SKIP_MACOS:-}"

WEB_PID=""
MACOS_PID=""
WEB_URL=""

log() {
  printf '[ui-inspect] %s\n' "$*"
}

err() {
  printf '[ui-inspect] ERROR: %s\n' "$*" >&2
}

usage() {
  cat <<'EOF'
Usage: ./scripts/ui-inspect-run.sh <command>

Commands:
  start    Start the web client, then the macOS client (sequentially). Prints
           the web URL, both PIDs, and both log paths. Returns once both flavors
           have printed their readiness markers.
  stop     Stop the clients started by `start`, using the PIDs recorded in
           .tmp/ui-inspect/state. Falls back to ./scripts/stop-server.sh and a
           best-effort lsof-based port sweep if the PIDs are missing/stale.
  status   Print current recorded state (PIDs, URL, log paths) and whether each
           PID is still alive.
  help     Show this help.

Environment:
  UI_INSPECT_SKIP_WEB=1        Skip the web client (start macOS only).
  UI_INSPECT_SKIP_MACOS=1      Skip the macOS client (start web only).
  UI_INSPECT_READY_TIMEOUT=N   Seconds to wait for each flavor's readiness
                               marker (default 240).

Notes:
  * The two run-local.sh invocations share .tmp/run-local-*.log. Running them
    in parallel races the log-rotation `mv file.N file.N+1` chain. This script
    avoids that by starting web first, waiting for its readiness marker, then
    starting macOS.
  * On macOS the native client build takes 1-2 min the first time. Bump
    UI_INSPECT_READY_TIMEOUT if your machine is slow.
  * localhost bind/build may require unsandboxed/elevated execution. If you see
    `listen ... bind: operation not permitted`, re-run outside the sandbox.
EOF
}

ensure_state_dir() {
  mkdir -p "${STATE_DIR}"
}

# Wait until `regex` appears in `file`, or `timeout_seconds` elapses. Also bail
# if the pid we're watching has exited — no point waiting on a dead process.
wait_for_log() {
  local file="$1"
  local regex="$2"
  local timeout_seconds="$3"
  local watch_pid="${4:-}"
  local deadline=$((SECONDS + timeout_seconds))

  while (( SECONDS < deadline )); do
    if [[ -f "${file}" ]] && grep -Eq "${regex}" "${file}"; then
      return 0
    fi
    if [[ -n "${watch_pid}" ]] && ! kill -0 "${watch_pid}" >/dev/null 2>&1; then
      return 2
    fi
    sleep 1
  done
  return 1
}

extract_web_url() {
  # `run-local.sh` prints: `Browser client URL: http://localhost:<port>`
  grep -Eo 'Browser client URL: http://[^[:space:]]+' "${WEB_LOG}" \
    | tail -n 1 \
    | awk '{print $NF}'
}

write_state() {
  {
    printf 'WEB_PID=%s\n' "${WEB_PID}"
    printf 'MACOS_PID=%s\n' "${MACOS_PID}"
    printf 'WEB_URL=%s\n' "${WEB_URL}"
    printf 'WEB_LOG=%s\n' "${WEB_LOG}"
    printf 'MACOS_LOG=%s\n' "${MACOS_LOG}"
    printf 'STARTED_AT=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  } >"${STATE_FILE}"
}

read_state() {
  WEB_PID=""
  MACOS_PID=""
  WEB_URL=""
  if [[ -f "${STATE_FILE}" ]]; then
    # shellcheck disable=SC1090
    source "${STATE_FILE}"
  fi
}

pid_alive() {
  local pid="$1"
  [[ -n "${pid}" ]] && kill -0 "${pid}" >/dev/null 2>&1
}

# Kill a process and all of its descendants. `pgrep -P` is the primary path;
# when pgrep is restricted we still send SIGTERM to the parent PID, and the
# caller falls back to stop-server.sh + port-based cleanup.
kill_tree() {
  local pid="$1"
  if [[ -z "${pid}" ]]; then
    return 0
  fi
  if ! pid_alive "${pid}"; then
    return 0
  fi
  if command -v pgrep >/dev/null 2>&1; then
    local children
    children="$(pgrep -P "${pid}" 2>/dev/null || true)"
    local child
    for child in ${children}; do
      kill_tree "${child}"
    done
  fi
  kill "${pid}" >/dev/null 2>&1 || true
  local waited=0
  while pid_alive "${pid}"; do
    if (( waited >= 10 )); then
      kill -9 "${pid}" >/dev/null 2>&1 || true
      break
    fi
    sleep 1
    waited=$((waited + 1))
  done
}

start_web() {
  log "Starting web client (./scripts/run-local.sh --platform web-server)..."
  : >"${WEB_LOG}"
  (
    cd "${ROOT_DIR}"
    RUN_LOCAL_OPEN_BROWSER=false \
      nohup ./scripts/run-local.sh --skip-bootstrap --platform web-server \
      >>"${WEB_LOG}" 2>&1 &
    echo $! >"${STATE_DIR}/web.pid"
  )
  WEB_PID="$(cat "${STATE_DIR}/web.pid")"
  log "  web pid=${WEB_PID}, log=${WEB_LOG}"

  if ! wait_for_log "${WEB_LOG}" 'Browser client URL:' "${READY_TIMEOUT}" "${WEB_PID}"; then
    err "web client did not reach readiness within ${READY_TIMEOUT}s (see ${WEB_LOG})"
    return 1
  fi
  WEB_URL="$(extract_web_url || true)"
  if [[ -z "${WEB_URL}" ]]; then
    err "web client logged readiness but no URL was extracted"
    return 1
  fi
  log "  web ready at ${WEB_URL}"
}

start_macos() {
  log "Starting macOS client (./scripts/run-local.sh --platform macos)..."
  : >"${MACOS_LOG}"
  (
    cd "${ROOT_DIR}"
    RUN_LOCAL_OPEN_BROWSER=false \
      nohup ./scripts/run-local.sh --skip-bootstrap --platform macos \
      >>"${MACOS_LOG}" 2>&1 &
    echo $! >"${STATE_DIR}/macos.pid"
  )
  MACOS_PID="$(cat "${STATE_DIR}/macos.pid")"
  log "  macOS pid=${MACOS_PID}, log=${MACOS_LOG} (first build takes 1-2 min)"

  # `Press Ctrl+C to stop both processes.` is printed by monitor_processes only
  # after the macOS .app has been built and launched successfully.
  if ! wait_for_log "${MACOS_LOG}" 'Press Ctrl\+C to stop both processes' "${READY_TIMEOUT}" "${MACOS_PID}"; then
    err "macOS client did not reach readiness within ${READY_TIMEOUT}s (see ${MACOS_LOG})"
    return 1
  fi
  log "  macOS client launched"
}

cmd_start() {
  ensure_state_dir

  if [[ -f "${STATE_FILE}" ]]; then
    read_state
    if pid_alive "${WEB_PID}" || pid_alive "${MACOS_PID}"; then
      err "a previous ui-inspect session is still running (web=${WEB_PID} macos=${MACOS_PID}). Run './scripts/ui-inspect-run.sh stop' first."
      return 1
    fi
  fi

  WEB_PID=""
  MACOS_PID=""
  WEB_URL=""

  if [[ -z "${SKIP_WEB}" ]]; then
    start_web
  fi
  if [[ -z "${SKIP_MACOS}" ]]; then
    start_macos
  fi

  write_state
  log ""
  log "READY"
  [[ -n "${WEB_URL}" ]] && log "  web url : ${WEB_URL}"
  [[ -n "${WEB_PID}" ]] && log "  web pid : ${WEB_PID}"
  [[ -n "${MACOS_PID}" ]] && log "  mac pid : ${MACOS_PID}"
  log "  web log : ${WEB_LOG}"
  log "  mac log : ${MACOS_LOG}"
  log "  state   : ${STATE_FILE}"
  log ""
  log "Run './scripts/ui-inspect-run.sh stop' when done."
}

# Port-based fallback: sweep the ports stop-server.sh knows about. This is the
# last-resort path when PID-based cleanup has done its part and we still want
# to be sure the server is gone.
fallback_stop_server() {
  if [[ -x "${ROOT_DIR}/scripts/stop-server.sh" ]]; then
    log "Running stop-server.sh as fallback..."
    "${ROOT_DIR}/scripts/stop-server.sh" || true
  fi
}

cmd_stop() {
  read_state

  if [[ -z "${WEB_PID}" && -z "${MACOS_PID}" ]]; then
    log "No recorded ui-inspect PIDs; running fallback cleanup."
    fallback_stop_server
    return 0
  fi

  if [[ -n "${MACOS_PID}" ]]; then
    log "Stopping macOS client pid=${MACOS_PID}..."
    kill_tree "${MACOS_PID}"
  fi
  if [[ -n "${WEB_PID}" ]]; then
    log "Stopping web client pid=${WEB_PID}..."
    kill_tree "${WEB_PID}"
  fi

  fallback_stop_server

  rm -f "${STATE_FILE}" "${STATE_DIR}/web.pid" "${STATE_DIR}/macos.pid"
  log "Stopped."
}

cmd_status() {
  if [[ ! -f "${STATE_FILE}" ]]; then
    log "No ui-inspect session recorded."
    return 0
  fi
  read_state
  log "Recorded state:"
  log "  web pid : ${WEB_PID:-<none>} ($(pid_alive "${WEB_PID}" && echo alive || echo gone))"
  log "  mac pid : ${MACOS_PID:-<none>} ($(pid_alive "${MACOS_PID}" && echo alive || echo gone))"
  log "  web url : ${WEB_URL:-<none>}"
  log "  web log : ${WEB_LOG}"
  log "  mac log : ${MACOS_LOG}"
}

on_err() {
  local exit_code="$?"
  err "ui-inspect-run.sh failed (exit ${exit_code})"
  # Best-effort cleanup of anything we started in this invocation before
  # bailing, so a partial start doesn't leave a stray process behind.
  if [[ -n "${WEB_PID}" ]] && pid_alive "${WEB_PID}"; then
    kill_tree "${WEB_PID}"
  fi
  if [[ -n "${MACOS_PID}" ]] && pid_alive "${MACOS_PID}"; then
    kill_tree "${MACOS_PID}"
  fi
  exit "${exit_code}"
}
trap on_err ERR

main() {
  if [[ "$#" -lt 1 ]]; then
    usage
    exit 1
  fi
  local cmd="$1"
  shift || true
  case "${cmd}" in
    start)  cmd_start "$@" ;;
    stop)   cmd_stop "$@" ;;
    status) cmd_status "$@" ;;
    -h|--help|help) usage ;;
    *)
      err "unknown command: ${cmd}"
      usage
      exit 1
      ;;
  esac
}

main "$@"
