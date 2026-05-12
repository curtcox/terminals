#!/usr/bin/env bash
# Terminal UI plan (Phase J): wiring checks for automated UI use cases.
# Invoked from `make usecase-wiring-audit` (part of `make all-check`).
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
cd "${REPO_ROOT}"

for id in UI1 UI2 UI3 UI4 UI5 UI6 UI7 UI8 UI9 UI10; do
  if ! grep -qE "\\b${id}\\b" scripts/usecase-validate.sh; then
    echo "audit: missing ${id} in scripts/usecase-validate.sh" >&2
    exit 1
  fi
done

if ! grep -q "validation: automated:UI1,UI2,UI3,UI4,UI5,UI6,UI7,UI8,UI9,UI10" \
  plans/features/terminal-ui/plan.md; then
  echo "audit: plans/features/terminal-ui/plan.md validation frontmatter drift" >&2
  exit 1
fi

if [[ ! -f usecases/terminal-ui.md ]]; then
  echo "audit: missing usecases/terminal-ui.md" >&2
  exit 1
fi

echo "terminal-ui audit: UI use-case wiring OK"
