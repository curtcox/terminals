# `ui` commands

The `ui` group manages authored view inventory and active UI operations for REPL-driven scenarios.

## Commands

- `ui push <device> <descriptor-expr> [--root <id>] [--json]`
- `ui patch <device> <component-id> <descriptor-expr> [--json]`
- `ui transition <device> <component-id> <transition> [--duration-ms <n>] [--json]`
- `ui broadcast <cohort> <descriptor-expr> [--patch <component-id>] [--json]`
- `ui subscribe <device> --to <activation|cohort> [--json]`
- `ui snapshot <device> [--json]`
- `ui views ls [--json]`
- `ui views show <view-id> [--json]`
- `ui views rm <view-id> [--json]`

## Notes

- `descriptor-expr` is passed as raw text to the server-side authored UI store.
- `ui broadcast` resolves cohort members using the named cohort selectors and records fan-out to each resolved device.
- `ui snapshot` returns the latest authored push/patch/transition state for one device.
- `ui views ls` prints a table by default and full JSON with `--json`.
- `ui views show` returns the stored authored-view record.
- `ui views rm` removes one authored-view record by id.
- View records are server-side metadata, not a new client primitive contract.
