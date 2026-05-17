---
title: "Adjacent Office Use Case Validation (AO1–AO7)"
kind: plan
status: planned
owner: curtcox
validation: none
last-reviewed: 2026-05-17
---

# Adjacent Office Use Case Validation

All seven AO use cases map to validated core scenarios. The work is writing
focused harness tests in an office context and registering them in
`make usecase-validate`.

## Use Cases in Scope

| ID | Description | Maps to | Work type |
|----|-------------|---------|-----------|
| AO1 | Cross-office intercom | C1 (intercom) | Validation |
| AO2 | Camera grid on dedicated monitor | S1–S3 (security/camera) | Validation |
| AO3 | Building-wide PA broadcast | C3 (PA mode) | Validation |
| AO4 | Shared screen/agenda on conference room device | Server-driven UI / D | Validation |
| AO5 | Terminal sessions to administer server from any device | P2–P4 (REPL) | Validation |
| AO6 | Idle lobby display with company branding | D1 (photo-frame / display) | Validation |
| AO7 | Persistent intercom channel between two team rooms | C1 extension (persistent route) | Validation |

## Approach

Write one `TestUseCaseAO<N>WithEvidence` per ID in
`terminal_server/internal/usecasevalidation/ao_test.go`.

### AO1 — Cross-Office Intercom
Two devices: `office_a` and `conference_room_b`. Trigger intercom intent.
Verify two-way audio route established. Identical to `c1_test` pattern.

### AO2 — Camera Grid on Dedicated Monitor
Register entry-point camera devices plus a `reception_monitor`. Trigger
multi-window camera view. Verify grid UI sent to reception monitor.
Pattern: `s1_test`.

### AO3 — Building-Wide PA
Register all-office devices. Activate PA from an `office_manager_terminal`.
Verify audio stream routed to all. Pattern: C3 tests.

### AO4 — Shared Screen on Conference Room Device
Register a `conf_room_display`. Send a `SetUI` with an agenda/shared-screen
scenario descriptor. Verify the terminal receives the rendered UI. Pattern:
server-driven UI tests in D family.

### AO5 — Admin Terminal Sessions
Register an `admin_workstation`. Start a REPL session from it. Verify session
connects, accepts a command, and returns output. Pattern: `p2_test` / `p3_test`.

### AO6 — Idle Lobby Display
Register a `lobby_display` in idle/photo-frame mode. Verify it receives
periodic `SetUI` frames with the configured branding content. Pattern: `d1_test`.

### AO7 — Persistent Intercom
Open an intercom between `team_room_a` and `team_room_b`. Verify the route
survives a separate scenario starting on unrelated devices (not preempted).
Extend C1 with a `persistent` flag if not already present; otherwise verify
existing preemption priority handling preserves the route.

## Milestones

1. **M1** — `ao_test.go` skeleton; all 7 stubs trivially passing.
2. **M2** — AO1, AO3, AO5, AO6 tests fully implemented (direct port of core tests).
3. **M3** — AO2, AO4 tests implemented.
4. **M4** — AO7 implemented, including persistent-route assertion. All 7 IDs
   registered in `scripts/usecase-validate.sh`; CI green.
