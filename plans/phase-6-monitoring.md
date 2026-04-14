# Phase 6 — Monitoring and Alerts

See [masterplan.md](../masterplan.md) for overall system context.

Ambient intelligence scenarios.

## Prerequisites

- [phase-5-voice.md](phase-5-voice.md) complete — STT/LLM/TTS pipelines exist for voice triggers and spoken notifications.

## Deliverables

- [ ] **Sound classification**: AI backend for detecting specific sounds (silence, beeps, alarms, etc.). See [technology.md](technology.md#ai-backend-pluggable).
- [ ] **Audio monitoring scenario**: "Tell me when X stops" voice command handling and monitoring. See [use-case-flows.md](use-case-flows.md#audio-monitoring-tell-me-when-the-dishwasher-stops).
- [ ] **Timer and reminder scenario**: Voice-commanded timers and reminders with scheduler persistence. See [use-case-flows.md](use-case-flows.md#timers-and-reminders).
- [ ] **Schedule monitoring scenario**: Time-triggered activity monitoring with escalating alerts. See [use-case-flows.md](use-case-flows.md#schedule-monitoring-watch-my-child).
- [ ] **Red alert scenario**: Broadcast preemption of all devices with alarm. See [use-case-flows.md](use-case-flows.md#red-alert).

## Milestone

"Tell me when the dishwasher stops" works end to end.

## Related Plans

- [scenario-engine.md](scenario-engine.md) — Priority levels and preemption (critical for red alert).
- [io-abstraction.md](io-abstraction.md) — Analyze primitive used by sound classification.
- [phase-7-polish.md](phase-7-polish.md) — Next phase.
