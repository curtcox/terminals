---
title: "Go-Dart Contract Golden Tests"
kind: plan
status: draft
owner: copilot
validation: automated
last-reviewed: 2026-05-02
---

# Go-Dart Contract Golden Tests

## Summary

Add a shared golden-message contract test suite that proves the Go server and Dart/Flutter client agree on protobuf wire behavior for the control stream. The suite will use one shared fixture corpus, decode those fixtures in both runtimes, assert the same compatibility-critical fields, and run in local validation and CI.

This plan implements the audit recommendation: **Add Go↔Dart contract golden tests**.

These tests do **not** replace Buf linting, Buf generation checks, or Buf breaking checks. Buf validates schema shape and generated-code freshness. Contract golden tests validate runtime behavior: what the generated Go and Dart code actually encodes, decodes, preserves, rejects, and normalizes.

## Problem Statement

The protocol is the most important compatibility boundary between `terminal_server` and `terminal_client`. Current validation covers schema-level safety, generated-code freshness, and language-specific tests, but it does not prove that both generated runtimes agree on representative real messages.

Without a shared corpus, regressions can still slip through:

- Go emits an envelope that Dart can parse but interprets differently.
- Dart emits a capability snapshot whose default values differ from server assumptions.
- A deprecated payload remains parseable in Go but becomes untested in Dart.
- Map fields, string extension points, and JSON-in-string fields drift semantically.
- Generated-code updates compile but change oneof, enum, default-value, or unknown-field behavior.
- Transport negotiation behavior around `protocol_version`, `CarrierKind`, `TransportHello`, and `TransportHelloAck` is not captured as reusable fixture data.

## Goals

1. Create a shared fixture corpus under `api/testdata/contract/`.
2. Decode the same binary protobuf fixtures in Go and Dart.
3. Assert the same semantic expectations in both languages.
4. Keep textproto source files beside binary fixtures for human review.
5. Provide a repeatable generator for `.textproto` to `.binpb` updates.
6. Add Make targets and CI jobs so proto-impacting changes run the contract suite.
7. Document the fixture-update workflow for humans and agents.

## Non-Goals

- Do not create a second schema language.
- Do not replace Buf lint, generation, or breaking-change checks.
- Do not require byte-for-byte round trips for map-heavy, unknown-field-heavy, or cross-runtime messages unless explicitly marked safe.
- Do not test every generated accessor.
- Do not add scenario-specific client logic.
- Do not bless new map keys, string enum-like values, or JSON-in-string schemas without documentation.
- Do not make JSON the primary wire contract. Binary protobuf remains the compatibility artifact.

## Target Plan Location

Recommended repository location:

```text
plans/features/protocol-contract-golden-tests/plan.md
plans/features/protocol-contract-golden-tests/progress.md
```

This keeps the work adjacent to the protocol feature plans while preserving the existing plan/progress convention.

## Proposed Repository Layout

```text
api/
  testdata/
    contract/
      README.md
      manifest.yaml
      fixtures/
        control_hello_v1.textproto
        control_hello_v1.binpb
        capability_snapshot_full_v1.textproto
        capability_snapshot_full_v1.binpb
        capability_delta_display_resize_v1.textproto
        capability_delta_display_resize_v1.binpb
        set_ui_basic_v1.textproto
        set_ui_basic_v1.binpb
        input_ui_action_v1.textproto
        input_ui_action_v1.binpb
        transport_hello_grpc_ws_v1.textproto
        transport_hello_grpc_ws_v1.binpb
      expected/
        control_hello_v1.yaml
        capability_snapshot_full_v1.yaml
        capability_delta_display_resize_v1.yaml
        set_ui_basic_v1.yaml
        input_ui_action_v1.yaml
        transport_hello_grpc_ws_v1.yaml
terminal_server/
  internal/
    contracttest/
      contract_test.go
      manifest.go
      assertions.go
terminal_client/
  test/
    contract/
      contract_golden_test.dart
      fixture_loader.dart
      assertions.dart
scripts/
  proto-contract-generate.sh
  proto-contract-test.sh
  proto-contract-verify.sh
```

