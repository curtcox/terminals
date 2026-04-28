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
- When a step declares `compatibility = "incompatible"`, it cannot also declare `drain_policy = "none"`.

## Implemented runtime migration guard

The runtime migration control path in [terminal_server/internal/appruntime/runtime.go](../terminal_server/internal/appruntime/runtime.go) now enforces an incompatible-step drain guard:

- If any `[[migrate.step]]` declares `compatibility = "incompatible"` and `drain_policy = "drain"`, `RetryMigration` refuses to run until drain readiness is explicitly marked.
- A blocked retry returns `ErrMigrationDrainTimeout`, marks migration `verdict = "aborted"`, and preserves the current checkpoint (no step advancement while drain is unsafe).
- Operators/orchestrators can mark readiness through `SetMigrationDrainReady`, after which retry proceeds normally.

## Implemented migration runtime journaling

The runtime now emits structured NDJSON migration journal entries directly from
`terminal_server/internal/appruntime/runtime.go` when operators invoke
migration control actions:

- `RetryMigration` writes `retry_started` and `retry_committed` entries on
	successful runs, and emits `step_started`/`step_committed` entries for each
	migration step so operators can see step-by-step progression.
- Blocked retries emit explicit events (`retry_blocked_reconcile_pending` and
	`retry_blocked_drain_timeout`) with current verdict/step context.
- Retry reconciliation guard now treats `verdict = reconcile_pending` as
	blocking even if pending-record details are temporarily unavailable, so
	operators must reconcile before retry can proceed.
- `AbortMigration` writes `aborted` entries including the selected target
	(`checkpoint` or `baseline`).
- `ReconcileMigration` writes `reconcile_record` entries with `record_id` and
	selected `resolution`.
- Checkpoint abort now leaves migration state at `verdict = "step_failed"`
	with `last_step` pinned to the failed step and checkpoint progress preserved
	for `apps migrate retry`; baseline abort remains `verdict = "aborted"`.

Retry now resumes at the first incomplete step (`steps_completed + 1`) instead
of replaying the entire migration range on every retry.

These entries are written to the status-provided `journal_path` consumed by
`/admin/api/apps/migrate/logs` and `apps migrate logs`.

Invalid layouts are rejected as `ErrInvalidManifest`.

## Test coverage

Validation coverage lives in [terminal_server/internal/apppackage/tap_test.go](../terminal_server/internal/apppackage/tap_test.go):

- `TestVerifyTapAcceptsCanonicalMigrateStepLayout`
- `TestVerifyTapRejectsMigrateStepNumberingGap`
- `TestVerifyTapRejectsMigrateDeclaredStepMismatch`
- `TestVerifyTapRejectsMigrateIncompatibleWithoutDrain`
- `TestVerifyTapAcceptsMigrateIncompatibleWithDrain`
- `TestRuntimeRetryMigrationRequiresDrainReadiness`
- `TestRuntimeMigrationLifecycleWithSteps`
- `TestRuntimeReconcileMigrationPendingRecords`

## Not yet implemented

This does not yet implement the full migration executor or drain policy orchestration. The `term apps migrate *` operational APIs now call runtime-backed status/retry/abort/reconcile state transitions, migration modules are restricted at package verification time, and rollback now enforces data-mode policy (`--keep-data` requires `migrate/downgrade/*.tal`; default mode is archive). Remaining executor work is tracked in [plans/features/app-migrations.md](../plans/features/app-migrations.md).
