---
title: "Agent Delegation Plan"
kind: plan
status: superseded
owner: copilot
validation: none
last-reviewed: 2026-04-26
---

# Agent Delegation Plan

Status: Completed and drained on 2026-04-26.

The completed work from this plan is now documented in:

- [docs/repl/agents/mcp-setup.md](../../docs/repl/agents/mcp-setup.md)
- [docs/repl/agents/approval-contract.md](../../docs/repl/agents/approval-contract.md)
- [docs/repl/agents/tool-catalog.md](../../docs/repl/agents/tool-catalog.md)
- [docs/repl/agents/claude-code-setup.md](../../docs/repl/agents/claude-code-setup.md)
- [docs/repl/agents/codex-setup.md](../../docs/repl/agents/codex-setup.md)
- [docs/repl/agents/troubleshooting.md](../../docs/repl/agents/troubleshooting.md)

Primary implementation evidence lives in:

- [terminal_server/internal/mcpadapter/adapter.go](../../terminal_server/internal/mcpadapter/adapter.go)
- [terminal_server/internal/mcpadapter/server.go](../../terminal_server/internal/mcpadapter/server.go)
- [terminal_server/cmd/server/main.go](../../terminal_server/cmd/server/main.go)
- [terminal_server/internal/mcpadapter/adapter_test.go](../../terminal_server/internal/mcpadapter/adapter_test.go)
- [terminal_server/internal/mcpadapter/server_test.go](../../terminal_server/internal/mcpadapter/server_test.go)
- [terminal_server/cmd/server/main_test.go](../../terminal_server/cmd/server/main_test.go)

Verification details and resolved follow-up issues are tracked in:

- [plans/audits/agent-delegation-verification-issues.md](../audits/agent-delegation-verification-issues.md)

There are no remaining active tasks in this plan. Future changes to MCP adapter
behavior, approval policy, or setup guidance should be tracked in dedicated
feature plans or audits rather than reopening this completed plan.
