# Protocol Design

See [masterplan.md](../masterplan.md) for overall system context.

The protocol has two layers: a **control plane** (gRPC) for commands and state, and a **media plane** (WebRTC) for real-time audio/video/data streams.

## Control Plane (gRPC)

Bidirectional streaming RPCs over gRPC. The client maintains a persistent control stream to the server.

```protobuf
service TerminalControl {
  // Persistent bidirectional control stream
  rpc Connect(stream ClientMessage) returns (stream ServerMessage);
}

message ClientMessage {
  oneof payload {
    RegisterDevice     register        = 1;
    CapabilityUpdate   capability      = 2;
    InputEvent         input           = 3;  // keyboard, pointer, touch
    SensorData         sensor          = 4;
    StreamReady        stream_ready    = 5;  // WebRTC session established
    CommandAck         ack             = 6;
    Heartbeat          heartbeat       = 7;
  }
}

message ServerMessage {
  oneof payload {
    RegisterAck        register_ack    = 1;
    SetUI              set_ui          = 2;  // Server-driven UI descriptor
    StartStream        start_stream    = 3;  // Begin audio/video/sensor stream
    StopStream         stop_stream     = 4;
    PlayAudio          play_audio      = 5;  // Play audio clip or TTS
    ShowMedia          show_media      = 6;  // Display image/video
    RouteStream        route_stream    = 7;  // Connect stream to another device
    Notification       notification    = 8;  // Toast/alert
    WebRTCSignal       webrtc_signal   = 9;  // SDP/ICE for media setup
    CommandRequest     command         = 10; // Generic command
    Heartbeat          heartbeat       = 11;
  }
}
```

## Media Plane (WebRTC)

WebRTC peer connections carry real-time media between clients and the server. The server acts as an SFU (Selective Forwarding Unit) — it receives media streams from clients and selectively forwards them to other clients or processes them locally.

```
Client A (mic) ──WebRTC──→ Server ──WebRTC──→ Client B (speaker)
                              │
                              ├──→ AI (speech-to-text)
                              └──→ Disk (recording)
```

WebRTC is used instead of raw streaming because:
- Built-in echo cancellation, noise suppression, and automatic gain control
- Adaptive bitrate based on network conditions
- NAT traversal (future-proofing for off-network use)
- Flutter has mature WebRTC support (`flutter_webrtc`)

## Data Streams (WebRTC DataChannel)

For non-media IO like sensor data, keyboard events, and low-latency commands, WebRTC DataChannels provide an unreliable-ordered or reliable channel alongside the media streams.

## Proto Files

Proto files live in `api/proto/` and are split by concern:

- `control.proto` — Device ↔ Server control messages
- `capabilities.proto` — Capability declarations
- `io.proto` — IO stream control
- `ui.proto` — Server-driven UI descriptors

Codegen is driven by Buf (`buf.yaml`, `buf.gen.yaml`). Go and Dart bindings are generated into the server and client trees respectively; CI verifies generated code is committed and up to date.

## Related Plans

- [architecture-client.md](architecture-client.md) — Client-side protocol use.
- [architecture-server.md](architecture-server.md) — Server-side protocol use.
- [server-driven-ui.md](server-driven-ui.md) — `SetUI` descriptor format.
- [io-abstraction.md](io-abstraction.md) — `StartStream` / `StopStream` / `RouteStream` semantics.
