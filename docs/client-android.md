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

`make android-client-build`, `android-client-test`, `android-client-lint`, and `android-client-compile-android-test` resolve a JDK automatically in this order: a working `JAVA_HOME` if set, Homebrew `openjdk@17`, Android Studio’s JBR under `Applications` or `~/Applications`, common Linux OpenJDK 17 paths, then macOS `/usr/libexec/java_home`. If no JDK is found, those targets skip with an explicit message instead of invoking Gradle with an empty `JAVA_HOME`.

### Apple Silicon and gRPC code generation

The protobuf Gradle task uses `io.grpc:protoc-gen-grpc-java` from Maven Central. Those macOS plugin binaries are **x86_64** (they rely on Rosetta on Apple Silicon). If `./gradlew` fails during `:app:generateDebugProto` with `bad CPU type` / `program not found or is not executable`, use one of:

1. **Rosetta 2** (no repo config): `softwareupdate --install-rosetta` (or install any x86 app and accept the Rosetta prompt), then re-run Gradle.
2. **Native plugin from Homebrew**: `brew install protoc-gen-grpc-java`, then either rely on auto-detection of `/opt/homebrew/bin/protoc-gen-grpc-java` (and `/usr/local/bin/protoc-gen-grpc-java` on Intel Homebrew), or set an explicit path in `android_client/local.properties`:
   ```properties
   grpc.java.plugin=/opt/homebrew/bin/protoc-gen-grpc-java
   ```
3. **Environment override** (CI or custom installs): `export GRPC_JAVA_PLUGIN=/absolute/path/to/protoc-gen-grpc-java`

Keep the Homebrew plugin reasonably close to the `io.grpc:grpc-bom` version in `app/build.gradle.kts` (same minor series is usually fine). Linux CI hosts use the Maven artifact as-is.

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
make android-client-compile-android-test
make android-client-boundary
```

`make android-client-test` and `make android-client-connected-test` run `./gradlew --stop`
after Gradle finishes so JUnit worker processes and daemons do not keep burning CPU if a
run was interrupted or overlapped with another. To stop daemons without running tests,
use `make android-client-gradle-stop`. Direct `./gradlew` invocations do not add this step.

Direct Gradle commands:

```bash
cd android_client
./gradlew testDebugUnitTest
./gradlew lintDebug
./gradlew compileDebugAndroidTestKotlin
```

Connected-device tests require an emulator or physical device:

```bash
adb devices
make android-client-connected-test
```

If `adb` is missing or no device is attached, the Make target skips with a clear
message instead of failing Android client validation.

Pull requests that touch `android_client/` (and related paths) also run
`connectedDebugAndroidTest` on an API 30 emulator in GitHub Actions (see
`.github/workflows/android-client-ci.yml`), in addition to JVM unit tests and
instrumentation compile checks.

Focused instrumentation runs (examples):

```bash
cd android_client
./gradlew connectedDebugAndroidTest --tests '*MainActivityLaunch*'
./gradlew connectedDebugAndroidTest --tests '*MainActivityConfiguration*'
./gradlew connectedDebugAndroidTest --tests '*Kiosk*'
./gradlew connectedDebugAndroidTest --tests '*Media*'
./gradlew connectedDebugAndroidTest --tests '*connectedDebug*'
```

The `*connectedDebug*` filter runs `AndroidTerminalAppSmokeTest` cases that assert connected-shell debug actions (runtime/device status, playback artifacts/metadata, scenario registry, open application) reach the control session. Additional cases cover the first inbound `RegisterAck` automatic `scenario_registry` query (Flutter parity) and a matching `CommandResult` that updates the application-intent list; examples: `./gradlew connectedDebugAndroidTest --tests '*connectedRegisterAck*'`, `./gradlew connectedDebugAndroidTest --tests '*ScenarioRegistryCommandResult*'`.

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
The terminal chrome includes local keep-awake, fullscreen, immersive-sticky
preference (controls how hidden system bars reappear while fullscreen is on),
and bright-display toggles for kiosk-style use. They are stored on device and
remain generic terminal behavior. Server-driven `FullscreenWidget` commands use
the same immersive preference when enabling fullscreen.

## Run

1. Start the server on the same LAN:

```bash
make run-server
```

2. Open Terminals on the Android device.
3. Enter a manual endpoint. Use an explicit scheme so the client picks the right
   transport:

```text
grpc://192.168.1.50:50051
http://192.168.1.50:50054/control
```

Bare `host:port` defaults to HTTP and the WebSocket upgrade (same as before).
`grpc://` / `grpcs://` use the gRPC control stream (plaintext / TLS). Discovery
metadata that only advertises `grpc=host:port` is normalized to `grpc://…` when
you select the server.

WebSocket connects send `TransportHello.resume_token` from the last successful
transport hello acknowledgement (same resume semantics as the Flutter web shell),
so brief reconnects can offer session resumption when the server supports it.
The `grpc://` / `grpcs://` carrier opens the protobuf control stream directly and
does not perform that envelope handshake, so the resume hint applies only to
WebSocket (and other envelope transports), not to gRPC.

The native Android client validates manual endpoints, manages the generic
control-session lifecycle, sends protobuf-backed hello/capability/action
messages, surfaces local diagnostics and permission education, and renders
server-driven UI primitives.

A `TextInputWidget` with component id `terminal_input` streams insertions,
backspaces, and IME newline to the server as protobuf `InputEvent.key` payloads
(the same shell convention as the Flutter client’s `terminal_input` binding),
rather than emitting a `submit` UI action on Done.

