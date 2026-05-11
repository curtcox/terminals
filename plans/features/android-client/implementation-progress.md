---
title: "Android Client — Implementation Progress"
kind: plan
status: shipped-validated
owner: curtcox
validation: automated
last-reviewed: 2026-05-11
---

# Android Client — Implementation Progress

This log was split from [plan.md](plan.md) so the main Android plan stays scannable.

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

### 2026-05-09 (Flutter parity: capability delta reason strings)

- Aligned native Android capability delta reasons with the Flutter reference client: lifecycle transitions now use `app_lifecycle_change` (underscores, not `app-lifecycle-change`); `MainActivity.onConfigurationChanged` capability refresh now uses `display_geometry_change` instead of `configuration`. Updated JVM and instrumentation expectations, `docs/client-android.md`, and smoke test naming.

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

- On each foreground/background transition while connected, Android sends a capability delta with reason `app_lifecycle_change`, matching Flutter `_sendLifecycleCapabilityUpdate(reason: 'app_lifecycle_change')`. Configuration-driven display refreshes use `display_geometry_change`, matching Flutter display-metric updates.
- Network-monitor capability refreshes are skipped while the activity is stopped (`capability_refresh_suppressed=app-background` in diagnostics) so background connectivity churn does not spam capability traffic.
- JVM coverage: `AndroidTerminalViewModelTest.appLifecycleChangeSendsCapabilityDeltaWhenConnected`, `networkMonitorSkipsCapabilityRefreshWhenBackgrounded`. Docs updated in `docs/client-android.md`.

### 2026-05-09 (background suppression: discovery restart)

- Suppressed network-triggered NSD discovery restarts while the activity is stopped (`discovery_restart_suppressed=app-background`), matching the existing network-callback capability refresh suppression and avoiding background NSD churn.
- JVM coverage: `AndroidTerminalViewModelTest.networkMonitorSkipsDiscoveryRestartWhenBackgrounded`. Docs updated in `docs/client-android.md`.

### 2026-05-09 (Compose `nativeCanvas` import)

- Fixed `ServerDrivenRenderer` Kotlin compile against Compose BOM `2025.10.00`: `drawContext.canvas.nativeCanvas` requires the explicit extension import `androidx.compose.ui.graphics.nativeCanvas` (otherwise `nativeCanvas` is an unresolved reference).

### 2026-05-09 (real MainActivity launch instrumentation)

- Added `MainActivityLaunchSmokeTest` using `createAndroidComposeRule<MainActivity>()` so connected-device runs exercise production startup (`AndroidClientDependencies.fromContext`, default session factory, activity permission requester) rather than only Compose-isolated smoke tests with injected factories. Asserts manual-connect chrome (`terminal-endpoint-field`, `terminal-connect-button`) is visible after launch.

### 2026-05-09 (launch smoke + dead code)

- Removed the unused `PowerCapabilityMonitor` holder class; battery snapshots remain `PowerCapabilityState` used by `ContextAndroidCapabilityProbe` and capability session tests.
- Extended `MainActivityLaunchSmokeTest` to assert discovery controls, live-media status chrome, and last-server-activity diagnostics are visible on real `MainActivity` launch (production deps). Re-run `cd android_client && ./gradlew compileDebugAndroidTestKotlin` / `connectedDebugAndroidTest` on a JDK 17 host with the Android SDK.

### 2026-05-09 (terminal_input key streaming)

- Matched Flutter shell `terminal_input` behavior: `TextInputWidget` nodes with that component id stream insertions, backspaces (`\b` repeats), and IME newline as protobuf `InputEvent.key.text` on the control stream (`ProtocolBuilders.keyInput`, `AndroidControlSession.sendKeyText`, `AndroidTerminalViewModel.sendTerminalKeyText`, `ServerDrivenRenderer` composition-local sink from `AndroidTerminalApp`). Regular text inputs still emit `submit` UI actions on Done. Added JVM and renderer instrumentation tests; documented in `docs/client-android.md`.

### 2026-05-09 (client bug report filing parity)

- Implemented Flutter-aligned on-device bug reporting for the native shell: shared `bugTokenWords` + `buildBugIdentifier` / `buildLocalBugReportId` (Calendar/TimeZone for API 25), `AndroidBugReportBuilder` for `Diagnostics.BugReport` + `ClientContext`, `ProtocolBuilders.bugReport`, `AndroidControlSession.sendBugReport`, ViewModel interception of `bug_report*` UI actions (no spurious `UIAction` to the server), shell **Report bug** chrome with queue-until-connect + flush after connect/reconnect, and JVM/instrumentation coverage. Re-run `make android-client-test` on a JDK 17 host.

