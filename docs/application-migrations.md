# Application Migrations

Durable reference for currently implemented migration validation behavior.
This is the shipped subset from [plans/features/app-migrations.md](../plans/features/app-migrations.md).

## Implemented Gate 1 checks

The .tap package verifier in [terminal_server/internal/apppackage/tap.go](../terminal_server/internal/apppackage/tap.go) now validates migration layout at package-load time:

- If no `migrate/*.tal` files exist, migration declarations must be absent (`declared_steps = 0` and no `[[migrate.step]]`).
- If migration files exist, `[migrate].declared_steps`, `[[migrate.step]]` count, and migration file count must match.
- Migration files must be one level under `migrate/` and match `<step>_<from>_to_<to>.tal`.
- Reverse migration scripts under `migrate/downgrade/*.tal` are allowed for rollback flows, but must remain single-level files (nested downgrade folders are rejected).
- Step numbers must be contiguous and start at `1` (no gaps).
- Each sorted migration file must match the corresponding manifest step `from`/`to` values.
- Invalid migration script paths now return specific diagnostics, including nested path violations under `migrate/` / `migrate/downgrade/` and malformed step filenames that do not match `<step>_<from>_to_<to>.tal`.
- When a step declares `compatibility = "incompatible"`, it cannot also declare `drain_policy = "none"`.
- Migration fixture NDJSON files are bounded to at most 4096 records per file to keep Gate 4 synthetic-store input sizes predictable.
- Migration seed fixture records are validated against each fixture's declared `prior_record_schema`; invalid seed rows now fail package verification with record-level diagnostics.
- Migration expected fixture records are validated against the target step record schema when a unique `[[storage.store_schema]]` entry exists for the step `to` version; invalid expected rows now fail package verification with record-level diagnostics.
- Migration fixture metadata now enforces step-edge consistency: `[[migrate.fixture]].prior_version` must match the corresponding migration script `from` version (`migrate/<step>_<from>_to_<to>.tal`).
- When declared, `[migrate].max_runtime_seconds` and `[migrate].checkpoint_every` must be positive integers; non-positive values now fail Gate 1 with explicit diagnostics.

## Implemented runtime migration guard

The runtime migration control path in [terminal_server/internal/appruntime/runtime.go](../terminal_server/internal/appruntime/runtime.go) now enforces an incompatible-step drain guard:

- If any `[[migrate.step]]` declares `compatibility = "incompatible"` and `drain_policy = "drain"`, `RetryMigration` refuses to run until drain readiness is explicitly marked.
- Blocked retries first return `ErrMigrationDrainPending` and set `verdict = "drain_pending"` while drain is still within its timeout window.
- Drain timeout uses `[migrate].drain_timeout_seconds` (default 90s). Once that window elapses, `RetryMigration` returns `ErrMigrationDrainTimeout`, marks `verdict = "aborted"`, and preserves the current checkpoint (no step advancement while drain is unsafe).
- Operators/orchestrators can mark readiness through runtime (`SetMigrationDrainReady`), admin API (`/admin/api/apps/migrate/drain-ready`), or REPL (`apps migrate drain-ready <app> <true|false>`), after which retry proceeds normally.

## Implemented migration runtime journaling

The runtime now emits structured NDJSON migration journal entries directly from
`terminal_server/internal/appruntime/runtime.go` when operators invoke
migration control actions:

- `RetryMigration` writes `retry_started` and `retry_committed` entries on
	successful runs, and emits `step_started`/`step_committed` entries for each
	migration step so operators can see step-by-step progression.
- Blocked retries emit explicit events (`retry_blocked_reconcile_pending` and
	`retry_blocked_drain_pending` / `retry_blocked_drain_timeout`) with current
	verdict/step context and `blocked_since` timing metadata.
- Retry reconciliation guard now treats `verdict = reconcile_pending` as
	blocking even if pending-record details are temporarily unavailable, so
	operators must reconcile before retry can proceed.
- `AbortMigration` writes `aborted` entries including the selected target
	(`checkpoint` or `baseline`).
- `ReconcileMigration` writes `reconcile_record` entries with `record_id` and
	selected `resolution`.
- Reconcile operations now report `ErrMigrationReconcilePending` when no
	pending records exist, including apps with no runnable migration steps,
	so operators see one consistent "nothing to reconcile yet" response.
