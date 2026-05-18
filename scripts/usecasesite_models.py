"""Data models for the use-case documentation site generator."""

from __future__ import annotations

from dataclasses import dataclass


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
    validation_depth: str
    validation_evidence: str

    @property
    def title(self) -> str:
        return self.action[:1].upper() + self.action[1:]


@dataclass(frozen=True)
class MediaAsset:
    label: str
    path: str
    kind: str
    source: str = ""
    rights_note: str = ""
    transcript: str = ""
    descriptor: dict | None = None


@dataclass(frozen=True)
class InteractionStep:
    kind: str
    summary: str
    terminal: str


@dataclass(frozen=True)
class Result:
    usecase_id: str
    run_id: str
    scenario_name: str
    timestamp_end: str
    pass_: bool
    failing_assertions: tuple[str, ...]
    interaction_trace: tuple[InteractionStep, ...]
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


@dataclass(frozen=True)
class ValidationLink:
    label: str
    path: str
