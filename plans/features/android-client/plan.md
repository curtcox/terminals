---
title: "Android Client"
kind: plan
status: building
owner: curtcox
validation: automated
last-reviewed: 2026-05-09
---

# Android Client

Target repository path: `plans/features/android-client/plan.md`

Extends `masterplan.md`, `docs/client-architecture.md`, `docs/client-web.md`, `docs/client-macos.md`, and the existing Flutter client under `terminal_client/`. Preserves the repository rule from `AGENTS.md`, `CLAUDE.md`, and `masterplan.md`:

> Add behavior on the server, not the client. The client remains a generic terminal.

This plan adds a native Android client capable of running on Kindle Fire tablets while preserving the client contract: the client reports capabilities, renders server-driven UI, forwards input/media/sensor events, and executes server-issued IO commands. The server remains authoritative for scenarios, policy, routing, orchestration, and application semantics.

## Problem

The repository currently has a Flutter client in `terminal_client/` and a Makefile target named `client-build-android`, but that target builds the Flutter APK and does not define a native Android client architecture, Fire OS compatibility contract, Android module layout, Kindle Fire installation flow, or Android-specific validation strategy.

Kindle Fire tablets are important because they are low-cost, always-on home terminals with touch screens, speakers, microphones, cameras on some models, and useful wall/tabletop form factors. They also require explicit treatment:

1. Fire OS devices span Android API levels and often lag current Android releases.
2. Fire tablets should work without Google Play Services or Play Store assumptions.
3. Kiosk-like operation needs keep-awake behavior, reconnect diagnostics, permission handling, and network-loss recovery.
4. Native Android integration should expose platform capabilities directly instead of treating Kindle Fire as only another Flutter packaging target.
5. The Android client must stay protocol-compatible with existing clients so server-side feature work does not require per-client scenario edits.

Without this plan, Android work risks becoming either a packaging-only task or a second client implementation that accumulates scenario-specific behavior.

## Goals

- Add a first-class native Android project under `android_client/`.
- Support Kindle Fire tablets running Fire OS 6 or newer as the initial target class.
- Avoid Google Play Services, Firebase, Google Nearby, Google Cast, Play Integrity, and Play Store-only APIs.
- Generate and consume protobuf classes from `api/terminals/**`.
- Implement the generic terminal contract: discovery/manual connect, control stream, hello, capability snapshot, capability deltas, heartbeat, server-driven UI rendering, UI action dispatch, notifications, diagnostics, and media hooks where supported.
- Preserve server authority over scenarios, routing, policy, and orchestration.
- Add Android unit, Compose, instrumentation, and Fire tablet smoke tests.
- Add Makefile validation targets and Android documentation.
- Keep the existing Flutter client unchanged except for shared docs/tooling updates.

## Non-Goals

- No replacement of the Flutter client.
- No server protocol change unless a separate protocol evolution plan is created.
- No scenario-specific Android screens, branches, package IDs, automation, or local policy.
- No Google Play Services, Firebase Cloud Messaging, Google sign-in, Google Nearby, Google Cast, or Play Store dependency.
- No Amazon account, Alexa, in-app purchase, or Amazon Device Messaging integration in the initial client.
- No Fire TV, Wear OS, Android Auto, ChromeOS, or phone-first UX scope.
- No local scenario engine, TAL runtime, AI backend, or server orchestration in Android.
- No Fire OS 5 support unless a later hardware inventory creates a follow-up plan.
- No visual redesign of server-driven content.

## Design Principles

1. **Android is a generic terminal.** It reports capabilities and executes server commands. It does not decide what scenario runs.
2. **Protobuf is the only wire contract.** Android uses generated protobuf classes; it does not define JSON mirrors or handwritten duplicate message models.
3. **Renderer parity first.** Android should render the same closed server-driven primitive set before adding platform-specific polish.
4. **Fire OS means Google-free.** Dependencies must work on Amazon tablets without Google Play Services.
5. **Platform APIs stay behind adapters.** Permissions, notifications, wake locks, discovery, media, WebRTC, clipboard, network state, and device info are isolated from renderer and connection code.
6. **Capabilities are runtime truth.** Hardware, permissions, API level, orientation, display metrics, privacy state, and media availability determine advertised capabilities.
7. **Kiosk behavior is terminal chrome.** Keep-awake, reconnect, diagnostics, pairing, permissions, and local status are client-owned operational behavior, not scenario logic.
8. **Automated validation is required.** Pure logic gets unit tests; UI gets Compose tests; device behavior gets instrumentation or documented smoke tests.
9. **No big-bang media requirement.** Control plane and UI can land before WebRTC/media, provided unsupported media is not advertised.

## Current State

### Repository architecture

The repository has:

- `terminal_server/`: Go server.
- `terminal_client/`: Flutter client.
- `api/terminals/`: protobuf definitions.

Repository rules state that clients are generic terminals, server code owns scenarios and orchestration, and protobuf is the canonical client/server contract.

### Existing client contract

`docs/client-architecture.md` defines the client responsibilities:

- discover or manually connect to a server,
- open a control stream and identify itself,
- send a capability snapshot baseline,
- publish capability deltas as runtime conditions change,
- execute server-issued IO/UI commands,
- stream input, sensor, and media data as directed.

The current Flutter `main.dart` is already a thin entry point that imports `TerminalClientApp` and calls `runApp(const TerminalClientApp())`.

### Existing client seams

The Flutter client has dependency seams for control clients, capability probes, media engine, audio playback, alert delivery, wake-word controller, screenshot capture, time provider, screen metrics, and media permission probing. The Android client should provide equivalent seams rather than hard-wiring platform services into UI code.

### Existing build gates

The Makefile already contains `client-test`, `client-lint`, `client-build-web`, `client-build-android`, `client-build-all`, and `all-check`. The existing `client-build-android` target builds the Flutter APK. Native Android should receive separate targets, for example `android-client-build`, `android-client-test`, `android-client-lint`, and `android-client-connected-test`.

## Target Layout

```text
android_client/
  settings.gradle.kts
  build.gradle.kts
  gradle.properties
  gradlew
  gradlew.bat
  gradle/wrapper/

  app/
    build.gradle.kts
    proguard-rules.pro
    src/
      main/
        AndroidManifest.xml
        java/com/curtcox/terminals/android/
          MainActivity.kt
          TerminalAndroidApplication.kt

          app/
            AndroidTerminalApp.kt
            AndroidTerminalViewModel.kt
            AndroidClientDependencies.kt
            AndroidTerminalViewState.kt

          connection/
            AndroidControlSessionController.kt
            AndroidControlClient.kt
            GrpcAndroidControlClient.kt
            WebSocketAndroidControlClient.kt
            CarrierPreference.kt
            EndpointResolution.kt
            TransportDiagnostics.kt
            ReconnectPolicy.kt
            ControlResponseDispatcher.kt

          discovery/
            AndroidNsdDiscovery.kt
            ManualEndpointParser.kt
            DiscoveredServer.kt

          capabilities/
            AndroidCapabilityProbe.kt
            AndroidCapabilitySession.kt
            AndroidScreenMetrics.kt
            PermissionCapabilityMonitor.kt
            PowerCapabilityMonitor.kt

          ui/
            ServerDrivenRenderer.kt
            ServerDrivenAction.kt
            RendererPolicy.kt
            PrimitiveProps.kt
            NodeKey.kt
            widgets/
              LayoutWidgets.kt
              TextWidgets.kt
              MediaWidgets.kt
              InputWidgets.kt
              OverlayWidgets.kt
              DeviceControlWidgets.kt

          diagnostics/
            AndroidClientChrome.kt
            AndroidBuildMetadata.kt
            AndroidBugReportChrome.kt
            DiagnosticClipboard.kt

          media/
            AndroidMediaEngine.kt
            AndroidAudioPlayback.kt
            AndroidWebRtcAdapter.kt
            AndroidMediaPermissionProbe.kt

          platform/
            AndroidKeepAwakeController.kt
            AndroidNotificationDelivery.kt
            AndroidClipboard.kt
            AndroidNetworkState.kt
            AndroidPermissionRequester.kt
            FireOsDeviceInfo.kt

          util/
            CoroutineDispatchers.kt
            Clock.kt
            Logger.kt

      test/java/com/curtcox/terminals/android/
        connection/
        capabilities/
        ui/
        diagnostics/

      androidTest/java/com/curtcox/terminals/android/
        smoke/
        ui/
        permissions/

  README.md

docs/
  client-android.md

scripts/
  check-android-client-boundary.sh
  test-android-client-boundary.sh
```

The default package name should be stable and generic:

```text
com.curtcox.terminals.android
```

If an Amazon Appstore package name or flavor is needed later, use Gradle flavors and do not fork client behavior.

## Module Contracts

### `android_client/app`

Owns process, activity, root Compose composition, dependency graph, and lifecycle binding.

Responsibilities:

- Create `Application`, `Activity`, root terminal shell, and dependency bundle.
- Bind Android lifecycle events to terminal session start/stop/pause/resume behavior.
- Expose test seams equivalent to the Flutter app constructor seams.
- Combine client chrome and server-driven content.

Does not:

- Render protobuf UI nodes directly in `MainActivity`.
- Decide carrier ordering in Compose widgets.
- Own scenario policy or server orchestration.

### `android_client/connection`

Owns control transport, stream lifecycle, reconnect policy, request building, and response dispatch.

Responsibilities:

- Define `AndroidControlClient`.
- Implement gRPC bidirectional control stream.
- Implement WebSocket fallback if required by Fire OS/network compatibility.
- Parse manual and discovered endpoints.
- Apply carrier preference based on server-advertised priority and local runtime support.
- Send hello, capability snapshot, capability delta, heartbeat, UI action, sensor telemetry, and media control messages.
- Dispatch incoming server responses to renderer state, diagnostics, notifications, media, and capability subsystems.

Does not:

- Render UI primitives.
- Parse visual props.
- Branch on scenario names or application package IDs.
- Depend on Google services.

### `android_client/discovery`

Owns LAN discovery and manual fallback.

Responsibilities:

- Use Android NSD/mDNS where available.
- Provide reliable manual `host:port` and URL parsing.
- Surface discovered names, addresses, endpoints, freshness, and errors.
- Restart discovery safely across Wi-Fi/network changes.

Does not:

- Require Google Nearby, Bluetooth pairing, or cloud rendezvous.
- Treat discovered metadata as scenario instructions.

### `android_client/capabilities`

Owns Android capability truth and generation tracking.

Responsibilities:

- Build `DeviceCapabilities` snapshots from hardware, permissions, API level, and runtime state.
- Report display size, density, orientation, safe-area/cutout insets, touch, audio output, microphone, camera, notification ability, keep-awake support, and network state when represented by the protobuf contract.
- Emit deltas when permissions, orientation, display metrics, network state, or privacy state change.
- Gate microphone/camera capabilities on both hardware and runtime permission.
- Rebaseline after stale-generation protocol errors.

Does not:

- Advertise unsupported or Google-dependent capabilities.
- Invent local scenario capabilities.

### `android_client/ui`

Owns server-driven UI rendering in Jetpack Compose.

Primary API shape:

```kotlin
data class ServerDrivenAction(
    val componentId: String,
    val action: String,
    val value: String = "",
)

@Composable
fun ServerDrivenRenderer(
    root: terminals.ui.v1.Node,
    onAction: (ServerDrivenAction) -> Unit,
    mediaSurface: @Composable (trackId: String) -> Unit = {},
    imageLoader: @Composable (url: String, contentDescription: String?) -> Unit,
    policy: RendererPolicy = RendererPolicy.default(),
)
```

Responsibilities:

- Render every current `terminals.ui.v1.Node` widget variant.
- Apply generic visual props through `PrimitiveProps`.
- Emit `ServerDrivenAction` for interactive widgets.
- Delegate image loading and media surfaces through injected adapters.
- Provide stable node keys/test tags.
- Apply one fallback policy for malformed or unsupported nodes.

