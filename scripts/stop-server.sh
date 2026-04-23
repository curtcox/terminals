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

has_pid() {
  local needle="$1"
  local pid
  for pid in "${PIDS[@]:-}"; do
    if [[ "${pid}" == "${needle}" ]]; then
      return 0
    fi
  done
  return 1
}

add_pid() {
  local pid="$1"
  if [[ -z "${pid}" ]]; then
    return 0
  fi
  if ! [[ "${pid}" =~ ^[0-9]+$ ]]; then
    return 0
  fi
  if [[ "${pid}" == "$$" ]]; then
    return 0
  fi
  if has_pid "${pid}"; then
    return 0
  fi
  PIDS+=("${pid}")
}

collect_port_pids() {
  local port="$1"
  local pid

  if ! command -v lsof >/dev/null 2>&1; then
    return 0
  fi

  while IFS= read -r pid; do
    add_pid "${pid}"
  done < <(lsof -tiTCP:"${port}" -sTCP:LISTEN 2>/dev/null || true)
}

collect_command_pids() {
  local pid
  while IFS= read -r pid; do
    add_pid "${pid}"
  done < <(
    (ps -axo pid=,command= 2>/dev/null || true) | awk '
      $0 ~ /go run (\.\/|[^ ]*\/)?cmd\/server([[:space:]]|$)/ { print $1; next }
      $0 ~ /\/tmp\/go-build[^[:space:]]*\/exe\/server([[:space:]]|$)/ { print $1; next }
      $0 ~ /\/terminal_server\/cmd\/server([[:space:]]|$)/ { print $1; next }
    '
  )
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

echo "Stopping terminal server process(es): ${PIDS[*]}"

for pid in "${PIDS[@]}"; do
  kill_gracefully "${pid}"
done

for pid in "${PIDS[@]}"; do
  if ! wait_for_exit "${pid}"; then
    echo "Process ${pid} did not exit in ${TERMINATE_TIMEOUT_SECONDS}s; sending SIGKILL."
    kill_forcefully "${pid}"
  fi
done

echo "Server stop complete."
