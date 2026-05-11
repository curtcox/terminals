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

# shellcheck source=lib/ui-inspect-run-impl.sh
source "${ROOT_DIR}/scripts/lib/ui-inspect-run-impl.sh"

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
