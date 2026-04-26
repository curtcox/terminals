---
title: "IO Abstraction Layer"
kind: plan
status: building
owner: copilot
validation: none
last-reviewed: 2026-04-26
---

# IO Abstraction Layer

See [masterplan.md](../archive/masterplan-duplicate.md) for overall system context.

Every IO capability maps to a uniform interface on both client and server.

## Progress (2026-04-26)

- Implemented endpoint-scoped resource compilation in `terminal_server/internal/transport/control_stream.go`.
    Capability snapshots now compile concrete endpoint resources alongside legacy aliases:
    - `display.<id>.main`, `display.<id>.overlay`
    - `audio_out.<id>`
    - `audio_in.<id>.capture`, `audio_in.<id>.analyze`
    - `camera.<id>.capture`, `camera.<id>.analyze`
- Added transport tests in `terminal_server/internal/transport/control_stream_test.go` for endpoint resource derivation and endpoint-claim invalidation on capability loss.
- Remaining work: wire endpoint-scoped resources end-to-end through scenario claim recipes and planner APIs, then promote this plan from `building` to shipped states.

## IO Categories

| Category | Inputs (client → server) | Outputs (server → client) |
|---------------|----------------------------------|-----------------------------------|
| **Display** | display state / resize / availability | UI descriptors, images, video |
| **Keyboard** | key events (down, up, char) | — |
| **Pointer** | move, click, scroll, hover | cursor changes |
| **Touch** | touch start/move/end, gestures | — |
| **Audio Input** | mic PCM/Opus stream | — |
| **Audio Output** | — | speaker PCM/Opus stream, clips |
| **Video Input** | camera H.264/VP8 stream | — |
| **Video Output** | — | display H.264/VP8 stream |
| **Bluetooth / Radios** | scan results, device data, route state | scan commands, connect commands |
| **Sensors** | accelerometer, gyro, compass, location, battery, ambient state | — |
| **USB / Peripherals** | device enumeration, device data | data, commands |
| **Haptic** | — | vibration patterns |

## Resources

A device exposes more than a bag of capabilities — it exposes **resources** that can be claimed independently.

The resource kind is the vocabulary scenarios use when they talk about what part of a device they need.

| Resource Kind | Description | Typical mode |
|-----------------------------|--------------------------------------------------------|--------------|
| `display.<id>.main` | Fullscreen UI layer on one display endpoint | exclusive |
| `display.<id>.overlay` | Toast / overlay layer above one display | shared |
| `audio_out.<id>` | One concrete speaker / output route | exclusive |
| `audio_in.<id>.capture` | One concrete microphone capture endpoint | exclusive |
| `audio_in.<id>.analyze` | Shared tap on one microphone endpoint | shared |
| `camera.<id>.capture` | Live camera capture routed elsewhere | exclusive |
| `camera.<id>.analyze` | Shared tap for vision models | shared |
| `keyboard.<id>` | Keyboard input focus | shared |
| `pointer.<id>` | Pointer input focus | shared |
| `touch.<id>` | Touch surface input focus | shared |
| `terminal.pty` | PTY session on this device | exclusive |
| `sensor.<id>` | One sensor endpoint | shared |
| `haptic.<id>` | One vibration / haptic endpoint | exclusive |

The conceptual kind is open-ended, but claims target **concrete compiled resources**, not vague device-level buckets.

A laptop with an internal display plus an HDMI projector is therefore not “one screen capability”; it is two display endpoints and at least four claimable display-layer resources.

## Resource compilation

Capabilities are client-declared facts. Resources are server-compiled runtime objects.

Compilation rules turn endpoint records into claimable resources.

Examples:

- `Display{id: "display-internal"}` → `display.display-internal.main`, `display.display-internal.overlay`
- `AudioOutput{id: "speaker-internal"}` → `audio_out.speaker-internal`
- `AudioInput{id: "mic-usb"}` → `audio_in.mic-usb.capture`, `audio_in.mic-usb.analyze`
- `Camera{id: "cam-front"}` → `camera.cam-front.capture`, `camera.cam-front.analyze`

This keeps the client protocol about reality and the server runtime about scheduling.

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
    Granted   []Claim
    Preempted []Claim // lower-priority claims that were suspended
}

