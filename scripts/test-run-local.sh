#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SOURCE_SCRIPT="${ROOT_DIR}/scripts/run-local.sh"

SANDBOX_DIR=""
RUN_REAL_SMOKE="false"

fail() {
  local message="$1"
  echo "FAIL: ${message}" >&2
  if [[ -n "${SANDBOX_DIR}" ]] && [[ -f "${SANDBOX_DIR}/test-output.log" ]]; then
    echo "--- test output ---" >&2
    cat "${SANDBOX_DIR}/test-output.log" >&2
    echo "--- end test output ---" >&2
  fi
  exit 1
}

assert_contains() {
  local file="$1"
  local needle="$2"
  if ! grep -Fq "${needle}" "${file}"; then
    fail "expected '${needle}' in ${file}"
  fi
}

assert_not_contains() {
  local file="$1"
  local needle="$2"
  if grep -Fq "${needle}" "${file}"; then
    fail "did not expect '${needle}' in ${file}"
  fi
}

require_cmd() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    fail "required command not found: ${name}"
  fi
}

reset_sandbox() {
  if [[ -n "${SANDBOX_DIR}" ]] && [[ -d "${SANDBOX_DIR}" ]]; then
    rm -rf "${SANDBOX_DIR}"
  fi

  SANDBOX_DIR="$(mktemp -d "${ROOT_DIR}/.tmp/run-local-test.XXXXXX")"
  mkdir -p "${SANDBOX_DIR}/scripts" "${SANDBOX_DIR}/terminal_server" "${SANDBOX_DIR}/terminal_client" "${SANDBOX_DIR}/.bin" "${SANDBOX_DIR}/.tmp"
  cp "${SOURCE_SCRIPT}" "${SANDBOX_DIR}/scripts/run-local.sh"
  chmod +x "${SANDBOX_DIR}/scripts/run-local.sh"

  cat >"${SANDBOX_DIR}/.bin/go" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

echo "go $*" >> "${RUN_LOCAL_TEST_COMMAND_LOG:?RUN_LOCAL_TEST_COMMAND_LOG not set}"

if [[ "$1" == "mod" && "$2" == "download" ]]; then
  exit 0
fi

if [[ "$1" == "run" && "$2" == "./cmd/server" ]]; then
  echo "control service ready"
  trap 'exit 0' TERM INT
  while true; do
    sleep 1
  done
fi

echo "unexpected go invocation: $*" >&2
exit 2
EOF
  chmod +x "${SANDBOX_DIR}/.bin/go"

  cat >"${SANDBOX_DIR}/.bin/flutter" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

echo "flutter $*" >> "${RUN_LOCAL_TEST_COMMAND_LOG:?RUN_LOCAL_TEST_COMMAND_LOG not set}"

if [[ "$1" == "pub" && "$2" == "get" ]]; then
  exit 0
fi

if [[ "$1" == "config" && "$2" == "--enable-web" ]]; then
  exit 0
fi

if [[ "$1" == "create" && "$2" == "." ]]; then
  mkdir -p web/icons
  : > web/index.html
  : > web/manifest.json
  : > web/favicon.png
  : > web/icons/Icon-192.png
  : > web/icons/Icon-512.png
  : > web/icons/Icon-maskable-192.png
  : > web/icons/Icon-maskable-512.png
  exit 0
fi

if [[ "$1" == "run" && "$2" == "-d" ]]; then
  echo "Launching lib/main.dart on $3 in debug mode..."
  if [[ "$3" == "web-server" ]]; then
    echo "lib/main.dart is being served at http://localhost:58080"
  fi
  trap 'exit 0' TERM INT
  while true; do
    sleep 1
  done
fi

echo "unexpected flutter invocation: $*" >&2
exit 2
EOF
  chmod +x "${SANDBOX_DIR}/.bin/flutter"

  cat >"${SANDBOX_DIR}/.bin/nc" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
exit 1
EOF
  chmod +x "${SANDBOX_DIR}/.bin/nc"

  cat >"${SANDBOX_DIR}/.bin/lsof" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

port=""
for arg in "$@"; do
  if [[ "${arg}" == -iTCP:* ]]; then
    port="${arg#-iTCP:}"
  fi
done

IFS=',' read -r -a busy_ports <<< "${BUSY_PORTS:-}"
for busy_port in "${busy_ports[@]}"; do
  if [[ -n "${busy_port}" && "${busy_port}" == "${port}" ]]; then
    exit 0
  fi
done

exit 1
EOF
  chmod +x "${SANDBOX_DIR}/.bin/lsof"
}

run_in_sandbox() {
  local output_file="$1"
  shift

  set +e
  (
    cd "${SANDBOX_DIR}"
    RUN_LOCAL_TEST_MODE=true \
    RUN_LOCAL_CLIENT_STARTUP_DELAY_SECONDS=0.1 \
    RUN_LOCAL_TEST_COMMAND_LOG="${SANDBOX_DIR}/commands.log" \
    "$@"
  ) >"${output_file}" 2>&1
  local status=$?
  set -e

  return "${status}"
}

parse_args() {
  while [[ "$#" -gt 0 ]]; do
    case "$1" in
      --real-smoke)
        RUN_REAL_SMOKE="true"
        ;;
      -h|--help)
        cat <<'EOF'
