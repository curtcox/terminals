---
title: "Implementation Risk Audit (Cross-Client)"
kind: audit
status: resolved
owner: cascade
validation: none
last-reviewed: 2026-04-26
---

# Implementation Risk Audit (Cross-Client)

## Scope and Inputs

This audit was produced by inferring missing placeholders from the repository state:

- Plan docs: [`masterplan.md`](../archive/masterplan-duplicate.md), `plans/phase-*.md`, and supporting docs in `plans/` + `docs/`.
- Target clients/platforms: Android, iOS, Web/Browser, macOS, Linux, Windows (from `plans/architecture-client.md`).
- Relevant code/config: `terminal_client/`, `terminal_server/`, `api/`, scripts, and CI workflow files.

## Rebaseline (2026-04-26)

This audit has been re-validated against current repository state and is now **resolved**.

Addressed high-severity findings:

- **P0-1 (6-platform workflow mismatch)**: platform scaffolding and build lanes now exist in repo and CI (`terminal_client/android`, `terminal_client/ios`, `terminal_client/linux`, `terminal_client/windows`, `Makefile` `client-build-*`, and `.github/workflows/client-ci.yml` build matrix).
- **P0-2 (web raw gRPC control path)**: browser-compatible transports are implemented and selected in client factory (`terminal_client/lib/connection/control_client_factory.dart`, `terminal_client/lib/connection/control_client_ws.dart`) with server websocket endpoint (`terminal_server/internal/transport/websocket_server.go`, tests in `terminal_server/internal/transport/websocket_server_test.go`).
- **P0-3 (missing real gRPC listener wiring)**: transport now binds real sockets with `grpc.NewServer` + generated service registration and serve/stop lifecycle (`terminal_server/internal/transport/grpc_server.go`, `terminal_server/internal/transport/grpc_service.go`, startup wiring in `terminal_server/cmd/server/main.go`).

Addressed remaining remediation findings (tracked in Stage 4-9 and now closed):

- **P0-4/P0-5/P0-6**: media output parity, capability truthfulness, and durable edge persistence/export behavior were completed with unit/widget/integration coverage and reflected in client/server docs.
- **P1-1/P1-2/P2-1**: permissions hardening, monitoring support-tier behavior, and native/web alert delivery parity are now implemented and documented.

Closure evidence:

- `make all-check` passed on 2026-04-26.
- `plans/audits/implementation-risk-remediation.md` was drained to docs and marked `superseded`.

---

## A) Capability Matrix

Legend per cell:

- `Supported`
- `Partial`
- `Unsupported`
- `Unknown` (treated as release blocker)

| Required capability | Web | macOS | Android | iOS | Linux | Windows |
|---|---|---|---|---|---|---|
| 1. Target is build/run-ready in current repo workflows | Supported [E2,E3] | Partial [E2] | Unsupported [E2,E3] | Unsupported [E2,E3] | Unsupported [E2,E3] | Unsupported [E2,E3] |
| 2. End-to-end control transport reachable | Unsupported [E4,E5] | Unknown [E4,E6] | Unsupported [E2] | Unsupported [E2] | Unsupported [E2] | Unsupported [E2] |
| 3. mDNS auto-discovery | Unknown [E7] | Partial [E7,E8] | Unsupported [E2] | Unsupported [E2] | Unsupported [E2] | Unsupported [E2] |
| 4. Manual host:port connect | Supported [E9] | Supported [E9] | Unsupported [E2] | Unsupported [E2] | Unsupported [E2] | Unsupported [E2] |
| 5. Mic capture usable for voice/intercom | Partial [E10] | Unsupported [E10,E14] | Unsupported [E2] | Unsupported [E2] | Unsupported [E2] | Unsupported [E2] |
| 6. Camera workflows (capture + capability declaration) | Unsupported [E10,E11] | Unsupported [E10,E11] | Unsupported [E2] | Unsupported [E2] | Unsupported [E2] | Unsupported [E2] |
| 7. Media receive/render/playback | Unsupported [E12] | Unsupported [E12] | Unsupported [E2] | Unsupported [E2] | Unsupported [E2] | Unsupported [E2] |
| 8. Real sensor/IMU telemetry | Unsupported [E13,E17] | Unsupported [E13,E17] | Unsupported [E2] | Unsupported [E2] | Unsupported [E2] | Unsupported [E2] |
| 9. Background monitoring while idle | Unknown [E18,E17] | Unknown [E18,E17] | Unsupported [E2] | Unsupported [E2] | Unsupported [E2] | Unsupported [E2] |
| 10. Notifications + speech UX | Partial [E15,E16] | Partial [E15,E16] | Unsupported [E2] | Unsupported [E2] | Unsupported [E2] | Unsupported [E2] |
| 11. Edge retention/artifact persistence | Unsupported [E19,E17] | Unsupported [E19,E17] | Unsupported [E2] | Unsupported [E2] | Unsupported [E2] | Unsupported [E2] |
| 12. Bluetooth/USB passthrough | Unsupported [E11,E20,E17] | Unsupported [E11,E20,E17] | Unsupported [E2] | Unsupported [E2] | Unsupported [E2] | Unsupported [E2] |
| 13. DataChannel low-latency path | Unsupported [E21] | Unsupported [E21] | Unsupported [E2] | Unsupported [E2] | Unsupported [E2] | Unsupported [E2] |

