#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ASSERT_SCRIPT="${ROOT_DIR}/scripts/assert-server-test-network-probe.sh"

fail() {
  echo "FAIL: $1" >&2
  exit 1
}

run_with_probe_output() {
  local output_file="$1"
  local probe_output="$2"

  set +e
  PROBE_OUTPUT="${probe_output}" bash "${ASSERT_SCRIPT}" >"${output_file}" 2>&1
  local status=$?
  set -e
  return "${status}"
}

assert_contains() {
  local file="$1"
  local needle="$2"
  if ! grep -Fq -- "${needle}" "${file}"; then
    fail "expected '${needle}' in ${file}"
  fi
}

test_passes_when_probe_is_supported() {
  local output_file
  output_file="$(mktemp)"

  if ! run_with_probe_output "${output_file}" $'loopback_listener=ok\nipv6_loopback_listener=unavailable\nhost_interfaces=ok\nnetwork_sensitive_tests=ok'; then
    cat "${output_file}" >&2
    rm -f "${output_file}"
    fail "expected assertion script to pass"
  fi

  assert_contains "${output_file}" "PASS: server network probe reports listener coverage support"
  rm -f "${output_file}"
}

test_fails_when_loopback_is_blocked() {
  local output_file
  output_file="$(mktemp)"

  if run_with_probe_output "${output_file}" $'loopback_listener=blocked\nipv6_loopback_listener=ok\nhost_interfaces=ok\nnetwork_sensitive_tests=blocked'; then
    cat "${output_file}" >&2
    rm -f "${output_file}"
    fail "expected assertion script to fail"
  fi

  assert_contains "${output_file}" "FAIL: loopback_listener must be ok in CI"
  rm -f "${output_file}"
}

test_fails_when_network_sensitive_is_not_ok() {
  local output_file
  output_file="$(mktemp)"

  if run_with_probe_output "${output_file}" $'loopback_listener=ok\nipv6_loopback_listener=ok\nhost_interfaces=ok\nnetwork_sensitive_tests=blocked'; then
    cat "${output_file}" >&2
    rm -f "${output_file}"
    fail "expected assertion script to fail"
  fi

  assert_contains "${output_file}" "FAIL: network_sensitive_tests must be ok in CI"
  rm -f "${output_file}"
}

test_passes_when_probe_is_supported
test_fails_when_loopback_is_blocked
test_fails_when_network_sensitive_is_not_ok

echo "PASS: server-test network probe assertion checks"
