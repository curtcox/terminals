---
title: "World Model and Calibration — Progress Log"
kind: progress-log
parent: plans/features/world-model-calibration/plan.md
---

## Implementation Progress (2026-04-26)

Implemented an initial world-model observation query path in the server runtime:

- Added `RecentObservations(zone, kind, since)` to the world-model interface and adapter wiring.
- Wired the media planner observation sink to store observations in the world model.
- Added filtering tests for world-model observation history queries.
- Enriched `DeviceGeometry` with optional sensor calibration fields (`MicArray`, `CameraIntrinsics`, `CameraExtrinsics`, `RadioBias`) for localization-heavy scenarios.
- Added per-device `CalibrationHistory(deviceID, limit)` backed by `VerifyDevice` events so admin surfaces can render verification timelines.
- Added world-model tests covering geometry retention and bounded calibration history queries.
- Added admin calibration APIs:
    - `GET /admin/api/world/calibration` to inspect terminal geometry and verification history.
    - `POST /admin/api/world/verify` to execute verification updates and return fresh state.
- Added admin endpoint tests for calibration listing and verification workflow behavior.

This plan is now shipped and validated for the current server-side scope.
