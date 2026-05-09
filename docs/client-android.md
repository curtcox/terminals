# Client (Native Android)

The native Android client lives in `android_client/`. It is a generic Terminals
thin client for Android and Kindle Fire tablets; scenario behavior, routing,
policy, and orchestration remain on the Go server.

The existing Flutter Android target still lives under `terminal_client/` and is
built by `make client-build-android`.

## Target

- Package: `com.curtcox.terminals.android`
- Initial device class: Kindle Fire tablets running Fire OS 6 or newer
- Android API: `minSdk 25`, `targetSdk 36`, `compileSdk 36`
- Google services: not used
- Install path: Android Studio, ADB, or Fire tablet sideloading

Do not add Google Play Services, Firebase, Nearby, Cast, Play Integrity,
SafetyNet, Play Store-only APIs, Amazon account APIs, or scenario-specific
Android code without a follow-up plan.

## Prerequisites

| Tool | Version | Notes |
| --- | --- | --- |
| Android Studio | current stable | Includes Android SDK and platform tools |
| Android SDK | API 36 installed | API 25+ runtime compatibility is required |
| JDK | 17 | Android Studio bundled JDK is fine |
| Gradle | wrapper included | `android_client/gradlew` downloads Gradle 8.13 when needed |
| ADB | current platform tools | Needed for Fire tablet install/smoke tests |

`make android-client-build` (and test/lint) resolve a JDK automatically in this order: a working `JAVA_HOME` if set, Homebrew `openjdk@17`, Android Studio’s JBR under `Applications` or `~/Applications`, common Linux OpenJDK 17 paths, then macOS `/usr/libexec/java_home`. If no JDK is found, those targets skip with an explicit message instead of invoking Gradle with an empty `JAVA_HOME`.

Set one SDK environment variable:

```bash
export ANDROID_SDK_ROOT="$HOME/Library/Android/sdk"
```

## Build

```bash
make android-client-build
```

Direct Gradle command:

```bash
cd android_client
./gradlew assembleDebug
```

Debug APK output:

```text
android_client/app/build/outputs/apk/debug/app-debug.apk
```

## Test And Lint

```bash
make android-client-test
make android-client-lint
make android-client-boundary
```

Direct Gradle commands:

```bash
cd android_client
./gradlew testDebugUnitTest
./gradlew lintDebug
```

Connected-device tests require an emulator or physical device:

```bash
adb devices
make android-client-connected-test
```

If `adb` is missing or no device is attached, the Make target skips with a clear
message instead of failing Android client validation.

## Fire Tablet Setup

1. Open Fire tablet settings.
2. Enable developer options.
3. Enable USB debugging.
4. Connect the tablet over USB.
5. Accept the debugging prompt on the tablet.
6. Confirm the device is visible:

```bash
adb devices
```

Install the debug APK:

```bash
adb install -r android_client/app/build/outputs/apk/debug/app-debug.apk
```

For kiosk-like smoke tests, also review Fire OS settings for screen timeout,
battery optimization, Wi-Fi sleep, and app background restrictions.
The terminal chrome includes local keep-awake, fullscreen, and bright-display
toggles for kiosk-style use. They are stored on device and remain generic
terminal behavior.

## Run

1. Start the server on the same LAN:

```bash
make run-server
```

2. Open Terminals on the Android device.
3. Enter a manual endpoint such as:

```text
192.168.1.50:50051
```

The native Android client validates manual endpoints, manages the generic
control-session lifecycle, sends protobuf-backed hello/capability/action
messages, surfaces local diagnostics and permission education, and renders
server-driven UI primitives.

The client can also start Android NSD/mDNS discovery for `_terminals._tcp.`
services. Discovered servers are shown as selectable endpoint options when the
network supports multicast; Fire OS or isolated Wi-Fi networks may still require
manual endpoint entry.

If the active control session is lost during heartbeat, the client closes the
failed stream and performs bounded reconnect attempts using exponential backoff.
Retry attempt, success, and exhaustion status are recorded in local diagnostics.

While connected, the client also sends periodic battery sensor telemetry on the
control stream (same fields and pacing as the Flutter reference client: about
every 15 seconds when battery capability is present). As with Flutter, periodic
heartbeat and sensor telemetry pause while the activity is stopped (app not
visible); the control session stays open. On each foreground/background
transition, the client sends a capability delta with reason `app-lifecycle-change`
(matching Flutter `app_lifecycle_change`). Network-callback capability refreshes
are suppressed while stopped so background network flapping does not spam the
control stream.

The APK declares microphone and camera permissions so capability reporting and
future media capture can reflect runtime permission state. WebRTC media
transport remains explicitly disabled until the dependency compatibility pass is
complete; the client reports that status in local diagnostics and does not
advertise unsupported media send/receive behavior.

Discovery, media transport, kiosk, and connected-device behavior continue to
mature under `plans/features/android-client/plan.md`.

## Boundary Rules

Production Android code is scanned for:

- scenario names and package-id branching,
- forbidden Google service dependencies,
- renderer imports of connection, discovery, media internals, or platform APIs.

Run:

```bash
./scripts/check-android-client-boundary.sh
./scripts/test-android-client-boundary.sh
```

## Troubleshooting

If `adb devices` shows no device, reconnect USB, approve the tablet prompt, and
verify platform tools are from the configured Android SDK.

If `adb` is not found, add Android platform tools to `PATH` (for example:
`$ANDROID_SDK_ROOT/platform-tools`).

If install fails, remove any existing incompatible package:

```bash
adb uninstall com.curtcox.terminals.android
adb install -r android_client/app/build/outputs/apk/debug/app-debug.apk
```

If discovery later fails on Fire OS, use manual endpoint entry first. Some Wi-Fi
networks block multicast or isolate clients.

### Discovery (NSD / mDNS) quirks

The client scans for `_terminals._tcp.` via Android `NsdManager`. Copyable
diagnostics include a short hint for known failure codes (for example
`internal_error`, `already_active`, `max_limit`). Typical causes:

- **Guest or “client isolation” Wi‑Fi** — APs often block mDNS between devices;
  connect the tablet and server to the same non-isolated LAN or use manual
  endpoint.
- **Multicast filtering** — some mesh or corporate networks drop multicast;
  manual endpoint still works when TCP to the server is allowed.
- **Fire OS** — NSD can be flaky depending on OS build and power state; manual
  endpoint is the supported fallback and matches the generic-terminal contract
  (discovery is optional convenience, not required for operation).

If Gradle cannot find the SDK, set `ANDROID_SDK_ROOT` or `ANDROID_HOME`.
