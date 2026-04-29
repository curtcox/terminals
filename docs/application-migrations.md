# Application Migrations

Durable reference for currently implemented migration validation behavior.
This is the shipped subset from [plans/features/app-migrations.md](../plans/features/app-migrations.md).

## Implemented Gate 1 checks

The .tap package verifier in [terminal_server/internal/apppackage/tap.go](../terminal_server/internal/apppackage/tap.go) now validates migration layout at package-load time:

- If no `migrate/*.tal` files exist, migration declarations must be absent (`declared_steps = 0` and no `[[migrate.step]]`).
- If migration files exist, `[migrate].declared_steps`, `[[migrate.step]]` count, and migration file count must match.
- Migration files must be one level under `migrate/` and match `<step>_<from>_to_<to>.tal`.
- Reverse migration scripts under `migrate/downgrade/*.tal` are allowed for rollback flows, but must remain single-level files (nested downgrade folders are rejected) and match `<step>_<from>_to_<to>.tal`.
- Step numbers must be contiguous and start at `1` (no gaps).
- Each sorted migration file must match the corresponding manifest step `from`/`to` values.
- Each `[[migrate.step]]` must explicitly declare both `compatibility` and
	`drain_policy`; missing policy fields are rejected instead of defaulting to a
	live-data behavior.
- Invalid migration script paths now return specific diagnostics, including nested path violations under `migrate/` / `migrate/downgrade/` and malformed forward or downgrade step filenames that do not match `<step>_<from>_to_<to>.tal`.
- When a step declares `compatibility = "incompatible"`, it cannot also declare `drain_policy = "none"`.
- Migration fixture NDJSON files are bounded to at most 4096 records per file to keep Gate 4 synthetic-store input sizes predictable.
- Migration seed fixture records are validated against each fixture's declared `prior_record_schema`; invalid seed rows now fail package verification with record-level diagnostics.
- Migration expected fixture records are validated against the target step record schema when a unique `[[storage.store_schema]]` entry exists for the step `to` version; invalid expected rows now fail package verification with record-level diagnostics.
- Migration fixture metadata now enforces step-edge consistency: `[[migrate.fixture]].prior_version` must match the corresponding migration script `from` version (`migrate/<step>_<from>_to_<to>.tal`).
- `multi_version` migration fixtures must declare a `read_adapter` script. The package verifier checks that the adapter file is present, non-empty, exposes `read(record)`, and only loads migration-safe modules.
- When declared, `[migrate].drain_timeout_seconds`, `[migrate].max_runtime_seconds`, and `[migrate].checkpoint_every` must be positive integers; non-positive values now fail Gate 1 with explicit diagnostics.

For `multi_version`, the fixture declaration adds the adapter path:

```toml
[[migrate.fixture]]
step = "0001_1_to_2"
prior_version = "1"
prior_record_schema = "schemas/history_v1.json"
seed = "tests/migrate_fixtures/history_v1_seed.ndjson"
expected = "tests/migrate_fixtures/history_v2_expected.ndjson"
read_adapter = "tests/migrate_fixtures/read_v2_as_v1.tal"
```

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
- Fixture-backed migration execution now journals accepted `log.*(...)` calls
	as `migration_log` entries with level, message, raw arguments, and step
	metadata, so `apps migrate logs` surfaces TAL migration log intent instead
	of silently treating those calls as no-ops.
- Blocked retries emit explicit events (`retry_blocked_reconcile_pending` and
	`retry_blocked_drain_pending` / `retry_blocked_drain_timeout`) with current
	verdict/step context and `blocked_since` timing metadata.
- Drain-readiness changes made through runtime/admin/REPL control surfaces now
	emit `drain_ready_changed` entries. Journal replay restores the latest
	ready/not-ready value so an operator-approved drain does not regress to
	`drain_pending` after a process restart.
- Migration status now exposes drain control-plane fields through runtime,
	admin API, and human-readable REPL output: whether the pending step still
	requires a drain, whether drain readiness is approved, the configured drain
	timeout in seconds, and the current `drain_pending` blocked-since timestamp
	when one exists.
