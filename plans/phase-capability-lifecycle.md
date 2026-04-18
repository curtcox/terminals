# Phase X — Dynamic Capability Lifecycle

See [masterplan.md](../masterplan.md) for overall system context.

Make terminal capabilities explicit, typed, and dynamic across the client/server boundary.

This phase deliberately ignores migration and backward-compatibility concerns. There are no existing users.

## Prerequisites

- [phase-1-foundation.md](phase-1-foundation.md) complete — base client/server protocol, registration, and capability reporting exist.
- [phase-3-media.md](phase-3-media.md) complete — media routing exists and can react to resource changes.

## Deliverables

- [ ] **Explicit capability lifecycle protocol**: Replace one-shot capability registration with `Hello`, `CapabilitySnapshot`, `CapabilityDelta`, and `CapabilityAck`. See [capability-lifecycle.md](capability-lifecycle.md) and [protocol.md](protocol.md).
- [ ] **Typed capability schema**: Define strongly typed capability messages for display, keyboard, pointer, touch, speaker, microphone, camera, sensors, haptics, and battery. Avoid opaque JSON blobs.
- [ ] **Generation-based synchronization**: Add monotonically increasing capability generations so the server can reject stale deltas and clients can re-baseline with a fresh snapshot.
- [ ] **Dynamic display support**: Model display size, density, orientation, safe areas, fullscreen support, and multi-window support as explicit display capability fields. Treat runtime display changes as capability updates, not ad-hoc UI events.
- [ ] **Multi-endpoint audio/video support**: Model built-in, external, USB, HDMI, and Bluetooth microphones, speakers, and cameras as individually addressable capabilities.
- [ ] **Client capability monitor**: Add client-side detection for hot-plug peripherals, permission changes, media-route changes, and display geometry changes. Publish batched capability deltas when they occur.
- [ ] **Server capability registry**: Store the latest accepted capability snapshot per terminal, including generation, timestamps, and derived resources.
- [ ] **Capability-to-resource compiler**: Compile capability inventory into claimable resources for the claim manager and IO router.
- [ ] **Claim invalidation on resource loss**: Revoke or degrade active claims when required resources disappear or become unavailable.
- [ ] **Router patching on capability change**: Patch or tear down live media plans when their source or sink resources change.
- [ ] **Typed capability events on the bus**: Emit events such as `terminal.capability.updated`, `terminal.resource.lost`, and `terminal.display.resized` so scenarios can react uniformly.
- [ ] **Tests from the start**: Add proto, client, and server tests for snapshot/delta handling, stale generation rejection, hot-plug behavior, display resize, and claim/routing reactions.

## Exit Criteria

- A terminal always sends a full capability snapshot on initial connection.
- The server does not assign UI, routing, or scenario work until it has accepted that snapshot.
- Runtime changes to displays, IO devices, and media routes are observable as capability deltas without reconnecting.
- The server's device registry, claim manager, and IO router all converge on the same accepted capability generation.
- Loss of a required capability produces deterministic claim and routing updates rather than silent failure.
