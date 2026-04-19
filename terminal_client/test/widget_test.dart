import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_webrtc/flutter_webrtc.dart';
import 'package:grpc/grpc.dart';
import 'package:terminal_client/connection/control_client.dart';
import 'package:terminal_client/connection/control_client_factory.dart';
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';
import 'package:terminal_client/gen/terminals/diagnostics/v1/diagnostics.pb.dart'
    as diagv1;
import 'package:terminal_client/gen/terminals/io/v1/io.pb.dart' as iov1;
import 'package:terminal_client/gen/terminals/ui/v1/ui.pb.dart' as uiv1;
import 'package:terminal_client/main.dart';
import 'package:terminal_client/media/webrtc_engine.dart';

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

  test('buildCarrierPreference respects priority and last successful carrier',
      () {
    final ordered = buildCarrierPreference(
      isWebRuntime: false,
      serverPriority: const <String>['http', 'tcp', 'websocket', 'grpc'],
      lastSuccessfulCarrier: ControlCarrierKind.websocket,
    );
    expect(
      ordered,
      <ControlCarrierKind>[
        ControlCarrierKind.websocket,
        ControlCarrierKind.http,
        ControlCarrierKind.tcp,
        ControlCarrierKind.grpc,
      ],
    );
  });

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
        mediaEngineFactory: harness.createMediaEngine,
        heartbeatInterval: const Duration(milliseconds: 40),
      ),
    );

    await tester.tap(find.text('Connect Stream'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 140));

    final requests = harness.lastClient.requests;
    final heartbeatCount = requests.where((r) => r.hasHeartbeat()).length;
    expect(heartbeatCount, greaterThanOrEqualTo(2));
    expect(requests.where((r) => r.hasHello()).length, 1);
    expect(requests.where((r) => r.hasCapabilitySnapshot()).length, 1);
  });

  testWidgets('sends periodic sensor telemetry while connected', (
    WidgetTester tester,
  ) async {
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        mediaEngineFactory: harness.createMediaEngine,
        heartbeatInterval: const Duration(seconds: 60),
        sensorTelemetryInterval: const Duration(milliseconds: 40),
      ),
    );

    await tester.tap(find.text('Connect Stream'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 140));

    final requests = harness.lastClient.requests;
    final sensorRequests = requests.where((r) => r.hasSensor()).toList();
    expect(sensorRequests.length, greaterThanOrEqualTo(2));
    expect(
      sensorRequests.every((request) =>
          request.sensor.values.containsKey('connectivity.reconnect_attempt') &&
          request.sensor.values.containsKey('time.utc_hour')),
      isTrue,
    );
    expect(find.textContaining('Sensor sends: '), findsOneWidget);
    expect(find.textContaining('Last sensor unix_ms: '), findsOneWidget);
    expect(find.textContaining('Stream-ready acks: 0'), findsOneWidget);
  });

  testWidgets('pauses heartbeat loop while app is backgrounded', (
    WidgetTester tester,
  ) async {
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        mediaEngineFactory: harness.createMediaEngine,
        heartbeatInterval: const Duration(milliseconds: 40),
      ),
    );

    await tester.tap(find.text('Connect Stream'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 120));

    final beforePause = harness.lastClient.requests
        .where((request) => request.hasHeartbeat())
        .length;
    expect(beforePause, greaterThan(0));

    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.paused);
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 120));

    final afterPause = harness.lastClient.requests
        .where((request) => request.hasHeartbeat())
        .length;
    expect(afterPause, beforePause);
  });

  testWidgets('reconnect creates a new control client after stream failure', (
    WidgetTester tester,
  ) async {
    final harness = _FakeClientHarness(failFirstConnectStream: true);
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
    expect(harness.createdClients.length, greaterThanOrEqualTo(1));

    for (var i = 0; i < 80; i++) {
      if (harness.createdClients.length >= 2) {
        break;
      }
      await tester.pump(const Duration(milliseconds: 10));
    }
    expect(harness.createdClients.length, 2);
  });

  testWidgets(
    'reconnect can switch carriers and recover after initial failure',
    (WidgetTester tester) async {
      final harness = _FakeClientHarness(failConnectAttempts: 1);
      await tester.pumpWidget(
        TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine,
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
      expect(
        harness.requestedCarriers.take(2).toList(),
        <ControlCarrierKind>[
          ControlCarrierKind.grpc,
          ControlCarrierKind.websocket,
        ],
      );

      harness.createdClients[1].emitResponse(
        ConnectResponse()
          ..registerAck = (RegisterAck()
            ..serverId = 'test-server'
            ..message = 'registered'),
      );
      await tester.pump();

      final helloCount = harness.createdClients[1].requests
          .where((request) => request.hasHello())
          .length;
      final capabilitySnapshotCount = harness.createdClients[1].requests
          .where((request) => request.hasCapabilitySnapshot())
          .length;
      expect(helloCount, 1);
      expect(capabilitySnapshotCount, 1);
    },
  );

  testWidgets('connection attempt immediately falls back to next carrier', (
    WidgetTester tester,
  ) async {
    final harness = _FakeClientHarness(failConnectAttempts: 1);
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
    final harness = _FakeClientHarness(failConnectAttempts: 4);
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
      final harness = _FakeClientHarness(failFirstConnectStream: true);
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
    final harness = _FakeClientHarness();
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
    final harness = _FakeClientHarness();
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
    final harness = _FakeClientHarness();
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

  testWidgets('shows bug report receipt error when ack has no report id', (
    WidgetTester tester,
  ) async {
    final harness = _FakeClientHarness();
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
    final harness = _FakeClientHarness();
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
    final harness = _FakeClientHarness();
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

  testWidgets('applies update_ui patch to active server-driven UI', (
    WidgetTester tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine),
    );
    await tester.tap(find.text('Connect Stream'));
    await tester.pump();

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..setUi = (uiv1.SetUI()
          ..root = (uiv1.Node()
            ..stack = (uiv1.StackWidget())
            ..children.add(
              uiv1.Node()
                ..id = 'greeting_text'
                ..text = (uiv1.TextWidget()..value = 'Hello'),
            ))),
    );
    await tester.pump();
    expect(find.text('Hello'), findsOneWidget);

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..updateUi = (uiv1.UpdateUI()
          ..componentId = 'greeting_text'
          ..node = (uiv1.Node()
            ..id = 'greeting_text'
            ..text = (uiv1.TextWidget()..value = 'Updated hello'))),
    );
    await tester.pump();
    expect(find.text('Hello'), findsNothing);
    expect(find.text('Updated hello'), findsOneWidget);
    expect(find.textContaining('Control Stream: UI patched'), findsOneWidget);
  });

  testWidgets('handles transition_ui responses', (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine),
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
                ..id = 'message'
                ..text = (uiv1.TextWidget()..value = 'Before transition'),
            ))),
    );
    await tester.pump();
    expect(find.text('Before transition'), findsOneWidget);

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..transitionUi = (uiv1.TransitionUI()
          ..transition = 'fade'
          ..durationMs = 220),
    );
    await tester.pump();

    expect(find.byType(FadeTransition), findsWidgets);
    expect(
        find.textContaining('Control Stream: UI transition'), findsOneWidget);
    expect(find.textContaining('Transition: fade (220ms)'), findsOneWidget);
  });

  testWidgets('uses slide transition hint for UI updates', (
    WidgetTester tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine),
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
                ..id = 'message'
                ..text = (uiv1.TextWidget()..value = 'Before'),
            ))),
    );
    await tester.pump();

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..transitionUi = (uiv1.TransitionUI()
          ..transition = 'slide_left'
          ..durationMs = 200),
    );
    harness.lastClient.emitResponse(
      ConnectResponse()
        ..updateUi = (uiv1.UpdateUI()
          ..componentId = 'message'
          ..node = (uiv1.Node()
            ..id = 'message'
            ..text = (uiv1.TextWidget()..value = 'After'))),
    );
    await tester.pump();

    expect(find.byType(SlideTransition), findsWidgets);
    expect(find.text('After'), findsOneWidget);
  });

  testWidgets('maps PA transition hints to explicit animations',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine),
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
                ..id = 'message'
                ..text = (uiv1.TextWidget()..value = 'PA transition body'),
            ))),
    );
    await tester.pump();

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..transitionUi = (uiv1.TransitionUI()
          ..transition = 'pa_source_enter'
          ..durationMs = 180),
    );
    await tester.pump();
    expect(find.byType(ScaleTransition), findsWidgets);

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..transitionUi = (uiv1.TransitionUI()
          ..transition = 'pa_receive_enter'
          ..durationMs = 180),
    );
    await tester.pump();
    expect(find.byType(SlideTransition), findsWidgets);
  });

  testWidgets('renders grid, padding, and progress primitives',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine),
    );
    await tester.tap(find.text('Connect Stream'));
    await tester.pump();

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..setUi = (uiv1.SetUI()
          ..root = (uiv1.Node()
            ..id = 'root'
            ..stack = (uiv1.StackWidget())
            ..children.addAll([
              uiv1.Node()
                ..padding = (uiv1.PaddingWidget()..all = 16)
                ..children.add(
                  uiv1.Node()
                    ..text = (uiv1.TextWidget()..value = 'Padded child'),
                ),
              uiv1.Node()
                ..center = (uiv1.CenterWidget())
                ..children.add(
                  uiv1.Node()
                    ..text = (uiv1.TextWidget()..value = 'Centered child'),
                ),
              uiv1.Node()
                ..grid = (uiv1.GridWidget()..columns = 2)
                ..children.addAll([
                  uiv1.Node()..text = (uiv1.TextWidget()..value = 'Cell A'),
                  uiv1.Node()..text = (uiv1.TextWidget()..value = 'Cell B'),
                ]),
              uiv1.Node()..progress = (uiv1.ProgressWidget()..value = 0.4),
            ]))),
    );
    await tester.pump();

    expect(find.byType(Wrap), findsWidgets);
    expect(find.byType(LinearProgressIndicator), findsOneWidget);
    expect(
      find.byWidgetPredicate(
        (widget) =>
            widget is Padding && widget.padding == const EdgeInsets.all(16),
      ),
      findsOneWidget,
    );
    expect(find.text('Cell A'), findsOneWidget);
    expect(find.text('Cell B'), findsOneWidget);
    expect(find.text('Centered child'), findsOneWidget);
  });

  testWidgets('renders expand primitive as Expanded widget',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine),
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
                ..expand = (uiv1.ExpandWidget())
                ..children.add(
                  uiv1.Node()
                    ..text = (uiv1.TextWidget()..value = 'Expanded child'),
                ),
            ))),
    );
    await tester.pump();

    expect(find.byType(Expanded), findsOneWidget);
    expect(find.text('Expanded child'), findsOneWidget);
  });

  testWidgets('renders image primitive as Image widget',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine),
    );
    await tester.tap(find.text('Connect Stream'));
    await tester.pump();

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..setUi = (uiv1.SetUI()
          ..root = (uiv1.Node()
            ..stack = (uiv1.StackWidget())
            ..children.add(
              uiv1.Node()
                ..image = (uiv1.ImageWidget()
                  ..url = 'https://example.com/camera.jpg'),
            ))),
    );
    await tester.pump();

    expect(find.byType(Image), findsOneWidget);
  });

  testWidgets('wires slider, toggle, and dropdown actions to UIAction',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine),
    );
    await tester.tap(find.text('Connect Stream'));
    await tester.pump();

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..setUi = (uiv1.SetUI()
          ..root = (uiv1.Node()
            ..id = 'root'
            ..stack = (uiv1.StackWidget())
            ..children.addAll([
              uiv1.Node()
                ..id = 'volume'
                ..slider = (uiv1.SliderWidget()
                  ..min = 0
                  ..max = 1
                  ..value = 0.25),
              uiv1.Node()
                ..id = 'mute'
                ..toggle = (uiv1.ToggleWidget()..value = false),
              uiv1.Node()
                ..id = 'channel'
                ..dropdown = (uiv1.DropdownWidget()
                  ..options.addAll(['alpha', 'beta'])
                  ..value = 'alpha'),
            ]))),
    );
    await tester.pump();

    final slider = tester.widget<Slider>(find.byType(Slider));
    slider.onChanged?.call(0.75);
    await tester.pump();

    final toggle = tester.widget<SwitchListTile>(find.byType(SwitchListTile));
    toggle.onChanged?.call(true);
    await tester.pump();

    final dropdown = tester
        .widgetList<DropdownButton<String>>(find.byType(DropdownButton<String>))
        .firstWhere((candidate) => candidate.value == 'alpha');
    dropdown.onChanged?.call('beta');
    await tester.pump();

    final actions = harness.lastClient.requests
        .where((request) => request.hasInput() && request.input.hasUiAction())
        .map((request) => request.input.uiAction)
        .toList();

    expect(
      actions.any(
        (action) =>
            action.componentId == 'volume' &&
            action.action == 'change' &&
            action.value == '0.75',
      ),
      isTrue,
    );
    expect(
      actions.any(
        (action) =>
            action.componentId == 'mute' &&
            action.action == 'toggle' &&
            action.value == 'true',
      ),
      isTrue,
    );
    expect(
      actions.any(
        (action) =>
            action.componentId == 'channel' &&
            action.action == 'select' &&
            action.value == 'beta',
      ),
      isTrue,
    );
  });

  testWidgets('wires terminal text input changes/submission to key events',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine),
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
                ..id = 'terminal_input'
                ..textInput =
                    (uiv1.TextInputWidget()..placeholder = 'Terminal prompt'),
            ))),
    );
    await tester.pump();

    final terminalField = find.byWidgetPredicate(
      (widget) =>
          widget is TextField &&
          widget.decoration?.hintText == 'Terminal prompt',
    );
    expect(terminalField, findsOneWidget);

    await tester.enterText(terminalField, 'ls');
    await tester.pump();
    await tester.testTextInput.receiveAction(TextInputAction.done);
    await tester.pump();

    final keyEvents = harness.lastClient.requests
        .where((request) => request.hasInput() && request.input.hasKey())
        .map((request) => request.input.key.text)
        .toList();
    expect(keyEvents.any((text) => text == 'ls'), isTrue);
    expect(keyEvents.any((text) => text == '\n'), isTrue);

    final terminalSubmitActions = harness.lastClient.requests
        .where((request) => request.hasInput() && request.input.hasUiAction())
        .map((request) => request.input.uiAction)
        .where(
          (action) =>
              action.componentId == 'terminal_input' &&
              action.action == 'submit',
        )
        .toList();
    expect(terminalSubmitActions, isEmpty);
  });

  testWidgets('keeps terminal input focused across terminal output patches',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine),
    );
    await tester.tap(find.text('Connect Stream'));
    await tester.pump();

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..setUi = (uiv1.SetUI()
          ..root = (uiv1.Node()
            ..id = 'terminal_root'
            ..stack = (uiv1.StackWidget())
            ..children.addAll([
              uiv1.Node()
                ..id = 'terminal_output'
                ..text = (uiv1.TextWidget()..value = 'repl> '),
              uiv1.Node()
                ..id = 'terminal_input'
                ..textInput =
                    (uiv1.TextInputWidget()..placeholder = 'Terminal prompt'),
            ]))),
    );
    await tester.pump();

    final terminalField = find.byWidgetPredicate(
      (widget) =>
          widget is TextField &&
          widget.decoration?.hintText == 'Terminal prompt',
    );
    expect(terminalField, findsOneWidget);

    await tester.tap(terminalField);
    await tester.pump();

    EditableText editableText = tester.widget<EditableText>(
      find.descendant(of: terminalField, matching: find.byType(EditableText)),
    );
    expect(editableText.focusNode.hasFocus, isTrue);

    await tester.enterText(terminalField, 'h');
    await tester.pump();

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..updateUi = (uiv1.UpdateUI()
          ..componentId = 'terminal_output'
          ..node = (uiv1.Node()
            ..id = 'terminal_output'
            ..text = (uiv1.TextWidget()..value = 'repl> h'))),
    );
    await tester.pump();

    editableText = tester.widget<EditableText>(
      find.descendant(of: terminalField, matching: find.byType(EditableText)),
    );
    expect(editableText.focusNode.hasFocus, isTrue);

    await tester.enterText(terminalField, 'he');
    await tester.pump();

    final keyEvents = harness.lastClient.requests
        .where((request) => request.hasInput() && request.input.hasKey())
        .map((request) => request.input.key.text)
        .toList();
    expect(keyEvents.where((text) => text == 'h').length, 1);
    expect(keyEvents.where((text) => text == 'e').length, 1);
  });

  testWidgets('renders overlay primitive with layered children',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine),
    );
    await tester.tap(find.text('Connect Stream'));
    await tester.pump();

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..setUi = (uiv1.SetUI()
          ..root = (uiv1.Node()
            ..stack = (uiv1.StackWidget())
            ..children.add(
              uiv1.Node()
                ..id = 'alert_overlay'
                ..overlay = (uiv1.OverlayWidget())
                ..children.addAll([
                  uiv1.Node()
                    ..text = (uiv1.TextWidget()..value = 'Base content'),
                  uiv1.Node()
                    ..text = (uiv1.TextWidget()..value = 'Overlay content'),
                ]),
            ))),
    );
    await tester.pump();

    expect(
      find.byKey(const ValueKey<String>('ui-overlay-alert_overlay')),
      findsOneWidget,
    );
    expect(find.text('Base content'), findsOneWidget);
    expect(find.text('Overlay content'), findsOneWidget);
  });

  testWidgets('renders PA overlay when global overlay slot is patched',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine),
    );
    await tester.tap(find.text('Connect Stream'));
    await tester.pump();

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..setUi = (uiv1.SetUI()
          ..root = (uiv1.Node()
            ..id = 'root'
            ..stack = (uiv1.StackWidget())
            ..children.addAll([
              uiv1.Node()..text = (uiv1.TextWidget()..value = 'Base content'),
              uiv1.Node()
                ..id = 'global_overlay'
                ..overlay = (uiv1.OverlayWidget()),
            ]))),
    );
    await tester.pump();

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..updateUi = (uiv1.UpdateUI()
          ..componentId = 'global_overlay'
          ..node = (uiv1.Node()
            ..id = 'global_overlay'
            ..overlay = (uiv1.OverlayWidget())
            ..children.add(
              uiv1.Node()
                ..text = (uiv1.TextWidget()..value = 'PA from device-1'),
            ))),
    );
    await tester.pump();

    expect(
      find.byKey(const ValueKey<String>('ui-overlay-global_overlay')),
      findsOneWidget,
    );
    expect(find.text('Base content'), findsOneWidget);
    expect(find.text('PA from device-1'), findsOneWidget);
  });

  testWidgets('wires gesture area tap to UIAction',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine),
    );
    await tester.tap(find.text('Connect Stream'));
    await tester.pump();

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..setUi = (uiv1.SetUI()
          ..root = (uiv1.Node()
            ..stack = (uiv1.StackWidget())
            ..children.add(
              uiv1.Node()
                ..id = 'gesture_tap_zone'
                ..gestureArea = (uiv1.GestureAreaWidget()..action = 'announce')
                ..children.add(
                  uiv1.Node()..text = (uiv1.TextWidget()..value = 'Tap me'),
                ),
            ))),
    );
    await tester.pump();

    final gesture = tester.widget<GestureDetector>(
      find.byKey(const ValueKey<String>('ui-gesture-gesture_tap_zone')),
    );
    gesture.onTap?.call();
    await tester.pump();

    final actions = harness.lastClient.requests
        .where((request) => request.hasInput() && request.input.hasUiAction())
        .map((request) => request.input.uiAction)
        .toList();
    expect(
      actions.any(
        (action) =>
            action.componentId == 'gesture_tap_zone' &&
            action.action == 'announce' &&
            action.value.isEmpty,
      ),
      isTrue,
    );
  });

  testWidgets('renders media and system hint primitives with live widgets',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine),
    );
    await tester.tap(find.text('Connect Stream'));
    await tester.pump();

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..setUi = (uiv1.SetUI()
          ..root = (uiv1.Node()
            ..scroll = (uiv1.ScrollWidget())
            ..children.addAll([
              uiv1.Node()
                ..id = 'camera_a'
                ..videoSurface =
                    (uiv1.VideoSurfaceWidget()..trackId = 'track-a'),
              uiv1.Node()
                ..id = 'mic_mix'
                ..audioVisualizer =
                    (uiv1.AudioVisualizerWidget()..streamId = 'stream-mix'),
              uiv1.Node()
                ..id = 'drawing'
                ..canvas = (uiv1.CanvasWidget()
                  ..drawOpsJson = '{"ops":[{"line":"x"}]}'),
              uiv1.Node()
                ..id = 'fs_hint'
                ..fullscreen = (uiv1.FullscreenWidget()..enabled = true)
                ..children.add(
                  uiv1.Node()
                    ..text = (uiv1.TextWidget()..value = 'Fullscreen body'),
                ),
              uiv1.Node()
                ..id = 'awake_hint'
                ..keepAwake = (uiv1.KeepAwakeWidget()..enabled = true)
                ..children.add(
                  uiv1.Node()..text = (uiv1.TextWidget()..value = 'Awake body'),
                ),
              uiv1.Node()
                ..id = 'brightness_hint'
                ..brightness = (uiv1.BrightnessWidget()..value = 0.8)
                ..children.add(
                  uiv1.Node()
                    ..text = (uiv1.TextWidget()..value = 'Brightness body'),
                ),
            ]))),
    );
    await tester.pump();

    expect(find.byKey(const ValueKey<String>('ui-video-surface-camera_a')),
        findsOneWidget);
    expect(find.byKey(const ValueKey<String>('ui-audio-visualizer-mic_mix')),
        findsOneWidget);
    expect(find.byKey(const ValueKey<String>('ui-canvas-drawing')),
        findsOneWidget);
    expect(find.byKey(const ValueKey<String>('ui-fullscreen-fs_hint')),
        findsOneWidget);
    expect(find.byKey(const ValueKey<String>('ui-keep-awake-awake_hint')),
        findsOneWidget);
    expect(
      find.byKey(const ValueKey<String>('ui-brightness-brightness_hint')),
      findsOneWidget,
    );
    expect(find.text('track-a'), findsOneWidget);
    expect(find.text('Fullscreen body'), findsOneWidget);
    expect(find.text('Awake body'), findsOneWidget);
    expect(find.text('Brightness body'), findsOneWidget);
  });

  testWidgets(
    'handles media control responses and acknowledges started streams',
    (WidgetTester tester) async {
      await tester.binding.setSurfaceSize(const Size(1200, 1400));
      addTearDown(() => tester.binding.setSurfaceSize(null));
      final harness = _FakeClientHarness();
      await tester.pumpWidget(
        TerminalClientApp(
            clientFactory: harness.createClient,
            mediaEngineFactory: harness.createMediaEngine),
      );
      await tester.tap(find.text('Connect Stream'));
      await tester.pump();

      harness.lastClient.emitResponse(
        ConnectResponse()
          ..startStream = (iov1.StartStream()
            ..streamId = 'stream-a'
            ..kind = 'audio'
            ..sourceDeviceId = 'device-1'
            ..targetDeviceId = 'device-2'),
      );
      await tester.pump();

      expect(find.textContaining('Control Stream: Stream started'),
          findsOneWidget);
      expect(find.textContaining('Active streams: 1'), findsOneWidget);
      expect(find.textContaining('Start stream: audio (stream-a)'),
          findsOneWidget);
      expect(find.textContaining('Stream-ready acks: 1'), findsOneWidget);

      final readyRequests = harness.lastClient.requests
          .where((request) => request.hasStreamReady())
          .toList();
      expect(readyRequests.length, 1);
      expect(readyRequests.first.streamReady.streamId, 'stream-a');
      final localOfferRequests = harness.lastClient.requests
          .where((request) =>
              request.hasWebrtcSignal() &&
              request.webrtcSignal.streamId == 'stream-a' &&
              request.webrtcSignal.signalType == 'offer')
          .toList();
      expect(localOfferRequests.length, 1);
      final initialCandidateRequests = harness.lastClient.requests
          .where((request) =>
              request.hasWebrtcSignal() &&
              request.webrtcSignal.streamId == 'stream-a' &&
              request.webrtcSignal.signalType == 'candidate')
          .toList();
      expect(initialCandidateRequests.length, 1);

      harness.lastClient.emitResponse(
        ConnectResponse()
          ..routeStream = (iov1.RouteStream()
            ..streamId = 'stream-a'
            ..sourceDeviceId = 'device-1'
            ..targetDeviceId = 'device-2'
            ..kind = 'audio'),
      );
      await tester.pump();
      expect(
          find.textContaining('Control Stream: Route updated'), findsOneWidget);
      expect(find.textContaining('Media routes: 1'), findsOneWidget);
      expect(
        find.textContaining('Route: device-1 -> device-2 (audio)'),
        findsOneWidget,
      );

      harness.lastClient.emitResponse(
        ConnectResponse()
          ..webrtcSignal = (WebRTCSignal()
            ..streamId = 'stream-a'
            ..signalType = 'offer'
            ..payload = 'sdp-offer'),
      );
      await tester.pump();
      expect(
          find.textContaining('Control Stream: WebRTC signal'), findsOneWidget);
      expect(find.textContaining('Signals: 1'), findsOneWidget);
      expect(find.textContaining('WebRTC signal: offer (stream-a)'),
          findsOneWidget);
      final localAnswerRequests = harness.lastClient.requests
          .where((request) =>
              request.hasWebrtcSignal() &&
              request.webrtcSignal.streamId == 'stream-a' &&
              request.webrtcSignal.signalType == 'answer')
          .toList();
      expect(localAnswerRequests.length, 1);
      final localCandidateRequests = harness.lastClient.requests
          .where((request) =>
              request.hasWebrtcSignal() &&
              request.webrtcSignal.streamId == 'stream-a' &&
              request.webrtcSignal.signalType == 'candidate')
          .toList();
      expect(localCandidateRequests.length, 2);

      harness.lastClient.emitResponse(
        ConnectResponse()
          ..stopStream = (iov1.StopStream()..streamId = 'stream-a'),
      );
      await tester.pump();
      expect(find.textContaining('Control Stream: Stream stopped'),
          findsOneWidget);
      expect(find.textContaining('Active streams: 0'), findsOneWidget);
      expect(find.textContaining('Media routes: 0'), findsOneWidget);
      expect(find.textContaining('Stop stream: stream-a'), findsOneWidget);
    },
  );

  testWidgets('handles play_audio responses and tracks playback status',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine),
    );
    await tester.tap(find.text('Connect Stream'));
    await tester.pump();

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..playAudio = (iov1.PlayAudio()
          ..requestId = 'playback-1'
          ..deviceId = 'hall-display'
          ..pcmData = <int>[1, 2, 3, 4, 5]),
    );
    await tester.pump();

    expect(find.textContaining('Control Stream: Play audio'), findsOneWidget);
    expect(find.textContaining('Play audio msgs: 1'), findsOneWidget);
    expect(find.textContaining('Last play bytes: 5'), findsOneWidget);
    expect(
        find.textContaining('Last play target: hall-display'), findsOneWidget);
    expect(find.textContaining('Last play source: pcm_data'), findsOneWidget);
    expect(
      find.textContaining('Play audio: hall-display (pcm_data, 5 bytes)'),
      findsOneWidget,
    );
  });

  testWidgets('responds to request_artifact after local artifact persistence',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine),
    );
    await tester.tap(find.text('Connect Stream'));
    await tester.pump();

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..playAudio = (iov1.PlayAudio()
          ..requestId = 'artifact-001'
          ..deviceId = 'hall-display'
          ..pcmData = <int>[7, 8, 9]),
    );
    await tester.pumpAndSettle();

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..requestArtifact =
            (iov1.RequestArtifact()..artifactId = 'play_audio/artifact-001'),
    );
    await tester.pumpAndSettle();

    final artifactAvailableRequest = harness.lastClient.requests.lastWhere(
      (request) => request.hasArtifactAvailable(),
    );
    expect(
      artifactAvailableRequest.artifactAvailable.artifact.id,
      'play_audio/artifact-001',
    );
    expect(
      artifactAvailableRequest.artifactAvailable.artifact.source.deviceId,
      isNotEmpty,
    );
    expect(find.textContaining('Artifact available: play_audio/artifact-001'),
        findsOneWidget);
  });

  testWidgets(
      'sends system and playback debug commands and renders diagnostics data',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine),
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

    expect(find.textContaining('Diagnostics: none'), findsOneWidget);

    await tester.tap(find.text('Runtime Status'));
    await tester.pump();
    final runtimeRequest = harness.lastClient.requests.lastWhere(
      (request) =>
          request.hasCommand() &&
          request.command.kind == CommandKind.COMMAND_KIND_SYSTEM &&
          request.command.intent == 'runtime_status',
    );
    final runtimeRequestID = runtimeRequest.command.requestId;
    expect(runtimeRequestID, isNotEmpty);

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..commandResult = (CommandResult()
          ..requestId = runtimeRequestID
          ..notification = 'System query: runtime_status'
          ..data.addAll({
            'active_routes': '1',
            'media_streams_active': '2',
          })),
    );
    await tester.pump();
    expect(find.textContaining('Diagnostics: runtime_status'), findsOneWidget);
    expect(find.textContaining('active_routes=1'), findsOneWidget);
    expect(find.textContaining('media_streams_active=2'), findsOneWidget);
    expect(find.text('terminal (REPL)'), findsOneWidget);

    await tester.tap(find.text('Refresh Applications'));
    await tester.pump();
    final appRegistryRequest = harness.lastClient.requests.lastWhere(
      (request) =>
          request.hasCommand() &&
          request.command.kind == CommandKind.COMMAND_KIND_SYSTEM &&
          request.command.intent == 'scenario_registry',
    );
    final appRegistryRequestID = appRegistryRequest.command.requestId;
    expect(appRegistryRequestID, isNotEmpty);

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..commandResult = (CommandResult()
          ..requestId = appRegistryRequestID
          ..notification = 'System query: scenario_registry'
          ..data.addAll({
            'red_alert': 'priority=100',
            'intercom': 'priority=50',
          })),
    );
    await tester.pumpAndSettle();
    expect(
        find.textContaining('Diagnostics: scenario_registry'), findsOneWidget);

    await tester.tap(
      find.byWidgetPredicate(
        (widget) =>
            widget is DropdownButtonFormField<String> &&
            widget.decoration.labelText == 'Available Application',
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('red_alert').last);
    await tester.pumpAndSettle();

    await tester.tap(find.text('Open Application'));
    await tester.pump();
    final launchRequest = harness.lastClient.requests.lastWhere(
      (request) =>
          request.hasCommand() &&
          request.command.kind == CommandKind.COMMAND_KIND_MANUAL &&
          request.command.intent == 'red_alert',
    );
    expect(launchRequest.command.action, CommandAction.COMMAND_ACTION_START);
    expect(launchRequest.command.deviceId, isNotEmpty);

    await tester.tap(find.text('Device Status'));
    await tester.pump();
    final deviceRequest = harness.lastClient.requests.lastWhere(
      (request) =>
          request.hasCommand() &&
          request.command.kind == CommandKind.COMMAND_KIND_SYSTEM &&
          request.command.intent.startsWith('device_status '),
    );
    final deviceRequestID = deviceRequest.command.requestId;
    expect(deviceRequestID, isNotEmpty);

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..commandResult = (CommandResult()
          ..requestId = deviceRequestID
          ..notification = 'System query: device_status'
          ..data.addAll({
            'device_id': 'flutter-test-device',
            'sensor.unix_ms': '1713000009999',
          })),
    );
    await tester.pump();
    expect(find.textContaining('Diagnostics: device_status'), findsOneWidget);
    expect(
        find.textContaining('device_id=flutter-test-device'), findsOneWidget);
    expect(find.textContaining('sensor.unix_ms=1713000009999'), findsOneWidget);

    await tester.tap(find.text('List Playback Artifacts'));
    await tester.pump();
    final playbackArtifactsRequest = harness.lastClient.requests.lastWhere(
      (request) =>
          request.hasCommand() &&
          request.command.kind == CommandKind.COMMAND_KIND_SYSTEM &&
          request.command.intent == 'list_playback_artifacts',
    );
    final playbackArtifactsRequestID =
        playbackArtifactsRequest.command.requestId;
    expect(playbackArtifactsRequestID, isNotEmpty);

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..commandResult = (CommandResult()
          ..requestId = playbackArtifactsRequestID
          ..notification = 'System query: list_playback_artifacts'
          ..data.addAll({
            '000':
                'route:device-a|device-b|audio|device-a|device-b|128|1713000011111|/tmp/audio-1.pcm',
          })),
    );
    await tester.pump();
    expect(find.textContaining('Diagnostics: list_playback_artifacts'),
        findsOneWidget);
    expect(
      find.textContaining(
          '000=route:device-a|device-b|audio|device-a|device-b|128|1713000011111|/tmp/audio-1.pcm'),
      findsOneWidget,
    );
    expect(find.text('route:device-a'), findsOneWidget);

    await tester.enterText(
      find.byWidgetPredicate(
        (widget) =>
            widget is TextField &&
            widget.decoration?.labelText == 'Playback Target Device ID',
      ),
      'kitchen-display',
    );
    await tester.pump();
    await tester.tap(find.text('Playback Metadata'));
    await tester.pump();

    final playbackMetadataRequest = harness.lastClient.requests.lastWhere(
      (request) =>
          request.hasCommand() &&
          request.command.kind == CommandKind.COMMAND_KIND_MANUAL &&
          request.command.intent == 'playback_metadata',
    );
    final playbackMetadataRequestID = playbackMetadataRequest.command.requestId;
    expect(playbackMetadataRequestID, isNotEmpty);
    expect(playbackMetadataRequest.command.deviceId, isNotEmpty);
    expect(
      playbackMetadataRequest.command.arguments['artifact_id'],
      'route:device-a',
    );
    expect(
      playbackMetadataRequest.command.arguments['target_device_id'],
      'kitchen-display',
    );

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..commandResult = (CommandResult()
          ..requestId = playbackMetadataRequestID
          ..notification = 'Playback metadata ready'
          ..data.addAll({
            'artifact_id': 'route:device-a',
            'target_device_id': 'kitchen-display',
            'audio_path': '/tmp/audio-1.pcm',
          })),
    );
    await tester.pump();
    expect(
        find.textContaining('Diagnostics: playback_metadata'), findsOneWidget);
    expect(find.textContaining('artifact_id=route:device-a'), findsOneWidget);
    expect(find.textContaining('target_device_id=kitchen-display'),
        findsOneWidget);
  });

  testWidgets('open application queues launch until register ack',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine),
    );

    await tester.tap(find.text('Open Application'));
    await tester.pump();

    expect(
      find.textContaining('Connecting control stream to open application:'),
      findsOneWidget,
    );
    expect(harness.createdClients, isNotEmpty);

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..registerAck = (RegisterAck()
          ..serverId = 'test-server'
          ..message = 'registered'),
    );
    await tester.pump();

    final launchRequest = harness.lastClient.requests.lastWhere(
      (request) =>
          request.hasCommand() &&
          request.command.kind == CommandKind.COMMAND_KIND_MANUAL &&
          request.command.intent == 'terminal',
    );
    expect(launchRequest.command.action, CommandAction.COMMAND_ACTION_START);
    expect(
        find.textContaining('Launching application: terminal'), findsOneWidget);
  });

  testWidgets('terminal root renders fullscreen app view',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine),
    );

    await tester.tap(find.text('Connect Stream'));
    await tester.pump();

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..setUi = (uiv1.SetUI()
          ..root = (uiv1.Node()
            ..id = 'terminal_root'
            ..stack = (uiv1.StackWidget())
            ..children.add(
              uiv1.Node()
                ..id = 'terminal_output'
                ..text = (uiv1.TextWidget()..value = 'repl>'),
            ))),
    );
    await tester.pump();

    expect(find.text('repl>'), findsOneWidget);
    expect(find.text('Server Host'), findsNothing);
    expect(find.text('Connect Stream'), findsNothing);
  });
}

