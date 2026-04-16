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

## App Runtime (`appruntime`)
- `appruntime.package.loaded`
- `appruntime.package.skipped`

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

## Housekeeping (`housekeeping`)
- `housekeeping.configured`
- `housekeeping.due_timers.processed`
- `housekeeping.due_timers.failed`
- `housekeeping.liveness.reconciled`
