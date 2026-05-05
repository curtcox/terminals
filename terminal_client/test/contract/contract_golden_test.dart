import 'dart:io';

import 'package:path/path.dart' as p;
import 'package:protobuf/protobuf.dart';
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';
import 'package:yaml/yaml.dart';

void main() {
  final root = _fixtureRoot();
  final manifest =
      loadYaml(File(p.join(root, 'manifest.yaml')).readAsStringSync())
          as YamlMap;
  final fixtures = manifest['fixtures'] as YamlList;

  for (final item in fixtures.cast<YamlMap>()) {
    final fixture = _Fixture.fromYaml(item);
    _runFixture(root, fixture);
  }
}

void _runFixture(String root, _Fixture fixture) {
  final bytes = File(p.join(root, fixture.file)).readAsBytesSync();
  final message = _newMessage(fixture.message)..mergeFromBuffer(bytes);

  _expectEqual(_payloadName(message), fixture.payload, '${fixture.id} payload');
  final expected = _Expected.fromYaml(
    loadYaml(File(p.join(root, fixture.expected)).readAsStringSync())
        as YamlMap,
  );
  _expectEqual(expected.message, fixture.message, '${fixture.id} message');
  _expectEqual(expected.payload, fixture.payload, '${fixture.id} expected');
  _assertMessage(message, expected.assertions);

  final second = _newMessage(fixture.message)
    ..mergeFromBuffer(message.writeToBuffer());
  _expectEqual(
      _payloadName(second), fixture.payload, '${fixture.id} round trip');
  _assertMessage(second, expected.assertions);

  if (fixture.roundTrip == 'byte_exact') {
    _expectEqual(second.writeToBuffer(), bytes, '${fixture.id} bytes');
  }
}

String _fixtureRoot() {
  final override = Platform.environment['TERMINALS_CONTRACT_FIXTURE_ROOT'];
  if (override != null && override.isNotEmpty) {
    return override;
  }
  return p.normalize(p.join('..', 'api', 'testdata', 'contract'));
}

GeneratedMessage _newMessage(String typeName) {
  switch (typeName) {
    case 'terminals.control.v1.ConnectRequest':
      return ConnectRequest();
    case 'terminals.control.v1.ConnectResponse':
      return ConnectResponse();
    case 'terminals.control.v1.WireEnvelope':
      return WireEnvelope();
    default:
      throw UnsupportedError('unsupported contract message type $typeName');
  }
}

String _payloadName(GeneratedMessage message) {
  if (message is ConnectRequest) {
    return _snakeCase(message.whichPayload().name);
  }
  if (message is ConnectResponse) {
    return _snakeCase(message.whichPayload().name);
  }
  if (message is WireEnvelope) {
    return _snakeCase(message.whichPayload().name);
  }
  throw UnsupportedError(
      'payload oneof unsupported for ${message.runtimeType}');
}

String _snakeCase(String value) {
  return value
      .replaceAllMapped(RegExp(r'(?<=[a-z0-9])[A-Z]'), (m) => '_${m[0]}')
      .toLowerCase();
}

void _assertMessage(
  GeneratedMessage message,
  List<_Assertion> assertions,
) {
  for (final assertion in assertions) {
    if (assertion.present != null) {
      _expectEqual(_pathPresent(message, assertion.path), assertion.present,
          assertion.path);
      continue;
    }
    if (assertion.absent != null) {
      _expectEqual(!_pathPresent(message, assertion.path), assertion.absent,
          assertion.path);
      continue;
    }
    final actual = _valueAtPath(message, assertion.path);
    if (assertion.length != null) {
      _expectEqual((actual as List).length, assertion.length, assertion.path);
    } else if (assertion.contains.isNotEmpty) {
      final list = actual as List;
      for (final expected in assertion.contains) {
        if (!list.contains(expected)) {
          throw StateError(
            '${assertion.path}: expected <$list> to contain <$expected>.',
          );
        }
      }
    } else {
      _expectEqual(actual, assertion.equals, assertion.path);
    }
  }
}

bool _pathPresent(GeneratedMessage message, String path) {
  switch (path) {
    case 'hello.identity':
      return (message as ConnectRequest).hello.hasIdentity();
    case 'command':
      return (message as ConnectRequest).hasCommand();
    case 'set_ui.root.children[1].button':
      return (message as ConnectResponse).setUi.root.children[1].hasButton();
    case 'set_ui.root.children[1].text':
      return (message as ConnectResponse).setUi.root.children[1].hasText();
    default:
      _valueAtPath(message, path);
      return true;
  }
}

