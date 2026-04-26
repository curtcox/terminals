# Observation Plane and Flow Model

This document is the durable reference for generalized observation flows.
It captures the contract implemented by protobuf, transport adapters, and
server flow planning.

## Scope

The observation plane generalizes media routing into one flow model so audio,
video, sensors, and radio sightings share one topology vocabulary.

- The control plane is protobuf over gRPC/websocket.
- The media plane is still WebRTC when raw live streams are required.
- Typed observations and lazy artifacts are preferred over always-on raw uplink.

## Core Model

The flow graph is represented by `FlowPlan`:

- `FlowNode`: graph node (`id`, `kind`, `args`, `exec`)
- `FlowEdge`: directed connection (`from`, `to`)
- `FlowPlan`: `nodes[]` + `edges[]`

Canonical protobuf definitions live in:

- `api/terminals/io/v1/io.proto`
- `api/terminals/control/v1/control.proto` (wire envelope integration)

Server runtime types and planner behavior live in:

- `terminal_server/internal/io/media_plan.go`

## Observation and Artifact Records

Edge operators emit compact typed `Observation` records, optionally with
`ArtifactRef` evidence.

Observation includes:

- classification/track identity (`kind`, `subject`, `track_id`)
- source and timing (`source_device`, `occurred_unix_ms`)
- confidence and spatial context (`confidence`, `zone`, `location`)
- attribution (`provenance.flow_id`, `provenance.node_id`, model/calibration)

Artifacts are referenced first, materialized on demand:

1. Device emits `ObservationMessage` and/or `ArtifactAvailable`.
2. Server requests evidence via `RequestArtifact` by id.
3. Device exports the exact clip/frame/excerpt.

This keeps baseline operation efficient while preserving debuggable evidence.

## Control Messages

Flow lifecycle:

- `StartFlow`
- `PatchFlow`
- `StopFlow`

Observation and artifact exchange:

- `ObservationMessage`
- `ArtifactAvailable`
- `RequestArtifact`

Operational telemetry:

- `FlowStats`
- `ClockSample`

## Implementation and Validation References

- Proto adapter mapping:
  `terminal_server/internal/transport/generated_proto_adapter.go`
- Planner routing and observation sink:
  `terminal_server/internal/io/media_plan.go`
- Adapter tests (mapping coverage):
  `terminal_server/internal/transport/generated_proto_adapter_test.go`
- Planner tests (flow apply/tear + analyzer path):
  `terminal_server/internal/io/media_plan_test.go`

Related scenario flow guidance:

- `docs/sensing-use-case-flows.md`
