---
title: "Protocol — Progress Log"
kind: progress-log
parent: plans/features/protocol/plan.md
---

## Implementation Progress (2026-04-26)

- Added explicit capability invalidation payloads to control-plane acknowledgements in `CapabilityAck.invalidations` (`api/terminals/control/v1/control.proto`).
- Wired server transport capability ack generation to include deterministic lost-resource invalidations (resource + reason) when snapshots/deltas remove claimable resources.
- Added/updated transport regression coverage for ack invalidation content and proto adapter mapping.
- Updated durable connection docs to describe `capability_ack` invalidation behavior.
- Removed client bootstrap emission of deprecated `RegisterDevice` requests; client bootstrap now sends `hello` + `capability_snapshot` and retries snapshot delivery until acknowledgement instead of retrying register payloads.
- Removed generated-proto ingest support for deprecated `CapabilityUpdate` client payloads; generated clients must use `capability_snapshot` / `capability_delta`.
- Normalized legacy generated `register` payload ingest through capability-snapshot handling while preserving compatibility (`register_ack` remains emitted for bootstrap clients).
- Added snapshot bootstrap fallback for unknown devices: capability snapshots now synthesize identity registration when needed before applying generation-ordered capability state.
- Preserved relay registration semantics for snapshot-first sessions so cross-session route/notification fan-out behavior remains stable.
- Updated transport carrier and websocket integration tests to accept capability lifecycle bootstrap ordering (`capability_ack` may precede `register_ack`).
- Re-ran repository validation gates (`make all-check`) and promoted this plan to shipped-validated status.

Any future compatibility-window cleanup (for example fully removing deprecated proto request fields) should be tracked as a separate follow-on task, not under this completed protocol design plan.

## Protocol Evolution Rules (2026-05-03)

- Added the protocol evolution policy, extension registry, compatibility notes, and API contract checklist.
- Inventoried current flexible protocol fields across control, UI, IO, capabilities, and diagnostics protos.
- Added an advisory `proto-flex-check` static guardrail and initial `proto-contract-test` Make target.
- Added PR checklist language for protocol-affecting changes.
- Added shared `WireEnvelope` golden fixtures under `api/testdata/envelopes/` for hello, capability snapshot, register ack metadata, set UI, start stream, flow plan, observation, unknown metadata, and deprecated register payloads.
- Added Go and Dart protocol contract checks that decode the same binary fixtures and assert flexible-field compatibility behavior.
- Expanded `proto-contract-test` to run proto lint, flex-field registry validation, Go fixture decoding, protocol-focused transport tests, and the Dart fixture decoder.

## Protocol Evolution Rules (2026-05-03, Build Metadata Typing)

- Added additive `BuildMetadata` and `ServerMetadata` messages to `api/terminals/control/v1/control.proto` and wired `RegisterAck.server_metadata` as typed metadata alongside legacy `metadata`.
- Updated server transport register responses (`ControlService` + generated proto adapter) to emit typed build/photo-frame metadata while preserving compatibility map keys.
- Updated Flutter register-ack parsing to prefer typed `server_metadata.build` values and fall back to legacy `metadata` keys.
- Extended Go/Dart protocol contract assertions and `register_ack_metadata_v1` fixture content to validate both typed and legacy metadata behavior.
- Ran `make proto-generate`, refreshed binary envelope fixtures via `go test ./internal/protocolcontract -run TestGoldenWireEnvelopeFixtures -update`, and re-ran Go transport/protocol-contract tests.

## Protocol Evolution Rules (2026-05-03, FlowState Enum Migration)

- Added additive typed enum `terminals.io.v1.FlowState` and `FlowStats.state_enum` while preserving the legacy `state` string.
- Updated the generated proto adapter to populate `FlowStatsRequest.StateEnum` from inbound payloads.
- Reworked the server flow stats handler to log a resolved state derived enum-first with legacy string fallback.
- Added a `flow_stats_v1` golden envelope fixture and registered it in both Go (`internal/protocolcontract`) and Dart (`test/protocol_contract_test.dart`) contract tests.
- Updated the protocol extension registry entry for `FlowStats.state` to describe typed-first compatibility behavior.
- Synced regenerated `terminals/io/v1` Dart bindings into `terminal_client/lib/gen/`; refreshed Go generated bindings via `make proto-generate`.
- Realigned `WebRTCSignalType_*` Go enum references in `generated_proto_adapter_test.go` to the regenerated `WEB_RTC_SIGNAL_TYPE_*` names so the transport test suite builds.
- Re-ran `make proto-contract-test` and the full server `go test ./...` suite.

