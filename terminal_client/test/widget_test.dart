import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_webrtc/flutter_webrtc.dart';
import 'package:grpc/grpc.dart';
import 'package:terminal_client/capabilities/probe.dart';
import 'package:terminal_client/connection/control_client.dart';
import 'package:terminal_client/connection/control_client_factory.dart';
import 'package:terminal_client/gen/terminals/capabilities/v1/capabilities.pb.dart'
    as capv1;
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';
import 'package:terminal_client/gen/terminals/diagnostics/v1/diagnostics.pb.dart'
    as diagv1;
import 'package:terminal_client/gen/terminals/io/v1/io.pb.dart' as iov1;
import 'package:terminal_client/gen/terminals/ui/v1/ui.pb.dart' as uiv1;
import 'package:terminal_client/main.dart';
import 'package:terminal_client/media/playback.dart';
import 'package:terminal_client/media/webrtc_engine.dart';

Finder _findClientChromePrivacyOrCaptureIndicators() {
  return find.byWidgetPredicate(
    (widget) {
      final key = widget.key;
      if (key is! ValueKey<String>) {
        return false;
      }
      final value = key.value;
      return value.contains('client.chrome.privacy') ||
          value.contains('client.chrome.capture');
    },
    description: 'client-chrome privacy/capture indicator',
  );
}

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

  test('resolveInitialControlHost preserves configured host off web', () {
    final host = resolveInitialControlHost(
      isWebRuntime: false,
      configuredHost: '127.0.0.1',
      pageHost: '192.168.0.138',
    );
    expect(host, '127.0.0.1');
  });

  test('resolveInitialControlHost uses page host for loopback on web', () {
    final host = resolveInitialControlHost(
      isWebRuntime: true,
      configuredHost: '127.0.0.1',
      pageHost: '192.168.0.138',
    );
    expect(host, '192.168.0.138');
  });

  test('resolveInitialControlHost uses page host when configured host is empty',
      () {
    final host = resolveInitialControlHost(
      isWebRuntime: true,
      configuredHost: '   ',
      pageHost: 'localhost',
    );
    expect(host, 'localhost');
  });

  test('resolveInitialControlHost keeps non-loopback host on web', () {
    final host = resolveInitialControlHost(
      isWebRuntime: true,
      configuredHost: 'terminals.internal',
      pageHost: '192.168.0.138',
    );
    expect(host, 'terminals.internal');
  });

  test('resolvePageHost prefers browser location host', () {
    final host = resolvePageHost(
      browserLocationHost: '192.168.0.138',
      uriBaseHost: '127.0.0.1',
    );
    expect(host, '192.168.0.138');
  });

  test('resolvePageHost falls back to Uri.base host when location host empty',
      () {
    final host = resolvePageHost(
      browserLocationHost: '   ',
      uriBaseHost: 'localhost',
    );
    expect(host, 'localhost');
  });

  test('buildMetadataLabel renders date and sha', () {
    final label = buildMetadataLabel(
      buildDate: '2026-04-21T14:20:00Z',
      buildSha: 'abc123def456',
    );
    expect(label, 'Build: 2026-04-21T14:20:00Z | SHA: abc123def456');
  });

  test('buildTransportDiagnosticsClipboardText returns single multiline block',
      () {
    final text = buildTransportDiagnosticsClipboardText(
      lastTransportDiagnostic: 'failed at stream_closed',
      recentAttempts: const <String>['attempt one', 'attempt two'],
    );
    expect(
      text,
      'Transport Diagnostics\n'
      'failed at stream_closed\n'
      'Recent Carrier Attempts\n'
      'attempt one\n'
      'attempt two',
    );
  });

  test('buildControlStreamClipboardText returns single multiline block', () {
    final text = buildControlStreamClipboardText(
      status: 'All control carriers failed',
      notification: 'WebSocket failed',
      transportDiagnostics: 'ws://192.168.0.138:50054/control',
    );
    expect(
      text,
      'Control Stream: All control carriers failed\n'
      'WebSocket failed\n'
      'Transport Diagnostics\n'
      'ws://192.168.0.138:50054/control',
    );
  });

  test('buildVersionParityNote reports same SHA', () {
    final note = buildVersionParityNote(
      clientBuildDate: '2026-04-21T14:55:56Z',
      clientBuildSha: 'ea99b3f38658',
      serverBuildDate: '2026-04-21T14:56:01Z',
      serverBuildSha: 'ea99b3f38658',
    );
    expect(note, 'Build Match: same SHA, different build date');
  });

  test('buildVersionParityNote reports different SHA', () {
    final note = buildVersionParityNote(
      clientBuildDate: '2026-04-21T14:55:56Z',
      clientBuildSha: 'ea99b3f38658',
      serverBuildDate: '2026-04-21T14:55:56Z',
      serverBuildSha: 'deadbeef0001',
    );
    expect(note, 'Build Match: different SHA');
  });

  test('buildServerBuildLine reports awaiting register ack before connect', () {
    final line = buildServerBuildLine(
      serverBuildDate: 'unknown',
      serverBuildSha: 'unknown',
      hasRegisterAck: false,
    );
    expect(line, 'Server Build: awaiting register ack');
  });

  test('buildWebConnectionChipLabel reports not connected before stream starts',
      () {
    final label = buildWebConnectionChipLabel(
      hasRegisterAck: false,
      isConnecting: false,
      shouldStayConnected: false,
    );
    expect(label, 'Not connected');
  });

  testWidgets('app renders MaterialApp', (WidgetTester tester) async {
    await tester.pumpWidget(const TerminalClientApp());
    expect(find.byType(MaterialApp), findsOneWidget);
  });

  testWidgets('auto-connect starts stream on launch when enabled',
      (WidgetTester tester) async {
    final harness = _FakeClientHarness();
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
    final harness = _FakeClientHarness(failConnectAttempts: 1);
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
    final harness = _FakeClientHarness();
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
      final harness = _FakeClientHarness();
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
      expect(screen.hasSafeArea(), isTrue);
      expect(screen.safeArea.left, 0);
      expect(screen.safeArea.top, 0);
      expect(screen.safeArea.right, 0);
      expect(screen.safeArea.bottom, 0);
    },
  );

  testWidgets(
    'deterministic metrics seam emits CapabilityDelta on rotation with fresh generation',
    (WidgetTester tester) async {
      final harness = _FakeClientHarness();
      final metrics = _TestScreenMetricsController(
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
    },
  );

  testWidgets(
    'deterministic metrics seam emits CapabilityDelta on resize',
    (WidgetTester tester) async {
      final harness = _FakeClientHarness();
      final metrics = _TestScreenMetricsController(
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
      final harness = _FakeClientHarness();
      final metrics = _TestScreenMetricsController(
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
    'rapid deterministic resize changes are coalesced to one CapabilityDelta per debounce interval',
    (WidgetTester tester) async {
      final harness = _FakeClientHarness();
      final metrics = _TestScreenMetricsController(
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
    'app lifecycle foreground/background emits capability deltas with lifecycle operator updates',
    (WidgetTester tester) async {
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
      await tester.pumpAndSettle();

      final baseDeltaCount = harness.lastClient.requests
          .where((request) => request.hasCapabilityDelta())
          .length;

      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.inactive);
      await tester.pumpAndSettle();
      final backgroundDelta = harness.lastClient.requests
          .where((request) => request.hasCapabilityDelta())
          .toList()
          .last
          .capabilityDelta;
      expect(
        harness.lastClient.requests
            .where((request) => request.hasCapabilityDelta())
            .length,
        baseDeltaCount + 1,
      );
      expect(backgroundDelta.reason, 'app_lifecycle_change');
      expect(
        backgroundDelta.capabilities.edge.operators,
        contains('monitor.lifecycle.background'),
      );
      expect(
        backgroundDelta.capabilities.edge.operators,
        isNot(contains('monitor.lifecycle.foreground')),
      );

      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
      await tester.pumpAndSettle();
      final allDeltas = harness.lastClient.requests
          .where((request) => request.hasCapabilityDelta())
          .toList();
      expect(allDeltas.length, baseDeltaCount + 2);
      final foregroundDelta = allDeltas.last.capabilityDelta;
      expect(foregroundDelta.reason, 'app_lifecycle_change');
      expect(
          foregroundDelta.generation, greaterThan(backgroundDelta.generation));
      expect(
        foregroundDelta.capabilities.edge.operators,
        contains('monitor.lifecycle.foreground'),
      );
      expect(
        foregroundDelta.capabilities.edge.operators,
        isNot(contains('monitor.lifecycle.background')),
      );
    },
  );

  testWidgets('sends periodic sensor telemetry for declared signals only', (
    WidgetTester tester,
  ) async {
    final harness = _FakeClientHarness();
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
        capabilityProbeFactory: () => _StaticCapabilityProbe(capabilities),
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
          request.sensor.values.containsKey('connectivity.reconnect_attempt') &&
          request.sensor.values.containsKey('battery.level') &&
          request.sensor.values.containsKey('battery.charging') &&
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
    final harness = _FakeClientHarness();
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
        capabilityProbeFactory: () => _StaticCapabilityProbe(capabilities),
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
    harness.lastClient.emitResponse(
      ConnectResponse()..registerAck = (RegisterAck()),
    );
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
    harness.lastClient.emitResponse(
      ConnectResponse()..registerAck = (RegisterAck()),
    );
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

  testWidgets(
    'privacy.toggle off restores mic/camera with fresh generation',
    (WidgetTester tester) async {
      await tester.binding.setSurfaceSize(const Size(1200, 1400));
      addTearDown(() => tester.binding.setSurfaceSize(null));
      final harness = _FakeClientHarness();
      await tester.pumpWidget(
        TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine,
          capabilityProbeFactory: () => _StaticCapabilityProbe(
            capv1.DeviceCapabilities()
              ..microphone = (capv1.AudioInputCapability()
                ..channels = 1
                ..endpoints.add(capv1.AudioEndpoint()..endpointId = 'mic-main'))
              ..camera = (capv1.CameraCapability()
                ..endpoints.add(
                  capv1.CameraEndpoint()..endpointId = 'camera-main',
                )),
          ),
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
      harness.lastClient.emitResponse(
        ConnectResponse()
          ..setUi = (uiv1.SetUI()
            ..root = (uiv1.Node()
              ..id = 'terminal_root'
              ..stack = (uiv1.StackWidget())
              ..children.add(
                uiv1.Node()
                  ..id = 'act:main/privacy_toggle'
                  ..button = (uiv1.ButtonWidget()
                    ..label = 'Privacy'
                    ..action = 'privacy.toggle'),
              ))),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.text('Privacy'));
      await tester.pumpAndSettle();
      final firstPrivacyDelta = harness.lastClient.requests
          .lastWhere((request) => request.hasCapabilityDelta())
          .capabilityDelta;
      expect(firstPrivacyDelta.reason, 'privacy.toggle');
      expect(firstPrivacyDelta.capabilities.hasMicrophone(), isFalse);
      expect(firstPrivacyDelta.capabilities.hasCamera(), isFalse);

      await tester.tap(find.text('Privacy'));
      await tester.pumpAndSettle();
      final capabilityDeltas = harness.lastClient.requests
          .where((request) => request.hasCapabilityDelta())
          .toList();
      expect(capabilityDeltas.length, greaterThanOrEqualTo(2));
      final restoredDelta = capabilityDeltas.last.capabilityDelta;
      expect(restoredDelta.reason, 'privacy.toggle');
      expect(
          restoredDelta.generation, greaterThan(firstPrivacyDelta.generation));
      expect(restoredDelta.capabilities.hasMicrophone(), isTrue);
      expect(restoredDelta.capabilities.hasCamera(), isTrue);
    },
  );

  testWidgets(
    'privacy.toggle does not render persistent client-chrome privacy/capture indicator',
    (WidgetTester tester) async {
      await tester.binding.setSurfaceSize(const Size(1200, 1400));
      addTearDown(() => tester.binding.setSurfaceSize(null));
      final harness = _FakeClientHarness();
      await tester.pumpWidget(
        TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine,
          capabilityProbeFactory: () => _StaticCapabilityProbe(
            capv1.DeviceCapabilities()
              ..microphone = (capv1.AudioInputCapability()
                ..channels = 1
                ..endpoints.add(capv1.AudioEndpoint()..endpointId = 'mic-main'))
              ..camera = (capv1.CameraCapability()
                ..endpoints.add(
                  capv1.CameraEndpoint()..endpointId = 'camera-main',
                )),
          ),
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

      harness.lastClient.emitResponse(
        ConnectResponse()
          ..setUi = (uiv1.SetUI()
            ..root = (uiv1.Node()
              ..id = 'terminal_root'
              ..stack = (uiv1.StackWidget())
              ..children.addAll([
                uiv1.Node()
                  ..id = 'act:main/privacy_toggle'
                  ..button = (uiv1.ButtonWidget()
                    ..label = 'Privacy'
                    ..action = 'privacy.toggle'),
                uiv1.Node()
                  ..id = 'server.descriptor.privacy_overlay'
                  ..overlay = (uiv1.OverlayWidget())
                  ..children.add(
                    uiv1.Node()
                      ..id = 'server.descriptor.privacy_text'
                      ..text = (uiv1.TextWidget()..value = 'Server indicator'),
                  ),
              ]))),
      );
      await tester.pumpAndSettle();

      expect(
        find.byKey(
          const ValueKey<String>(
              'ui-overlay-server.descriptor.privacy_overlay'),
        ),
        findsOneWidget,
      );
      expect(_findClientChromePrivacyOrCaptureIndicators(), findsNothing);

      await tester.tap(find.text('Privacy'));
      await tester.pumpAndSettle();
      expect(_findClientChromePrivacyOrCaptureIndicators(), findsNothing);

      await tester.tap(find.text('Privacy'));
      await tester.pumpAndSettle();
      expect(_findClientChromePrivacyOrCaptureIndicators(), findsNothing);
    },
  );

  testWidgets(
      'wake-word detector toggles with microphone capability and privacy', (
    WidgetTester tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    final detector = _FakeWakeWordDetectorController();
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        mediaEngineFactory: harness.createMediaEngine,
        wakeWordDetectorFactory: () => detector,
        capabilityProbeFactory: () => _StaticCapabilityProbe(
          capv1.DeviceCapabilities()
            ..microphone = (capv1.AudioInputCapability()
              ..channels = 1
              ..endpoints.add(capv1.AudioEndpoint()..endpointId = 'mic-main')),
        ),
      ),
    );

    await tester.tap(find.text('Connect Stream'));
    await tester.pump();

    expect(detector.enabledStates, <bool>[true]);

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..registerAck = (RegisterAck()
          ..serverId = 'test-server'
          ..message = 'registered'),
    );
    await tester.pumpAndSettle();

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..setUi = (uiv1.SetUI()
          ..root = (uiv1.Node()
            ..id = 'terminal_root'
            ..stack = (uiv1.StackWidget())
            ..children.add(
              uiv1.Node()
                ..id = 'act:main/privacy_toggle'
                ..button = (uiv1.ButtonWidget()
                  ..label = 'Privacy'
                  ..action = 'privacy.toggle'),
            ))),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Privacy'));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Privacy'));
    await tester.pumpAndSettle();

    expect(detector.enabledStates, <bool>[true, false, true]);
  });

  testWidgets('wake-word utterance sends VoiceAudio when microphone is enabled',
      (
    WidgetTester tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    final detector = _FakeWakeWordDetectorController();
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        mediaEngineFactory: harness.createMediaEngine,
        wakeWordDetectorFactory: () => detector,
        capabilityProbeFactory: () => _StaticCapabilityProbe(
          capv1.DeviceCapabilities()
            ..microphone = (capv1.AudioInputCapability()
              ..channels = 1
              ..endpoints.add(capv1.AudioEndpoint()..endpointId = 'mic-main')),
        ),
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

    detector.simulateUtterance(
      audio: <int>[1, 2, 3, 4],
      sampleRate: 16000,
      isFinal: true,
    );
    await tester.pumpAndSettle();

    final voiceAudioRequests = harness.lastClient.requests
        .where((request) => request.hasVoiceAudio())
        .toList(growable: false);
    expect(voiceAudioRequests, hasLength(1));
    final voiceAudio = voiceAudioRequests.single.voiceAudio;
    expect(voiceAudio.deviceId, isNotEmpty);
    expect(voiceAudio.audio, <int>[1, 2, 3, 4]);
    expect(voiceAudio.sampleRate, 16000);
    expect(voiceAudio.isFinal, isTrue);
  });

  testWidgets(
    'wake-word utterance does not send VoiceAudio after privacy.toggle withdraws microphone capability',
    (WidgetTester tester) async {
      await tester.binding.setSurfaceSize(const Size(1200, 1400));
      addTearDown(() => tester.binding.setSurfaceSize(null));
      final harness = _FakeClientHarness();
      final detector = _FakeWakeWordDetectorController();
      await tester.pumpWidget(
        TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine,
          wakeWordDetectorFactory: () => detector,
          capabilityProbeFactory: () => _StaticCapabilityProbe(
            capv1.DeviceCapabilities()
              ..microphone = (capv1.AudioInputCapability()
                ..channels = 1
                ..endpoints
                    .add(capv1.AudioEndpoint()..endpointId = 'mic-main')),
          ),
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

      harness.lastClient.emitResponse(
        ConnectResponse()
          ..setUi = (uiv1.SetUI()
            ..root = (uiv1.Node()
              ..id = 'terminal_root'
              ..stack = (uiv1.StackWidget())
              ..children.add(
                uiv1.Node()
                  ..id = 'act:main/privacy_toggle'
                  ..button = (uiv1.ButtonWidget()
                    ..label = 'Privacy'
                    ..action = 'privacy.toggle'),
              ))),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.text('Privacy'));
      await tester.pumpAndSettle();

      detector.simulateUtterance(
        audio: <int>[9, 8, 7, 6],
        sampleRate: 16000,
        isFinal: true,
      );
      await tester.pumpAndSettle();

      final voiceAudioRequests = harness.lastClient.requests
          .where((request) => request.hasVoiceAudio())
          .toList(growable: false);
      expect(voiceAudioRequests, isEmpty);
    },
  );

  testWidgets('wake-word response disposition: silent service', (
    WidgetTester tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    final detector = _FakeWakeWordDetectorController();
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        mediaEngineFactory: harness.createMediaEngine,
        wakeWordDetectorFactory: () => detector,
        capabilityProbeFactory: () => _StaticCapabilityProbe(
          capv1.DeviceCapabilities()
            ..microphone = (capv1.AudioInputCapability()
              ..channels = 1
              ..endpoints.add(capv1.AudioEndpoint()..endpointId = 'mic-main')),
        ),
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

    detector.simulateUtterance(
      audio: <int>[11, 12, 13, 14],
      sampleRate: 16000,
      isFinal: true,
    );
    await tester.pumpAndSettle();

    final voiceAudioRequests = harness.lastClient.requests
        .where((request) => request.hasVoiceAudio())
        .toList(growable: false);
    expect(voiceAudioRequests, hasLength(1));
    expect(find.text('Notification: '), findsNothing);
    expect(find.text('wake service launched'), findsNothing);
    expect(find.text('Wake acknowledged'), findsNothing);
  });

  testWidgets('wake-word response disposition: activation launch', (
    WidgetTester tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    final detector = _FakeWakeWordDetectorController();
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        mediaEngineFactory: harness.createMediaEngine,
        wakeWordDetectorFactory: () => detector,
        capabilityProbeFactory: () => _StaticCapabilityProbe(
          capv1.DeviceCapabilities()
            ..microphone = (capv1.AudioInputCapability()
              ..channels = 1
              ..endpoints.add(capv1.AudioEndpoint()..endpointId = 'mic-main')),
        ),
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

    detector.simulateUtterance(
      audio: <int>[21, 22, 23, 24],
      sampleRate: 16000,
      isFinal: true,
    );
    await tester.pumpAndSettle();

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..setUi = (uiv1.SetUI()
          ..root = (uiv1.Node()
            ..id = 'terminal_root'
            ..stack = (uiv1.StackWidget())
            ..children.add(
              uiv1.Node()
                ..id = 'act:wake_service/output'
                ..text = (uiv1.TextWidget()..value = 'wake service launched'),
            ))),
    );
    await tester.pumpAndSettle();

    expect(find.text('wake service launched'), findsOneWidget);
    expect(find.text('Wake acknowledged'), findsNothing);
  });

  testWidgets(
      'wake-word response disposition: audible visible descriptor update', (
    WidgetTester tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    final detector = _FakeWakeWordDetectorController();
    final audioPlayback = _FakeAudioPlayback();
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        mediaEngineFactory: harness.createMediaEngine,
        wakeWordDetectorFactory: () => detector,
        audioPlaybackFactory: () => audioPlayback,
        capabilityProbeFactory: () => _StaticCapabilityProbe(
          capv1.DeviceCapabilities()
            ..microphone = (capv1.AudioInputCapability()
              ..channels = 1
              ..endpoints.add(capv1.AudioEndpoint()..endpointId = 'mic-main')),
        ),
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

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..setUi = (uiv1.SetUI()
          ..root = (uiv1.Node()
            ..id = 'terminal_root'
            ..stack = (uiv1.StackWidget())
            ..children.add(
              uiv1.Node()
                ..id = 'act:wake_feedback/banner'
                ..text =
                    (uiv1.TextWidget()..value = 'Waiting for wake response'),
            ))),
    );
    await tester.pumpAndSettle();

    detector.simulateUtterance(
      audio: <int>[31, 32, 33, 34],
      sampleRate: 16000,
      isFinal: true,
    );
    await tester.pumpAndSettle();

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..playAudio = (iov1.PlayAudio()
          ..requestId = 'wake-audio-1'
          ..pcmData = <int>[1, 2, 3, 4]),
    );
    harness.lastClient.emitResponse(
      ConnectResponse()
        ..updateUi = (uiv1.UpdateUI()
          ..componentId = 'act:wake_feedback/banner'
          ..node = (uiv1.Node()
            ..id = 'act:wake_feedback/banner'
            ..text = (uiv1.TextWidget()..value = 'Wake acknowledged'))),
    );
    await tester.pumpAndSettle();

    expect(audioPlayback.playedRequests, hasLength(1));
    expect(audioPlayback.playedRequests.single.requestId, 'wake-audio-1');
    expect(find.text('Wake acknowledged'), findsOneWidget);
  });

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
    final harness = _FakeClientHarness();
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
      ConnectResponse()..registerAck = (RegisterAck()),
    );
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
      ConnectResponse()..registerAck = (RegisterAck()),
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
      ConnectResponse()..registerAck = (RegisterAck()),
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
      ConnectResponse()..registerAck = (RegisterAck()),
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
      ConnectResponse()..registerAck = (RegisterAck()),
    );
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
      ConnectResponse()..registerAck = (RegisterAck()),
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
      ConnectResponse()..registerAck = (RegisterAck()),
    );
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
      ConnectResponse()..registerAck = (RegisterAck()),
    );
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
      ConnectResponse()..registerAck = (RegisterAck()),
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
      ConnectResponse()..registerAck = (RegisterAck()),
    );
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
      ConnectResponse()..registerAck = (RegisterAck()),
    );
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

  testWidgets(
    'privacy.toggle stops local capture before sending capability delta',
    (WidgetTester tester) async {
      await tester.binding.setSurfaceSize(const Size(1200, 1400));
      addTearDown(() => tester.binding.setSurfaceSize(null));
      final stopBlocker = Completer<void>();
      final harness = _FakeClientHarness(stopStreamBlocker: stopBlocker);
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
      final localDeviceID = harness.lastClient.requests
          .firstWhere((request) => request.hasHello())
          .hello
          .deviceId;

      harness.lastClient.emitResponse(
        ConnectResponse()
          ..startStream = (iov1.StartStream()
            ..streamId = 'audio-stream'
            ..kind = 'audio'
            ..sourceDeviceId = localDeviceID
            ..targetDeviceId = 'device-2'),
      );
      harness.lastClient.emitResponse(
        ConnectResponse()
          ..startStream = (iov1.StartStream()
            ..streamId = 'video-stream'
            ..kind = 'video'
            ..sourceDeviceId = localDeviceID
            ..targetDeviceId = 'device-2'),
      );
      harness.lastClient.emitResponse(
        ConnectResponse()
          ..setUi = (uiv1.SetUI()
            ..root = (uiv1.Node()
              ..id = 'terminal_root'
              ..stack = (uiv1.StackWidget())
              ..children.add(
                uiv1.Node()
                  ..id = 'act:main/privacy_toggle'
                  ..button = (uiv1.ButtonWidget()
                    ..label = 'Privacy'
                    ..action = 'privacy.toggle'),
              ))),
      );
      await tester.pumpAndSettle();
      final maxCapabilityGenerationBeforePrivacy = harness.lastClient.requests
          .where(
            (request) =>
                request.hasCapabilitySnapshot() || request.hasCapabilityDelta(),
          )
          .map(
            (request) => request.hasCapabilitySnapshot()
                ? request.capabilitySnapshot.generation
                : request.capabilityDelta.generation,
          )
          .fold<int>(
            0,
            (current, next) => current > next.toInt() ? current : next.toInt(),
          );

      await tester.tap(find.text('Privacy'));
      await tester.pump();

      expect(harness.lastMediaEngine.stopStreamCalls, <String>['audio-stream']);
      expect(
        harness.lastClient.requests
            .where((request) => request.hasCapabilityDelta()),
        isEmpty,
        reason: 'capability delta must wait until local capture stop completes',
      );

      stopBlocker.complete();
      await tester.pumpAndSettle();

      final capabilityDeltas = harness.lastClient.requests
          .where((request) => request.hasCapabilityDelta())
          .toList();
      expect(capabilityDeltas, isNotEmpty);
      expect(
        harness.lastMediaEngine.stopStreamCalls,
        containsAll(<String>['audio-stream', 'video-stream']),
      );
      final delta = capabilityDeltas.last.capabilityDelta;
      expect(delta.reason, 'privacy.toggle');
      expect(
        delta.generation.toInt(),
        greaterThan(maxCapabilityGenerationBeforePrivacy),
      );
      expect(delta.capabilities.hasMicrophone(), isFalse);
      expect(delta.capabilities.hasCamera(), isFalse);
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
    for (var i = 0; i < 20; i++) {
      if (harness.lastClient.requests.any(
        (request) =>
            request.hasCommand() &&
            request.command.kind == CommandKind.COMMAND_KIND_SYSTEM &&
            request.command.intent == 'runtime_status',
      )) {
        break;
      }
      await tester.pump(const Duration(milliseconds: 20));
    }
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

  testWidgets('runtime status query queues until register ack',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        mediaEngineFactory: harness.createMediaEngine,
      ),
    );

    await tester.tap(find.text('Connect Stream'));
    await tester.pump();

    await tester.tap(find.text('Runtime Status'));
    await tester.pump();
    expect(
      harness.lastClient.requests.where(
        (request) =>
            request.hasCommand() && request.command.intent == 'runtime_status',
      ),
      isEmpty,
    );

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..registerAck = (RegisterAck()
          ..serverId = 'test-server'
          ..message = 'registered'),
    );
    for (var i = 0; i < 20; i++) {
      await tester.pump(const Duration(milliseconds: 20));
      if (harness.lastClient.requests.any(
        (request) =>
            request.hasCommand() && request.command.intent == 'runtime_status',
      )) {
        break;
      }
    }

    expect(
      harness.lastClient.requests.where(
        (request) =>
            request.hasCommand() && request.command.intent == 'runtime_status',
      ),
      hasLength(1),
    );

    await tester.pump(const Duration(milliseconds: 150));
  });

  testWidgets(
      'connect bootstrap sends register request so metadata and app list hydrate without reconnect',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        mediaEngineFactory: harness.createMediaEngine,
      ),
    );

    await tester.tap(find.text('Connect Stream'));
    await tester.pump();

    for (var i = 0; i < 80; i++) {
      if (harness.lastClient.requests.any((request) => request.hasRegister())) {
        break;
      }
      await tester.pump(const Duration(milliseconds: 25));
    }

    expect(
      harness.lastClient.requests.any((request) => request.hasRegister()),
      isTrue,
      reason: 'bootstrap should include register request on first connect',
    );

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..registerAck = (RegisterAck()
          ..serverId = 'test-server'
          ..message = 'registered'
          ..metadata.addAll({
            'server_build_sha': 'srv-sha-001',
            'server_build_date': '2026-04-21T19:15:00Z',
          })),
    );
    await tester.pumpAndSettle();

    expect(
      find.textContaining(
          'Server Build: 2026-04-21T19:15:00Z | SHA: srv-sha-001'),
      findsOneWidget,
    );

    final appRegistryRequest = harness.lastClient.requests.lastWhere(
      (request) =>
          request.hasCommand() &&
          request.command.kind == CommandKind.COMMAND_KIND_SYSTEM &&
          request.command.intent == 'scenario_registry',
    );
    harness.lastClient.emitResponse(
      ConnectResponse()
        ..commandResult = (CommandResult()
          ..requestId = appRegistryRequest.command.requestId
          ..notification = 'System query: scenario_registry'
          ..data.addAll({
            'photo_frame': 'priority=40',
          })),
    );
    await tester.pumpAndSettle();

    await tester.tap(
      find.byWidgetPredicate(
        (widget) =>
            widget is DropdownButtonFormField<String> &&
            widget.decoration.labelText == 'Available Application',
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('photo_frame').last, findsOneWidget);
  });

  testWidgets(
      'connect bootstrap retries register when transport attaches request stream late',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness(
      requestSubscriptionDelay: const Duration(milliseconds: 2800),
    );
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        mediaEngineFactory: harness.createMediaEngine,
      ),
    );

    await tester.tap(find.text('Connect Stream'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 4200));

    expect(
      harness.lastClient.requests.any((request) => request.hasRegister()),
      isTrue,
      reason:
          'client should retry bootstrap register even when request stream subscription is delayed',
    );
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

  testWidgets('scoped terminal root renders fullscreen app view',
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
            ..id = 'act:test-activation/terminal_root'
            ..stack = (uiv1.StackWidget())
            ..children.add(
              uiv1.Node()
                ..id = 'act:test-activation/terminal_output'
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
    this.requestSubscriptionDelay = Duration.zero,
    this.stopStreamBlocker,
  });

  final List<_FakeTerminalControlClient> createdClients =
      <_FakeTerminalControlClient>[];
  final List<_FakeMediaEngine> createdMediaEngines = <_FakeMediaEngine>[];
  final List<ControlCarrierKind> requestedCarriers = <ControlCarrierKind>[];
  final bool failFirstConnectStream;
  final int failConnectAttempts;
  final Duration requestSubscriptionDelay;
  final Completer<void>? stopStreamBlocker;

  _FakeTerminalControlClient get lastClient => createdClients.last;
  _FakeMediaEngine get lastMediaEngine => createdMediaEngines.last;

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
      requestSubscriptionDelay: requestSubscriptionDelay,
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
      stopStreamBlocker: stopStreamBlocker,
    );
    createdMediaEngines.add(engine);
    return engine;
  }
}

