# Agent Delegation Verification Issues

Date verified: 2026-04-20  
Verifier: Codex agent session via `mcp__terminals__` tools  
Source plan: [plans/agent-delegation.md](/Users/curt/me/terminals/plans/agent-delegation.md)

This file now tracks only currently outstanding issues.

## Outstanding Issue 1: Mutating flows are blocked for Codex client (`mutating_unavailable`)

- Severity: High
- Area: MCP capability negotiation (elicitation and fallback approval transport)
- Plan references:
  - [plans/agent-delegation.md:255](/Users/curt/me/terminals/plans/agent-delegation.md:255)
  - [plans/agent-delegation.md:338](/Users/curt/me/terminals/plans/agent-delegation.md:338)

### Expected

Mutating tools should be executable with user approval via elicitation, or via `confirmation_id` fallback if elicitation is unavailable.

### Actual (directly verified)

Mutating tool calls are rejected in this Codex MCP session with `unsupported_client`.

### Reproduction

1. Call `mcp__terminals__.sessions_terminate` with any value, e.g. `session="sess_nonexistent"`.
2. Observe the returned error:
   - `error_code: "unsupported_client"`
   - `error_message: "mutating tools require elicitation or confirmation_id fallback support"`

### Notes for implementation

- Confirm why this Codex client session is not negotiating either `mutating_via_elicitation` or `mutating_via_fallback`.
- Verify handshake/probe behavior for elicitation and fallback carriers.
- Add integration coverage for this Codex negotiation path so mutating calls can proceed with explicit user approval.
