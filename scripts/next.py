#!/usr/bin/env python3
"""Single canonical answer to "what should I work on next?".

Wraps `pick-next-work.py` (priority buckets across plans) and additionally
surfaces four drift signals the pick-next-work ranker doesn't capture:

  a. Un-validated planned use-case IDs — IDs in `usecases.md` that no plan
     references via `validation: automated:<ID>` and that
     `scripts/usecase-validate.sh` doesn't already cover.
  b. Open audits and incidents — kind: audit/incident with status: open.
  c. Stale building plans — status: building plans whose progress log
     hasn't been touched in STALE_DAYS days.
  d. Oversized tracked files — files exceeding their per-category size
     threshold (see `scripts/find-oversized-files.py`). Indicates work
     that needs splitting, refactoring, or git-lfs treatment.

Replaces the hand-edited `next.md` pointer (see plans/audits/
markdown-organization-audit.md F5+S2).

Usage:
  python3 scripts/next.py              # human-readable report
  python3 scripts/next.py --json       # machine-readable JSON
"""
import argparse
import datetime as _dt
import importlib.util
import json
import os
import re
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
PLANS = REPO / "plans"
USECASES = REPO / "usecases.md"
USECASES_DIR = REPO / "usecases"
USECASE_VALIDATE_SH = REPO / "scripts" / "usecase-validate.sh"
BUG_REPORTS_DIR = REPO / "terminal_server" / "logs" / "bug_reports"

STALE_DAYS = 14


def _load(name: str, path: Path):
    spec = importlib.util.spec_from_file_location(name, path)
    mod = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(mod)
    return mod


pnw = _load("pick_next_work", REPO / "scripts" / "pick-next-work.py")
gvm = _load("gen_validation_matrix", REPO / "scripts" / "generate-validation-matrix.py")
fof = _load("find_oversized_files", REPO / "scripts" / "find-oversized-files.py")

# Cap the number of oversized-file entries shown in the inline drift report.
# Anything beyond this many is summarized as "(+N more)" with a hint to run
# `find-oversized-files.py` directly for the full list. Keeps `next.py`
# output scannable when the repo accumulates many large files.
OVERSIZED_DISPLAY_LIMIT = 15


def collect_usecase_ids() -> "list[str]":
    """Return ordered list of IDs from usecases/ (preferred) or usecases.md."""
    if USECASES_DIR.is_dir():
        ids: list[str] = []
        for p in sorted(USECASES_DIR.glob("*.md")):
            if p.name == "INDEX.md":
                continue
            for raw in p.read_text().splitlines():
                cells = gvm.extract_table_cells(raw)
                if not cells:
                    continue
                first = cells[0]
                if gvm.ID_RE.match(first) and first not in ids:
                    ids.append(first)
        return ids
    return list(gvm.parse_usecases().keys())


def unvalidated_ids() -> "list[str]":
    ids = collect_usecase_ids()
    plan_by_id, _ = gvm.collect_plan_validations()
    sh_ids = set(gvm.parse_usecase_validate_sh())
    automated = set(plan_by_id) | sh_ids
    return [uid for uid in ids if uid not in automated]


def progress_sink_for(plan_path_rel: str) -> "Path | None":
    plan_path = PLANS / plan_path_rel
    sibling = plan_path.parent / "progress.md"
    if sibling.exists():
        return sibling
    return None


def days_since(date_str: str, today: _dt.date) -> "int | None":
    try:
        d = _dt.date.fromisoformat(date_str)
    except (ValueError, TypeError):
        return None
    return (today - d).days


def stale_building(plans, today: _dt.date) -> "list[dict]":
    out = []
    for p in plans:
        if p["status"] != "building":
            continue
        sink = progress_sink_for(p["path"])
        if sink is not None:
            mtime = _dt.datetime.fromtimestamp(sink.stat().st_mtime).date()
            age = (today - mtime).days
            source = f"progress.md (mtime {mtime.isoformat()})"
        else:
            age = days_since(p["last_reviewed"], today)
            source = f"last-reviewed: {p['last_reviewed']}"
            if age is None:
                continue
        if age >= STALE_DAYS:
            out.append({
                "title": p["title"],
                "path": p["path"],
                "age_days": age,
                "signal": source,
            })
    out.sort(key=lambda x: -x["age_days"])
    return out


def collect_bug_summary() -> "list[dict]":
    """Return distinct bug descriptions with counts from the bug report log.

    Groups by normalized description (whitespace collapsed, 80-char cap).
    Returns [] when the directory is absent (CI without a local log tree).
    Each entry: {description, count, most_recent_date}.
    """
    if not BUG_REPORTS_DIR.is_dir():
        return []
    from collections import Counter
    counts: Counter[str] = Counter()
    most_recent: dict[str, str] = {}
    for day_dir in sorted(BUG_REPORTS_DIR.iterdir()):
        if not day_dir.is_dir():
            continue
        date_str = day_dir.name
        for report_file in sorted(day_dir.glob("*.json")):
            try:
                import json as _json
                d = _json.loads(report_file.read_bytes())
            except Exception:
                continue
            raw = (d.get("summary") or {}).get("description") or ""
            desc = " ".join(raw.split())[:80] or "(no description)"
            counts[desc] += 1
            if desc not in most_recent or date_str > most_recent[desc]:
                most_recent[desc] = date_str
    return [
        {"description": desc, "count": cnt, "most_recent_date": most_recent.get(desc, "?")}
        for desc, cnt in counts.most_common()
    ]


