#!/usr/bin/env python3
"""Detect and stop stuck Gradle/JVM Android test workers for this repo."""

from __future__ import annotations

import argparse
import json
import os
import signal
import subprocess
import sys
import time
from dataclasses import dataclass
from pathlib import Path


DISCOVERY_NEEDLES = (
    "Gradle Test Executor",
    "testDebugUnitTest",
)
PROCESS_KEYWORDS = ("java", "gradle", "kotlin")
IDE_KEYWORDS = ("android studio", "idea", "intellij", "code ", "cursor", "eclipse")


@dataclass
class Proc:
    pid: int
    ppid: int
    cpu: float
    etime: str
    elapsed_sec: int
    command: str


def run(cmd: "list[str]", cwd: "str | None" = None) -> "subprocess.CompletedProcess[str]":
    return subprocess.run(cmd, cwd=cwd, text=True, capture_output=True, check=False)


def etime_to_seconds(etime: str) -> int:
    etime = etime.strip()
    if not etime:
        return 0
    days = 0
    time_part = etime
    if "-" in etime:
        maybe_days, time_part = etime.split("-", 1)
        try:
            days = int(maybe_days)
        except ValueError:
            days = 0
    parts = time_part.split(":")
    try:
        if len(parts) == 3:
            hours, minutes, seconds = map(int, parts)
        elif len(parts) == 2:
            hours = 0
            minutes, seconds = map(int, parts)
        elif len(parts) == 1:
            hours = 0
            minutes = 0
            seconds = int(parts[0])
        else:
            return 0
    except ValueError:
        return 0
    return days * 86400 + hours * 3600 + minutes * 60 + seconds


def list_processes() -> list[Proc]:
    ps = run(["ps", "-axo", "pid=,ppid=,%cpu=,etime=,args="])
    if ps.returncode != 0:
        return []
    out: list[Proc] = []
    for line in ps.stdout.splitlines():
        parts = line.rstrip().split(None, 4)
        if len(parts) < 5:
            continue
        pid_s, ppid_s, cpu_s, etime_s, cmd = parts
        try:
            out.append(
                Proc(
                    pid=int(pid_s),
                    ppid=int(ppid_s),
                    cpu=float(cpu_s),
                    etime=etime_s,
                    elapsed_sec=etime_to_seconds(etime_s),
                    command=cmd,
                )
            )
        except ValueError:
            continue
    return out


def compact(p: Proc) -> dict[str, object]:
    return {
        "pid": p.pid,
        "ppid": p.ppid,
        "cpu": round(p.cpu, 1),
        "etime": p.etime,
        "command": p.command[:220],
    }


def alive(pid: int) -> bool:
    try:
        os.kill(pid, 0)
        return True
    except ProcessLookupError:
        return False
    except PermissionError:
        return True


def explicit_gradle_worker_or_daemon(command: str) -> bool:
    return "Gradle Test Executor" in command or "org.gradle.launcher.daemon.bootstrap.GradleDaemon" in command