---

## B) Mismatch Findings (Sorted by Severity)

### P0-1: Declared 6-client plan vs 2-client implementation workflow

- Why it fails (root cause):
  - Plan and docs declare 6 platforms, but local launcher and CI/workflow only operate on web/macos paths in this repo state.
- Fix options per client:
  - Web/macOS: keep current support and harden.
  - Android/iOS/Linux/Windows: scaffold and commit platform targets, permissions, and run/build paths before claiming parity.
- Required plan changes:
  - Add a platform readiness gate before any phase is marked complete.
- Required code/config changes:
  - Add missing Flutter platform directories and platform-specific project files.
  - Expand scripts/Make targets for all clients.
- Test coverage to add:
  - Unit: platform readiness checks for missing target directories.
  - Integration: `flutter build` matrix for all 6 targets.
  - E2E: connect/register smoke tests per platform target.

### P0-2: Web control path depends on unsupported raw gRPC sockets

- Why it fails (root cause):
  - Browser client uses raw `grpc` `ClientChannel`; diagnostics/tests already encode browser raw socket failure.
- Fix options per client:
  - Web: use gRPC-Web (proxy like Envoy) or browser-compatible transport adapter.
  - Native clients: keep raw gRPC if server listener exists.
- Required plan changes:
  - Split control transport into native lane and browser lane with explicit acceptance criteria.
- Required code/config changes:
  - Add web-specific control client implementation.
  - Add gRPC-Web proxy config/docs and local dev wiring.
- Test coverage to add:
  - Unit: web transport adapter path selection.
  - Integration: web client connects via proxy, receives register ack.
  - E2E: browser connect + reconnect + command roundtrip using gRPC-Web.

### P0-3: Server-side gRPC listener wiring is missing/unclear

- Why it fails (root cause):
  - Transport `Start()` currently marks running/logs but does not show actual socket/server startup in transport layer.
- Fix options per client:
  - All clients: blocked until real listener and streaming handler are live.
- Required plan changes:
  - Add “real network listener and Connect stream reachable” as non-negotiable phase-1 milestone.
- Required code/config changes:
  - Implement real `grpc.NewServer`, register generated service, bind with `net.Listen`, serve/stop lifecycle.
- Test coverage to add:
  - Unit: server lifecycle validates listener state.
  - Integration: open TCP port, generated gRPC client connects and exchanges register/heartbeat.
  - E2E: scripted local run verifies actual stream operations, not only logs.

### P0-4: Media receive/render/playback capabilities required by plan are not implemented

- Why it fails (root cause):
  - `video_surface` and `audio_visualizer` are rendered as placeholders; `PlayAudio` handling updates status counters only.
- Fix options per client:
  - Web/macOS first: implement real media rendering and playback primitives.
  - Other clients: port after scaffold readiness.
- Required plan changes:
  - Phase-3/4 completion criteria must require real output behavior (audible/visible), not signaling alone.
- Required code/config changes:
  - Bind remote tracks to renderers.
  - Implement playback for `url`, `pcm_data`, and `tts_text` paths.
- Test coverage to add:
  - Unit: widget primitives map to active media state.
  - Integration: start/stop stream causes renderer attach/detach.
  - E2E: intercom audio heard; multi-window camera tiles visibly update; play-audio command is audible.

### P0-5: Capability declaration is not truthful for camera/sensors/connectivity

- Why it fails (root cause):
  - Capability proto supports richer hardware declaration but current registration path omits key fields; telemetry values are synthetic constants.
- Fix options per client:
  - Implement per-platform capability probing and report only verifiable features.
- Required plan changes:
  - Add a “capability truthfulness” gate before placement/sensing scenarios.
- Required code/config changes:
  - Build capability registry for camera/sensors/connectivity fields.
  - Replace synthetic telemetry defaults with real sensor providers where present.