### 2026-05-09 (bug report polish)

- `flushQueuedBugReports` now updates `lastBugReportSubmitStatus` after sending queued reports (initial connect and reconnect), including multi-report and partial-failure summaries.
- Added `AndroidBugReportBuilderTest`, `AndroidTerminalViewModelTest.chromeBugReportQueuedThenFlushedOnConnect`, and `MainActivityLaunchSmokeTest.reportBugWhileOfflineShowsQueuedStatus`.
- Documented shell bug reporting, queueing, `bug_report*` actions, and `BugReportAck` diagnostics in `docs/client-android.md`.
- Added `AndroidTerminalViewModelTest` coverage for queued bug-report flush when multiple reports are pending (`chromeBugReportFlushMultipleQueuedAllSucceed`), when the first send fails and a later one succeeds (`chromeBugReportFlushPartialFailureSummarizesCounts`), and when every queued send fails (`chromeBugReportFlushAllFailRecordsFailure`), using a `FakeSession` per-attempt failure pattern.

### 2026-05-09 (Compose KeyboardActions API)

- Fixed `ServerDrivenRenderer` `OutlinedTextField` IME Done handling for Compose BOM `2025.10.00`: `KeyboardActions.onDone` expects `(KeyboardActionScope) -> Unit`, so the handler is now wrapped as `{ onDone() }` instead of passing a `() -> Unit` reference. Unblocks `compileDebugKotlin` / `make android-client-test`.

### 2026-05-09 (gRPC control transport)

- Implemented real `GrpcAndroidControlClient` using `io.grpc:grpc-okhttp` + generated `TerminalControlServiceGrpc` stubs (protobuf codegen `protoc-gen-grpc-java` with `lite`), replacing the previous stub that threw.
- Added `EndpointResolution.carrier` (`CarrierPreference`), `CarrierSelectingAndroidControlClient` in the default session factory, and `grpc://` / `grpcs://` support in `ManualEndpointParser` with JVM tests.
- Normalized discovery-only gRPC TXT values (`grpc=host:port`) to `grpc://…` in `AndroidTerminalViewModel` so selecting a discovered server no longer mis-routes through the WebSocket upgrade on the gRPC port.
- Updated `docs/client-android.md` manual endpoint examples (explicit gRPC vs WebSocket URLs).
- Fixed `AndroidControlSessionControllerTest.heartbeatAndUiActionUseProtocolBuilders` to `connect()` first so `sendSensorTelemetry` aligns with the “last registered capability snapshot” contract.

### 2026-05-09 (ViewModel test scheduler + network-restore timing)

- Recreated the test `StandardTestDispatcher` in `@Before` so heartbeat/sensor loops don’t leave infinite delayed work on a shared queue across tests.
- Ended `bugReportServerDrivenActionSendsBugReportNotUiAction` with `disconnect()` + `advanceUntilIdle()` so connected periodic loops never leak into the next case.
- Avoided `advanceUntilIdle()` while a positive-interval heartbeat could run forever (`appLifecycleChangeSendsCapabilityDeltaWhenConnected` uses zero heartbeat/sensor intervals).
- For network-restore debouncing tests, replaced `advanceUntilIdle()` after reconnect with bounded `advanceTimeBy(400)` + `runCurrent()` so async reconnect settles without draining an infinite heartbeat loop; aligned `networkMonitorSkipsCapabilityRefreshWhenBackgrounded` with lifecycle capability deltas after backgrounding (`app_lifecycle_change`).
- Pinned JVM default `TimeZone` in `AndroidBugReportBuilderTest` so `buildBugIdentifier` (uses `TimeZone.getDefault()`) matches expected bug-token hints on any host.

### 2026-05-09 (server notification TTS / Flutter alert parity)

- Added `AndroidTerminalSpeech` + `ContextAndroidTerminalSpeech` so explicit server `Notification` responses speak the same text the Flutter `AlertDeliveryService` chooses (trimmed body if non-empty, else trimmed title), after attempting status-bar delivery.
- Skipped delivery and speech when both title and body trim empty; tightened `last_notification` diagnostics to prefer trimmed title, else trimmed body.
- Documented behavior in `docs/client-android.md`; extended `AndroidTerminalViewModelTest`.
- Hardened `ContextAndroidTerminalSpeech` engine creation so `OnInit` applies `Locale.getDefault()` using an instance reachable from a one-element holder populated immediately after the `TextToSpeech` constructor returns (typical async `OnInit` always sees `holder[0]`; synchronous init callbacks may still skip locale and rely on the platform default).

