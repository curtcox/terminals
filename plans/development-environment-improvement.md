# Development Environment Improvement Plan

This plan addresses the issues recorded in
[`docs/development-environment-improvement-log.md`](../docs/development-environment-improvement-log.md).
It is scoped to making server-side app development less surprising, more
testable, and easier to validate without weakening the core architecture:
clients remain generic terminals, protobuf remains the client/server contract,
and scenario/app behavior stays in the Go server or TAL runtime.

## Goals

- Make documentation clearly distinguish implemented behavior from planned
  TAL/TAR behavior.
- Make use-case validation communicate coverage depth, not just pass/fail.
- Replace opaque scheduler string keys with structured scheduled records.
- Let scenario/app starts return typed operations so Go scenarios and TAL apps
  share a more consistent commit model.

## Non-Goals

- Do not add scenario-specific Flutter client behavior.
- Do not introduce ad-hoc JSON payloads on the control plane.
- Do not replace the existing scenario engine in one large rewrite.
- Do not require every built-in scenario to migrate before the first structured
  scheduler and operation-returning paths ship.

## Issue 1: TAL Docs Outrun The Runtime

### Problem

`docs/tal-example-kitchen-timer.md` describes a TAL package that returns
host operations from lifecycle hooks, but the current runtime mostly validates
manifests, registers exported app definitions, and stubs activation lifecycle
methods. Developers can reasonably infer the example is executable end-to-end
when only part of that contract exists today.

### Target State

Every TAL/TAR doc and example should clearly identify:

- executable today,
- runtime-loaded but not interpreted yet,
- planned contract,
- closest current Go-side implementation path.

The kitchen timer example should remain useful as a contract example, but it
should not be mistaken for a fully interpreted TAL app until the interpreter and
host operation bridge exist.

### Work Plan

1. Add a short status block to `docs/tal-example-kitchen-timer.md`:
   - current executable path: `TimerReminderScenario`,
   - package-load path: `terminal_server/apps/kitchen_timer`,
   - planned TAL interpretation path: `plans/application-runtime.md`.
2. Add a status vocabulary to `plans/application-runtime.md`:
   - `Implemented`,
   - `Partially implemented`,
   - `Planned`.
3. Add a small runtime status table to `plans/application-runtime.md` covering:
   - package loading,
   - manifest validation,
   - exported definitions,
   - TAL parsing/interpretation,
   - lifecycle state snapshots,
   - host operation commit,
   - simulation harness.
4. Update `terminal_server/apps/kitchen_timer/README.md` or add one if absent:
   - explain that `main.tal` mirrors the contract,
   - point to the Go scenario implementation used today,
   - list the test command currently supported by `term app test`.
5. Add a doc check in CI or `make all-check` that fails when TAL example docs
   omit a status block. A simple first version can be a script that scans
   `docs/tal-example-*.md` for `## Implementation Status`.

### Acceptance Criteria

- A new developer can tell from the kitchen timer docs which behavior runs now.
- `term app test kitchen_timer` is documented as a package smoke test, not a
  lifecycle simulation.
- `plans/application-runtime.md` no longer reads as entirely implemented.
- The doc check catches new TAL examples without implementation status.

## Issue 2: Use-Case Validation Lacks Coverage Depth

### Problem

`make usecase-validate USECASE=T1` already existed before the kitchen timer
matched the richer example. The validation matrix said T1 was automated, but it
did not say whether that meant smoke coverage, transport coverage, or full
example parity.

### Target State

Use-case validation should remain fast and simple, but the matrix should expose
coverage depth. A passing use case can be marked as a smoke test, integration
test, contract test, simulation, or full scenario validation.

### Work Plan

1. Add a `Coverage Depth` column to
   `docs/usecase-validation-matrix.md`.
2. Define standard depth labels:
   - `Smoke`: proves a narrow server loop or command path.
   - `Transport`: proves generated/wire control-plane behavior.
   - `Scenario`: proves scenario matching and server-side side effects.
   - `Contract`: proves app/package/runtime contract surfaces.
   - `Simulation`: proves lifecycle behavior against synthetic time/events.
   - `Full`: covers trigger, placement, UI, scheduling, side effects, and
     expiry/cancel/resume behavior.
3. Update all automated rows with the best current label, not an aspirational
   label.
