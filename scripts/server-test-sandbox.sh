#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SERVER_DIR="${ROOT_DIR}/terminal_server"

# Keep Go cache in a writable location when running from restricted sandboxes.
export GOCACHE="${GOCACHE:-/tmp/terminals-go-build}"

NETWORK_PACKAGES=(
  "./cmd/server"
  "./internal/admin"
  "./internal/transport"
  "./internal/mcpadapter"
  "./internal/repl"
  "./internal/discovery"
)

declare -A NETWORK_SET=()
for pkg in "${NETWORK_PACKAGES[@]}"; do
  NETWORK_SET["${pkg}"]=1
done

mapfile -t all_packages < <(cd "${SERVER_DIR}" && go list ./...)
non_network_packages=()
for pkg in "${all_packages[@]}"; do
  rel="./${pkg#*terminal_server/}"
  if [[ -z "${NETWORK_SET[${rel}]:-}" ]]; then
    non_network_packages+=("${rel}")
  fi
done

if [[ "${#non_network_packages[@]}" -gt 0 ]]; then
  echo "==> go test (sandbox-safe packages): ${non_network_packages[*]}"
  (
    cd "${SERVER_DIR}"
    go test "${non_network_packages[@]}"
  )
fi

declare -A PROBE=()
while IFS='=' read -r key value; do
  if [[ -n "${key}" ]]; then
    PROBE["${key}"]="${value}"
  fi
done < <(cd "${ROOT_DIR}" && go run ./scripts/probe_server_test_network.go)

echo "==> network probe"
echo "loopback_listener=${PROBE[loopback_listener]:-unknown}"
echo "ipv6_loopback_listener=${PROBE[ipv6_loopback_listener]:-unknown}"
echo "host_interfaces=${PROBE[host_interfaces]:-unknown}"

auto_network="false"
if [[ "${PROBE[loopback_listener]:-blocked}" == "ok" && "${PROBE[host_interfaces]:-blocked}" == "ok" ]]; then
  case "${PROBE[ipv6_loopback_listener]:-blocked}" in
    ok|unavailable)
      auto_network="true"
      ;;
  esac
fi

if [[ "${auto_network}" == "true" ]]; then
  echo "==> go test (networked packages): ${NETWORK_PACKAGES[*]}"
  (
    cd "${SERVER_DIR}"
    go test "${NETWORK_PACKAGES[@]}"
  )
  exit 0
fi

echo "==> skipped networked package group"
echo "network_sensitive_tests=${PROBE[network_sensitive_tests]:-blocked}"
echo "loopback_listener_reason=${PROBE[loopback_listener_reason]:-none}"
echo "ipv6_loopback_listener_reason=${PROBE[ipv6_loopback_listener_reason]:-none}"
echo "host_interfaces_reason=${PROBE[host_interfaces_reason]:-none}"
echo "skipped_packages=${NETWORK_PACKAGES[*]}"
echo "run_full_validation=make server-test"