- Checkpoint abort now leaves migration state at `verdict = "step_failed"`
	with `last_step` pinned to the failed step and checkpoint progress preserved
	for `apps migrate retry`; baseline abort remains `verdict = "aborted"`.

Retry now resumes at the first incomplete step (`steps_completed + 1`) instead
of replaying the entire migration range on every retry.
- Retry execution now uses a parsed runtime step plan from `migrate/*.tal`
	filenames (`<step>_<from>_to_<to>.tal`) instead of a raw file count.
	Runtime journal entries for `step_started` / `step_committed` now include
	`from_version`, `to_version`, and `script` metadata so operator logs show the
	exact version edge and script for each executed step.
- Retry now verifies each pending migration script still exists on disk at
	execution time. If a pending script is unavailable, retry stops with
	`ErrMigrationStepUnavailable`, preserves completed checkpoint progress, marks
	`verdict = step_failed`, and emits `step_failed_unavailable` journal metadata
	for the failed step.
- Retry now also validates pending migration script content at execution time:
	every step script must include a `migrate()` entrypoint and may only
	load migration-safe modules (`store`, `artifact.self`, `log`,
	`migrate.env`). A script that fails these checks stops retry with
	`ErrMigrationStepInvalid`, preserves checkpoint progress, marks
	`verdict = step_failed`, and emits `step_failed_invalid_script` journal
	metadata for the failed step.
	Commented example text (for example `# load("bus", ...)`) is ignored when
	parsing module imports so documentation comments do not trigger false
	disallowed-module failures.
- Retry now validates declared migration fixture expected output before
	committing each step. When `[[migrate.fixture]]` declares a fixture for
	the pending step, runtime executes the current deterministic fixture
	subset for `migrate(record)` scripts before comparing actual output to the
	expected envelopes. The subset supports field copy, string lowercasing,
	JSON literal assignment, and field deletion. Mismatches stop retry with
	`ErrMigrationFixtureMismatch`, mark `verdict = step_failed`, and emit
	`step_failed_fixture_mismatch` entries.
	Value mismatches now include the first divergent key plus canonical
	expected/actual JSON bytes in the journal error evidence.
- If a declared fixture file cannot be read at execution time, retry stops
	with `ErrMigrationFixtureUnavailable`, marks `verdict = step_failed`, and
	emits `step_failed_fixture_unavailable` journal entries.
- Runtime fixture reads now enforce the same 4096-record ceiling used by
	Gate 4 package verification; oversized fixture files fail retry with
	`ErrMigrationFixtureMismatch` before step commit and emit
	`step_failed_fixture_mismatch` journal entries.
- Runtime fixture reads now enforce canonical NDJSON structure at
	execution time (LF line endings, trailing LF, no blank lines,
	strict `{"key":...,"value":...}` envelopes, and ascending key
	order). Non-canonical fixture mutations now fail retry with
	`ErrMigrationFixtureMismatch` before step commit and emit
	`step_failed_fixture_mismatch` journal entries.
- Runtime fixture reads now reject fixture paths that escape the loaded
	package root (for example `../outside_seed.ndjson`). Escape attempts fail
	retry with `ErrMigrationFixtureMismatch` before step commit and emit
	`step_failed_fixture_mismatch` journal entries.
- Runtime fixture path checks now also resolve symlink targets before reads;
	fixture files that lexically appear under the package root but resolve
	outside it (for example via `tests/migrate_fixtures/history_seed.ndjson`
	-> `/tmp/outside_seed.ndjson`) are rejected with
	`ErrMigrationFixtureMismatch`.
- When fixture declarations are present in `manifest.toml`, retry now also
	requires a `[[migrate.fixture]]` entry for each pending step. Missing
	per-step fixture metadata fails retry with `ErrMigrationFixtureUnavailable`
	and emits `step_failed_fixture_unavailable` journal entries.
- Runtime retry now supports crash-injection testing at journal boundaries
	(`retry_started`, `step_started`, `step_committed`). Injected interruptions
	persist as `verdict = running` snapshots in the journal; restart replay
	normalizes them to `verdict = step_failed` with interruption context so
	operators can retry from the last committed checkpoint.
- Reload-time migration state now baselines `steps_completed` from the
	installed package version before retrying. Upgrades that include historical
	migration scripts (for example `0001_1_to_2.tal` and `0002_2_to_3.tal` while
	reloading from version `2` to `3`) now start retry at the pending boundary
	instead of replaying earlier edges.
