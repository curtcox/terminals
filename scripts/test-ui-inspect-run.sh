#!/usr/bin/env bash
# scripts/test-ui-inspect-run.sh — smoke tests for ui-inspect-run.sh.
#
# Covers the four safety properties the script needs to hold:
#   1. Normal start/stop lifecycle writes and clears state correctly.
#   2. A stale state file whose recorded PID has been reused by an unrelated
#      process must NOT cause that unrelated process to be killed.
#   3. A malformed state file must be treated as stale, not executed.
#   4. On partial start failure no risky state is left behind.
#
# Run: ./scripts/test-ui-inspect-run.sh

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT="${ROOT_DIR}/scripts/ui-inspect-run.sh"

SANDBOX=""
FAILS=0

cleanup() {
  if [[ -n "${SANDBOX}" && -d "${SANDBOX}" ]]; then
    # Best-effort: kill anything still running that we spawned inside sandbox
    # pid files. We never signal pids we didn't spawn.
    local f
    for f in "${SANDBOX}/pids"/*; do
      [[ -f "${f}" ]] || continue
      local pid
      pid="$(cat "${f}" 2>/dev/null || true)"
      if [[ "${pid}" =~ ^[0-9]+$ ]] && kill -0 "${pid}" 2>/dev/null; then
        kill "${pid}" 2>/dev/null || true
      fi
    done
    rm -rf "${SANDBOX}"
  fi
}
trap cleanup EXIT

pass() { printf '  PASS: %s\n' "$1"; }
fail() { printf '  FAIL: %s\n' "$1" >&2; FAILS=$((FAILS + 1)); }

expect() {
  local msg="$1"; shift
  if "$@"; then pass "${msg}"; else fail "${msg}"; fi
}

mk_sandbox() {
  SANDBOX="$(mktemp -d -t ui-inspect-test.XXXXXX)"
  mkdir -p "${SANDBOX}/pids"
}

# Spawn a background process we control and set VICTIM_PID. We don't use
# command substitution to return the pid because bash 3.2 on macOS may reap
# background children started inside a `$(...)` subshell when that subshell
# exits. Spawning in the caller's shell keeps the victim alive as long as the
# test process is.
VICTIM_PID=""
spawn_victim() {
  local label="$1"
  ( exec -a "ui-inspect-test-victim-${label}" sleep 600 ) &
  VICTIM_PID=$!
  echo "${VICTIM_PID}" >"${SANDBOX}/pids/${label}"
}

kill_pid() {
  local pid="$1"
  if [[ "${pid}" =~ ^[0-9]+$ ]] && kill -0 "${pid}" 2>/dev/null; then
    # `disown` drops bash's job-control record so it won't print
    # "Terminated: 15" to stderr when the child dies.
    disown "${pid}" 2>/dev/null || true
    kill "${pid}" 2>/dev/null || true
    local waited=0
    while kill -0 "${pid}" 2>/dev/null && (( waited < 10 )); do
      sleep 0.2
      waited=$((waited + 1))
    done
  fi
}

# Source the script so we can call parse_state_file, verify_identity,
# kill_tree_verified directly. BASH_SOURCE guard in the script prevents main
# from running on source.
# shellcheck disable=SC1090
source "${SCRIPT}"

# -----------------------------------------------------------------------------
# Test 1: parse_state_file — normal, valid content.
# -----------------------------------------------------------------------------
test_parse_normal() {
  printf '\n[test 1] normal state parse\n'
  mk_sandbox
  local sf="${SANDBOX}/state"
  cat >"${sf}" <<'EOF'
WEB_PID=12345
MACOS_PID=12346
WEB_URL=http://localhost:60739
WEB_LOG=/tmp/ui-inspect/web.log
MACOS_LOG=/tmp/ui-inspect/macos.log
WEB_CMD_PATTERN=run-local.sh --platform web-server
MACOS_CMD_PATTERN=run-local.sh --platform macos
WEB_STARTED_AT=2026-04-23T12:34:56Z
MACOS_STARTED_AT=2026-04-23T12:35:12Z
EOF
  parse_state_file "${sf}"
  expect "WEB_PID parsed"                   test "${WEB_PID}" = "12345"
  expect "MACOS_PID parsed"                 test "${MACOS_PID}" = "12346"
  expect "WEB_URL parsed"                   test "${WEB_URL}" = "http://localhost:60739"
  expect "WEB_CMD_PATTERN parsed"           test "${WEB_CMD_PATTERN}" = "run-local.sh --platform web-server"
  expect "MACOS_CMD_PATTERN parsed"         test "${MACOS_CMD_PATTERN}" = "run-local.sh --platform macos"
  expect "WEB_STARTED_AT parsed"            test "${WEB_STARTED_AT}" = "2026-04-23T12:34:56Z"
}

# -----------------------------------------------------------------------------
# Test 2: parse_state_file — malformed input is never executed.
# A prior `source "${STATE_FILE}"` implementation would run arbitrary commands.
# The strict parser must drop invalid lines silently.
# -----------------------------------------------------------------------------
test_parse_malformed() {
  printf '\n[test 2] malformed state treated as stale\n'
  mk_sandbox
  local sf="${SANDBOX}/state"
  local canary="${SANDBOX}/canary-should-not-exist"
  # NB: use an unquoted heredoc but escape every `$(...)` / `${HOME}` so they
  # appear literally in the file. A bare `$(touch …)` would run during heredoc
  # expansion (when we're writing the file), not during parse — which would
  # hide a real parser bug behind a test-authorship bug.
  cat >"${sf}" <<EOF
# comment line
WEB_PID=not-a-number
MACOS_PID=-1
WEB_URL=http://localhost:60739; touch ${canary}
WEB_LOG=/etc/passwd
MACOS_LOG=../../../etc/passwd
WEB_CMD_PATTERN=rm -rf \$HOME
MACOS_CMD_PATTERN=\$(touch ${canary})
WEB_STARTED_AT=yesterday
garbage line with no equals
=no-key
lower_case_key=ignored
\$(touch "${canary}")
EOF
  parse_state_file "${sf}"

  expect "canary was NOT created (no code execution)" test ! -e "${canary}"
  expect "WEB_PID rejected (non-numeric)"             test -z "${WEB_PID}"
  expect "MACOS_PID rejected (negative)"              test -z "${MACOS_PID}"
  expect "WEB_URL rejected (contains shell metachar)" test -z "${WEB_URL}"
  expect "WEB_CMD_PATTERN rejected (\$ not allowed)"   test -z "${WEB_CMD_PATTERN}"
  expect "MACOS_CMD_PATTERN rejected"                 test -z "${MACOS_CMD_PATTERN}"
  expect "WEB_STARTED_AT rejected (bad format)"       test -z "${WEB_STARTED_AT}"
}

# -----------------------------------------------------------------------------
# Test 3: stale state + PID reuse — identity verification must refuse to kill.
# Simulate: state file recorded "some command", PID is now ours with a DIFFERENT
# command. verify_identity must return non-zero, and kill_tree_verified must
# leave the process alive.
# -----------------------------------------------------------------------------
test_pid_reuse_safety() {
  printf '\n[test 3] stale state + PID reuse is not killed\n'
  mk_sandbox

  spawn_victim web
  local victim_pid="${VICTIM_PID}"

  # Recorded pattern that does NOT match the victim's command.
  local pattern="run-local.sh --platform web-server"
  local recorded_iso="2026-01-01T00:00:00Z"

  # Sanity: victim is alive.
  expect "victim is alive before verify" kill -0 "${victim_pid}"

  # verify_identity must reject (return 1).
  local rc=0
  verify_identity "${victim_pid}" "${pattern}" "$(iso_to_epoch "${recorded_iso}")" || rc=$?
  expect "verify_identity rejects reused pid" test "${rc}" -ne 0

  # kill_tree_verified must NOT signal the victim.
  kill_tree_verified "${victim_pid}" "${pattern}" "${recorded_iso}" "web" >/dev/null 2>&1 || true
  expect "victim survives kill_tree_verified" kill -0 "${victim_pid}"

  kill_pid "${victim_pid}"
}

# -----------------------------------------------------------------------------
# Test 4: identity match — verify_identity accepts a matching live process.
# -----------------------------------------------------------------------------
test_identity_match_positive() {
  printf '\n[test 4] matching identity is accepted\n'
  mk_sandbox

  spawn_victim macos
  local pid="${VICTIM_PID}"

  # Pattern chosen to be a substring of the victim's argv (see spawn_victim).
  local pattern="ui-inspect-test-victim-macos"
  local recorded_iso
  recorded_iso="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

  local rc=0
  verify_identity "${pid}" "${pattern}" "$(iso_to_epoch "${recorded_iso}")" || rc=$?
  expect "verify_identity accepts matching pid" test "${rc}" -eq 0

  kill_pid "${pid}"
}

# -----------------------------------------------------------------------------
# Test 5: normal start/stop lifecycle using test hooks. Drives the real
# cmd_start / cmd_stop paths with lightweight fake workloads.
# -----------------------------------------------------------------------------
test_normal_lifecycle() {
  printf '\n[test 5] normal start/stop lifecycle\n'
  mk_sandbox
  local tmp_state="${SANDBOX}/ui-inspect"

  # Override the script's state dir by invoking a subshell that re-defines the
  # constants. Easiest: set STATE_DIR/STATE_FILE/WEB_LOG/MACOS_LOG via env and
  # run the script in a subshell. The script reads these from locals seeded by
  # initial variable defs, so we re-run it as a child with a wrapper.

  cat >"${SANDBOX}/run.sh" <<'WRAPPER'
#!/usr/bin/env bash
set -Eeuo pipefail
ROOT_DIR="$1"
STATE_ROOT="$2"
shift 2
export UI_INSPECT_SKIP_FALLBACK=1
export UI_INSPECT_READY_TIMEOUT=15
export UI_INSPECT_WEB_READY_REGEX='FAKE_WEB_READY'
export UI_INSPECT_MACOS_READY_REGEX='FAKE_MACOS_READY'
export UI_INSPECT_WEB_CMD="echo FAKE_WEB_READY; exec -a ui-inspect-test-web sleep 600"
export UI_INSPECT_MACOS_CMD="echo FAKE_MACOS_READY; exec -a ui-inspect-test-macos sleep 600"
# Redirect state dir: patch the script via symlinking .tmp/ui-inspect.
mkdir -p "${ROOT_DIR}/.tmp"
rm -rf "${ROOT_DIR}/.tmp/ui-inspect"
ln -s "${STATE_ROOT}" "${ROOT_DIR}/.tmp/ui-inspect"
"${ROOT_DIR}/scripts/ui-inspect-run.sh" "$@"
WRAPPER
  chmod +x "${SANDBOX}/run.sh"

  mkdir -p "${tmp_state}"

  # Start.
  if ! "${SANDBOX}/run.sh" "${ROOT_DIR}" "${tmp_state}" start >"${SANDBOX}/start.out" 2>&1; then
    fail "cmd_start exited non-zero"
    sed 's/^/    | /' "${SANDBOX}/start.out" >&2
    return
  fi
  pass "cmd_start exited zero"

  # State file should exist and parse cleanly.
  expect "state file created"                       test -f "${tmp_state}/state"
  parse_state_file "${tmp_state}/state"
  expect "WEB_PID recorded"                         test -n "${WEB_PID}"
  expect "MACOS_PID recorded"                       test -n "${MACOS_PID}"
  expect "WEB_CMD_PATTERN recorded"                 test -n "${WEB_CMD_PATTERN}"
  expect "MACOS_CMD_PATTERN recorded"               test -n "${MACOS_CMD_PATTERN}"
  expect "WEB_STARTED_AT recorded"                  test -n "${WEB_STARTED_AT}"

  local w_pid="${WEB_PID}" m_pid="${MACOS_PID}"

  # Both fake processes should be alive.
  expect "fake web pid alive"   kill -0 "${w_pid}"
  expect "fake macos pid alive" kill -0 "${m_pid}"

  # Stop.
  if ! "${SANDBOX}/run.sh" "${ROOT_DIR}" "${tmp_state}" stop >"${SANDBOX}/stop.out" 2>&1; then
    fail "cmd_stop exited non-zero"
    sed 's/^/    | /' "${SANDBOX}/stop.out" >&2
  else
    pass "cmd_stop exited zero"
  fi

  # State file should be cleaned up.
  expect "state file removed after stop"    test ! -f "${tmp_state}/state"

  # Processes should be gone.
  local waited=0
  while (( waited < 15 )) && ( kill -0 "${w_pid}" 2>/dev/null || kill -0 "${m_pid}" 2>/dev/null ); do
    sleep 1
    waited=$((waited + 1))
  done
  expect "fake web pid killed"   bash -c "! kill -0 ${w_pid} 2>/dev/null"
  expect "fake macos pid killed" bash -c "! kill -0 ${m_pid} 2>/dev/null"

  # Remove the symlink we installed.
  rm -f "${ROOT_DIR}/.tmp/ui-inspect"
}

# -----------------------------------------------------------------------------
# Test 6: partial start failure leaves no risky state behind.
# Web succeeds, macOS fails (readiness timeout with an exiting command).
# The script must kill the web process and leave NO state file behind.
# -----------------------------------------------------------------------------
test_partial_start_cleanup() {
  printf '\n[test 6] partial start failure cleans up\n'
  mk_sandbox
  local tmp_state="${SANDBOX}/ui-inspect"
  mkdir -p "${tmp_state}"

  cat >"${SANDBOX}/run.sh" <<'WRAPPER'
#!/usr/bin/env bash
set -Eeuo pipefail
ROOT_DIR="$1"
STATE_ROOT="$2"
shift 2
export UI_INSPECT_SKIP_FALLBACK=1
export UI_INSPECT_READY_TIMEOUT=3
export UI_INSPECT_WEB_READY_REGEX='FAKE_WEB_READY'
# macOS readiness marker that never arrives + short-lived command.
export UI_INSPECT_MACOS_READY_REGEX='NEVER_MATCHES_anything_xyz'
export UI_INSPECT_WEB_CMD="echo FAKE_WEB_READY; exec -a ui-inspect-test-web sleep 600"
export UI_INSPECT_MACOS_CMD="echo started; exec -a ui-inspect-test-macos sleep 600"
mkdir -p "${ROOT_DIR}/.tmp"
rm -rf "${ROOT_DIR}/.tmp/ui-inspect"
ln -s "${STATE_ROOT}" "${ROOT_DIR}/.tmp/ui-inspect"
"${ROOT_DIR}/scripts/ui-inspect-run.sh" "$@"
WRAPPER
  chmod +x "${SANDBOX}/run.sh"

  local rc=0
  "${SANDBOX}/run.sh" "${ROOT_DIR}" "${tmp_state}" start >"${SANDBOX}/partial.out" 2>&1 || rc=$?

  expect "cmd_start reported failure" test "${rc}" -ne 0
  expect "no state file after partial start" test ! -f "${tmp_state}/state"
  expect "no lingering web.pid file"  test ! -f "${tmp_state}/web.pid"
  expect "no lingering macos.pid file" test ! -f "${tmp_state}/macos.pid"

  # Any process we spawned should be gone — scan our pid files (none) and the
  # recorded pid names via pgrep when available.
  if command -v pgrep >/dev/null 2>&1; then
    local leftover
    leftover="$(pgrep -f 'ui-inspect-test-(web|macos)' 2>/dev/null || true)"
    if [[ -n "${leftover}" ]]; then
      # Tolerate — but report. Clean up so the test run doesn't leak.
      fail "leftover ui-inspect-test-fake processes after partial start: ${leftover}"
      for p in ${leftover}; do kill "${p}" 2>/dev/null || true; done
    else
      pass "no leftover fake processes after partial start"
    fi
  fi

  rm -f "${ROOT_DIR}/.tmp/ui-inspect"
}

main() {
  printf 'ui-inspect-run.sh smoke tests\n'
  test_parse_normal
  test_parse_malformed
  test_pid_reuse_safety
  test_identity_match_positive
  test_normal_lifecycle
  test_partial_start_cleanup

  printf '\n'
  if (( FAILS > 0 )); then
    printf 'RESULT: %d failure(s)\n' "${FAILS}" >&2
    exit 1
  fi
  printf 'RESULT: all passed\n'
}

main "$@"
