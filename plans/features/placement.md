# Placement and World Model

See [masterplan.md](../masterplan.md) for overall system context.

Scenarios reason about devices in the language of the use cases: "the kitchen", "the nearest screen", "the child's room", "all cameras", "the lobby display". The Placement Engine translates those semantic queries into concrete `DeviceRef` sets using the device registry plus zone/role metadata.

Without this layer, device-selection logic ends up duplicated in every scenario — one scenario knows how to find "the kitchen screen", another re-implements it for "all cameras". Centralizing it keeps scenarios declarative and close to the user-story language in [usecases.md](../usecases.md), and makes adding a new room or role a configuration change rather than a code change.

## Device Metadata

Each device, in addition to its raw capability manifest, carries server-assigned semantic metadata:

- **Zone**: room or area name (`kitchen`, `living_room`, `alice_room`).
- **Roles**: tags that express function (`kitchen_display`, `child_room`, `lobby_screen`, `doorbell`, `waiting_area`).
- **Mobility**: `fixed` vs `mobile` (a wall tablet vs a phone).
- **Affinity**: optional user/owner association.
- **Liveness**: `online`, `idle`, `busy` — derived from claims and heartbeats.
- **Proximity hints**: optional signal-based or user-asserted adjacency between devices.

The client never asserts its own zone or roles. The client declares capabilities; the server assigns semantic placement via admin UI, configuration, or a directory service. This keeps the thin-client contract intact and lets placement evolve without touching clients.

## Placement Queries

```go
type DeviceRef struct {
    DeviceID string
}

type TargetScope struct {
    DeviceID  string      // exact device
    Zone      string      // "kitchen"
    Role      string      // "lobby_screen"
    Nearest   bool        // nearest to Source
    Source    DeviceRef   // origin for relative queries
    Broadcast bool        // all matching devices
}

type PlacementQuery struct {
    Scope         TargetScope
    RequiredCaps  []string          // from the capability manifest
    PreferredCaps []string
    RequiredRes   []ResourceKind    // must expose these resources (see io-abstraction.md)
    ExcludeBusy   bool
    Count         int               // 0 = unlimited
}

type PlacementEngine interface {
    Find(ctx context.Context, q PlacementQuery) ([]DeviceRef, error)
    NearestWith(ctx context.Context, source DeviceRef, cap string) (DeviceRef, error)
    DevicesInZone(ctx context.Context, zone string) ([]DeviceRef, error)
    DevicesWithRole(ctx context.Context, role string) ([]DeviceRef, error)
}
```

The engine composes the device registry, zone/role configuration, and liveness hints into a single answer. Scenarios typically receive a `TargetScope` embedded in their `Intent` and call `Find` during target resolution (or let their recipe do it).

## Examples

- **"Intercom to kitchen"** → `Find({Scope: {Zone: "kitchen"}, RequiredRes: [speaker.main, mic.capture]})`
- **"Show the recipe on the nearest screen"** → `NearestWith(source, "screen")`
- **"Show all cameras"** → `Find({Scope: {Broadcast: true}, RequiredCaps: ["camera"]})`
- **"Watch my child's room"** → `DevicesInZone("alice_room")` filtered by camera + mic capability.
- **"Announce in the lobby"** → `DevicesWithRole("lobby_screen")`.

## Related Plans

- [architecture-server.md](architecture-server.md) — `internal/placement/` module layout and device metadata storage.
- [scenario-engine.md](scenario-engine.md) — How activations use `TargetScope` and the placement engine.
- [io-abstraction.md](io-abstraction.md) — Resource kinds referenced by placement queries.
- [discovery.md](discovery.md) — How devices register before placement metadata is assigned.
