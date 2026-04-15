# Client (Android) — Build & Run

The Flutter client lives in `terminal_client/`. This guide covers the **Android** target.

## Prerequisites

| Tool | Minimum version | Install |
|------|-----------------|---------|
| Flutter SDK | 3.4+ | <https://docs.flutter.dev/get-started/install> |
| Android Studio | latest | <https://developer.android.com/studio> |
| Android SDK | API 21+ | Via Android Studio SDK Manager |
| Java JDK | 17 | Bundled with Android Studio or install separately |

Verify your environment:

```bash
flutter doctor
```

Ensure `flutter doctor` shows no issues for the Android toolchain.

## Install dependencies

```bash
cd terminal_client
flutter pub get
```

## Run (Emulator)

1. Create an AVD (Android Virtual Device) via Android Studio's Device Manager.
2. Start the emulator, then:

```bash
cd terminal_client && flutter run
```

Or target a specific emulator:

```bash
flutter devices                        # list available devices
cd terminal_client && flutter run -d <emulator-id>
```

## Run (physical device)

1. Enable **Developer Options** and **USB Debugging** on the device.
2. Connect via USB and accept the debugging prompt.
3. Run:

```bash
cd terminal_client && flutter run -d <device-id>
```

## Build (release)

### APK

```bash
cd terminal_client && flutter build apk
```

Output: `terminal_client/build/app/outputs/flutter-apk/app-release.apk`

### App Bundle (for Play Store)

```bash
cd terminal_client && flutter build appbundle
```

Output: `terminal_client/build/app/outputs/bundle/release/app-release.aab`

## Permissions

The Android manifest (`terminal_client/android/app/src/main/AndroidManifest.xml`) must include:

```xml
<uses-permission android:name="android.permission.INTERNET" />
<uses-permission android:name="android.permission.RECORD_AUDIO" />
<uses-permission android:name="android.permission.CHANGE_WIFI_MULTICAST_STATE" />
```

- `INTERNET` — gRPC and WebRTC communication.
- `RECORD_AUDIO` — microphone access for voice/WebRTC features.
- `CHANGE_WIFI_MULTICAST_STATE` — mDNS server discovery on the local network.

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
2. The device and server must be on the same local network for mDNS discovery.
3. If using an emulator, the host machine's `localhost` is reachable at `10.0.2.2`. Enter `10.0.2.2:50051` in the manual connect screen.
4. Communication uses gRPC (port 50051 by default) with WebRTC for media.
