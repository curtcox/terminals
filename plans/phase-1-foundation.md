# Phase 1 — Foundation

See [masterplan.md](../masterplan.md) for overall system context.

Establish the core client-server communication and prove the architecture.

## Prerequisites

- [phase-0-setup.md](phase-0-setup.md) complete — repo, tooling, and CI are in place.

## Deliverables

- [ ] **Proto definitions**: Define the gRPC protobuf schemas for control messages, capability declarations, and UI descriptors. See [protocol.md](protocol.md) and [server-driven-ui.md](server-driven-ui.md).
- [ ] **Buf codegen**: `buf generate` produces Go and Dart bindings; CI verifies generated code is up to date.
- [ ] **Server skeleton**: Go project with gRPC server, device manager, and mDNS advertisement. See [architecture-server.md](architecture-server.md) and [discovery.md](discovery.md).
- [ ] **Client skeleton**: Flutter app with mDNS discovery, manual connect fallback, gRPC connection, and capability reporting. See [architecture-client.md](architecture-client.md).
- [ ] **Server-driven UI**: Client renders basic UI descriptors from the server (text, buttons, layout). Server sends a "hello world" UI on connect. See [server-driven-ui.md](server-driven-ui.md).
- [ ] **Heartbeat and reconnection**: Connection health monitoring and automatic reconnection.
- [ ] **Tests from the start**: Unit tests for proto serialization, device registration, capability parsing. CI enforces passing tests and lint on every PR.

## Milestone

Client connects to server, sends capabilities, server sends a UI that the client renders. CI is green.

## Related Plans

- [protocol.md](protocol.md) — gRPC + WebRTC contract.
- [discovery.md](discovery.md) — mDNS + manual connect.
- [server-driven-ui.md](server-driven-ui.md) — UI primitive set.
- [architecture-client.md](architecture-client.md) — Client module layout.
- [architecture-server.md](architecture-server.md) — Server module layout.
- [phase-2-terminal.md](phase-2-terminal.md) — Next phase.
