#!/usr/bin/env python3
"""Generate docs/usecases-site from usecases/*.md.

Milestone 1 intentionally uses only the use-case catalog and automation
wiring. Later milestones can join result.json media without changing the
public URLs emitted here.
"""

from __future__ import annotations

import argparse
import html
import json
import re
import shutil
import sys
from collections import OrderedDict
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
USECASES_DIR = REPO / "usecases"
VALIDATOR = REPO / "scripts" / "usecase-validate.sh"
OUT = REPO / "docs" / "usecases-site"
EMBED_OUT = REPO / "terminal_server" / "internal" / "admin" / "usecases_site_static"
USECASE_RESULTS = REPO / "artifacts" / "usecases"
USECASE_VALIDATION = REPO / "artifacts" / "usecase-validation"
BUG_REPORTS = REPO / "terminal_server" / "logs" / "bug_reports"
RESOLVED_BUGS = REPO / "terminal_server" / "bug_reports" / "resolved"
UI_AUDIT = REPO / "terminal_server" / "internal" / "scenario" / "audit" / "verify_terminal_ui_usecases.sh"
STALE_RESULT_DAYS = 30

ID_RE = re.compile(r"^[A-Z]+\d+$")
FRONTMATTER_RE = re.compile(r"^---\n(.*?)\n---\n", re.DOTALL)
ALL_IDS_RE = re.compile(r"^all_ids=\(([^)]*)\)\s*$")


@dataclass(frozen=True)
class UseCase:
    id: str
    family: str
    family_title: str
    source: str
    actor: str
    action: str
    outcome: str
    automated: bool

    @property
    def title(self) -> str:
        return self.action[:1].upper() + self.action[1:]


@dataclass(frozen=True)
class MediaAsset:
    label: str
    path: str
    kind: str


@dataclass(frozen=True)
class Result:
    usecase_id: str
    run_id: str
    scenario_name: str
    timestamp_end: str
    pass_: bool
    failing_assertions: tuple[str, ...]
    interaction_trace: tuple[str, ...]
    frames: tuple[MediaAsset, ...]
    videos: tuple[MediaAsset, ...]
    audio: tuple[MediaAsset, ...]
    source: str

    @property
    def first_failure(self) -> str:
        if self.failing_assertions:
            return self.failing_assertions[0]
        return "validation failed"


@dataclass(frozen=True)
class BugReport:
    report_id: str
    description: str
    tags: tuple[str, ...]
    source: str

    @property
    def label(self) -> str:
        if self.description:
            return f"{self.report_id}: {self.description}"
        return self.report_id


def parse_frontmatter(text: str) -> dict[str, str]:
    match = FRONTMATTER_RE.match(text)
    if not match:
        return {}
    out: dict[str, str] = {}
    for raw in match.group(1).splitlines():
        key, sep, value = raw.partition(":")
        if not sep:
            continue
        value = value.strip().strip('"')
        out[key.strip()] = value
    return out


def table_cells(line: str) -> list[str] | None:
    stripped = line.strip()
    if not stripped.startswith("|") or not stripped.endswith("|"):
        return None
    return [clean_markdown(cell.strip()) for cell in stripped.strip("|").split("|")]


def clean_markdown(value: str) -> str:
    return value.replace("**", "").replace("`", "")


def automated_ids() -> set[str]:
    if not VALIDATOR.exists():
        return set()
    for raw in VALIDATOR.read_text().splitlines():
        match = ALL_IDS_RE.match(raw.strip())
        if match:
            return {tok for tok in match.group(1).split() if ID_RE.match(tok)}
    return set()


def parse_timestamp(value: str) -> datetime:
    if not value:
        return datetime.min.replace(tzinfo=timezone.utc)
    normalized = value.replace("Z", "+00:00")
    try:
        parsed = datetime.fromisoformat(normalized)
    except ValueError:
        return datetime.min.replace(tzinfo=timezone.utc)
    if parsed.tzinfo is None:
        return parsed.replace(tzinfo=timezone.utc)
    return parsed.astimezone(timezone.utc)


