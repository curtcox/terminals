# ai

Current AI selection commands:

- `ai providers [--json]` list configured AI providers.
- `ai models [provider] [--json]` list configured models for one provider.
- `ai use <provider> <model> [--json]` set sticky provider/model selection for the current REPL session.
- `ai status [--json]` show the sticky provider/model selection for the current REPL session.

Selections are stored per REPL session and persist across detach/reattach.
