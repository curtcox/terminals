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

## Resources

A device exposes more than a bag of capabilities — it exposes **resources** that can be claimed independently. The resource kind is the vocabulary scenarios use when they talk about what part of a device they need:

| Resource Kind       | Description                                    | Typical mode |
|---------------------|------------------------------------------------|--------------|
| `screen.main`       | Fullscreen UI layer                            | exclusive    |
| `screen.overlay`    | Toast / overlay layer above main UI            | shared       |
| `speaker.main`      | Audible output                                 | exclusive    |
| `mic.capture`       | Live mic capture routed elsewhere              | exclusive    |
| `mic.analyze`       | Mic tap for STT, classifiers                   | shared       |
| `camera.capture`    | Live camera capture routed elsewhere           | exclusive    |
| `camera.analyze`    | Camera tap for vision models                   | shared       |
| `terminal.pty`      | PTY session on this device                     | exclusive    |
| `sensor.*`          | Accelerometer, GPS, etc.                       | shared       |
| `haptic`            | Vibration                                      | exclusive    |

The list is an open enum — new kinds are added here as the system gains new IO modalities. Resource kinds are the atoms of preemption.

## Resource Claims

Preemption happens **per-resource**, not per-device. The claim manager is the arbiter.

```go
type ClaimMode string

const (
    ClaimExclusive ClaimMode = "exclusive"
    ClaimShared    ClaimMode = "shared"
)

type Claim struct {
    ActivationID string
    DeviceID     string
    Resource     ResourceKind
    Mode         ClaimMode
    Priority     int
}

type Grant struct {
    Granted    []Claim
    Preempted  []Claim     // lower-priority claims that were suspended
}

type ClaimManager interface {
    Request(ctx context.Context, claims []Claim) (Grant, error)
    Release(ctx context.Context, activationID string) error
    Snapshot(deviceID string) []Claim
}
```

The claim manager:

- Grants non-conflicting claims immediately.
- On conflict, compares priorities: higher wins; the lower-priority activation is **suspended** (not terminated) with its claims parked.
- Restores parked claims when the preemptor releases them, waking the suspended activation via `Resume`.
- Treats `ClaimShared` kinds (mic taps, camera taps, sensor streams) as non-conflicting with each other; an exclusive claim on the underlying resource still preempts them.

The scenario engine reads the claim list from each granted activation and drives suspend/resume hooks accordingly. See [scenario-engine.md](scenario-engine.md#resource-claims-and-preemption).

## Media Topology: Plans, not Connects

The conceptual router operations are:

- **Consume** a stream: mic → STT
- **Produce** a stream: TTS → speaker
- **Forward**: Client A mic → Client B speaker
- **Fork**: mic → STT + recording + remote speaker
- **Mix**: many mics → one mixed output
- **Composite**: many videos → grid on one screen
- **Record**: any stream → disk
- **Analyze**: audio → classifier, video → vision

Scenarios do not call these as individual primitives. They declare a **media plan** — a small topology graph — and hand it to the router, which compiles it into concrete stream start/stop, transport routing, and WebRTC signaling.

```go
type MediaNodeKind string

const (
    NodeSourceMic    MediaNodeKind = "source.mic"
    NodeSourceCamera MediaNodeKind = "source.camera"
    NodeSinkSpeaker  MediaNodeKind = "sink.speaker"
    NodeSinkDisplay  MediaNodeKind = "sink.display"
    NodeMixer        MediaNodeKind = "mixer"
    NodeCompositor   MediaNodeKind = "compositor"
    NodeAnalyzer     MediaNodeKind = "analyzer"
    NodeRecorder     MediaNodeKind = "recorder"
    NodeSTT          MediaNodeKind = "stt"
    NodeTTS          MediaNodeKind = "tts"
    NodeFork         MediaNodeKind = "fork"
)

type MediaNode struct {
    ID   string
    Kind MediaNodeKind
    Args map[string]any     // device refs, format hints, model names, etc.
}

type MediaEdge struct {
    From string             // source node ID
    To   string             // sink node ID
}

type MediaPlan struct {
    Nodes []MediaNode
    Edges []MediaEdge
}

type PlanHandle string

type MediaPlanner interface {
    Apply(ctx context.Context, plan MediaPlan) (PlanHandle, error)
    Patch(ctx context.Context, h PlanHandle, plan MediaPlan) error
    Tear(ctx context.Context, h PlanHandle) error
}
```

Examples (elided node IDs for readability):

- **Intercom**: `mic(A) → speaker(B)`, `mic(B) → speaker(A)`.
- **PA**: `mic(source) → fork → speakers[*]`.
- **Voice assistant**: `mic → fork → [STT, recorder]`; `TTS → speaker`.
- **Multi-camera grid**: `cameras[*] → compositor → display`; `mics[*] → mixer → speaker`.
- **Monitoring**: `mic → analyzer(sound_classifier) → event bus`.

Because the full topology is known up front, the router handles teardown, reconnection on roaming, and observability uniformly. Stream-kind magic strings (`audio_mix`, `pa_audio`) disappear — the plan's node/edge shape is the semantic.

## Router Responsibilities

The IO Router is the only component that knows the runtime topology of streams. It:

- Accepts media plans, returns a handle, and compiles to concrete `StartStream` / `StopStream` / `RouteStream` messages plus WebRTC signaling.
- Maintains the live graph so later patches (add a node, swap a source, change a mix) are cheap.
- Emits events onto the intent/event bus when analyzer nodes fire (`Event{Kind: "sound.detected", ...}`). See [scenario-engine.md](scenario-engine.md#triggers-intents-and-events).
- Cleans up deterministically when a plan handle is torn down.

## Related Plans

- [protocol.md](protocol.md) — Transport-level stream control and WebRTC signaling the planner compiles to.
- [architecture-server.md](architecture-server.md) — Router, claim manager, and media-planner module layout.
- [scenario-engine.md](scenario-engine.md) — How activations use claims and media plans.
- [placement.md](placement.md) — Resolving device targets before building plans.