4. Extend `scripts/usecase-validate.sh` with an optional metadata command:
   - `make usecase-validate USECASE=T1 INFO=1`, or
   - `scripts/usecase-validate.sh --info T1`.
5. Add a matrix consistency test that verifies every ID in the script has a row
   and a non-empty coverage depth.
6. For T1 specifically, split evidence into:
   - due timer loop,
   - transport `run_due_timers`,
   - kitchen timer package smoke test,
   - future simulation coverage once TAL simulation exists.

### Acceptance Criteria

- T1 no longer appears equivalent to a fully interpreted TAL kitchen timer.
- Automated IDs list their validation depth in a consistent vocabulary.
- The validation script and matrix cannot silently drift.
- `USECASE=all` remains as easy to run as it is today.

## Issue 3: Scheduler Entries Are Opaque String Keys

### Problem

Timers currently encode metadata into scheduler keys such as
`timer:<device>:<timestamp>[:duration][:label]`. This is brittle: adding a
label or duration requires escaping, parsing, and backward compatibility logic.
It also makes querying, migration, and app-level scheduler semantics harder than
they need to be.

### Target State

The scheduler should store structured records with typed fields and optional
metadata while preserving compatibility with existing key-only callers.

### Proposed API

```go
type ScheduleRecord struct {
    Key       string
    Kind      string
    Subject   string
    DeviceID  string
    UnixMS    int64
    Payload   map[string]string
    CreatedMS int64
}

type Scheduler interface {
    Schedule(ctx context.Context, key string, unixMS int64) error
    ScheduleRecord(ctx context.Context, record ScheduleRecord) error
    Due(unixMS int64) []string
    DueRecords(unixMS int64) []ScheduleRecord
    Remove(ctx context.Context, key string) error
}
```

The existing methods stay until every caller has migrated.

### Work Plan

1. Add `ScheduleRecord`, `ScheduleRecord` storage, and `DueRecords` to
   `internal/storage.MemoryScheduler`.
2. Keep `Schedule` as a compatibility wrapper that writes a record with:
   - `Key`,
   - `UnixMS`,
   - inferred `Kind` when the key starts with a known prefix.
3. Update `scenario.Scheduler` to include optional structured methods through
   a narrow extension interface first, so existing test fakes do not break.
4. Migrate `TimerReminderScenario` to call `ScheduleRecord` when available:
   - `Kind: "timer"`,
   - `Subject: label`,
   - `DeviceID: source device`,
   - payload `duration_seconds`.
5. Migrate `Runtime.ProcessDueTimers` to prefer `DueRecords` and fall back to
   parsing legacy keys.
6. Update system `pending_timers` output to display structured fields where
   available.
7. Add storage tests for:
   - key-only compatibility,
   - structured record round trip,
   - deterministic due ordering,
   - removal by key,
   - legacy timer key fallback.
8. After all internal callers migrate, decide whether to keep key-only methods
   permanently as the minimal scheduler API or mark them deprecated in comments.

### Acceptance Criteria

- New timer metadata no longer requires key encoding.
- Existing scheduled keys still fire and are removed.
- `pending_timers` can show meaningful timer label/duration metadata.
- T1 validation still passes during and after migration.

## Issue 4: Scenario Starts Cannot Return Typed Operations

### Problem

TAL lifecycle hooks return a `Result` containing state, operations, emitted
triggers, and done status. Go scenarios currently perform side effects directly
inside `Start` and usually communicate with clients by calling broadcaster or
router interfaces. This makes all-or-nothing TAL-style commits hard to model,
harder to test, and different from built-in scenarios.

### Target State

Built-in Go scenarios and TAL app activations should converge on a shared
operation result model without forcing an immediate rewrite of every scenario.

### Proposed API

```go
type Operation struct {
    Kind   string
    Target string
    Args   map[string]string
}

type ScenarioResult struct {
    State any
    Ops   []Operation
    Emit  []Trigger
    Done  bool
}

type ResultScenario interface {
    Scenario
    StartResult(ctx context.Context, env *Environment) (ScenarioResult, error)
}
```

The engine can detect `ResultScenario`, validate/commit its operations through
a new operation executor, and otherwise keep using legacy `Start`.

### Work Plan

