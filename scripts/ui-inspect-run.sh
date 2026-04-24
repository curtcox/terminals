#!/usr/bin/env bash
# scripts/ui-inspect-run.sh — start/stop the Flutter web + macOS clients for UI
# inspection. Runs the two flavors of run-local.sh sequentially (not in parallel)
# so they cannot race on the shared .tmp/run-local-*.log rotation. Records PIDs
# with identity metadata so the `stop` subcommand has a deterministic, safe
# handle on the processes instead of relying on `pkill`/process-listing, which
# can be restricted in sandboxed environments. Before killing any recorded PID
# we verify that the live process still matches what we recorded (command-line
# substring + start-time sanity check) so a reused PID from an unrelated process
# never gets signalled.

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE_DIR="${ROOT_DIR}/.tmp/ui-inspect"
WEB_LOG="${STATE_DIR}/web.log"
MACOS_LOG="${STATE_DIR}/macos.log"
STATE_FILE="${STATE_DIR}/state"

READY_TIMEOUT="${UI_INSPECT_READY_TIMEOUT:-240}"
SKIP_WEB="${UI_INSPECT_SKIP_WEB:-}"
SKIP_MACOS="${UI_INSPECT_SKIP_MACOS:-}"

# Test hooks. When set, start_web/start_macos launch these commands instead of
# run-local.sh. The smoke test uses these to exercise lifecycle behaviour
# without building a Flutter app.
WEB_CMD_OVERRIDE="${UI_INSPECT_WEB_CMD:-}"
MACOS_CMD_OVERRIDE="${UI_INSPECT_MACOS_CMD:-}"
WEB_READY_REGEX="${UI_INSPECT_WEB_READY_REGEX:-Browser client URL:}"
MACOS_READY_REGEX="${UI_INSPECT_MACOS_READY_REGEX:-Press Ctrl\+C to stop both processes}"

# Identity patterns recorded with each PID. Verified against `ps -o args=` at
# stop time. Keep them as fixed substrings (no regex chars) — `verify_identity`
# uses substring matching.
WEB_CMD_PATTERN_DEFAULT="run-local.sh --skip-bootstrap --platform web-server"
MACOS_CMD_PATTERN_DEFAULT="run-local.sh --skip-bootstrap --platform macos"

WEB_PID=""
MACOS_PID=""
WEB_URL=""
WEB_CMD_PATTERN=""
MACOS_CMD_PATTERN=""
WEB_STARTED_AT=""
MACOS_STARTED_AT=""

log() {
  printf '[ui-inspect] %s\n' "$*"
}

err() {
  printf '[ui-inspect] ERROR: %s\n' "$*" >&2
}

warn() {
  printf '[ui-inspect] WARN: %s\n' "$*" >&2
}