Phase 3 adds more fixture files to the same directories.

## Fixture Source of Truth

Each fixture has two representations:

- `.textproto` is the human-edited review source.
- `.binpb` is the binary protobuf artifact decoded by both Go and Dart tests.

The generator updates `.binpb` from `.textproto`. Tests should read `.binpb` only. CI should verify that `.binpb` is current by regenerating it and checking `git diff`.

## Manifest Design

Create `api/testdata/contract/manifest.yaml`:

```yaml
version: 1
fixtures:
  - id: control_hello_v1
    file: fixtures/control_hello_v1.binpb
    textproto: fixtures/control_hello_v1.textproto
    message: terminals.control.v1.ConnectRequest
    payload: hello
    direction: client_to_server
    purpose: Initial client hello with identity and client version.
    round_trip: semantic
    expected: expected/control_hello_v1.yaml
    tags: [control, handshake]

  - id: capability_snapshot_full_v1
    file: fixtures/capability_snapshot_full_v1.binpb
    textproto: fixtures/capability_snapshot_full_v1.textproto
    message: terminals.control.v1.ConnectRequest
    payload: capability_snapshot
    direction: client_to_server
    purpose: Full capability baseline with display, input, media, sensor, battery, edge, and haptic data.
    round_trip: semantic
    expected: expected/capability_snapshot_full_v1.yaml
    tags: [capabilities, snapshot]

  - id: set_ui_basic_v1
    file: fixtures/set_ui_basic_v1.binpb
    textproto: fixtures/set_ui_basic_v1.textproto
    message: terminals.control.v1.ConnectResponse
    payload: set_ui
    direction: server_to_client
    purpose: Basic server-driven UI tree with layout, text, button, and action identifiers.
    round_trip: semantic
    expected: expected/set_ui_basic_v1.yaml
    tags: [ui]
```

### Manifest Field Rules

- `id` must be stable, unique, lowercase snake_case, and match the fixture basename.
- `file` points to the binary protobuf fixture, relative to `api/testdata/contract/`.
- `textproto` points to the editable source fixture, relative to `api/testdata/contract/`.
- `message` must be a fully qualified protobuf type generated in both Go and Dart.
- `payload` must name the expected oneof case for envelope messages.
- `direction` must be `client_to_server`, `server_to_client`, or `transport`.
- `purpose` must describe the compatibility behavior being preserved.
- `round_trip` must be `semantic` or `byte_exact`.
- `expected` points to a YAML assertion file.
- `tags` are optional labels for targeted test runs later.

### Round-Trip Rule

Use `round_trip: semantic` by default.

A semantic round trip means:

1. decode fixture bytes;
2. assert expected fields;
3. encode the decoded message;
4. decode the encoded bytes again;
5. assert expected fields again.

Use `round_trip: byte_exact` only for simple messages with no maps, no unknown fields, and deterministic behavior in both runtimes. Do not use byte-exact checks for capability snapshots, metadata maps, argument maps, or observation attributes.

## Expected Assertion Files

Expected assertion files are small compatibility checklists, not full snapshots.

Example `api/testdata/contract/expected/control_hello_v1.yaml`:

```yaml
message: terminals.control.v1.ConnectRequest
payload: hello
assertions:
  - path: hello.device_id
    equals: kitchen-tablet-01
  - path: hello.identity.device_name
    equals: Kitchen Tablet
  - path: hello.identity.device_type
    equals: tablet
  - path: hello.identity.platform
    equals: android
  - path: hello.client_version
    equals: 0.1.0-dev
```

Example `api/testdata/contract/expected/transport_hello_grpc_ws_v1.yaml`:

