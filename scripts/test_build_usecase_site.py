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
        self.module.BUG_REPORTS = self.root / "terminal_server" / "logs" / "bug_reports"
        self.module.RESOLVED_BUGS = self.root / "terminal_server" / "bug_reports" / "resolved"
        self.module.UI_AUDIT = (
            self.root / "terminal_server" / "internal" / "scenario" / "audit" / "verify_terminal_ui_usecases.sh"
        )
        self.module.UI_INSPECT_SKILL = self.root / ".claude" / "skills" / "ui-inspect" / "SKILL.md"
        self.module.RESULTS = {}
        self.module.BUG_REPORTS_BY_USECASE = {}
        self.module.USECASES_DIR.mkdir(parents=True)
        self.module.VALIDATOR.parent.mkdir(parents=True)
        self.module.VALIDATOR.write_text(
            """metadata() {
  local id="$1"
  case "${id}" in
    C1) echo "C1|Transport|generated intercom route test; harness evidence" ;;
    T1) echo "T1|Simulation|timer harness evidence" ;;
  esac
}
all_ids=(C1 T1)
"""
        )
        self.module.UI_AUDIT.parent.mkdir(parents=True)
        self.module.UI_AUDIT.write_text("#!/usr/bin/env bash\n")
        self.module.UI_INSPECT_SKILL.parent.mkdir(parents=True)
        self.module.UI_INSPECT_SKILL.write_text("# ui-inspect\n")
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

    def write_bug(
        self,
        report_id: str,
        description: str,
        tags: list[str],
        date: str = "2026-05-17",
    ) -> Path:
        path = self.module.BUG_REPORTS / date / f"{report_id}.json"
        path.parent.mkdir(parents=True)
        path.write_text(
            json.dumps(
                {
                    "summary": {
                        "report_id": report_id,
                        "description": description,
                        "tags": tags,
                    }
                }
            )
        )
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

    def test_result_feed_renders_media_assets(self) -> None:
        path = self.write_result("C1", "2026-05-17T12:00:00Z", True)
        data = json.loads(path.read_text())
        data["media"] = {
            "frames": [
                {"step_id": "connected", "path": "frames/connected.png"},
            ],
            "videos": [
                {"label": "Intercom flow", "path": "video/intercom.mp4"},
            ],
            "audio": [
                {"label": "Caller audio", "path": "audio/caller.wav"},
            ],
        }
        path.write_text(json.dumps(data))

        self.module.RESULTS = self.module.latest_results(include_results=True)
        usecases = self.module.parse_usecases()
        c1_page = self.module.render_usecase(next(usecase for usecase in usecases if usecase.id == "C1"))

        self.assertIn("Rendered from server primitives", c1_page)
        self.assertIn(
            '<a href="../../terminal_server/internal/scenario/audit/verify_terminal_ui_usecases.sh">manual UI audit</a>',
            c1_page,
        )
        self.assertIn('<video controls autoplay muted loop playsinline src="../../artifacts/usecases/C1/video/intercom.mp4">', c1_page)
        self.assertIn('<div class="frame-scrubber"', c1_page)
        self.assertIn('<input class="frame-range" type="range" min="0" max="0" value="0"', c1_page)
        self.assertIn('<img class="frame-preview-image" src="../../artifacts/usecases/C1/frames/connected.png"', c1_page)
        self.assertIn('<img src="../../artifacts/usecases/C1/frames/connected.png"', c1_page)
        self.assertIn('<audio controls src="../../artifacts/usecases/C1/audio/caller.wav">', c1_page)
        self.assertNotIn("Rendered server-primitive screenshots are not captured yet", c1_page)
        self.assertNotIn("Audio artifacts are not captured yet", c1_page)

    def test_frame_strip_is_capped_but_scrubber_keeps_all_frames(self) -> None:
        path = self.write_result("C1", "2026-05-17T12:00:00Z", True)
        data = json.loads(path.read_text())
        data["media"] = {
            "frames": [
                {"step_id": f"step-{index:02d}", "path": f"frames/step-{index:02d}.png"}
                for index in range(1, 31)
            ],
        }
        path.write_text(json.dumps(data))

        self.module.RESULTS = self.module.latest_results(include_results=True)
        usecases = self.module.parse_usecases()
        c1_page = self.module.render_usecase(next(usecase for usecase in usecases if usecase.id == "C1"))

        self.assertIn("&quot;label&quot;: &quot;step-30&quot;", c1_page)
        self.assertIn('max="29"', c1_page)
        self.assertIn("6 additional frames are available through the scrubber and raw result manifest.", c1_page)
        self.assertEqual(c1_page.count('class="frame-link"'), 24)
        self.assertIn("frames/step-24.png", c1_page)
        self.assertNotIn("frames/step-25.png\"><img", c1_page)

    def test_result_feed_marks_failed_usecase_as_defect(self) -> None:
        path = self.write_result("C1", "2026-05-17T12:00:00Z", False, "C1-route-stream")
        data = json.loads(path.read_text())
        data["media"] = {
            "frames": [
                {"step_id": "C1-route-stream", "path": "frames/route-stream.png"},
            ],
        }
        path.write_text(json.dumps(data))

        self.module.RESULTS = self.module.latest_results(include_results=True)
        usecases = self.module.parse_usecases()
        index = self.module.render_index(usecases)
        c1_page = self.module.render_usecase(next(usecase for usecase in usecases if usecase.id == "C1"))

        self.assertIn('<span class="badge defect">DEFECT</span>', index)
        self.assertIn("Failed on 2026-05-17 12:00 UTC: C1-route-stream.", c1_page)
        self.assertIn("Latest validation failed:", c1_page)
        self.assertIn("<li>C1-route-stream", c1_page)
        self.assertIn('<a href="../../artifacts/usecases/C1/frames/route-stream.png">failure frame</a>', c1_page)
        self.assertIn("Latest result manifest", c1_page)

    def test_usecase_page_links_client_ui_inspection_evidence(self) -> None:
        usecases = self.module.parse_usecases()
        c1_page = self.module.render_usecase(next(usecase for usecase in usecases if usecase.id == "C1"))

        self.assertIn(
            "Primary validation evidence: Transport: generated intercom route test; harness evidence",
            c1_page,
        )
        self.assertIn(
            '<a href="../../.claude/skills/ui-inspect/SKILL.md">Client UI inspection workflow</a>',
            c1_page,
        )
        self.assertIn("<code>make usecase-wiring-audit</code>", c1_page)

    def test_index_renders_status_overview_counts(self) -> None:
        self.write_result("C1", "2026-05-17T12:00:00Z", False, "C1-route-stream")

        self.module.RESULTS = self.module.latest_results(include_results=True)
        usecases = self.module.parse_usecases()
        index = self.module.render_index(usecases)

        self.assertIn('<section class="status-overview" aria-label="Use case status summary">', index)
        self.assertIn('<li><span class="badge defect">DEFECT</span><strong>1</strong></li>', index)
        self.assertIn('<li><span class="badge untested">UNTESTED</span><strong>1</strong></li>', index)
        self.assertIn('<li><span class="badge passing">PASSING</span><strong>0</strong></li>', index)

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

    def test_open_bug_tag_marks_usecase_as_defect(self) -> None:
        self.write_result("C1", "2026-05-17T12:00:00Z", True)
        self.write_bug("bug-c1", "Intercom route fails", ["usecase:C1", "bug_word:route"])

        self.module.RESULTS = self.module.latest_results(include_results=True)
        self.module.BUG_REPORTS_BY_USECASE = self.module.open_bug_reports(include_bugs=True)
        usecases = self.module.parse_usecases()
        index = self.module.render_index(usecases)
        c1_page = self.module.render_usecase(next(usecase for usecase in usecases if usecase.id == "C1"))

        self.assertIn('<span class="badge defect">DEFECT</span>', index)
        self.assertIn("1 open defect tagged for this use case.", c1_page)
        self.assertIn("bug-c1: Intercom route fails", c1_page)
        self.assertIn("terminal_server/logs/bug_reports/2026-05-17/bug-c1.json", c1_page)

    def test_resolved_bug_description_is_excluded_from_defect_feed(self) -> None:
        self.write_bug("bug-c1", "  Intercom   route fails  ", ["usecase:C1"])
        self.module.RESOLVED_BUGS.mkdir(parents=True)
        (self.module.RESOLVED_BUGS / "bug-c1.json").write_text(
            json.dumps({"report_id": "bug-c1", "description": "Intercom route fails"})
        )

        self.module.BUG_REPORTS_BY_USECASE = self.module.open_bug_reports(include_bugs=True)

        self.assertNotIn("C1", self.module.BUG_REPORTS_BY_USECASE)


if __name__ == "__main__":
    unittest.main()
