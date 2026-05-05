#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$ROOT_DIR/terminal_server"
GOCACHE="${GOCACHE:-$ROOT_DIR/.cache/go-build}" go run ./cmd/proto-contract-generate \
  --manifest ../api/testdata/contract/manifest.yaml \
  --root ..
