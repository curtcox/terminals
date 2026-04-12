1. Install `buf`, `flutter`, and `golangci-lint`.
2. Run `make server-lint proto-lint proto-generate`.
3. Implement the gRPC `Connect` control stream using generated Go protobuf types.
4. Wire client discovery + register + heartbeat flow against the control stream.
