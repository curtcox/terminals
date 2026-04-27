---
title: "Capability Lifecycle and Dynamic Terminal Capabilities"
kind: plan
status: building
owner: cascade
validation: none
last-reviewed: 2026-04-26
---

# Capability Lifecycle and Dynamic Terminal Capabilities

See [masterplan.md](../archive/masterplan-duplicate.md) for overall system context.

The system already assumes thin clients report capabilities and the server owns all behavior. This plan makes that contract explicit and dynamic: every terminal declares a typed capability manifest on initial connection, and publishes capability-change events whenever its usable IO surface changes.

This plan deliberately ignores migration and backward-compatibility concerns. There are no existing users.

## Goals

1. Make capability disclosure a first-class protocol concept rather than an informal registration detail.
2. Support terminals whose capabilities can change at runtime without reconnecting.
3. Let the server reason about both **what a terminal can ever do** and **what it can do right now**.
4. Preserve the thin-client rule: capability interpretation, placement, routing, and scenario behavior remain server-side.

## Non-Goals

1. No scenario-specific client behavior.
2. No policy for server-side placement decisions beyond exposing the data needed to make them.
3. No backward-compatible wire format. The protocol can change cleanly.

## Design Principle

A terminal is not a static device profile. It is a live collection of resources whose availability, geometry, and quality may change over time.

Examples:

- A USB keyboard is attached or removed.
- A Bluetooth headset becomes the active speaker and microphone.
- A camera permission is granted or revoked.
- A foldable or resizable display changes usable size.
- A browser tab moves between monitors with different pixel ratios.
- An external display appears or disappears.

The protocol must model these as explicit capability state transitions, not implicit side effects.

## Capability Model

Split terminal description into three layers.

### 1. Terminal Identity

Stable identity for the connected client instance:

- `terminal_id`
- `device_id`
- `connection_id`
- human-readable name
- platform / runtime
- client build metadata

This identifies **who** is connected, not what it can do.

### 2. Capability Inventory

A structured inventory of device functions and resources the client can expose.

Capability families include:

- display
- keyboard
- pointer
- touch
- speaker
- microphone
- camera
- haptic
- battery / power
- network observations
- sensors
- removable / attachable peripherals

Each capability record should include:

- stable `capability_id`
- `kind`
- `status`
- optional `display_name`
- static properties
- dynamic properties
- exposed resource kinds

`status` values:

- `ENABLED` — usable now
- `DISABLED` — present but intentionally disabled
- `UNAVAILABLE` — known but currently unusable
- `REMOVED` — no longer present

### 3. Resource Surface

Capabilities are descriptive. Resources are what the server can claim and route.

Examples:

- a display capability exposes `screen.main` and `screen.overlay`
- a speaker capability exposes `speaker.main`
- a microphone capability exposes `mic.capture` and possibly `mic.analyze`
- a camera capability exposes `camera.capture` and possibly `camera.analyze`
- a keyboard capability exposes `keyboard.primary`

The client declares which resource kinds each capability contributes. The server compiles those into the device registry and claim manager state.

## Static vs Dynamic Properties

Each capability has static and dynamic properties.

### Static Properties

Rarely change during a connection.

Examples:

- keyboard physical vs virtual
- camera facing mode
- microphone channel count
- speaker channel layout
- sensor type

### Dynamic Properties

May change at runtime and must be updateable.

Examples:

- display pixel size, logical size, orientation, scale factor, safe insets
- active input focus support
- battery level / charging state
- active audio route
- camera availability due to permission or hardware contention
- current sample-rate options exposed by the platform

The server should treat dynamic-property changes as ordinary capability updates.

## Protocol Changes

## Control Stream

Replace the current one-shot capability registration shape with an explicit lifecycle.

```protobuf
service TerminalControl {
  rpc Connect(stream ClientMessage) returns (stream ServerMessage);
}

message ClientMessage {
  oneof payload {
    Hello hello = 1;
    CapabilitySnapshot capability_snapshot = 2;
    CapabilityDelta capability_delta = 3;
    InputEvent input = 4;
    SensorData sensor = 5;
    StreamReady stream_ready = 6;
    CommandAck ack = 7;
    Heartbeat heartbeat = 8;
  }
}

message ServerMessage {
  oneof payload {
    HelloAck hello_ack = 1;
    CapabilityAck capability_ack = 2;
    SetUI set_ui = 3;
    StartStream start_stream = 4;
    StopStream stop_stream = 5;
    RouteStream route_stream = 6;
    Notification notification = 7;
    WebRTCSignal webrtc_signal = 8;
    CommandRequest command = 9;
    Heartbeat heartbeat = 10;
  }
}
```

