import 'dart:io';

import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';
import 'package:terminal_client/gen/terminals/io/v1/io.pb.dart';
import 'package:terminal_client/gen/terminals/io/v1/io.pbenum.dart';

void main() {
  final cases = <String, void Function(WireEnvelope)>{
    'hello_snapshot_v1': _assertHelloSnapshot,
    'capability_snapshot_v1': _assertCapabilitySnapshot,
    'register_ack_metadata_v1': _assertRegisterAckMetadata,
    'set_ui_basic_v1': _assertSetUIBasic,
    'set_ui_canvas_v1': _assertSetUICanvas,
    'start_stream_audio_v1': _assertStartStreamAudio,
    'start_stream_route_delta_v1': _assertStartStreamRouteDelta,
    'route_stream_route_delta_v1': _assertRouteStreamRouteDelta,
    'flow_plan_basic_v1': _assertFlowPlanBasic,
    'observation_sound_v1': _assertObservationSound,
    'flow_stats_v1': _assertFlowStats,
    'unknown_metadata_key_v1': _assertUnknownMetadataKey,
    'deprecated_register_device_v1': _assertDeprecatedRegisterDevice,
  };

  for (final entry in cases.entries) {
    final envelope = _readFixture(entry.key);
    WireEnvelope.fromBuffer(envelope.writeToBuffer());
    entry.value(envelope);
  }
}

WireEnvelope _readFixture(String name) {
  final file = File('../api/testdata/envelopes/$name.binpb');
  return WireEnvelope.fromBuffer(file.readAsBytesSync());
}

void _assertHelloSnapshot(WireEnvelope envelope) {
  final hello = envelope.clientMessage.hello;

  _expectEqual(hello.deviceId, 'terminal-kitchen');
  _expectEqual(hello.identity.platform, 'flutter_test');
}

void _assertCapabilitySnapshot(WireEnvelope envelope) {
  final snapshot = envelope.clientMessage.capabilitySnapshot;

  _expectEqual(snapshot.generation.toInt(), 7);
  _expectEqual(
      snapshot.capabilities.pointer.type, 'unknown_pointer_from_future');
}

void _assertRegisterAckMetadata(WireEnvelope envelope) {
  final ack = envelope.serverMessage.registerAck;
  final metadata = envelope.serverMessage.registerAck.metadata;

  _expectEqual(metadata['server_build_sha'], 'abc1234');
  _expectTrue(metadata['photo_frame_asset_base_url']?.isNotEmpty ?? false);
  _expectEqual(ack.serverMetadata.build.sha, 'abc1234');
  _expectEqual(
    ack.serverMetadata.build.dateRfc3339,
    '2026-05-03T14:00:00Z',
  );
  _expectTrue(ack.serverMetadata.photoFrameAssetBaseUrl.isNotEmpty);
}

void _assertSetUIBasic(WireEnvelope envelope) {
  final root = envelope.serverMessage.setUi.root;

  _expectEqual(root.id, 'root');
  _expectEqual(root.children.length, 2);
  _expectEqual(root.children.first.text.style, 'title');
}

void _assertSetUICanvas(WireEnvelope envelope) {
  final root = envelope.serverMessage.setUi.root;

  _expectEqual(root.children.length, 1);
  final canvas = root.children.first.canvas;
  _expectTrue(canvas.drawOpsJson.isNotEmpty);
  _expectEqual(canvas.drawOps.length, 2);
  _expectEqual(canvas.drawOps[0].rect.fill, '#abc');
  _expectEqual(canvas.drawOps[0].rect.width, 3);
  _expectEqual(canvas.drawOps[1].line.stroke, '#000');
  _expectEqual(canvas.drawOps[1].line.strokeWidth, 1.5);
}

void _assertStartStreamAudio(WireEnvelope envelope) {
  final stream = envelope.serverMessage.startStream;

  _expectEqual(stream.kind, 'audio');
  _expectEqual(stream.streamKind, StreamKind.STREAM_KIND_AUDIO);
  _expectEqual(stream.audioMetadata.sampleRate, 16000);
  _expectEqual(stream.audioMetadata.channels, 1);
  _expectEqual(stream.audioMetadata.codec, 'pcm_s16le');
  _expectEqual(stream.metadata['sample_rate'], '16000');
  _expectEqual(stream.metadata['channels'], '1');
  _expectEqual(stream.metadata['codec'], 'pcm_s16le');
}

