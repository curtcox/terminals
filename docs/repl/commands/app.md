# app

- `app ls [--json]`
- `app logs <app> [<query>]`
- `app reload <app> [--json]`
- `app rollback <app> [--keep-data|--archive-data|--purge] [--json]`
- `apps migrate status <app> [--json]`
- `apps migrate logs <app> [--step <n>] [--json]`
- `apps migrate retry <app> [--json]`
- `apps migrate abort <app> [--to <checkpoint|baseline>] [--json]`
- `apps migrate reconcile <app> <record-id> <resolution> [--json]`

`apps migrate status` reports the migration verdict, step progress, `last_step`,
`last_error`, and pending reconciliation record IDs so operators can decide
whether to retry, abort, or reconcile.

`apps migrate logs` tails structured migration journal lines and supports
`--step <n>` to focus on one step's records. Entries are emitted by runtime
migration actions (`retry_*`, `aborted`, `reconcile_record`) and include step
and verdict context.

`app rollback` defaults to `--archive-data`. `--keep-data` is refused when the
rollback span has no `migrate/downgrade/*.tal` reverse steps.