## Handshake Sequence

Initial connection sequence:

1. Client opens `Connect` stream.
2. Client sends `Hello` with terminal identity and coarse platform metadata.
3. Client sends full `CapabilitySnapshot` representing all currently known capabilities.
4. Server validates and stores the snapshot.
5. Server responds with `HelloAck` and `CapabilityAck`.
6. Server begins normal command / UI / media orchestration.

A capability snapshot is mandatory on connection. The server should not assign work until it has the initial snapshot.

## CapabilitySnapshot

`CapabilitySnapshot` is a full replacement view of the terminal's current capability state.

Use it:

- immediately after connect
- after reconnect
- when the client detects desynchronization and wants to re-baseline

```protobuf
message CapabilitySnapshot {
  string terminal_id = 1;
  uint64 generation = 2;
  repeated Capability capabilities = 3;
}
```

`generation` is monotonically increasing within a connection. The server uses it to reject stale updates.

## CapabilityDelta

`CapabilityDelta` is an incremental update for runtime changes.

Use it for:

- add capability
- update capability properties
- change capability status
- remove capability

```protobuf
message CapabilityDelta {
  string terminal_id = 1;
  uint64 generation = 2;
  repeated CapabilityChange changes = 3;
}

message CapabilityChange {
  oneof op {
    CapabilityAdded added = 1;
    CapabilityUpdated updated = 2;
    CapabilityStatusChanged status_changed = 3;
    CapabilityRemoved removed = 4;
  }
}
```

The client may batch several related changes into one delta, such as a monitor hot-plug that removes one display capability, adds another, and updates pointer geometry.

## Capability Shape

```protobuf
message Capability {
  string capability_id = 1;
  CapabilityKind kind = 2;
  CapabilityStatus status = 3;
  string display_name = 4;
  map<string, string> labels = 5;
  repeated ResourceDescriptor resources = 6;

  oneof details {
    DisplayCapability display = 20;
    KeyboardCapability keyboard = 21;
    PointerCapability pointer = 22;
    TouchCapability touch = 23;
    SpeakerCapability speaker = 24;
    MicrophoneCapability microphone = 25;
    CameraCapability camera = 26;
    SensorCapability sensor = 27;
    HapticCapability haptic = 28;
    BatteryCapability battery = 29;
  }
}
```

The exact detail messages should stay strongly typed. Avoid opaque JSON blobs.

## Display Capability

Display handling should move from a single implicit `screen` object to explicit display capabilities.

Each display capability should expose at least:

- logical width and height
- physical pixel width and height
- device pixel ratio
- orientation
- safe-area insets
- touch support on that display
- fullscreen support
- multi-window support
- refresh-rate hint if available

A display resize or orientation change is a capability update, not a generic input event.

## Audio and Video Devices

Audio and video devices should be individually addressable capabilities.

Examples:

- built-in speakers
- HDMI display speakers
- Bluetooth headset speaker
- built-in microphone
- USB microphone
- front camera
- rear camera
- external USB camera

This lets the server reason about routeable media endpoints instead of assuming one mic / one speaker / one camera per client.

## Capability Acknowledgement

The server should acknowledge accepted generations.

```protobuf
message CapabilityAck {
  string terminal_id = 1;
  uint64 accepted_generation = 2;
}
```

This gives the client a synchronization point. If an ack is missing or lagging, the client can resend a fresh snapshot.

## Server-Side Model

## Device Registry

The device registry should store:

- terminal identity
- latest accepted capability generation
- full capability inventory
- derived resource set
- timestamps for last snapshot and last delta

The registry becomes the source of truth for both placement and claim compilation.

## Capability Compiler

Add a server-side compiler that turns capability records into routable resources.

Responsibilities:

- validate capability schema
- derive resource kinds from capabilities
- assign stable resource IDs scoped to terminal + capability
- emit add/update/remove events to claim manager and IO router

This keeps protobuf structure separate from runtime scheduling structures.

## Claim Manager Integration

A capability change can invalidate active claims.

Examples:

- active speaker removed
- display resized below the minimum needed for a UI
- camera disappears during monitoring

Required behavior:

1. Detect which active claims reference removed or disabled resources.
2. Revoke those claims.
3. Notify the owning activation.
4. Let the scenario engine re-place, degrade, suspend, or terminate based on policy.

