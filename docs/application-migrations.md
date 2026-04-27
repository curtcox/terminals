# Application Migrations

Durable reference for currently implemented migration validation behavior.
This is the shipped subset from [plans/features/app-migrations.md](../plans/features/app-migrations.md).

## Implemented Gate 1 checks

The .tap package verifier in [terminal_server/internal/apppackage/tap.go](../terminal_server/internal/apppackage/tap.go) now validates migration layout at package-load time:

- If no `migrate/*.tal` files exist, migration declarations must be absent (`declared_steps = 0` and no `[[migrate.step]]`).
- If migration files exist, `[migrate].declared_steps`, `[[migrate.step]]` count, and migration file count must match.
- Migration files must be one level under `migrate/` and match `<step>_<from>_to_<to>.tal`.
- Step numbers must be contiguous and start at `1` (no gaps).
- Each sorted migration file must match the corresponding manifest step `from`/`to` values.

Invalid layouts are rejected as `ErrInvalidManifest`.

## Test coverage

Validation coverage lives in [terminal_server/internal/apppackage/tap_test.go](../terminal_server/internal/apppackage/tap_test.go):

- `TestVerifyTapAcceptsCanonicalMigrateStepLayout`
- `TestVerifyTapRejectsMigrateStepNumberingGap`
- `TestVerifyTapRejectsMigrateDeclaredStepMismatch`

## Not yet implemented

This does not yet implement the migration executor, drain policy orchestration, rollback/reconcile flow, migration runtime module restrictions, or `term apps migrate *` operational APIs. Those remain tracked in [plans/features/app-migrations.md](../plans/features/app-migrations.md).