## Protocol Evolution Rules (2026-05-03, ScrollDirection Enum Migration)

- Added additive typed enum `terminals.ui.v1.ScrollDirection` and `ScrollWidget.direction_enum` while preserving the legacy `direction` string.
- Updated the generated proto adapter (`applyWidgetFromDescriptor`) to populate both typed enum and legacy string from `props["direction"]`.
- Updated the Flutter server-driven renderer to prefer `direction_enum` for axis selection and fall back to the legacy string when the enum is unspecified.
- Updated the protocol extension registry entry for `ScrollWidget.direction` to describe typed-first compatibility behavior.
- Reverted brittle Dart `switch` expressions over `ProtobufEnum` constants in `control_response_dispatcher` and `webrtc_engine` to defensive `if`/`==` chains so the legacy fallback survives unknown enum values from older generators/clients.
- Split the `typed enum fields override legacy labels` dispatcher test into per-payload responses (start/route/signal share a `ConnectResponse.payload` oneof, so they cannot be exercised in a single response).
- Re-ran `make proto-contract-test`, `make server-test`, and `make client-test`; all green.

## Protocol Evolution Rules (2026-05-04, StartStream audio metadata typing)

- Added additive typed `terminals.io.v1.StreamAudioMetadata` and `StartStream.audio_metadata = 8` in `api/terminals/io/v1/io.proto`, preserving legacy `StartStream.metadata` keys `sample_rate`/`channels`/`codec` during the compatibility window.
- Updated transport proto adapter (`generated_proto_adapter.go`) to emit typed `audio_metadata` and legacy metadata keys together: producers use explicit typed metadata when present, otherwise derive typed values from legacy map keys; legacy map keys are mirrored from typed fields for consistency.
- Updated server media stream registration (`media_control_state.go`) to prefer typed `AudioMetadata` and fall back to legacy metadata keys, then normalize stored/recorded metadata to the typed values during migration.
- Added transport regression coverage for typed-first + fallback behavior in `generated_proto_adapter_test.go` and media-control metadata normalization coverage in `media_control_state_test.go`.
- Extended shared Go/Dart protocol contract assertions and `start_stream_audio_v1` fixture content to validate typed `audio_metadata` plus legacy metadata coexistence.
- Updated `docs/protocol-extension-registry.md` and `docs/compatibility.md` to track the new compatibility window (`StartStream.audio_metadata` -> legacy metadata keys) and refreshed pending-migration notes for `StartStream.metadata` cleanup.

## Protocol Evolution Rules (2026-05-04, Proto CI contract gate + flex-check annotations)

- Updated `.github/workflows/proto-ci.yml` to run the full repository-level `make proto-contract-test` gate (in addition to existing Buf format/lint/generate/breaking checks), and expanded workflow path triggers to include protocol-relevant server/client/docs/script/Makefile changes.
- Added CI toolchain setup in Proto CI (`actions/setup-go` + `subosito/flutter-action`) so `make proto-contract-test` can execute Go + Dart protocol contract suites in workflow runs.
- Kept Buf steps in `api/` via per-step `working-directory`, then executed the contract gate from repo root to match local Makefile behavior.
- Enhanced `scripts/check-proto-flex-fields.py` to emit GitHub Actions `::error` annotations for each missing registry entry when running in CI, making `proto-flex-check` failures directly clickable/actionable in PRs.
- Re-ran `make proto-flex-check` and `make proto-contract-test`; both passed.

## Protocol Evolution Rules (2026-05-03, StreamKind/WebRTCSignalType Enum Migration)

- Added additive typed enums `StreamKind` and `WebRTCSignalType` in `api/terminals/io/v1/io.proto` and `api/terminals/control/v1/control.proto` with new `stream_kind` and `signal_type_enum` fields while preserving legacy string fields.
- Updated generated transport adapter mappings to emit typed+legacy values and to resolve inbound WebRTC signal type from enum first with legacy-string fallback.
- Updated Flutter control/media handling to prefer typed stream/signal enums for notifications and runtime media behavior, while retaining legacy fallback.
- Extended Go adapter tests and Go/Dart protocol contract assertions to validate typed enum emission and compatibility fallback behavior.
- Updated `start_stream_audio_v1` text/bin fixtures to include typed `stream_kind`, and refreshed protocol registry migration notes for typed-first semantics.

