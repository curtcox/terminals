import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:terminal_client/main.dart';

void main() {
  testWidgets('app renders MaterialApp', (WidgetTester tester) async {
    await tester.pumpWidget(const TerminalClientApp());
    expect(find.byType(MaterialApp), findsOneWidget);
  });
}
