#!/usr/bin/env bash
set -euo pipefail

GRPC_PORT="${TERMINALS_GRPC_PORT:-50051}"
CONTROL_WS_PORT="${TERMINALS_CONTROL_WS_PORT:-50054}"
CONTROL_TCP_PORT="${TERMINALS_CONTROL_TCP_PORT:-50055}"
CONTROL_HTTP_PORT="${TERMINALS_CONTROL_HTTP_PORT:-50056}"
PHOTO_FRAME_HTTP_PORT="${TERMINALS_PHOTO_FRAME_HTTP_PORT:-50052}"
ADMIN_HTTP_PORT="${TERMINALS_ADMIN_HTTP_PORT:-50053}"
TERMINATE_TIMEOUT_SECONDS="${TERMINALS_STOP_TIMEOUT_SECONDS:-5}"

PORTS=(
  "${GRPC_PORT}"
  "${CONTROL_WS_PORT}"
  "${CONTROL_TCP_PORT}"
  "${CONTROL_HTTP_PORT}"
  "${PHOTO_FRAME_HTTP_PORT}"
  "${ADMIN_HTTP_PORT}"
)

PIDS=()
PID_SOURCES=()
PID_START_EPOCHS=()

warn() {
  echo "WARN: $*" >&2
}

pid_index() {
  local needle="$1"
  local i
  for i in "${!PIDS[@]}"; do
    if [[ "${PIDS[$i]}" == "${needle}" ]]; then
      echo "${i}"
      return 0
    fi
  done
  return 1
}

