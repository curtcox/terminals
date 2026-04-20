# Agent Delegation Verification Issues

Date verified: 2026-04-20  
Verifier: Codex agent session via `mcp__terminals__` tools  
Source plan: [plans/agent-delegation.md](/Users/curt/me/terminals/plans/agent-delegation.md)

This file now tracks only currently outstanding issues.

## Outstanding Issues

1. `repl_describe` does not support the setup-doc verification flow without arguments.
- Verified behavior: calling `repl_describe` with empty arguments returns `unknown_command`.
- Current implementation requires a `command` argument in both schema and execution path.
- Impact: setup docs currently imply a no-argument discovery/summary call should work, but runtime behavior rejects it.
- Candidate fixes:
  - Make `repl_describe` support no-argument summary output, or
  - Update setup docs to require `command` explicitly.

2. MCP tool argument schemas incorrectly mark optional positional arguments as required.
- Verified behavior: generated schemas for commands like `app logs <app> [<query>]` and `logs tail [<query>]` mark `query` as required.
- Root cause is argument extraction in `usageParams`/schema generation treating bracketed positional args as required.
- Impact: agents are over-constrained by tool schemas and may fail or over-specify arguments for optional inputs.
- Candidate fix: track optional vs required parameters during usage parsing and only include truly required params in JSON Schema `required`.