class _FakeClientHarness {
  _FakeClientHarness({
    this.failFirstConnectStream = false,
    this.failConnectAttempts = 0,
  });

  final List<_FakeTerminalControlClient> createdClients =
      <_FakeTerminalControlClient>[];
  final List<_FakeMediaEngine> createdMediaEngines = <_FakeMediaEngine>[];
  final List<ControlCarrierKind> requestedCarriers = <ControlCarrierKind>[];
  final bool failFirstConnectStream;
  final int failConnectAttempts;

  _FakeTerminalControlClient get lastClient => createdClients.last;

  TerminalControlClient createClient({
    required String host,
    required int port,
  }) {
    requestedCarriers.add(ControlClientTransportHint.preferredCarrier);
    final client = _FakeTerminalControlClient(
      host: host,
      port: port,
      failOnConnectStream: createdClients.length < failConnectAttempts ||
          (failFirstConnectStream && createdClients.isEmpty),
    );
    createdClients.add(client);
    return client;
  }

  ClientMediaEngine createMediaEngine({
    required String localDeviceID,
    required OutboundSignalCallback onSignal,
  }) {
    final engine = _FakeMediaEngine(
      localDeviceID: localDeviceID,
      onSignal: onSignal,
    );
    createdMediaEngines.add(engine);
    return engine;
  }
}

