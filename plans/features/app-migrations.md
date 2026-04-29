---
title: "Application Migrations"
kind: plan
status: building
owner: github-copilot
validation: none
last-reviewed: 2026-04-29
---

# Application Migrations

See [masterplan.md](../archive/masterplan-duplicate.md) for overall system context.
Extends [application-runtime.md](application-runtime.md)
(hot reload, per-activation version pinning, optional
`migrate(from_version, state)` export) and
[shared-artifacts.md](shared-artifacts.md) (durable artifacts).
Referenced by [application-distribution.md](application-distribution.md)
(upgrade lifecycle, Install Transaction) and
[signing-and-trust.md](signing-and-trust.md) (app_id lineage,
who may authorize a migration).

## Implementation Progress

- 2026-04-29: Tightened migration manifest limit validation for
  `[migrate].drain_timeout_seconds` in both package verification
  and runtime migration plan parsing. Explicit non-positive drain
  timeouts now fail Gate 1 with a specific diagnostic or leave the
  runtime executor disabled with the same `last_error`, instead of
  silently falling back to the default timeout. Added regression
  coverage in `terminal_server/internal/apppackage/tap_test.go`
  (`TestVerifyTapRejectsMigrateNonPositiveDrainTimeoutSeconds`)
  and `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeMigrationInvalidLimitsDisableExecutor`), and
  documented the guard in `docs/application-migrations.md`.

- 2026-04-29: Added explicit runtime regression coverage for
  key-rotation upgrade semantics in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeReloadMigrationAfterKeyRotationUsesAppIDAndPendingVersionWindow`).
  Reloaded packages that keep the same lineage `app_id` but change
  author-key metadata now preserve app-ID-scoped migration journal
  paths while computing pending migration work from the installed
  package version, so a version `2` to `3` upgrade runs only the
  `2 -> 3` step rather than replaying `1 -> 2`. Documented the
  behavior in `docs/application-migrations.md`.

- 2026-04-29: Aligned runtime fixture execution for
  `migrate.env.abort(reason)` with the migration module alias
  contract in `terminal_server/internal/appruntime/runtime.go`.
  Fixture replay now recognizes abort calls through any loaded
  `abort = "<alias>"` binding in both direct `migrate(record)`
  scripts and the worked-example paged store-loop subset, failing
  the current step with `ErrMigrationAborted` and preserving
  `step_failed_aborted` evidence. Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationAbortCallFailsCurrentStep`,
  `TestRuntimeMigrationFixtureParsesAbortAliases`) and documented
  the behavior in `docs/application-migrations.md`.

- 2026-04-29: Added accepted `artifact.self.patch(...)` declaration
  journaling in `terminal_server/internal/appruntime/runtime.go`.
  Runtime retry now parses lineage-validated patch calls into host
  effect evidence and emits `artifact_patch_planned` journal entries
  with artifact ID, owner app ID, effect sequence, step, version edge,
  and script metadata before committing the step. Added regression
  coverage in `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationJournalsAcceptedArtifactPatchDeclarations`)
  and documented the behavior in `docs/application-migrations.md`.

- 2026-04-29: Added runtime baseline-abort reconciliation handling in
  `terminal_server/internal/appruntime/runtime.go`. `AbortMigration`
  now scans migration journal evidence for unresolved
  `artifact_inverse_failed` entries before reporting a baseline abort
  as complete; if any remain, it rewinds step progress to baseline,
  returns `ErrMigrationReconcilePending`, preserves sorted pending
  reconciliation records/status, emits `reconcile_pending` journal
  evidence, and replays that state after restart. Added regression
  coverage in `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeAbortBaselineEntersReconcilePendingWhenArtifactInverseFails`)
  and documented the behavior in `docs/application-migrations.md`.

- 2026-04-29: Extended the runtime fixture-backed deterministic
  migration subset in `terminal_server/internal/appruntime/runtime.go`
  to support loaded `store.delete` aliases in the worked-example
  paged store-loop shape. Fixture replay now accepts `delete(key)`,
  removes matching fixture records from expected output, and counts
  successful deletes as synthetic store effects for checkpoint
  evidence and hard-cap accounting. Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationAppliesStoreDeleteFixtureEffects`) and
  documented the expanded subset in `docs/application-migrations.md`.

- 2026-04-28: Extended the runtime fixture-backed deterministic
  migration subset in `terminal_server/internal/appruntime/runtime.go`
  to support the worked-example paged `store` loop shape with loaded
  `list_keys`/`get`/`put` aliases, literal-prefix fixture scans,
  `rec = get(key)`, idempotent presence guards, `_normalize(...)`
  assignments, `put(key, rec)`, and no-op checkpoint/log calls.
  Fixture replay now applies these store-loop transforms only to
  matching fixture keys and counts successful `put` calls as synthetic
  store effects for checkpoint evidence and resource caps. Added
  regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationAppliesPagedStoreFixtureTransforms`) and
  documented the expanded subset in `docs/application-migrations.md`.

- 2026-04-28: Extended the runtime fixture-backed deterministic
  migration subset in `terminal_server/internal/appruntime/runtime.go`
  to support idempotent `if "field" in record: continue` guards.
  Gate 4 fixture replay can now prove the worked-example pattern
  where already-migrated rows are left untouched while missing rows
  receive normalized defaults. Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationAppliesIdempotentFixtureGuard`) and
  documented the expanded subset in `docs/application-migrations.md`.

- 2026-04-28: Extended Gate 4 migration crash-replay coverage to
  include fixture-backed `checkpoint_committed` journal boundaries.
  Runtime checkpoint journaling in
  `terminal_server/internal/appruntime/runtime.go` is now
  interruptible through the same migration hook used for retry and
  step boundaries, and dry-run boundary discovery includes checkpoint
  commits only for steps whose fixture execution actually emits
  checkpoint evidence. Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeDryRunMigrationJournalReplayExercisesAllBoundaries`)
  and documented the behavior in `docs/application-migrations.md`.

- 2026-04-28: Added runtime migration hard-cap enforcement for
  the current execution scaffold in
  `terminal_server/internal/appruntime/runtime.go`.
  Fixture-backed store effects now track synthetic store-op count
  and write volume against the plan's per-step 1,000,000-op and
  100 MB hard caps, while declared `artifact.self.patch(...)`
  calls are counted against the 10,000-patch cap before step
  execution starts. Cap violations now fail the current step with
  `ErrMigrationResourceLimit`, preserve checkpoint progress, and
  emit `step_failed_resource_limit` journal evidence. Added
  regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationRejectsArtifactPatchHardCap`,
  `TestRuntimeMigrationResourceLimitValidation`) and documented the
  behavior in `docs/application-migrations.md`.

- 2026-04-28: Expanded the runtime fixture-backed deterministic
  migration subset in `terminal_server/internal/appruntime/runtime.go`
  to support `record.get("field", "default")` defaults in direct
  assignment, `trim(...)`, `lower(...)`, and
  `lower(trim(...))` fixture transforms. This lets Gate 4 fixture
  replay cover idempotent/default-read normalization patterns from
  the worked example without broadening the runtime into a general
  TAL interpreter. Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationAppliesRecordGetFixtureTransforms`) and
  documented the expanded subset in
  `docs/application-migrations.md`.

- 2026-04-28: Aligned runtime migration timeout enforcement with
  the plan's per-step budget semantics in
  `terminal_server/internal/appruntime/runtime.go`.
  `RetryMigration` now starts a fresh
  `[migrate].max_runtime_seconds` timer for each pending step
  instead of sharing one run-wide timer across the whole retry,
  so multi-step migrations whose individual steps stay under the
  configured budget can complete. Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationAppliesMaxRuntimePerStep`) and
  documented the per-step timeout boundary in
  `docs/application-migrations.md`.

- 2026-04-28: Extended the runtime fixture-backed deterministic
  migration subset in `terminal_server/internal/appruntime/runtime.go`
  to treat structured `log` calls as no-op execution statements when
  they are reached through aliases loaded from the allowed `log`
  migration module. This lets Gate 4 fixture replay accept the
  worked-example pattern of transforming records and emitting
  `info(...)` without granting side effects outside the migration log
  surface. Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationAllowsLogCallsInFixtureTransforms`) and
  documented the expanded subset in `docs/application-migrations.md`.

- 2026-04-28: Hardened runtime host-layer validation for
  `artifact.self.patch(...)` migration effects that omit
  ownership evidence. Retry now rejects patch calls without
  `owner_app_id` before step start with
  `ErrMigrationArtifactOwnership`, marks the step failed, and
  emits `step_failed_host_rejected` journal evidence instead of
  allowing an unverifiable artifact patch through the migration
  scaffold. Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationRejectsArtifactPatchWithoutOwnerAppID`)
  and documented the behavior in `docs/application-migrations.md`.

- 2026-04-28: Extended the runtime fixture-backed deterministic
  migration subset in `terminal_server/internal/appruntime/runtime.go`
  to support `trim(record["field"])` and
  `lower(trim(record["field"]))` assignments. This lets Gate 4
  fixture replay cover the label-normalization pattern used by the
  worked example while still rejecting unsupported statements
  explicitly. Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationAppliesTrimFixtureTransforms`) and
  documented the expanded subset in `docs/application-migrations.md`.

