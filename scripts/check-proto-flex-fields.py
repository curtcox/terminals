#!/usr/bin/env python3
"""Advisory check for flexible protobuf fields.

The check scans api/terminals/**/*.proto for maps, JSON string fields, and
field names that commonly hide protocol semantics. A field is considered
governed when docs/protocol-extension-registry.md contains:

    Field: <package>.<Message>.<field>
"""

from __future__ import annotations

import argparse
import re
import sys
from dataclasses import dataclass
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
PROTO_ROOT = ROOT / "api" / "terminals"
REGISTRY = ROOT / "docs" / "protocol-extension-registry.md"

WATCH_NAMES = {
    "metadata",
    "attributes",
    "props",
    "args",
    "data",
    "kind",
    "action",
    "state",
    "type",
    "format",
}

PACKAGE_RE = re.compile(r"^\s*package\s+([A-Za-z0-9_.]+)\s*;")
MESSAGE_RE = re.compile(r"^\s*message\s+([A-Za-z_][A-Za-z0-9_]*)\s*\{")
ENUM_RE = re.compile(r"^\s*enum\s+[A-Za-z_][A-Za-z0-9_]*\s*\{")
FIELD_RE = re.compile(
    r"^\s*(?:optional\s+)?(?P<type>map\s*<[^>]+>|[.A-Za-z_][.A-Za-z0-9_]*)\s+"
    r"(?P<name>[A-Za-z_][A-Za-z0-9_]*)\s*=\s*(?P<number>\d+)\b"
)


@dataclass(frozen=True)
class Finding:
    path: Path
    line: int
    field: str
    reason: str
    source: str


def strip_inline_comment(line: str) -> str:
    return line.split("//", 1)[0]


def brace_delta(line: str) -> int:
    stripped = strip_inline_comment(line)
    return stripped.count("{") - stripped.count("}")


def scan_proto(path: Path) -> list[Finding]:
    package = ""
    findings: list[Finding] = []
    contexts: list[dict[str, object]] = []

    for line_no, line in enumerate(path.read_text(encoding="utf-8").splitlines(), start=1):
        package_match = PACKAGE_RE.match(line)
        if package_match:
            package = package_match.group(1)

        message_match = MESSAGE_RE.match(line)
        enum_match = ENUM_RE.match(line)
        if message_match:
            contexts.append(
                {
                    "kind": "message",
                    "name": message_match.group(1),
                    "depth": brace_delta(line),
                }
            )
            continue
        if enum_match:
            contexts.append({"kind": "enum", "name": "", "depth": brace_delta(line)})
            continue

        if contexts and contexts[-1]["kind"] == "message":
            field_match = FIELD_RE.match(strip_inline_comment(line))
            if field_match and package:
                field_type = re.sub(r"\s+", " ", field_match.group("type").strip())
                field_name = field_match.group("name")
                reason = flexible_reason(field_type, field_name)
                if reason:
                    message_name = ".".join(
                        str(context["name"])
                        for context in contexts
                        if context["kind"] == "message"
                    )
                    findings.append(
                        Finding(
                            path=path,
                            line=line_no,
                            field=f"{package}.{message_name}.{field_name}",
                            reason=reason,
                            source=line.strip(),
                        )
                    )

        delta = brace_delta(line)
        while delta < 0 and contexts:
            context = contexts[-1]
            context["depth"] = int(context["depth"]) + delta
            if int(context["depth"]) <= 0:
                contexts.pop()
                delta = int(context["depth"])
            else:
                delta = 0
        if delta > 0 and contexts:
            contexts[-1]["depth"] = int(contexts[-1]["depth"]) + delta
        while contexts and int(contexts[-1]["depth"]) <= 0:
            contexts.pop()

    return findings


def flexible_reason(field_type: str, field_name: str) -> str:
    reasons: list[str] = []
    if field_type.startswith("map<"):
        reasons.append("map field")
    if field_name.endswith("_json"):
        reasons.append("JSON payload field")
    if field_name in WATCH_NAMES:
        reasons.append(f"watched field name {field_name!r}")
    return ", ".join(reasons)


def registry_fields() -> set[str]:
    if not REGISTRY.exists():
        return set()
    content = REGISTRY.read_text(encoding="utf-8")
    return set(re.findall(r"Field:\s+([A-Za-z0-9_.]+)", content))


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--enforce",
        action="store_true",
        help="exit non-zero when detected flexible fields are missing registry entries",
    )
    args = parser.parse_args()

    registered = registry_fields()
    findings: list[Finding] = []
    for proto in sorted(PROTO_ROOT.glob("**/*.proto")):
        findings.extend(scan_proto(proto))

    missing = [finding for finding in findings if finding.field not in registered]

    if missing:
        mode = "ERROR" if args.enforce else "ADVISORY"
        print(f"proto-flex-check: {mode}: {len(missing)} flexible field(s) missing registry entries")
        for finding in missing:
            rel = finding.path.relative_to(ROOT)
            print(f"{rel}:{finding.line}: {finding.field}: {finding.reason}")
            print(f"  {finding.source}")
            print(f"  add to {REGISTRY.relative_to(ROOT)} as: Field: {finding.field}")
        return 1 if args.enforce else 0

    print(f"proto-flex-check: {len(findings)} flexible field(s) detected; all are registered")
    return 0


if __name__ == "__main__":
    sys.exit(main())