## Protocol Evolution Rules (2026-05-03, ExecPolicy Enum Migration)

- Added additive typed enum `terminals.io.v1.ExecPolicy` and `FlowNode.exec_policy` (field 5) while preserving the legacy `exec` string.
- Updated `flowPlanToProto` in `terminal_server/internal/transport/generated_proto_adapter.go` to emit both the typed `exec_policy` enum and the legacy `exec` string from internal `iorouter.ExecPolicy` values via a new `protoExecPolicyFromInternal` helper (covering `auto`, `prefer_client`, `require_client`, `server_only`).
- Extended the `flow_plan_basic_v1` golden envelope fixture (textproto + binpb) to include typed `exec_policy` values alongside legacy `exec` strings; refreshed binary fixture via `go test ./internal/protocolcontract -run TestGoldenWireEnvelopeFixtures -update`.
- Updated Go and Dart contract assertions to verify typed-enum + legacy-string compatibility for both flow plan nodes.
- Updated the protocol extension registry entry for `FlowNode.exec` to describe typed-first compatibility semantics.
- Synced regenerated `terminals/io/v1` Dart bindings into `terminal_client/lib/gen/`.
- Re-ran `make proto-contract-test` and full `go test ./...` in `terminal_server`; all green.

## Protocol Evolution Rules (2026-05-04, CanvasWidget Typed DrawOps)

- Added additive typed drawing primitives to `api/terminals/ui/v1/ui.proto`: `DrawLine`, `DrawRect`, `DrawCircle`, `DrawText`, `DrawPath`, and `DrawOp` (oneof of those primitives).
- Added `repeated DrawOp draw_ops = 2` to `CanvasWidget` while preserving the legacy `string draw_ops_json = 1` for the compatibility window.
- Regenerated Go bindings via `make proto-generate` and synced refreshed `terminals/ui/v1` Dart bindings (`ui.pb.dart`, `ui.pbenum.dart`, `ui.pbjson.dart`) into `terminal_client/lib/gen/`.
- Updated the protocol extension registry entry for `CanvasWidget.draw_ops_json` to describe typed-first compatibility semantics and the deferred-wiring posture.
- No application code currently produces or consumes canvas draw ops, so adapter wiring is deferred until the first real consumer lands; the typed schema is now available.
- Re-ran `make proto-contract-test` and `make server-test`; all green.

## Protocol Evolution Rules (2026-05-04, PointerAction/TouchAction Enum Migration)

- Added additive typed enums `terminals.io.v1.PointerAction` and `terminals.io.v1.TouchAction` and the corresponding `PointerEvent.action_enum` (field 7) and `TouchEvent.action_enum` (field 3) while preserving legacy `action` string fields.
- Regenerated Go bindings via `make proto-generate` and synced the refreshed `terminals/io/v1` Dart bindings (`io.pb.dart`, `io.pbenum.dart`, `io.pbjson.dart`) into `terminal_client/lib/gen/`.
- Updated the protocol extension registry entries for `PointerEvent.action` and `TouchEvent.action` to describe typed-first compatibility semantics (typed enum preferred, legacy string fallback during the migration window).
- No application-code paths currently route pointer/touch input, so adapter wiring is deferred until a producer/consumer lands; the typed fields are now available for the first non-test consumer.
- Re-ran `make proto-contract-test` and `make server-test`; all green.

## Protocol Evolution Rules (2026-05-04, Diagnostics StreamEntry/RouteEntry typed mirror)

- Added additive `terminals.io.v1.StreamKind stream_kind = 5` to both `StreamEntry` and `RouteEntry` in `api/terminals/diagnostics/v1/diagnostics.proto`, mirroring the typed enum from the underlying `StartStream`/`RouteStream` while preserving the legacy `kind` strings.
- Wired client diagnostics capture (`terminal_client_shell.dart`) to populate `streamKind` on `StreamEntry` / `RouteEntry` from `start.streamKind` / `route.streamKind` whenever the source is non-unspecified, alongside the legacy string `kind`.
- Skipped the analogous `WebrtcSignalEntry.signal_type` typed mirror: `control.proto` already imports `diagnostics.proto`, so a reverse import to use `WebRTCSignalType` would create an import cycle. Documented the deferral in `diagnostics.proto`, the registry, and `docs/compatibility.md` so the migration can be revisited once `WebRTCSignalType` moves to a shared package.
- Regenerated Go bindings via `make proto-generate` and synced refreshed `terminals/diagnostics/v1` Dart bindings into `terminal_client/lib/gen/`.
- Updated registry entries for `StreamEntry.kind` and `RouteEntry.kind` to describe typed-first compatibility behavior; added a row to `docs/compatibility.md`'s open-windows table.
- Re-ran `make proto-contract-test` (lint + flex-check + Go contract + Dart contract) and `make client-test`; all green.

