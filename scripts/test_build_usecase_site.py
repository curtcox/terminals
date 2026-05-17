#!/usr/bin/env python3
"""Focused tests for scripts/build-usecase-site.py."""

from __future__ import annotations

import importlib.util
import json
import sys
import tempfile
import unittest
from pathlib import Path


SCRIPT = Path(__file__).resolve().parent / "build-usecase-site.py"


def load_generator():
    spec = importlib.util.spec_from_file_location("build_usecase_site", SCRIPT)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"could not load {SCRIPT}")
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


class BuildUsecaseSiteTest(unittest.TestCase):
    def setUp(self) -> None:
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name)
        self.module = load_generator()
        self.module.REPO = self.root
        self.module.USECASES_DIR = self.root / "usecases"
        self.module.VALIDATOR = self.root / "scripts" / "usecase-validate.sh"
        self.module.OUT = self.root / "docs" / "usecases-site"
        self.module.EMBED_OUT = self.root / "terminal_server" / "internal" / "admin" / "usecases_site_static"
        self.module.USECASE_RESULTS = self.root / "artifacts" / "usecases"
        self.module.USECASE_VALIDATION = self.root / "artifacts" / "usecase-validation"
        self.module.RESULTS = {}
        self.module.USECASES_DIR.mkdir(parents=True)
        self.module.VALIDATOR.parent.mkdir(parents=True)
        self.module.VALIDATOR.write_text("all_ids=(C1 T1)\n")
        (self.module.USECASES_DIR / "communication.md").write_text(
            """---
title: "Communication"
family: "C"
---

| ID | Actor | Action | Outcome |
|---|---|---|---|
| C1 | parent | call the kitchen | talk hands-free |
| C2 | parent | announce dinner | everyone hears it |
"""
        )

    def tearDown(self) -> None:
        self.tmp.cleanup()

    def write_result(self, usecase_id: str, timestamp: str, pass_: bool, failure: str = "") -> Path:
        path = self.module.USECASE_RESULTS / usecase_id / "result.json"
        path.parent.mkdir(parents=True)
        data = {
            "run_id": f"run-{timestamp}",
            "usecase_id": usecase_id,
            "scenario_name": f"scenario {usecase_id}",
            "timestamp_end": timestamp,
            "pass": pass_,
        }
        if failure:
            data["failing_assertions"] = [failure]
        path.write_text(json.dumps(data))
        return path

    def test_result_feed_renders_interaction_trace(self) -> None:
        path = self.write_result("C1", "2026-05-17T12:00:00Z", True)
        data = json.loads(path.read_text())
        data["interaction_trace"] = [
            {"kind": "voice", "summary": 'Say "call the kitchen".', "terminal": "hall"},
            {"kind": "command", "summary": 'Tap "End call".', "terminal": "hall"},
        ]
        path.write_text(json.dumps(data))

        self.module.RESULTS = self.module.latest_results(include_results=True)
        usecases = self.module.parse_usecases()
        c1_page = self.module.render_usecase(next(usecase for usecase in usecases if usecase.id == "C1"))

        self.assertIn("<ol>", c1_page)
        self.assertIn("Say &quot;call the kitchen&quot;.", c1_page)
        self.assertIn("Tap &quot;End call&quot;.", c1_page)
        self.assertNotIn("Interaction traces are not captured yet", c1_page)

    def test_result_feed_marks_failed_usecase_as_defect(self) -> None:
        self.write_result("C1", "2026-05-17T12:00:00Z", False, "C1-route-stream")

        self.module.RESULTS = self.module.latest_results(include_results=True)
        usecases = self.module.parse_usecases()
        index = self.module.render_index(usecases)
        c1_page = self.module.render_usecase(next(usecase for usecase in usecases if usecase.id == "C1"))

        self.assertIn('<span class="badge defect">DEFECT</span>', index)
        self.assertIn("Failed on 2026-05-17 12:00 UTC: C1-route-stream.", c1_page)
        self.assertIn("Latest validation failed: C1-route-stream.", c1_page)
        self.assertIn("Latest result manifest", c1_page)

    def test_latest_result_prefers_newest_manifest(self) -> None:
        self.write_result("C1", "2026-05-17T11:00:00Z", False, "old-failure")
        newer = self.module.USECASE_VALIDATION / "newer" / "manifest.json"
        newer.parent.mkdir(parents=True)
        newer.write_text(
            json.dumps(
                {
                    "run_id": "newer",
                    "usecase_id": "C1",
                    "scenario_name": "newer scenario",
                    "timestamp_end": "2026-05-17T12:30:00Z",
                    "pass": True,
                }
            )
        )

        results = self.module.latest_results(include_results=True, include_validation_runs=True)

        self.assertTrue(results["C1"].pass_)
        self.assertEqual(results["C1"].source, "artifacts/usecase-validation/newer/manifest.json")

    def test_old_passing_result_is_marked_stale(self) -> None:
        self.write_result("C1", "2024-01-01T00:00:00Z", True)

        self.module.RESULTS = self.module.latest_results(include_results=True)
        usecases = self.module.parse_usecases()
        index = self.module.render_index(usecases)
        c1_page = self.module.render_usecase(next(usecase for usecase in usecases if usecase.id == "C1"))

        self.assertIn('<span class="badge stale">STALE</span>', index)
        self.assertIn("result is older than 30 days", c1_page)


if __name__ == "__main__":
    unittest.main()