### 2026-05-09 (instrumentation compile + validation refresh)

- Fixed `compileDebugAndroidTestKotlin`: removed duplicate `terminals.capabilities.v1.Capabilities` import in `AndroidTerminalAppSmokeTest`, and dropped the obsolete top-level `fetchSemanticsNode` import in `ServerDrivenRendererTest` (calls use `SemanticsNodeInteraction.fetchSemanticsNode()` without that import on Compose BOM `2025.10.00`).
- Re-ran `make android-client-test`, `make android-client-lint`, `make android-client-build`, and `./gradlew compileDebugAndroidTestKotlin` with Makefile-resolved JDK.
- Refreshed **Current Validation Evidence** and clarified remaining checks (NSD doc location, explicit WebRTC-disabled posture via `AndroidWebRtcAdapter`).
- Updated `docs/client-architecture.md` to describe `android_client/` as the shipped native thin client (not only a scaffold).

### 2026-05-09 (Makefile instrumentation compile gate)

- Added `make android-client-compile-android-test`, which runs `./gradlew compileDebugAndroidTestKotlin` with the same Android SDK / JDK discovery rules as the other native Android Make targets (skips with guidance when SDK or JDK is missing).
- Wired the target into `make all-test` so CI and `make all-check` catch instrumentation-only Kotlin compile regressions without requiring a connected device.
- Documented the target alongside existing Android validation commands in `docs/client-android.md` and in this plan’s required-validation block.

### 2026-05-09 (StartStream StreamReady control parity)

- Matched Flutter shell streaming handshake: on inbound `ConnectResponse.start_stream` with non-blank `stream_id`, `AndroidTerminalViewModel` now calls `AndroidControlSession.sendStreamReady` (trimmed id) before merging dispatcher state; send failures route through `handleControlLoss` like heartbeat errors.
- Added `ProtocolBuilders.streamReady`, `AndroidControlSessionController.sendStreamReady`, JVM coverage (`ProtocolBuildersTest`, `AndroidControlSessionControllerTest`, extended `opaqueStartStreamSummary…` / blank-id regression), and documented the behavior in `docs/client-android.md`.

### 2026-05-09 (control transport stream termination → reconnect)

- Extended `AndroidControlResponseSink` with `onTransportTerminated` (default no-op). `GrpcAndroidControlClient` forwards gRPC `StreamObserver.onError` / `onCompleted` once per stream (guarded against intentional `close()`); `WebSocketAndroidControlClient` notifies on read-loop failure without firing during intentional `close()` (`closingSocket`).
- `AndroidTerminalViewModel` handles termination like other control loss via `handleControlLoss` (EOF when the server half-closes cleanly). JVM coverage: `transportTerminationTriggersReconnectWithBackoff`. Docs: reconnect paragraph in `docs/client-android.md`.
- Hardened WebSocket intentional-close suppression so the read loop keeps treating socket errors as close-related until a later successful `connect()` re-enables termination callbacks, avoiding stale old-transport callbacks during reconnect/disconnect races.
- Re-ran `make android-client-test`, `make android-client-lint`, `make android-client-build`, `make android-client-compile-android-test`, boundary scripts, and `git diff --check`.

### 2026-05-09 (Flutter live-media control-response parity)

- Added `AndroidLiveMediaSession` / `LiveMediaSessionResult` behind `AndroidWebRtcAdapter` (`WebRtcGatedLiveMediaSession`), delegated from `AndroidMediaEngine` for `StartStream`, `StopStream`, `RouteStream`, and `WebRTCSignal` responses.
- `AndroidTerminalViewModel` calls the seam after `sendStreamReady` for non-blank stream ids; surfaces `last_live_media` / `lastLiveMediaLine` when start fails (`Unsupported`), clears the line in `withoutHandshake`.
- `AndroidClientDependencies.fromContext` uses one shared disabled `AndroidWebRtcAdapter` for both capability reporting and live-media gating; default JVM test dependencies pair `AndroidMediaEngine` live-media with the same `webRtcAdapter` instance as the ViewModel helper.
- JVM coverage: `AndroidMediaEngineTest.liveMediaDelegatesStopRouteAndSignalToSession`, extended `opaqueStartStreamSummary…`, `startStreamWithWebRtcDisabledSurfacesAdapterReasonInLiveMediaDiagnostics`; documented in `docs/client-android.md`.

### 2026-05-09 (live-media session JVM contract)

