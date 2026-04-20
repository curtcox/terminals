---
name: terminals-mcp
description: Drive the Terminals server via its MCP adapter when the `terminals` MCP server is connected. Use when the user asks you to operate, inspect, or change the live Terminals system (devices, activations, claims, apps, logs, scheduler, observe) through Claude Code / Codex rather than by editing code. Triggers include "use the terminals MCP", "list devices", "what activations are running", "stop act_X", "reload the dryer_chime app", "tail the logs", "why is the hallway screen frozen". Skip for code edits in this repo — that's normal file work, not MCP work.
---

# Terminals MCP: how to drive the live server

The `terminals` MCP server is a thin adapter over the REPL command registry. Every tool is a REPL command. Your access equals a REPL user's access — no hidden powers, no shortcuts.

Design doc: [../../../plans/agent-delegation.md](../../../plans/agent-delegation.md).
User-facing setup: [../../../docs/repl/agents/](../../../docs/repl/agents/).

## First moves on a fresh session

1. **Call `repl_describe` with no arguments** to get the command registry summary. This is the authoritative tool catalog; tool descriptions are a subset.
2. **Skim classifications.** Each command is `read_only`, `operational`, or `mutating`. Plan your calls accordingly.
3. **Use `repl_complete`** to probe argument values (device IDs, activation IDs, app names) before committing to a call. Cheap, always safe, prevents wasted approval round-trips on malformed mutating calls.

## Classification rules

| Tier | What you should do |
|---|---|
| `read_only` | Call freely. Exploration, inspection, queries. |
| `operational` | Call freely, but each session has a concurrent-stream cap and stream-TTL budget. Don't stack multiple `logs tail` / `observe tail` streams in parallel when one would do. Cancel streams you don't need. |
| `mutating` | The adapter will pop an approval prompt (MCP elicitation) for the user. Expect a round-trip; don't loop these. If the user said "yes, do it" in chat, that is **not** approval — the MCP elicitation is the approval gate. |

**Never ask the user for a `confirm` or `force` argument.** No tool schema has one. If you think you need one, you're misreading the schema.

## Recommended working loop

1. **Describe the goal back to the user in one sentence** before opening tool calls. Mutating work especially — you're about to ask them to approve prompts.
2. **Explore with `read_only` first.** Gather facts (`devices_ls`, `activations_ls`, `claims_tree`, `logs_query`, `sessions_ls`) before proposing a mutation.
3. **Propose the mutation in plain English**, then call the `mutating` tool once. Wait for the elicitation round-trip. If rejected, stop — don't retry.
4. **Verify.** After a mutation, re-run the relevant `read_only` call to confirm the state changed as expected.

## Tools you should usually avoid

Commands flagged `discouraged_for_agents` in the tool description — honor the flag. Current likely flags:

- **`ai_*` tools** (when they appear). These call the server's own LLM from inside your own LLM turn. Almost always wasteful; prefer your own reasoning and use the REPL's `ai` group only when the user explicitly asks for it.
- **Paged doc browsers / confirmation-loop UIs.** Designed for interactive humans.

The tools are fully callable — the flag is a ranking hint, not a gate. But default to not calling them.

## Common tasks

### "What's broken on device X?"

```
devices_ls                              # confirm device exists, see last-seen
activations_ls --device <id>            # what's running there
claims_tree --device <id>               # what holds which outputs
logs_query 'device_id == "<id>"' --last 15m
```

Read what comes back before proposing anything.

### "Stop activation Y"

```
activations_show <id>                   # read_only — confirm ID and state
activations_stop <id>                   # mutating — will elicit
activations_ls --device <same-device>   # verify resumption / cleanup
```

### "Reload the dryer_chime app after my edits"

```
app_check dryer_chime                   # operational
app_test dryer_chime                    # operational
app_reload dryer_chime                  # mutating — will elicit
```

If `check` or `test` fails, stop and report. Don't reload on a red build.

### "Tail logs for the hallway screen for a few minutes"

```
logs_tail 'device_id == "hallway-screen"'    # operational, streaming
```

Cancel when done. Don't open multiple tails in the same session.

### "Why is X happening?"

Prefer `logs_query` (bounded, one-shot) over `logs_tail` (stream). Only escalate to tail if the signal is live.

## File authoring

You have **no direct filesystem tools.** Any file reading or writing is mediated by REPL commands (today: `ai gen --out ... --write`, `app check/test/reload/rollback`, `docs open`). If an REPL file command exists, it's in the catalog. If it isn't, the capability isn't available through MCP either — don't ask for a workaround; there isn't one.

## Session hygiene

- Client disconnects **detach**, they don't terminate. The session keeps its history, pinned context, and any open streams subject to TTL.
- If you've left junk state (streams, pinned context) and want a clean slate, ask the user if they want `sessions_terminate <id>`.

## When a mutating call returns `unsupported_client`

The client didn't negotiate elicitation or the fallback carrier; this session can only do `read_only` + `operational` work. Tell the user and point them at [../../../docs/repl/agents/troubleshooting.md](../../../docs/repl/agents/troubleshooting.md). Do not try to work around it.

## Parity principle

If you catch yourself wanting "just one more tool" that isn't in the registry, that's a REPL gap, not an MCP gap. Describe what would close it and let the user decide whether to add the REPL command — don't invent an alternate path.
