---
title: "Android Client: Fire Tablet Validation & WebRTC Enablement"
kind: plan
status: building
owner: curtcox
validation: manual
last-reviewed: 2026-05-18
---

# Android Client: Fire Tablet Validation & WebRTC Enablement

Target repository path: `plans/features/android-client-fire-tablet-validation/plan.md`

Continuation of [`plans/features/android-client/plan.md`](../android-client/plan.md), which is code-complete (`shipped-validated`). All ten implementation phases of the native Android client are merged and pass automated repository gates plus the CI emulator instrumentation job. The remaining work is **physical** validation on Fire OS (smoke tests, real LAN discovery, and on-device live WebRTC audio) — none of which can be closed by the existing automated gates.

Preserves the repository rules from `AGENTS.md`, `CLAUDE.md`, `masterplan.md`, and the parent plan:

> Add behavior on the server, not the client. The client remains a generic terminal.

## Problem

The native Android client under `android_client/` ships behind two open gates:

1. **No physical Fire tablet has been used to confirm protocol parity end-to-end.** CI runs `connectedDebugAndroidTest` on a `reactivecircus/android-emulator-runner@v2` API 30 `google_apis` x86_64 emulator. That covers instrumentation regressions but does not exercise Fire OS 6+ devices, Amazon-tablet network stacks, Fire OS power/wake behavior, or Fire-tablet-specific permission UX.
2. **Live WebRTC on real Fire hardware is unvalidated.** `io.github.webrtc-sdk:android` is wired through `WebRtcSdkAndroidAdapter` + `WebRtcSdkLiveMediaSession` with a `disabled(...)` fallback when the engine does not load. Emulator and JVM tests cover the enabled paths, but no Fire tablet has yet confirmed an audible server-issued `StartStream` or capture lifecycle end-to-end.

These items appear under **Remaining validation** in the parent plan but never block CI, so they tend to be deferred. Splitting them into a dedicated plan makes them trackable independent of further protocol/parity work landing against the now-complete parent plan.

## Goals

- Confirm the existing native Android client works end-to-end on at least one Kindle Fire tablet running Fire OS 6 or newer.
- Confirm Android NSD/mDNS discovery functions on a multicast-capable home Wi-Fi network, or that the documented manual-connect fallback engages cleanly when it does not.
- Add captured evidence (logs, copied diagnostics, screenshots, or short notes) for each Fire-tablet smoke step from the parent plan's **Device smoke tests** section.
- Select a Fire-OS-compatible WebRTC dependency, wire it through the existing `AndroidWebRtcAdapter` + `AndroidLiveMediaSession` seams, and enable live media without removing the disabled-fallback path.
- Keep the client a generic terminal: no scenario-specific code, no Google Play Services, no Amazon-account or Alexa coupling.

## Non-Goals

- No new protocol features. If the device run uncovers a missing protocol affordance, open a separate protocol-evolution plan rather than expanding this one.
- No visual redesign of server-driven UI.
- No Fire TV, Wear OS, Android Auto, or ChromeOS support.
- No Fire OS 5 (API < 25) support — covered by a separate follow-up if needed.
- No Amazon Appstore submission.
- No replacement of the Flutter client.
- No removal of `AndroidWebRtcAdapter.disabled(...)` as a code path; it must remain available when the selected WebRTC dependency is absent, fails to initialize, or is gated off at runtime.

## Current State

- `android_client/` builds a debug APK that installs via `adb install` on Fire tablets per `docs/client-android.md`.
- `AndroidClientDependencies.fromContext` wires `WebRtcSdkAndroidAdapter` + `WebRtcSdkLiveMediaSession` when the WebRTC AAR loads; on init failure or absent dependency it falls back to `AndroidWebRtcAdapter.disabled(...)` and the gated not-implemented session (same diagnostic shape as before).
- JVM and instrumentation tests cover enabled-adapter paths; smoke records non-`Unsupported` `last_live_media` lines when the fake supported adapter is used. Real-device audio playback for server-issued `StartStream` is still unconfirmed pending Phase A hardware.
- CI emulator runs cover orientation, lifecycle, permission, kiosk, discovery-fallback, media-command, bug-report, copy-diagnostics, and reconnect smoke paths.
- `docs/client-android.md` documents Fire developer-mode, ADB install, manual endpoint format (`ws://`, `wss://`, `grpc://`, `grpcs://`), NSD/mDNS quirks, kiosk toggles, WebRTC SDK version override (`webrtc.sdk.version`), and runtime fallback to the disabled adapter.
- No physical Fire tablet smoke results have been recorded in this repository (`progress.md` logs code milestones; device evidence sections remain to be filled after Phases A–B).