Must not:

- Open sockets.
- Access Android permissions directly.
- Send protobuf requests directly.
- Know scenario names.
- Import connection, discovery, server orchestration, or media engine internals.

### `android_client/diagnostics`

Owns Android-local terminal chrome and debugging surfaces.

Responsibilities:

- Display connection state, endpoint attempts, carrier failures, build metadata, capability summary, permission state, and last control error.
- Provide copyable diagnostics text.
- Provide generic bug-report chrome when server bug-report protocol is available.

Does not:

- Render scenario-specific dashboards.
- Interpret scenario package IDs except as opaque server-provided diagnostics.

### `android_client/media`

Owns Android media execution adapters.

Responsibilities:

- Provide audio playback for explicit server notifications or media commands.
- Probe/request microphone and camera permissions.
- Implement WebRTC adapter when Fire tablet dependency compatibility is confirmed.
- Report unsupported media deterministically and avoid advertising unavailable media capabilities.

Does not:

- Decide media routing.
- Start media without server command.
- Depend on Google media services.

### `android_client/platform`

Owns Android and Fire OS platform APIs.

Responsibilities:

- Keep screen awake while terminal/kiosk policy requests it.
- Deliver Android notifications where permission allows.
- Access clipboard for diagnostics.
- Inspect SDK level, manufacturer/model, and Fire OS-like build characteristics.
- Isolate permission prompts, lifecycle effects, and network state.

Does not:

- Leak Android framework calls into renderer or protocol code.
- Add Amazon account or Google account dependencies.

## Android and Kindle Fire Compatibility Contract

Initial target:

- `minSdk`: 25, covering Fire OS 6 and newer.
- `targetSdk`: current stable Android SDK used by the build environment.
- `compileSdk`: current stable Android SDK used by the build environment.
- Runtime guards for APIs newer than API 25.
- No `maxSdkVersion`.
- No Google Play Services dependency.
- No Play Integrity, SafetyNet, FCM, Firebase, Google Maps, Google Cast, or Nearby dependency.
- APK installs through Android Studio/ADB and Fire tablet sideloading.
- Release packaging remains Amazon Appstore-compatible, but Appstore submission is out of scope.

If Fire OS 5 support becomes required, create a follow-up plan to lower `minSdk` to 22 and audit dependencies/API calls.

## Server-Driven UI Primitive Contract

| Proto widget | Android renderer behavior | Required test |
| --- | --- | --- |
| `StackWidget` | Compose `Box`/stack preserving child order. | child order and test tags |
| `RowWidget` | Horizontal layout. | layout smoke test |
| `GridWidget` | Fixed-column grid. | column count honored |
| `ScrollWidget` | Vertical/horizontal scroll container. | scroll behavior |
| `PaddingWidget` | Applies parsed padding. | padding honored |
| `CenterWidget` | Centers child content. | center wrapper |
| `ExpandWidget` | Expands within flex-like parent. | weight/fill behavior |
| `TextWidget` | Text, style, color, semantic label. | text and style mapping |
| `ImageWidget` | Delegates URL loading. | URL passed to loader |
| `VideoSurfaceWidget` | Delegates to media surface by track ID. | track ID preserved |
| `AudioVisualizerWidget` | Generic visualizer or delegated stream surface. | stream ID preserved |
| `CanvasWidget` | Renders supported draw ops or safe fallback. | malformed draw ops safe |
| `TextInputWidget` | Text field and value action. | input emits action |
| `ButtonWidget` | Button and action. | tap emits action |
| `SliderWidget` | Slider and value change. | value emitted |
| `ToggleWidget` | Switch and value change. | value emitted |
| `DropdownWidget` | Selection control and value. | selection emitted |
| `GestureAreaWidget` | Generic gesture action. | tap emitted |
| `OverlayWidget` | Overlay children generically. | overlay path |
| `ProgressWidget` | Progress indicator. | value bounds behavior |
| `FullscreenWidget` | Delegates intent through platform effect channel. | no scenario behavior |
| `KeepAwakeWidget` | Delegates keep-awake request through platform adapter. | adapter invoked |
| `BrightnessWidget` | Delegates brightness request when permitted. | adapter invoked or diagnostic |

## Explicit Boundary Rules

1. `android_client/app/**` may compose lifecycle, dependency injection, and terminal shell chrome.
2. `android_client/ui/**` may import Compose and generated UI protobufs.
3. `android_client/ui/**` may emit `ServerDrivenAction` but may not send protobuf requests.
4. `android_client/connection/**` may translate `ServerDrivenAction` to protobuf `UIAction`.
5. `android_client/ui/**` may not import discovery, media engine internals, Android permissions, server orchestration, REPL, TAL, or scenario modules.
6. `android_client/**` production code may not branch on scenario names or server application package IDs.
7. Android platform APIs must be accessed through focused adapters.
8. Google service dependencies are forbidden unless a follow-up plan changes the Fire OS compatibility contract.
9. New UI primitives require Android renderer tests and cross-client parity review.

## Implementation Phases

### Phase 0: Characterization and Scaffolding Decision

Status: implemented

Tasks:

- Inventory generated protobuf packages currently consumed by `terminal_client/`.
- Decide whether Kotlin protobuf generation runs from `api/buf.gen.yaml` or the Gradle protobuf plugin.
- Choose Android Gradle Plugin, Kotlin, Compose BOM, protobuf, gRPC, coroutines, and test dependency versions.
- Confirm all dependencies support `minSdk 25` and avoid Google Play Services.
- Decide package name and app display name.
- Record initial Kindle Fire hardware targets.

Acceptance criteria:

- Build/protobuf strategy is documented before implementation code lands.
- Fire OS assumptions are explicit.
- Dependency choices include a Google-service exclusion check.

Validation:

```bash
./scripts/generate-plans-index.py
```

### Phase 1: Native Android Project Skeleton

Status: implemented

Tasks:

- Add `android_client/` Gradle project and `app` module.
- Enable Kotlin and Jetpack Compose.
- Add Android manifest with baseline permissions: `INTERNET`, `ACCESS_NETWORK_STATE`, `ACCESS_WIFI_STATE`, notification permission guarded by API level, and media permissions only when media lands.
- Add `MainActivity`, `TerminalAndroidApplication`, and generic root shell.
- Add deterministic build SHA/date injection.
- Add `docs/client-android.md` with prerequisites, Fire tablet setup, build, install, run, test, lint, and connection instructions.
- Add Makefile targets: `android-client-build`, `android-client-test`, `android-client-lint`, and `android-client-connected-test`.

Acceptance criteria:

- Debug APK builds.
- APK launches to a generic terminal connection screen.
- Gradle dependency output has no Google service dependency.
- Documentation includes Fire developer-mode and ADB install steps.

Validation:

```bash
cd android_client && ./gradlew assembleDebug
cd android_client && ./gradlew testDebugUnitTest
cd android_client && ./gradlew lintDebug
```

### Phase 2: Shared Protobuf and Request Builders

Status: implemented

Tasks:

- Generate Kotlin/Java protobuf classes from `api/terminals/**`.
- Add protocol helpers for hello, register compatibility, capability snapshot, capability delta, heartbeat, and UI action.
- Add fixture/round-trip tests.
- Add guard for generated-code freshness when protos change.

Acceptance criteria:

- Android compiles against generated protobufs.
- No copied schemas or handwritten duplicate message models exist.
- Builder tests verify required fields and wire protocol version.

Validation:

```bash
make proto-generate
cd android_client && ./gradlew testDebugUnitTest --tests '*Protocol*'
```

### Phase 3: Connection, Discovery, and Manual Endpoint Flow

Status: implemented; connected-device validation pending

Tasks:

- Implement `AndroidControlClient`.
- Implement gRPC bidirectional stream and WebSocket fallback if needed.
- Implement manual endpoint parser.
- Implement Android NSD/mDNS discovery.
- Implement carrier preference, endpoint resolution, reconnect/backoff, heartbeat, stream cancellation, and shutdown.
- Surface transport diagnostics in local chrome.

Acceptance criteria:

- Tablet connects to local server by manual host/port.
- Discovery works on supported Wi-Fi networks or falls back clearly.
- Reconnect attempts are visible and bounded.
- Diagnostics can be copied.
- Unit tests cover endpoint parsing, carrier ordering, and error classification.

Validation:

```bash
cd android_client && ./gradlew testDebugUnitTest --tests '*connection*'
cd android_client && ./gradlew connectedDebugAndroidTest --tests '*ConnectionSmoke*'
make run-server
```

### Phase 4: Capability Snapshot and Delta Lifecycle

Status: implemented; connected-device validation pending

Tasks:

- Implement `AndroidCapabilityProbe`, `AndroidCapabilitySession`, and `AndroidScreenMetrics`.
- Report display metrics, density, orientation, safe-area/cutout, touch, audio output, mic, camera, notification, keep-awake, and network capabilities as supported by current protobufs.
- Gate sensitive capabilities on permissions.
- Emit generation-ordered deltas.
- Rebaseline after stale-generation errors.
- Add orientation, permission, and network lifecycle observers.

Acceptance criteria:

- Initial connect sends hello and capability snapshot.
- Orientation/display changes emit debounced deltas.
- Permission denial withdraws sensitive capabilities.
- Stale-generation errors trigger full snapshot rebaseline.

Validation:

```bash
cd android_client && ./gradlew testDebugUnitTest --tests '*capabilities*'
cd android_client && ./gradlew connectedDebugAndroidTest --tests '*Capability*'
```

### Phase 5: Server-Driven Compose Renderer

Status: implemented; connected-device validation pending

Tasks:

- Implement `ServerDrivenRenderer`, `PrimitiveProps`, `RendererPolicy`, and node keys/test tags.
- Implement every current UI primitive.
- Emit `ServerDrivenAction` for interactive primitives.
- Delegate image loading and media surfaces.
- Add fallback handling for malformed/unsupported nodes.
- Add Compose tests for all primitives.

Acceptance criteria:

- Renderer supports every current `uiv1.Node` variant.
- Renderer tests cover primitives and action emission.
- Renderer has no scenario branches and no connection/discovery/media-internal imports.

Validation:

```bash
cd android_client && ./gradlew testDebugUnitTest --tests '*ServerDrivenRenderer*'
cd android_client && ./gradlew connectedDebugAndroidTest --tests '*Renderer*'
```

### Phase 6: UI Action Dispatch and Response Handling

Status: implemented; connected-device validation pending

Tasks:

- Translate `ServerDrivenAction` into protobuf UI action requests.
- Dispatch `SetUI`, `UpdateUI`, and `TransitionUI` into renderer state.
- Dispatch explicit server notifications to Android notification/TTS/status adapters where supported.
- Dispatch server build metadata into diagnostics chrome.
- Add response dispatcher tests with synthetic responses.

Acceptance criteria:

- Interactive actions reach server as `UIAction` messages.
- `SetUI` replaces rendered root.
- `UpdateUI` patches target components without resetting unrelated state.
- Server notifications remain generic terminal notifications.

Validation:

```bash
cd android_client && ./gradlew testDebugUnitTest --tests '*ResponseDispatcher*'
cd android_client && ./gradlew connectedDebugAndroidTest --tests '*UiActionSmoke*'
```

### Phase 7: Fire Tablet Kiosk Hardening

Status: implemented locally; Fire tablet smoke validation pending

Tasks:

- Implement keep-screen-awake adapter.
- Add optional immersive/sticky mode as local terminal setting.
- Handle foreground/background transitions without duplicate streams.
- Add Wi-Fi/network loss diagnostics.
- Add permission education surfaces.
- Optionally remember last manual endpoint as local terminal setting.
- Document Fire tablet settings for developer mode, sideloading, battery optimization, screen timeout, and Wi-Fi sleep.

Acceptance criteria:

