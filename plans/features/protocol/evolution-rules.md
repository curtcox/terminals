---
title: "Protocol Evolution Rules"
kind: plan
status: building
owner: cascade
validation: manual
last-reviewed: 2026-05-03
---

# Protocol Evolution Rules

This plan defines how Terminals evolves protobuf contracts when a proposed change would otherwise use `map<string, string>`, free-form string tokens, or JSON embedded inside protobuf messages.

The goal is not to remove every flexible field immediately. The goal is to ensure every flexible field is intentional, documented, tested, and either governed as a stable extension point or migrated toward a typed protobuf contract.

## Context

Terminals depends on a stable client/server protocol:

- Flutter clients remain generic terminals.
- Server behavior evolves without scenario-specific client code.
- Protobuf is the canonical client/server contract.
- Buf catches schema-level breaking changes, but it cannot catch semantic drift inside metadata maps, free-form string tokens, or embedded JSON.

The audit identified that several fields remain string/map escape hatches, including `RegisterAck.metadata`, `CommandRequest.arguments`, `CommandResult.data`, `StartStream.kind`, `StartStream.metadata`, `FlowNode.kind`, `FlowNode.args`, `Observation.kind`, `Observation.attributes`, `Node.props`, and `CanvasWidget.draw_ops_json`.

Those fields are reasonable early-stage choices, but they need governance before they become invisible protocol dependencies.

## Problem Statement

The protocol currently allows durable client/server behavior to be added without changing the typed schema. Examples:

- A server can emit a new metadata key that a client starts depending on.
- A client and server can disagree about the allowed values of a string field such as `kind`, `state`, or `action`.
- A JSON payload can change shape without Buf detecting a breaking change.
- Generated-code checks can pass while older clients fail at runtime.
- Humans or agents can add behavior through metadata because it is faster than evolving protobuf.

This conflicts with the core project rule: client/server messages should be defined in protobuf, not ad-hoc JSON or undocumented maps.

## Goals

1. Preserve development speed while preventing accidental protocol drift.
2. Make every flexible protocol field documented and owned.
3. Prefer typed protobuf fields, enums, messages, and `oneof`s for stable semantics.
4. Allow explicitly governed extension points where flexibility is part of the design.
5. Add cross-language contract tests so Go and Dart agree on message semantics.
6. Give humans and agents a clear protocol-change workflow.

## Non-Goals

- Do not rewrite the entire protocol in one pass.
- Do not remove every map, string, or JSON field immediately.
- Do not block experimental server-only work that never crosses the client/server boundary.
- Do not introduce a separate schema language unless protobuf cannot express the needed shape.
- Do not couple protocol evolution to one specific scenario implementation.

## Classification Taxonomy

Use exactly one of these classifications for every flexible field in the registry.

### Typed Contract

A field whose semantics are represented directly in `.proto` using scalars, enums, messages, `oneof`, or repeated typed records.

Examples:

- `CapabilitySnapshot.capabilities`
- `Hello.identity`
- `PlayAudio.source`
- `DeviceCapabilities.screen`

Typed contracts do not need registry entries unless they also contain a flexible subfield.

### Constrained Scalar

A scalar field that is intentionally represented as a string, bytes, integer, or float, but whose format is constrained outside the protobuf type system.

Examples:

- URI strings
- RFC3339 timestamps
- SHA-256 digest strings
- semantic version strings
- MIME-like media types

A constrained scalar must document its format and validation behavior.

### Registry-Backed Extension

A flexible field that is intentionally retained because unknown keys or values are part of the design. It must have:

- an owner
- an allowed namespace or key list
- producer and consumer expectations
- unknown-key or unknown-value behavior
- validation behavior
- tests for unknown values
- promotion criteria for frequently used keys

### Transitional Escape Hatch

A temporary flexible field used while a concept is still unstable. It must have:

- an owner
- a review date
- a migration path
- current consumers
- tests for current behavior
- a rule forbidding new durable dependencies without either typing the field or reclassifying it as a registry-backed extension

### Display or Debug String

A string whose value is not part of machine-readable protocol behavior.

Examples:

- human-readable error messages
- user-facing notification text
- diagnostic details

Display/debug strings do not require enum migration, but code must not branch on their contents.

### External Payload

A field whose payload is defined by an external standard or separately versioned format.

Examples:

- SDP payloads
- ICE candidate payloads
- content addressed artifact URIs

External payloads must document the external format, size limits, and malformed-payload behavior.

## Current Flexible Fields to Classify

This inventory is the starting point for implementation. Phase 0 must verify it against `api/terminals/**/v1/*.proto` before creating the registry.

### `api/terminals/control/v1/control.proto`

| Field | Current Use Pattern | Classification | Target State |
|---|---|---|---|
| `TransportHelloAck.limits` | Transport/session limits | Registry-backed extension | Keep as map; document standard keys, value formats, and unknown-key behavior |
| `RegisterAck.metadata` | Server build metadata and asset base URLs | Transitional escape hatch | Add typed `ServerMetadata`; keep legacy map during compatibility window |
| `CommandRequest.intent` | Intent/scenario selector | Registry-backed extension | Document intent namespace; type high-use commands later if traffic stabilizes |
| `CommandRequest.arguments` | Command-specific arguments | Transitional escape hatch | Registry first; replace stable command shapes with typed messages or `oneof`s |
| `CommandResult.notification` | Human-readable result text | Display/debug string | Keep string; forbid machine branching |
| `CommandResult.data` | Command-specific output | Transitional escape hatch | Type durable result data; document temporary keys |
| `WebRTCSignal.signal_type` | SDP/ICE signal selector | Transitional escape hatch | Add `WebRTCSignalType` enum; keep string fallback during migration |
| `WebRTCSignal.payload` | SDP/ICE payload | External payload | Keep string; document payload format by signal type |

### `api/terminals/ui/v1/ui.proto`

| Field | Current Use Pattern | Classification | Target State |
|---|---|---|---|
| `Node.props` | Generic widget props | Transitional escape hatch | Move durable props into widget messages; reserve map for documented metadata only |
| `TextWidget.style` | Style selector | Registry-backed extension | Document style token registry; type only if stable and low-cardinality |
| `TextWidget.color` | Color value | Constrained scalar | Document accepted formats, such as named token or hex value |
| `ImageWidget.url` | Media source URI | Constrained scalar | Keep string URI; document allowed schemes |
| `ScrollWidget.direction` | Direction selector | Transitional escape hatch | Replace with enum |
| `ButtonWidget.action` | UI action token | Registry-backed extension | Document action namespace and server handling |
| `GestureAreaWidget.action` | UI action token | Registry-backed extension | Document action namespace and server handling |
| `TransitionUI.transition` | Transition token | Registry-backed extension | Document transition tokens; type only if stable and low-cardinality |
| `CanvasWidget.draw_ops_json` | JSON drawing operations | Transitional escape hatch | Prefer typed `DrawOp` messages; otherwise define strict JSON schema |

### `api/terminals/io/v1/io.proto`

| Field | Current Use Pattern | Classification | Target State |
|---|---|---|---|
| `StartStream.kind` | Stream category | Transitional escape hatch | Add `StreamKind` enum; keep string fallback during migration |
| `StartStream.metadata` | Routing/session hints | Transitional escape hatch | Type stable keys; document remaining extension namespace |
| `RouteStream.kind` | Stream category | Transitional escape hatch | Add `StreamKind` enum; keep string fallback during migration |
| `PlayAudio.format` | Audio format token | Constrained scalar | Document MIME-like format rules; consider enum only for built-in formats |
| `ShowMedia.media_type` | Media type token | Constrained scalar | Document MIME-like format rules; consider enum only for built-in formats |
| `PointerEvent.action` | Pointer action selector | Transitional escape hatch | Replace with enum |
| `TouchEvent.action` | Touch action selector | Transitional escape hatch | Replace with enum |
| `UIAction.action` | Client-originated UI action token | Registry-backed extension | Document action namespace and ownership |
| `SensorData.values` | Sensor readings | Registry-backed extension | Document sensor key registry, units, and unknown-key behavior |
| `FlowNode.kind` | Edge flow operator selector | Registry-backed extension | Document operator registry and versioning |
| `FlowNode.args` | Operator-specific arguments | Transitional escape hatch | Type built-in operator arguments once stable |
| `FlowNode.exec` | Execution target selector | Transitional escape hatch | Replace with enum |
| `Observation.kind` | Observation category | Registry-backed extension | Document observation taxonomy |
| `Observation.attributes` | Observation-specific attributes | Transitional escape hatch | Type stable attributes per observation kind |
| `ArtifactRef.kind` | Artifact category | Registry-backed extension | Document artifact kind registry; enum only for stable built-ins |
| `ArtifactRef.uri` | Artifact URI | Constrained scalar | Keep string URI; document allowed schemes |
| `FlowStats.state` | Flow state selector | Transitional escape hatch | Replace with enum |
| `FlowStats.error` | Human/debug error | Display/debug string | Keep string; forbid machine branching |
| `InstallBundle.bundle_id` | Bundle identifier | Constrained scalar | Document naming rules |
| `InstallBundle.version` | Bundle version | Constrained scalar | Document version format |
| `InstallBundle.sha256` | Bundle digest | Constrained scalar | Document SHA-256 hex format and validation |