- Test coverage to add:
  - Unit: manifest generation by platform.
  - Integration: server placement filters camera-capable devices correctly.
  - E2E: “show all cameras” excludes devices without camera capability.

### P0-6: Edge observation/runtime phases depend on persistence/export features that are still stubs

- Why it fails (root cause):
  - Edge host/retention/artifact components exist as lightweight scaffolds and are not durable or fully wired for cross-client production behavior.
- Fix options per client:
  - Web: IndexedDB-backed retention/artifact storage.
  - Native: filesystem-backed buffers with retention quotas.
- Required plan changes:
  - Re-scope phase-6B status from “complete” to “scaffold complete / production pending”.
- Required code/config changes:
  - Implement durable buffers, artifact exporter implementations, and control-plane plumbing.
- Test coverage to add:
  - Unit: retention window eviction behavior.
  - Integration: artifact request/available roundtrip.
  - E2E: restart-survivability for recent observation queries.

### P1-1: macOS mic permissions are not configured for voice features

- Why it fails (root cause):
  - macOS runner entitlements/plist do not show microphone permission setup required by documented voice features.
- Fix options per client:
  - macOS: add required entitlement/plist key and runtime checks.
  - iOS/Android: when scaffolded, enforce equivalent permission flow.
- Required plan changes:
  - Add permission checklist as prerequisite for voice/comms milestones.
- Required code/config changes:
  - Update macOS entitlements and app metadata.
  - Add user-visible permission failure guidance.
- Test coverage to add:
  - Integration: permission denied/allowed behavior.
  - E2E: mic-dependent scenario start with permission prompts.

### P1-2: Continuous monitoring assumptions are not backed by background execution model

- Why it fails (root cause):
  - Monitoring loops are timer-based in app lifecycle; no explicit per-platform background policy is modeled.
- Fix options per client:
  - Define support tiers by client (foreground-only vs background-capable) and route scenarios accordingly.
- Required plan changes:
  - Encode monitoring support tier matrix in phase-5/6 docs.
- Required code/config changes:
  - Add lifecycle hooks and state-aware scheduling.
  - Prevent server from assigning unsupported background roles to incapable clients.
- Test coverage to add:
  - Unit: lifecycle transition handling.
  - Integration: scenario degrades gracefully when app backgrounds.
  - E2E: foreground/background monitoring correctness by client tier.

### P2-1: Notification/speech UX is degraded outside web

- Why it fails (root cause):
  - Notification path is mostly in-app text; speech helper is web-only export with non-web stub no-op.
- Fix options per client:
  - Implement native notification delivery and cross-platform speech strategy.
- Required plan changes:
  - Separate “status text update” from “user alert delivery” requirements.
- Required code/config changes:
  - Add notification plugin integration and non-web speech fallback/engine.
- Test coverage to add:
  - Unit: alert routing decision logic.
  - Integration: timer/reminder alert delivery semantics.
  - E2E: alert visible/audible in expected app states.

---

## C) Revised Implementation Plan (Valid for All Target Clients)

1. Add a **Client Support Contract** phase gate:
   - Either officially narrow target platforms, or scaffold Android/iOS/Linux/Windows immediately.
2. Implement and verify **real control transport baseline**:
   - Real server gRPC listener.
   - Browser-compatible gRPC-Web path.
3. Add **cross-client CI/platform gates**:
   - Build matrix and connect/register smoke tests per target.
4. Complete **media output parity**:
   - Real remote track rendering + audio playback behavior.
5. Enforce **capability truthfulness**:
   - Per-platform capability probe and verified manifests.
6. Define and implement **monitoring support tiers**:
   - Foreground/background policies by client.
7. Complete **edge persistence/export**:
   - Durable retention buffers and artifact lifecycle by platform.
8. Re-validate scenario milestones with per-client **integration + E2E**:
   - Intercom, all-cameras view, timer/reminder, audio monitor, edge observation query.

---

## D) No-Go List (Assumptions That Must Not Appear in Future Plans)

1. “Flutter single codebase implies capability parity.”
2. “Web can use raw gRPC sockets directly.”
3. “mDNS works uniformly across all clients and environments.”
4. “`Timer.periodic` is sufficient for background monitoring guarantees.”
5. “Synthetic capability/telemetry values are acceptable for placement decisions.”
6. “Placeholder UI primitives count as completed media implementation.”
7. “Unknown capability status is acceptable for release.”

---

## E) Confidence and Remaining Unknowns

- Confidence:
  - High for static code/doc mismatches.
  - Medium for runtime behavior not fully executable in this sandbox environment.