```yaml
message: terminals.control.v1.WireEnvelope
payload: transport_hello
assertions:
  - path: protocol_version
    equals: 1
  - path: sequence
    equals: 1
  - path: transport_hello.protocol_version
    equals: 1
  - path: transport_hello.supported_carriers
    contains:
      - CARRIER_KIND_GRPC
      - CARRIER_KIND_WEBSOCKET
  - path: transport_hello.desired_device_id
    equals: kitchen-tablet-01
```

### Assertion Operators

Phase 1 supports:

- `equals`
- `contains`
- `length`

Phase 2 adds:

- `present`
- `absent`

Enum assertions should use proto enum names, not numeric values. Test helpers should normalize Go and Dart enum representations to the same name string.

Oneof payload assertions should use snake_case names from the proto field, such as `capability_snapshot` and `set_ui`. Dart helpers should normalize generated camelCase oneof names back to snake_case.

## Initial Fixture Set

Phase 1 includes exactly six fixtures. This keeps the first implementation small enough to finish while still covering the highest-risk message families.

### 1. `control_hello_v1`

- **Message:** `terminals.control.v1.ConnectRequest`
- **Payload:** `hello`
- **Direction:** `client_to_server`
- **Purpose:** Prove both languages agree on initial hello identity fields.
- **Assertions:** device id, device name, device type, platform, client version.

### 2. `capability_snapshot_full_v1`

- **Message:** `terminals.control.v1.ConnectRequest`
- **Payload:** `capability_snapshot`
- **Direction:** `client_to_server`
- **Purpose:** Prove a realistic full capability baseline decodes consistently.
- **Assertions:** device id, generation, screen geometry, keyboard, pointer, touch, speaker endpoint, microphone endpoint, camera mode, sensors, connectivity, battery, edge runtime/operator, display count, haptics.

### 3. `capability_delta_display_resize_v1`

- **Message:** `terminals.control.v1.ConnectRequest`
- **Payload:** `capability_delta`
- **Direction:** `client_to_server`
- **Purpose:** Prove display geometry changes are represented consistently.
- **Assertions:** device id, generation, reason, width, height, orientation, safe-area values.

### 4. `set_ui_basic_v1`

- **Message:** `terminals.control.v1.ConnectResponse`
- **Payload:** `set_ui`
- **Direction:** `server_to_client`
- **Purpose:** Prove server-driven UI trees preserve widget oneof cases across Go and Dart.
- **Assertions:** target device id, root node id, root widget case, child widget cases, button action, text value/style/color.

### 5. `input_ui_action_v1`

- **Message:** `terminals.control.v1.ConnectRequest`
- **Payload:** `input`
- **Direction:** `client_to_server`
- **Purpose:** Prove client UI action messages preserve component id, action, and value.
- **Assertions:** device id, component id, action, value.

### 6. `transport_hello_grpc_ws_v1`

- **Message:** `terminals.control.v1.WireEnvelope`
- **Payload:** `transport_hello`
- **Direction:** `transport`
- **Purpose:** Prove transport envelope and carrier negotiation fields are stable.
- **Assertions:** outer protocol version, sequence, inner protocol version, supported gRPC/WebSocket carriers, desired device id.

## Expanded Fixture Set

Phase 3 expands the corpus with these fixtures:

1. `update_ui_component_v1` — component patch semantics.
2. `transport_hello_ack_v1` — server negotiation ack fields.
3. `deprecated_register_device_v1` — parse-only coverage for deprecated registration payload.
4. `command_request_manual_start_v1` — command action/kind enums and selected argument keys.
5. `observation_with_artifact_v1` — edge observations with evidence artifacts.
6. `flow_start_basic_v1` — flow plan node/edge topology.
7. `control_error_protocol_violation_v1` — protocol error enum and message.
8. `notification_v1` — notification envelope distinct from command-result notifications.
9. `bug_report_basic_v1` — diagnostics bug-report payload.
10. `bug_report_ack_v1` — diagnostics bug-report acknowledgement payload.