## Protocol Evolution Principles

### 1. Stable behavior belongs in typed protobuf

If both client and server must agree on machine-readable semantics, prefer a typed field, enum, message, or `oneof`.

Acceptable exceptions:

- display/debug strings
- constrained scalar values
- registry-backed extension surfaces
- external payloads with documented formats
- transitional escape hatches with active migration plans

### 2. Maps are registries, not junk drawers

Every protocol map must declare:

- owner
- allowed keys or key namespace
- producer and consumer
- value format
- unknown-key behavior
- validation behavior
- promotion trigger for keys that become durable

### 3. String selectors need an explicit policy

A string selector is any string field that code compares against a fixed or semi-fixed set of values.

For each string selector, choose one target:

- replace with enum
- keep as registry-backed extension
- keep as constrained scalar if it follows an external format, such as MIME

Do not leave string selectors undocumented.

### 4. JSON-in-protobuf requires a schema

A JSON string in protobuf is allowed only when all of the following are documented and tested:

- schema version
- allowed top-level shape
- unknown-field behavior
- size limit
- malformed JSON behavior
- migration path to typed protobuf or justification for keeping JSON

### 5. Unknown values must fail predictably

For each flexible field, choose one behavior:

- ignore unknown value
- preserve unknown value without interpreting it
- reject the message with a typed protocol error
- downgrade to a safe default

No consumer should silently reinterpret an unknown value.

### 6. Compatibility is additive first

Typed replacements must be additive during migration:

1. Add the new typed field using a new field number.
2. Emit both old and new fields from the producer.
3. Read the new field first in consumers.
4. Fall back to the old field while old producers are supported.
5. Mark the old field or key deprecated only after fallback behavior is tested.
6. Remove old behavior only after the documented compatibility window closes.

### 7. Cross-language behavior must be tested

Go and Dart must agree on:

- default values
- enum fallback behavior
- string fallback behavior
- unknown metadata behavior
- oneof selection
- deprecated field handling
- malformed JSON behavior where JSON remains

### 8. Agents must use the protocol-change workflow

Agent-generated changes must not add new flexible fields, metadata keys, string tokens, or JSON payloads without updating the protocol registry and contract tests.

## Proposed Artifacts

### `docs/protocol-evolution.md`

Human-readable policy for protocol changes. It should include:

- classification taxonomy
- when to add typed fields
- when maps are acceptable
- when strings are acceptable
- when JSON is acceptable
- deprecation rules
- compatibility rules
- reviewer checklist
- agent checklist

### `docs/protocol-extension-registry.md`

Registry of all registry-backed extensions and transitional escape hatches.

Suggested entry format:

```markdown
## Field: terminals.control.v1.RegisterAck.metadata

Owner: transport/control
Classification: transitional_escape_hatch
Target state: typed ServerMetadata
Review date: 2026-06-15

Allowed keys:

### photo_frame_asset_base_url

Type: URI string
Producer: server
Consumer: client
Unknown behavior: ignored by client
Validation: must parse as absolute HTTP(S) URL when present
Promotion trigger: if another server asset/service URL is added, introduce typed ServerMetadata

### server_build_sha

Type: short git SHA string
Producer: server
Consumer: client diagnostics
Unknown behavior: ignored by client
Validation: non-empty string; "unknown" allowed for local builds
Promotion trigger: move into typed BuildMetadata

### server_build_date

Type: RFC3339 timestamp string
Producer: server
Consumer: client diagnostics
Unknown behavior: ignored by client
Validation: RFC3339 timestamp or "unknown"
Promotion trigger: move into typed BuildMetadata

Tests:

- Go server emits only registered keys.
- Dart client ignores unknown keys.
- Dart client prefers typed metadata once available.
```