class _FakeMediaEngine implements ClientMediaEngine {
  _FakeMediaEngine({
    required this.localDeviceID,
    required this.onSignal,
  });

  final String localDeviceID;
  final OutboundSignalCallback onSignal;
  final Set<String> _activeStreamIDs = <String>{};
  final Map<String, ValueNotifier<MediaStream?>> _remoteStreamsByID =
      <String, ValueNotifier<MediaStream?>>{};
  final Map<String, ValueNotifier<double>> _audioLevelsByID =
      <String, ValueNotifier<double>>{};

  @override
  Future<void> startStream(iov1.StartStream start) async {
    if (start.streamId.isEmpty || _activeStreamIDs.contains(start.streamId)) {
      return;
    }
    _activeStreamIDs.add(start.streamId);
    _remoteStreamsByID.putIfAbsent(
      start.streamId,
      () => ValueNotifier<MediaStream?>(null),
    );
    _audioLevelsByID.putIfAbsent(
      start.streamId,
      () => ValueNotifier<double>(0.5),
    );
    onSignal(
      WebRTCSignal()
        ..streamId = start.streamId
        ..signalType = 'offer'
        ..payload = '{"sdp":"fake-offer-for-${start.streamId}"}',
    );
    onSignal(
      WebRTCSignal()
        ..streamId = start.streamId
        ..signalType = 'candidate'
        ..payload = '{"candidate":"fake-local-candidate-1"}',
    );
  }

