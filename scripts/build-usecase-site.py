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
from datetime import datetime, timezone
from pathlib import Path

from usecasesite_assets import javascript, stylesheet
from usecasesite_models import (
    BugReport,
    InteractionStep,
    MediaAsset,
    Result,
    UseCase,
    ValidationLink,
)

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
UI_INSPECT_SKILL = REPO / ".claude" / "skills" / "ui-inspect" / "SKILL.md"
STALE_RESULT_DAYS = 30
MAX_FRAME_STRIP_ITEMS = 24

ID_RE = re.compile(r"^[A-Z]+\d+$")
FRONTMATTER_RE = re.compile(r"^---\n(.*?)\n---\n", re.DOTALL)
ALL_IDS_RE = re.compile(r"^all_ids=\(([^)]*)\)\s*$")
METADATA_RE = re.compile(r"^\s*([A-Z]+\d+)\)\s+echo\s+\"([A-Z]+\d+)\|([^|]*)\|([^\"]*)\"\s+;;")
VALIDATOR_YAML_RE = re.compile(
    r"(?:terminal_server/)?(internal/usecasevalidation/testdata/[A-Za-z0-9_.-]+\.yaml)"
)
RUN_GO_TEST_RE = re.compile(r"run_go_test\s+(\./\S+)\s+'([^']+)'")
RUN_FLUTTER_TEST_RE = re.compile(r"run_flutter_test\s+(\S+)\s+'([^']+)'")
RUN_APP_TEST_RE = re.compile(r"run_app_test\s+'([^']+)'")
CASE_BLOCK_RE = re.compile(
    r"^\s+([A-Z]+\d+)\)\s*\n((?:[ \t]+.*\n)*?[ \t]+;;\s*\n)",
    re.MULTILINE,
)



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


def validation_metadata() -> dict[str, tuple[str, str]]:
    if not VALIDATOR.exists():
        return {}
    metadata: dict[str, tuple[str, str]] = {}
    for raw in VALIDATOR.read_text().splitlines():
        match = METADATA_RE.match(raw)
        if not match:
            continue
        case_id, echoed_id, depth, evidence = match.groups()
        if case_id != echoed_id:
            continue
        metadata[case_id] = (depth.strip(), evidence.strip())
    return metadata


def validator_text() -> str:
    if not VALIDATOR.exists():
        return ""
    return VALIDATOR.read_text()


def run_usecase_block(usecase_id: str) -> str:
    match = CASE_BLOCK_RE.search(validator_text())
    while match:
        if match.group(1) == usecase_id:
            return match.group(2)
        match = CASE_BLOCK_RE.search(validator_text(), match.end())
    return ""


def terminal_server_package_dir(package: str) -> Path:
    return REPO / "terminal_server" / package.removeprefix("./")


def primary_go_test_name(run_pattern: str) -> str | None:
    candidate = run_pattern.rstrip("$")
    direct = re.search(r"Test(?:UseCase[A-Z0-9]+WithEvidence|YAMLScenario[A-Za-z0-9]+)", candidate)
    if direct:
        return direct.group(0)
    if candidate.startswith("Test") and "|" not in candidate and "(" not in candidate:
        return candidate
    return None


def discover_go_test_files(package: str, run_pattern: str) -> list[Path]:
    cache_key = (package, run_pattern)
    if cache_key in GO_TEST_FILE_CACHE:
        return GO_TEST_FILE_CACHE[cache_key]
    test_name = primary_go_test_name(run_pattern)
    if not test_name:
        GO_TEST_FILE_CACHE[cache_key] = []
        return []
    pkg_dir = terminal_server_package_dir(package)
    matches: list[Path] = []
    if pkg_dir.exists():
        for path in pkg_dir.rglob("*_test.go"):
            try:
                text = path.read_text()
            except OSError:
                continue
            if f"func {test_name}(" in text:
                matches.append(path)
    GO_TEST_FILE_CACHE[cache_key] = matches
    return matches


def yaml_paths_from_evidence(evidence: str) -> list[str]:
    paths: list[str] = []
    for match in VALIDATOR_YAML_RE.findall(evidence):
        normalized = match if match.startswith("terminal_server/") else f"terminal_server/{match}"
        if (REPO / normalized).exists():
            paths.append(normalized)
    return sorted(dict.fromkeys(paths))