- Keep-awake prevents display sleep when enabled.
- Lifecycle transitions do not leak streams or reconnect loops.
- Network loss/recovery updates diagnostics and reconnect state.
- Kiosk settings are generic terminal settings.

Validation:

```bash
cd android_client && ./gradlew connectedDebugAndroidTest --tests '*Kiosk*'
```

### Phase 8: Media and Sensor Integration

Status: implemented locally; WebRTC compatibility and device validation pending

Tasks:

- Implement Android audio playback.
- Implement mic/camera permission probe.
- Implement WebRTC adapter if Fire tablet dependency compatibility is confirmed.
- Report unsupported media deterministically.
- Add permission/capability delta tests and audio notification smoke tests.

Acceptance criteria:

- Audio playback works for explicit server-issued audio/notification behavior.
- Mic/camera permission changes update capabilities.
- WebRTC is either functional or explicitly disabled with diagnostics and no false capability advertisement.

Validation:

```bash
cd android_client && ./gradlew testDebugUnitTest --tests '*media*'
cd android_client && ./gradlew connectedDebugAndroidTest --tests '*Media*'
```

### Phase 9: Boundary Enforcement, CI, and Documentation

Status: implemented; optional Android SDK gates remain host-dependent

Tasks:

- Add `scripts/check-android-client-boundary.sh`.
- Add `scripts/test-android-client-boundary.sh`.
- Scan Android production source for known scenario names and forbidden Google service dependencies.
- Add Android targets to CI/all-check only when Android SDK is available; otherwise document them as optional local gates.
- Cross-link `docs/client-android.md` from client architecture docs.
- Add troubleshooting for ADB, Fire OS permissions, discovery failure, and manual connect.

Acceptance criteria:

- Boundary scan catches scenario-name leakage in production Android code.
- Boundary scan catches obvious Google service dependencies.
- Android docs are sufficient to build, install, and connect on a Fire tablet.

Validation:

```bash
./scripts/check-android-client-boundary.sh
./scripts/test-android-client-boundary.sh
cd android_client && ./gradlew testDebugUnitTest lintDebug assembleDebug
```

## Acceptance Criteria

The Android client is acceptable when:

- `android_client/` exists as a native Android project.
- Debug APK builds without Google Play Services.
- APK installs and launches on at least one Kindle Fire tablet running Fire OS 6 or newer.
- Manual connection to a local Terminals server works.
- Discovery works where supported or clearly degrades to manual entry.
- Android sends hello, capability snapshot, capability deltas, heartbeat, and UI action messages using generated protobufs.
- Android renders every current server-driven UI primitive or applies the documented fallback policy.
- Capabilities reflect actual hardware, permissions, display metrics, and runtime state.
- Transport and permission diagnostics are visible and copyable.
- Unit tests cover protocol builders, endpoint resolution, carrier preference, capability lifecycle, renderer primitives, action emission, and response dispatch.
- At least one connected-device smoke test covers launch, manual connection, server-driven UI render, and UI action dispatch.
- Production Android code contains no scenario-specific branches and no Google service dependency.
- `docs/client-android.md` exists.
- Android validation commands are documented and wired into Makefile targets.

## Validation Commands

Required local validation for Android-only PRs:

```bash
cd android_client && ./gradlew testDebugUnitTest
cd android_client && ./gradlew lintDebug
cd android_client && ./gradlew assembleDebug
./scripts/check-android-client-boundary.sh
```

Required connected-device validation for platform, permission, discovery, media, or renderer instrumentation changes:

```bash
adb devices
cd android_client && ./gradlew connectedDebugAndroidTest
```

Required cross-repo validation when touching protobufs or shared client/server contracts:

```bash
make proto-generate
make proto-contract-test
make server-test
make client-test
cd android_client && ./gradlew testDebugUnitTest
```

Optional full validation when Android SDK is installed:

```bash
make all-check
make android-client-build
make android-client-test
make android-client-lint
```

## Current Validation Evidence

Last local validation: 2026-05-09 (boundary scripts on agent host).

Passed:

```bash
./scripts/check-android-client-boundary.sh
./scripts/test-android-client-boundary.sh
git diff --check
```

Gradle (`make android-client-test`, `lintDebug`, `assembleDebug`) should be re-run on a host with JDK 17 and `ANDROID_SDK_ROOT` configured; plain `./gradlew` without `JAVA_HOME` may fail on macOS stubs.

Remaining validation:

- Run `cd android_client && ./gradlew connectedDebugAndroidTest` on an emulator or Fire tablet.
- Smoke-test manual connection to `make run-server` on a physical Fire OS 6+ tablet.
- Confirm Android NSD discovery behavior on a multicast-capable Wi-Fi network and document Fire OS fallback behavior.
- Complete the WebRTC dependency compatibility decision before advertising live media send/receive capabilities.

## Implementation Progress

### 2026-05-06

- Added a dependency-free Android WebSocket control carrier implementation that performs the RFC 6455 upgrade, sends the required transport hello, wraps outgoing `ConnectRequest` messages in protobuf `WireEnvelope` binary frames, tracks session/sequence metadata, handles ping/close frames, and exposes incoming `ConnectResponse` messages through a generic response sink.
- Added focused WebSocket frame codec tests for masked and extended-length binary frames.
- Verified Android client boundary scan with `./scripts/check-android-client-boundary.sh`.
- Gradle Android unit validation is currently blocked on this machine because the only installed JDK is Java 25.0.3, which the Kotlin Gradle DSL cannot parse during settings script compilation.
- Added a native Android NSD/mDNS discovery adapter for `_terminals._tcp.` services, including TXT metadata parsing for generic carrier endpoints and priority.
- Verified Android client boundary scan and boundary test with `./scripts/check-android-client-boundary.sh` and `./scripts/test-android-client-boundary.sh`.
- Attempted focused Android unit validation with `cd android_client && ./gradlew testDebugUnitTest --tests '*AndroidNsdDiscoveryTest*'`; it remains blocked by the local Java 25.0.3 Gradle/Kotlin incompatibility and requires JDK 17.
- Wired the Android terminal ViewModel manual-connect path to the generic control session factory so valid endpoints now create a WebSocket-backed session, send hello/capability snapshot through `AndroidControlSessionController`, dispatch incoming control responses into renderer state, and forward `ServerDrivenAction` values as protobuf UI actions.
- Added an `AndroidControlSession` interface seam and ViewModel unit tests covering successful connect, connect failure diagnostics, server `SetUI` dispatch, and UI action forwarding.
- Re-verified Android boundary scan and boundary tests with `./scripts/check-android-client-boundary.sh` and `./scripts/test-android-client-boundary.sh`.
- Attempted focused Android unit validation with `cd android_client && ./gradlew testDebugUnitTest --tests '*AndroidTerminalViewModelTest*'`; it remains blocked by the local Java 25.0.3 Gradle/Kotlin incompatibility and requires JDK 17.
- Replaced the placeholder static Android capability probe with a context-backed runtime probe that reports device identity, display metrics, orientation, system-bar/display-cutout safe area on API 30+, hardware features, permission-gated microphone/camera/notification state, haptics, and battery/charging state.
- Wired `MainActivity` to construct the terminal ViewModel with Android runtime dependencies and request capability deltas on activity resume and configuration changes.
- Added ViewModel coverage for capability delta refresh dispatch through the connected control session seam.
- Re-verified Android boundary scan, boundary tests, and diff whitespace checks with `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, and `git diff --check`.
- Attempted focused Android unit validation with `cd android_client && ./gradlew testDebugUnitTest --tests '*AndroidTerminalViewModelTest*'`; it remains blocked by the local Java 25.0.3 Gradle/Kotlin incompatibility and requires JDK 17.
- Wired stale capability-generation protocol errors from the Android response sink to `AndroidControlSession.rebaselineCapabilitiesAfterStaleGeneration()`, preserving generic terminal behavior while sending a fresh protobuf capability snapshot.
- Added controller and ViewModel tests covering stale-generation rebaseline dispatch, snapshot generation/status updates, and unrelated protocol errors that should not rebaseline.
- Re-verified Android boundary scan, boundary tests, and diff whitespace checks with `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, and `git diff --check`.
- Attempted focused Android unit validation with `cd android_client && ./gradlew testDebugUnitTest --tests '*AndroidTerminalViewModelTest*' --tests '*AndroidControlSessionControllerTest*'`; it remains blocked by the local Java 25.0.3 Gradle/Kotlin incompatibility and requires JDK 17.
- Wired server-driven Android `KeepAwakeWidget` rendering through an injected device-control effect, ViewModel platform seam, and concrete `Window` flag adapter so keep-awake remains generic terminal behavior rather than renderer-owned Android logic.
- Added renderer and ViewModel coverage for keep-awake effect dispatch.
- Re-verified Android boundary scan, boundary tests, and diff whitespace checks with `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, and `git diff --check`.
- Attempted focused Android unit validation with `cd android_client && ./gradlew testDebugUnitTest --tests '*AndroidTerminalViewModelTest*'`; it remains blocked by the local Java 25.0.3 Gradle/Kotlin incompatibility and requires JDK 17.
- Wired server-driven Android `FullscreenWidget` and `BrightnessWidget` through the same generic device-control effect path, ViewModel platform seams, and concrete `Window` adapters.
- Added renderer and ViewModel coverage for fullscreen and brightness effect dispatch.
- Wired generic server `Notification` responses through the Android response sink to an injected `AndroidNotificationDelivery` platform seam, added a status-bar notification adapter with Android 13 permission gating and an API 26+ notification channel, and covered ViewModel notification delivery in unit tests.
- Re-verified Android boundary scan, boundary tests, and diff whitespace checks with `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, and `git diff --check`.
- Attempted focused Android unit validation with `cd android_client && ./gradlew testDebugUnitTest --tests '*AndroidTerminalViewModelTest*'`; it remains blocked by the local Java 25.0.3 Gradle/Kotlin incompatibility and requires JDK 17.
- Wired generic server `PlayAudio` and `ShowMedia` responses through an injected `AndroidMediaEngine` seam so media commands are dispatched as terminal IO commands and unsupported media is recorded deterministically in diagnostics instead of being falsely advertised.
- Added ViewModel coverage for delegated audio playback, delegated media display, and unsupported media diagnostics.
- Re-verified Android boundary scan, boundary tests, and diff whitespace checks with `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, and `git diff --check`.
- Attempted focused Android unit validation with `cd android_client && ./gradlew testDebugUnitTest --tests '*AndroidTerminalViewModelTest*'`; it remains blocked by the local Java 25.0.3 Gradle/Kotlin incompatibility and requires JDK 17.
- Wired Android network-state diagnostics through a platform provider seam and context-backed `ConnectivityManager` adapter, including lifecycle refreshes from `MainActivity` so connection chrome records current connectivity and metered state.
- Added ViewModel coverage for network diagnostic sampling during connect and explicit network refresh.
- Re-verified Android boundary scan, boundary tests, and diff whitespace checks with `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, and `git diff --check`.
- Attempted focused Android unit validation with `cd android_client && ./gradlew testDebugUnitTest --tests '*AndroidTerminalViewModelTest*' --tests '*AndroidClientChromeTest*'`; it remains blocked by the local Java 25.0.3 Gradle/Kotlin incompatibility and requires JDK 17.
- Added a generic terminal settings seam backed by Android `SharedPreferences` so the native client restores and remembers the last valid manual endpoint without adding scenario behavior.
- Hardened the ViewModel connection lifecycle by closing any existing control session before reconnect, starting a periodic heartbeat loop after successful connect, and cancelling the heartbeat on reconnect/failure/clear.
- Added ViewModel coverage for remembered manual endpoints, reconnect session shutdown, and periodic heartbeat dispatch.
- Re-verified Android boundary scan, boundary tests, and diff whitespace checks with `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, and `git diff --check`.
- Attempted focused Android unit validation with `cd android_client && ./gradlew testDebugUnitTest --tests '*AndroidTerminalViewModelTest*'`; it remains blocked by the local Java 25.0.3 Gradle/Kotlin incompatibility and requires JDK 17.
- Added generic permission education to Android terminal chrome using the same runtime capability probe that drives advertised capabilities, so notification, microphone, and camera availability are visible without adding scenario behavior.
- Added a static default Android capability probe implementation for dependency defaults and ViewModel tests.
- Added ViewModel coverage for initial permission education and lifecycle permission refresh diagnostics.
- Re-verified Android boundary scan, boundary tests, and diff whitespace checks with `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, and `git diff --check`.
- Attempted focused Android unit validation with `cd android_client && ./gradlew testDebugUnitTest --tests '*AndroidTerminalViewModelTest*'`; it remains blocked by the local Java 25.0.3 Gradle/Kotlin incompatibility and requires JDK 17.

