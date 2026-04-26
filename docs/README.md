# Terminals — Documentation

Detailed build-and-run guides for each component of the Terminals system.

## Guides

| Guide | Description |
|-------|-------------|
| [Server](server.md) | Go server — configuration, build, run, test |
| [Code Quality and CI](code-quality-and-ci.md) | Local gates, workflow coverage, and quality-tool contract |
| [Agent Configuration](agent-configuration.md) | Durable conventions and required files for Claude Code, Codex, and Copilot |
| [Technology Choices](technology-choices.md) | Durable server/client technology decisions and evidence |
| [Discovery and Connection](discovery-and-connection.md) | mDNS discovery, manual connect, and carrier fallback behavior |
| [Client — Web](client-web.md) | Flutter web client |
| [Client — macOS](client-macos.md) | Flutter macOS desktop client |
| [Client — iOS](client-ios.md) | Flutter iOS client |
| [Client — Android](client-android.md) | Flutter Android client |
| [Client — Linux](client-linux.md) | Flutter Linux desktop client |
| [Client — Windows](client-windows.md) | Flutter Windows desktop client |
| [Event Taxonomy](event-taxonomy.md) | Server event names emitted to JSONL logs |
| [Use Case Validation Matrix](usecase-validation-matrix.md) | Mapping from use-case IDs to automated validation commands |
| [Use Case Flows](use-case-flows.md) | Durable baseline scenario flow reference |
| [Sensing and Edge Observation Use Case Flows](sensing-use-case-flows.md) | Durable sensing-heavy edge-observation flow reference |
| [Development Environment Improvement Log](development-environment-improvement-log.md) | Completed improvements plus local sandbox listener/network test restrictions |
| [Bug Reporting and Diagnostics](bug-reporting.md) | Bug-intake contracts, endpoints, persistence paths, and validation references |

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
