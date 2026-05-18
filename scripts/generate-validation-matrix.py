#!/usr/bin/env python3
"""Generate docs/usecase-validation-matrix.md from usecases.md and plan frontmatter.

Walks usecases.md to extract every use-case ID (rows whose first column matches
^[A-Z]+\\d+$) and the family heading it sits under. Walks plans/ recursively for
plans whose YAML frontmatter declares ``validation: automated:<ID>`` (or a
comma-separated list ``automated:<ID1>,<ID2>,...`` for plans that span multiple
use cases). Writes
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
USECASES = REPO / "usecases.md"  # legacy single-file form (fallback)
USECASES_DIR = REPO / "usecases"
PLANS = REPO / "plans"
OUT = REPO / "docs" / "usecase-validation-matrix.md"
USECASE_VALIDATE_SH = REPO / "scripts" / "usecase-validate.sh"
USECASE_VALIDATE_YAML = REPO / "scripts" / "usecase-validate.yaml"

ID_RE = re.compile(r"^[A-Z]+\d+$")
HEADING_RE = re.compile(r"^(#{2,4})\s+(.*)$")
AUTOMATED_RE = re.compile(r"^automated:([A-Za-z0-9]+(?:,[A-Za-z0-9]+)*)$")
ALL_IDS_RE = re.compile(r"^all_ids=\(([^)]*)\)\s*$")
# Matches: `    AA1) echo "AA1|Simulation|description..." ;;`
METADATA_RE = re.compile(r"""^\s+\w+\)\s+echo\s+"([A-Z]+\d+)\|([^|"]+)\|([^"]+)"\s+;;\s*$""")

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


def parse_frontmatter(text: str) -> "dict | None":
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


def extract_table_cells(line: str) -> "list[str] | None":
    s = line.strip()
    if not s.startswith("|") or not s.endswith("|"):
        return None
    cells = [c.strip() for c in s.strip("|").split("|")]
    return cells


def _scan_usecase_text(text: str, default_family: "str | None",
                       out: "OrderedDict[str, dict]") -> None:
    family = default_family
    for raw in text.splitlines():
        m = HEADING_RE.match(raw)
        if m:
            level = len(m.group(1))
            title = m.group(2).strip()
            if level in (1, 2, 3, 4):
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


def parse_usecases() -> "OrderedDict[str, dict]":
    """Return ordered mapping: id -> {family, scenario}.

    Prefers the per-family layout under `usecases/*.md` (post-F2). Falls back
    to the legacy single-file `usecases.md` if the directory is absent.
    """
    out: "OrderedDict[str, dict]" = OrderedDict()
    if USECASES_DIR.is_dir():
        for path in sorted(USECASES_DIR.glob("*.md")):
            if path.name == "INDEX.md":
                continue
            text = path.read_text()
            # Use the H1 heading (or filename) as the default family label so
            # rows in single-section files get a sensible family even without
            # a heading above the table.
            default = path.stem.replace("-", " ").title()
            _scan_usecase_text(text, default, out)
        return out
    if USECASES.exists():
        _scan_usecase_text(USECASES.read_text(), None, out)
    return out


def _parse_yaml_usecases_simple(text: str) -> list[dict]:
    """Minimal YAML parser sufficient for usecase-validate.yaml.

    Extracts id, family, and description from the `usecases:` list.
    Used as a fallback when pyyaml is not installed.
    """
    entries: list[dict] = []
    current: "dict | None" = None
    for line in text.splitlines():
        stripped = line.rstrip()
        content = stripped.lstrip()
        if content.startswith("- id:"):
            if current is not None:
                entries.append(current)
            current = {"id": content[len("- id:"):].strip().strip('"').strip("'")}
            continue
        if current is None:
            continue
        for key in ("family", "description"):
            prefix = f"{key}:"
            if content.startswith(prefix) and not content.startswith("- "):
                val = content[len(prefix):].strip()
                if val.startswith('"') and val.endswith('"'):
                    val = val[1:-1]
                elif val.startswith("'") and val.endswith("'"):
                    val = val[1:-1]
                current[key] = val
                break
    if current is not None:
        entries.append(current)
    return entries


def load_validate_yaml() -> dict[str, dict]:
    """Load scripts/usecase-validate.yaml and return {id: entry} mapping."""
    if not USECASE_VALIDATE_YAML.exists():
        return {}
    text = USECASE_VALIDATE_YAML.read_text()
    try:
        import yaml
        data = yaml.safe_load(text)
        return {uc["id"]: uc for uc in data.get("usecases", [])}
    except ImportError:
        entries = _parse_yaml_usecases_simple(text)
        return {e["id"]: e for e in entries}


def parse_usecase_validate_sh() -> list[str]:
    """Return validated IDs from scripts/usecase-validate.yaml (preferred) or
    the legacy all_ids=(...) line in scripts/usecase-validate.sh."""
    yaml_entries = load_validate_yaml()
    if yaml_entries:
        # Use the canonical order from the helper module's ALL_IDS_ORDER list.
        # We replicate the order here to avoid importing the helper.
        canonical_order = [
            "AB1", "AB2", "AB3", "AB4", "AB5", "AB6", "AB7",
            "AA1", "AA2", "AA3", "AA4", "AA5", "AA6",
            "B1", "B2", "B3", "B4", "B5",
            "C1", "C2", "C3", "C5",
            "I3", "I4", "I6",
            "D3", "D1", "D2",
            "M1", "M2", "M3", "M4", "M5",
            "P2", "P3", "P4",
            "S1", "S2", "S3",
            "P1",
            "PL1", "PL8", "PL20",
            "T1", "T2", "T3", "T4",
            "UI1", "UI2", "UI3", "UI4", "UI5", "UI6", "UI7", "UI8", "UI9", "UI10",
            "V1", "V2", "V3",
        ]
        return [uid for uid in canonical_order if uid in yaml_entries]
    # Legacy fallback: parse all_ids=(...) from the shell script.
    if not USECASE_VALIDATE_SH.exists():
        return []
    for raw in USECASE_VALIDATE_SH.read_text().splitlines():
        m = ALL_IDS_RE.match(raw.strip())
        if not m:
            continue
        return [tok for tok in m.group(1).split() if ID_RE.match(tok)]
    return []


def parse_sh_metadata() -> dict[str, tuple[str, str]]:
    """Return {id: (depth, evidence)} from scripts/usecase-validate.yaml (preferred)
    or from the metadata() function body in usecase-validate.sh (legacy fallback).

    This is the authoritative source for coverage depth and primary evidence.
    """
    yaml_entries = load_validate_yaml()
    if yaml_entries:
        out: dict[str, tuple[str, str]] = {}
        for uid, entry in yaml_entries.items():
            depth = entry.get("family", "TODO")
            evidence = entry.get("description", "TODO")
            out[uid] = (depth, evidence)
        return out
    # Legacy fallback: parse shell script metadata() case statement.
    if not USECASE_VALIDATE_SH.exists():
        return {}
    out = {}
    in_metadata = False
    for raw in USECASE_VALIDATE_SH.read_text().splitlines():
        stripped = raw.strip()
        if stripped.startswith("metadata()"):
            in_metadata = True
            continue
        if in_metadata and stripped == "}":
            break
        if not in_metadata:
            continue
        m = METADATA_RE.match(raw)
        if m:
            uid, depth, evidence = m.group(1), m.group(2).strip(), m.group(3).strip()
            out[uid] = (depth, evidence)
    return out


def collect_plan_validations() -> tuple[dict[str, list[dict]], list[str]]:
    """Walk plans/, return {id: [{path,title,status}, ...]} and errors."""
    by_id: dict[str, list[dict]] = {}
    errors: list[str] = []
    for root, _, names in os.walk(PLANS):
        for n in names:
            if not n.endswith(".md"):
                continue
            if n in ("INDEX.md", "README.md", "progress.md"):
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
            rel = path.relative_to(REPO).as_posix()
            entry = {
                "path": rel,
                "title": fm.get("title", rel),
                "status": fm.get("status", "?"),
            }
            for uid in m.group(1).split(","):
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
    sh_metadata: dict[str, tuple[str, str]] | None = None,
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
        "`usecases/*.md`, plan frontmatter (`validation: automated:<ID>`), and "
        "the `all_ids` list in `scripts/usecase-validate.yaml`. Do not edit by "
        "hand — update the source files and run `make validation-matrix`._"
    )
    lines.append("")
    lines.append("This matrix maps `usecases/*.md` IDs to current automated validation coverage.")
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
        # sh_metadata (from usecase-validate.sh metadata()) is the primary source;
        # fall back to the existing hand-maintained matrix for IDs not yet in metadata().
        if sh_metadata and uid in sh_metadata:
            depth, evidence = sh_metadata[uid]
        else:
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
    sh_metadata = parse_sh_metadata()
    rendered, todos = render(usecases, plan_by_id, sh_ids, existing, sh_metadata)

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
