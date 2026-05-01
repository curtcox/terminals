---
title: "IO Abstraction Layer — Progress Log"
kind: progress-log
parent: plans/features/io-abstraction/plan.md
---

## Implementation Progress (2026-04-26)

Shipped and validated via repository gates (`make all-check`) after endpoint-scoped
claim/planner wiring and regression coverage updates.

- Implemented endpoint-scoped resource compilation in `terminal_server/internal/transport/control_stream.go`.
    Capability snapshots now compile concrete endpoint resources alongside legacy aliases:
    - `display.<id>.main`, `display.<id>.overlay`
    - `audio_out.<id>`
    - `audio_in.<id>.capture`, `audio_in.<id>.analyze`
    - `camera.<id>.capture`, `camera.<id>.analyze`
- Added transport tests in `terminal_server/internal/transport/control_stream_test.go` for endpoint resource derivation and endpoint-claim invalidation on capability loss.
- Wired endpoint-scoped resources through scenario claim recipes in `terminal_server/internal/scenario/builtin.go`:
    - PA + announcement now claim endpoint resources (`audio_in.<id>.capture`, `audio_out.<id>`) with legacy fallback aliases when endpoint metadata is absent.
    - Voice assistant and audio monitor now claim endpoint analyze/output/display resources where available.
    - Media-plan nodes now carry optional `resource` args to preserve resolved endpoint intent.
- Updated media planner compilation in `terminal_server/internal/io/media_plan.go` to infer stream kind from endpoint-scoped node resources when supplied (`audio_in.*.capture -> audio_out.*`, `camera.*.capture -> display.*.main`).
- Added regression coverage:
    - `terminal_server/internal/scenario/runtime_test.go` validates PA claims endpoint-scoped resources from capability snapshots.
    - `terminal_server/internal/io/media_plan_test.go` validates resource-arg stream-kind inference.

