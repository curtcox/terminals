---
title: "Phase 0 — Repo Setup, Tooling, and CI"
kind: plan
status: superseded
owner: copilot
validation: none
last-reviewed: 2026-04-25
---

# Phase 0 — Repo Setup, Tooling, and CI

Status: Completed and drained on 2026-04-25.

The durable behavior from this phase now lives in:

- [`README.md`](../../README.md) (project setup and quick-start workflow)
- [`docs/README.md`](../../docs/README.md) (project documentation entry points)
- [`plans/features/ci.md`](../features/ci.md) (build, lint, test, and CI contract)
- [`plans/features/agent-config.md`](../features/agent-config.md) (agent configuration contract)
- [`terminal_server/CLAUDE.md`](../../terminal_server/CLAUDE.md), [`terminal_client/CLAUDE.md`](../../terminal_client/CLAUDE.md), and [`api/CLAUDE.md`](../../api/CLAUDE.md) (subproject conventions)

There are no remaining active tasks in this phase plan. Future setup/tooling
changes should be tracked in feature-level plans and docs updates rather than
reopening this bootstrap phase.

See [masterplan.md](../archive/masterplan-duplicate.md) for overall system context.

Establish the repo structure, agent configuration, code quality tooling, and CI pipelines before writing any application code. This phase ensures that every subsequent phase starts with working builds, linting, and tests from the first commit.

## Prerequisites

None — this is the first phase.

## Deliverables

- [x] **Repo structure**: Create `terminal_server/`, `terminal_client/`, `api/proto/` directories.
- [x] **Go module init**: `go mod init` in `terminal_server/` with initial `main.go` that compiles.
- [x] **Flutter project init**: `flutter create` in `terminal_client/` with default app that builds.
- [x] **Buf init**: `buf.yaml` and `buf.gen.yaml` in `api/` with Go and Dart codegen configured.
- [x] **Root CLAUDE.md**: Project overview, repo layout, build commands, architectural rules, local dev instructions.
- [x] **Root AGENTS.md**: Codex-compatible version of CLAUDE.md (symlink or tailored copy).
- [x] **Subproject CLAUDE.md files**: One each in `terminal_server/`, `terminal_client/`, `api/` with language-specific conventions.
- [x] **.editorconfig**: Tabs for Go, 2-space for Dart/proto/YAML, UTF-8, final newline.
- [x] **.gitignore**: Comprehensive ignores for Go, Flutter, proto generated code, IDE files, OS files, build artifacts.
- [x] **Makefile**: All targets listed in [ci.md](../features/ci.md#makefile) — `make all-check` works from day one.
- [x] **golangci-lint config**: `.golangci.yml` in `terminal_server/` with the linters listed in [ci.md](../features/ci.md#go-server).
- [x] **Flutter analysis config**: `analysis_options.yaml` in `terminal_client/` with strict rules.
- [x] **GitHub Actions — server CI**: `.github/workflows/server-ci.yml` — build, lint, test, coverage, govulncheck.
- [x] **GitHub Actions — client CI**: `.github/workflows/client-ci.yml` — build, analyze, format check, test, coverage.
- [x] **GitHub Actions — proto CI**: `.github/workflows/proto-ci.yml` — buf lint, buf format, buf breaking.
- [x] **Dependabot config**: `.github/dependabot.yml` for Go modules and pub packages.
- [x] **README.md**: Brief project description, build prerequisites, quick start instructions.

## Milestone

Empty project skeleton where `make all-check` passes, all three CI pipelines go green, and agents have full context via `CLAUDE.md` / `AGENTS.md`.

## Related Plans

- [agent-config.md](../features/agent-config.md) — Agent configuration files produced in this phase.
- [ci.md](../features/ci.md) — CI pipelines configured in this phase.
- [technology.md](../features/technology.md) — Tool choices behind each file.
- [phase-1-foundation.md](phase-1-foundation.md) — Next phase.
