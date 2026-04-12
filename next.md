1. Install `buf`, `flutter`, and `golangci-lint`.
2. Run `make server-lint proto-lint proto-generate`.
3. Replace the in-memory transport stream handler with real gRPC `Connect` using generated protobuf types.
4. Wire server `SetUI` protobuf payloads from typed UI descriptor builders.
