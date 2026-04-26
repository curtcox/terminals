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
