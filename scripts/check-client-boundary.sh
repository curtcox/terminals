#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CLIENT_LIB="${CLIENT_LIB:-${ROOT_DIR}/terminal_client/lib}"
UI_DIR="${CLIENT_LIB}/ui"

if [[ ! -d "${CLIENT_LIB}" ]]; then
  echo "missing client lib directory: ${CLIENT_LIB}"
  exit 1
fi

if ! command -v rg >/dev/null 2>&1; then
  echo "ERROR: rg is required for client boundary scanning"
  exit 1
fi

pattern='terminal_root|photo[_ -]?frame|red[_ -]?alert|kitchen[_ -]?timer|package_id|com\.example'

matches="$(
  rg --line-number --no-heading --glob '!gen/**' --glob '!**/*.pb*.dart' \
    --regexp "${pattern}" "${CLIENT_LIB}" || true
)"

if [[ -n "${matches}" ]]; then
  echo "ERROR: production Flutter client contains scenario or package-id tokens"
  echo
  printf '%s\n' "${matches}"
  echo
  echo "Move scenario behavior to the server, or keep server-provided names in tests/fixtures only."
  exit 1
fi

if [[ -d "${UI_DIR}" ]]; then
  ui_import_pattern='import ['\''"]package:terminal_client/(app|capabilities|connection|diagnostics|discovery|edge|io|media|testing|util)/|import ['\''"]\.\./(app|capabilities|connection|diagnostics|discovery|edge|io|media|testing|util)/'
  ui_import_matches="$(
    rg --line-number --no-heading --glob '*.dart' \
      --regexp "${ui_import_pattern}" "${UI_DIR}" || true
  )"

  if [[ -n "${ui_import_matches}" ]]; then
    echo "ERROR: server-driven renderer code imports client subsystems"
    echo
    printf '%s\n' "${ui_import_matches}"
    echo
    echo "Keep terminal_client/lib/ui generic: render protobuf UI descriptors and emit ServerDrivenAction only."
    exit 1
  fi
fi

echo "client boundary scan passed"