- 2026-04-28: Added runtime host-layer lineage validation for
  declared `artifact.self.patch(...)` migration effects in
  `terminal_server/internal/appruntime/runtime.go`. Retry now
  rejects patch calls whose script-provided `owner_app_id` differs
  from the migrating package `app_id`, including the same manifest
  name under a different lineage, marks the step failed, and emits
  `step_failed_host_rejected` journal evidence with the structured
  ownership error. Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationRejectsArtifactPatchForDifferentLineage`)
  and documented the behavior in `docs/application-migrations.md`.

- 2026-04-28: Added first-class `migrate.env.abort(reason)`
  handling to the runtime fixture execution scaffold in
  `terminal_server/internal/appruntime/runtime.go`. Abort calls
  now fail the current step with `ErrMigrationAborted`, preserve
  checkpoint progress, mark `verdict = step_failed`, and emit
  `step_failed_aborted` journal evidence with the script-provided
  reason instead of being reported as an unsupported fixture
  statement. Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationAbortCallFailsCurrentStep`) and
  documented the behavior in `docs/application-migrations.md`.

- 2026-04-28: Wired `[migrate].checkpoint_every` into the
  runtime migration retry scaffold in
  `terminal_server/internal/appruntime/runtime.go`. Fixture-backed
  deterministic migrations now count transformed fixture records as
  synthetic store effects and emit `checkpoint_committed` journal
  evidence at the configured cadence before step commit. Added
  regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationEmitsCheckpointEveryForFixtureEffects`)
  and documented the current checkpoint-evidence behavior in
  `docs/application-migrations.md`.

- 2026-04-28: Made Gate 4 load-time dry-run enforcement
  conservatively block `drain_policy = "multi_version"` migration
  steps until read-adapter replay validation exists. This prevents
  packages from passing the dry-run gate without proving the
  backward-read contract required by the plan. Added regression
  coverage in `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeLoadPackageRejectsMultiVersionWithoutReadAdapterDuringDryRunGate`)
  and documented the current block in `docs/application-migrations.md`.

- 2026-04-28: Added a deterministic execution subset for runtime
  migration fixture dry-runs in
  `terminal_server/internal/appruntime/runtime.go`. Fixture-backed
  `migrate(record)` scripts can now prove real seeded-record
  transformations before step commit, covering field copy,
  `lower(record["field"])`, JSON literal assignment, and field
  deletion while leaving unsupported statements as explicit
  `ErrMigrationFixtureMismatch` failures. Added regression coverage
  in `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationAppliesFixtureTransforms`) and
  documented the shipped subset in `docs/application-migrations.md`.

- 2026-04-28: Enforced `[migrate].max_runtime_seconds` during
  runtime migration retry in
  `terminal_server/internal/appruntime/runtime.go`. Retry now tracks
  the run budget around the current execution scaffold, fails with
  `ErrMigrationRuntimeTimeout` when the budget elapses, preserves
  committed checkpoint progress, marks `verdict = step_failed`, and
  emits `step_failed_timeout` journal evidence with the configured
  budget. Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationFailsWhenMaxRuntimeExceeded`) and
  documented the behavior in `docs/application-migrations.md`.

- 2026-04-28: Added Gate 4 dry-run plan validation for drain
  declarations in `terminal_server/internal/appruntime/runtime.go`.
  Load-time dry-run enforcement now rejects migration steps that
  declare `drain_policy = "drain"` without
  `compatibility = "incompatible"`, returning
  `ErrMigrationDryRunFailed` before registration. Added regression
  coverage in `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeLoadPackageRejectsDrainPolicyWithoutIncompatibleDuringDryRunGate`)
  and documented the Gate 4 block in `docs/application-migrations.md`.

- 2026-04-28: Tightened runtime Gate 4 fixture mismatch evidence in
  `terminal_server/internal/appruntime/runtime.go`. Fixture value
  mismatches now report the first divergent key with canonical
  `expected` and `actual` JSON bytes in the
  `step_failed_fixture_mismatch` journal entry, making dry-run
  failures actionable for package authors. Added regression coverage
  in `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationFailsWhenFixtureExpectedMismatch`) and
  documented the evidence shape in `docs/application-migrations.md`.

- 2026-04-28: Added positive-value manifest limit validation for
  `migrate.max_runtime_seconds` and `migrate.checkpoint_every`
  in both Gate 1 package verification
  (`terminal_server/internal/apppackage/tap.go`) and runtime
  migration plan parsing (`terminal_server/internal/appruntime/runtime.go`).
  Non-positive values now fail with explicit diagnostics and leave
  runtime migration status with `executor_ready = false` for
  operator visibility. Added regression coverage in
  `terminal_server/internal/apppackage/tap_test.go`
  (`TestVerifyTapRejectsMigrateNonPositiveMaxRuntimeSeconds`,
  `TestVerifyTapRejectsMigrateNonPositiveCheckpointEvery`) and
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeMigrationInvalidLimitsDisableExecutor`).

- 2026-04-28: Baseline-aware migration retries now start from the
  installed-version boundary after reload in
  `terminal_server/internal/appruntime/runtime.go`.
  `newMigrationState` now computes initial `steps_completed` from the
  previously installed package version when available, so upgrades that
  carry historical edges (for example `0001_1_to_2.tal` and
  `0002_2_to_3.tal`) resume at the pending edge instead of replaying
  already-applied steps. Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeReloadMigrationStateStartsFromInstalledVersion`) and
  documented behavior in `docs/application-migrations.md`.

- 2026-04-28: Wired Gate 4 migration replay enforcement into local
  `term` app-runtime flows in `terminal_server/cmd/term/main.go`.
  Added `newAppRuntime()` to enable
  `SetMigrationDryRunGateEnabled(true)` for `app check`, `app load`,
  `app test`, local `app reload` fallback, and `sim run` package load
  paths so local operator/developer commands reject migration-bearing
  packages that fail replay normalization. Added regression coverage in
  `terminal_server/cmd/term/main_test.go`
  (`TestRunAppCheckRejectsMigrationWhenDryRunGateFails`) and
  documented behavior in `docs/application-migrations.md`.

- 2026-04-28: Wired Gate 4 load-time migration replay enforcement
  into the server startup install path by default in
  `terminal_server/cmd/server/main.go`.
  Added `newServerAppRuntime()` to enable
  `SetMigrationDryRunGateEnabled(true)` before package discovery/load,
  and added regression coverage in
  `terminal_server/cmd/server/main_test.go`
  (`TestNewServerAppRuntimeEnablesMigrationDryRunGate`) to confirm
  migration-bearing packages with invalid execution-time scripts fail load
  with `ErrMigrationDryRunFailed` under server runtime defaults.

- 2026-04-28: Added optional Gate 4 load-time migration replay enforcement
  in `terminal_server/internal/appruntime/runtime.go`.
  Runtime now exposes `SetMigrationDryRunGateEnabled(true)`;
  when enabled, `LoadPackage` runs migration crash-replay checks via
  `DryRunMigrationJournalReplay` and rejects migration-bearing packages with
  `ErrMigrationDryRunFailed` if any journal boundary replay does not
  normalize and resume to `verdict = ok`.
  Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeLoadPackageRejectsMigrationWhenDryRunGateFails`) and
  documented behavior/remaining install-path wiring in
  `docs/application-migrations.md`.

- 2026-04-28: Added a reusable migration crash-replay dry-run harness in
  `terminal_server/internal/appruntime/runtime.go` via
  `DryRunMigrationJournalReplay`.
  The harness now runs boundary interruption checks across
  `retry_started` plus each planned step's
  `step_started`/`step_committed` events in isolated package copies,
  then verifies replay normalization (`step_failed`) and resumed
  completion (`ok`). Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeDryRunMigrationJournalReplayExercisesAllBoundaries` and
  `TestRuntimeDryRunMigrationJournalReplayReturnsEmptyWhenNoSteps`) and
  documented scope/limits in `docs/application-migrations.md`.

- 2026-04-28: Hardened execution-time migration fixture root checks
  against symlink traversal in
  `terminal_server/internal/appruntime/runtime.go`.
  `resolveRuntimeFixturePath` now resolves fixture symlink targets
  when available and rejects paths that resolve outside the loaded
  package root, preventing fixture declarations such as
  `tests/migrate_fixtures/history_seed.ndjson` from escaping via a
  symlink to external files. Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationFailsWhenFixturePathEscapesRootViaSymlink`)
  and documented behavior in `docs/application-migrations.md`.

- 2026-04-28: Hardened execution-time migration fixture path resolution
  in `terminal_server/internal/appruntime/runtime.go`.
  `readRuntimeFixtureRecords` now rejects fixture seed/expected paths
  that escape the loaded package root (for example `../outside.ndjson`)
  and fails retry with `ErrMigrationFixtureMismatch` while preserving
  existing `step_failed_fixture_mismatch` journaling semantics. Added
  regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationFailsWhenFixturePathEscapesRoot`) and
  documented behavior in `docs/application-migrations.md`.

