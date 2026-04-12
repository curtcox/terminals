1. Install `buf`, `flutter`, and `golangci-lint`.
2. Run `make server-lint proto-lint proto-generate`.
3. Replace `PassthroughProtoAdapter` with a generated-protobuf adapter for `control.Connect`.
4. Add protobuf-level integration tests covering register, capability update, heartbeat, disconnect, and initial `SetUI`.
