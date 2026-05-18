# Terminals Agent Guide

## Project Overview

Terminals is a thin-client system:

- Flutter clients are generic terminals
- Go server owns scenarios, routing, and orchestration
- Protobuf is the canonical contract for client/server IO

## Repo Layout

- `terminal_server/`: Go server
- `terminal_client/`: Flutter client (Flutter/Dart, targets web and macOS)
- `android_client/`: native Android/Kindle terminal client
- `web_client/`: plain HTML/JS client for browser-first smoke tests
- `api/terminals/`: protobuf definitions
- `scripts/`: helper scripts invoked by `make` targets
- `plans/`: one directory per plan; `plans/INDEX.md` is auto-generated
- `usecases/`: user-story use-case files; IDs are stable contracts
- `docs/`: build-and-run guides and architecture documents

## Navigation

Hubs to check before grepping:

- [docs/glossary.md](docs/glossary.md) — domain terms with canonical code paths
- [usecases/INDEX.md](usecases/INDEX.md) — auto-generated index of use cases and their validation status
- [plans/INDEX.md](plans/INDEX.md) — auto-generated plan index; `BUILDING` rows are in-flight work
- [SKILLS.md](SKILLS.md) — repo-local skills and trigger phrases

## Engineering Priorities

These rules always win over shipping more features. Apply them in order before starting any new work.

1. **Quality before quantity.** Prioritize clearer specs, better design, and better code over adding features. If a change can be smaller, simpler, or better-named, do that first.
2. **Fix failing CI gates before features.** Before picking up any feature work, run `make ci-status` to probe the current gate state, then `make next` to get the priority recommendation. Any failing gate (detekt, lint, etc.) appears as Priority 0 quality debt and must be cleared before new feature work begins.
3. **Fix bugs before features.** If you discover an existing bug or known-broken code path in the area you are about to touch, fix it before adding anything new. Do not pile features on top of broken foundations.
4. **Fix missing tests and static analysis before features.** If a unit you are about to extend lacks tests, or a lint/static-analysis check is disabled or missing where it would catch the kind of bug you just saw, add the test or enable the check first.
5. **Use CI to enforce quality.** When you fix a class of bug or add a new invariant, add a CI check (test, lint rule, generated-code drift check, `make` target wired into a workflow) so it cannot regress silently. Prefer automation over written rules.
6. **Simplicity over backward compatibility.** This project favors clean, simple code over preserving old shapes. Delete dead code, rename for clarity, change protobuf and APIs when the new design is better, and update all callers. Do not add compatibility shims, deprecated aliases, or `// kept for compatibility` comments unless an external constraint requires it.

## Bug Handling

When working a bug report:

1. Write an automated test that attempts to reproduce the bug.
2. If the test fails (bug reproduced), fix the bug and confirm the test passes.
3. If the test passes (bug not reproduced), the bug is already fixed — close it.

## Core Rules

1. Never add scenario-specific behavior to the client.
2. Define all client/server messages in protobuf, not ad-hoc JSON.
3. Keep AI providers behind interfaces in server code.
4. Build UIs from shared server-driven primitives.

## Build and Check Commands

Start here every session:

```bash
make ci-status          # probe CI gates → writes scripts/ci-status.json
make next               # what to work on (reads ci-status + plan frontmatter)
```

Full command reference:

```bash
make server-build
make server-test
make server-lint
make client-build
make client-test
make client-lint
make proto-lint
make proto-generate
make all-check          # full gate, stops on first failure
make check-fast         # lint + cheap checks only (no builds, no integration tests)
make check-all-keep-going  # same as all-check but -k, surfaces every failure
```

## Side-effecting Targets

These targets start processes, write to disk outside the repo, or mutate running state. Do not run them inside tests or in restricted CI sandboxes without understanding their blast radius.

| Target | What it mutates |
|--------|-----------------|
| `make run-server` | Starts server process on ports 50051–50056 |
| `make run-client-web` | Builds Flutter web and starts HTTP server on port 8080 |
| `make run-local` | Starts server + client; writes `.tmp/run-local-*.log` |
| `make stop-server` | Kills running server process(es) |
| `make usecase-validate USECASE=X` | Runs in-process server; writes to `artifacts/` |
| `make ci-status` | Writes `scripts/ci-status.json` |
| `make bug-resolve` | Writes to `terminal_server/bug_reports/` |
| `make usecases-site` | Generates `docs/usecases-site/` |
| `make ui-inspect` | Launches clients and captures screenshots to `.tmp/` |

### Running a single test

Use these for fast feedback while iterating; the full suites are slow.

```bash
# Go server: one package, optionally one test (regex on test name)
cd terminal_server && go test ./internal/transport/... -run TestNameHere -v
cd terminal_server && go test ./internal/transport/ -run '^TestFoo$' -count=1

# Flutter client: one test file
cd terminal_client && flutter test test/path/to/foo_test.dart
cd terminal_client && flutter test --plain-name "specific test name"

# Android client: one test class
cd android_client && ./gradlew app:testDebugUnitTest --tests com.example.FooTest

# Protobuf: lint a single file
buf lint api/terminals/io/v1/io.proto
```

## Local Development

1. Start server: `make run-server`
2. Start web client: `make run-client-web`
3. Connect via discovery/manual connect screen

## Source of Truth

Architectural details live in `masterplan.md`.

## Git Conventions

Commit messages follow Conventional Commits with a scope:

```
type(scope): short imperative summary

# Examples:
docs(plans): introduce progress-log rollover convention
refactor(scripts): extract use-case ID metadata to YAML sidecar
build: add check-fast and check-all-keep-going targets
fix(transport): handle nil capability snapshot in claim resolution
```

Common types: `feat`, `fix`, `refactor`, `docs`, `build`, `test`, `chore`, `skills`.  
Scope is the top-level directory or subsystem (`server`, `client`, `proto`, `scripts`, `plans`, `transport`, etc.).  
Branch names: `<type>/<short-description>` (e.g., `feat/timer-reminder`, `fix/placement-nil`).

## Oversized Files

Files over ~300 LOC should include a `// CONTENTS:` TOC block near the top. Run `make quality-check` to see what's flagged. See recent commits for examples of the block format.

## Skills

1. Before acting on any request that names a skill, read `.claude/skills/<name>/SKILL.md` first.
2. Use `SKILLS.md` as the quick index for available skills, trigger phrases, and "must use when" guidance.

