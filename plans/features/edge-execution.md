# Edge Execution and Operator Runtime
See [masterplan.md](../masterplan.md) for overall system context. The server still owns behavior. This plan adds a **generic operator runtime** to capable clients so expensive perception work can run at the edge when appropriate.

## Design Principle
Scenario logic stays on the server. Clients do **not** gain scenario-specific behavior. Instead, a client may advertise that it can host generic operators chosen by the server.

The server remains responsible for:
- selecting targets
- deciding what to observe
- deciding when work runs on edge vs server
- fusing observations into one world view
- enforcing claims, priorities, and policy

The client becomes:
- a renderer for server-driven UI
- an IO bridge for capture and playback
- an optional host for portable operator kernels

## Capability Manifest Extension
The existing capability manifest reports physical hardware. Add an `edge` section for compute and retention.

```json
{
  "device_id": "uuid",
  "capabilities": {
    "screen": {"width": 1920, "height": 1080},
    "microphone": {"channels": 2, "sample_rates": [16000, 48000]},
    "camera": {"front": {"width": 1280, "height": 720, "fps": 30}},
    "bluetooth": {"version": "5.3"},
    "accelerometer": {"axes": 3},
    "edge": {
      "runtimes": ["wasm32-wasi-preview2", "onnx"],
      "compute": {
        "cpu_realtime": 2,
        "gpu_realtime": 1,
        "npu_realtime": 0,
        "mem_mb": 512
      },
      "operators": [
        "audio.onset",
        "audio.classify",
        "audio.localize",
        "vision.motion",
        "vision.object_detect",
        "vision.track",
        "sensor.anomaly",
        "radio.ble_scan"
      ],
      "retention": {
        "audio_sec": 20,
        "video_sec": 10,
        "sensor_sec": 600,
        "radio_sec": 300
      },
      "timing": {"sync_error_ms": 2.5},
      "geometry": {
        "mic_array": false,
        "camera_intrinsics": true,
        "compass": true
      }
    }
  }
}
```

The client still declares **what it physically is**. It does not declare semantic placement such as zone or role. See [placement.md](placement.md).

## Portable Operator Model
Edge work is described as a graph of generic operators. The same operator bundle may run on a client or on the server.

Operator classes:
- `source` — mic, camera, accelerometer, gyroscope, BLE scan, Wi-Fi scan
- `buffer` — rolling recent history for retrospective queries
- `feature` — spectrograms, embeddings, motion vectors, RSSI summaries
- `analyzer` — classifiers and anomaly detectors
- `tracker` — object, person, and motion trackers
- `localizer` — audio or radio location estimates
- `fusion` — merge many partial observations into one result
- `artifact` — clips, snapshots, timeseries excerpts
- `sink` — speaker, display, storage, event bus

Portable kernels are loaded from an app bundle and run inside a sandboxed operator host.

## Execution Policy
Every operator node declares an execution policy:

```go
type ExecPolicy string

const (
    ExecAuto         ExecPolicy = "auto"
    ExecPreferClient ExecPolicy = "prefer_client"
    ExecRequireClient ExecPolicy = "require_client"
    ExecServerOnly   ExecPolicy = "server_only"
)
```

Planner rules:
- `auto` chooses the least expensive safe location.
- `prefer_client` uses edge when the device advertises support and has budget.
- `require_client` fails placement if no suitable client host exists.
- `server_only` pins work to the server even if the client could run it.

## Resource Claims for Edge Compute
Compute and retention are claimable resources just like screen, speaker, and mic.

Add resource kinds:
- `compute.cpu.shared`
- `compute.gpu.shared`
- `compute.npu.shared`
- `buffer.audio.recent`
- `buffer.video.recent`
- `buffer.sensor.recent`
- `buffer.radio.recent`
- `radio.ble.scan`
- `radio.wifi.scan`

This prevents the server from overcommitting a phone or tablet by scheduling object tracking, sound classification, and BLE scanning simultaneously.

## Client Runtime Modules
Add a generic edge layer to the client.

```text
terminal_client/
└── lib/
    ├── edge/
    │   ├── host.dart              # operator runtime and lifecycle
    │   ├── bundle_store.dart      # installed kernels and models
    │   ├── scheduler.dart         # compute budgeting and admission
    │   ├── retention.dart         # rolling buffers for audio/video/sensors/radios
    │   ├── clock_sync.dart        # timestamp discipline for localization
    │   ├── artifact_export.dart   # clip/snapshot/timeseries export
    │   └── sandbox.dart           # runtime policy enforcement
    ├── io/
    │   ├── audio_streamer.dart
    │   ├── video_streamer.dart
    │   └── sensor_streamer.dart
    └── connection/
        └── grpc_channel.dart
```

Nothing in this module is scenario-specific. It is a generic execution substrate.

## Runtime Safety Model
The operator host must enforce:
- no arbitrary network access from kernels
- no filesystem access outside managed bundle and cache directories
- no subprocess creation
- explicit CPU, memory, and wall-time limits
- bounded output rates per flow
- signature or hash validation for installed bundles

The server may revoke a flow at any time if a device overheats, disconnects, or loses capability.

## Bundle Lifecycle
The control plane grows a small edge lifecycle:
- `InstallBundle`
- `RemoveBundle`
- `StartFlow`
- `PatchFlow`
- `StopFlow`
- `RequestArtifact`
- `FlowStats`
- `ClockSample`

These are **generic** transport messages. They do not encode application meaning. Application meaning still lives in server-side TAL code.

## Scheduling Heuristics
The planner should prefer edge execution when all of the following hold:
- the client advertises the required runtime and operator support
- the data source is local to that client
- the output is compact relative to the raw stream
- the client has sufficient compute and retention budget
- latency or privacy improve materially

The planner should prefer server execution when:
- many-device fusion is required
- the result depends on house-wide context
- the client lacks calibrated geometry or clock quality
- the operator would interfere with foreground user experience

## Failure Behavior
If an edge flow fails:
- the client emits `FlowStats` with failure details
- the planner may re-place the same operator on the server
- any dependent application sees a typed degraded-mode event
- persistent activations keep running unless the failure invalidates the scenario

## Related Plans
- [architecture-client.md](architecture-client.md) — Base client architecture the edge host extends.
- [observation-plane.md](observation-plane.md) — `FlowPlan` structure and observation transport.
- [io-abstraction.md](io-abstraction.md) — Existing resource and claim concepts this plan extends.
- [world-model-calibration.md](world-model-calibration.md) — Geometry and clock-quality data used by edge placement.
- [application-runtime.md](application-runtime.md) — Server-side app model that requests observations.
