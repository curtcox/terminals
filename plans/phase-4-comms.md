# Phase 4 — Intercom and Calls

See [masterplan.md](../masterplan.md) for overall system context.

Build on media streams for communication scenarios.

## Prerequisites

- [phase-3-media.md](phase-3-media.md) complete — WebRTC media plane and IO router work end-to-end.

## Deliverables

- [ ] **Intercom scenario**: Voice-activated or button-activated two-way audio between devices. See [use-case-flows.md](use-case-flows.md#intercom).
- [ ] **Whole-house announcement**: One-to-many audio broadcast.
- [ ] **PA system scenario**: One mic → all speakers with feedback suppression and PA overlay UI. See [use-case-flows.md](use-case-flows.md#pa-system).
- [ ] **Audio mixer**: Server-side mixing of multiple audio streams into a single output track.
- [ ] **Multi-window scenario**: Grid UI of all camera feeds on one device with mixed or selectable audio. See [use-case-flows.md](use-case-flows.md#multi-window-security-camera--multi-feed-view).
- [ ] **Internal video call**: Client-to-client video call through the server SFU. See [use-case-flows.md](use-case-flows.md#audio-and-video-calls).
- [ ] **SIP integration**: Register with a SIP provider for external phone calls.
- [ ] **WebRTC-SIP bridge**: Bridge internal WebRTC streams to external SIP calls.

## Milestone

Intercom between rooms. Place a phone call from any client.

## Related Plans

- [scenario-engine.md](scenario-engine.md) — Priority and preemption rules these scenarios live under.
- [io-abstraction.md](io-abstraction.md) — Fork, forward, mix primitives.
- [server-driven-ui.md](server-driven-ui.md) — Call, intercom, and PA UIs composed from primitives.
- [phase-5-voice.md](phase-5-voice.md) — Next phase.