class _StaticCapabilityProbe implements CapabilityProbe {
  _StaticCapabilityProbe(this.capabilities);

  final capv1.DeviceCapabilities capabilities;

  @override
  Future<capv1.DeviceCapabilities> probe(CapabilityProbeContext context) async {
    return capabilities.deepCopy();
  }
}

class _TestScreenMetricsController {
  _TestScreenMetricsController(this._metrics);

  ScreenMetrics _metrics;
  final ValueNotifier<int> changes = ValueNotifier<int>(0);

  ScreenMetrics read() => _metrics;

  void update(ScreenMetrics metrics) {
    _metrics = metrics;
    changes.value = changes.value + 1;
  }
}

class _FakeWakeWordDetectorController implements WakeWordDetectorController {
  final List<bool> enabledStates = <bool>[];
  void Function(WakeWordUtterance utterance)? _onUtterance;

  @override
  Future<void> setEnabled(bool enabled) async {
    enabledStates.add(enabled);
  }

  @override
  void setOnUtterance(void Function(WakeWordUtterance utterance)? onUtterance) {
    _onUtterance = onUtterance;
  }

  void simulateUtterance({
    required List<int> audio,
    required int sampleRate,
    required bool isFinal,
  }) {
    _onUtterance?.call(
      WakeWordUtterance(
        audio: audio,
        sampleRate: sampleRate,
        isFinal: isFinal,
      ),
    );
  }

