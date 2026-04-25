# Multi-Transport Control Plane
See [masterplan.md](../masterplan.md) for overall system context. Extend the control protocol so clients and server can communicate over **gRPC, WebSocket, raw TCP sockets, or HTTP** without changing scenario semantics or client behavior. The client still acts as a generic terminal. Only the transport carrying typed protocol messages changes.

## Design Principle
Treat transport as a **carrier**, not a behavior boundary. The protobuf message model remains the source of truth; all four connection types carry the same logical `ClientMessage` / `ServerMessage` exchange.

This plan explicitly ignores migration and backward-compatibility concerns. There are no existing users.

## Goals
- Preserve one logical control session model across all supported carriers.
- Let the client automatically use any transport the server exposes and the network permits.
- Let the server expose multiple transports simultaneously.
- Keep scenario logic, IO routing, placement, and UI generation transport-agnostic.
- Produce useful connection-failure diagnostics through any output resource the client still has available.

## Non-Goals
- No compatibility shims for previous protocol versions.
- No attempt to preserve current wire compatibility.
- No scenario-specific fallback behavior in the client.
- No requirement that all transports support the same performance profile. They only need to preserve correctness for control-plane semantics.

## Transport Model
The protocol gains a **session layer** above concrete networking.

```text
Scenario Engine / Device Manager / IO Router
                 │
          Session Layer
                 │
   ┌─────────────┼─────────────┬──────────────┬──────────────┐
   │             │             │              │              │
 gRPC      WebSocket      TCP stream     HTTP fallback   (future)
 bidi        bidi          framed bidi    request/stream
```

A session is defined by:
- one authenticated-free trusted-LAN connection attempt
- one chosen carrier
- one ordered outbound server message stream
- one ordered inbound client message stream
- one heartbeat/health model
- one reconnect/resume identity

The server owns session semantics. Transport adapters only translate bytes and connection events into the shared session interface.

## Wire Contract
Keep protobuf as the canonical schema. Replace the protocol assumption that control always means gRPC with the rule that control means **protobuf messages over any supported carrier**.

### Envelope
Introduce an explicit framing envelope for non-gRPC carriers:

```protobuf
message WireEnvelope {
  uint32 protocol_version = 1;
  string session_id = 2;
  uint64 sequence = 3;
  oneof payload {
    ClientMessage client_message = 10;
    ServerMessage server_message = 11;
    TransportHello transport_hello = 12;
    TransportHelloAck transport_hello_ack = 13;
    TransportHeartbeat transport_heartbeat = 14;
    TransportError transport_error = 15;
  }
}
```

Notes:
- gRPC may continue to carry `ClientMessage` / `ServerMessage` directly in generated RPCs, but the session layer should treat them as equivalent to `WireEnvelope` frames.
- WebSocket and TCP carry length-delimited binary `WireEnvelope` messages.
- HTTP carries binary protobuf request/response bodies containing `WireEnvelope` batches or a streamed response body containing successive length-delimited `WireEnvelope` messages.

### Transport Hello
Before device registration, every carrier performs a transport handshake:
- client sends `TransportHello` with supported carriers, protocol version, desired device ID, and optional resume token
- server replies with `TransportHelloAck` including accepted protocol version, negotiated carrier semantics, heartbeat interval, and transport-specific limits
- client then sends `RegisterDevice`

This makes carrier selection explicit instead of inferring everything from socket type.

## Supported Carriers
### gRPC
Bidirectional streaming remains the preferred carrier when available.

Use when:
- HTTP/2 is available
- generated client/server bindings are supported on the platform
- no intermediary blocks long-lived gRPC streams

Keeps the current `TerminalControl.Connect` shape, but the session layer should no longer assume every client can establish it.

### WebSocket
WebSocket becomes the default fallback for environments that can do HTTP(S) upgrade but not gRPC.

Properties:
- bidirectional
- message oriented
- easy to support in browsers and constrained environments
- binary protobuf frames

Recommended mapping:
- one WebSocket connection per control session
- each frame = one length-delimited `WireEnvelope`

### TCP Stream
Raw TCP provides the lowest-common-denominator persistent socket when HTTP infrastructure is unavailable or undesirable.

Properties:
- bidirectional
- simple framing
- useful on trusted LANs and embedded devices
- no browser support expected

