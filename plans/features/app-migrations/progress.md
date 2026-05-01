---
title: "Application Migrations — Progress Log"
kind: progress-log
parent: plans/features/app-migrations/plan.md
---

## Implementation Progress

- 2026-04-30: Tightened Gate 1 fixture schema enforcement for
  incompatible durable-data migrations in
  `terminal_server/internal/apppackage/tap.go`. Package verification
  now rejects an incompatible step when its `expected` fixture cannot
  be validated against an unambiguous target-version
  `[[storage.store_schema]]`, instead of treating the missing target
  schema as optional. Added regression coverage in
  `terminal_server/internal/apppackage/tap_test.go`
  (`TestVerifyTapRejectsIncompatibleMigrationWithoutTargetSchema`) and
  documented the behavior in `docs/application-migrations.md`.

- 2026-04-30: Aligned Gate 1 `.tap` verification with runtime
  read-adapter validation in
  `terminal_server/internal/apppackage/tap.go`. Multi-version
  `read_adapter` scripts now reject unsupported `return`
  expressions such as `return {}` during package verification,
  preserving quoted `#` characters while stripping TAL line comments
  before inspecting return statements. Added regression coverage in
  `terminal_server/internal/apppackage/tap_test.go`
  (`TestVerifyTapRejectsUnsupportedReadAdapterReturn`) and recorded
  the evidence in `docs/application-migrations.md`.

- 2026-04-30: Hardened `multi_version` read-adapter validation in
  `terminal_server/internal/appruntime/runtime.go`. Gate 4 runtime
  dry-run replay now rejects unsupported `return` expressions in
  read adapters (for example `return {}`) instead of letting the
  shared fixture parser ignore them as identity transforms. Added
  regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeLoadPackageRejectsUnsupportedReadAdapterReturnDuringDryRunGate`)
  and documented the deterministic read-adapter subset in
  `docs/application-migrations.md`.

- 2026-04-30: Aligned deterministic migration fixture parsing with
  the documented worked-example TAL control-flow style in
  `terminal_server/internal/appruntime/runtime.go`. Runtime dry-run
  replay now accepts multiline idempotency guards
  (`if "field" in record:` followed by `continue`) for direct
  `migrate(record)` scripts and multiline empty-page `break` plus
  `rec` presence guards in paged `store` loops. Updated regression
  coverage in `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationAppliesIdempotentFixtureGuard` and
  `TestRuntimeRetryMigrationAppliesPagedStoreFixtureTransforms`) and
  documented the accepted control-flow shape in
  `docs/application-migrations.md`.

- 2026-04-29: Added byte-level fixture mismatch evidence to
  runtime migration dry-run comparisons in
  `terminal_server/internal/appruntime/runtime.go`. Value
  mismatches now include canonical expected/actual JSON plus
  the first differing byte offset and byte values in
  `step_failed_fixture_mismatch` journal evidence, matching the
  Gate 4 comparison contract. Updated regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationFailsWhenFixtureExpectedMismatch`)
  and documented the evidence shape in
  `docs/application-migrations.md`.

- 2026-04-29: Aligned paged `store` loop fixture accounting with the
  direct `migrate(record)` fixture path in
  `terminal_server/internal/appruntime/runtime.go`. Store-loop dry-run
  replay now counts only changed `put` results plus successful deletes
  as synthetic store effects for checkpoint evidence and resource caps,
  so no-op idempotent writes do not inflate `[migrate].checkpoint_every`
  boundaries. Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationDoesNotCheckpointUnchangedStorePut`) and
  documented the boundary in `docs/application-migrations.md`.

- 2026-04-29: Aligned direct `abort(...)` reason parsing in the
  runtime migration fixture subset with loaded abort aliases and the
  existing accepted TAL string literal surfaces. Direct abort calls now
  accept single-quoted reasons such as
  `abort('unsafe # record shape')`, preserving `#` characters inside the
  literal instead of treating them as line comments. Updated regression
  coverage in `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeMigrationFixtureParsesAbortAliases`) and documented the
  parser boundary in `docs/application-migrations.md`.

- 2026-04-29: Aligned direct literal assignment parsing in the
  runtime migration fixture subset with the existing accepted TAL
  string literal surfaces. Direct `migrate(record)` scripts now
  accept single-quoted string assignments such as
  `record["source"] = 'fixture#migration'`, preserving `#`
  characters inside the literal instead of treating the expression
  as unsupported JSON. Updated regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationAppliesRecordGetFixtureTransforms`)
  and documented the parser boundary in
  `docs/application-migrations.md`.

- 2026-04-29: Hardened TAL line-comment stripping in the
  runtime migration fixture parser. The deterministic migration
  subset now preserves `#` characters inside single-quoted string
  literals, matching the existing double-quoted handling and the
  accepted single-quoted literal surfaces for migration log calls
  and `record.get(...)` defaults. Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationPreservesHashInSingleQuotedStrings`)
  and documented the parser boundary in
  `docs/application-migrations.md`.

- 2026-04-29: Expanded fixture-backed `record.get(...)`
  default handling again in the runtime migration subset. Direct
  `migrate(record)` scripts now accept structured JSON defaults
  (`object` and `array`) in addition to scalar defaults when
  replaying fixtures, matching the subset's existing JSON literal
  assignment support for common schema-fill migrations. Updated
  regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationAppliesRecordGetFixtureTransforms`)
  and documented the expanded subset in
  `docs/application-migrations.md`.