- Retry reconciliation guard now treats `verdict = reconcile_pending` as
	blocking even if pending-record details are temporarily unavailable, so
	operators must reconcile before retry can proceed.
- `AbortMigration` writes `aborted` entries including the selected target
	(`checkpoint` or `baseline`).
- Baseline abort now scans migration journal evidence for unresolved
	`artifact_inverse_failed` entries. If any inverse failure remains unresolved,
	the runtime returns `ErrMigrationReconcilePending`, rewinds step progress to
	baseline, preserves pending records and `reconciliation_path`, emits a
	`reconcile_pending` journal entry, and replays that state across restart.
- `ReconcileMigration` writes `reconcile_record` entries with `record_id` and
	selected `resolution`.
- Reconcile operations now report `ErrMigrationReconcilePending` when no
	pending records exist, including apps with no runnable migration steps,
	so operators see one consistent "nothing to reconcile yet" response.
- Checkpoint abort now leaves migration state at `verdict = "step_failed"`
	with `last_step` pinned to the failed step and checkpoint progress preserved
	for `apps migrate retry`; baseline abort remains `verdict = "aborted"`.
	If the abort targets an in-flight step, previously committed steps remain
	committed; aborting an already-committed terminal state rewinds the most
	recent completed step for retry.

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
	string trimming, `lower(trim(record["field"]))`, `record.get("field",
	default)` JSON literal defaults in those same assignment forms, JSON literal
	assignment, field deletion, idempotent `if "field" in record: continue`
	guards, and no-op structured `log` calls through loaded
	`debug`/`info`/`warn`/`error` aliases. Mismatches stop retry with
	`ErrMigrationFixtureMismatch`, mark `verdict = step_failed`, and emit
	`step_failed_fixture_mismatch` entries.
	The parser preserves `#` characters inside both double-quoted and
	single-quoted TAL string literals before stripping line comments, so
	accepted single-quoted log messages and `record.get(...)` defaults can
	include hash-prefixed labels without being truncated as comments.
	The deterministic fixture subset also supports the worked-example paged
	`store` loop shape: loaded `list_keys`/`get`/`put`/`delete` aliases, a literal
	`prefix` scan, `rec = get(key)`, idempotent presence guards,
	`rec[...]` assignment transforms including `_normalize(rec.get(...))`,
	`put(key, rec)`, `delete(key)`, no-op loaded `checkpoint` calls, and no-op
	structured log calls. Fixture replay applies the transform only to matching
	keys and counts successful `put`/`delete` calls as synthetic store effects
	for checkpoint evidence and hard-cap accounting.
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
- Runtime fixture reads also re-check key constraints at execution time:
	keys must be valid UTF-8, NFC-normalized, and 1..256 bytes. Post-load
	fixture mutations that violate those constraints fail retry with
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
	Runtime fixture lookup accepts the package-format step identifier
	(`<step>_<from>_to_<to>`, for example `0001_1_to_2`) used by Gate 1
	verification, with numeric identifiers such as `0001` retained as a local
	compatibility fallback.
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
- Rollback data-mode policy is explicit and non-mutating on rejection:
	`--keep-data` requires a valid `migrate/downgrade/*.tal` reverse step, while
	`--archive-data` remains allowed without reverse steps. A failed keep-data
	rollback leaves package history untouched so operators can immediately retry
	the rollback with archive-data.
- Rollback is also blocked while the current migration is
	`reconcile_pending`. The admin rollback API returns HTTP 409 with the current
	migration status payload, including pending reconciliation records and
	`reconciliation_path`, so operators have the same follow-up context available
	from `apps migrate status`.