### `api/CONTRACTS.md`

API-near checklist for proto changes:

- Did this add or change a flexible field?
- Did this add a metadata key, string token, or JSON shape?
- Is the registry updated?
- Is the field classification correct?
- Are Go and Dart behavior tests included?
- Are golden fixtures added or updated?
- Did `make proto-generate` and `make proto-lint` pass?
- Did protocol contract tests pass?
- Does `docs/compatibility.md` need an update?

### `api/testdata/envelopes/`

Golden protocol fixtures shared by Go and Dart tests.

Initial fixtures:

- `hello_snapshot_v1.textproto`
- `hello_snapshot_v1.binpb`
- `register_ack_metadata_v1.textproto`
- `register_ack_metadata_v1.binpb`
- `set_ui_basic_v1.textproto`
- `set_ui_basic_v1.binpb`
- `start_stream_audio_v1.textproto`
- `start_stream_audio_v1.binpb`
- `flow_plan_basic_v1.textproto`
- `flow_plan_basic_v1.binpb`
- `observation_sound_v1.textproto`
- `observation_sound_v1.binpb`
- `unknown_metadata_key_v1.textproto`
- `unknown_metadata_key_v1.binpb`
- `deprecated_register_device_v1.textproto`
- `deprecated_register_device_v1.binpb`

### `scripts/check-proto-flex-fields.py`

Static check that scans `api/terminals/**/*.proto` for likely flexible fields:

- `map<`
- field names ending in `_json`
- field names equal to `metadata`, `attributes`, `props`, `args`, `data`, `kind`, `action`, `state`, `type`, or `format`

The check should verify that each detected field is listed in `docs/protocol-extension-registry.md` unless it is explicitly suppressed as a typed contract or display/debug string.

Start in advisory mode. Flip to required after the initial registry is complete.

### `make proto-contract-test`

Make target that runs:

- existing proto lint and generation checks
- flexible-field registry check
- Go golden fixture decode/validate tests
- Dart golden fixture decode/validate tests

## Implementation Plan

### Phase 0 - Inventory and Classification

Effort: small  
Risk: low

Tasks:

1. Scan `api/terminals/**/v1/*.proto` for flexible fields.
2. Classify each detected field using the taxonomy in this plan.
3. Create initial `docs/protocol-extension-registry.md` entries.
4. Add `api/CONTRACTS.md` with the protocol-change checklist.
5. Record any field that cannot yet be classified under an explicit `Unresolved` section with an owner and review date.

Acceptance criteria:

- Every current map field appears in the registry.
- Every current JSON field appears in the registry.
- Every current string selector appears in the registry or in `Unresolved` with an owner.
- Each registry entry has owner, classification, producer, consumer, unknown behavior, validation behavior, tests, and target state.

Validation:

```bash
rg 'map<|_json|metadata|attributes|props|args|data|kind|action|state|format' api/terminals
make proto-lint
```

### Phase 1 - Documentation and Review Rules

Effort: small  
Risk: low

Tasks:

1. Create `docs/protocol-evolution.md` from the durable policy sections of this plan.
2. Add protocol-change checklist language to `.github/pull_request_template.md` if that file exists; otherwise create it.
3. Add agent-facing protocol rules to `api/CONTRACTS.md`.
4. Link protocol evolution docs from:
   - `masterplan.md`
   - `plans/features/protocol/plan.md`
   - `api/CONTRACTS.md`

Acceptance criteria:

- A reviewer can evaluate a protocol change using one checklist.
- Agents have a concise workflow for proto changes.
- Existing protocol docs link to the new policy.

Validation:

```bash
make development-docs-test
rg 'protocol-evolution|protocol-extension-registry|CONTRACTS.md' .
```

### Phase 2 - Static Guardrail

Effort: medium  
Risk: low

Tasks:

1. Add `scripts/check-proto-flex-fields.py`.
2. Report file, line, message, field name, and reason for every detected flexible field.
3. Verify detected fields have registry entries.
4. Add Make target:

```make
proto-flex-check:
	python3 ./scripts/check-proto-flex-fields.py
```

5. Add `proto-flex-check` to `proto-contract-test` in advisory mode.
6. Flip `proto-flex-check` to required after the registry covers all current fields.

