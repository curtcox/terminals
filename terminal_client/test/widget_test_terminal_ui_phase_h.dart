import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/app/terminal_client_app.dart';
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';
import 'package:terminal_client/gen/terminals/ui/v1/ui.pb.dart' as uiv1;

import 'widget_test_helpers.dart';

void main() {
  testWidgets(
    'displaySurfaceMode: register without SetUI shows server-shaped placeholder',
    (WidgetTester tester) async {
      await tester.binding.setSurfaceSize(const Size(1200, 1400));
      addTearDown(() => tester.binding.setSurfaceSize(null));
      final harness = FakeClientHarness();
      await tester.pumpWidget(
        TerminalClientApp(
          displaySurfaceMode: true,
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine,
        ),
      );
      await tester.tap(find.text('Connect Stream'));
      await tester.pump();
      harness.lastClient.emitResponse(
        ConnectResponse()..registerAck = RegisterAck(),
      );
      await tester.pump();
      expect(find.text('Awaiting server UI'), findsOneWidget);
      expect(find.text('Server Host'), findsNothing);
    },
  );

  testWidgets(
    'displaySurfaceMode: first SetUI replaces placeholder',
    (WidgetTester tester) async {
      await tester.binding.setSurfaceSize(const Size(1200, 1400));
      addTearDown(() => tester.binding.setSurfaceSize(null));
      final harness = FakeClientHarness();
      await tester.pumpWidget(
        TerminalClientApp(
          displaySurfaceMode: true,
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine,
        ),
      );
      await tester.tap(find.text('Connect Stream'));
      await tester.pump();
      harness.lastClient.emitResponse(
        ConnectResponse()..registerAck = RegisterAck(),
      );
      await tester.pump();
      expect(find.text('Awaiting server UI'), findsOneWidget);

      harness.lastClient.emitResponse(
        ConnectResponse()
          ..setUi = (uiv1.SetUI()
            ..root = (uiv1.Node()
              ..id = 'root'
              ..props['client_chrome'] = 'hidden'
              ..stack = uiv1.StackWidget()
              ..children.add(
                uiv1.Node()
                  ..id = 'msg'
                  ..text = (uiv1.TextWidget()..value = 'PhaseHHello'),
              ))),
      );
      await tester.pump();
      expect(find.text('PhaseHHello'), findsOneWidget);
      expect(find.text('Awaiting server UI'), findsNothing);
    },
  );

  testWidgets(
    'displaySurfaceMode: prior session idle tree is not reused on fresh shell',
    (WidgetTester tester) async {
      await tester.binding.setSurfaceSize(const Size(1200, 1400));
      addTearDown(() => tester.binding.setSurfaceSize(null));

      Future<void> runSession(FakeClientHarness harness) async {
        await tester.pumpWidget(
          TerminalClientApp(
            displaySurfaceMode: true,
            clientFactory: harness.createClient,
            mediaEngineFactory: harness.createMediaEngine,
          ),
        );
        await tester.tap(find.text('Connect Stream'));
        await tester.pump();
        harness.lastClient.emitResponse(
          ConnectResponse()..registerAck = RegisterAck(),
        );
        await tester.pump();
      }

      final first = FakeClientHarness();
      await runSession(first);
      first.lastClient.emitResponse(
        ConnectResponse()
          ..setUi = (uiv1.SetUI()
            ..root = (uiv1.Node()
              ..id = 'idle_root'
              ..props['client_chrome'] = 'hidden'
              ..stack = uiv1.StackWidget()
              ..children.add(
                uiv1.Node()
                  ..id = 'idle_msg'
                  ..text = (uiv1.TextWidget()..value = 'IdleCacheProbe'),
              ))),
      );
      await tester.pump();
      expect(find.text('IdleCacheProbe'), findsOneWidget);

      await tester.pumpWidget(const SizedBox.shrink());
      await tester.pump();

      final second = FakeClientHarness();
      await runSession(second);
      expect(find.text('IdleCacheProbe'), findsNothing);
      expect(find.text('Awaiting server UI'), findsOneWidget);
    },
  );
}