The claim manager must treat capability loss as a first-class source of preemption-like state change.

## IO Router Integration

The IO router should subscribe to resource-surface changes.

When a capability delta changes the routeable graph, the router must:

- patch affected media plans when possible
- tear down edges that target vanished resources
- emit typed events when plans become degraded or broken

This is essential for hot-plug audio/video devices and for display geometry changes.

## Scenario Engine Semantics

Capability changes should surface as typed events on the common event bus.

Examples:

- `terminal.capability.added`
- `terminal.capability.updated`
- `terminal.capability.removed`
- `terminal.resource.lost`
- `terminal.display.resized`
- `terminal.audio_route.changed`

The scenario engine can then react uniformly:

- move a photo frame to another display
- re-render UI for a new size class
- switch a call from built-in audio to headset audio
- suspend a camera-dependent scenario when all cameras disappear

## Client Responsibilities

The client must include a capability-monitoring layer separate from scenario execution.

Suggested module additions:

```text
terminal_client/
└── lib/
    ├── capabilities/
    │   ├── capability_registry.dart
    │   ├── capability_snapshot_builder.dart
    │   ├── capability_change_detector.dart
    │   ├── capability_publisher.dart
    │   ├── display_capability.dart
    │   ├── audio_device_capability.dart
    │   ├── video_device_capability.dart
    │   └── peripheral_capability.dart
```

Responsibilities:

- enumerate capabilities on startup
- subscribe to OS / platform events for hot-plug and geometry change
- coalesce rapid bursts of change into batched deltas
- maintain a local generation counter
- resend a full snapshot when synchronization is uncertain

## Change Detection Rules

The client should publish capability changes when any of the following occur:

- an IO device is added
- an IO device is removed
- an IO device becomes enabled, disabled, or unavailable
- a display's size, orientation, scale, or safe area changes
- permissions change in a way that affects availability
- the active media route changes and alters effective speaker / microphone capabilities

The client should not publish meaningless churn. Transient platform noise should be debounced and coalesced.

## Failure Handling

Rules:

1. If the server rejects or ignores a delta because of generation skew, the client sends a full snapshot.
2. If the client reconnects, it always starts with `Hello` plus `CapabilitySnapshot`.
3. If a capability disappears mid-stream, the client stops producing the affected stream immediately and separately reports the capability delta.

## Testing

## Proto Tests

- snapshot and delta serialization
- generation ordering
- add / update / remove semantics
- typed capability round-trips

## Client Tests

- display resize emits update
- headset hot-plug emits add/remove or status change
- permission revoke marks affected capabilities unavailable
- bursty platform events coalesce into a single delta batch

## Server Tests

- registry replaces snapshot atomically
- stale deltas are rejected
- resource compiler emits correct claimable resources
- active claims are revoked on resource loss
- router patches or tears down media plans after capability changes

## Incremental Progress

