# Use Case Flows

This document captures the baseline scenario flows from [usecases.md](../usecases.md).

A single trigger produces an Intent or Event on the server bus. The server
constructs a ScenarioActivation, resolves targets, requests resource claims,
applies a media plan, and sends server-driven UI. Client behavior remains
generic across all scenarios; the scenario logic is server-side.

## Text Terminal

Trigger: user selects Terminal from a device menu, or server auto-assigns to
laptops/Chromebooks.

Flow:

1. Server sends SetUI with a terminal layout (monospace text area + input).
2. Client forwards keyboard input events via gRPC.
3. Server runs a PTY session and sends output as UI text updates.
4. Server can multiplex: multiple terminals on one device or one terminal
   session across multiple devices.

## Audio and Video Calls

Trigger: voice command ("call Mom") or UI action.

Flow:

1. Server looks up the contact and determines call type (internal or SIP).
2. Internal call: server establishes WebRTC streams via the SFU.
3. External call: SIP client initiates outbound call; server bridges WebRTC and
   SIP.
4. Server sends call UI to both devices (video surface, mute/hangup controls).
5. On hangup, server tears down streams and restores prior UI/scenario.

## Intercom

Trigger: voice command ("intercom to kitchen") or button press.

Flow:

1. Server routes mic audio from source to target speaker (and optionally back).
2. Server sends intercom UI to both devices (indicator + active speaker).
3. Supports one-way announcements and two-way conversation.
4. Supports all-device announcements.

## Smart Speaker and Voice Assistant

Trigger: wake word detection from idle-device mic audio.

Flow:

1. Client streams mic audio to server.
2. Server runs STT.
3. Transcribed text goes to LLM response generation.
4. Response text goes through TTS.
5. Audio response plays on device speaker.
6. If response has visual content, server sends SetUI alongside audio.

## Smart Photo Frame

Trigger: device idle state or user selecting Photo Frame mode.

Flow:

1. Server sends SetUI with a fullscreen image and keep-awake semantics.
2. Server rotates images on a configurable timer.
3. Photos come from a configured server-side directory.
4. Higher-priority scenarios preempt; photo frame resumes afterward.

## Timers and Reminders

Trigger: voice command ("set a timer", "remind me at 3 PM").

Flow:

1. STT + LLM extract intent and parameters.
2. Server stores timer/reminder in scheduler.
3. On trigger, server announces via TTS on the originating or all devices.
4. Server displays a notification overlay until dismissed.

## Audio Monitoring (Dishwasher and Dryer)

Trigger: voice command.

Flow:

1. Server streams mic audio from nearest capable device to classifier.
2. AI backend detects end-of-cycle condition (silence or specific beep).
3. Server sends TTS + UI notification to requesting device or all devices.
4. Monitoring stream stops to conserve resources.

## Schedule Monitoring (Watch My Child)

Trigger: configured schedule (for example school days) or voice command.

Flow:

1. At trigger time, server activates relevant camera.
2. Vision analysis checks activity/presence.
3. Server cross-references stored schedule data.
4. If behind schedule, server plays spoken warning.
5. Server can escalate to parent notification on another device.

## Red Alert

Trigger: voice command ("red alert") or UI action.

Flow:

1. Server preempts all active scenarios on all devices.
2. Server sends SetUI to every device with red alert visuals.
3. Server plays alarm audio on every speaker-capable device.
4. Server can optionally activate all cameras/mics.
5. Dismiss via voice command ("stand down") or any-device UI action.
6. Prior scenarios resume after dismissal.

## PA System

Trigger: voice command ("PA mode") or UI action on broadcasting device.

Flow:

1. Server marks triggering device as PA source.
2. Server starts mic stream on source.
3. Server stops mic streams on receivers and parks their scenario audio.
4. Server forks incoming source mic to speakers on all other devices.
5. Source device gets PA controls and visual feedback.
6. Receiving devices get overlay notification without main-screen preemption.
7. On end, server restores normal routing and clears overlays.

Key claims: speaker.main (exclusive) on receivers and screen.overlay (shared)
for notification. Receiving screen.main claims remain untouched.

## Multi-Window (Security Camera and Multi-Feed View)

Trigger: voice command ("show all cameras") or UI action.

Flow:

1. Server queries connected devices with camera capability.
2. Server starts camera streams on each source.
3. Server establishes WebRTC video tracks from sources to viewing device.
4. Server sends SetUI to viewer with grid layout and per-source video surfaces.
5. Server mixes audio from sources into one track, with optional single-source
   focus on selection.
6. Grid layout adapts to source count.
7. On end, server stops source streams and restores previous viewer state.

## Related References

- [Use Cases](../usecases.md)
- [Use Case Validation Matrix](usecase-validation-matrix.md)
- [Sensing and Edge Observation Use Case Flows](sensing-use-case-flows.md)
- [Scenario Engine Plan](../plans/features/scenario-engine.md)
- [Application Runtime](application-runtime.md)
- [IO Abstraction Plan](../plans/features/io-abstraction.md)