Usage: ./scripts/test-run-local.sh [--real-smoke]

Options:
  --real-smoke   Also run an opt-in smoke test using real go/flutter tools.
EOF
        exit 0
        ;;
      *)
        fail "unknown argument: $1"
        ;;
    esac
    shift
  done
}

test_skip_bootstrap_requires_web_support() {
  reset_sandbox
  : >"${SANDBOX_DIR}/commands.log"
  local output_file="${SANDBOX_DIR}/test-output.log"

  if run_in_sandbox "${output_file}" ./scripts/run-local.sh --skip-bootstrap; then
    fail "expected failure when web support is missing and --skip-bootstrap is used"
  fi

  assert_contains "${output_file}" "web support is not configured"
  echo "PASS: skip-bootstrap validates missing web support"
}

test_explicit_busy_port_fails_early() {
  reset_sandbox
  : >"${SANDBOX_DIR}/commands.log"
  local output_file="${SANDBOX_DIR}/test-output.log"

  set +e
  (
    cd "${SANDBOX_DIR}"
    BUSY_PORTS=59999 \
    TERMINALS_ADMIN_HTTP_PORT=59999 \
    RUN_LOCAL_TEST_MODE=true \
    RUN_LOCAL_CLIENT_STARTUP_DELAY_SECONDS=0.1 \
    RUN_LOCAL_TEST_COMMAND_LOG="${SANDBOX_DIR}/commands.log" \
    ./scripts/run-local.sh --skip-bootstrap
  ) >"${output_file}" 2>&1
  local status=$?
  set -e

  if [[ "${status}" -eq 0 ]]; then
    fail "expected failure for busy explicit admin port"
  fi

  assert_contains "${output_file}" "admin port 59999 is already in use"
  echo "PASS: explicit busy port is reported clearly"
}

test_bootstrap_web_and_auto_port_selection() {
  reset_sandbox
  : >"${SANDBOX_DIR}/commands.log"
  local output_file="${SANDBOX_DIR}/test-output.log"

  set +e
  (
    cd "${SANDBOX_DIR}"
    BUSY_PORTS=50053 \
    RUN_LOCAL_TEST_MODE=true \
    RUN_LOCAL_CLIENT_STARTUP_DELAY_SECONDS=0.1 \
    RUN_LOCAL_TEST_COMMAND_LOG="${SANDBOX_DIR}/commands.log" \
    ./scripts/run-local.sh
  ) >"${output_file}" 2>&1
  local status=$?
  set -e

  if [[ "${status}" -ne 0 ]]; then
    fail "expected run-local success in test mode"
  fi

  if [[ ! -f "${SANDBOX_DIR}/terminal_client/web/index.html" ]]; then
    fail "expected web bootstrap to create terminal_client/web/index.html"
  fi

  assert_contains "${output_file}" "Using ports: grpc=50051 admin=50054 photo=50052"
  assert_contains "${output_file}" "Test mode: startup checks passed."
  assert_contains "${output_file}" "Browser client URL: http://localhost:58080"
  assert_contains "${SANDBOX_DIR}/commands.log" "go mod download"
  assert_contains "${SANDBOX_DIR}/commands.log" "flutter pub get"
  assert_contains "${SANDBOX_DIR}/commands.log" "flutter config --enable-web"
  assert_contains "${SANDBOX_DIR}/commands.log" "flutter create . --platforms=web"
  assert_contains "${SANDBOX_DIR}/commands.log" "go run ./cmd/server"
  assert_contains "${SANDBOX_DIR}/commands.log" "flutter run -d web-server"

  echo "PASS: web bootstrap and auto-port selection work"
}

test_real_smoke_run_local() {
  local output_file="${ROOT_DIR}/.tmp/run-local-real-smoke.log"

  require_cmd go
  require_cmd flutter
  require_cmd nc

  set +e
  (
    cd "${ROOT_DIR}"
    RUN_LOCAL_TEST_MODE=true \
    RUN_LOCAL_CLIENT_STARTUP_DELAY_SECONDS=5 \
    ./scripts/run-local.sh
  ) >"${output_file}" 2>&1
  local status=$?
  set -e

  if [[ "${status}" -ne 0 ]]; then
    fail "real smoke test failed (see ${output_file})"
  fi

  assert_contains "${output_file}" "Test mode: startup checks passed."
  assert_contains "${output_file}" "Browser client URL: http://localhost:"
  assert_contains "${ROOT_DIR}/.tmp/run-local-client.log" "Launching lib/main.dart on Web Server in debug mode"
  assert_not_contains "${ROOT_DIR}/.tmp/run-local-client.log" "This application is not configured to build on the web"

  echo "PASS: real smoke run-local works"
}

main() {
  parse_args "$@"
  test_skip_bootstrap_requires_web_support
  test_explicit_busy_port_fails_early
  test_bootstrap_web_and_auto_port_selection
  if [[ "${RUN_REAL_SMOKE}" == "true" ]]; then
    test_real_smoke_run_local
  else
    echo "SKIP: real smoke test (run with --real-smoke to enable)"
  fi
  if [[ -n "${SANDBOX_DIR}" ]] && [[ -d "${SANDBOX_DIR}" ]]; then
    rm -rf "${SANDBOX_DIR}"
  fi
  echo "All run-local script tests passed."
}

main "$@"