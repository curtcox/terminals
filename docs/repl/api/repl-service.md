# Repl Service

Concrete typed REPL dispatch surface (implemented in `terminal_server/internal/repl/api.go`):

- `ExecuteCommand(ctx, line, opts) (ExecuteResult, error)`  
  Unary command dispatch that captures full output.
- `ExecuteCommandStream(ctx, line, opts, onChunk) (ExecuteResult, error)`  
  Streaming command dispatch used by MCP adapters and other clients that need chunked output.
- `CommandSpecs() []CommandSpec`  
  Snapshot of registry metadata for tool catalog generation and help surfaces.
- `DescribeCommand(command) (CommandSpec, bool)`  
  Typed metadata lookup for one command.
- `Complete(prefix, limit) []string`  
  Typed completion API for command/argument discovery.

Supporting docs and examples remain in the REPL plan and command docs; this page tracks the typed service contract that is actually shipped.
