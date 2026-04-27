# ai

Current AI selection commands:

- `ai providers [--json]` list configured AI providers.
- `ai models [provider] [--json]` list configured models for one provider.
- `ai use <provider> <model> [--json]` set sticky provider/model selection for the current REPL session.
- `ai status [--json]` show the sticky provider/model selection for the current REPL session.
- `ai ask <prompt> [--json]` ask the configured model a question and record the exchange in session history.
- `ai gen <description> [--json]` request generated output from the configured model and record the exchange in session history.
- `ai run [--json]` execute the pending AI-proposed command (alias of `ai approve`).
- `ai approve [--json]` approve and execute the pending AI-proposed command.
- `ai reject [--json]` reject and clear the pending AI-proposed command.
- `ai context [--json]` show pinned context refs that persist across turns.
- `ai context add <ref> [--json]` add one-shot context for the next turn.
- `ai context pin <ref> [--json]` pin a context ref across turns.
- `ai context unpin <ref> [--json]` remove one pinned context ref.
- `ai context clear [--json]` clear all pinned context refs.
- `ai policy show [--json]` show current approval policy (`prompt-mutating` by default).
- `ai policy set <auto-readonly|prompt-all|prompt-mutating> [--json]` update approval policy for the current session.
- `ai history [--json]` show the current AI thread id and recent exchange history.
- `ai reset [--json]` clear the current AI thread id and exchange history.

Selections are stored per REPL session and persist across detach/reattach.

When `ai ask` or `ai gen` returns a `proposed_command`, the REPL stores it as pending and prints an approval prompt. `ai approve` / `ai run` executes that pending command through the typed REPL command surface; `ai reject` clears it without executing.

`auto-readonly` is accepted as an alias for `prompt-mutating`.
