# REPL and Interactive Shell Plan

See [masterplan.md](../masterplan.md) for overall system context. See [usecases.md](../usecases.md) for the user stories this plan needs to satisfy. See [application-runtime.md](application-runtime.md) for the runtime model this plan extends.

## Design Principle

Interactive shell and REPL access are **server-side control-plane sessions** exposed through the existing PTY-backed terminal path. The client remains a generic keyboard/display terminal. New shell and REPL behavior must ship without scenario-specific client updates.

The shell/REPL must:

- work from any attached client terminal with a keyboard and display
- reuse the existing PTY terminal activation path rather than inventing a second terminal transport
- let the user query the system and operate the runtime
- support all existing terminal and app-development use cases
- provide built-in documentation, API references, and examples from inside the REPL

## Goals

- Fulfill **P1** by letting a developer or power user open a text terminal from any attached laptop, Chromebook, or similar keyboard/display client into the server shell.
- Fulfill **P2** by supporting multiple sessions per device and attach/detach across devices without losing context.
- Extend the terminal path from a raw shell into a structured control-plane REPL that can inspect and operate the system.
- Preserve the terminal-first development loop described in [application-runtime.md](application-runtime.md).
- Keep the shell/REPL aligned with the typed runtime architecture rather than bypassing it with ad hoc scripts.

## Non-Goals

- No scenario-specific client logic.
- No separate network-facing remote login protocol beyond the existing trusted-LAN terminal model.
- No direct mutation of server internals outside typed control-plane APIs.
- No requirement that TAL apps get unrestricted shell access.

## User Experience

From any attached client with keyboard and display capability, the user can:

1. Open a terminal session.
2. Choose a session mode:
   - `shell` — raw PTY shell on the server
   - `repl` — structured Terminals REPL inside a PTY
3. Run commands to inspect and operate the system.
4. Detach and later reattach the same live session from another client.
5. Read built-in help, examples, and API documentation without leaving the REPL.

Example session:

```text
$ term session new --mode repl
Connected to repl session repl_42

repl> help
repl> devices ls
repl> activations ls
repl> claims tree
repl> app reload sound_watch
repl> docs open examples/app-dev-loop
```

## Architecture

```text
keyboard/display client
        |
existing terminal UI + key forwarding
        |
PTY-backed terminal activation
        |
termsh (server process)
   |                |
raw shell         Terminals REPL
                    |
             typed control-plane APIs
                    |
 registry / placement / claims / scenario engine /
 TAR / scheduler / observe / store / telephony / logs
```

The important split is:

- **Transport/UI** — unchanged terminal transport and terminal rendering path
- **Execution** — PTY-hosted shell or REPL process on the server
- **Authority** — typed server APIs, not direct internal object access
- **Documentation** — generated from the same command and API metadata the REPL uses

This keeps the design consistent with the master plan rule that the client stays generic and the server owns behavior.

## Session Model

Interactive terminal access becomes a first-class activation type backed by PTY resources.

### Session types

- **ShellSession** — raw shell inside a PTY
- **ReplSession** — structured REPL inside a PTY
- **AttachedSession** — additional client attached to an existing session
- **DetachedSession** — live session preserved without a current viewer

### Lifecycle

- `CreateSession`
- `AttachSession`
- `DetachSession`
- `ResizeSession`
- `SendInput`
- `TerminateSession`
- `ListSessions`
- `GetSession`

Each session has:

- stable session ID
- owner activation ID
- mode (`shell` or `repl`)
- attached client set
- PTY handle ref
- current working directory or REPL state snapshot
- environment profile
- creation time, last attach time, and idle state

Session state must be serializable enough to support suspend/resume and reconnect behavior consistent with the wider activation model.

## Shell Mode

Shell mode provides the direct P1 path.

### Requirements

- launch a login shell inside a PTY
- map keyboard input and terminal resize events from the client
- support detach/reattach
- support multiple concurrent shell sessions
- record lifecycle events in structured logs
- be explicitly configurable in trusted-LAN deployments

### Intended use

- low-level server administration
- editing files directly on the server
- running local tools
- using existing command-line workflows that are not yet promoted into the structured REPL

## REPL Mode

REPL mode is the structured control-plane console for the system.

### Design rule

Every authoritative REPL command maps to a typed request/response operation in the server control plane. The REPL may delegate to the shell for convenience commands, but core system state must come from typed services.

### Core command groups

- `help` — top-level and command-specific help
- `docs` — browse/search built-in documentation
- `devices` — registry, capabilities, liveness, placement metadata
- `sessions` — shell/REPL session management
- `activations` — live scenario/application activations
- `claims` — resource claims, suspension, preemption, resume chains
- `ui` — active UI descriptors and patch state
- `flow` — active flow plans and routing state
- `observe` — observations and recent evidence windows
- `presence` — fused presence state
- `world` — spatial and world-model queries
- `scheduler` — timers, reminders, schedules, queued jobs
- `app` — TAR apps: list, check, test, load, reload, rollback, logs, trace
- `logs` — structured event-log query and tail
- `telephony` — call state and bridge status
- `ai` — provider status and request diagnostics
- `config` — effective configuration with source provenance
- `term` — compatibility surface for existing terminal-first commands

