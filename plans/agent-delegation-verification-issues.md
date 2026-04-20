# Agent Delegation Verification Issues

Date verified: 2026-04-20  
Verifier: Codex agent session via `mcp__terminals__` tools  
Source plan: [plans/agent-delegation.md](/Users/curt/me/terminals/plans/agent-delegation.md)

This file tracks the verification issues that were previously outstanding and are now resolved.

## Resolved Issues

1. `repl_describe` now supports setup-doc verification flow without arguments.
- Previous behavior: calling `repl_describe` with empty arguments returned `unknown_command`.
- Fix: no-argument calls now return a registry summary payload in `metadata.commands`.
- Implementation: `terminal_server/internal/mcpadapter/adapter.go` (`ToolReplDescribe` execution path and schema generation).
- Verification: `TestReplDescribeWithoutCommandReturnsRegistrySummary` in `terminal_server/internal/mcpadapter/adapter_test.go`.

2. MCP tool argument schemas no longer mark optional positional arguments as required.
- Previous behavior: optional positionals such as `query` in `app logs <app> [<query>]` and `logs tail [<query>]` were incorrectly required.
- Fix: usage parsing now tracks optional vs required parameters; schema `required` only includes required parameters.
- Runtime fix: command rendering no longer errors for omitted optional positionals.
- Implementation: `usageParams`, `buildArgumentsSchema`, and `renderCommand` in `terminal_server/internal/mcpadapter/adapter.go`.
- Verification: `TestOptionalPositionalArgsAreNotRequiredInSchemas` and `TestOptionalPositionalArgsAreOptionalAtCallTime` in `terminal_server/internal/mcpadapter/adapter_test.go`.

## Current Status

No outstanding issues remain in this verification file as of 2026-04-20.