def validation_evidence_links(usecase_id: str) -> tuple[ValidationLink, ...]:
    if not VALIDATOR.exists():
        return ()
    metadata = validation_metadata()
    _, evidence = metadata.get(usecase_id, ("", ""))
    links: list[ValidationLink] = []
    seen: set[str] = set()

    def add(label: str, path: str) -> None:
        if path in seen:
            return
        seen.add(path)
        links.append(ValidationLink(label=label, path=path))

    for yaml_path in yaml_paths_from_evidence(evidence):
        add(f"YAML scenario: {Path(yaml_path).name}", yaml_path)

    block = run_usecase_block(usecase_id)
    for package, run_pattern in RUN_GO_TEST_RE.findall(block):
        for path in discover_go_test_files(package, run_pattern):
            rel = path.relative_to(REPO).as_posix()
            test_name = primary_go_test_name(run_pattern) or run_pattern.rstrip("$")
            add(f"Go test: {test_name} ({path.name})", rel)
    for rel_path, plain_name in RUN_FLUTTER_TEST_RE.findall(block):
        client_path = f"terminal_client/{rel_path}"
        if (REPO / client_path).exists():
            add(f"Flutter test: {plain_name}", client_path)
    for app_name in RUN_APP_TEST_RE.findall(block):
        add(f"App test: {app_name}", "terminal_server/cmd/term")

    return tuple(links)


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
        interaction_trace=tuple(interaction_steps(raw.get("interaction_trace", []))),
        frames=tuple(media_assets(raw, "frames", "screenshot", path)),
        videos=tuple(media_assets(raw, "videos", "video", path)),
        audio=tuple(media_assets(raw, "audio", "audio", path)),
        source=path.relative_to(REPO).as_posix(),
    )


def interaction_steps(raw: object) -> list[InteractionStep]:
    if not isinstance(raw, list):
        return []
    steps: list[InteractionStep] = []
    for item in raw:
        if not isinstance(item, dict):
            continue
        summary = str(item.get("summary", "")).strip()
        if summary:
            steps.append(
                InteractionStep(
                    kind=str(item.get("kind", "")).strip(),
                    summary=summary,
                    terminal=str(item.get("terminal", "")).strip(),
                )
            )
    return steps


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
    source = ""
    rights_note = ""
    transcript = ""
    if isinstance(item, dict):
        source = str(item.get("source", "")).strip()
        rights_note = str(item.get("rights_note", "") or item.get("rightsNote", "")).strip()
        transcript = str(item.get("transcript", "")).strip()
    return MediaAsset(
        label=label,
        path=site_relative_asset_path(path, manifest_path),
        kind=kind,
        source=source,
        rights_note=rights_note,
        transcript=transcript,
    )


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
    metadata = validation_metadata()
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
            depth, evidence = metadata.get(cells[0], ("", ""))
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
                    validation_depth=depth,
                    validation_evidence=evidence,
                )
            )
    return usecases


def status_rank(usecase: UseCase) -> int:
    result = RESULTS.get(usecase.id)
    if has_defect(usecase):
        return 0
    if not usecase.automated:
        return 1
    if not result:
        return 2
    if is_stale(result):
        return 3
    return 4


def status_label(usecase: UseCase) -> str:
    result = RESULTS.get(usecase.id)
    if has_defect(usecase):
        return "DEFECT"
    if not usecase.automated:
        return "UNTESTED"
    if not result:
        return "NOT VALIDATED"
    if result and is_stale(result):
        return "STALE"
    return "PASSING"


def status_class(usecase: UseCase) -> str:
    result = RESULTS.get(usecase.id)
    if has_defect(usecase):
        return "defect"
    if not usecase.automated:
        return "untested"
    if not result:
        return "not-validated"
    if result and is_stale(result):
        return "stale"
    return "passing"


def last_validated(usecase: UseCase) -> str:
    result = RESULTS.get(usecase.id)
    if not result:
        return "not captured yet"
    parsed = parse_timestamp(result.timestamp_end)
    if parsed == datetime.min.replace(tzinfo=timezone.utc):
        return html.escape(result.timestamp_end)
    return parsed.strftime("%Y-%m-%d %H:%M UTC")


