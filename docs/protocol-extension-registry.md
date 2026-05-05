---
title: "Protocol Extension Registry"
kind: reference
status: active
owner: curtcox
last-reviewed: 2026-05-04
---

# Protocol Extension Registry

This registry covers current flexible protobuf fields under `api/terminals/**/v1/*.proto`. Every entry includes the governance facts reviewers need before a flexible field becomes a durable client/server dependency.

Unknown keys and values must follow the documented behavior for the field. New durable keys, tokens, or JSON shapes require a registry update and contract tests.

## Control

### Field: terminals.control.v1.TransportHelloAck.limits

Owner: transport/control  
Classification: registry_backed_extension  
Target state: keep as a map for negotiated transport limits.
Producer: server  
Consumer: client connection layer  
Unknown behavior: clients preserve or ignore unknown keys without changing carrier negotiation.  
Validation: values are decimal strings unless a key documents a stricter format. Invalid known values are ignored and treated as unset.  
Tests: Go transport round-trip coverage; add Dart unknown-key coverage with golden fixtures.  
Promotion trigger: introduce typed fields when a limit affects core connection behavior across two client releases.

Allowed keys:

- `max_frame_bytes`: positive decimal byte count
- `max_inflight_messages`: positive decimal count
- `heartbeat_interval_ms`: positive decimal milliseconds when emitted as a limit

### Field: terminals.control.v1.RegisterAck.metadata

Owner: transport/control  
Classification: transitional_escape_hatch  
Target state: typed `ServerMetadata` is now emitted; keep legacy map during compatibility window.  
Review date: 2026-06-15  
Producer: server  
Consumer: client diagnostics and media asset setup  
Unknown behavior: clients ignore unknown keys.  
Validation: known keys use the formats below; unknown keys are advisory only.  
Tests: Go server emits registered keys plus typed metadata; client prefers typed `server_metadata.build` fields and falls back to map keys for older servers.

Allowed keys:

- `photo_frame_asset_base_url`: absolute HTTP(S) URI
- `server_build_sha`: non-empty short git SHA string; `unknown` allowed for local builds
- `server_build_date`: RFC3339 timestamp; `unknown` allowed for local builds

### Field: terminals.control.v1.CommandRequest.action

Owner: transport/control  
Classification: typed_contract  
Target state: keep `CommandAction` enum.
Producer: client  
Consumer: server command dispatcher  
Unknown behavior: generated enum unknowns are rejected as invalid command actions.  
Validation: enum must be `START` or `STOP` when dispatching a command.  
Tests: server command dispatcher rejects invalid actions.

### Field: terminals.control.v1.CommandRequest.kind

Owner: transport/control  
Classification: typed_contract  
Target state: keep `CommandKind` enum.
Producer: client  
Consumer: server command dispatcher  
Unknown behavior: generated enum unknowns are rejected as invalid command kinds.  
Validation: enum must be known for command dispatch.  
Tests: server command dispatcher rejects invalid kinds.

### Field: terminals.control.v1.CommandRequest.intent

Owner: scenario/transport  
Classification: registry_backed_extension  
Target state: keep registry-backed until high-use commands stabilize, then type stable command shapes.
Producer: client or system command producer  
Consumer: server scenario and system intent dispatchers  
Unknown behavior: server rejects unknown durable intents with a typed command result or control error.  
Validation: non-empty string using namespace tokens such as `system.*` or scenario-owned identifiers.  
Tests: server command tests cover missing and unknown intents.

Token namespaces:

- `system.*`: transport-owned system intents
- scenario-owned names: resolved by the server scenario registry

### Field: terminals.control.v1.CommandRequest.arguments

Owner: scenario/transport  
Classification: transitional_escape_hatch  
Target state: typed `CommandRequest.typed_arguments` now mirrors durable command arguments; keep legacy `arguments` map during the compatibility window.  
Review date: 2026-06-15  
Producer: client or system command producer emits both typed entries and the legacy map for durable argument keys; shared client helpers mirror generic string arguments into `CommandTypedValue.string_value` entries, including playback metadata arguments and optional manual application-launch arguments.  
Consumer: server command dispatcher and scenarios prefer typed entries when present and fall back to the legacy map for older clients.  
Unknown behavior: unknown keys are ignored unless the selected intent explicitly requires them.  
Validation: typed values use `CommandTypedValue` (`string_value`, `int64_value`, `bool_value`, `double_value`, or `string_list_value`); legacy string values remain for compatibility and known keys document stricter formats in scenario or system-intent tests.  
Tests: command dispatcher tests cover current required keys and unknown-key tolerance; `command_request_typed_arguments_v1` golden fixture is decoded by both Go and Dart contract tests and pins typed + legacy coexistence; Flutter command-builder tests assert playback metadata and manual application-launch string arguments are emitted through both typed and legacy surfaces.