def result_from_manifest(path: Path) -> Result | None:
    try:
        raw = json.loads(path.read_text())
    except (OSError, json.JSONDecodeError):
        return None
    usecase_id = str(raw.get("usecase_id", "")).strip()
    if not ID_RE.match(usecase_id):
        return None
    return Result(
        usecase_id=usecase_id,
        run_id=str(raw.get("run_id", "")),
        scenario_name=str(raw.get("scenario_name", "")),
        timestamp_end=str(raw.get("timestamp_end", "")),
        pass_=bool(raw.get("pass", False)),
        failing_assertions=tuple(str(item) for item in raw.get("failing_assertions", [])),
        interaction_trace=tuple(interaction_summaries(raw.get("interaction_trace", []))),
        frames=tuple(media_assets(raw, "frames", "screenshot", path)),
        videos=tuple(media_assets(raw, "videos", "video", path)),
        audio=tuple(media_assets(raw, "audio", "audio", path)),
        source=path.relative_to(REPO).as_posix(),
    )


def interaction_summaries(raw: object) -> list[str]:
    if not isinstance(raw, list):
        return []
    summaries: list[str] = []
    for item in raw:
        if not isinstance(item, dict):
            continue
        summary = str(item.get("summary", "")).strip()
        if summary:
            summaries.append(summary)
    return summaries


def media_assets(raw: dict[str, object], key: str, kind: str, manifest_path: Path) -> list[MediaAsset]:
    assets: list[MediaAsset] = []
    candidates: list[object] = []
    direct = raw.get(key)
    if isinstance(direct, list):
        candidates.extend(direct)
    media = raw.get("media")
    if isinstance(media, dict):
        nested = media.get(key)
        if isinstance(nested, list):
            candidates.extend(nested)
    for index, item in enumerate(candidates, start=1):
        asset = media_asset(item, kind, index, manifest_path)
        if asset is not None:
            assets.append(asset)
    return assets


def media_asset(item: object, kind: str, index: int, manifest_path: Path) -> MediaAsset | None:
    label = f"{kind.title()} {index}"
    path = ""
    if isinstance(item, str):
        path = item
    elif isinstance(item, dict):
        for key in ("path", "href", "uri", "source"):
            value = str(item.get(key, "")).strip()
            if value:
                path = value
                break
        label = str(item.get("label") or item.get("step_id") or item.get("id") or label)
    if not path:
        return None
    return MediaAsset(label=label, path=site_relative_asset_path(path, manifest_path), kind=kind)


def site_relative_asset_path(path: str, manifest_path: Path) -> str:
    if re.match(r"^[a-z][a-z0-9+.-]*:", path, re.IGNORECASE) or path.startswith("/"):
        return path
    manifest_dir = manifest_path.parent
    candidate = (manifest_dir / path).resolve()
    try:
        repo_relative = candidate.relative_to(REPO.resolve())
    except ValueError:
        repo_relative = Path(path)
    return "../../" + repo_relative.as_posix()


def latest_results(include_results: bool = False, include_validation_runs: bool = False) -> dict[str, Result]:
    results: dict[str, Result] = {}
    sources = []
    if include_results:
        sources.append((USECASE_RESULTS, "*/result.json"))
    if include_validation_runs:
        sources.append((USECASE_VALIDATION, "*/manifest.json"))
    for base, pattern in sources:
        if not base.exists():
            continue
        for path in base.glob(pattern):
            result = result_from_manifest(path)
            if result is None:
                continue
            previous = results.get(result.usecase_id)
            if previous is None or parse_timestamp(result.timestamp_end) > parse_timestamp(previous.timestamp_end):
                results[result.usecase_id] = result
    return results


def normalize_description(raw: object) -> str:
    return " ".join(str(raw or "").split())[:80] or "(no description)"


def resolved_descriptions() -> set[str]:
    resolved: set[str] = set()
    if not RESOLVED_BUGS.exists():
        return resolved
    for path in RESOLVED_BUGS.glob("*.json"):
        try:
            raw = json.loads(path.read_text())
        except (OSError, json.JSONDecodeError):
            continue
        resolved.add(normalize_description(raw.get("description")))
    return resolved


def tagged_usecase_ids(tags: object) -> set[str]:
    if not isinstance(tags, list):
        return set()
    ids: set[str] = set()
    for raw in tags:
        tag = str(raw).strip()
        normalized = tag.upper()
        if ID_RE.match(normalized):
            ids.add(normalized)
            continue
        for prefix in ("USECASE:", "USECASE=", "USE_CASE:", "USE_CASE=", "USE-CASE:", "USE-CASE="):
            if normalized.startswith(prefix):
                candidate = normalized[len(prefix) :]
                if ID_RE.match(candidate):
                    ids.add(candidate)
    return ids


