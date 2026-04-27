# AI Service

Current typed operations exposed via admin-backed REPL APIs:

- Provider and model discovery: `ai providers`, `ai models`
- Sticky provider/model selection per REPL session: `ai use`, `ai status`
- Session context management: `ai context`, `ai context add`, `ai context pin`, `ai context unpin`, `ai context clear`
- Session approval policy: `ai policy show`, `ai policy set`

Still planned:

- Streaming `ai ask` and `ai gen`
- Tool-call proposal/approval loop (`ai approve`, `ai reject`, pending tool-call lifecycle)
- Managed AI thread history and reset APIs