- Added `AndroidMediaEngineTest` coverage for `AndroidLiveMediaSession.disabled(...)`, `fromAdapter(disabled)`, and `fromAdapter` when the adapter reports `supported=true` (still returns `live-media-session-not-implemented` for start/WebRTC signal until real WebRTC lands).
- Re-ran `make android-client-test`; remaining plan gaps are physical-device checks listed under **Remaining validation**.

### 2026-05-09 (notification permission JVM coverage)

- Added `AndroidClientDependencies.runtimeNotificationPermissionPromptSupported` (defaults to API 33+) so `AndroidTerminalViewModel.requestNotificationPermission` does not rely on stubbed `Build.VERSION.SDK_INT` on JVM unit-test hosts.
- `AndroidTerminalViewModelTest.requestMissingPermissionsRequestsNotificationPermissionWhenRuntimePromptIsSupported` now runs as **PASSED** instead of skipping via `assumeTrue`.
- Re-ran `make android-client-test`.

### 2026-05-09 (Flutter runtime capability monitor parity)

- Added `AndroidClientDependencies.capabilityMonitorIntervalMillis` (default 0 for deterministic JVM tests; `fromContext` sets 2000 ms to match Flutter `_capabilityMonitorInterval`). While connected and foregrounded, `AndroidTerminalViewModel` periodically calls `sendCapabilityDeltaIfChanged("runtime_monitor_poll")`, paused in background like heartbeat/sensor telemetry.
- Extended `AndroidTerminalViewModelTest` (periodic poll, lifecycle pause/resume). Documented in `docs/client-android.md`.

### 2026-05-09 (plan continuation: automated gates)

- Confirmed the native client matches the plan’s automated scope: `make android-client-test`, `make android-client-lint`, `make android-client-build`, `make android-client-compile-android-test`, `./scripts/check-android-client-boundary.sh`, and `./scripts/test-android-client-boundary.sh` all pass on a JDK 17 + Android SDK host. Code-complete items under **Remaining validation** remain physical/network smoke (connected instrumentation, Fire tablet manual connect, real LAN NSD, WebRTC dependency follow-up).

### 2026-05-09 (GitHub Actions instrumentation compile parity)

- Extended `.github/workflows/android-client-ci.yml` so PR/push checks run `./gradlew compileDebugAndroidTestKotlin` alongside `testDebugUnitTest`, `lintDebug`, and `assembleDebug`, matching `make android-client-compile-android-test` / `make all-test` without requiring a connected device or emulator.

### 2026-05-09 (MainActivity kiosk instrumentation)

- Added `AndroidTerminalKioskSmokeTest` (`smoke/AndroidTerminalKioskSmokeTest.kt`) so `./gradlew connectedDebugAndroidTest --tests '*Kiosk*'` exercises production `MainActivity` chrome: local keep-awake, fullscreen, and bright-display toggles and labels.

### 2026-05-09 (WebSocket transport resume token parity)

- Added shared `TransportResumeTokenStore` on `AndroidClientDependencies`, threaded through `CarrierSelectingAndroidControlClient` into `WebSocketAndroidControlClient`.
- WebSocket handshake now sends `TransportHello.resume_token` from the store and captures non-empty `TransportHelloAck.resume_token` after acknowledgement (Flutter `ControlClientTransportHint` parity).
- Re-used `ProtocolBuilders.transportHello` for the opening envelope; added `TransportResumeTokenStoreTest`.
- Documented resume behavior in `docs/client-android.md`.

### 2026-05-09 (resume token docs + automated gate verification)

- Clarified in `docs/client-android.md` that envelope resume hints apply to WebSocket (and similar transports), not the `grpc://` / `grpcs://` carrier.
- Re-ran `make android-client-test`, `make android-client-lint`, `make android-client-build`, `make android-client-compile-android-test`, `./scripts/check-android-client-boundary.sh`, and `./scripts/test-android-client-boundary.sh`.

### 2026-05-09 (CI: emulator instrumentation)

- Added parallel job `android-client-instrumentation` to `.github/workflows/android-client-ci.yml` using `reactivecircus/android-emulator-runner@v2` (API 30, `google_apis`, x86_64) to run `./gradlew connectedDebugAndroidTest`, so instrumentation smoke runs on every qualifying push/PR without a physical tablet.
- Documented PR emulator coverage in `docs/client-android.md` and clarified **Remaining validation** above (CI emulator vs local/Fire tablet smoke).

### 2026-05-10 (Makefile: Gradle stop after Android tests)

- Added `make android-client-gradle-stop` (`./gradlew --stop` with Makefile JDK resolution) and run the same stop step automatically after `make android-client-test` and `make android-client-connected-test`, preserving the test exit code so CI and `make all-test` still fail correctly. Prevents orphaned `Gradle Test Executor` JVMs when runs overlap or agents time out. Documented in `docs/client-android.md`.