void _assertStartStreamRouteDelta(WireEnvelope envelope) {
  final stream = envelope.serverMessage.startStream;

  _expectEqual(stream.metadata['origin'], 'route_delta');
  _expectEqual(stream.metadata['webrtc_mode'], 'server_managed');
  _expectEqual(stream.routing.origin, StreamOrigin.STREAM_ORIGIN_ROUTE_DELTA);
  _expectEqual(
      stream.routing.webrtcMode, WebRTCMode.WEB_RTC_MODE_SERVER_MANAGED);
}

void _assertRouteStreamRouteDelta(WireEnvelope envelope) {
  final route = envelope.serverMessage.routeStream;

  _expectEqual(route.kind, 'audio');
  _expectEqual(route.streamKind, StreamKind.STREAM_KIND_AUDIO);
  _expectEqual(route.routing.origin, StreamOrigin.STREAM_ORIGIN_ROUTE_DELTA);
  _expectEqual(
      route.routing.webrtcMode, WebRTCMode.WEB_RTC_MODE_SERVER_MANAGED);
}

void _assertFlowPlanBasic(WireEnvelope envelope) {
  final plan = envelope.serverMessage.startFlow.plan;

  _expectEqual(plan.nodes.length, 2);
  _expectEqual(plan.edges.length, 1);
  _expectEqual(plan.nodes.first.args['stream_id'], 'stream-audio-1');
  _expectEqual(plan.nodes[0].exec, 'edge');
  _expectEqual(plan.nodes[0].execPolicy, ExecPolicy.EXEC_POLICY_PREFER_CLIENT);
  _expectEqual(plan.nodes[1].exec, 'server');
  _expectEqual(plan.nodes[1].execPolicy, ExecPolicy.EXEC_POLICY_SERVER_ONLY);
  _expectEqual(plan.nodes[0].typedArgs.deviceId, 'kitchen-terminal');
  _expectEqual(plan.nodes[0].typedArgs.resource, 'microphone');
  _expectEqual(plan.nodes[0].typedArgs.streamKind, 'audio');
  _expectEqual(
      plan.nodes[0].typedArgs.streamKindEnum, StreamKind.STREAM_KIND_AUDIO);
  _expectEqual(plan.nodes[0].args['device_id'], 'kitchen-terminal');
}

void _assertObservationSound(WireEnvelope envelope) {
  final observation = envelope.clientMessage.observationMessage.observation;

  _expectEqual(observation.kind, 'sound.detected');
  _expectEqual(observation.attributes['loudness_db'], '72.5');
  _expectEqual(observation.attributes['label'], 'whistle');
  _expectEqual(observation.typedAttributes.label, 'whistle');
  _expectEqual(observation.typedAttributes.device, 'kettle');
  _expectEqual(observation.evidence.length, 1);
}

void _assertFlowStats(WireEnvelope envelope) {
  final stats = envelope.clientMessage.flowStats;

  _expectEqual(stats.flowId, 'flow-edge-1');
  _expectEqual(stats.state, 'running');
  _expectEqual(stats.stateEnum, FlowState.FLOW_STATE_RUNNING);
}

void _assertUnknownMetadataKey(WireEnvelope envelope) {
  final metadata = envelope.serverMessage.registerAck.metadata;

  _expectEqual(metadata['future.experimental_key'], 'preserve-but-ignore');
}

void _assertDeprecatedRegisterDevice(WireEnvelope envelope) {
  final register = envelope.clientMessage.register;

  _expectEqual(register.capabilities.deviceId, 'legacy-terminal');
}

void _expectEqual(Object? actual, Object? expected) {
  if (actual != expected) {
    throw StateError('Expected <$expected>, got <$actual>.');
  }
}

void _expectTrue(bool value) {
  if (!value) {
    throw StateError('Expected true.');
  }
}