Deprecated fixtures must be documented as parse-only. They prove old messages still decode; they do not authorize new code to emit deprecated payloads.

## Go Test Design

Add `terminal_server/internal/contracttest/contract_test.go`.

Responsibilities:

1. Load `api/testdata/contract/manifest.yaml`.
2. For each fixture, read `.binpb`.
3. Instantiate the expected generated Go message type.
4. Unmarshal binary protobuf into that message.
5. Assert the expected oneof payload case.
6. Load the expected assertion YAML file.
7. Evaluate assertions against the decoded message.
8. Re-marshal and decode again.
9. Re-run the same assertions against the second decoded message.
10. If `round_trip: byte_exact`, compare the re-marshaled bytes to the fixture bytes.

### Go Message Factory

Start with explicit mapping. Do not build a general-purpose reflection framework in Phase 1.

```go
func newMessage(typeName string) (proto.Message, error) {
    switch typeName {
    case "terminals.control.v1.ConnectRequest":
        return &controlv1.ConnectRequest{}, nil
    case "terminals.control.v1.ConnectResponse":
        return &controlv1.ConnectResponse{}, nil
    case "terminals.control.v1.WireEnvelope":
        return &controlv1.WireEnvelope{}, nil
    default:
        return nil, fmt.Errorf("unsupported contract message type %q", typeName)
    }
}
```

Phase 3 can extend this factory for direct `diagnostics/v1` fixtures if those are not wrapped in `ConnectRequest` or `ConnectResponse`.

### Go Assertion Strategy

Use explicit path traversal in Phase 1. Supported paths should be the small set used by fixtures, such as:

```text
hello.device_id
hello.identity.platform
capability_snapshot.capabilities.screen.width
set_ui.root.children[0].button.action
transport_hello.supported_carriers
```

Move to `protoreflect` only if explicit path support becomes difficult to maintain.

### Go Test Shape

```go
func TestContractGoldenFixtures(t *testing.T) {
    manifest := loadManifest(t)
    for _, fixture := range manifest.Fixtures {
        t.Run(fixture.ID, func(t *testing.T) {
            // decode, assert payload, assert fields, semantic round trip
        })
    }
}
```

## Dart Test Design

Add `terminal_client/test/contract/contract_golden_test.dart`.

Responsibilities mirror the Go test:

1. Load `../api/testdata/contract/manifest.yaml`.
2. For each fixture, read `.binpb`.
3. Instantiate the expected generated Dart message type.
4. Decode binary protobuf into that message.
5. Assert the expected oneof payload case.
6. Load the expected assertion YAML file.
7. Evaluate assertions against the decoded message.
8. Write the message back to bytes and decode again.
9. Re-run assertions.
10. If `round_trip: byte_exact`, compare bytes exactly.

### Dart Dev Dependencies

Add test-only dependencies if missing:

```yaml
dev_dependencies:
  path: any
  yaml: ^3.1.2
```

Use normal `dart:io` file reads in Flutter tests. Do not add production dependencies for fixture parsing.

### Dart Message Factory

Use explicit generated-type factories:

```dart
GeneratedMessage newMessage(String typeName) {
  switch (typeName) {
    case 'terminals.control.v1.ConnectRequest':
      return ConnectRequest();
    case 'terminals.control.v1.ConnectResponse':
      return ConnectResponse();
    case 'terminals.control.v1.WireEnvelope':
      return WireEnvelope();
    default:
      throw UnsupportedError('unsupported contract message type $typeName');
  }
}
```

### Dart Oneof Normalization

Dart generated oneof helpers may expose camelCase names, such as `setUi`, while the manifest uses proto snake_case names, such as `set_ui`. Add one helper that normalizes generated names before comparison.

Example expectation:

```dart
expect(normalizeOneofName(response.whichPayload().name), 'set_ui');
```

## Fixture Generation Workflow

Add `scripts/proto-contract-generate.sh` as the stable entry point:

