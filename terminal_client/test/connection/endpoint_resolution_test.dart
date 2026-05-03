import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/connection/endpoint_resolution.dart';

void main() {
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
    expect(
      websocketPathFromEndpoint('ws://192.168.0.2:50054/custom'),
      '/custom',
    );
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
