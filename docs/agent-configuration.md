# Agent Configuration

This document is the durable reference for repository configuration consumed by Claude Code, Codex, and GitHub Copilot.

## Required Files

At repository root:

- `CLAUDE.md`: root project guidance for Claude Code.
- `AGENTS.md`: Codex-focused equivalent guidance.
- `.github/copilot-instructions.md`: Copilot workspace guidance.
- `.github/CODEOWNERS`: review ownership routing.
- `.github/workflows/server-ci.yml`: server CI workflow.
- `.github/workflows/client-ci.yml`: client CI workflow.
- `.github/workflows/proto-ci.yml`: proto CI workflow.
- `.github/dependabot.yml`: automated dependency update policy.
- `.editorconfig`: shared formatting defaults.
- `.gitignore`: repository ignores.
- `Makefile`: canonical build/test/lint/validation commands.

Subproject guidance:

- `terminal_server/CLAUDE.md`
- `terminal_client/CLAUDE.md`
- `api/CLAUDE.md`

## Repository Rules Carried by Agent Guidance

All top-level and subproject guidance should remain aligned with these architecture constraints:

1. Keep the Flutter client generic. Scenario behavior belongs on the Go server.
2. Keep client/server contracts in protobuf under `api/terminals/`.
3. Keep AI providers behind interfaces in server code.
4. Build UI using shared server-driven primitives.

## Validation and Maintenance

When changing agent guidance files:

1. Keep `CLAUDE.md` and `AGENTS.md` semantically aligned (Codex-specific additions are fine).
2. Keep protobuf path references accurate (`api/terminals/`, not legacy `api/proto/`).
3. Run `make all-check` from repo root.
4. Regenerate plans index after plan metadata updates with `make plans-index`.
