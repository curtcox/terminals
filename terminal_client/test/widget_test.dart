import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:grpc/grpc.dart';
import 'package:terminal_client/connection/control_client.dart';
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';
import 'package:terminal_client/gen/terminals/io/v1/io.pb.dart' as iov1;
import 'package:terminal_client/gen/terminals/ui/v1/ui.pb.dart' as uiv1;
import 'package:terminal_client/main.dart';

void main() {
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
        heartbeatInterval: const Duration(milliseconds: 40),
      ),
    );

    await tester.tap(find.text('Connect Stream'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 140));

    final requests = harness.lastClient.requests;
    final heartbeatCount = requests.where((r) => r.hasHeartbeat()).length;
    expect(heartbeatCount, greaterThanOrEqualTo(2));
    expect(requests.where((r) => r.hasRegister()).length, 1);
  });

  testWidgets('sends periodic sensor telemetry while connected', (
    WidgetTester tester,
  ) async {
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
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
          request.sensor.values.containsKey('battery.level') &&
          request.sensor.values.containsKey('connectivity.online') &&
          request.sensor.values.containsKey('time.utc_hour')),
      isTrue,
    );
  });

  testWidgets('reconnect creates a new control client after stream failure', (
    WidgetTester tester,
  ) async {
    final harness = _FakeClientHarness(failFirstConnectStream: true);
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        reconnectDelayBase: const Duration(milliseconds: 30),
        reconnectDelayMaxSeconds: 1,
      ),
    );

    await tester.tap(find.text('Connect Stream'));
    await tester.pump();
    expect(harness.createdClients.length, 1);

    for (var i = 0; i < 80; i++) {
      if (harness.createdClients.length >= 2) {
        break;
      }
      await tester.pump(const Duration(milliseconds: 10));
    }
    expect(harness.createdClients.length, 2);
  });

  testWidgets(
    'reconnect re-registers capabilities after control stream failure',
    (WidgetTester tester) async {
      final harness = _FakeClientHarness(failFirstConnectStream: true);
      await tester.pumpWidget(
        TerminalClientApp(
          clientFactory: harness.createClient,
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
        final reconnectRegisters = harness.createdClients[1].requests
            .where((request) => request.hasRegister())
            .length;
        if (reconnectRegisters > 0) {
          break;
        }
        await tester.pump(const Duration(milliseconds: 10));
      }

      final reconnectRegisters = harness.createdClients[1].requests
          .where((request) => request.hasRegister())
          .length;
      expect(reconnectRegisters, 1);
    },
  );

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
      TerminalClientApp(clientFactory: harness.createClient),
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
      TerminalClientApp(clientFactory: harness.createClient),
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
      TerminalClientApp(clientFactory: harness.createClient),
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

  testWidgets('renders grid, padding, and progress primitives',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(clientFactory: harness.createClient),
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
      TerminalClientApp(clientFactory: harness.createClient),
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
      TerminalClientApp(clientFactory: harness.createClient),
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
      TerminalClientApp(clientFactory: harness.createClient),
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
        .widget<DropdownButton<String>>(find.byType(DropdownButton<String>));
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
      TerminalClientApp(clientFactory: harness.createClient),
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

  testWidgets('renders overlay primitive with layered children',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(clientFactory: harness.createClient),
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

  testWidgets('wires gesture area tap to UIAction',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(clientFactory: harness.createClient),
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

  testWidgets(
      'renders safe placeholders for canvas, media, and system hint primitives',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = _FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(clientFactory: harness.createClient),
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
        TerminalClientApp(clientFactory: harness.createClient),
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

      final readyRequests = harness.lastClient.requests
          .where((request) => request.hasStreamReady())
          .toList();
      expect(readyRequests.length, 1);
      expect(readyRequests.first.streamReady.streamId, 'stream-a');

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
}

class _FakeClientHarness {
  _FakeClientHarness({this.failFirstConnectStream = false});

  final List<_FakeTerminalControlClient> createdClients =
      <_FakeTerminalControlClient>[];
  final bool failFirstConnectStream;

  _FakeTerminalControlClient get lastClient => createdClients.last;

  TerminalControlClient createClient({
    required String host,
    required int port,
  }) {
    final client = _FakeTerminalControlClient(
      host: host,
      port: port,
      failOnConnectStream: failFirstConnectStream && createdClients.isEmpty,
    );
    createdClients.add(client);
    return client;
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
  }

  void emitResponse(ConnectResponse response) {
    _responses.add(response);
  }
}
