#!/usr/bin/env python3
"""Audit that YAML entries and usecases/*.md are in sync.

Fails if:
- A YAML id has no matching row in usecases/<family>.md
- A plan file with `validation: automated:<ID>` has no YAML entry
"""
import os
import re
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
YAML_PATH = REPO / "scripts" / "usecase-validate.yaml"
USECASES_DIR = REPO / "usecases"
PLANS_DIR = REPO / "plans"

# Map ID prefix -> usecases/<file>.md
PREFIX_TO_FILE = {
    "PL": "plato.md",      # Must come before "P" to take priority
    "AB": "adjacent-business.md",
    "AA": "software-agents.md",
    "AH": "adjacent-home.md",
    "AO": "adjacent-office.md",
    "B":  "bug-reporting.md",
    "C":  "communication.md",
    "D":  "display.md",
    "I":  "infrastructure.md",
    "M":  "monitoring.md",
    "P":  "productivity.md",
    "S":  "security.md",
    "T":  "timers.md",
    "UI": "terminal-ui.md",
    "V":  "voice.md",
}

AUTOMATED_RE = re.compile(r"^automated:([A-Za-z0-9]+(?:,[A-Za-z0-9]+)*)$")
ID_RE = re.compile(r"^[A-Z]+\d+$")


def id_to_file(uid: str) -> "str | None":
    """Return the usecases/*.md filename for a given ID, or None if unmapped."""
    # Try longest prefix first (e.g., "PL" before "P", "AB" before "A").
    for prefix, filename in PREFIX_TO_FILE.items():
        if uid.startswith(prefix):
            return filename
    return None


def load_yaml_ids() -> list[str]:
    """Return list of IDs from YAML, in order.

    Uses pyyaml if available; falls back to regex extraction (sufficient for
    the simple `- id: <ID>` structure in usecase-validate.yaml).
    """
    if not YAML_PATH.exists():
        print(f"ERROR: {YAML_PATH.relative_to(REPO)} not found", file=sys.stderr)
        sys.exit(2)
    text = YAML_PATH.read_text()
    try:
        import yaml
        data = yaml.safe_load(text)
        return [uc["id"] for uc in data.get("usecases", [])]
    except ImportError:
        # Regex fallback: extract `- id: <ID>` lines.
        return re.findall(r"^\s*-\s+id:\s+([A-Z]+\d+)\s*$", text, re.MULTILINE)


def check_id_in_usecases(uid: str, errors: list[str]) -> None:
    """Check that uid appears in its expected usecases/*.md file."""
    filename = id_to_file(uid)
    if filename is None:
        errors.append(f"  {uid}: no usecases/ file mapping for this prefix")
        return
    path = USECASES_DIR / filename
    if not path.exists():
        errors.append(f"  {uid}: expected file {path.relative_to(REPO)} does not exist")
        return
    text = path.read_text()
    # Accept the ID appearing in a table row (e.g. "| C1 |" or "C1" anywhere).
    pattern = re.compile(r"\b" + re.escape(uid) + r"\b")
    if not pattern.search(text):
        errors.append(
            f"  {uid}: not found in {path.relative_to(REPO)}"
        )


def collect_plan_automated_ids() -> set[str]:
    """Walk plans/ and collect IDs declared automated in frontmatter."""
    ids: set[str] = set()
    for root, _, names in os.walk(PLANS_DIR):
        for n in names:
            if not n.endswith(".md"):
                continue
            if n in ("INDEX.md", "README.md", "progress.md"):
                continue
            path = Path(root) / n
            text = path.read_text()
            if not text.startswith("---\n"):
                continue
            end = text.find("\n---\n", 4)
            if end < 0:
                continue
            block = text[4:end]
            for line in block.splitlines():
                if not line.startswith("validation:"):
                    continue
                _, _, val = line.partition(":")
                val = val.strip().strip('"')
                m = AUTOMATED_RE.match(val)
                if m:
                    for uid in m.group(1).split(","):
                        if ID_RE.match(uid):
                            ids.add(uid)
    return ids


def main() -> int:
    errors: list[str] = []

    yaml_ids = load_yaml_ids()
    yaml_id_set = set(yaml_ids)

    # Check 1: every YAML id has a row in usecases/*.md
    for uid in yaml_ids:
        check_id_in_usecases(uid, errors)

    # Check 2: every plan `validation: automated:<ID>` has a YAML entry
    plan_ids = collect_plan_automated_ids()
    missing_from_yaml = sorted(plan_ids - yaml_id_set)
    for uid in missing_from_yaml:
        errors.append(
            f"  {uid}: plan declares automated:{uid} but ID is not in "
            f"scripts/usecase-validate.yaml"
        )

    if errors:
        print("usecase-wiring audit FAILED:", file=sys.stderr)
        for e in errors:
            print(e, file=sys.stderr)
        return 1

    print(
        f"usecase-wiring audit OK: {len(yaml_ids)} YAML IDs verified against "
        f"usecases/*.md; {len(plan_ids)} plan automated IDs cross-checked"
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
