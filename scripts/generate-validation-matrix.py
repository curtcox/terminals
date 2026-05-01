#!/usr/bin/env python3
"""Generate docs/usecase-validation-matrix.md from usecases.md and plan frontmatter.

Walks usecases.md to extract every use-case ID (rows whose first column matches
^[A-Z]+\\d+$) and the family heading it sits under. Walks plans/ recursively for
plans whose YAML frontmatter declares ``validation: automated:<ID>``. Writes
docs/usecase-validation-matrix.md with two sections:

- "Automated IDs": IDs declared automated by some plan, joined to the
  use-case scenario, with hand-written evidence/coverage carried forward
  from the existing matrix.
- "Planned / Not Yet Automated": IDs defined in usecases.md but not declared
  automated by any plan.

Exits non-zero if a plan declares ``automated:X`` for an X not defined in
usecases.md, and (with --strict) if any TODO placeholders remain.
"""
import argparse
import os
import re
import sys
from collections import OrderedDict
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
USECASES = REPO / "usecases.md"
PLANS = REPO / "plans"
OUT = REPO / "docs" / "usecase-validation-matrix.md"
USECASE_VALIDATE_SH = REPO / "scripts" / "usecase-validate.sh"

ID_RE = re.compile(r"^[A-Z]+\d+$")
HEADING_RE = re.compile(r"^(#{2,4})\s+(.*)$")
AUTOMATED_RE = re.compile(r"^automated:([A-Za-z0-9]+)$")
ALL_IDS_RE = re.compile(r"^all_ids=\(([^)]*)\)\s*$")

COVERAGE_DEPTH_LEGEND = [
    ("Smoke", "proves a narrow server loop or command path."),
    ("Transport", "proves generated/wire control-plane behavior."),
    ("Scenario", "proves scenario matching and server-side side effects."),
    ("Contract", "proves app/package/runtime contract surfaces."),
    ("Simulation", "proves lifecycle behavior against synthetic time/events."),
    (
        "Full",
        "covers trigger, placement, UI, scheduling, side effects, and "
        "expiry/cancel/resume behavior.",
    ),
]


def parse_frontmatter(text: str) -> dict | None:
    if not text.startswith("---\n"):
        return None
    end = text.find("\n---\n", 4)
    if end < 0:
        return None
    block = text[4:end]
    fm = {}
    for line in block.splitlines():
        if ":" not in line:
            continue
        key, _, value = line.partition(":")
        value = value.strip()
        if value.startswith('"') and value.endswith('"'):
            value = value[1:-1]
        fm[key.strip()] = value
    return fm


def extract_table_cells(line: str) -> list[str] | None:
    s = line.strip()
    if not s.startswith("|") or not s.endswith("|"):
        return None
    cells = [c.strip() for c in s.strip("|").split("|")]
    return cells


def parse_usecases() -> "OrderedDict[str, dict]":
    """Return ordered mapping: id -> {family, scenario, source_line}."""
    text = USECASES.read_text()
    out: "OrderedDict[str, dict]" = OrderedDict()
    family = None
    for raw in text.splitlines():
        m = HEADING_RE.match(raw)
        if m:
            level = len(m.group(1))
            title = m.group(2).strip()
            if level in (3, 4):
                family = title
            continue
        cells = extract_table_cells(raw)
        if not cells:
            continue
        first = cells[0]
        if not ID_RE.match(first):
            continue
        scenario = cells[2] if len(cells) > 2 else ""
        scenario = scenario.replace("**", "")
        if first in out:
            continue
        out[first] = {"family": family or "(unknown)", "scenario": scenario}
    return out


def parse_usecase_validate_sh() -> list[str]:
    """Extract IDs from `all_ids=(...)` in scripts/usecase-validate.sh."""
    if not USECASE_VALIDATE_SH.exists():
        return []
    for raw in USECASE_VALIDATE_SH.read_text().splitlines():
        m = ALL_IDS_RE.match(raw.strip())
        if not m:
            continue
        return [tok for tok in m.group(1).split() if ID_RE.match(tok)]
    return []


def collect_plan_validations() -> tuple[dict[str, list[dict]], list[str]]:
    """Walk plans/, return {id: [{path,title,status}, ...]} and errors."""
    by_id: dict[str, list[dict]] = {}
    errors: list[str] = []
    for root, _, names in os.walk(PLANS):
        for n in names:
            if not n.endswith(".md"):
                continue
            if n in ("INDEX.md", "README.md"):
                continue
            path = Path(root) / n
            text = path.read_text()
            fm = parse_frontmatter(text)
            if fm is None:
                continue
            validation = fm.get("validation", "")
            m = AUTOMATED_RE.match(validation)
            if not m:
                continue
            uid = m.group(1)
            rel = path.relative_to(REPO).as_posix()
            entry = {
                "path": rel,
                "title": fm.get("title", rel),
                "status": fm.get("status", "?"),
            }
            by_id.setdefault(uid, []).append(entry)
    return by_id, errors


def parse_existing_evidence() -> dict[str, tuple[str, str]]:
    """Read the existing matrix and return {id: (evidence, depth)} pairs."""
    if not OUT.exists():
        return {}
    out: dict[str, tuple[str, str]] = {}
    in_table = False
    for raw in OUT.read_text().splitlines():
        s = raw.strip()
        if s.startswith("|---"):
            in_table = True
            continue
        if not s.startswith("|"):
            in_table = False
            continue
        if not in_table:
            continue
        cells = extract_table_cells(s)
        if not cells or len(cells) < 5:
            continue
        uid = cells[0]
        if not ID_RE.match(uid):
            continue
        evidence = cells[3]
        depth = cells[4]
        out[uid] = (evidence, depth)
    return out


