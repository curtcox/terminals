#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

make -C "$ROOT_DIR" proto-contract-generate proto-contract-test
git -C "$ROOT_DIR" diff --exit-code -- api/testdata/contract
