# IO Abstraction Layer

See [masterplan.md](../masterplan.md) for overall system context.

Every IO capability maps to a uniform interface on both client and server.

## IO Categories

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

## IO Routing

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

## Router Responsibilities

The IO Router is the only component that knows the runtime topology of streams. Scenarios request routing operations in terms of source/destination devices and stream kinds; the router translates these into `StartStream`, `StopStream`, and `RouteStream` messages plus the WebRTC signaling needed to realize the topology.

## Related Plans

- [protocol.md](protocol.md) — Stream control messages on the control plane, media on the media plane.
- [architecture-server.md](architecture-server.md) — Server-side router module (`internal/io/`).
- [scenario-engine.md](scenario-engine.md) — Scenarios drive routing through the router.
