# Client (macOS) — Build & Run

The Flutter client lives in `terminal_client/`. This guide covers the **macOS** desktop target.

## Prerequisites

| Tool | Minimum version | Install |
|------|-----------------|---------|
| Flutter SDK | 3.4+ | <https://docs.flutter.dev/get-started/install> |
| Xcode | latest | Mac App Store |
| CocoaPods | latest | `brew install cocoapods` or `gem install cocoapods` |

Enable the macOS desktop target if not already enabled:

```bash
flutter config --enable-macos-desktop
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
cd terminal_client && flutter run -d macos
```

A native macOS window will open with the terminal client. The discovery screen will appear; select a discovered server or enter a server address manually.

## Build (release)

```bash
cd terminal_client && flutter build macos
```

The built `.app` bundle is placed in:

```
terminal_client/build/macos/Build/Products/Release/terminal_client.app
```

## Entitlements

The macOS build requires network entitlements for gRPC and WebRTC. The Flutter macOS runner includes default entitlements at:

```
terminal_client/macos/Runner/DebugProfile.entitlements
terminal_client/macos/Runner/Release.entitlements
```

Ensure the following are present:

```xml
<key>com.apple.security.network.client</key>
<true/>
<key>com.apple.security.network.server</key>
<true/>
```

For live media scenarios, include both microphone and camera entitlements:

```xml
<key>com.apple.security.device.audio-input</key>
<true/>
<key>com.apple.security.device.camera</key>
<true/>
```

When the OS denies media permissions at runtime, the client surfaces a
deterministic control-stream status (`Media permission required`) and records a
failure notification for the rejected stream start.

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

1. Start the server first (see [server.md](server.md)).
2. Launch the macOS client.
3. The client discovers the server via mDNS on the local network, or you can enter `host:port` manually.
4. Communication uses gRPC (port 50051 by default) with WebRTC for media.
