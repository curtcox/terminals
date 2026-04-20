# Agent Delegation Plan

See [masterplan.md](../masterplan.md) for overall system context. See [usecases.md](../usecases.md) for the user stories this plan needs to satisfy. See [repl-and-shell.md](repl-and-shell.md) for the REPL command surface this plan exposes.

## Design Principle

Users who have Claude Code or Codex desktop apps should be able to delegate to those agents **exactly the work they could do themselves by typing into the Terminals REPL** — nothing more, nothing less.

The access Claude Code / Codex has is the access the user has when sitting at a REPL. There is no separate privilege tier, no additional command surface, no alternate approval path, and no bypass around REPL semantics. The desktop agent calls the same typed command registry the REPL dispatches to, through an MCP server that is a thin adapter over the REPL, and results come back in the same structured form a REPL session would see.

After the initial Go server work that ships the MCP adapter (the phases in this plan), **no further Go server changes or restarts are required** to let users do new work through Claude Code / Codex — as long as that work is reachable via commands the REPL already exposes. New REPL commands themselves still require a Go build and restart, per the REPL plan. This is a direct consequence of "agent access equals REPL access": both grow together, from the same source.

This plan is deliberately symmetric between Claude Code and Codex. Both consume the same MCP server over the same transports, and no plan feature depends on a specific desktop app.

## Non-Goals

