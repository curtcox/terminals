import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/diagnostics/diagnostic_clipboard.dart';

void main() {
  test('buildTransportDiagnosticsClipboardText returns single multiline block',
      () {
    final text = buildTransportDiagnosticsClipboardText(
      lastTransportDiagnostic: 'gRPC unavailable',
      recentAttempts: const <String>[
        'websocket connect failed',
        'http poll failed',
      ],
    );

    expect(
      text,
      [
        'Transport Diagnostics',
        'gRPC unavailable',
        'Recent Carrier Attempts',
        'websocket connect failed',
        'http poll failed',
      ].join('\n'),
    );
  });

  test('buildControlStreamClipboardText returns single multiline block', () {
    final text = buildControlStreamClipboardText(
      status: 'All control carriers failed',
      notification: 'Socket constructor unavailable',
      transportDiagnostics: 'Browser runtime cannot open raw gRPC sockets',
    );

    expect(
      text,
      [
        'Control Stream: All control carriers failed',
        'Socket constructor unavailable',
        'Transport Diagnostics',
        'Browser runtime cannot open raw gRPC sockets',
      ].join('\n'),
    );
  });
}
