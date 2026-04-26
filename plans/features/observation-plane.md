---
title: "Observation Plane and Flow Plans"
kind: plan
status: superseded
owner: copilot
validation: none
last-reviewed: 2026-04-26
---

# Observation Plane and Flow Plans
Status: Completed and drained on 2026-04-26.

The durable behavior from this plan is implemented and documented in:

- [../../api/terminals/io/v1/io.proto](../../api/terminals/io/v1/io.proto)
- [../../api/terminals/control/v1/control.proto](../../api/terminals/control/v1/control.proto)
- [../../terminal_server/internal/io/media_plan.go](../../terminal_server/internal/io/media_plan.go)
- [../../terminal_server/internal/io/media_plan_test.go](../../terminal_server/internal/io/media_plan_test.go)
- [../../terminal_server/internal/transport/control_stream.go](../../terminal_server/internal/transport/control_stream.go)
- [../../terminal_server/internal/transport/generated_proto_adapter.go](../../terminal_server/internal/transport/generated_proto_adapter.go)
- [../../terminal_server/internal/transport/generated_proto_adapter_test.go](../../terminal_server/internal/transport/generated_proto_adapter_test.go)
- [../../docs/observation-plane.md](../../docs/observation-plane.md)
- [../../docs/sensing-use-case-flows.md](../../docs/sensing-use-case-flows.md)
- [../../docs/server.md](../../docs/server.md)

There are no remaining active tasks in this plan. Future observation-plane
expansion (for example additional flow operators or richer artifact retention
policy behavior) should be scoped in focused runtime plans.

See [masterplan.md](../archive/masterplan-duplicate.md) for overall system context.
