# Client (Web) — Build & Run

The Flutter client lives in `terminal_client/`. This guide covers the **web** target.

## Prerequisites

| Tool | Minimum version | Install |
|------|-----------------|---------|
| Flutter SDK | 3.4+ | <https://docs.flutter.dev/get-started/install> |
| Chrome (or Chromium) | latest | For running/debugging the web build |

Verify your environment:

```bash
flutter doctor
```

## Install dependencies

```bash
cd terminal_client
flutter pub get
```

## Run (development)

```bash
# From the repo root:
make run-client-web

# Or directly:
cd terminal_client && flutter build web --no-wasm-dry-run --pwa-strategy=none \
  --dart-define=TERMINALS_BUILD_SHA=$(git rev-parse --short=12 HEAD) \
  --dart-define=TERMINALS_BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
cd terminal_client && python3 -m http.server 60739 --bind 0.0.0.0 --directory build/web
```

This builds the web client and serves `build/web` via a static HTTP server.
The footer shows `Build` date and `SHA`, making stale deployments easy to identify.

The client will show a discovery/manual-connect screen. If the server is running on the same machine, mDNS discovery should find it automatically. Otherwise, enter the server address manually (e.g. `localhost:50051`).

## Build (release)

```bash
# From the repo root:
make client-build

# Or directly:
cd terminal_client && flutter build web
```

Build output is placed in `terminal_client/build/web/`. Serve these files with any static HTTP server.

## Test

```bash
make client-test

# Or directly:
cd terminal_client && flutter test
```

## Lint

```bash
make client-lint

# Or directly:
cd terminal_client && flutter analyze && dart format --set-exit-if-changed .
```

## Coverage

```bash
make client-coverage

# Or directly:
cd terminal_client && flutter test --coverage
```

Coverage data is written to `terminal_client/coverage/lcov.info`.

## Connecting to the server

1. Start the server first (see [server.md](server.md)).
2. Start the web client.
3. The client discovers the server via mDNS, or enter `host:port` manually.
4. The client communicates over gRPC (port 50051 by default) and uses WebRTC for media streams.
