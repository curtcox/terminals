# Terminals Agent Guide

## Project Overview

Terminals is a thin-client system:

- Flutter clients are generic terminals
- Go server owns scenarios, routing, and orchestration
- Protobuf is the canonical contract for client/server IO

## Repo Layout

- `terminal_server/`: Go server
- `terminal_client/`: Flutter client
- `api/terminals/`: protobuf definitions

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

```bash
make ci-status          # probe CI gates and write scripts/ci-status.json (run before make next)
make server-build
make server-test
make server-lint
make client-build
make client-test
make client-lint
make proto-lint
make proto-generate
make all-check
```

## Local Development

1. Start server: `make run-server`
2. Start web client: `make run-client-web`
3. Connect via discovery/manual connect screen

## Source of Truth

Architectural details live in `masterplan.md`.

## Skills

1. Before acting on any request that names a skill, read `.claude/skills/<name>/SKILL.md` first.
2. Use `SKILLS.md` as the quick index for available skills, trigger phrases, and "must use when" guidance.