- 2026-04-28: Hardened execution-time fixture canonicality checks in
  `terminal_server/internal/appruntime/runtime.go`.
  `readRuntimeFixtureRecords` now enforces LF-only NDJSON, trailing
  newline, non-blank rows, strict `key`/`value` envelopes, and
  ascending key order before comparing fixture expected output.
  Non-canonical fixture edits now fail retry with
  `ErrMigrationFixtureMismatch` and preserve
  `step_failed_fixture_mismatch` journaling semantics. Added
  regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationFailsWhenFixtureRecordNotCanonical`) and
  documented behavior in `docs/application-migrations.md`.

- 2026-04-28: Expanded migration crash-injection replay coverage in
  `terminal_server/internal/appruntime/runtime_test.go`.
  `TestRuntimeRetryMigrationCrashInjectionReplaysAtJournalBoundaries`
  now exercises boundary interruptions on later pending steps
  (step 2 `step_started` and `step_committed`) in addition to first-step
  boundaries, verifying journal replay preserves completed-step checkpoints
  and per-step interruption context before retry resumes to `verdict = ok`.
  Documented the broader coverage in `docs/application-migrations.md`.

- 2026-04-28: Enforced execution-time migration fixture record limits
  in `terminal_server/internal/appruntime/runtime.go`.
  `readRuntimeFixtureRecords` now rejects fixture files with more than
  4096 records using `ErrMigrationFixtureMismatch`, preserving step
  checkpoint state and existing `step_failed_fixture_mismatch` journal
  semantics when oversized fixtures are encountered after package load.
  Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationFailsWhenFixtureRecordLimitExceeded`) and
  documented behavior in `docs/application-migrations.md`.

- 2026-04-27: Hardened migration module-load parsing to ignore
  commented `load(...)` text while still enforcing disallowed
  module imports in real statements. Updated
  `terminal_server/internal/apppackage/tap.go` and
  `terminal_server/internal/appruntime/runtime.go` to anchor
  migration `load(...)` scanning at statement starts, preventing
  false disallowed-module failures from comment examples in
  migration scripts. Added regression coverage in
  `terminal_server/internal/apppackage/tap_test.go`
  (`TestVerifyTapIgnoresCommentedDisallowedLoadStatements`) and
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationIgnoresCommentedLoadStatements`).

- 2026-04-27: Hardened runtime fixture declaration enforcement in
  `terminal_server/internal/appruntime/runtime.go`.
  `RetryMigration` now treats missing `[[migrate.fixture]]` metadata
  for a pending step as execution-time fixture unavailability when
  fixture mode is active (manifest declares fixtures), preventing
  step commits that bypass fixture coverage after package mutation.
  Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationFailsWhenFixtureDeclarationMissingForPendingStep`),
  and documented behavior in `docs/application-migrations.md`.

- 2026-04-27: Added runtime crash-injection replay coverage at
  migration journal boundaries in
  `terminal_server/internal/appruntime/runtime.go`.
  `RetryMigration` now supports an internal interruption hook at
  `retry_started`, `step_started`, and `step_committed` boundaries,
  persisting in-progress `verdict = running` state for restart replay
  normalization tests. Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationCrashInjectionReplaysAtJournalBoundaries`),
  and documented behavior in `docs/application-migrations.md`.

- 2026-04-27: Added runtime migration fixture expected-output
  comparison guard in `terminal_server/internal/appruntime/runtime.go`.
  `RetryMigration` now checks declared `[[migrate.fixture]]` entries
  for each pending step and compares fixture seed/expected envelopes
  with key-set equality plus canonical JSON value equality before
  marking a step committed. Divergence now fails retry with
  `ErrMigrationFixtureMismatch`, sets `verdict = step_failed`, and
  appends `step_failed_fixture_mismatch` journal metadata. Missing
  fixture files now fail with `ErrMigrationFixtureUnavailable` and
  `step_failed_fixture_unavailable` journal metadata. Added regression
  coverage in `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationWithFixtureExpectedMatch`,
  `TestRuntimeRetryMigrationFailsWhenFixtureExpectedMismatch`) and
  documented behavior in `docs/application-migrations.md`.

- 2026-04-27: Added execution-time migration script integrity
  enforcement in `terminal_server/internal/appruntime/runtime.go`.
  `RetryMigration` now validates each pending step script contains a
  `migrate()` entrypoint and only loads allowed migration modules
  (`store`, `artifact.self`, `log`, `migrate.env`) before step
  lifecycle progression. Invalid scripts now fail with
  `ErrMigrationStepInvalid`, preserve completed checkpoint progress,
  set `verdict = step_failed`, and append `step_failed_invalid_script`
  entries with per-step metadata to the migration journal. Added
  regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationFailsWhenPendingScriptInvalid`) and
  documented the behavior in `docs/application-migrations.md`.

- 2026-04-27: Added Gate 1 expected-fixture target schema validation
  in `terminal_server/internal/apppackage/tap.go`. When a migration
  fixture step's target version resolves to exactly one declared
  `[[storage.store_schema]]` entry, `VerifyTap` now validates each
  `expected` NDJSON record's `value` against that target record
  schema and rejects schema-invalid expected fixtures with explicit
  diagnostics. Added regression coverage in
  `terminal_server/internal/apppackage/tap_test.go`
  (`TestVerifyTapRejectsMigrateFixtureExpectedSchemaMismatch`) and
  documented the behavior in `docs/application-migrations.md`.

- 2026-04-27: Hardened migration replay recovery for interrupted runs
  in `terminal_server/internal/appruntime/runtime.go`.
  Journal replay now normalizes stale `verdict = running` state
  to `verdict = step_failed` with explicit interruption context,
  so restart surfaces actionable retry state instead of showing
  indefinitely running migrations after process interruption.
  Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeInterruptedMigrationReplaysAsStepFailedAndResumes`)
  and documented the behavior in `docs/application-migrations.md`.

- 2026-04-27: Added execution-time migration script availability
  enforcement in `terminal_server/internal/appruntime/runtime.go`.
  `RetryMigration` now verifies each pending step script exists on
  disk before execution; missing scripts fail fast with
  `ErrMigrationStepUnavailable`, preserve completed checkpoint
  progress, set `verdict = step_failed`, and append
  `step_failed_unavailable` entries with per-step metadata in the
  migration journal. Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationFailsWhenPendingScriptUnavailable`) and
  documented the behavior in `docs/application-migrations.md`.

- 2026-04-27: Enforced migration fixture version-edge consistency
  in `terminal_server/internal/apppackage/tap.go`.
  `VerifyTap` now requires each `[[migrate.fixture]].prior_version`
  to match the referenced migration step's `from` version derived
  from `migrate/<step>_<from>_to_<to>.tal`, preventing fixture
  metadata drift from the declared step edge. Added regression
  coverage in `terminal_server/internal/apppackage/tap_test.go`
  (`TestVerifyTapRejectsMigrateFixturePriorVersionMismatch`) and
  documented the behavior in `docs/application-migrations.md`.

- 2026-04-27: Hardened runtime migration step execution planning in
  `terminal_server/internal/appruntime/runtime.go`. Retry now
  builds a deterministic step plan from `migrate/*.tal`
  (`<step>_<from>_to_<to>.tal`) rather than iterating by file
  count alone, emits `from_version`/`to_version`/`script` metadata
  on per-step journal events, and disables executor readiness with
  surfaced `last_error` when runtime step plans are malformed
  (filename format, numbering gaps, or manifest-step mismatch).
  Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeMigrationInvalidStepPlanDisablesExecutor` plus
  metadata assertions in `TestRuntimeMigrationLifecycleWithSteps`),
  and documented the operator-visible behavior in
  `docs/application-migrations.md`.

- 2026-04-27: Added Gate 4 seed-schema validation during package
  verification in `terminal_server/internal/apppackage/tap.go`.
  Migration seed fixtures are now parsed into canonical records and
  each `value` object is validated against the fixture's declared
  `prior_record_schema` (`[[migrate.fixture]].prior_record_schema`).
  Added regression coverage in
  `terminal_server/internal/apppackage/tap_test.go`
  (`TestVerifyTapRejectsMigrateFixtureSeedSchemaMismatch`) and
  documented the behavior in `docs/application-migrations.md`.

- 2026-04-27: Added operator-surface drain-readiness controls
  across admin API and REPL. `terminal_server/internal/admin/server.go`
  now serves `/admin/api/apps/migrate/drain-ready` and wires
  `action=drain-ready` to runtime `SetMigrationDrainReady` with
  refreshed status payloads. `terminal_server/internal/repl/repl.go`
  now includes `apps migrate drain-ready <app> <true|false>` in
  command metadata, usage, and execution flow. Added regression
  coverage in `terminal_server/internal/admin/server_test.go`
  and `terminal_server/internal/repl/repl_test.go`, and documented
  the command in `docs/repl/commands/app.md` and
  `docs/application-migrations.md`.

- 2026-04-27: Keyed runtime migration journals by lineage identity
  when available. `terminal_server/internal/appruntime/runtime.go`
  now parses optional `app_id` from `manifest.toml`, validates
  `app:sha256:<64-hex>` format, and writes migration journal paths
  under `apps/<app_id>/migrate/...` (with manifest-name fallback
  when `app_id` is absent). Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeMigrationJournalPathUsesAppID`,
  `TestRuntimeRejectsInvalidAppID`) and documented the behavior in
  `docs/application-migrations.md`.