usage() {
  cat <<'EOF'
Usage: ./scripts/ui-inspect-run.sh <command>

Commands:
  start    Start the web client, then the macOS client (sequentially). Prints
           the web URL, both PIDs, and both log paths. Returns once both flavors
           have printed their readiness markers.
  stop     Stop the clients started by `start`, using the PIDs recorded in
           .tmp/ui-inspect/state. Verifies each PID still matches its recorded
           command/start-time before killing — a reused or unrelated PID is
           skipped with a warning. Falls back to ./scripts/stop-server.sh.
  status   Print current recorded state (PIDs, URL, log paths) and whether each
           PID is still alive and identity-verified.
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

# Epoch seconds for a PID's start time. Empty string if the PID is gone or ps
# can't report it. Used as a strong identity marker: a reused PID will have a
# start time later than what we recorded at spawn.
process_start_epoch() {
  local pid="$1"
  [[ -n "${pid}" ]] || { printf ''; return; }
  command -v ps >/dev/null 2>&1 || { printf ''; return; }
  # -o lstart= works on macOS and Linux. Format: "Thu Apr 23 15:30:58 2026".
  local lstart
  lstart="$(ps -o lstart= -p "${pid}" 2>/dev/null || true)"
  lstart="${lstart#"${lstart%%[![:space:]]*}"}"
  lstart="${lstart%"${lstart##*[![:space:]]}"}"
  [[ -n "${lstart}" ]] || { printf ''; return; }
  # `date -j -f` on macOS, `date -d` on GNU.
  local epoch=""
  if date -j -f "%a %b %e %T %Y" "${lstart}" +%s >/dev/null 2>&1; then
    epoch="$(date -j -f "%a %b %e %T %Y" "${lstart}" +%s 2>/dev/null || true)"
  elif date -d "${lstart}" +%s >/dev/null 2>&1; then
    epoch="$(date -d "${lstart}" +%s 2>/dev/null || true)"
  fi
  printf '%s' "${epoch}"
}

# Verify that `pid` is still the process we spawned. Two checks:
#   1. ps args contains the expected substring.
#   2. ps start-time is not significantly after our recorded start time (which
#      would indicate PID reuse).
# Return codes:
#   0  identity matches — safe to kill.
#   1  identity mismatch — skip kill.
#   2  unable to verify (ps unavailable / restricted) — skip kill.
verify_identity() {
  local pid="$1"
  local pattern="$2"
  local recorded_epoch="${3:-}"

  [[ -n "${pid}" ]] || return 1
  [[ "${pid}" =~ ^[0-9]+$ ]] || return 1
  if ! command -v ps >/dev/null 2>&1; then
    return 2
  fi
  local args
  args="$(ps -o args= -p "${pid}" 2>/dev/null || true)"
  if [[ -z "${args}" ]]; then
    # Process is gone, or ps refused. Either way, not safe/necessary to kill.
    return 1
  fi
  if [[ -n "${pattern}" && "${args}" != *"${pattern}"* ]]; then
    return 1
  fi

  if [[ -n "${recorded_epoch}" && "${recorded_epoch}" =~ ^[0-9]+$ ]]; then
    local live_epoch
    live_epoch="$(process_start_epoch "${pid}")"
    if [[ "${live_epoch}" =~ ^[0-9]+$ ]]; then
      # Allow up to 5s clock skew between spawn-record and ps snapshot.
      local delta=$(( live_epoch - recorded_epoch ))
      if (( delta > 5 )); then
        return 1
      fi
    fi
  fi
  return 0
}

# Convert an ISO-8601 Z timestamp (as produced by `date -u +%Y-%m-%dT%H:%M:%SZ`)
# to epoch seconds. Best-effort; returns empty on failure. Used to validate
# STARTED_AT fields read from the state file.
iso_to_epoch() {
  local iso="$1"
  [[ -n "${iso}" ]] || { printf ''; return; }
  if date -j -u -f "%Y-%m-%dT%H:%M:%SZ" "${iso}" +%s >/dev/null 2>&1; then
    date -j -u -f "%Y-%m-%dT%H:%M:%SZ" "${iso}" +%s 2>/dev/null || true
  elif date -u -d "${iso}" +%s >/dev/null 2>&1; then
    date -u -d "${iso}" +%s 2>/dev/null || true
  fi
}

write_state() {
  # Fail loudly if any field contains a newline — the reader is line-oriented.
  local k v
  for k in WEB_PID MACOS_PID WEB_URL WEB_CMD_PATTERN MACOS_CMD_PATTERN WEB_STARTED_AT MACOS_STARTED_AT; do
    v="${!k-}"
    if [[ "${v}" == *$'\n'* ]]; then
      err "refusing to write state: ${k} contains newline"
      return 1
    fi
  done
  # Log paths are derived constants (STATE_DIR/*), not persisted — no point
  # serialising them and every extra field is another malformed-input vector.
  {
    printf 'WEB_PID=%s\n'           "${WEB_PID}"
    printf 'MACOS_PID=%s\n'         "${MACOS_PID}"
    printf 'WEB_URL=%s\n'           "${WEB_URL}"
    printf 'WEB_CMD_PATTERN=%s\n'   "${WEB_CMD_PATTERN}"
    printf 'MACOS_CMD_PATTERN=%s\n' "${MACOS_CMD_PATTERN}"
    printf 'WEB_STARTED_AT=%s\n'    "${WEB_STARTED_AT}"
    printf 'MACOS_STARTED_AT=%s\n'  "${MACOS_STARTED_AT}"
  } >"${STATE_FILE}"
}

# Strict KV parser. Replaces the previous `source "${STATE_FILE}"` which
# executed the state file as shell — any write to that path could run code.
# Now: parse line-by-line, validate each value against a per-key regex, drop
# anything that doesn't match. Malformed files just produce empty fields and
# are treated as stale.
parse_state_file() {
  local file="$1"
  WEB_PID=""
  MACOS_PID=""
  WEB_URL=""
  WEB_CMD_PATTERN=""
  MACOS_CMD_PATTERN=""
  WEB_STARTED_AT=""
  MACOS_STARTED_AT=""

  [[ -f "${file}" ]] || return 0

  # Refuse pathological files (symlink, too large, non-regular).
  if [[ -L "${file}" ]]; then
    warn "state file is a symlink; refusing to parse: ${file}"
    return 0
  fi
  local size
  size="$(wc -c <"${file}" 2>/dev/null | tr -d ' ')"
  if [[ -n "${size}" && "${size}" =~ ^[0-9]+$ ]] && (( size > 65536 )); then
    warn "state file unexpectedly large (${size} bytes); treating as stale"
    return 0
  fi

  local line key value
  while IFS= read -r line || [[ -n "${line}" ]]; do
    [[ -z "${line}" ]] && continue
    [[ "${line}" =~ ^# ]] && continue
    if [[ "${line}" =~ ^([A-Z_][A-Z0-9_]*)=(.*)$ ]]; then
      key="${BASH_REMATCH[1]}"
      value="${BASH_REMATCH[2]}"
    else
      continue
    fi
    case "${key}" in
      WEB_PID|MACOS_PID)
        if [[ "${value}" =~ ^[0-9]+$ ]] && (( value > 0 )) && (( value < 4194304 )); then
          printf -v "${key}" '%s' "${value}"
        fi
        ;;
      WEB_URL)
        if [[ "${value}" =~ ^https?://[A-Za-z0-9._:/?#=%@-]+$ ]]; then
          WEB_URL="${value}"
        fi
        ;;
      WEB_LOG|MACOS_LOG)
        # Legacy field from earlier format. Ignore — log paths are derived
        # constants and never read from state.
        ;;
      WEB_CMD_PATTERN|MACOS_CMD_PATTERN)
        # Restrict to a predictable substring alphabet. Command patterns we
        # record are literal argv fragments like
        # "run-local.sh --skip-bootstrap --platform web-server".
        local cmd_re='^[A-Za-z0-9._/+=:,@ -]+$'
        if (( ${#value} <= 200 )) && [[ "${value}" =~ $cmd_re ]]; then
          printf -v "${key}" '%s' "${value}"
        fi
        ;;
      WEB_STARTED_AT|MACOS_STARTED_AT)
        if [[ "${value}" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$ ]]; then
          printf -v "${key}" '%s' "${value}"
        fi
        ;;
      *)
        # Unknown key — ignore.
        ;;
    esac
  done <"${file}"
  return 0
}

read_state() {
  parse_state_file "${STATE_FILE}"
}

pid_alive() {
  local pid="$1"
  [[ -n "${pid}" ]] && [[ "${pid}" =~ ^[0-9]+$ ]] && kill -0 "${pid}" >/dev/null 2>&1
}

# Kill a process tree rooted at `pid`, but only after verifying the root's
# identity. Children are discovered from the verified parent via `pgrep -P` —
# their identity is inherited from the parent match. Without verification we
# won't signal anything.
kill_tree_verified() {
  local pid="$1"
  local pattern="$2"
  local recorded_iso="${3:-}"
  local label="${4:-process}"

  if [[ -z "${pid}" ]]; then
    return 0
  fi
  if ! pid_alive "${pid}"; then
    log "  ${label} pid=${pid} already gone"
    return 0
  fi

  local recorded_epoch=""
  recorded_epoch="$(iso_to_epoch "${recorded_iso}")"

  local rc=0
  verify_identity "${pid}" "${pattern}" "${recorded_epoch}" || rc=$?
  case "${rc}" in
    0) ;;
    1)
      warn "${label} pid=${pid} no longer matches recorded identity (pattern='${pattern}'); skipping kill"
      return 0
      ;;
    2)
      warn "${label} pid=${pid}: cannot verify identity (ps unavailable); skipping kill for safety"
      return 0
      ;;
  esac

  _kill_tree_unchecked "${pid}"
}