### 2026-05-10 (bug report screenshot parity)

- Matched Flutter shell bug-report attachments: `AndroidBugReportBuilder` sets protobuf `screenshot_png` and `screenshot_byte_count` when non-empty bytes are supplied; `AndroidTerminalViewModel` captures via `AndroidClientDependencies.bugReportScreenshotCapture` (production `MainActivity` wires `WindowBugReportScreenshotCapture.capturePngOrNull` on the activity window). Failures or zero-size views omit the field. Added JVM coverage (`AndroidBugReportBuilderTest`, `AndroidTerminalViewModelTest`). Documented in `docs/client-android.md`. Re-run `make android-client-test`, `make android-client-lint`, and boundary scripts on a JDK 17 + Android SDK host.

### 2026-05-10 (renderer scroll + TextWidget color parsing tests)

- Added `ColorParsingTest.parseColorOrUnspecifiedAcceptsValidHexLikeTextWidget` so `parseColorOrUnspecified` coverage matches `TextWidget` color wiring (hash and bare RGB).
- Added `ServerDrivenRendererTest.verticalScrollDeprecatedStringDirectionRendersChildrenInColumn` for legacy `ScrollWidget.direction == "vertical"` (Flutter parity with enum vertical / non-horizontal string). Re-run `make android-client-test` and `make android-client-compile-android-test` on a JDK 17 + Android SDK host.

### 2026-05-10 (Makefile Gradle stop + Apple Silicon gRPC codegen)

- Fixed `Makefile` `android-client-test` and `android-client-connected-test`: the post-task `./gradlew --stop` line used a second `cd android_client` while the shell was already inside `android_client`, so `--stop` never ran and Make reported `cd: android_client: No such directory`.
- `android_client/app/build.gradle.kts`: resolve `protoc-gen-grpc-java` via optional `grpc.java.plugin` in `local.properties`, `GRPC_JAVA_PLUGIN` env, or auto-detected Homebrew `protoc-gen-grpc-java` on macOS aarch64; otherwise keep the Maven artifact (x86_64 / Rosetta on Apple Silicon). Documented prerequisites and overrides in `docs/client-android.md` (**Apple Silicon and gRPC code generation**).

### 2026-05-10 (reconnect test stability hardening)

- Hardened reconnect-path JVM tests in `AndroidTerminalViewModelTest` to always call `disconnect()` in `finally` blocks for reconnect scenarios that start heartbeat jobs (`heartbeatFailureReconnectsWithBoundedBackoff`, `reconnectAttemptCounterTracksLoopAndResetsOnSuccess`, `networkRestoreRetriesConnectAfterReconnectIsExhausted`), preventing stuck `runTest` cleanup when assertions fail mid-test.
- Updated brittle reconnect diagnostics assertions to focus on stable behavior (`Connected` state and endpoint/session transitions) instead of exact transient diagnostics strings that can vary with scheduler timing.
- Re-ran `./gradlew testDebugUnitTest --no-daemon` from `android_client/` on JDK 17+/Android Studio JBR; suite passed.

### 2026-05-09 (Phase 7: immersive/sticky kiosk preference)

- Implemented optional **immersive sticky** as a persisted local terminal setting (`SharedPreferencesAndroidTerminalSettings` / `inMemory`), default **on** (matches prior legacy behavior).
- Extended `AndroidFullscreenController` / `WindowAndroidFullscreenController`: API 30+ uses `BEHAVIOR_SHOW_TRANSIENT_BARS_BY_SWIPE` vs `BEHAVIOR_DEFAULT`; pre-R uses `IMMERSIVE_STICKY` vs `IMMERSIVE`.
- ViewModel applies the preference for local fullscreen, startup restore, server-driven `FullscreenWidget`, and re-applies when toggling sticky while fullscreen is already on; copyable diagnostics include `local_immersive_sticky` alongside other kiosk lines (fixed stale `it` usage when rebuilding diagnostics after local kiosk toggles).
- Terminal chrome: `terminal-local-immersive-sticky-button`; `AndroidTerminalKioskSmokeTest.localImmersiveStickyToggleUpdatesChromeLabel`; JVM and Compose smoke updates.
- Documented in `docs/client-android.md`. Re-verified `make android-client-test`, `make android-client-lint`, `make android-client-compile-android-test`, and boundary scripts.

### 2026-05-10 (MainActivity configuration / orientation smoke)

