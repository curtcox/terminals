---
title: "Scenario Engine"
kind: plan
status: superseded
owner: copilot
validation: none
last-reviewed: 2026-04-26
---

# Scenario Engine

Status: Completed and drained on 2026-04-26.

The durable behavior from this plan is implemented and documented in:

- [`terminal_server/internal/scenario/scenario.go`](../../terminal_server/internal/scenario/scenario.go)
- [`terminal_server/internal/scenario/engine.go`](../../terminal_server/internal/scenario/engine.go)
- [`terminal_server/internal/scenario/runtime.go`](../../terminal_server/internal/scenario/runtime.go)
- [`terminal_server/internal/scenario/trigger_bus.go`](../../terminal_server/internal/scenario/trigger_bus.go)
- [`terminal_server/internal/scenario/recipe.go`](../../terminal_server/internal/scenario/recipe.go)
- [`terminal_server/internal/scenario/engine_test.go`](../../terminal_server/internal/scenario/engine_test.go)
- [`terminal_server/internal/scenario/runtime_test.go`](../../terminal_server/internal/scenario/runtime_test.go)
- [`terminal_server/internal/scenario/trigger_bus_test.go`](../../terminal_server/internal/scenario/trigger_bus_test.go)
- [`docs/server.md`](../../docs/server.md)
- [`docs/event-taxonomy.md`](../../docs/event-taxonomy.md)

There are no remaining active tasks in this plan. Future scenario-engine
expansion (for example richer activation record persistence and TAR-driven
scenario composition) should be scoped in focused runtime plans.

See [masterplan.md](../archive/masterplan-duplicate.md) for overall system context.
