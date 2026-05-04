---
title: "Compatibility"
kind: reference
status: active
owner: curtcox
last-reviewed: 2026-05-04
---

# Compatibility

This document tracks protocol compatibility windows and migration notes.

Current compatibility policy:

- Protocol migrations are additive first.
- Producers emit both typed and legacy fields during a migration window.
- Consumers prefer typed fields and fall back to legacy fields while old clients or servers are supported.
- Deprecated fields remain decodable until their documented removal criteria are met.
- Flexible fields are governed by [protocol-evolution.md](protocol-evolution.md) and [protocol-extension-registry.md](protocol-extension-registry.md).

## Open Windows

The following additive typed replacements have shipped. Producers emit both the typed field and the legacy field; consumers prefer the typed field and fall back to the legacy field. Legacy fields remain decodable for at least two tagged releases after the typed replacement shipped (the suggested default in `plans/features/protocol/evolution-rules.md`).

| Typed replacement | Legacy field | Shipped | Earliest legacy removal |
|---|---|---|---|
| `RegisterAck.server_metadata` (`ServerMetadata` / `BuildMetadata`) | `RegisterAck.metadata` map keys (`server_build_sha`, `server_build_date`, `photo_frame_asset_base_url`) | 2026-05-03 | After two tagged releases past 2026-05-03 |
| `StartStream.stream_kind` (`StreamKind`) | `StartStream.kind` string | 2026-05-03 | After two tagged releases past 2026-05-03 |
| `RouteStream.stream_kind` (`StreamKind`) | `RouteStream.kind` string | 2026-05-03 | After two tagged releases past 2026-05-03 |
| `WebRTCSignal.signal_type_enum` (`WebRTCSignalType`) | `WebRTCSignal.signal_type` string | 2026-05-03 | After two tagged releases past 2026-05-03 |
| `ScrollWidget.direction_enum` (`ScrollDirection`) | `ScrollWidget.direction` string | 2026-05-03 | After two tagged releases past 2026-05-03 |
| `FlowStats.state_enum` (`FlowState`) | `FlowStats.state` string | 2026-05-03 | After two tagged releases past 2026-05-03 |
| `FlowNode.exec_policy` (`ExecPolicy`) | `FlowNode.exec` string | 2026-05-03 | After two tagged releases past 2026-05-03 |
| `CanvasWidget.draw_ops` (`repeated DrawOp`) | `CanvasWidget.draw_ops_json` string | 2026-05-04 (schema only; producer/consumer wiring deferred) | After typed wiring lands plus two tagged releases |
| `PointerEvent.action_enum` (`PointerAction`) | `PointerEvent.action` string | 2026-05-04 (schema only; producer/consumer wiring deferred) | After typed wiring lands plus two tagged releases |
| `TouchEvent.action_enum` (`TouchAction`) | `TouchEvent.action` string | 2026-05-04 (schema only; producer/consumer wiring deferred) | After typed wiring lands plus two tagged releases |

## Pending Migrations

The protocol extension registry identifies remaining transitional escape hatches that should be reviewed after 2026-06-15, including `RegisterAck.metadata` (post-typed cleanup), `CommandRequest.arguments`, `CommandResult.data`, `Node.props`, `StartStream.metadata`, `FlowNode.args`, and `Observation.attributes`.
