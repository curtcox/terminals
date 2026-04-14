# Discovery and Connection

See [masterplan.md](../masterplan.md) for overall system context.

## Automatic Discovery (mDNS)

The server advertises itself via mDNS (Bonjour) on the local network:

- Service type: `_terminals._tcp.local.`
- TXT records: `version=1`, `name=HomeServer`

The client scans for this service on startup. On a trusted home LAN, no authentication is required — the first server found is used automatically. If multiple servers exist (not planned, but handled gracefully), the client presents a list.

## Manual Connection

If mDNS fails (e.g., network segmentation, mDNS blocked), the client shows a simple screen:

- Server address text field (IP or hostname)
- Port number (default pre-filled)
- Connect button

This is the **only** client-native UI. Everything else is server-driven.

## Connection Lifecycle

```
Client                          Server
  │                                │
  │──── mDNS query ───────────────→│ (or manual IP)
  │←─── mDNS response ────────────│
  │                                │
  │──── gRPC Connect ─────────────→│
  │──── RegisterDevice ───────────→│
  │←─── RegisterAck ──────────────│
  │                                │
  │←─── SetUI (initial screen) ───│
  │←─── StartStream (if needed) ──│
  │                                │
  │←──→ Heartbeat (periodic) ←──→ │
  │                                │
  │     (ongoing command/event     │
  │      exchange on the stream)   │
```

On disconnect, the client returns to the discovery/manual connect screen. On reconnect, the server restores the device's previous state if still applicable.

## Trust Model

For a home network, mDNS discovery + direct connection with no authentication keeps things simple. If this assumption changes, TLS mutual auth can be added at the transport layer without protocol changes.

## Related Plans

- [protocol.md](protocol.md) — The gRPC stream established after discovery.
- [architecture-client.md](architecture-client.md) — Client-side discovery module (`lib/discovery/`).
- [architecture-server.md](architecture-server.md) — Server-side mDNS module (`internal/discovery/`).