- Remaining unknowns (treated as blockers):
  1. Whether an external gRPC-Web proxy exists outside repo and is the intended web path.
  2. Whether a real gRPC listener exists in code paths not currently wired by startup.
  3. Runtime behavior of mDNS/WebRTC across all non-web/non-macOS targets after scaffold.
  4. Final platform policy constraints for background monitoring on each client.

---

## Evidence Index

- E2: [`scripts/run-local.sh`](../../scripts/run-local.sh#L31), [`scripts/run-local.sh`](../../scripts/run-local.sh#L237)
- E3: [`Makefile`](../../Makefile#L26), [`Makefile`](../../Makefile#L56)
- E4: [`terminal_client/lib/connection/control_client.dart`](../../terminal_client/lib/connection/control_client.dart#L23)
- E5: [`terminal_client/lib/main.dart`](../../terminal_client/lib/main.dart#L401), [`terminal_client/test/widget_test.dart`](../../terminal_client/test/widget_test.dart#L16)
- E6: [`terminal_server/internal/transport/grpc_server.go`](../../terminal_server/internal/transport/grpc_server.go#L41), [`terminal_server/internal/transport/grpc_server_test.go`](../../terminal_server/internal/transport/grpc_server_test.go#L12)
- E7: [`terminal_client/lib/discovery/mdns_scanner.dart`](../../terminal_client/lib/discovery/mdns_scanner.dart#L25)
- E8: [`docs/client-ios.md`](../../docs/client-ios.md#L112), [`docs/client-linux.md`](../../docs/client-linux.md#L81), [`docs/client-windows.md`](../../docs/client-windows.md#L79)
- E9: [`terminal_client/lib/main.dart`](../../terminal_client/lib/main.dart#L509), [`terminal_client/lib/main.dart`](../../terminal_client/lib/main.dart#L654)
- E10: [`terminal_client/lib/media/webrtc_engine.dart`](../../terminal_client/lib/media/webrtc_engine.dart#L74)
- E11: [`terminal_client/lib/connection/control_client.dart`](../../terminal_client/lib/connection/control_client.dart#L83), [`api/terminals/capabilities/v1/capabilities.proto`](../../api/terminals/capabilities/v1/capabilities.proto#L16)
- E12: [`terminal_client/lib/main.dart`](../../terminal_client/lib/main.dart#L2277), [`terminal_client/lib/main.dart`](../../terminal_client/lib/main.dart#L2720)
- E13: [`terminal_client/lib/main.dart`](../../terminal_client/lib/main.dart#L872)
- E14: [`terminal_client/macos/Runner/DebugProfile.entitlements`](../../terminal_client/macos/Runner/DebugProfile.entitlements#L5), [`terminal_client/macos/Runner/Release.entitlements`](../../terminal_client/macos/Runner/Release.entitlements#L5), [`terminal_client/macos/Runner/Info.plist`](../../terminal_client/macos/Runner/Info.plist#L4), [`docs/client-macos.md`](../../docs/client-macos.md#L70)
- E15: [`terminal_client/lib/main.dart`](../../terminal_client/lib/main.dart#L746), [`terminal_client/lib/util/speech.dart`](../../terminal_client/lib/util/speech.dart#L1)
- E16: [`terminal_client/lib/util/speech_stub.dart`](../../terminal_client/lib/util/speech_stub.dart#L1), [`terminal_client/lib/util/speech_web.dart`](../../terminal_client/lib/util/speech_web.dart#L3)
- E17: [`plans/phase-5-voice.md`](../phases/phase-5-voice.md#L15), [`plans/phase-6b-edge-sensing.md`](../phases/phase-6b-edge-sensing.md#L11), [`plans/phase-7-polish.md`](../phases/phase-7-polish.md#L16)
- E18: [`terminal_client/lib/main.dart`](../../terminal_client/lib/main.dart#L842), [`terminal_client/lib/main.dart`](../../terminal_client/lib/main.dart#L862)
- E19: [`terminal_client/lib/edge/host.dart`](../../terminal_client/lib/edge/host.dart#L5), [`terminal_client/lib/edge/bundle_store.dart`](../../terminal_client/lib/edge/bundle_store.dart#L3), [`terminal_client/lib/edge/artifact_export.dart`](../../terminal_client/lib/edge/artifact_export.dart#L4)
- E20: [`terminal_client/pubspec.yaml`](../../terminal_client/pubspec.yaml#L9)
- E21: [`plans/protocol.md`](../features/protocol.md#L63), [`terminal_client/lib/main.dart`](../../terminal_client/lib/main.dart#L2455), [`terminal_client/lib/main.dart`](../../terminal_client/lib/main.dart#L881)

---

PLAN NOT SAFE FOR ALL CLIENTS