- 2026-04-29: Expanded fixture-backed `record.get(...)`
  default handling in the runtime migration subset. Direct
  `migrate(record)` scripts now accept JSON scalar defaults
  (`number`, `bool`, and `null`) in addition to string defaults
  when replaying fixtures, so common schema-fill migrations can
  be verified without reporting an unsupported assignment.
  Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationAppliesRecordGetFixtureTransforms`)
  and documented the expanded subset in
  `docs/application-migrations.md`.

- 2026-04-29: Added operator-visible rollback blocking details for
  migrations in `reconcile_pending`. The admin rollback endpoint now
  returns HTTP 409 with the current migration status payload, including
  pending reconciliation records and `reconciliation_path`, when runtime
  rejects rollback with `ErrMigrationReconcilePending`. Added regression
  coverage in `terminal_server/internal/admin/server_test.go`
  (`TestAppsRollbackBlockedByReconcilePendingReturnsMigrationStatus`) and
  documented the behavior in `docs/application-migrations.md`.

- 2026-04-29: Hardened artifact patch host-effect collection in
  `terminal_server/internal/appruntime/runtime.go`. Runtime retry
  now rejects `artifact.self.patch(...)` declarations whose first
  argument is not an explicit, non-empty artifact ID literal before
  journaling patch evidence, failing the step with
  `ErrMigrationArtifactOwnership` and `step_failed_host_rejected`
  evidence rather than accepting an empty artifact identifier. Added
  regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationRejectsArtifactPatchWithoutArtifactID`)
  and documented the behavior in `docs/application-migrations.md`.

- 2026-04-29: Tightened migration step policy requirements in
  `terminal_server/internal/apppackage/tap.go` and runtime plan
  parsing in `terminal_server/internal/appruntime/runtime.go`.
  Gate 1 now rejects `[[migrate.step]]` entries that omit
  `compatibility` or `drain_policy` instead of allowing an
  implicit execution policy, and runtime status leaves the
  executor disabled with the same specific diagnostics if package
  contents drift after verification. Added regression coverage in
  `terminal_server/internal/apppackage/tap_test.go`
  (`TestVerifyTapRejectsMigrateStepMissingPolicy`) and
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeMigrationInvalidManifestPolicyDisablesExecutor`),
  and documented the behavior in `docs/application-migrations.md`.

- 2026-04-29: Tightened fixture-backed synthetic store-effect
  accounting for direct `migrate(record)` dry-run execution in
  `terminal_server/internal/appruntime/runtime.go`. Runtime now
  counts only fixture rows whose canonical value changes as
  synthetic store writes for checkpoint evidence and resource
  accounting, so rows skipped by idempotency guards no longer
  inflate `[migrate].checkpoint_every` boundaries. Added regression
  coverage in `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationAppliesIdempotentFixtureGuard`) and
  documented the behavior in `docs/application-migrations.md`.

- 2026-04-29: Added operator-visible drain state to migration
  status surfaces. Runtime `MigrationStatus`, the admin API payload,
  and human-readable `apps migrate status` now report whether a
  pending step requires drain, whether drain readiness is approved,
  the configured drain timeout, and the current blocked-since
  timestamp while drain is pending. Tightened regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationRequiresDrainReadiness`) and
  `terminal_server/internal/repl/repl_test.go`
  (`TestAppsMigrateStatusUsesAdminAPI`), added admin API assertions,
  and documented the behavior in `docs/application-migrations.md`.
  Also pinned reconciliation journal replay after the final
  `reconcile_record` in
  `TestRuntimeReconcileMigrationPendingRecords`.

- 2026-04-29: Hardened runtime migration manifest policy parsing in
  `terminal_server/internal/appruntime/runtime.go`. Runtime migration
  plan parsing now mirrors Gate 1 validation for
  `[[migrate.step]].compatibility`, `drain_policy`, and the
  `incompatible + none` combination, leaving `executor_ready = false`
  with a specific `last_error` if package contents drift after
  verification. Added regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeMigrationInvalidManifestPolicyDisablesExecutor`) and
  documented the behavior in `docs/application-migrations.md`.

- 2026-04-29: Persisted migration drain-readiness decisions in
  `terminal_server/internal/appruntime/runtime.go`. Runtime
  `SetMigrationDrainReady` now appends `drain_ready_changed`
  journal evidence, and migration journal replay restores the
  latest ready/not-ready value so an operator-approved drain does
  not regress to `drain_pending` after process restart. Added
  regression coverage in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeDrainReadyReplaysFromJournal`) and documented the
  behavior in `docs/application-migrations.md`.

