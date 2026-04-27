# ai

Current AI selection commands:

- `ai providers [--json]` list configured AI providers.
- `ai models [provider] [--json]` list configured models for one provider.
- `ai use <provider> <model> [--json]` set sticky provider/model selection for the current REPL session.
- `ai status [--json]` show the sticky provider/model selection for the current REPL session.
- `ai context [--json]` show pinned context refs that persist across turns.
- `ai context add <ref> [--json]` add one-shot context for the next turn.
- `ai context pin <ref> [--json]` pin a context ref across turns.
- `ai context unpin <ref> [--json]` remove one pinned context ref.
- `ai context clear [--json]` clear all pinned context refs.
- `ai policy show [--json]` show current approval policy (`prompt-mutating` by default).
- `ai policy set <auto-readonly|prompt-all|prompt-mutating> [--json]` update approval policy for the current session.

Selections are stored per REPL session and persist across detach/reattach.

`auto-readonly` is accepted as an alias for `prompt-mutating`.
