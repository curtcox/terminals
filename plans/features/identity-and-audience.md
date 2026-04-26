---
title: "Identity and Audience Plan"
kind: plan
status: superseded
owner: copilot
validation: none
last-reviewed: 2026-04-26
---

# Identity and Audience Plan

Status: Completed and drained on 2026-04-26.

The shipped identity/audience behavior from this plan is documented in:

- [`docs/repl/api/identity-service.md`](../../docs/repl/api/identity-service.md)
- [`docs/repl/commands/identity.md`](../../docs/repl/commands/identity.md)
- [`docs/repl/quickstart.md`](../../docs/repl/quickstart.md)
- [`docs/server.md`](../../docs/server.md)

Validation evidence and implementation coverage live in the server test suite,
including identity and acknowledgement behavior under:

- `terminal_server/internal/capability/service_test.go`
- `terminal_server/internal/repl/repl_test.go`

There are no remaining active tasks in this plan. Future identity-model
expansion work should be tracked as new scoped plans under `plans/features/`.
