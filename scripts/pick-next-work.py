#!/usr/bin/env python3
"""Pick the next plan to work on, by the priority order in the agent prompt.

Walks plans/ and parses YAML frontmatter, then groups candidates into five
priority buckets used by the "pick the next work" prompt:

  0. Quality debt      — open bug reports; FLAG-severity oversized source files;
                         failing CI gates recorded in scripts/ci-status.json
  1. Needs attention   — shipped-buggy, shipped-untested, open audits/incidents
  2. In flight         — status: building
  3. Promote to validated — shipped-untested with no automated validation wired
  4. Planned, small    — status: planned, sorted ascending by file line count
                         (proxy for "small, well-scoped")

Outputs a markdown report. The first non-empty bucket's first entry is the
recommended pick. Larger plans are flagged as non-trivial so the agent knows
to STOP and report a plan before implementing.

CI gate status is read from scripts/ci-status.json, written by
`scripts/check-ci-gates.sh` (run via `make ci-status`).  If the file is
absent, CI gates are not surfaced — run `make ci-status` to populate it.

Usage:
  python3 scripts/pick-next-work.py            # human/agent-readable report
  python3 scripts/pick-next-work.py --json     # machine-readable JSON
"""
import json
import os
import sys
from collections import defaultdict
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
PLANS = REPO / "plans"
BUG_REPORTS_DIR = REPO / "terminal_server" / "logs" / "bug_reports"
RESOLVED_BUGS_DIR = REPO / "terminal_server" / "bug_reports" / "resolved"

ATTENTION_STATUSES = {"shipped-buggy", "shipped-untested", "open"}
NONTRIVIAL_LINE_THRESHOLD = 200  # plans larger than this need a plan-first pass
CI_STATUS_FILE = REPO / "scripts" / "ci-status.json"


def _resolved_bug_descriptions() -> "set[str]":
    """Return descriptions of resolved bugs that should be excluded from the work queue.

    Reads `terminal_server/bug_reports/resolved/*.json`. Each file is one
    resolved-bug record written by `scripts/bug-resolve.py`. The `description`
    field is normalized the same way live report descriptions are normalized
    below (whitespace collapsed, 80-char cap).
    """
    if not RESOLVED_BUGS_DIR.is_dir():
        return set()
    resolved: "set[str]" = set()
    for record_file in RESOLVED_BUGS_DIR.glob("*.json"):
        try:
            data = json.loads(record_file.read_bytes())
        except Exception:
            continue
        desc = data.get("description")
        if isinstance(desc, str) and desc:
            resolved.add(" ".join(desc.split())[:80])
    return resolved


def collect_open_bugs() -> "list[dict]":
    """Return one synthetic entry per distinct bug description in the bug log.

    Reads JSON reports under terminal_server/logs/bug_reports/.  Groups by
    description (trimmed to 80 chars), counts occurrences, and returns them
    sorted most-frequent first so the heaviest hitter is picked first.

    Descriptions with a matching record under
    `terminal_server/bug_reports/resolved/` are excluded (bug is fixed).

    Returns [] when the directory is absent (CI without a local log tree).
    """
    if not BUG_REPORTS_DIR.is_dir():
        return []
    from collections import Counter
    resolved = _resolved_bug_descriptions()
    counts: Counter[str] = Counter()
    for day_dir in sorted(BUG_REPORTS_DIR.iterdir()):
        if not day_dir.is_dir():
            continue
        for report_file in sorted(day_dir.glob("*.json")):
            try:
                import json as _json
                d = _json.loads(report_file.read_bytes())
            except Exception:
                continue
            desc = (d.get("summary") or {}).get("description") or ""
            desc = " ".join(desc.split())[:80] or "(no description)"
            if desc in resolved:
                continue
            counts[desc] += 1
    bugs = []
    for desc, cnt in counts.most_common():
        label = f"{desc} (×{cnt})" if cnt > 1 else desc
        bugs.append({
            "path": "terminal_server/logs/bug_reports",
            "title": f"Fix bug: {label}",
            "kind": "quality-debt",
            "status": "quality-debt",
            "owner": "unowned",
            "validation": "none",
            "lines": 0,
            "_debt_kind": "bug",
            "_count": cnt,
        })
    return bugs


