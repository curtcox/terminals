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

## Protocol Evolution Rules (2026-05-03, StreamKind/WebRTCSignalType Enum Migration)

- Added additive typed enums `StreamKind` and `WebRTCSignalType` in `api/terminals/io/v1/io.proto` and `api/terminals/control/v1/control.proto` with new `stream_kind` and `signal_type_enum` fields while preserving legacy string fields.
- Updated generated transport adapter mappings to emit typed+legacy values and to resolve inbound WebRTC signal type from enum first with legacy-string fallback.
- Updated Flutter control/media handling to prefer typed stream/signal enums for notifications and runtime media behavior, while retaining legacy fallback.
- Extended Go adapter tests and Go/Dart protocol contract assertions to validate typed enum emission and compatibility fallback behavior.
- Updated `start_stream_audio_v1` text/bin fixtures to include typed `stream_kind`, and refreshed protocol registry migration notes for typed-first semantics.