## Protocol Evolution Rules (2026-05-04, Phase 5 enforcement + compatibility windows)

- Flipped `proto-flex-check` from advisory to required by passing `--enforce` in the Makefile target. The registry now covers all 31 detected flexible fields, so missing-entry detections fail the gate.
- Updated `docs/compatibility.md` to enumerate the typed-replacement migration windows currently open (RegisterAck typed metadata, StreamKind, WebRTCSignalType, ScrollDirection, FlowState, ExecPolicy, CanvasWidget DrawOps, PointerAction, TouchAction) with shipped dates and earliest legacy-removal criteria, and refreshed the pending-migrations summary against the current registry.
- Re-ran `make proto-flex-check` (now in enforce mode) and `make proto-contract-test`; all green.

## Protocol Evolution Rules (2026-05-04, UiEventEntry typed kind enum)

- Added additive typed enum `terminals.diagnostics.v1.UiEventKind` (`UNSPECIFIED`/`SET_UI`/`UPDATE_UI`/`TRANSITION_UI`) and `UiEventEntry.kind_enum = 5` while preserving the legacy `kind` string in `api/terminals/diagnostics/v1/diagnostics.proto`.
- Updated `serverDrivenUiUpdateFromResponse` in `terminal_client/lib/connection/control_response_dispatcher.dart` to emit `kindEnum` alongside the legacy `kind` string for `set_ui`/`update_ui`/`transition_ui` events.
- Updated `_recordUiEvent` in `terminal_client/lib/app/terminal_client_shell.dart` to copy the typed enum onto each `UiEventEntry` when present, leaving the legacy string in place as a fallback.
- Reclassified the registry entry for `UiEventEntry.kind` from `registry_backed_extension` to `transitional_escape_hatch` describing typed-first compatibility semantics.
- Extended dispatcher tests in `terminal_client/test/connection/control_response_dispatcher_test.dart` to assert the typed enum mirrors the legacy string per UI response payload.
- Regenerated Go bindings via `make proto-generate`, synced refreshed `terminals/diagnostics/v1` Dart bindings into `terminal_client/lib/gen/`, and re-ran `make proto-contract-test`, `make server-test`, and `make client-test`; all green.

## Protocol Evolution Rules (2026-05-04, WebrtcSignalEntry typed mirror via parallel io/v1 enum)

- Resolved the deferred `WebrtcSignalEntry.signal_type_enum` typed mirror by adding a parallel `terminals.io.v1.WebRTCSignalType` enum (numerically aligned with `terminals.control.v1.WebRTCSignalType`). Diagnostics now references the io/v1 copy, breaking the would-be import cycle (control/v1 already imports diagnostics/v1).
- Did not move `WebRTCSignalType` out of `control/v1`; that would change the FQN of `WebRTCSignal.signal_type_enum`'s field type and trip `make proto-breaking`. Documented the parallel-enum approach in `io.proto`, `diagnostics.proto`, and `docs/compatibility.md`; consolidation onto a single shared package is deferred until a buf-breaking-friendly path is available.
- Added `WebrtcSignalEntry.signal_type_enum = 4` referencing `terminals.io.v1.WebRTCSignalType` while preserving the legacy `signal_type` string for compatibility.
- Wired client diagnostics capture in `terminal_client/lib/app/terminal_client_shell.dart` to populate `signalTypeEnum` from `WebRTCSignal.signalTypeEnum.value` (cross-package enum lookup via `iov1.WebRTCSignalType.valueOf`), alongside the legacy `signalType` string.
- Regenerated Go bindings via `make proto-generate` and synced refreshed `terminals/io/v1` and `terminals/diagnostics/v1` Dart bindings into `terminal_client/lib/gen/`.
- Updated `docs/compatibility.md` to add a new open-window row for `WebrtcSignalEntry.signal_type_enum` and replace the prior deferral note with the parallel-enum rationale.
- Re-ran `make proto-lint`, `make proto-breaking`, `make proto-flex-check`, `make proto-contract-test`, `make server-test`, and `make client-test`; all green.

