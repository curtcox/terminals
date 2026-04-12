import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:grpc/grpc.dart';
import 'package:terminal_client/connection/control_client.dart';
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';
import 'package:terminal_client/main.dart';

void main() {
  testWidgets('app renders MaterialApp', (WidgetTester tester) async {
    await tester.pumpWidget(const TerminalClientApp());
    expect(find.byType(MaterialApp), findsOneWidget);
  });

  testWidgets('sends periodic heartbeat messages while connected', (
    WidgetTester tester,
  ) async {
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        heartbeatInterval: const Duration(milliseconds: 40),
      ),
    );

    await tester.tap(find.text('Connect Stream'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 140));

    final requests = harness.lastClient.requests;
    final heartbeatCount = requests.where((r) => r.hasHeartbeat()).length;
    expect(heartbeatCount, greaterThanOrEqualTo(2));
    expect(requests.where((r) => r.hasRegister()).length, 1);
  });

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
}

class _FakeClientHarness {
  final List<_FakeTerminalControlClient> createdClients =
      <_FakeTerminalControlClient>[];

  _FakeTerminalControlClient get lastClient => createdClients.last;

  TerminalControlClient createClient({
    required String host,
    required int port,
  }) {
    final client = _FakeTerminalControlClient(host: host, port: port);
    createdClients.add(client);
    return client;
  }
}

class _FakeTerminalControlClient implements TerminalControlClient {
  _FakeTerminalControlClient({
    required this.host,
    required this.port,
  });

  final String host;
  final int port;
  final List<ConnectRequest> requests = <ConnectRequest>[];
  final StreamController<ConnectResponse> _responses =
      StreamController<ConnectResponse>.broadcast();
  StreamSubscription<ConnectRequest>? _requestSubscription;

  @override
  Stream<ConnectResponse> connect(
    Stream<ConnectRequest> requests, {
    CallOptions? options,
  }) {
    _requestSubscription = requests.listen(this.requests.add);
    return _responses.stream;
  }

  @override
  Future<void> shutdown() async {
    await _requestSubscription?.cancel();
  }
}