# Internal: kill descendants-first, then the pid, then SIGKILL after 10s. Only
# call this after verify_identity passes for the root pid.
_kill_tree_unchecked() {
  local pid="$1"
  if ! pid_alive "${pid}"; then
    return 0
  fi
  if command -v pgrep >/dev/null 2>&1; then
    local children child
    children="$(pgrep -P "${pid}" 2>/dev/null || true)"
    for child in ${children}; do
      _kill_tree_unchecked "${child}"
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

# Record start-time metadata for a freshly-spawned pid. Called right after we
# fork so the recorded epoch is within a second or two of the true value.
record_start_metadata() {
  local which="$1"   # WEB | MACOS
  local pid="$2"
  local iso
  iso="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  case "${which}" in
    WEB)   WEB_STARTED_AT="${iso}" ;;
    MACOS) MACOS_STARTED_AT="${iso}" ;;
  esac
}

start_web() {
  local cmd_label="${WEB_CMD_OVERRIDE:-./scripts/run-local.sh --skip-bootstrap --platform web-server}"
  WEB_CMD_PATTERN="${WEB_CMD_OVERRIDE:+ui-inspect-test-web}"
  WEB_CMD_PATTERN="${WEB_CMD_PATTERN:-${WEB_CMD_PATTERN_DEFAULT}}"

  log "Starting web client (${cmd_label})..."
  : >"${WEB_LOG}"
  (
    cd "${ROOT_DIR}"
    if [[ -n "${WEB_CMD_OVERRIDE}" ]]; then
      # shellcheck disable=SC2086
      nohup bash -c "${WEB_CMD_OVERRIDE}" >>"${WEB_LOG}" 2>&1 &
    else
      RUN_LOCAL_OPEN_BROWSER=false \
        nohup ./scripts/run-local.sh --skip-bootstrap --platform web-server \
        >>"${WEB_LOG}" 2>&1 &
    fi
    echo $! >"${STATE_DIR}/web.pid"
  )
  WEB_PID="$(cat "${STATE_DIR}/web.pid")"
  record_start_metadata WEB "${WEB_PID}"
  log "  web pid=${WEB_PID}, log=${WEB_LOG}"

  if ! wait_for_log "${WEB_LOG}" "${WEB_READY_REGEX}" "${READY_TIMEOUT}" "${WEB_PID}"; then
    err "web client did not reach readiness within ${READY_TIMEOUT}s (see ${WEB_LOG})"
    return 1
  fi
  if [[ -z "${WEB_CMD_OVERRIDE}" ]]; then
    WEB_URL="$(extract_web_url || true)"
    if [[ -z "${WEB_URL}" ]]; then
      err "web client logged readiness but no URL was extracted"
      return 1
    fi
    log "  web ready at ${WEB_URL}"
  else
    WEB_URL=""
    log "  web ready (test override)"
  fi
}

