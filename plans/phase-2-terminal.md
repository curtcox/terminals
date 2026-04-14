# Phase 2 — Text Terminal

See [masterplan.md](../masterplan.md) for overall system context.

First real use case. Validates keyboard input forwarding and text-based server-driven UI.

## Prerequisites

- [phase-1-foundation.md](phase-1-foundation.md) complete — basic gRPC control channel, capability reporting, and server-driven UI rendering exist.

## Deliverables

- [x] **PTY management**: Server spawns and manages pseudo-terminal sessions.
- [x] **Terminal UI descriptor**: Monospace scrollable text output + text input. Composed from existing primitives (see [server-driven-ui.md](server-driven-ui.md)).
- [x] **Keyboard forwarding**: Client sends key events, server feeds them to PTY. See `InputEvent` in [protocol.md](protocol.md).
- [x] **Terminal output**: Server captures PTY output, sends UI updates to client.

## Milestone

Use a Chromebook or laptop as a functional text terminal into the Mac mini.

## Related Plans

- [scenario-engine.md](scenario-engine.md) — Scenario interface this fits into.
- [use-case-flows.md](use-case-flows.md#text-terminal) — Flow detail.
- [io-abstraction.md](io-abstraction.md) — Keyboard category.
- [phase-3-media.md](phase-3-media.md) — Next phase.