```bash
#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$ROOT_DIR/terminal_server"
go run ./cmd/proto-contract-generate \
  --manifest ../api/testdata/contract/manifest.yaml \
  --root ..
```

Implement the generator as a small Go command:

```text
terminal_server/cmd/proto-contract-generate/main.go
```

The Go command should:

1. load `manifest.yaml`;
2. read each `.textproto` file;
3. instantiate the generated message type named by `message`;
4. parse textproto using `google.golang.org/protobuf/encoding/prototext`;
5. marshal binary protobuf;
6. write the `.binpb` file.

Do not use `protojson` for textproto parsing. `protojson` is for proto JSON, not `.textproto`.

## Verification Workflow

Use three Make targets with distinct meanings:

```makefile
.PHONY: proto-contract-generate proto-contract-test proto-contract-verify

proto-contract-generate:
	./scripts/proto-contract-generate.sh

proto-contract-test:
	./scripts/proto-contract-test.sh

proto-contract-verify: proto-contract-generate proto-contract-test
	git diff --exit-code -- api/testdata/contract
```

Add `scripts/proto-contract-test.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$ROOT_DIR/terminal_server"
go test ./internal/contracttest

cd "$ROOT_DIR/terminal_client"
flutter test test/contract/contract_golden_test.dart
```

Definitions:

- `proto-contract-generate` regenerates binary fixtures from textproto.
- `proto-contract-test` runs semantic tests only.
- `proto-contract-verify` regenerates, tests, and fails if generated fixtures are not committed.

Local `all-check` should include `proto-contract-test`, not `proto-contract-verify`, to avoid unexpected file writes during normal validation. CI should run `proto-contract-verify` for proto-impacting changes.

## CI Changes

### Proto CI

Keep the existing Buf job fast. Add a second job named `contract-golden` that depends on the Buf job.

The `contract-golden` job should:

1. check out the repository;
2. install Go using `terminal_server/go.mod`;
3. install Flutter stable;
4. restore Go/Pub/Flutter caches where practical;
5. run `make proto-contract-verify`.

Example sketch:

```yaml
  contract-golden:
    needs: proto
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - uses: actions/setup-go@v6
        with:
          go-version-file: terminal_server/go.mod
      - uses: subosito/flutter-action@v2
        with:
          channel: stable
      - name: Flutter pub get
        working-directory: terminal_client
        run: flutter pub get
      - name: Go-Dart contract golden tests
        run: make proto-contract-verify
```

### Server CI

The Go side should run automatically through `go test ./...` if `terminal_server/internal/contracttest` is inside the module and can find fixture paths.

If fixture path resolution is awkward, add an explicit step:

```yaml
      - name: Go contract golden tests
        run: go test ./internal/contracttest
```

### Client CI

The Dart side should run automatically through `flutter test --coverage` once the test lives under `terminal_client/test/`.

If runtime or fixture path resolution requires isolation, add:

```yaml
      - name: Dart contract golden tests
        run: flutter test test/contract/contract_golden_test.dart
```

### Workflow Triggers

Run the contract job for PRs touching:

- `api/**`
- `terminal_server/gen/**`
- `terminal_server/internal/transport/**`
- `terminal_server/internal/contracttest/**`
- `terminal_client/lib/connection/**`
- `terminal_client/lib/gen/**`
- `terminal_client/lib/ui/**`
- `terminal_client/test/contract/**`
- `scripts/proto-contract-*`
- `Makefile`

Once stable, make the contract job a required status check for protocol-impacting PRs.

## Developer Workflow

### When changing `.proto` files

1. Edit `.proto` files under `api/terminals/**`.
2. Run `make proto-generate`.
3. Add or update `.textproto` fixtures.
4. Run `make proto-contract-generate`.
5. Run `make proto-contract-test`.
6. Update `api/testdata/contract/README.md` if the fixture adds a new category.
7. Commit `.proto`, generated code, `.textproto`, `.binpb`, expected assertion YAML, and docs together.

### When changing Go or Dart protocol handling

