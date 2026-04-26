#!/usr/bin/env python3
"""Pick the next plan to work on, by the priority order in the agent prompt.

Walks plans/ and parses YAML frontmatter, then groups candidates into the four
priority buckets used by the "pick the next work" prompt:

  1. Needs attention   — shipped-buggy, shipped-untested, open audits/incidents
  2. In flight         — status: building
  3. Promote to validated — shipped-untested with no automated validation wired
  4. Planned, small    — status: planned, sorted ascending by file line count
                         (proxy for "small, well-scoped")

Outputs a markdown report. The first non-empty bucket's first entry is the
recommended pick. Larger plans are flagged as non-trivial so the agent knows
to STOP and report a plan before implementing.

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

ATTENTION_STATUSES = {"shipped-buggy", "shipped-untested", "open"}
NONTRIVIAL_LINE_THRESHOLD = 200  # plans larger than this need a plan-first pass


def parse_frontmatter(text: str) -> dict | None:
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
            if not n.endswith(".md") or n in ("INDEX.md", "README.md"):
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
    order = ["needs_attention", "in_flight", "promote_to_validated", "planned_small"]
    bucket_reason = {
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
        "needs_attention": "Priority 1 — Needs attention",
        "in_flight": "Priority 2 — In flight (building)",
        "promote_to_validated": "Priority 3 — Ready to promote to shipped-validated",
        "planned_small": "Priority 4 — Planned (sorted small → large)",
    }
    for name in ["needs_attention", "in_flight", "promote_to_validated", "planned_small"]:
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
    out.append("Tie-breakers used: 'needs_attention' ordered open → shipped-buggy → shipped-untested; 'planned_small' ordered ascending by line count as a proxy for scope.")
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
