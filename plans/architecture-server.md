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
│   │   ├── metadata.go              # Zone, roles, mobility, affinity
│   │   └── state.go                 # Per-device state tracking
│   ├── placement/
│   │   ├── engine.go                # PlacementEngine (semantic target resolution)
│   │   └── world.go                 # Zone/role configuration and adjacency
│   ├── io/
│   │   ├── router.go                # Applies MediaPlans to transport
│   │   ├── plan.go                  # MediaPlan / MediaNode / MediaEdge types
│   │   ├── claims.go                # ClaimManager (per-resource preemption)
│   │   ├── recorder.go              # Records IO streams to disk
│   │   ├── mixer.go                 # Audio mixing for multi-party
│   │   └── transcoder.go            # Format conversion
│   ├── intent/
│   │   ├── bus.go                   # Typed Intent/Event dispatch
│   │   ├── voice_parser.go          # Voice transcript → Intent
│   │   ├── schedule.go              # Scheduler → Event
│   │   └── webhook.go               # External → Intent/Event
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
│   │   ├── engine.go                # Matches intents/events, supervises activations
│   │   ├── definition.go            # ScenarioDefinition interface
│   │   ├── activation.go            # ScenarioActivation + ActivationRecord
│   │   ├── recipe.go                # ScenarioRecipe workflow builder
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

- **Device Manager**: Maintains the registry of connected devices, their capabilities, and per-device state including zones, roles, mobility, and affinity.
- **Placement Engine**: Resolves `TargetScope` queries ("kitchen", "nearest screen", "all cameras") into concrete `[]DeviceRef` sets using device metadata. See [placement.md](placement.md).
- **IO Router + Media Planner**: Compiles declarative `MediaPlan`s into transport messages, manages the live media graph, and emits analyzer-derived events onto the intent/event bus. See [io-abstraction.md](io-abstraction.md).
- **Claim Manager**: Arbitrates per-resource claims (speakers, main screen, overlay, mic, camera, PTY) across activations; drives preemption, suspension, and restoration. Hosted alongside the IO Router. See [io-abstraction.md](io-abstraction.md#resource-claims).
- **Intent/Event Bus**: Normalized trigger ingress from voice, UI, schedule, IO analyzers, webhooks, and automation agents — all produce `Intent` or `Event` records. See [scenario-engine.md](scenario-engine.md#triggers-intents-and-events).
- **Scenario Engine**: Matches intents/events to scenario definitions, constructs activations, resolves targets, requests claims, and supervises lifecycle including suspend/resume. See [scenario-engine.md](scenario-engine.md).
- **AI Backend**: Pluggable interfaces for STT, TTS, LLM, vision, and sound classification. See [technology.md](technology.md#ai-backend-pluggable).
- **Telephony Bridge**: SIP client + WebRTC/SIP bridge for external calls.
- **Storage**: SQLite for config/state (including activation records for crash recovery), filesystem for media, dedicated store for timers and reminders.

## Related Plans

- [protocol.md](protocol.md) — Wire contract with clients.
- [scenario-engine.md](scenario-engine.md) — Definitions, activations, intents/events, recipes.
- [placement.md](placement.md) — Semantic target resolution.
- [io-abstraction.md](io-abstraction.md) — Media plans, claims, and resource kinds.
- [server-driven-ui.md](server-driven-ui.md) — UI descriptor generation.
- [technology.md](technology.md) — Library/framework choices.
