---
title: "Phase 4 — Intercom and Calls"
kind: plan
status: planned
owner: unowned
validation: none
last-reviewed: 2026-04-25
---

# Phase 4 — Intercom and Calls

See [masterplan.md](../archive/masterplan-duplicate.md) for overall system context.

Build on media streams for communication scenarios.

## Prerequisites

- [phase-3-media.md](phase-3-media.md) complete — WebRTC media plane and IO router work end-to-end.

## Deliverables

- [x] **Intercom scenario**: Voice-activated or button-activated two-way audio between devices. See [use-case-flows.md](../features/use-case-flows.md#intercom). Target form: a `MediaPlan` with two mic→speaker edges.
- [x] **Whole-house announcement**: One-to-many audio broadcast. Target form: a fork node.
- [x] **PA system scenario**: One mic → all speakers with feedback suppression and PA overlay UI. See [use-case-flows.md](../features/use-case-flows.md#pa-system). Target form: claims only `speaker.main` and `screen.overlay`, leaving `screen.main` untouched — the claim-driven-preemption item below tracks that refinement.
- [x] **Audio mixer**: Server-side mixing of multiple audio streams into a single output track. Target form: a mixer node in the media plan.
- [x] **Multi-window scenario**: Grid UI of all camera feeds on one device with mixed or selectable audio. See [use-case-flows.md](../features/use-case-flows.md#multi-window-security-camera--multi-feed-view). Target form: cameras[*] → compositor → display, mics[*] → mixer → speaker.
- [x] **Internal video call**: Client-to-client video call through the server SFU; each call its own activation. See [use-case-flows.md](../features/use-case-flows.md#audio-and-video-calls).
- [x] **SIP integration**: Register with a SIP provider for external phone calls.
- [x] **WebRTC-SIP bridge**: Bridge internal WebRTC streams to external SIP calls.
- [x] **Claim-driven preemption**: Extend the claim manager to handle cross-activation preemption — PA claims `speaker.main` on receiving devices without evicting their main-screen scenarios; receiving activations keep running with their audio parked. See [io-abstraction.md](../features/io-abstraction.md#resource-claims) and [scenario-engine.md](../features/scenario-engine.md#resource-claims-and-preemption).

## Milestone

Intercom between rooms. Place a phone call from any client.

## Related Plans

- [scenario-engine.md](../features/scenario-engine.md) — Priority and preemption rules these scenarios live under.
- [io-abstraction.md](../features/io-abstraction.md) — Fork, forward, mix primitives.
- [server-driven-ui.md](../features/server-driven-ui.md) — Call, intercom, and PA UIs composed from primitives.
- [phase-5-voice.md](phase-5-voice.md) — Next phase.
