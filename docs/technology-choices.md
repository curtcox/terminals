# Technology Choices

This document records the durable technology choices that are currently implemented in this repository.

## Server

| Component | Technology | Evidence |
|---|---|---|
| Language | Go | `terminal_server/go.mod`, server package layout under `terminal_server/` |
| Control API contract | Protobuf | `api/terminals/` and generated bindings under `api/gen/go/` + `api/gen/dart/` |
| Control transports | gRPC + WebSocket + TCP + HTTP fallback | `docs/server.md` (Control Transport Carriers), listeners in `terminal_server/cmd/server/` |
| Media transport | WebRTC | `docs/server.md`, media/session handling in `terminal_server/internal/transport/` |
| Discovery | mDNS | `terminal_server/internal/discovery/mdns.go` |
| Persistence | SQLite (modernc.org/sqlite) | server storage integration via Go modules |
| AI integration | Go interfaces behind provider abstractions | architecture constraints in `AGENTS.md` / `CLAUDE.md` and server internal interfaces |

## Client

| Component | Technology | Evidence |
|---|---|---|
| Framework | Flutter | `terminal_client/pubspec.yaml`, platform targets under `terminal_client/` |
| API bindings | Protobuf + Dart generated code | `api/gen/dart/` |
| Media transport | WebRTC (`flutter_webrtc`) | `terminal_client/pubspec.yaml`, platform client docs |
| Discovery | mDNS scanner(s) | `terminal_client/lib/discovery/mdns_scanner.dart`, `docs/discovery-and-connection.md` |
| Platform integration | Flutter plugin and platform channels where needed | platform folders under `terminal_client/` |

## Architecture Constraints

The repository-wide constraints that shape these choices are:

1. Keep the client generic (no scenario-specific behavior in Flutter).
2. Define communication contracts in protobuf.
3. Keep orchestration and scenarios in Go server code.
4. Keep AI providers behind interfaces.

These constraints are documented in `AGENTS.md`, `CLAUDE.md`, and `masterplan.md`.

## Related References

- `docs/server.md`
- `docs/discovery-and-connection.md`
- `docs/client-web.md`
- `docs/client-macos.md`
- `docs/client-ios.md`
- `docs/client-android.md`
- `docs/client-linux.md`
- `docs/client-windows.md`
