---
title: "Adjacent Business Use Case Validation (AB1–AB7)"
kind: plan
status: planned
owner: curtcox
validation: none
last-reviewed: 2026-05-17
---

# Adjacent Business Use Case Validation

All seven AB use cases map directly to validated core scenarios — the features
already work. The work here is writing focused harness tests that demonstrate
each use case in a business context and registering them in `make usecase-validate`.

## Use Cases in Scope

| ID | Description | Maps to | Work type |
|----|-------------|---------|-----------|
| AB1 | Kitchen/front-of-house intercom | C1 (intercom) | Validation |
| AB2 | Broadcast PA to sales floor | C3 (PA mode) | Validation |
| AB3 | Display guest welcome messages on idle screens | D family (display) | Validation |
| AB4 | Multi-window camera grid at loading docks | S1–S3 (security/camera) | Validation |
| AB5 | Audio monitoring for alarm/glass-break after hours | M1–M2 (audio monitor) | Validation |
| AB6 | Voice timer "timer 12 minutes table 5" | T1–T2 (timers) | Validation |
| AB7 | Voice announcement "patient Smith, room 3 is ready" | C2 (announcement) | Validation |

## Approach

Each test is a thin scenario-context wrapper around the existing validated
behavior. The scenario and harness infrastructure is reused; only the device
roles, room names, and assertion strings differ from the core tests.

Write one `TestUseCaseAB<N>WithEvidence` per ID in
`terminal_server/internal/usecasevalidation/ab_test.go`.

### AB1 — Kitchen/Front-of-House Intercom
Two devices: `kitchen_terminal` and `front_of_house`. Trigger intercom
intent on front-of-house, verify two-way audio route opened to kitchen.
Harness pattern: same as `c1_test.go`.

### AB2 — PA Mode Broadcast to Sales Floor
Register three `sales_floor_*` devices. Activate PA mode on a
`manager_terminal`. Verify all sales floor devices receive the PA audio
stream. Harness pattern: same as C3 transport tests.

### AB3 — Guest Welcome Display
Register a `lobby_display` device. Send a `SetUI` with a welcome message
scenario. Verify the terminal receives the UI descriptor with the guest name.
Harness pattern: same as D-family display tests.

### AB4 — Multi-Window Camera Grid
Register four `dock_camera_*` devices plus a `supervisor_terminal`. Trigger
multi-window view intent. Verify the supervisor terminal receives a grid-layout
UI with camera feeds. Harness pattern: same as `s1_test` / `s2_test`.

### AB5 — After-Hours Alarm Monitoring
Register an `audio_monitor` terminal with sound classifier armed. Inject a
`glass_break` sound event via `FakeSoundClassifier`. Verify alert broadcast
to configured `security_terminal`. Harness pattern: same as `m2_test`.

### AB6 — Voice Timer for Food Service
Voice command "timer 12 minutes table 5" parsed to `CreateTimer` intent with
label "table 5" and duration 12m. Advance fake clock. Verify "Timer done!"
broadcast with label. Harness pattern: same as `t1_test`.

### AB7 — Voice Announcement to Waiting Area
Voice command "announce: patient Smith, room 3 is ready" triggers
`AnnouncementAudioScenario`. Verify `waiting_area_terminal` receives
announcement audio. Harness pattern: same as `c2_test`.

## Milestones

1. **M1** — `ab_test.go` skeleton with all 7 test stubs; all pass trivially.
2. **M2** — AB7, AB6, AB5 tests fully implemented and passing (simplest first).
3. **M3** — AB1, AB2, AB3, AB4 tests fully implemented and passing.
4. **M4** — All 7 IDs registered in `scripts/usecase-validate.sh` and CI green.