def bug_report_from_path(path: Path, resolved: set[str]) -> tuple[set[str], BugReport] | None:
    try:
        raw = json.loads(path.read_text())
    except (OSError, json.JSONDecodeError):
        return None
    summary = raw.get("summary")
    if not isinstance(summary, dict):
        return None
    description = normalize_description(summary.get("description"))
    if description in resolved:
        return None
    tags = tuple(str(item) for item in summary.get("tags", []) if str(item).strip())
    ids = tagged_usecase_ids(list(tags))
    if not ids:
        return None
    report_id = str(summary.get("report_id") or path.stem)
    return (
        ids,
        BugReport(
            report_id=report_id,
            description="" if description == "(no description)" else description,
            tags=tags,
            source=path.relative_to(REPO).as_posix(),
        ),
    )


def open_bug_reports(include_bugs: bool = False) -> dict[str, tuple[BugReport, ...]]:
    if not include_bugs or not BUG_REPORTS.exists():
        return {}
    resolved = resolved_descriptions()
    by_usecase: dict[str, list[BugReport]] = {}
    for path in sorted(BUG_REPORTS.glob("*/*.json")):
        parsed = bug_report_from_path(path, resolved)
        if parsed is None:
            continue
        ids, report = parsed
        for usecase_id in ids:
            by_usecase.setdefault(usecase_id, []).append(report)
    return {usecase_id: tuple(reports) for usecase_id, reports in by_usecase.items()}


def parse_usecases() -> list[UseCase]:
    automated = automated_ids()
    usecases: list[UseCase] = []
    for path in sorted(USECASES_DIR.glob("*.md")):
        if path.name == "INDEX.md":
            continue
        text = path.read_text()
        fm = parse_frontmatter(text)
        family = fm.get("family", path.stem.upper())
        family_title = fm.get("title", path.stem.replace("-", " ").title())
        for raw in text.splitlines():
            cells = table_cells(raw)
            if not cells or len(cells) < 4 or not ID_RE.match(cells[0]):
                continue
            usecases.append(
                UseCase(
                    id=cells[0],
                    family=family,
                    family_title=family_title,
                    source=path.relative_to(REPO).as_posix(),
                    actor=cells[1],
                    action=cells[2],
                    outcome=cells[3],
                    automated=cells[0] in automated,
                )
            )
    return usecases


def status_rank(usecase: UseCase) -> int:
    result = RESULTS.get(usecase.id)
    if has_defect(usecase):
        return 0
    if not usecase.automated:
        return 1
    if result and is_stale(result):
        return 2
    return 3


def status_label(usecase: UseCase) -> str:
    result = RESULTS.get(usecase.id)
    if has_defect(usecase):
        return "DEFECT"
    if result and is_stale(result):
        return "STALE"
    return "PASSING" if usecase.automated else "UNTESTED"


def status_class(usecase: UseCase) -> str:
    result = RESULTS.get(usecase.id)
    if has_defect(usecase):
        return "defect"
    if result and is_stale(result):
        return "stale"
    return "passing" if usecase.automated else "untested"


def last_validated(usecase: UseCase) -> str:
    result = RESULTS.get(usecase.id)
    if not result:
        return "not captured yet"
    parsed = parse_timestamp(result.timestamp_end)
    if parsed == datetime.min.replace(tzinfo=timezone.utc):
        return html.escape(result.timestamp_end)
    return parsed.strftime("%Y-%m-%d %H:%M UTC")


def is_stale(result: Result) -> bool:
    if not result.pass_:
        return False
    parsed = parse_timestamp(result.timestamp_end)
    if parsed == datetime.min.replace(tzinfo=timezone.utc):
        return False
    return (datetime.now(timezone.utc) - parsed).days > STALE_RESULT_DAYS


def has_defect(usecase: UseCase) -> bool:
    result = RESULTS.get(usecase.id)
    return bool(result and not result.pass_) or bool(BUG_REPORTS_BY_USECASE.get(usecase.id))


def id_sort_key(usecase: UseCase) -> tuple[str, int, str]:
    match = re.match(r"^([A-Z]+)(\d+)$", usecase.id)
    if not match:
        return (usecase.id, 0, usecase.id)
    return (match.group(1), int(match.group(2)), usecase.id)