Known keys:

- `device_id`: terminal device id
- `device_ids`: comma-separated terminal device ids for multi-device commands
- `activation_id`: scenario activation id
- `artifact_id`: playback/artifact identifier for diagnostic artifact commands
- `target_device_id`: terminal device id selected as the playback target

### Field: terminals.control.v1.CommandResult.notification

Owner: scenario/transport  
Classification: display_debug_string  
Target state: keep as human-readable result text.  
Producer: server  
Consumer: client notification surfaces  
Unknown behavior: not applicable; consumers must not branch on message text.  
Validation: optional UTF-8 display text.  
Tests: UI/transport tests assert presence when needed, not exact behavior based on contents.

### Field: terminals.control.v1.CommandResult.data

Owner: scenario/transport  
Classification: transitional_escape_hatch  
Target state: typed `CommandResult.typed_data` now mirrors durable command result data; keep legacy `data` map during the compatibility window.  
Review date: 2026-06-15  
Producer: server command dispatcher and scenarios emit both typed entries and the legacy map.  
Consumer: clients and diagnostics surfaces prefer typed entries when present and fall back to the legacy map for older servers.  
Unknown behavior: clients ignore unknown keys.  
Validation: typed values use `CommandTypedValue` (`string_value`, `int64_value`, `bool_value`, `double_value`, or `string_list_value`); legacy string values remain for compatibility and known keys document stricter formats with the owning command.  
Tests: command result tests cover currently consumed keys; `command_result_typed_data_v1` golden fixture is decoded by both Go and Dart contract tests and pins typed + legacy coexistence.

### Field: terminals.control.v1.WebRTCSignal.signal_type

Owner: transport/media  
Classification: transitional_escape_hatch  
Target state: typed `signal_type_enum` now emitted/consumed first; keep string fallback during compatibility window.  
Review date: 2026-06-15  
Producer: client and server WebRTC signaling engines  
Consumer: client and server WebRTC signaling engines  
Unknown behavior: reject unknown signal types for media setup.  
Validation: enum values `OFFER`, `ANSWER`, `ICE_CANDIDATE`; legacy string fallback accepts `offer`, `answer`, `candidate`/`ice_candidate`.  
Tests: adapter + client media tests cover enum-first resolution, legacy fallback, and malformed payload behavior.

### Field: terminals.control.v1.WebRTCSignal.payload

Owner: transport/media  
Classification: external_payload  
Target state: keep string payload defined by WebRTC SDP/ICE formats.  
Producer: client and server WebRTC signaling engines  
Consumer: client and server WebRTC signaling engines  
Unknown behavior: malformed external payloads are rejected by the signaling engine.  
Validation: SDP for `offer` and `answer`; ICE candidate JSON/string form for `ice_candidate`; size capped by carrier frame limits.  
Tests: WebRTC signaling tests cover offer, answer, and ICE candidate routing.

## UI

### Field: terminals.ui.v1.Node.props

Owner: ui/server-driven-renderer  
Classification: transitional_escape_hatch  
Target state: move durable props into widget messages; reserve map for documented metadata only.  
Review date: 2026-06-15  
Producer: server UI composer  
Consumer: Flutter server-driven renderer  
Unknown behavior: client renderer ignores unknown props.  
Validation: known props are parsed by shared primitive prop helpers; invalid values fall back to widget defaults.  
Tests: Flutter renderer tests cover primitive prop parsing and unknown props.

Known keys include layout and accessibility hints consumed by shared renderer primitives.

### Field: terminals.ui.v1.TextWidget.style

Owner: ui/server-driven-renderer  
Classification: registry_backed_extension  
Target state: keep documented style token registry; type only if tokens become low-cardinality and stable.  
Producer: server UI composer  
Consumer: Flutter renderer  
Unknown behavior: client falls back to default body style.  
Validation: token string from the UI style registry.  
Tests: renderer tests cover known style tokens and fallback.

Token namespace: shared UI style tokens such as `body`, `title`, `caption`, and scenario-neutral semantic aliases.

### Field: terminals.ui.v1.TextWidget.color

Owner: ui/server-driven-renderer  
Classification: constrained_scalar  
Target state: keep string color token or hex scalar.  
Producer: server UI composer  
Consumer: Flutter renderer  
Unknown behavior: invalid values fall back to inherited/default color.  
Validation: named design token or `#RRGGBB` / `#AARRGGBB` hex value.  
Tests: renderer policy tests cover valid and invalid color values.

### Field: terminals.ui.v1.ImageWidget.url

