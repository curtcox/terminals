# Client Architecture

See [masterplan.md](../masterplan.md) for overall system context.

## Design Principle

The client is a **generic IO terminal**. It has no scenario-specific logic. It:

1. Discovers the server (or accepts a manually specified address)
2. Registers itself and declares its capabilities
3. Receives commands and executes them
4. Streams IO data as directed

## Capability Declaration

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

## Client Module Structure

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

## Platform Support

| Platform    | Build Target           | Notes                              |
|-------------|------------------------|------------------------------------|
| Android     | Flutter Android        | Phones and tablets                 |
| iOS         | Flutter iOS            | Phones and tablets                 |
| Web/Browser | Flutter Web            | Any device with a modern browser   |
| macOS       | Flutter macOS          | Laptops                            |
| Linux       | Flutter Linux          | Chromebooks (Linux container), PCs |
| Windows     | Flutter Windows        | Laptops, desktops                  |

Flutter's cross-platform support means a single codebase produces all client variants. Platform-specific capability detection (e.g., accelerometer on mobile, USB on desktop) uses platform channels where Flutter plugins don't cover it.

## Related Plans

- [protocol.md](protocol.md) — How the client talks to the server.
- [discovery.md](discovery.md) — How the client finds the server.
- [server-driven-ui.md](server-driven-ui.md) — The UI primitives the client renders.
- [io-abstraction.md](io-abstraction.md) — IO categories and routing semantics.
