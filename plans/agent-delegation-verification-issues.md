# Agent Delegation Verification Issues

Date verified: 2026-04-20  
Verifier: Codex agent session via `mcp__terminals__` tools  
Source plan: [plans/agent-delegation.md](/Users/curt/me/terminals/plans/agent-delegation.md)

## Context

This document captures concrete issues found while validating the MCP access behavior described in the agent delegation plan. Each issue includes reproduction steps and enough technical detail to implement and verify a fix.

## Issue 1: `docs` commands fail because runtime cannot find `docs/repl`

- Severity: High
- Area: REPL docs command runtime path resolution / process working directory
- Plan references:
  - [plans/agent-delegation.md:114](/Users/curt/me/terminals/plans/agent-delegation.md:114)
  - [plans/agent-delegation.md:198](/Users/curt/me/terminals/plans/agent-delegation.md:198)
  - [plans/agent-delegation.md:347](/Users/curt/me/terminals/plans/agent-delegation.md:347)

### Expected

`docs ls`, `docs search`, `docs examples`, and `docs open` should return machine-readable docs content via MCP.

### Actual

All docs calls fail with filesystem-not-found errors.

### Reproduction

1. Call `mcp__terminals__.docs_ls`.
2. Call `mcp__terminals__.docs_search` with query `"mcp setup"`.
3. Call `mcp__terminals__.docs_examples`.
4. Call `mcp__terminals__.docs_open` with topic `"agents/codex-setup"`.

### Observed errors

- `lstat docs/repl: no such file or directory`
- `lstat docs/repl/examples: no such file or directory`
- `open docs/repl/agents/codex-setup.md: no such file or directory`

### Notes for implementation

- Validate how docs root is computed by REPL command handlers in MCP-backed sessions.
- Confirm whether server process cwd differs from repo root and whether docs path should be absolute, config-derived, or repo-root relative.
- Add tests covering docs command behavior when cwd is not repo root.

## Issue 2: Mutating flows are blocked for Codex client (`mutating_unavailable`)

- Severity: High
- Area: MCP capability negotiation (elicitation and fallback approval transport)
- Plan references:
  - [plans/agent-delegation.md:51](/Users/curt/me/terminals/plans/agent-delegation.md:51)
  - [plans/agent-delegation.md:255](/Users/curt/me/terminals/plans/agent-delegation.md:255)
  - [plans/agent-delegation.md:338](/Users/curt/me/terminals/plans/agent-delegation.md:338)

### Expected

Mutating tools should be executable with user approval via elicitation, or via `confirmation_id` fallback if elicitation is unavailable.

### Actual

Current Codex MCP client session is pinned to `mutating_unavailable`; mutating calls are rejected.

### Reproduction

1. Call `mcp__terminals__.sessions_ls` with `json=true`.
2. Observe `AgentCapability` is `mutating_unavailable`.
3. Call `mcp__terminals__.sessions_terminate` with any session id.
4. Call `mcp__terminals__.app_reload` with any app id.

### Observed errors

- `unsupported_client`
- `mutating tools require elicitation or confirmation_id fallback support`

### Notes for implementation

- Confirm the capability probe behavior against current Codex MCP client behavior.
- Check whether elicitation is not negotiated, or whether fallback carrier probing is failing incorrectly.
- Verify session capability transitions and debug logs during handshake.
- Add integration coverage for Codex client negotiation path.

## Issue 3: `repl_describe` success payload is not surfaced with command metadata

- Severity: Medium
- Area: MCP tool result shaping / metadata serialization path
- Plan references:
  - [plans/agent-delegation.md:155](/Users/curt/me/terminals/plans/agent-delegation.md:155)
  - [plans/agent-delegation.md:346](/Users/curt/me/terminals/plans/agent-delegation.md:346)

### Expected

`repl_describe` should return rich per-command metadata usable by agents.

### Actual

Successful calls return only status JSON in visible text payload:

```json
{"confirmation_id":"","error_code":"","error_message":"","rendered_command":"","status":"ok"}
```

### Reproduction

