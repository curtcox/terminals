---
title: "Application Runtime"
kind: plan
status: superseded
owner: copilot
validation: none
last-reviewed: 2026-04-26
---

# Application Runtime

Status: Completed and drained on 2026-04-26.

The durable behavior from this plan is implemented and documented in:

- [`terminal_server/internal/appruntime/runtime.go`](../../terminal_server/internal/appruntime/runtime.go)
- [`terminal_server/internal/appruntime/definitions.go`](../../terminal_server/internal/appruntime/definitions.go)
- [`terminal_server/internal/appruntime/runtime_test.go`](../../terminal_server/internal/appruntime/runtime_test.go)
- [`terminal_server/cmd/term/main.go`](../../terminal_server/cmd/term/main.go)
- [`terminal_server/cmd/term/main_test.go`](../../terminal_server/cmd/term/main_test.go)
- [`docs/application-runtime.md`](../../docs/application-runtime.md)
- [`docs/tal-example-kitchen-timer.md`](../../docs/tal-example-kitchen-timer.md)
- [`docs/server.md`](../../docs/server.md)

There are no remaining active tasks in this plan. Future TAL interpreter,
simulation depth, and activation snapshot persistence work should be scoped in
focused runtime plans instead of reopening this drained baseline plan.

See [masterplan.md](../archive/masterplan-duplicate.md) for overall system context.
