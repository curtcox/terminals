# Terminals — Documentation

Detailed build-and-run guides for each component of the Terminals system.

## Guides

| Guide | Description |
|-------|-------------|
| [Server](server.md) | Go server — configuration, build, run, test |
| [Client — Web](client-web.md) | Flutter web client |
| [Client — macOS](client-macos.md) | Flutter macOS desktop client |
| [Client — iOS](client-ios.md) | Flutter iOS client |
| [Client — Android](client-android.md) | Flutter Android client |
| [Client — Linux](client-linux.md) | Flutter Linux desktop client |
| [Client — Windows](client-windows.md) | Flutter Windows desktop client |
| [Event Taxonomy](event-taxonomy.md) | Server event names emitted to JSONL logs |
| [Use Case Validation Matrix](usecase-validation-matrix.md) | Mapping from use-case IDs to automated validation commands |
| [Development Environment Improvement Log](development-environment-improvement-log.md) | Completed improvements plus local sandbox listener/network test restrictions |

## Quick start

```bash
# 1. Start the server
make run-server

# 2. Start the web client (in a second terminal)
make run-client-web
```

The client discovers the server via mDNS automatically.

## Full validation

Run all linters, tests, and proto checks:

```bash
make all-check
```

Run validation for one mapped use case:

```bash
make usecase-validate USECASE=C1
```

## Protobuf

The canonical API contract lives in `api/terminals/`. After editing `.proto` files:

```bash
make proto-generate   # regenerate Go + Dart stubs
make proto-lint        # lint
make proto-breaking    # check for breaking changes vs main
```

Requires [buf](https://buf.build/docs/installation) to be installed.
