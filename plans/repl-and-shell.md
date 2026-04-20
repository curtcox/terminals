# REPL Plan

See [masterplan.md](../masterplan.md) for overall system context. See [usecases.md](../usecases.md) for the user stories this plan needs to satisfy. See [application-runtime.md](application-runtime.md) for the runtime model this plan extends.

## Design Principle

The Terminals REPL is a **server-side, control-plane, read-eval-print loop**. It is an *interactive shell* in the REPL sense — like a database CLI, a Python REPL, or a router's management console — not a Unix shell. It never gives the user access to bash, zsh, the host filesystem, or any host process on the machine the server runs on. Every command goes through typed control-plane APIs.

A client with keyboard and display capability attaches to a REPL session via the existing PTY-backed terminal activation path. The client stays a generic keyboard/display terminal. New REPL behavior ships without scenario-specific client updates.

The REPL must:

- work from any attached client terminal with a keyboard and display
- reuse the existing PTY terminal activation path rather than inventing a second transport
- let the user query and operate the system through typed APIs only
- support all existing terminal and app-development use cases
- provide built-in documentation, API references, and examples from inside the REPL
- provide an LLM-assisted mode that can answer questions, generate code, and propose Terminals commands, with human review before anything mutating runs

## Non-Goals

- No bash/zsh shell. The user never gets a Unix shell on the host.
- No direct host filesystem, process, or network access outside typed host APIs.
- No scenario-specific client logic.
- No separate network-facing remote login protocol beyond the existing trusted-LAN terminal model.
- No direct mutation of server internals outside typed control-plane APIs.
- No requirement that TAL apps get unrestricted REPL access.

## User Experience

From any attached client with keyboard and display capability, the user can:

1. Open a REPL session.
2. Run commands to inspect and operate the system.
3. Ask the LLM questions, have it generate code, or have it propose Terminals commands — reviewing and approving each mutating proposal before it executes.
4. Detach and later reattach the same live session from another client.
5. Read built-in help, examples, and API documentation without leaving the REPL.

Example session:

```text
$ term session new
Connected to repl session repl_42

repl> help
repl> devices ls
repl> activations ls
repl> claims tree
repl> app reload sound_watch
repl> docs open examples/app-dev-loop

repl> ai use ollama llama3.1
repl> ai ask why is act_42 suspended?
[auto-context: activations ls, claims tree, logs last=2m]
act_42 (photo_frame) is suspended because screen.main on hallway-screen was
preempted at 14:02:11 by act_51 (red_alert). It will resume when act_51
releases its claim on screen.main.

repl> ai gen a TAL app that rings a chime when the dryer beeps
proposed files (run `ai diff` to review, `ai gen --out apps/dryer_chime/ --write` to apply):
  apps/dryer_chime/manifest.toml
  apps/dryer_chime/main.tal
  apps/dryer_chime/tests/dryer_chime_test.tal
```

## Architecture

```text
keyboard/display client
        |
existing terminal UI + key forwarding
        |
PTY-backed terminal activation
        |
Terminals REPL interpreter (termrepl)
        |
typed control-plane APIs
        |
 registry / placement / claims / scenario engine /
 TAR / scheduler / observe / store / telephony / logs / ai
```

Split:

- **Transport/UI** — unchanged terminal transport and terminal rendering path
- **Execution** — the REPL interpreter hosted inside a PTY on the server
- **Authority** — typed server APIs; no direct internal object access
- **Documentation** — generated from the same command and API metadata the REPL uses

There is no second execution path. The PTY hosts the REPL interpreter; no login shell is ever spawned.

## Session Model

Interactive REPL access is a first-class activation type backed by a PTY.

### Session types

- **ReplSession** — the REPL interpreter inside a PTY
- **AttachedSession** — an additional client attached to an existing REPL session
- **DetachedSession** — a live REPL session preserved without a current viewer

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
- attached client set
- PTY handle ref
- REPL state snapshot: history, pinned context, LLM thread, approval policy, sticky provider/model selection, pending tool-call (if any)
- creation time, last attach time, idle state

Session state must be serializable enough to support suspend/resume and reconnect behavior consistent with the wider activation model.