Owner: ui/media  
Classification: constrained_scalar  
Target state: keep URI string.  
Producer: server UI composer  
Consumer: Flutter renderer/media loader  
Unknown behavior: unsupported schemes are rejected or shown as a load failure.  
Validation: absolute `http`, `https`, or server artifact URI unless a local test fixture explicitly allows another scheme.  
Tests: renderer/media tests cover accepted and rejected URLs.

### Field: terminals.ui.v1.ScrollWidget.direction

Owner: ui/server-driven-renderer  
Classification: transitional_escape_hatch  
Target state: typed `direction_enum` (`ScrollDirection`) now read first; legacy `direction` string is marked `[deprecated = true]` in `api/terminals/ui/v1/ui.proto` and is retained only as a compatibility mirror until two tagged releases past 2026-05-03 elapse.  
Review date: 2026-06-15  
Producer: server UI composer (still mirrors typed enum into the deprecated legacy string during the compatibility window).  
Consumer: Flutter renderer  
Unknown behavior: consumers prefer `direction_enum`, fall back to legacy string, and treat unknown values as vertical.  
Validation: enum values `VERTICAL`, `HORIZONTAL`; legacy string accepts the lowercase forms.  
Tests: renderer tests cover both directions and fallback; adapter populates typed enum + legacy string.

### Field: terminals.ui.v1.ButtonWidget.action

Owner: ui/actions  
Classification: registry_backed_extension  
Target state: keep server-owned action namespace.  
Producer: server UI composer  
Consumer: Flutter client emits `UIAction`; server handles action.  
Unknown behavior: server ignores or rejects unknown actions without client-specific behavior.  
Validation: non-empty action token from server-owned namespace.  
Tests: UI action dispatch tests cover known and unknown actions.

### Field: terminals.ui.v1.GestureAreaWidget.action

Owner: ui/actions  
Classification: registry_backed_extension  
Target state: keep server-owned action namespace.  
Producer: server UI composer  
Consumer: Flutter client emits `UIAction`; server handles action.  
Unknown behavior: server ignores or rejects unknown actions without client-specific behavior.  
Validation: non-empty action token from server-owned namespace.  
Tests: UI action dispatch tests cover known and unknown actions.

### Field: terminals.ui.v1.TransitionUI.transition

Owner: ui/server-driven-renderer  
Classification: registry_backed_extension  
Target state: keep token registry; type only if stable and low-cardinality.  
Producer: server UI composer  
Consumer: Flutter renderer  
Unknown behavior: client applies a safe default transition or no transition.  
Validation: transition token string and non-negative duration.  
Tests: renderer tests cover known tokens and fallback.

### Field: terminals.ui.v1.CanvasWidget.draw_ops_json

