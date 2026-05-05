#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$ROOT_DIR/terminal_server"
GOCACHE="${GOCACHE:-$ROOT_DIR/.cache/go-build}" go test ./internal/contracttest

cd "$ROOT_DIR/terminal_client"
mkdir -p "$ROOT_DIR/.home"
HOME="${TERMINALS_CONTRACT_HOME:-$ROOT_DIR/.home}" \
  PUB_CACHE="${PUB_CACHE:-$ROOT_DIR/.home/.pub-cache}" \
  FLUTTER_SUPPRESS_ANALYTICS=true \
  DART_SUPPRESS_ANALYTICS=true \
  "$ROOT_DIR/.sdk/flutter/bin/dart" test/contract/contract_golden_test.dart
