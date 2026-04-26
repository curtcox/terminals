# Sensing and Edge Observation Use Case Flows

This document captures the sensing-heavy flows that extend the core use-case
flows with edge-first observation behavior.

Reference context:

- Master context: [../masterplan.md](../masterplan.md)
- Use-case catalog: [../usecases.md](../usecases.md)
- Base flow set: [../plans/features/use-case-flows.md](../plans/features/use-case-flows.md)

A single trigger still produces a typed server request. The server resolves
targets, requests claims, applies a flow plan, and renders UI. For sensing
flows, expensive operators may execute at the edge and emit compact
observations back to the server.

## Recent IMU Anomaly ("Did You Feel That?")

Trigger: voice command, UI action, or automatic anomaly notification.

Flow:

1. Nearby capable devices keep a short rolling accelerometer buffer and a
   low-cost anomaly score using shared sensor claims.
2. On request, the server queries recent observations from devices in the
   relevant zone (or nearest mobile/fixed terminals).
3. The server requests timeseries excerpts only for windows crossing threshold.
4. The server fuses excerpts into a timeline summary (magnitude, direction,
   involved devices, confidence).
5. The server sends a UI card and optional spoken answer.

Representative flow plan: sensor(accelerometer) -> buffer.recent ->
analyzer(imu_anomaly) -> sink.store + sink.event_bus.

## Sound Identification ("What Was That Sound?")

Trigger: voice command immediately after a sound, or a notification tap.

Flow:

1. Nearby devices keep a short rolling audio buffer and lightweight onset
   detector.
2. The server requests onset observations and top classifier labels.
3. Capable devices classify locally; others can upload only the short
   candidate clip for server-side classification.
4. The server compares confidence, timing, and zone fit and selects the best
   explanation.
5. The server returns spoken/visual answer with optional clip playback.

Representative flow plan: mic -> buffer.recent -> analyzer(sound_classifier)
-> artifact(audio_clip) -> sink.event_bus.

## Sound Localization ("Where Did That Sound Come From?")

Trigger: voice command, or follow-up after sound identification.

Flow:

1. Edge hosts emit onset timestamps or local direction-of-arrival estimates.
2. The server consults geometry, clock sync quality, and world-model
   verification.
3. If a calibrated array is available, use its local angle; otherwise fuse
   timestamps across multiple devices.
4. The server computes a zone/spatial estimate with confidence.
5. The server renders a floor-plan or zone answer and can offer an intercom
   follow-on action.

Representative flow plan: mic[*] -> feature(onset/timestamp) -> localizer ->
fusion -> sink.event_bus.

## Presence Query ("Who Is in the House and Where?")

Trigger: voice command, dashboard open, or automation query.

Flow:

1. Camera, BLE, and interaction-capable clients produce local presence
   observations.
2. The server fuses observations into a person-keyed presence graph.
3. The world model maps each person to best-known zone (and optional pose).
4. The server answers in room language first and can render richer admin
   detail.
5. Selecting a person can launch follow-on scenarios (for example, intercom
   or call).

## Change Detection Notification

Trigger: user subscription, schedule, or admin rule.

Flow:

1. The server places lightweight diff/motion operators on selected devices.
2. Edge hosts compare current input to short baselines and emit compact change
   events.
3. The server deduplicates repeated events across nearby devices.
4. The server sends notifications with optional snapshot/clip evidence.
5. Monitoring persists until canceled or schedule end.

Representative flow plan: camera|sensor -> analyzer(change_detect) ->
sink.event_bus.

## Unusual Detection Notification

Trigger: user subscription, schedule, or admin rule.

Flow:

1. The server provisions modality-specific anomaly detectors.
2. Edge hosts maintain baselines and emit anomaly scores instead of raw
   streams.
3. The server applies house context (time, occupancy, schedule, recent alerts).
4. High-confidence anomalies become notifications with evidence and
   confidence.
5. Users can drill into evidence or dismiss and retrain later.

Difference from change detection: anomaly compares against learned normal
behavior, not only immediate deltas.

## Object Location Tracking

Trigger: object query, dashboard open, or automatic last-seen update.

Flow:

1. Capable cameras run local object detection/tracking and emit compact
   observations (optionally embeddings).
2. The server associates detections with known objects using history, BLE tags,
   and zone constraints.
3. Snapshot artifacts are requested only for relevant sightings.
4. The world model updates last-known object location and timestamp.
5. The server answers in zone-first language, with evidence on demand.

Representative flow plan: camera -> tracker(object_tracker) ->
artifact(snapshot) -> sink.store + sink.event_bus.

## Bluetooth Inventory and Location

Trigger: admin dashboard, voice query, or scheduled inventory.

Flow:

1. Bluetooth-capable clients scan under shared radio claims.
2. Clients report compact sightings (identifier, RSSI summary, timestamps,
   confidence).
3. The server fuses sightings by zone and over time into house-wide inventory.
4. Known devices are labeled; unknown devices are surfaced for review.
5. Admin UI presents active devices, strongest zones, and reporting terminals.

Representative flow plan: bluetooth.scan -> feature(rssi_summary) -> fusion ->
sink.store.

## Terminal Location Verification

Trigger: admin action from device management.

Flow:

1. Admin selects a terminal and verification method (manual, visual marker,
   audio chirp, RF fingerprint).
2. The server runs the selected verification flow (using nearby clients when
   needed).
3. Resulting pose estimate is compared with stored terminal pose.
4. The server updates verification state, confidence, and calibration history.
5. If the delta is large, admin can accept new pose or keep prior pose.

World-model requirement: fixed devices preserve pose, verification method, and
calibration version.

## Related Design References

- [observation-plane.md](observation-plane.md)
- [edge-execution-runtime.md](edge-execution-runtime.md)
- [../plans/features/world-model-calibration.md](../plans/features/world-model-calibration.md)
- [application-runtime.md](application-runtime.md)
