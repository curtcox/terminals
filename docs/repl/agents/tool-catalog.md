# Tool Catalog

The adapter generates MCP tools from the REPL command registry at server start.

## Shape

- **One MCP tool per REPL command.** `app reload` → `app_reload`, `activations stop` → `activations_stop`.
- **Schema comes from registry metadata.** Argument names, types, required/optional, and defaults mirror the REPL's own `describe` output.
- **Classification is copied through.** Each tool description includes `classification: read_only | operational | mutating`.
- **`discouraged_for_agents` hints are copied through** into descriptions. Honor them — they flag tools that are usually wasteful for an agent turn (e.g. nested `ai_*` calls).
- **No `confirm` / `force` arguments.** Approval for mutating calls is out-of-band; see [approval-contract.md](approval-contract.md).
- **No `repl_eval` escape hatch.** If a command isn't in the registry, it isn't reachable — through either the REPL or MCP.

## Discovery tools

Two tools are always published, independent of registry contents:

- **`repl_describe`** — returns richer per-command metadata than the tool description can carry (full synopsis, argument reference, classification, examples, discouragement flag).
- **`repl_complete`** — mirrors the REPL's completion API, so agents can probe argument values before committing to a call.

Use these before calling mutating tools. Cheap, always safe, and prevents wasted elicitation round-trips on malformed calls.

## Catalog refresh

The catalog is generated at server start. Adding or removing REPL commands requires a server restart. On reconnect, the adapter advertises its adapter version and the registry version; desktop clients should re-read the tool list if either changes.

## Docs output

`docs open` and `docs search` called via MCP return plain Markdown — not paged terminal output. Safe to parse.

## File I/O

Agents have no direct filesystem MCP tools. Any file reading or writing is mediated by REPL commands (today: `ai gen --out ... --write`, `app check/test/reload/rollback`, `docs open`). This is deliberate: agent access equals REPL access.
