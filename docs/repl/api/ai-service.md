# AI Service

Current typed operations exposed via admin-backed REPL APIs:

- Provider and model discovery: `ai providers`, `ai models`
- Sticky provider/model selection per REPL session: `ai use`, `ai status`
- Session context management: `ai context`, `ai context add`, `ai context pin`, `ai context unpin`, `ai context clear`
- Session approval policy: `ai policy show`, `ai policy set`
- Session thread inspection and reset: `ai history`, `ai reset`
- Pending proposal lifecycle in REPL: `ai run`, `ai approve`, `ai reject` (triggered when ask/gen responses include `proposed_command`)

Still planned:

- Streaming `ai ask` and `ai gen`
- Server-backed tool-call proposal/approval loop state and reconciliation across clients
