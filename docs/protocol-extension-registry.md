---
title: "Protocol Extension Registry"
kind: reference
status: active
owner: curtcox
last-reviewed: 2026-05-03
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
Target state: type stable command argument shapes or move them to command-specific messages.  
Review date: 2026-06-15  
Producer: client or system command producer  
Consumer: server command dispatcher and scenarios  
Unknown behavior: unknown keys are ignored unless the selected intent explicitly requires them.  
Validation: values are strings; known keys document stricter formats in scenario or system-intent tests.  
Tests: command dispatcher tests cover current required keys and unknown-key tolerance.

Known keys:

- `device_id`: terminal device id
- `device_ids`: comma-separated terminal device ids for multi-device commands
- `activation_id`: scenario activation id

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
Target state: type durable command result data.  
Review date: 2026-06-15  
Producer: server command dispatcher and scenarios  
Consumer: clients and diagnostics surfaces  
Unknown behavior: clients ignore unknown keys.  
Validation: string values; known keys document stricter formats with the owning command.  
Tests: command result tests cover currently consumed keys.

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
Target state: typed `direction_enum` (`ScrollDirection`) now read first; keep legacy string fallback during compatibility window.  
Review date: 2026-06-15  
Producer: server UI composer  
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
Target state: prefer typed `DrawOp` messages; otherwise define strict JSON schema.  
Review date: 2026-06-15  
Producer: server UI composer  
Consumer: Flutter renderer  
Unknown behavior: malformed JSON is ignored and canvas renders empty.  
Validation: JSON array of drawing operation objects; current use must remain experimental.  
Tests: add malformed JSON and typed replacement tests before durable canvas use.

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
Target state: type stable routing/session hints; document remaining extension namespace.  
Review date: 2026-06-15  
Producer: server stream planner  
Consumer: client media and edge stream setup  
Unknown behavior: clients ignore unknown keys.  
Validation: string values; known keys document stricter formats.  
Tests: stream setup tests cover current keys and unknown-key tolerance.

Known keys:

- `sample_rate`: decimal hertz
- `channels`: decimal count
- `codec`: media codec token

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
Target state: replace with enum.  
Review date: 2026-06-15  
Producer: client input layer  
Consumer: server input dispatcher  
Unknown behavior: server ignores unknown actions or records protocol error.  
Validation: `down`, `move`, `up`, `cancel`, `scroll`.  
Tests: input tests cover known actions and unknown-action handling.

### Field: terminals.io.v1.TouchEvent.action

Owner: transport/input  
Classification: transitional_escape_hatch  
Target state: replace with enum.  
Review date: 2026-06-15  
Producer: client input layer  
Consumer: server input dispatcher  
Unknown behavior: server ignores unknown actions or records protocol error.  
Validation: `start`, `move`, `end`, `cancel`.  
Tests: input tests cover known actions and unknown-action handling.

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
Target state: type built-in operator arguments once stable.  
Review date: 2026-06-15  
Producer: server flow planner  
Consumer: client/server edge execution runtime  
Unknown behavior: executor ignores unknown args unless the operator declares them required.  
Validation: string values with operator-specific validation.  
Tests: flow execution tests cover current operator args.

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
Target state: type stable attributes per observation kind.  
Review date: 2026-06-15  
Producer: sensing/observation pipeline  
Consumer: server scenario engine, world model, diagnostics  
Unknown behavior: consumers ignore unknown attributes.  
Validation: string values with observation-kind-specific formats.  
Tests: observation tests cover current attributes and unknown-key tolerance.

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
Classification: registry_backed_extension  
Target state: keep diagnostic event kind registry.  
Producer: client diagnostics capture  
Consumer: server bug report intake and diagnostics views  
Unknown behavior: server preserves unknown kinds as diagnostic facts.  
Validation: `set_ui`, `update_ui`, or `transition_ui` for current UI events.  
Tests: bug report tests cover UI event capture.

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
Target state: follow `StartStream.kind` migration to `StreamKind`.  
Review date: 2026-06-15  
Producer: client diagnostics capture  
Consumer: server bug report intake and diagnostics views  
Unknown behavior: server preserves unknown kinds as diagnostic facts.  
Validation: `audio`, `video`, `sensor`, or `data` when known.  
Tests: bug report tests cover stream capture.

### Field: terminals.diagnostics.v1.RouteEntry.kind

Owner: diagnostics/bug-reporting  
Classification: transitional_escape_hatch  
Target state: follow `RouteStream.kind` migration to `StreamKind`.  
Review date: 2026-06-15  
Producer: client diagnostics capture  
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