- Reload across author-key rotation keeps migration state anchored to the
	stable `app_id` lineage while still using the installed package version to
	compute the pending version window. A rotated package that keeps the same
	`app_id` and upgrades from version `2` to `3` runs only the `2 -> 3` step;
	the migration journal remains under `apps/<app_id>/migrate/...`.
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
- Gate 4 now supports fixture-backed `multi_version` read-adapter replay. Each
	`multi_version` step must declare `read_adapter`; runtime executes the
	adapter's `read(record)` deterministic subset against the migrated
	`expected` fixture records and verifies the adapted records match the
	prior-version `seed` records before the package can load under the dry-run
	gate. Runtime resolves read-adapter paths with the same package-root and
	symlink checks used for seed and expected fixture files, so adapters cannot
	escape the package payload during dry-run replay.
- The `term` CLI now creates app runtimes with Gate 4 enabled by default for
	`app check`, `app load`, `app test`, local `app reload` fallback, and
	`sim run`, so local operator/developer flows reject migration-bearing
	packages whose replay dry-run cannot normalize to `verdict = ok`.
- Invalid runtime migration step plans (for example malformed script filenames,
	numbering gaps, or manifest `[[migrate.step]]` / script-count mismatches) now
	leave migration status with `executor_ready = false` and a specific
	`last_error`, preventing retries from running against ambiguous step plans.
- Runtime migration plan parsing now applies the same positive-value guard for
	`[migrate].drain_timeout_seconds`, `[migrate].max_runtime_seconds`, and
	`[migrate].checkpoint_every`; invalid
	limits set `executor_ready = false` with a field-specific `last_error` and
	keep retry from executing.
- Runtime migration plan parsing also mirrors Gate 1 policy validation for
	`[[migrate.step]].compatibility`, `drain_policy`, and the
	`incompatible + none` combination. Missing `compatibility` or `drain_policy`
	fields are treated as policy errors as well. If package contents drift after
	verification, migration status remains visible but `executor_ready = false`
	with the specific policy error.
- Retry now enforces `[migrate].max_runtime_seconds` as a per-step execution
	budget. If the budget elapses before the current step commits, retry fails
	with `ErrMigrationRuntimeTimeout`, keeps checkpoint progress at the last
	committed step, marks `verdict = step_failed`, and emits a
	`step_failed_timeout` journal entry with the configured budget.
- The deterministic fixture execution subset now treats
	`migrate.env.abort(reason)` calls, including calls through loaded aliases,
	as first-class executor aborts. Runtime
	retry fails the current step with `ErrMigrationAborted`, keeps checkpoint
	progress at the last committed step, marks `verdict = step_failed`, and
	emits `step_failed_aborted` journal evidence with the script-provided
	reason.
- Runtime retry now validates declared `artifact.self.patch(...)` host effects
	against the migrating package lineage using explicit artifact ID and
	`owner_app_id` evidence from the script. Patch calls that omit a literal,
	non-empty artifact ID, omit `owner_app_id`, or whose `owner_app_id` differs
	from the package `app_id`, fail before step start with
	`ErrMigrationArtifactOwnership`, mark `verdict = step_failed`, and emit
	`step_failed_host_rejected` journal evidence. This prevents packages from
	patching artifacts without host-checkable artifact and lineage evidence or
	patching artifacts owned by another lineage, including lineages that share
	the same manifest name.
	Accepted patch declarations now emit `artifact_patch_planned` journal
	evidence with artifact ID, owner app ID, effect sequence, step, version
	edge, and script metadata before the step commits. This is declaration
	evidence for the current scaffold; durable artifact patch execution still
	belongs to the remaining executor work.
- Runtime retry now enforces the plan's hard migration resource caps in the
	current execution scaffold. Fixture-backed store effects are counted against
	the 1,000,000 store-op and 100 MB write-volume per-step caps, and declared
	`artifact.self.patch(...)` calls are counted against the 10,000 patch cap
	before step execution starts. Cap violations fail the step with
	`ErrMigrationResourceLimit`, keep checkpoint progress at the last committed
	step, and emit `step_failed_resource_limit` journal evidence.