Owner: ui/canvas  
Classification: transitional_escape_hatch  
Target state: typed `repeated DrawOp draw_ops` (field 2) is the preferred shape; legacy `draw_ops_json` retained during the compatibility window.  
Review date: 2026-06-15  
Producer: server UI composer  
Consumer: Flutter renderer  
Unknown behavior: client prefers typed `draw_ops` when present; if absent or empty, legacy `draw_ops_json` is consumed; malformed JSON is ignored and canvas renders empty.  
Validation: typed `DrawOp` messages use the `oneof op { line, rect, circle, text, path }` schema in `api/terminals/ui/v1/ui.proto`; legacy JSON is an array of drawing operation objects and must remain experimental.  
Tests: typed messages compile and decode in Go and Dart; `TestCanvasDrawOpsFromJSONParsesAllVariants`/`TestCanvasDrawOpsFromJSONReturnsNilForMalformedOrEmpty`/`TestCanvasDrawOpsFromJSONSkipsInvalidOpsKeepsValidOnes`/`TestDescriptorToUINodeCanvasEmitsTypedAndLegacyDrawOps`/`TestDescriptorToUINodeCanvasMalformedJSONLeavesTypedDrawOpsEmpty`/`TestDescriptorToUINodeCanvasNativeTypedOpsBypassJSONParsing`/`TestDescriptorToUINodeCanvasNativeTypedOpsTrumpsLegacyJSON`/`TestDescriptorToUINodeCanvasNativeEmptyTypedFallsBackToJSON`/`TestDiagnosticsConnectionPulseOverlayProducesTypedCanvasOnWire` in `terminal_server/internal/transport/generated_proto_adapter_test.go` pin the parser, JSON producer, native-typed producer, and end-to-end diagnostics surface; `TestCanvasBuildersPopulateTaggedUnion`/`TestCanvasNodeProducesDescriptorWithTypedAndLegacyMirror`/`TestCanvasNodeIsolatesOpsSlice`/`TestCanvasOpsToJSONReturnsEmptyForOnlyUnspecified`/`TestCanvasOpsToJSONSkipsBogusKeepsValid`/`TestDiagnosticsConnectionPulseOverlayHealthy`/`TestDiagnosticsConnectionPulseOverlayUnhealthy` in `terminal_server/internal/ui/canvas_test.go` pin the typed builders and diagnostics view; the `set_ui_canvas_v1` golden envelope (textproto + binpb) plus matching Go (`assertSetUICanvas` in `terminal_server/internal/protocolcontract/fixtures_test.go`) and Dart (`_assertSetUICanvas` in `terminal_client/test/protocol_contract_test.dart`) assertions pin typed-vs-legacy coexistence on the wire; existing renderer tests cover the typed-first/legacy-fallback consumer path.  
Migration status: typed `DrawOp` schema landed 2026-05-04 (additive); Flutter `ServerDrivenRenderer` consumer wiring shipped 2026-05-04 (typed-first with legacy preview fallback). First (JSON-driven) producer landed 2026-05-05: `descriptorToUINode`'s `case "canvas":` invokes `canvasDrawOpsFromJSON` (in `terminal_server/internal/transport/canvas_draw_ops.go`) to parse `props["draw_ops_json"]` into typed `DrawOp` oneofs while preserving the legacy JSON string verbatim. The parser accepts the `{"ops":[{"line":{...}}|{"rect":{...}}|{"circle":{...}}|{"text":{...}}|{"path":{...}}, ...]}` envelope shape; malformed envelopes return nil, individual ops with zero or multiple variants are skipped, and empty/whitespace input yields nil. Native-typed producer landed 2026-05-05: `terminal_server/internal/ui/canvas.go` adds typed primitive builders (`Line`/`Rect`/`Circle`/`Text`/`Path`), a `CanvasNode(id, ops...)` constructor that produces a `Descriptor` carrying typed ops on a new `Descriptor.CanvasOps` field (`json:"-"`) plus a serialized legacy `draw_ops_json` mirror in props, and a concrete diagnostics surface `DiagnosticsConnectionPulseOverlay(deviceID, healthy, rttMs)` (component id `diagnostics_connection_pulse`). `descriptorToUINode`'s canvas arm now prefers `Descriptor.CanvasOps` via `canvasDrawOpsFromUI` and falls back to JSON parsing when no native ops are set; legacy JSON is preserved verbatim when both surfaces are populated. Two-release legacy-removal clock continues to start 2026-05-05.

## IO

### Field: terminals.io.v1.StartStream.kind

Owner: transport/io  
Classification: transitional_escape_hatch  
Target state: typed `stream_kind` now emitted/consumed first; keep string fallback during compatibility window.  
Review date: 2026-06-15  
Producer: server stream planner  
Consumer: client media and edge stream setup  
Unknown behavior: client rejects unknown stream kinds or treats them as unsupported.  
Validation: enum values `AUDIO`, `VIDEO`, `SENSOR`, `DATA`; legacy string fallback accepts `audio`, `video`, `sensor`, or `data`.  
Tests: transport adapter + contract fixtures cover enum-first resolution and legacy fallback.

### Field: terminals.io.v1.StartStream.metadata

Owner: transport/io  
Classification: transitional_escape_hatch  
Target state: typed `StartStream.routing` (`StreamRouting`) and `StartStream.audio_metadata` (`StreamAudioMetadata`) now carry the stable routing + audio hints; keep legacy map keys as compatibility mirrors during the migration window.  
Review date: 2026-06-15  
Producer: server stream planner emits typed fields and legacy map mirrors. The generated proto adapter derives `audio_metadata` from legacy keys when needed and mirrors typed values back to legacy keys during migration.  
Consumer: server media-control state prefers typed `routing.webrtc_mode` / `audio_metadata` and falls back to legacy `webrtc_mode` / `sample_rate` / `channels` / `codec` map keys. Recording paths consume the normalized metadata map. Client media and edge stream setup ignore unknown keys.  
Unknown behavior: clients and server ignore unknown map keys; consumers prefer typed fields when present.  
Validation: typed `routing.origin` ∈ {`STREAM_ORIGIN_ROUTE_DELTA`, `STREAM_ORIGIN_RESTORE`}; typed `routing.webrtc_mode` ∈ {`WEB_RTC_MODE_SERVER_MANAGED`, `WEB_RTC_MODE_PEER_MANAGED`}. Typed `audio_metadata.sample_rate` and `audio_metadata.channels` are positive integers when set; `audio_metadata.codec` is a non-empty token when set. Legacy map values mirror typed values in lowercase/decimal form.  
Tests: protocol contract fixture `start_stream_audio_v1` covers typed `audio_metadata` + legacy map coexistence; `start_stream_route_delta_v1` and `route_stream_route_delta_v1` cover typed routing + legacy-map coexistence. Transport/media-control tests cover typed-first fallback and metadata normalization.

