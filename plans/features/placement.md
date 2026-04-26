---
title: "Placement and World Model"
kind: plan
status: superseded
owner: copilot
validation: none
last-reviewed: 2026-04-25
---

# Placement and World Model

Status: Completed and drained on 2026-04-25.

The durable behavior from this plan is implemented and documented in:

- [`terminal_server/internal/placement/engine.go`](../../terminal_server/internal/placement/engine.go)
- [`terminal_server/internal/device/manager.go`](../../terminal_server/internal/device/manager.go)
- [`terminal_server/internal/scenario/runtime.go`](../../terminal_server/internal/scenario/runtime.go)
- [`terminal_server/internal/placement/engine_test.go`](../../terminal_server/internal/placement/engine_test.go)
- [`docs/server.md`](../../docs/server.md)

There are no remaining active tasks in this plan. Future placement expansion
(for example richer world-model geometry and calibration flows) should live in
the dedicated world-model and sensing plans.

See [masterplan.md](../archive/masterplan-duplicate.md) for overall system context.
