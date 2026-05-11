import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/app/terminal_client_app.dart';
import 'package:terminal_client/capabilities/screen_metrics.dart';
import 'package:terminal_client/gen/terminals/capabilities/v1/capabilities.pb.dart'
    as capv1;
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';
import 'package:terminal_client/gen/terminals/ui/v1/ui.pb.dart' as uiv1;

import 'widget_test_helpers.dart';

void main() {
  testWidgets('app renders MaterialApp', (WidgetTester tester) async {
    await tester.pumpWidget(const TerminalClientApp());
    expect(find.byType(MaterialApp), findsOneWidget);
  });

  testWidgets('auto-connect starts stream on launch when enabled',
      (WidgetTester tester) async {
    final harness = FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        mediaEngineFactory: harness.createMediaEngine,
        autoConnectOnStartup: true,
      ),
    );
    await tester.pump(const Duration(milliseconds: 300));

    expect(harness.createdClients, isNotEmpty);
  });

  testWidgets(
    'notification envelope triggers alert delivery callback only for explicit notifications',
    (WidgetTester tester) async {
      final harness = FakeClientHarness();
      final deliveredAlerts = <String>[];

      await tester.pumpWidget(
        TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine,
          alertDelivery: ({
            required String title,
            required String body,
            required String level,
          }) {
            deliveredAlerts.add('$title|$body|$level');
          },
        ),
      );

      await tester.tap(find.text('Connect Stream'));
      await tester.pump();

      harness.lastClient.emitResponse(
        ConnectResponse()
          ..notification = (uiv1.Notification()
            ..title = 'Timer'
            ..body = 'Dishwasher finished'
            ..level = 'info'),
      );
      await tester.pump();

      expect(deliveredAlerts, <String>['Timer|Dishwasher finished|info']);

      harness.lastClient.emitResponse(
        ConnectResponse()
          ..commandResult =
              (CommandResult()..notification = 'in-app status only'),
      );
      await tester.pump();

      expect(deliveredAlerts, <String>['Timer|Dishwasher finished|info']);
    },
  );

  testWidgets('app shows build metadata footer', (WidgetTester tester) async {
    await tester.pumpWidget(const TerminalClientApp());
    expect(find.textContaining('Build:'), findsWidgets);
    expect(find.textContaining('SHA:'), findsWidgets);
    expect(find.textContaining('Client / Server Build'), findsOneWidget);
    expect(find.textContaining('Build Match:'), findsOneWidget);
    expect(
      find.textContaining('Server Build: awaiting register ack'),
      findsOneWidget,
    );
    expect(find.textContaining('Control Stream:'), findsOneWidget);
    expect(find.text('Connect Stream'), findsOneWidget);
    expect(
      find.byWidgetPredicate(
        (widget) =>
            widget is SelectableText &&
            (widget.data?.contains('Control Stream:') ?? false),
      ),
      findsOneWidget,
    );
  });

  testWidgets('transport diagnostics block is selectable after failure', (
    WidgetTester tester,
  ) async {
    final harness = FakeClientHarness(failConnectAttempts: 1);
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        mediaEngineFactory: harness.createMediaEngine,
        reconnectDelayBase: const Duration(milliseconds: 30),
        reconnectDelayMaxSeconds: 1,
      ),
    );

    final connectButton = find.widgetWithText(ElevatedButton, 'Connect Stream');
    await tester.ensureVisible(connectButton);
    await tester.tap(connectButton);
    await tester.pump();
    for (var i = 0; i < 80; i++) {
      await tester.pump(const Duration(milliseconds: 10));
    }

    expect(
      find.byWidgetPredicate(
        (widget) =>
            widget is SelectableText &&
            (widget.data?.contains('Transport Diagnostics') ?? false),
      ),
      findsOneWidget,
    );
  });

  testWidgets('shows server build metadata after register ack', (
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
          ..message = 'registered'
          ..metadata.addAll({
            'server_build_sha': 'ea99b3f38658',
            'server_build_date': '2026-04-21T14:55:56Z',
          })),
    );
    await tester.pump();

    expect(find.textContaining('Client / Server Build'), findsOneWidget);
    expect(
      find.textContaining(
          'Server Build: 2026-04-21T14:55:56Z | SHA: ea99b3f38658'),
      findsOneWidget,
    );
    expect(find.textContaining('Build Match:'), findsOneWidget);
  });

  testWidgets('sends periodic heartbeat messages while connected', (
    WidgetTester tester,
  ) async {
    final harness = FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        mediaEngineFactory: harness.createMediaEngine,
        heartbeatInterval: const Duration(milliseconds: 40),
      ),
    );

    await tester.tap(find.text('Connect Stream'));
    await tester.pump();
    harness.lastClient.emitResponse(
      ConnectResponse()..registerAck = (RegisterAck()),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 140));

    final requests = harness.lastClient.requests;
    final heartbeatCount = requests.where((r) => r.hasHeartbeat()).length;
    expect(heartbeatCount, greaterThanOrEqualTo(2));
    expect(requests.where((r) => r.hasHello()).length, 1);
    expect(requests.where((r) => r.hasCapabilitySnapshot()).length, 1);
  });

  testWidgets(
    'connect emits CapabilitySnapshot with ScreenCapability geometry metadata',
    (WidgetTester tester) async {
      final harness = FakeClientHarness();
      await tester.pumpWidget(
        TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine,
        ),
      );
      final mediaQuery = tester.widget<MediaQuery>(
        find.byType(MediaQuery).first,
      );
      final expectedWidth = mediaQuery.data.size.width.round();
      final expectedHeight = mediaQuery.data.size.height.round();
      final expectedOrientation =
          expectedWidth >= expectedHeight ? 'landscape' : 'portrait';
      final expectedDensity = mediaQuery.data.devicePixelRatio;
      final expectedSafeArea = mediaQuery.data.viewPadding;

      await tester.tap(find.text('Connect Stream'));
      await tester.pump();

      final snapshot = harness.lastClient.requests
          .firstWhere((request) => request.hasCapabilitySnapshot())
          .capabilitySnapshot;
      final screen = snapshot.capabilities.screen;
      expect(snapshot.capabilities.hasScreen(), isTrue);
      expect(screen.width, expectedWidth);
      expect(screen.height, expectedHeight);
      expect(screen.orientation, expectedOrientation);
      expect(screen.density, expectedDensity);
      expect(screen.hasFullscreenSupported(), isFalse);
      expect(screen.hasMultiWindowSupported(), isFalse);
      expect(screen.hasSafeArea(), isTrue);
      expect(screen.safeArea.left, expectedSafeArea.left.round());
      expect(screen.safeArea.top, expectedSafeArea.top.round());
      expect(screen.safeArea.right, expectedSafeArea.right.round());
      expect(screen.safeArea.bottom, expectedSafeArea.bottom.round());
      expect(snapshot.capabilities.displays, isNotEmpty);
      expect(snapshot.capabilities.displays.first.hasPrimary(), isFalse);
    },
  );

  testWidgets(
    'deterministic metrics seam emits CapabilityDelta on rotation with fresh generation',
    (WidgetTester tester) async {
      final harness = FakeClientHarness();
      final metrics = TestScreenMetricsController(
        ScreenMetrics(
          logicalSize: Size(1280, 720),
          devicePixelRatio: 2.0,
        ),
      );
      await tester.pumpWidget(
        TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine,
          screenMetricsProvider: metrics.read,
          screenMetricsChangeListenable: metrics.changes,
          displayGeometryDebounceInterval: const Duration(milliseconds: 80),
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
      await tester.pumpAndSettle();

      final snapshot = harness.lastClient.requests
          .firstWhere((request) => request.hasCapabilitySnapshot())
          .capabilitySnapshot;
      expect(snapshot.capabilities.screen.orientation, 'landscape');

      final deltaCountBefore = harness.lastClient.requests
          .where((request) => request.hasCapabilityDelta())
          .length;
      metrics.update(
        ScreenMetrics(
          logicalSize: Size(720, 1280),
          devicePixelRatio: 2.0,
        ),
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 80));

      final deltas = harness.lastClient.requests
          .where((request) => request.hasCapabilityDelta())
          .toList();
      expect(deltas.length, deltaCountBefore + 1);
      final delta = deltas.last.capabilityDelta;
      expect(delta.reason, 'display_geometry_change');
      expect(delta.generation, greaterThan(snapshot.generation));
      expect(delta.capabilities.screen.orientation, 'portrait');
      expect(delta.capabilities.screen.width, 720);
      expect(delta.capabilities.screen.height, 1280);
      expect(delta.capabilities.displays, isNotEmpty);
      expect(delta.capabilities.displays.first.hasPrimary(), isFalse);
    },
  );

  testWidgets(
    'deterministic metrics seam emits CapabilityDelta on resize',
    (WidgetTester tester) async {
      final harness = FakeClientHarness();
      final metrics = TestScreenMetricsController(
        ScreenMetrics(
          logicalSize: Size(1024, 768),
          devicePixelRatio: 1.5,
        ),
      );
      await tester.pumpWidget(
        TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine,
          screenMetricsProvider: metrics.read,
          screenMetricsChangeListenable: metrics.changes,
          displayGeometryDebounceInterval: const Duration(milliseconds: 80),
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
      await tester.pumpAndSettle();

      final deltaCountBefore = harness.lastClient.requests
          .where((request) => request.hasCapabilityDelta())
          .length;
      metrics.update(
        ScreenMetrics(
          logicalSize: Size(900, 700),
          devicePixelRatio: 1.5,
        ),
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 80));

      final deltas = harness.lastClient.requests
          .where((request) => request.hasCapabilityDelta())
          .toList();
      expect(deltas.length, deltaCountBefore + 1);
      final delta = deltas.last.capabilityDelta;
      expect(delta.capabilities.screen.width, 900);
      expect(delta.capabilities.screen.height, 700);
    },
  );

  testWidgets(
    'deterministic metrics seam emits CapabilityDelta on browser zoom dimension change',
    (WidgetTester tester) async {
      final harness = FakeClientHarness();
      final metrics = TestScreenMetricsController(
        ScreenMetrics(
          logicalSize: Size(1440, 900),
          devicePixelRatio: 1.0,
        ),
      );
      await tester.pumpWidget(
        TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine,
          screenMetricsProvider: metrics.read,
          screenMetricsChangeListenable: metrics.changes,
          displayGeometryDebounceInterval: const Duration(milliseconds: 80),
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
      await tester.pumpAndSettle();

      final deltaCountBefore = harness.lastClient.requests
          .where((request) => request.hasCapabilityDelta())
          .length;
      metrics.update(
        ScreenMetrics(
          logicalSize: Size(1152, 720),
          devicePixelRatio: 1.25,
        ),
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 80));

      final deltas = harness.lastClient.requests
          .where((request) => request.hasCapabilityDelta())
          .toList();
      expect(deltas.length, deltaCountBefore + 1);
      final delta = deltas.last.capabilityDelta;
      expect(delta.capabilities.screen.width, 1152);
      expect(delta.capabilities.screen.height, 720);
      expect(delta.capabilities.screen.density, 1.25);
    },
  );

  testWidgets(
    'deterministic metrics seam emits CapabilityDelta on safe-area change',
    (WidgetTester tester) async {
      final harness = FakeClientHarness();
      final metrics = TestScreenMetricsController(
        ScreenMetrics(
          logicalSize: Size(1024, 768),
          devicePixelRatio: 2.0,
          safeAreaInsets: EdgeInsets.zero,
        ),
      );
      await tester.pumpWidget(
        TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine,
          screenMetricsProvider: metrics.read,
          screenMetricsChangeListenable: metrics.changes,
          displayGeometryDebounceInterval: const Duration(milliseconds: 80),
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
      await tester.pumpAndSettle();

      final deltaCountBefore = harness.lastClient.requests
          .where((request) => request.hasCapabilityDelta())
          .length;
      metrics.update(
        ScreenMetrics(
          logicalSize: Size(1024, 768),
          devicePixelRatio: 2.0,
          safeAreaInsets: const EdgeInsets.only(top: 24, bottom: 16),
        ),
      );
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 80));

      final deltas = harness.lastClient.requests
          .where((request) => request.hasCapabilityDelta())
          .toList();
      expect(deltas.length, deltaCountBefore + 1);
      final delta = deltas.last.capabilityDelta;
      expect(delta.reason, 'display_geometry_change');
      expect(delta.capabilities.screen.safeArea.top, 24);
      expect(delta.capabilities.screen.safeArea.bottom, 16);
      expect(delta.capabilities.screen.safeArea.left, 0);
      expect(delta.capabilities.screen.safeArea.right, 0);
    },
  );

  testWidgets(
    'rapid deterministic resize changes are coalesced to one CapabilityDelta per debounce interval',
    (WidgetTester tester) async {
      final harness = FakeClientHarness();
      final metrics = TestScreenMetricsController(
        ScreenMetrics(
          logicalSize: Size(1280, 720),
          devicePixelRatio: 2.0,
        ),
      );
      await tester.pumpWidget(
        TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine,
          screenMetricsProvider: metrics.read,
          screenMetricsChangeListenable: metrics.changes,
          displayGeometryDebounceInterval: kDisplayGeometryDebounceInterval,
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
      await tester.pumpAndSettle();

      final deltaCountBefore = harness.lastClient.requests
          .where((request) => request.hasCapabilityDelta())
          .length;
      metrics.update(
        ScreenMetrics(
          logicalSize: Size(1200, 700),
          devicePixelRatio: 2.0,
        ),
      );
      await tester.pump(const Duration(milliseconds: 30));
      metrics.update(
        ScreenMetrics(
          logicalSize: Size(1000, 680),
          devicePixelRatio: 2.0,
        ),
      );
      await tester.pump(const Duration(milliseconds: 30));
      metrics.update(
        ScreenMetrics(
          logicalSize: Size(900, 650),
          devicePixelRatio: 2.0,
        ),
      );
      await tester.pump();

      await tester.pump(kDisplayGeometryDebounceInterval);
      final deltas = harness.lastClient.requests
          .where((request) => request.hasCapabilityDelta())
          .toList();
      expect(deltas.length, deltaCountBefore + 1);
      final delta = deltas.last.capabilityDelta;
      expect(delta.capabilities.screen.width, 900);
      expect(delta.capabilities.screen.height, 650);
    },
  );

  testWidgets(
    'app lifecycle capability updates omit synthetic monitor operators',
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
      await tester.pumpAndSettle();

      final bootstrapSnapshotCapabilities = harness.lastClient.requests
          .where((request) => request.hasCapabilitySnapshot())
          .toList()
          .last
          .capabilitySnapshot
          .capabilities;

      expect(bootstrapSnapshotCapabilities.hasEdge(), isFalse);

      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.inactive);
      await tester.pumpAndSettle();
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
      await tester.pumpAndSettle();
      final deltas = harness.lastClient.requests
          .where((request) => request.hasCapabilityDelta())
          .map((request) => request.capabilityDelta)
          .where((delta) => delta.reason == 'app_lifecycle_change')
          .toList();

      expect(
        deltas.where((delta) => delta.capabilities.hasEdge()).every(
              (delta) =>
                  !delta.capabilities.edge.operators
                      .contains('monitor.lifecycle.foreground') &&
                  !delta.capabilities.edge.operators
                      .contains('monitor.lifecycle.background'),
            ),
        isTrue,
      );
    },
  );

  testWidgets(
    'stale capability generation error triggers forced capability snapshot rebaseline',
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
      await tester.pumpAndSettle();

      final snapshotsBeforeError = harness.lastClient.requests
          .where((request) => request.hasCapabilitySnapshot())
          .toList();
      expect(snapshotsBeforeError.length, 1);
      final baselineGeneration =
          snapshotsBeforeError.single.capabilitySnapshot.generation.toInt();

      harness.lastClient.emitResponse(
        ConnectResponse()
          ..error = (ControlError()
            ..code = ControlErrorCode.CONTROL_ERROR_CODE_PROTOCOL_VIOLATION
            ..message = 'stale capability generation: expected newer'),
      );
      await tester.pumpAndSettle();

      final snapshotsAfterError = harness.lastClient.requests
          .where((request) => request.hasCapabilitySnapshot())
          .toList();
      expect(snapshotsAfterError.length, 2);
      final rebaselineGeneration =
          snapshotsAfterError.last.capabilitySnapshot.generation.toInt();
      expect(rebaselineGeneration, greaterThan(baselineGeneration));

      final deltas = harness.lastClient.requests
          .where((request) => request.hasCapabilityDelta())
          .map((request) => request.capabilityDelta)
          .where((delta) => delta.reason == 'stale_generation_rebaseline')
          .toList();
      expect(deltas, isEmpty);
    },
  );

  testWidgets('sends periodic sensor telemetry for declared signals only', (
    WidgetTester tester,
  ) async {
    final harness = FakeClientHarness();
    final capabilities = capv1.DeviceCapabilities()
      ..screen = (capv1.ScreenCapability()
        ..width = 1200
        ..height = 800
        ..density = 2.0
        ..touch = false
        ..orientation = 'landscape')
      ..connectivity =
          (capv1.ConnectivityCapability()..wifiSignalStrength = true)
      ..battery = (capv1.BatteryCapability()
        ..level = 0.75
        ..charging = true);
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        capabilityProbeFactory: () => StaticCapabilityProbe(capabilities),
        mediaEngineFactory: harness.createMediaEngine,
        heartbeatInterval: const Duration(seconds: 60),
        sensorTelemetryInterval: const Duration(milliseconds: 40),
      ),
    );

    await tester.tap(find.text('Connect Stream'));
    await tester.pump();
    harness.lastClient.emitResponse(
      ConnectResponse()..registerAck = (RegisterAck()),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 140));

    final requests = harness.lastClient.requests;
    final sensorRequests = requests.where((r) => r.hasSensor()).toList();
    expect(sensorRequests.length, greaterThanOrEqualTo(2));
    expect(
      sensorRequests.every((request) =>
          request.sensor.values.containsKey('battery.level') &&
          request.sensor.values.containsKey('battery.charging') &&
          !request.sensor.values
              .containsKey('connectivity.reconnect_attempt') &&
          !request.sensor.values.containsKey('time.utc_hour') &&
          !request.sensor.values.containsKey('time.utc_weekday') &&
          !request.sensor.values.containsKey('time.utc_minute')),
      isTrue,
    );
    expect(find.textContaining('Sensor sends: '), findsOneWidget);
    expect(find.textContaining('Last sensor unix_ms: '), findsOneWidget);
    expect(find.textContaining('Stream-ready acks: 0'), findsOneWidget);
  });

  testWidgets('skips sensor telemetry without declared signal capabilities', (
    WidgetTester tester,
  ) async {
    final harness = FakeClientHarness();
    final capabilities = capv1.DeviceCapabilities()
      ..screen = (capv1.ScreenCapability()
        ..width = 1200
        ..height = 800
        ..density = 2.0
        ..touch = false
        ..orientation = 'landscape');
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        capabilityProbeFactory: () => StaticCapabilityProbe(capabilities),
        mediaEngineFactory: harness.createMediaEngine,
        heartbeatInterval: const Duration(seconds: 60),
        sensorTelemetryInterval: const Duration(milliseconds: 40),
      ),
    );

    await tester.tap(find.text('Connect Stream'));
    await tester.pump();
    harness.lastClient.emitResponse(
      ConnectResponse()..registerAck = (RegisterAck()),
    );
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 140));

    final sensorRequests = harness.lastClient.requests
        .where((request) => request.hasSensor())
        .toList();
    expect(sensorRequests, isEmpty);
    expect(find.textContaining('Sensor sends: 0'), findsOneWidget);
  });
}