Known keys:

- `origin`: legacy mirror of `routing.origin` — `route_delta` or `restore`
- `webrtc_mode`: legacy mirror of `routing.webrtc_mode` — `server_managed` or `peer_managed`
- `sample_rate`: legacy mirror of `audio_metadata.sample_rate` (decimal hertz)
- `channels`: legacy mirror of `audio_metadata.channels` (decimal count)
- `codec`: legacy mirror of `audio_metadata.codec` (media codec token)

### Field: terminals.io.v1.RouteStream.kind

Owner: transport/io  
Classification: transitional_escape_hatch  
Target state: typed `stream_kind` now emitted/consumed first; keep string fallback during compatibility window.  
Review date: 2026-06-15  
Producer: server stream router  
Consumer: client media stream routing  
Unknown behavior: client rejects unknown route kinds or treats them as unsupported.  
Validation: enum values `AUDIO`, `VIDEO`, `SENSOR`, `DATA`; legacy string fallback accepts `audio`, `video`, `sensor`, or `data`.  
Tests: transport adapter + client response tests cover enum-first resolution and legacy fallback.

### Field: terminals.io.v1.PlayAudio.format

Owner: transport/media  
Classification: constrained_scalar  
Target state: keep MIME-like audio format string; consider enum for built-ins later.  
Producer: server media planner  
Consumer: client audio playback  
Unknown behavior: client rejects unsupported formats.  
Validation: MIME-like token such as `audio/wav`, `audio/mpeg`, or PCM descriptor used by the audio layer.  
Tests: playback tests cover accepted formats and unsupported-format failure.

### Field: terminals.io.v1.ShowMedia.media_type

Owner: transport/media  
Classification: constrained_scalar  
Target state: keep MIME-like media type string; consider enum for built-ins later.  
Producer: server media planner  
Consumer: client media renderer  
Unknown behavior: client rejects unsupported media types.  
Validation: MIME-like type such as `image/png`, `image/jpeg`, or `video/mp4`.  
Tests: media renderer tests cover accepted and unsupported media types.

### Field: terminals.io.v1.PointerEvent.action

Owner: transport/input  
Classification: transitional_escape_hatch  
Target state: typed `PointerAction` enum (`PointerEvent.action_enum`, field 7) is the typed replacement; legacy `action` string remains during the compatibility window.  
Review date: 2026-06-15  
Producer: client input layer (emits both typed enum and legacy string when both are known)  
Consumer: server input dispatcher (generated proto adapter prefers typed enum, falls back to legacy string when unspecified, and normalizes the resolved action into the generic `InputRequest.Action` path)  
Unknown behavior: server ignores unknown actions or records protocol error; unspecified enum + unknown string is treated as unknown.  
Validation: legacy `down`, `move`, `up`, `cancel`, `scroll`; typed `POINTER_ACTION_DOWN`, `POINTER_ACTION_MOVE`, `POINTER_ACTION_UP`, `POINTER_ACTION_CANCEL`, `POINTER_ACTION_SCROLL`.  
Tests: generated proto adapter tests cover enum-first resolution and legacy fallback for pointer actions; `input_pointer_action_v1` golden envelope (textproto + binpb) plus matching Go and Dart assertions pin typed enum coexistence with the legacy string.

### Field: terminals.io.v1.TouchEvent.action

Owner: transport/input  
Classification: transitional_escape_hatch  
Target state: typed `TouchAction` enum (`TouchEvent.action_enum`, field 3) is the typed replacement; legacy `action` string remains during the compatibility window.  
Review date: 2026-06-15  
Producer: client input layer (emits both typed enum and legacy string when both are known)  
Consumer: server input dispatcher (generated proto adapter prefers typed enum, falls back to legacy string when unspecified, and normalizes the resolved action into the generic `InputRequest.Action` path)  
Unknown behavior: server ignores unknown actions or records protocol error; unspecified enum + unknown string is treated as unknown.  
Validation: legacy `start`, `move`, `end`, `cancel`; typed `TOUCH_ACTION_START`, `TOUCH_ACTION_MOVE`, `TOUCH_ACTION_END`, `TOUCH_ACTION_CANCEL`.  
Tests: generated proto adapter tests cover enum-first resolution for touch actions; add a shared golden fixture when the first durable touch producer lands.

### Field: terminals.io.v1.UIAction.action

