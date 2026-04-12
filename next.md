1. Install `buf`, `flutter`, and `golangci-lint`.
2. Run `make server-lint proto-lint proto-generate`.
3. Implement generated-protobuf adapter mapping `control.Connect` stream messages to internal transport messages (including command `request_id`, `action=start|stop`, `kind=system`, `command_ack`, structured error responses, and data payloads).
4. Add protobuf-level integration tests covering register, capability update, heartbeat, disconnect, initial `SetUI`, command start/stop dedupe, system query results, and recoverable error continuation.
