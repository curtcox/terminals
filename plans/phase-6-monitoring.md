# Phase 6 — Monitoring and Alerts

See [masterplan.md](../masterplan.md) for overall system context.

Ambient intelligence scenarios.

## Prerequisites

- [phase-5-voice.md](phase-5-voice.md) complete — STT/LLM/TTS pipelines exist for voice triggers and spoken notifications.

## Deliverables

- [x] **Sound classification**: AI backend for detecting specific sounds (silence, beeps, alarms, etc.). See [technology.md](technology.md#ai-backend-pluggable).
- [x] **Audio monitoring scenario**: "Tell me when X stops" voice command handling and monitoring. See [use-case-flows.md](use-case-flows.md#audio-monitoring-tell-me-when-the-dishwasher-stops). Target form: activation with a shared `mic.analyze` claim and a `mic → analyzer → event` media plan.
- [x] **Timer and reminder scenario**: Voice-commanded timers and reminders with scheduler persistence. See [use-case-flows.md](use-case-flows.md#timers-and-reminders). Target form: each timer is its own activation, persisted via `ActivationRecord` so the scheduler survives restarts.
- [x] **Schedule monitoring scenario**: Time-triggered activity monitoring with escalating alerts. See [use-case-flows.md](use-case-flows.md#schedule-monitoring-watch-my-child). Target form: activation targets a zone via placement (e.g. `DevicesInZone("alice_room")`).
- [x] **Red alert scenario**: Broadcast preemption of all devices with alarm. See [use-case-flows.md](use-case-flows.md#red-alert). Target form: critical-priority activation that claims every exclusive resource on every device; cascade trigger that suspends every lower-priority activation via the claim manager.
- [ ] **Placement engine**: Ship zone/role metadata on devices plus the `PlacementEngine` API (`Find`, `NearestWith`, `DevicesInZone`, `DevicesWithRole`). Ambient scenarios target "the kitchen" or "the child's room" by scope, not device ID. See [placement.md](placement.md).
- [ ] **Analyzer nodes emit events**: The media planner's analyzer nodes publish `Event{Kind: "sound.detected", ...}` onto the intent/event bus; scenarios subscribe rather than polling. See [io-abstraction.md](io-abstraction.md#router-responsibilities).

## Milestone

"Tell me when the dishwasher stops" works end to end.

## Related Plans

- [scenario-engine.md](scenario-engine.md) — Priority levels and preemption (critical for red alert).
- [io-abstraction.md](io-abstraction.md) — Analyze primitive used by sound classification.
- [phase-7-polish.md](phase-7-polish.md) — Next phase.
