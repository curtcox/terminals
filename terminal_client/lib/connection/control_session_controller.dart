import 'dart:math' as math;

import 'package:fixnum/fixnum.dart';
import 'package:terminal_client/connection/control_client_factory.dart';
import 'package:terminal_client/gen/terminals/capabilities/v1/capabilities.pb.dart'
    as capv1;
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';
import 'package:terminal_client/gen/terminals/io/v1/io.pb.dart' as iov1;
import 'package:terminal_client/ui/server_driven_action.dart';

Duration calculateReconnectDelay({
  required int reconnectAttempt,
  required Duration reconnectDelayBase,
  required int reconnectDelayMaxSeconds,
}) {
  final scaledMs = reconnectDelayBase.inMilliseconds *
      math.pow(2, reconnectAttempt - 1).toInt();
  final maxMs = reconnectDelayMaxSeconds * 1000;
  final delayMs = math.min(maxMs, math.max(1, scaledMs));
  return Duration(milliseconds: delayMs);
}

class CarrierAttemptDiagnostic {
  const CarrierAttemptDiagnostic({
    required this.carrier,
    required this.endpoint,
    required this.stage,
    required this.failureClass,
    required this.error,
    required this.elapsed,
  });

  final ControlCarrierKind carrier;
  final String endpoint;
  final String stage;
  final String failureClass;
  final String error;
  final Duration elapsed;

  String get carrierLabel => controlCarrierLabel(carrier);
}

String controlCarrierLabel(ControlCarrierKind carrier) {
  switch (carrier) {
    case ControlCarrierKind.grpc:
      return 'gRPC';
    case ControlCarrierKind.websocket:
      return 'WebSocket';
    case ControlCarrierKind.tcp:
      return 'TCP';
    case ControlCarrierKind.http:
      return 'HTTP';
  }
}

String formatCarrierAttempt(CarrierAttemptDiagnostic attempt) {
  final elapsedMs = attempt.elapsed.inMilliseconds;
  return '${attempt.carrierLabel} failed at ${attempt.stage} '
      '[${attempt.failureClass}] (${attempt.endpoint}) '
      'after ${elapsedMs}ms: ${attempt.error}';
}

class ConnectionTarget {
  const ConnectionTarget({required this.host, required this.port});

  final String host;
  final int port;
}

int grpcPortFromEndpoint({
  required String endpoint,
  required int fallbackPort,
}) {
  final trimmed = endpoint.trim();
  if (trimmed.isEmpty) {
    return fallbackPort;
  }
  final parsed = Uri.tryParse('tcp://$trimmed');
  if (parsed == null || parsed.port <= 0) {
    return fallbackPort;
  }
  return parsed.port;
}

ConnectionTarget resolveConnectionTargetForCarrier({
  required ControlCarrierKind carrier,
  required String endpoint,
  required String fallbackHost,
  required int fallbackPort,
}) {
  switch (carrier) {
    case ControlCarrierKind.grpc:
      return _resolveHostPortEndpoint(
        endpoint: endpoint,
        fallbackHost: fallbackHost,
        fallbackPort: fallbackPort,
      );
    case ControlCarrierKind.websocket:
      return _resolveUriEndpoint(
        endpoint: endpoint,
        fallbackHost: fallbackHost,
        fallbackPort: fallbackPort,
      );
    case ControlCarrierKind.tcp:
      return _resolveHostPortEndpoint(
        endpoint: endpoint,
        fallbackHost: fallbackHost,
        fallbackPort: 50055,
      );
    case ControlCarrierKind.http:
      return _resolveUriEndpoint(
        endpoint: endpoint,
        fallbackHost: fallbackHost,
        fallbackPort: 50056,
      );
  }
}

ConnectionTarget _resolveHostPortEndpoint({
  required String endpoint,
  required String fallbackHost,
  required int fallbackPort,
}) {
  final trimmed = endpoint.trim();
  if (trimmed.isEmpty) {
    return ConnectionTarget(host: fallbackHost, port: fallbackPort);
  }
  final parsed = Uri.tryParse('tcp://$trimmed');
  if (parsed == null || parsed.host.isEmpty || parsed.port <= 0) {
    return ConnectionTarget(host: fallbackHost, port: fallbackPort);
  }
  return ConnectionTarget(host: parsed.host, port: parsed.port);
}