### Human mode

Readable table-oriented output:

```text
repl> devices ls
ID              ZONE      CAPS                      STATE
kitchen-tablet  kitchen   display mic speaker       online
office-laptop   office    display keyboard mic      online

repl> activations ls
ID         DEF            STATE      PRIORITY  TARGETS
act_42     photo_frame    active     low       hallway-screen
act_51     terminal       active     normal    office-laptop
```

### Script mode

Stable machine-readable output:

```text
repl> devices ls --json
repl> activations ls --json
repl> claims tree --yaml
repl> docs open api/ReplService --format markdown
```

## API Plan

The REPL sits on typed control-plane APIs. The exact transport can follow the existing protobuf/gRPC rule from the master plan.

### Session service

```go
type SessionService interface {
    CreateSession(ctx context.Context, req CreateSessionRequest) (*CreateSessionResponse, error)
    AttachSession(ctx context.Context, req AttachSessionRequest) (*AttachSessionResponse, error)
    DetachSession(ctx context.Context, req DetachSessionRequest) (*DetachSessionResponse, error)
    ResizeSession(ctx context.Context, req ResizeSessionRequest) (*ResizeSessionResponse, error)
    SendInput(ctx context.Context, req SendInputRequest) (*SendInputResponse, error)
    TerminateSession(ctx context.Context, req TerminateSessionRequest) (*TerminateSessionResponse, error)
    ListSessions(ctx context.Context, req ListSessionsRequest) (*ListSessionsResponse, error)
    GetSession(ctx context.Context, req GetSessionRequest) (*GetSessionResponse, error)
}
```

### REPL service

```go
type ReplService interface {
    EvalCommand(ctx context.Context, req EvalCommandRequest) (*EvalCommandResponse, error)
    Complete(ctx context.Context, req CompleteRequest) (*CompleteResponse, error)
    DescribeCommand(ctx context.Context, req DescribeCommandRequest) (*DescribeCommandResponse, error)
    SearchDocs(ctx context.Context, req SearchDocsRequest) (*SearchDocsResponse, error)
    GetDocTopic(ctx context.Context, req GetDocTopicRequest) (*GetDocTopicResponse, error)
    ListExamples(ctx context.Context, req ListExamplesRequest) (*ListExamplesResponse, error)
}
```

### Example request/response contracts

```go
type EvalCommandRequest struct {
    SessionID string
    Input string
    TTYWidth int
    TTYHeight int
}

type EvalCommandResponse struct {
    Output []ReplChunk
    ExitCode int
    Suggestions []Suggestion
    RelatedDocs []DocRef
}
```

```go
type CreateSessionRequest struct {
    Mode string // shell | repl
    TargetDevice string
    Attach bool
    ReadOnly bool
}

type SessionSummary struct {
    SessionID string
    Mode string
    State string
    OwnerActivationID string
    AttachedClients []string
    CreatedAt string
}
```

All authoritative system query commands should bind to typed APIs for the corresponding kernel services named in [application-runtime.md](application-runtime.md): `placement`, `claims`, `ui`, `flow`, `observe`, `recent`, `presence`, `world`, `scheduler`, `store`, `telephony`, `pty`, `ai`, `bus`, and `log`.

## Command Surface

### Top-level examples

```text
help
help devices
help app reload

devices ls
devices show <device>
devices where zone=kitchen capability=display

sessions ls
sessions show <session>
sessions attach <session>
sessions detach <session>
sessions terminate <session>

activations ls
activations show <activation>
activations stop <activation>

claims tree
claims show <resource-or-activation>

app ls
app show <app>
app check <app>
app test <app>
app load <app>
app reload <app>
app rollback <app>
app logs <app>
app trace <app>

observe tail <kind>
observe recent <kind> --window 30s

scheduler ls
scheduler show <job>
scheduler run <job>

logs tail
logs query 'kind == "session.created"'

docs ls
docs search <query>
docs open <topic>
docs examples <topic>
```

### Completion and discovery

The REPL should support:

- command and subcommand completion
- argument completion where sensible
- `help <command>` for concise command help
- `describe <symbol>` for a richer API/command description
- related-command and related-doc suggestions after execution errors

## Documentation Plan

Documentation is a first-class deliverable.

### Requirements

- document REPL concepts and session lifecycle
- document command groups and command syntax
- document the underlying control-plane APIs
- include runnable examples for common workflows
- make all of the above accessible from inside the REPL
- generate as much as possible from typed metadata so docs stay version-aligned with the running build

### Source layout

```text
docs/repl/
  index.md
  quickstart.md
  sessions.md
  modes.md
  commands/
    devices.md
    sessions.md
    activations.md
    claims.md
    app.md
    observe.md
    logs.md
    docs.md
  api/
    session-service.md
    repl-service.md
    devices-service.md
    app-service.md
  examples/
    app-dev-loop.md
    inspect-preemption.md
    trace-audio-watch.md
    recover-session.md
```

### In-REPL access mechanism

The REPL exposes a documentation index built at server startup from:

