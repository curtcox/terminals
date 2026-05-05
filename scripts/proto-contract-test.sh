#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DART_BIN="${TERMINALS_DART_BIN:-}"
if [[ -z "$DART_BIN" ]]; then
  if [[ -x "$ROOT_DIR/.sdk/flutter/bin/dart" ]]; then
    DART_BIN="$ROOT_DIR/.sdk/flutter/bin/dart"
  else
    DART_BIN="dart"
  fi
fi

cd "$ROOT_DIR/terminal_server"
GOCACHE="${GOCACHE:-$ROOT_DIR/.cache/go-build}" go test ./internal/contracttest -count=1

cd "$ROOT_DIR/terminal_client"
mkdir -p "$ROOT_DIR/.home"
HOME="${TERMINALS_CONTRACT_HOME:-$ROOT_DIR/.home}" \
  PUB_CACHE="${PUB_CACHE:-$ROOT_DIR/.home/.pub-cache}" \
  FLUTTER_SUPPRESS_ANALYTICS=true \
  DART_SUPPRESS_ANALYTICS=true \
  "$DART_BIN" test/contract/contract_golden_test.dart
