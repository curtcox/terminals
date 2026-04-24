#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MAKEFILE="${ROOT_DIR}/Makefile"
STOP_SCRIPT="${ROOT_DIR}/scripts/stop-server.sh"
STOP_SAFETY_TEST_SCRIPT="${ROOT_DIR}/scripts/test-stop-server-safety.sh"

fail() {
  echo "FAIL: $1" >&2
  exit 1
}

assert_contains() {
  local needle="$1"
  if ! grep -Fq -- "$needle" "$MAKEFILE"; then
    fail "expected Makefile to contain: $needle"
  fi
}

assert_contains "stop-server:"
assert_contains "./scripts/stop-server.sh"
assert_contains "stop-server-test:"
assert_contains "./scripts/test-stop-server-safety.sh"

if [[ ! -x "${STOP_SCRIPT}" ]]; then
  fail "expected stop script to be executable: ${STOP_SCRIPT}"
fi

if [[ ! -x "${STOP_SAFETY_TEST_SCRIPT}" ]]; then
  fail "expected stop-server safety test script to be executable: ${STOP_SAFETY_TEST_SCRIPT}"
fi

echo "PASS: stop-server target and script are configured"
