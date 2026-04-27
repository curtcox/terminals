# `ui` commands

The `ui` group manages authored UI view inventory records used by REPL-driven scenarios.

## Commands

- `ui views ls [--json]`
- `ui views show <view-id> [--json]`
- `ui views rm <view-id> [--json]`

## Notes

- `ui views ls` prints a table by default and full JSON with `--json`.
- `ui views show` returns the stored authored-view record.
- `ui views rm` removes one authored-view record by id.
- View records are server-side metadata, not a new client primitive contract.
