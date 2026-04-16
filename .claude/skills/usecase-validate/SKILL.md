---
name: usecase-validate
description: Run the automated validation for a use case ID from usecases.md (e.g. "validate C1", "does M3 still pass?", "run the use-case gate", "validate all use cases"). Read-only — executes tests via `make usecase-validate` and reports results. Does NOT write code or add new ID mappings; for that, use the usecase-implement skill.
---

# Validate a Use Case

The project maps use-case IDs in [usecases.md](../../../usecases.md) to concrete Go test invocations through [scripts/usecase-validate.sh](../../../scripts/usecase-validate.sh). This skill runs those tests and reports results. It is read-only: if the requested ID isn't already automated, hand off to the `usecase-implement` skill — do not invent mappings here.

## Currently automated IDs

`C1`, `C3`, `C5`, `D1`, `M1`, `M3`, `M4`, `S1`, `S2`, `S3`, `P1`, `T1`.

`USECASE=all` runs exactly this set (see the `ids=(…)` array at the bottom of [scripts/usecase-validate.sh](../../../scripts/usecase-validate.sh)). All other IDs — every `C2/C4/C6/V*/T2–T4/M2/M5/D2/D3/P2/I*` and every adjacent `AH*/AO*/AB*/AA*` — are **not** automated and will exit 2.

The authoritative mapping is the `run_usecase` switch in [scripts/usecase-validate.sh](../../../scripts/usecase-validate.sh). [docs/usecase-validation-matrix.md](../../../docs/usecase-validation-matrix.md) mirrors it for human reference — if they disagree, the script wins.

## Procedure

1. **Identify the ID.** If the user gave a phrase ("red alert"), map it to an ID by reading [usecases.md](../../../usecases.md). Confirm with the user before running.
2. **Check coverage.** If the ID is not in the list above (or the user asked for an `AH*`/`AO*`/`AB*`/`AA*` ID), stop and tell them it has no automated validation. Offer:
   - `make all-check` as the baseline repository gate, or
   - handing off to `usecase-implement` to add the mapping.
3. **Run it.** From the repo root:
   ```bash
   make usecase-validate USECASE=<ID>
   # or
   make usecase-validate USECASE=all
   ```
   No server needs to be running — the script invokes `go test` directly against packages under `terminal_server/`.
4. **On pass:** report the ID(s), the test pattern(s) that ran (copy from the `==> go test …` line), and "passed".
5. **On failure:** report the failing test name and the first assertion failure from the output. Then debug — do **not** mutate the test to make it pass:
   - Re-read the target row in [usecases.md](../../../usecases.md) to confirm intended behavior.
   - Locate the test source under `terminal_server/internal/transport/` or `terminal_server/cmd/server/` and read the assertions.
   - Inspect recent changes to the implicated server package (`git log -p -- <path>`).
   - If the failure reflects a real regression, surface it for the user to decide how to proceed; if the test itself is stale, flag it as a separate task (via `spawn_task` or by asking the user).

## Out of scope for this skill

- Adding a new ID to the switch or to the `ids=()` array.
- Editing [docs/usecase-validation-matrix.md](../../../docs/usecase-validation-matrix.md).
- Writing new integration tests.
- Any protobuf, Go, or Flutter code changes.

All of the above are the job of the `usecase-implement` skill.
