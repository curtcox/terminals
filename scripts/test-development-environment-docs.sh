#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MATRIX="${ROOT_DIR}/docs/usecase-validation-matrix.md"
VALIDATOR="${ROOT_DIR}/scripts/usecase-validate.sh"

fail() {
  echo "FAIL: $1" >&2
  exit 1
}

if ! grep -Fq "## Implementation Status" "${ROOT_DIR}"/docs/tal-example-*.md; then
  fail "each docs/tal-example-*.md file must include ## Implementation Status"
fi

if ! grep -Fq "Coverage Depth" "${MATRIX}"; then
  fail "use-case validation matrix must include a Coverage Depth column"
fi

mapfile -t script_ids < <(
  sed -n '/case "${id}" in/,/\*)/p' "${VALIDATOR}" |
    sed -n 's/^[[:space:]]*\([A-Z][0-9][0-9]*\))$/\1/p'
)

if [[ "${#script_ids[@]}" -eq 0 ]]; then
  fail "could not find use-case IDs in ${VALIDATOR}"
fi

for id in "${script_ids[@]}"; do
  row="$(grep -E "^[|][[:space:]]*${id}[[:space:]]*[|]" "${MATRIX}" || true)"
  if [[ -z "${row}" ]]; then
    fail "missing matrix row for ${id}"
  fi

  depth="$(awk -F'|' '{gsub(/^[[:space:]]+|[[:space:]]+$/, "", $6); print $6}' <<<"${row}")"
  case "${depth}" in
    Smoke|Transport|Scenario|Contract|Simulation|Full)
      ;;
    *)
      fail "matrix row ${id} has invalid or empty Coverage Depth: ${depth:-<empty>}"
      ;;
  esac
done

echo "PASS: development environment docs are consistent"
