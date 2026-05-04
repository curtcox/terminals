import 'dart:io';

import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';

void main() {
  final cases = <String, void Function(WireEnvelope)>{
    'hello_snapshot_v1': _assertHelloSnapshot,
    'capability_snapshot_v1': _assertCapabilitySnapshot,
    'register_ack_metadata_v1': _assertRegisterAckMetadata,
    'set_ui_basic_v1': _assertSetUIBasic,
    'start_stream_audio_v1': _assertStartStreamAudio,
    'flow_plan_basic_v1': _assertFlowPlanBasic,
    'observation_sound_v1': _assertObservationSound,
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

void _assertStartStreamAudio(WireEnvelope envelope) {
  final stream = envelope.serverMessage.startStream;

  _expectEqual(stream.kind, 'audio');
  _expectEqual(stream.streamKind, StreamKind.STREAM_KIND_AUDIO);
  _expectEqual(stream.metadata['sample_rate'], '16000');
}

void _assertFlowPlanBasic(WireEnvelope envelope) {
  final plan = envelope.serverMessage.startFlow.plan;

  _expectEqual(plan.nodes.length, 2);
  _expectEqual(plan.edges.length, 1);
  _expectEqual(plan.nodes.first.args['stream_id'], 'stream-audio-1');
}

void _assertObservationSound(WireEnvelope envelope) {
  final observation = envelope.clientMessage.observationMessage.observation;

  _expectEqual(observation.kind, 'sound.detected');
  _expectEqual(observation.attributes['loudness_db'], '72.5');
  _expectEqual(observation.evidence.length, 1);
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