Acceptance criteria:

- New flexible fields are detected automatically.
- Existing flexible fields are matched to registry entries.
- False positives can be explicitly suppressed with a registry entry, not by hiding the field.
- Output is actionable enough for a PR author to fix without reading the script.

Validation after target exists:

```bash
make proto-flex-check
```

### Phase 3 - Cross-Language Golden Contract Tests

Effort: medium  
Risk: medium

Tasks:

1. Create `api/testdata/envelopes/`.
2. Add textproto and binary fixtures for the highest-risk messages:
   - `ConnectRequest.Hello`
   - `ConnectRequest.CapabilitySnapshot`
   - `ConnectResponse.RegisterAck`
   - `ConnectResponse.SetUI`
   - `ConnectResponse.StartStream`
   - `ConnectResponse.StartFlow`
   - `ConnectRequest.ObservationMessage`
3. Add Go tests that decode fixtures, validate expected fields, verify fallback behavior, and re-encode for semantic equality.
4. Add Dart tests that decode the same fixtures and validate the same behavior.
5. Add Make target:

```make
proto-contract-test:
	$(MAKE) proto-lint
	$(MAKE) proto-flex-check
	cd terminal_server && go test ./internal/protocolcontract ./internal/transport
	cd terminal_client && flutter test test/protocol_contract_test.dart
```

Acceptance criteria:

- Go and Dart decode the same golden fixtures.
- Unknown metadata keys follow documented behavior.
- Deprecated fields remain decodable for compatibility fixtures.
- Golden tests run locally and in CI.

Validation after target exists:

```bash
make proto-generate
make proto-contract-test
```

### Phase 4 - Type the Highest-Value Fields

Effort: medium to large  
Risk: medium

Convert the most durable flexible fields into additive typed fields. Do not remove legacy fields in this phase.

#### Candidate 1: Build and Server Metadata

Current field:

- `RegisterAck.metadata["server_build_sha"]`
- `RegisterAck.metadata["server_build_date"]`
- `RegisterAck.metadata["photo_frame_asset_base_url"]`

Proposed additive fields:

```protobuf
message BuildMetadata {
  string sha = 1;
  string date_rfc3339 = 2;
}

message ServerMetadata {
  BuildMetadata build = 1;
  string photo_frame_asset_base_url = 2;
}

message RegisterAck {
  string server_id = 1;
  string message = 2;
  map<string, string> metadata = 3;
  ServerMetadata server_metadata = 4;
}
```

Migration steps:

1. Server emits both `metadata` and `server_metadata`.
2. Client prefers `server_metadata` when present.
3. Client falls back to `metadata` for old servers.
4. Golden tests cover old-only, new-only, and both-fields payloads.
5. Mark `metadata` deprecated in a later PR after compatibility behavior is verified.

#### Candidate 2: Stream Kind

Current fields:

- `StartStream.kind`
- `RouteStream.kind`

Proposed enum:

```protobuf
enum StreamKind {
  STREAM_KIND_UNSPECIFIED = 0;
  STREAM_KIND_AUDIO = 1;
  STREAM_KIND_VIDEO = 2;
  STREAM_KIND_SENSOR = 3;
  STREAM_KIND_DATA = 4;
}
```

Migration steps:

1. Add new enum fields using unused field numbers.
2. Server emits both enum and legacy string.
3. Client prefers enum when non-zero.
4. Client falls back to string while old servers are supported.
5. Tests cover unknown enum, unspecified enum, and legacy string fallback.

#### Candidate 3: WebRTC Signal Type

Current field:

- `WebRTCSignal.signal_type`

Proposed enum:

```protobuf
enum WebRTCSignalType {
  WEBRTC_SIGNAL_TYPE_UNSPECIFIED = 0;
  WEBRTC_SIGNAL_TYPE_OFFER = 1;
  WEBRTC_SIGNAL_TYPE_ANSWER = 2;
  WEBRTC_SIGNAL_TYPE_ICE_CANDIDATE = 3;
}
```

Migration steps:

1. Add a new enum field using an unused field number.
2. Keep `signal_type` during transition.
3. Validate that payload shape matches resolved signal type.
4. Add golden fixtures for offer, answer, and ICE candidate.

#### Candidate 4: Flow State

Current field:

- `FlowStats.state`