- Runtime now exposes a reusable dry-run harness,
	`DryRunMigrationJournalReplay`, which executes crash-injection replay checks
	against every declared journal boundary (`retry_started`, plus
	`step_started`/`step_committed` for each planned step) in isolated package
	copies. Each boundary run verifies interrupted replay state (`step_failed`)
	and resumed completion (`ok`) before returning.
- Runtime now also supports a blocking Gate 4 load-time check via
	`SetMigrationDryRunGateEnabled(true)`. When enabled, `LoadPackage` runs
	`DryRunMigrationJournalReplay` for migration-bearing packages and rejects
	load with `ErrMigrationDryRunFailed` if any replay boundary does not
	normalize and resume to `verdict = ok`.
- The same Gate 4 load-time check now rejects migration steps that declare
	`drain_policy = "drain"` without `compatibility = "incompatible"`, so drain
	dry-runs only cover migrations that actually require the drained execution
	path.
- Until read-adapter replay exists, the Gate 4 load-time check also rejects
	`drain_policy = "multi_version"` steps instead of allowing them through
	without proving the required backward-read contract.
- The `term` CLI now creates app runtimes with Gate 4 enabled by default for
	`app check`, `app load`, `app test`, local `app reload` fallback, and
	`sim run`, so local operator/developer flows reject migration-bearing
	packages whose replay dry-run cannot normalize to `verdict = ok`.
- Invalid runtime migration step plans (for example malformed script filenames,
	numbering gaps, or manifest `[[migrate.step]]` / script-count mismatches) now
	leave migration status with `executor_ready = false` and a specific
	`last_error`, preventing retries from running against ambiguous step plans.
- Runtime migration plan parsing now applies the same positive-value guard for
	`[migrate].max_runtime_seconds` and `[migrate].checkpoint_every`; invalid
	limits set `executor_ready = false` with a field-specific `last_error` and
	keep retry from executing.
- Retry now enforces `[migrate].max_runtime_seconds` as an execution budget
	around the runtime migration path. If the budget elapses before the run
	commits, retry fails with `ErrMigrationRuntimeTimeout`, keeps checkpoint
	progress at the last committed step, marks `verdict = step_failed`, and
	emits a `step_failed_timeout` journal entry with the configured budget.
- The deterministic fixture execution subset now treats
	`migrate.env.abort(reason)` calls as first-class executor aborts. Runtime
	retry fails the current step with `ErrMigrationAborted`, keeps checkpoint
	progress at the last committed step, marks `verdict = step_failed`, and
	emits `step_failed_aborted` journal evidence with the script-provided
	reason.
- Runtime retry now validates declared `artifact.self.patch(...)` host effects
	against the migrating package lineage when the script provides
	`owner_app_id`. Patch calls whose `owner_app_id` differs from the package
	`app_id` fail before step start with `ErrMigrationArtifactOwnership`, mark
	`verdict = step_failed`, and emit `step_failed_host_rejected` journal
	evidence. This prevents packages from patching artifacts owned by another
	lineage, including lineages that share the same manifest name.
- Retry now carries `[migrate].checkpoint_every` into the fixture-backed
	execution scaffold. When deterministic fixture transforms touch records,
	runtime treats each transformed fixture row as a synthetic store effect and
	emits `checkpoint_committed` journal evidence at the configured effect
	cadence before committing the step. Packages without fixture-backed effects
	continue to produce no checkpoint entries until the full durable store
	executor lands.

When `manifest.toml` declares `app_id`, migration journal paths are now rooted
under `apps/<app_id>/migrate/...` instead of `apps/<manifest_name>/...` so
runtime migration state remains anchored to lineage identity during key
rotation. Packages that omit `app_id` keep the existing manifest-name fallback.

These entries are written to the status-provided `journal_path` consumed by
`/admin/api/apps/migrate/logs` and `apps migrate logs`.

On package load, runtime now replays existing migration journal entries for the
current revision so `apps migrate status` resumes the last known
`verdict`/`steps_completed`/`last_step`/`last_error` instead of resetting to an
empty state after process restart. Drain-guard retries also replay
`blocked_since` so timeout windows continue across restart instead of resetting
to a fresh pending window.

If the last replayed journal state is `verdict = running` (for example, a
process crash after `step_started` but before `step_committed`), runtime now
normalizes status to `verdict = step_failed` with an explicit interruption
error. This keeps migration state operator-visible and retryable from the last
committed checkpoint instead of leaving status indefinitely in `running`.