start_macos() {
  local cmd_label="${MACOS_CMD_OVERRIDE:-./scripts/run-local.sh --skip-bootstrap --platform macos}"
  MACOS_CMD_PATTERN="${MACOS_CMD_OVERRIDE:+ui-inspect-test-macos}"
  MACOS_CMD_PATTERN="${MACOS_CMD_PATTERN:-${MACOS_CMD_PATTERN_DEFAULT}}"

  log "Starting macOS client (${cmd_label})..."
  : >"${MACOS_LOG}"
  (
    cd "${ROOT_DIR}"
    if [[ -n "${MACOS_CMD_OVERRIDE}" ]]; then
      nohup bash -c "${MACOS_CMD_OVERRIDE}" >>"${MACOS_LOG}" 2>&1 &
    else
      RUN_LOCAL_OPEN_BROWSER=false \
        nohup ./scripts/run-local.sh --skip-bootstrap --platform macos \
        >>"${MACOS_LOG}" 2>&1 &
    fi
    echo $! >"${STATE_DIR}/macos.pid"
  )
  MACOS_PID="$(cat "${STATE_DIR}/macos.pid")"
  record_start_metadata MACOS "${MACOS_PID}"
  log "  macOS pid=${MACOS_PID}, log=${MACOS_LOG} (first build takes 1-2 min)"

  if ! wait_for_log "${MACOS_LOG}" "${MACOS_READY_REGEX}" "${READY_TIMEOUT}" "${MACOS_PID}"; then
    err "macOS client did not reach readiness within ${READY_TIMEOUT}s (see ${MACOS_LOG})"
    return 1
  fi
  log "  macOS client launched"
}

# Clean up anything we spawned in this invocation and any stale helper files.
# Called from on_err and also from cmd_start before bailing out so a partial
# start never leaves a half-written state file or orphaned pid file behind.
cleanup_partial_start() {
  if [[ -n "${WEB_PID}" ]] && pid_alive "${WEB_PID}"; then
    # Identity is guaranteed — we literally just spawned this pid a few seconds
    # ago — so we can call the unchecked killer directly.
    _kill_tree_unchecked "${WEB_PID}"
  fi
  if [[ -n "${MACOS_PID}" ]] && pid_alive "${MACOS_PID}"; then
    _kill_tree_unchecked "${MACOS_PID}"
  fi
  rm -f "${STATE_FILE}" "${STATE_DIR}/web.pid" "${STATE_DIR}/macos.pid"
}