Proposed enum:

```protobuf
enum FlowState {
  FLOW_STATE_UNSPECIFIED = 0;
  FLOW_STATE_STARTING = 1;
  FLOW_STATE_RUNNING = 2;
  FLOW_STATE_DEGRADED = 3;
  FLOW_STATE_STOPPING = 4;
  FLOW_STATE_STOPPED = 5;
  FLOW_STATE_FAILED = 6;
}
```

Migration steps:

1. Add a new enum field using an unused field number.
2. Server emits both enum and legacy string.
3. Client/admin surfaces read enum first.
4. Registry documents the legacy string fallback.

#### Candidate 5: Canvas Draw Operations

Current field:

- `CanvasWidget.draw_ops_json`

Preferred typed replacement:

```protobuf
message DrawOp {
  oneof op {
    DrawLine line = 1;
    DrawRect rect = 2;
    DrawCircle circle = 3;
    DrawText text = 4;
    DrawPath path = 5;
  }
}

message CanvasWidget {
  repeated DrawOp draw_ops = 1;
  string draw_ops_json = 2;
}
```

Migration steps:

1. Add typed draw operation messages.
2. Renderer prefers `draw_ops` when present.
3. Keep `draw_ops_json` only for compatibility or explicitly experimental operations.
4. Add malformed JSON tests if JSON remains supported.
5. Mark `draw_ops_json` deprecated in a later PR if typed operations cover active use cases.

Acceptance criteria:

- At least three high-value flexible fields have additive typed replacements.
- Old fields remain readable during the compatibility window.
- Producers emit both old and new fields where needed.
- Consumers prefer new fields and fall back to old fields.
- Registry marks old fields/keys as transitional or deprecated only after fallback tests exist.

Validation:

```bash
make proto-generate
make proto-contract-test
make server-test
make client-test
```

### Phase 5 - CI Enforcement

Effort: small to medium  
Risk: low

Tasks:

1. Add `make proto-contract-test` to `proto-ci.yml`.
2. Make `proto-flex-check` required after advisory mode is clean.
3. Add docs validation for protocol registry links.
4. Add PR template checks.
5. Emit GitHub workflow annotations for new flexible fields.

Acceptance criteria:

- PRs adding new flexible fields fail unless the registry is updated.
- PRs changing typed protocol semantics update golden fixtures or tests.
- Proto CI remains the canonical gate for generated code and protocol compatibility.

Validation:

```bash
make all-check
make proto-contract-test
```

### Phase 6 - Cleanup and Deprecation

Effort: ongoing  
Risk: medium

Tasks:

1. Track compatibility windows in `docs/compatibility.md`.
2. Mark old fields or keys deprecated only after consumers prefer typed replacements.
3. Remove old code paths only after the documented compatibility window closes.
4. Keep golden fixtures for old payloads while old clients or servers remain supported.
5. Add release notes for each protocol migration.

Acceptance criteria:

- Deprecated metadata keys and fields have removal criteria.
- Compatibility docs identify supported old clients and servers.
- Release notes explain protocol migrations.
- No long-lived flexible field lacks a registry entry.

Validation after `docs-check` exists:

```bash
make proto-contract-test
make docs-check
```

## Review Checklist for Future Protocol Changes

Use this checklist in PRs that touch `api/terminals/**`.

### Schema Design

- [ ] Is this behavior expressed with typed protobuf fields where practical?
- [ ] If this uses a scalar string, is it a constrained scalar, display/debug string, external payload, or selector?
- [ ] If this is a selector, is it an enum, a registry-backed extension, or a documented transitional escape hatch?
- [ ] If this uses a map, is it a documented registry-backed extension or transitional escape hatch?
- [ ] If this uses JSON, is there a schema and version?
- [ ] Are unknown values handled predictably?

### Compatibility

- [ ] Is the change additive for existing clients and servers?
- [ ] Are deprecated fields still decoded where needed?
- [ ] Are new fields optional or safely defaulted?
- [ ] Do producers emit both old and new fields during migration?
- [ ] Do consumers prefer new fields and fall back to old fields during migration?
- [ ] Is `docs/compatibility.md` updated if behavior changes?

### Validation

- [ ] Did Buf format, lint, generate, and breaking checks pass?
- [ ] Did Go contract tests pass?
- [ ] Did Dart contract tests pass?
- [ ] Are golden fixtures added or updated?
- [ ] Is the protocol extension registry updated?
- [ ] Does `proto-flex-check` pass?

