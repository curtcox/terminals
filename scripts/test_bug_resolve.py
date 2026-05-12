#!/usr/bin/env python3
"""Unit tests for scripts/bug-resolve.py and scripts/check-resolved-bugs.py."""
import importlib.util
import io
import json
import os
import subprocess
import sys
import tempfile
import unittest
from contextlib import redirect_stderr, redirect_stdout
from pathlib import Path

HERE = Path(__file__).resolve().parent


def _load(name: str, filename: str):
    spec = importlib.util.spec_from_file_location(name, HERE / filename)
    mod = importlib.util.module_from_spec(spec)
    assert spec.loader is not None
    spec.loader.exec_module(mod)
    return mod


bug_resolve = _load("bug_resolve", "bug-resolve.py")
check_resolved = _load("check_resolved_bugs", "check-resolved-bugs.py")


def _write_report(reports_dir: Path, report_id: str, description: str, date: str = "2026-05-12") -> Path:
    day = reports_dir / date
    day.mkdir(parents=True, exist_ok=True)
    payload = {
        "summary": {
            "report_id": report_id,
            "description": description,
        }
    }
    path = day / f"{report_id}.json"
    path.write_text(json.dumps(payload))
    return path


def _run(args, expect_rc=0):
    out = io.StringIO()
    err = io.StringIO()
    with redirect_stdout(out), redirect_stderr(err):
        rc = bug_resolve.main(args)
    if rc != expect_rc:
        raise AssertionError(f"rc={rc} (want {expect_rc}); stderr={err.getvalue()!r}")
    return out.getvalue(), err.getvalue()


class BugResolveTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        self.reports = self.root / "reports"
        self.resolved = self.root / "resolved"
        self.reports.mkdir()
        self.resolved.mkdir()

    def tearDown(self):
        self.tmp.cleanup()

    def _base_args(self, report_id):
        return [
            "--report-id", report_id,
            "--token-word", "photo",
            "--fix-commit", "abc1234",
            "--regression-test", "pkg/foo_test.go::TestThing",
            "--root-cause", "blah",
            "--reports-dir", str(self.reports),
            "--resolved-dir", str(self.resolved),
            "--today", "2026-05-12",
        ]

    def test_writes_record_from_report(self):
        report_id = "bug-20260512t100000.000-aaaa1111"
        _write_report(self.reports, report_id, "  Scan LAN   button reports an    error.\n")
        _run(self._base_args(report_id))
        out_path = self.resolved / f"{report_id}.json"
        self.assertTrue(out_path.is_file())
        data = json.loads(out_path.read_text())
        self.assertEqual(data["report_id"], report_id)
        # Whitespace collapsed; matches the readers' normalization.
        self.assertEqual(data["description"], "Scan LAN button reports an error.")
        self.assertEqual(data["bug_token_word"], "photo")
        self.assertEqual(data["fix_commits"], ["abc1234"])
        self.assertEqual(data["regression_tests"], ["pkg/foo_test.go::TestThing"])
        self.assertEqual(data["resolved_at"], "2026-05-12")

    def test_missing_report_errors(self):
        _, err = _run(self._base_args("bug-does-not-exist"), expect_rc=2)
        self.assertIn("not found", err)

    def test_description_must_exist(self):
        report_id = "bug-20260512t100000.000-empty"
        _write_report(self.reports, report_id, "")
        _, err = _run(self._base_args(report_id), expect_rc=2)
        self.assertIn("no summary.description", err)

    def test_refuses_overwrite_without_force(self):
        report_id = "bug-20260512t100000.000-bbbb2222"
        _write_report(self.reports, report_id, "the bug")
        _run(self._base_args(report_id))
        _, err = _run(self._base_args(report_id), expect_rc=2)
        self.assertIn("already exists", err)
        # --force succeeds.
        _run(self._base_args(report_id) + ["--force"])

    def test_normalizes_description_to_80_chars(self):
        report_id = "bug-20260512t100000.000-cccc3333"
        long_desc = "x" * 200
        _write_report(self.reports, report_id, long_desc)
        _run(self._base_args(report_id))
        data = json.loads((self.resolved / f"{report_id}.json").read_text())
        self.assertEqual(len(data["description"]), 80)


class CheckResolvedBugsTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        self.resolved = self.root / "resolved"
        self.resolved.mkdir()
        # Make a tiny git repo so _commit_exists() can find a real SHA.
        subprocess.run(["git", "-C", str(self.root), "init", "-q"], check=True)
        subprocess.run(["git", "-C", str(self.root), "config", "user.email", "t@t"], check=True)
        subprocess.run(["git", "-C", str(self.root), "config", "user.name", "t"], check=True)
        subprocess.run(["git", "-C", str(self.root), "config", "commit.gpgsign", "false"], check=True)
        (self.root / "f.txt").write_text("hi")
        subprocess.run(["git", "-C", str(self.root), "add", "f.txt"], check=True)
        subprocess.run(["git", "-C", str(self.root), "commit", "-q", "-m", "init"], check=True)
        self.real_sha = subprocess.run(
            ["git", "-C", str(self.root), "rev-parse", "HEAD"],
            check=True, capture_output=True, text=True,
        ).stdout.strip()
        # Point the check script at our fake repo.
        self._orig_resolved = check_resolved.RESOLVED_DIR
        self._orig_repo = check_resolved.REPO
        check_resolved.RESOLVED_DIR = self.resolved
        check_resolved.REPO = self.root

    def tearDown(self):
        check_resolved.RESOLVED_DIR = self._orig_resolved
        check_resolved.REPO = self._orig_repo
        self.tmp.cleanup()

    def _write(self, name, **overrides):
        record = {
            "report_id": name,
            "bug_token_word": "photo",
            "description": "the bug",
            "resolved_at": "2026-05-12",
            "fix_commits": [self.real_sha],
            "regression_tests": ["pkg/foo_test.go::TestThing"],
            "root_cause": "blah",
            "notes": "",
        }
        record.update(overrides)
        (self.resolved / f"{name}.json").write_text(json.dumps(record))

    def _run_check(self):
        out, err = io.StringIO(), io.StringIO()
        with redirect_stdout(out), redirect_stderr(err):
            rc = check_resolved.main()
        return rc, out.getvalue(), err.getvalue()

    def test_valid_record_passes(self):
        self._write("bug-aaa")
        rc, out, _ = self._run_check()
        self.assertEqual(rc, 0)
        self.assertIn("OK: 1", out)

    def test_missing_field_fails(self):
        self._write("bug-bbb", root_cause=None)
        # Remove the field entirely so the type check fires.
        path = self.resolved / "bug-bbb.json"
        data = json.loads(path.read_text())
        del data["root_cause"]
        path.write_text(json.dumps(data))
        rc, _, err = self._run_check()
        self.assertEqual(rc, 1)
        self.assertIn("missing field 'root_cause'", err)

    def test_bad_sha_fails(self):
        self._write("bug-ccc", fix_commits=["deadbeefdeadbeef"])
        rc, _, err = self._run_check()
        self.assertEqual(rc, 1)
        self.assertIn("not found in git history", err)

    def test_report_id_must_match_filename(self):
        self._write("bug-ddd", report_id="bug-other")
        rc, _, err = self._run_check()
        self.assertEqual(rc, 1)
        self.assertIn("does not match filename", err)

    def test_empty_fix_commits_fails(self):
        self._write("bug-eee", fix_commits=[])
        rc, _, err = self._run_check()
        self.assertEqual(rc, 1)
        self.assertIn("must not be empty", err)


if __name__ == "__main__":
    unittest.main()
