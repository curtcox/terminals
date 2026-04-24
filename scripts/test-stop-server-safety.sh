#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT="${ROOT_DIR}/scripts/stop-server.sh"

SANDBOX=""
MATCH_PID=""
MISMATCH_PID=""

cleanup() {
  if [[ -n "${MATCH_PID}" && "${MATCH_PID}" =~ ^[0-9]+$ ]] && kill -0 "${MATCH_PID}" >/dev/null 2>&1; then
    kill "${MATCH_PID}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${MISMATCH_PID}" && "${MISMATCH_PID}" =~ ^[0-9]+$ ]] && kill -0 "${MISMATCH_PID}" >/dev/null 2>&1; then
    kill "${MISMATCH_PID}" >/dev/null 2>&1 || true
  fi
  if [[ -n "${SANDBOX}" && -d "${SANDBOX}" ]]; then
    rm -rf "${SANDBOX}"
  fi
}
trap cleanup EXIT

fail() {
  echo "FAIL: $1" >&2
  exit 1
}

assert_contains() {
  local file="$1"
  local needle="$2"
  if ! grep -Fq -- "${needle}" "${file}"; then
    fail "expected '${needle}' in ${file}"
  fi
}

pick_free_ports() {
  python3 - <<'PY'
import socket
ports = []
for _ in range(6):
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.bind(("127.0.0.1", 0))
    ports.append(str(s.getsockname()[1]))
    s.close()
print(" ".join(ports))
PY
}

start_listener() {
  local argv0="$1"
  local port="$2"
  (
    exec -a "${argv0}" python3 -m http.server "${port}" --bind 127.0.0.1 >/dev/null 2>&1
  ) &
  echo $!
}

wait_for_listen() {
  local port="$1"
  local deadline=$((SECONDS + 10))
  while (( SECONDS < deadline )); do
    if lsof -nP -iTCP:"${port}" -sTCP:LISTEN >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.1
  done
  return 1
}

install_ps_wrapper() {
  mkdir -p "${SANDBOX}/bin"
  cat >"${SANDBOX}/bin/ps" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
real_ps="/bin/ps"
match_pid="${STOP_SERVER_TEST_MATCH_PID:-}"
mismatch_pid="${STOP_SERVER_TEST_MISMATCH_PID:-}"

if [[ "$#" -ge 2 && "$1" == "-axo" && "$2" == "pid=,command=" ]]; then
  if [[ -n "${match_pid}" ]]; then
    echo "${match_pid} go run ./cmd/server"
  fi
  if [[ -n "${mismatch_pid}" ]]; then
    echo "${mismatch_pid} ui-test-nonserver-listener"
  fi
  exit 0
fi

if [[ "$#" -ge 4 && "$1" == "-o" && "$2" == "command=" && "$3" == "-p" ]]; then
  pid="$4"
  if [[ -n "${match_pid}" && "${pid}" == "${match_pid}" ]]; then
    echo "go run ./cmd/server"
    exit 0
  fi
  if [[ -n "${mismatch_pid}" && "${pid}" == "${mismatch_pid}" ]]; then
    echo "ui-test-nonserver-listener"
    exit 0
  fi
fi

exec "${real_ps}" "$@"
EOF
  chmod +x "${SANDBOX}/bin/ps"
}

main() {
  SANDBOX="$(mktemp -d -t stop-server-safety.XXXXXX)"

  read -r P1 P2 P3 P4 P5 P6 <<<"$(pick_free_ports)"

  MATCH_PID="$(start_listener "go run ./cmd/server" "${P1}")"
  MISMATCH_PID="$(start_listener "ui-test-nonserver-listener" "${P2}")"

  wait_for_listen "${P1}" || fail "matching listener did not start on ${P1}"
  wait_for_listen "${P2}" || fail "mismatch listener did not start on ${P2}"

  install_ps_wrapper

  OUT="${SANDBOX}/stop.out"
  PATH="${SANDBOX}/bin:${PATH}" \
    STOP_SERVER_TEST_MATCH_PID="${MATCH_PID}" \
    STOP_SERVER_TEST_MISMATCH_PID="${MISMATCH_PID}" \
    TERMINALS_GRPC_PORT="${P1}" \
    TERMINALS_CONTROL_WS_PORT="${P2}" \
    TERMINALS_CONTROL_TCP_PORT="${P3}" \
    TERMINALS_CONTROL_HTTP_PORT="${P4}" \
    TERMINALS_PHOTO_FRAME_HTTP_PORT="${P5}" \
    TERMINALS_ADMIN_HTTP_PORT="${P6}" \
    TERMINALS_STOP_TIMEOUT_SECONDS=1 \
    "${SCRIPT}" >"${OUT}" 2>&1 || fail "stop-server script returned non-zero"

  assert_contains "${OUT}" "Stopping terminal server process(es): ${MATCH_PID}"
  assert_contains "${OUT}" "Skipping pid ${MISMATCH_PID}"

  if kill -0 "${MATCH_PID}" >/dev/null 2>&1; then
    fail "matching server-like process should have been stopped"
  fi
  if ! kill -0 "${MISMATCH_PID}" >/dev/null 2>&1; then
    fail "non-server process should not have been killed"
  fi

  echo "PASS: stop-server identity verification safety checks"
}

main "$@"