## Protocol Evolution Rules (2026-05-04, StartStream/RouteStream typed routing)

- Added additive typed enums `terminals.io.v1.StreamOrigin` (`UNSPECIFIED`/`ROUTE_DELTA`/`RESTORE`) and `terminals.io.v1.WebRTCMode` (`UNSPECIFIED`/`SERVER_MANAGED`/`PEER_MANAGED`), and a `StreamRouting` message wrapping both, in `api/terminals/io/v1/io.proto`.
- Added `StreamRouting routing = 7` to `StartStream` and `StreamRouting routing = 6` to `RouteStream` (both additive); legacy `StartStream.metadata` map keys `origin` and `webrtc_mode` remain populated during the compatibility window.
- Wired the live route-delta producer in `terminal_server/internal/transport/control_stream.go` and the captured-route replay producer in `route_replay.go` to emit both typed `Routing` and the legacy metadata map. Added a new `stream_routing.go` helper file with `routeDeltaStreamRouting`, `streamRoutingFromMetadata`, and enum<->string converters.
- Updated `generated_proto_adapter.go` to populate `StartStream.routing` (preferring the producer-supplied `Routing` and falling back to deriving from the legacy metadata map) and `RouteStream.routing` on outbound `ConnectResponse` payloads.
- Migrated the only real consumer (`MediaControlState.ServerManagedSignalEngine` in `terminal_server/internal/transport/media_control_state.go`) to prefer typed `routing.webrtc_mode` and fall back to the legacy `webrtc_mode` map key. Added a `RoutingWebRTCMode` field to `mediaStreamState` populated in `RegisterStream`.
- Added two new golden envelope fixtures (`api/testdata/envelopes/start_stream_route_delta_v1.{textproto,binpb}` and `route_stream_route_delta_v1.{textproto,binpb}`) covering typed-enum + legacy-map coexistence; refreshed binary fixtures via `go test ./internal/protocolcontract -run TestGoldenWireEnvelopeFixtures -update`. Added matching assertions in the Go and Dart contract tests.
- Updated the protocol extension registry entry for `StartStream.metadata` to reclassify the stable `origin`/`webrtc_mode` keys as legacy mirrors of the typed `routing` field; the residual codec/media keys remain documented extension namespace.
- Added an open-window row to `docs/compatibility.md` for `StartStream.routing` / `RouteStream.routing` and updated the pending-migrations summary.
- Regenerated Go bindings via `make proto-generate` and synced the refreshed `terminals/io/v1` Dart bindings into `terminal_client/lib/gen/`.
- Re-ran `make proto-contract-test`, `make server-test`, and `make client-test`; all green.

## Protocol Evolution Rules (2026-05-04, Observation/FlowNode typed mirrors + ScrollWidget legacy deprecation + CanvasWidget consumer wiring)

- Added additive typed `terminals.io.v1.ObservationAttributes` (`label`, `device`, `mac`, `duration_seconds`) and `Observation.typed_attributes = 12` while preserving the legacy `attributes` map; updated `observationFromProto` in `terminal_server/internal/transport/generated_proto_adapter.go` to merge typed mirror over the legacy map with typed-first preference. Added `observationTypedAttributesFromInternal` helper for future outbound typed emission.
- Added additive typed `terminals.io.v1.FlowNodeArgs` (`device_id`, `resource`, `stream_kind` + typed `stream_kind_enum`, `name`) and `FlowNode.typed_args = 6` while preserving the legacy `args` map; flow planner now emits both via the new `flowNodeTypedArgsFromArgs` helper in the generated proto adapter.
- Marked `terminals.ui.v1.ScrollWidget.direction` as `[deprecated = true]` in `api/terminals/ui/v1/ui.proto` while keeping the legacy mirror populated by producers during the compatibility window.
- Wired the Flutter `ServerDrivenRenderer` `Node_Widget.canvas` case to consume typed `draw_ops` first (summarized via a per-`oneof` arm primitive count) and fall back to the legacy `draw_ops_json` preview only when typed ops are empty.
- Extended `flow_plan_basic_v1` and `observation_sound_v1` golden envelope fixtures (textproto + binpb) to populate typed mirrors alongside legacy map keys; refreshed binary fixtures via `go test ./internal/protocolcontract -run TestGoldenWireEnvelopeFixtures -update`. Extended Go (`fixtures_test.go`) and Dart (`protocol_contract_test.dart`) contract assertions to validate typed-vs-legacy coexistence for both fixtures.
- Updated registry entries for `Observation.attributes`, `FlowNode.args`, `ScrollWidget.direction`, and `CanvasWidget.draw_ops_json` to describe typed-first compatibility semantics; added compatibility windows to `docs/compatibility.md` for the new typed mirrors and the canvas consumer wiring; refreshed the pending-migrations summary.
- Regenerated Go bindings via `make proto-generate` and synced refreshed `terminals/io/v1` and `terminals/ui/v1` Dart bindings into `terminal_client/lib/gen/`.
- Re-ran `make proto-flex-check` (still 31/31 registered, enforce mode), `make proto-lint`, `make proto-breaking`, `make proto-contract-test`, `make client-test`, and `go test ./...` in `terminal_server`. All targets pass except a pre-existing failure in `TestGeneratedProtoAdapterFromInternal` (`typed_data[a].int64_value`) that reproduces on `main` with my changes stashed and is unrelated to this work.

