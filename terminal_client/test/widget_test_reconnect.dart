import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/app/terminal_client_app.dart';
import 'package:terminal_client/connection/control_client_factory.dart';
import 'package:terminal_client/gen/terminals/capabilities/v1/capabilities.pb.dart'
    as capv1;
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';
import 'package:terminal_client/gen/terminals/io/v1/io.pb.dart' as iov1;
import 'package:terminal_client/gen/terminals/ui/v1/ui.pb.dart' as uiv1;

import 'widget_test_helpers.dart';

void main() {
  testWidgets('pauses heartbeat loop while app is backgrounded', (
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

  testWidgets(
    'does not reconnect-loop when stream drops while app is backgrounded',
    (WidgetTester tester) async {
      final harness = FakeClientHarness();
      await tester.pumpWidget(
        TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine,
          reconnectDelayBase: const Duration(milliseconds: 20),
          reconnectDelayMaxSeconds: 1,
        ),
      );

      await tester.tap(find.text('Connect Stream'));
      await tester.pump();
      harness.lastClient.emitResponse(
        ConnectResponse()..registerAck = (RegisterAck()),
      );
      await tester.pump();
      expect(harness.createdClients.length, 1);

      // Background the app — heartbeats are suppressed in background.
      tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.paused);
      await tester.pump();

      // Server drops the connection (as it would after a heartbeat timeout).
      await harness.lastClient.closeStream();
      await tester.pump();

      // Wait well past the reconnect delay — no reconnect should occur while backgrounded.
      for (var i = 0; i < 20; i++) {
        await tester.pump(const Duration(milliseconds: 10));
      }
      expect(harness.createdClients.length, 1,
          reason: 'should not reconnect while backgrounded');

      // Return to foreground via the valid state sequence.
      tester.binding
          .handleAppLifecycleStateChanged(AppLifecycleState.hidden);
      tester.binding
          .handleAppLifecycleStateChanged(AppLifecycleState.inactive);
      tester.binding
          .handleAppLifecycleStateChanged(AppLifecycleState.resumed);
      await tester.pump();

      for (var i = 0; i < 20; i++) {
        if (harness.createdClients.length >= 2) break;
        await tester.pump(const Duration(milliseconds: 10));
      }
      expect(harness.createdClients.length, 2,
          reason: 'should reconnect on foreground resume');
    },
  );

  testWidgets('reconnect creates a new control client after stream failure', (
    WidgetTester tester,
  ) async {
    final harness = FakeClientHarness(failFirstConnectStream: true);
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
      final harness = FakeClientHarness(failConnectAttempts: 1);
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
      final harness = FakeClientHarness();
      await tester.pumpWidget(
        TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine,
          capabilityProbeFactory: () => StaticCapabilityProbe(
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
      final harness = FakeClientHarness();
      await tester.pumpWidget(
        TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine,
          capabilityProbeFactory: () => StaticCapabilityProbe(
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
      expect(findClientChromePrivacyOrCaptureIndicators(), findsNothing);

      await tester.tap(find.text('Privacy'));
      await tester.pumpAndSettle();
      expect(findClientChromePrivacyOrCaptureIndicators(), findsNothing);

      await tester.tap(find.text('Privacy'));
      await tester.pumpAndSettle();
      expect(findClientChromePrivacyOrCaptureIndicators(), findsNothing);
    },
  );

  testWidgets(
      'wake-word detector toggles with microphone capability and privacy', (
    WidgetTester tester,
  ) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = FakeClientHarness();
    final detector = FakeWakeWordDetectorController();
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        mediaEngineFactory: harness.createMediaEngine,
        wakeWordDetectorFactory: () => detector,
        capabilityProbeFactory: () => StaticCapabilityProbe(
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
    final harness = FakeClientHarness();
    final detector = FakeWakeWordDetectorController();
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        mediaEngineFactory: harness.createMediaEngine,
        wakeWordDetectorFactory: () => detector,
        capabilityProbeFactory: () => StaticCapabilityProbe(
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
      final harness = FakeClientHarness();
      final detector = FakeWakeWordDetectorController();
      await tester.pumpWidget(
        TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine,
          wakeWordDetectorFactory: () => detector,
          capabilityProbeFactory: () => StaticCapabilityProbe(
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
    final harness = FakeClientHarness();
    final detector = FakeWakeWordDetectorController();
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        mediaEngineFactory: harness.createMediaEngine,
        wakeWordDetectorFactory: () => detector,
        capabilityProbeFactory: () => StaticCapabilityProbe(
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
    final harness = FakeClientHarness();
    final detector = FakeWakeWordDetectorController();
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        mediaEngineFactory: harness.createMediaEngine,
        wakeWordDetectorFactory: () => detector,
        capabilityProbeFactory: () => StaticCapabilityProbe(
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
    final harness = FakeClientHarness();
    final detector = FakeWakeWordDetectorController();
    final audioPlayback = FakeAudioPlayback();
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        mediaEngineFactory: harness.createMediaEngine,
        wakeWordDetectorFactory: () => detector,
        audioPlaybackFactory: () => audioPlayback,
        capabilityProbeFactory: () => StaticCapabilityProbe(
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
}
