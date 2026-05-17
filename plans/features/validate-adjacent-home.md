---
title: "Adjacent Home Use Case Validation (AH1–AH17)"
kind: plan
status: planned
owner: curtcox
validation: none
last-reviewed: 2026-05-17
---

# Adjacent Home Use Case Validation

AH1–AH17 cover home-automation extensions. Some map directly to validated core
scenarios; others require new server features before a test can pass.

## Use Cases in Scope

| ID | Description | Maps to | Work type |
|----|-------------|---------|-----------|
| AH1 | Doorbell arrival announcement | C2 (announcement) + M (sensor trigger) | Validation |
| AH2 | Voice "call for help" → caregiver alert | C1/V1 (voice intent + intercom) | Validation |
| AH3 | Stream music to one or more speakers | New IO stream scenario | Feature + validation |
| AH4 | Voice → smart home device control (lights, thermostat) | New smart-home intent + stub integration | Feature + validation |
| AH5 | TTS bedtime story on child's device | TTS + D/announcement | Validation |
| AH6 | Morning weather + schedule summary via TTS | V1 (voice assistant) + TTS | Validation |
| AH7 | "Check on the dog" → camera feed | S family (camera/focus) | Validation |
| AH8 | Smoke/CO alarm sound detected → alert all devices | M2 (audio monitor) | Validation |
| AH9 | Query recent accelerometer events | Sensor history query | Validation (sensor log exists) |
| AH10 | Identify a sound I just heard | Sound classification | Validation (FakeSoundClassifier) |
| AH11 | Locate where a sound came from | Multi-device sound localization | Feature + validation |
| AH12 | Who is currently in the house and where | Presence detection | Feature + validation |
| AH13 | Notify on relevant sensor change | M family (monitoring) | Validation |
| AH14 | Notify on anomaly / unusual behavior | M family + anomaly classifier | Validation (FakeClassifier) |
| AH15 | Locate important objects (Bluetooth) | Bluetooth scanner + world model | Feature + validation |
| AH16 | View active Bluetooth devices and locations | Bluetooth scanner + registry | Feature + validation |
| AH17 | Verify physical device placement | Device registry + placement audit | Validation (placement plan exists) |

## Phase 1 — Validation-only (no new features)

These reuse existing harness infrastructure. Write in `ah_test.go`.

**AH1** — Register a `doorbell_sensor` device. Inject a sensor activation via
the sensor helper (motion/sound). Verify announcement broadcast to all house
terminals. Pattern: AA1 (webhook announce) + M-family sensor trigger.

**AH2** — Voice command "help" or "emergency" on an elderly user's device
triggers an alert to the caregiver's device. Pattern: V1 voice intent +
C1 intercom/alert route.

**AH5** — TTS bedtime story: `SetUI` scenario delivers a `PlayAudio` event
with TTS text "Once upon a time..." to `child_room_terminal`. Verify FakeTTS
receives the text and PlayAudio appears in the terminal's stream.

**AH6** — Voice query "what's the weather today" on a kitchen device triggers
`VoiceAssistantScenario`. FakeLLM returns a weather summary. Verify TTS
response broadcast back. Pattern: V2 recipe test.

**AH7** — Voice "check on the dog". Intent parsed as camera-focus on
`living_room_camera`. Verify `focus_terminal` receives the camera stream.
Pattern: `s2_test`.

**AH8** — `FakeSoundClassifier` emits `smoke_alarm` event. Verify broadcast
to all devices including sleeping/idle terminals. Pattern: M2 dryer-beep test
extended to whole-house alert.

**AH9** — Register a terminal with an accelerometer sensor. Inject an
accelerometer reading via sensor helper. Query recent events via voice command.
Verify response includes the injected reading.

**AH10** — Voice "what was that sound?". FakeSoundClassifier returns a label.
Verify the label is broadcast back as a text announcement. Pattern: M2 +
VoiceAssistant.

**AH13** — Arm a monitoring scenario. Inject a sensor state change. Verify
notification broadcast. Pattern: M family + AA2.

**AH14** — Arm anomaly detection. FakeClassifier emits `anomaly` label.
Verify alert broadcast. Pattern: M2 extended with anomaly classifier.

**AH17** — Run `make usecase-wiring-audit` and verify device placement data
is present for all registered terminals. Or write a harness test that queries
the placement engine and asserts positions are non-null.

## Phase 2 — Feature + Validation

### AH3 — Music Streaming
1. Define a `MusicStreamScenario` that opens a continuous audio output stream
   to one or more `speaker` devices.
2. Add `FakeMusicSource` that yields deterministic PCM chunks.
3. Write `TestUseCaseAH3WithEvidence`: start scenario, verify speakers receive
   audio frames, pause/resume.

### AH4 — Smart Home Control
1. Define a `SmartHomeClient` interface and `FakeSmartHomeClient` stub.
2. Add a `SmartHomeScenario` that handles intents like `set_lights_dim` and
   `set_thermostat_to`.
3. Write `TestUseCaseAH4WithEvidence`: voice "dim the lights" →
   `FakeSmartHomeClient.SetLight()` called; voice "set thermostat to 70" →
   `FakeSmartHomeClient.SetThermostat()` called.

### AH11 — Sound Localization
1. Add `LocationHint` to `SoundEvent` (which device heard it loudest).
2. Implement trivial localization: pick the device with the highest amplitude.
3. Write `TestUseCaseAH11WithEvidence`: inject sound on two devices with
   different amplitudes; verify response names the louder device's room.

### AH12 — Presence Detection
1. Add a `PresenceScenario` that tracks wake-word activations and camera
   motion events as presence signals per room.
2. Write `TestUseCaseAH12WithEvidence`: inject activity on `living_room` and
   `kitchen` devices; query "who is home"; verify both rooms reported.

### AH15 — Object Tracking (Bluetooth)
1. Add a `BluetoothScanner` interface and `FakeBluetoothScanner` that returns
   deterministic beacon observations.
2. Extend the world model with object-location entries.
3. Write `TestUseCaseAH15WithEvidence`: scanner reports `keys_tag` near
   `hallway_terminal`; query "where are my keys"; verify hallway room returned.

### AH16 — Bluetooth Device Inventory
1. Reuse `FakeBluetoothScanner` from AH15.
2. Write `TestUseCaseAH16WithEvidence`: scanner reports N devices; query
   "what Bluetooth devices are active"; verify list matches scanner output.

## Milestones

1. **M1** — Phase 1 tests (AH1, AH2, AH5–AH10, AH13, AH14, AH17) written and
   passing. All registered in validate script.
2. **M2** — AH3 (music) and AH4 (smart home) interfaces + fakes + tests.
3. **M3** — AH11 (localization) and AH12 (presence) implemented + tested.
4. **M4** — AH15, AH16 (Bluetooth) implemented + tested. All 17 IDs green in CI.
