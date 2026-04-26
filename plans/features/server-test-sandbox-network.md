---
title: "Server Test Sandbox Network Plan"
kind: plan
status: shipped-validated
owner: curtcox
validation: manual
last-reviewed: 2026-04-26
---

# Server Test Sandbox Network Plan

## Status (2026-04-26)

Shipped and validated (manual + CI assertions):

- `scripts/probe_server_test_network.go` reports
  `loopback_listener`, `ipv6_loopback_listener`, `host_interfaces`, and a
  rolled-up `network_sensitive_tests` flag.
- `scripts/server-test-sandbox.sh` runs all non-networked server packages,
  invokes the probe, then runs the networked packages
  (`./cmd/server`, `./internal/admin`, `./internal/transport`,
  `./internal/mcpadapter`, `./internal/repl`, `./internal/discovery`) only
  when the probe reports `ok`. When skipped, it prints the package list and
  the command (`make server-test`) needed for full validation.
- `make server-test-sandbox` and `make server-test-network-probe` targets
  expose the script and probe.
- `make server-test-network-probe-assert` now enforces the CI contract that
  probe output must report loopback listener and host-interface support.
- `.github/workflows/server-ci.yml` runs the assertion before server tests, so
  CI cannot silently degrade networked test coverage.
- `docs/code-quality-and-ci.md` documents the new targets and that
  `make server-test-sandbox` is a development convenience, not a release
  gate.
- `terminal_server/internal/discovery/mdns.go` now factors TXT metadata
  construction into `buildMDNSTXTRecords`, with unit coverage in
  `mdns_test.go`, so record construction can be tested independently from
  network-bound advertiser startup.

## Context

`make server-test` and `make all-check` run server tests that intentionally use
local networking:

- admin HTTP listener tests bind `127.0.0.1:0`,
- WebSocket transport tests bind `127.0.0.1:0`,
- MCP/REPL tests use `httptest.NewServer`, often on `[::1]:0`,
- discovery tests inspect host interfaces while building mDNS zones.

Inside restricted agent sandboxes these can fail with `bind: operation not
permitted` or host-IP discovery errors even when the code passes in a normal
developer environment.

## Goals

- Make sandbox failures obvious and actionable.
- Keep CI and real integration tests exercising actual listeners.
- Let agents run a meaningful default server gate without asking for broad
  permissions for every test loop.
- Avoid weakening production listener behavior or hiding genuine networking
  regressions.

## Non-Goals

- Do not remove real listener coverage from CI.
- Do not replace transport integration tests with mocks only.
- Do not add scenario-specific client behavior.

## Plan

1. **Classify Networked Tests**
   - Add a small convention for tests that require loopback listeners or host
     interface inspection.
   - Candidate labels:
     - `requiresLoopbackListener`,
     - `requiresHTTPTestServer`,
     - `requiresHostInterfaces`.
   - Start with the currently observed packages:
     - `cmd/server`,
     - `internal/transport`,
     - `internal/mcpadapter`,
     - `internal/repl`,
     - `internal/discovery`.

2. **Add an Environment Probe**
   - Add a tiny Go or shell probe under `scripts/` that checks:
     - can bind `127.0.0.1:0`,
     - can bind `[::1]:0` or cleanly detect IPv6 unavailability,
     - can enumerate at least one usable host interface for mDNS tests.
   - The probe should print machine-readable keys such as
     `loopback_listener=ok` and `host_interfaces=blocked`.

3. **Split Fast Sandbox Gate From Full Server Gate**
   - Keep `make server-test` as the full CI-equivalent server test.
   - Add a new target such as `make server-test-sandbox` that:
     - sets `GOCACHE ?= /tmp/terminals-go-build` when not already set,
     - runs non-networked server packages,
     - runs networked packages only when the probe says local listeners are
       available.
   - Document that `server-test-sandbox` is an agent/development convenience,
     not a release gate.

4. **Make Skips Explicit**
   - If tests are skipped because the probe says the environment blocks local
     listeners, print the exact skipped package/test group and the command to
     run for full validation outside the sandbox.
   - Prefer package-level make orchestration over hidden `t.Skip` calls when a
     full integration test should remain mandatory in CI.

5. **Refactor Where It Improves Coverage**
   - For listener lifecycle code, factor pure setup/validation logic so it can
     be tested without a socket.
   - Keep at least one real listener integration test per transport in the full
     gate.
   - For mDNS, separate record construction from interface discovery so record
     metadata can be tested without network-interface access.

6. **CI Contract**
   - CI should continue running `make all-check`.
   - Add a CI assertion that the network probe reports listener support, so CI
     cannot silently degrade to the sandbox gate.

## Acceptance Criteria

- A sandboxed agent can run one documented server test target without false
  failures from blocked loopback listeners.
- The output names any skipped networked test groups and points to the full
  validation command.
- `make all-check` remains the full gate and still runs real listener and mDNS
  integration tests.
- The Go build cache location no longer causes permission failures in the
  documented agent test path.

## Immediate Workaround

Use an unrestricted local environment for the full server test suite:

```bash
cd terminal_server
GOCACHE=/tmp/terminals-go-build go test ./...
```

For the full repository gate:

```bash
GOCACHE=/tmp/terminals-go-build \
GOLANGCI_LINT_CACHE=/tmp/terminals-golangci-lint \
HOME=/Users/curtcox/me/terminals/.home \
PUB_CACHE=/Users/curtcox/me/terminals/.home/.pub-cache \
make all-check
```