## Protocol Evolution Rules (2026-05-04, fix CommandResult.typed_data int64 detection)

- Fixed pre-existing `TestGeneratedProtoAdapterFromInternal` failure in `terminal_server/internal/transport`: `commandTypedValueFromLegacyString` was calling `strconv.ParseBool` before `strconv.ParseInt`, so the integer string `"1"` was being typed as `bool_value=true` instead of `int64_value=1` (since `ParseBool` accepts `"1"`/`"0"`/`"t"`/`"f"`). Reordered the checks to try `ParseInt` first and restricted bool detection to the literal strings `true`/`false` (case-insensitive) so numeric strings can no longer collide with bool typing.
- Verified: `go test ./...` in `terminal_server` is now fully green; `make proto-flex-check`, `make proto-lint`, `make proto-contract-test` all pass. No fixture or registry changes were needed (legacy `data` map continues to mirror the typed value via `commandTypedValueToString`).

## Protocol Evolution Rules (2026-05-04, Observation typed_attributes outbound producer status)

- Investigated wiring `observationTypedAttributesFromInternal` into an outbound `*iov1.Observation` producer per the next-priority item. Confirmed there is no production outbound producer today: `ObservationMessage` is only a `ConnectRequest` payload (client→server) — `ConnectResponse` has no observation payload, and no non-test code in `terminal_server/` constructs `iov1.Observation`. The server's only role is the inbound consumer (`observationFromProto` → `iorouter.Observation` → observation sink), and `handleObservationMessage` writes to the sink without re-emitting anything outbound.
- Updated the registry entry for `Observation.attributes` (`docs/protocol-extension-registry.md`) to record that the outbound producer wiring is deferred until a real producer lands; `observationTypedAttributesFromInternal` is staged for that first producer.
- Updated the open-window row for `Observation.typed_attributes` in `docs/compatibility.md` to reflect that legacy-removal also requires the first outbound producer to ship with the typed mirror populated (in addition to the two-release clock past 2026-05-04).
- Added unit-test coverage for `observationTypedAttributesFromInternal` in `terminal_server/internal/transport/generated_proto_adapter_test.go` (known typed keys populated; whitespace trimming; nil/empty/only-unknown input returns nil; blank typed values return nil) so the staged helper is exercised today and won't bit-rot before the producer ships.
- Re-ran `make proto-flex-check`, `make proto-lint`, `make proto-contract-test`, and `go test ./...` in `terminal_server`; all green.

## Protocol Evolution Rules (2026-05-04, FlowNode.typed_args producer round-trip test)

