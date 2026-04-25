# Sensing and Edge Observation Use Case Flows
See [masterplan.md](../masterplan.md) for overall system context. See [../usecases.md](../usecases.md) for user-story-style use cases. These flows extend [use-case-flows.md](use-case-flows.md) with sensing-heavy scenarios that should prefer edge execution when the client supports it.

A single trigger still produces a typed request on the server. The server resolves targets, requests claims, applies a `FlowPlan`, and renders UI. The difference is that expensive operators may be placed on the client and only emit compact observations back to the server.

## Recent IMU Anomaly ("Did You Feel That?")
**Trigger**: Voice command ("Did you feel that?"), UI action, or an automatic anomaly notification.

**Flow**:
1. Nearby capable devices continuously maintain a short rolling accelerometer buffer and low-cost anomaly score using shared `sensor.*` and `buffer.sensor.recent` claims.
2. When the user asks, the server queries recent observations from devices in the relevant zone or from the nearest mobile/fixed terminals.
3. The server requests one or more timeseries excerpts only for windows whose anomaly score crosses threshold.
4. The server fuses the excerpts into a timeline summary: magnitude, direction if available, involved devices, and confidence.
5. The server sends a UI card and optional spoken answer: "Yes — unusual motion was recorded in the living room 6 seconds ago."

**Key flow plan**: `sensor(accelerometer) -> buffer.recent -> analyzer(imu_anomaly) -> sink.store + sink.event_bus`.

## Sound Identification ("What Was That Sound?")
**Trigger**: Voice command immediately after a sound, or a tap on a notification.

**Flow**:
1. Nearby devices keep a short rolling audio buffer and lightweight onset detector.
2. On request, the server asks candidate devices for the most recent onset observations and their top classifier labels.
3. Capable devices classify the buffered audio locally; incapable devices may upload only the short candidate clip for server-side classification.
4. The server compares confidence, timing, and zone fit, then picks the best explanation.
5. The server returns a spoken and visual answer, optionally with a "play clip" action.

**Key flow plan**: `mic -> buffer.recent -> analyzer(sound_classifier) -> artifact(audio_clip) -> sink.event_bus`.

## Sound Localization ("Where Did That Sound Come From?")
**Trigger**: Voice command, or follow-up action after sound identification.

**Flow**:
1. Edge hosts emit onset timestamps or local direction-of-arrival estimates from the recent audio window.
2. The server consults device geometry, clock sync quality, and verification state from the world model.
3. If one device has a calibrated microphone array, its local angle estimate is used directly; otherwise the server fuses timestamps from multiple devices.
4. The server computes a zone or spatial estimate with confidence.
5. The server renders a floor-plan or zone answer and may offer to open an intercom to that room.

**Key flow plan**: `mic[*] -> feature(onset/timestamp) -> localizer -> fusion -> sink.event_bus`.

## Presence Query ("Who Is in the House and Where?")
**Trigger**: Voice command, dashboard open, or automation query.

**Flow**:
1. Clients with cameras, BLE radios, or strong interaction signals produce local observations such as person detections, tracked occupants, device interactions, and radio sightings.
2. The server fuses observations into one presence graph keyed by person.
3. The placement/world model maps each person to a best-known zone and optional pose.
4. The server answers in room language first and can render a richer admin view if requested.
5. If the user chooses a person, the server can immediately launch a follow-on scenario such as intercom or call.

**Key claims**: shared `camera.analyze`, `radio.ble.scan`, and `buffer.video.recent` where supported.

## Change Detection Notification
**Trigger**: User subscription ("Tell me when something changes in the garage"), schedule, or admin rule.

**Flow**:
1. The server places lightweight diff or motion operators on the selected devices.
2. Edge hosts compare the current input against a short baseline or prior frame and emit only change events.
3. The server deduplicates repeated events across nearby devices.
4. The server sends a notification, optionally attaching a snapshot or short clip.
5. The monitoring activation persists until canceled or until its schedule ends.

**Key flow plan**: `camera -> analyzer(change_detect)` or `sensor -> analyzer(change_detect) -> sink.event_bus`.

## Unusual Detection Notification
**Trigger**: User subscription ("Tell me if anything unusual happens"), schedule, or admin rule.

**Flow**:
1. The server provisions anomaly detectors for the selected modalities: audio, video, IMU, or radio.
2. Edge hosts maintain modality-specific baselines and emit anomaly scores instead of raw streams.
3. The server applies house-wide context: time of day, known occupants, schedules, and recent alerts.
4. High-confidence anomalies become notifications with evidence and confidence.
5. Users can drill into evidence or dismiss and retrain the baseline later.

**Key difference from change detection**: anomaly detectors compare against learned normal behavior, not only immediate deltas.

## Object Location Tracking
**Trigger**: User asks for an object ("Where are my keys?"), opens a dashboard, or the system updates last-seen state automatically.

**Flow**:
1. Cameras with adequate edge support run object detection and tracking locally, emitting compact object observations and optional embeddings.
2. The server associates detections with known objects or tags using last-seen history, optional BLE tags, and zone constraints.
3. Snapshot artifacts are pulled only for interesting sightings.
4. The world model updates each object's `LastKnown` location and last-seen time.
5. The server answers with zone language first, then with evidence if requested.

**Key flow plan**: `camera -> tracker(object_tracker) -> artifact(snapshot) -> sink.store + sink.event_bus`.

## Bluetooth Inventory and Location
**Trigger**: Admin dashboard, voice query, or scheduled network inventory.

**Flow**:
1. Clients with Bluetooth capability run scans locally under shared `radio.ble.scan` claims.
2. The client reports compact sighting records: device identifier, RSSI summary, timestamps, and local confidence.
3. The server fuses sightings by zone and across time to build a house-wide Bluetooth inventory.
4. Known devices are labeled; unknown devices are surfaced for review.
5. The admin UI shows what devices are present, where they were last strongest, and which terminals reported them.

**Key flow plan**: `bluetooth.scan -> feature(rssi_summary) -> fusion -> sink.store`.

## Terminal Location Verification
**Trigger**: Admin action from a device-management screen.

**Flow**:
1. The admin selects a terminal and chooses a verification method: manual confirm, visual marker, audio chirp, or RF fingerprint.
2. The server runs the appropriate verification flow using nearby clients as needed.
3. The resulting pose estimate is compared to the stored terminal pose.
4. The server updates verification state, confidence, and calibration history.
5. If the delta is large, the admin UI offers to accept the new pose or keep the old one.

**Key world-model requirement**: fixed devices preserve pose, verification method, and calibration version.

## Related Plans
- [observation-plane.md](observation-plane.md) — `FlowPlan`, observations, artifacts, and buffers.
- [edge-execution.md](edge-execution.md) — Edge operator placement and runtime constraints.
- [world-model-calibration.md](world-model-calibration.md) — Spatial model and terminal verification.
- [application-runtime.md](application-runtime.md) — TAL applications that request these flows.
- [phase-6b-edge-sensing.md](phase-6b-edge-sensing.md) — Suggested implementation phase.
