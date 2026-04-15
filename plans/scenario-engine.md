# Scenario Engine

See [masterplan.md](../masterplan.md) for overall system context.

Scenarios are server-side modules that implement specific behaviors. They are the only place where "what the system does" is defined.

## Definitions vs Activations

A scenario has two layers:

- A **ScenarioDefinition** is a singleton registered at startup. It knows how to match triggers and construct activations. It holds no mutable per-run state.
- A **ScenarioActivation** is a live instance of a scenario. Many can be active at once — multiple timers, multiple terminal sessions, multiple simultaneous calls. Each activation has its own ID, lifecycle, claims, and snapshot.

```go
type ScenarioDefinition interface {
    Name() string
    Match(req ActivationRequest) bool
    NewActivation(req ActivationRequest) (ScenarioActivation, error)
}

type ScenarioActivation interface {
    ID() string
    DefinitionName() string
    Start(ctx context.Context, env *Environment) error
    Stop(ctx context.Context, env *Environment) error
    Suspend(ctx context.Context, env *Environment) error
    Resume(ctx context.Context, env *Environment) error
}

type ActivationRequest struct {
    Trigger     Trigger
    Targets     []DeviceRef    // optional pre-resolved targets
    RequestedAt time.Time
}

type ActivationRecord struct {
    ActivationID string
    Definition   string
    Trigger      Trigger
    Targets      []DeviceRef
    Claims       []Claim
    State        ActivationState   // Pending | Active | Suspended | Stopped
    Snapshot     any               // serializable state for resume
}

type Environment struct {
    Devices    DeviceManager
    Placement  PlacementEngine
    Router     IORouter            // applies media plans
    Claims     ClaimManager
    AI         AIBackend
    Telephony  TelephonyBridge
    Storage    StorageManager
    Scheduler  Scheduler
    Broadcast  Broadcaster
}
```

This split makes multi-instance scenarios, persistence, crash recovery, and targeted suspend/resume all natural. Stop/resume address the exact activation instance rather than the scenario as a whole.

## Triggers: Intents and Events

Triggers feed the matcher. They come from voice, UI actions, schedules, external webhooks, classifiers, automation agents, and one-scenario-cascading-another. All of them collapse to two typed records on a single bus:

```go
type TriggerSource string

const (
    SourceVoice    TriggerSource = "voice"
    SourceUI       TriggerSource = "ui"
    SourceSchedule TriggerSource = "schedule"
    SourceEvent    TriggerSource = "event"
    SourceCascade  TriggerSource = "cascade"
    SourceAgent    TriggerSource = "agent"
    SourceWebhook  TriggerSource = "webhook"
)

type Intent struct {
    Action     string          // "intercom", "call", "show", "timer.set"
    Object     string          // "kitchen", "mom", "recipe"
    Slots      map[string]any
    Scope      TargetScope     // device, zone, nearest, broadcast — see placement.md
    Confidence float64
    RawText    string          // original utterance, if any
    Source     TriggerSource
}

type Event struct {
    Kind       string          // "sound.detected", "timer.due", "vision.motion"
    Subject    string          // e.g. "dryer_beep", "oven_check"
    Attributes map[string]any
    Source     TriggerSource
    OccurredAt time.Time
}

type Trigger struct {
    Intent *Intent
    Event  *Event
}
```

Voice parsers, schedulers, webhook receivers, LLM intent resolvers, and automation agents all **produce** intents or events. The scenario matcher consumes them uniformly — there is no separate voice path or schedule path at the matching layer. This also gives every trigger explicit provenance, which makes auditing and testing straightforward.

## Activation Lifecycle

1. A trigger lands on the intent/event bus.
2. The engine asks every `ScenarioDefinition.Match` whether the trigger applies.
3. Matching definitions construct activations via `NewActivation`.
4. For each activation the engine:
   a. Resolves targets via the [Placement Engine](placement.md) (`TargetScope` → `[]DeviceRef`).
   b. Requests resource [claims](#resource-claims-and-preemption) from the claim manager.
   c. If granted (possibly preempting lower-priority claims), calls `Start`.
5. Running activations may request additional claims, patch their UI, or mutate their media plan at any time.
6. On end, the engine calls `Stop`, releases claims, and restores any activations whose claims become available again.

## Resource Claims and Preemption

Preemption is enforced **per-resource**, not per-device. A device exposes many resources — main screen, overlay layer, main speaker, mic capture, camera capture, PTY, sensor taps. Scenarios claim only what they need. PA can take speakers without evicting the photo frame; a voice reply can overlay a panel without replacing the terminal; a sound monitor can tap a mic shared with other analyzers.

See [io-abstraction.md](io-abstraction.md#resource-claims) for claim types and the claim manager. The engine's role is to:

- Map activation priority onto claim priority.
- Suspend activations whose claims are revoked by a higher-priority request.
- Restore suspended activations when released claims become available again.
- Capture enough snapshot state on suspend that `Resume` faithfully recreates the prior UI and routes.

### Activation Priority

| Priority | Examples                                    |
|----------|---------------------------------------------|
| Critical | Red alert, emergency                        |
| High     | Active phone call, intercom, PA, doorbell   |
| Normal   | Terminal session, voice query, multi-window |
| Low      | Photo frame, ambient monitoring, music      |
| Idle     | Clock display, standby screen               |

Priority is advisory for arbitration; the resource kind itself dictates whether a claim is exclusive or shareable.

## Scenario Recipes

Most scenarios follow the same skeleton: resolve targets, claim resources, build a media plan, send UI, schedule cleanup, restore prior state on end. A lightweight recipe builder packages this shape so authors write only the pieces that differ:

```go
type ScenarioRecipe struct {
    ResolveTargets func(ctx context.Context, env *Environment, req ActivationRequest) ([]DeviceRef, error)
    BuildClaims    func(ctx context.Context, env *Environment, req ActivationRequest, targets []DeviceRef) ([]Claim, error)
    BuildMedia     func(ctx context.Context, env *Environment, req ActivationRequest, targets []DeviceRef) (*MediaPlan, error)
    BuildUI        func(ctx context.Context, env *Environment, req ActivationRequest, targets []DeviceRef) ([]UICommand, error)
    OnStop         func(ctx context.Context, env *Environment, rec *ActivationRecord) error
}
```

Recipes are optional — scenarios can implement `ScenarioActivation` directly — but they are the default path for anything that looks like "claim some resources, run a media plan, show UI, tear it all down."

## Engine Responsibilities

- **Registration**: Definitions register at startup.
- **Matching**: Dispatch intents and events to matching definitions.
- **Lifecycle**: Start, stop, suspend, resume activations.
- **Claim arbitration**: Work with the claim manager to grant, revoke, and restore resource claims per activation priority.
- **Target resolution**: Delegate to the placement engine.
- **State restoration**: Persist and replay enough of each activation's snapshot that suspend/resume (and crash recovery) is seamless.
- **Isolation**: Supervise activations — a crash in one must not bring down the engine.

## Related Plans

- [architecture-server.md](architecture-server.md) — Scenario, placement, claim, intent-bus, and media-planner modules.
- [io-abstraction.md](io-abstraction.md) — Claims, resource kinds, and the media-plan model.
- [placement.md](placement.md) — Semantic target resolution.
- [server-driven-ui.md](server-driven-ui.md) — How scenarios render their UIs.
- [use-case-flows.md](use-case-flows.md) — Concrete scenarios expressed against these abstractions.
