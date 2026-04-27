# Event Taxonomy

Canonical server event names written to `logs/terminals.jsonl`.

## Lifecycle (`main`)
- `server.starting`
- `server.started`
- `server.stopping`
- `server.stopped`
- `config.loaded`

## Transport (`transport.*`)
- `transport.grpc.listener_ready`
- `transport.grpc.request.started`
- `transport.grpc.request.finished`
- `transport.grpc.stopped`
- `transport.stream.ready`

## Discovery (`discovery.mdns`)
- `discovery.mdns.failed`
- `discovery.mdns.stop_failed`

## Scenario (`scenario.*`)
- `scenario.definition.registered`
- `scenario.recovery.failed`
- `scenario.trigger.received`
- `scenario.trigger.matched`
- `scenario.trigger.unmatched`
- `scenario.activation.started`
- `scenario.activation.stopped`
- `scenario.activation.failed`
- `scenario.recovery.started`
- `scenario.recovery.finished`

## App Runtime (`appruntime`)
- `appruntime.package.loaded`
- `appruntime.package.skipped`
- `appruntime.op.emitted`

## IO / Flow (`io.router`, `io.flow`)
- `io.route.applied`
- `io.route.torn_down`
- `io.flow.started`
- `io.flow.patched`
- `io.flow.stopped`
- `io.flow.stats`
- `io.analyzer.event`

## Capability Lifecycle (`terminal.*`)
- `terminal.capability.added`
- `terminal.capability.updated`
- `terminal.capability.removed`
- `terminal.display.resized`
- `terminal.audio_route.changed`
- `terminal.resource.lost`
- `terminal.resource.lost:<resource-id>`

## Observation (`observation.store`)
- `observation.emitted`

## Recording (`recording.disk`)
- `recording.started`
- `recording.segment_flushed`

## Telephony (`telephony.sip`)
- `telephony.bridge.disabled`
- `telephony.bridge.registered`
- `telephony.bridge.failed`
- `telephony.bridge.stop_failed`
- `telephony.call.incoming`
- `telephony.call.ended`
- `telephony.media.rtp_in`
- `telephony.media.rtp_out`

## Admin (`admin.http`)
- `admin.http.listener_ready`
- `admin.http.request`
- `admin.http.ready`
- `admin.http.server_error`
- `admin.action.applied`

## Housekeeping (`housekeeping`)
- `housekeeping.configured`
- `housekeeping.due_timers.processed`
- `housekeeping.due_timers.failed`
- `housekeeping.liveness.reconciled`
