---
title: "Client Architecture"
kind: plan
status: planned
owner: unowned
validation: none
last-reviewed: 2026-04-25
---

# Client Architecture

See [masterplan.md](../archive/masterplan-duplicate.md) for overall system context.

## Design Principle

The client is a **generic IO terminal**. It has no scenario-specific logic. It:

1. discovers the server (or accepts a manually specified address)
2. connects and identifies itself
3. reports a full capability snapshot
4. watches for capability changes and reports deltas
5. receives commands and executes them
6. streams IO data as directed

## Capability Declaration

On connection, the client sends a capability snapshot describing the device's currently available IO endpoints.

A capability is not “supported by the product line”; it is present only if it is currently usable.

Example:

```json
{
  "device_id": "uuid",
  "device_name": "Kitchen Chromebook",
  "device_type": "laptop",
  "platform": "chromeos",
  "capabilities": {
    "displays": [
      {
        "id": "display-internal",
        "role": "main",
        "width_px": 1920,
        "height_px": 1080,
        "density": 1.0,
        "refresh_hz": 60,
        "touch": false,
        "available": true
      }
    ],
    "keyboards": [
      {
        "id": "kbd-internal",
        "kind": "physical",
        "layout": "en-US",
        "available": true
      }
    ],
    "pointers": [
      {
        "id": "ptr-trackpad",
        "kind": "trackpad",
        "hover": true,
        "available": true
      }
    ],
    "audio_outputs": [
      {
        "id": "speaker-internal",
        "kind": "built_in_speaker",
        "channels": 2,
        "sample_rates": [44100, 48000],
        "default_route": true,
        "available": true
      }
    ],
    "audio_inputs": [
      {
        "id": "mic-internal",
        "kind": "built_in_mic",
        "channels": 1,
        "sample_rates": [16000, 44100, 48000],
        "echo_canceled": true,
        "default_route": true,
        "available": true
      }
    ],
    "cameras": [
      {
        "id": "cam-front",
        "kind": "front",
        "modes": [
          {"width": 1280, "height": 720, "fps": 30}
        ],
        "available": true
      }
    ],
    "sensors": [
      {"id": "battery", "kind": "battery", "available": true}
    ],
    "peripherals": [
      {"id": "usb-bus", "kind": "usb_host", "available": true}
    ]
  }
}
```

The server uses this to determine what the device can do **right now** — it will never send a command or compile a media plan that requires absent endpoints.

Semantic placement (zone, role tags, mobility) is **not** part of the client-declared manifest. The client declares only what it physically is; the server assigns zone and role metadata via admin configuration. See [placement.md](placement.md).

## Capability Change Detection

The client is responsible for observing runtime capability changes and reporting them promptly.

Sources of change include:

- window resize, orientation change, fullscreen change, external display attach/detach
- keyboard, mouse, trackpad, gamepad, headset, USB mic, USB camera, Bluetooth device attach/detach
- audio route changes between speaker, headphones, HDMI, Bluetooth, etc.
- camera / microphone / Bluetooth / location permission changes
- browser or OS policy changes that disable previously usable APIs
- battery / charging / thermal / network state changes if they are modeled as capabilities or routing-relevant properties

When a change occurs, the client either:

- sends a targeted `CapabilityDelta`, or
- sends a fresh `CapabilitySnapshot` if recomputing the entire graph is simpler

The client does not try to preserve stale endpoint IDs across fundamentally different hardware enumerations. IDs must be stable when possible, but correctness matters more than identity continuity.

## Capability Runtime

Capability handling is its own subsystem, not a one-time registration helper.

```text
terminal_client/
├── lib/
│   ├── main.dart
│   ├── discovery/
│   │   ├── mdns_scanner.dart          # mDNS/Bonjour discovery
│   │   └── manual_connect.dart        # Manual server entry UI
│   ├── connection/
│   │   ├── grpc_channel.dart          # gRPC control channel
│   │   ├── session_controller.dart    # Hello / snapshot / delta sequencing
│   │   └── webrtc_manager.dart        # WebRTC media streams
│   ├── capabilities/
│   │   ├── capability_runtime.dart    # Owns current graph + revision counter
│   │   ├── capability_registry.dart   # Collects providers into one graph
│   │   ├── capability_diff.dart       # snapshot -> delta computation
│   │   ├── display_cap.dart
│   │   ├── audio_cap.dart
│   │   ├── camera_cap.dart
│   │   ├── input_cap.dart
│   │   ├── sensor_cap.dart
│   │   └── peripheral_cap.dart
│   ├── io/
│   │   ├── io_controller.dart         # Dispatches server commands to IO
│   │   ├── screen_renderer.dart       # Renders server-driven UI
│   │   ├── audio_streamer.dart        # Mic capture / speaker playback
│   │   ├── video_streamer.dart        # Camera capture / screen display
│   │   ├── keyboard_input.dart        # Keyboard event forwarding
│   │   ├── pointer_input.dart         # Mouse/trackpad forwarding
│   │   ├── touch_input.dart           # Touch event forwarding
│   │   └── sensor_streamer.dart       # Sensors and status streams
│   └── ui/
│       ├── server_driven_ui.dart      # Renders UI from server descriptors
│       └── fallback_ui.dart           # Connection/discovery UI only
```

`capability_runtime.dart` owns the authoritative local view of current capabilities. It:

- gathers provider output at startup
- subscribes to platform-specific change notifications
- coalesces bursts of changes
- increments the capability revision
- emits either snapshot or delta messages on the control stream
- waits for `CapabilityAck` before considering the new revision applied

## Display model

The client must treat display characteristics as live state, not constants captured at startup.

That includes at least:

- pixel size
- density / DPR
- rotation / orientation
- refresh rate if exposed
- safe insets / cutouts if relevant
- availability of each display endpoint

Server-driven UI and media compositor targets depend on these values being current.

## Audio model

The client must model inputs and outputs separately.

Examples:

- built-in mic + built-in speakers
- headset mic + headset speakers
- Bluetooth speaker with no microphone
- HDMI output only
- multiple simultaneous output routes on desktop-class platforms

The protocol must not collapse all of that into a single `microphone` field and a single `speaker` field.

## Platform support

| Platform | Build Target | Notes |
|-------------|------------------------|------------------------------------|
| Android | Flutter Android | Phones and tablets |
| iOS | Flutter iOS | Phones and tablets |
| Web/Browser | Flutter Web | Any device with a modern browser |
| macOS | Flutter macOS | Laptops and desktops |
| Linux | Flutter Linux | Chromebooks, PCs |
| Windows | Flutter Windows | Laptops and desktops |

Flutter's cross-platform support means a single codebase produces all client variants.

Platform-specific capability detection (for example USB attach/detach, permission changes, audio route changes, screen enumeration, browser device enumeration) uses platform channels or per-platform adapters where Flutter plugins do not cover it.

## Failure policy

The client prefers over-reporting change to under-reporting it.

If the platform makes precise deltas awkward, emit a fresh snapshot.
If endpoint IDs are unstable, emit a fresh snapshot.
If a route might have changed in a way that invalidates server assumptions, emit a fresh snapshot.

The server can handle redundant truth better than stale truth.

## Related Plans

- [protocol.md](protocol.md) — How the client talks to the server.
- [discovery.md](discovery.md) — How the client finds the server.
- [server-driven-ui.md](server-driven-ui.md) — The UI primitives the client renders.
- [io-abstraction.md](io-abstraction.md) — IO categories and routing semantics.