cmd_start() {
  ensure_state_dir

  if [[ -f "${STATE_FILE}" ]]; then
    read_state
    local still_alive=0
    if [[ -n "${WEB_PID}" ]] && pid_alive "${WEB_PID}"; then
      if verify_identity "${WEB_PID}" "${WEB_CMD_PATTERN}" "$(iso_to_epoch "${WEB_STARTED_AT}")"; then
        still_alive=1
      fi
    fi
    if [[ -n "${MACOS_PID}" ]] && pid_alive "${MACOS_PID}"; then
      if verify_identity "${MACOS_PID}" "${MACOS_CMD_PATTERN}" "$(iso_to_epoch "${MACOS_STARTED_AT}")"; then
        still_alive=1
      fi
    fi
    if (( still_alive )); then
      err "a previous ui-inspect session is still running (web=${WEB_PID} macos=${MACOS_PID}). Run './scripts/ui-inspect-run.sh stop' first."
      return 1
    fi
    # State exists but nothing recognisable is alive — stale. Clear it so we
    # start from a clean slate and never conflate old PIDs with new ones.
    log "clearing stale state file from previous session"
    rm -f "${STATE_FILE}" "${STATE_DIR}/web.pid" "${STATE_DIR}/macos.pid"
  fi

  WEB_PID=""
  MACOS_PID=""
  WEB_URL=""
  WEB_CMD_PATTERN=""
  MACOS_CMD_PATTERN=""
  WEB_STARTED_AT=""
  MACOS_STARTED_AT=""

  if [[ -z "${SKIP_WEB}" ]]; then
    if ! start_web; then
      cleanup_partial_start
      return 1
    fi
  fi
  if [[ -z "${SKIP_MACOS}" ]]; then
    if ! start_macos; then
      cleanup_partial_start
      return 1
    fi
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
  if [[ -n "${UI_INSPECT_SKIP_FALLBACK:-}" ]]; then
    return 0
  fi
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
    rm -f "${STATE_FILE}" "${STATE_DIR}/web.pid" "${STATE_DIR}/macos.pid"
    return 0
  fi

  if [[ -n "${MACOS_PID}" ]]; then
    log "Stopping macOS client pid=${MACOS_PID} (pattern='${MACOS_CMD_PATTERN}')..."
    kill_tree_verified "${MACOS_PID}" "${MACOS_CMD_PATTERN}" "${MACOS_STARTED_AT}" "macOS"
  fi
  if [[ -n "${WEB_PID}" ]]; then
    log "Stopping web client pid=${WEB_PID} (pattern='${WEB_CMD_PATTERN}')..."
    kill_tree_verified "${WEB_PID}" "${WEB_CMD_PATTERN}" "${WEB_STARTED_AT}" "web"
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
  local web_state="gone" mac_state="gone"
  if pid_alive "${WEB_PID}"; then
    if verify_identity "${WEB_PID}" "${WEB_CMD_PATTERN}" "$(iso_to_epoch "${WEB_STARTED_AT}")"; then
      web_state="alive"
    else
      web_state="alive-but-mismatched"
    fi
  fi
  if pid_alive "${MACOS_PID}"; then
    if verify_identity "${MACOS_PID}" "${MACOS_CMD_PATTERN}" "$(iso_to_epoch "${MACOS_STARTED_AT}")"; then
      mac_state="alive"
    else
      mac_state="alive-but-mismatched"
    fi
  fi
  log "Recorded state:"
  log "  web pid : ${WEB_PID:-<none>} (${web_state})"
  log "  mac pid : ${MACOS_PID:-<none>} (${mac_state})"
  log "  web url : ${WEB_URL:-<none>}"
  log "  web log : ${WEB_LOG}"
  log "  mac log : ${MACOS_LOG}"
  log "  started : web=${WEB_STARTED_AT:-<none>} macos=${MACOS_STARTED_AT:-<none>}"
}

on_err() {
  local exit_code="$?"
  err "ui-inspect-run.sh failed (exit ${exit_code})"
  cleanup_partial_start
  exit "${exit_code}"
}

main() {
  trap on_err ERR
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

# Only run main when executed directly; tests source this file and call the
# internal functions (parse_state_file, verify_identity, kill_tree_verified, …)
# without spawning a real Flutter build.
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main "$@"
fi