- Added `TestFlowPlanToProtoEmitsTypedArgsAlongsideLegacyArgs` and `TestFlowPlanToProtoTypedArgsNilWhenNoKnownKeys` in `terminal_server/internal/transport/generated_proto_adapter_test.go`. The first builds an `iorouter.FlowPlan` whose node carries all four known args (`device_id`, `resource`, `stream_kind=audio`, `name`) plus an unrelated extension key, runs it through `flowPlanToProto`, and asserts that (a) the legacy `FlowNode.args` map preserves every key including the extension passthrough and (b) `FlowNode.typed_args` mirrors all four typed fields with `stream_kind_enum=STREAM_KIND_AUDIO`. The second covers the nil-typed-mirror branch (only-unknown args populate legacy but leave `typed_args` nil), exercising the helper's empty-trim shortcut.
- No production code changed; the producer side (`flowPlanToProto` + `flowNodeTypedArgsFromArgs`) already emits both surfaces. The new tests pin that contract so a regression in the typed-mirror emission would be caught even before any inbound consumer materializes. If/when a `flowPlanFromProto` consumer is added it will need a sibling round-trip test that asserts typed-first preference with legacy fallback per the registry entry.
- Re-ran `make proto-flex-check`, `make proto-lint`, `make proto-breaking`, `make proto-contract-test`, `make client-test`, and `go test ./...` in `terminal_server`; all green.

## Protocol Evolution Rules (2026-05-05, CanvasWidget.draw_ops first producer)

- Added `terminal_server/internal/transport/canvas_draw_ops.go` with `canvasDrawOpsFromJSON`, a JSON→typed-DrawOp parser that decodes the `{"ops":[{"line":{...}}|{"rect":{...}}|{"circle":{...}}|{"text":{...}}|{"path":{...}}, ...]}` envelope shape into a `[]*uiv1.DrawOp` (one oneof arm per primitive). Malformed envelopes return nil; ops with zero or multiple variants set are skipped; empty/whitespace input returns nil. Per-primitive structs use snake_case JSON tags (`stroke_width`, `font_size`, `font_family`) matching the proto field names.
- Wired `descriptorToUINode`'s `case "canvas":` arm in `terminal_server/internal/transport/generated_proto_adapter.go` to call `canvasDrawOpsFromJSON(props["draw_ops_json"])` and populate both `CanvasWidget.DrawOpsJson` (preserved verbatim) and `CanvasWidget.DrawOps` (typed mirror) on every emitted canvas widget. This is the first producer for the typed `draw_ops` field; the consumer side (`ServerDrivenRenderer.canvas` typed-first preference) shipped 2026-05-04.
- Added unit tests in `terminal_server/internal/transport/generated_proto_adapter_test.go`:
  - `TestCanvasDrawOpsFromJSONParsesAllVariants` — well-formed envelope with one op per oneof arm round-trips into typed `DrawOp` messages with all numeric/string fields preserved.
  - `TestCanvasDrawOpsFromJSONReturnsNilForMalformedOrEmpty` — empty/whitespace, non-JSON, the existing test fixture's `{"ops":[{"line":"x"}]}` malformed shape, empty ops, multi-variant ops, and `{"ops":[]}` all return nil.
  - `TestCanvasDrawOpsFromJSONSkipsInvalidOpsKeepsValidOnes` — partial-failure mix yields the valid ops and silently drops the invalid ones.
  - `TestDescriptorToUINodeCanvasEmitsTypedAndLegacyDrawOps` — end-to-end via `descriptorToUINode` confirms both surfaces populated on the produced `*uiv1.Node`.
  - `TestDescriptorToUINodeCanvasMalformedJSONLeavesTypedDrawOpsEmpty` — malformed JSON: legacy string preserved, typed `DrawOps` empty (consumer falls back to legacy preview).
- Added a new golden envelope `api/testdata/envelopes/set_ui_canvas_v1.{textproto,binpb}` carrying a `SetUI` payload whose root contains a canvas child with both `draw_ops_json` and a typed `draw_ops` list (one rect + one line). Refreshed the binary sibling via `go test ./internal/protocolcontract -run TestGoldenWireEnvelopeFixtures -update -count=1`. Wired matching Go (`assertSetUICanvas` in `terminal_server/internal/protocolcontract/fixtures_test.go`) and Dart (`_assertSetUICanvas` in `terminal_client/test/protocol_contract_test.dart`) assertions so both sides agree on typed-vs-legacy coexistence.
- Updated the `CanvasWidget.draw_ops_json` registry entry in `docs/protocol-extension-registry.md` to record the producer landing on 2026-05-05, list the new test names + golden fixture, and document the `{"ops":[...]}` envelope shape the parser accepts. Updated the open-window row in `docs/compatibility.md` to replace "first producer still pending" with the producer-shipped date and start the two-release legacy-removal clock from 2026-05-05.
- Re-ran `make proto-flex-check`, `make proto-lint`, `make proto-breaking`, `make proto-contract-test`, `make client-test`, and `go test ./...` in `terminal_server`; all green. No proto regeneration was needed (no schema changes).

