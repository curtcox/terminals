# Phase 0 — Repo Setup, Tooling, and CI

See [masterplan.md](../masterplan.md) for overall system context.

Establish the repo structure, agent configuration, code quality tooling, and CI pipelines before writing any application code. This phase ensures that every subsequent phase starts with working builds, linting, and tests from the first commit.

## Prerequisites

None — this is the first phase.

## Deliverables

- [ ] **Repo structure**: Create `terminal_server/`, `terminal_client/`, `api/proto/` directories.
- [ ] **Go module init**: `go mod init` in `terminal_server/` with initial `main.go` that compiles.
- [ ] **Flutter project init**: `flutter create` in `terminal_client/` with default app that builds.
- [ ] **Buf init**: `buf.yaml` and `buf.gen.yaml` in `api/` with Go and Dart codegen configured.
- [ ] **Root CLAUDE.md**: Project overview, repo layout, build commands, architectural rules, local dev instructions.
- [ ] **Root AGENTS.md**: Codex-compatible version of CLAUDE.md (symlink or tailored copy).
- [ ] **Subproject CLAUDE.md files**: One each in `terminal_server/`, `terminal_client/`, `api/` with language-specific conventions.
- [ ] **.editorconfig**: Tabs for Go, 2-space for Dart/proto/YAML, UTF-8, final newline.
- [ ] **.gitignore**: Comprehensive ignores for Go, Flutter, proto generated code, IDE files, OS files, build artifacts.
- [ ] **Makefile**: All targets listed in [ci.md](ci.md#makefile) — `make all-check` works from day one.
- [ ] **golangci-lint config**: `.golangci.yml` in `terminal_server/` with the linters listed in [ci.md](ci.md#go-server).
- [ ] **Flutter analysis config**: `analysis_options.yaml` in `terminal_client/` with strict rules.
- [ ] **GitHub Actions — server CI**: `.github/workflows/server-ci.yml` — build, lint, test, coverage, govulncheck.
- [ ] **GitHub Actions — client CI**: `.github/workflows/client-ci.yml` — build, analyze, format check, test, coverage.
- [ ] **GitHub Actions — proto CI**: `.github/workflows/proto-ci.yml` — buf lint, buf format, buf breaking.
- [ ] **Dependabot config**: `.github/dependabot.yml` for Go modules and pub packages.
- [ ] **README.md**: Brief project description, build prerequisites, quick start instructions.

## Milestone

Empty project skeleton where `make all-check` passes, all three CI pipelines go green, and agents have full context via `CLAUDE.md` / `AGENTS.md`.

## Related Plans

- [agent-config.md](agent-config.md) — Agent configuration files produced in this phase.
- [ci.md](ci.md) — CI pipelines configured in this phase.
- [technology.md](technology.md) — Tool choices behind each file.
- [phase-1-foundation.md](phase-1-foundation.md) — Next phase.
