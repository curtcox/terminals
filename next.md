1. Install `buf`, `flutter`, and `golangci-lint`.
2. Run `make server-lint proto-lint proto-generate`.
3. Implement generated-protobuf adapter mapping `control.Connect` stream messages to internal transport messages (including command `request_id`, `action=start|stop`, `kind=system`, `command_ack`, structured error responses, and data payloads).
4. Add protobuf-level integration tests covering register-first enforcement, per-session device-id consistency, register/capability/heartbeat/disconnect lifecycle, initial `SetUI`, command start/stop dedupe, system query results (`server_status`, `list_devices`, `active_scenarios`), and recoverable error continuation.
