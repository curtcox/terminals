# connection

Control-stream client implementations and session management.

`control_client.dart` defines the `ControlClient` interface. Platform-specific implementations cover gRPC (`control_client_tcp_io.dart`), WebSocket (`control_client_ws.dart`), and HTTP (`control_client_http_io.dart`), with stub variants for unsupported platforms. `control_session_controller.dart` manages reconnection and backoff. `endpoint_resolution.dart` resolves server addresses from config or discovery. `reliability.dart` wraps streams with retry logic. `transport_diagnostics.dart` surfaces transport-layer events.