- 2026-04-26 (Stage 1 stale-generation client rebaseline coverage): Added Flutter widget regression coverage in `terminal_client/test/widget_test.dart` (`stale capability generation error triggers forced capability snapshot rebaseline`) to lock in that protocol-violation stale-generation errors trigger a forced `CapabilitySnapshot` rebaseline (fresh generation) instead of emitting a `CapabilityDelta`.
- 2026-04-26 (Stage 1 speaker endpoint availability route-teardown parity): Added transport regression coverage in `terminal_server/internal/transport/control_stream_test.go` (`TestHandleMessageCapabilityDeltaStopsAudioRouteOnSpeakerEndpointAvailabilityLoss`) to lock in that explicit speaker endpoint availability loss (`speakers.endpoint.<index>.available=false`) tears down affected audio routes while preserving unrelated video routes.
- 2026-04-26 (Stage 1 audio endpoint availability truthfulness): Added transport regression coverage in `terminal_server/internal/transport/control_stream_test.go` (`TestHandleMessageCapabilityDeltaStopsAudioRouteOnEndpointAvailabilityLoss`) to lock in that explicit microphone endpoint availability loss (`microphone.endpoint.<index>.available=false`) tears down affected audio routes while preserving unrelated video routes. Extended `TestCapabilityResourcesCompilesEndpointScopedResources` assertions to ensure unavailable audio endpoints do not compile endpoint-scoped resources.
- 2026-04-26 (Stage 1 endpoint availability route-teardown parity): Added transport regression coverage in `terminal_server/internal/transport/control_stream_test.go` (`TestHandleMessageCapabilityDeltaStopsVideoRouteOnEndpointAvailabilityLoss`) to lock in that explicit endpoint availability loss (`camera.endpoint.<index>.available=false`) tears down affected video routes while preserving unrelated audio routes.
- 2026-04-26 (Stage 1 no-op rebaseline snapshot event suppression): Added transport regression coverage in `terminal_server/internal/transport/control_stream_test.go` (`TestHandleMessageCapabilitySnapshotNoOpRebaselineDoesNotEmitCapabilityEvents`) to lock in that generation-advancing capability snapshots with unchanged capability maps do not emit capability lifecycle events.
- 2026-04-26 (Stage 1 no-op lifecycle event suppression): Hardened `terminal_server/internal/transport/control_stream.go` so no-op capability updates (new generation with unchanged capability map) do not emit typed capability lifecycle events. Added regression coverage in `terminal_server/internal/transport/control_stream_test.go` (`TestHandleMessageCapabilityDeltaNoOpDoesNotEmitCapabilityEvents`) to lock in quiet behavior for no-change deltas.
- 2026-04-26 (Stage 1 initial snapshot event semantics): Updated `terminal_server/internal/transport/control_stream.go` so the first capability snapshot baseline (`generation=1` from empty pre-snapshot state) does not emit capability-change side effects/events, while rebaseline snapshots continue emitting typed lifecycle events. Added transport regression coverage in `terminal_server/internal/transport/control_stream_test.go` (`TestHandleMessageCapabilitySnapshotInitialBaselineDoesNotEmitCapabilityEvents`, `TestHandleMessageCapabilitySnapshotRebaselineEmitsTypedCapabilityEvents`).
- 2026-04-26 (Stage 1 snapshot rebaseline route-teardown parity): Added transport regression coverage in `terminal_server/internal/transport/control_stream_test.go` (`TestHandleMessageCapabilitySnapshotStopsOnlyAffectedRoutesOnRebaselineLoss`) to enforce that capability snapshot rebaseline loss tears down only affected route kinds (microphone loss stops audio route while preserving unrelated video route).
- 2026-04-26 (Stage 1 endpoint-scoped route teardown hardening): Updated transport capability-loss route teardown matching in `terminal_server/internal/transport/control_stream.go` so endpoint-scoped resource removals (`audio_in.*`, `audio_out.*`, `camera.<id>.*`, `display.<id>.*`) are treated as affected media capabilities and disconnect only corresponding audio/video route kinds for the impacted device. Added regression coverage in `terminal_server/internal/transport/control_stream_test.go` (`TestHandleMessageCapabilityDeltaStopsAudioRouteOnEndpointLoss`) to preserve unrelated video routes during microphone endpoint swaps.
- 2026-04-26 (Stage 1 partial-regain claim restoration hardening): Added transport regression coverage in `terminal_server/internal/transport/control_stream_test.go` (`TestHandleMessageCapabilityDeltaRestoresOnlyMatchingSuspendedClaimsOnPartialRegain`) to ensure capability regain restores only matching suspended claims, preserves unrelated suspended claims across partial resource return, and fully drains suspension state when all resources are re-added.
- 2026-04-26 (Stage 1 scoped route teardown): Tightened `terminal_server/internal/transport/control_stream.go` capability-loss disconnect logic so route teardown is limited to stream kinds affected by lost resources (e.g., microphone loss tears down audio routes without dropping unrelated video routes). Added regression coverage in `terminal_server/internal/transport/control_stream_test.go` (`TestHandleMessageCapabilityDeltaStopsOnlyAffectedRoutesOnPartialLoss`).
- 2026-04-26 (Stage 1 display capability geometry events): Extended `terminal_server/internal/transport/control_stream.go` capability-change event detection so endpoint-scoped display geometry keys (`display.<index>.(width|height|density|orientation|safe.*)`) trigger `terminal.display.resized` alongside legacy `screen.*` geometry. Added regression coverage in `terminal_server/internal/transport/control_stream_test.go` (`TestHandleMessageCapabilityDeltaEmitsDisplayResizedForDisplayCapabilityGeometry`).
- 2026-04-26 (Stage 1 generation guardrails): Extended `terminal_server/internal/transport/control_stream_test.go` stale `CapabilityDelta` coverage to assert protocol-violation rejection also preserves previously accepted capability generation/state (no stale delta mutation).
- 2026-04-26 (Stage 1 registry lifecycle hardening): Added device-manager regression coverage in `terminal_server/internal/device/manager_test.go` for capability lifecycle invariants: snapshot/delta capability replacement semantics, generation monotonicity enforcement (stale snapshot/delta rejection), and timestamp tracking (`LastSnapshot`/`LastDelta`) on accepted updates.
- 2026-04-26 (Stage 1 capability events): Server capability delta handling now emits explicit lifecycle events for capability gain/loss and audio route change (`terminal.capability.added`, `terminal.capability.removed`, `terminal.audio_route.changed`) in addition to existing update/loss events. Added transport regression coverage in `terminal_server/internal/transport/control_stream_test.go` for both endpoint gain and endpoint loss/resized paths.
- 2026-04-26 (Stage 1 generation guardrails): Added `terminal_server/internal/transport/control_stream_test.go` regression coverage asserting stale `CapabilitySnapshot` generations are rejected at the stream boundary with protocol-violation responses and without mutating previously accepted capability state.
- 2026-04-26 (Stage 1 lifecycle acknowledgment semantics): Hardened capability-ack coverage so transport lifecycle handling and generated-proto response encoding both assert snapshot-vs-delta `snapshot_applied` semantics and accepted generation propagation (`terminal_server/internal/transport/control_stream_test.go`, `terminal_server/internal/transport/generated_proto_adapter_test.go`).
- 2026-04-26 (Stage 1 scoped claim revocation): Fixed capability-loss claim handling to revoke only claims tied to lost resources (instead of releasing whole activations) by adding targeted claim-manager release support and wiring transport capability effects to use it (`terminal_server/internal/io/claims.go`, `terminal_server/internal/transport/control_stream.go`). Added regression coverage in `terminal_server/internal/io/claims_test.go` and `terminal_server/internal/transport/control_stream_test.go` to preserve unaffected claims on partial capability loss.
- 2026-04-26 (Stage 1 endpoint-scoped video route teardown hardening): Added transport regression coverage in `terminal_server/internal/transport/control_stream_test.go` (`TestHandleMessageCapabilityDeltaStopsVideoRouteOnEndpointLoss`) to lock in that camera endpoint swaps trigger video `stop_stream` teardown while preserving unrelated audio routes.
- 2026-04-26 (Stage 1 display endpoint route teardown hardening): Added transport regression coverage in `terminal_server/internal/transport/control_stream_test.go` (`TestHandleMessageCapabilityDeltaStopsVideoRouteOnDisplayEndpointLoss`) to lock in that display endpoint loss tears down only affected inbound video routes while preserving unrelated outbound audio routes.
- 2026-04-26 (Stage 1 snapshot claim-regain parity): Added transport regression coverage in `terminal_server/internal/transport/control_stream_test.go` (`TestHandleMessageCapabilitySnapshotRestoresSuspendedClaimsOnRebaselineRegain`) to lock in that capability snapshot rebaseline loss/regain follows the same scoped claim suspension and restoration behavior as generation-ordered deltas.