- 2026-04-29: Added explicit rollback data-mode regression
  coverage in `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRollbackKeepDataRequiresDowngradeSteps`).
  A rollback across a version with no reverse migration now has
  pinned evidence that `--keep-data` fails without mutating package
  history, and a follow-up `--archive-data` rollback succeeds.
  Documented the operator-visible behavior in
  `docs/application-migrations.md`.

- 2026-04-29: Hardened execution-time migration fixture key
  validation in `terminal_server/internal/appruntime/runtime.go`.
  Runtime retry now mirrors the Gate 1 fixture envelope checks for
  keys by rejecting non-UTF-8, non-NFC, empty, or >256-byte fixture
  keys after package load, failing the step with
  `ErrMigrationFixtureMismatch` before commit and preserving
  `step_failed_fixture_mismatch` journal evidence. Added regression
  coverage in `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationFailsWhenFixtureKeyInvalid`) and
  documented the behavior in `docs/application-migrations.md`.

- 2026-04-29: Added operator-visible journaling for
  fixture-backed migration `log.*(...)` calls in
  `terminal_server/internal/appruntime/runtime.go`. Runtime
  retry now parses accepted `log` aliases, records
  `migration_log` journal entries with level, message, raw
  arguments, and step/version metadata, and keeps those entries
  available through existing `apps migrate logs` surfaces. Added
  regression coverage to
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationAppliesPagedStoreFixtureTransforms`)
  and documented the behavior in `docs/application-migrations.md`.

- 2026-04-29: Tightened the human-readable migration operator
  status surface in `terminal_server/internal/repl/repl.go`.
  `apps migrate status` now prints sorted pending reconciliation
  records as `record_id:recommended_resolution` and includes
  `reconciliation_path`, so the plain-text command carries the
  same reconciliation detail needed for follow-up operator action
  as the admin API payload. Updated regression coverage in
  `terminal_server/internal/repl/repl_test.go`
  (`TestAppsMigrateStatusUsesAdminAPI`) and documented the
  behavior in `docs/application-migrations.md`.

- 2026-04-29: Tightened checkpoint abort semantics for
  in-flight migration steps in
  `terminal_server/internal/appruntime/runtime.go`. Operator
  checkpoint aborts now preserve already-committed step progress
  when `last_step` points at an uncommitted in-flight step,
  while retaining the existing completed-state rewind behavior
  for aborting after a terminal commit. Added regression coverage
  in `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeMigrationLifecycleWithSteps`) and documented the
  behavior in `docs/application-migrations.md`.

- 2026-04-29: Hardened Gate 4 `multi_version` read-adapter
  path diagnostics and regression coverage. Runtime dry-run
  replay already resolves `read_adapter` paths through the same
  package-root and symlink guard used for seed/expected fixture
  files; errors now identify the failing read-adapter path, and
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeLoadPackageRejectsMultiVersionReadAdapterEscapingRootDuringDryRunGate`)
  pins that packages cannot point adapters outside the package
  payload. Documented the behavior in
  `docs/application-migrations.md`.

- 2026-04-29: Tightened rollback reverse-step validation for
  `migrate/downgrade/*.tal`. Gate 1 package verification now
  requires downgrade scripts to use the same
  `<step>_<from>_to_<to>.tal` edge filename shape as forward
  migrations, and runtime `--keep-data` rollback only treats
  valid downgrade edge filenames as reverse-step evidence.
  Added regression coverage in
  `terminal_server/internal/apppackage/tap_test.go`
  (`TestVerifyTapRejectsMalformedMigrateDowngradeFilename`) and
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRollbackKeepDataRejectsMalformedDowngradeStep`),
  and documented the behavior in `docs/application-migrations.md`.

- 2026-04-29: Aligned runtime fixture lookup with Gate 1 package
  fixture identifiers. Runtime dry-run and retry fixture matching now
  accepts canonical `[[migrate.fixture]].step` values in
  `<step>_<from>_to_<to>` form (for example `0001_1_to_2`), matching
  package verification and docs, while retaining numeric step IDs such
  as `0001` as a local compatibility fallback. Updated
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeLoadPackageValidatesMultiVersionReadAdapterDuringDryRunGate`)
  to exercise the canonical package-format identifier and documented
  the behavior in `docs/application-migrations.md`.

- 2026-04-29: Added fixture-backed `multi_version` read-adapter
  replay for Gate 4 migration dry-runs. Multi-version fixtures
  now declare a `read_adapter`; package verification requires the
  adapter file to exist, expose `read(record)`, and only load
  migration-safe modules, while runtime dry-run replay executes
  the adapter against migrated fixture output and verifies the
  adapted records match the prior-version seed shape. Added
  regression coverage in `terminal_server/internal/apppackage/tap_test.go`
  (`TestVerifyTapRequiresReadAdapterForMultiVersionMigration`,
  `TestVerifyTapAcceptsMultiVersionReadAdapter`) and
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeLoadPackageValidatesMultiVersionReadAdapterDuringDryRunGate`),
  and documented the behavior in `docs/application-migrations.md`.

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
