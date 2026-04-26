# Client (Linux) — Build & Run

The Flutter client lives in `terminal_client/`. This guide covers the **Linux** desktop target.

## Prerequisites

| Tool | Minimum version | Install |
|------|-----------------|---------|
| Flutter SDK | 3.4+ | <https://docs.flutter.dev/get-started/install> |
| clang / CMake / ninja / pkg-config | latest | See below |
| GTK 3 development headers | 3.x | See below |

Install build dependencies (Debian/Ubuntu):

```bash
sudo apt update
sudo apt install clang cmake ninja-build pkg-config libgtk-3-dev
```

Enable the Linux desktop target if not already enabled:

```bash
flutter config --enable-linux-desktop
```

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
cd terminal_client && flutter run -d linux
```

A native GTK window will open with the terminal client.

## Build (release)

```bash
cd terminal_client && flutter build linux
```

Output is placed in:

```
terminal_client/build/linux/x64/release/bundle/
```

The `bundle/` directory is self-contained and can be copied to another machine with the same architecture.

## Media Permission Limits

Linux desktop builds rely on host portal/device-stack behavior for camera and
microphone access, and prompt UX is compositor/desktop-environment dependent.
If prompts are blocked or denied, media starts fail with a deterministic
client status (`Media permission required`) plus a stream-start failure
notification instead of hanging.

## Test

```bash
cd terminal_client && flutter test
```

Tests are platform-independent and shared across all client targets.

## Lint

```bash
cd terminal_client && flutter analyze && dart format --set-exit-if-changed .
```

## Connecting to the server

1. Start the server (see [server.md](server.md)).
2. The client discovers the server via mDNS on the local network, or you can enter `host:port` manually.
3. Communication uses gRPC (port 50051 by default) with WebRTC for media.

> **Note:** If mDNS discovery is not working, ensure Avahi is installed and running (`sudo systemctl start avahi-daemon`), and that multicast traffic is allowed by your firewall.
