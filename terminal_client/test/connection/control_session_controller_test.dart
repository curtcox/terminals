import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/connection/control_client_factory.dart';
import 'package:terminal_client/connection/control_session_controller.dart';
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';

void main() {
  test('reconnect delay grows exponentially and caps at max', () {
    expect(
      calculateReconnectDelay(
        reconnectAttempt: 1,
        reconnectDelayBase: const Duration(milliseconds: 30),
        reconnectDelayMaxSeconds: 1,
      ),
      const Duration(milliseconds: 30),
    );
    expect(
      calculateReconnectDelay(
        reconnectAttempt: 2,
        reconnectDelayBase: const Duration(milliseconds: 30),
        reconnectDelayMaxSeconds: 1,
      ),
      const Duration(milliseconds: 60),
    );
    expect(
      calculateReconnectDelay(
        reconnectAttempt: 10,
        reconnectDelayBase: const Duration(milliseconds: 30),
        reconnectDelayMaxSeconds: 1,
      ),
      const Duration(seconds: 1),
    );
  });

  test('carrier attempt formatting includes carrier and diagnosis', () {
    final attempt = CarrierAttemptDiagnostic(
      carrier: ControlCarrierKind.websocket,
      endpoint: 'ws://127.0.0.1:50054/control',
      stage: 'stream',
      failureClass: 'timeout',
      error: 'timed out',
      elapsed: const Duration(milliseconds: 125),
    );

    expect(
      formatCarrierAttempt(attempt),
      'WebSocket failed at stream [timeout] '
      '(ws://127.0.0.1:50054/control) after 125ms: timed out',
    );
  });

  test('connection target carries resolved host and port', () {
    const target = ConnectionTarget(host: '127.0.0.1', port: 50051);

    expect(target.host, '127.0.0.1');
    expect(target.port, 50051);
  });

  test('carrier labels are stable for diagnostics and status text', () {
    expect(controlCarrierLabel(ControlCarrierKind.grpc), 'gRPC');
    expect(controlCarrierLabel(ControlCarrierKind.websocket), 'WebSocket');
    expect(controlCarrierLabel(ControlCarrierKind.tcp), 'TCP');
    expect(controlCarrierLabel(ControlCarrierKind.http), 'HTTP');
  });

  test('resolves connection targets from carrier endpoints', () {
    expect(
      resolveConnectionTargetForCarrier(
        carrier: ControlCarrierKind.grpc,
        endpoint: 'server.local:50051',
        fallbackHost: '127.0.0.1',
        fallbackPort: 50051,
      ),
      isA<ConnectionTarget>()
          .having((target) => target.host, 'host', 'server.local')
          .having((target) => target.port, 'port', 50051),
    );
    expect(
      resolveConnectionTargetForCarrier(
        carrier: ControlCarrierKind.websocket,
        endpoint: 'ws://server.local:50054/control',
        fallbackHost: '127.0.0.1',
        fallbackPort: 50054,
      ),
      isA<ConnectionTarget>()
          .having((target) => target.host, 'host', 'server.local')
          .having((target) => target.port, 'port', 50054),
    );
    expect(
      resolveConnectionTargetForCarrier(
        carrier: ControlCarrierKind.tcp,
        endpoint: '',
        fallbackHost: '127.0.0.1',
        fallbackPort: 50051,
      ).port,
      50055,
    );
    expect(
      resolveConnectionTargetForCarrier(
        carrier: ControlCarrierKind.http,
        endpoint: 'not a uri',
        fallbackHost: '127.0.0.1',
        fallbackPort: 50051,
      ).port,
      50056,
    );
  });

  test('formats carrier endpoint labels without shell state', () {
    const target = ConnectionTarget(host: 'server.local', port: 50054);

    expect(
      buildCarrierEndpointLabel(
        carrier: ControlCarrierKind.websocket,
        target: target,
        websocketPath: '/control/v2',
      ),
      'ws://server.local:50054/control/v2',
    );
    expect(
      buildCarrierEndpointLabel(
        carrier: ControlCarrierKind.grpc,
        target: target,
      ),
      'server.local:50054',
    );
    expect(
      buildCarrierEndpointLabel(
        carrier: ControlCarrierKind.http,
        target: target,
      ),
      'http://server.local:50054',
    );
  });

  test('parses grpc ports with fallback for empty or malformed endpoints', () {
    expect(
      grpcPortFromEndpoint(endpoint: 'server.local:50061', fallbackPort: 50051),
      50061,
    );
    expect(grpcPortFromEndpoint(endpoint: '', fallbackPort: 50051), 50051);
    expect(
      grpcPortFromEndpoint(endpoint: 'server.local', fallbackPort: 50051),
      50051,
    );
  });

  test('builds system command requests with deterministic intents', () {
    final runtime = buildRuntimeStatusQueryRequest('runtime-1');
    expect(runtime.command.requestId, 'runtime-1');
    expect(runtime.command.intent, 'runtime_status');

    final device = buildDeviceStatusQueryRequest(
      requestID: 'device-1',
      deviceID: 'terminal-a',
    );
    expect(device.command.requestId, 'device-1');
    expect(device.command.intent, 'device_status terminal-a');

    final registry = buildScenarioRegistryQueryRequest('registry-1');
    expect(registry.command.requestId, 'registry-1');
    expect(registry.command.intent, 'scenario_registry');
  });

  test('builds manual application launch requests', () {
    final request = buildApplicationLaunchCommandRequest(
      requestID: 'launch-1',
      deviceID: 'terminal-a',
      intent: 'dashboard',
    );

    expect(request.command.requestId, 'launch-1');
    expect(request.command.deviceId, 'terminal-a');
    expect(request.command.action, CommandAction.COMMAND_ACTION_START);
    expect(request.command.kind, CommandKind.COMMAND_KIND_MANUAL);
    expect(request.command.intent, 'dashboard');
  });

  test('builds playback diagnostics requests', () {
    final artifacts = buildPlaybackArtifactsQueryRequest('artifacts-1');
    expect(artifacts.command.requestId, 'artifacts-1');
    expect(artifacts.command.kind, CommandKind.COMMAND_KIND_SYSTEM);
    expect(artifacts.command.intent, 'list_playback_artifacts');

    final metadata = buildPlaybackMetadataQueryRequest(
      requestID: 'metadata-1',
      deviceID: 'terminal-a',
      artifactID: 'playback-1',
      targetDeviceID: 'terminal-b',
    );
    expect(metadata.command.requestId, 'metadata-1');
    expect(metadata.command.deviceId, 'terminal-a');
    expect(metadata.command.kind, CommandKind.COMMAND_KIND_MANUAL);
    expect(metadata.command.intent, 'playback_metadata');
    expect(metadata.command.arguments['artifact_id'], 'playback-1');
    expect(metadata.command.arguments['target_device_id'], 'terminal-b');
  });
}
