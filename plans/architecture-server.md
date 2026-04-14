# Server Architecture

See [masterplan.md](../masterplan.md) for overall system context.

## Overview

The server is a Go application running on a Mac mini. It is the brain of the system — all behavior, decision-making, and IO routing originates here.

## Server Module Structure

```
terminal_server/
├── cmd/
│   └── server/
│       └── main.go                  # Entry point
├── internal/
│   ├── discovery/
│   │   └── mdns.go                  # mDNS service advertisement
│   ├── transport/
│   │   ├── grpc_server.go           # gRPC control plane
│   │   └── webrtc_signaling.go      # WebRTC session management
│   ├── device/
│   │   ├── manager.go               # Device registry and lifecycle
│   │   ├── capabilities.go          # Capability querying
│   │   └── state.go                 # Per-device state tracking
│   ├── io/
│   │   ├── router.go                # Routes IO streams between devices
│   │   ├── recorder.go              # Records IO streams to disk
│   │   ├── mixer.go                 # Audio mixing for multi-party
│   │   └── transcoder.go            # Format conversion
│   ├── ai/
│   │   ├── backend.go               # AI backend interface
│   │   ├── speech_to_text.go        # STT adapter
│   │   ├── text_to_speech.go        # TTS adapter
│   │   ├── llm.go                   # LLM query adapter
│   │   ├── vision.go                # Image/video analysis adapter
│   │   └── sound_classify.go        # Audio event detection adapter
│   ├── telephony/
│   │   ├── sip_client.go            # SIP registration and calls
│   │   └── bridge.go                # Bridges WebRTC <-> SIP
│   ├── scenario/
│   │   ├── engine.go                # Scenario lifecycle management
│   │   ├── scenario.go              # Scenario interface
│   │   ├── terminal.go              # Text terminal on laptop/Chromebook
│   │   ├── intercom.go              # Intercom between rooms
│   │   ├── voice_assistant.go       # Smart speaker behavior
│   │   ├── photo_frame.go           # Photo frame rotation
│   │   ├── phone_call.go            # Audio/video calling
│   │   ├── audio_monitor.go         # "Tell me when X stops"
│   │   ├── schedule_monitor.go      # "Watch clock, warn if late"
│   │   ├── timer_reminder.go        # Verbal timer/reminder requests
│   │   ├── alert.go                 # Red alert broadcast
│   │   ├── pa_system.go             # PA broadcast (one mic to all speakers)
│   │   └── multi_window.go          # Multi-camera grid with mixed audio
│   ├── ui/
│   │   └── descriptor.go            # Server-driven UI generation
│   └── storage/
│       ├── db.go                    # SQLite for config and state
│       ├── media.go                 # Media file storage
│       └── schedule.go              # Timers, reminders, schedules
├── api/
│   └── proto/
│       ├── control.proto            # Device ↔ Server control messages
│       ├── capabilities.proto       # Capability declarations
│       ├── io.proto                 # IO stream control
│       └── ui.proto                 # Server-driven UI descriptors
└── configs/
    └── server.yaml                  # Server configuration
```

## Responsibilities

- **Device Manager**: Maintains the registry of connected devices, their capabilities, and per-device state. All scenarios query the Device Manager to discover what IO surfaces are available.
- **IO Router**: Owns the runtime topology of media and data streams. Consumes, produces, forwards, forks, mixes, composites, records, or analyzes any stream. See [io-abstraction.md](io-abstraction.md).
- **Scenario Engine**: Manages scenario lifecycle, priority, preemption, and suspend/resume. See [scenario-engine.md](scenario-engine.md).
- **AI Backend**: Pluggable interfaces for STT, TTS, LLM, vision, and sound classification. See [technology.md](technology.md#ai-backend-pluggable).
- **Telephony Bridge**: SIP client + WebRTC/SIP bridge for external calls.
- **Storage**: SQLite for config/state, filesystem for media, dedicated store for timers and reminders.

## Related Plans

- [protocol.md](protocol.md) — Wire contract with clients.
- [scenario-engine.md](scenario-engine.md) — Scenario contract and lifecycle.
- [io-abstraction.md](io-abstraction.md) — Stream routing primitives.
- [server-driven-ui.md](server-driven-ui.md) — UI descriptor generation.
- [technology.md](technology.md) — Library/framework choices.