def page(title: str, body: str) -> str:
    return f"""<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{html.escape(title)} - Terminals Use Cases</title>
  <link rel="stylesheet" href="style.css">
  <script src="site.js" defer></script>
</head>
<body>
  <main>
{body}
  </main>
</body>
</html>
"""


def render_index(usecases: list[UseCase]) -> str:
    grouped: OrderedDict[str, list[UseCase]] = OrderedDict()
    for usecase in sorted(usecases, key=lambda u: (status_rank(u), u.family, id_sort_key(u))):
        grouped.setdefault(usecase.family_title, []).append(usecase)

    rows: list[str] = [
        "    <header class=\"site-header\">",
        "      <p class=\"eyebrow\">Generated from usecases/*.md</p>",
        "      <h1>Terminals Use Cases</h1>",
        "      <p class=\"lede\">One browseable index of what the system is meant to do and which behaviors are already covered by automated validation.</p>",
        "    </header>",
    ]
    for family, items in grouped.items():
        rows.append(f"    <section class=\"family\"><h2>{html.escape(family)}</h2>")
        rows.append("      <table><thead><tr><th><button type=\"button\">ID</button></th><th><button type=\"button\">Status</button></th><th><button type=\"button\">Use case</button></th><th><button type=\"button\">Last validated</button></th></tr></thead><tbody>")
        for usecase in items:
            rows.append(
                "        <tr>"
                f"<td><a href=\"{usecase.id}.html\">{usecase.id}</a></td>"
                f"<td><span class=\"badge {status_class(usecase)}\">{status_label(usecase)}</span></td>"
                f"<td>{html.escape(usecase.title)}</td>"
                f"<td>{last_validated(usecase)}</td>"
                "</tr>"
            )
        rows.append("      </tbody></table></section>")
    return page("Index", "\n".join(rows))


def render_usecase(usecase: UseCase) -> str:
    header_class = status_class(usecase)
    result = RESULTS.get(usecase.id)
    bugs = BUG_REPORTS_BY_USECASE.get(usecase.id, ())
    if result and not result.pass_:
        header_text = f"Failed on {last_validated(usecase)}: {result.first_failure}."
    elif bugs:
        noun = "defect" if len(bugs) == 1 else "defects"
        header_text = f"{len(bugs)} open {noun} tagged for this use case."
    elif result and is_stale(result):
        header_text = f"Last validated on {last_validated(usecase)} - result is older than {STALE_RESULT_DAYS} days."
    elif result:
        header_text = f"Validated on {last_validated(usecase)} - all assertions passed."
    elif usecase.automated:
        header_text = f"Automated validation is wired. Run 'make usecase-validate USECASE={usecase.id}' to generate results."
    else:
        header_text = "UNTESTED - no automated scenario exists for this use case."

    defect_html = render_defects(result, bugs)
    evidence_items = [
        f'<li><a href="../../{html.escape(usecase.source)}">{html.escape(usecase.source)}</a></li>',
    ]
    if usecase.automated:
        evidence_items.append(f"<li><code>make usecase-validate USECASE={usecase.id}</code></li>")
    else:
        evidence_items.append("<li>No automated validation command wired yet.</li>")
    if result:
        evidence_items.append(f'<li><a href="../../{html.escape(result.source)}">Latest result manifest</a></li>')
        if result.scenario_name:
            evidence_items.append(f"<li>Scenario: {html.escape(result.scenario_name)}</li>")
    evidence_html = "\n        ".join(evidence_items)
    if result and result.interaction_trace:
        interaction_items = "\n        ".join(f"<li>{html.escape(item)}</li>" for item in result.interaction_trace)
        interaction_html = f"<ol>\n        {interaction_items}\n      </ol>"
    elif usecase.automated:
        interaction_html = f'<p class="placeholder">Run <code>make usecase-validate USECASE={usecase.id}</code> to generate interaction traces for this use case.</p>'
    else:
        interaction_html = '<p class="placeholder">No automated scenario exists yet. Interaction trace not available.</p>'
    visual_html = render_visual_media(result)
    audio_html = render_audio_media(result)

    body = f"""    <p class="back"><a href="index.html">Back to all use cases</a></p>
    <header class="case-header {header_class}">
      <p class="eyebrow">{html.escape(usecase.family_title)} / {usecase.id}</p>
      <h1>{html.escape(usecase.title)}</h1>
      <p>{html.escape(header_text)}</p>
    </header>
    <section>
      <h2>What it does</h2>
      <p>As a {html.escape(usecase.actor)}, I would like to {html.escape(usecase.action)} so that I can {html.escape(usecase.outcome)}.</p>
    </section>
    <section>
      <h2>How to use it</h2>
      {interaction_html}
    </section>
    <section>
      <h2>What you see</h2>
      {visual_html}
    </section>
    <section>
      <h2>What you hear</h2>
      {audio_html}
    </section>
    <section>
      <h2>Defects</h2>
      {defect_html}
    </section>
    <section>
      <h2>Evidence</h2>
      <ul>
        {evidence_html}
      </ul>
    </section>"""
    return page(f"{usecase.id} {usecase.title}", body)


