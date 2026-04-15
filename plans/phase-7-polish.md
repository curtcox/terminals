# Phase 7 — Polish and Expansion

See [masterplan.md](../masterplan.md) for overall system context.

Refinement, additional scenarios, and robustness.

## Prerequisites

- [phase-6-monitoring.md](phase-6-monitoring.md) complete — core scenarios work end to end.

## Deliverables

- [x] **Photo frame scenario**: Idle-screen photo rotation with preemption support. See [use-case-flows.md](use-case-flows.md#smart-photo-frame).
- [x] **Scenario priority and preemption**: Robust suspend/resume of scenarios across devices. See [scenario-engine.md](scenario-engine.md#scenario-priority-and-preemption).
- [x] **Multi-device scenario coordination**: Single scenario spanning multiple devices.
- [x] **Sensor data streaming**: Accelerometer, gyroscope, compass data to server for future scenarios. See [io-abstraction.md](io-abstraction.md).
- [x] **Bluetooth and USB passthrough**: Server-directed BLE scanning and USB device access.
- [x] **Recording and playback**: Server records streams to disk, plays back on demand.
- [ ] **Admin UI**: Web-based dashboard for server configuration, device management, and scenario control.

## Milestone

System handles all described use cases. New scenarios require only server-side code.

## Related Plans

- [scenario-engine.md](scenario-engine.md) — Priority/preemption machinery that gets hardened here.
- [io-abstraction.md](io-abstraction.md) — Additional IO categories activated here.
- [server-driven-ui.md](server-driven-ui.md) — Admin UI composed from the same primitive set.