- Retry now carries `[migrate].checkpoint_every` into the fixture-backed
	execution scaffold. When deterministic fixture transforms change records,
	runtime treats each changed fixture row as a synthetic store effect and
	emits `checkpoint_committed` journal evidence at the configured effect
	cadence before committing the step. Rows skipped by an idempotency guard are
	not counted as writes. Packages without fixture-backed effects continue to
	produce no checkpoint entries until the full durable store executor lands.
- The Gate 4 crash-replay harness now treats fixture-backed
	`checkpoint_committed` entries as interruptible journal boundaries. Dry-run
	replay only injects this boundary for steps whose fixture execution actually
	emits checkpoint evidence, then reloads from the journal and verifies retry
	resumes to `verdict = ok`.

When `manifest.toml` declares `app_id`, migration journal paths are now rooted
under `apps/<app_id>/migrate/...` instead of `apps/<manifest_name>/...` so
runtime migration state remains anchored to lineage identity during key
rotation. Packages that omit `app_id` keep the existing manifest-name fallback.

These entries are written to the status-provided `journal_path` consumed by
`/admin/api/apps/migrate/logs` and `apps migrate logs`.

Human-readable `apps migrate status` output now includes `last_step`,
`last_error`, sorted pending records as
`record_id:recommended_resolution`, and `reconciliation_path` so an operator
has the record IDs, suggested resolution policy, and durable reconcile file
needed before invoking `apps migrate reconcile`.

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
- `TestVerifyTapRejectsMigrateNonPositiveDrainTimeoutSeconds`
- `TestVerifyTapRejectsMigrateIncompatibleWithoutDrain`
- `TestVerifyTapRejectsMigrateStepMissingPolicy`
- `TestVerifyTapAcceptsMigrateIncompatibleWithDrain`
- `TestVerifyTapRejectsMigrateFixtureTooManyRecords`
- `TestVerifyTapAcceptsMigrateFixtureAtRecordLimit`
- `TestVerifyTapRequiresReadAdapterForMultiVersionMigration`
- `TestVerifyTapAcceptsMultiVersionReadAdapter`
- `TestRuntimeRetryMigrationRequiresDrainReadiness`
- `TestRuntimeMigrationLifecycleWithSteps`
	(covers checkpoint aborts for both completed-state rewind and in-flight
	step preservation)
- `TestRuntimeAbortBaselineEntersReconcilePendingWhenArtifactInverseFails`
- `TestRuntimeReloadMigrationStateStartsFromInstalledVersion`
- `TestRuntimeDrainPendingBlockedAtReplaysFromJournal`
- `TestRuntimeDrainReadyReplaysFromJournal`
- `TestRuntimeReconcileMigrationPendingRecords`
	(covers replaying a fully resolved reconciliation journal back to `ok`)
- `TestRuntimeInterruptedMigrationReplaysAsStepFailedAndResumes`
- `TestRuntimeRetryMigrationCrashInjectionReplaysAtJournalBoundaries`
	(covers retry/step journal-boundary interruption replay across first and
	later pending steps)
- `TestRuntimeDryRunMigrationJournalReplayExercisesAllBoundaries`
- `TestRuntimeDryRunMigrationJournalReplayReturnsEmptyWhenNoSteps`
- `TestRuntimeLoadPackageRejectsMigrationWhenDryRunGateFails`
- `TestRuntimeLoadPackageRejectsMultiVersionWithoutReadAdapterDuringDryRunGate`
- `TestRuntimeLoadPackageValidatesMultiVersionReadAdapterDuringDryRunGate`
- `TestRuntimeLoadPackageRejectsMultiVersionReadAdapterEscapingRootDuringDryRunGate`
- `TestRuntimeMigrationJournalPathUsesAppID`
- `TestRuntimeRetryMigrationFailsWhenPendingScriptInvalid`
- `TestRuntimeRetryMigrationIgnoresCommentedLoadStatements`
- `TestVerifyTapIgnoresCommentedDisallowedLoadStatements`
- `TestRuntimeRetryMigrationAppliesFixtureTransforms`
- `TestRuntimeRetryMigrationAppliesTrimFixtureTransforms`
- `TestRuntimeRetryMigrationAppliesRecordGetFixtureTransforms`
- `TestRuntimeRetryMigrationPreservesHashInSingleQuotedStrings`
- `TestRuntimeRetryMigrationAppliesIdempotentFixtureGuard`
	(covers that idempotently skipped fixture rows do not count as synthetic
	store effects for checkpoint evidence)