## Phases

### Phase A: Fire Tablet Smoke

Status: not started

Tasks:

1. Pick a baseline Fire tablet (Fire HD 8 Plus or Fire HD 10, Fire OS 6+ preferred). Record manufacturer, model, Fire OS version, and Android API level from **Settings → Device Options → System Updates / About**.
2. Enable developer options and ADB on the tablet (`Settings → Device Options → tap Serial Number 7×`).
3. Install the latest debug APK: `cd android_client && ./gradlew assembleDebug && adb install -r app/build/outputs/apk/debug/app-debug.apk`.
4. Run `make run-server` on a workstation on the same LAN. Note the LAN IP and the gRPC + WebSocket ports the server prints.
5. Execute every step from the parent plan's **Device smoke tests** (parent plan §Test Plan → Device smoke tests, items 1–10):
   - Install debug APK with ADB.
   - Start server with `make run-server` on the same LAN.
   - Open Android client and connect by manual endpoint (try both `ws://host:port` and `grpc://host:port`).
   - Verify capability snapshot reaches the server (check `terminal_server` logs for the hello + capability snapshot lines).
   - Push a server-driven UI tree (use an existing scenario that issues `SetUI`, or the debug **Refresh applications** + **Open application** flow with a registered scenario).
   - Tap a button and verify the resulting `UIAction` reaches the server.
   - Rotate the tablet and verify a `display_geometry_change` capability delta reaches the server.
   - Toggle Wi-Fi off and on; verify diagnostics show reconnect attempts within the configured `ReconnectPolicy` bound and that a successful reconnect surfaces `reconnect_success` in copyable diagnostics.
   - Enable the local **Keep awake** kiosk toggle and verify the display stays on for the duration of the smoke window.
   - Tap **Copy diagnostics**; paste the result into the validation evidence file.
6. Repeat the smoke run while the tablet is **backgrounded** (press Home, wait ≥30s, return) to confirm `app_lifecycle_change` capability deltas are sent on both transitions and that heartbeat/sensor loops pause in background and resume on foreground (parity behavior already covered by JVM tests).
7. Try the **Report bug** chrome with both an online session and an offline session (Wi-Fi off → tap **Report bug** → restore Wi-Fi → confirm `BugReportAck` line appears in diagnostics after flush).

Acceptance criteria:

