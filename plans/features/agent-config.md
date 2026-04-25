---
title: "Agent Configuration"
kind: plan
status: planned
owner: unowned
validation: none
last-reviewed: 2026-04-25
---

# Agent Configuration

See [masterplan.md](../archive/masterplan-duplicate.md) for overall system context.

Development is primarily driven by Claude Code and Codex. The repo must contain configuration files that give these agents the context they need to work effectively.

## Repo Root Files

```
terminals/
├── CLAUDE.md                    # Claude Code project instructions
├── AGENTS.md                    # Codex agent instructions (same role, Codex convention)
├── .github/
│   ├── workflows/
│   │   ├── server-ci.yml        # Go CI pipeline
│   │   ├── client-ci.yml        # Flutter CI pipeline
│   │   └── proto-ci.yml         # Protobuf lint and breaking change detection
│   ├── copilot-instructions.md  # GitHub Copilot context (optional)
│   └── CODEOWNERS               # PR review routing
├── terminal_server/
│   └── CLAUDE.md                # Server-specific agent instructions
├── terminal_client/
│   └── CLAUDE.md                # Client-specific agent instructions
└── api/
    └── CLAUDE.md                # Proto-specific agent instructions
```

## CLAUDE.md (Root)

The root `CLAUDE.md` gives any agent a map of the project:

- Project overview: what this system is, the thin-client architecture, the "client never changes" constraint.
- Repo layout: where server, client, and proto code live.
- Build commands: how to build, test, lint, and run each component.
- Key architectural rules agents must follow:
  - All new behavior goes in server-side scenarios — never add scenario logic to the client.
  - All IO between client and server is defined in protobuf — never use ad-hoc serialization.
  - AI backends are behind interfaces — never import a specific AI provider directly in scenario code.
  - Server-driven UI uses only the defined primitive components — never add client-side UI components for specific scenarios.
- How to run the full system locally (server + one client).
- Links to this masterplan for deeper context.

## AGENTS.md (Root)

Same content as `CLAUDE.md` — Codex reads `AGENTS.md` by convention. Can be a symlink or a copy with Codex-specific additions (e.g., sandbox setup commands, environment variables for headless testing).

## Subproject CLAUDE.md Files

Each subproject (`terminal_server/`, `terminal_client/`, `api/`) gets its own `CLAUDE.md` with:

- Language/framework-specific conventions (Go idioms, Flutter patterns, proto style).
- How to run tests for that subproject alone.
- Common pitfalls and patterns specific to that codebase.
- Dependency management instructions (go mod, pub, buf).

## Additional Agent-Useful Files

| File | Purpose |
|------|---------|
| `.editorconfig` | Consistent indentation/encoding across editors and agents |
| `.gitignore` | Comprehensive ignores for Go, Flutter, proto, IDE files, build artifacts |
| `Makefile` | Unified build/test/lint commands agents can discover and run |
| `README.md` | Human-readable project overview (agents also read this) |
| `terminal_server/go.mod` | Go module definition — agents need this to understand import paths |
| `terminal_client/pubspec.yaml` | Flutter dependencies — agents need this to understand available packages |
| `api/buf.yaml` | Buf configuration for proto linting and breaking change detection |
| `api/buf.gen.yaml` | Buf codegen config — agents need this to regenerate proto bindings |
| `.github/dependabot.yml` | Automated dependency updates for Go and Flutter |

## Related Plans

- [ci.md](ci.md) — CI pipelines referenced here.
- [phase-0-setup.md](../phases/phase-0-setup.md) — Phase that lands these files.