- command registry metadata
- service/API schema metadata
- hand-authored markdown topics
- example catalog metadata

REPL access commands:

```text
help
help app reload
docs ls
docs search preemption
docs open claims/tree
docs open api/ReplService
docs examples app
```

`help` should be concise and command-oriented. `docs open` should render full topic content in terminal-friendly paged form. `docs search` should search both human-authored docs and generated API docs.

## Code Examples

### Example: app development loop

This extends the terminal-first development loop already described for TAR.

```bash
term app new sound_watch
term app check sound_watch
term app test sound_watch
term app load sound_watch
term app reload sound_watch
term app logs sound_watch
term app trace sound_watch
term sim run sound_watch --fixture kitchen_house.yaml
```

Equivalent REPL usage:

```text
repl> app check sound_watch
OK  manifest valid
OK  permissions valid
OK  TAL compile succeeded

repl> app test sound_watch
PASS  audio_watch_test.tal

repl> app reload sound_watch
OK  reloaded version 0.4.3
```

### Example: claims and activations

```text
repl> activations ls
ID         DEF            STATE      PRIORITY  TARGETS
act_42     photo_frame    active     low       hallway-screen
act_51     terminal       active     normal    office-laptop

repl> claims tree
office-laptop:
  screen.main   -> act_51 (terminal)
  keyboard.main -> act_51 (terminal)
hallway-screen:
  screen.main   -> act_42 (photo_frame)
```

### Example: documentation from inside the REPL

```text
repl> help observe recent
observe recent <kind> --window <duration>

Return retrospective evidence windows for a typed observation kind.

Examples:
  observe recent sound.detected --window 30s
  observe recent vision.motion --window 2m

See also:
  docs open api/ObserveService
  docs open examples/trace-audio-watch
```

## Use-Case Coverage

### Direct coverage

- **P1** — open a text terminal on a laptop or Chromebook that connects to the server shell.
- **P2** — have multiple terminal sessions on one device or access one session from multiple devices.

### Operational and development coverage

The REPL also helps implement or validate the rest of the architecture by giving operators and developers a way to inspect:

- device registration and capability manifests
- placement decisions
- claim ownership and preemption chains
- scenario and app activations
- observation and evidence streams
- timers and scheduled actions
- telephony state
- runtime logs and traces
- app package validation, testing, reload, rollback, and simulation

This does not replace user-facing scenarios. It provides the operator and development surface needed to inspect, debug, and extend them using the same server-owned architecture.

## Security and Permissions

The repo currently assumes a trusted LAN and no user auth. This plan should preserve that assumption while still keeping shell and REPL behavior capability-gated and ready for future hardening.

### Initial rules

- only clients with keyboard and display capability may launch or attach to shell/REPL sessions
- shell mode is configurable and can be disabled independently of REPL mode
- REPL commands execute through typed services that can enforce permissions later
- destructive commands require explicit confirmation or `--force`
- session lifecycle events and command execution are logged in structured form

## Implementation Phases

### Phase A — session substrate

- add `ShellSession` and `ReplSession` records
- reuse PTY-backed terminal activation path
- implement create/attach/detach/resize/terminate/list/get session APIs
- persist enough session metadata for reconnect and resume

### Phase B — REPL core

- implement parser, command registry, completion, paging, and history
- support human-readable and machine-readable output formats
- add command metadata model: synopsis, args, examples, related docs
- implement `help`, `describe`, and error suggestions

### Phase C — typed introspection APIs

- add typed services for devices, activations, claims, logs, scheduler, apps, observations, and configuration
- ensure core REPL commands use typed backends rather than shelling out for authoritative state

### Phase D — documentation system

- add hand-authored REPL docs
- generate API docs from service and command metadata
- build searchable docs index at startup
- implement `docs ls/search/open/examples`

### Phase E — multi-client mobility

- support detach/reattach across devices
- add read-only secondary attach mode first
- add shared interactive mode with ownership/conflict rules

### Phase F — developer workflow integration

- expose TAR development commands through the REPL
- integrate logs, traces, simulation, reload, and rollback flows
- add tutorial-style examples for the common terminal-first authoring loop

## Acceptance Criteria

- a user can open a shell or REPL from any attached client with keyboard and display capability
- the same live session can be detached and reattached from another client
- multiple sessions can coexist on a single device
- the REPL can query devices, activations, claims, logs, schedules, apps, observations, and configuration
- built-in help, API docs, and examples are available entirely from within the REPL
- the documentation index is generated from command/API metadata plus hand-authored topics so it stays aligned with the running build
- the shell/REPL path supports the existing terminal and application-runtime development use cases without client changes

## Related Plans

- [masterplan.md](../masterplan.md) — overall architecture and client/server rules
- [usecases.md](../usecases.md) — user stories, especially P1 and P2
- [phase-2-terminal.md](phase-2-terminal.md) — PTY-backed text terminal foundation
- [scenario-engine.md](scenario-engine.md) — activation model, lifecycle, claims, suspend/resume
- [application-runtime.md](application-runtime.md) — TAR/TAL runtime, `pty` host module, terminal-first development loop
