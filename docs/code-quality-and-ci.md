# Code Quality and CI

This document is the durable build, lint, and validation contract for Terminals.

## Goals

- Keep local checks aligned with CI checks.
- Make quality gates discoverable through top-level make targets.
- Ensure server, client, and protobuf changes are validated together.

## Top-Level Commands

Use these from repository root.

| Target | Purpose |
|---|---|
| `make server-build` | Build Go server |
| `make server-test` | Run server tests |
| `make server-lint` | Run Go lint checks |
| `make server-coverage` | Generate Go coverage profile |
| `make client-build` | Build Flutter web client |
| `make client-test` | Run Flutter tests |
| `make client-lint` | Run analyze and format checks |
| `make client-coverage` | Generate Flutter coverage |
| `make proto-lint` | Run protobuf lint and round-trip test |
| `make proto-breaking` | Check protobuf compatibility vs main |
| `make proto-generate` | Regenerate Go and Dart protobuf bindings |
| `make all-lint` | Run all lint checks |
| `make all-test` | Run all tests |
| `make all-check` | Full repository gate used for routine validation |

## CI Workflows

The repository maintains three workflows:

- `.github/workflows/server-ci.yml`
- `.github/workflows/client-ci.yml`
- `.github/workflows/proto-ci.yml`

### Server CI

Server CI runs on changes in `terminal_server/` and `api/` and includes:

- `go build ./...`
- `golangci-lint`
- `go test ./...`
- `go test -race ./...`
- `go test ./... -coverprofile=coverage.out`
- `govulncheck ./...`
- Coverage artifact upload

### Client CI

Client CI runs on changes in `terminal_client/` and `api/` and includes:

- `flutter analyze`
- `dart format --set-exit-if-changed .`
- `flutter test --coverage`
- Coverage artifact upload
- `dart pub outdated` (informational)
- Build matrix for web, android, linux, windows, macos, and ios

### Proto CI

Proto CI runs on changes in `api/` and includes:

- `buf format -d --exit-code`
- `buf lint`
- `buf generate`
- Generated-code drift check (`git diff --exit-code`)
- `buf breaking` against `main`

## Quality Configuration

### Go lint configuration

`terminal_server/.golangci.yml` enables the core lint set used by CI,
including `errcheck`, `staticcheck` (including gosimple-equivalent checks), `govet`, `ineffassign`,
`unused`, `gocritic`, `revive`, `misspell`, `prealloc`, `bodyclose`,
`exhaustive`, and `gofumpt`.

### Flutter analysis configuration

`terminal_client/analysis_options.yaml` extends `flutter_lints` and enforces:

- `strict-casts`
- `strict-inference`
- `strict-raw-types`

## Related References

- `README.md`
- `docs/agent-configuration.md`
- `docs/usecase-validation-matrix.md`
