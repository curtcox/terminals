---
title: "Phase 2 — Text Terminal"
kind: plan
status: superseded
owner: copilot
validation: automated:P1
last-reviewed: 2026-04-25
---

# Phase 2 — Text Terminal

Status: Completed and drained on 2026-04-25.

The durable behavior from this phase now lives in:

- [`docs/server.md`](../../docs/server.md) (text terminal runtime behavior and operations)
- [`docs/usecase-validation-matrix.md`](../../docs/usecase-validation-matrix.md) (automated validation mapping for `P1`)
- [`usecases.md`](../../usecases.md) (`P1` and related terminal/productivity use cases)
- [`plans/features/use-case-flows.md`](../features/use-case-flows.md#text-terminal) (flow-level scenario detail)

There are no remaining active tasks in this phase plan. Future terminal work should
be tracked under the feature plans and corresponding use-case automation tasks.

See [masterplan.md](../archive/masterplan-duplicate.md) for overall system context.

First real use case. Validates keyboard input forwarding and text-based server-driven UI.

## Prerequisites

- [phase-1-foundation.md](phase-1-foundation.md) complete — basic gRPC control channel, capability reporting, and server-driven UI rendering exist.

## Deliverables

- [x] **Scenario definition/activation split**: Introduce `ScenarioDefinition` and `ScenarioActivation` with a trivial engine that can match a trigger and start/stop a single activation. See [scenario-engine.md](../features/scenario-engine.md#definitions-vs-activations). The terminal scenario is the first definition; each PTY session becomes its own activation so multiple terminals are natural.
- [x] **PTY management**: Server spawns and manages pseudo-terminal sessions (one per activation once the split above lands).
- [x] **Terminal UI descriptor**: Monospace scrollable text output + text input. Composed from existing primitives (see [server-driven-ui.md](../features/server-driven-ui.md)).
- [x] **Keyboard forwarding**: Client sends key events, server feeds them to the PTY. See `InputEvent` in [protocol.md](../features/protocol.md).
- [x] **Terminal output**: Server captures PTY output, sends UI updates to the client.

## Milestone

Use a Chromebook or laptop as a functional text terminal into the Mac mini.

## Related Plans

- [scenario-engine.md](../features/scenario-engine.md) — Definition/activation split this fits into.
- [use-case-flows.md](../features/use-case-flows.md#text-terminal) — Flow detail.
- [io-abstraction.md](../features/io-abstraction.md) — Keyboard category.
- [phase-3-media.md](phase-3-media.md) — Next phase.