### 2026-05-07

- Updated phase status labels to reflect the native Android implementation now present in `android_client/`, while leaving connected-device, Fire tablet, NSD-on-real-network, and WebRTC compatibility validation explicitly pending.
- Re-verified the current local Android gate with `cd android_client && JAVA_HOME=/opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home ./gradlew testDebugUnitTest`, `cd android_client && JAVA_HOME=/opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home ./gradlew lintDebug`, `cd android_client && JAVA_HOME=/opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home ./gradlew assembleDebug`, `./scripts/check-android-client-boundary.sh`, and `./scripts/test-android-client-boundary.sh`.
- Added a context-backed Android audio playback adapter for generic server-issued `PlayAudio` commands. URL sources are played with `MediaPlayer`, TTS sources use Android `TextToSpeech`, and raw PCM remains explicitly unsupported with a diagnostic reason until the protocol supplies playback metadata.
- Wired the real audio adapter into Android runtime dependencies through the existing `AndroidMediaEngine` seam, preserving unsupported media display behavior and avoiding any scenario-specific client behavior.
- Re-verified Android boundary scan, boundary tests, and diff whitespace checks with `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, and `git diff --check`.
- Attempted focused Android unit validation with `cd android_client && ./gradlew testDebugUnitTest --tests '*AndroidTerminalViewModelTest*'`; it remains blocked by the local Java 25.0.3 Gradle/Kotlin incompatibility and requires JDK 17.
- Added explicit Android microphone/camera manifest permissions for the native client and a context-backed media permission probe that reports runtime microphone/camera grants through the Android dependency seam.
- Added an explicit WebRTC support adapter that reports deterministic disabled status until Fire OS-compatible media transport dependencies are selected, preventing false media capability advertisement.
- Surfaced media permission and WebRTC support state in ViewModel diagnostics and documentation, with unit coverage for permission/media diagnostics.
- Re-verified Android boundary scan, boundary tests, and diff whitespace checks with `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, and `git diff --check`.
- Attempted focused Android unit validation with `cd android_client && ./gradlew testDebugUnitTest --tests '*AndroidTerminalViewModelTest*'`; it remains blocked by the local Java 25.0.3 Gradle/Kotlin incompatibility and requires JDK 17.
- Fixed local Android Gradle validation by configuring user-level Gradle JVM selection in `/Users/curt/.gradle/gradle.properties` to use the existing Homebrew JDK 17 at `/opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home`.
- Re-ran focused Android unit validation with `cd android_client && ./gradlew testDebugUnitTest --tests '*AndroidTerminalViewModelTest*'`; validation now passes with plain Gradle invocation.
- Re-ran the full Android unit suite with `cd android_client && ./gradlew testDebugUnitTest`; validation passes.
- Fixed the native Android theme API boundary by moving `android:windowLightNavigationBar` from base `values/` resources into an API 27-qualified resource, preserving `minSdk 25` compatibility for Fire OS 6 devices.
- Re-verified Android boundary scan, boundary tests, diff whitespace checks, lint, and debug APK assembly with `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, `git diff --check`, and `cd android_client && ./gradlew lintDebug assembleDebug`.
- Wired native Android validation into repository-wide gates using the existing SDK-aware Make targets, so `all-lint`, `all-test`, and `all-check` run native Android lint, unit tests, and debug APK assembly when an Android SDK is configured and skip clearly otherwise.
- Fixed the documentation index to describe `docs/client-android.md` as the native Android/Kindle Fire client guide rather than the Flutter Android target.
- Added stable Compose test tags to the native Android manual endpoint and connect controls, then added an instrumentation smoke test for manual endpoint entry, fake-session connection, synthetic server-driven UI render, and UI action dispatch.
- Fixed the existing renderer instrumentation helper so the Android test source set compiles, then verified boundary scans, whitespace, instrumentation compilation, unit tests, and lint with `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, `git diff --check`, `cd android_client && ./gradlew compileDebugAndroidTestKotlin`, `cd android_client && ./gradlew testDebugUnitTest`, and `cd android_client && ./gradlew lintDebug`.
- Extended the native Android instrumentation smoke test to cover server-driven `KeepAwakeWidget`, `FullscreenWidget`, and `BrightnessWidget` dispatch through the app-level platform adapter seams, preserving generic terminal behavior for kiosk/device-control effects.
- Re-verified Android boundary scan, boundary tests, diff whitespace checks, and instrumentation compilation with `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, `git diff --check`, and `cd android_client && ./gradlew compileDebugAndroidTestKotlin`.
- Wired diagnostic copying into the native Android terminal chrome through an injected clipboard adapter seam, with a context-backed Android clipboard implementation and success/failure state surfaced in the app.
- Added ViewModel and instrumentation smoke coverage for copying the current diagnostics text from terminal chrome.
- Re-verified Android boundary scan, boundary tests, diff whitespace checks, focused ViewModel unit tests, and instrumentation compilation with `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, `git diff --check`, `cd android_client && ./gradlew testDebugUnitTest --tests '*AndroidTerminalViewModelTest*'`, and `cd android_client && ./gradlew compileDebugAndroidTestKotlin`.
- Wired bounded reconnect handling into the native Android control-session lifecycle. Heartbeat loss now closes the failed stream, schedules reconnect attempts through the existing `ReconnectPolicy`, records retry/success/exhaustion diagnostics, and restores the session through the generic session factory without adding scenario behavior.
- Added ViewModel coverage for heartbeat-triggered reconnect success and exhausted reconnect attempts.
- Re-verified focused reconnect coverage and the full Android unit suite with `cd android_client && ./gradlew testDebugUnitTest --tests 'com.curtcox.terminals.android.app.AndroidTerminalViewModelTest.heartbeatFailureReconnectsWithBoundedBackoff' --tests 'com.curtcox.terminals.android.app.AndroidTerminalViewModelTest.heartbeatFailureStopsAfterReconnectAttemptsAreExhausted'` and `cd android_client && ./gradlew testDebugUnitTest`.
- Wired Android NSD/mDNS discovery into the native client chrome through an injected discovery seam. The app can start/stop discovery, list discovered `_terminals._tcp.` servers, select a discovered endpoint, and fall back clearly to manual entry when discovery errors.
- Added ViewModel coverage for discovered-server selection and discovery error diagnostics.
- Re-verified Android boundary scan, boundary tests, diff whitespace checks, focused ViewModel unit tests, full Android unit tests, lint, and debug APK assembly with `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, `git diff --check`, `cd android_client && ./gradlew testDebugUnitTest --tests '*AndroidTerminalViewModelTest*'`, `cd android_client && ./gradlew testDebugUnitTest`, `cd android_client && ./gradlew lintDebug`, and `cd android_client && ./gradlew assembleDebug`.
- Extended native Android app smoke coverage for lifecycle/configuration-triggered capability refreshes so a connected fake control session receives the refresh reason and terminal chrome records `last_capability_delta=configuration`.
- Re-verified Android boundary scan, boundary tests, diff whitespace checks, instrumentation test source compilation, and full Android unit tests with `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, `git diff --check`, `cd android_client && ./gradlew compileDebugAndroidTestKotlin`, and `cd android_client && ./gradlew testDebugUnitTest`.
- Added focused JVM coverage for the Android media seams, including unsupported audio/display reasons, adapter delegation, and the explicit disabled WebRTC compatibility decision.
- Re-verified focused media coverage, the full Android unit suite, boundary tests, and diff whitespace checks with `cd android_client && ./gradlew testDebugUnitTest --tests '*AndroidMediaEngineTest*'`, `cd android_client && ./gradlew testDebugUnitTest`, `./scripts/test-android-client-boundary.sh`, and `git diff --check`.

### 2026-05-08