- 2026-04-27: Tightened migration operator-surface coverage in
  `terminal_server/internal/repl/repl.go` and
  `terminal_server/internal/repl/repl_test.go`. The
  `apps migrate` top-level usage string now correctly includes
  the `logs` subcommand (`<status|logs|retry|abort|reconcile>`),
  and added regression tests for reconcile form wiring/output
  (`TestAppsMigrateReconcileUsesAdminAPI`) plus usage guidance
  (`TestExecuteCommandAppsMigrateUsageIncludesLogs`). Documented
  this coverage in `docs/application-migrations.md`.

- 2026-04-27: Normalized migration reconcile semantics for apps
  without runnable migration steps. `ReconcileMigration` in
  `terminal_server/internal/appruntime/runtime.go` now returns
  `ErrMigrationReconcilePending` ("nothing to reconcile") instead
  of the stale `ErrMigrationExecutorUnavailable` sentinel. Removed
  the corresponding unsupported-action branch in
  `terminal_server/internal/admin/server.go`, and updated runtime
  coverage in `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeMigrationStatusAndActions`).

- 2026-04-27: Persisted drain-guard blocked timing across process
  restart in `terminal_server/internal/appruntime/runtime.go`.
  `RetryMigration` now records `blocked_since` on
  `retry_blocked_drain_pending` journal entries, and migration state
  replay restores `DrainBlockedAt` from the journal so timeout windows
  continue across runtime restart instead of resetting. Added
  regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeDrainPendingBlockedAtReplaysFromJournal`) and
  documented the behavior in `docs/application-migrations.md`.

- 2026-04-27: Implemented drain-timeout orchestration in
  `terminal_server/internal/appruntime/runtime.go` so
  incompatible `drain` migrations no longer abort immediately.
  `RetryMigration` now enters `drain_pending`, tracks first
  blocked-at time, honors `[migrate].drain_timeout_seconds`
  (default 90s), and only returns `ErrMigrationDrainTimeout`
  after the timeout window elapses. Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationRequiresDrainReadiness`) for both
  pre-timeout pending and post-timeout abort behavior.

- 2026-04-27: Added migration journal replay in
  `terminal_server/internal/appruntime/runtime.go` so package load now
  restores migration status fields (`verdict`, `steps_completed`,
  `last_step`, `last_error`) from existing NDJSON entries for the
  current revision journal. Journal events now include `last_error`
  to preserve blocked/aborted context across process restart. Added
  regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeMigrationStateReplaysFromJournal`) and documented
  replay behavior in `docs/application-migrations.md`.

- 2026-04-27: Added explicit Gate 1 migration path diagnostics in
  `terminal_server/internal/apppackage/tap.go` for invalid nested
  `migrate/` layouts and malformed migration filenames. `VerifyTap`
  now reports specific `ErrInvalidManifest` context when downgrade
  scripts are nested under `migrate/downgrade/` or migration steps do
  not follow `<step>_<from>_to_<to>.tal`. Added regression coverage in
  `terminal_server/internal/apppackage/tap_test.go`
  (`TestVerifyTapRejectsNestedMigrateDowngradePath`,
  `TestVerifyTapRejectsMigrateMalformedStepFilename`) and documented
  the diagnostics in `docs/application-migrations.md`.

- 2026-04-27: Added migration fixture volume guard in
  `terminal_server/internal/apppackage/tap.go` so
  `validateMigrationFixtureNDJSON` rejects seed/expected fixture
  files that exceed 4096 records, keeping Gate 4 synthetic-store
  inputs bounded per fixture. Added regression coverage in
  `terminal_server/internal/apppackage/tap_test.go`
  (`TestVerifyTapRejectsMigrateFixtureTooManyRecords`,
  `TestVerifyTapAcceptsMigrateFixtureAtRecordLimit`) and
  documented the limit in `docs/application-migrations.md`.

- 2026-04-27: Tightened runtime reconciliation guard semantics in
  `terminal_server/internal/appruntime/runtime.go` so
  `RetryMigration` now blocks whenever migration status is
  `reconcile_pending` (even if pending-record detail is
  temporarily absent), matching abort/rollback guard behavior and
  preventing retries from bypassing reconciliation-required state.
  Added regression assertions in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeReconcileMigrationPendingRecords`) for the
  verdict-only reconcile-pending edge case, and documented the
  operator-visible behavior in `docs/application-migrations.md`.

- 2026-04-27: Aligned checkpoint abort semantics with design intent in
  `terminal_server/internal/appruntime/runtime.go`. `AbortMigration`
  now returns `verdict = step_failed` for checkpoint-target aborts,
  preserves retry checkpoint progress, and surfaces failed-step detail in
  `last_step` / `last_error`; baseline aborts remain `verdict = aborted`.
  Added regression assertions in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeMigrationLifecycleWithSteps`) and documented the behavior in
  `docs/application-migrations.md`.

- 2026-04-27: Improved runtime migration retry lifecycle in
  `terminal_server/internal/appruntime/runtime.go` so retries now
  resume from the first incomplete step instead of replaying all
  steps. `RetryMigration` now emits per-step journal events
  (`step_started`/`step_committed`) between
  `retry_started`/`retry_committed`, giving operators and tests a
  concrete step-level execution trace. Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeMigrationLifecycleWithSteps`) asserting per-step
  journal events and second-retry resume behavior after
  checkpoint-target abort.

