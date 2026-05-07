---
title: "Android Client"
kind: plan
status: building
owner: curtcox
validation: automated
last-reviewed: 2026-05-07
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

Status: planned

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

Status: planned

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

Status: planned

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

Status: planned

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

Status: planned

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

Status: planned

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

Status: planned

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

Status: planned

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

Status: planned

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

Status: planned

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