- Added native Android instrumentation smoke coverage for server-issued `PlayAudio` and `ShowMedia` control responses, verifying media commands dispatch through the `AndroidMediaEngine` seam and that terminal diagnostics/state capture the resulting media status.
- Attempted Android instrumentation test-source compilation with `cd android_client && ./gradlew compileDebugAndroidTestKotlin`; this shell session did not have `java` available, so Gradle validation was blocked.
- Made the native Android terminal chrome always surface the live media transport/WebRTC compatibility status, even when there are no permission education warnings, so Fire OS users can see that live media remains intentionally unavailable until the dependency decision is complete.
- Added Compose smoke coverage for the no-permission-warning path to ensure the disabled live media status remains visible independently of microphone, camera, or notification permission messages.
- Re-verified Android boundary scan, boundary tests, diff whitespace checks, Android instrumentation test-source compilation, the full Android unit suite, lint, and debug APK assembly with `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, `git diff --check`, `cd android_client && ./gradlew compileDebugAndroidTestKotlin testDebugUnitTest`, and `cd android_client && ./gradlew lintDebug assembleDebug`.
- Added a generic local keep-awake kiosk setting backed by Android terminal settings and the existing keep-awake platform adapter, with terminal chrome to toggle it and restore it on launch.
- Added ViewModel and instrumentation smoke coverage for restoring, persisting, and toggling the local keep-awake setting.
- Re-verified focused ViewModel tests, Android instrumentation test-source compilation, the full Android unit suite, boundary scans, diff whitespace checks, lint, and debug APK assembly with `cd android_client && ./gradlew testDebugUnitTest --tests '*AndroidTerminalViewModelTest*'`, `cd android_client && ./gradlew compileDebugAndroidTestKotlin`, `cd android_client && ./gradlew testDebugUnitTest`, `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, `git diff --check`, and `cd android_client && ./gradlew lintDebug assembleDebug`.
- Added a generic local fullscreen kiosk setting backed by Android terminal settings and the existing fullscreen platform adapter, with terminal chrome to toggle it and restore it on launch.
- Added ViewModel and instrumentation smoke coverage for restoring, persisting, and toggling the local fullscreen setting.
- Re-verified focused ViewModel tests, Android instrumentation test-source compilation, boundary scans, boundary tests, and diff whitespace checks with `cd android_client && ./gradlew testDebugUnitTest --tests '*AndroidTerminalViewModelTest*'`, `cd android_client && ./gradlew compileDebugAndroidTestKotlin`, `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, and `git diff --check`.
- Added a generic local bright-display kiosk setting backed by Android terminal settings and the existing brightness platform adapter, with terminal chrome to toggle it and restore full brightness on launch.
- Added ViewModel and instrumentation smoke coverage for restoring, persisting, and toggling the local bright-display setting.
- Documented the native Android kiosk chrome controls as keep-awake, fullscreen, and bright-display toggles in `docs/client-android.md`.
- Re-verified Android unit tests, lint, debug APK assembly, instrumentation test-source compilation, boundary scans, and diff whitespace checks with `make android-client-test`, `make android-client-lint`, `make android-client-build`, `cd android_client && ./gradlew compileDebugAndroidTestKotlin`, `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, and `git diff --check`.
- Added a context-backed Android network monitor using `ConnectivityManager.NetworkCallback` so live network availability/capability changes refresh terminal diagnostics and connected capability deltas without waiting for activity resume/configuration events.
- Added ViewModel coverage for network-callback diagnostics and capability delta dispatch through the existing generic control-session seam.
- Re-verified focused ViewModel tests, Android debug compilation, lint, boundary scans, boundary tests, and diff whitespace checks with `cd android_client && ./gradlew testDebugUnitTest --tests '*AndroidTerminalViewModelTest*'`, `cd android_client && ./gradlew compileDebugKotlin lintDebug`, `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, and `git diff --check`.
- Wired runtime Fire OS device metadata through a dedicated platform adapter seam so diagnostics now report manufacturer, model, SDK level, and whether the device is likely Fire OS instead of only static target assumptions.
- Added diagnostics/unit coverage for Fire OS metadata in both `AndroidClientChromeTest` and `AndroidTerminalViewModelTest`.
- Re-verified Android boundary scan and boundary tests with `./scripts/check-android-client-boundary.sh` and `./scripts/test-android-client-boundary.sh`; focused Gradle unit tests were not runnable in this session because `java` was unavailable in the shell environment.
- Added native Android smoke coverage for discovery fallback and discovered-server endpoint selection, verifying that discovery errors are surfaced as manual-connect guidance and that selecting a discovered server applies its endpoint and returns the client to manual connect mode.
- Re-verified Android boundary scan with `./scripts/check-android-client-boundary.sh`.
- Attempted Android instrumentation test-source compilation with `cd android_client && ./gradlew compileDebugAndroidTestKotlin`; this shell session did not have `java` available, so Gradle validation was blocked.
- Added native Android smoke coverage for malformed manual endpoints, verifying that invalid endpoint text keeps the connect action disabled, surfaces endpoint guidance in terminal chrome, and leaves the client in `InvalidEndpoint` state instead of attempting transport setup.
- Re-verified Android boundary scan and boundary tests with `./scripts/check-android-client-boundary.sh` and `./scripts/test-android-client-boundary.sh`.
- Attempted Android instrumentation test-source compilation with `cd android_client && ./gradlew compileDebugAndroidTestKotlin`; this shell session did not have `java` available, so Gradle validation remained blocked.
- Expanded native Android diagnostics chrome to include live capability-summary fields (orientation, display metrics, density, touch support, mic/camera presence and permission state, and notification permission state) so copied diagnostics reflect runtime capability truth instead of only connection/device metadata.
- Added diagnostics unit coverage for capability-summary rendering in `AndroidClientChromeTest`.
- Re-verified focused Android JVM validation with `cd android_client && JAVA_HOME=/opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home ./gradlew testDebugUnitTest --tests '*AndroidClientChromeTest*' --tests '*AndroidTerminalViewModelTest.diagnosticsIncludeFireOsDeviceInfoWhenAvailable'`, plus boundary scan/test with `./scripts/check-android-client-boundary.sh` and `./scripts/test-android-client-boundary.sh`.
- Added native Android smoke coverage for runtime capability transitions: orientation/display diagnostics now update under configuration-driven capability refresh, and runtime permission-loss refresh now surfaces notification/microphone/camera guidance while preserving generic terminal behavior.
- Re-verified Android instrumentation test-source compilation for the new smoke coverage with `cd android_client && JAVA_HOME=/opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home ./gradlew compileDebugAndroidTestKotlin`.
- Added native Android smoke coverage for heartbeat-loss reconnect flow, verifying that a failed heartbeat closes the failed control session, reconnects through the existing bounded reconnect policy, and records reconnect success diagnostics without adding scenario behavior.
- Re-verified Android boundary scan/test plus instrumentation test-source compilation with `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, and `cd android_client && JAVA_HOME=/opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home ./gradlew compileDebugAndroidTestKotlin`.
- Added native Android smoke coverage for bounded reconnect exhaustion, verifying that heartbeat-triggered reconnect failures stop after the configured attempt budget, close retry sessions, and surface `reconnect_exhausted` diagnostics.
- Re-verified reconnect exhaustion behavior and Android test/build gates with `cd android_client && JAVA_HOME=/opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home ./gradlew testDebugUnitTest --tests '*AndroidTerminalViewModelTest.heartbeatFailureStopsAfterReconnectAttemptsAreExhausted'`, `cd android_client && JAVA_HOME=/opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home ./gradlew compileDebugAndroidTestKotlin`, `./scripts/check-android-client-boundary.sh`, and `./scripts/test-android-client-boundary.sh`.
- Attempted connected-device validation with `cd android_client && JAVA_HOME=/opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home ./gradlew connectedDebugAndroidTest`; Android instrumentation build/test APK packaging succeeded but execution failed with `No connected devices!`.
- Attempted `adb devices` in this shell session, but `adb` was not available in PATH (`command not found: adb`), so connected-device smoke remains blocked on host/device setup rather than Android client code.
- Hardened `make android-client-connected-test` to preflight `adb` availability and attached devices before invoking Gradle instrumentation, so host/device setup gaps now skip with deterministic guidance instead of surfacing as Android client failures.
- Documented the connected-test preflight skip behavior and `platform-tools` PATH guidance in `docs/client-android.md`.
- Wired a real Android runtime permission-request seam into the native client using an activity-backed requester adapter, and threaded it through `MainActivity` plus the Android dependency graph so permission handling remains behind a platform adapter.
- Added terminal chrome controls for notification, microphone, and camera permission requests, and wired ViewModel callbacks to refresh permission education plus capability deltas after request results.
- Added JVM and Compose smoke coverage for the new permission-request flow, then re-verified `make android-client-test`, `make android-client-lint`, `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, and `cd android_client && JAVA_HOME=/opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home ./gradlew compileDebugAndroidTestKotlin`.
- Added a terminal-chrome "Enable missing permissions" action that requests every currently-missing runtime permission through the existing platform adapter seam, while preserving generic terminal behavior and permission-specific controls.
- Added JVM and Compose smoke coverage for grouped missing-permission requests (microphone/camera), then re-verified `cd android_client && JAVA_HOME=/opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home ./gradlew testDebugUnitTest` and `cd android_client && JAVA_HOME=/opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home ./gradlew compileDebugAndroidTestKotlin`.
- Added explicit notification-permission coverage for the grouped "Enable missing permissions" path so Android 13+ runtime notification prompts are validated alongside microphone and camera prompts in both ViewModel JVM tests and Compose smoke tests.
- Re-verified focused JVM coverage and Android instrumentation test-source compilation with `cd android_client && JAVA_HOME=/opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home ./gradlew testDebugUnitTest --tests '*AndroidTerminalViewModelTest.requestMissingPermissionsRequestsNotificationPermissionWhenRuntimePromptIsSupported'` and `cd android_client && JAVA_HOME=/opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home ./gradlew compileDebugAndroidTestKotlin`.
- Wired network-change discovery hardening in `AndroidTerminalViewModel`: when mDNS scanning is active and `AndroidNetworkMonitor` reports a callback, the client now restarts discovery automatically so Wi-Fi transitions recover without manual stop/start.
- Added focused JVM coverage for network-triggered discovery restart in `AndroidTerminalViewModelTest.networkMonitorRestartsDiscoveryWhenScanning`.
- Added restart-rate limiting for network-triggered discovery restarts so callback bursts do not thrash NSD scanning; suppressed restarts are recorded in diagnostics and covered by `AndroidTerminalViewModelTest.networkMonitorDebouncesDiscoveryRestartWhenCallbacksBurst`.
- Added matching rate limiting for network-triggered capability delta refreshes so connectivity callback bursts do not spam control traffic; suppressed refreshes are now recorded in diagnostics and covered by `AndroidTerminalViewModelTest.networkMonitorDebouncesCapabilityRefreshWhenCallbacksBurst`.
- Hardened Android diagnostics composition so permission education and media transport status are now always included in the baseline diagnostics text across connection/discovery/lifecycle transitions, instead of only after explicit permission-refresh actions.
- Added JVM coverage in `AndroidTerminalViewModelTest.baselineDiagnosticsAlwaysIncludePermissionAndMediaStatus` to lock the baseline diagnostics contract for notification/microphone/camera and WebRTC availability fields.
- Re-verified focused Android ViewModel JVM coverage with `cd android_client && JAVA_HOME=/opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home ./gradlew testDebugUnitTest --tests '*AndroidTerminalViewModelTest*'`.
- Fixed disconnect-state diagnostics in `AndroidTerminalViewModel` so manual endpoints return to `ReadyToConnect` with matching diagnostics text instead of reporting a stale `Disconnected` status.
- Added `AndroidTerminalViewModelTest.disconnectWithValidEndpointReturnsToReadyStateDiagnostics` regression coverage and re-verified with `cd android_client && JAVA_HOME=/opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home ./gradlew testDebugUnitTest --tests '*AndroidTerminalViewModelTest.disconnectWithValidEndpointReturnsToReadyStateDiagnostics' --tests '*AndroidTerminalViewModelTest.connectCreatesSessionAndMarksStateConnected'`.
- Hardened `AndroidTerminalViewModel` network monitoring lifecycle by making `startNetworkMonitoring()` and `stopNetworkMonitoring()` idempotent, preventing duplicate `AndroidNetworkMonitor` callback registration across repeated lifecycle starts.
- Added `AndroidTerminalViewModelTest.networkMonitoringStartStopIsIdempotent` regression coverage and re-verified focused ViewModel JVM tests with `cd android_client && JAVA_HOME=/opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home ./gradlew testDebugUnitTest --tests '*AndroidTerminalViewModelTest*'`.
- Implemented generic server `BugReportAck` handling: `AndroidBugReportChrome` formats copyable diagnostics lines, `ControlResponseDispatcher` records `lastBugReportAckDiagnostics` on `AndroidTerminalViewState`, and `AndroidTerminalViewModel` appends them to live diagnostics when acknowledgements arrive over the control stream (no scenario branching).
- Added JVM coverage in `AndroidBugReportChromeTest`, `ControlResponseDispatcherTest.bugReportAckRecordsDiagnosticsChrome`, and `AndroidTerminalViewModelTest.serverBugReportAckIsSurfacedInDiagnostics`, plus re-verified `./scripts/check-android-client-boundary.sh` and `./scripts/test-android-client-boundary.sh`.
- Surfaced generic server `TransitionUI` responses in copyable terminal diagnostics (`last_transition`, optional `last_transition_duration_ms` when non-zero), fixed `UpdateUI` subtree merge to detect child replacements with protobuf value equality, and added dispatcher/ViewModel coverage.
- Wired server `HelloAck` into native Android terminal state and heartbeat pacing (restart heartbeat when the server supplies a positive `heartbeat_interval_ms`), surfaced handshake lines in copyable diagnostics, and cleared handshake metadata on connect failure, disconnect, and fresh connect attempts.
- Wired server `RegisterAck` typed `ServerMetadata` plus legacy `metadata` map fallback (`server_build_sha` / `server_build_date`) into terminal diagnostics, preserving generic terminal semantics without scenario branching.
- Record server `CapabilityAck.accepted_generation` in terminal state and diagnostics for protocol debugging; extended `ControlResponseDispatcherTest` for hello/register/capability acknowledgement paths.
- Surfaced server-originated `Heartbeat` (`last_server_heartbeat_unix_ms`) and `CommandResult` (`last_command_result_request_id`, `last_command_result_notification`) responses in `ControlResponseDispatcher` and `AndroidTerminalViewModel` diagnostics. These payloads were previously dropped by the response sink. Generic terminal semantics preserved (no scenario branching, no command-action interpretation).
- Added `ControlResponseDispatcherTest.serverHeartbeatRecordsLastServerUnixMs`, `ControlResponseDispatcherTest.serverHeartbeatWithoutTimestampClearsRecordedValue`, `ControlResponseDispatcherTest.commandResultRecordsRequestIdAndNotification`, `AndroidTerminalViewModelTest.serverHeartbeatIsSurfacedInDiagnostics`, and `AndroidTerminalViewModelTest.serverCommandResultIsSurfacedInDiagnostics`.
- Cleared the new server-heartbeat and command-result fields alongside the existing handshake reset path in `AndroidTerminalViewModel.withoutHandshake`, so reconnect/disconnect transitions do not leak previous server protocol activity into fresh sessions.
- Re-verified Android boundary scan, boundary tests, and diff whitespace checks with `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, and `git diff --check`. Local Gradle unit/lint validation remained host-skipped because the Android SDK is not configured (`ANDROID_SDK_ROOT`/`ANDROID_HOME` unset, no `android_client/local.properties`) and JDK 17 is not present on this machine; `make android-client-test` reports the documented skip.

