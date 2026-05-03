import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/diagnostics/bug_report_chrome.dart';

void main() {
  Widget harness(Widget child) {
    return MaterialApp(home: Scaffold(body: child));
  }

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
