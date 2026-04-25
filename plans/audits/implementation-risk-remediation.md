---
title: "Implementation Risk Remediation Plan"
kind: plan
status: building
owner: cascade
validation: none
last-reviewed: 2026-04-25
---

# Implementation Risk Remediation Plan

Addresses the findings in [implementation-risk-audit.md](implementation-risk-audit.md). Executed strictly serially — each stage must meet its acceptance criteria before the next begins. Adds a dual-transport server (gRPC for native, WebSocket for web) so the browser client stops depending on raw gRPC sockets.

See [masterplan.md](../archive/masterplan-duplicate.md) for system context.

## Incremental Progress

- 2026-04-25: Removed synthetic pointer capability declarations derived from platform touch heuristics so `pointer.type`/`pointer.hover` are no longer advertised without explicit probe evidence, and added probe regression assertions that pointer remains omitted unless explicitly probed.
- 2026-04-25: Removed heuristic touch declarations from capability probing so `touch.supported` and `screen.touch` are no longer advertised from platform guesses alone, and added regression assertions that touch fields stay omitted unless explicit probe evidence is available.
- 2026-04-25: Expanded Stage 4 placement strict-filter regression coverage to camera, microphone, and speakers by asserting `RequiredCaps` excludes devices when those capability fields are missing or explicitly `false`, preserving capability-truthful targeting across media routes.
- 2026-04-25: Removed synthetic default audio channel counts from client capability probing (`microphone.channels`, `speakers.channels`, and endpoint channel fields) so channel capacity is no longer advertised without verified measurement, and added probe regression assertions that those values remain unset/default unless explicitly probed.
- 2026-04-25: Removed synthetic probe-only defaults for `screen.safe_area` zero insets and `touch.max_points=1` so capability probes now omit those fields unless runtime metadata can provide real values, and added probe regression assertions while preserving runtime snapshot safe-area coverage through display metadata wiring.
- 2026-04-25: Removed synthetic default edge-retention capability declaration from the client capability probe so retention windows are no longer advertised without verified platform-backed evidence, and added probe unit assertions that edge retention remains omitted while real runtime/operators metadata is still emitted.
- 2026-04-25: Hardened Stage 2 websocket origin policy to enforce explicit allow-list entries only (rejecting wildcard `*`), added config and transport regression coverage, and documented dev-time same-origin/loopback behavior for `TERMINALS_CONTROL_WS_ALLOWED_ORIGINS`.
- 2026-04-25: Added deterministic video-surface stream state (`Waiting for media` / `Attached`) backed by the media engine attach signal so start/stop transitions are directly observable in widget tests.
- 2026-04-25: Split explicit user-alert delivery from in-app status text handling so `ConnectResponse.notification` now routes through a dedicated alert callback while `command_result.notification` remains status-only; added deterministic widget coverage for both paths.
- 2026-04-25: Hardened Stage 5 playback handling with deterministic unit coverage for `PlayAudio` source routing (`url`, `pcm_data`, `tts_text`), including WAV wrapping for raw PCM and trimmed TTS dispatch behavior.
- 2026-04-25: Added deterministic runtime permission-denied handling coverage for local media starts by injecting media permission probes in the client and asserting denied starts set a stable status/notification without emitting local offers.
- 2026-04-25: Replaced synthetic zero-valued screen safe-area capability metadata with real view/injected metrics in capability snapshots and deltas, and added deterministic widget coverage proving safe-area changes trigger `display_geometry_change` capability updates.
- 2026-04-25: Added server placement regression coverage to lock Stage 4 behavior that missing or explicitly disabled capability fields are treated as unsupported when evaluating `RequiredCaps` filters.
- 2026-04-25: Removed synthetic default connectivity declaration from client capability probing, and gated outbound sensor telemetry to emit only declared capability-backed signals (connectivity/battery), with widget coverage proving time-derived synthetic keys are no longer sent.
- 2026-04-25: Removed synthetic default `screen.fullscreen_supported` and `screen.multi_window_supported` declarations from client capability probing and runtime display metadata updates so unverified support is omitted, with probe/widget regression assertions locking those fields absent unless explicitly probed.

## Guiding Decisions

1. **Dual transport**: Go server exposes both a real gRPC listener and a WebSocket endpoint on the same process. Both carry the same protobuf `ConnectRequest`/`ConnectResponse` frames through a shared session handler. Native clients prefer gRPC; web clients use WebSocket. No external proxy.
2. **All six platforms scaffolded**: Android, iOS, Linux, Windows are added as real Flutter platform targets in this plan — not deferred. CI builds a matrix across all six.
3. **Serial stages**: no parallel tracks. Each stage lands and is green in CI before the next starts.
4. **No-Go guardrails** (from audit §D) are enforced throughout: no raw gRPC in browsers, no synthetic capabilities, no placeholder media counting as "done", no `Timer.periodic` as a background monitoring guarantee.

