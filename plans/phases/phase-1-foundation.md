---
title: "Phase 1 — Foundation"
kind: plan
status: superseded
owner: copilot
validation: none
last-reviewed: 2026-04-25
---

# Phase 1 — Foundation

Status: Completed and drained on 2026-04-25.

The durable behavior from this phase is documented in:

- [Protocol Design](../features/protocol.md)
- [Server-Driven UI](../features/server-driven-ui.md)
- [Client Architecture](../features/architecture-client.md)
- [Server Architecture](../features/architecture-server.md)
- [Discovery and Connection](../features/discovery.md)
- [IO Abstraction Layer](../features/io-abstraction.md)
- [Capability Lifecycle and Dynamic Terminal Capabilities](../features/capability-lifecycle.md)

See [masterplan.md](../archive/masterplan-duplicate.md) for overall system context.

Establish the core client-server communication and prove the architecture.

## Prerequisites

- [phase-0-setup.md](phase-0-setup.md) complete — repo, tooling, and CI are in place.

## Deliverables

- [x] **Proto definitions**: Define the gRPC protobuf schemas for control messages, capability lifecycle messages, and UI descriptors. The control plane must include an explicit handshake, an initial capability snapshot, and runtime capability delta messages. See [protocol.md](../features/protocol.md) and [server-driven-ui.md](../features/server-driven-ui.md).
- [x] **Buf codegen**: `buf generate` produces Go and Dart bindings; CI verifies generated code is up to date.
- [x] **Server skeleton**: Go project with gRPC server, device manager, mDNS advertisement, and an in-memory capability registry keyed by device and endpoint. See [architecture-server.md](../features/architecture-server.md) and [discovery.md](../features/discovery.md).
- [x] **Client skeleton**: Flutter app with mDNS discovery, manual connect fallback, gRPC connection, capability discovery, and capability reporting. On initial connect the client sends a full snapshot. See [architecture-client.md](../features/architecture-client.md).
- [x] **Runtime capability monitoring**: Client watches for capability changes that affect routing and UI composition: display resize/orientation changes, keyboard attach/detach, camera availability changes, microphone/speaker route changes, and permission changes. When state changes, the client emits explicit deltas. See [architecture-client.md](../features/architecture-client.md) and [capability-lifecycle.md](../features/capability-lifecycle.md).
- [x] **Capability apply path (server)**: Server accepts snapshots and deltas, updates the canonical terminal record, recompiles claimable resources from capabilities, and publishes device-state changes to interested subsystems. See [io-abstraction.md](../features/io-abstraction.md) and [capability-lifecycle.md](../features/capability-lifecycle.md).
- [x] **Server-driven UI**: Client renders basic UI descriptors from the server (text, buttons, layout). Server sends a "hello world" UI on connect and can choose descriptors based on the currently reported terminal shape.
- [x] **Heartbeat and reconnection**: Connection health monitoring and automatic reconnection. Reconnect performs a fresh handshake and a fresh full capability snapshot.
- [x] **Tests from the start**: Unit tests for proto serialization, snapshot/delta application, device registration, capability parsing, and capability-to-resource compilation. CI enforces passing tests and lint on every PR.

## Milestone

Client connects to server, sends a capability snapshot, sends capability deltas as local state changes, and renders a server-driven UI chosen from the current terminal capabilities. CI is green.

## Related Plans

- [protocol.md](../features/protocol.md) — gRPC + WebRTC contract.
- [capability-lifecycle.md](../features/capability-lifecycle.md) — Handshake, snapshots, deltas, and acknowledgements.
- [discovery.md](../features/discovery.md) — mDNS + manual connect.
- [server-driven-ui.md](../features/server-driven-ui.md) — UI primitive set.
- [architecture-client.md](../features/architecture-client.md) — Client module layout.
- [architecture-server.md](../features/architecture-server.md) — Server module layout.
- [io-abstraction.md](../features/io-abstraction.md) — Capability compilation into claimable resources.
- [phase-2-terminal.md](phase-2-terminal.md) — Next phase.