### Agent Safety

- [ ] Would an agent be tempted to use metadata, string tokens, or JSON instead of typing this?
- [ ] Is the intended extension path documented?
- [ ] Are review instructions clear enough for agent-generated PRs?

## Whole-Plan Acceptance Criteria

The plan is complete when:

1. Every current flexible protocol field is inventoried.
2. Every flexible field has a registry entry, typed replacement, or explicit unresolved owner.
3. CI detects new flexible fields.
4. Go and Dart share golden contract fixtures.
5. At least three high-value fields have typed replacements or strict schemas.
6. PR and agent workflows require registry and test updates for protocol changes.
7. Compatibility docs track deprecated fields and migration windows.

## Suggested File Changes

```text
docs/protocol-evolution.md
docs/protocol-extension-registry.md
docs/compatibility.md
api/CONTRACTS.md
api/testdata/envelopes/*.textproto
api/testdata/envelopes/*.binpb
scripts/check-proto-flex-fields.py
terminal_server/internal/protocolcontract/*_test.go
terminal_client/test/protocol_contract_test.dart
.github/pull_request_template.md
.github/workflows/proto-ci.yml
Makefile
```

## Suggested Make Targets

```make
proto-flex-check:
	python3 ./scripts/check-proto-flex-fields.py

proto-contract-test:
	$(MAKE) proto-lint
	$(MAKE) proto-flex-check
	cd terminal_server && go test ./internal/protocolcontract ./internal/transport
	cd terminal_client && flutter test test/protocol_contract_test.dart

docs-check:
	./scripts/test-development-environment-docs.sh
	python3 ./scripts/generate-plans-index.py --check
	python3 ./scripts/generate-validation-matrix.py --check
	python3 ./scripts/generate-usecases-index.py --check
```

## Initial Work Breakdown

### PR 1 - Inventory and Docs

- Add `docs/protocol-evolution.md`.
- Add `docs/protocol-extension-registry.md`.
- Add `api/CONTRACTS.md`.
- Inventory current flexible fields.
- Add PR checklist language.

Risk: low  
Review focus: policy clarity and completeness.

### PR 2 - Static Guardrail

- Add `scripts/check-proto-flex-fields.py`.
- Add `make proto-flex-check`.
- Run in advisory mode.
- Add CI step that reports but does not fail while the registry is incomplete.

Risk: low  
Review focus: signal quality and false positives.

### PR 3 - Golden Contract Harness

- Add fixture directory.
- Add first textproto/binpb fixtures.
- Add Go decode tests.
- Add Dart decode tests.
- Add `make proto-contract-test`.

Risk: medium  
Review focus: fixture generation and reproducibility.

### PR 4 - Build Metadata Typing

- Add `BuildMetadata` and `ServerMetadata`.
- Server emits typed metadata and legacy metadata.
- Client prefers typed metadata.
- Add Go/Dart fixtures and tests.

Risk: medium  
Review focus: compatibility fallback.

Incremental progress (2026-05-03):

- Added additive typed `BuildMetadata` and `ServerMetadata` messages to `RegisterAck` while preserving legacy `metadata`.
- Updated server register-ack generation to emit both typed `server_metadata` and legacy map values.
- Updated client register-ack handling to prefer typed build metadata and fall back to legacy map keys.
- Extended Go and Dart contract tests plus envelope fixtures to validate typed + legacy compatibility behavior.
- Ran `make proto-generate`, refreshed binary fixtures via `go test ./internal/protocolcontract -run TestGoldenWireEnvelopeFixtures -update`, and re-ran targeted Go transport/contract tests.

### PR 5 - Stream and WebRTC Enum Migration

- Add typed enums for stream kind and WebRTC signal type.
- Server emits typed fields and legacy strings.
- Client prefers typed fields.
- Add unknown/fallback tests.

Risk: medium  
Review focus: old-client behavior and route/media regressions.

Incremental progress (2026-05-03, ScrollDirection):