def last_validated_sort_key(usecase: UseCase) -> str:
    result = RESULTS.get(usecase.id)
    if not result:
        return ""
    parsed = parse_timestamp(result.timestamp_end)
    if parsed == datetime.min.replace(tzinfo=timezone.utc):
        return result.timestamp_end
    return parsed.isoformat()


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

    status_counts = OrderedDict(
        (
            ("defect", sum(1 for usecase in usecases if status_class(usecase) == "defect")),
            ("untested", sum(1 for usecase in usecases if status_class(usecase) == "untested")),
            ("not-validated", sum(1 for usecase in usecases if status_class(usecase) == "not-validated")),
            ("stale", sum(1 for usecase in usecases if status_class(usecase) == "stale")),
            ("passing", sum(1 for usecase in usecases if status_class(usecase) == "passing")),
        )
    )
    status_summary = "\n".join(
        f'        <li><span class="badge {name}">{status_display_name(name)}</span><strong>{count}</strong></li>'
        for name, count in status_counts.items()
    )
    rows: list[str] = [
        "    <header class=\"site-header\">",
        "      <p class=\"eyebrow\">Generated from usecases/*.md</p>",
        "      <h1>Terminals Use Cases</h1>",
        "      <p class=\"lede\">One browseable index of what the system is meant to do and which behaviors are already covered by automated validation.</p>",
        "    </header>",
        "    <section class=\"status-overview\" aria-label=\"Use case status summary\">",
        "      <ul>",
        status_summary,
        "      </ul>",
        "    </section>",
        "    <section class=\"index-filter\" aria-label=\"Filter use cases\">",
        "      <label for=\"usecase-filter\">Filter</label>",
        "      <input id=\"usecase-filter\" type=\"search\" placeholder=\"Search ID, status, or use case\">",
        "      <select id=\"status-filter\" aria-label=\"Filter by status\">",
        "        <option value=\"\">All statuses</option>",
        "        <option value=\"defect\">DEFECT</option>",
        "        <option value=\"untested\">UNTESTED</option>",
        "        <option value=\"not-validated\">NOT VALIDATED</option>",
        "        <option value=\"stale\">STALE</option>",
        "        <option value=\"passing\">PASSING</option>",
        "      </select>",
        "    </section>",
    ]
    for family, items in grouped.items():
        rows.append(f"    <section class=\"family\"><h2>{html.escape(family)}</h2>")
        rows.append(
            "      <table><thead><tr>"
            "<th aria-sort=\"none\"><button type=\"button\" data-sort-key=\"id\">ID</button></th>"
            "<th aria-sort=\"none\"><button type=\"button\" data-sort-key=\"status\">Status</button></th>"
            "<th aria-sort=\"none\"><button type=\"button\" data-sort-key=\"title\">Use case</button></th>"
            "<th aria-sort=\"none\"><button type=\"button\" data-sort-key=\"validated\">Last validated</button></th>"
            "</tr></thead><tbody>"
        )
        for usecase in items:
            rows.append(
                "        <tr"
                f" data-status=\"{status_class(usecase)}\""
                f" data-filter=\"{html.escape((usecase.id + ' ' + status_label(usecase) + ' ' + usecase.title + ' ' + usecase.family_title).lower())}\""
                f" data-sort-id=\"{html.escape(usecase.id)}\""
                f" data-sort-status=\"{status_rank(usecase)}\""
                f" data-sort-title=\"{html.escape(usecase.title.lower())}\""
                f" data-sort-validated=\"{html.escape(last_validated_sort_key(usecase))}\""
                ">"
                f"<td><a href=\"{usecase.id}.html\">{usecase.id}</a></td>"
                f"<td><span class=\"badge {status_class(usecase)}\">{status_label(usecase)}</span></td>"
                f"<td>{html.escape(usecase.title)}</td>"
                f"<td>{last_validated(usecase)}</td>"
                "</tr>"
            )
        rows.append("      </tbody></table></section>")
    return page("Index", "\n".join(rows))


