# Phase 3 — Media Streams

See [masterplan.md](../masterplan.md) for overall system context.

Enable audio and video streaming between clients and server.

## Prerequisites

- [phase-1-foundation.md](phase-1-foundation.md) complete — gRPC control stream and capability reporting exist.

## Deliverables

- [x] **WebRTC integration (server)**: Pion-based SFU — accept, forward, and process media streams. See [protocol.md](protocol.md#media-plane-webrtc).
- [x] **WebRTC integration (client)**: `flutter_webrtc` — send/receive audio and video.
- [x] **Signaling over gRPC**: SDP and ICE candidate exchange through the existing control channel (`WebRTCSignal` messages in [protocol.md](protocol.md)).
- [x] **IO Router (imperative)**: Initial `Connect`/`Disconnect`-style routing of media streams between devices. See [io-abstraction.md](io-abstraction.md). Superseded by the media planner below.
- [ ] **Media planner**: Scenarios declare a `MediaPlan` (node/edge graph); the router compiles it to `StartStream`/`StopStream`/`RouteStream` and WebRTC signaling. See [io-abstraction.md](io-abstraction.md#media-topology-plans-not-connects). Start with source → sink and fork nodes; mix, composite, analyze, and record land in later phases.
- [ ] **Claim manager (basics)**: Per-resource exclusive/shared claims so two activations can coexist on one device (e.g. overlay above main screen). See [io-abstraction.md](io-abstraction.md#resource-claims).
- [x] **Audio playback**: Server sends audio clips (TTS, alerts) to specific devices. Will migrate onto a one-node-pair media plan once the planner lands.

## Milestone

Stream audio from one client's mic to another client's speaker.

## Related Plans

- [protocol.md](protocol.md) — WebRTC signaling over gRPC.
- [io-abstraction.md](io-abstraction.md) — Router primitives activated here.
- [technology.md](technology.md) — Pion (server) and `flutter_webrtc` (client).
- [phase-4-comms.md](phase-4-comms.md) — Next phase.
