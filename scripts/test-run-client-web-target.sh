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
  if ! grep -Fq "$needle" "$MAKEFILE"; then
    fail "expected Makefile to contain: $needle"
  fi
}

assert_not_contains() {
  local needle="$1"
  if grep -Fq "$needle" "$MAKEFILE"; then
    fail "did not expect Makefile to contain: $needle"
  fi
}

assert_contains "run-client-web:"
assert_contains "flutter build web --no-wasm-dry-run"
assert_contains 'python3 -m http.server $(CLIENT_WEB_PORT) --bind $(CLIENT_WEB_HOST) --directory build/web'
assert_not_contains 'cd terminal_client && flutter run -d web-server --web-port=$(CLIENT_WEB_PORT) --web-hostname=$(CLIENT_WEB_HOST)'

echo "PASS: run-client-web target uses build + static server"
