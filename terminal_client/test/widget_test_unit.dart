import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/connection/carrier_preference.dart';
import 'package:terminal_client/connection/control_client_factory.dart';
import 'package:terminal_client/connection/endpoint_resolution.dart';
import 'package:terminal_client/connection/transport_diagnostics.dart';

void main() {
  test('diagnoseTransportError identifies grpc unavailable socket issue', () {
    final diagnosis = diagnoseTransportError(
      StateError(
        'gRPC Error (code: 14, codeName: UNAVAILABLE, message: Error connecting: Unsupported operation: Socket constructor, details: null, rawResponse: null, trailers: {})',
      ),
      isWeb: true,
    );
    expect(diagnosis.summary, 'gRPC UNAVAILABLE (14)');
    expect(diagnosis.grpcCode, 14);
    expect(
      diagnosis.notificationText(),
      contains('Browser runtime cannot open raw gRPC sockets'),
    );
  });

  test('diagnoseTransportError identifies grpc unavailable generally', () {
    final diagnosis = diagnoseTransportError(
      StateError(
        'gRPC Error (code: 14, codeName: UNAVAILABLE, message: connection refused)',
      ),
      isWeb: false,
    );
    expect(diagnosis.summary, 'gRPC UNAVAILABLE (14)');
    expect(
      diagnosis.notificationText(),
      contains('Server is unreachable or transport is unavailable'),
    );
  });

  test('buildCarrierPreference respects priority and last successful carrier',
      () {
    final ordered = buildCarrierPreference(
      isWebRuntime: false,
      serverPriority: const <String>['http', 'tcp', 'websocket', 'grpc'],
      lastSuccessfulCarrier: ControlCarrierKind.websocket,
    );
    expect(
      ordered,
      <ControlCarrierKind>[
        ControlCarrierKind.websocket,
        ControlCarrierKind.http,
        ControlCarrierKind.tcp,
        ControlCarrierKind.grpc,
      ],
    );
  });

  test('resolvePreferredEndpoint prefers manual override when provided', () {
    final endpoint = resolvePreferredEndpoint(
      manualEndpoint: '  ws://manual.example:50054/control  ',
      discoveredEndpoint: 'ws://mdns.example:50054/control',
    );
    expect(endpoint, 'ws://manual.example:50054/control');
  });

  test('resolvePreferredEndpoint falls back to discovered endpoint', () {
    final endpoint = resolvePreferredEndpoint(
      manualEndpoint: '   ',
      discoveredEndpoint: 'host.example:50051',
    );
    expect(endpoint, 'host.example:50051');
  });

  test('websocketPathFromEndpoint returns path from endpoint uri', () {
    final path = websocketPathFromEndpoint('ws://192.168.0.2:50054/custom');
    expect(path, '/custom');
  });

  test('websocketPathFromEndpoint falls back to default path', () {
    expect(websocketPathFromEndpoint('ws://192.168.0.2:50054'), '/control');
    expect(websocketPathFromEndpoint(''), '/control');
    expect(websocketPathFromEndpoint('not a uri'), '/control');
  });

  test('resolveInitialControlHost preserves configured host off web', () {
    final host = resolveInitialControlHost(
      isWebRuntime: false,
      configuredHost: '127.0.0.1',
      pageHost: '192.168.0.138',
    );
    expect(host, '127.0.0.1');
  });

  test('resolveInitialControlHost uses page host for loopback on web', () {
    final host = resolveInitialControlHost(
      isWebRuntime: true,
      configuredHost: '127.0.0.1',
      pageHost: '192.168.0.138',
    );
    expect(host, '192.168.0.138');
  });

  test('resolveInitialControlHost uses page host when configured host is empty',
      () {
    final host = resolveInitialControlHost(
      isWebRuntime: true,
      configuredHost: '   ',
      pageHost: 'localhost',
    );
    expect(host, 'localhost');
  });

  test('resolveInitialControlHost keeps non-loopback host on web', () {
    final host = resolveInitialControlHost(
      isWebRuntime: true,
      configuredHost: 'terminals.internal',
      pageHost: '192.168.0.138',
    );
    expect(host, 'terminals.internal');
  });

  test('resolvePageHost prefers browser location host', () {
    final host = resolvePageHost(
      browserLocationHost: '192.168.0.138',
      uriBaseHost: '127.0.0.1',
    );
    expect(host, '192.168.0.138');
  });

  test('resolvePageHost falls back to Uri.base host when location host empty',
      () {
    final host = resolvePageHost(
      browserLocationHost: '   ',
      uriBaseHost: 'localhost',
    );
    expect(host, 'localhost');
  });
}
