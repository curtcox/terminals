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
class Result:
    usecase_id: str
    run_id: str
    scenario_name: str
    timestamp_end: str
    pass_: bool
    failing_assertions: tuple[str, ...]
    source: str

    @property
    def first_failure(self) -> str:
        if self.failing_assertions:
            return self.failing_assertions[0]
        return "validation failed"


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
        source=path.relative_to(REPO).as_posix(),
    )


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
    if result and not result.pass_:
        return 0
    return 3 if usecase.automated else 1


def status_label(usecase: UseCase) -> str:
    result = RESULTS.get(usecase.id)
    if result and not result.pass_:
        return "DEFECT"
    return "PASSING" if usecase.automated else "UNTESTED"


def status_class(usecase: UseCase) -> str:
    result = RESULTS.get(usecase.id)
    if result and not result.pass_:
        return "defect"
    return "passing" if usecase.automated else "untested"


def last_validated(usecase: UseCase) -> str:
    result = RESULTS.get(usecase.id)
    if not result:
        return "not captured yet"
    parsed = parse_timestamp(result.timestamp_end)
    if parsed == datetime.min.replace(tzinfo=timezone.utc):
        return html.escape(result.timestamp_end)
    return parsed.strftime("%Y-%m-%d %H:%M UTC")


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
    if result and not result.pass_:
        header_text = f"Failed on {last_validated(usecase)}: {result.first_failure}."
    elif result:
        header_text = f"Validated on {last_validated(usecase)} - all assertions passed."
    elif usecase.automated:
        header_text = "Automated validation is wired for this use case. Result capture will land in the next milestone."
    else:
        header_text = "UNTESTED - no automated scenario exists for this use case."

    defect_text = "No defects detected in the latest captured result." if result and result.pass_ else "No result.json defect feed is available yet."
    if result and not result.pass_:
        failures = ", ".join(html.escape(item) for item in result.failing_assertions) or "validation failed"
        defect_text = f"Latest validation failed: {failures}."
    evidence_items = [
        f'<li><a href="../../{html.escape(usecase.source)}">{html.escape(usecase.source)}</a></li>',
    ]
    if usecase.automated:
        evidence_items.append(f"<li>make usecase-validate USECASE={usecase.id}</li>")
    else:
        evidence_items.append("<li>No automated validation command wired yet.</li>")
    if result:
        evidence_items.append(f'<li><a href="../../{html.escape(result.source)}">Latest result manifest</a></li>')
        if result.scenario_name:
            evidence_items.append(f"<li>Scenario: {html.escape(result.scenario_name)}</li>")
    evidence_html = "\n        ".join(evidence_items)

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
      <p class="placeholder">Interaction traces are not captured yet. This section will be generated from validation scenarios.</p>
    </section>
    <section>
      <h2>What you see</h2>
      <p class="placeholder">Rendered server-primitive screenshots are not captured yet.</p>
    </section>
    <section>
      <h2>What you hear</h2>
      <p class="placeholder">Audio artifacts are not captured yet.</p>
    </section>
    <section>
      <h2>Defects</h2>
      <p>{defect_text}</p>
    </section>
    <section>
      <h2>Evidence</h2>
      <ul>
        {evidence_html}
      </ul>
    </section>"""
    return page(f"{usecase.id} {usecase.title}", body)


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
.badge.defect { background: #b42318; color: #fff; }

.placeholder {
  color: #586467;
}

.back {
  margin: 0 0 18px;
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
    args = parser.parse_args()

    global RESULTS
    RESULTS = latest_results(args.include_results, args.include_validation_runs)
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

if __name__ == "__main__":
    sys.exit(main())
