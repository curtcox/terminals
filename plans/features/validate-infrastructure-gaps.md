---
title: "Infrastructure Use Case Validation — Gaps (I1, I2, I5, I7–I11)"
kind: plan
status: planned
owner: curtcox
validation: none
last-reviewed: 2026-05-17
---

# Infrastructure Use Case Validation — Gaps

I3, I4, and I6 are already automated. This plan covers the eight remaining
infrastructure use cases. Most of the underlying behavior already exists;
the main work is writing harness tests and wiring each ID into
`make usecase-validate`.

## Use Cases in Scope

| ID | Description | Work type |
|----|-------------|-----------|
| I1 | mDNS discovery — client finds server automatically on LAN | Validation (discovery package exists) |
| I2 | Manual fallback — client connects via explicit address | Validation (already exercised in transport tests) |
| I5 | IO stream orchestration — consume, produce, fork, mix, record, analyze streams | Feature + validation (advanced routing not fully wired) |
| I7 | AI backend swap via config — local vs cloud LLM | Validation (interface exists; fake already used) |
| I8 | `make all-check` validates entire codebase | Validation (CI exists; needs use-case mapping) |
| I9 | CLAUDE.md + AGENTS.md exist and contain required sections | Validation (docs exist; needs a test) |
| I10 | New scenario added server-side, no client changes | Validation (design principle; verify with a trivial new scenario) |
| I11 | Reconnect restores previous UI state | Validation (UI9 covers the harness path; needs I11 mapping) |

## Per-ID Notes

### I1 — mDNS Discovery
The `terminal_server/internal/discovery` package implements mDNS advertising.
Write `TestUseCaseI1WithEvidence` that starts the server, browses via the
discovery client, and verifies the server address is returned without manual
configuration.

### I2 — Manual Fallback
Transport tests already connect via explicit address. Extract or alias the
relevant test as `TestUseCaseI2WithEvidence` (or write a minimal one) and
register it.

### I5 — IO Stream Orchestration
This is the most complex gap. The server can fork and mix audio via the IO
router, but a comprehensive orchestration test (record → analyze → re-route)
is missing.

Steps:
1. Identify the existing IO router path for fork/mix/record.
2. Write `TestUseCaseI5WithEvidence` that exercises:
   - Fork: one input routed to two outputs simultaneously.
   - Mix: two audio inputs merged into one output.
   - Record: stream captured to an in-memory buffer.
   - Analyze: fake classifier applied to captured stream.
3. Add new routing operations (composite, produce) if not yet implemented.

### I7 — AI Backend Swap
`FakeLLM` is already wired via `SetLLM`. Write
`TestUseCaseI7WithEvidence` that swaps from `FakeLLM` to a second `FakeLLM`
mid-session and verifies subsequent queries route to the new backend.

### I8 — CI Pipeline (`make all-check`)
I8 is a meta-use-case: the build system validates itself. Register it in
`usecase-validate.sh` as running `make all-check` (server build + test + lint +
proto). The "test" is that the command exits 0. No new Go test needed.

### I9 — Dev Agent Onboarding (CLAUDE.md)
Write `TestUseCaseI9WithEvidence` that:
- Opens `CLAUDE.md` from the repo root.
- Asserts it contains the required sections (Build and Check Commands,
  Core Rules, Bug Handling).
- Opens `AGENTS.md` if present and asserts it exists.

### I10 — Add Scenario Without Client Changes
Write `TestUseCaseI10WithEvidence` that registers a minimal new server-side
scenario at test time, connects a client terminal, triggers the scenario, and
verifies the client receives the scenario's output without any client-side code
change. This is structurally identical to existing harness tests — the test is
the proof.

### I11 — Reconnect + State Restore
UI9 already covers the harness path for reconnect restoring UI state. Map I11
to the same test (`TestUseCaseUI9WithEvidence`) via an alias test or a thin
`TestUseCaseI11WithEvidence` that delegates to the same assertions.

## Milestones

1. **M1** — I2, I7, I8, I9, I10, I11: each registered in validate script.
   These require little or no new code.
2. **M2** — I1 discovery test written and passing.
3. **M3** — I5 fork/mix/record test written; existing router behavior verified.
4. **M4** — I5 composite/produce path implemented if missing; full I5 test
   passing. All 8 IDs green in CI.
