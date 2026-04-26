#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

probe_output="${PROBE_OUTPUT:-}"
if [[ -z "${probe_output}" ]]; then
  probe_output="$(cd "${ROOT_DIR}" && go run ./scripts/probe_server_test_network.go)"
fi

probe_value() {
  local key="$1"
  awk -F'=' -v key="${key}" '$1 == key { print substr($0, index($0, "=") + 1) }' <<<"${probe_output}" | tail -n1
}

fail_probe() {
  local message="$1"
  echo "FAIL: ${message}" >&2
  echo "--- probe output ---" >&2
  printf '%s\n' "${probe_output}" >&2
  echo "--- end probe output ---" >&2
  exit 1
}

loopback="$(probe_value loopback_listener)"
ipv6_loopback="$(probe_value ipv6_loopback_listener)"
host_ifaces="$(probe_value host_interfaces)"
network_sensitive="$(probe_value network_sensitive_tests)"

if [[ "${loopback}" != "ok" ]]; then
  fail_probe "loopback_listener must be ok in CI"
fi

case "${ipv6_loopback}" in
  ok|unavailable)
    ;;
  *)
    fail_probe "ipv6_loopback_listener must be ok or unavailable in CI"
    ;;
esac

if [[ "${host_ifaces}" != "ok" ]]; then
  fail_probe "host_interfaces must be ok in CI"
fi

if [[ "${network_sensitive}" != "ok" ]]; then
  fail_probe "network_sensitive_tests must be ok in CI"
fi

echo "PASS: server network probe reports listener coverage support"