---

## Stage 1 — Real gRPC listener (P0-3)

Replace the lifecycle-only `Server.Start` with a real network listener and registered gRPC service.

**Scope**
- Implement `grpc.NewServer`, bind via `net.Listen("tcp", addr)`, register the generated `TerminalControlService`, serve in a goroutine, and drain via `GracefulStop` on `Stop`.
- Bridge generated `Control_ConnectServer` into existing `ProtoStream` abstraction via a thin adapter (keeps `RunProtoSession` untouched).
- Emit `transport.grpc.listener_ready` only after `net.Listen` succeeds and a socket is actually bound.

**Files**
- `terminal_server/internal/transport/grpc_server.go` — real listener lifecycle.
- `terminal_server/internal/transport/grpc_service.go` (new) — generated-service adapter.
- `terminal_server/cmd/...` startup wiring.

**Acceptance**
- Unit: `Server.Running()` is false until `Listen` returns; `Stop` unblocks `Serve`.
- Integration: generated Go gRPC client connects to the bound port, completes `Register` → `RegisterAck` → `Heartbeat` round trip.
- E2E: `scripts/run-local.sh` launches a server and a native client reaches registered state without mock transport.

---

## Stage 2 — WebSocket transport alongside gRPC (P0-2)

Add a WebSocket endpoint that speaks the same control-plane protocol.

**Scope**
- New HTTP server on a separate port (or same port, different path — decide during design; default: separate port for cleanliness) exposing `GET /control` with a WebSocket upgrade.
- Each binary message is a single protobuf-encoded `ConnectRequest` (client→server) or `ConnectResponse` (server→client). No JSON, no length prefix beyond the WebSocket frame.
- Implement `WebSocketProtoStream` that satisfies the same `ProtoStream` interface as the gRPC adapter, so `Server.Connect` handles both identically.
- Browser client: a new `TerminalControlWebSocketClient` implementing `TerminalControlClient`. `main.dart` picks transport via `kIsWeb`.
- CORS: explicit allow-list, same-origin preferred; document dev-time behavior.

**Files**
- `terminal_server/internal/transport/websocket_server.go` (new).
- `terminal_server/internal/transport/websocket_stream.go` (new) — `ProtoStream` adapter.
- `terminal_client/lib/connection/control_client_ws.dart` (new).
- `terminal_client/lib/connection/control_client.dart` — factory selecting gRPC vs WebSocket.

**Acceptance**
- Unit: WebSocket stream adapter round-trips arbitrary `ConnectRequest` bytes through `RunProtoSession`.
- Integration: headless web client connects, registers, and receives `RegisterAck` over WebSocket.
- E2E: web app in Chromium completes connect → register → heartbeat → reconnect.

---

## Stage 3 — Scaffold all six platform targets (P0-1)

Bring Android, iOS, Linux, Windows up to the same minimum bar as Web and macOS.

**Scope**
- `cd terminal_client && flutter create --platforms=android,ios,linux,windows .` (preserving existing web/macos trees; verify no overwrite).
- Commit all generated platform directories.
- Permissions placeholders for mic/camera on Android (`AndroidManifest.xml`), iOS (`Info.plist` `NSMicrophoneUsageDescription`, `NSCameraUsageDescription`), Linux/Windows (document limits).
- Makefile: `client-build-{android,ios,linux,windows,web,macos}` and `client-build-all`.
- CI workflow: build matrix across all six targets. Smoke test per target is `flutter build` only in Stage 3; runtime smoke tests come in Stage 4+.
- `scripts/run-local.sh` gains `--platform` flag.

**Files**
- `terminal_client/{android,ios,linux,windows}/**` (generated).
- `Makefile`, `.github/workflows/*`, `scripts/run-local.sh`.

**Acceptance**
- `make client-build-all` succeeds locally on macOS (iOS build may require Xcode; gate on availability).
- CI matrix goes green for all six `flutter build` targets.
- New platform-readiness test asserts all platform directories exist and have expected permission entries.

---

## Stage 4 — Capability truthfulness (P0-5)

Replace synthetic capability/telemetry values with verified per-platform probes.

**Scope**
- Introduce `CapabilityProbe` per platform that inspects real OS state (camera present, mic channels, screen size, sensor availability, edge compute hints).
- Registration pipeline declares only probed capabilities; omit fields rather than lie.
- Remove synthetic sensor-value constants; gate sensor telemetry behind presence checks.
- Server placement filters updated to treat absent fields as "not supported" rather than defaulting.

**Files**
- `terminal_client/lib/capabilities/` (new) — per-platform probes behind a shared interface.
- Wire probes through `TerminalControlGrpcClient.registerRequest` call sites.
- `terminal_server/internal/placement/...` — strict filtering.

