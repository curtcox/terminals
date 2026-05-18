# transport

This package owns all client-facing communication: server endpoints (gRPC, TCP, WebSocket, HTTP), the control-stream state machine, command dispatch, wire encoding, media pipelines, and UI surface management. It is the boundary between the network and the scenario/routing layer.

## File groups

### Server endpoints
- `grpc_server.go` — gRPC server lifecycle and connection management
- `grpc_service.go` — gRPC service implementation wiring
- `tcp_server.go` — TCP server hosting framed control streams
- `tcp_stream.go` — framed TCP stream read/write
- `websocket_server.go` — WebSocket server setup and upgrade
- `websocket_stream.go` — WebSocket stream read/write
- `http_control_server.go` — HTTP control endpoint (REST bridge)
- `control_service.go` — service-level glue wiring all server types together

### Control stream
The core per-connection state machine; most feature code lives here.
- `control_stream.go` — main handler entry point and message routing
- `control_stream_handler.go` — StreamHandler type and top-level dispatch
- `control_stream_input.go` — raw input processing (key, text, touch)
- `control_stream_types.go` — shared types used across control_stream_*.go files
- `control_stream_route.go` — route-delta application and stream-routing helpers
- `control_stream_terminal.go` — terminal-mode handling
- `control_stream_media.go` — media state transitions within the stream
- `control_stream_scenario_ui.go` — scenario-driven UI updates
- `control_stream_menu_overlay.go` — overlay push/pop lifecycle
- `control_stream_sensors.go` — sensor event ingestion
- `control_session.go` — ControlSession interface (transport-neutral stream)
- `envelope_session.go` — session wrapper that tags messages with envelope metadata
- `session_relay_registry.go` — registry for cross-device relay senders

### Capability negotiation
- `capability_lifecycle.go` — hello/register/snapshot/delta protocol handling
- `control_stream_capability.go` — capability-change effects on the stream

### Command dispatch
- `command_dispatcher.go` — orchestrates pre/post command flow and response assembly
- `command_model.go` — command value types shared across dispatch
- `control_stream_command.go` — per-command routing to scenario handlers
- `control_stream_command_response.go` — response message construction
- `system_intents.go` — system-level intent constants

### Wire and codec
- `wire_adapter.go` — WireProtoAdapter mapping ProtoEnvelope ↔ ClientMessage
- `wire_convert.go` — low-level proto ↔ internal type conversions
- `wire_messages.go` — wire-level message structs
- `proto_adapter.go` — ProtoAdapter interface and error sentinels
- `passthrough_adapter.go` — no-op adapter for testing
- `payload_map.go` — typed payload extraction helpers
- `strconv.go` — small string/type conversion utilities

### Generated proto adapters
- `generated_proto_adapter.go` — generated adapter bridging protobuf ↔ internal types
- `generated_proto_adapter_capabilities.go` — capability-specific generated adapter

### Media and real-time streaming
- `voice_pipeline.go` — VAD/STT/TTS pipeline orchestration
- `stream_audio_metadata.go` — audio stream metadata handling
- `stream_routing.go` — StreamRouting helpers (origin/WebRTC mode)
- `media_control_state.go` — media state machine (idle, listening, speaking, …)
- `webrtc_pion.go` — Pion WebRTC peer connection management
- `chat_wire.go` — chat message wire format
- `wake_word_dedupe.go` — deduplicate rapid wake-word activations

### UI surfaces and overlays
- `canvas_draw_ops.go` — canvas draw-op encoding
- `corner_affordance.go` — corner-affordance gesture handling
- `ui_scoping.go` — scoped UI update routing
- `ui_session_state.go` — per-session UI state tracking
- `route_replay.go` — route-replay for reconnecting clients

### Diagnostics and observability
- `diagnostics_intake.go` — diagnostics event intake from client
- `metrics.go` — transport-level metrics registration
- `frame_counter_harness.go` — test harness for frame-count assertions

### Shared types and errors
- `errors.go` — package-level sentinel errors
- `command_model.go` — (see Command dispatch above)