Run `make proto-contract-test` if the change touches:

- envelope conversion;
- connection lifecycle;
- transport negotiation;
- capability lifecycle;
- server-driven UI decoding/rendering;
- generated proto usage;
- deprecated payload handling;
- map/string/JSON extension semantics.

## PR Checklist Addition

Add this to the PR template:

```markdown
### Protocol Contract

- [ ] Updated `.proto` files, if required.
- [ ] Ran `make proto-generate`, if `.proto` files changed.
- [ ] Added or updated Go-Dart golden fixtures for changed message semantics.
- [ ] Ran `make proto-contract-test`.
- [ ] Confirmed no scenario-specific behavior was added to the Flutter client.
- [ ] Documented any new map keys, string enum-like values, or JSON-in-string schemas.
- [ ] Updated compatibility notes if old clients or servers are affected.
```

## Acceptance Criteria

### Phase 0 — Path and Tooling Spike

- `api/testdata/contract/README.md` exists.
- Empty `manifest.yaml` loads in both Go and Dart tests.
- Go and Dart tests can resolve fixture paths from their normal working directories.
- The project has an agreed choice of YAML for manifest/assertion parsing.

### Phase 1 — Minimal Shared Corpus

- Six Phase 1 `.textproto` fixtures exist.
- Six Phase 1 `.binpb` fixtures exist.
- Six expected assertion YAML files exist.
- Go contract tests decode all six fixtures and assert key semantic fields.
- Dart contract tests decode all six fixtures and assert the same key semantic fields.
- `make proto-contract-test` runs both sides.
- A changed compatibility-critical field fails at least one Go test and one Dart test.

### Phase 2 — Generation and CI

- `proto-contract-generate` regenerates `.binpb` from `.textproto`.
- `proto-contract-verify` fails when `.textproto` and `.binpb` drift.
- Proto CI runs `make proto-contract-verify` for proto-impacting changes.
- Contract test runtime is within budget or documented as an accepted cost.

### Phase 3 — Coverage Expansion

- Expanded fixtures cover representative messages from these packages:
  - `control/v1`
  - `capabilities/v1`
  - `io/v1`
  - `ui/v1`
  - `diagnostics/v1`
- Deprecated register fixture is parse-only and documented as such.
- Assertion helpers support `equals`, `contains`, `length`, `present`, and `absent`.

### Phase 4 — Protocol Evolution Guardrails

- `docs/protocol-evolution.md` exists or the existing equivalent doc is updated.
- `docs/protocol-extension-registry.md` documents map keys, string enum-like values, and JSON-in-string schemas used by fixtures.
- PR template includes the protocol contract checklist.
- `make ci-local` includes `proto-contract-test`.

## Phased Implementation Plan

## Phase 0 — Confirm Generated Code and Fixture Paths

**Owner role:** Protocol owner

**Tasks:**

- Confirm generated Go import paths for `control`, `capabilities`, `io`, `ui`, and `diagnostics`.
- Confirm generated Dart import paths for the same packages.
- Confirm Go tests can read `../api/testdata/contract` when run from `terminal_server`.
- Confirm Dart tests can read `../api/testdata/contract` when run from `terminal_client`.
- Add `TERMINALS_CONTRACT_FIXTURE_ROOT` override for CI/debugging.

**Deliverables:**

- `api/testdata/contract/README.md` with path assumptions.
- Empty manifest loader tests in Go and Dart.

**Exit criteria:**

- `make proto-contract-test` passes with an empty manifest.

## Phase 1 — Build the Minimal Shared Corpus

**Owner role:** Protocol owner with Go and Flutter reviewers

**Tasks:**

- Add the six Phase 1 fixtures.
- Add expected assertion YAML files.
- Add Go fixture loader and explicit message factory.
- Add Dart fixture loader and explicit message factory.
- Implement `equals`, `contains`, and `length`.
- Add payload/oneof normalization helpers.
- Add `scripts/proto-contract-test.sh`.
- Add `make proto-contract-test`.

