---
title: "Fire Tablet Validation — Progress Log"
plan: "plans/features/android-client-fire-tablet-validation/plan.md"
last-updated: 2026-05-12
---

# Progress Log

## 2026-05-12 — Phase C: Update docs to reflect enabled WebRTC posture

Updated `docs/client-android.md` to remove stale "WebRTC media transport remains
explicitly disabled" language. Now describes the enabled posture: `WebRtcSdkAndroidAdapter`
initializes `PeerConnectionFactory` at startup and falls back to `disabled(...)` automatically
on init failure. Added `webrtc.sdk.version` local.properties override docs and explained the
runtime fallback diagnostic (`last_live_media=start_stream:<id>:<reason>`). Phase C is **code and docs complete** (automated gates pass); the plan marks on-device live-audio acceptance as pending Phase A hardware.

---

## 2026-05-12 — Phase C: Fix compileDebugAndroidTestKotlin

`AndroidTerminalMediaSmokeTest.FakeSession` was missing `sendWebRtcSignal` (added to
`AndroidControlSession` interface as part of Phase C Step 2). Added the no-op override.
All four automated gates now pass: `android-client-test`, `android-client-lint`,
`android-client-build`, `compileDebugAndroidTestKotlin`, plus both boundary scripts.

## 2026-05-12 — Phase C Step 2: WebRTC Library Wiring

**Status:** Complete. All JVM unit tests pass. Detekt clean on new code.

### Changes shipped

- `app/build.gradle.kts`: added `io.github.webrtc-sdk:android:144.7559.05` with `local.properties` version override (`webrtc.sdk.version`)
- `WebRtcSdkAndroidAdapter` (new): implements `AndroidWebRtcAdapter`, initialises `PeerConnectionFactory`, reports `supported=true`
- `WebRtcSdkLiveMediaSession` (new): implements `AndroidLiveMediaSession`, manages one `PeerConnection` per stream ID, handles SDP OFFER/ANSWER and ICE candidates
- `AndroidControlSession` + `AndroidControlSessionController`: added `sendWebRtcSignal` to forward client-generated signals (ICE candidates, SDP answer) to the server
- `ProtocolBuilders`: added `webRtcSignal` request builder
- `AndroidClientDependencies.fromContext`: wires `WebRtcSdkAndroidAdapter` and `WebRtcSdkLiveMediaSession` with a capture-variable pattern for the signal sender
- `AndroidTerminalInboundSink`: `start_stream` and `webrtc_signal` now record `applied` in `last_live_media` diagnostics
- Fixed pre-existing missing `kotlinx.coroutines.flow.update` imports in 4 extension files
- New JVM tests: `AndroidMediaEngineTest` (6 tests), `AndroidTerminalViewModelServerResponseTest` (4 tests)
- New smoke test: `AndroidTerminalAppSmokeTest.startStreamAndWebRtcSignalWithSupportedAdapterRecordAppliedLiveMediaLines`

### Remaining for full Phase C completion

On-device validation (phases A, B, D) requires physical Fire OS 6+ hardware.

---

## 2026-05-11 — Phase C Step 1: WebRTC Dependency Survey

**Status:** Survey complete. Dependency selected. Code wiring not yet started.

### Candidates evaluated

#### 1. `io.github.webrtc-sdk:android` — **SELECTED**

| Attribute | Value |
|-----------|-------|
| Latest version | `144.7559.05` |
| Released | 2026-04-30 |
| License | BSD-3-Clause (GitHub shows MIT — both are permissive) |
| Hosted | Maven Central |
| Google Play Services | None declared |
| Transitive deps | None (self-contained native AAR) |
| Variants | `android`, `android-prefixed`, `android-prefixed-stripped` |

Rationale:
- Pre-compiled native WebRTC AAR — no Google Play Services transitive pull. The boundary
  script at `scripts/check-android-client-boundary.sh` checks `app/build.gradle.kts` for
  `play-services|firebase|com.google.android.gms` patterns; this dependency will not trigger it.
- Active maintenance cadence: v144.7559.05 released 2026-04-30.
- The same library underpins LiveKit Android SDK (`android-prefixed` variant), confirming it
  is production-tested at scale on standard Android API levels.
- `minSdk = 25` (Fire OS 6+) is comfortably within WebRTC-for-Android's supported range
  (native WebRTC targets API 21+).
- Use the `android-prefixed-stripped` variant if APK size is a concern after initial
  integration; start with `android` to keep the org.webrtc package namespace familiar.

Gradle coordinate to add to `android_client/app/build.gradle.kts`:

```kotlin
implementation("io.github.webrtc-sdk:android:144.7559.05")
```

Add a `local.properties` override property (e.g. `webrtc.sdk.version`) to allow pinning or
disabling without code changes, matching the existing `grpc.java.plugin` pattern.

#### 2. `org.webrtc:google-webrtc` — **REJECTED**

Last published: 2021. Google has not released an updated AAR since the project moved to
Chromium's internal build system. The artifact is archived and unmaintained. Do not use.

#### 3. `io.livekit:livekit-android` v2.25.2 — **NOT SELECTED (overkill)**

License: Apache 2.0. No Google Play Services dependency. However, LiveKit is a high-level
SDK that bundles signaling, room management, participant tracking, and DataChannel helpers
on top of WebRTC. Adopting it would couple the Android terminal to LiveKit's server-side
signaling model, violating the thin-client rule ("behavior on the server, not the client").
The project's existing `AndroidWebRtcAdapter` / `AndroidLiveMediaSession` seams are designed
for a raw WebRTC stack, not a managed-room abstraction.

LiveKit internally uses `io.github.webrtc-sdk:android-prefixed` — using the raw SDK directly
gets the same native layer without the coupling.

#### 4. Jitsi-style SDKs (org.jitsi / lib-jitsi-meet) — **NOT EVALUATED**

`lib-jitsi-meet` targets web (JavaScript). The `org.jitsi:jitsi-meet-sdk` Android artifact
bundles React Native and is not suitable for a native Kotlin client. Skipped.

### Next step

Phase C Step 2: add `io.github.webrtc-sdk:android:144.7559.05` to `app/build.gradle.kts`
with a `local.properties` version override, implement `WebRtcSdkAndroidAdapter`, and extend
`WebRtcGatedLiveMediaSession` with real StartStream / WebRTCSignal handling. Confirm boundary
scripts still pass.

---

## Phases A, B, D (device + evidence)

Not started for **device evidence** (hardware smoke, LAN notes, pasted diagnostics). `progress.md` already records Phase C implementation; fill evidence sections after Phases A–B on a Fire OS 6+ tablet.
