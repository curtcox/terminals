# Phase 6B — Edge Observation and Sensing
See [masterplan.md](../masterplan.md) for overall system context. This phase extends the monitoring work so observation-heavy scenarios can run efficiently without pushing all raw media and sensor traffic to the server.

## Prerequisites
- [phase-6-monitoring.md](phase-6-monitoring.md) complete — baseline monitoring, analyzer events, and placement already exist.
- [phase-3-media.md](phase-3-media.md) complete — WebRTC media plane and IO router exist.
- [phase-5-voice.md](phase-5-voice.md) complete — voice triggers and spoken responses exist.

## Deliverables
- [x] **Application runtime**: add TAR/TAL package loading, permissions, hot reload, and PTY-first development tooling. See [application-runtime.md](application-runtime.md).
- [x] **Edge operator host**: add a generic client runtime for portable compute kernels, retention buffers, clock sync, and artifact export. See [edge-execution.md](edge-execution.md).
- [x] **FlowPlan generalization**: broaden `MediaPlan` into `FlowPlan` so audio, video, sensors, and radio observations all use one planner. See [observation-plane.md](observation-plane.md).
- [x] **Observation plane**: ship typed `Observation`, `ArtifactRef`, `FlowStats`, and artifact-pull messages over the control plane. See [observation-plane.md](observation-plane.md#observation-plane-messages).
- [x] **Compute and buffer claims**: add `compute.*`, `buffer.*`, and `radio.*` resource kinds so edge work is schedulable and preemptable. See [edge-execution.md](edge-execution.md#resource-claims-for-edge-compute).
- [x] **World model upgrade**: add device pose, geometry, verification state, entity location, and calibration workflows. See [world-model-calibration.md](world-model-calibration.md).
- [x] **Recent IMU anomaly scenario**: support retrospective "did you feel that?" queries using shared sensor taps and recent buffers. See [sensing-use-case-flows.md](sensing-use-case-flows.md#recent-imu-anomaly-did-you-feel-that).
- [x] **Sound identification scenario**: support retrospective sound labeling with local-first classification and optional evidence clips. See [sensing-use-case-flows.md](sensing-use-case-flows.md#sound-identification-what-was-that-sound).
- [x] **Sound localization scenario**: support zone-level or spatial sound origin estimates using calibrated device geometry and synchronized timestamps. See [sensing-use-case-flows.md](sensing-use-case-flows.md#sound-localization-where-did-that-sound-come-from).
- [x] **Presence and object tracking**: support fused person presence, object last-seen state, and follow-on actions such as intercom or locate. See [sensing-use-case-flows.md](sensing-use-case-flows.md#presence-query-who-is-in-the-house-and-where).
- [x] **Bluetooth inventory**: surface known and unknown Bluetooth devices by zone and time. See [sensing-use-case-flows.md](sensing-use-case-flows.md#bluetooth-inventory-and-location).
- [x] **Terminal verification tooling**: add admin flows for manual, marker, chirp, and RF-based terminal location verification. See [sensing-use-case-flows.md](sensing-use-case-flows.md#terminal-location-verification).

## Milestone
A capable client can classify and localize a just-heard sound locally, send only compact observations to the server, and the server can answer both "what was that?" and "where did it come from?" while preserving the thin-client architecture.

## Related Plans
- [phase-6-monitoring.md](phase-6-monitoring.md) — Prior monitoring phase this extends.
- [application-runtime.md](application-runtime.md) — Application packaging and language.
- [edge-execution.md](edge-execution.md) — Client-side operator hosting.
- [observation-plane.md](observation-plane.md) — Generalized flow planner and observation transport.
- [world-model-calibration.md](world-model-calibration.md) — Spatial model and verification.
