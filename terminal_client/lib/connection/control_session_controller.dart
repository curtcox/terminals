import 'dart:math' as math;

import 'package:terminal_client/connection/control_client_factory.dart';
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';

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

  String get carrierLabel {
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