- 2026-04-27: Runtime migration actions now emit structured
  journal NDJSON entries directly from
  `terminal_server/internal/appruntime/runtime.go` so operator
  logs reflect real state transitions instead of only fixture data.
  `RetryMigration` writes `retry_started`/`retry_committed` (or
  blocked events for reconcile/drain guard), `AbortMigration`
  writes `aborted` with target metadata, and
  `ReconcileMigration` writes `reconcile_record` entries with
  record/resolution context. Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeMigrationLifecycleWithSteps`,
  `TestRuntimeReconcileMigrationPendingRecords`) to assert
  journal file creation and emitted event payloads.

- 2026-04-27: Updated Gate 1 migration package validation to
  allow reverse-step scripts under `migrate/downgrade/*.tal`
  for rollback workflows while continuing to reject unsupported
  nested downgrade paths. Added regression coverage in
  `terminal_server/internal/apppackage/tap_test.go`
  (`TestVerifyTapAcceptsMigrateDowngradeScripts`,
  `TestVerifyTapRejectsNestedMigrateDowngradePath`) so
  keep-data rollback packages with reverse scripts pass
  verification.
- 2026-04-27: Added runtime drain-guard enforcement for incompatible
  migration steps in `terminal_server/internal/appruntime/runtime.go`.
  `RetryMigration` now aborts with
  `ErrMigrationDrainTimeout` when a package declares
  `[[migrate.step]] compatibility = "incompatible"` with
  `drain_policy = "drain"` and drain readiness has not been
  explicitly marked; step progress is preserved in that blocked
  state. Added `SetMigrationDrainReady` as an executor/orchestrator
  hook and regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationRequiresDrainReadiness`) for blocked
  and resumed retry paths.
- 2026-04-27: Implemented `apps migrate logs` operator surface
  across runtime-backed admin API and REPL. Added
  `/admin/api/apps/migrate/logs` in
  `terminal_server/internal/admin/server.go` with optional
  `step` filtering and bounded tail reads from migration
  journals, then wired `apps migrate logs <app> [--step <n>]`
  in `terminal_server/internal/repl/repl.go`. Added regression
  coverage in `terminal_server/internal/admin/server_test.go`
  and `terminal_server/internal/repl/repl_test.go`, and
  documented command usage in `docs/repl/commands/app.md`.
- 2026-04-27: Enforced reconciliation guard semantics for
  migration abort/rewind in runtime so `AbortMigration` now
  refuses while reconciliation is pending (`verdict ==
  reconcile_pending` or unresolved pending records). Abort no
  longer clears unresolved records or rewrites status to
  `aborted` in that state, preserving operator-required
  reconciliation behavior. Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeReconcileMigrationPendingRecords`) to verify
  abort is blocked and pending record IDs remain intact.
- 2026-04-27: Implemented rollback data-mode enforcement for
  `app(s) rollback` in runtime, admin API, and REPL. Rollback
  now defaults to archive mode, rejects `--keep-data` when no
  `migrate/downgrade/*.tal` reverse steps exist across the
  rollback span, and accepts keep-data when reverse steps are
  present. Added coverage in
  `terminal_server/internal/appruntime/runtime_test.go`,
  `terminal_server/internal/admin/server_test.go`, and
  `terminal_server/internal/repl/repl_test.go`.
- 2026-04-27: Implemented operator-selectable migration abort targets
  (`checkpoint` vs `baseline`) across runtime, admin API, and REPL.
  `apps migrate abort` now accepts `--to <checkpoint|baseline>`,
  runtime enforces target validation and baseline rewinds progress to
  step 0, and admin responses surface the resolved `to` value for
  operator visibility. Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`,
  `terminal_server/internal/admin/server_test.go`, and
  `terminal_server/internal/repl/repl_test.go`.
- 2026-04-27: Enforced rollback guard semantics in runtime so
  `RollbackPackage` refuses while migration reconciliation is
  pending (`verdict == reconcile_pending` or unresolved pending
  records). This aligns runtime behavior with the plan's rollback
  contract and prevents silently burying partial rollback state.
  Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRollbackBlockedWhenMigrationReconcilePending`) to
  verify rollback is blocked and package/history state remains
  unchanged.
- 2026-04-27: Extended migration operator status detail end-to-end
  across runtime, admin API, and REPL. `MigrationStatus` now includes
  explicit `last_step` and sorted pending reconciliation record details
  (`record_id`, `recommended_resolution`) from
  `terminal_server/internal/appruntime/runtime.go`; admin status payloads
  expose those fields via `mapMigrationStatus` in
  `terminal_server/internal/admin/server.go`; and
  `apps migrate status` now prints last-step, last-error, and pending
  record IDs in `terminal_server/internal/repl/repl.go`. Added/updated
  coverage in `terminal_server/internal/appruntime/runtime_test.go`,
  `terminal_server/internal/admin/server_test.go`, and
  `terminal_server/internal/repl/repl_test.go`.
- 2026-04-27: Added Gate 1 migration fixture NDJSON validation in
  `.tap` verification (`terminal_server/internal/apppackage/tap.go`):
  fixture files now require LF-only line endings with trailing LF,
  canonical per-line JSON envelope (`{"key":...,"value":...}`),
  UTF-8 NFC key validation, strict key byte-order sorting, and
  duplicate-key rejection. Added focused coverage in
  `terminal_server/internal/apppackage/tap_test.go` via
  `TestVerifyTapRejectsMigrateFixtureCRLFLineEndings`,
  `TestVerifyTapRejectsMigrateFixtureOutOfOrderKeys`,
  `TestVerifyTapRejectsMigrateFixtureDuplicateKeys`, and
  `TestVerifyTapRejectsMigrateFixtureNonCanonicalJSONLine`.
- 2026-04-27: Added specific Gate 1 migration failure diagnostics
  for numbering gaps and incompatible drain policy combinations
  in `.tap` verification (`terminal_server/internal/apppackage/tap.go`).
  Added message-level assertions in
  `TestVerifyTapRejectsMigrateStepNumberingGap` and
  `TestVerifyTapRejectsMigrateIncompatibleWithoutDrain` to lock
  these errors as explicit acceptance behavior.
- 2026-04-27: Added Gate 1 migration module-set enforcement in
  `.tap` verification (`terminal_server/internal/apppackage/tap.go`)
  so `migrate/*.tal` may only `load("store")`,
  `load("artifact.self")`, `load("log")`, and
  `load("migrate.env")`; disallowed modules (for example `bus`)
  now fail verification with a specific error message. Covered by
  `TestVerifyTapRejectsMigrateLoadBusModule` and
  `TestVerifyTapAcceptsMigrateAllowedModules` in
  `terminal_server/internal/apppackage/tap_test.go`.
- 2026-04-27: Implemented Gate 1 migration package-structure
  checks in `.tap` verification (`terminal_server/internal/apppackage/tap.go`)
  with unit coverage in
  `terminal_server/internal/apppackage/tap_test.go`.
- 2026-04-27: Added Gate 1 manifest policy validation that rejects
  `compatibility = "incompatible"` paired with
  `drain_policy = "none"`, with explicit accept/reject unit
  coverage in `terminal_server/internal/apppackage/tap_test.go`.
- 2026-04-27: Added Gate 1 migration fixture/schema enforcement:
  packages with `migrate/*.tal` must declare `[[storage.store_schema]]`
  and one `[[migrate.fixture]]` per migration step, with fixture
  file-path presence checks in `.tap` verification and coverage in
  `terminal_server/internal/apppackage/tap_test.go`.
- Implemented rules enforce contiguous migration step numbering,
  `manifest.toml` declaration/file-count consistency, and file ↔
  manifest step mapping for `migrate/*.tal` files.
- 2026-04-27: Wired migration operator actions to runtime-backed
  state transitions in `terminal_server/internal/appruntime/runtime.go`.
  `apps migrate status` now reports whether migration steps exist
  for the loaded package, `apps migrate retry` marks step progress
  as completed for ready packages, `apps migrate abort` records an
  explicit aborted verdict, and `apps migrate reconcile` resolves
  pending reconciliation records with guarded resolution values.
  Admin + REPL tests now cover the non-stubbed API path and command
  output for migration retry.
- Remaining work in this plan includes the full executor lifecycle
  from this design (actual TAL step execution against synthetic
  fixture-backed stores plus per-step timeout/checkpoint/hard-cap
  enforcement semantics).

## Problem

The distribution plan originally asserted that migrations
"operate on store snapshots and artifact patches, run in a
transaction, and are idempotent." Review flagged several
problems:

- Artifacts are first-class, identity-owned, referenced by other
  apps. A migration that patches them silently can corrupt data
  the migrating app does not own.
- Runtime specifies *per-activation* version pinning and a
  per-package `migrate(from_version, state)` function for
  *durable cross-version resume* — not a bulk data-mutating
  upgrade. The distribution plan conflated the two.
- No executor contract defines what a migration may call, what
  happens on crash mid-run, or how rollback undoes effects
  outside the app's own stores.
- Pre-upgrade activations pinned to the old version continue
  running *concurrently* with the migration, so an
  incompatible store-schema migration can corrupt live reads.
- `artifact.self` ownership was keyed by raw author key, which
  strands artifacts across legitimate key rotations.
- "Partial rollback" was treated as a routine warning, not as
  a blocking state that requires reconciliation.
- `abort` semantics were unspecified: rewind to which
  checkpoint, what happens to committed artifact patches, and
  what the resulting upgrade status looks like were undefined.
- Packages that ship migrations were not required to prove the
  migration is crash-safe before installation.

This document fixes those gaps. It defines two distinct
migration concepts, both inside the TAR runtime, and specifies
the executor that runs them.

## Design Principles

1. **Migrations are code running under a constrained sub-runtime,
   not a free-form upgrade script.** Permissions narrow during
   migration; they never widen.
2. **Data the app does not own is out of reach.** A migration
   may propose a patch to an artifact its app lineage
   (`app_id`, see [signing-and-trust.md](signing-and-trust.md)
   §1.4) created, but cannot reach into artifacts owned by other
   identities or touch the bus, telephony, HTTP, scheduler, or
   UI.
3. **Incompatible schema changes require draining old
   activations first.** Concurrent reads on incompatible schemas
   are forbidden by construction, not by author discipline.
4. **Atomicity matches the boundary of the owning subsystem.**
   App-scoped stores are transactional. Artifact patches are
   journaled and reversible. Anything else is simply not
   reachable from a migration.
5. **Crash mid-run is a normal case.** Every migration is
   resumable from the last committed step and every operator
   surface has an explicit rewind semantics.
6. **Partial rollback is a blocking state.** An upgrade whose
   rollback could not fully reverse its artifact patches holds
   in a named `reconcile_pending` state until an operator
   resolves it; it never silently degrades to `ok`.
7. **Version pinning is preserved for compatible migrations.**
   Pre-upgrade activations continue on their old version;
   migration affects only durable state and newly-started
   activations. Incompatible migrations require drain first
   (§3.1).

## Non-Goals

- No schema DSL. Migrations are TAL (deterministic) with a
  narrowed module set.
- No automatic inference of the migration path. The author
  supplies forward migrations explicitly.
- No downgrade migrations beyond the minimum required to support
  `term apps rollback`; downgrade with data mutation is
  intentionally hard.
- No distributed transaction across subsystems. Stores are
  transactional; artifact patches are journaled and reversible
  best-effort. See §3.4 and §4.

---

## 1. Two Kinds of Migration

### 1.1 Activation-state migration (already in runtime)

Runtime defines `migrate(from_version, state)` as an optional
package export. It is called when an *existing* activation's
pinned version is older than the current loaded version and the
activation resumes. This is lightweight: it transforms a single
activation's JSON state dict.

This document does not modify that contract. It is called out
here because distribution review conflated it with §1.2.

### 1.2 Durable-data migration (new)

A durable-data migration transforms **app-owned durable data**
when the app is upgraded across a version boundary declared by
the author. It runs at most once per server per version step,
ordered forward, inside the executor defined in §3.

Durable data means:

- **App-scoped stores.** Declared in `manifest.toml`'s
  `[storage].stores`. Fully owned by the app; freely readable
  and writable from a migration.
- **App-lineage artifacts.** Artifacts whose recorded
  `owner_app_id` (see [signing-and-trust.md](signing-and-trust.md)
  §1.4) equals the migrating app's `app_id`. Only these may be
  patched from a migration. Artifacts merely referenced or
  annotated are read-only. The `owner_app_id` is stable across
  author key rotations, so a rotated package can still migrate
  the artifacts its earlier versions created.

Nothing else is reachable. A migration cannot emit on the bus,
schedule a future trigger, open a UI view, place a call, hit
HTTP, invoke AI, or read presence / placement.

---

## 2. Manifest and Package Layout

A package declares durable-data migrations under `migrate/`:

```text
kitchen_timer/
├── manifest.toml
├── main.tal
└── migrate/
    ├── 0001_v1_to_v2.tal
    └── 0002_v2_to_v3.tal
```

File naming: `<step_number>_<from>_to_<to>.tal`. `<step_number>`
is zero-padded, monotonic, and gapless within a package. The
executor rejects a package with gaps or out-of-order numbering
at Gate 1 (package format).

### 2.1 Manifest block

```toml
[migrate]
declared_steps       = 2        # sanity check
max_runtime_seconds  = 120      # executor kills runaway migrations
checkpoint_every     = 500      # store ops between checkpoints

[[migrate.step]]
from              = "1"
to                = "2"
compatibility     = "incompatible"  # "compatible" | "incompatible"
drain_policy      = "drain"         # "drain" | "multi_version"
reason            = "adds required label_normalized field"

[[migrate.step]]
from              = "2"
to                = "3"
compatibility     = "compatible"
drain_policy      = "none"
reason            = "adds optional tag index; reads of v2 records still succeed"
```

Manifest runtime fields are advisory ceilings, not floors. The
executor enforces them; a migration that exceeds
`max_runtime_seconds` is rolled back to its last checkpoint and
the upgrade aborts.

If a migration's declared `max_runtime_seconds` exceeds the
server's configured default (`policy.migrate.max_runtime_seconds`,
default 120), the install transaction must present a
**preflight estimate** to the operator before running: current
store sizes, declared worst-case write volume, and estimated
runtime at the declared rate. The operator acknowledges the
estimate (ordinary `mutating`) or aborts the upgrade. A
migration whose live store size exceeds the budget implied by
`max_runtime_seconds` produces a `block`-level preflight error
that requires policy override, not just acknowledgment.

### 2.1.1 Manifest store schemas and migration fixtures

Packages that ship a non-empty `migrate/` directory MUST also
declare, for each store they read or write during migration,
both a **record schema** and a **migration fixture**. Gate 4
(§6) cannot run the dry-run harness otherwise.

```toml
[[storage.store_schema]]
store          = "history"
version        = "2"                    # matches manifest.version
record_schema  = "schemas/history_v2.json"  # JSON Schema draft 2020-12

[[migrate.fixture]]
step              = "0001_v1_to_v2"
prior_version     = "1"
prior_record_schema = "schemas/history_v1.json"
seed              = "tests/migrate_fixtures/history_v1_seed.ndjson"
expected          = "tests/migrate_fixtures/history_v2_expected.ndjson"
```

Rules:

- `prior_record_schema` describes the record shape the migration
  expects to read. For `compatibility = "incompatible"` steps,
  both `prior_record_schema` and the new `record_schema` are
  required.
- `seed` is an ndjson file of **fixture records** conforming to
  `prior_record_schema`; `expected` is the ndjson file of
  fixture records the dry-run must produce. Both are part of
  the package and travel in the `.tap` under `tests/`.
- A package that declares `migrate/*.tal` without
  corresponding `[[migrate.fixture]]` entries fails at Gate 1.
- The fixtures are capped at 4096 records per step (operator
  policy can raise) so Gate 4 runs bounded.

#### 2.1.2 Fixture record envelope

Every line of `seed` and `expected` is a **fixture record**:
one JSON object with exactly two fields, no others.

```json
{ "key": "<string, ≤ 256 bytes, UTF-8 NFC>",
  "value": { ... } }
```

Rules:

- `key` is an opaque string from the app's store namespace. It
  is not interpreted by the executor beyond equality and
  ordering. Keys MUST be unique within a fixture file;
  duplicate `key` is a Gate 1 rejection (the fixture itself is
  ill-formed, independent of the migration).
- `value` is a JSON object. It is validated against the
  declared `record_schema` (for `expected`) or
  `prior_record_schema` (for `seed`). A value that fails
  schema validation is a Gate 1 rejection.
- Fixture files MUST be sorted ascending by `key` under strict
  byte-lexicographic ordering of the UTF-8 NFC key bytes. A
  mis-sorted line is a Gate 1 rejection. Sorting is load-
  bearing so that two independent tools generate byte-
  identical fixtures for the same semantic set.
- Lines are separated by a single `\n` (LF). Trailing `\n` on
  the last line is required. No `\r\n`. No blank lines. No
  comments. Any deviation is a Gate 1 rejection.
- Each line, parsed as JSON, MUST equal the same JSON value
  when re-serialized under RFC 8785 canonical JSON. Gate 4
  rejects the fixture otherwise — this ensures the fixture
  author did not rely on whitespace or member-order quirks.

#### 2.1.3 Fixture comparison semantics

Gate 4 compares the migrated fixture output against
`expected` as follows:

1. The executor writes migrated output to an in-memory
   ndjson stream using the same envelope.
2. Both `expected` and `output` are parsed into in-memory
   maps from `key → value`. The parse rejects duplicates
   (which §2.1.2 already excludes from well-formed fixtures).
3. Key sets MUST match exactly. A key present in one and not
   the other is a test failure; the fixture report names the
   missing key(s) and the containing file.
4. For each key, `expected[key]` and `output[key]` are
   compared under RFC 8785 canonical JSON — each is
   re-serialized and the two byte strings are compared for
   equality. This is the authoritative equality; no deep
   structural equality or library-specific comparison is
   used.
5. Gate 4 passes the step only when all key sets match and
   every value passes canonical-JSON equality. Any mismatch
   aborts Gate 4 with the first differing key and a byte-
   level diff of the two canonical JSON forms.

### 2.2 Compatibility declaration

Every declared step names one of:

| `compatibility` | Meaning                                                                         |
|-----------------|---------------------------------------------------------------------------------|
| `compatible`    | v(to) code can read all records written by v(from) code without transformation. |
| `incompatible`  | v(from) code reading post-migration records is undefined behavior.              |

And one of:

| `drain_policy`   | Meaning                                                                                   |
|------------------|-------------------------------------------------------------------------------------------|
| `none`           | Only valid with `compatibility = "compatible"`.                                           |
| `drain`          | Scenario engine drains all pre-upgrade activations before the executor runs.              |
| `multi_version`  | Package ships read adapters so v(from) code can read post-migration records (see §3.1.2). |

The executor rejects the combination `incompatible + none` at
Gate 1. The combination `compatible + drain` is permitted but
flagged; it's rarely needed.

### 2.3 Version window

Each migration file declares a single (from, to) step matching
a `[[migrate.step]]` entry. The executor computes the shortest
forward path from the installed version to the target version
and runs the files in order. A missing intermediate step is a
package-format error caught at Gate 1, not at runtime.

---

## 3. Executor Contract

### 3.1 When the executor runs

A durable-data migration runs during `term apps install` /
`term apps upgrade` **after** the full vetting pipeline passes
and **before** the new package is registered with the scenario
engine. The exact ordering depends on `drain_policy`:

#### 3.1.1 `drain` (default for incompatible)

1. Scenario engine stops accepting new activations for this app.
2. Existing activations are drained:
   - Activations with a `suspend(reason)` path are asked to
     suspend and recorded as pending.
   - Activations without a suspend path are terminated at their
     next yield boundary, per the runtime's existing
     cooperative shutdown protocol.
   - Drain intents are persisted in
     `apps/<app_id>/drain/intents.ndjson`, fsynced before the
     executor starts, so a server crash mid-drain resumes the
     drain rather than starting the migration on live data.
     Each line conforms to schema `drain-intent/1`:
     `{ "schema": "drain-intent/1", "tx_id", "app_id",
        "activation_id", "from_version", "action":
        "suspend"|"terminate"|"acknowledged"|"expired", "state":
        "pending"|"complete"|"failed", "at": <unix ms>,
        "reason": <tstr|null> }`.
     Entries append in order; a later entry for the same
     `activation_id` supersedes earlier ones. v1 readers reject
     unknown `schema` strings.
3. When all activations have drained (or `drain_timeout`
   elapses — default 90s, per-app override in manifest), the
   executor runs.
4. On migration success, the scenario engine registers the new
   package. Drained activations are **terminated**, not
   resumed: their snapshots are retained per §3.3 for
   post-mortem but they do not continue as live activations.
   New `ActivationRequest`s that arrive after commit start
   fresh activations at the new version. This preserves the
   invariant from
   [application-runtime.md](application-runtime.md) that no
   operation migrates a running activation across a version
   boundary.
5. On migration failure, the old package is re-registered and
   drained activations are terminated the same way; new
   activation requests resume at the old version.

`drain_timeout` expiry is a migration abort (§3.7), not a
forced kill: the executor does not run against a non-drained
app.

#### 3.1.2 `multi_version` (concurrent reads allowed)

The package ships read adapters that make v(from) code able to
read v(to) records. Executor steps:

1. Executor runs with old activations still live.
2. Every write the migration makes is also written through the
   author-supplied adapter, so v(from) readers see records in
   their expected shape.
3. On success, scenario engine swaps to new definitions. Old
   activations drain naturally at their own pace.

This path is opt-in, heavier on the author, and reserved for
long-running apps (telephony dispatchers, etc.) where drain
cost is unacceptable.

#### 3.1.3 `none` (compatible migrations only)

The executor runs with the old definitions still registered and
all activations live. Existing activations keep working on the
old version, per
[application-runtime.md](application-runtime.md). This path is
the common case for additive schema changes.

In all three paths, failure leaves the old package as the
current package and the new package in staging.

### 3.2 Narrowed module set

Inside migration files, `load(…)` is restricted to:

| Module               | Why                                             |
|----------------------|-------------------------------------------------|
| `store`              | Read/write app-scoped KV namespaces.            |
| `artifact.self`      | Patch artifacts whose owner is this `app_id`.   |
| `log`                | Structured logs scoped to the migration run.   |
| `migrate.env`        | Versions, checkpoint helpers, abort helper.    |

Everything else (`ui`, `bus`, `scheduler`, `placement`, `ai.*`,
`telephony`, `http`, `presence`, `world`, `claims`, `flow`,
`recent`, `pty`, `observe`) is unavailable — a `load("bus", …)`
inside `migrate/*.tal` fails at compile time with a specific
error.

`artifact.self` is a new host surface distinct from the general
`artifact` module in
[shared-artifacts.md](shared-artifacts.md). Its writes are
filtered by an owner check at the host layer: the artifact's
recorded `owner_app_id` must equal the migrating app's `app_id`
(the lineage-stable identifier from
[signing-and-trust.md](signing-and-trust.md) §1.4, not the
current author key). A package that tries to patch artifacts it
did not author — including artifacts authored by an unrelated
app that shares a manifest name — is rejected by the executor
with a structured error.

### 3.3 Journaled effects

The executor maintains a per-run journal under paths keyed by
`app_id`, not manifest name (see
[signing-and-trust.md](signing-and-trust.md) §1.4):

- `apps/<app_id>/migrate/<run_id>/journal.ndjson` — append-only
  list of effects (`store.put`, `store.delete`,
  `artifact.self.patch`) with before/after content hashes and
  effect sequence numbers.
- `apps/<app_id>/migrate/<run_id>/checkpoint.json` — last
  committed effect sequence and a logical cursor supplied by
  the migration via `migrate.env.checkpoint(cursor=…)`.
- `apps/<app_id>/migrate/<run_id>/baseline.json` — snapshot
  pointers captured before step 1 runs (store generation
  numbers, list of artifact IDs and their head revisions). Used
  for pre-upgrade rewind (§3.7).

`run_id` is a monotonic counter per `app_id`; retries after
abort start a new `run_id` but read baseline from the previous
run.

Every `checkpoint_every` effects the executor:

1. Flushes buffered writes to the underlying transactional
   store.
2. Appends journal entries.
3. Updates the checkpoint file with an fsync.

Failure modes:

- **Crash between effect and journal.** On restart, the executor
  sees the effect is not journaled and re-runs the migration
  from the last checkpoint. This is why §3.5 requires idempotent
  migrations.
- **Crash between journal and store commit.** The store
  transaction has not committed, so re-running replays the
  effect safely. Journal entries are idempotent on replay.
- **Crash after store commit but before checkpoint.** The
  checkpoint is behind the store; re-running replays committed
  effects against the now-newer state. Idempotency keeps this
  safe.

### 3.4 Transactional boundary

Store writes are transactional at the subsystem level: all store
effects in a single checkpoint group commit or none do. Artifact
patches are reversible via `artifact.self.patch`'s journal — a
failed migration rewinds artifact patches by applying their
inverse in reverse order.

**There is no distributed transaction across stores and
artifacts.** See §3.7 for rollback semantics and when the
upgrade enters `reconcile_pending` rather than `ok` or
`aborted`.

### 3.5 Idempotency requirement

Every migration function MUST be a pure function of inputs
(current store state, current artifact contents) under the
executor's deterministic TAL runtime. The executor does not
verify idempotency statically, but the Gate 4 migration
dry-run harness (§6) injects crashes between checkpoints and
verifies that re-running reaches the same terminal state.

### 3.6 Resource limits

- `max_runtime_seconds` (per step, from manifest).
- Hard caps independent of manifest: 100 MB total write volume
  per step, 10⁶ store ops per step, 10⁴ artifact patches per
  step. A migration that exceeds any hard cap is aborted and
  flagged as `block` on retry — an app whose migration needs
  more than this should redesign, not raise the cap.

### 3.7 Abort and rewind semantics

`migrate.env.abort(reason)` and executor-initiated abort (timeout,
hard cap exceeded, host error) have identical rewind semantics.
The rewind target depends on which step is active:

#### 3.7.1 Rewind to the step's checkpoint

If abort occurs mid-step, the executor:

1. Rolls the store transaction back to the last checkpoint in
   this run.
2. Replays the journal's artifact-patch inverses for this step,
   in reverse order, stopping at the step's start marker.
3. Leaves earlier successfully-committed steps in place.

The upgrade's resulting status is `step_failed` with the
offending step id. `term apps migrate retry` resumes the failed
step from its last checkpoint.

#### 3.7.2 Rewind to pre-upgrade baseline

`term apps migrate abort <app> --to=baseline` (or operator
decision after repeated `retry` failures) rewinds *all* steps
in this upgrade:

1. For each committed step in reverse order, replay artifact
   inverses.
2. Restore each store to its baseline generation pointer. This
   is only safe if the store has not been read or written by
   anything other than this migration since baseline, which is
   guaranteed for `drain`, checked at rewind time for
   `multi_version`, and best-effort for `none` (compatible
   migrations do not change record semantics so rewind of new
   writes is safe).
3. Re-point `apps/<app_id>/current` at the prior
   `versions/<package_id>` via the install-transaction pointer
   flip ([application-distribution.md](application-distribution.md)
   §6.a.1). This is the visibility barrier for the rewind. The
   old package directory is already immutable and present; the
   flip is a `rename(2)` of a fresh `current.new` symlink.
4. Drained activations are **not** resumed. The runtime
   invariant from §3.1.1 holds through abort-to-baseline:
   every activation drained at `drain_required` is terminated
   and its termination recorded in
   `drain/intents.ndjson`. After the pointer flip, the
   scenario engine resolves `current` to the prior
   `package_id` and creates new activations at the old
   version from the next `ActivationRequest` onward. Clients
   observe the app as "was drained, now accepting new
   activations at the prior version."
5. Append a `verdict-log/1` entry with
   `decision = "rolled-back"` and a `verdict/1` bundle with
   `final_action = "rolled-back"`. `prev_hash` and
   `verdict_bundle_sha256` chain as usual.

If any artifact inverse fails during rewind (the artifact was
deleted by its owner between patch and rewind, or its current
revision is no longer a descendant of the one the journal
patched), the upgrade enters `reconcile_pending`, not `aborted`.

#### 3.7.3 `reconcile_pending`

When rewind cannot fully reverse its artifact patches, the
upgrade transitions to `reconcile_pending`:

- `apps/<app_id>/current` is re-pointed at the prior
  `versions/<package_id>` via the same pointer flip as
  §3.7.2, so the app runs on the old version. Drained
  activations are not resumed; new requests start at the old
  version.
- A reconciliation record is written to
  `apps/<app_id>/migrate/<run_id>/reconcile.json` listing every
  artifact whose inverse failed, with: artifact id, journaled
  patch, current head revision, and suggested resolution
  (`accept_current`, `force_rewind`, `manual`).
- `term apps migrate status <app>` surfaces the record.
- `term apps migrate reconcile <app> --artifact=<id>
  --resolution=<accept_current|force_rewind|manual>` resolves
  one artifact at a time.
- The upgrade status is *not* `ok` and *not* `aborted` until
  every reconciliation record is resolved. Activations run; the
  upgrade pipeline is simply blocked on this app.
- `term apps migrate reconcile` is `critical_mutating` per
  [signing-and-trust.md](signing-and-trust.md) §7, so AI agents
  cannot resolve it unilaterally.

### 3.8 Concurrency

At most one migration executor runs per `app_id` at a time.
This is enforced by the per-app install lock defined in
[application-distribution.md](application-distribution.md) (the
Install Transaction section), not by the migration executor
itself. The executor assumes the lock is held by its caller.

---

## 4. Authority and Signing

A migration inherits the authority of the package it ships in.
Specifically:

- A migration runs only if the package was installed through the
  normal vetting pipeline and the installed trust level permits
  it (per [signing-and-trust.md](signing-and-trust.md)).
- Under quarantine, migrations are **disabled** in v1 (the
  quarantine sandbox is a separate deferred plan). A package
  that ships `migrate/` and is installed at `quarantined` trust
  fails install rather than silently skipping migrations.
- Key rotation does not replay old migrations. Because paths
  are keyed by `app_id` (lineage), a rotated package's next
  upgrade runs only the migrations between the installed version
  and the new target version — exactly as a non-rotated
  package's upgrade would.
- `term apps migrate abort --to=baseline` and `term apps migrate
  reconcile` are `critical_mutating` operations and are subject
  to the voucher/approval rules in
  [signing-and-trust.md](signing-and-trust.md) §7.

---

## 5. Downgrade

`term apps rollback` installs an older package over a newer one.
The executor handles this by *not* running forward migrations
in reverse. Instead:

- If the older version ships an optional `migrate/downgrade/`
  directory with reverse steps, they are run in reverse order
  under the same executor rules.
- If not, the operator must choose `--archive-data` or `--purge`
  at the rollback command line. `--keep-data` is refused on a
  rollback that spans a version with no reverse migration.
- If the currently-installed version is in `reconcile_pending`,
  rollback is refused until reconciliation completes. This
  prevents silently burying a partial-rollback state.

Reverse migrations are optional by design: requiring them would
force authors to implement round-trip for every schema change,
which either makes authors avoid schema changes or ship
half-working reverse paths.

---

## 6. Gate 4 Migration Dry-Run

Packages that ship a non-empty `migrate/` directory are subject
to an additional vetting gate during Gate 4 (see
[application-distribution.md](application-distribution.md)
vetting pipeline). The dry-run harness:

1. Validates every fixture file end-to-end against the
   envelope rules in §2.1.2 (record shape, key uniqueness,
   sort order, line separators, canonical-JSON round-trip).
   Any violation is a Gate 4 `block` before any step runs.
2. Spawns an isolated executor instance against a synthetic
   store seeded from the `[[migrate.fixture]].seed` file. Each
   seeded record's `value` must validate against the declared
   `prior_record_schema`; a seed that fails schema validation
   is a Gate 4 `block`.
3. Runs every declared migration step end-to-end against its
   seed, producing an in-memory output stream in the fixture
   envelope.
4. Compares `output` against `expected` per §2.1.3 (key-set
   equality plus RFC 8785 canonical-JSON value equality).
   Any divergence is a Gate 4 `block` with the byte-level
   diff attached to the gate evidence.
5. For each step, re-runs it with an induced crash injected at
   every journal boundary (after `store.put`, before checkpoint;
   after checkpoint, before next op). Verifies the replayed
   state equals the non-crashed terminal state (which must in
   turn pass §2.1.3 equality against `expected`).
6. For `drain` steps, verifies the package declares
   `compatibility = "incompatible"`.
7. For `multi_version` steps, runs the author-supplied read
   adapter against migrated records and verifies adapter
   output validates against the declared
   `prior_record_schema`.

Any dry-run failure is a Gate 4 block. A package that fails the
dry-run cannot be installed through the normal pipeline;
quarantine is not an escape hatch because quarantine disables
migrations (§4).

The synthetic store generator is bounded — at most 4096 records
per fixture per §2.1.1. It is not a substitute for production
testing. It exists to catch the cheap bugs — non-idempotent
migrations, inverses that don't inverse, missing drain
declarations — before they reach the executor.

---

## 7. Operator Surface

Additions to the distribution plan's `term apps` commands:

```text
term apps migrate status     <app>                           # current step, last checkpoint, reconcile records
term apps migrate retry      <app>                           # restart from last checkpoint
term apps migrate abort      <app> [--to=checkpoint|baseline]# roll back current step or full upgrade
term apps migrate reconcile  <app> --artifact=<id> --resolution=<accept_current|force_rewind|manual>
term apps migrate logs       <app> [--step=N]                # tail of structured migration logs
```

`term apps upgrade` returns a structured result including:

- migration steps planned,
- steps completed,
- final verdict per step (`ok` / `step_failed` /
  `reconcile_pending` / `aborted`),
- a pointer to the journal files for post-hoc inspection,
- a pointer to any reconciliation records.

`term apps migrate abort` and `term apps migrate reconcile` are
`critical_mutating` per
[signing-and-trust.md](signing-and-trust.md) §7.

---

## 8. Worked Example

`kitchen_timer` v1 stores completed-timer records in
`store.history`. v2 adds a required `label_normalized` field
used by a new search feature. Because the field is required by
v2 code, the author declares:

```toml
[[migrate.step]]
from          = "1"
to            = "2"
compatibility = "incompatible"
drain_policy  = "drain"
reason        = "v2 search requires label_normalized on every history record"
```

`migrate/0001_v1_to_v2.tal`:

```python
load("store",        list_keys = "list_keys", get = "get", put = "put")
load("migrate.env",  checkpoint = "checkpoint", abort = "abort")
load("log",          info = "info")

def migrate():
    cursor = None
    count  = 0
    while True:
        page = list_keys(prefix = "history/", after = cursor, limit = 500)
        if len(page) == 0:
            break
        for key in page:
            rec = get(key)
            if "label_normalized" in rec:
                continue                 # idempotent: already migrated
            rec["label_normalized"] = _normalize(rec.get("label", ""))
            put(key, rec)
            count += 1
        cursor = page[-1]
        checkpoint(cursor = cursor)
    info("history.migrated", records = count)


def _normalize(label):
    return label.strip().lower()
```

Properties that make this migration safe under the executor:

- Touches only `store`, which is app-scoped.
- Pages through work and checkpoints after each page, with a
  cursor so resume is O(remaining) not O(total).
- Early-returns on records already carrying `label_normalized`,
  so re-running after a crash is a no-op for completed keys.
- Emits a single structured log at completion; nothing on the
  bus.
- The author declared `drain_policy = "drain"`, so no v1
  activation is live while the migration runs — a v1 read path
  cannot observe a half-migrated record.

A malformed alternative — "also emit a `history.migrated` event
on the bus" — would fail at compile time because `bus` is not
loadable from migration files. A second malformed alternative —
declaring `compatibility = "incompatible"` with `drain_policy =
"none"` — would fail at Gate 1.

---

## 9. Acceptance Criteria

- A package with gaps in its migration numbering fails Gate 1
  with a specific error.
- A package declaring `incompatible + none` fails Gate 1 with a
  specific error.
- A migration that tries to `load("bus")` fails at compile time
  with a specific error.
- A crash injected between `store.put` and the executor's
  checkpoint leaves durable state consistent on restart, and
  re-running the migration reaches the same final state. The
  Gate 4 dry-run harness exercises this for every declared step
  at every journal boundary.
- A migration that attempts `artifact.self.patch` on an
  artifact whose `owner_app_id` is not this app's `app_id` is
  rejected at the host layer with a structured error — including
  when the artifact's owner shares this app's manifest name but
  has a different lineage.
- An author key rotation followed by `term apps upgrade` runs
  only the migrations between the installed version and the new
  target; paths under `apps/<app_id>/migrate/...` continue to
  reference the same directory tree as before rotation.
- A rollback across a version with no reverse migration fails
  with `--keep-data` and succeeds with `--archive-data`.
- A rollback while the current version is in `reconcile_pending`
  fails with a specific error.
- `term apps migrate status` returns last-step, last-error, and
  reconciliation-record details sufficient for `term apps
  migrate retry`, `term apps migrate abort`, or `term apps
  migrate reconcile`.
- A migration whose artifact-patch inverse fails during rewind
  leaves the upgrade in `reconcile_pending`, not `aborted` or
  `ok`, until every reconciliation record is resolved.
- A `drain` migration whose drain timeout elapses aborts rather
  than running against live activations.

---

## Open Questions

- **Long-running migrations without an operator.** The 120-
  second default is appropriate for small household apps but
  wrong for any app with large stores. Per-app ceilings are
  configurable, but the question is whether the executor should
  support chunked execution across server restarts as a first-
  class feature or keep forcing the migration to finish in a
  single run.
- **Migration cost budgets in the risk gate.** Gate 7 of
  distribution could use declared migration size to adjust risk
  scoring. Not specified here; noted so the distribution plan
  can reference it later.
- **Reconciliation auto-resolution policies.** v1 requires an
  operator for every `reconcile_pending` artifact. A future
  policy extension could allow `accept_current` as a default
  for specific authored artifact classes, but only after the
  v1 flow has been exercised in practice.