Owner: ui/actions  
Classification: registry_backed_extension  
Target state: keep server-owned action namespace.  
Producer: Flutter client from server-provided UI descriptors  
Consumer: server UI action dispatcher  
Unknown behavior: server ignores or rejects unknown actions without client scenario logic.  
Validation: action token must match a server-rendered component/action pair.  
Tests: server UI action tests cover allowed and unknown actions.

### Field: terminals.io.v1.SensorData.values

Owner: transport/sensing  
Classification: registry_backed_extension  
Target state: keep documented sensor key registry with units.  
Producer: client sensor streamer  
Consumer: server sensing and observation pipeline  
Unknown behavior: server ignores unknown sensor keys.  
Validation: numeric readings using key-specific units.  
Tests: sensor streamer and sensing scenario tests cover known keys and unknown-key tolerance.

Known keys:

- `accelerometer_x`, `accelerometer_y`, `accelerometer_z`: m/s^2
- `gyroscope_x`, `gyroscope_y`, `gyroscope_z`: rad/s
- `ambient_light`: lux
- `proximity`: implementation-defined normalized distance

### Field: terminals.io.v1.FlowNode.kind

Owner: edge/flow  
Classification: registry_backed_extension  
Target state: keep operator registry and versioning; type built-in families only when stable.  
Producer: server flow planner  
Consumer: client/server edge execution runtime  
Unknown behavior: executor rejects unknown operator kinds.  
Validation: operator token from runtime registry.  
Tests: flow execution tests cover known and unknown operators.

### Field: terminals.io.v1.FlowNode.args

Owner: edge/flow  
Classification: transitional_escape_hatch  
Target state: typed `FlowNode.typed_args` (`FlowNodeArgs`) now mirrors the stable built-in keys (`device_id`, `resource`, `stream_kind` + typed `stream_kind_enum`, `name`); legacy `args` map remains as a compatibility mirror and as the namespace for operator-specific extension keys until two tagged releases past 2026-05-04 elapse.  
Review date: 2026-06-15  
Producer: server flow planner emits both typed `FlowNodeArgs` (via `flowNodeTypedArgsFromArgs` in `terminal_server/internal/transport/generated_proto_adapter.go`) and the legacy `args` map.  
Consumer: client/server edge execution runtime; consumers prefer typed fields when set and fall back to legacy `args` keys for unknown or older payloads.  
Unknown behavior: executor ignores unknown args unless the operator declares them required.  
Validation: typed `device_id`, `resource`, `name` are non-empty trimmed strings when set; typed `stream_kind` mirrors legacy `args["stream_kind"]` and `stream_kind_enum` resolves to `terminals.io.v1.StreamKind`.  
Tests: flow_plan_basic_v1 contract fixture covers typed `FlowNodeArgs` + legacy map coexistence; flow execution tests cover current operator args.

### Field: terminals.io.v1.FlowNode.exec

Owner: edge/flow  
Classification: transitional_escape_hatch  
Target state: typed `ExecPolicy` (`exec_policy` field) is preferred; legacy `exec` string remains during the compatibility window.  
Review date: 2026-06-15  
Producer: server flow planner emits both typed `exec_policy` enum and legacy `exec` string.  
Consumer: client/server edge execution runtime prefers typed `exec_policy` when not `EXEC_POLICY_UNSPECIFIED`, otherwise falls back to the legacy `exec` string.  
Unknown behavior: executor rejects unknown execution targets; unknown enum values fall through to legacy string handling.  
Validation: typed enum values `EXEC_POLICY_AUTO`, `EXEC_POLICY_PREFER_CLIENT`, `EXEC_POLICY_REQUIRE_CLIENT`, or `EXEC_POLICY_SERVER_ONLY`; legacy string mirrors with `auto`, `prefer_client`, `require_client`, or `server_only`.  
Tests: Go contract tests cover typed-enum emission and legacy-string compatibility for flow plan envelopes; flow execution tests cover execution target selection.

### Field: terminals.io.v1.ArtifactRef.kind

Owner: edge/artifacts  
Classification: registry_backed_extension  
Target state: keep artifact kind registry; enum only for stable built-ins.  
Producer: client/server artifact producers  
Consumer: artifact store, diagnostics, and request handlers  
Unknown behavior: consumers preserve kind and avoid kind-specific handling.  
Validation: artifact kind token from registry.  
Tests: artifact tests cover known kinds and unknown preservation.

### Field: terminals.io.v1.ArtifactRef.uri

Owner: edge/artifacts  
Classification: constrained_scalar  
Target state: keep URI string.  
Producer: artifact store or producer  
Consumer: clients, diagnostics, and artifact request handlers  
Unknown behavior: unsupported schemes are rejected by consumers that need to dereference the URI.  
Validation: content-addressed artifact URI, server artifact URI, or absolute HTTP(S) URI.  
Tests: artifact tests cover accepted and rejected URI schemes.