def render_visual_media(result: Result | None) -> str:
    if not result or (not result.frames and not result.videos):
        return '<p class="placeholder">Rendered server-primitive screenshots are not captured yet.</p>'
    audit_href = "../../" + UI_AUDIT.relative_to(REPO).as_posix()
    parts = [
        f'<p class="media-note">Rendered from server primitives; client pixel parity is covered by the <a href="{html.escape(audit_href)}">manual UI audit</a>.</p>',
    ]
    for video in result.videos:
        parts.append(
            f'<figure class="media-block"><video controls muted loop playsinline src="{html.escape(video.path)}"></video>'
            f"<figcaption>{html.escape(video.label)}</figcaption></figure>"
        )
    if result.frames:
        frame_items = "\n        ".join(
            f'<a class="frame-link" href="{html.escape(frame.path)}"><img src="{html.escape(frame.path)}" alt="{html.escape(frame.label)}"><span>{html.escape(frame.label)}</span></a>'
            for frame in result.frames
        )
        parts.append(f'<div class="frame-strip">\n        {frame_items}\n      </div>')
    return "\n      ".join(parts)


def render_audio_media(result: Result | None) -> str:
    if not result or not result.audio:
        return '<p class="placeholder">Audio artifacts are not captured yet.</p>'
    items = "\n        ".join(
        f'<figure class="media-block audio-block"><figcaption>{html.escape(asset.label)}</figcaption><audio controls src="{html.escape(asset.path)}"></audio></figure>'
        for asset in result.audio
    )
    return items


def render_defects(result: Result | None, bugs: tuple[BugReport, ...]) -> str:
    parts: list[str] = []
    if result and not result.pass_:
        failures = result.failing_assertions or ("validation failed",)
        frame_by_label = {frame.label: frame for frame in result.frames}
        items: list[str] = []
        for failure in failures:
            frame = frame_by_label.get(failure)
            frame_link = ""
            if frame is not None:
                frame_link = f' <a href="{html.escape(frame.path)}">failure frame</a>'
            items.append(f"<li>{html.escape(failure)}{frame_link}</li>")
        parts.append("<p>Latest validation failed:</p>")
        parts.append("<ul>\n        " + "\n        ".join(items) + "\n      </ul>")
    elif result and result.pass_:
        parts.append("<p>No defects detected in the latest captured result.</p>")
    else:
        parts.append("<p>No result.json defect feed is available yet.</p>")

    if bugs:
        bug_items = "\n        ".join(
            f'<li><a href="../../{html.escape(bug.source)}">{html.escape(bug.label)}</a></li>' for bug in bugs
        )
        parts.append("<p>Open bug reports:</p>")
        parts.append(f"<ul>\n        {bug_items}\n      </ul>")
    return "\n      ".join(parts)


