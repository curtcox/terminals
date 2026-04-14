# Code Quality and CI

See [masterplan.md](../masterplan.md) for overall system context.

Every PR is validated by GitHub Actions. Agents should be able to run the same checks locally before pushing.

## Go (Server)

| Tool | Purpose | CI Check |
|------|---------|----------|
| `go test ./...` | Unit and integration tests | Required to pass |
| `go test -race ./...` | Race condition detection | Required to pass |
| `go test -coverprofile` | Code coverage report | Report uploaded, trend tracked |
| `golangci-lint` | Meta-linter (staticcheck, errcheck, govet, gosimple, ineffassign, unused, etc.) | Required to pass |
| `go vet ./...` | Compiler-level static analysis | Included in golangci-lint |
| `govulncheck` | Known vulnerability detection in dependencies | Required to pass |
| `gofumpt` | Strict formatting (superset of gofmt) | Required — formatting is checked, not auto-fixed |
| `buf lint` | Protobuf style and correctness (run in server CI too) | Required to pass |

**golangci-lint configuration** (`.golangci.yml` in `terminal_server/`):

- Enable: `errcheck`, `staticcheck`, `govet`, `gosimple`, `ineffassign`, `unused`, `gocritic`, `revive`, `misspell`, `prealloc`, `bodyclose`, `exhaustive`
- Enforce: `gofumpt` formatting
- Set appropriate thresholds for cyclomatic complexity

## Flutter (Client)

| Tool | Purpose | CI Check |
|------|---------|----------|
| `flutter test` | Widget and unit tests | Required to pass |
| `flutter analyze` | Dart static analysis (dart analyzer) | Required to pass (zero issues) |
| `dart format --set-exit-if-changed .` | Formatting check | Required to pass |
| `flutter test --coverage` | Code coverage report | Report uploaded, trend tracked |
| `dart pub outdated` | Dependency freshness report | Informational (PR comment) |
| Custom lint rules (`custom_lint`) | Project-specific lint rules via `analysis_options.yaml` | Required to pass |

**analysis_options.yaml** in `terminal_client/`:

- Extend `flutter_lints` (or `very_good_analysis` for stricter rules)
- Enable `strict-casts`, `strict-inference`, `strict-raw-types`
- Project-specific rules: no direct platform imports in non-capability code, etc.

## Protobuf

| Tool | Purpose | CI Check |
|------|---------|----------|
| `buf lint` | Proto style guide enforcement | Required to pass |
| `buf breaking` | Backward compatibility check against main branch | Required to pass |
| `buf generate` | Codegen (Go + Dart) — verify generated code is committed and up to date | Required to pass |
| `buf format -d --exit-code` | Proto formatting | Required to pass |

## CI Pipeline Structure

```yaml
# .github/workflows/server-ci.yml — triggers on changes to terminal_server/ or api/
# .github/workflows/client-ci.yml — triggers on changes to terminal_client/ or api/
# .github/workflows/proto-ci.yml  — triggers on changes to api/
```

Each pipeline:

1. Checks out the repo.
2. Sets up the toolchain (Go, Flutter, Buf).
3. Caches dependencies (`go mod cache`, `pub cache`, `buf cache`).
4. Runs formatting check (fail fast).
5. Runs linters.
6. Runs tests with coverage.
7. Uploads coverage reports (Codecov or similar).
8. For proto: runs `buf breaking` against `origin/main`.

## Makefile

A top-level `Makefile` provides a unified interface for agents and humans:

```makefile
# Top-level targets that agents can discover and run
make server-build      # Build the Go server
make server-test       # Run server tests
make server-lint       # Run golangci-lint
make server-coverage   # Run tests with coverage report
make client-build      # Build the Flutter client (all platforms)
make client-test       # Run Flutter tests
make client-lint       # Run flutter analyze + dart format check
make client-coverage   # Run tests with coverage report
make proto-lint        # Lint proto files with buf
make proto-breaking    # Check proto backward compatibility
make proto-generate    # Regenerate Go + Dart bindings from proto
make all-lint          # Run all linters
make all-test          # Run all tests
make all-check         # Full CI-equivalent check (lint + test + proto)
make run-server        # Start the server locally
make run-client-web    # Start the Flutter web client locally
```

## Related Plans

- [agent-config.md](agent-config.md) — How agents discover these commands.
- [phase-0-setup.md](phase-0-setup.md) — Phase that lands CI.
- [technology.md](technology.md) — Underlying tool choices.
