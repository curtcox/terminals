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
  if ! grep -Fq -- "${needle}" "${file}"; then
    fail "expected '${needle}' in ${file}"
  fi
}

assert_not_contains() {
  local file="$1"
  local needle="$2"
  if grep -Fq -- "${needle}" "${file}"; then
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
  mkdir -p "${SANDBOX_DIR}/scripts" "${SANDBOX_DIR}/terminal_server" "${SANDBOX_DIR}/terminal_client" "${SANDBOX_DIR}/terminal_client/macos" "${SANDBOX_DIR}/.bin" "${SANDBOX_DIR}/.tmp"
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

if [[ "$1" == "build" && "$2" == "macos" ]]; then
  mkdir -p build/macos/Build/Products/Debug/terminal_client.app/Contents/MacOS
  cat > build/macos/Build/Products/Debug/terminal_client.app/Contents/Info.plist <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleExecutable</key>
  <string>terminal_client</string>
</dict>
</plist>
PLIST
  cat > build/macos/Build/Products/Debug/terminal_client.app/Contents/MacOS/terminal_client <<'APP'
#!/usr/bin/env bash
set -euo pipefail
echo "fake macOS app launched"
trap 'exit 0' TERM INT
while true; do
  sleep 1
done
APP
  chmod +x build/macos/Build/Products/Debug/terminal_client.app/Contents/MacOS/terminal_client
  exit 0
fi

if [[ "$1" == "run" && "$2" == "-d" ]]; then
  echo "Launching lib/main.dart on $3 in debug mode..."
  if [[ "$3" == "web-server" ]]; then
    web_host="localhost"
    web_port="58080"
    for arg in "$@"; do
      if [[ "${arg}" == --web-hostname=* ]]; then
        web_host="${arg#--web-hostname=}"
      fi
      if [[ "${arg}" == --web-port=* ]]; then
        web_port="${arg#--web-port=}"
      fi
    done
    echo "lib/main.dart is being served at http://${web_host}:${web_port}"
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

  cat >"${SANDBOX_DIR}/.bin/pod" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
echo "pod $*" >> "${RUN_LOCAL_TEST_COMMAND_LOG:?RUN_LOCAL_TEST_COMMAND_LOG not set}"
exit 0
EOF
  chmod +x "${SANDBOX_DIR}/.bin/pod"

  cat >"${SANDBOX_DIR}/.bin/xcodebuild" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
echo "xcodebuild $*" >> "${RUN_LOCAL_TEST_COMMAND_LOG:?RUN_LOCAL_TEST_COMMAND_LOG not set}"
exit 0
EOF
  chmod +x "${SANDBOX_DIR}/.bin/xcodebuild"

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

  assert_contains "${output_file}" "Using ports: grpc=50051 control_ws=50055 admin=50054 photo=50052 control_tcp=50056 control_http=50057"
  assert_contains "${output_file}" "Using build metadata: sha="
  assert_contains "${output_file}" "Test mode: startup checks passed."
  assert_contains "${output_file}" "Browser client URL: http://0.0.0.0:60739"
  assert_contains "${SANDBOX_DIR}/commands.log" "go mod download"
  assert_contains "${SANDBOX_DIR}/commands.log" "flutter pub get"
  assert_contains "${SANDBOX_DIR}/commands.log" "flutter config --enable-web"
  assert_contains "${SANDBOX_DIR}/commands.log" "flutter create . --platforms=web"
  assert_contains "${SANDBOX_DIR}/commands.log" "go run ./cmd/server"
  assert_contains "${SANDBOX_DIR}/commands.log" "flutter run -d web-server"
  assert_contains "${SANDBOX_DIR}/commands.log" "--dart-define=TERMINALS_GRPC_PORT=50051"
  assert_contains "${SANDBOX_DIR}/commands.log" "--dart-define=TERMINALS_CONTROL_WS_PORT=50055"
  assert_contains "${SANDBOX_DIR}/commands.log" "--dart-define=TERMINALS_CONTROL_TCP_PORT=50056"
  assert_contains "${SANDBOX_DIR}/commands.log" "--dart-define=TERMINALS_CONTROL_HTTP_PORT=50057"
  assert_contains "${SANDBOX_DIR}/commands.log" "--dart-define=TERMINALS_BUILD_SHA="
  assert_contains "${SANDBOX_DIR}/commands.log" "--dart-define=TERMINALS_BUILD_DATE="
  assert_contains "${SANDBOX_DIR}/commands.log" "--dart-define=TERMINALS_AUTO_CONNECT_ON_STARTUP=true"

  echo "PASS: web bootstrap and auto-port selection work"
}