type ClaimManager interface {
    Request(ctx context.Context, claims []Claim) (Grant, error)
    Release(ctx context.Context, activationID string) error
    Snapshot(deviceID string) []Claim
    ReconcileCapabilities(ctx context.Context, deviceID string, resources []ResourceKind) error
}
```

The claim manager:

- grants non-conflicting claims immediately
- on conflict, compares priorities: higher wins; the lower-priority activation is **suspended** (not terminated) with its claims parked
- restores parked claims when the preemptor releases them, waking the suspended activation via `Resume`
- treats shared tap kinds as non-conflicting with each other; an exclusive claim on the underlying endpoint still preempts them
- invalidates claims whose backing resource disappeared after a capability change

The scenario engine reads the claim list from each granted activation and drives suspend / resume hooks accordingly. See [scenario-engine.md](scenario-engine.md#resource-claims-and-preemption).

## Capability-driven invalidation

Capability changes can invalidate active resources.

Examples:

- display resized: resource survives, but layout / compositor targets must be patched
- display removed: all claims on that display endpoint are invalidated
- headset unplugged: claims on that output route are invalidated or migrated by policy
- camera permission revoked: camera claims are invalidated immediately
- Bluetooth speaker added: no current claim is invalid, but placement candidates change

The router and claim manager must react to capability changes as first-class events.

## Media Topology: Plans, not Connects

The conceptual router operations are:

- **consume** a stream: mic → STT
- **produce** a stream: TTS → speaker
- **forward**: Client A mic → Client B speaker
- **fork**: mic → STT + recording + remote speaker
- **mix**: many mics → one mixed output
- **composite**: many videos → one display
- **record**: any stream → disk
- **analyze**: audio → classifier, video → vision

Scenarios do not call these as individual primitives.
They declare a **media plan** — a small topology graph — and hand it to the router, which compiles it into concrete stream start/stop, transport routing, and WebRTC signaling.

```go
type MediaNodeKind string

const (
    NodeSourceAudioIn   MediaNodeKind = "source.audio_in"
    NodeSourceCamera    MediaNodeKind = "source.camera"
    NodeSinkAudioOut    MediaNodeKind = "sink.audio_out"
    NodeSinkDisplay     MediaNodeKind = "sink.display"
    NodeMixer           MediaNodeKind = "mixer"
    NodeCompositor      MediaNodeKind = "compositor"
    NodeAnalyzer        MediaNodeKind = "analyzer"
    NodeRecorder        MediaNodeKind = "recorder"
    NodeSTT             MediaNodeKind = "stt"
    NodeTTS             MediaNodeKind = "tts"
    NodeFork            MediaNodeKind = "fork"
)

type MediaNode struct {
    ID   string
    Kind MediaNodeKind
    Args map[string]any // resource refs, format hints, model names, etc.
}

type MediaEdge struct {
    From string // source node ID
    To   string // sink node ID
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

- **Intercom**: `audio_in(A) → audio_out(B)`, `audio_in(B) → audio_out(A)`
- **PA**: `audio_in(source) → fork → audio_out[*]`
- **Voice assistant**: `audio_in → fork → [STT, recorder]`; `TTS → audio_out`
- **Multi-camera grid**: `camera[*] → compositor → display`; `audio_in[*] → mixer → audio_out`
- **Monitoring**: `audio_in → analyzer(sound_classifier) → event bus`

Because the full topology is known up front, the router handles teardown, reconnection on roaming, and observability uniformly. Stream-kind magic strings disappear — the plan's node/edge shape is the semantic.

## Router Responsibilities

The IO Router is the only component that knows the runtime topology of streams. It:

- accepts media plans, returns a handle, and compiles to concrete `StartStream` / `StopStream` / `RouteStream` messages plus WebRTC signaling
- maintains the live graph so later patches (add a node, swap a source, change a mix, move an output) are cheap
- reacts to capability changes by patching or tearing plans whose resource set changed
- emits events onto the intent/event bus when analyzer nodes fire (`Event{Kind: "sound.detected", ...}`)
- cleans up deterministically when a plan handle is torn down

## Related Plans

- [protocol.md](protocol.md) — transport-level handshake, capability updates, and WebRTC signaling.
- [architecture-server.md](architecture-server.md) — router, claim manager, and media-planner module layout.
- [scenario-engine.md](scenario-engine.md) — how activations use claims and media plans.
- [placement.md](placement.md) — resolving device targets before building plans.