  @override
  Future<void> stopStream(String streamID) async {
    _activeStreamIDs.remove(streamID);
    _remoteStreamsByID
        .putIfAbsent(
          streamID,
          () => ValueNotifier<MediaStream?>(null),
        )
        .value = null;
    _audioLevelsByID
        .putIfAbsent(
          streamID,
          () => ValueNotifier<double>(0.0),
        )
        .value = 0.0;
  }

  @override
  Future<void> handleSignal(WebRTCSignal signal) async {
    if (signal.signalType.trim().toLowerCase() != 'offer') {
      return;
    }
    onSignal(
      WebRTCSignal()
        ..streamId = signal.streamId
        ..signalType = 'answer'
        ..payload = '{"sdp":"fake-answer-for-${signal.streamId}"}',
    );
    onSignal(
      WebRTCSignal()
        ..streamId = signal.streamId
        ..signalType = 'candidate'
        ..payload = '{"candidate":"fake-local-candidate-2"}',
    );
  }

  @override
  Future<void> dispose() async {}

  @override
  ValueListenable<MediaStream?> remoteStream(String streamID) {
    return _remoteStreamsByID.putIfAbsent(
      streamID,
      () => ValueNotifier<MediaStream?>(null),
    );
  }

  @override
  ValueListenable<double> audioLevel(String streamID) {
    return _audioLevelsByID.putIfAbsent(
      streamID,
      () => ValueNotifier<double>(0.0),
    );
  }
}

class _FakeTerminalControlClient implements TerminalControlClient {
  _FakeTerminalControlClient({
    required this.host,
    required this.port,
    required this.failOnConnectStream,
  });

  final String host;
  final int port;
  final bool failOnConnectStream;
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
    if (failOnConnectStream) {
      return Stream<ConnectResponse>.error(StateError('stream dropped'));
    }
    return _responses.stream;
  }

  @override
  Future<void> shutdown() async {
    await _requestSubscription?.cancel();
    if (!_responses.isClosed) {
      await _responses.close();
    }
  }

  void emitResponse(ConnectResponse response) {
    _responses.add(response);
  }

  Future<void> closeStream() async {
    if (!_responses.isClosed) {
      await _responses.close();
    }
  }
}