  @override
  Future<void> dispose() async {}
}

class _FakeAudioPlayback implements AudioPlayback {
  final List<iov1.PlayAudio> playedRequests = <iov1.PlayAudio>[];

  @override
  Future<void> play(iov1.PlayAudio playAudio) async {
    playedRequests.add(playAudio.deepCopy());
  }

  @override
  Future<void> dispose() async {}
}

class _FakeMediaEngine implements ClientMediaEngine {
  _FakeMediaEngine({
    required this.localDeviceID,
    required this.onSignal,
    this.stopStreamBlocker,
  });

  final String localDeviceID;
  final OutboundSignalCallback onSignal;
  final Completer<void>? stopStreamBlocker;
  final Set<String> _activeStreamIDs = <String>{};
  final List<String> stopStreamCalls = <String>[];
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
    stopStreamCalls.add(streamID);
    final blocker = stopStreamBlocker;
    if (blocker != null) {
      await blocker.future;
    }
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
    required this.requestSubscriptionDelay,
  });

  final String host;
  final int port;
  final bool failOnConnectStream;
  final Duration requestSubscriptionDelay;
  final List<ConnectRequest> requests = <ConnectRequest>[];
  final StreamController<ConnectResponse> _responses =
      StreamController<ConnectResponse>.broadcast();
  StreamSubscription<ConnectRequest>? _requestSubscription;

  @override
  Stream<ConnectResponse> connect(
    Stream<ConnectRequest> requests, {
    CallOptions? options,
  }) {
    if (requestSubscriptionDelay > Duration.zero) {
      unawaited(
        Future<void>.delayed(requestSubscriptionDelay).then((_) {
          _requestSubscription = requests.listen(this.requests.add);
        }),
      );
    } else {
      _requestSubscription = requests.listen(this.requests.add);
    }
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
