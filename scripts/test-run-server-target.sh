#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MAKEFILE="${ROOT_DIR}/Makefile"

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

assert_contains "run-server:"
assert_contains 'TERMINALS_GRPC_HOST=0.0.0.0'
assert_contains 'TERMINALS_CONTROL_WS_HOST=0.0.0.0'
assert_contains 'TERMINALS_CONTROL_TCP_HOST=0.0.0.0'
assert_contains 'TERMINALS_CONTROL_HTTP_HOST=0.0.0.0'
assert_contains 'TERMINALS_ADMIN_HTTP_HOST=0.0.0.0'
assert_contains 'TERMINALS_BUILD_SHA=$(BUILD_SHA)'
assert_contains 'TERMINALS_BUILD_DATE=$(BUILD_DATE)'

echo "PASS: run-server target defaults listeners to LAN-safe bind hosts"
