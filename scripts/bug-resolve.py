#!/usr/bin/env python3
"""Record a bug as resolved.

Writes one JSON file per resolved bug to
`terminal_server/bug_reports/resolved/<report_id>.json`. Invoked by the
`bugword` skill after a fix lands and the regression test(s) pass.

The written `description` field is the dedupe key used by
`scripts/next.py` and `scripts/pick-next-work.py` to exclude already-fixed
bugs from the work queue. It is sourced from the original report's
`summary.description`, normalized the same way the readers normalize it
(whitespace collapsed, 80-char cap).

Usage:
  scripts/bug-resolve.py \\
      --report-id bug-20260418t222807.151-44d18db8 \\
      --token-word photo \\
      --fix-commit abc1234 [--fix-commit ...] \\
      --regression-test "terminal_client/test/foo_test.dart::scanLan_noOpOnWeb" \\
                       [--regression-test ...] \\
      --root-cause "InternetAddress.anyIPv4 is unavailable on Flutter Web." \\
      [--notes "extra detail"] \\
      [--force]
"""
import argparse
import datetime as _dt
import json
import subprocess
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
REPORTS_DIR = REPO / "terminal_server" / "logs" / "bug_reports"
RESOLVED_DIR = REPO / "terminal_server" / "bug_reports" / "resolved"


def _normalize_description(raw: str) -> str:
    return " ".join((raw or "").split())[:80] or "(no description)"


def _find_report(report_id: str) -> "Path | None":
    if not REPORTS_DIR.is_dir():
        return None
    for day_dir in REPORTS_DIR.iterdir():
        if not day_dir.is_dir():
            continue
        candidate = day_dir / f"{report_id}.json"
        if candidate.is_file():
            return candidate
    return None


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


def main(argv: "list[str] | None" = None) -> int:
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--report-id", required=True)
    p.add_argument("--token-word", required=True)
    p.add_argument("--fix-commit", action="append", default=[], required=True,
                   help="Git SHA of the fix. Repeat for multiple commits.")
    p.add_argument("--regression-test", action="append", default=[], required=True,
                   help="Test that verifies the fix. Repeat for multiple tests.")
    p.add_argument("--root-cause", required=True, help="One-line root-cause summary.")
    p.add_argument("--notes", default="", help="Optional free-form notes.")
    p.add_argument("--force", action="store_true",
                   help="Overwrite an existing resolved-bug record.")
    p.add_argument("--reports-dir", default=None,
                   help="Override the bug-reports directory (testing).")
    p.add_argument("--resolved-dir", default=None,
                   help="Override the resolved directory (testing).")
    p.add_argument("--today", default=None,
                   help="Override today's date as YYYY-MM-DD (testing).")
    args = p.parse_args(argv)

    reports_dir = Path(args.reports_dir).resolve() if args.reports_dir else REPORTS_DIR
    resolved_dir = Path(args.resolved_dir).resolve() if args.resolved_dir else RESOLVED_DIR

    # Locate the original report so we can extract the description.
    report_path: "Path | None" = None
    if reports_dir.is_dir():
        for day_dir in reports_dir.iterdir():
            if not day_dir.is_dir():
                continue
            candidate = day_dir / f"{args.report_id}.json"
            if candidate.is_file():
                report_path = candidate
                break
    if report_path is None:
        print(f"error: report {args.report_id} not found under {reports_dir}", file=sys.stderr)
        return 2

    try:
        report = json.loads(report_path.read_bytes())
    except Exception as exc:
        print(f"error: failed to parse {report_path}: {exc}", file=sys.stderr)
        return 2

    raw_desc = (report.get("summary") or {}).get("description") or ""
    description = _normalize_description(raw_desc)
    if description == "(no description)":
        print(f"error: report {args.report_id} has no summary.description", file=sys.stderr)
        return 2

    # Validate every commit SHA exists. Skipped when reports-dir is overridden
    # (test mode runs outside a populated git history).
    if args.reports_dir is None:
        bad = [sha for sha in args.fix_commit if not _commit_exists(sha)]
        if bad:
            print(f"error: unknown git commit(s): {', '.join(bad)}", file=sys.stderr)
            return 2

    today = args.today or _dt.datetime.utcnow().strftime("%Y-%m-%d")

    out_path = resolved_dir / f"{args.report_id}.json"
    if out_path.exists() and not args.force:
        print(f"error: {out_path} already exists; pass --force to overwrite", file=sys.stderr)
        return 2

    record = {
        "report_id": args.report_id,
        "bug_token_word": args.token_word,
        "description": description,
        "resolved_at": today,
        "fix_commits": list(args.fix_commit),
        "regression_tests": list(args.regression_test),
        "root_cause": args.root_cause,
        "notes": args.notes,
    }

    resolved_dir.mkdir(parents=True, exist_ok=True)
    out_path.write_text(json.dumps(record, indent=2) + "\n")
    print(f"wrote {out_path.relative_to(REPO) if out_path.is_relative_to(REPO) else out_path}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