## Protocol Evolution Rules (2026-05-05, Native-typed canvas producer + DiagnosticsConnectionPulseOverlay)

- Added `terminal_server/internal/ui/canvas.go` with typed primitive builders (`Line`, `Rect`, `Circle`, `Text`, `Path`), a `CanvasOp` tagged-union value type (kind enum + per-variant pointer fields), `CanvasOpsToJSON` (typed→legacy `{"ops":[...]}` envelope serializer), and a `CanvasNode(id, ops...)` constructor that returns a `Descriptor` carrying typed ops in a new `Descriptor.CanvasOps []CanvasOp` field (with `json:"-"`) plus a serialized legacy `draw_ops_json` prop for the compatibility window. This is the first server-side producer that constructs typed `DrawOp` values directly in Go rather than going through the JSON-parse path.
- Added `DiagnosticsConnectionPulseOverlay(deviceID, healthy, rttMs)` in `internal/ui/canvas.go` as a concrete diagnostics surface: a global-overlay patch (`GlobalOverlayComponentID`) carrying a stack with a status text label and a typed `CanvasNode` rendering a status-color dot (green `#33CC66` healthy, red `#CC3333` down), a baseline rectangle, separator line, and an embedded RTT text label. Component id `diagnostics_connection_pulse`. The view is wired through the existing `descriptorToUINode → SetUI/UpdateUI` pipeline; the new transport-level test `TestDiagnosticsConnectionPulseOverlayProducesTypedCanvasOnWire` exercises the full path and asserts the expected typed `DrawOp` sequence reaches the wire.
- Updated `terminal_server/internal/transport/generated_proto_adapter.go` `applyWidgetFromDescriptor` to take a full `ui.Descriptor` (not just type+props) so the canvas arm can read `Descriptor.CanvasOps`. Canvas arm now prefers native typed ops via `canvasDrawOpsFromUI` (new helper in `internal/transport/canvas_draw_ops.go`) and falls back to `canvasDrawOpsFromJSON` only when `Descriptor.CanvasOps` is empty. Legacy `draw_ops_json` is preserved verbatim when both surfaces are populated, and is auto-derived from typed ops via `ui.CanvasOpsToJSON` only when the props map didn't already supply one.
- Added unit tests:
  - `internal/ui/canvas_test.go`: `TestCanvasBuildersPopulateTaggedUnion`, `TestCanvasNodeProducesDescriptorWithTypedAndLegacyMirror`, `TestCanvasNodeIsolatesOpsSlice`, `TestCanvasOpsToJSONReturnsEmptyForOnlyUnspecified`, `TestCanvasOpsToJSONSkipsBogusKeepsValid`, `TestDiagnosticsConnectionPulseOverlayHealthy`, `TestDiagnosticsConnectionPulseOverlayUnhealthy` cover builder semantics, descriptor shape, slice-aliasing protection, JSON envelope edge cases, and healthy/unhealthy diagnostics output.
  - `internal/transport/generated_proto_adapter_test.go`: `TestDescriptorToUINodeCanvasNativeTypedOpsBypassJSONParsing`, `TestDescriptorToUINodeCanvasNativeTypedOpsTrumpsLegacyJSON`, `TestDescriptorToUINodeCanvasNativeEmptyTypedFallsBackToJSON`, `TestDiagnosticsConnectionPulseOverlayProducesTypedCanvasOnWire` cover the native-typed adapter path, the precedence rule when both surfaces are populated (legacy JSON preserved verbatim, typed ops not re-parsed), the empty-typed→JSON-fallback edge, and the end-to-end diagnostics view.
- Updated the `CanvasWidget.draw_ops_json` registry entry (`docs/protocol-extension-registry.md`) to list the new builders, diagnostics surface, the native-typed adapter path, and all new test names. Updated the open-window row in `docs/compatibility.md` to note the native-typed producer landing alongside the JSON-driven producer (both shipped 2026-05-05) and bumped `last-reviewed` to 2026-05-05.
- Re-ran `make proto-flex-check`, `make proto-lint`, `make proto-breaking`, `make proto-contract-test`, `make client-test`, and `go test ./...` in `terminal_server`; all green. No proto regeneration was needed (no schema changes).
