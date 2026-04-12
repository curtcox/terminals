1. Install `buf`, `flutter`, and `golangci-lint`.
2. Run `make server-lint proto-lint proto-generate`.
3. Add gRPC/protobuf adapter layer that maps generated `Connect` stream messages to `transport.Session` messages.
4. Replace `SetUI` placeholder transport payloads with generated protobuf UI messages from validated descriptors.