void _expectEqual(Object? actual, Object? expected, String context) {
  if (actual is List && expected is List) {
    if (actual.length != expected.length) {
      throw StateError('$context: expected <$expected>, got <$actual>.');
    }
    for (var i = 0; i < actual.length; i++) {
      if (actual[i] != expected[i]) {
        throw StateError('$context: expected <$expected>, got <$actual>.');
      }
    }
    return;
  }
  if (actual != expected) {
    throw StateError('$context: expected <$expected>, got <$actual>.');
  }
}

Object? _valueAtPath(GeneratedMessage message, String path) {
  switch (path) {
    case 'hello.device_id':
      return (message as ConnectRequest).hello.deviceId;
    case 'hello.identity.device_name':
      return (message as ConnectRequest).hello.identity.deviceName;
    case 'hello.identity.device_type':
      return (message as ConnectRequest).hello.identity.deviceType;
    case 'hello.identity.platform':
      return (message as ConnectRequest).hello.identity.platform;
    case 'hello.client_version':
      return (message as ConnectRequest).hello.clientVersion;
    case 'capability_snapshot.device_id':
      return (message as ConnectRequest).capabilitySnapshot.deviceId;
    case 'capability_snapshot.generation':
      return (message as ConnectRequest).capabilitySnapshot.generation.toInt();
    case 'capability_snapshot.capabilities.screen.width':
      return (message as ConnectRequest)
          .capabilitySnapshot
          .capabilities
          .screen
          .width;
    case 'capability_snapshot.capabilities.screen.height':
      return (message as ConnectRequest)
          .capabilitySnapshot
          .capabilities
          .screen
          .height;
    case 'capability_snapshot.capabilities.screen.orientation':
      return (message as ConnectRequest)
          .capabilitySnapshot
          .capabilities
          .screen
          .orientation;
    case 'capability_snapshot.capabilities.keyboard.layout':
      return (message as ConnectRequest)
          .capabilitySnapshot
          .capabilities
          .keyboard
          .layout;
    case 'capability_snapshot.capabilities.pointer.type':
      return (message as ConnectRequest)
          .capabilitySnapshot
          .capabilities
          .pointer
          .type;
    case 'capability_snapshot.capabilities.touch.max_points':
      return (message as ConnectRequest)
          .capabilitySnapshot
          .capabilities
          .touch
          .maxPoints;
    case 'capability_snapshot.capabilities.speakers.endpoints[0].endpoint_id':
      return (message as ConnectRequest)
          .capabilitySnapshot
          .capabilities
          .speakers
          .endpoints
          .first
          .endpointId;
    case 'capability_snapshot.capabilities.microphone.endpoints[0].endpoint_id':
      return (message as ConnectRequest)
          .capabilitySnapshot
          .capabilities
          .microphone
          .endpoints
          .first
          .endpointId;
    case 'capability_snapshot.capabilities.camera.endpoints[0].modes[0].width':
      return (message as ConnectRequest)
          .capabilitySnapshot
          .capabilities
          .camera
          .endpoints
          .first
          .modes
          .first
          .width;
    case 'capability_snapshot.capabilities.sensors.accelerometer':
      return (message as ConnectRequest)
          .capabilitySnapshot
          .capabilities
          .sensors
          .accelerometer;
    case 'capability_snapshot.capabilities.connectivity.bluetooth_version':
      return (message as ConnectRequest)
          .capabilitySnapshot
          .capabilities
          .connectivity
          .bluetoothVersion;
    case 'capability_snapshot.capabilities.battery.charging':
      return (message as ConnectRequest)
          .capabilitySnapshot
          .capabilities
          .battery
          .charging;
    case 'capability_snapshot.capabilities.edge.runtimes':
      return (message as ConnectRequest)
          .capabilitySnapshot
          .capabilities
          .edge
          .runtimes;
    case 'capability_snapshot.capabilities.edge.operators':
      return (message as ConnectRequest)
          .capabilitySnapshot
          .capabilities
          .edge
          .operators;
    case 'capability_snapshot.capabilities.displays':
      return (message as ConnectRequest)
          .capabilitySnapshot
          .capabilities
          .displays;
    case 'capability_snapshot.capabilities.haptics.vibration':
      return (message as ConnectRequest)
          .capabilitySnapshot
          .capabilities
          .haptics
          .vibration;
    case 'capability_delta.device_id':
      return (message as ConnectRequest).capabilityDelta.deviceId;
    case 'capability_delta.generation':
      return (message as ConnectRequest).capabilityDelta.generation.toInt();
    case 'capability_delta.reason':
      return (message as ConnectRequest).capabilityDelta.reason;
    case 'capability_delta.capabilities.screen.width':
      return (message as ConnectRequest)
          .capabilityDelta
          .capabilities
          .screen
          .width;
    case 'capability_delta.capabilities.screen.height':
      return (message as ConnectRequest)
          .capabilityDelta
          .capabilities
          .screen
          .height;
    case 'capability_delta.capabilities.screen.orientation':
      return (message as ConnectRequest)
          .capabilityDelta
          .capabilities
          .screen
          .orientation;
    case 'capability_delta.capabilities.screen.safe_area.top':
      return (message as ConnectRequest)
          .capabilityDelta
          .capabilities
          .screen
          .safeArea
          .top;
    case 'capability_delta.capabilities.screen.safe_area.bottom':
      return (message as ConnectRequest)
          .capabilityDelta
          .capabilities
          .screen
          .safeArea
          .bottom;
    case 'set_ui.device_id':
      return (message as ConnectResponse).setUi.deviceId;
    case 'set_ui.root.id':
      return (message as ConnectResponse).setUi.root.id;
    case 'set_ui.root.widget':
      return _snakeCase(
          (message as ConnectResponse).setUi.root.whichWidget().name);
    case 'set_ui.root.children':
      return (message as ConnectResponse).setUi.root.children;
    case 'set_ui.root.children[0].widget':
      return _snakeCase(
        (message as ConnectResponse).setUi.root.children[0].whichWidget().name,
      );
    case 'set_ui.root.children[0].text.value':
      return (message as ConnectResponse).setUi.root.children[0].text.value;
    case 'set_ui.root.children[0].text.style':
      return (message as ConnectResponse).setUi.root.children[0].text.style;
    case 'set_ui.root.children[0].text.color':
      return (message as ConnectResponse).setUi.root.children[0].text.color;
    case 'set_ui.root.children[1].widget':
      return _snakeCase(
        (message as ConnectResponse).setUi.root.children[1].whichWidget().name,
      );
    case 'set_ui.root.children[1].button.action':
      return (message as ConnectResponse).setUi.root.children[1].button.action;
    case 'input.device_id':
      return (message as ConnectRequest).input.deviceId;
    case 'input.ui_action.component_id':
      return (message as ConnectRequest).input.uiAction.componentId;
    case 'input.ui_action.action':
      return (message as ConnectRequest).input.uiAction.action;
    case 'input.ui_action.value':
      return (message as ConnectRequest).input.uiAction.value;
    case 'command.request_id':
      return (message as ConnectRequest).command.requestId;
    case 'command.device_id':
      return (message as ConnectRequest).command.deviceId;
    case 'command.action':
      return (message as ConnectRequest).command.action.name;
    case 'command.kind':
      return (message as ConnectRequest).command.kind.name;
    case 'command.text':
      return (message as ConnectRequest).command.text;
    case 'command.intent':
      return (message as ConnectRequest).command.intent;
    case 'command.arguments[activation_id]':
      return (message as ConnectRequest).command.arguments['activation_id'];
    case 'command.arguments[device_ids]':
      return (message as ConnectRequest).command.arguments['device_ids'];
    case 'command.typed_arguments':
      return (message as ConnectRequest).command.typedArguments;
    case 'command.typed_arguments[0].key':
      return (message as ConnectRequest).command.typedArguments[0].key;
    case 'command.typed_arguments[0].value.kind':
      return _snakeCase((message as ConnectRequest)
          .command
          .typedArguments[0]
          .value
          .whichKind()
          .name);
    case 'command.typed_arguments[0].value.string_value':
      return (message as ConnectRequest)
          .command
          .typedArguments[0]
          .value
          .stringValue;
    case 'command.typed_arguments[1].key':
      return (message as ConnectRequest).command.typedArguments[1].key;
    case 'command.typed_arguments[1].value.kind':
      return _snakeCase((message as ConnectRequest)
          .command
          .typedArguments[1]
          .value
          .whichKind()
          .name);
    case 'command.typed_arguments[1].value.string_list_value.values':
      return (message as ConnectRequest)
          .command
          .typedArguments[1]
          .value
          .stringListValue
          .values;
    case 'command.typed_arguments[2].key':
      return (message as ConnectRequest).command.typedArguments[2].key;
    case 'command.typed_arguments[2].value.bool_value':
      return (message as ConnectRequest)
          .command
          .typedArguments[2]
          .value
          .boolValue;
    case 'command.typed_arguments[3].key':
      return (message as ConnectRequest).command.typedArguments[3].key;
    case 'command.typed_arguments[3].value.int64_value':
      return (message as ConnectRequest)
          .command
          .typedArguments[3]
          .value
          .int64Value
          .toInt();
    case 'protocol_version':
      return (message as WireEnvelope).protocolVersion;
    case 'session_id':
      return (message as WireEnvelope).sessionId;
    case 'sequence':
      return (message as WireEnvelope).sequence.toInt();
    case 'transport_hello.protocol_version':
      return (message as WireEnvelope).transportHello.protocolVersion;
    case 'transport_hello.supported_carriers':
      return (message as WireEnvelope)
          .transportHello
          .supportedCarriers
          .map((carrier) => carrier.name)
          .toList();
    case 'transport_hello.desired_device_id':
      return (message as WireEnvelope).transportHello.desiredDeviceId;
    case 'transport_hello_ack.accepted_protocol_version':
      return (message as WireEnvelope)
          .transportHelloAck
          .acceptedProtocolVersion;
    case 'transport_hello_ack.negotiated_carrier':
      return (message as WireEnvelope).transportHelloAck.negotiatedCarrier.name;
    case 'transport_hello_ack.session_id':
      return (message as WireEnvelope).transportHelloAck.sessionId;
    case 'transport_hello_ack.resume_token':
      return (message as WireEnvelope).transportHelloAck.resumeToken;
    case 'transport_hello_ack.heartbeat_interval_ms':
      return (message as WireEnvelope)
          .transportHelloAck
          .heartbeatIntervalMs
          .toInt();
    case 'transport_hello_ack.limits[max_frame_bytes]':
      return (message as WireEnvelope)
          .transportHelloAck
          .limits['max_frame_bytes'];
    case 'transport_hello_ack.limits[max_inflight_messages]':
      return (message as WireEnvelope)
          .transportHelloAck
          .limits['max_inflight_messages'];
    case 'transport_hello_ack.limits[heartbeat_interval_ms]':
      return (message as WireEnvelope)
          .transportHelloAck
          .limits['heartbeat_interval_ms'];
    default:
      throw UnsupportedError('unsupported assertion path $path');
  }
}

