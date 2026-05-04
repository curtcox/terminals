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

## Protocol Evolution Rules (2026-05-04, PointerAction/TouchAction Enum Migration)

- Added additive typed enums `terminals.io.v1.PointerAction` and `terminals.io.v1.TouchAction` and the corresponding `PointerEvent.action_enum` (field 7) and `TouchEvent.action_enum` (field 3) while preserving legacy `action` string fields.
- Regenerated Go bindings via `make proto-generate` and synced the refreshed `terminals/io/v1` Dart bindings (`io.pb.dart`, `io.pbenum.dart`, `io.pbjson.dart`) into `terminal_client/lib/gen/`.
- Updated the protocol extension registry entries for `PointerEvent.action` and `TouchEvent.action` to describe typed-first compatibility semantics (typed enum preferred, legacy string fallback during the migration window).
- No application-code paths currently route pointer/touch input, so adapter wiring is deferred until a producer/consumer lands; the typed fields are now available for the first non-test consumer.
- Re-ran `make proto-contract-test` and `make server-test`; all green.
