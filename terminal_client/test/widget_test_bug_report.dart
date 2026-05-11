import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/app/terminal_client_app.dart';
import 'package:terminal_client/connection/control_client_factory.dart';
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';
import 'package:terminal_client/gen/terminals/diagnostics/v1/diagnostics.pb.dart'
    as diagv1;
import 'package:terminal_client/gen/terminals/ui/v1/ui.pb.dart' as uiv1;

import 'widget_test_helpers.dart';

void main() {
  testWidgets('connection attempt immediately falls back to next carrier', (
    WidgetTester tester,
  ) async {
    final harness = FakeClientHarness(failConnectAttempts: 1);
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        mediaEngineFactory: harness.createMediaEngine,
        reconnectDelayBase: const Duration(seconds: 5),
        reconnectDelayMaxSeconds: 5,
      ),
    );

    await tester.tap(find.text('Connect Stream'));
    await tester.pump();

    for (var i = 0; i < 40; i++) {
      if (harness.createdClients.length >= 2) {
        break;
      }
      await tester.pump(const Duration(milliseconds: 10));
    }

    expect(harness.createdClients.length, 2);
    expect(harness.requestedCarriers.take(2).toList(), <ControlCarrierKind>[
      ControlCarrierKind.grpc,
      ControlCarrierKind.websocket,
    ]);
    expect(find.textContaining('Notification: gRPC failed at'), findsOneWidget);
  });

  testWidgets('all failed carriers surface local diagnostic summary', (
    WidgetTester tester,
  ) async {
    final harness = FakeClientHarness(failConnectAttempts: 4);
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        mediaEngineFactory: harness.createMediaEngine,
        reconnectDelayBase: const Duration(seconds: 5),
        reconnectDelayMaxSeconds: 5,
      ),
    );

    await tester.tap(find.text('Connect Stream'));
    await tester.pump();

    for (var i = 0; i < 80; i++) {
      if (harness.createdClients.length >= 4) {
        break;
      }
      await tester.pump(const Duration(milliseconds: 10));
    }

    expect(harness.createdClients.length, greaterThanOrEqualTo(4));
    final notification = tester.widget<Text>(
      find.textContaining('Notification: gRPC failed at'),
    );
    expect(notification.data, contains('WebSocket failed at'));
  });

  testWidgets(
    'reconnect re-registers capabilities after control stream failure',
    (WidgetTester tester) async {
      final harness = FakeClientHarness(failFirstConnectStream: true);
      await tester.pumpWidget(
        TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine,
          heartbeatInterval: const Duration(milliseconds: 40),
          reconnectDelayBase: const Duration(milliseconds: 30),
          reconnectDelayMaxSeconds: 1,
        ),
      );

      await tester.tap(find.text('Connect Stream'));
      await tester.pump();

      for (var i = 0; i < 80; i++) {
        if (harness.createdClients.length >= 2) {
          break;
        }
        await tester.pump(const Duration(milliseconds: 10));
      }
      expect(harness.createdClients.length, 2);

      for (var i = 0; i < 50; i++) {
        final reconnectHellos = harness.createdClients[1].requests
            .where((request) => request.hasHello())
            .length;
        final reconnectSnapshots = harness.createdClients[1].requests
            .where((request) => request.hasCapabilitySnapshot())
            .length;
        if (reconnectHellos > 0 && reconnectSnapshots > 0) {
          break;
        }
        await tester.pump(const Duration(milliseconds: 10));
      }

      final reconnectHellos = harness.createdClients[1].requests
          .where((request) => request.hasHello())
          .length;
      final reconnectSnapshots = harness.createdClients[1].requests
          .where((request) => request.hasCapabilitySnapshot())
          .length;
      expect(reconnectHellos, 1);
      expect(reconnectSnapshots, 1);
    },
  );

  testWidgets('queues bug report until register ack is received', (
    WidgetTester tester,
  ) async {
    final harness = FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        mediaEngineFactory: harness.createMediaEngine,
      ),
    );

    final connectButton = find.widgetWithText(ElevatedButton, 'Connect Stream');
    await tester.ensureVisible(connectButton);
    await tester.tap(connectButton);
    await tester.pump();

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..setUi = (uiv1.SetUI()
          ..root = (uiv1.Node()
            ..id = 'root'
            ..stack = (uiv1.StackWidget())
            ..children.add(
              uiv1.Node()
                ..id = 'server_bug_button'
                ..button = (uiv1.ButtonWidget()
                  ..label = 'Report a bug'
                  ..action = 'bug_report:subject-1'),
            ))),
    );
    await tester.pump();

    final serverBugButton = find.text('Report a bug');
    await tester.ensureVisible(serverBugButton);
    await tester.pumpAndSettle();
    await tester.tap(serverBugButton);
    await tester.pump();

    expect(
      harness.lastClient.requests.where((request) => request.hasBugReport()),
      isEmpty,
    );
    expect(find.textContaining('Control Stream: Bug report queued'),
        findsOneWidget);

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..registerAck = (RegisterAck()
          ..serverId = 'test-server'
          ..message = 'registered'),
    );
    await tester.pump();

    expect(
      harness.lastClient.requests.where((request) => request.hasBugReport()),
      hasLength(1),
    );
    expect(find.textContaining('Control Stream: Queued bug reports sent'),
        findsOneWidget);
  });

  testWidgets('shows positive bug report receipt after server ack', (
    WidgetTester tester,
  ) async {
    final harness = FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        mediaEngineFactory: harness.createMediaEngine,
      ),
    );

    final connectButton = find.widgetWithText(ElevatedButton, 'Connect Stream');
    await tester.ensureVisible(connectButton);
    await tester.tap(connectButton);
    await tester.pump();
    harness.lastClient.emitResponse(
      ConnectResponse()
        ..registerAck = (RegisterAck()
          ..serverId = 'test-server'
          ..message = 'registered'),
    );
    await tester.pump();

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..setUi = (uiv1.SetUI()
          ..root = (uiv1.Node()
            ..id = 'root'
            ..stack = (uiv1.StackWidget())
            ..children.add(
              uiv1.Node()
                ..id = 'server_bug_button'
                ..button = (uiv1.ButtonWidget()
                  ..label = 'Report a bug'
                  ..action = 'bug_report:subject-1'),
            ))),
    );
    await tester.pump();

    final serverBugButton = find.text('Report a bug');
    await tester.ensureVisible(serverBugButton);
    await tester.pumpAndSettle();
    await tester.tap(serverBugButton);
    await tester.pump();

    expect(
      harness.lastClient.requests.where((request) => request.hasBugReport()),
      hasLength(1),
    );
    expect(
      find.textContaining('Bug Report Receipt: Pending'),
      findsOneWidget,
    );

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..bugReportAck = (diagv1.BugReportAck()
          ..reportId = 'bug-rcpt-123'
          ..status = diagv1.BugReportStatus.BUG_REPORT_STATUS_FILED),
    );
    await tester.pump();

    expect(
      find.textContaining('Bug Report Receipt: Received'),
      findsOneWidget,
    );
    expect(find.textContaining('Receipt ID: bug-rcpt-123'), findsOneWidget);
  });

  testWidgets('attaches screenshot bytes to submitted bug reports', (
    WidgetTester tester,
  ) async {
    const screenshotBytes = <int>[0x89, 0x50, 0x4e, 0x47];
    final harness = FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        mediaEngineFactory: harness.createMediaEngine,
        bugReportScreenshotCapture: () async => screenshotBytes,
      ),
    );

    await tester.tap(find.text('Connect Stream'));
    await tester.pump();
    harness.lastClient.emitResponse(
      ConnectResponse()
        ..registerAck = (RegisterAck()
          ..serverId = 'test-server'
          ..message = 'registered'),
    );
    await tester.pump();

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..setUi = (uiv1.SetUI()
          ..root = (uiv1.Node()
            ..id = 'root'
            ..stack = (uiv1.StackWidget())
            ..children.add(
              uiv1.Node()
                ..id = 'server_bug_button'
                ..button = (uiv1.ButtonWidget()
                  ..label = 'Report a bug'
                  ..action = 'bug_report:subject-1'),
            ))),
    );
    await tester.pump();

    final serverBugButton = find.text('Report a bug');
    await tester.ensureVisible(serverBugButton);
    await tester.pumpAndSettle();
    await tester.tap(serverBugButton);
    await tester.pump();

    final bugReportRequest = harness.lastClient.requests.lastWhere(
      (request) => request.hasBugReport(),
    );
    expect(bugReportRequest.bugReport.screenshotPng, screenshotBytes);
    expect(
      bugReportRequest.bugReport.sourceHints['screenshot_byte_count'],
      screenshotBytes.length.toString(),
    );
  });

  testWidgets(
    'bug report client context omits battery fields without sensor snapshot',
    (WidgetTester tester) async {
      final harness = FakeClientHarness();
      await tester.pumpWidget(
        TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine,
        ),
      );

      await tester.tap(find.text('Connect Stream'));
      await tester.pump();
      harness.lastClient.emitResponse(
        ConnectResponse()
          ..registerAck = (RegisterAck()
            ..serverId = 'test-server'
            ..message = 'registered'),
      );
      await tester.pump();

      harness.lastClient.emitResponse(
        ConnectResponse()
          ..setUi = (uiv1.SetUI()
            ..root = (uiv1.Node()
              ..id = 'root'
              ..stack = (uiv1.StackWidget())
              ..children.add(
                uiv1.Node()
                  ..id = 'server_bug_button'
                  ..button = (uiv1.ButtonWidget()
                    ..label = 'Report a bug'
                    ..action = 'bug_report:subject-1'),
              ))),
      );
      await tester.pump();

      final serverBugButton = find.text('Report a bug');
      await tester.ensureVisible(serverBugButton);
      await tester.pumpAndSettle();
      await tester.tap(serverBugButton);
      await tester.pump();

      final bugReportRequest = harness.lastClient.requests.lastWhere(
        (request) => request.hasBugReport(),
      );
      final hardware = bugReportRequest.bugReport.clientContext.hardware;
      expect(hardware.hasBatteryLevel(), isFalse);
      expect(hardware.hasBatteryCharging(), isFalse);
      expect(hardware.sensorSnapshot, isEmpty);
    },
  );

  testWidgets('shows bug report receipt error when ack has no report id', (
    WidgetTester tester,
  ) async {
    final harness = FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        mediaEngineFactory: harness.createMediaEngine,
      ),
    );

    await tester.tap(find.text('Connect Stream'));
    await tester.pump();
    harness.lastClient.emitResponse(
      ConnectResponse()
        ..registerAck = (RegisterAck()
          ..serverId = 'test-server'
          ..message = 'registered'),
    );
    await tester.pump();

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..setUi = (uiv1.SetUI()
          ..root = (uiv1.Node()
            ..id = 'root'
            ..stack = (uiv1.StackWidget())
            ..children.add(
              uiv1.Node()
                ..id = 'server_bug_button'
                ..button = (uiv1.ButtonWidget()
                  ..label = 'Report a bug'
                  ..action = 'bug_report:subject-1'),
            ))),
    );
    await tester.pump();

    final serverBugButton = find.text('Report a bug');
    await tester.ensureVisible(serverBugButton);
    await tester.pumpAndSettle();
    await tester.tap(serverBugButton);
    await tester.pump();

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..bugReportAck = (diagv1.BugReportAck()
          ..status = diagv1.BugReportStatus.BUG_REPORT_STATUS_FILED),
    );
    await tester.pump();

    expect(
      find.textContaining('Bug Report Receipt: Error'),
      findsOneWidget,
    );
    expect(
      find.textContaining('No positive receipt was generated by the server'),
      findsOneWidget,
    );
  });

  testWidgets('fails bug report immediately when server returns control error',
      (
    WidgetTester tester,
  ) async {
    final harness = FakeClientHarness();
    var fakeNow = DateTime.utc(2026, 1, 1, 0, 0, 0).millisecondsSinceEpoch;
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        mediaEngineFactory: harness.createMediaEngine,
        nowUnixMsProvider: () => fakeNow,
      ),
    );

    await tester.tap(find.text('Connect Stream'));
    await tester.pump();
    harness.lastClient.emitResponse(
      ConnectResponse()
        ..registerAck = (RegisterAck()
          ..serverId = 'test-server'
          ..message = 'registered'),
    );
    await tester.pump();

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..setUi = (uiv1.SetUI()
          ..root = (uiv1.Node()
            ..id = 'root'
            ..stack = (uiv1.StackWidget())
            ..children.add(
              uiv1.Node()
                ..id = 'server_bug_button'
                ..button = (uiv1.ButtonWidget()
                  ..label = 'Report a bug'
                  ..action = 'bug_report:subject-1'),
            ))),
    );
    await tester.pump();

    final serverBugButton = find.text('Report a bug');
    await tester.ensureVisible(serverBugButton);
    await tester.pumpAndSettle();
    await tester.tap(serverBugButton);
    await tester.pump();

    expect(
      find.textContaining('Bug Report Receipt: Pending'),
      findsOneWidget,
    );

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..error = (ControlError()
          ..code = ControlErrorCode.CONTROL_ERROR_CODE_PROTOCOL_VIOLATION
          ..message = 'bug report intake unavailable'),
    );
    await tester.pump();

    expect(
      find.textContaining('Bug Report Receipt: Error'),
      findsOneWidget,
    );
    expect(
      find.textContaining(
        'No positive receipt could be generated: bug report intake unavailable',
      ),
      findsOneWidget,
    );

    fakeNow += const Duration(seconds: 21).inMilliseconds;
    await tester.pump(const Duration(seconds: 1));
    expect(
      find.textContaining(
        'No positive receipt could be generated: bug report intake unavailable',
      ),
      findsOneWidget,
    );
  });

  testWidgets('shows bug report receipt error when report stays queued', (
    WidgetTester tester,
  ) async {
    final harness = FakeClientHarness();
    var fakeNow = DateTime.utc(2026, 1, 1, 0, 0, 0).millisecondsSinceEpoch;
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        mediaEngineFactory: harness.createMediaEngine,
        nowUnixMsProvider: () => fakeNow,
      ),
    );

    await tester.tap(find.text('Connect Stream'));
    await tester.pump();

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..setUi = (uiv1.SetUI()
          ..root = (uiv1.Node()
            ..id = 'root'
            ..stack = (uiv1.StackWidget())
            ..children.add(
              uiv1.Node()
                ..id = 'server_bug_button'
                ..button = (uiv1.ButtonWidget()
                  ..label = 'Report a bug'
                  ..action = 'bug_report:subject-1'),
            ))),
    );
    await tester.pump();

    final serverBugButton = find.text('Report a bug');
    await tester.ensureVisible(serverBugButton);
    await tester.pumpAndSettle();
    await tester.tap(serverBugButton);
    await tester.pump();

    expect(
      harness.lastClient.requests.where((request) => request.hasBugReport()),
      isEmpty,
    );
    expect(
      find.textContaining('Bug Report Receipt: Pending'),
      findsOneWidget,
    );

    fakeNow += const Duration(seconds: 21).inMilliseconds;
    await tester.pump(const Duration(seconds: 1));

    expect(
      find.textContaining('Bug Report Receipt: Error'),
      findsOneWidget,
    );
    expect(
      find.textContaining('remained queued for more than'),
      findsOneWidget,
    );
  });
}
