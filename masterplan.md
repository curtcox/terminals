# Terminals — Master Plan

A client/server system where devices on the same network serve as terminals for a single unified system. The server orchestrates all behavior; clients are generic IO surfaces that never need updating as new capabilities are added.

## Table of Contents

- [Vision](#vision)
- [Architecture Overview](#architecture-overview)
- [Client Architecture](#client-architecture)
- [Server Architecture](#server-architecture)
- [Protocol Design](#protocol-design)
- [Discovery and Connection](#discovery-and-connection)
- [IO Abstraction Layer](#io-abstraction-layer)
- [Server-Driven UI](#server-driven-ui)
- [Scenario Engine](#scenario-engine)
- [Use Cases](#use-cases)
- [Technology Choices](#technology-choices)
- [Agent Configuration](#agent-configuration)
- [Code Quality and CI](#code-quality-and-ci)
- [Development Phases](#development-phases)

---

## Vision

Every screen, speaker, microphone, and sensor in the home becomes part of a single system. A Chromebook on the kitchen counter is an intercom. A tablet on the wall is a smart photo frame — until someone says "red alert" and every screen in the house lights up. A phone on the nightstand listens for the dishwasher to stop. The old laptop in the kid's room watches the clock and says "you're going to be late."

None of this requires updating the client app. The Flutter client is a generic terminal — it reports its capabilities and does what the server tells it. All intelligence, all scenarios, all behavior lives on the server. Adding a new scenario means writing server-side code only.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                     Mac mini (Server)                    │
│                                                         │
│  ┌─────────────┐  ┌──────────────┐  ┌───────────────┐  │
│  │  Scenario    │  │  IO Router   │  │  AI Backend   │  │
│  │  Engine      │  │              │  │  (pluggable)  │  │
│  └──────┬───── │  └──────┬───────│  └───────┬───────│  │
│         │       │         │       │          │        │  │
│  ┌──────┴───────┴─────────┴───────┴──────────┴──────┐│  │
│  │              Device Manager                       ││  │
│  │     (registry, capabilities, state, routing)      ││  │
│  └──────────────────────┬────────────────────────────┘│  │
│                         │                              │  │
│  ┌──────────────────────┴────────────────────────────┐│  │
│  │              Transport Layer                       ││  │
│  │         gRPC (control) + WebRTC (media)            ││  │
│  └──────────────────────┬────────────────────────────┘│  │
│                         │                              │  │
│  ┌──────────────────────┴────────────────────────────┐│  │
│  │              Telephony Bridge                      ││  │
│  │           SIP/VoIP (external calls)                ││  │
│  └────────────────────────────────────────────────────┘│  │
└─────────────────────────┬───────────────────────────────┘
                          │ LAN (mDNS discovery)
          ┌───────────────┼───────────────┐
          │               │               │
     ┌────┴────┐    ┌────┴────┐    ┌────┴────┐
     │  Phone  │    │ Tablet  │    │ Laptop  │
     │ (Flutter)│   │(Flutter)│    │(Flutter)│
     └─────────┘    └─────────┘    └─────────┘
```

## Client Architecture

### Design Principle

The client is a **generic IO terminal**. It has no scenario-specific logic. It:

1. Discovers the server (or accepts a manually specified address)
2. Registers itself and declares its capabilities
3. Receives commands and executes them
4. Streams IO data as directed

### Capability Declaration

On connection, the client sends a capability manifest:

```json
{
  "device_id": "uuid",
  "device_name": "Kitchen Chromebook",
  "device_type": "laptop",
  "platform": "chromeos",
  "capabilities": {
    "screen": {
      "width": 1920,
      "height": 1080,
      "density": 1.0,
      "touch": false
    },
    "keyboard": {
      "physical": true,
      "layout": "en-US"
    },
    "pointer": {
      "type": "trackpad",
      "hover": true
    },
    "touch": null,
    "speakers": {
      "channels": 2,
      "sample_rates": [44100, 48000]
    },
    "microphone": {
      "channels": 1,
      "sample_rates": [16000, 44100, 48000]
    },
    "camera": {
      "front": { "width": 1280, "height": 720, "fps": 30 },
      "back": null
    },
    "bluetooth": { "version": "5.0" },
    "accelerometer": null,
    "gyroscope": null,
    "compass": null,
    "wifi": { "signal_strength": true },
    "usb": { "host": true, "ports": 2 },
    "gps": null,
    "nfc": null,
    "haptic": null,
    "ambient_light": null,
    "proximity": null,
    "battery": { "level": 0.85, "charging": true }
  }
}
```

A capability is present if non-null. The server uses this to determine what the device can do — it will never send a command the device can't handle.

### Client Module Structure

```
terminal_client/
├── lib/
│   ├── main.dart
│   ├── discovery/
│   │   ├── mdns_scanner.dart        # mDNS/Bonjour discovery
│   │   └── manual_connect.dart      # Manual server entry UI
│   ├── connection/
│   │   ├── grpc_channel.dart        # gRPC control channel
│   │   └── webrtc_manager.dart      # WebRTC media streams
│   ├── capabilities/
│   │   ├── capability_registry.dart # Enumerate device capabilities
│   │   ├── screen_cap.dart
│   │   ├── audio_cap.dart
│   │   ├── camera_cap.dart
│   │   ├── sensor_cap.dart
│   │   └── peripheral_cap.dart
│   ├── io/
│   │   ├── io_controller.dart       # Dispatches server commands to IO
│   │   ├── screen_renderer.dart     # Renders server-driven UI
│   │   ├── audio_streamer.dart      # Mic capture / speaker playback
│   │   ├── video_streamer.dart      # Camera capture / screen display
│   │   ├── keyboard_input.dart      # Keyboard event forwarding
│   │   ├── pointer_input.dart       # Mouse/trackpad forwarding
│   │   ├── touch_input.dart         # Touch event forwarding
│   │   └── sensor_streamer.dart     # Accelerometer, gyro, compass
│   └── ui/
│       ├── server_driven_ui.dart    # Renders UI from server descriptors
│       └── fallback_ui.dart         # Connection/discovery UI only
```

### Platform Support

| Platform    | Build Target           | Notes                              |
|-------------|------------------------|------------------------------------|
| Android     | Flutter Android        | Phones and tablets                 |
| iOS         | Flutter iOS            | Phones and tablets                 |
| Web/Browser | Flutter Web            | Any device with a modern browser   |
| macOS       | Flutter macOS          | Laptops                           |
| Linux       | Flutter Linux          | Chromebooks (Linux container), PCs |
| Windows     | Flutter Windows        | Laptops, desktops                  |

Flutter's cross-platform support means a single codebase produces all client variants. Platform-specific capability detection (e.g., accelerometer on mobile, USB on desktop) uses platform channels where Flutter plugins don't cover it.

## Server Architecture

### Overview

The server is a Go application running on a Mac mini. It is the brain of the system — all behavior, decision-making, and IO routing originates here.

### Server Module Structure

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

## Protocol Design

The protocol has two layers: a **control plane** (gRPC) for commands and state, and a **media plane** (WebRTC) for real-time audio/video/data streams.

### Control Plane (gRPC)

Bidirectional streaming RPCs over gRPC. The client maintains a persistent control stream to the server.

```protobuf
service TerminalControl {
  // Persistent bidirectional control stream
  rpc Connect(stream ClientMessage) returns (stream ServerMessage);
}

message ClientMessage {
  oneof payload {
    RegisterDevice     register        = 1;
    CapabilityUpdate   capability      = 2;
    InputEvent         input           = 3;  // keyboard, pointer, touch
    SensorData         sensor          = 4;
    StreamReady        stream_ready    = 5;  // WebRTC session established
    CommandAck         ack             = 6;
    Heartbeat          heartbeat       = 7;
  }
}

message ServerMessage {
  oneof payload {
    RegisterAck        register_ack    = 1;
    SetUI              set_ui          = 2;  // Server-driven UI descriptor
    StartStream        start_stream    = 3;  // Begin audio/video/sensor stream
    StopStream         stop_stream     = 4;
    PlayAudio          play_audio      = 5;  // Play audio clip or TTS
    ShowMedia          show_media      = 6;  // Display image/video
    RouteStream        route_stream    = 7;  // Connect stream to another device
    Notification       notification    = 8;  // Toast/alert
    WebRTCSignal       webrtc_signal   = 9;  // SDP/ICE for media setup
    CommandRequest     command         = 10; // Generic command
    Heartbeat          heartbeat       = 11;
  }
}
```

### Media Plane (WebRTC)

WebRTC peer connections carry real-time media between clients and the server. The server acts as an SFU (Selective Forwarding Unit) — it receives media streams from clients and selectively forwards them to other clients or processes them locally.

```
Client A (mic) ──WebRTC──→ Server ──WebRTC──→ Client B (speaker)
                              │
                              ├──→ AI (speech-to-text)
                              └──→ Disk (recording)
```

WebRTC is used instead of raw streaming because:
- Built-in echo cancellation, noise suppression, and automatic gain control
- Adaptive bitrate based on network conditions
- NAT traversal (future-proofing for off-network use)
- Flutter has mature WebRTC support (`flutter_webrtc`)

### Data Streams (WebRTC DataChannel)

For non-media IO like sensor data, keyboard events, and low-latency commands, WebRTC DataChannels provide an unreliable-ordered or reliable channel alongside the media streams.

## Discovery and Connection

### Automatic Discovery (mDNS)

The server advertises itself via mDNS (Bonjour) on the local network:

- Service type: `_terminals._tcp.local.`
- TXT records: `version=1`, `name=HomeServer`

The client scans for this service on startup. On a trusted home LAN, no authentication is required — the first server found is used automatically. If multiple servers exist (not planned, but handled gracefully), the client presents a list.

### Manual Connection

If mDNS fails (e.g., network segmentation, mDNS blocked), the client shows a simple screen:
- Server address text field (IP or hostname)
- Port number (default pre-filled)
- Connect button

This is the **only** client-native UI. Everything else is server-driven.

### Connection Lifecycle

```
Client                          Server
  │                                │
  │──── mDNS query ───────────────→│ (or manual IP)
  │←─── mDNS response ────────────│
  │                                │
  │──── gRPC Connect ─────────────→│
  │──── RegisterDevice ───────────→│
  │←─── RegisterAck ──────────────│
  │                                │
  │←─── SetUI (initial screen) ───│
  │←─── StartStream (if needed) ──│
  │                                │
  │←──→ Heartbeat (periodic) ←──→ │
  │                                │
  │     (ongoing command/event     │
  │      exchange on the stream)   │
```

On disconnect, the client returns to the discovery/manual connect screen. On reconnect, the server restores the device's previous state if still applicable.

## IO Abstraction Layer

Every IO capability maps to a uniform interface on both client and server.

### IO Categories

| Category      | Inputs (client → server)         | Outputs (server → client)         |
|---------------|----------------------------------|-----------------------------------|
| **Screen**    | —                                | UI descriptors, images, video     |
| **Keyboard**  | Key events (down, up, char)      | —                                 |
| **Pointer**   | Move, click, scroll, hover       | Cursor changes                    |
| **Touch**     | Touch start/move/end, gestures   | —                                 |
| **Audio**     | Mic PCM/Opus stream              | Speaker PCM/Opus stream, clips    |
| **Video**     | Camera H.264/VP8 stream          | Display H.264/VP8 stream          |
| **Bluetooth** | Scan results, device data        | Scan commands, connect commands   |
| **Sensors**   | Accelerometer, gyro, compass     | —                                 |
| **WiFi**      | Signal strength, scan results    | Scan commands                     |
| **USB**       | Device enumeration, data         | Data, commands                    |
| **GPS**       | Location updates                 | —                                 |
| **Haptic**    | —                                | Vibration patterns                |
| **Battery**   | Level, charging state            | —                                 |

### IO Routing

The server's IO Router can:

- **Consume** a stream: mic audio → speech-to-text
- **Produce** a stream: TTS output → speaker
- **Forward** a stream: Client A mic → Client B speaker (intercom)
- **Fork** a stream: mic audio → STT + recording + Client B speaker
- **Mix** streams: multiple mic streams → single mixed output
- **Composite** streams: multiple video streams → grid/layout on a single screen (each stream bound to its own `video_surface` in the UI)
- **Record** a stream: any stream → disk
- **Analyze** a stream: audio → sound classifier, video → vision model

All routing is dynamic and reconfigurable at runtime. The client doesn't know or care where its streams go.

## Server-Driven UI

The server sends UI descriptors that the client renders. This is the mechanism by which the client displays anything beyond the connection screen.

### UI Descriptor Format

```json
{
  "type": "stack",
  "children": [
    {
      "type": "text",
      "value": "user@home:~$ ",
      "style": "monospace",
      "color": "#00FF00"
    },
    {
      "type": "text_input",
      "id": "terminal_input",
      "style": "monospace",
      "autofocus": true
    }
  ],
  "background": "#000000"
}
```

### Supported UI Components

The client renders a fixed set of primitive UI components. These primitives are rich enough to compose any interface the server needs:

- **Layout**: `stack` (vertical), `row` (horizontal), `grid`, `scroll`, `padding`, `center`, `expand`
- **Content**: `text`, `image`, `video_surface`, `audio_visualizer`, `canvas`
- **Input**: `text_input`, `button`, `slider`, `toggle`, `dropdown`, `gesture_area`
- **Feedback**: `notification`, `overlay`, `progress`
- **System**: `fullscreen`, `keep_awake`, `brightness`

Because these are generic primitives, the server can compose them into:
- A terminal emulator (monospace text + text input)
- A video call UI (video surfaces + mute button)
- A photo frame (fullscreen image + timer-based rotation)
- An intercom panel (push-to-talk button + audio visualizer)
- An alert screen (red background + flashing text + alarm audio)
- A PA console (audio visualizer + mute toggle + end button)
- A multi-camera grid (grid of video surfaces + device labels + audio controls)
- Any future UI without client changes

### UI Updates

The server can:
- **Replace** the entire UI: `SetUI` with a full descriptor
- **Patch** the UI: `UpdateUI` targeting a component by ID
- **Animate** transitions: `TransitionUI` with from/to states and duration

## Scenario Engine

Scenarios are server-side modules that implement specific behaviors. They are the only place where "what the system does" is defined.

### Scenario Interface

```go
type Scenario interface {
    // Name returns the scenario identifier.
    Name() string

    // Match reports whether this scenario should activate
    // given the current trigger (voice command, schedule, event, etc.).
    Match(trigger Trigger) bool

    // Start activates the scenario on the given set of devices.
    Start(ctx context.Context, env *Environment) error

    // Stop deactivates the scenario, releasing all resources.
    Stop() error
}

type Environment struct {
    Devices    DeviceManager    // Query and command devices
    IO         IORouter         // Route IO streams
    AI         AIBackend        // Speech, vision, LLM, etc.
    Telephony  TelephonyBridge  // External calls
    Storage    StorageManager   // Persistence
    Scheduler  Scheduler        // Timers and reminders
    Broadcast  Broadcaster      // Send to all/subset of devices
}
```

### Scenario Activation

Scenarios activate via triggers:

- **Voice**: User says a wake word + command → STT → scenario matching
- **Schedule**: Cron-like time triggers (e.g., check school schedule at 7:30 AM)
- **Event**: An IO analysis result (e.g., sound classifier detects silence after running water)
- **Manual**: User selects a scenario via the UI
- **Cascade**: One scenario triggers another (e.g., "red alert" stops all other scenarios)

### Scenario Priority and Preemption

Scenarios have priority levels. Higher-priority scenarios can preempt lower-priority ones on a device:

| Priority | Examples                        |
|----------|---------------------------------|
| Critical | Red alert, emergency            |
| High     | Active phone call, intercom, PA |
| Normal   | Terminal session, voice query, multi-window |
| Low      | Photo frame, ambient monitoring |
| Idle     | Clock display, standby screen   |

When a higher-priority scenario needs a device, the lower-priority scenario is suspended (not terminated). When the higher-priority scenario ends, the suspended one resumes.

## Use Cases

Each use case maps to a scenario. The client code is identical in every case — only the server scenario differs.

### Text Terminal

**Trigger**: User selects "Terminal" from device menu, or server auto-assigns to laptops/Chromebooks.

**Flow**:
1. Server sends `SetUI` with a terminal layout (monospace text area + input field).
2. Client forwards keyboard input events via gRPC.
3. Server runs a PTY (pseudo-terminal) session, sends output back as UI text updates.
4. Server can multiplex — multiple terminals on one device or one terminal session accessed from multiple devices.

### Audio and Video Calls

**Trigger**: Voice command ("call Mom") or UI action.

**Flow**:
1. Server looks up contact, determines call type (internal client-to-client or external via SIP).
2. For internal: establishes WebRTC streams between two clients via the server SFU.
3. For external: SIP client initiates outbound call, server bridges WebRTC ↔ SIP.
4. Server sends call UI to both devices (video surface, mute/hangup buttons).
5. On hangup, server tears down streams and restores previous UI/scenario.

### Intercom

**Trigger**: Voice command ("intercom to kitchen") or button press.

**Flow**:
1. Server routes mic audio from source device to speaker on target device (and vice versa).
2. Server sends intercom UI to both devices (visual indicator + active speaker display).
3. Can be one-way (announcement) or two-way (conversation).
4. Can target all devices simultaneously (whole-house announcement).

### Smart Speaker / Voice Assistant

**Trigger**: Wake word detected (server continuously analyzes mic audio from idle devices).

**Flow**:
1. Client streams mic audio to server.
2. Server runs STT on the audio.
3. Transcribed text goes to LLM for response generation.
4. Response text goes through TTS.
5. Audio response plays on the device's speaker.
6. If the response includes visual content (weather, recipe, etc.), server sends `SetUI` with the content alongside audio.

### Smart Photo Frame

**Trigger**: Device is idle, or user selects "Photo Frame" mode.

**Flow**:
1. Server sends `SetUI` with a fullscreen image + `keep_awake` flag.
2. Server rotates images on a timer (configurable interval).
3. Photos sourced from a configured directory on the server.
4. Preemptable — any higher-priority scenario takes over the screen, photo frame resumes after.

### Timers and Reminders

**Trigger**: Voice command ("set a timer for 10 minutes", "remind me to check the oven at 3 PM").

**Flow**:
1. STT + LLM extract the timer/reminder intent and parameters.
2. Server stores the timer/reminder in the scheduler.
3. When triggered, server plays a TTS announcement on the originating device (or all devices).
4. Server sends a notification UI overlay until dismissed.

### Audio Monitoring ("Tell Me When the Dishwasher Stops")

**Trigger**: Voice command.

**Flow**:
1. Server begins streaming mic audio from the nearest device to the sound classification AI.
2. AI backend monitors for the transition from "machine running" to "silence" (or a specific end-of-cycle beep pattern).
3. When detected, server sends notification to the requesting device (or all devices) via TTS + UI notification.
4. Monitoring stream stops to conserve resources.

### Schedule Monitoring ("Watch My Child")

**Trigger**: Configured schedule (e.g., school days at 7:00 AM) or voice command.

**Flow**:
1. At trigger time, server activates camera on the relevant device.
2. Video stream analyzed by vision AI for activity/presence detection.
3. Server cross-references with stored schedule data (school starts at 8:15, bus comes at 7:50, etc.).
4. If child appears to be behind schedule, server sends TTS warning through the device speaker: "It's 7:40 and you haven't left yet — the bus comes in 10 minutes."
5. Can escalate to parent notification on another device.

### Red Alert

**Trigger**: Voice command ("red alert") or UI action.

**Flow**:
1. Server preempts ALL active scenarios on ALL devices (critical priority).
2. Server sends `SetUI` to every device: red background, flashing alert text, alarm icon.
3. Server plays alarm audio on every device with speakers.
4. Server optionally activates all cameras and mics for full situational awareness.
5. Dismissed by voice command ("stand down") or UI action on any device.
6. All previously active scenarios resume.

### PA System

**Trigger**: Voice command ("PA mode") or UI action on the broadcasting device.

**Flow**:
1. Server designates the triggering device as the PA source.
2. Server sends `StartStream` (mic) to the source device.
3. Server sends `StopStream` (mic) to all other devices to avoid feedback loops, and mutes any active scenario audio on their speakers.
4. Server forks the incoming mic audio stream from the source and forwards it to the speakers of all other connected devices simultaneously.
5. The source device gets a PA UI: a live audio visualizer, a mute/unmute toggle, and an "End PA" button.
6. Receiving devices get a notification overlay ("PA from {device_name}") on top of their current UI — their active scenario is **not** preempted, only their audio output is temporarily taken over.
7. On end, server restores normal audio routing and removes the overlay from receiving devices.

**Key IO Router operations**: Fork (one mic → many speakers), with echo suppression via muting receiver mics.

```
Source (mic) ──WebRTC──→ Server ──fork──→ Speaker A
                                 ├──────→ Speaker B
                                 ├──────→ Speaker C
                                 └──────→ Speaker N
```

### Multi-Window (Security Camera / Multi-Feed View)

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

## Technology Choices

### Server

| Component       | Technology                | Rationale                                    |
|-----------------|---------------------------|----------------------------------------------|
| Language        | Go                        | Specified requirement                        |
| RPC Framework   | gRPC + protobuf           | Bidirectional streaming, strong typing, codegen for Go + Dart |
| Media Transport | WebRTC (Pion)             | Go-native WebRTC. SFU capability, adaptive bitrate |
| SIP/VoIP       | Pion/SIP or go-ozzo/sip   | External phone calls                         |
| Database        | SQLite (via modernc.org)  | Zero-dependency, sufficient for single-server |
| mDNS            | hashicorp/mdns            | Service discovery on LAN                     |
| AI Interface    | Plugin-based (Go interfaces) | Swap implementations without changing server core |
| Config          | YAML                      | Human-readable server configuration          |

### Client

| Component       | Technology                | Rationale                                    |
|-----------------|---------------------------|----------------------------------------------|
| Framework       | Flutter                   | Specified requirement. Single codebase for all platforms |
| gRPC            | grpc-dart                 | Matches server gRPC. Protobuf codegen for Dart |
| WebRTC          | flutter_webrtc            | Mature, cross-platform WebRTC                |
| mDNS            | multicast_dns / nsd       | Platform-aware mDNS scanning                 |
| Sensors         | sensors_plus              | Accelerometer, gyroscope, compass            |
| Camera          | camera                    | Cross-platform camera access                 |
| Audio           | Built into WebRTC + record | Recording and playback                       |
| Bluetooth       | flutter_blue_plus         | BLE scanning and connection                  |
| Platform channels | Custom per-platform      | USB, NFC, and other platform-specific IO     |

### AI Backend (Pluggable)

The server defines Go interfaces for each AI capability. Implementations can target local or cloud services:

```go
type SpeechToText interface {
    Transcribe(ctx context.Context, audio io.Reader, opts STTOptions) (<-chan Transcript, error)
}

type TextToSpeech interface {
    Synthesize(ctx context.Context, text string, opts TTSOptions) (io.Reader, error)
}

type LLM interface {
    Query(ctx context.Context, messages []Message, opts LLMOptions) (*Response, error)
}

type VisionAnalyzer interface {
    Analyze(ctx context.Context, frame image.Image, prompt string) (*Analysis, error)
}

type SoundClassifier interface {
    Classify(ctx context.Context, audio io.Reader) (<-chan SoundEvent, error)
}
```

Potential implementations:
- **Local**: Whisper (STT), Piper (TTS), llama.cpp (LLM), ONNX Runtime (vision/sound)
- **Cloud**: OpenAI, Google Cloud, Anthropic, ElevenLabs, Deepgram

The choice of implementation is a server configuration decision, not an architectural one.

## Agent Configuration

Development is primarily driven by Claude Code and Codex. The repo must contain configuration files that give these agents the context they need to work effectively.

### Repo Root Files

```
terminals/
├── CLAUDE.md                    # Claude Code project instructions
├── AGENTS.md                    # Codex agent instructions (same role, Codex convention)
├── .github/
│   ├── workflows/
│   │   ├── server-ci.yml        # Go CI pipeline
│   │   ├── client-ci.yml        # Flutter CI pipeline
│   │   └── proto-ci.yml         # Protobuf lint and breaking change detection
│   ├── copilot-instructions.md  # GitHub Copilot context (optional)
│   └── CODEOWNERS               # PR review routing
├── terminal_server/
│   └── CLAUDE.md                # Server-specific agent instructions
├── terminal_client/
│   └── CLAUDE.md                # Client-specific agent instructions
└── api/
    └── CLAUDE.md                # Proto-specific agent instructions
```

### CLAUDE.md (Root)

The root `CLAUDE.md` gives any agent a map of the project:

- Project overview: what this system is, the thin-client architecture, the "client never changes" constraint.
- Repo layout: where server, client, and proto code live.
- Build commands: how to build, test, lint, and run each component.
- Key architectural rules agents must follow:
  - All new behavior goes in server-side scenarios — never add scenario logic to the client.
  - All IO between client and server is defined in protobuf — never use ad-hoc serialization.
  - AI backends are behind interfaces — never import a specific AI provider directly in scenario code.
  - Server-driven UI uses only the defined primitive components — never add client-side UI components for specific scenarios.
- How to run the full system locally (server + one client).
- Links to this masterplan for deeper context.

### AGENTS.md (Root)

Same content as `CLAUDE.md` — Codex reads `AGENTS.md` by convention. Can be a symlink or a copy with Codex-specific additions (e.g., sandbox setup commands, environment variables for headless testing).

### Subproject CLAUDE.md Files

Each subproject (`terminal_server/`, `terminal_client/`, `api/`) gets its own `CLAUDE.md` with:

- Language/framework-specific conventions (Go idioms, Flutter patterns, proto style).
- How to run tests for that subproject alone.
- Common pitfalls and patterns specific to that codebase.
- Dependency management instructions (go mod, pub, buf).

### Additional Agent-Useful Files

| File | Purpose |
|------|---------|
| `.editorconfig` | Consistent indentation/encoding across editors and agents |
| `.gitignore` | Comprehensive ignores for Go, Flutter, proto, IDE files, build artifacts |
| `Makefile` | Unified build/test/lint commands agents can discover and run |
| `README.md` | Human-readable project overview (agents also read this) |
| `terminal_server/go.mod` | Go module definition — agents need this to understand import paths |
| `terminal_client/pubspec.yaml` | Flutter dependencies — agents need this to understand available packages |
| `api/buf.yaml` | Buf configuration for proto linting and breaking change detection |
| `api/buf.gen.yaml` | Buf codegen config — agents need this to regenerate proto bindings |
| `.github/dependabot.yml` | Automated dependency updates for Go and Flutter |

## Code Quality and CI

Every PR is validated by GitHub Actions. Agents should be able to run the same checks locally before pushing.

### Go (Server)

| Tool | Purpose | CI Check |
|------|---------|----------|
| `go test ./...` | Unit and integration tests | Required to pass |
| `go test -race ./...` | Race condition detection | Required to pass |
| `go test -coverprofile` | Code coverage report | Report uploaded, trend tracked |
| `golangci-lint` | Meta-linter (staticcheck, errcheck, govet, gosimple, ineffassign, unused, etc.) | Required to pass |
| `go vet ./...` | Compiler-level static analysis | Included in golangci-lint |
| `govulncheck` | Known vulnerability detection in dependencies | Required to pass |
| `gofumpt` | Strict formatting (superset of gofmt) | Required — formatting is checked, not auto-fixed |
| `buf lint` | Protobuf style and correctness (run in server CI too) | Required to pass |

**golangci-lint configuration** (`.golangci.yml` in `terminal_server/`):
- Enable: `errcheck`, `staticcheck`, `govet`, `gosimple`, `ineffassign`, `unused`, `gocritic`, `revive`, `misspell`, `prealloc`, `bodyclose`, `exhaustive`
- Enforce: `gofumpt` formatting
- Set appropriate thresholds for cyclomatic complexity

### Flutter (Client)

| Tool | Purpose | CI Check |
|------|---------|----------|
| `flutter test` | Widget and unit tests | Required to pass |
| `flutter analyze` | Dart static analysis (dart analyzer) | Required to pass (zero issues) |
| `dart format --set-exit-if-changed .` | Formatting check | Required to pass |
| `flutter test --coverage` | Code coverage report | Report uploaded, trend tracked |
| `dart pub outdated` | Dependency freshness report | Informational (PR comment) |
| Custom lint rules (`custom_lint`) | Project-specific lint rules via `analysis_options.yaml` | Required to pass |

**analysis_options.yaml** in `terminal_client/`:
- Extend `flutter_lints` (or `very_good_analysis` for stricter rules)
- Enable `strict-casts`, `strict-inference`, `strict-raw-types`
- Project-specific rules: no direct platform imports in non-capability code, etc.

### Protobuf

| Tool | Purpose | CI Check |
|------|---------|----------|
| `buf lint` | Proto style guide enforcement | Required to pass |
| `buf breaking` | Backward compatibility check against main branch | Required to pass |
| `buf generate` | Codegen (Go + Dart) — verify generated code is committed and up to date | Required to pass |
| `buf format -d --exit-code` | Proto formatting | Required to pass |

### CI Pipeline Structure

```yaml
# .github/workflows/server-ci.yml — triggers on changes to terminal_server/ or api/
# .github/workflows/client-ci.yml — triggers on changes to terminal_client/ or api/
# .github/workflows/proto-ci.yml  — triggers on changes to api/
```

Each pipeline:
1. Checks out the repo.
2. Sets up the toolchain (Go, Flutter, Buf).
3. Caches dependencies (`go mod cache`, `pub cache`, `buf cache`).
4. Runs formatting check (fail fast).
5. Runs linters.
6. Runs tests with coverage.
7. Uploads coverage reports (Codecov or similar).
8. For proto: runs `buf breaking` against `origin/main`.

### Makefile

A top-level `Makefile` provides a unified interface for agents and humans:

```makefile
# Top-level targets that agents can discover and run
make server-build      # Build the Go server
make server-test       # Run server tests
make server-lint       # Run golangci-lint
make server-coverage   # Run tests with coverage report
make client-build      # Build the Flutter client (all platforms)
make client-test       # Run Flutter tests
make client-lint       # Run flutter analyze + dart format check
make client-coverage   # Run tests with coverage report
make proto-lint        # Lint proto files with buf
make proto-breaking    # Check proto backward compatibility
make proto-generate    # Regenerate Go + Dart bindings from proto
make all-lint          # Run all linters
make all-test          # Run all tests
make all-check         # Full CI-equivalent check (lint + test + proto)
make run-server        # Start the server locally
make run-client-web    # Start the Flutter web client locally
```

## Development Phases

### Phase 0 — Repo Setup, Tooling, and CI

Establish the repo structure, agent configuration, code quality tooling, and CI pipelines before writing any application code. This phase ensures that every subsequent phase starts with working builds, linting, and tests from the first commit.

- [ ] **Repo structure**: Create `terminal_server/`, `terminal_client/`, `api/proto/` directories.
- [ ] **Go module init**: `go mod init` in `terminal_server/` with initial `main.go` that compiles.
- [ ] **Flutter project init**: `flutter create` in `terminal_client/` with default app that builds.
- [ ] **Buf init**: `buf.yaml` and `buf.gen.yaml` in `api/` with Go and Dart codegen configured.
- [ ] **Root CLAUDE.md**: Project overview, repo layout, build commands, architectural rules, local dev instructions.
- [ ] **Root AGENTS.md**: Codex-compatible version of CLAUDE.md (symlink or tailored copy).
- [ ] **Subproject CLAUDE.md files**: One each in `terminal_server/`, `terminal_client/`, `api/` with language-specific conventions.
- [ ] **.editorconfig**: Tabs for Go, 2-space for Dart/proto/YAML, UTF-8, final newline.
- [ ] **.gitignore**: Comprehensive ignores for Go, Flutter, proto generated code, IDE files, OS files, build artifacts.
- [ ] **Makefile**: All targets listed in the Code Quality section — `make all-check` works from day one.
- [ ] **golangci-lint config**: `.golangci.yml` in `terminal_server/` with the linters listed above.
- [ ] **Flutter analysis config**: `analysis_options.yaml` in `terminal_client/` with strict rules.
- [ ] **GitHub Actions — server CI**: `.github/workflows/server-ci.yml` — build, lint, test, coverage, govulncheck.
- [ ] **GitHub Actions — client CI**: `.github/workflows/client-ci.yml` — build, analyze, format check, test, coverage.
- [ ] **GitHub Actions — proto CI**: `.github/workflows/proto-ci.yml` — buf lint, buf format, buf breaking.
- [ ] **Dependabot config**: `.github/dependabot.yml` for Go modules and pub packages.
- [ ] **README.md**: Brief project description, build prerequisites, quick start instructions.

**Milestone**: Empty project skeleton where `make all-check` passes, all three CI pipelines go green, and agents have full context via CLAUDE.md / AGENTS.md.

### Phase 1 — Foundation

Establish the core client-server communication and prove the architecture.

- [ ] **Proto definitions**: Define the gRPC protobuf schemas for control messages, capability declarations, and UI descriptors.
- [ ] **Buf codegen**: `buf generate` produces Go and Dart bindings; CI verifies generated code is up to date.
- [ ] **Server skeleton**: Go project with gRPC server, device manager, and mDNS advertisement.
- [ ] **Client skeleton**: Flutter app with mDNS discovery, manual connect fallback, gRPC connection, and capability reporting.
- [ ] **Server-driven UI**: Client renders basic UI descriptors from the server (text, buttons, layout). Server sends a "hello world" UI on connect.
- [ ] **Heartbeat and reconnection**: Connection health monitoring and automatic reconnection.
- [ ] **Tests from the start**: Unit tests for proto serialization, device registration, capability parsing. CI enforces passing tests and lint on every PR.

**Milestone**: Client connects to server, sends capabilities, server sends a UI that the client renders. CI is green.

### Phase 2 — Text Terminal

First real use case. Validates keyboard input forwarding and text-based server-driven UI.

- [ ] **PTY management**: Server spawns and manages pseudo-terminal sessions.
- [ ] **Terminal UI descriptor**: Monospace scrollable text output + text input.
- [ ] **Keyboard forwarding**: Client sends key events, server feeds them to PTY.
- [ ] **Terminal output**: Server captures PTY output, sends UI updates to client.

**Milestone**: Use a Chromebook or laptop as a functional text terminal into the Mac mini.

### Phase 3 — Media Streams

Enable audio and video streaming between clients and server.

- [ ] **WebRTC integration (server)**: Pion-based SFU — accept, forward, and process media streams.
- [ ] **WebRTC integration (client)**: flutter_webrtc — send/receive audio and video.
- [ ] **Signaling over gRPC**: SDP and ICE candidate exchange through the existing control channel.
- [ ] **IO Router**: Server-side routing of media streams between devices.
- [ ] **Audio playback**: Server sends audio clips (TTS, alerts) to specific devices.

**Milestone**: Stream audio from one client's mic to another client's speaker.

### Phase 4 — Intercom and Calls

Build on media streams for communication scenarios.

- [ ] **Intercom scenario**: Voice-activated or button-activated two-way audio between devices.
- [ ] **Whole-house announcement**: One-to-many audio broadcast.
- [ ] **PA system scenario**: One mic → all speakers with feedback suppression and PA overlay UI.
- [ ] **Audio mixer**: Server-side mixing of multiple audio streams into a single output track.
- [ ] **Multi-window scenario**: Grid UI of all camera feeds on one device with mixed or selectable audio.
- [ ] **Internal video call**: Client-to-client video call through the server SFU.
- [ ] **SIP integration**: Register with a SIP provider for external phone calls.
- [ ] **WebRTC-SIP bridge**: Bridge internal WebRTC streams to external SIP calls.

**Milestone**: Intercom between rooms. Place a phone call from any client.

### Phase 5 — Voice Assistant

Add AI-powered voice interaction.

- [ ] **AI backend interfaces**: Define and implement the pluggable AI interfaces.
- [ ] **Wake word detection**: Continuous low-power audio monitoring on idle devices.
- [ ] **STT pipeline**: Mic → server → speech-to-text.
- [ ] **LLM query pipeline**: Transcribed text → LLM → response text.
- [ ] **TTS pipeline**: Response text → TTS → speaker.
- [ ] **Rich responses**: Voice response + accompanying visual UI on the device screen.

**Milestone**: Say a wake word, ask a question, get a spoken and visual answer.

### Phase 6 — Monitoring and Alerts

Ambient intelligence scenarios.

- [ ] **Sound classification**: AI backend for detecting specific sounds (silence, beeps, alarms, etc.).
- [ ] **Audio monitoring scenario**: "Tell me when X stops" voice command handling and monitoring.
- [ ] **Timer and reminder scenario**: Voice-commanded timers and reminders with scheduler persistence.
- [ ] **Schedule monitoring scenario**: Time-triggered activity monitoring with escalating alerts.
- [ ] **Red alert scenario**: Broadcast preemption of all devices with alarm.

**Milestone**: "Tell me when the dishwasher stops" works end to end.

### Phase 7 — Polish and Expansion

Refinement, additional scenarios, and robustness.

- [ ] **Photo frame scenario**: Idle-screen photo rotation with preemption support.
- [ ] **Scenario priority and preemption**: Robust suspend/resume of scenarios across devices.
- [ ] **Multi-device scenario coordination**: Single scenario spanning multiple devices.
- [ ] **Sensor data streaming**: Accelerometer, gyroscope, compass data to server for future scenarios.
- [ ] **Bluetooth and USB passthrough**: Server-directed BLE scanning and USB device access.
- [ ] **Recording and playback**: Server records streams to disk, plays back on demand.
- [ ] **Admin UI**: Web-based dashboard for server configuration, device management, and scenario control.

**Milestone**: System handles all described use cases. New scenarios require only server-side code.

---

## Key Design Decisions

1. **Client is stateless (except connection state)**. All scenario logic, UI generation, and IO routing lives on the server. The client is a render engine + IO bridge.

2. **gRPC for control, WebRTC for media**. gRPC gives us strong typing, bidirectional streaming, and great codegen. WebRTC gives us battle-tested real-time media with built-in echo cancellation and adaptive bitrate.

3. **Server-driven UI with fixed primitives**. The client has a finite set of UI components it can render. The server composes them. This is the contract that lets the client stay unchanged while the server evolves.

4. **Pluggable AI backends**. The system doesn't couple to any specific AI provider. Interfaces allow swapping between local and cloud implementations based on the user's preference and hardware capability.

5. **Scenario engine with priority preemption**. Real-world use requires graceful handling of competing demands for device IO. Priority-based preemption with suspend/resume handles this cleanly.

6. **Trusted LAN, no auth**. For a home network, mDNS discovery + direct connection with no authentication keeps things simple. If this assumption changes, TLS mutual auth can be added at the transport layer without protocol changes.

7. **Agent-first development**. Claude Code and Codex are primary contributors. The repo is structured so agents can orient themselves (CLAUDE.md, AGENTS.md), run checks (`make all-check`), and validate their own work (CI). Code quality tools enforce consistency regardless of whether a human or agent wrote the code. Every linter, formatter, and test runs identically in local dev and CI — no "works on my machine" gaps.
