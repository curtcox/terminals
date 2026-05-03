import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/connection/reliability.dart';
import 'package:terminal_client/diagnostics/bug_report_chrome.dart';
import 'package:terminal_client/diagnostics/client_chrome.dart';

void main() {
  Widget harness(Widget child) {
    return MaterialApp(home: Scaffold(body: child));
  }

  testWidgets('renders build metadata and parity chrome', (tester) async {
    await tester.pumpWidget(
      harness(
        const Column(
          children: [
            ClientMetadataFooter(buildDate: '2026-05-01', buildSha: 'abcdef0'),
            BuildParityPanel(
              clientBuildDate: '2026-05-01',
              clientBuildSha: 'abcdef0',
              serverBuildDate: '2026-05-01',
              serverBuildSha: 'abcdef0',
              hasRegisterAck: true,
            ),
          ],
        ),
      ),
    );

    expect(find.textContaining('2026-05-01'), findsWidgets);
    expect(find.text('Client / Server Build'), findsOneWidget);
    expect(find.textContaining('Client'), findsWidgets);
    expect(find.textContaining('Server'), findsWidgets);
  });

  testWidgets('renders connection status with bug receipt details',
      (tester) async {
    await tester.pumpWidget(
      harness(
        const ControlStreamStatusCard(
          status: 'Connected',
          notification: 'Ready',
          transportDiagnostics: 'last carrier ok',
          bugReceiptState: BugReceiptChromeState.received,
          bugReceiptReportId: 'bug-123',
          bugReceiptDetail: 'Filed successfully',
        ),
      ),
    );

    expect(find.textContaining('Connected'), findsOneWidget);
    expect(find.textContaining('Ready'), findsOneWidget);
    expect(find.text('Bug Report Receipt: Received'), findsOneWidget);
    expect(find.text('Receipt ID: bug-123'), findsOneWidget);
    expect(find.text('Filed successfully'), findsOneWidget);
  });

  testWidgets('renders diagnostics and transport panels', (tester) async {
    await tester.pumpWidget(
      harness(
        const Column(
          children: [
            DiagnosticsPanel(
              title: 'runtime_status',
              data: <String, String>{'uptime': '42', 'version': 'dev'},
            ),
            TransportDiagnosticsPanel(
              lastTransportDiagnostic: 'websocket failed',
              recentAttempts: <String>['gRPC failed', 'WebSocket failed'],
            ),
          ],
        ),
      ),
    );

    expect(find.text('Diagnostics: runtime_status'), findsOneWidget);
    expect(find.text('uptime=42'), findsOneWidget);
    expect(find.textContaining('websocket failed'), findsOneWidget);
    expect(find.textContaining('gRPC failed'), findsOneWidget);
  });

  testWidgets('renders connection phase chip', (tester) async {
    await tester.pumpWidget(
      harness(
        const ConnectionPhaseChip(
          phase: ConnectionPhase.registered,
          isRegistered: true,
        ),
      ),
    );

    expect(find.byIcon(Icons.check_circle_outline), findsOneWidget);
    expect(find.text('Connected'), findsOneWidget);
  });
}
