import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/connection/control_client_factory.dart';
import 'package:terminal_client/connection/control_session_controller.dart';

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
}
