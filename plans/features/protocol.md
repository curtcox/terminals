---
title: "Protocol Design"
kind: plan
status: building
owner: copilot
validation: none
last-reviewed: 2026-04-26
---

# Protocol Design

See [masterplan.md](../archive/masterplan-duplicate.md) for overall system context.

The protocol has two layers:

- a **control plane** (gRPC) for commands, state, registration, and capability lifecycle
- a **media plane** (WebRTC) for real-time audio/video/data streams

## Control Plane (gRPC)

Bidirectional streaming RPCs over gRPC. The client maintains a persistent control stream to the server.

```protobuf
service TerminalControl {
  // Persistent bidirectional control stream.
  rpc Connect(stream ClientMessage) returns (stream ServerMessage);
}

message ClientMessage {
  oneof payload {
    Hello hello = 1;
    CapabilitySnapshot capability_snapshot = 2;
    CapabilityDelta capability_delta = 3;
    InputEvent input = 4;              // keyboard, pointer, touch, HID
    SensorData sensor = 5;
    StreamReady stream_ready = 6;      // WebRTC session established
    CommandAck ack = 7;
    Heartbeat heartbeat = 8;
  }
}

message ServerMessage {
  oneof payload {
    HelloAck hello_ack = 1;
    CapabilityAck capability_ack = 2;
    SetUI set_ui = 3;                  // Server-driven UI descriptor
    StartStream start_stream = 4;      // Begin audio/video/sensor stream
    StopStream stop_stream = 5;
    PlayAudio play_audio = 6;          // Play audio clip or TTS
    ShowMedia show_media = 7;          // Display image/video
    RouteStream route_stream = 8;      // Connect stream to another device
    Notification notification = 9;     // Toast/alert
    WebRTCSignal webrtc_signal = 10;   // SDP/ICE for media setup
    CommandRequest command = 11;       // Generic command
    Heartbeat heartbeat = 12;
  }
}
```

The connection handshake is:

1. client opens `Connect`
2. client sends `Hello`
3. client sends a complete `CapabilitySnapshot`
4. server persists the device record, compiles resources from capabilities, and replies with `HelloAck`
5. server replies with `CapabilityAck`
6. normal command / stream traffic begins

The server does not infer the current terminal shape from stale registration records. The client is the source of truth for its current capabilities.

## Capability Lifecycle

Capabilities are not a one-time registration blob. They are a live contract between terminal and server.

A terminal must send:

- **one full `CapabilitySnapshot` on initial connection**
- **one `CapabilityDelta` whenever capabilities change**

Capability changes include:

- display size, rotation, density, refresh rate, safe insets, or fullscreen state changing
- a keyboard, pointer, camera, microphone, speaker, or headset being added or removed
- permissions changing for camera, mic, Bluetooth, location, notifications, etc.
- route changes such as speaker vs headphones vs Bluetooth audio device
- sensor availability changing
- battery / power-state fields that affect placement or policy
- any client-detectable capability that changes what commands or stream plans are valid

The server acknowledges every snapshot and delta with `CapabilityAck`. The ack includes the applied capability revision and any resulting resource invalidations.

```protobuf
message Hello {
  string device_id = 1;
  string device_name = 2;
  string device_type = 3;
  string platform = 4;
  string client_version = 5;
}

message HelloAck {
  string session_id = 1;
  uint64 capability_revision = 2;
}

message CapabilitySnapshot {
  uint64 revision = 1;
  DeviceCapabilities capabilities = 2;
}

message CapabilityDelta {
  uint64 revision = 1;
  repeated CapabilityChange changes = 2;
  string reason = 3; // resize, hotplug, permission_change, route_change, etc.
}

message CapabilityAck {
  uint64 applied_revision = 1;
  repeated ResourceInvalidation invalidations = 2;
}
```

Revisions are monotonic per connection session. The server applies deltas in revision order and treats a fresh snapshot as authoritative.

## Capability Model

A capability manifest describes **what exists now**, not abstract product marketing claims.

The model is endpoint-oriented:

- a device can expose zero, one, or many displays
- zero, one, or many cameras
- zero, one, or many microphones
- zero, one, or many speakers / audio outputs
- zero, one, or many HID-style inputs
- zero, one, or many sensors or radios

That avoids hard-coding assumptions like “one front camera, one back camera, one screen.”

