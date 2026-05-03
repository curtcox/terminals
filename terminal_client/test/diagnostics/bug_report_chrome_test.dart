import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/diagnostics/bug_report_chrome.dart';

void main() {
  Widget harness(Widget child) {
    return MaterialApp(home: Scaffold(body: child));
  }

  test('buildBugIdentifier derives deterministic word code and qr payload', () {
    final identifier = buildBugIdentifier(
      DateTime(2026, 5, 3, 1, 2, 3),
      words: const <String>['alpha', 'beta', 'gamma'],
    );

    expect(identifier.word, 'alpha');
    expect(identifier.code, '010203-alpha');
    expect(identifier.qrPayload, 'terminals-bug://010203-alpha');
  });

  test('buildLocalBugReportId sanitizes report id components', () {
    final reportId = buildLocalBugReportId(
      now: DateTime.utc(2026, 5, 3, 12),
      identifier: const BugIdentifier(
        word: 'alpha',
        code: '12:00 alpha',
        qrPayload: 'ignored',
      ),
      reporterDeviceID: 'Flutter Client!',
      subjectDeviceID: 'Kitchen Display #1',
    );

    expect(
      reportId,
      'clientbug-1777809600000-flutter-client-kitchen-display-1-12-00-alpha',
    );
  });

  testWidgets('renders bug report button and invokes callback', (tester) async {
    var pressed = false;
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          floatingActionButton: BugReportButton(
            onPressed: () {
              pressed = true;
            },
          ),
        ),
      ),
    );

    await tester.tap(find.text('Report Bug'));
    await tester.pump();

    expect(pressed, isTrue);
    expect(find.byIcon(Icons.bug_report_outlined), findsOneWidget);
  });

  testWidgets('renders bug receipt states', (tester) async {
    await tester.pumpWidget(
      harness(
        const Column(
          children: [
            BugReceiptPanel(state: BugReceiptChromeState.waiting),
            BugReceiptPanel(
              state: BugReceiptChromeState.error,
              detail: 'No positive receipt',
            ),
          ],
        ),
      ),
    );

    expect(find.text('Bug Report Receipt: Pending'), findsOneWidget);
    expect(find.text('Bug Report Receipt: Error'), findsOneWidget);
    expect(find.text('No positive receipt'), findsOneWidget);
  });

  testWidgets('hides empty receipt state', (tester) async {
    await tester.pumpWidget(
      harness(const BugReceiptPanel(state: BugReceiptChromeState.none)),
    );

    expect(find.textContaining('Bug Report Receipt'), findsNothing);
    expect(find.byType(SizedBox), findsOneWidget);
  });
}
