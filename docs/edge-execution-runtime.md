# Edge Execution and Operator Runtime

This document is the durable reference for generic edge operator execution in
Terminals. Scenario behavior remains server-owned; clients expose a generic
runtime surface that the server can target.

## Scope

- Server-owned planning and orchestration.
- Client-hosted generic operator lifecycle.
- Typed control-plane messages for bundles, flows, observations, and artifacts.

## Capability Contract

The device capability contract includes an `edge` section in
`DeviceCapabilities`.

- Proto: `api/terminals/capabilities/v1/capabilities.proto`
- Message: `EdgeCapability`
- Subfields:
  - `runtimes[]`
  - `compute` (`cpu_realtime`, `gpu_realtime`, `npu_realtime`, `mem_mb`)
  - `operators[]`
  - `retention` (`audio_sec`, `video_sec`, `sensor_sec`, `radio_sec`)
  - `timing` (`sync_error_ms`)
  - `geometry` (`mic_array`, `camera_intrinsics`, `compass`)

Capability flattening and tier notes are documented in `docs/server.md` under
"Monitoring Support Tiers".

## Flow and Bundle Lifecycle

The control-plane lifecycle for edge execution is carried by protobuf and
transport adapters.

- Lifecycle control messages:
  - `InstallBundle`
  - `RemoveBundle`
  - `StartFlow`
  - `PatchFlow`
  - `StopFlow`
  - `RequestArtifact`
- Telemetry messages:
  - `FlowStats`
  - `ClockSample`

Canonical proto definitions:

- `api/terminals/io/v1/io.proto`
- `api/terminals/control/v1/control.proto`

## Server Implementation

Flow planning and operator placement vocabulary are implemented in:

- `terminal_server/internal/io/media_plan.go`

Key server-side contracts include:

- `FlowNodeKind` classes (`source`, `buffer`, `feature`, `analyzer`,
  `tracker`, `localizer`, `fusion`, `artifact`, `sink`, etc.)
- `ExecPolicy` values:
  - `auto`
  - `prefer_client`
  - `require_client`
  - `server_only`

Proto-to-internal mapping coverage for lifecycle and telemetry messages lives
in:

- `terminal_server/internal/transport/generated_proto_adapter.go`
- `terminal_server/internal/transport/generated_proto_adapter_test.go`

## Client Runtime Implementation

The generic edge runtime is implemented under:

- `terminal_client/lib/edge/`

Primary modules:

- `host.dart`: bundle/flow lifecycle (`installBundle`, `removeBundle`,
  `startFlow`, `patchFlow`, `stopFlow`) with durable host-state hydration.
- `bundle_store.dart`: installed bundle persistence.
- `scheduler.dart`: admission control by CPU and memory budget.
- `retention.dart`: rolling retention windows for audio/video/sensor/radio
  samples.
- `clock_sync.dart`: coarse timing error state.
- `artifact_export.dart`: artifact export/materialization.
- `sandbox.dart`: runtime policy flags for network/subprocess/filesystem writes.

Control-stream wiring in the Flutter client handles lifecycle responses and
artifact fetch requests in:

- `terminal_client/lib/main.dart`

## Validation References

- Client edge storage/runtime tests:
  - `terminal_client/test/edge_storage_io_test.dart`
  - `terminal_client/test/retention_buffer_test.dart`
- Server flow/proto adapter tests:
  - `terminal_server/internal/io/media_plan_test.go`
  - `terminal_server/internal/transport/generated_proto_adapter_test.go`

## Related Documentation

- `docs/observation-plane.md`
- `docs/sensing-use-case-flows.md`
- `docs/server.md`