- Added `MainActivityConfigurationSmokeTest` for production `MainActivity` with manifest `configChanges`: toggling `requestedOrientation` runs `onConfigurationChanged` and terminal chrome shows `last_permission_refresh=configuration` (`refreshPermissionEducation` alongside network + capability refresh). `@After` resets orientation. Assertion uses `ComposeTestRule.waitUntil` so slower emulators stay stable. Filter: `./gradlew connectedDebugAndroidTest --tests '*MainActivityConfiguration*'`.
- Documented example focused `--tests` filters in `docs/client-android.md` (launch, configuration, kiosk, media).
- Re-ran `make android-client-test`, `make android-client-compile-android-test`, `./scripts/check-android-client-boundary.sh`, and `./scripts/test-android-client-boundary.sh`.

### 2026-05-10 (lifecycle diagnostics: combined network + permission refresh)

- Fixed `MainActivity` `onResume` / `onConfigurationChanged` ordering: `refreshPermissionEducation` rebuilt diagnostics and dropped the prior `last_network_refresh` line from `refreshNetworkDiagnostics`. Added `AndroidTerminalViewModel.refreshShellDiagnosticsAndCapabilities` to refresh permission/media education and append both `last_network_refresh` and `last_permission_refresh` in one update, then call `refreshCapabilities` with the lifecycle reason (`activity-resume` vs `display_geometry_change`).
- Extended `MainActivityConfigurationSmokeTest` and JVM coverage (`refreshShellDiagnosticsAndCapabilitiesKeepsNetworkAndPermissionRefreshLines`). Documented in `docs/client-android.md`.

### 2026-05-10 (Flutter shell debug system-query parity)

- Added `ProtocolBuilders.systemCommand` and `AndroidControlSession.sendSystemCommand` / controller wiring for `COMMAND_KIND_SYSTEM` intents (matches Flutter `buildSystemCommandRequest`).
- `AndroidTerminalViewModel` exposes `sendRuntimeStatusQuery` and `sendDeviceStatusQuery` with the same intents as the Flutter shell (`runtime_status`, `device_status <deviceId>`); terminal chrome shows the actions when connected.
- JVM coverage: `ProtocolBuildersTest`, `AndroidControlSessionControllerTest`, `AndroidTerminalViewModelTest.debugSystemQueriesAreSentThroughConnectedSession`; instrumentation fakes updated for the new session method.
- Documented in `docs/client-android.md`. Re-run `make android-client-test`, `make android-client-compile-android-test`, and boundary scripts on a JDK 17 + Android SDK host.

### 2026-05-10 (Phase 9: Konsist UI layering + instrumentation Werror)

- Added `AndroidUiLayeringKonsistTest` (`com.lemonappdev:konsist`) to assert `com.curtcox.terminals.android.ui` production files do not import `connection`, `discovery`, `media`, or `platform` packages — a JVM mirror of the `ui` subtree check in `scripts/check-android-client-boundary.sh`.
- Fixed `compileDebugAndroidTestKotlin` under `allWarningsAsErrors`: legacy-scroll parity tests that call deprecated protobuf `ScrollWidget.setDirection(String)` now use `@Suppress("DEPRECATION")` on the test methods so intentional Flutter/string-direction coverage does not fail the instrumentation compile gate.

### 2026-05-10 (Flutter `privacy.toggle` parity)

- Added privacy mode to `AndroidCapabilitySession` (masks microphone/camera in snapshots and deltas), `AndroidControlSession.setPrivacyMode`, connect/reconnect handshake application from `AndroidTerminalViewState.privacyModeEnabled`, and `AndroidTerminalViewModel.togglePrivacyMode` / interception of server-driven `privacy.toggle` (no spurious `UIAction`). Wired `AndroidMediaEngine.stopLocalCaptureStreamsForPrivacy` through `AndroidLiveMediaSession` (no-op until real live capture tracks streams). Shell **Privacy** button + `privacy_mode` diagnostics; documented in `docs/client-android.md`. JVM: `AndroidCapabilitySessionTest`, `AndroidControlSessionControllerTest.privacyModeStripsMicAndCameraFromCapabilityDelta`, `AndroidTerminalViewModelTest` privacy cases; instrumentation fakes implement `setPrivacyMode`; `MainActivityLaunchSmokeTest` asserts `terminal-privacy-toggle-button`. Minor lint hygiene: removed redundant pre-Lollipop branch in `AndroidNsdDiscovery.terminalTxtMetadata` (minSdk 25), `ManualEndpointParser` path `orEmpty()`, ViewModel `compareBy`/`mutableState.update` clarity; lint baseline line drift. Re-verified `make android-client-test`, `make android-client-lint`, `make android-client-compile-android-test`, and boundary scripts.