def open_audits_and_incidents(plans) -> "list[dict]":
    return sorted(
        [
            {"title": p["title"], "path": p["path"], "kind": p["kind"]}
            for p in plans
            if p["kind"] in {"audit", "incident"} and p["status"] == "open"
        ],
        key=lambda x: x["path"],
    )


def render(plans, section, signals, today: _dt.date) -> str:
    out: list[str] = []
    out.append(pnw.render_markdown(section))
    out.append("")
    out.append("---")
    out.append("")
    out.append("# Drift signals")
    out.append("")
    out.append(f"_Snapshot at {today.isoformat()}._")
    out.append("")

    out.append("## Un-validated planned use-case IDs"
               f" — {len(signals['unvalidated_ids'])}")
    out.append("")
    if signals["unvalidated_ids"]:
        out.append(", ".join(f"`{i}`" for i in signals["unvalidated_ids"]))
    else:
        out.append("_(none — every defined use-case ID is referenced by a plan or "
                   "by `scripts/usecase-validate.sh`.)_")
    out.append("")

    out.append("## Open audits and incidents"
               f" — {len(signals['open_items'])}")
    out.append("")
    if signals["open_items"]:
        for item in signals["open_items"]:
            out.append(f"- **{item['title']}** "
                       f"(`{item['kind']}`) — plans/{item['path']}")
    else:
        out.append("_(none open.)_")
    out.append("")

    out.append(f"## Stale `building` plans (≥ {STALE_DAYS} days)"
               f" — {len(signals['stale'])}")
    out.append("")
    if signals["stale"]:
        for item in signals["stale"]:
            out.append(f"- **{item['title']}** — plans/{item['path']} "
                       f"({item['age_days']}d via {item['signal']})")
    else:
        out.append("_(none stale.)_")
    out.append("")

    oversized = signals["oversized"]
    flag_count = sum(1 for r in oversized if r["severity"] == "flag")
    out.append(f"## Oversized tracked files — {len(oversized)} "
               f"({flag_count} at flag severity)")
    out.append("")
    if oversized:
        shown = oversized[:OVERSIZED_DISPLAY_LIMIT]
        for r in shown:
            tag = "[FLAG]" if r["severity"] == "flag" else "[warn]"
            size_s = fof.fmt_size(r["unit"], r["size"])
            out.append(f"- {tag} `{r['path']}` "
                       f"({r['category']}, {size_s}) — {r['suggestion']}")
        if len(oversized) > OVERSIZED_DISPLAY_LIMIT:
            remaining = len(oversized) - OVERSIZED_DISPLAY_LIMIT
            out.append(f"- _(+{remaining} more — run "
                       f"`python3 scripts/find-oversized-files.py` for the "
                       f"full list.)_")
    else:
        out.append("_(no tracked file exceeds its category threshold.)_")
    out.append("")

    bugs = signals.get("bugs", [])
    total_bug_count = sum(b["count"] for b in bugs)
    out.append(f"## Bug reports — {len(bugs)} distinct description(s), "
               f"{total_bug_count} total")
    out.append("")
    if bugs:
        for b in bugs[:10]:
            cnt_s = f"×{b['count']}" if b["count"] > 1 else "×1"
            out.append(f"- {cnt_s} `{b['description']}` "
                       f"(last seen {b['most_recent_date']})")
        if len(bugs) > 10:
            out.append(f"- _(+{len(bugs) - 10} more)_")
    else:
        out.append("_(no bug reports found in terminal_server/logs/bug_reports/.)_")
    out.append("")

    return "\n".join(out)


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--json", action="store_true")
    args = ap.parse_args()

    today = _dt.date.today()

    plans = pnw.collect()
    section = pnw.bucket(plans)

    signals = {
        "unvalidated_ids": unvalidated_ids(),
        "open_items": open_audits_and_incidents(plans),
        "stale": stale_building(plans, today),
        "oversized": fof.scan(),
        "bugs": collect_bug_summary(),
    }

    if args.json:
        bucket_name, reason, pick = pnw.first_nonempty(section)
        print(json.dumps({
            "pick": pick,
            "bucket": bucket_name,
            "reason": reason,
            "buckets": section,
            "signals": signals,
        }, indent=2, default=str))
        return 0

    print(render(plans, section, signals, today))
    return 0


if __name__ == "__main__":
    sys.exit(main())
