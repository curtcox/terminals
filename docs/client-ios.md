# Client (iOS) — Build & Run

The Flutter client lives in `terminal_client/`. This guide covers the **iOS** target.

## Prerequisites

| Tool | Minimum version | Install |
|------|-----------------|---------|
| Flutter SDK | 3.4+ | <https://docs.flutter.dev/get-started/install> |
| Xcode | latest | Mac App Store |
| CocoaPods | latest | `brew install cocoapods` or `gem install cocoapods` |
| iOS Simulator or physical device | iOS 16+ | Via Xcode |

Verify your environment:

```bash
flutter doctor
```

## Install dependencies

```bash
cd terminal_client
flutter pub get
cd ios && pod install && cd ..
```

## Run (Simulator)

```bash
cd terminal_client && flutter run -d <simulator-id>
```

List available simulators with:

```bash
flutter devices
```

Or open the iOS Simulator first (`open -a Simulator`) and then:

```bash
cd terminal_client && flutter run
```

Flutter will target the running simulator by default.

## Run (physical device)

1. Connect the device via USB or ensure it is on the same Wi-Fi network.
2. Open `terminal_client/ios/Runner.xcworkspace` in Xcode.
3. Select your development team under **Signing & Capabilities**.
4. Run from Xcode, or use `flutter run -d <device-id>`.

## Build (release)

```bash
cd terminal_client && flutter build ios
```

This produces an `.app` bundle. To create a distributable `.ipa`:

```bash
cd terminal_client && flutter build ipa
```

Output is in `terminal_client/build/ios/ipa/`.

## Permissions

The iOS `Info.plist` (`terminal_client/ios/Runner/Info.plist`) must include usage descriptions for any hardware the app accesses:

| Key | When needed |
|-----|-------------|
| `NSMicrophoneUsageDescription` | Audio input / WebRTC |
| `NSLocalNetworkUsageDescription` | mDNS server discovery |
| `NSBonjourServices` | mDNS discovery (`_terminals._tcp`) |

Example entries:

```xml
<key>NSLocalNetworkUsageDescription</key>
<string>Terminals uses the local network to discover the server.</string>
<key>NSBonjourServices</key>
<array>
    <string>_terminals._tcp</string>
</array>
<key>NSMicrophoneUsageDescription</key>
<string>Terminals uses the microphone for voice features.</string>
```

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
2. The device and server must be on the same local network for mDNS discovery. Alternatively, enter the server IP and port manually.
3. Communication uses gRPC (port 50051 by default) with WebRTC for media.

> **Note:** The iOS Simulator cannot access mDNS on all network configurations. If discovery fails, use manual connect with `localhost:50051` (simulator) or `<server-ip>:50051` (device).