### 2026-05-10 (privacy toggle: stop capture only when enabling)

- Matched Flutter `_handlePrivacyToggleAction`: `stopLocalCaptureStreamsForPrivacy` runs only when turning privacy **on** (not when turning it off). JVM: `AndroidTerminalViewModelTest.privacyToggleStopsLocalCaptureOnlyWhenEnablingPrivacy`. Doc tweak in `docs/client-android.md`.

### 2026-05-10 (Flutter shell playback diagnostics parity)

- Added **List playback artifacts** (`COMMAND_KIND_SYSTEM` / `list_playback_artifacts`) and **Playback metadata** (`COMMAND_KIND_MANUAL` / `playback_metadata` with `artifact_id` + `target_device_id` typed args) to match Flutter `buildPlaybackArtifactsQueryRequest` / `buildPlaybackMetadataQueryRequest`.
- `ProtocolBuilders.playbackMetadataCommand`, `AndroidControlSession.sendPlaybackMetadataQuery`, ViewModel fields for artifact/target text inputs, shell chrome (test tags `terminal-debug-playback-artifacts-button`, `terminal-playback-artifact-field`, `terminal-playback-target-device-field`, `terminal-debug-playback-metadata-button`).
- JVM: `ProtocolBuildersTest`, `AndroidControlSessionControllerTest`, `AndroidTerminalViewModelTest`; instrumentation fakes implement the new session method. Documented in `docs/client-android.md`. Re-ran `make android-client-test`, `make android-client-compile-android-test`, and boundary scripts.

### 2026-05-11 (Flutter `CommandResult` diagnostics parity)

- Added `commandResultDataMap`, `diagnosticsTitleForCommandResult`, `applicationIntentsFromDiagnostics`, and `firstPlaybackArtifactId` in `CommandResultDiagnostics.kt` with `CommandDiagnosticsRequestIds` (mirrors Flutter `control_response_dispatcher.dart`).
- `AndroidTerminalViewModel` applies `applyCommandResultDiagnostics` after `ControlResponseDispatcher` so inbound `CommandResult` payloads with data refresh shell state consistently with Flutter: scenario-registry intent list, playback artifact pre-fill, and clearing of pending debug request ids when titles match.
- JVM: `CommandResultDiagnosticsTest`; extended `AndroidTerminalViewModelTest` / `ProtocolBuildersTest` as needed. Documented typed_data vs legacy `data` and classification behavior in `docs/client-android.md`.
- Re-verified `make android-client-test`, `make android-client-lint`, `make android-client-compile-android-test`, `./scripts/check-android-client-boundary.sh`, and `./scripts/test-android-client-boundary.sh`.

### 2026-05-11 (Compose smoke: connected debug chrome → session)

- Extended `AndroidTerminalAppSmokeTest` so the fake session records `sendSystemCommand`, `sendPlaybackMetadataQuery`, and `sendApplicationLaunchCommand`; added Compose coverage for **Runtime status**, **Device status**, **List playback artifacts** + **Refresh applications**, **Playback metadata** (default target device id), and **Open application** after a successful connect. Closes the gap between JVM-only debug-query tests and UI wiring on the generic shell.
- Re-verified `make android-client-test`, `make android-client-compile-android-test`, and boundary scripts on a JDK 17 + Android SDK host.

### 2026-05-11 (Flutter shell: **Open application** waits for `RegisterAck`)

- Matched Flutter `_launchSelectedApplication` / `_pendingLaunchApplicationIntent`: `AndroidTerminalViewModel` queues the selected intent in `AndroidTerminalViewState` (`applicationLaunchQueuedIntent`) until any inbound `RegisterAck`, then runs `sendApplicationLaunchNow` after the first-ack automatic `scenario_registry` dispatch; copyable diagnostics include `application_launch_queued_until_register_ack=` while waiting.
- `handleControlLoss` clears `sawRegisterAck`, `registerAckScenarioQuerySent`, and the queued intent so reconnect sessions repeat the hello / RegisterAck handshake correctly.
- JVM: `applicationLaunchQueuesUntilRegisterAckThenSends`; tightened `submitApplicationLaunchCommandSendsManualStart` (RegisterAck before launch). Compose: `AndroidTerminalAppSmokeTest.connectedOpenApplicationSendsManualLaunchCommand` delivers `RegisterAck` before **Open application**.
- Documented in `docs/client-android.md`. Re-verified `./gradlew testDebugUnitTest`, `compileDebugAndroidTestKotlin`, `lintDebug`, and Android boundary scripts with `JAVA_HOME` pointing at JDK 17.

