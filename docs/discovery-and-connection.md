# Discovery and Connection

This document is the durable reference for client/server discovery and control
stream connection behavior.

## Automatic Discovery (mDNS)

- Server advertises `_terminals._tcp.local.` with identity and transport TXT
  metadata.
- Core TXT keys:
  - `version`
  - `name`
  - `grpc`
  - `ws`
  - `tcp`
  - `http`
  - `mcp`
  - `priority`
- Client scans for this service and lists discovered servers.

## Manual Connection

When discovery does not produce a usable target, the client supports manual
connection with:

- Server host field
- Server port field
- Connect button

This is the only client-native connection UI. Application UI after registration
is server-driven.

## Connection Lifecycle

1. Client discovers server via mDNS or receives a manual host/port.
2. Client opens a control transport carrier (runtime-dependent).
3. Client sends bootstrap messages (`hello`, `register`, capability snapshot,
   heartbeat).
4. Server returns `register_ack` and initial UI (`set_ui`).
5. Client and server continue heartbeat and command/event exchange.

On disconnect, the client remains on the connection screen and retries while
the session is expected to stay online.

## Carrier Selection and Fallback

- Server advertises endpoints and carrier priority in mDNS TXT metadata.
- Client computes runtime-supported carrier order from that priority.
- If a carrier fails, client rotates to the next available carrier and records
  diagnostics.
- If all carriers fail, client schedules reconnect with backoff.

## Reliability Model

The client uses one shared reliability path for outbound control messages.

### Connection phase source of truth

Client connection status is derived from a single phase enum:

- `disconnected`
- `connecting`
- `connected_unregistered`
- `registered`
- `degraded`

UI connection chips, readiness checks, and dispatch gating use this shared phase
model rather than per-feature booleans.

### Shared readiness gateway

Before queue-until-ready and ack-required sends, the client calls a shared
readiness helper (`ensureConnectedAndRegistered`) that:

1. starts transport/registration when needed,
2. polls for `registered` phase using shared retry timing,
3. returns typed readiness outcomes (`ready`, `timeout`, `failed`).

### Shared outbound routing rules

Outbound operations declare one routing rule each:

- send mode (`fire_and_forget`, `queue_until_ready`, `require_ack`),
- replay safety (`safe_to_replay`),
- whether ack is required.

These rules are centralized in
`terminal_client/lib/connection/reliability.dart` and consumed by
`_sendWhenReady(...)` in `terminal_client/lib/main.dart`.

### Shared retry policy

Retry intervals and timeout windows are defined by reusable retry policies and
controllers (fixed or exponential backoff) instead of duplicated watchdogs in
individual features.

## Trust Model

For trusted home LAN deployments, discovery plus direct connection is
intentionally simple. Transport hardening (for example, TLS mutual
authentication) can be introduced without changing the app-level control
protocol.

## Primary Implementation References

- Server mDNS advertiser: `terminal_server/internal/discovery/mdns.go`
- Server startup wiring: `terminal_server/cmd/server/main.go`
- Client scanner: `terminal_client/lib/discovery/mdns_scanner.dart`
- Client connection lifecycle/UI: `terminal_client/lib/main.dart`
