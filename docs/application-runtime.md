# Application Runtime

This document is the durable reference for the Terminals application runtime.
It captures what is executable today versus what remains planned for TAL/TAR.

See [masterplan.md](../masterplan.md) for overall system context and
[server.md](server.md) for current server implementation details.

## Design Principle

Applications are server-side control-plane programs. The client remains a
generic terminal, so new application behavior ships without scenario-specific
client updates.

The runtime is layered as:

- Kernel (Go): transport, registry, placement, claims, routing, storage,
  telephony, and AI adapters.
- TAR (Terminals Application Runtime): package loading, lifecycle, permissions,
  app registration, and reload behavior.
- TAL (Terminals Application Language): deterministic app logic contract.
- Optional operator kernels: portable compute units placed on server or edge.

## Runtime Status Vocabulary

Use these labels to distinguish shipped behavior from planned contract:

- Implemented: runs in the current codebase and has executable validation.
- Partially implemented: supporting path exists, but end-to-end runtime
  behavior is incomplete.
- Planned: documented target behavior not yet executable.

| Surface | Status | Current validation |
|---|---|---|
| Package loading | Implemented | `term app check <name>` loads app directories through `internal/appruntime`. |
| Package file format (`.tap`) | Partially implemented | `internal/apppackage` builds deterministic `.tap` archives and verifies canonical tar rules (`go test ./internal/apppackage`). |
| Manifest validation | Implemented | package load rejects invalid manifest shape and unsupported language declarations. |
| Exported definitions | Partially implemented | manifests declare exports and packages register exported app definitions, but TAL `match` bodies are not interpreted. |
| TAL parsing/interpretation | Planned | TAL files are present as contract examples; lifecycle hooks are not executed by an interpreter. |
| Lifecycle state snapshots | Planned | `Result.State` is the target contract; durable snapshots are not committed for interpreted TAL activations yet. |
| Host operation commit | Partially implemented | Go `ResultScenario` activations can return UI, scheduler, TTS, broadcast, and bus operations committed by `internal/scenario.ExecuteOperations`; TAL results can be adapted to the same model, but interpreted TAL lifecycle hooks are not wired end-to-end yet. |
| Simulation harness | Partially implemented | `term app test` smoke-tests packages and test declarations; full synthetic-time lifecycle simulation is planned. |

## TAL Design Constraints

TAL is intentionally small and deterministic:

- No threads.
- No arbitrary sockets.
- No raw filesystem access.
- No direct clock access outside host APIs.
- No mutable global state after module load.

A TAL program reacts to typed triggers, updates serializable activation state,
and returns typed operations for the kernel to execute.

## App Package Format

Each application is a directory loaded from disk by the server:

```text
terminal_server/apps/
├── app_name/
│   ├── manifest.toml
│   ├── main.tal
│   ├── lib/
│   ├── tests/
│   ├── kernels/
│   ├── models/
│   └── assets/
```

`manifest.toml` contains:

- `name`
- `version`
- `language = "tal/1"`
- `requires_kernel_api`
- `description`
- `permissions`
- `exports`
- `kernels`
- `models`
- `migrate` (optional)
- `dev_mode`

## Runtime Contracts

TAR loads an app package, validates permissions and API compatibility, compiles
TAL modules, and registers definitions with the scenario engine.

```go
type AppManifest struct {
    Name              string
    Version           string
    Language          string
    RequiresKernelAPI string
    Permissions       []Permission
    Exports           []string
    Kernels           []KernelRef
    Models            []ModelRef
    DevMode           bool
}

type AppDefinition interface {
    Name() string
    Match(req ActivationRequest) bool
    NewActivation(req ActivationRequest) (AppActivation, error)
}

type AppActivation interface {
    ID() string
    DefinitionName() string
    Start(ctx context.Context, env *Environment) error
    Handle(ctx context.Context, env *Environment, trigger Trigger) error
    Stop(ctx context.Context, env *Environment) error
    Suspend(ctx context.Context, env *Environment) error
    Resume(ctx context.Context, env *Environment) error
}
```

The scenario engine remains the supervisor. TAR is the package and execution
layer that feeds it definitions and activations.

## TAL Execution Model

TAL is event-driven, not thread-driven. A handler receives immutable input and
returns:

```go
type Result struct {
    State any
    Ops   []Op
    Emit  []Trigger
    Done  bool
}
```

Execution rules:

- One mailbox per activation.
- Events for one activation are processed serially.
- Activation state must be serializable.
- Live handles are stored as opaque refs, not raw host objects.
- A handler commits all returned ops or none.
- Every commit snapshots activation state for suspend/resume and crash
  recovery.

## TAL Host Module Surface

TAL gets a small host surface behind Go kernel modules:

- `placement`
- `claims`
- `ui`
- `flow`
- `observe`
- `recent`
- `presence`
- `world`
- `scheduler`
- `store`
- `telephony`
- `pty`
- `ai`
- `http`
- `bus`
- `log`

## Permissions

Apps declare permissions explicitly. TAR does not expose undeclared host
services.

Core permissions include:

- `placement.read`
- `claims.request`
- `ui.set`, `ui.patch`, `ui.transition`
- `flow.apply`, `flow.patch`, `flow.stop`
- `recent.pull`
- `presence.read`
- `world.read`, `world.verify`
- `store.kv`, `store.query`
- `scheduler`
- `pty`
- `telephony`
- `ai.stt`, `ai.tts`, `ai.llm`
- `http.outbound`
- `bus.emit`

## Hot Reload and Versioning

Rules:

- Source edits are loaded from disk without server restart.
- New activations use the latest successfully loaded version.
- Existing activations stay pinned to their creation version.
- A package may export `migrate(from_version, state)` for cross-version resume.
- Failed reload keeps the last good version active.

## Terminal-First Development Loop

Authoring path from any attached terminal:

```bash
term app new sound_watch
term app check sound_watch
term app test sound_watch
term app load sound_watch
term app reload sound_watch
term app logs sound_watch
term app trace sound_watch
term app rollback sound_watch
term sim run sound_watch --fixture kitchen_house.yaml
```

Development loop:

1. Open a PTY session into the server.
2. Edit TAL source or app assets.
3. Run `term app check` for schema and permission validation.
4. Run `term app test` or `term sim run`.
5. Run `term app reload`.
6. Observe logs and traces in the same terminal session.

## Heavy Compute Packaging

TAL remains small by design. Performance-sensitive compute does not run in
TAL hooks.

Apps may ship:

- Operator kernels compiled to a portable runtime format.
- Model artifacts for classifiers, trackers, and localizers.

TAL requests observations/actions. The kernel decides where compute runs.

## Related References

- [server.md](server.md)
- [tal-example-kitchen-timer.md](tal-example-kitchen-timer.md)
- [use-case-flows.md](use-case-flows.md)
- [edge-execution-runtime.md](edge-execution-runtime.md)
- [observation-plane.md](observation-plane.md)