**Acceptance**
- Unit: per-platform probe emits only fields backed by real queries.
- Integration: server's "all cameras" placement excludes devices that did not declare camera capability.
- E2E: on a macOS host with no camera, camera-dependent scenario does not target it.

---

## Stage 5 — Real media rendering and playback (P0-4)

Replace `video_surface`, `audio_visualizer`, and `PlayAudio` placeholders with real output.

**Scope**
- Bind `flutter_webrtc` remote tracks to `RTCVideoView` / audio sinks in UI primitives (not placeholder widgets).
- Implement `PlayAudio` for `url` (HTTP stream), `pcm_data` (inline), and `tts_text` (platform TTS where available; fallback queued for Stage 9).
- `StartStream` / `StopStream` must attach and detach real renderers.

**Files**
- `terminal_client/lib/main.dart` (video_surface/audio_visualizer sites).
- `terminal_client/lib/media/webrtc_engine.dart`.
- New `terminal_client/lib/media/playback.dart`.

**Acceptance**
- Unit: widget primitive renders the bound track id, not a placeholder string.
- Integration: start/stop stream produces attach/detach events observable in a widget test.
- E2E: intercom audio is audible across a two-client local run; multi-window camera tiles show live frames; `PlayAudio` url path is audible.

---

## Stage 6 — Edge persistence and artifact export (P0-6)

Make edge retention/artifact stores durable, per platform.

**Scope**
- Web: IndexedDB-backed retention buffer and artifact store.
- Native: filesystem-backed buffers under app-support dir, with quota enforcement.
- Wire control-plane artifact request/available round trip through existing edge plumbing.
- Phase-6B status in `plans/phase-6b-edge-sensing.md` re-marked "scaffold complete / production pending" until this stage is done.

**Files**
- `terminal_client/lib/edge/host.dart`, `bundle_store.dart`, `artifact_export.dart` and new per-platform implementations.

**Acceptance**
- Unit: retention-window eviction behavior verified with synthetic clock.
- Integration: artifact request/available round trip.
- E2E: restart survivability — kill the client, relaunch, recent observation query returns results.

---

## Stage 7 — Permissions and macOS mic entitlement (P1-1)

**Scope**
- macOS: add `NSMicrophoneUsageDescription` to `Info.plist`, `com.apple.security.device.audio-input` to entitlements.
- iOS: mic + camera usage strings; runtime permission gate before stream start.
- Android: runtime permissions with rationale; declare in manifest.
- Linux/Windows: document limits and any portal/DirectShow prerequisites.
- User-visible failure path when permission denied (not a silent hang).

**Acceptance**
- Integration: permission-denied path produces a deterministic error event, not a stuck stream.
- E2E: first mic-dependent scenario triggers the OS prompt.

---

## Stage 8 — Monitoring support tiers (P1-2)

**Scope**
- Define support tier matrix (foreground-only vs background-capable) and embed in capability declaration.
- Replace `Timer.periodic`-only assumptions with lifecycle-aware scheduling (WorkManager on Android, BGTasks on iOS where applicable; document web and desktop limits).
- Server placement refuses to assign background roles to foreground-only clients.

**Acceptance**
- Unit: lifecycle transitions degrade scenarios rather than silently continuing.
- Integration: server assignment respects tier.
- E2E: foregrounding/backgrounding behaves correctly per tier on at least macOS + web.

---

## Stage 9 — Notification and speech parity (P2-1)

**Scope**
- Native notification delivery via `flutter_local_notifications` (or platform equivalent).
- Non-web speech engine path: `flutter_tts` or platform-native; keep the web implementation via SpeechSynthesis.
- Separate "status text update" (in-app) from "user alert delivery" (notification) in server-driven UI descriptors where needed.

**Acceptance**
- Unit: alert routing picks notification vs in-app based on descriptor.
- Integration: timer/reminder alerts fire via notification on native, web notifications on web.
- E2E: alert is visible or audible in the expected app states per platform.

---

## Cross-cutting CI / Gate Changes

Added once, enforced from Stage 3 onward:

- `flutter build` matrix for all six targets.
- Go `go test ./...` plus a new integration suite that boots both gRPC and WebSocket endpoints.
- Headless browser test (Chromium) exercising the WebSocket connect path.
- `all-check` Make target includes the new matrix.
- Phase completion gate: no phase is marked complete until (a) transport baseline is green, (b) capability manifests are verified, (c) real output is observable for any media the phase claims.

## Open Items Deferred Beyond This Plan

- TLS mutual auth on either transport (non-goal per masterplan §10).
- External (non-LAN) reachability.
- gRPC-Web as a fallback path — not pursued; WebSocket covers the browser need.