**Deliverables:**

- First working shared Go-Dart contract suite.

**Exit criteria:**

- Both languages decode the same six binary fixtures.
- Both languages fail on intentional fixture assertion mismatches.

## Phase 2 — Add Generation and CI Enforcement

**Owner role:** DevOps reviewer

**Tasks:**

- Add `terminal_server/cmd/proto-contract-generate`.
- Add `scripts/proto-contract-generate.sh`.
- Add `make proto-contract-generate` and `make proto-contract-verify`.
- Add a Proto CI `contract-golden` job.
- Add generated fixture diff check.
- Add CI caching if runtime is material.

**Deliverables:**

- Repeatable fixture generation.
- CI-enforced fixture freshness.

**Exit criteria:**

- CI fails when `.textproto` and `.binpb` drift.
- CI fails when Go or Dart semantic expectations drift.

## Phase 3 — Expand Protocol Coverage

**Owner role:** Protocol owner

**Tasks:**

- Add expanded fixture set.
- Add diagnostics fixture support.
- Add `present` and `absent` operators.
- Add negative fixtures only where they validate clear rejection behavior.
- Add fixture category documentation.

**Deliverables:**

- Contract corpus covers primary client/server traffic families.

**Exit criteria:**

- At least one fixture exists for every protobuf package used on the control stream.

## Phase 4 — Add Evolution Guardrails

**Owner role:** Architecture owner

**Tasks:**

- Add or update protocol evolution docs.
- Add protocol extension registry.
- Add PR checklist.
- Add a lightweight proto diff scanner that warns on new `map<`, `string kind`, `string type`, `*_json`, `metadata`, `arguments`, or `attributes` fields.

**Deliverables:**

- Review-time guardrails for untyped extension points.

**Exit criteria:**

- New untyped extension points are either converted to typed fields or documented with contract fixtures.

## Test Data Design Principles

### Keep fixtures realistic

Fixtures should resemble actual home devices and traffic: real-looking device ids, session ids, component ids, route ids, capability data, and UI trees.

### Keep assertions compatibility-focused

Assert fields that define behavior or compatibility. Do not snapshot every incidental field.

### Prefer semantic checks over byte equality

Use binary protobuf as the shared artifact, but assert semantics unless byte equality is explicitly safe.

### Avoid scenario-specific client assumptions

UI fixtures may include action strings and component ids. Dart tests should assert generic decode/render semantics, not scenario behavior.

### Treat deprecated fields intentionally

Deprecated fixtures verify parse compatibility only. They should never be used as examples for new emissions.

### Treat maps carefully

For maps, assert selected keys and values. Never rely on serialized map ordering.

## Relative Path Handling

Default Go fixture root:

```go
filepath.Join("..", "api", "testdata", "contract")
```

Default Dart fixture root:

```dart
path.normalize(path.join('..', 'api', 'testdata', 'contract'))
```

Override for both runtimes:

```text
TERMINALS_CONTRACT_FIXTURE_ROOT=/absolute/path/to/api/testdata/contract
```

## CI Runtime Budget

Targets:

- Go contract tests: under 5 seconds.
- Dart contract tests: under 15 seconds after dependencies are available.
- Full contract CI job: under 2 minutes with cache.

If runtime exceeds budget:

1. keep semantic tests in proto-impacting CI;
2. keep only a smoke subset in `all-check`;
3. split generation verification from semantic tests;
4. add cache before reducing coverage.

## Risks and Mitigations