Invalid layouts are rejected as `ErrInvalidManifest`.

## Test coverage

Validation coverage lives in [terminal_server/internal/apppackage/tap_test.go](../terminal_server/internal/apppackage/tap_test.go):

- `TestVerifyTapAcceptsCanonicalMigrateStepLayout`
- `TestVerifyTapRejectsMigrateStepNumberingGap`
- `TestVerifyTapRejectsMigrateDeclaredStepMismatch`
- `TestVerifyTapRejectsMigrateIncompatibleWithoutDrain`
- `TestVerifyTapAcceptsMigrateIncompatibleWithDrain`
- `TestVerifyTapRejectsMigrateFixtureTooManyRecords`
- `TestVerifyTapAcceptsMigrateFixtureAtRecordLimit`
- `TestRuntimeRetryMigrationRequiresDrainReadiness`
- `TestRuntimeMigrationLifecycleWithSteps`
- `TestRuntimeReloadMigrationStateStartsFromInstalledVersion`
- `TestRuntimeDrainPendingBlockedAtReplaysFromJournal`
- `TestRuntimeReconcileMigrationPendingRecords`
- `TestRuntimeInterruptedMigrationReplaysAsStepFailedAndResumes`
- `TestRuntimeRetryMigrationCrashInjectionReplaysAtJournalBoundaries`
	(covers retry/step journal-boundary interruption replay across first and
	later pending steps)
- `TestRuntimeDryRunMigrationJournalReplayExercisesAllBoundaries`
- `TestRuntimeDryRunMigrationJournalReplayReturnsEmptyWhenNoSteps`
- `TestRuntimeLoadPackageRejectsMigrationWhenDryRunGateFails`
- `TestRuntimeLoadPackageRejectsMultiVersionWithoutReadAdapterDuringDryRunGate`
- `TestRuntimeMigrationJournalPathUsesAppID`
- `TestRuntimeRetryMigrationFailsWhenPendingScriptInvalid`
- `TestRuntimeRetryMigrationIgnoresCommentedLoadStatements`
- `TestVerifyTapIgnoresCommentedDisallowedLoadStatements`
- `TestRuntimeRetryMigrationAppliesFixtureTransforms`
- `TestRuntimeRetryMigrationFailsWhenFixtureDeclarationMissingForPendingStep`
- `TestRuntimeRetryMigrationFailsWhenFixtureRecordLimitExceeded`
- `TestRuntimeRetryMigrationFailsWhenFixtureRecordNotCanonical`
- `TestRuntimeRetryMigrationFailsWhenFixturePathEscapesRoot`
- `TestRuntimeRetryMigrationFailsWhenFixturePathEscapesRootViaSymlink`
- `TestRuntimeRetryMigrationFailsWhenMaxRuntimeExceeded`
- `TestRuntimeRetryMigrationAbortCallFailsCurrentStep`
- `TestRuntimeRetryMigrationRejectsArtifactPatchForDifferentLineage`
- `TestRuntimeRetryMigrationEmitsCheckpointEveryForFixtureEffects`
- `TestAppsMigrateLogsUsesAdminAPIStepFilter`
- `TestAppsMigrateReconcileUsesAdminAPI`
- `TestExecuteCommandAppsMigrateUsageIncludesLogs`

## Not yet implemented

This does not yet implement the full migration executor lifecycle for durable stores and artifact patches. Runtime now enforces Gate 4 replay as a blocking load-time gate in both server startup defaults (via `newServerAppRuntime` in `terminal_server/cmd/server/main.go`) and `term` local app-runtime flows (`terminal_server/cmd/term/main.go`). The `term apps migrate *` operational APIs now call runtime-backed status/retry/abort/reconcile state transitions, migration modules are restricted at package verification time, retry executes a small deterministic `migrate(record)` fixture subset before expected-output comparison, retry honors `migrate.env.abort(reason)` in that subset, retry enforces the configured max-runtime budget around the current execution scaffold, runtime replay now has journal-boundary crash-injection coverage, retry emits checkpoint evidence for fixture-backed synthetic effects at `[migrate].checkpoint_every`, rollback enforces data-mode policy (`--keep-data` requires `migrate/downgrade/*.tal`; default mode is archive), and migration status now replays from journal state across restart (including interrupted-run normalization). Remaining executor work is tracked in [plans/features/app-migrations.md](../plans/features/app-migrations.md).
