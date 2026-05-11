# Sourced by scripts/run-local.sh after globals are initialized.
# Keeps the entry script under size tooling thresholds; see scripts/find-oversized-files.py.
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

append_diagnostic_path() {
  if [[ -f "${CLIENT_DIAG_LOG}" ]]; then
    echo "--- client diagnostics: ${CLIENT_DIAG_LOG} ---" >&3
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
  append_diagnostic_path
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
  append_diagnostic_path
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
  grep -Eo 'http://[^ ]+:[0-9]+' "${file}" | tail -n 1
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

  if [[ -n "${CONTROL_HTTP_PORT}" ]]; then
    require_available_port "${CONTROL_HTTP_PORT}" "control HTTP" "TERMINALS_CONTROL_HTTP_PORT"
  else
    CONTROL_HTTP_PORT="$(find_available_port 50056 200 || true)"
    if [[ -z "${CONTROL_HTTP_PORT}" ]]; then
      fail "unable to find open port for control HTTP starting at 50056"
    fi
    reserve_port "${CONTROL_HTTP_PORT}"
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

resolve_build_metadata() {
  if [[ -z "${BUILD_SHA}" ]]; then
    BUILD_SHA="$(git -C "${ROOT_DIR}" rev-parse --short=12 HEAD 2>/dev/null || true)"
  fi
  BUILD_SHA="$(echo "${BUILD_SHA}" | tr -d '[:space:]')"
  if [[ -z "${BUILD_SHA}" ]]; then
    BUILD_SHA="unknown"
  fi

  if [[ -z "${BUILD_DATE}" ]]; then
    BUILD_DATE="$(date -u +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null || true)"
  fi
  BUILD_DATE="$(echo "${BUILD_DATE}" | tr -d '[:space:]')"
  if [[ -z "${BUILD_DATE}" ]]; then
    BUILD_DATE="unknown"
  fi
}

configure_client_dart_defines() {
  CLIENT_DART_DEFINE_ARGS=(
    "--dart-define=TERMINALS_CONTROL_HOST=${CLIENT_CONTROL_HOST}"
    "--dart-define=TERMINALS_GRPC_PORT=${GRPC_PORT}"
    "--dart-define=TERMINALS_CONTROL_WS_PORT=${CONTROL_WS_PORT}"
    "--dart-define=TERMINALS_CONTROL_TCP_PORT=${CONTROL_TCP_PORT}"
    "--dart-define=TERMINALS_CONTROL_HTTP_PORT=${CONTROL_HTTP_PORT}"
    "--dart-define=TERMINALS_ADMIN_HTTP_PORT=${ADMIN_PORT}"
    "--dart-define=TERMINALS_PHOTO_FRAME_HTTP_PORT=${PHOTO_PORT}"
    "--dart-define=TERMINALS_BUILD_SHA=${BUILD_SHA}"
    "--dart-define=TERMINALS_BUILD_DATE=${BUILD_DATE}"
    "--dart-define=TERMINALS_AUTO_CONNECT_ON_STARTUP=${CLIENT_AUTO_CONNECT_ON_STARTUP}"
  )
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
