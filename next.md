1. Install `buf`, `flutter`, and `golangci-lint`.
2. Run `make server-lint proto-lint proto-generate`.
3. Replace `WireProtoAdapter` in server/grpc wiring with generated-protobuf adapter using the same structured semantics already modeled in wire types (`CommandRequest`, `CommandAction` enum, `CommandResult`, `ControlError`, `error_code`, deterministic map payload encoding).
4. Port current wire-level integration coverage to generated protobuf types (`register-first`, device-id consistency, lifecycle, system intents, dedupe, missing-field validation, structured error codes, recoverable error continuation, deterministic map payload checks).
