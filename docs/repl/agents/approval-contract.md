# Approval Contract

Mutating tool calls are approved out-of-band.

## Primary path

- Client supports elicitation
- Adapter sends approval prompt for the rendered command
- Command executes only on explicit approval

## Fallback path

- On initialize, clients that declare fallback support receive `fallback_probe_token`
- Client must echo that token once via fallback carrier:
  - Streamable HTTP: `Mcp-Confirmation-Id` header
  - stdio: `_meta.terminals_confirmation_id`
- After probe success, first mutating call returns `confirmation_required` plus `confirmation_id`
- Replay with the same command arguments and the confirmation ID
- ID is session-bound, arg-bound, expiring, and single-use

## Fail-closed behavior

If a client supports neither path, mutating commands return `unsupported_client`.