def status_display_name(status: str) -> str:
    return status.replace("-", " ").upper()


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
        header_text = f"NOT VALIDATED - automated validation is wired, but no captured passing result exists yet. Run 'make usecase-validate USECASE={usecase.id}' to generate results."
    else:
        header_text = "UNTESTED - no automated scenario exists for this use case."

    defect_html = render_defects(result, bugs)
    evidence_items = [
        f'<li><a href="../../{html.escape(usecase.source)}">{html.escape(usecase.source)}</a></li>',
    ]
    if usecase.automated:
        evidence_items.append(f"<li><code>make usecase-validate USECASE={usecase.id}</code></li>")
        if usecase.validation_evidence:
            depth = f"{usecase.validation_depth}: " if usecase.validation_depth else ""
            evidence_items.append(
                f"<li>Primary validation evidence: {html.escape(depth + usecase.validation_evidence)}</li>"
            )
        for link in validation_evidence_links(usecase.id):
            evidence_items.append(
                f'<li><a href="../../{html.escape(link.path)}">{html.escape(link.label)}</a></li>'
            )
    else:
        evidence_items.append("<li>No automated validation command wired yet.</li>")
    if UI_INSPECT_SKILL.exists():
        evidence_items.append(
            f'<li><a href="../../{html.escape(UI_INSPECT_SKILL.relative_to(REPO).as_posix())}">Client UI inspection workflow</a></li>'
        )
    if UI_AUDIT.exists():
        evidence_items.append(f"<li><code>make usecase-wiring-audit</code></li>")
    if result:
        evidence_items.append(f'<li><a href="../../{html.escape(result.source)}">Latest result manifest</a></li>')
        if result.scenario_name:
            evidence_items.append(f"<li>Scenario: {html.escape(result.scenario_name)}</li>")
    evidence_html = "\n        ".join(evidence_items)
    if result and result.interaction_trace:
        interaction_items = "\n        ".join(f"<li>{html.escape(item.summary)}</li>" for item in result.interaction_trace)
        interaction_html = f"<ol>\n        {interaction_items}\n      </ol>"
    elif usecase.automated:
        interaction_html = f'<p class="placeholder">Run <code>make usecase-validate USECASE={usecase.id}</code> to generate interaction traces for this use case.</p>'
    else:
        interaction_html = '<p class="placeholder">No automated scenario exists yet. Interaction trace not available.</p>'

    if usecase.automated:
        visual_html = render_visual_media(result)
        audio_html = render_audio_media(result)
        media_sections = f"""    <section>
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
    </section>"""
    else:
        media_sections = ""

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
    {media_sections}
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
            f'<figure class="media-block"><video controls autoplay muted loop playsinline src="{html.escape(video.path)}"></video>'
            f"<figcaption>{html.escape(video.label)}</figcaption></figure>"
        )
    if result.frames:
        first = result.frames[0]
        max_index = len(result.frames) - 1
        preview_options = html.escape(json.dumps([{"src": frame.path, "label": frame.label} for frame in result.frames]))
        parts.append(
            f'<div class="frame-scrubber" data-frames="{preview_options}">'
            f'<img class="frame-preview-image" src="{html.escape(first.path)}" alt="{html.escape(first.label)}">'
            f'<div class="frame-scrubber-controls"><input class="frame-range" type="range" min="0" max="{max_index}" value="0" aria-label="Scrub captured frames">'
            f'<span class="frame-preview-label">{html.escape(first.label)}</span></div></div>'
        )
        strip_frames = result.frames[:MAX_FRAME_STRIP_ITEMS]
        frame_items = "\n        ".join(
            f'<a class="frame-link" href="{html.escape(frame.path)}"><img src="{html.escape(frame.path)}" alt="{html.escape(frame.label)}"><span>{html.escape(frame.label)}</span></a>'
            for frame in strip_frames
        )
        parts.append(f'<div class="frame-strip">\n        {frame_items}\n      </div>')
        if len(result.frames) > len(strip_frames):
            hidden = len(result.frames) - len(strip_frames)
            parts.append(
                f'<p class="media-note">{hidden} additional frames are available through the scrubber and raw result manifest.</p>'
            )
    if result.interaction_trace:
        parts.append(render_interaction_transcript(result.interaction_trace))
    return "\n      ".join(parts)


def render_interaction_transcript(interactions: tuple[InteractionStep, ...]) -> str:
    rows: list[str] = []
    for index, interaction in enumerate(interactions, start=1):
        detail = interaction.summary
        if interaction.terminal:
            detail = f"{detail} ({interaction.terminal})"
        kind = interaction.kind.upper() if interaction.kind else "STEP"
        rows.append(
            f"<li><span>{index}</span><strong>{html.escape(kind)}</strong>{html.escape(detail)}</li>"
        )
    items = "\n          ".join(rows)
    return (
        '<aside class="interaction-transcript" aria-label="Interaction transcript">'
        "<h3>Interaction transcript</h3>"
        f"<ol>\n          {items}\n        </ol>"
        "</aside>"
    )


def render_audio_media(result: Result | None) -> str:
    if not result or not result.audio:
        return '<p class="placeholder">Audio artifacts are not captured yet.</p>'
    figures: list[str] = []
    for asset in result.audio:
        notes: list[str] = []
        if asset.source:
            notes.append(f"Source: {html.escape(asset.source)}")
        if asset.transcript:
            notes.append(f"Transcript: {html.escape(asset.transcript)}")
        if asset.rights_note:
            notes.append(html.escape(asset.rights_note))
        notes_html = ""
        if notes:
            notes_html = '<p class="media-note">' + "<br>".join(notes) + "</p>"
        figures.append(
            f'<figure class="media-block audio-block"><figcaption>{html.escape(asset.label)}</figcaption>'
            f'<audio controls src="{html.escape(asset.path)}"></audio>{notes_html}</figure>'
        )
    return "\n        ".join(figures)


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
GO_TEST_FILE_CACHE: dict[tuple[str, str], list[Path]] = {}

if __name__ == "__main__":
    sys.exit(main())
