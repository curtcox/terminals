#!/usr/bin/env python3
"""Validate every resolved-bug record under terminal_server/bug_reports/resolved/.

Run from `make all-check` (via the `bug-resolved-check` target). Catches drift:
malformed JSON, missing required fields, or `fix_commits` SHAs that don't
exist in the local git history (e.g. record from a force-pushed branch).

Exit status is non-zero on any failure.
"""
import json
import subprocess
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
RESOLVED_DIR = REPO / "terminal_server" / "bug_reports" / "resolved"

REQUIRED_FIELDS = {
    "report_id": str,
    "bug_token_word": str,
    "description": str,
    "resolved_at": str,
    "fix_commits": list,
    "regression_tests": list,
    "root_cause": str,
}


def _commit_exists(sha: str) -> bool:
    try:
        subprocess.run(
            ["git", "-C", str(REPO), "cat-file", "-e", f"{sha}^{{commit}}"],
            check=True,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )
        return True
    except subprocess.CalledProcessError:
        return False


def _check_one(path: Path) -> "list[str]":
    errors: "list[str]" = []
    try:
        data = json.loads(path.read_bytes())
    except Exception as exc:
        return [f"{path.name}: invalid JSON ({exc})"]
    if not isinstance(data, dict):
        return [f"{path.name}: top-level value is not an object"]
    for field, ftype in REQUIRED_FIELDS.items():
        if field not in data:
            errors.append(f"{path.name}: missing field '{field}'")
        elif not isinstance(data[field], ftype):
            errors.append(f"{path.name}: field '{field}' must be {ftype.__name__}")
    if data.get("fix_commits") == []:
        errors.append(f"{path.name}: 'fix_commits' must not be empty")
    if data.get("regression_tests") == []:
        errors.append(f"{path.name}: 'regression_tests' must not be empty")
    for sha in data.get("fix_commits") or []:
        if not isinstance(sha, str):
            errors.append(f"{path.name}: fix_commits entries must be strings")
            continue
        if not _commit_exists(sha):
            errors.append(f"{path.name}: fix_commits SHA '{sha}' not found in git history")
    # report_id must match filename stem so files stay traceable.
    if data.get("report_id") and data["report_id"] != path.stem:
        errors.append(f"{path.name}: report_id '{data['report_id']}' does not match filename")
    return errors


def main() -> int:
    if not RESOLVED_DIR.is_dir():
        # Dir is committed (with .gitkeep); absence is a real problem.
        print(f"error: {RESOLVED_DIR.relative_to(REPO)} is missing", file=sys.stderr)
        return 1
    failures: "list[str]" = []
    count = 0
    for path in sorted(RESOLVED_DIR.glob("*.json")):
        count += 1
        failures.extend(_check_one(path))
    if failures:
        for line in failures:
            print(line, file=sys.stderr)
        print(f"\n{len(failures)} problem(s) in {count} resolved-bug record(s)", file=sys.stderr)
        return 1
    print(f"OK: {count} resolved-bug record(s) validated")
    return 0


if __name__ == "__main__":
    sys.exit(main())
