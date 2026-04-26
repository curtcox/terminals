---
title: "Edge Execution and Operator Runtime"
kind: plan
status: superseded
owner: copilot
validation: none
last-reviewed: 2026-04-26
---

# Edge Execution and Operator Runtime
Status: Completed and drained on 2026-04-26.

The completed durable behavior from this plan now lives in:

- [../../docs/edge-execution-runtime.md](../../docs/edge-execution-runtime.md)
- [../../docs/observation-plane.md](../../docs/observation-plane.md)
- [../../docs/server.md](../../docs/server.md)
- [../../docs/sensing-use-case-flows.md](../../docs/sensing-use-case-flows.md)

Implementation references for this drained plan include:

- `api/terminals/capabilities/v1/capabilities.proto`
- `api/terminals/io/v1/io.proto`
- `terminal_server/internal/io/media_plan.go`
- `terminal_server/internal/transport/generated_proto_adapter.go`
- `terminal_client/lib/edge/`
- `terminal_client/lib/main.dart`

Future changes to edge execution should update the durable docs above and, when
needed, create a new focused plan rather than reopening this completed one.