- Each smoke step has a recorded result (pass / fail / partial) in the validation evidence file below.
- Manual connection works against `make run-server` on at least one Fire OS 6+ device with both WebSocket and gRPC carrier URLs.
- Any failure is either fixed in the client (with the fix added to the parent plan's progress log) or filed as a follow-up issue referenced from this plan.

### Phase B: LAN Discovery on Real Wi-Fi

Status: not started

Tasks:

1. Identify a home/office Wi-Fi network whose AP is known to forward mDNS multicast (most consumer routers do; many enterprise APs do not, and AP isolation blocks it). Record SSID type (consumer / enterprise) and any AP-isolation setting if visible.
2. With the server running, tap **Start discovery** in the Android shell. Confirm the discovered `_terminals._tcp.` service appears, includes TXT carrier/priority metadata, and that selecting it populates the manual endpoint (gRPC TXT values must be normalized to `grpc://…` per the existing `AndroidTerminalViewModel` rule).
3. Disconnect the tablet from that Wi-Fi and reconnect; confirm `AndroidNetworkMonitor` callbacks restart discovery (subject to the existing debounce) and that diagnostics show `discovery_restart` or `discovery_restart_suppressed=app-background` as appropriate.
4. Try a network where multicast is blocked (Fire OS guest Wi-Fi, an enterprise AP with isolation, or a hotspot). Confirm `formatNsdFailureDetail` surfaces a generic, copyable failure hint and that the manual endpoint flow still works.

Acceptance criteria:

- At least one network where discovery succeeds end-to-end is documented.
- At least one network where discovery fails is documented along with the exact diagnostic line surfaced (so the existing fallback copy in `docs/client-android.md` can be confirmed accurate).
- No additional scenario branching introduced in `discovery/` or `app/` to make discovery work.

### Phase C: WebRTC Dependency Selection & Enablement

Status: code complete in repo (2026-05-12); on-device live-audio acceptance from Phase C acceptance criteria awaits Phase A Fire hardware run.

Tasks:

1. **Survey candidates.** Evaluate at least:
   - `io.github.webrtc-sdk:android` (community fork, AAR, no Google Play dependency).
   - `org.webrtc:google-webrtc` (deprecated, last published 2021 — likely unsuitable; document why).
   - `livekit-client` / `org.jitsi:libjitsi-meet` style higher-level SDKs (only if they remain Fire-OS-compatible and Google-service-free).
   For each, record: license, last release date, `minSdk`, transitive Google Play Services usage (`./gradlew app:dependencies | grep -i play`), AAR size impact, and Fire OS install footprint.
2. **Pick one** and update `android_client/app/build.gradle.kts`. Add a `local.properties` / Gradle property override (similar to the existing `grpc.java.plugin` pattern) so the dependency can be swapped or disabled without code changes.
3. **Wire it through the existing seams.** Implement a new `AndroidWebRtcAdapter` variant (e.g. `WebRtcSdkAndroidAdapter`) that returns `supported=true` with the real engine reason, and an `AndroidLiveMediaSession` implementation backed by the chosen SDK. Both must be selected by `AndroidClientDependencies.fromContext` only when the dependency loads at runtime; otherwise the existing `disabled(...)` adapter and `live-media-session-not-implemented` session must remain.
4. **Capability lifecycle.** Confirm live media is only advertised when (a) the adapter reports supported, (b) microphone/camera permissions match the requested media direction, and (c) privacy mode is off. The existing `AndroidCapabilitySession` privacy mask and `stopLocalCaptureStreamsForPrivacy` plumbing must continue to apply.
5. **Tests.**
   - JVM: extend `AndroidMediaEngineTest` and `AndroidTerminalViewModelTest` with a `supported=true` adapter fake to cover the previously-uncovered enabled paths (`StartStream` returns a non-error result, `WebRTCSignal` round-trips, `StopStream` releases the session, `RouteStream` updates routing metadata).
   - Instrumentation: extend `AndroidTerminalAppSmokeTest` so a synthetic `StartStream` + `WebRTCSignal` sequence with the fake supported adapter records non-`Unsupported` `last_live_media` lines.
   - Device: confirm an actual audio-only `StartStream` from a server scenario produces audible playback on the Fire tablet and a follow-up `StopStream` cleanly releases the session (no residual mic/camera capture, no leaked notifications).
6. **Docs.** Update `docs/client-android.md` "Live media transport" / WebRTC sections to describe the enabled posture, the override property, and how to fall back to the disabled adapter for debugging.
7. **Boundary scan.** Re-run `./scripts/check-android-client-boundary.sh` and `./scripts/test-android-client-boundary.sh` to confirm no Google Play Services / scenario-name leak from the new dependency.

Acceptance criteria:

- Fire tablet plays back at least one server-issued live audio stream end-to-end without crashing or leaking capture.
- Live-media capability is advertised only when the adapter is actually supported; `disabled(...)` remains the default when the dependency cannot load.
- `make android-client-test`, `make android-client-lint`, `make android-client-build`, `make android-client-compile-android-test`, and both boundary scripts pass on a JDK 17 + Android SDK host.
- CI emulator `connectedDebugAndroidTest` still passes (use the disabled adapter for emulator runs unless the chosen SDK supports x86_64 emulators).

### Phase D: Evidence & Plan Closure

Status: in progress — `progress.md` exists and records Phase C implementation; paste device diagnostics and LAN notes when Phases A–B complete.

Tasks:

1. ~~Create~~ Maintain `plans/features/android-client-fire-tablet-validation/progress.md` (mirror the pattern from `plans/features/io-abstraction/progress.md`). Record:
   - Tablet hardware (manufacturer, model, Fire OS version, API level).
   - LAN details (router/AP, multicast behavior).
   - Pasted **Copy diagnostics** output from a successful connected session.
   - Pasted **Copy diagnostics** output after an enabled live-media session.
   - Chosen WebRTC dependency, version, license, and last-release date.
2. Update `last-reviewed:` here as smoke and WebRTC work lands; flip status to `shipped-validated` only after Phases A–C all pass.
3. Append a closing entry to the parent plan's **Implementation Progress** log noting the validation outcome and linking back to this plan.

Acceptance criteria:

- `progress.md` exists, is indexed by `./scripts/generate-plans-index.py`, and contains real device output rather than placeholders.
- Parent plan retains `status: shipped-validated`; this plan flips to `status: shipped-validated` once Phases A–C complete.

## Validation Commands

Automated gates (must still pass on every PR under this plan, even though the focus is manual validation):

```bash
make android-client-test
make android-client-lint
make android-client-build
make android-client-compile-android-test
./scripts/check-android-client-boundary.sh
./scripts/test-android-client-boundary.sh
```

Device-only validation (cannot run from CI):

```bash
adb devices                                                    # confirm Fire tablet is attached
cd android_client && ./gradlew connectedDebugAndroidTest       # full instrumentation on tablet
make android-client-connected-test                             # preflighted wrapper (skips with guidance if no device)
```

For focused instrumentation filters on the connected tablet, use the runner-argument form documented in `docs/client-android.md` (Compose `connectedDebugAndroidTest` does not accept `--tests`):

```bash
cd android_client && ./gradlew connectedDebugAndroidTest \
  -Pandroid.testInstrumentationRunnerArguments.class=com.curtcox.terminals.android.smoke.MainActivityLaunchSmokeTest
```

## Validation Evidence

To be filled in under `progress.md` as smoke results land. Required artifacts:

- Diagnostics paste from manual connect over WebSocket.
- Diagnostics paste from manual connect over gRPC.
- Diagnostics paste after orientation change (must include `last_capability_delta=display_geometry_change`).
- Diagnostics paste after Wi-Fi toggle reconnect (must include `reconnect_success` and an updated `inbound_connect_response_count`).
- Diagnostics paste after an offline → online **Report bug** flush (must include `last_bug_report_ack_*`).
- Diagnostics paste from an enabled live-media `StartStream` (must include a non-`Unsupported` `last_live_media` line).

## Review Checklist

Every PR under this plan should answer:

- Did this PR keep the Android client a generic terminal?
- Did this PR avoid adding scenario-specific branches or Google service dependencies?
- If a new WebRTC dependency was added, does `./gradlew app:dependencies` confirm no Google Play Services transitive pull?
- Does the disabled `AndroidWebRtcAdapter` path still work when the dependency is removed?
- Does capability reporting still withhold live-media when permissions / privacy mode require it?
- Are new tests added at the level the change demands (JVM for pure logic, instrumentation for UI/platform behavior)?
- Are Fire tablet observations (or lack of access) explicit in the PR description?

## Risks & Open Questions

- **Fire tablet availability.** The plan requires at least one physical Fire OS 6+ device. If unavailable, Phase A blocks indefinitely; consider documenting an Amazon-emulator workaround if one exists, but treat it as a partial result.
- **Multicast on home networks.** Some routers silently drop mDNS; Phase B should not be treated as a failure of the client if the network is the bottleneck. Record the network behavior, then exercise the documented manual fallback.
- **WebRTC SDK churn.** The community `webrtc-sdk/android` fork is the most likely candidate but its release cadence is irregular. If no acceptable Fire-OS-compatible build exists, Phase C can land as a documented decision to keep the disabled adapter rather than a code change — but that decision must be recorded in `progress.md` and `docs/client-android.md`.
- **Live capture privacy.** Enabling WebRTC enables actual microphone/camera capture for the first time. Re-audit privacy-toggle plumbing and the existing `stopLocalCaptureStreamsForPrivacy` path on a real device before merging Phase C.
