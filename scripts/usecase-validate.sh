#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
export PATH="${ROOT_DIR}/.bin:${ROOT_DIR}/.sdk/flutter/bin:${PATH}"

INFO="${INFO:-}"
if [[ "${1:-}" == "--info" ]]; then
  INFO="1"
  shift
fi

USECASE="${1:-${USECASE:-}}"
USECASE="$(echo "${USECASE}" | tr -d '[:space:]')"

if [[ -z "${USECASE}" ]]; then
  echo "usage: make usecase-validate USECASE=<ID|all> [INFO=1]"
  echo "   or: scripts/usecase-validate.sh --info <ID|all>"
  echo "example: make usecase-validate USECASE=C1"
  exit 2
fi

HELPER="${ROOT_DIR}/scripts/usecase-validate-helper.py"

metadata() {
  python3 "${HELPER}" --info "$1"
}

IFS=' ' read -r -a all_ids <<< "$(python3 "${HELPER}" --ids)"

run_usecase() {
  python3 "${HELPER}" --run "$1"
}

write_usecase_result() {
  local id="$1"
  local started="$2"
  local status="$3"
  local failure="${4:-}"
  RESULT_USECASE_ID="${id}" \
    RESULT_STARTED="${started}" \
    RESULT_STATUS="${status}" \
    RESULT_FAILURE="${failure}" \
    RESULT_ROOT="${ROOT_DIR}" \
    python3 - <<'PY'
import json
import os
from datetime import datetime, timezone
from pathlib import Path

root = Path(os.environ["RESULT_ROOT"])
usecase_id = os.environ["RESULT_USECASE_ID"]
started = os.environ["RESULT_STARTED"]
status = int(os.environ["RESULT_STATUS"])
failure = os.environ.get("RESULT_FAILURE", "")
ended = datetime.now(timezone.utc)
result = {
    "run_id": str(int(ended.timestamp() * 1_000_000_000)),
    "usecase_id": usecase_id,
    "scenario_name": f"scripts/usecase-validate.sh {usecase_id}",
    "timestamp_start": started,
    "timestamp_end": ended.isoformat().replace("+00:00", "Z"),
    "pass": status == 0,
}
if status != 0:
    result["failing_assertions"] = [failure or f"use-case validation exited with status {status}"]

out = root / "artifacts" / "usecases" / usecase_id / "result.json"
if out.exists():
    try:
        previous = json.loads(out.read_text())
    except json.JSONDecodeError:
        previous = {}
    if previous.get("usecase_id") == usecase_id and previous.get("interaction_trace"):
        result["interaction_trace"] = previous["interaction_trace"]
    if previous.get("usecase_id") == usecase_id and previous.get("media"):
        result["media"] = previous["media"]
out.parent.mkdir(parents=True, exist_ok=True)
out.write_text(json.dumps(result, indent=2) + "\n")
PY
}

run_and_record_usecase() {
  local id="$1"
  local started
  started="$(python3 - <<'PY'
from datetime import datetime, timezone
print(datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"))
PY
)"
  if run_usecase "${id}"; then
    write_usecase_result "${id}" "${started}" 0
    return 0
  fi
  local status="$?"
  write_usecase_result "${id}" "${started}" "${status}" "use-case validation exited with status ${status}"
  return "${status}"
}

if [[ "${INFO}" == "1" ]]; then
  if [[ "${USECASE}" == "all" ]]; then
    for id in "${all_ids[@]}"; do
      metadata "${id}"
    done
    exit 0
  fi
  metadata "${USECASE}"
  exit 0
fi

if [[ "${USECASE}" == "all" ]]; then
  for id in "${all_ids[@]}"; do
    printf "\n### Validating %s\n" "${id}"
    run_and_record_usecase "${id}"
  done
  printf "\nAll supported use-case validations passed.\n"
  exit 0
fi

run_and_record_usecase "${USECASE}"
printf "\nUse-case %s validation passed.\n" "${USECASE}"