| Risk | Severity | Mitigation |
|---|---:|---|
| Fixture generator becomes too complex | Medium | Start with a small Go generator using generated protos and `prototext`; avoid a general framework. |
| Tests become brittle due to map ordering | Medium | Use semantic assertions and avoid byte-exact checks for map-heavy fixtures. |
| Dart tests cannot find repo-root fixtures in CI | Medium | Add `TERMINALS_CONTRACT_FIXTURE_ROOT` and test path resolution in Phase 0. |
| Golden corpus grows stale | High | Require fixture updates in PR checklist and run `proto-contract-verify` in CI. |
| Developers add untyped fields without fixtures | High | Add extension registry and proto diff scanner warning. |
| Contract tests duplicate unit tests | Low | Keep tests cross-language and compatibility-focused only. |
| Deprecated fixture encourages deprecated emissions | Low | Mark deprecated fixtures parse-only in manifest and README. |

## Open Questions

1. Should expected assertions remain YAML?
   - Default answer: yes, because readability matters and YAML is already common for metadata-like files.
   - Revisit only if adding the Dart `yaml` dev dependency creates friction.

2. Should direct package fixtures be added, or should all fixtures be wrapped in `ConnectRequest`, `ConnectResponse`, or `WireEnvelope`?
   - Default answer: use envelopes for Phase 1 and direct package fixtures only when Phase 3 needs them.

3. Should proto JSON be tested?
   - Default answer: no for Phase 1.
   - Add proto JSON fixtures later only for HTTP fallback or admin APIs that expose proto JSON.

4. Should unknown-field preservation be tested?
   - Default answer: no for Phase 1.
   - Revisit when the project formalizes forward-compatibility policy.

5. Should malformed fixtures be included?
   - Default answer: not initially.
   - Add negative fixtures later for explicit rejection behavior, such as unsupported protocol versions.

## Suggested PR Breakdown

### PR 1 — Contract Test Skeleton

- Add `api/testdata/contract/README.md`.
- Add empty `manifest.yaml`.
- Add Go empty-manifest loader test.
- Add Dart empty-manifest loader test.
- Add `scripts/proto-contract-test.sh`.
- Add `make proto-contract-test`.

### PR 2 — First Six Fixtures

- Add six Phase 1 `.textproto` and `.binpb` files.
- Add six expected assertion YAML files.
- Implement assertion helpers.
- Verify Go and Dart tests pass.

### PR 3 — Fixture Generation

- Add Go generator command.
- Add `scripts/proto-contract-generate.sh`.
- Add `make proto-contract-generate` and `make proto-contract-verify`.
- Verify fixture generation is repeatable.

### PR 4 — CI Integration

- Add Proto CI `contract-golden` job.
- Add caching where useful.
- Run `make proto-contract-verify` in CI.

### PR 5 — Expand Corpus and Guardrails

- Add expanded fixture set.
- Add diagnostics fixture support.
- Add protocol evolution docs and extension registry.
- Add PR checklist and proto diff scanner reminder.

## Success Metrics

- Every proto-impacting PR either adds/updates a fixture or explains why no fixture is needed.
- Go and Dart contract suites run in CI for proto-impacting changes.
- A stale `.binpb` fixture is caught before merge.
- A semantic change to oneof handling, enum values, map keys, capability defaults, or transport negotiation fails at least one contract test.
- New contributors and agents can add a fixture by following `api/testdata/contract/README.md` without ad hoc instructions.

## Related Files

- `api/terminals/control/v1/control.proto`
- `api/terminals/capabilities/v1/capabilities.proto`
- `api/terminals/io/v1/io.proto`
- `api/terminals/ui/v1/ui.proto`
- `api/terminals/diagnostics/v1/diagnostics.proto`
- `api/buf.yaml`
- `api/buf.gen.yaml`
- `terminal_server/internal/transport/`
- `terminal_client/lib/connection/`
- `terminal_client/lib/gen/`
- `terminal_client/lib/ui/`
- `.github/workflows/proto-ci.yml`
- `Makefile`

## Done Definition

This plan is done when the repository has a CI-enforced, shared Go-Dart golden-message suite that decodes representative protobuf control-stream fixtures in both runtimes, asserts the same compatibility-critical behavior, verifies generated binary fixtures are current, and documents the workflow for updating fixtures as the protocol evolves.
