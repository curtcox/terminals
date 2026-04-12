1. Install `buf`, `flutter`, and `golangci-lint`.
2. Run `make server-lint proto-lint proto-generate`.
3. Implement a concrete protobuf adapter using generated `control/capabilities/io/ui` Go types.
4. Wire `grpc_server` `Connect` to `RunProtoSession` and add integration tests for stream lifecycle.