ConnectionTarget _resolveUriEndpoint({
  required String endpoint,
  required String fallbackHost,
  required int fallbackPort,
}) {
  final trimmed = endpoint.trim();
  if (trimmed.isEmpty) {
    return ConnectionTarget(host: fallbackHost, port: fallbackPort);
  }
  final parsed = Uri.tryParse(trimmed);
  if (parsed == null || parsed.host.isEmpty || parsed.port <= 0) {
    return ConnectionTarget(host: fallbackHost, port: fallbackPort);
  }
  return ConnectionTarget(host: parsed.host, port: parsed.port);
}

String buildCarrierEndpointLabel({
  required ControlCarrierKind carrier,
  required ConnectionTarget target,
  String websocketPath = '/control',
}) {
  switch (carrier) {
    case ControlCarrierKind.grpc:
      return '${target.host}:${target.port}';
    case ControlCarrierKind.websocket:
      return 'ws://${target.host}:${target.port}$websocketPath';
    case ControlCarrierKind.tcp:
      return '${target.host}:${target.port}';
    case ControlCarrierKind.http:
      return 'http://${target.host}:${target.port}';
  }
}

ConnectRequest buildSystemCommandRequest({
  required String requestID,
  required String intent,
}) {
  return ConnectRequest()
    ..command = (CommandRequest()
      ..requestId = requestID
      ..kind = CommandKind.COMMAND_KIND_SYSTEM
      ..intent = intent);
}

ConnectRequest buildRuntimeStatusQueryRequest(String requestID) {
  return buildSystemCommandRequest(
    requestID: requestID,
    intent: 'runtime_status',
  );
}

ConnectRequest buildDeviceStatusQueryRequest({
  required String requestID,
  required String deviceID,
}) {
  return buildSystemCommandRequest(
    requestID: requestID,
    intent: 'device_status $deviceID',
  );
}

ConnectRequest buildScenarioRegistryQueryRequest(String requestID) {
  return buildSystemCommandRequest(
    requestID: requestID,
    intent: 'scenario_registry',
  );
}

ConnectRequest buildApplicationLaunchCommandRequest({
  required String requestID,
  required String deviceID,
  required String intent,
}) {
  return ConnectRequest()
    ..command = (CommandRequest()
      ..requestId = requestID
      ..deviceId = deviceID
      ..action = CommandAction.COMMAND_ACTION_START
      ..kind = CommandKind.COMMAND_KIND_MANUAL
      ..intent = intent);
}

ConnectRequest buildUiActionInputRequest({
  required String deviceID,
  required ServerDrivenAction action,
}) {
  return ConnectRequest()
    ..input = (iov1.InputEvent()
      ..deviceId = deviceID
      ..uiAction = (iov1.UIAction()
        ..componentId = action.componentId
        ..action = action.action
        ..value = action.value));
}

ConnectRequest buildKeyInputRequest({
  required String deviceID,
  required String text,
}) {
  return ConnectRequest()
    ..input = (iov1.InputEvent()
      ..deviceId = deviceID
      ..key = (iov1.KeyEvent()..text = text));
}

ConnectRequest? buildSensorTelemetryRequest({
  required String deviceID,
  required capv1.DeviceCapabilities? capabilities,
  required int unixMs,
}) {
  if (deviceID.isEmpty || capabilities == null) {
    return null;
  }
  final values = <String, double>{};
  if (capabilities.hasBattery()) {
    values['battery.level'] = capabilities.battery.level.toDouble();
    values['battery.charging'] = capabilities.battery.charging ? 1.0 : 0.0;
  }
  if (values.isEmpty) {
    return null;
  }
  return ConnectRequest()
    ..sensor = (iov1.SensorData()
      ..deviceId = deviceID
      ..unixMs = Int64(unixMs)
      ..values.addAll(values));
}

ConnectRequest buildPlaybackArtifactsQueryRequest(String requestID) {
  return buildSystemCommandRequest(
    requestID: requestID,
    intent: 'list_playback_artifacts',
  );
}

ConnectRequest buildPlaybackMetadataQueryRequest({
  required String requestID,
  required String deviceID,
  required String artifactID,
  required String targetDeviceID,
}) {
  return ConnectRequest()
    ..command = (CommandRequest()
      ..requestId = requestID
      ..deviceId = deviceID
      ..kind = CommandKind.COMMAND_KIND_MANUAL
      ..intent = 'playback_metadata'
      ..arguments['artifact_id'] = artifactID
      ..arguments['target_device_id'] = targetDeviceID);
}
