#!/usr/bin/env python3
"""Helper for scripts/usecase-validate.sh — reads scripts/usecase-validate.yaml.

Usage:
  --info <ID>        Print "ID|Family|Description" for one ID
  --info all         Print all metadata rows
  --ids              Print all IDs space-separated
  --run <ID>         Execute test steps for ID; exit non-zero on failure
"""
import argparse
import os
import subprocess
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
YAML_PATH = SCRIPT_DIR / "usecase-validate.yaml"
ROOT_DIR = SCRIPT_DIR.parent
SERVER_DIR = ROOT_DIR / "terminal_server"
CLIENT_DIR = ROOT_DIR / "terminal_client"

# Ordered list of IDs matching the all_ids array in the original shell script.
ALL_IDS_ORDER = [
    "AB1", "AB2", "AB3", "AB4", "AB5", "AB6", "AB7",
    "AA1", "AA2", "AA3", "AA4", "AA5", "AA6",
    "B1", "B2", "B3", "B4", "B5",
    "C1", "C2", "C3", "C5",
    "I3", "I4", "I6",
    "D3", "D1", "D2",
    "M1", "M2", "M3", "M4", "M5",
    "P2", "P3", "P4",
    "S1", "S2", "S3",
    "P1",
    "PL1", "PL8", "PL20",
    "T1", "T2", "T3", "T4",
    "UI1", "UI2", "UI3", "UI4", "UI5", "UI6", "UI7", "UI8", "UI9", "UI10",
    "V1", "V2", "V3",
]


def load_yaml() -> dict[str, dict]:
    """Load the YAML file and return {id: entry} mapping."""
    text = YAML_PATH.read_text()
    try:
        import yaml
        data = yaml.safe_load(text)
        return {uc["id"]: uc for uc in data["usecases"]}
    except ImportError:
        data = _parse_yaml_fallback(text)
        return {uc["id"]: uc for uc in data["usecases"]}


def _parse_yaml_fallback(text: str) -> dict:
    """Minimal YAML parser sufficient for usecase-validate.yaml."""
    import re
    usecases = []
    current: dict | None = None
    current_steps: list | None = None
    current_step: dict | None = None
    indent_stack: list[int] = []

    for line in text.splitlines():
        stripped = line.rstrip()
        if not stripped or stripped.lstrip().startswith("#"):
            continue
        indent = len(stripped) - len(stripped.lstrip())
        content = stripped.lstrip()

        if content == "usecases:":
            continue

        if re.match(r"^- id:", content):
            if current_step and current_steps is not None:
                current_steps.append(current_step)
                current_step = None
            if current is not None and current_steps is not None:
                current["steps"] = current_steps
                usecases.append(current)
            current = {"id": content[len("- id:"):].strip().strip('"')}
            current_steps = []
            continue

        if content.startswith("steps:"):
            current_steps = []
            continue

        if content.startswith("- kind:"):
            if current_step is not None and current_steps is not None:
                current_steps.append(current_step)
            current_step = {"kind": content[len("- kind:"):].strip().strip('"')}
            continue

        m = re.match(r"^([a-z_]+):\s*(.*)", content)
        if not m:
            continue
        key, val = m.group(1), m.group(2).strip()
        if val.startswith('"') and val.endswith('"'):
            val = val[1:-1]
        elif val.startswith("'") and val.endswith("'"):
            val = val[1:-1]

        if current_step is not None and indent >= 6:
            current_step[key] = val
        elif current is not None:
            current[key] = val

    # Flush final entries
    if current_step is not None and current_steps is not None:
        current_steps.append(current_step)
    if current is not None and current_steps is not None:
        current["steps"] = current_steps
        usecases.append(current)

    return {"usecases": usecases}


def cmd_info(entries: dict[str, dict], id_arg: str) -> int:
    """Print ID|Family|Description for one ID or all."""
    if id_arg == "all":
        for uid in ALL_IDS_ORDER:
            if uid in entries:
                e = entries[uid]
                print(f"{e['id']}|{e['family']}|{e['description']}")
        return 0
    if id_arg not in entries:
        print(f"unsupported use case id: {id_arg}", file=sys.stderr)
        return 2
    e = entries[id_arg]
    print(f"{e['id']}|{e['family']}|{e['description']}")
    return 0


def cmd_ids(entries: dict[str, dict]) -> int:
    """Print all IDs space-separated in canonical order."""
    ids = [uid for uid in ALL_IDS_ORDER if uid in entries]
    print(" ".join(ids))
    return 0


def cmd_run(entries: dict[str, dict], id_arg: str) -> int:
    """Execute test steps for a use-case ID."""
    if id_arg not in entries:
        print(f"unsupported use case id: {id_arg}", file=sys.stderr)
        print("see docs/usecase-validation-matrix.md for supported IDs", file=sys.stderr)
        return 2

    # Build PATH with .bin and flutter/bin prepended.
    env = dict(os.environ)
    bin_dir = str(ROOT_DIR / ".bin")
    flutter_bin = str(ROOT_DIR / ".sdk" / "flutter" / "bin")
    env["PATH"] = bin_dir + os.pathsep + flutter_bin + os.pathsep + env.get("PATH", "")

    entry = entries[id_arg]
    for step in entry.get("steps", []):
        kind = step["kind"]
        if kind == "go_test":
            pkg = step["pkg"]
            run = step["run"]
            print(f"==> go test {pkg} -run {run}")
            sys.stdout.flush()
            result = subprocess.run(
                ["go", "test", pkg, "-run", run, "-count=1"],
                cwd=str(SERVER_DIR),
                env=env,
            )
            if result.returncode != 0:
                return result.returncode
        elif kind == "app_test":
            name = step["name"]
            print(f"==> go run ./cmd/term app test {name}")
            sys.stdout.flush()
            result = subprocess.run(
                ["go", "run", "./cmd/term", "app", "test", name],
                cwd=str(SERVER_DIR),
                env=env,
            )
            if result.returncode != 0:
                return result.returncode
        elif kind == "flutter_test":
            path = step["path"]
            plain_name = step["plain_name"]
            print(f"==> flutter test {path} --plain-name {plain_name}")
            sys.stdout.flush()
            result = subprocess.run(
                ["flutter", "test", path, "--plain-name", plain_name],
                cwd=str(CLIENT_DIR),
                env=env,
            )
            if result.returncode != 0:
                return result.returncode
        else:
            print(f"unknown step kind: {kind}", file=sys.stderr)
            return 2
    return 0


def main() -> int:
    ap = argparse.ArgumentParser(description=__doc__)
    group = ap.add_mutually_exclusive_group(required=True)
    group.add_argument("--info", metavar="ID", help="Print ID|Family|Description")
    group.add_argument("--ids", action="store_true", help="Print all IDs space-separated")
    group.add_argument("--run", metavar="ID", help="Execute test steps for ID")
    args = ap.parse_args()

    entries = load_yaml()

    if args.info is not None:
        return cmd_info(entries, args.info)
    if args.ids:
        return cmd_ids(entries)
    if args.run is not None:
        return cmd_run(entries, args.run)
    return 0


if __name__ == "__main__":
    sys.exit(main())