1. Call `mcp__terminals__.repl_describe` with `command="sessions terminate"`.
2. Compare result with `mcp__terminals__.describe` for same command.

### Notes for implementation

- Adapter appears to populate metadata in `CallToolResponse.Metadata`.
- Verify that MCP server result rendering includes serialized metadata in user-visible content for this tool, or that clients reliably expose `_meta.raw_tool_metadata`.
- Current behavior makes discovery tool far less useful despite `status=ok`.

## Issue 4: `repl_complete` success payload is not surfaced with completion matches

- Severity: Medium
- Area: MCP tool result shaping / completion metadata exposure
- Plan references:
  - [plans/agent-delegation.md:154](/Users/curt/me/terminals/plans/agent-delegation.md:154)
  - [plans/agent-delegation.md:346](/Users/curt/me/terminals/plans/agent-delegation.md:346)

### Expected

`repl_complete` should return completion candidates for the provided prefix.

### Actual

Successful calls return only status JSON in visible text payload, with no candidate list.

### Reproduction

1. Call `mcp__terminals__.repl_complete` with `prefix=""` and `limit=20`.
2. Call `mcp__terminals__.repl_complete` with `prefix="sessions "` and `limit=20`.

### Notes for implementation

- Similar to Issue 3, likely metadata-only output not exposed.
- Ensure this tool is practically consumable by agent clients, not only technically successful.

## Issue 5: `complete` command appears not to respect prefix filtering

- Severity: Medium
- Area: REPL completion command logic
- Plan references:
  - [plans/agent-delegation.md:154](/Users/curt/me/terminals/plans/agent-delegation.md:154)

### Expected

`complete <prefix>` should provide completions constrained by prefix.

### Actual

Returned lists appear broad and nearly global even for narrow prefixes.

### Reproduction

1. Call `mcp__terminals__.complete` with `prefix="l"`.
2. Call `mcp__terminals__.complete` with `prefix="sessions s"`.
3. Call `mcp__terminals__.complete` with `prefix="docs o"`.

### Observed behavior

- Results include many unrelated commands that do not match provided prefix scope.

### Notes for implementation

- Validate tokenizer/split logic for subcommand completion prefixes.
- Add targeted unit tests for prefix-restricted completion cases.

## Issue 6: Streaming/operational command set from plan not present in current surface

- Severity: Medium
- Area: REPL registry contents and MCP-generated catalog parity
- Plan references:
  - [plans/agent-delegation.md:112](/Users/curt/me/terminals/plans/agent-delegation.md:112)
  - [plans/agent-delegation.md:268](/Users/curt/me/terminals/plans/agent-delegation.md:268)
  - [plans/agent-delegation.md:324](/Users/curt/me/terminals/plans/agent-delegation.md:324)

### Expected

Operational/streaming commands such as `logs tail`, `observe tail`, and `app logs -f` should exist and map to streaming MCP tool behavior if this phase is intended to be complete.

### Actual

- `describe logs tail` returns unknown command.
- Generated tool catalog currently does not include `logs_*`/`observe_*` tools in this environment.

### Reproduction

1. Call `mcp__terminals__.describe` with `command="logs tail"`.
2. Inspect available `mcp__terminals__` tools and command completions.

### Notes for implementation

- Confirm whether missing commands are expected in current branch/state, or represent a delivery gap.
- If expected later, track as sequencing/dependency gap; if not, implement command registry entries and classification as `operational`.

## Verified Behaviors (Not Issues)

- `sessions ls/show` expose MCP session metadata including:
  - `Origin: "mcp"`
  - `AgentIdentity`
  - `AgentCapability`
- Mutating rejection uses structured `unsupported_client` error rather than silently executing.

## Suggested Fix Order

1. Fix docs path/runtime resolution (Issue 1) to restore core discovery and setup flow.
2. Resolve capability negotiation for Codex client or fallback transport (Issue 2) to unlock mutating approvals.
3. Fix `repl_describe`/`repl_complete` output surfacing (Issues 3-4) for discovery quality.
4. Fix `complete` prefix behavior (Issue 5).
5. Reconcile streaming/operational command availability vs stated phase completeness (Issue 6).