process_start_epoch() {
  local pid="$1"
  [[ -n "${pid}" ]] || { printf ''; return; }
  command -v ps >/dev/null 2>&1 || { printf ''; return; }

  local lstart
  lstart="$(ps -o lstart= -p "${pid}" 2>/dev/null || true)"
  lstart="${lstart#"${lstart%%[![:space:]]*}"}"
  lstart="${lstart%"${lstart##*[![:space:]]}"}"
  [[ -n "${lstart}" ]] || { printf ''; return; }

  local epoch=""
  if date -j -f "%a %b %e %T %Y" "${lstart}" +%s >/dev/null 2>&1; then
    epoch="$(date -j -f "%a %b %e %T %Y" "${lstart}" +%s 2>/dev/null || true)"
  elif date -d "${lstart}" +%s >/dev/null 2>&1; then
    epoch="$(date -d "${lstart}" +%s 2>/dev/null || true)"
  fi
  printf '%s' "${epoch}"
}

add_pid() {
  local pid="$1"
  local source="${2:-unknown}"
  if [[ -z "${pid}" ]]; then
    return 0
  fi
  if ! [[ "${pid}" =~ ^[0-9]+$ ]]; then
    return 0
  fi
  if [[ "${pid}" == "$$" ]]; then
    return 0
  fi

  local idx
  if idx="$(pid_index "${pid}")"; then
    if [[ "${PID_SOURCES[$idx]}" != *"${source}"* ]]; then
      PID_SOURCES[$idx]="${PID_SOURCES[$idx]},${source}"
    fi
    return 0
  fi

  PIDS+=("${pid}")
  PID_SOURCES+=("${source}")
  PID_START_EPOCHS+=("$(process_start_epoch "${pid}")")
}

collect_port_pids() {
  local port="$1"
  local pid

  if ! command -v lsof >/dev/null 2>&1; then
    return 0
  fi

  while IFS= read -r pid; do
    add_pid "${pid}" "port:${port}"
  done < <(lsof -tiTCP:"${port}" -sTCP:LISTEN 2>/dev/null || true)
}

collect_command_pids() {
  local pid
  while IFS= read -r pid; do
    add_pid "${pid}" "command-scan"
  done < <(
    (ps -axo pid=,command= 2>/dev/null || true) | awk '
      $0 ~ /go run (\.\/|[^ ]*\/)?cmd\/server([[:space:]]|$)/ { print $1; next }
      $0 ~ /\/tmp\/go-build[^[:space:]]*\/exe\/server([[:space:]]|$)/ { print $1; next }
      $0 ~ /\/terminal_server\/cmd\/server([[:space:]]|$)/ { print $1; next }
      $0 ~ /\/terminal_server\/server([[:space:]]|$)/ { print $1; next }
    '
  )
}

is_server_command_line() {
  local line="$1"
  [[ "${line}" =~ go\ run\ (\./|[^[:space:]]*/)?cmd/server([[:space:]]|$) ]] && return 0
  [[ "${line}" =~ /tmp/go-build[^[:space:]]*/exe/server([[:space:]]|$) ]] && return 0
  [[ "${line}" =~ /terminal_server/cmd/server([[:space:]]|$) ]] && return 0
  [[ "${line}" =~ /terminal_server/server([[:space:]]|$) ]] && return 0
  return 1
}

# Return codes:
#   0: identity verified
#   1: identity mismatch
#   2: unable to verify
verify_pid_identity() {
  local pid="$1"
  local recorded_start_epoch="${2:-}"
  [[ -n "${pid}" ]] || return 1

  if ! command -v ps >/dev/null 2>&1; then
    return 2
  fi

  local cmd
  cmd="$(ps -o command= -p "${pid}" 2>/dev/null || true)"
  if [[ -z "${cmd}" ]]; then
    return 1
  fi
  if ! is_server_command_line "${cmd}"; then
    return 1
  fi

  if [[ -n "${recorded_start_epoch}" && "${recorded_start_epoch}" =~ ^[0-9]+$ ]]; then
    local live_start_epoch
    live_start_epoch="$(process_start_epoch "${pid}")"
    if [[ "${live_start_epoch}" =~ ^[0-9]+$ ]]; then
      local delta=$(( live_start_epoch - recorded_start_epoch ))
      if (( delta > 5 )); then
        return 1
      fi
    fi
  fi

  return 0
}

kill_gracefully() {
  local pid="$1"
  if kill -0 "${pid}" >/dev/null 2>&1; then
    kill "${pid}" >/dev/null 2>&1 || true
  fi
}

kill_forcefully() {
  local pid="$1"
  if kill -0 "${pid}" >/dev/null 2>&1; then
    kill -9 "${pid}" >/dev/null 2>&1 || true
  fi
}

wait_for_exit() {
  local pid="$1"
  local waited=0
  while kill -0 "${pid}" >/dev/null 2>&1; do
    if (( waited >= TERMINATE_TIMEOUT_SECONDS )); then
      return 1
    fi
    sleep 1
    waited=$((waited + 1))
  done
  return 0
}

for port in "${PORTS[@]}"; do
  collect_port_pids "${port}"
done

collect_command_pids

if [[ "${#PIDS[@]}" -eq 0 ]]; then
  echo "No running terminal server process found."
  exit 0
fi

VERIFIED_PIDS=()
for i in "${!PIDS[@]}"; do
  pid="${PIDS[$i]}"
  source="${PID_SOURCES[$i]:-unknown}"
  recorded_start="${PID_START_EPOCHS[$i]:-}"
  rc=0
  verify_pid_identity "${pid}" "${recorded_start}" || rc=$?
  case "${rc}" in
    0)
      VERIFIED_PIDS+=("${pid}")
      ;;
    1)
      warn "Skipping pid ${pid} from ${source}: identity mismatch."
      ;;
    2)
      warn "Skipping pid ${pid} from ${source}: unable to verify identity (ps unavailable/restricted)."
      ;;
  esac
done

if [[ "${#VERIFIED_PIDS[@]}" -eq 0 ]]; then
  echo "No verified terminal server process found."
  exit 0
fi

echo "Stopping terminal server process(es): ${VERIFIED_PIDS[*]}"

for pid in "${VERIFIED_PIDS[@]}"; do
  kill_gracefully "${pid}"
done

for pid in "${VERIFIED_PIDS[@]}"; do
  if ! wait_for_exit "${pid}"; then
    echo "Process ${pid} did not exit in ${TERMINATE_TIMEOUT_SECONDS}s; sending SIGKILL."
    kill_forcefully "${pid}"
  fi
done

echo "Server stop complete."
