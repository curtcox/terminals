# Tool Catalog

The adapter generates tools from the REPL command registry on startup.

- One tool per REPL command (`app reload` -> `app_reload`)
- Classification copied from registry metadata (`read_only | operational | mutating`)
- `discouraged_for_agents` hints copied into tool descriptions
- Additional discovery tools: `repl_complete`, `repl_describe`