### 2026-05-08 (opaque IO diagnostics)

- Surfaced server `ConnectResponse` payloads that were previously ignored by `ControlResponseDispatcher` (`StartStream`, `StopStream`, `RouteStream`, `WebRTCSignal`, `InstallBundle`, `RemoveBundle`, `StartFlow`, `PatchFlow`, `StopFlow`, `RequestArtifact`) as generic `last_opaque_control_io` lines in copyable diagnostics; WebRTC summaries omit SDP payloads; bundle installs record bundle id/version/sha prefix/tar byte length without dumping artifact bytes.
- Cleared opaque summaries on media commands (`PlayAudio`/`ShowMedia`) and on session teardown via `withoutHandshake`; forward-unknown payload cases still record `payload_case` for protocol debugging.
- Added dispatcher and ViewModel regression coverage for opaque IO summaries.

### 2026-05-08 (transition diagnostics persistence)

- Persisted `TransitionUI` transition name and non-zero duration in `AndroidTerminalViewState`, cleared on `withoutHandshake`, and included both in baseline `formatDiagnostics` so copyable diagnostics keep the last transition after network-driven refreshes instead of only on the inbound response tick.
- Extended dispatcher and ViewModel tests (`serverTransitionUiRemainsInDiagnosticsAfterNetworkRefresh`).
- Hardened bug-report acknowledgement diagnostics across disconnect and reconnect: `lastBugReportAckDiagnostics` is cleared in `withoutHandshake` so a new connect does not show a prior session’s ack, while disconnect rebuilds copyable diagnostics with the last ack preserved for support copy/paste; centralized ack lines in `formatDiagnostics` (removed duplicate append from the control response sink). Added ViewModel coverage (`serverBugReportAckRemainsInDiagnosticsAfterDisconnect`, `newConnectClearsBugReportAckFromPriorSession`).

### 2026-05-08 (capability ack diagnostics)

- Surfaced server `CapabilityAck.snapshot_applied` and a compact `invalidations` summary in copyable terminal diagnostics (aligned with server capability-lifecycle invalidation signaling). Extended `ControlResponseDispatcher` state, `withoutHandshake` clearing, dispatcher/ViewModel tests.

### 2026-05-08 (server control error diagnostics)

- Record `ControlError.code` as `lastControlErrorCode` on `AndroidTerminalViewState`, include `last_error` and `last_control_error_code` in baseline `formatDiagnostics`, preserve error code across disconnect (like bug-report ack), and clear it on `withoutHandshake` for new sessions.
- Refactored connect failure, heartbeat loss, and reconnect diagnostics to rely on `formatDiagnostics` for `last_error` instead of ad-hoc suffixes (avoids stale or duplicate lines).
- On `HelloAck`, treat non-positive `heartbeat_interval_ms` as “use client default”: reset `effectiveHeartbeatMillis` and restart the heartbeat loop.
- Extended `ControlResponseDispatcherTest` and `AndroidTerminalViewModelTest`; re-verified `./scripts/check-android-client-boundary.sh` and `./scripts/test-android-client-boundary.sh`. Full Gradle unit tests require a host JDK (not available in this session).

### 2026-05-08 (opaque IO dispatcher JVM coverage)

- Extended `ControlResponseDispatcherTest` with coverage for `RouteStream`, `RemoveBundle`, `StartFlow`, `PatchFlow`, `StopFlow`, and `RequestArtifact` opaque diagnostic summaries; added regression tests for `CommandResult` field clearing, `ShowMedia` clearing `last_opaque_control_io`, and re-verified `./scripts/check-android-client-boundary.sh` and `./scripts/test-android-client-boundary.sh`.

### 2026-05-08 (dispatcher edge-case JVM coverage)

- Added `ControlResponseDispatcherTest` coverage for `ConnectResponse` with no payload (`getDefaultInstance()`), `HelloAck` with non-positive `heartbeat_interval_ms` (client default pacing), and `TransitionUI` with blank transition and zero duration clearing prior transition metadata. Re-verified `./scripts/check-android-client-boundary.sh` and `./scripts/test-android-client-boundary.sh`.

### 2026-05-08 (RegisterAck message diagnostics)

- Recorded non-empty `RegisterAck.message` on `AndroidTerminalViewState`, merged in `ControlResponseDispatcher` (blank follow-up ack preserves the prior message), surfaced as `register_ack_message` in copyable diagnostics, cleared on `withoutHandshake` for new sessions, preserved across disconnect for support copy/paste (same pattern as bug-report ack), and added dispatcher/ViewModel JVM coverage. Re-verified `./scripts/check-android-client-boundary.sh` and `./scripts/test-android-client-boundary.sh`.

### 2026-05-08 (RegisterAck server id diagnostics)

- Recorded non-blank `RegisterAck.server_id` as `registerAckServerId`, merged with blank follow-up preserving the prior id (same pattern as `register_ack_message`), surfaced as `register_ack_server_id` in copyable diagnostics, cleared on `withoutHandshake`, and preserved across disconnect together with `register_ack_asset_base_url` (previously cleared from state on disconnect even when still useful for support paste). Extended dispatcher and ViewModel JVM coverage. Re-verified `./scripts/check-android-client-boundary.sh` and `./scripts/test-android-client-boundary.sh`.

### 2026-05-08 (last control activity / Flutter status parity)

- Added `connectResponseActivityStatus` labels aligned with Flutter `statusFromConnectResponse` (plus explicit `Show media` and `Notification`), stored as `lastControlResponseActivity` on `AndroidTerminalViewState`, merged by `ControlResponseDispatcher` for every non–`PAYLOAD_NOT_SET` response, surfaced as `last_control_activity` in copyable diagnostics and a small terminal-chrome line (`terminal-last-server-activity` test tag).
- Cleared activity on `withoutHandshake` for new sessions; preserved across disconnect for support paste (same pattern as register ack / command result metadata). Refactored `UpdateUI` dispatch to avoid an early return so activity labeling still applies when the UI root is absent.
- Extended dispatcher and ViewModel JVM coverage. Re-verified `./scripts/check-android-client-boundary.sh` and `./scripts/test-android-client-boundary.sh`. Full Gradle unit tests require a host JDK (not verified in this session).

### 2026-05-08 (Flutter status labels for show media / notification)

- Extended Flutter `statusFromConnectResponse` to return `Show media` and `Notification` for the corresponding `ConnectResponse` payloads (matching native Android `connectResponseActivityStatus` and avoiding a misleading `Connected` status). Added `control_response_dispatcher` unit expectations; `flutter test test/connection/control_response_dispatcher_test.dart` passes.

### 2026-05-08 (UpdateUI parity: optional node + Android dispatch)

- Marked `terminals.ui.v1.UpdateUI.node` as `optional` in `api/terminals/ui/v1/ui.proto` so generated bindings track presence on the wire (compatible with existing clients; `buf breaking` clean).
- Regenerated protobufs (`make proto-generate`); Go suite and Flutter `control_response_dispatcher` tests pass.
- Aligned native Android `ControlResponseDispatcher` `UpdateUI` handling with Flutter `applyUpdateUi`: no `node` leaves the tree unchanged; blank `component_id` replaces the entire root (including establishing a root when it was previously absent).
- Extended `ControlResponseDispatcherTest` for blank component id, null-root bootstrap, and missing-node no-ops.

### 2026-05-08 (UpdateUI target id parity with Flutter)

- Matched Flutter `serverDrivenNodeId` in `ControlResponseDispatcher.replaceNode`: resolve targets by protobuf `id` when non-empty, otherwise `props["id"]`, so `UpdateUI` patches nested nodes that only identify via props.
- Centralized the same rule in `util/ServerDrivenNodeId.kt` and used it from `PrimitiveProps` and `NodeKey.testTag` so widget actions, default Compose tags, and `UpdateUI` patch targets stay aligned.
- Added JVM coverage for props-id patching, unknown-target no-ops, and direct unit tests for `serverDrivenNodeId`.

### 2026-05-08 (serverDrivenNodeId wire parity)

- Aligned Kotlin `serverDrivenNodeId` with Flutter `server_driven_node_key.dart` (no trimming of protobuf `id` or `props["id"]`; same `isNotEmpty` / empty-string fallback semantics) so action `componentId` values and `UpdateUI` targeting stay cross-client consistent.

### 2026-05-08 (Flutter activity-label JVM parity)

- Added `ConnectResponseActivityStatusTest` to lock every `connectResponseActivityStatus` label against Flutter `statusFromConnectResponse`, including handshake payloads that intentionally map to `Connected`.

### 2026-05-08 (TransitionUI normalization parity)

- Aligned `ControlResponseDispatcher` `TRANSITION_UI` handling with Flutter `transitionHintFromResponse` in `terminal_client/lib/connection/control_response_dispatcher.dart`: trim and lowercase the `transition` string, treat blank or `"none"` (case-insensitive) as no transition (clearing both `lastTransition` and `lastTransitionDurationMs`), and default `lastTransitionDurationMs` to 250 ms when the transition is meaningful but the server omitted `duration_ms`.
- Added `ControlResponseDispatcherTest` coverage for whitespace/uppercase normalization (`"  Fade  "` → `"fade"`), `"None"` clearing prior transition, and the 250 ms default for meaningful transitions with zero duration.
- Re-verified `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, and `git diff --check`. Full Gradle unit tests were host-skipped (`make android-client-test` reports `Skipping native Android tests: Android SDK path is not configured`); JDK was unavailable in this shell.

### 2026-05-08 (RegisterAck asset URL merge JVM coverage)

- Added `ControlResponseDispatcherTest.registerAckBlankAssetBaseUrlPreservesPriorUrl` so a follow-up `RegisterAck` without `ServerMetadata.photo_frame_asset_base_url` keeps the prior asset base URL (same merge pattern as `register_ack_message` / `register_ack_server_id`).
- Re-verified `./scripts/check-android-client-boundary.sh` and `./scripts/test-android-client-boundary.sh`.

### 2026-05-08 (activity status ordering parity)

- Reordered `connectResponseActivityStatus` in `ControlResponseDispatcher.kt` so explicit payload branches follow the same sequence as Flutter `statusFromConnectResponse` (notably `NOTIFICATION` immediately after `ROUTE_STREAM`), keeping cross-client diagnostics conventions easy to diff by eye.
- Re-verified `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, and `git diff --check`. Gradle unit tests were not run on this host (no JRE installed).

### 2026-05-08 (renderer instrumentation: layout primitives)