- 2026-04-26 (Stage 1 endpoint availability truthfulness): Hardened endpoint resource compilation in terminal_server/internal/transport/control_stream.go so endpoint-scoped resources are skipped when an endpoint is explicitly marked unavailable (*.endpoint.<index>.available=false), preventing false-positive capability side effects for unavailable media hardware. Added regression assertions in terminal_server/internal/transport/control_stream_test.go (TestCapabilityResourcesCompilesEndpointScopedResources) to lock in suppression of unavailable camera endpoint resources while preserving available endpoint fallback behavior.

- 2026-04-26 (Stage 1 display endpoint availability route-teardown parity): Added transport regression coverage in `terminal_server/internal/transport/control_stream_test.go` (`TestHandleMessageCapabilityDeltaStopsVideoRouteOnDisplayEndpointAvailabilityLoss`) to lock in that explicit display endpoint availability loss (`display.<index>.available=false`) tears down affected inbound video routes while preserving unrelated outbound audio routes.

## Related Plans

- [protocol.md](protocol.md) — Control-plane message flow and media-plane setup.
- [architecture-client.md](architecture-client.md) — Client-side capability enumeration and publication.
- [io-abstraction.md](io-abstraction.md) — Resource kinds and routing semantics derived from capabilities.
- [placement.md](placement.md) — Placement decisions over the live capability/resource model.
- [scenario-engine.md](scenario-engine.md) — Scenario reactions to capability and resource change events.