### Field: terminals.io.v1.Observation.kind

Owner: observation  
Classification: registry_backed_extension  
Target state: keep observation taxonomy registry; enum only for low-cardinality stable categories.  
Producer: sensing/observation pipeline  
Consumer: server scenario engine, world model, diagnostics  
Unknown behavior: preserve observation and avoid kind-specific behavior.  
Validation: taxonomy token from observation registry.  
Tests: observation store tests cover known and unknown kinds.

### Field: terminals.io.v1.Observation.attributes

Owner: observation  
Classification: transitional_escape_hatch  
Target state: typed `Observation.typed_attributes` (`ObservationAttributes`) now mirrors the stable common keys (`label`, `device`, `mac`, `duration_seconds`); legacy `attributes` map remains as the compatibility mirror and the namespace for observation-kind-specific extension keys until two tagged releases past 2026-05-04 elapse.  
Review date: 2026-06-15  
Producer: sensing/observation pipeline; today only the inbound (client→server) path exists in production code (`ObservationMessage` is part of `ConnectRequest`, not `ConnectResponse`), so no outbound proto producer wires the typed mirror yet. The helper `observationTypedAttributesFromInternal` (in `terminal_server/internal/transport/generated_proto_adapter.go`) is staged for the first real outbound producer; when one lands it must populate both `typed_attributes` and the legacy map for the stable keys.  
Consumer: server scenario engine, world model, diagnostics; the generated proto adapter (`observationAttributesFromProto` in `terminal_server/internal/transport/generated_proto_adapter.go`) merges typed fields over the legacy map so internal `iorouter.Observation.Attributes` reflects the typed-first preference.  
Unknown behavior: consumers ignore unknown attributes; legacy keys not represented by typed fields pass through unchanged.  
Validation: typed `label`, `device`, `mac` are trimmed strings when set; `duration_seconds` is a decimal-seconds string mirroring legacy `attributes["duration_seconds"]`.  
Tests: adapter merges typed + legacy keys with typed-first preference; observation contract fixtures cover typed-vs-legacy preference and unknown-key tolerance.

### Field: terminals.io.v1.FlowStats.state

Owner: edge/flow  
Classification: transitional_escape_hatch  
Target state: typed `state_enum` (`FlowState`) now read first; keep legacy string fallback during compatibility window.  
Review date: 2026-06-15  
Producer: client/server edge execution runtime  
Consumer: server diagnostics and admin surfaces  
Unknown behavior: consumers prefer `state_enum`, fall back to legacy string, and treat unknown values as non-running.  
Validation: enum values `STARTING`, `RUNNING`, `DEGRADED`, `STOPPING`, `STOPPED`, `FAILED`; legacy string fallback accepts the lowercase forms.  
Tests: Go transport handler resolves enum-first; protocol contract fixtures cover typed + legacy compatibility.

### Field: terminals.io.v1.FlowStats.error

Owner: edge/flow  
Classification: display_debug_string  
Target state: keep human-readable error text.  
Producer: client/server edge execution runtime  
Consumer: diagnostics and admin surfaces  
Unknown behavior: not applicable; consumers must not branch on message text.  
Validation: optional UTF-8 diagnostic text.  
Tests: diagnostics tests assert propagation, not behavior based on contents.

### Field: terminals.io.v1.InstallBundle.bundle_id

Owner: application-runtime  
Classification: constrained_scalar  
Target state: keep bundle identifier string.  
Producer: server application distribution  
Consumer: client edge bundle store  
Unknown behavior: invalid identifiers are rejected.  
Validation: reverse-DNS or package-style identifier using lowercase letters, digits, dots, underscores, and hyphens.  
Tests: bundle install tests cover valid and invalid identifiers.

### Field: terminals.io.v1.InstallBundle.version

Owner: application-runtime  
Classification: constrained_scalar  
Target state: keep version string.  
Producer: server application distribution  
Consumer: client edge bundle store  
Unknown behavior: invalid versions are rejected.  
Validation: semantic version string unless the package manifest documents a stricter format.  
Tests: bundle install tests cover version validation.

### Field: terminals.io.v1.InstallBundle.sha256

Owner: application-runtime  
Classification: constrained_scalar  
Target state: keep SHA-256 digest string.  
Producer: server application distribution  
Consumer: client edge bundle store  
Unknown behavior: invalid digests are rejected.  
Validation: lowercase 64-character hex SHA-256 digest matching `tar_gz`.  
Tests: bundle install tests cover digest validation.

## Capabilities

### Field: terminals.capabilities.v1.PointerCapability.type

