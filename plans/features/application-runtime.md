# Application Runtime
See [masterplan.md](../masterplan.md) for overall system context. See [scenario-engine.md](scenario-engine.md) for the existing activation model this runtime extends.

## Design Principle
Applications are **server-side control-plane programs**. The client remains a generic terminal. New application behavior still ships without scenario-specific client updates.

The runtime adds one layer above the existing Go server modules:
- **Kernel (Go)**: transport, registry, placement, claims, routing, storage, telephony, AI adapters.
- **TAR (Terminals Application Runtime)**: package loader, sandbox, hot reload, lifecycle, permissions.
- **TAL (Terminals Application Language)**: deterministic application language for scenario logic.
- **Optional operator kernels**: portable compute units that TAR may place on a capable client or on the server. See [edge-execution.md](edge-execution.md).

## Runtime Status Vocabulary

Runtime docs use these labels so implemented behavior is distinguishable from
the planned TAL/TAR contract:

- `Implemented`: runs in the current codebase and is covered by executable validation.
- `Partially implemented`: some supporting path exists, but the full runtime contract is not executable yet.
- `Planned`: documented target behavior that still needs runtime implementation.

| Surface | Status | Current validation |
|---|---|---|
| Package loading | Implemented | `term app check <name>` loads app directories through `internal/appruntime`. |
| Manifest validation | Implemented | package load rejects invalid manifest shape and unsupported language declarations. |
| Exported definitions | Partially implemented | manifests declare exports and packages register exported app definitions, but TAL `match` bodies are not interpreted. |
| TAL parsing/interpretation | Planned | TAL files are present as contract examples; lifecycle hooks are not executed by an interpreter. |
| Lifecycle state snapshots | Planned | `Result.State` is the target contract; durable snapshots are not committed for interpreted TAL activations yet. |
| Host operation commit | Partially implemented | Go `ResultScenario` activations can return UI, scheduler, TTS, broadcast, and bus operations committed by `internal/scenario.ExecuteOperations`; TAL results can be adapted to the same model, but interpreted TAL lifecycle hooks are not wired end-to-end yet. |
| Simulation harness | Partially implemented | `term app test` smoke-tests packages and test declarations; full synthetic-time lifecycle simulation is planned. |

## Language Choice
TAL is a **Starlark-like**, deterministic, embeddable language hosted inside the Go server. It is intentionally small:
- No threads.
- No arbitrary sockets.
- No raw filesystem access.
- No direct clock access outside host APIs.
- No mutable global state after module load.

A TAL program reacts to typed triggers, updates serializable activation state, and returns typed operations for the kernel to execute.

## App Package Format
Each application is a directory loaded from disk by the server.

```text
terminal_server/apps/
├── audio_watch/
│   ├── manifest.toml
│   ├── main.tal
│   ├── lib/
│   │   └── helpers.tal
│   ├── tests/
│   │   └── audio_watch_test.tal
│   ├── kernels/
│   │   └── sound_loc.wasm
│   ├── models/
│   │   └── home_sounds.onnx
│   └── assets/
│       └── chime.wav
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
TAR loads an app package, validates permissions and API compatibility, compiles TAL modules, and registers one or more definitions with the scenario engine.

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

The existing [scenario engine](scenario-engine.md) remains the supervisor. TAR is the package and execution layer that feeds it definitions and activations.

## TAL Execution Model
TAL is **event-driven**, not thread-driven. A handler receives an immutable input and returns a `Result` value.

```go
type Result struct {
    State any
    Ops   []Op
    Emit  []Trigger
    Done  bool
}
```

Rules:
- One mailbox per activation.
- Events for one activation are processed serially.
- Activation state must be serializable.
- Live handles are stored as opaque refs, not raw host objects.
- A handler either commits all of its returned ops or none.
- Every commit snapshots activation state for suspend/resume and crash recovery.

## TAL Surface Area
TAL gets a small set of host modules. Everything else is hidden behind the Go kernel.

- `placement` — semantic targeting via zones, roles, nearest, broadcast.
- `claims` — resource requests and release helpers.
- `ui` — server-driven UI composition.
- `flow` — declare `FlowPlan`s. See [observation-plane.md](observation-plane.md).
- `observe` — subscribe to typed observations from audio, video, sensors, and radios.
- `recent` — request retrospective evidence windows.
- `presence` — query fused person/device/object presence state.
- `world` — spatial queries and verification workflows.
- `scheduler` — timers, reminders, recurring schedules.
- `store` — key/value and structured records.
- `telephony` — SIP and call control.
- `pty` — terminal sessions.
- `ai` — server-side STT/TTS/LLM entry points.
- `http` — outbound HTTP under explicit permission.
- `bus` — emit typed intents/events.
- `log` — structured logs.

## Permissions
An app must declare the capabilities it intends to use. TAR exposes no undeclared host service.

Core permissions:
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
Applications are designed for iterative development from attached terminals.

Rules:
- Source edits are loaded from disk with no server restart.
- New activations use the latest successfully loaded version.
- Existing activations stay pinned to the version that created them.
- A package may export `migrate(from_version, state)` for durable cross-version resume.
- A failed reload leaves the last good version active.

## Terminal-First Development Loop
The system already supports PTY-backed terminals. That becomes the authoring path for applications.

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

A development loop from any attached laptop, tablet with keyboard, or Chromebook is:
1. Open a PTY session into the server.
2. Edit TAL source or app assets.
3. Run `term app check` for schema and permission validation.
4. Run `term app test` or `term sim run`.
5. Run `term app reload`.
6. Observe logs and traces without disconnecting the terminal session.

## Packaging of Heavy Compute
TAL remains small on purpose. Performance-sensitive work does not run in TAL.

Apps may ship:
- **operator kernels** compiled to a portable runtime format for edge or server execution
- **model artifacts** for classifiers, trackers, and localizers

TAL requests observations or actions. The kernel decides where operator kernels run. See [edge-execution.md](edge-execution.md).

## Related Plans
- [scenario-engine.md](scenario-engine.md) — Activations, lifecycle, and trigger matching.
- [edge-execution.md](edge-execution.md) — Generic client-side operator hosting.
- [observation-plane.md](observation-plane.md) — `FlowPlan`, observations, artifacts, and buffers.
- [world-model-calibration.md](world-model-calibration.md) — Spatial model, entity location, and verification.
- [phase-6b-edge-sensing.md](phase-6b-edge-sensing.md) — Suggested implementation phase for this runtime extension.
