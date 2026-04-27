# Capability Lifecycle

This document defines the durable capability-lifecycle contract between generic
terminal clients and the server.

Capability state is first-class runtime data. Clients publish full capability
state on connect and incremental updates as local resources change. The server
treats accepted capability generations as the source of truth for placement,
claims, routing, and scenario reactions.

## Control-Plane Contract

Control-plane messages are defined in `api/terminals/control/v1/control.proto`.

- Client to server: `Hello`, `CapabilitySnapshot`, `CapabilityDelta`
- Server to client: `HelloAck`, `CapabilityAck`, orchestration messages

`CapabilityAck` includes:

- `accepted_generation`: latest accepted capability generation
- `snapshot_applied`: whether the accepted update was a snapshot rebaseline

## Handshake and Ordering Rules

Connection bootstrap:

1. Client opens the control stream and sends `Hello`.
2. Client sends `CapabilitySnapshot` baseline for current capabilities.
3. Server validates and applies the snapshot.
4. Server responds with `HelloAck` and `CapabilityAck`.

Generation ordering:

- Generations are monotonic per device connection.
- Stale snapshot/delta generations are rejected.
- On generation skew or protocol-violation errors, clients rebaseline with a
  fresh snapshot.

## Capability and Resource Model

Capabilities are reported as typed records and compiled server-side into
claimable/routable resources.

Current runtime behavior includes:

- Endpoint-scoped audio/video/display resources
- Availability-aware resource compilation
- Dynamic display geometry metadata (size, density, orientation, safe area)
- Runtime capability loss/regain handling with scoped side effects

## Server Runtime Behavior

Primary server apply path:

- `terminal_server/internal/transport/control_stream.go`
- `terminal_server/internal/device/manager.go`

Expected behavior:

- Snapshot applies replace capability state atomically.
- Delta applies update state at newer generations only.
- Lost resources revoke only impacted claims (not whole activations).
- Route teardown is scoped to affected stream kinds/resources.
- Rebaseline and regain paths restore suspended claims when resources return.

## Client Runtime Behavior

Primary client runtime path:

- `terminal_client/lib/main.dart`
- `terminal_client/lib/capabilities/probe.dart`
- `terminal_client/lib/connection/control_client.dart`

Expected behavior:

- Emit `CapabilitySnapshot` baseline on connect/bootstrap.
- Emit `CapabilityDelta` on meaningful runtime changes.
- Coalesce noisy display changes using debounce windows.
- Rebaseline with snapshot (not delta) when stale generation is reported.

## Capability Lifecycle Events

Server emits typed capability/runtime events, including:

- `terminal.capability.added`
- `terminal.capability.updated`
- `terminal.capability.removed`
- `terminal.display.resized`
- `terminal.audio_route.changed`
- `terminal.resource.lost`
- `terminal.resource.lost:<resource-id>`

These events are emitted from transport capability-apply paths and consumed by
runtime orchestration logic.

## Validation Evidence

Representative regression coverage:

- `terminal_server/internal/transport/control_stream_test.go`
- `terminal_server/internal/device/manager_test.go`
- `terminal_server/internal/transport/generated_proto_adapter_test.go`
- `terminal_client/test/widget_test.dart`

Repository gates:

```bash
make all-check
```

No use-case ID is currently mapped specifically to capability-lifecycle work;
validation remains covered by repository-wide gates and targeted transport/
client tests.

## Related References

- `docs/client-architecture.md`
- `docs/server.md`
- `docs/event-taxonomy.md`
- `plans/features/protocol.md`
- `plans/features/io-abstraction.md`