class _Fixture {
  _Fixture({
    required this.id,
    required this.file,
    required this.message,
    required this.payload,
    required this.roundTrip,
    required this.expected,
  });

  factory _Fixture.fromYaml(YamlMap yaml) => _Fixture(
        id: yaml['id'] as String,
        file: yaml['file'] as String,
        message: yaml['message'] as String,
        payload: yaml['payload'] as String,
        roundTrip: yaml['round_trip'] as String,
        expected: yaml['expected'] as String,
      );

  final String id;
  final String file;
  final String message;
  final String payload;
  final String roundTrip;
  final String expected;
}

class _Expected {
  _Expected({
    required this.message,
    required this.payload,
    required this.assertions,
  });

  factory _Expected.fromYaml(YamlMap yaml) => _Expected(
        message: yaml['message'] as String,
        payload: yaml['payload'] as String,
        assertions: (yaml['assertions'] as YamlList)
            .cast<YamlMap>()
            .map(_Assertion.fromYaml)
            .toList(),
      );

  final String message;
  final String payload;
  final List<_Assertion> assertions;
}

class _Assertion {
  _Assertion({
    required this.path,
    required this.equals,
    required this.contains,
    required this.length,
    required this.present,
    required this.absent,
  });

  factory _Assertion.fromYaml(YamlMap yaml) => _Assertion(
        path: yaml['path'] as String,
        equals: yaml['equals'],
        contains: yaml['contains'] == null
            ? const []
            : List<Object?>.from(yaml['contains'] as YamlList),
        length: yaml['length'] as int?,
        present: yaml['present'] as bool?,
        absent: yaml['absent'] as bool?,
      );

  final String path;
  final Object? equals;
  final List<Object?> contains;
  final int? length;
  final bool? present;
  final bool? absent;
}