def render(
    usecases: "OrderedDict[str, dict]",
    plan_by_id: dict[str, list[dict]],
    sh_ids: set[str],
    existing: dict[str, tuple[str, str]],
) -> tuple[str, list[str]]:
    """Return (rendered text, list of TODO IDs)."""
    automated_set = set(plan_by_id.keys()) | sh_ids
    automated_ids = [uid for uid in usecases.keys() if uid in automated_set]
    planned_ids = [uid for uid in usecases.keys() if uid not in automated_set]

    todos: list[str] = []
    lines: list[str] = []
    lines.append("# Use Case Validation Matrix")
    lines.append("")
    lines.append(
        "_Auto-generated by `scripts/generate-validation-matrix.py` from "
        "`usecases.md`, plan frontmatter (`validation: automated:<ID>`), and "
        "the `all_ids` list in `scripts/usecase-validate.sh`. Do not edit by "
        "hand — update the source files and run `make validation-matrix`._"
    )
    lines.append("")
    lines.append("This matrix maps `usecases.md` IDs to current automated validation coverage.")
    lines.append("")
    lines.append("Primary command:")
    lines.append("")
    lines.append("```bash")
    lines.append("make usecase-validate USECASE=<ID>")
    lines.append("# or")
    lines.append("make usecase-validate USECASE=all")
    lines.append("```")
    lines.append("")
    lines.append("## Automated IDs")
    lines.append("")
    lines.append("Coverage depth labels:")
    lines.append("")
    for label, desc in COVERAGE_DEPTH_LEGEND:
        lines.append(f"- `{label}`: {desc}")
    lines.append("")
    lines.append("| ID | Scenario | Validation Command | Primary Evidence | Coverage Depth |")
    lines.append("|---|---|---|---|---|")
    for uid in automated_ids:
        scenario = usecases[uid]["scenario"]
        evidence, depth = existing.get(uid, ("TODO", "TODO"))
        if evidence == "TODO" or depth == "TODO":
            todos.append(uid)
        cmd = f"`make usecase-validate USECASE={uid}`"
        lines.append(f"| {uid} | {scenario} | {cmd} | {evidence} | {depth} |")
    lines.append("")
    lines.append("## Planned / Not Yet Automated")
    lines.append("")
    lines.append(
        "The following planned IDs are not declared automated by any plan and "
        "are not wired into `scripts/usecase-validate.sh`:"
    )
    lines.append("")
    lines.append(", ".join(f"`{uid}`" for uid in planned_ids) + ".")
    lines.append("")
    lines.append(
        "Use `make all-check` as the baseline repository gate while dedicated "
        "use-case mappings are added."
    )
    lines.append("")
    return "\n".join(lines), todos


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument(
        "--strict",
        action="store_true",
        help="Exit non-zero if any TODO placeholders remain in the output.",
    )
    ap.add_argument(
        "--check",
        action="store_true",
        help="Do not write; exit non-zero if the generated content differs from disk.",
    )
    args = ap.parse_args()

    usecases = parse_usecases()
    plan_by_id, errors = collect_plan_validations()
    sh_ids_list = parse_usecase_validate_sh()
    sh_ids = set(sh_ids_list)

    # Drift check (a): plan declares automated:X where X is not in usecases.md.
    broken: list[tuple[str, list[dict]]] = []
    for uid, entries in plan_by_id.items():
        if uid not in usecases:
            broken.append((uid, entries))
    if broken:
        print("ERROR: plans declare validation against unknown use-case IDs:", file=sys.stderr)
        for uid, entries in broken:
            for e in entries:
                print(f"  - {uid}: {e['path']}", file=sys.stderr)
        return 2

    # Drift check (b): scripts/usecase-validate.sh references unknown ID.
    sh_unknown = [uid for uid in sh_ids_list if uid not in usecases]
    if sh_unknown:
        print(
            "ERROR: scripts/usecase-validate.sh references unknown IDs: "
            + ", ".join(sh_unknown),
            file=sys.stderr,
        )
        return 2

    # Soft drift signals (warnings, do not fail): mismatch between the two
    # automated-source signals.
    plan_only = sorted(set(plan_by_id) - sh_ids)
    sh_only = sorted(sh_ids - set(plan_by_id))
    if plan_only:
        print(
            "warning: plan declares automated:<ID> but scripts/usecase-validate.sh "
            "has no entry: " + ", ".join(plan_only),
            file=sys.stderr,
        )
    if sh_only:
        print(
            "warning: scripts/usecase-validate.sh validates IDs but no plan "
            "declares automated:<ID>: " + ", ".join(sh_only),
            file=sys.stderr,
        )

    existing = parse_existing_evidence()
    rendered, todos = render(usecases, plan_by_id, sh_ids, existing)

    # Warn about new IDs that need TODO replacement.
    if todos:
        print(
            "warning: TODO placeholders for IDs (fill in evidence/depth manually): "
            + ", ".join(todos),
            file=sys.stderr,
        )

    if args.check:
        current = OUT.read_text() if OUT.exists() else ""
        if current != rendered:
            print(
                f"ERROR: {OUT.relative_to(REPO)} is out of date. "
                "Run `make validation-matrix`.",
                file=sys.stderr,
            )
            return 1
    else:
        OUT.write_text(rendered)
        automated_total = len(set(plan_by_id) | sh_ids)
        planned_total = sum(
            1 for u in usecases if u not in plan_by_id and u not in sh_ids
        )
        print(
            f"wrote {OUT.relative_to(REPO)} "
            f"({automated_total} automated IDs, {planned_total} planned)"
        )

    if args.strict and todos:
        print("ERROR: --strict and TODO placeholders remain.", file=sys.stderr)
        return 1

    if errors:
        for e in errors:
            print(e, file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