- Extended `ServerDrivenRendererTest` with instrumentation coverage for `StackWidget`, `RowWidget`, `GridWidget`, `ScrollWidget`, `PaddingWidget`, `CenterWidget`, `ExpandWidget` (inside a row), `OverlayWidget`, `ProgressWidget`, `SliderWidget` (display smoke), `CanvasWidget` with a `DrawLine` op, and `AudioVisualizerWidget` media-surface delegation, closing gaps against the Phase 5 “Compose tests for all primitives” table for layout and canvas/audio-visualizer paths.
- Re-verified `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, and `git diff --check`. Full Gradle validation was not run in this environment (no JDK on the agent host).

### 2026-05-08 (ScrollWidget parity with Flutter)

- Fixed native Android `ScrollWidget` rendering to match Flutter `server_driven_renderer.dart`: horizontal scrolling uses a `Row` inside `horizontalScroll` (instead of a `Column`), vertical scrolling uses a `Column` with `verticalScroll` and start horizontal alignment; when `direction_enum` is unset, the deprecated `direction` string still selects horizontal (case-insensitive `"horizontal"`), matching the Flutter fallback.
- Added instrumentation coverage for horizontal scroll (deprecated string + enum), and for slider drag emitting a `change` action with an updated value.

### 2026-05-08 (renderer / UI-action parity with Flutter)

- Matched Flutter `stack` layout: `StackWidget` now uses a top-start `Column` of children (not overlapping `Box` children) and applies `background` from the same `props["background"]` hex rules as Flutter `parseHexColor` (optional `#`, 6-digit RGB expands to ARGB).
- Removed extra `Row` spacing so `RowWidget` matches Flutter’s tight `Row`.
- Aligned control actions with Flutter: button taps omit a synthetic `pressed` value; toggle uses action `toggle`; dropdown selection uses `select`; text fields emit `submit` on IME Done (and clear local text), not per-keystroke `change`.
- Updated JVM and instrumentation expectations (`ProtocolBuildersTest`, `AndroidTerminalViewModelTest`, `ServerDrivenRendererTest`, `AndroidTerminalAppSmokeTest`).

### 2026-05-08 (TextWidget style/color parity helper)

- Added shared Android UI color parsing helpers (`ui/ColorParsing.kt`) aligned with Flutter `parseHexColor`: optional `#`, 6-digit RGB expansion to ARGB, and deterministic `Color.Unspecified` fallback.
- Updated `ServerDrivenRenderer` text rendering to honor `TextWidget.style == "monospace"` via `FontFamily.Monospace`, and switched text/canvas/background color parsing to the shared helper for consistent behavior across text and draw ops.
- Added focused JVM coverage in `ColorParsingTest` for RGB/ARGB parsing and invalid-value fallback.
- Re-verified `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, and `git diff --check`.

### 2026-05-08 (color parsing hardening follow-up)

- Removed the renderer-only color wrapper and called shared `parseHexColor` directly for stack backgrounds so all color parsing behavior stays centralized in `ui/ColorParsing.kt`.
- Expanded `ColorParsingTest` with null/blank input and whitespace-wrapped `#RRGGBB` coverage, plus explicit `parseColorOrUnspecified(null)` fallback coverage, to lock Flutter parity semantics around trimming and absent values.
- Re-verified `./scripts/check-android-client-boundary.sh` and `./scripts/test-android-client-boundary.sh`.
- Attempted focused Gradle JVM validation for `ColorParsingTest`, but this shell session has no installed Java runtime (`JAVA_HOME` path missing and `/usr/libexec/java_home -V` reports no JRE), so Gradle execution is host-blocked.

### 2026-05-08 (TextInput autofocus parity)

- Wired native Android `TextInputWidget.autofocus` in `ServerDrivenRenderer` via a Compose `FocusRequester`, so server-driven text inputs can request focus on initial render like Flutter.
- Added renderer instrumentation coverage in `ServerDrivenRendererTest.textInputAutofocusRequestsFocus`.
- Re-verified Android boundary scan/test with `./scripts/check-android-client-boundary.sh` and `./scripts/test-android-client-boundary.sh`.
- Attempted Android instrumentation source validation with `cd android_client && ./gradlew compileDebugAndroidTestKotlin`, but this shell session has no Java runtime configured (`Unable to locate a Java Runtime`), so Gradle validation remains host-blocked.

### 2026-05-08 (action component-id fallback parity)

- Aligned native Android action component-id fallback with Flutter `server_driven_renderer.dart`: when `serverDrivenNodeId(node)` is blank, interactive widgets now emit stable fallback IDs (`button`, `slider`, `toggle`, `dropdown`, `text_input`, `gesture_area`) instead of an empty string.
- Added renderer instrumentation coverage for fallback component IDs on button taps and gesture-area taps with nodes that omit `id`/`props.id`.
- Re-verified Android boundary scan/test with `./scripts/check-android-client-boundary.sh` and `./scripts/test-android-client-boundary.sh`.
- Attempted Android instrumentation source validation with `cd android_client && JAVA_HOME=/opt/homebrew/opt/openjdk@17/libexec/openjdk.jdk/Contents/Home ./gradlew compileDebugAndroidTestKotlin`, but this host currently has no Java runtime at that path (and `/usr/libexec/java_home -V` reports none), so Gradle validation remains host-blocked.

### 2026-05-08 (fullscreen API modernization)

- Updated `WindowAndroidFullscreenController` to use `WindowInsetsControllerCompat` + `WindowCompat.setDecorFitsSystemWindows` on API 30+ and keep the existing immersive-flag behavior only as the API 29-and-lower fallback, preserving Fire OS 6 compatibility while removing the deprecated-only path from modern Android/Fire builds.
- Added JVM regression coverage in `WindowAndroidFullscreenControllerTest` for the legacy fallback bitmask contract (`enabled` immersive flags and `disabled` stable-layout-only flags).
- Re-verified Android boundary scan/test with `./scripts/check-android-client-boundary.sh` and `./scripts/test-android-client-boundary.sh`.
- Attempted focused Gradle JVM validation with `cd android_client && ./gradlew testDebugUnitTest --tests '*WindowAndroidFullscreenControllerTest*'`, but this shell session has no Java runtime configured (`Unable to locate a Java Runtime`), so host-side Gradle validation remains blocked.

### 2026-05-09

- Matched Flutter `DropdownWidget` selection rules: when the server `value` is missing from `options`, the visible selection falls back to the first option; empty option lists show the same `Select option` hint as Flutter.
- Matched Flutter `Expanded` behavior for `ExpandWidget`: direct children of `Row` and `Column` (including stack’s column, scroll row/column, and flex rows/columns) now use `RowScope`/`ColumnScope` helpers so `Expand` applies `Modifier.weight(1f)` in the correct flex scope; non-flex parents keep `fillMaxWidth` on `Expand` via `RenderPlainChildren`.
- Added renderer instrumentation coverage for invalid dropdown values, empty dropdown options, and `Expand` inside `Stack`.
- Matched Flutter `GridWidget` layout: replaced `LazyVerticalGrid` with `BoxWithConstraints` + `FlowRow`, 8dp horizontal/vertical gaps, and the same per-cell width formula as Flutter (`Wrap` + `SizedBox`) so column counts and wrapping match the reference client.
- Re-verified `./scripts/check-android-client-boundary.sh` and `./scripts/test-android-client-boundary.sh`. Gradle tasks were not executed on this host (no Java runtime).
- Matched Flutter `TextWidget` / `ButtonWidget` spacing: 4dp vertical padding on native `Text` and `Button` nodes.
- Matched Flutter `GestureAreaWidget` empty-child behavior: 48×48 dp minimum hit target; added `ServerDrivenRendererTest.gestureAreaWithNoChildrenExposesMinimumTapTarget`.
- Re-verified `./scripts/check-android-client-boundary.sh` and `./scripts/test-android-client-boundary.sh` after renderer parity tweaks (Gradle not run on this host: no Java runtime).
- Added Compose instrumentation coverage for `ProgressWidget` value clamping to `[0, 1]` via semantics (`ProgressBarRangeInfo`), matching Flutter `LinearProgressIndicator` clamp behavior for out-of-range server values.

### 2026-05-09 (Flutter parity: fallback + device-control copy)

- Aligned native Android unsupported-node fallback copy with Flutter (`RendererPolicy.unsupportedText` → `Unsupported UI node`).
- Aligned server-driven `FullscreenWidget` / `KeepAwakeWidget` labels with Flutter `_placeholderPrimitive` titles (`Fullscreen enabled` / `disabled`, `Keep awake enabled` / `disabled`).
- Aligned `BrightnessWidget` with Flutter title `Brightness hint` plus two-decimal detail (`%.2f`) on a secondary line; `DeviceControlNode` now keys `LaunchedEffect` on typed effect state instead of the full label string.
- Updated renderer and app smoke instrumentation expectations. Re-verified `./scripts/check-android-client-boundary.sh` and `./scripts/test-android-client-boundary.sh`. Gradle not run on this agent host (no JRE).

### 2026-05-09 (unit test visibility + session teardown)

- Enabled Gradle `testLogEvent` output for debug unit tests so `testDebugUnitTest` prints **PASSED/SKIPPED/FAILED** per method (default Gradle output stops at the task line, which looks “stuck” for long JVM suites).
- Added explicit `disconnect()` + `advanceUntilIdle()` at the end of the three `newConnectClears*` ViewModel tests that intentionally finished while still **Connected**, so `viewModelScope` work is torn down before `runTest` returns (kotlinx `runTest` will not finish while the shared `TestCoroutineScheduler` still has unfinished work scheduled on `Dispatchers.Main`).
- Re-run: `make android-client-test` (or `cd android_client && ./gradlew testDebugUnitTest`); use `./gradlew --stop` if workers misbehave.

### 2026-05-09 (Flutter media surface parity)

- Aligned native Android with Flutter’s split between generic renderer placeholders and shell chrome: `ServerDrivenRenderer` now takes optional `mediaSurface` and `audioVisualizerSurface` (audio falls back to `mediaSurface` for the existing single-lambda tests). When both are null for a node type, video/audio use `TerminalMediaPlaceholder`, matching Flutter `server_driven_renderer` `_placeholderPrimitive` titles (`Video surface` / `Audio level`) and trimmed track/stream ids.
- Added `TerminalShellVideoSurface` and `TerminalShellAudioVisualizer` composables mirroring Flutter `terminal_client_shell` `_buildVideoSurface` / `_buildAudioVisualizer` chrome (waiting copy, layout, and indeterminate audio level bar until stream attach is wired).
- Wired `AndroidTerminalApp` to pass those shell composables so production manual-connect UI matches the Flutter app’s media presentation instead of rendering empty nodes.
- Added instrumentation tests for the null-builder placeholder paths. Re-verified `./scripts/check-android-client-boundary.sh` and `./scripts/test-android-client-boundary.sh`.

### 2026-05-09 (canvas TEXT/PATH + selectable TextWidget)

- Implemented `DrawText` and `DrawPath` in native `TerminalCanvas` via `nativeCanvas` (fill/stroke for paths, `PathParser.createPathFromPathData` with safe skip on invalid `d`), closing the gap where proto `DrawOp` variants were no-ops beyond line/rect/circle.
- Wrapped server-driven `TextWidget` in Compose `SelectionContainer` for Flutter `SelectableText`-style selection parity.
- Added instrumentation coverage for draw-text, draw-path, and malformed path data (no crash). Re-verified `./scripts/check-android-client-boundary.sh` and `./scripts/test-android-client-boundary.sh`. Gradle compile not run on this agent host (no JDK).

### 2026-05-09 (single-child wrapper + device-control parity)

- Added a `WrappedChild` helper in `ServerDrivenRenderer.kt` mirroring Flutter `_renderNodeChildren` semantics (empty → render nothing, 1 child → render directly, >1 children → start-aligned `Column`) and routed `PaddingWidget`, `CenterWidget`, `ExpandWidget` (non-flex parents), and `GestureAreaWidget` (with children) through it so multi-child wrappers no longer overlap inside a `Box`.
- Updated `FullscreenWidget`, `KeepAwakeWidget`, and `BrightnessWidget` to render their wrapped children below the device-control hint label via `DeviceControlNode` content slot (matching Flutter `_placeholderPrimitive` which renders `_renderNodeChildren` alongside the title), and clamped the displayed `brightness=` label value plus the forwarded effect call to `[0, 1]` to match Flutter `node.brightness.value.clamp(0.0, 1.0)`.
- Annotated the `OverlayWidget` `Box` to make its `Stack(fit: StackFit.loose)` parity intent explicit (still uses `RenderPlainChildren`).
- Added instrumentation coverage in `ServerDrivenRendererTest`: brightness clamp for negative values, device-control children rendering for keep-awake/fullscreen/brightness, and multi-child padding rendering as a `Column`.
- Re-verified `./scripts/check-android-client-boundary.sh`, `./scripts/test-android-client-boundary.sh`, and `git diff --check`. Gradle was not runnable on this agent host (no Java runtime).