The shell **Report bug** control files an on-device bug report on the control
stream as protobuf `Diagnostics.BugReport` (same token-word scheme as the
Flutter client). When the activity window has non-zero size, the shell attaches
a best-effort PNG of the decor view (`screenshot_png` plus `screenshot_byte_count`
in source hints), matching the Flutter shell’s RepaintBoundary capture. Empty
captures omit the field. Reports filed while disconnected are queued and sent after the
next successful connect. Server-driven actions whose `action` starts with
`bug_report` (optional `bug_report:<subject-device-id>`) file a report instead of
emitting a `UIAction`. `BugReportAck` responses are merged into copyable
diagnostics like other terminal chrome.

**Privacy** (Flutter `privacy.toggle` parity): the shell **Privacy** button and
any server-driven action with `action` `privacy.toggle` toggle local privacy mode
(withdraws microphone and camera from the next capability snapshot/delta; when
**enabling** privacy, stops local capture via the live-media seam first; does not
send a `UIAction`). Toggles
while connected request a capability delta with reason `privacy.toggle`.
Copyable diagnostics include `privacy_mode=true|false`.

While connected, the shell exposes **Runtime status**, **Device status**,
**List playback artifacts**, and **Playback metadata** actions (Flutter
`terminal_client_shell` parity). The first two send `COMMAND_KIND_SYSTEM`
intents `runtime_status` and `device_status <deviceId>`. **List playback
artifacts** sends `COMMAND_KIND_SYSTEM` / `list_playback_artifacts`. **Playback
metadata** sends `COMMAND_KIND_MANUAL` / `playback_metadata` with
`artifact_id` and `target_device_id` (map + typed string arguments); when the
target field is empty, it defaults to this device id like Flutter. Successful
system sends append `last_system_command=`; manual playback metadata appends
`last_manual_command=playback_metadata:<requestId>`. Server replies surface
through existing `CommandResult` diagnostics when present: non-empty `typed_data`
entries are merged to a string map (same precedence as Flutter
`commandResultDataMap`); otherwise the legacy `data` map is used. When that map
is non-empty, the shell classifies the reply by matching `request_id` to the
pending debug query ids or by known `notification` strings, then updates the
**Open application** intent list (`scenario_registry`), pre-fills the playback
artifact field (`list_playback_artifacts`), or clears the matching pending id
(`runtime_status`, `device_status`, `playback_metadata`), matching Flutter
`commandDiagnosticsFromResponse`.

**Open application** sends the manual start command for the dropdown-selected
intent only after the control stream has received `RegisterAck` (Flutter
`_isConnectionRegistered` / `_pendingLaunchApplicationIntent` parity). If the
user taps **Open application** first, the intent is queued and flushed on the
next `RegisterAck`; copyable diagnostics include
`application_launch_queued_until_register_ack=<intent>` until then.

Server `Notification` control responses post a status-bar notification when
`POST_NOTIFICATIONS` is granted (Android 13+) and speak the message with
on-device TTS, matching the Flutter client’s notification-plus-speech alert
path (body if non-empty, otherwise title). Blank title and body are ignored.

The client can also start Android NSD/mDNS discovery for `_terminals._tcp.`
services. Discovered servers are shown as selectable endpoint options when the
network supports multicast; Fire OS or isolated Wi-Fi networks may still require
manual endpoint entry.

If the active control session is lost during heartbeat, send failures (including
failed `StreamReady`), WebSocket read errors, or gRPC stream termination
(`onError` / server half-close), the client closes the failed transport and
performs bounded reconnect attempts using exponential backoff.
Retry attempt, success, and exhaustion status are recorded in local diagnostics.

While connected, the client sends an initial heartbeat and battery sensor sample
right after the hello/capability snapshot succeeds when foregrounded (Flutter
bootstrap parity), then periodic telemetry on the control stream (same fields and
pacing as the Flutter reference client: about every 15 seconds when battery
capability is present). Copyable diagnostics include `outbound_heartbeat_count`,
`last_outbound_heartbeat_unix_ms`, `outbound_sensor_send_count`,
`last_outbound_sensor_unix_ms`, and `stream_ready_send_count` (stream-ready
acks after `StartStream`). While foregrounded, it
also mirrors Flutter’s `terminal_client_shell` capability monitor timer (about
every 2 seconds from production dependencies): each tick probes the capability
session and sends a `capability_delta` with reason `runtime_monitor_poll` when
the snapshot changed (permission drift, battery, etc.), without waiting for an
`Activity` lifecycle callback. When the server sends
`StartStream` with a non-empty `stream_id`, the client acknowledges with the same
`StreamReady` control payload as the Flutter shell so generic streaming hooks can
progress. It then forwards `StartStream`, `StopStream`, `RouteStream`, and
`WebRTCSignal` responses through the `AndroidMediaEngine` live-media seam
(`AndroidLiveMediaSession`), matching Flutter’s media-engine hooks; while WebRTC
remains disabled, `StartStream` surfaces a `last_live_media=` diagnostic with the
adapter reason (or a not-implemented placeholder when the adapter reports
supported). As with Flutter, periodic
heartbeat and sensor telemetry pause while the activity is stopped (app not
visible); the control session stays open. On each foreground/background
transition, the client sends a capability delta with reason `app_lifecycle_change`
(same string as the Flutter reference client). On `Activity` configuration changes
that affect display metrics, capability refresh uses `display_geometry_change`,
also matching Flutter. The same transitions refresh copyable network and
permission education lines together (`last_network_refresh` and
`last_permission_refresh`) so neither overwrites the other. Network-callback capability refreshes
and automatic discovery restarts are suppressed while stopped so background
network flapping does not spam the control stream or thrash NSD.

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
