# Development Environment Improvement Log

This log captures friction noticed while implementing the kitchen timer and the
repo changes that resolved it.

## 2026-04-24 Kitchen Timer

- `docs/tal-example-kitchen-timer.md` describes a TAL package contract, but the current runtime only loads manifests and stubs activations. It would be easier to implement examples if docs clearly marked which parts are aspirational versus executable today.
- `make usecase-validate USECASE=T1` already existed even though the kitchen timer example called for richer behavior. A small "coverage depth" note in the validation matrix could distinguish smoke coverage from full example parity.
- Timer scheduler entries are opaque string keys, so adding label/duration metadata required key encoding and backward-compatible parsing. A structured scheduler payload would make app development less brittle.
- Scenario starts can broadcast notifications, but they cannot directly return typed UI/TTS/bus operations. That makes TAL-style all-or-nothing operation commits hard to model in the current Go scenario layer.

## 2026-04-25 Completed Improvements

The development-environment improvement plan has been completed and drained
from `plans/development-environment-improvement.md`. The durable behavior is
documented in the guides below.

### TAL Runtime Status Documentation

- `docs/tal-example-kitchen-timer.md` now includes an `Implementation Status`
  table that distinguishes the executable Go-side `TimerReminderScenario`, the
  loadable `terminal_server/apps/kitchen_timer` package, and the planned TAL
  interpretation path.
- `terminal_server/apps/kitchen_timer/README.md` documents the package as a
  TAL/TAR contract example and records that `go run ./cmd/term app test
  kitchen_timer` is a package smoke test, not a synthetic lifecycle simulation.
- `plans/application-runtime.md` defines the `Implemented`, `Partially
  implemented`, and `Planned` status vocabulary and includes a runtime status
  table for package loading, manifest validation, exported definitions, TAL
  interpretation, lifecycle snapshots, host operation commit, and simulation.
- `scripts/test-development-environment-docs.sh` is wired into `make
  all-check` through the `development-docs-test` target and fails if TAL
  example docs omit `## Implementation Status`.

### Use-Case Validation Coverage Depth

- `docs/usecase-validation-matrix.md` now has a `Coverage Depth` column and the
  standard labels `Smoke`, `Transport`, `Scenario`, `Contract`, `Simulation`,
  and `Full`.
- Every automated use-case row has a current, non-aspirational depth label.
  `T1` is explicitly marked `Smoke`, with evidence split across the due-timer
  loop, transport `run_due_timers`, and the kitchen timer package smoke test.
- `scripts/usecase-validate.sh` supports metadata output with either
  `make usecase-validate USECASE=T1 INFO=1` or
  `scripts/usecase-validate.sh --info T1`.
- `scripts/test-development-environment-docs.sh` verifies that every ID mapped
  by `scripts/usecase-validate.sh` has a matrix row and a valid coverage depth.

### Structured Scheduler Records

- `terminal_server/internal/storage.MemoryScheduler` stores
  `storage.ScheduleRecord` values with typed fields for key, kind, subject,
  device ID, trigger time, payload, and creation time.
- The legacy `Schedule(ctx, key, unixMS)` and `Due(unixMS)` methods remain
  compatible; key-only callers are stored as records and known key prefixes
  infer a schedule kind.
- Structured callers can use `ScheduleRecord` and `DueRecords`; returned
  records clone payload maps so callers cannot mutate scheduler state.
- `TimerReminderScenario` schedules timers through a `scheduler.after`
  operation that records `Kind: "timer"`, source device, label, and
  `duration_seconds` metadata when the scheduler supports structured records.
- `Runtime.ProcessDueTimers` prefers `DueRecords` and falls back to legacy key
  parsing, so existing scheduled keys still fire and are removed.
- Pending timer reporting uses structured timer fields when available, while
  preserving legacy key display behavior.

### Result-Returning Scenario Operations

- `terminal_server/internal/scenario` defines `Operation`, `ScenarioResult`,
  and `ResultScenario` so scenarios can return typed side effects instead of
  performing them directly in `Start`.
- `ExecuteOperations` validates all operations before committing any side
  effects, executes them in order, and emits `scenario.ops.validated`,
  `scenario.ops.committed`, and `scenario.ops.failed` eventlog entries.
- The executable operation set currently includes `scheduler.after`,
  `scheduler.cancel`, `broadcast.notify`, `ai.tts`, and `bus.emit`. UI and flow
  operation constants are defined but rejected by validation until executors
  exist for those host capabilities.
- `Engine.ActivateMatched` detects `ResultScenario` implementations, commits
  returned operations, emits returned triggers, and leaves legacy scenarios on
  their existing `Start` path.
- `TimerReminderScenario` is the first built-in result-returning scenario. Its
  legacy `Start` method delegates to `StartResult` and the shared executor.
- `ResultFromAppRuntime` adapts `appruntime.Result` values into
  `scenario.ScenarioResult`, giving TAL activations and Go scenarios a shared
  operation model once TAL interpretation is active.

## 2026-04-25 Kitchen Timer Completion Pass

- Go tests may try to write under the user-level Go build cache, which is
  outside the workspace sandbox. Running focused tests with
  `GOCACHE=/tmp/terminals-go-build` avoids the permission failure; a make-level
  default would make this smoother for app-development loops.
- Scenario-authored UI now has an in-memory host and transport polling bridge,
  but the response-only stream shape means updates are delivered on command or
  heartbeat turns rather than pushed independently from the housekeeping loop.
- The current scheduler models 1 Hz timer ticks as structured one-shot records.
  That keeps the implementation small, but a first-class recurring scheduler
  operation would better match the TAL `scheduler.every` contract.
- Timer cancellation currently finds all pending timer/tick records for a
  source device by scanning scheduler records. An indexed scheduler query by
  kind/device would make cancellation and future multi-timer management cleaner.