def collect_ci_failures() -> "list[dict]":
    """Return synthetic quality-debt entries for failing CI gates.

    Reads ``scripts/ci-status.json``, written by ``scripts/check-ci-gates.sh``
    (run via ``make ci-status``).  Each gate whose ``status`` is ``"fail"``
    becomes a Priority 0 quality-debt item so it blocks feature work.

    Returns [] when the file is absent (gates not yet probed) or unreadable.
    """
    if not CI_STATUS_FILE.exists():
        return []
    try:
        data = json.loads(CI_STATUS_FILE.read_bytes())
    except Exception:
        return []
    entries = []
    for gate in data.get("gates", []):
        if gate.get("status") != "fail":
            continue
        name = gate.get("name", "unknown-gate")
        count = gate.get("violation_count", 0)
        count_s = f" ({count} violation(s))" if count else ""
        generated = data.get("generated", "unknown date")
        entries.append({
            "path": f"scripts/ci-status.json (gate: {name})",
            "title": f"Fix CI gate failure: {name}{count_s} — as of {generated}",
            "kind": "quality-debt",
            "status": "quality-debt",
            "owner": "unowned",
            "validation": "none",
            "lines": 0,
            "_debt_kind": "ci-failure",
            "_gate": name,
            "_violation_count": count,
        })
    return entries


def collect_flag_oversized() -> "list[dict]":
    """Return synthetic entries for FLAG-severity tracked source/test files.

    Avoids importing the full find-oversized-files scanner at module load time
    by running it as a subprocess with --json.  Falls back to [] on error.
    """
    import subprocess
    import json as _json
    try:
        result = subprocess.run(
            [sys.executable, str(REPO / "scripts" / "find-oversized-files.py"), "--json"],
            capture_output=True, text=True, check=True, cwd=REPO,
        )
        items = _json.loads(result.stdout)
    except Exception:
        return []
    entries = []
    for item in items:
        if item.get("severity") != "flag":
            continue
        if item.get("category") not in ("source", "test"):
            continue
        size_s = f"{item['size']} {item['unit']}"
        entries.append({
            "path": item["path"],
            "title": f"Split {item['path'].rsplit('/', 1)[-1]} ({size_s}, FLAG)",
            "kind": "quality-debt",
            "status": "quality-debt",
            "owner": "unowned",
            "validation": "none",
            "lines": 0,
            "_debt_kind": "oversized",
        })
    return entries


def parse_frontmatter(text: str) -> "dict | None":
    if not text.startswith("---\n"):
        return None
    end = text.find("\n---\n", 4)
    if end < 0:
        return None
    fm = {}
    for line in text[4:end].splitlines():
        if ":" not in line:
            continue
        key, _, value = line.partition(":")
        value = value.strip()
        if value.startswith('"') and value.endswith('"'):
            value = value[1:-1]
        fm[key.strip()] = value
    return fm


def collect():
    plans = []
    for root, _, names in os.walk(PLANS):
        for n in names:
            if not n.endswith(".md") or n in ("INDEX.md", "README.md", "progress.md"):
                continue
            path = Path(root) / n
            text = path.read_text()
            fm = parse_frontmatter(text)
            if fm is None:
                continue
            rel = path.relative_to(PLANS).as_posix()
            line_count = text.count("\n") + 1
            plans.append({
                "path": rel,
                "title": fm.get("title", rel),
                "kind": fm.get("kind", "plan"),
                "status": fm.get("status", ""),
                "owner": fm.get("owner", "unowned"),
                "validation": fm.get("validation", "none"),
                "last_reviewed": fm.get("last-reviewed", "?"),
                "lines": line_count,
            })
    return plans


def bucket(plans):
    section = defaultdict(list)

    # Priority 0: quality debt — CI failures first (broken gate blocks all
    # other work), then bug reports (broken behavior), then FLAG oversized files.
    quality_debt = collect_ci_failures() + collect_open_bugs() + collect_flag_oversized()
    section["quality_debt"] = quality_debt

    needs_attention = [p for p in plans if p["status"] in ATTENTION_STATUSES]
    # Order: open incidents/audits, shipped-buggy, then shipped-untested.
    attention_order = {"open": 0, "shipped-buggy": 1, "shipped-untested": 2}
    needs_attention.sort(key=lambda p: (attention_order.get(p["status"], 9), p["path"]))
    section["needs_attention"] = needs_attention

    section["in_flight"] = sorted(
        [p for p in plans if p["status"] == "building"],
        key=lambda p: p["path"],
    )

    section["promote_to_validated"] = sorted(
        [p for p in plans
         if p["status"] == "shipped-untested"
         and not p["validation"].startswith("automated:")],
        key=lambda p: p["path"],
    )

    # Planned, sorted by line count (ascending = smaller = preferred).
    section["planned_small"] = sorted(
        [p for p in plans if p["status"] == "planned"],
        key=lambda p: (p["lines"], p["path"]),
    )

    return section


