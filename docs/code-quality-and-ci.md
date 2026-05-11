# Code Quality and CI

This document is the durable build, lint, and validation contract for Terminals.

## Goals

- Keep local checks aligned with CI checks.
- Make quality gates discoverable through top-level make targets.
- Ensure server, client, and protobuf changes are validated together.

## Engineering Priorities

Quality work comes before feature work. In order:

1. **Specs, design, and code quality beat new features.** A clearer spec, a simpler design, or better-named code is always a valid change to ship on its own.
2. **Fix bugs before adding features.** When touching an area with a known bug or broken path, the bug fix lands first (and ideally as its own commit).
3. **Fix missing tests and static-analysis gaps before adding features.** If the code you are extending lacks tests, or a lint/static-analysis check that would have caught the bug at hand is disabled or missing, add the test or enable the check first.
4. **Encode invariants in CI.** Prefer an automated check (test, lint rule, generated-code drift check, `make` target wired into a workflow under `.github/workflows/`) over a written rule. If you fixed a class of bug, add the check that prevents the regression.
5. **Simplicity over backward compatibility.** Change protobuf, Go, and Dart APIs when the new design is clearer, and update all callers. Do not add compatibility shims, deprecated aliases, or `// kept for compatibility` comments unless an external constraint requires it.

These priorities apply across `terminal_server/`, `terminal_client/`, `android_client/`, `web_client/`, `api/`, and `scripts/`.

## Top-Level Commands

Use these from repository root.

| Target | Purpose |
|---|---|
| `make server-build` | Build Go server |
| `make server-test` | Run server tests |
| `make server-test-sandbox` | Sandbox-friendly server tests; skips packages that need real listeners when blocked |
| `make server-test-network-probe` | Print whether the current environment can bind loopback listeners and enumerate host interfaces |
| `make server-test-network-probe-assert` | Fail unless the network probe reports CI-ready listener/interface coverage |
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
- `make server-test-network-probe-assert`
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

## Sandboxed Server Tests

`make server-test` is the canonical server gate and the form CI runs. Some
restricted local environments (most notably automated agent sandboxes) cannot
bind loopback listeners or enumerate host interfaces, which causes false
failures in `cmd/server`, `internal/admin`, `internal/transport`,
`internal/mcpadapter`, `internal/repl`, and `internal/discovery`.

For those environments, use `make server-test-sandbox`. It:

- runs every server package that does not need real networking,
- runs `make server-test-network-probe` to detect listener support,
- runs the networked package group only when the probe reports that loopback
  binds and host-interface enumeration both work,
- when the networked group is skipped, prints the package list and the exact
  command to run for full validation (`make server-test`).

`make server-test-sandbox` is a development convenience, not a release gate.
CI continues to run the full `make server-test` (and `make all-check`) and
must not be replaced by the sandbox target.

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
