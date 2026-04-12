1. Install `buf`, `flutter`, and `golangci-lint`.
2. Run `make server-lint proto-lint proto-generate`.
3. Implement generated-protobuf adapter mapping `control.Connect` stream messages to internal transport messages (including `CommandRequest`).
4. Add protobuf-level integration tests covering register, capability update, heartbeat, disconnect, initial `SetUI`, and command-triggered scenario start.
