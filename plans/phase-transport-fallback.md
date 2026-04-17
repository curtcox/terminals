# Phase X — Multi-Transport Connection Fallback
See [masterplan.md](../masterplan.md) for overall system context. Extend the control plane so a client can connect over gRPC, WebSocket, TCP, or HTTP and automatically use any carrier the server exposes and the network permits.

This phase explicitly ignores migration and backward-compatibility concerns. There are no existing users.

## Prerequisites
- [phase-1-foundation.md](phase-1-foundation.md) complete — typed control messages, capability reporting, and initial connection lifecycle exist.
- [phase-3-media.md](phase-3-media.md) complete — WebRTC signaling already rides on the control plane and must be preserved across carriers.

## Deliverables
- [ ] **Carrier-neutral session layer**: Introduce a shared session abstraction above concrete transports. All higher layers consume ordered inbound/outbound envelopes rather than gRPC-specific stream types. See [transport-multiplexing.md](transport-multiplexing.md).
- [ ] **Explicit transport envelope**: Add a protobuf `WireEnvelope` and transport handshake messages for non-gRPC carriers, including version negotiation, session identity, sequencing, and heartbeat payloads.
- [ ] **Server multi-listener support**: Add control-plane listeners for gRPC, WebSocket, TCP, and HTTP fallback under `internal/transport/`, with identical session semantics once connected.
- [ ] **Client carrier implementations**: Add `grpc`, `websocket`, `tcp`, and `http` carrier adapters in the Flutter client, each implementing one common connection interface.
- [ ] **Automatic carrier selection**: Client discovers or derives all candidate endpoints, filters them by local runtime support, orders them by priority, and attempts them until one succeeds.
- [ ] **Discovery metadata expansion**: mDNS advertisement includes transport availability and preferred order; manual-connect UI can derive default endpoints from a base host or accept explicit per-carrier overrides.
- [ ] **Carrier-independent reconnect/resume**: Device identity, heartbeat, disconnect detection, and state restoration work even when reconnect chooses a different carrier than the last successful session.
- [ ] **Failure diagnostics pipeline**: If all carriers fail, the client emits a structured local diagnostic record and reports the failure through any available local output resource — display first, then audio/indicator/haptic as supported.
- [ ] **HTTP correctness fallback**: Implement queued request/response semantics for restrictive environments where full-duplex streaming is unavailable, preserving ordered delivery even if latency is worse.
- [ ] **Transport-agnostic WebRTC signaling**: WebRTC setup messages continue to work regardless of which control carrier established the session.
- [ ] **Tests**: Add server and client tests covering carrier selection order, handshake failure paths, heartbeat/liveness, reconnect on carrier switch, and no-carriers-available diagnostics.

## Milestone
Take one client device and prove the following sequence without changing scenario code: connect over gRPC on an open network, connect over WebSocket when gRPC is blocked, connect over TCP when HTTP upgrade is blocked, connect over HTTP when only request/response works, and produce a visible local diagnostic report when all four are unavailable.

## Related Plans
- [transport-multiplexing.md](transport-multiplexing.md) — Design for the carrier-neutral control plane.
- [protocol.md](protocol.md) — Existing wire contract to revise around envelopes and carrier-neutral signaling.
- [discovery.md](discovery.md) — Discovery and manual connect rules to extend.
- [architecture-client.md](architecture-client.md) — Client module changes for carrier selection and reporting.
- [architecture-server.md](architecture-server.md) — Server module changes for multi-listener transport.