- Added additive typed enum `terminals.ui.v1.ScrollDirection` and `ScrollWidget.direction_enum` while preserving the legacy `direction` string for compatibility.
- Adapter `applyWidgetFromDescriptor` now populates both typed enum and legacy string from `props["direction"]`.
- Flutter server-driven renderer prefers `direction_enum` and falls back to the legacy string when unspecified.
- Updated registry entry for `ScrollWidget.direction` to describe typed-first compatibility semantics.
- Replaced brittle Dart `switch` expressions over `ProtobufEnum` constants in `control_response_dispatcher` / `webrtc_engine` with defensive `if`/`==` chains; split the `typed enum fields override` dispatcher test into per-payload responses (oneof-aware).

Incremental progress (2026-05-03, FlowState):

- Added additive typed enum `terminals.io.v1.FlowState` and `FlowStats.state_enum` while keeping legacy `state` string for compatibility.
- Adapter now reads the typed enum into `FlowStatsRequest.StateEnum`; server flow stats handler logs the resolved state enum-first with legacy string fallback.
- Added `flow_stats_v1` golden envelope fixture exercised by both Go and Dart contract tests.
- Updated registry entry for `FlowStats.state` to describe typed-first compatibility semantics.

Incremental progress (2026-05-03):

- Added additive typed enums: `terminals.io.v1.StreamKind` and `terminals.control.v1.WebRTCSignalType`, with new `StartStream.stream_kind`, `RouteStream.stream_kind`, and `WebRTCSignal.signal_type_enum` fields while preserving legacy string fields.
- Updated generated-proto transport adapter to emit typed enum + legacy string fields and to resolve inbound WebRTC signal type from enum first with legacy fallback.
- Updated Flutter client media/control response handling to prefer typed enum fields and fall back to legacy string values during compatibility.
- Expanded Go and Dart protocol contract/assertion coverage for typed stream kind behavior, plus adapter tests for enum-first WebRTC fallback semantics.
- Updated protocol extension registry entries for `StartStream.kind`, `RouteStream.kind`, and `WebRTCSignal.signal_type` to describe typed-first compatibility behavior.

Incremental progress (2026-05-03, ExecPolicy):

- Added additive typed enum `terminals.io.v1.ExecPolicy` and `FlowNode.exec_policy` while preserving the legacy `exec` string.
- Generated proto adapter emits both typed `exec_policy` and legacy `exec` from internal `iorouter.ExecPolicy` values (`auto`, `prefer_client`, `require_client`, `server_only`).
- Extended `flow_plan_basic_v1` envelope fixture and Go/Dart contract assertions for typed-first + legacy-string compatibility.
- Updated registry entry for `FlowNode.exec` to describe typed-first compatibility semantics.

### PR 6 - Enforcement

- Flip `proto-flex-check` from advisory to required.
- Add docs link validation.
- Add release/compatibility entries for deprecated fields.

Risk: low to medium  
Review focus: avoiding excessive friction.

## Open Questions

1. What compatibility window should old protocol fields receive?
   - Suggested default: keep old fields for at least two tagged releases after typed replacements ship.

2. Should extension registry entries live in Markdown or machine-readable YAML?
   - Suggested start: Markdown for readability.
   - Upgrade to YAML only if static checks need structured metadata that Markdown cannot provide reliably.

3. Should `CommandRequest.arguments` become a typed `oneof` per command, or remain a registry-backed extension surface?
   - Suggested approach: keep registry-backed until command traffic stabilizes, then type high-use commands.

4. Should observation taxonomy be governed in protobuf enums or as a docs-backed registry?
   - Suggested approach: registry first, enum only for low-cardinality stable observation categories.

5. Should `CanvasWidget.draw_ops_json` be removed before any real canvas-based app depends on it?
   - Suggested approach: yes, unless strict JSON schema and tests are added immediately.

## Related Plans and Docs

- `plans/features/protocol/plan.md`
- `plans/features/server-driven-ui.md`
- `plans/features/io-abstraction/plan.md`
- `plans/features/application-runtime.md`
- `masterplan.md`
- `api/terminals/control/v1/control.proto`
- `api/terminals/ui/v1/ui.proto`
- `api/terminals/io/v1/io.proto`

## Implementation Progress

Create `progress.md` next to this file once work begins. Keep it append-only.

Suggested path:

```text
plans/features/protocol-evolution/progress.md
```

Suggested first entry:

```markdown
## 2026-05-02

- Created protocol evolution rules plan.
- Classified current flexible protocol field categories.
- Proposed registry, static guardrail, and Go/Dart golden contract tests.
```