1. Define operation kinds that match existing host capabilities:
   - `ui.set`,
   - `ui.patch`,
   - `ui.transition`,
   - `scheduler.after`,
   - `scheduler.cancel`,
   - `ai.tts`,
   - `bus.emit`,
   - `broadcast.notify`,
   - `flow.apply`,
   - `flow.stop`.
2. Add an operation validator/executor in `internal/scenario` or a small
   `internal/appruntime/hostops` package.
3. Start with operation execution that is deterministic and testable:
   - validate every op first,
   - execute in order only after validation succeeds,
   - return the first execution error,
   - avoid persisting state until commit succeeds.
4. Add `ResultScenario` support to `Engine.ActivateMatched`.
5. Convert `TimerReminderScenario` first:
   - start returns scheduler + notification ops,
   - due processing can be modeled as scheduler-triggered result handling in a
     later step.
6. Add bridge code from `appruntime.Result` to `scenario.ScenarioResult` so TAL
   activations and Go scenarios use the same executor.
7. Add tests for:
   - successful multi-op commit,
   - validation failure prevents all side effects,
   - scheduler op plus broadcast op,
   - TTS op plus bus emit,
   - legacy scenarios still activate normally.
8. Add eventlog entries around operation validation and commit:
   - `scenario.ops.validated`,
   - `scenario.ops.committed`,
   - `scenario.ops.failed`.

### Acceptance Criteria

- At least one built-in scenario uses `ScenarioResult`.
- Legacy scenarios still pass existing validation.
- A failed operation validation produces no partial side effects.
- TAL result execution can reuse the same operation executor.
- The kitchen timer behavior can be expressed without hand-coding every side
  effect directly in `Start`.

## Implementation Sequence

1. Documentation status and matrix depth
   - low risk,
   - improves developer expectations immediately,
   - no runtime behavior changes.
2. Structured scheduler records
   - medium risk,
   - directly removes the key-encoding friction seen in T1,
   - can ship behind compatibility methods.
3. Operation result executor
   - higher risk,
   - touches scenario activation semantics,
   - should start with one opt-in scenario.
4. TAL runtime bridge
   - builds on the operation executor,
   - should not start until operation validation/commit is covered by tests.
5. Simulation depth for kitchen timer
   - depends on enough TAL/runtime bridge support to drive synthetic ticks and
     expiry events without real wall-clock time.

## Suggested Milestones

### Milestone 1: Honest Docs

- Add implementation status blocks to TAL examples.
- Add coverage depth to the validation matrix.
- Add script/matrix consistency checks.

Exit criteria: `make all-check` passes and T1 has an explicit depth label.

### Milestone 2: Structured Timers

- Add structured scheduler records.
- Migrate T1 scheduling and due processing.
- Preserve legacy timer key compatibility.

Exit criteria: `make usecase-validate USECASE=T1` passes, and storage tests
cover both record and legacy key paths.

### Milestone 3: Result-Returning Scenario Pilot

- Add operation result types and executor.
- Convert kitchen timer or another narrow scenario to opt in.
- Prove no partial side effects on validation failure.

Exit criteria: one built-in scenario uses `ScenarioResult`, and legacy scenario
tests are unchanged.

### Milestone 4: TAL Execution Bridge

- Translate TAL results into scenario operations.
- Add simulation support for synthetic scheduler events.
- Upgrade `terminal_server/apps/kitchen_timer/tests/kitchen_timer_test.tal` from
  a smoke test to a lifecycle test.

Exit criteria: the kitchen timer TAL test can assert expiry TTS, UI patch, and
`timer.expired` emission without real time.

## Validation Plan

Run these gates as the work progresses:

```bash
go test ./internal/storage ./internal/scenario ./internal/transport -count=1
go run ./cmd/term app test kitchen_timer
make usecase-validate USECASE=T1
make all-check
```

When the validation matrix gains coverage-depth consistency checks, include the
new check in `make all-check` so drift is caught before merge.

## Open Questions

- Should structured schedule payloads become protobuf-backed records before
  persistence moves beyond memory storage?
- Should `broadcast.notify` remain a scenario operation or be folded into
  `ui.notification` for tighter client semantics?
- Should the operation executor live in `internal/scenario` or
  `internal/appruntime` once TAL interpretation is active?
- How much simulation syntax should `term app test` support before a separate
  `term sim run` command is required?