### 2026-05-09 (renderer instrumentation: toggle, surfaces, image a11y)

- Added instrumentation coverage for `ToggleWidget` action `componentId` fallback to `toggle` when protobuf/props ids are absent (parity with `ButtonWidget` fallback tests).
- Added instrumentation coverage that `VideoSurfaceWidget` and `AudioVisualizerWidget` call distinct `mediaSurface` vs `audioVisualizerSurface` lambdas when both are supplied (shell chrome split).
- Added instrumentation coverage that `ImageWidget` forwards `props["contentDescription"]` to the injected `imageLoader`.
- Re-verified `./scripts/check-android-client-boundary.sh` and `./scripts/test-android-client-boundary.sh`. Host Gradle/instrumentation execution was not run here (no Java runtime).

### 2026-05-09 (NSD diagnostics + Fire OS discovery doc)

- Added `formatNsdFailureDetail` so `NsdAndroidDiscovery` maps Android `NsdManager` failure codes to generic, copyable hints (multicast/Wi‑Fi isolation, already-active discovery, max listener limit) that always steer toward manual endpoint fallback—terminal chrome only, no scenario branching.
- Extended JVM coverage in `AndroidNsdDiscoveryTest` for known codes and unknown codes.
- Documented discovery/NSD quirks and Fire OS fallback expectations under `docs/client-android.md` (“Discovery (NSD / mDNS) quirks”), closing the plan’s documentation gap for LAN multicast behavior without requiring a physical tablet run in this session.

### 2026-05-09 (sensor telemetry parity)

- Implemented periodic control-stream sensor telemetry aligned with Flutter `buildSensorTelemetryRequest`: `battery.level` and `battery.charging` from the last registered capability snapshot, default interval 15 seconds (`AndroidClientDependencies.sensorTelemetryIntervalMillis`), wired through `ProtocolBuilders.sensorTelemetryFromCapabilities`, `AndroidControlSession.sendSensorTelemetry`, and `AndroidTerminalViewModel` alongside the heartbeat loop (including reconnect and HelloAck restart behavior). Added JVM coverage (`ProtocolBuildersTest`, `AndroidControlSessionControllerTest`, `AndroidTerminalViewModelTest.connectedSessionSendsPeriodicSensorTelemetry`); instrumentation fakes set `sensorTelemetryIntervalMillis = 0` where heartbeats are disabled. Documented behavior in `docs/client-android.md`.

### 2026-05-09 (Flutter lifecycle parity: pause heartbeat/sensor)

- Paused periodic heartbeat and battery sensor telemetry while the activity is stopped (`MainActivity.onStop`), matching Flutter shell behavior when `AppLifecycleState` is not `resumed`. Added `AndroidTerminalViewModel.setAppForegrounded`, made `startHeartbeat` / `startSensorTelemetry` idempotent and foreground-gated, and JVM tests (`backgroundPausesHeartbeatAndSensorTelemetryLoops`, `connectWhileBackgroundedDoesNotStartLoopsUntilForegrounded`). Documented pause semantics in `docs/client-android.md`. Agent host had no JRE; run `cd android_client && ./gradlew testDebugUnitTest --tests '*AndroidTerminalViewModelTest*'` locally.

### 2026-05-09 (Flutter lifecycle parity: capability delta + network suppression)

- On each foreground/background transition while connected, Android now sends a capability delta with reason `app-lifecycle-change`, matching Flutter `_sendLifecycleCapabilityUpdate(reason: 'app_lifecycle_change')`.
- Network-monitor capability refreshes are skipped while the activity is stopped (`capability_refresh_suppressed=app-background` in diagnostics) so background connectivity churn does not spam capability traffic.
- JVM coverage: `AndroidTerminalViewModelTest.appLifecycleChangeSendsCapabilityDeltaWhenConnected`, `networkMonitorSkipsCapabilityRefreshWhenBackgrounded`. Docs updated in `docs/client-android.md`.

## Test Plan

### Unit tests

- Protobuf request builders.
- Wire protocol version assignment.
- Manual endpoint parsing.
- Carrier preference and fallback ordering.
- Transport error diagnosis.
- Reconnect policy backoff.
- Capability snapshot construction.
- Capability delta generation and debouncing.
- Permission-to-capability mapping.
- Stale-generation rebaseline.
- Primitive prop parsing.
- Renderer fallback policy.
- UI action translation.
- Response dispatcher behavior.
- Diagnostics clipboard formatting.

### Compose tests

- Every server-driven UI primitive.
- Text/style/color/semantics mapping.
- Input action emission.
- Stable node tags.
- Malformed node fallback.
- Client chrome connection states.
- Permission education surfaces.
- Build metadata display.

### Instrumentation tests

- App launches on emulator or device.
- Manual endpoint accepts valid local endpoint and rejects malformed input.
- Renderer displays synthetic `SetUI` tree.
- UI action from Compose reaches fake control client.
- Orientation change updates display metrics.
- Permission denial updates capability state.
- Kiosk keep-awake flag is applied and cleared.

### Device smoke tests

Run on at least one Kindle Fire tablet before declaring the plan complete:

1. Install debug APK with ADB.
2. Start server with `make run-server` on the same LAN.
3. Open Android client and connect by manual endpoint.
4. Verify capability snapshot reaches server.
5. Push a server-driven UI tree.
6. Tap a button and verify `UIAction` reaches server.
7. Rotate tablet and verify display capability delta.
8. Toggle network off/on and verify diagnostics/reconnect behavior.
9. Enable keep-awake and verify display remains active for the smoke-test window.
10. Copy diagnostics and confirm build metadata and transport attempts are included.

### Negative tests

- No server available produces deterministic diagnostics.
- Invalid endpoint is rejected locally.
- Unknown UI primitive does not crash renderer.
- Malformed props do not crash renderer.
- Permission-denied microphone/camera are not advertised.
- Missing WebRTC support does not advertise media send/receive capability.
- Boundary scan fails if a known scenario name appears in production Android source.
- Boundary scan fails if a Google service dependency appears in Gradle files.

## Migration Strategy

This is an additive client target. Existing Flutter client behavior must continue to pass.

1. Add native Android skeleton without touching server behavior.
2. Generate and consume the same protobuf contracts as existing clients.
3. Bring up manual connection before discovery.
4. Bring up capability snapshot before deltas.
5. Bring up renderer before media.
6. Bring up UI action dispatch before advanced diagnostics.
7. Add Fire tablet hardening after protocol parity works.
8. Add media/sensing only after capability reporting can accurately withdraw unsupported resources.
9. Add boundary scans and docs before marking the plan complete.

Any protocol gap discovered during Android implementation should be handled by a separate protocol evolution PR and must preserve existing Flutter client compatibility.

## Review Checklist

Every PR under this plan should answer:

- Does the Android client remain a generic terminal?
- Did any scenario name or server application package ID enter production Android behavior?
- Are all client/server messages generated from protobuf definitions?
- Does the code avoid Google Play Services and Google-only APIs?
- Does capability reporting reflect real hardware, permissions, and runtime state?
- Does renderer code avoid importing connection, discovery, media internals, and platform permissions?
- Does connection code avoid parsing visual primitive props?
- Are API guards present for APIs newer than `minSdk`?
- Are new platform effects behind adapters?
- Are unit tests present for pure logic?
- Are Compose/instrumentation tests present for UI or platform behavior?
- Do Android validation commands pass locally?
- Do existing Flutter client tests still pass if shared contracts changed?
- Did the PR avoid visual redesign and protocol changes unless explicitly scoped?

## Suggested PR Sequence

### PR 1: Android project skeleton and docs

Expected changed files:

- `android_client/settings.gradle.kts`
- `android_client/build.gradle.kts`
- `android_client/app/build.gradle.kts`
- `android_client/app/src/main/AndroidManifest.xml`
- `android_client/app/src/main/java/com/curtcox/terminals/android/MainActivity.kt`
- `android_client/README.md`
- `docs/client-android.md`
- `Makefile`

### PR 2: Protobuf generation and protocol builders

Expected changed files:

- Android Gradle protobuf configuration.
- Protocol builder helpers.
- Protocol fixture tests.
- Documentation for regenerating Android protobufs.

### PR 3: Manual connection and control stream

Expected changed files:

- `connection/AndroidControlClient.kt`
- `connection/GrpcAndroidControlClient.kt`
- `connection/WebSocketAndroidControlClient.kt` if needed.
- `connection/EndpointResolution.kt`
- `connection/CarrierPreference.kt`
- `connection/ReconnectPolicy.kt`
- connection tests.

### PR 4: Discovery and diagnostics chrome

Expected changed files:

- `discovery/AndroidNsdDiscovery.kt`
- `diagnostics/AndroidClientChrome.kt`
- `diagnostics/DiagnosticClipboard.kt`
- discovery and diagnostics tests.

### PR 5: Capability lifecycle

Expected changed files:

- `capabilities/AndroidCapabilityProbe.kt`
- `capabilities/AndroidCapabilitySession.kt`
- `capabilities/AndroidScreenMetrics.kt`
- `capabilities/PermissionCapabilityMonitor.kt`
- capability tests and instrumentation smoke tests.

### PR 6: Compose server-driven renderer

Expected changed files:

- `ui/ServerDrivenRenderer.kt`
- `ui/ServerDrivenAction.kt`
- `ui/RendererPolicy.kt`
- `ui/PrimitiveProps.kt`
- `ui/widgets/**`
- renderer unit/Compose tests.

### PR 7: Response dispatch and UI action flow

Expected changed files:

- `connection/AndroidControlSessionController.kt`
- `connection/ControlResponseDispatcher.kt`
- UI action translation tests.
- synthetic server response tests.

### PR 8: Fire tablet kiosk hardening

Expected changed files:

- `platform/AndroidKeepAwakeController.kt`
- `platform/AndroidNotificationDelivery.kt`
- `platform/AndroidNetworkState.kt`
- `platform/FireOsDeviceInfo.kt`
- kiosk docs and instrumentation tests.

### PR 9: Media, permissions, and optional WebRTC

Expected changed files:

- `media/AndroidMediaEngine.kt`
- `media/AndroidAudioPlayback.kt`
- `media/AndroidWebRtcAdapter.kt`
- `media/AndroidMediaPermissionProbe.kt`
- media tests.

### PR 10: Boundary enforcement and CI integration

Expected changed files:

- `scripts/check-android-client-boundary.sh`
- `scripts/test-android-client-boundary.sh`
- Makefile/CI updates.
- `docs/client-architecture.md` Android cross-link.

## Whole-Plan Acceptance Criteria

- `plans/features/android-client/plan.md` exists and is indexed by plans tooling.
- `android_client/` is a native Android project, not a Flutter target wrapper.
- Native Android debug APK builds with Gradle.
- APK installs and runs on a Kindle Fire tablet running Fire OS 6 or newer.
- Manual connection to a local Terminals server works.
- LAN discovery works where supported or degrades clearly to manual connection.
- Android uses generated protobufs from `api/terminals/**`.
- Android sends hello, capability snapshot, capability delta, heartbeat, and UI action messages.
- Android renderer supports every current server-driven UI primitive or documents safe fallback.
- Android capability reporting is live and permission-aware.
- Transport, permission, and build diagnostics are visible and copyable.
- Kiosk keep-awake behavior is available as generic terminal chrome.
- Production Android code contains no scenario-specific branches.
- Production Android code contains no Google service dependencies.
- `docs/client-android.md` covers build, install, run, Fire tablet setup, validation, and troubleshooting.
- Android Makefile targets exist for build, test, lint, and connected tests.
- Boundary scan exists and passes.
- Existing `terminal_client` tests still pass after shared contract changes.
- `make proto-contract-test`, `make client-test`, and Android unit tests pass before closing the plan.
