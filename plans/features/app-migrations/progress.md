---
title: "Application Migrations — Progress Log"
kind: progress-log
parent: plans/features/app-migrations/plan.md
---

## Implementation Progress

- 2026-05-01: Locked in §9 acceptance criterion 11 (drain timeout
  aborts) with journal-evidence assertions in
  `terminal_server/internal/appruntime/runtime_test.go`
  (`TestRuntimeRetryMigrationDrainTimeoutAbortsWithoutRunningStep`).
  The test verifies `retry_blocked_drain_pending` and
  `retry_blocked_drain_timeout` journal entries are emitted, and
  that no `retry_started` / `step_started` / `step_committed` /
  `retry_committed` events follow — proving the executor never runs
  the migration body while drain is unsatisfied. Existing
  `TestRuntimeRetryMigrationRequiresDrainReadiness` covered the
  verdict/error returns; this complements it on the audit-trail
  side.

_Entries before 2026-05: archived at [plans/archive/app-migrations/progress-2026-04.md](../../../archive/app-migrations/progress-2026-04.md)._