## Commands

### Design rule

Every REPL command maps to a typed request/response operation in the server control plane. No command drops to a host shell. Each command in the registry declares `read_only | operational | mutating` metadata; the LLM approval pipeline uses that classification directly.

- `read_only` — pure one-shot reads with no resource hold.
- `operational` — non-mutating but resource-holding or subscription-shaped: `logs tail`, `observe tail`, `app logs -f`, and similar commands that open streams or have externally observable effects. Budget-capped per session (concurrent-stream and TTL limits) in both REPL and MCP origins; budgets are origin-blind.
- `mutating` — changes persistent server state. Requires explicit confirmation or `--force` at the REPL; MCP origins carry the approval out-of-band via elicitation or the `confirmation_id` fallback, per [agent-delegation.md](agent-delegation.md#approval-model).

The REPL command-service also exposes a streaming dispatch RPC alongside the unary `EvalCommand` for `operational`-tier commands that emit continuous output. This RPC is introduced in this file's service contract but authored and first consumed in [agent-delegation.md](agent-delegation.md#streaming); any REPL client may use it.

### Core command groups

- `help` — top-level and command-specific help
- `docs` — browse/search built-in documentation
- `devices` — registry, capabilities, liveness, placement metadata
- `sessions` — REPL session management
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
- `ai` — LLM-assisted commands (see [LLM Assistance](#llm-assistance))
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

## LLM Assistance

The `ai` command group lets the user ask questions, generate code, and have the LLM operate the system on their behalf through the same typed command surface that humans use. The LLM never executes anything directly: it proposes commands, and those proposals flow through an approval pipeline before the REPL runs them.

### Pluggable providers

AI providers sit behind the existing server-side AI interface (per [masterplan.md](../masterplan.md) core rule 3). Two providers ship initially:

- **OpenRouter** — hosted models via the OpenRouter API.
- **Ollama** — locally hosted models via a configured Ollama base URL.

Provider selection is **sticky**: `ai use <provider> <model>` sets the active model for the current session and persists across detach/reattach. A server-level default applies until overridden. API keys and base URLs are only configurable in server config — never from inside the REPL.

### Command surface

```text
ai providers                         list configured providers
ai models [provider]                 list models available from a provider
ai use <provider> <model>            sticky selection for this session
ai status                            current provider, model, thread size, policy, pinned context

ai ask <prompt>                      ask a question; LLM may call read-only tools
                                     and propose mutating ones for approval
ai gen <description>                 generate code; stdout by default
                                     --out <path> stages a single file or a bundle
                                     --write applies the staged bundle to disk
ai diff                              show diff of the staged bundle
ai run                               execute the most recent proposed mutating
                                     command (same effect as `ai approve`)
ai approve | ai reject               respond to the pending approval
ai cancel                            cancel the in-flight LLM request

ai history                           recent exchanges in this session's thread
ai reset                             clear the thread (keeps sticky selection and policy)

ai context                           show active context attached to the next prompt
ai context add <ref>                 e.g. devices:ls, claims:tree, logs:last=5m,
                                     file:apps/foo/main.tal
ai context pin <ref>                 keep <ref> across turns until unpinned
ai context unpin <ref>
ai context clear                     remove manual context (auto-context still applies)

ai policy show
ai policy set <auto-readonly|prompt-all|prompt-mutating>
                                     default: prompt-mutating
```

### Tool-use loop

When the LLM is invoked, the server exposes a tool surface built from the typed command registry. The LLM proposes tool calls one at a time. Each proposal is classified by the registry:

- **read-only** (`devices ls`, `claims tree`, `logs query`, `docs search`, `activations show`, …) — under the default policy, executes immediately through the same typed API path a human command would use; the result is returned to the model and the exchange is recorded in the session transcript.
- **mutating** (`app reload`, `app rollback`, `activations stop`, `sessions terminate`, `scheduler run`, file writes under `ai gen --out ... --write`, …) — the REPL **pauses**, prints the proposed command and its rendered arguments, and waits for `ai approve` / `ai run` / `ai reject`. On approval, the command executes through the typed API and the result is returned to the model. On rejection, the model receives a rejection note and may propose an alternative.

Policy rules:

- Default policy is `prompt-mutating`.
- `prompt-all` additionally prompts for read-only calls.
- `auto-readonly` is an alias for `prompt-mutating` and exists only for clarity.
- There is **no** `auto-mutating` option. Mutating commands always require a human-in-the-loop approval step.

### Context management

Context is auto-curated by default and user-manageable. The LLM never sees anything the user cannot inspect with `ai context`.

Auto-context on each prompt includes:

- relevant command registry metadata (what commands exist and when to use them)
- recent session transcript and command history
- small live snapshots of system state that the prompt appears to reference (devices, activations, claims, recent logs)

Manual context overrides auto-curation:

- `ai context add <ref>` — attach once for the next turn
- `ai context pin <ref>` — keep across turns
- `ai context clear` — drop all manual context

Context refs resolve through typed services (e.g., `devices:ls`, `logs:last=5m`) or through a typed file-read service scoped to the app tree (e.g., `file:apps/foo/main.tal`). There is no context ref that reaches outside the typed-service boundary.

### Code generation

`ai gen` emits code to stdout by default.

- `ai gen --out <file>` stages a single-file proposal.
- `ai gen --out <dir>/` stages a multi-file bundle (for example, a TAR app scaffold).
- `ai diff` shows a rendered diff of the staged bundle against the current filesystem.
- `ai gen ... --write` applies the staged bundle through the typed file-write service; this counts as a mutating tool call and honors the approval policy.

### Transport

`ai` commands are structured REPL commands like any other. They are NOT a separate protocol: they call `AiService` in the server, which handles provider selection, prompt construction, tool-call mediation, streaming output, and logging. The PTY carries rendered output to the client.

## API Plan

The REPL sits on typed control-plane APIs. The exact transport follows the existing protobuf/gRPC rule from the master plan.

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

### AI service

```go
type AiService interface {
    ListProviders(ctx context.Context, req ListProvidersRequest) (*ListProvidersResponse, error)
    ListModels(ctx context.Context, req ListModelsRequest) (*ListModelsResponse, error)
    GetSelection(ctx context.Context, req GetSelectionRequest) (*GetSelectionResponse, error)
    SetSelection(ctx context.Context, req SetSelectionRequest) (*SetSelectionResponse, error)

    Ask(ctx context.Context, req AskRequest) (AskStream, error)
    Gen(ctx context.Context, req GenRequest) (GenStream, error)
    Cancel(ctx context.Context, req CancelRequest) (*CancelResponse, error)

    ApproveToolCall(ctx context.Context, req ApproveToolCallRequest) (*ApproveToolCallResponse, error)
    RejectToolCall(ctx context.Context, req RejectToolCallRequest) (*RejectToolCallResponse, error)

    GetThread(ctx context.Context, req GetThreadRequest) (*GetThreadResponse, error)
    ResetThread(ctx context.Context, req ResetThreadRequest) (*ResetThreadResponse, error)

    GetContext(ctx context.Context, req GetContextRequest) (*GetContextResponse, error)
    AddContext(ctx context.Context, req AddContextRequest) (*AddContextResponse, error)
    PinContext(ctx context.Context, req PinContextRequest) (*PinContextResponse, error)
    UnpinContext(ctx context.Context, req UnpinContextRequest) (*UnpinContextResponse, error)
    ClearContext(ctx context.Context, req ClearContextRequest) (*ClearContextResponse, error)

    GetPolicy(ctx context.Context, req GetPolicyRequest) (*GetPolicyResponse, error)
    SetPolicy(ctx context.Context, req SetPolicyRequest) (*SetPolicyResponse, error)
}
```

Tool calls issued by the LLM are mediated by `AiService`. For read-only calls it dispatches through the same typed service routing used by `EvalCommand`. For mutating calls it emits a `PendingToolCall` on the session's stream, pauses the LLM turn, and resumes when `ApproveToolCall` or `RejectToolCall` is received. There is no path by which an LLM tool call reaches the host OS.

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

type AskRequest struct {
    SessionID string
    Prompt string
    ExtraContext []ContextRef
}

type PendingToolCall struct {
    ToolCallID string
    Command string
    Args map[string]any
    Classification string // read_only | mutating
    Rendered string
}

type SetSelectionRequest struct {
    SessionID string
    Provider string // openrouter | ollama
    Model string
}
```

### Server configuration

Provider credentials and endpoints live only in server config. They are never settable from the REPL.

```toml
[ai]
default_provider = "ollama"
default_model = "llama3.1"

[ai.providers.openrouter]
base_url = "https://openrouter.ai/api/v1"
api_key_env = "OPENROUTER_API_KEY"

[ai.providers.ollama]
base_url = "http://127.0.0.1:11434"
```

API keys are resolved from environment variables referenced by `api_key_env` — never from REPL input.

All authoritative system query commands bind to typed APIs for the corresponding kernel services named in [application-runtime.md](application-runtime.md): `placement`, `claims`, `ui`, `flow`, `observe`, `recent`, `presence`, `world`, `scheduler`, `store`, `telephony`, `pty`, `ai`, `bus`, and `log`.

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

ai providers
ai models ollama
ai use ollama llama3.1
ai status

ai ask what claimed screen.main on hallway-screen in the last minute?
ai ask why did act_42 get preempted?
ai gen a TAL app that rings a chime when the dryer beeps
ai gen --out apps/dryer_chime/ a TAL app that rings a chime when the dryer beeps
ai diff
ai approve
ai reject

ai context add devices:ls
ai context pin claims:tree
ai context show
ai context clear

ai policy set prompt-all
```

### Completion and discovery

The REPL supports:

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
  commands/
    devices.md
    sessions.md
    activations.md
    claims.md
    app.md
    observe.md
    logs.md
    ai.md
    docs.md
  api/
    session-service.md
    repl-service.md
    ai-service.md
    devices-service.md
    app-service.md
  examples/
    app-dev-loop.md
    inspect-preemption.md
    trace-audio-watch.md
    recover-session.md
    ai-debug-claim.md
    ai-generate-app.md
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
docs open api/AiService
docs examples app
docs examples ai
```

`help` is concise and command-oriented. `docs open` renders full topic content in terminal-friendly paged form. `docs search` searches both human-authored docs and generated API docs.

## Code Examples

### Example: app development loop

This extends the terminal-first development loop already described for TAR.

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

### Example: LLM-assisted debugging

```text
repl> ai use openrouter anthropic/claude-sonnet-4-6
provider: openrouter  model: anthropic/claude-sonnet-4-6 (sticky for repl_42)

repl> ai ask why is act_42 suspended?
[auto-context: activations ls, claims tree, logs last=2m]
act_42 (photo_frame) is suspended because screen.main on hallway-screen was
preempted at 14:02:11 by act_51 (red_alert). It will resume when act_51
releases its claim on screen.main.

repl> ai ask can you stop act_51 so the photo frame resumes?
proposed tool call (mutating):
  activations stop act_51
approve? (ai approve / ai reject)

repl> ai approve
OK  act_51 stopped
act_42 resumed on hallway-screen.
```

### Example: LLM-assisted code generation

```text
repl> ai gen --out apps/dryer_chime/ a TAL app that rings a chime when the dryer beeps
staged 3 files under apps/dryer_chime/ (run `ai diff`, then `ai gen --write` to apply)

repl> ai diff
apps/dryer_chime/manifest.toml   (new, 14 lines)
apps/dryer_chime/main.tal        (new, 42 lines)
apps/dryer_chime/tests/dryer_chime_test.tal  (new, 18 lines)

repl> ai gen --write
proposed tool call (mutating):
  file.write apps/dryer_chime/manifest.toml
  file.write apps/dryer_chime/main.tal
  file.write apps/dryer_chime/tests/dryer_chime_test.tal
approve? (ai approve / ai reject)

repl> ai approve
OK  3 files written
repl> app check dryer_chime
OK  manifest valid
OK  permissions valid
OK  TAL compile succeeded
```

## Use-Case Coverage

### Direct coverage

- **P1** — open an interactive REPL session on a laptop or Chromebook that connects to the server.
- **P2** — multiple REPL sessions on one device or one session accessed from multiple devices.
- **P3** — ask the LLM questions, generate code, and have it propose system commands with review before mutating operations execute.
- **P4** — pick the LLM provider (OpenRouter or Ollama) and model with a sticky selection.

### Operational and development coverage

The REPL helps implement or validate the rest of the architecture by letting operators and developers inspect:

- device registration and capability manifests
- placement decisions
- claim ownership and preemption chains
- scenario and app activations
- observation and evidence streams
- timers and scheduled actions
- telephony state
- runtime logs and traces
- app package validation, testing, reload, rollback, and simulation
- LLM provider and model configuration, proposals, and approvals

This does not replace user-facing scenarios. It provides the operator and development surface needed to inspect, debug, and extend them using the same server-owned architecture.

## Security and Permissions

The repo currently assumes a trusted LAN and no user auth. This plan preserves that assumption while keeping REPL behavior capability-gated and ready for future hardening.

Invariants:

- The REPL does not, and cannot, drop to a Unix shell on the host.
- Every action traverses a typed control-plane API; there is no host-exec primitive.
- Only clients with keyboard and display capability may launch or attach to REPL sessions.
- Mutating commands — whether typed by a human or proposed by the LLM — require explicit confirmation or `--force`.
- LLM tool calls are subject to the same `read_only | operational | mutating` registry classification as human-typed commands; the model cannot escalate by wording.
- `operational`-tier commands are budget-capped per session (concurrent-stream and TTL limits); caps are origin-blind and apply equally to human and MCP sessions.
- Mutating LLM-proposed commands always require a human-in-the-loop approval; there is no `auto-mutating` policy.
- Session lifecycle events, command execution, and LLM proposals/approvals/rejections are all logged in structured form.
- API credentials for AI providers live in server config only and are never exposed to the REPL or to the LLM.

## Implementation Phases

### Phase A — session substrate

- add `ReplSession` records
- reuse PTY-backed terminal activation path
- implement create/attach/detach/resize/terminate/list/get session APIs
- persist enough session metadata for reconnect and resume

### Phase B — REPL core

- implement parser, command registry (with `read_only | mutating` metadata), completion, paging, and history
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

### Phase G — LLM assistance

- implement `AiService`
- add OpenRouter and Ollama provider adapters behind the existing AI interface
- persist sticky provider/model selection on the session
- auto-context curation plus manual `ai context` commands
- `ai ask` and `ai gen` with streaming output
- tool-use loop wired to the command registry with read-only vs mutating classification
- approval pipeline (`ai approve` / `ai reject`, `ai policy set`)
- structured logging of all proposals, approvals, and rejections
- documentation topics for `ai` commands and `AiService`

## Acceptance Criteria

- a user can open a REPL session from any attached client with keyboard and display capability
- no REPL command path provides access to a Unix shell or to the host filesystem outside typed APIs
- the same live session can be detached and reattached from another client
- multiple sessions can coexist on a single device
- the REPL can query devices, activations, claims, logs, schedules, apps, observations, and configuration
- built-in help, API docs, and examples are available entirely from within the REPL
- the documentation index is generated from command/API metadata plus hand-authored topics so it stays aligned with the running build
- the user can ask the LLM questions and have it call read-only commands and propose mutating commands
- mutating LLM-proposed commands never execute without an explicit human approval
- `ai use <provider> <model>` sets a sticky provider/model selection that survives detach/reattach
- AI provider credentials are settable only in server config
- the REPL path supports the existing terminal and application-runtime development use cases without client changes

## Related Plans

- [masterplan.md](../masterplan.md) — overall architecture and client/server rules
- [usecases.md](../usecases.md) — user stories, especially P1–P4
- [phase-2-terminal.md](phase-2-terminal.md) — PTY-backed text terminal foundation
- [scenario-engine.md](scenario-engine.md) — activation model, lifecycle, claims, suspend/resume
- [application-runtime.md](application-runtime.md) — TAR/TAL runtime, `pty` host module, terminal-first development loop
- [agent-delegation.md](agent-delegation.md) — exposing the REPL command surface to Claude Code / Codex desktop apps via MCP