Recommended mapping:
- one TCP connection per control session
- 4-byte big-endian frame length + protobuf bytes
- explicit idle timeout and heartbeat enforcement at the session layer

### HTTP Fallback
HTTP is the lowest-capability carrier and should be designed for correctness, not elegance.

Properties:
- works when only plain request/response is possible
- may be half-duplex depending on environment
- suitable for restrictive captive portals, proxies, or limited runtimes

Recommended mapping:
- `POST /v1/control/poll` for client → server message batches
- `GET /v1/control/stream` for server → client stream using chunked transfer, SSE-style framing, or hanging GET
- if long-lived response streaming is blocked, fall back further to repeated long-poll GETs returning queued server frames

The HTTP carrier is the only one allowed to degrade from fully bidirectional streaming to queued half-duplex exchange, but it must preserve ordered delivery within each direction.

## Session Interface
Define a shared server-side and client-side transport interface.

```text
type SessionTransport interface {
  Open(ctx) error
  ReadEnvelope(ctx) (WireEnvelope, error)
  WriteEnvelope(ctx, WireEnvelope) error
  Close() error
  Carrier() CarrierKind
}
```

Server-side adapters:
- `GrpcSessionTransport`
- `WebSocketSessionTransport`
- `TcpSessionTransport`
- `HttpSessionTransport`

Client-side adapters:
- `GrpcCarrier`
- `WebSocketCarrier`
- `TcpCarrier`
- `HttpCarrier`

Everything above this layer talks in terms of sessions, envelopes, liveness, and backpressure — never direct socket APIs.

## Discovery and Advertised Endpoints
Discovery must advertise **transport availability**, not just one port.

mDNS TXT records should publish:
- `version=<protocol_version>`
- `name=<server_name>`
- `grpc=<host:port or 0>`
- `ws=<url or path hint>`
- `tcp=<host:port or 0>`
- `http=<base_url or 0>`
- `priority=<ordered carrier preference>`

Manual connect UI should accept either:
- a base host/IP and derive transport endpoints from defaults, or
- a full endpoint override per carrier for debugging/admin use

The client-native connection UI remains the only client-native UI.

## Carrier Selection Algorithm
The client automatically chooses the best working carrier.

### Ordered Attempt Policy
1. Build a candidate set from discovery or manual configuration.
2. Remove carriers the local platform cannot support.
3. Order remaining carriers by server-advertised priority, overridden by client hard exclusions for platform/runtime limits.
4. Attempt carriers sequentially with bounded timeouts.
5. On first successful `TransportHelloAck`, bind the session to that carrier and stop probing.
6. Cache the most recent successful carrier for that server and try it first on the next reconnect unless discovery data changed.

Recommended default preference:
1. gRPC
2. WebSocket
3. TCP
4. HTTP

### Failure Memory
For each attempt, persist an in-memory diagnostic record:
- discovery source
- carrier
- endpoint
- DNS result
- TCP connect result
- TLS/upgrade result if applicable
- handshake result
- protocol rejection reason
- elapsed time

The diagnostic record feeds the user-visible failure report if all carriers fail.

## Heartbeats and Liveness
Unify liveness across carriers.

- The session layer owns heartbeat timers.
- `TransportHeartbeat` is transport-neutral.
- gRPC can map transport heartbeat to app-level heartbeat instead of relying only on HTTP/2 keepalive.
- WebSocket/TCP require explicit heartbeat frames.
- HTTP long-poll uses server-issued expiry plus client poll deadlines; missing two consecutive windows marks the session unhealthy.

Device reconnection behavior stays the same: if the device reconnects with the same logical identity, the server restores state when appropriate.

## Resume Semantics
Resume should be carrier-independent.

Add a resume token to `RegisterAck` or `TransportHelloAck`. On reconnect:
- client presents device ID + resume token
- server rebinds the logical device session if the token is valid and the prior session is gone or expired
- scenario/UI restoration happens above transport, exactly once

The important rule is: switching carriers during reconnect is normal. Resume does not depend on reusing the same carrier.

## Server Structure Changes
Extend the server transport package from a gRPC/WebRTC assumption to a multi-carrier control stack.

