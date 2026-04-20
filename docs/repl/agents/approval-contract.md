# Approval Contract

The REPL's approval contract is enforced at the MCP adapter boundary, out-of-band from the tool-call arguments. No tool schema exposes `confirm` or `force`; those would be cosmetic because the model can populate them itself.

## Classification

| Tier | Behavior |
|---|---|
| `read_only` | Executes immediately. |
| `operational` | Executes immediately. Subject to a per-session concurrent-stream cap (`agent.operational.max_streams`) and stream-TTL budget (`agent.operational.stream_ttl`). Exceeding either returns a structured rate-limit error. |
| `mutating` | Requires out-of-band approval via MCP elicitation or the fallback protocol. Executes only on explicit approve. |

The same limits apply for human-origin and MCP-origin sessions.

## Primary path: MCP elicitation

1. Agent calls a `mutating` tool.
2. Adapter issues an MCP `elicitRequest` carrying the rendered command string, arguments, and classification.
3. Client surfaces the prompt to the user.
4. On approve → adapter dispatches as if a human had typed the command with `--force` (or answered "yes" at the REPL prompt).
5. On reject or timeout → adapter returns a structured rejection; no mutation occurs.

## Fallback path: `confirmation_id`

For clients that don't yet support elicitation (spec 2025-06-18 and later):

1. On initialize, clients that declare fallback support receive a `fallback_probe_token`.
2. Client echoes the token once via the fallback carrier to prove the carrier survives the client's transport stack:
   - Streamable HTTP: `Mcp-Confirmation-Id` request header.
   - stdio: `_meta.terminals_confirmation_id` field in the `tools/call` envelope.
3. Probe passes → client is marked `mutating_via_fallback`.
4. First mutating call returns:
   ```json
   {
     "status": "confirmation_required",
     "confirmation_id": "<opaque>",
     "expires_at": "<RFC3339>",
     "rendered_command": "activations stop act_51",
     "classification": "mutating"
   }
   ```
5. Client surfaces a user prompt, then replays the same call carrying the ID on the fallback carrier.
6. Adapter validates session-bound + command-bound + arg-bound + not expired + not previously consumed. On success → dispatch. On mismatch → fresh `confirmation_required` with a new ID.

## Fail-closed

A session is classified at connect time into one of three states:

- `mutating_via_elicitation` — client advertises elicitation support.
- `mutating_via_fallback` — client lacks elicitation but passed the fallback probe.
- `mutating_unavailable` — client supports neither. Mutating tools return `unsupported_client`. **No silent fallthrough to trusting model arguments.**

State is visible in `sessions show <id>`.

## Suspicious approval

Approvals returning faster than `agent.approval.min_human_latency_ms` (default 500 ms) emit an `unsafe_confirmation_protocol` audit event with client identity, session ID, rendered command, arg hash, observed latency, and outcome. The mutation still runs if approved — this is an audit signal for operators, not a second gate.

Query with `logs query 'kind == "unsafe_confirmation_protocol"'`.

## No agent-only gates

The classification is the same for human REPL users and for MCP agents. Agents inherit the gate; they don't face an extra one. Parity runs both directions.