def stylesheet() -> str:
    return """body {
  margin: 0;
  background: #f7f7f4;
  color: #1d2528;
  font: 16px/1.5 -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
}

main {
  max-width: 1120px;
  margin: 0 auto;
  padding: 32px 20px 56px;
}

a { color: #0b5cad; }

.site-header, .case-header, section {
  margin-bottom: 24px;
}

.site-header {
  border-bottom: 3px solid #2d5a63;
  padding-bottom: 18px;
}

.case-header {
  padding: 22px;
  color: #111;
}

.case-header.passing { background: #dff1df; border-left: 8px solid #247a35; }
.case-header.untested { background: #fff2cc; border-left: 8px solid #a06c00; }
.case-header.stale { background: #fff6d8; border-left: 8px solid #b08a00; }
.case-header.defect { background: #ffe1df; border-left: 8px solid #b42318; }

.eyebrow {
  margin: 0 0 6px;
  color: #526064;
  font-size: 0.8rem;
  font-weight: 700;
  letter-spacing: 0;
  text-transform: uppercase;
}

h1, h2 { line-height: 1.15; }
h1 { margin: 0 0 12px; font-size: 2.2rem; }
h2 { margin: 0 0 10px; font-size: 1.25rem; }

.lede { max-width: 760px; font-size: 1.1rem; }

.family {
  margin-top: 28px;
}

table {
  width: 100%;
  border-collapse: collapse;
  background: #fff;
}

th, td {
  border-bottom: 1px solid #d9dedc;
  padding: 10px 12px;
  text-align: left;
  vertical-align: top;
}

th {
  background: #e8ece9;
  color: #334044;
  font-size: 0.85rem;
}

th button {
  appearance: none;
  border: 0;
  background: transparent;
  color: inherit;
  cursor: pointer;
  font: inherit;
  font-weight: 700;
  padding: 0;
  text-align: left;
}

.badge {
  display: inline-block;
  min-width: 82px;
  padding: 3px 7px;
  border-radius: 4px;
  font-size: 0.78rem;
  font-weight: 700;
  text-align: center;
}

.badge.passing { background: #247a35; color: #fff; }
.badge.untested { background: #a06c00; color: #fff; }
.badge.stale { background: #b08a00; color: #111; }
.badge.defect { background: #b42318; color: #fff; }

.placeholder {
  color: #586467;
}

.media-note {
  color: #586467;
  margin-top: 0;
}

.media-block {
  margin: 0 0 16px;
}

.media-block video {
  width: 100%;
  max-height: 560px;
  background: #111;
}

.media-block audio {
  width: 100%;
}

.media-block figcaption {
  color: #465154;
  font-size: 0.9rem;
  margin-top: 6px;
}

.audio-block figcaption {
  margin: 0 0 6px;
}

.frame-strip {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 12px;
}

.frame-link {
  background: #fff;
  border: 1px solid #d9dedc;
  color: #1d2528;
  display: block;
  text-decoration: none;
}

.frame-link img {
  aspect-ratio: 16 / 9;
  background: #111;
  display: block;
  object-fit: cover;
  width: 100%;
}

.frame-link span {
  display: block;
  font-size: 0.85rem;
  padding: 6px 8px;
}

.back {
  margin: 0 0 18px;
}

code {
  background: #eef1ee;
  border-radius: 3px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 0.875em;
  padding: 2px 5px;
}

dialog#frame-lightbox {
  background: rgba(0, 0, 0, 0.92);
  border: 0;
  max-height: 95vh;
  max-width: 95vw;
  padding: 0;
}

dialog#frame-lightbox figure {
  margin: 0;
  padding: 0;
  position: relative;
}

dialog#frame-lightbox img {
  display: block;
  max-height: 88vh;
  max-width: 92vw;
  object-fit: contain;
}

dialog#frame-lightbox figcaption {
  color: #bcc5c7;
  font-size: 0.88rem;
  padding: 8px 40px 10px 14px;
  text-align: center;
}

.lightbox-close {
  appearance: none;
  background: transparent;
  border: 0;
  color: #fff;
  cursor: pointer;
  font-size: 1.4rem;
  line-height: 1;
  padding: 8px 12px;
  position: absolute;
  right: 0;
  top: 0;
}

dialog::backdrop {
  background: rgba(0, 0, 0, 0.65);
}

@media (max-width: 680px) {
  main { padding: 20px 12px 40px; }
  h1 { font-size: 1.7rem; }
  th, td { padding: 8px 6px; }
}
"""