Owner: capability-lifecycle  
Classification: constrained_scalar  
Target state: keep pointer type token until endpoint model is expanded.  
Producer: client capability probe  
Consumer: server capability lifecycle and placement policy  
Unknown behavior: server preserves unknown type and avoids type-specific assumptions.  
Validation: known values include `mouse`, `trackpad`, `stylus`, `remote`, and `unknown`.  
Tests: capability lifecycle tests cover preservation and placement defaults.

## Diagnostics

### Field: terminals.diagnostics.v1.BugReport.source_hints

Owner: diagnostics/bug-reporting  
Classification: registry_backed_extension  
Target state: keep source-specific hint registry.  
Producer: client bug report entry points  
Consumer: server bug report intake and diagnostics views  
Unknown behavior: server preserves unknown hints in the report record without acting on them.  
Validation: values are redacted strings with source-specific size caps.  
Tests: bug report intake tests cover persistence and redaction.

Known keys include `voice_transcript`, `voice_confidence`, `nfc_tag_id`, `sip_caller_id`, and `gesture_name`.

### Field: terminals.diagnostics.v1.UiEventEntry.kind

Owner: diagnostics/bug-reporting  
Classification: transitional_escape_hatch  
Target state: typed `kind_enum` (`terminals.diagnostics.v1.UiEventKind`) mirrors the legacy string for current UI event kinds; keep the legacy string fallback during the compatibility window.  
Review date: 2026-06-15  
Producer: client diagnostics capture (sets typed `kind_enum` alongside the legacy `kind` string for `set_ui`/`update_ui`/`transition_ui`)  
Consumer: server bug report intake and diagnostics views (prefer typed `kind_enum` when non-zero, fall back to legacy string)  
Unknown behavior: server preserves unknown kinds as diagnostic facts.  
Validation: `set_ui`, `update_ui`, or `transition_ui` when known; typed enum matches the values in `terminals.diagnostics.v1.UiEventKind`.  
Tests: bug report tests cover UI event capture; dispatcher tests assert typed-enum mirroring per UI response payload.

### Field: terminals.diagnostics.v1.UiActionEntry.action

Owner: diagnostics/bug-reporting  
Classification: registry_backed_extension  
Target state: mirror server-owned UI action namespace for diagnostics only.  
Producer: client diagnostics capture  
Consumer: server bug report intake and diagnostics views  
Unknown behavior: server preserves unknown actions without executing them.  
Validation: redacted action token captured from a prior UI action.  
Tests: bug report tests cover UI action capture.

### Field: terminals.diagnostics.v1.StreamEntry.kind

Owner: diagnostics/bug-reporting  
Classification: transitional_escape_hatch  
Target state: typed `stream_kind` (`terminals.io.v1.StreamKind`) now mirrors `StartStream.stream_kind`; keep legacy string fallback during compatibility window.  
Review date: 2026-06-15  
Producer: client diagnostics capture (mirrors typed enum from `StartStream.stream_kind` when present, alongside the legacy string)  
Consumer: server bug report intake and diagnostics views  
Unknown behavior: server preserves unknown kinds as diagnostic facts; consumers prefer typed `stream_kind` when non-zero, fall back to legacy string.  
Validation: `audio`, `video`, `sensor`, or `data` when known; typed enum matches the values from `terminals.io.v1.StreamKind`.  
Tests: bug report tests cover stream capture; client widget bootstrap exercises typed-enum mirroring.

### Field: terminals.diagnostics.v1.RouteEntry.kind

Owner: diagnostics/bug-reporting  
Classification: transitional_escape_hatch  
Target state: typed `stream_kind` (`terminals.io.v1.StreamKind`) now mirrors `RouteStream.stream_kind`; keep legacy string fallback during compatibility window.  
Review date: 2026-06-15  
Producer: client diagnostics capture (mirrors typed enum from `RouteStream.stream_kind` when present, alongside the legacy string)  
Consumer: server bug report intake and diagnostics views  
Unknown behavior: server preserves unknown kinds as diagnostic facts.  
Validation: `audio`, `video`, `sensor`, or `data` when known.  
Tests: bug report tests cover route capture.

### Field: terminals.diagnostics.v1.HardwareState.sensor_snapshot

Owner: diagnostics/bug-reporting  
Classification: registry_backed_extension  
Target state: keep snapshot key registry aligned with `SensorData.values`.  
Producer: client diagnostics capture  
Consumer: server bug report intake and diagnostics views  
Unknown behavior: server preserves unknown sensor keys without interpreting them.  
Validation: numeric readings using key-specific units.  
Tests: bug report tests cover sensor snapshot persistence.

## Unresolved

No current flexible fields are unresolved as of 2026-05-03.
