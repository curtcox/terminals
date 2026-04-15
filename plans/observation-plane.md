# Observation Plane and Flow Plans
See [masterplan.md](../masterplan.md) for overall system context. This plan generalizes the existing media-routing model so audio, video, sensors, and radio observations all use one vocabulary.

## From MediaPlan to FlowPlan
The current [IO abstraction](io-abstraction.md) defines a `MediaPlan` for stream graphs. Keep that shape, but broaden it into a **FlowPlan** so observations, artifacts, and retrospective buffers are first-class.

A `MediaPlan` becomes a specialized `FlowPlan` focused on live audio/video routing.

```go
type FlowNodeKind string

const (
    NodeSourceMic          FlowNodeKind = "source.mic"
    NodeSourceCamera       FlowNodeKind = "source.camera"
    NodeSourceSensor       FlowNodeKind = "source.sensor"
    NodeSourceBluetooth    FlowNodeKind = "source.bluetooth"
    NodeSourceWiFi         FlowNodeKind = "source.wifi"
    NodeBufferRecent       FlowNodeKind = "buffer.recent"
    NodeFeature            FlowNodeKind = "feature"
    NodeAnalyzer           FlowNodeKind = "analyzer"
    NodeTracker            FlowNodeKind = "tracker"
    NodeLocalizer          FlowNodeKind = "localizer"
    NodeFusion             FlowNodeKind = "fusion"
    NodeMixer              FlowNodeKind = "mixer"
    NodeCompositor         FlowNodeKind = "compositor"
    NodeRecorder           FlowNodeKind = "recorder"
    NodeArtifact           FlowNodeKind = "artifact"
    NodeSinkSpeaker        FlowNodeKind = "sink.speaker"
    NodeSinkDisplay        FlowNodeKind = "sink.display"
    NodeSinkStore          FlowNodeKind = "sink.store"
    NodeSinkEventBus       FlowNodeKind = "sink.event_bus"
)

type FlowNode struct {
    ID    string
    Kind  FlowNodeKind
    Args  map[string]any
    Exec  ExecPolicy
}

type FlowEdge struct {
    From string
    To   string
}

type FlowPlan struct {
    Nodes []FlowNode
    Edges []FlowEdge
}
```

## Observation-First Transport
The transport keeps two familiar layers:
- **gRPC control plane** for commands, lifecycle, and typed observations
- **WebRTC media plane** for live audio/video when raw streams are actually needed

The new rule is:
- raw continuous media flows only when a scenario genuinely needs raw media
- compact observations flow by default
- evidence artifacts are pulled on demand

This prevents constant uplink of audio/video that is only needed for classification, tracking, or anomaly detection.

## Observation Record
Every analyzer, tracker, or localizer emits a typed observation.

```go
type Observation struct {
    Kind        string
    Subject     string
    SourceDevice DeviceRef
    OccurredAt  time.Time
    Confidence  float64
    Zone        string
    Location    *LocationEstimate
    TrackID     string
    Attributes  map[string]any
    Evidence    []ArtifactRef
    Provenance  ObservationProvenance
}

type ObservationProvenance struct {
    FlowID             string
    NodeID             string
    ExecSite           string // client:<device> or server
    ModelID            string
    CalibrationVersion string
}

type ArtifactRef struct {
    ID        string
    Kind      string // audio_clip, image_frame, video_clip, imu_excerpt, radio_excerpt
    Source    DeviceRef
    StartTime time.Time
    EndTime   time.Time
    URI       string
}
```

The scenario engine may project an observation into the existing `Event` bus shape, but the richer record is preserved for auditing, debugging, and UI.

## Retrospective Buffers
A flow may keep a short rolling buffer on the client or server.

Supported buffer classes:
- audio
- video
- sensor timeseries
- radio sightings

The buffer is claimable and bounded. It is not an implicit always-on recorder.

Example uses:
- "Did you feel that just now?"
- "What was that sound?"
- "Show me the frame where the package disappeared."
- "What BLE devices were present five minutes ago?"

## Artifact Pull Model
Artifacts are created lazily.

Flow behavior:
1. edge or server operator emits an `Observation` with an `ArtifactRef`
2. the artifact may initially be metadata only
3. the server requests materialization via `RequestArtifact`
4. the host exports the exact clip, frame, or timeseries excerpt
5. storage assigns retention policy and access scope

This keeps always-on observation cheap while still preserving evidence when needed.

## Observation Plane Messages
Add typed control-plane messages:

```protobuf
message StartFlow { FlowPlan plan = 1; }
message PatchFlow { string flow_id = 1; FlowPlan plan = 2; }
message StopFlow { string flow_id = 1; }
message ObservationMessage { Observation observation = 1; }
message ArtifactAvailable { ArtifactRef artifact = 1; }
message RequestArtifact { string artifact_id = 1; }
message FlowStats {
  string flow_id = 1;
  double cpu_pct = 2;
  double mem_mb = 3;
  uint64 dropped_frames = 4;
  string state = 5;
}
```

WebRTC DataChannels may carry burstier feature streams or low-latency timing packets, but the type system stays anchored in protobuf.

## Examples
### Sound Identification
`mic -> buffer.recent -> analyzer(sound_classifier) -> sink.event_bus`

### Sound Localization
`mic[*] -> feature(onset/timestamp) -> localizer -> fusion -> sink.event_bus`

### Recent IMU Anomaly
`accelerometer -> buffer.recent -> analyzer(imu_anomaly) -> sink.store + sink.event_bus`

### Object Tracking
`camera -> tracker(object_tracker) -> artifact(snapshot) -> sink.event_bus`

### Bluetooth Inventory
`bluetooth.scan -> feature(rssi_summary) -> fusion -> sink.store`

## Router Responsibilities
The IO Router becomes a **Flow Planner**. It:
- applies and tears down flow graphs
- places nodes on server or clients
- owns flow handles and stats
- emits typed observations
- resolves artifact requests
- reconnects flows after client roam or disconnect when possible

Media routing remains a core responsibility, but it is now one specialization of the wider observation model.

## Related Plans
- [io-abstraction.md](io-abstraction.md) — Existing `MediaPlan` abstraction generalized by this doc.
- [edge-execution.md](edge-execution.md) — Client operator hosts and execution policy.
- [protocol.md](protocol.md) — Base transport this plan extends.
- [scenario-engine.md](scenario-engine.md) — Observation projection onto triggers and events.
- [sensing-use-case-flows.md](sensing-use-case-flows.md) — Concrete flows built from these primitives.
