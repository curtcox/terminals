#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CLIENT_LIB="${ROOT_DIR}/terminal_client/lib"

if [[ ! -d "${CLIENT_LIB}" ]]; then
  echo "missing client lib directory: ${CLIENT_LIB}"
  exit 1
fi

if ! command -v rg >/dev/null 2>&1; then
  echo "ERROR: rg is required for client boundary scanning"
  exit 1
fi

pattern='photo[_ -]?frame|red[_ -]?alert|kitchen[_ -]?timer|package_id|com\.example'

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

echo "client boundary scan passed"