```protobuf
message DeviceCapabilities {
  repeated Display displays = 1;
  repeated Keyboard keyboards = 2;
  repeated Pointer pointers = 3;
  repeated TouchSurface touch_surfaces = 4;
  repeated AudioInput audio_inputs = 5;
  repeated AudioOutput audio_outputs = 6;
  repeated Camera cameras = 7;
  repeated Sensor sensors = 8;
  repeated Radio radios = 9;
  repeated Peripheral peripherals = 10;
  Power power = 11;
  Connectivity connectivity = 12;
}

message Display {
  string id = 1;
  string role = 2; // main, external, projector, virtual, etc.
  uint32 width_px = 3;
  uint32 height_px = 4;
  float density = 5;
  float refresh_hz = 6;
  bool touch = 7;
  bool hdr = 8;
  bool available = 9;
}

message AudioInput {
  string id = 1;
  string kind = 2; // built_in_mic, usb_mic, bluetooth_headset_mic, etc.
  uint32 channels = 3;
  repeated uint32 sample_rates = 4;
  bool echo_canceled = 5;
  bool available = 6;
  bool default_route = 7;
}

message AudioOutput {
  string id = 1;
  string kind = 2; // built_in_speaker, headphones, bluetooth_speaker, etc.
  uint32 channels = 3;
  repeated uint32 sample_rates = 4;
  bool available = 5;
  bool default_route = 6;
}

message Camera {
  string id = 1;
  string kind = 2; // front, back, usb, external, virtual
  repeated VideoMode modes = 3;
  bool available = 4;
}
```

The exact proto layout can evolve, but the shape must stay endpoint-oriented and hot-plug-friendly.

## Why snapshots and deltas

The server needs two things that are in tension:

- a complete view for reconnect / reconciliation
- a cheap update path for runtime changes

So the protocol supports both:

- `CapabilitySnapshot` is a full replacement state
- `CapabilityDelta` is an incremental change list

The client may send a fresh snapshot instead of a delta whenever that is simpler.

## Server behavior on capability change

When capabilities change, the server:

1. updates the device record
2. recompiles claimable resources from the new capability graph
3. checks active claims and media plans against the new resource set
4. tears down or patches plans that became invalid
5. emits lifecycle events for the scenario engine if needed

Examples:

- if the only speaker disappears because headphones were unplugged and no fallback route exists, affected speaker sinks are stopped
- if a display resizes, the server may patch server-driven UI layout and compositor targets
- if a camera permission is revoked, camera-based claims are released and dependent plans are torn down
- if a better output route appears, the placement / routing policy may move future sinks without disturbing unrelated claims

Capability change is therefore a first-class driver of routing and activation lifecycle, not just metadata refresh.

## Media Plane (WebRTC)

WebRTC peer connections carry real-time media between clients and the server.

The server acts as an SFU (Selective Forwarding Unit) — it receives media streams from clients and selectively forwards them to other clients or processes them locally.

```text
Client A (mic) ──WebRTC──→ Server ──WebRTC──→ Client B (speaker)
                                │
                                ├──→ AI (speech-to-text)
                                └──→ Disk (recording)
```

WebRTC is used instead of raw streaming because:

- built-in echo cancellation, noise suppression, and automatic gain control
- adaptive bitrate based on network conditions
- NAT traversal (future-proofing for off-network use)
- Flutter has mature WebRTC support (`flutter_webrtc`)

## Data Streams (WebRTC DataChannel)

For non-media IO like sensor data and low-latency control sidebands, WebRTC DataChannels provide an unreliable-ordered or reliable channel alongside media streams.

Keyboard / pointer / touch input still belongs on the typed gRPC control stream unless a future use case proves that an in-band media-adjacent path is necessary.

## Proto Files

Proto files live in `api/terminals/` and are split by concern:

- `control/v1/control.proto` — device ↔ server control messages and handshake
- `capabilities/v1/capabilities.proto` — capability snapshots/deltas and endpoint records
- `io/v1/io.proto` — IO stream control and runtime stream records
- `ui/v1/ui.proto` — server-driven UI descriptors

Codegen is driven by Buf (`buf.yaml`, `buf.gen.yaml`). Go and Dart bindings are generated into the server and client trees respectively; CI verifies generated code is committed and up to date.

## Progress (2026-04-26)

- Added explicit capability invalidation payloads to control-plane acknowledgements in `CapabilityAck.invalidations` (`api/terminals/control/v1/control.proto`).
- Wired server transport capability ack generation to include deterministic lost-resource invalidations (resource + reason) when snapshots/deltas remove claimable resources.
- Added/updated transport regression coverage for ack invalidation content and proto adapter mapping.
- Updated durable connection docs to describe `capability_ack` invalidation behavior.
- Removed client bootstrap emission of deprecated `RegisterDevice` requests; client bootstrap now sends `hello` + `capability_snapshot` and retries snapshot delivery until acknowledgement instead of retrying register payloads.

Remaining protocol-plan work includes server/proto deprecation cleanup for legacy `RegisterDevice` / `CapabilityUpdate` ingest paths and final reconciliation with capability-lifecycle design targets.

## Related Plans

- [architecture-client.md](architecture-client.md) — Client-side protocol use.
- [architecture-server.md](architecture-server.md) — Server-side protocol use.
- [server-driven-ui.md](server-driven-ui.md) — `SetUI` descriptor format.
- [io-abstraction.md](io-abstraction.md) — resource model and stream semantics.