def javascript() -> str:
    return """document.querySelectorAll("table").forEach((table) => {
  const tbody = table.querySelector("tbody");
  table.querySelectorAll("th button").forEach((button, column) => {
    button.addEventListener("click", () => {
      const rows = Array.from(tbody.querySelectorAll("tr"));
      const direction = button.dataset.direction === "asc" ? -1 : 1;
      table.querySelectorAll("th button").forEach((other) => {
        other.dataset.direction = "";
      });
      button.dataset.direction = direction === 1 ? "asc" : "desc";
      rows.sort((a, b) => {
        const left = a.children[column].innerText.trim();
        const right = b.children[column].innerText.trim();
        return left.localeCompare(right, undefined, { numeric: true }) * direction;
      });
      rows.forEach((row) => tbody.appendChild(row));
    });
  });
});

(function () {
  const links = document.querySelectorAll(".frame-link");
  if (!links.length) return;
  const dialog = document.createElement("dialog");
  dialog.id = "frame-lightbox";
  dialog.innerHTML =
    '<figure><button class="lightbox-close" aria-label="Close">×</button>' +
    '<img id="lightbox-img" src="" alt=""><figcaption id="lightbox-caption"></figcaption></figure>';
  document.body.appendChild(dialog);
  dialog.querySelector(".lightbox-close").addEventListener("click", () => dialog.close());
  dialog.addEventListener("click", (e) => { if (e.target === dialog) dialog.close(); });
  links.forEach((link) => {
    link.addEventListener("click", (e) => {
      e.preventDefault();
      document.getElementById("lightbox-img").src = link.href;
      document.getElementById("lightbox-img").alt = link.querySelector("span").textContent;
      document.getElementById("lightbox-caption").textContent = link.querySelector("span").textContent;
      dialog.showModal();
    });
  });
}());
"""


def generated_files(usecases: list[UseCase]) -> dict[Path, str]:
    files = {
        OUT / "index.html": render_index(usecases),
        OUT / "style.css": stylesheet(),
        OUT / "site.js": javascript(),
    }
    for usecase in usecases:
        files[OUT / f"{usecase.id}.html"] = render_usecase(usecase)
    return files


def write_files(files: dict[Path, str]) -> None:
    if OUT.exists():
        shutil.rmtree(OUT)
    if EMBED_OUT.exists():
        shutil.rmtree(EMBED_OUT)
    OUT.mkdir(parents=True)
    for path, content in files.items():
        path.write_text(content)
    shutil.copytree(OUT, EMBED_OUT)


def check_files(files: dict[Path, str]) -> int:
    missing: list[str] = []
    changed: list[str] = []
    extra: list[str] = []
    for base in (OUT, EMBED_OUT):
        expected = {base / path.name for path in files}
        actual = set(base.glob("*")) if base.exists() else set()
        for path, content in ((base / p.name, content) for p, content in files.items()):
            if not path.exists():
                missing.append(path.relative_to(REPO).as_posix())
            elif path.read_text() != content:
                changed.append(path.relative_to(REPO).as_posix())
        extra.extend(sorted(path.relative_to(REPO).as_posix() for path in actual - expected))
    if missing or changed or extra:
        for label, paths in (("missing", missing), ("stale", changed), ("extra", extra)):
            for path in paths:
                print(f"{label}: {path}", file=sys.stderr)
        print("ERROR: docs/usecases-site is out of date. Run `make usecases-site`.", file=sys.stderr)
        return 1
    return 0


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--check", action="store_true", help="fail if generated output differs from disk")
    parser.add_argument(
        "--include-results",
        action="store_true",
        help="read artifacts/usecases/<ID>/result.json files into the generated status pages",
    )
    parser.add_argument(
        "--include-validation-runs",
        action="store_true",
        help="also read ephemeral artifacts/usecase-validation/*/manifest.json files",
    )
    parser.add_argument(
        "--include-bugs",
        action="store_true",
        help="read terminal_server/logs/bug_reports entries tagged with use-case IDs into the generated status pages",
    )
    args = parser.parse_args()

    global RESULTS
    global BUG_REPORTS_BY_USECASE
    RESULTS = latest_results(args.include_results, args.include_validation_runs)
    BUG_REPORTS_BY_USECASE = open_bug_reports(args.include_bugs)
    usecases = parse_usecases()
    if not usecases:
        print("ERROR: no use cases found under usecases/.", file=sys.stderr)
        return 2
    files = generated_files(usecases)
    if args.check:
        return check_files(files)
    write_files(files)
    untested = sum(1 for usecase in usecases if not usecase.automated)
    print(f"wrote {OUT.relative_to(REPO)} ({len(usecases)} use cases, {untested} untested)")
    return 0


RESULTS: dict[str, Result] = {}
BUG_REPORTS_BY_USECASE: dict[str, tuple[BugReport, ...]] = {}

if __name__ == "__main__":
    sys.exit(main())