### 2026-05-11 (instrumentation: RegisterAck → scenario registry + CommandResult intents)

- Added `AndroidTerminalAppSmokeTest.connectedRegisterAckTriggersAutomaticScenarioRegistryQuery` so the Compose shell asserts the first inbound `RegisterAck` dispatches exactly one automatic `scenario_registry` system command (Flutter first-ack parity) and surfaces `last_system_command=` in chrome.
- Added `AndroidTerminalAppSmokeTest.connectedScenarioRegistryCommandResultUpdatesAvailableIntents` so a matching `CommandResult` after that query updates `availableApplicationIntents` end-to-end (mirrors JVM `scenarioRegistryCommandResultUpdatesApplicationIntents` through the UI stack).
- Re-verified `make android-client-test`, `./gradlew compileDebugAndroidTestKotlin`, and Android boundary scripts.

### 2026-05-11 (playback metadata: explicit target device tests)

- Added `AndroidTerminalViewModelTest.playbackMetadataUsesExplicitTargetDeviceWhenProvided` so JVM coverage matches Flutter shell behavior when **Target device (optional)** is non-empty (no default substitution to `deviceId`).
- Added `AndroidTerminalAppSmokeTest.connectedDebugPlaybackMetadataSendsManualQueryWithExplicitTargetDevice` so Compose instrumentation exercises `terminal-playback-target-device-field` end-to-end with the fake session.
- Re-run `make android-client-test`, `make android-client-compile-android-test`, `./scripts/check-android-client-boundary.sh`, and `./scripts/test-android-client-boundary.sh` on a JDK 17 + Android SDK host.

### 2026-05-11 (MainActivity: copy diagnostics smoke)

- Tagged diagnostics copy feedback with `terminal-diagnostics-copy-status` in `AndroidTerminalApp` for stable instrumentation.
- Extended `MainActivityLaunchSmokeTest` with `copyDiagnosticsFromMainActivityShowsCopiedStatus` so the real launcher path (production `ContextDiagnosticClipboard`) asserts **Copy diagnostics** surfaces `copied`, advancing the plan **Device smoke tests** item on copyable diagnostics toward CI/emulator coverage.

### 2026-05-11 (terminal shell scroll + MainActivityLaunch on small screens)

- Wrapped the native terminal shell `Column` in `verticalScroll` so lower chrome (privacy, copy diagnostics, report bug, etc.) is reachable on short viewports (small-phone emulators, dense DPI).
- `MainActivityLaunchSmokeTest` uses `performScrollTo()` before asserting or clicking those tags so connected runs stay stable.
- Corrected `docs/client-android.md` focused instrumentation examples: use `-Pandroid.testInstrumentationRunnerArguments.class=…` (or `package=…`) instead of unsupported `--tests` on `connectedDebugAndroidTest`.
- Re-verified on `Small_Phone(AVD)` with JDK 17: `./gradlew testDebugUnitTest`, `compileDebugAndroidTestKotlin`, and `connectedDebugAndroidTest` with `class=com.curtcox.terminals.android.smoke.MainActivityLaunchSmokeTest`.

### 2026-05-11 (Flutter shell: inbound control response counter)

- Added `AndroidTerminalViewState.inboundConnectResponseCount`, incremented on each handled `ConnectResponse` in `AndroidTerminalViewModel` (parity with Flutter `terminal_client_shell` `Responses: $_responses`).
- Copyable diagnostics include `inbound_connect_response_count=` alongside existing outbound heartbeat/sensor and `stream_ready_send_count` lines; counter resets with `withoutHandshake` and on reconnect session refresh.
- JVM: extended `serverSetUiResponseUpdatesRenderedRoot`; added `inboundConnectResponseCountIncrementsPerControlMessage`. Re-verified `make android-client-test` and Android boundary scripts on JDK 17.

### 2026-05-11 (Flutter shell: visible Responses + sensor telemetry chrome)

- `AndroidTerminalApp` now shows the same on-screen counters as Flutter `terminal_client_shell` after **Last server activity**: `Responses:` (backed by `inboundConnectResponseCount`) and the sensor / stream-ready / capability-ack line (`terminal-responses-count`, `terminal-sensor-telemetry-line` test tags). Flutter’s separate **Media routes / Active streams / Signals** line remains unmirrored until native live-media bookkeeping exists.
- `MainActivityLaunchSmokeTest` scrolls to and asserts the new tags; `AndroidTerminalAppSmokeTest.manualEndpointConnectsRendersServerUiAndDispatchesAction` asserts `Responses: 1` after a synthetic `SetUI` inbound message.

