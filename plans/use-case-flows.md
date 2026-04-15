# Use Case Flows

See [masterplan.md](../masterplan.md) for overall system context. See [../usecases.md](../usecases.md) for user-story-style use cases.

Each use case maps to a scenario definition. A single trigger produces an `Intent` or `Event` on the bus; the engine constructs a `ScenarioActivation`; the activation resolves targets via the [placement engine](placement.md), requests [resource claims](io-abstraction.md#resource-claims), applies a [`MediaPlan`](io-abstraction.md#media-topology-plans-not-connects), and sends UI. The client code is identical in every case — only the server scenario differs.

## Text Terminal

**Trigger**: User selects "Terminal" from device menu, or server auto-assigns to laptops/Chromebooks.

**Flow**:

1. Server sends `SetUI` with a terminal layout (monospace text area + input field).
2. Client forwards keyboard input events via gRPC.
3. Server runs a PTY (pseudo-terminal) session, sends output back as UI text updates.
4. Server can multiplex — multiple terminals on one device or one terminal session accessed from multiple devices.

## Audio and Video Calls

**Trigger**: Voice command ("call Mom") or UI action.

**Flow**:

1. Server looks up contact, determines call type (internal client-to-client or external via SIP).
2. For internal: establishes WebRTC streams between two clients via the server SFU.
3. For external: SIP client initiates outbound call, server bridges WebRTC ↔ SIP.
4. Server sends call UI to both devices (video surface, mute/hangup buttons).
5. On hangup, server tears down streams and restores previous UI/scenario.

## Intercom

**Trigger**: Voice command ("intercom to kitchen") or button press.

**Flow**:

1. Server routes mic audio from source device to speaker on target device (and vice versa).
2. Server sends intercom UI to both devices (visual indicator + active speaker display).
3. Can be one-way (announcement) or two-way (conversation).
4. Can target all devices simultaneously (whole-house announcement).

## Smart Speaker / Voice Assistant

**Trigger**: Wake word detected (server continuously analyzes mic audio from idle devices).

**Flow**:

1. Client streams mic audio to server.
2. Server runs STT on the audio.
3. Transcribed text goes to LLM for response generation.
4. Response text goes through TTS.
5. Audio response plays on the device's speaker.
6. If the response includes visual content (weather, recipe, etc.), server sends `SetUI` with the content alongside audio.

## Smart Photo Frame

**Trigger**: Device is idle, or user selects "Photo Frame" mode.

**Flow**:

1. Server sends `SetUI` with a fullscreen image + `keep_awake` flag.
2. Server rotates images on a timer (configurable interval).
3. Photos sourced from a configured directory on the server.
4. Preemptable — any higher-priority scenario takes over the screen, photo frame resumes after.

## Timers and Reminders

**Trigger**: Voice command ("set a timer for 10 minutes", "remind me to check the oven at 3 PM").

**Flow**:

1. STT + LLM extract the timer/reminder intent and parameters.
2. Server stores the timer/reminder in the scheduler.
3. When triggered, server plays a TTS announcement on the originating device (or all devices).
4. Server sends a notification UI overlay until dismissed.

## Audio Monitoring ("Tell Me When the Dishwasher Stops")

**Trigger**: Voice command.

**Flow**:

1. Server begins streaming mic audio from the nearest device to the sound classification AI.
2. AI backend monitors for the transition from "machine running" to "silence" (or a specific end-of-cycle beep pattern).
3. When detected, server sends notification to the requesting device (or all devices) via TTS + UI notification.
4. Monitoring stream stops to conserve resources.

## Schedule Monitoring ("Watch My Child")

**Trigger**: Configured schedule (e.g., school days at 7:00 AM) or voice command.

**Flow**:

1. At trigger time, server activates camera on the relevant device.
2. Video stream analyzed by vision AI for activity/presence detection.
3. Server cross-references with stored schedule data (school starts at 8:15, bus comes at 7:50, etc.).
4. If child appears to be behind schedule, server sends TTS warning through the device speaker: "It's 7:40 and you haven't left yet — the bus comes in 10 minutes."
5. Can escalate to parent notification on another device.

## Red Alert

**Trigger**: Voice command ("red alert") or UI action.

**Flow**:

1. Server preempts ALL active scenarios on ALL devices (critical priority).
2. Server sends `SetUI` to every device: red background, flashing alert text, alarm icon.
3. Server plays alarm audio on every device with speakers.
4. Server optionally activates all cameras and mics for full situational awareness.
5. Dismissed by voice command ("stand down") or UI action on any device.
6. All previously active scenarios resume.

## PA System

**Trigger**: Voice command ("PA mode") or UI action on the broadcasting device.

**Flow**:

1. Server designates the triggering device as the PA source.
2. Server sends `StartStream` (mic) to the source device.
3. Server sends `StopStream` (mic) to all other devices to avoid feedback loops, and mutes any active scenario audio on their speakers.
4. Server forks the incoming mic audio stream from the source and forwards it to the speakers of all other connected devices simultaneously.
5. The source device gets a PA UI: a live audio visualizer, a mute/unmute toggle, and an "End PA" button.
6. Receiving devices get a notification overlay ("PA from {device_name}") on top of their current UI — their active scenario is **not** preempted, only their audio output is temporarily taken over.
7. On end, server restores normal audio routing and removes the overlay from receiving devices.

**Key claims**: `speaker.main` (exclusive) on every receiving device and `screen.overlay` (shared) for the PA notification. The receiving devices' `screen.main` claims are untouched — their active scenarios keep running with their audio parked. **Key media plan**: mic(source) → fork → speaker(A..N).

```
Source (mic) ──WebRTC──→ Server ──fork──→ Speaker A
                                 ├──────→ Speaker B
                                 ├──────→ Speaker C
                                 └──────→ Speaker N
```

## Multi-Window (Security Camera / Multi-Feed View)

**Trigger**: Voice command ("show all cameras") or UI action.

**Flow**:

1. Server queries all connected devices with camera capabilities.
2. Server sends `StartStream` (camera) to each of those devices.
3. Server establishes WebRTC video tracks from each camera device to the viewing device.
4. Server sends `SetUI` to the viewing device with a `grid` layout containing one `video_surface` per camera feed, each bound to its corresponding WebRTC video track. Each cell is labeled with the source device name.
5. Server mixes audio from all streaming devices into a single combined audio track and routes it to the viewing device's speakers. Alternatively, the viewer can tap a cell to hear only that device's audio (server switches from mixed to single-source routing).
6. The grid layout adapts based on the number of cameras: 1 = fullscreen, 2 = side-by-side, 3–4 = 2×2 grid, 5–6 = 2×3 grid, etc.
7. On end, server sends `StopStream` (camera) to all source devices and restores the viewing device's previous UI/scenario.

**Key IO Router operations**: Multiple simultaneous forwards (N cameras → N video surfaces), Mix (N mic streams → 1 combined audio track), with optional single-source selection.

```
Camera A ──WebRTC──→ Server ──→ video_surface[0] ──→ ┌─────────────────┐
Camera B ──WebRTC──→ Server ──→ video_surface[1] ──→ │ A │ B │          │
Camera C ──WebRTC──→ Server ──→ video_surface[2] ──→ │───┼───│ Viewer   │
Camera D ──WebRTC──→ Server ──→ video_surface[3] ──→ │ C │ D │          │
                                                      └─────────────────┘
Mic A ─┐
Mic B ─┼──→ Server (audio mixer) ──→ Viewer speakers
Mic C ─┤
Mic D ─┘
```

## Related Plans

- [scenario-engine.md](scenario-engine.md) — Scenario definitions, activations, and lifecycle.
- [io-abstraction.md](io-abstraction.md) — Router primitives used by each flow.
- [server-driven-ui.md](server-driven-ui.md) — UI primitives used by each flow.
- [../usecases.md](../usecases.md) — User-story form of each scenario.
