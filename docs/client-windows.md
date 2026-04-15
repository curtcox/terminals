# Client (Windows) — Build & Run

The Flutter client lives in `terminal_client/`. This guide covers the **Windows** desktop target.

## Prerequisites

| Tool | Minimum version | Install |
|------|-----------------|---------|
| Flutter SDK | 3.4+ | <https://docs.flutter.dev/get-started/install> |
| Visual Studio 2022 | latest | With the **Desktop development with C++** workload |
| Windows 10 | 1903+ | — |

Enable the Windows desktop target if not already enabled:

```powershell
flutter config --enable-windows-desktop
```

Verify your environment:

```powershell
flutter doctor
```

## Install dependencies

```powershell
cd terminal_client
flutter pub get
```

## Run (development)

```powershell
cd terminal_client
flutter run -d windows
```

A native Win32 window will open with the terminal client.

## Build (release)

```powershell
cd terminal_client
flutter build windows
```

Output is placed in:

```
terminal_client\build\windows\x64\runner\Release\
```

The `Release\` directory is self-contained and can be distributed to other Windows machines.

## Test

```powershell
cd terminal_client
flutter test
```

Tests are platform-independent and shared across all client targets.

## Lint

```powershell
cd terminal_client
flutter analyze
dart format --set-exit-if-changed .
```

## Connecting to the server

1. Start the server (see [server.md](server.md)).
2. The client discovers the server via mDNS on the local network, or you can enter `host:port` manually.
3. Communication uses gRPC (port 50051 by default) with WebRTC for media.

> **Note:** Windows Firewall may block mDNS multicast traffic. If discovery fails, allow UDP port 5353 or use manual connect with `<server-ip>:50051`.
