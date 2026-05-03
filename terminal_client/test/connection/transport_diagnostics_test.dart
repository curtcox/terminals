import 'package:flutter_test/flutter_test.dart';
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

  test('classifyCarrierFailure maps common transport failures', () {
    expect(
      classifyCarrierFailure(
        stage: 'connect',
        rawError: 'Connection refused',
      ),
      'tcp_connect',
    );
    expect(
      classifyCarrierFailure(
        stage: 'stream',
        rawError: 'TLS handshake failed',
      ),
      'tls_or_handshake',
    );
    expect(
      classifyCarrierFailure(
        stage: 'stream_closed',
        rawError: 'done',
      ),
      'stream_closed',
    );
  });
}