def first_nonempty(section):
    order = ["quality_debt", "needs_attention", "in_flight", "promote_to_validated", "planned_small"]
    bucket_reason = {
        "quality_debt": "Priority 0: Quality debt (failing CI gates / open bugs / FLAG-severity oversized source) — fix before new features",
        "needs_attention": "Priority 1: Needs attention (shipped-buggy / shipped-untested / open)",
        "in_flight": "Priority 2: Already in flight (status: building) — finish it before starting new work",
        "promote_to_validated": "Priority 3: Promote shipped-untested → shipped-validated by wiring an automated usecase",
        "planned_small": "Priority 4: Smallest planned plan (proxy for well-scoped)",
    }
    for name in order:
        if section[name]:
            return name, bucket_reason[name], section[name][0]
    return None, None, None


def fmt_row(p, show_validation=True):
    bits = [f"  - **{p['title']}** ({p['path']})",
            f"status={p['status']}",
            f"owner={p['owner']}"]
    if show_validation:
        bits.append(f"validation={p['validation']}")
    bits.append(f"lines={p['lines']}")
    return " · ".join(bits)


def render_markdown(section):
    out = []
    out.append("# Next work — recommendation")
    out.append("")

    bucket_name, reason, pick = first_nonempty(section)
    if pick is None:
        out.append("_No candidate plans found in any priority bucket._")
        return "\n".join(out)

    nontrivial = pick["lines"] >= NONTRIVIAL_LINE_THRESHOLD
    out.append(f"**Pick:** [{pick['title']}](plans/{pick['path']})")
    out.append("")
    out.append(f"- Reason: {reason}")
    out.append(f"- Status: `{pick['status']}` · Owner: `{pick['owner']}` · Validation: `{pick['validation']}` · Size: {pick['lines']} lines")
    if nontrivial:
        out.append(f"- ⚠️  Non-trivial ({pick['lines']} ≥ {NONTRIVIAL_LINE_THRESHOLD} lines): STOP and report a one-paragraph plan before implementing.")
    out.append("")
    out.append("---")
    out.append("")

    headings = {
        "quality_debt": "Priority 0 — Quality debt (CI failures / bugs / FLAG oversized)",
        "needs_attention": "Priority 1 — Needs attention",
        "in_flight": "Priority 2 — In flight (building)",
        "promote_to_validated": "Priority 3 — Ready to promote to shipped-validated",
        "planned_small": "Priority 4 — Planned (sorted small → large)",
    }
    for name in ["quality_debt", "needs_attention", "in_flight", "promote_to_validated", "planned_small"]:
        items = section[name]
        out.append(f"## {headings[name]} — {len(items)} candidate(s)")
        out.append("")
        if not items:
            out.append("_(none)_")
            out.append("")
            continue
        # Cap planned list to keep the report scannable.
        cap = 10 if name == "planned_small" else None
        shown = items if cap is None else items[:cap]
        for p in shown:
            out.append(fmt_row(p))
        if cap is not None and len(items) > cap:
            out.append(f"  - … {len(items) - cap} more")
        out.append("")

    out.append("---")
    out.append("")
    out.append("Tie-breakers used: 'quality_debt' bugs ordered most-frequent first; 'needs_attention' ordered open → shipped-buggy → shipped-untested; 'planned_small' ordered ascending by line count as a proxy for scope.")
    out.append("")
    out.append("This script does not check inter-plan dependencies — confirm a planned pick's deps are shipped before starting.")
    return "\n".join(out)


def main():
    plans = collect()
    section = bucket(plans)
    if "--json" in sys.argv[1:]:
        bucket_name, reason, pick = first_nonempty(section)
        print(json.dumps({
            "pick": pick,
            "bucket": bucket_name,
            "reason": reason,
            "buckets": section,
        }, indent=2))
        return
    print(render_markdown(section))


if __name__ == "__main__":
    main()