- `TestRuntimeRetryMigrationAppliesPagedStoreFixtureTransforms`
- `TestRuntimeRetryMigrationAllowsLogCallsInFixtureTransforms`
- `TestRuntimeRetryMigrationFailsWhenFixtureDeclarationMissingForPendingStep`
- `TestRuntimeRetryMigrationFailsWhenFixtureRecordLimitExceeded`
- `TestRuntimeRetryMigrationFailsWhenFixtureRecordNotCanonical`
- `TestRuntimeRetryMigrationFailsWhenFixtureKeyInvalid`
- `TestRuntimeRetryMigrationFailsWhenFixturePathEscapesRoot`
- `TestRuntimeRetryMigrationFailsWhenFixturePathEscapesRootViaSymlink`
- `TestRuntimeRetryMigrationFailsWhenMaxRuntimeExceeded`
- `TestRuntimeRetryMigrationAppliesMaxRuntimePerStep`
- `TestRuntimeRetryMigrationAbortCallFailsCurrentStep`
- `TestRuntimeRetryMigrationRejectsArtifactPatchForDifferentLineage`
- `TestRuntimeRetryMigrationRejectsArtifactPatchWithoutOwnerAppID`
- `TestRuntimeRetryMigrationRejectsArtifactPatchWithoutArtifactID`
- `TestRuntimeRetryMigrationJournalsAcceptedArtifactPatchDeclarations`
- `TestRuntimeRetryMigrationRejectsArtifactPatchHardCap`
- `TestRuntimeMigrationResourceLimitValidation`
- `TestRuntimeRetryMigrationEmitsCheckpointEveryForFixtureEffects`
- `TestRuntimeReloadMigrationAfterKeyRotationUsesAppIDAndPendingVersionWindow`
- `TestAppsRollbackBlockedByReconcilePendingReturnsMigrationStatus`
- `TestAppsMigrateLogsUsesAdminAPIStepFilter`
- `TestAppsMigrateReconcileUsesAdminAPI`
- `TestAppsMigrateStatusUsesAdminAPI`
- `TestExecuteCommandAppsMigrateUsageIncludesLogs`

## Not yet implemented

This does not yet implement the full migration executor lifecycle for durable stores and artifact patches. Runtime now enforces Gate 4 replay as a blocking load-time gate in both server startup defaults (via `newServerAppRuntime` in `terminal_server/cmd/server/main.go`) and `term` local app-runtime flows (`terminal_server/cmd/term/main.go`). The `term apps migrate *` operational APIs now call runtime-backed status/retry/abort/reconcile state transitions, migration modules are restricted at package verification time, retry executes small deterministic fixture subsets for `migrate(record)` scripts and the worked-example paged `store` loop before expected-output comparison, `multi_version` steps must prove a fixture-backed `read_adapter` can recover the prior fixture shape from migrated records, retry honors `migrate.env.abort(reason)` in that subset, retry enforces the configured max-runtime budget and hard resource caps around the current execution scaffold, runtime replay now has journal-boundary crash-injection coverage, retry emits checkpoint evidence for fixture-backed synthetic effects at `[migrate].checkpoint_every`, baseline abort preserves unresolved artifact inverse failures as `reconcile_pending`, rollback enforces data-mode policy (`--keep-data` requires `migrate/downgrade/*.tal`; default mode is archive), and migration status now replays from journal state across restart (including interrupted-run normalization and reconciliation records). Remaining executor work is tracked in [plans/features/app-migrations.md](../plans/features/app-migrations.md).