- **No new command surface.** The MCP adapter exposes the REPL command registry. There is no MCP-only tool, no MCP-only file service, no MCP-only authority.
- **No elevated privileges for agents.** Agents cannot do anything a REPL user cannot. Commands are classified `read_only | operational | mutating` in the REPL command registry; `mutating` calls require out-of-band user approval via MCP elicitation (see [Approval Model](#approval-model)); there is no bypass path to the host filesystem, no host shell, no direct kernel-object manipulation.
- **No model-visible approval argument.** Tool schemas never expose `confirm` or `force`. Approval is carried out-of-band from the tool arguments so the model cannot self-approve.
- **No alternate approval path.** The adapter does not offload mutation safety to the desktop app's UI. The REPL's approval contract is enforced at the adapter boundary regardless of which client is on the other end.
- **No scenario-specific client behavior.** The Flutter client is unaffected.
- **No Go server hot-reload.** Go code changes still require a server restart. This plan scopes itself to "what the REPL can already do without a restart."
- **No auth.** Trusted-LAN only, consistent with the rest of the system today.
- **Not a replacement for the REPL's own `ai ...` commands.** Those remain, for the distinct use case where the user has no Claude Code / Codex, or no internet. This plan adds a second, independent agent surface; it does not supplant the first.

## Two Distinct LLM Use Cases

The system now has two independent LLM-assistance surfaces. They must not be conflated.

| Surface | Who drives the LLM? | Where does the LLM run? | When is it used? |
|---|---|---|---|
| **REPL `ai ...` commands** ([repl-and-shell.md](repl-and-shell.md)) | The server's `AiService`, via OpenRouter or Ollama | Remote (OpenRouter) or local (Ollama) | User has no Claude Code / Codex, or wants a server-managed, offline-capable path |
| **Agent delegation via MCP** (this plan) | Claude Code or Codex desktop app | Wherever the desktop app runs | User has a Claude / Codex account and prefers that UX |

Both paths ultimately call the **same** typed REPL command registry. The difference is only who is holding the LLM turn and where the prompt/response stream lives.

Note: an agent on the MCP surface *can* call the REPL's own `ai ...` tools (parity demands nothing be stripped), but doing so nests one LLM inside another and is rarely useful. The tool descriptions for the `ai` group say so. They are not hidden; they are not encouraged.

## User Experience

### From Claude Code or Codex

The user has Claude Code (or Codex) running on their laptop. The Terminals server is on a Mac mini on the same LAN. In the desktop app, the user has configured the Terminals MCP server once. Then:

> **User:** "The photo frame on the hallway screen is frozen. Figure out why and fix it."
>
> **Claude Code / Codex:** *(calls `activations_ls`, `claims_tree`, `logs_query` — all `read_only` — and returns results)* "`act_42` is suspended because `screen.main` on `hallway-screen` was preempted by `act_51` at 14:02:11. `act_51` is an orphaned `red_alert` that never released. Want me to stop it?"
>
> **User:** "Yes."
>
> **Claude Code / Codex:** *(calls `activations_stop` with `act_51`. Classification is `mutating`, so the adapter issues an MCP `elicitRequest` showing the rendered command to the user; the user approves in the desktop client; the adapter dispatches through the REPL command registry.)* "Stopped. `act_42` resumed on `hallway-screen`."

A second example — authoring a TAL app:

> **User:** "Write me a TAL app that rings a chime when the dryer beeps."
>
> **Claude Code / Codex:** *(calls the same REPL commands a human at the REPL would use to author a TAL app — e.g. `app new`, `ai gen --out ... --write`, `app check`, `app test`, `app reload`. Each mutating call triggers an MCP elicitation; the user approves a batch or each step in the desktop client.)* "Wrote `apps/dryer_chime/`, `app check` passed, `app test` passed, reloaded to 0.1.0."

Nothing the agent did required a server restart. Everything the agent did is something a REPL user could also have done by typing.

### From the REPL (unchanged)

Users without Claude Code / Codex continue to use `ai use ollama ...` and `ai ask ...` as described in [repl-and-shell.md](repl-and-shell.md). That path does not depend on this plan.

## Architecture

```text
┌──────────────────────────────┐         ┌──────────────────────────────┐
│ Claude Code / Codex desktop  │         │ Claude Code / Codex desktop  │
│ (same machine as server)     │         │ (laptop on LAN)              │
│                              │         │                              │
│  MCP client ──── stdio ────┐ │         │  MCP client ── Streamable ─┐ │
│                            │ │         │               HTTP          │ │
└────────────────────────────┼─┘         └────────────────────────────┼─┘
                             │                                        │
                             ▼                                        ▼
                  ┌────────────────────────────────────────────────────┐
                  │              Terminals MCP Adapter                 │
                  │  (thin; in-process in the Go server)               │
                  │                                                    │
                  │  • stdio listener                                  │
                  │  • Streamable HTTP listener on LAN                 │
                  │  • tool catalog generated from REPL command        │
                  │    registry metadata                               │
                  │  • every tool call → REPL dispatch on the          │
                  │    connection's ReplSession → typed control-plane  │
                  │  • enforces REPL approval out-of-band via MCP     │
                  │    elicitation (or confirmation_id fallback)       │
                  └───────────────────────┬────────────────────────────┘
                                          │
                                          ▼
                  ┌────────────────────────────────────────────────────┐
                  │  REPL command registry  (unchanged)                │
                  │  devices / activations / claims / app / logs /     │
                  │  scheduler / observe / docs / ai / ...             │
                  └───────────────────────┬────────────────────────────┘
                                          │
                                          ▼
                  ┌────────────────────────────────────────────────────┐
                  │  Kernel services (unchanged)                       │
                  │  registry / placement / claims / TAR / scheduler / │
                  │  observe / store / telephony / log / ...           │
                  └────────────────────────────────────────────────────┘
```

### The adapter is a thin shim

The adapter does these things only:

1. **Advertise tools.** At startup it walks the REPL command registry and publishes one MCP tool per command (Shape A; see [Tool Catalog Shape](#tool-catalog-shape)). Tool names, descriptions, argument schemas, and `read_only | operational | mutating` classification flags are derived from registry metadata. No hand-maintained duplicate list. Completion and command-describe metadata are exposed as dedicated tools (`repl_complete`, `repl_describe`) for agent discovery. **No `confirm` or `force` argument appears in any tool schema** — approval is not a model-visible arg (see [Approval Model](#approval-model)).
2. **Dispatch.** Each MCP tool call is translated into a REPL command invocation against the connection's `ReplSession` and routed through the same REPL pipeline that a human-typed command uses. The result stream is returned as the MCP tool result.
3. **Stream.** Long-running or streaming REPL commands (`logs tail`, `observe tail`, `app logs -f`) map to MCP streaming tool results; see [Streaming](#streaming).
4. **Enforce the REPL's approval contract out-of-band.** `mutating` calls trigger an MCP elicitation round-trip to the user before dispatch; `operational` calls are metered against a per-session budget; `read_only` calls execute directly. See [Approval Model](#approval-model) for the full contract and the load-bearing design.
5. **Render docs for agents.** `docs open` / `docs search` called through MCP return plain Markdown (not paged terminal output). This is a rendering mode selection at the dispatcher, not a new command.

The adapter holds **no authority** of its own. If the REPL cannot do something, the adapter cannot do it.

### Every MCP connection gets a real REPL session

Each MCP client connection is backed by a real `ReplSession` (as defined in [repl-and-shell.md](repl-and-shell.md)), identified, logged, and tracked like any other session. A new `origin` field on the session record distinguishes `human` from `mcp` for metrics; the type itself is unchanged.

Consequences:

- `sessions ls` shows MCP-origin sessions alongside human sessions; `sessions show <id>` surfaces the origin and the connecting agent's self-reported identity where available.
- Session history, pinned context, and policy are all per-session and survive disconnect/reconnect within the REPL's existing `DetachedSession` rules.
- **Disconnect detaches; it does not terminate.** When an MCP client disconnects, the session moves to `DetachedSession` state exactly as a human terminal client's disconnect would. Termination requires an explicit `sessions terminate <id>` command or an idle-timeout expiry configured at the server.
- `sessions terminate <id>` works on MCP-origin sessions.
- Every MCP call is a structured-logged REPL command — the existing audit path covers it automatically.

Session state is the REPL's state. Whether a human or an agent is on the other end is metadata on the session.

### Why this shape preserves the "no restart" property

Once the MCP adapter is running:

- TAL app changes authored through the REPL (edits + `app check`/`app test`/`app reload`) are visible via TAR's existing hot-reload path. The MCP adapter inherits this for free because every MCP call dispatches through the REPL.
- Mutating control-plane operations (start/stop activations, release claims, schedule jobs) go through the same typed services that already support live execution.
- Adding a **new REPL command** requires a Go build and server restart, same as it does for the REPL itself. When that restart happens, the tool catalog regenerates from the registry and the new tool becomes visible to connected agents. This plan does not promise otherwise.

The no-restart claim is therefore precisely: *no restart is required for any new work that is reachable through commands already in the REPL registry.* That covers the vast majority of day-to-day operation and TAL app development.

## Tool Catalog Shape

**Shape A — one MCP tool per REPL command.** `devices_ls`, `activations_show`, `app_reload`, `claims_tree`, etc. Each tool has a typed argument schema derived from the command's registry metadata. This keeps the LLM's call intent legible to the desktop app's approval UI (the app sees `activations_stop`, not `repl_eval("activations stop act_51")`).

If the tool count grows uncomfortably large, group related read-only commands under a single tool with a subcommand argument — but keep each distinct mutating command as its own tool so the intent-to-tool mapping stays 1:1 for the commands where precision matters most.

There is **no** generic `repl_eval` escape hatch. Shape A only. If a command is not in the registry, it is not reachable — through either the REPL or MCP.

Every generated tool's MCP description includes the command synopsis, a short argument reference, its classification (`read_only | operational | mutating`), a `discouraged_for_agents` hint where applicable (see [Discouragement Hints](#discouragement-hints)), and a human-readable examples block — all sourced from the same command metadata that powers `help <command>` in the REPL.

Two discovery tools are also published, regardless of registry contents:

- `repl_complete` — mirrors the REPL's completion API ([repl-and-shell.md](repl-and-shell.md)'s `Complete`), so agents can probe argument values.
- `repl_describe` — mirrors `DescribeCommand`, giving richer per-command metadata than a tool description can carry.

## Transport

Two transports, both required.

### Stdio

For the case where Claude Code / Codex runs on the same machine as the server, the desktop app launches the adapter as a subprocess and speaks MCP over stdio. This is the desktop-MCP default and gives the lowest-friction single-machine setup.

The subprocess invocation runs the server binary in a mode that attaches its stdio to a new MCP connection against the already-running in-process adapter — or, if no server is running, rejects with a clear message. (It does **not** spin up a second server.) One shared, already-running server; many stdio front-doors.

### Streamable HTTP

For the case where Claude Code / Codex runs on a laptop and the server runs on the Mac mini, the adapter exposes an MCP endpoint using the **Streamable HTTP** transport defined by the current MCP specification. This supersedes the older "HTTP + SSE" shape: SSE remains as a mechanism within Streamable HTTP, not as a separate transport.

mDNS advertisement piggybacks on the existing discovery service so the desktop app can locate the endpoint the same way the Flutter client locates the server. Manual entry is the fallback, matching the existing discovery policy ([plans/discovery.md](discovery.md)).

No authentication. Trusted LAN only. Matches current assumption. If the trust model changes later (see masterplan.md key design decision 10), TLS + a shared-secret or mutual-auth header can be added at that time; no protocol change needed.

The adapter internally treats transport as a pluggable detail. If the MCP spec's network transport story evolves again, only the transport adapter changes.

### File I/O goes through REPL commands only

The desktop app may be remote and therefore not share the server's filesystem. This is fine: **it never needs direct filesystem access**. Any file reading or writing the agent does is mediated by REPL commands — whatever commands the REPL exposes at the time. Today those are `ai gen --out ... --write` (LLM-generated bundle), `app check/test/reload/rollback`, and `docs open` (read-only). If the REPL plan adds explicit `file read` / `file write` commands scoped to the apps tree, those become available to MCP automatically; until then, agents have exactly the same file-authoring surface a human REPL user has.

There is no MCP-only file service. This is a strict consequence of the governing rule: agent access equals REPL access.

## Streaming

Several REPL commands are long-lived — `logs tail`, `observe tail`, `app logs -f`, anything that emits a continuous result stream. The MCP adapter surfaces these as streaming tool results.

This requires a streaming-capable dispatch path beyond the unary `EvalCommand` RPC in [repl-and-shell.md](repl-and-shell.md). Adding it is a **deliverable of this plan** (Phase MCP-1), not a prerequisite in another plan. A streaming REPL dispatch RPC (e.g., `ReplStream`) is introduced alongside the MCP adapter and is usable by any REPL client that needs it — the authoring home is this plan because MCP is the consumer that forces the requirement. A one-line reference note is added to the REPL plan's service contract for discoverability.

A `cancel` message from the desktop app maps to the REPL's existing cancellation path for the in-flight command.

## Discovery and Configuration

A one-time-per-desktop setup step tells Claude Code / Codex how to reach the MCP server:

- **Stdio case:** the user adds an MCP entry pointing at the local server binary with the `mcp-stdio` subcommand.
- **Streamable HTTP case:** the user adds an MCP entry pointing at `http://<server-host>:<port>/mcp` (or resolves it via mDNS).

The server ships a generated MCP config snippet for both cases, available through the REPL as `docs open agents/mcp-setup` so a user on a REPL session can copy-paste it into their Claude Code / Codex config.

The MCP adapter advertises its version and the REPL command-registry version on connection. If the registry changes between connection and a subsequent call (for example, after a server restart with an updated build that added or removed REPL commands), the adapter signals a catalog-refresh event so the desktop app re-reads the tool list.

## Approval Model

The REPL's approval contract is the single source of truth, enforced at the adapter boundary out-of-band from the tool-call arguments.

### Why approval must be out-of-band

A model-visible `confirm=true` argument is **cosmetic**: the model sees the arg in the tool schema, sets it itself, and the adapter has no way to distinguish model-set from human-set. Any gate built on an argument the model can populate adds no safety over trusting the desktop app's UX outright. The adversarial check during review exposed this; the fix below restores the gate's load-bearing property.

### Primary mechanism: MCP elicitation

- `mutating` tool calls: before dispatching, the adapter issues an MCP `elicitRequest` to the client carrying the rendered command string, its arguments, and its classification. MCP clients are obligated to surface elicitations to the user. On user-approve, the adapter dispatches as if a human had typed the command with `--force` (or answered "yes" at the REPL prompt). On user-reject or elicitation timeout, the call returns a structured rejection and no mutation occurs.
- `operational` tool calls: execute directly, but are metered against a per-session budget; see [Operational Commands](#operational-commands).
- `read_only` tool calls: execute directly.
- **No tool schema exposes `confirm` or `force`.** The approval signal arrives only via the elicitation round-trip, which the model cannot fabricate — only the server can originate an elicitation, and only the client-surfaced user can respond. This is what makes the gate load-bearing.

This is the same safety contract the REPL enforces for human-typed commands and for LLM-proposed commands in the REPL's own `ai` flow ([repl-and-shell.md Security and Permissions](repl-and-shell.md#security-and-permissions)): the human approval arrives out of band from the command text.

The desktop app may already prompt the user before issuing the tool call (both Claude Code and Codex do). That prompt is the model → client gate. The adapter's elicitation is the client → server gate. They are distinct and both are cheap. Operators who want to merge them in the client UI may do so; the server does not rely on the client doing anything beyond honoring MCP elicitation.

### Suspicious-approval logging

A silently auto-approving client collapses the gate. The adapter cannot prevent this, but it can surface it to operators. When an elicitation response returns with `elicit_response_latency_ms < 500` (configurable as `agent.approval.min_human_latency_ms`, default 500), the adapter logs `unsafe_confirmation_protocol` with the client identity, session ID, command, a hash of the rendered args, the observed latency, and the approval outcome. The mutation still executes if the elicitation was approved (the adapter does not invent a second refusal), but the event is indexed and visible in `logs query 'kind == "unsafe_confirmation_protocol"'` for audit. Humans virtually never approve in under 500 ms; the threshold exists to flag clients that bypass user interaction.

### Fallback: two-call confirmation_id protocol

For MCP clients that do not yet implement elicitation (spec 2025-06-18 and later), a fallback gate applies. The protocol has one concrete carrier per transport:

1. First mutating call returns a `confirmation_required` tool response with a JSON body:
   ```json
   {
     "status": "confirmation_required",
     "confirmation_id": "<server-generated opaque token>",
     "expires_at": "<RFC3339 timestamp>",
     "rendered_command": "activations stop act_51",
     "classification": "mutating"
   }
   ```
   The `confirmation_id` is server-generated, bound to `(session_id, command, canonicalized args, expires_at)`, single-use, and opaque to the caller.
2. Client replays the same tool call, carrying the ID out-of-band from tool arguments:
   - **Streamable HTTP transport:** `Mcp-Confirmation-Id: <id>` HTTP request header on the replay call.
   - **stdio transport:** `_meta.terminals_confirmation_id` field in the MCP `tools/call` request envelope (MCP's reserved meta slot), alongside the original `arguments` block.
3. Adapter validates: ID exists, bound session matches, bound command matches, bound canonicalized args match, not expired, not previously consumed. On success, marks consumed and dispatches. On any mismatch, rejects with a fresh `confirmation_required` carrying a new ID.

The ID is never in the model-visible tool schema and is not forgeable by the model. Clients are expected to wrap step 2 in a user-visible prompt before replaying; clients that auto-replay without user interaction collapse the safety back to cosmetic, and the adapter logs `unsafe_confirmation_protocol` with the client identity when the round-trip between steps 1 and 2 is under the same latency threshold used for elicitation. The expectation is that this fallback is transitional; new clients should implement elicitation.

### Capability negotiation and fail-closed behavior

Neither elicitation nor every fallback carrier can be assumed to work across all MCP clients. Custom HTTP headers may be stripped by middleware or client stacks; some typed MCP clients normalize `tools/call` to `{name, arguments}` only and drop the `_meta` envelope. The adapter therefore negotiates explicitly at connection time:

1. On connect, the adapter reads the client's declared MCP capabilities (including whether it advertises elicitation support).
2. The session is classified into one of three capability states:
   - `mutating_via_elicitation` — client supports MCP elicitation; all `mutating` calls use the primary mechanism.
   - `mutating_via_fallback` — client lacks elicitation and uses the `confirmation_id` protocol for `mutating` calls.
   - `mutating_unavailable` — client supports neither approval carrier. The session is restricted to `read_only` and `operational` calls only. Any `mutating` tool call on such a session returns a structured error (`unsupported_client`) explaining that approval cannot be round-tripped, with a pointer to the setup docs. **The adapter never silently executes a mutation on a session that cannot carry approval.**
3. The session's capability state is logged and visible in `sessions show <id>`. Operators can audit which connected clients are capable of mutating operations.

This preserves the load-bearing property of the approval gate under client heterogeneity: in the worst case an unsupported client sees a degraded surface, but at no point does the adapter fall through to trusting the model's tool arguments.

### No agent-only gates

There is no "destructive-op gate" specific to agents. The same REPL-registry classification applies identically for humans at the REPL and agents over MCP. Nothing is special-cased.

Everything the adapter executes is logged through the existing structured session log, including elicitation outcomes (approved / rejected / timed-out). `logs query 'session_origin == "mcp"'` and friends work out of the box.

## Operational Commands

A third classification tier sits between `read_only` and `mutating`: **`operational`**. Commands in this tier neither mutate persistent state nor fit the pure one-shot read pattern; they hold resources, open streams, or have externally observable effects. Examples: `logs tail`, `observe tail`, `app logs -f`, any subscription-shaped command.

Why a third tier: treating these as `read_only` and letting agents open unbounded streams lets a runaway agent exhaust buffers, bandwidth, or pipeline capacity without tripping any safety. Treating them as `mutating` and prompting the user for every `logs tail` is ergonomically unacceptable.

Adapter behavior:

- `operational` calls execute directly without elicitation.
- Each session has a concurrent-stream cap and an aggregate open-duration budget, configured server-side (`agent.operational.max_streams`, `agent.operational.stream_ttl`). Exceeding either returns a structured rate-limited response that the model can reason about and back off from.
- The cap is the same for human-origin and MCP-origin sessions; humans just rarely hit it.
- **Any service-level rate limiting or backpressure applied to `read_only` or `operational` commands is origin-blind.** If a future rate limit is added to `logs query` or any other read path, it applies equally to human-origin and MCP-origin sessions. There is no agent-only throttle; parity is preserved in the "MCP can do what the REPL can do, no more, no less" direction. An agent that hammers `logs query` in a loop hits the same limits a human scripting against the REPL would.

**This tier is a REPL-plan edit as much as an MCP-plan edit.** The REPL command registry itself grows from `read_only | mutating` to `read_only | operational | mutating`. The list of currently-streaming commands that should be reclassified is small and is called out as a required update in `repl-and-shell.md`. MCP inherits the classification automatically from the registry.

## Use-Case Coverage

This plan is a second path to cover the same use cases P1–P4 from [usecases.md](../usecases.md):

- **P1** — the MCP session is a REPL session; it is an "interactive REPL session" in every structural sense except that the keystrokes come from an LLM instead of a keyboard.
- **P2** — multiple MCP sessions coexist just like multiple REPL sessions.
- **P3** — LLM-assisted command proposal with review is native to Claude Code / Codex; this plan wires them up to the same typed command registry and approval contract `ai` uses.
- **P4** — provider/model selection is handled by the desktop app itself for this surface. The REPL's `ai use ...` remains the path for the independent server-managed LLM.

It also lightly extends the **I9/I10** development-agent use cases by letting a development agent operate the live server (read-only by default, mutating with confirmation) the same way it reads CLAUDE.md files today.

## Implementation Phases

### Phase 0 — Prerequisites

- [repl-and-shell.md](repl-and-shell.md) Phases A–D (session substrate, REPL core with command registry + classification metadata, typed introspection APIs, documentation system).
- Phase G is **not** a prerequisite. The `ai` command group appears in the MCP tool catalog only after Phase G ships, because the catalog is registry-derived.
- **REPL-plan edits required before MCP-1 ships:** extend the classification model from `read_only | mutating` to `read_only | operational | mutating`, reclassify streaming commands (`logs tail`, `observe tail`, `app logs -f`, and peers) as `operational`, and add a discoverability note pointing at this plan's streaming dispatch RPC. These edits live in repl-and-shell.md; they are small and block MCP-1 tool-catalog generation.

### Phase MCP-1 — Adapter skeleton

- Add a `mcp-adapter` package inside `terminal_server/` that depends only on the REPL command registry.
- Implement registry-walking tool-catalog generation (Shape A), including `repl_complete` and `repl_describe`.
- Implement the dispatch path: MCP tool call → REPL command invocation on the connection's `ReplSession` → result returned as MCP tool result.
- **Add the streaming REPL dispatch RPC** (`ReplStream` or equivalent) used by both the MCP adapter for streaming tool results and by any future REPL client that needs it. This is local to this plan's scope; a reference note is added in repl-and-shell.md's service contract.
- Enforce the approval contract at the adapter boundary:
  - `mutating` calls trigger an MCP `elicitRequest`; dispatch only on user-approve.
  - `operational` calls execute directly under per-session budget.
  - `read_only` calls execute directly.
  - Fallback `confirmation_id` protocol for clients without elicitation support, delivered via Streamable HTTP `Mcp-Confirmation-Id` header or stdio `_meta.terminals_confirmation_id` (see [Approval Model → Fallback](#fallback-two-call-confirmation_id-protocol)).
- **Connection-time capability negotiation** with a fail-closed outcome: clients that support neither elicitation nor the fallback carrier are pinned to `mutating_unavailable` for the session. The adapter never silently executes a mutation on such a session.
- No tool schema contains `confirm` or `force` arguments.
- `unsafe_confirmation_protocol` audit event emitted when elicitation or fallback round-trip latency falls below the configured threshold (default 500 ms).
- Add `origin=mcp` to `ReplSession` records; surface in `sessions ls` / `sessions show`.
- Wire session lifecycle: connect creates a session, **disconnect detaches** (moves to `DetachedSession`), explicit terminate or idle timeout ends it.
- Implement stdio transport and Streamable HTTP transport behind the same dispatcher.
- Structured logging of every MCP call through the existing log path, with `session_origin == "mcp"` and elicitation outcomes.
- mDNS advertisement of the Streamable HTTP endpoint.
- Generated config snippet available via `docs open agents/mcp-setup`.
- Force machine-readable (non-paged) rendering mode for `docs open` / `docs search` when called via MCP.

### Phase MCP-2 — Streaming and cancellation

- Map REPL commands classified `operational` and streaming-shaped (`logs tail`, `observe tail`, `app logs -f`) to MCP streaming tool results with backpressure, using the streaming RPC added in MCP-1.
- Wire MCP `cancel` to the REPL's cancellation path for in-flight commands.
- Enforce the per-session concurrent-stream cap and stream-TTL budget described in [Operational Commands](#operational-commands).

### Phase MCP-3 — Discovery polish and setup docs

- Hand-authored `docs/repl/agents/` topics: `mcp-setup`, `claude-code-setup`, `codex-setup`, `tool-catalog`, `approval-contract`, `troubleshooting`.
- `sessions show` extension to surface agent-identifying metadata where the desktop app provides it.
- `logs` convenience views for MCP-origin activity.

All three phases are the initial Go work to ship the feature. Once they are in, no further Go changes are required to let users do new REPL-reachable work through Claude Code / Codex — new behavior flows in via new REPL commands (which have their own build/restart cost, same as before this plan existed) and via TAL apps on TAR's hot-reload path.

## Acceptance Criteria

- A user with Claude Code or Codex configured against the Terminals MCP server can perform every action they could perform by typing into the REPL, with no additional action the REPL does not expose and no shortcut around the REPL's approval contract.
- The tool catalog is generated from the REPL command registry at server start. Adding or changing a REPL command changes the MCP surface without requiring MCP-adapter code changes.
- After the MCP adapter is deployed, no further Go server restarts are required to let users do new work through Claude Code / Codex **as long as** that work is reachable via commands already in the REPL registry (TAL authoring, `app reload`, control-plane mutations, scheduler management, observation queries). Adding new REPL commands still requires a restart, same as before.
- Both stdio and Streamable HTTP transports are supported and documented.
- Every MCP call is a structured-logged REPL session event, queryable via `logs query`. Elicitation outcomes (approved / rejected / timed-out) are part of the log record.
- No MCP tool schema exposes a `confirm` or `force` argument. Approval for `mutating` calls arrives via MCP elicitation (or the transport-level `confirmation_id` fallback). A model cannot self-approve a mutation by setting an argument.
- Connection-time capability negotiation classifies each session as `mutating_via_elicitation`, `mutating_via_fallback`, or `mutating_unavailable`. Sessions in the third state reject `mutating` tool calls with a structured `unsupported_client` error; the adapter never falls through to trusting the model's arguments.
- `operational`-tier calls execute without elicitation but are subject to a per-session concurrent-stream cap and stream-TTL budget; the same limits apply for human-origin and MCP-origin sessions.
- `repl_complete` and `repl_describe` are first-class MCP tools.
- `docs` calls via MCP return machine-readable Markdown, not paged terminal output.
- MCP-client disconnect detaches the backing `ReplSession`; it does not terminate it. Reattach follows the REPL's existing `DetachedSession` rules.
- The REPL's own `ai ...` commands continue to work independently and are not affected by this plan. They appear in the MCP tool catalog once Phase G of the REPL plan ships, and not before.
- No auth is required on the LAN. No authority is exposed that the REPL does not already expose.

## Discouragement Hints

Some REPL commands are technically reachable by agents but rarely useful for them. Rather than hiding or denying tools (which would violate the exact-equivalence rule), the registry carries an optional `discouraged_for_agents` metadata flag. The adapter copies this flag into the MCP tool description so the model can down-rank the tool in planning without the adapter having to enforce a policy.

Current intended uses:

- `ai_*` tools (once REPL Phase G ships): an agent calling the server's LLM from inside its own LLM turn is almost always wasteful; flag them as discouraged.
- Any future REPL command that exists primarily for human interactive flows (paged doc browsers, confirmation-loop UIs, etc.).

This is a ranking hint, not a gate. The tools are fully callable. The flag is a subset-of-REPL-metadata addition, not an agent-specific restriction; humans at the REPL see the same flag in `describe` output.

## Decisions Made (previously Open Questions)

Resolved during the first two Claude↔Codex review cycles:

1. **Tool catalog shape — Shape A only.** No `repl_eval` escape hatch. Flat registry-derived tool list with a 1:1 mapping between mutating commands and MCP tools so intent stays legible to the desktop app's approval UI.
2. **Approval contract — out-of-band, not model-visible.** A `confirm=true` tool argument is cosmetic (the model can set it itself). Approval instead flows through MCP elicitation; a transport-level `confirmation_id` fallback covers clients that don't yet support elicitation. No tool schema exposes `confirm` or `force`.
3. **Classification tiers — three, not two.** `read_only | operational | mutating`. `operational` covers long-running streams and subscription-shaped commands that have externally observable effects without mutating state; they execute without elicitation but are budget-capped. This is a REPL-plan edit that MCP inherits.
4. **Session type — keep `ReplSession` with an `origin` field.** No parallel `AgentSession` type. `origin=mcp` distinguishes for metrics and session views.
5. **File I/O — through REPL commands only.** No direct file-service MCP tool. Agents use whatever file-writing commands the REPL itself exposes.
6. **Transport — Streamable HTTP, not legacy HTTP+SSE.** Per the MCP spec's 2025-03-26 and 2025-11-25 revisions, Streamable HTTP is the network transport; SSE is a mechanism within it. Stdio remains as the second transport.
7. **Streaming RPC — added in this plan.** Rather than blocking on a streaming addition to repl-and-shell.md's Phase A–D, the streaming REPL dispatch RPC is a deliverable of MCP-1 with a reference note back-ported into the REPL plan's service contract. Reduces cross-plan sequencing.
8. **`ai_*` surfacing — surfaced with a `discouraged_for_agents` hint, not denied.** Denying violates exact-equivalence; a ranking hint preserves access parity while nudging agents away from wasteful nested-LLM calls.

## Related Plans

- [masterplan.md](../masterplan.md) — overall architecture and client/server rules, especially the "all changes live on the server" invariant this plan preserves.
- [usecases.md](../usecases.md) — user stories, especially P1–P4 and I9–I10.
- [repl-and-shell.md](repl-and-shell.md) — the REPL command surface this plan exposes through MCP; the authoritative definition of every capability reachable through agent delegation, and the source of the approval contract the adapter enforces.
- [application-runtime.md](application-runtime.md) — TAR hot reload, the mechanism that makes TAL app changes visible without a server restart.
- [scenario-engine.md](scenario-engine.md) — activation, claims, and lifecycle semantics the REPL (and therefore the MCP adapter) drives.
- [discovery.md](discovery.md) — mDNS + manual-entry discovery, reused for the Streamable HTTP MCP endpoint.
