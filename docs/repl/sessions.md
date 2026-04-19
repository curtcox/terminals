# Sessions

A REPL session is a PTY-backed server process with typed lifecycle APIs:

- create
- attach
- detach
- resize
- send input
- terminate
- list/get

Sessions are modeled as typed control-plane state and can be inspected via `sessions ls` and `sessions show <id>`.