def classify(
    processes: list[Proc],
    repo_root: str,
    android_root: str,
    cpu_threshold: float,
    elapsed_threshold_sec: int,
) -> tuple[list[Proc], list[Proc], list[Proc], list[dict[str, object]]]:
    candidates: list[Proc] = []
    safe: list[Proc] = []
    excluded: list[dict[str, object]] = []
    for proc in processes:
        cmd_l = proc.command.lower()
        if not any(k in cmd_l for k in PROCESS_KEYWORDS):
            continue
        if not (
            android_root in proc.command
            or any(needle.lower() in cmd_l for needle in DISCOVERY_NEEDLES)
        ):
            continue
        candidates.append(proc)

        if repo_root not in proc.command and android_root not in proc.command:
            excluded.append({"pid": proc.pid, "reason": "no_repo_path"})
            continue
        if any(ide in cmd_l for ide in IDE_KEYWORDS) and not explicit_gradle_worker_or_daemon(proc.command):
            excluded.append({"pid": proc.pid, "reason": "ide_excluded"})
            continue
        safe.append(proc)

    stuck = [
        proc
        for proc in safe
        if proc.elapsed_sec >= elapsed_threshold_sec and proc.cpu >= cpu_threshold
    ]
    return candidates, safe, stuck, excluded


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo-root", default=str(Path(__file__).resolve().parents[1]))
    parser.add_argument("--android-root", default="")
    parser.add_argument("--java-home", default="")
    parser.add_argument("--cpu-threshold", type=float, default=70.0)
    parser.add_argument("--elapsed-minutes", type=int, default=10)
    parser.add_argument("--settle-seconds", type=int, default=5)
    args = parser.parse_args()

    repo_root = os.path.abspath(args.repo_root)
    android_root = os.path.abspath(args.android_root or os.path.join(repo_root, "android_client"))
    elapsed_threshold_sec = args.elapsed_minutes * 60
    settle = max(1, args.settle_seconds)

    report: dict[str, object] = {
        "repo": repo_root,
        "thresholds": {
            "elapsed_sec_gte": elapsed_threshold_sec,
            "cpu_gte": args.cpu_threshold,
        },
        "actions": [],
    }

    initial = list_processes()
    candidates0, safe0, stuck0, excluded0 = classify(
        initial, repo_root, android_root, args.cpu_threshold, elapsed_threshold_sec
    )
    report["initial"] = {
        "candidates_found": [compact(p) for p in candidates0],
        "safe_candidates": [compact(p) for p in safe0],
        "stuck_pids": [p.pid for p in stuck0],
        "excluded": excluded0,
    }

    if os.path.isdir(android_root):
        gradlew_cmd = "./gradlew --stop"
        env_prefix = f'JAVA_HOME="{args.java_home}" ' if args.java_home else ""
        stop = run(["bash", "-lc", f"{env_prefix}{gradlew_cmd}"], cwd=android_root)
        report["actions"].append(
            {
                "step": "gradlew_stop",
                "cwd": android_root,
                "exit_code": stop.returncode,
                "stdout_tail": "\n".join(stop.stdout.splitlines()[-8:]),
                "stderr_tail": "\n".join(stop.stderr.splitlines()[-8:]),
            }
        )
    else:
        report["actions"].append(
            {"step": "gradlew_stop", "cwd": android_root, "exit_code": 1, "error": "android_root_missing"}
        )

    time.sleep(settle)
    _, safe1, stuck1, _ = classify(
        list_processes(), repo_root, android_root, args.cpu_threshold, elapsed_threshold_sec
    )
    report["after_gradlew_stop"] = {
        "stuck_pids": [p.pid for p in stuck1],
        "matching_processes": [compact(p) for p in safe1],
    }

    term_sent: list[int] = []
    for proc in stuck1:
        if alive(proc.pid):
            try:
                os.kill(proc.pid, signal.SIGTERM)
                term_sent.append(proc.pid)
            except ProcessLookupError:
                pass
    report["actions"].append({"step": "sigterm", "pids": term_sent})

    time.sleep(settle)
    _, safe2, stuck2, _ = classify(
        list_processes(), repo_root, android_root, args.cpu_threshold, elapsed_threshold_sec
    )
    report["after_sigterm"] = {
        "stuck_pids": [p.pid for p in stuck2],
        "matching_processes": [compact(p) for p in safe2],
    }

    kill_sent: list[int] = []
    for proc in stuck2:
        if alive(proc.pid):
            try:
                os.kill(proc.pid, signal.SIGKILL)
                kill_sent.append(proc.pid)
            except ProcessLookupError:
                pass
    report["actions"].append({"step": "sigkill", "pids": kill_sent})

    time.sleep(1)
    _, safe_final, stuck_final, _ = classify(
        list_processes(), repo_root, android_root, args.cpu_threshold, elapsed_threshold_sec
    )
    report["final"] = {
        "stuck_pids": [p.pid for p in stuck_final],
        "remaining_matching_processes": [compact(p) for p in safe_final],
    }
    report["exit_code"] = 0 if not stuck_final else 2

    print(json.dumps(report, separators=(",", ":")))
    return int(report["exit_code"])


if __name__ == "__main__":
    raise SystemExit(main())
