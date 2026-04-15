# Phase 7 — Polish and Expansion

See [masterplan.md](../masterplan.md) for overall system context.

Refinement, additional scenarios, and robustness.

## Prerequisites

- [phase-6-monitoring.md](phase-6-monitoring.md) complete — core scenarios work end to end.

## Deliverables

- [ ] **Photo frame scenario**: Idle-screen photo rotation with low-priority `screen.main` claim; preempted and resumed precisely by the claim manager. See [use-case-flows.md](use-case-flows.md#smart-photo-frame).
- [ ] **Preemption and resume hardening**: Activation snapshots persisted in storage; crash recovery replays the active set; soak tests exercise nested suspend/resume (PA on top of photo frame on top of voice overlay). See [scenario-engine.md](scenario-engine.md#resource-claims-and-preemption).
- [ ] **Scenario recipe builder**: Extract the common "resolve targets → claim → media plan → UI → cleanup" skeleton into the `ScenarioRecipe` helper and port suitable built-in scenarios onto it. See [scenario-engine.md](scenario-engine.md#scenario-recipes).
- [ ] **Multi-device scenario coordination**: Single activation spanning multiple devices with one claim set and one media plan.
- [ ] **Automation/webhook triggers**: Webhook and automation-agent producers on the intent/event bus so external systems can drive activations with the same `Intent`/`Event` shape. See [scenario-engine.md](scenario-engine.md#triggers-intents-and-events).
- [ ] **Admin UI for world model**: Manage zones, roles, and device metadata from the admin dashboard so adding a room is configuration. See [placement.md](placement.md).
- [ ] **Sensor data streaming**: Accelerometer, gyroscope, compass data to server as shared `sensor.*` claims. See [io-abstraction.md](io-abstraction.md).
- [ ] **Bluetooth and USB passthrough**: Server-directed BLE scanning and USB device access.
- [ ] **Recording and playback**: Recorder node in the media planner; playback as a source node. Records stream to disk, plays back on demand.
- [ ] **Admin UI**: Web-based dashboard for server configuration, device management, scenario control, and activation inspection (live claims, suspended activations, intent/event tail).

## Milestone

System handles all described use cases. New scenarios require only server-side code.

## Related Plans

- [scenario-engine.md](scenario-engine.md) — Priority/preemption machinery that gets hardened here.
- [io-abstraction.md](io-abstraction.md) — Additional IO categories activated here.
- [server-driven-ui.md](server-driven-ui.md) — Admin UI composed from the same primitive set.