test_duplicate_explicit_ports_fail_early() {
  reset_sandbox
  : >"${SANDBOX_DIR}/commands.log"
  local output_file="${SANDBOX_DIR}/test-output.log"

  set +e
  (
    cd "${SANDBOX_DIR}"
    TERMINALS_ADMIN_HTTP_PORT=59998 \
    TERMINALS_CONTROL_WS_PORT=59998 \
    RUN_LOCAL_TEST_MODE=true \
    RUN_LOCAL_CLIENT_STARTUP_DELAY_SECONDS=0.1 \
    RUN_LOCAL_TEST_COMMAND_LOG="${SANDBOX_DIR}/commands.log" \
    ./scripts/run-local.sh --skip-bootstrap
  ) >"${output_file}" 2>&1
  local status=$?
  set -e

  if [[ "${status}" -eq 0 ]]; then
    fail "expected failure for duplicate explicit admin/control websocket ports"
  fi

  assert_contains "${output_file}" "control websocket port 59998 conflicts with another selected local port"
  echo "PASS: duplicate explicit ports fail early with a clear message"
}

test_macos_build_and_launch_uses_runtime_defines() {
  reset_sandbox
  : >"${SANDBOX_DIR}/commands.log"
  local output_file="${SANDBOX_DIR}/test-output.log"

  set +e
  (
    cd "${SANDBOX_DIR}"
    RUN_LOCAL_TEST_MODE=true \
    RUN_LOCAL_CLIENT_STARTUP_DELAY_SECONDS=0.1 \
    RUN_LOCAL_TEST_COMMAND_LOG="${SANDBOX_DIR}/commands.log" \
    TERMINALS_GRPC_PORT=51051 \
    TERMINALS_CONTROL_WS_PORT=51054 \
    TERMINALS_CONTROL_TCP_PORT=51055 \
    TERMINALS_CONTROL_HTTP_PORT=51056 \
    TERMINALS_ADMIN_HTTP_PORT=51053 \
    TERMINALS_PHOTO_FRAME_HTTP_PORT=51052 \
    ./scripts/run-local.sh --client macos
  ) >"${output_file}" 2>&1
  local status=$?
  set -e

  if [[ "${status}" -ne 0 ]]; then
    fail "expected run-local macOS success in test mode"
  fi

  assert_contains "${output_file}" "Using ports: grpc=51051 control_ws=51054 admin=51053 photo=51052 control_tcp=51055 control_http=51056"
  assert_contains "${output_file}" "Test mode: startup checks passed."
  assert_contains "${SANDBOX_DIR}/commands.log" "pod install"
  assert_contains "${SANDBOX_DIR}/commands.log" "flutter build macos --debug --config-only --no-pub"
  assert_contains "${SANDBOX_DIR}/commands.log" "flutter build macos --debug --no-pub"
  assert_contains "${SANDBOX_DIR}/commands.log" "--dart-define=TERMINALS_GRPC_PORT=51051"
  assert_contains "${SANDBOX_DIR}/commands.log" "--dart-define=TERMINALS_CONTROL_WS_PORT=51054"
  assert_contains "${SANDBOX_DIR}/commands.log" "--dart-define=TERMINALS_CONTROL_TCP_PORT=51055"
  assert_contains "${SANDBOX_DIR}/commands.log" "--dart-define=TERMINALS_CONTROL_HTTP_PORT=51056"
  assert_contains "${SANDBOX_DIR}/commands.log" "--dart-define=TERMINALS_BUILD_SHA="
  assert_contains "${SANDBOX_DIR}/commands.log" "--dart-define=TERMINALS_BUILD_DATE="
  assert_contains "${SANDBOX_DIR}/commands.log" "--dart-define=TERMINALS_AUTO_CONNECT_ON_STARTUP=true"

  echo "PASS: macOS build and launch include runtime defines"
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
  assert_contains "${output_file}" "Browser client URL: http://"
  assert_contains "${ROOT_DIR}/.tmp/run-local-client.log" "Launching lib/main.dart on Web Server in debug mode"
  assert_not_contains "${ROOT_DIR}/.tmp/run-local-client.log" "This application is not configured to build on the web"

  echo "PASS: real smoke run-local works"
}

main() {
  parse_args "$@"
  test_skip_bootstrap_requires_web_support
  test_explicit_busy_port_fails_early
  test_bootstrap_web_and_auto_port_selection
  test_duplicate_explicit_ports_fail_early
  test_macos_build_and_launch_uses_runtime_defines
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