```text
terminal_server/
└── internal/
    ├── discovery/
    │   └── mdns.go
    ├── transport/
    │   ├── session.go
    │   ├── envelope.go
    │   ├── grpc_server.go
    │   ├── websocket_server.go
    │   ├── tcp_server.go
    │   ├── http_control.go
    │   ├── registry.go
    │   ├── liveness.go
    │   └── webrtc_signaling.go
    └── device/
        └── manager.go
```

Responsibilities:
- `session.go`: carrier-neutral session state machine
- `envelope.go`: framing, sequencing, batching helpers
- `registry.go`: configured listeners and advertised endpoints
- `liveness.go`: heartbeat, idle timeout, reconnect windows
- carrier files: concrete listener/acceptor implementations only

## Client Structure Changes
Extend the client connection module from one gRPC path to a carrier-selection layer.

```text
terminal_client/
└── lib/
    ├── discovery/
    │   ├── mdns_scanner.dart
    │   └── manual_connect.dart
    ├── connection/
    │   ├── connection_manager.dart
    │   ├── transport_selection.dart
    │   ├── transport_diagnostics.dart
    │   ├── carriers/
    │   │   ├── grpc_carrier.dart
    │   │   ├── websocket_carrier.dart
    │   │   ├── tcp_carrier.dart
    │   │   └── http_carrier.dart
    │   └── webrtc_manager.dart
    └── diagnostics/
        ├── connection_reporter.dart
        ├── display_reporter.dart
        ├── audio_reporter.dart
        └── indicator_reporter.dart
```

Responsibilities:
- `connection_manager.dart`: owns attempt order, session lifecycle, reconnect
- `transport_selection.dart`: capability and priority filtering
- `transport_diagnostics.dart`: per-attempt evidence capture
- `connection_reporter.dart`: chooses usable output resources for failure reporting

## Failure Reporting
If no carrier succeeds, the client must report connection failure diagnostic information through any output resource it has.

### Reporting Rule
Use every available reporting channel that the device can safely drive without a server connection.

Possible channels:
- display: full-screen diagnostic page with attempt log
- speaker: spoken summary or alert tones encoding failure class
- LED / screen flash / blinking light: coarse failure indicator for headless devices
- haptic: vibration pattern for handheld devices
- local log storage: structured failure record for later inspection

### Diagnostic Content
At minimum include:
- discovered server identity/endpoints
- attempted carriers in order
- failure stage for each carrier
- local network facts available to the client (DNS failure, timeout, refused, HTTP status, upgrade rejected, protobuf version mismatch)
- timestamp
- next retry timing/state

### Human-Facing Summary
The display/audio summary should answer:
- did discovery work?
- which carriers were attempted?
- where did each fail?
- is the client still retrying?

Example display copy:
- `Found HomeServer via mDNS.`
- `gRPC failed: timeout during HTTP/2 connect.`
- `WebSocket failed: upgrade rejected (HTTP 403).`
- `TCP failed: connection refused.`
- `HTTP failed: no response to long-poll request.`

## Interaction with WebRTC
This plan is primarily about the control plane. WebRTC remains the preferred real-time media plane when media is needed.

However:
- media setup commands must be deliverable over any control carrier
- WebRTC signaling messages become carrier-neutral session messages
- if the chosen control carrier is HTTP fallback, WebRTC setup may be slower, but semantics remain the same

No alternative media plane is introduced here. Only the control plane becomes multi-transport.

## Protocol Rules
1. Never add scenario-specific logic to transport adapters.
2. Never make a scenario depend on a specific carrier.
3. Keep protobuf as the canonical schema for all carriers.
4. Treat reconnect over a different carrier as normal, not exceptional.
5. Report connection failure through any local output resource available to the client.

## Related Plans
- [protocol.md](protocol.md) — Existing control/media split to revise around the session layer.
- [discovery.md](discovery.md) — Discovery records and manual-connect UX to extend with carrier metadata.
- [architecture-client.md](architecture-client.md) — Client transport-selection and diagnostics modules.
- [architecture-server.md](architecture-server.md) — Server transport listener/module layout.
- [phase-1-foundation.md](phase-1-foundation.md) — Foundational control-plane work that this supersedes conceptually.
- [phase-3-media.md](phase-3-media.md) — WebRTC signaling remains above the multi-carrier control layer.
