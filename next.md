1. Install `buf`, `flutter`, and `golangci-lint`.
2. Run `make server-lint proto-lint proto-generate`.
3. Implement gRPC `Connect` stream handlers using generated Go protobuf types.
4. Add server integration tests for register/capability/heartbeat over the stream.
