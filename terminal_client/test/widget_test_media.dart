import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/app/terminal_client_app.dart';
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';
import 'package:terminal_client/gen/terminals/io/v1/io.pb.dart' as iov1;
import 'package:terminal_client/gen/terminals/ui/v1/ui.pb.dart' as uiv1;

import 'widget_test_helpers.dart';

void main() {
  testWidgets('video surface stream state toggles on start and stop stream',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = FakeClientHarness();
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
                ..id = 'camera_a'
                ..videoSurface =
                    (uiv1.VideoSurfaceWidget()..trackId = 'stream-a'),
            ))),
    );
    await tester.pump();

    final stateFinder = find.byKey(
      const ValueKey<String>('ui-video-surface-state-camera_a'),
    );
    expect(stateFinder, findsOneWidget);
    expect(find.descendant(of: stateFinder, matching: find.text('Attached')),
        findsNothing);
    expect(
      find.descendant(
        of: stateFinder,
        matching: find.text('Waiting for media'),
      ),
      findsOneWidget,
    );

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..startStream = (iov1.StartStream()
          ..streamId = 'stream-a'
          ..kind = 'video'
          ..sourceDeviceId = 'device-1'
          ..targetDeviceId = 'device-2'),
    );
    await tester.pump();

    expect(find.descendant(of: stateFinder, matching: find.text('Attached')),
        findsOneWidget);

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..stopStream = (iov1.StopStream()..streamId = 'stream-a'),
    );
    await tester.pump();

    expect(find.descendant(of: stateFinder, matching: find.text('Attached')),
        findsNothing);
    expect(
      find.descendant(
        of: stateFinder,
        matching: find.text('Waiting for media'),
      ),
      findsOneWidget,
    );
  });

  testWidgets('audio visualizer stream state toggles on start and stop stream',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = FakeClientHarness();
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
                ..id = 'mic_mix'
                ..audioVisualizer =
                    (uiv1.AudioVisualizerWidget()..streamId = 'stream-audio'),
            ))),
    );
    await tester.pump();

    final stateFinder = find.byKey(
      const ValueKey<String>('ui-audio-visualizer-state-mic_mix'),
    );
    expect(stateFinder, findsOneWidget);
    expect(find.descendant(of: stateFinder, matching: find.text('Attached')),
        findsNothing);
    expect(
      find.descendant(
        of: stateFinder,
        matching: find.text('Waiting for media'),
      ),
      findsOneWidget,
    );

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..startStream = (iov1.StartStream()
          ..streamId = 'stream-audio'
          ..kind = 'audio'
          ..sourceDeviceId = 'device-1'
          ..targetDeviceId = 'device-2'),
    );
    await tester.pump();

    expect(find.descendant(of: stateFinder, matching: find.text('Attached')),
        findsOneWidget);
    final attachedProgressIndicator = tester.widget<LinearProgressIndicator>(
      find.descendant(
        of: find.byKey(const ValueKey<String>('ui-audio-visualizer-mic_mix')),
        matching: find.byType(LinearProgressIndicator),
      ),
    );
    expect(attachedProgressIndicator.value, isNotNull);

    harness.lastClient.emitResponse(
      ConnectResponse()
        ..stopStream = (iov1.StopStream()..streamId = 'stream-audio'),
    );
    await tester.pump();

    expect(find.descendant(of: stateFinder, matching: find.text('Attached')),
        findsNothing);
    expect(
      find.descendant(
        of: stateFinder,
        matching: find.text('Waiting for media'),
      ),
      findsOneWidget,
    );
    final waitingProgressIndicator = tester.widget<LinearProgressIndicator>(
      find.descendant(
        of: find.byKey(const ValueKey<String>('ui-audio-visualizer-mic_mix')),
        matching: find.byType(LinearProgressIndicator),
      ),
    );
    expect(waitingProgressIndicator.value, isNull);
  });

  testWidgets(
    'handles media control responses and acknowledges started streams',
    (WidgetTester tester) async {
      await tester.binding.setSurfaceSize(const Size(1200, 1400));
      addTearDown(() => tester.binding.setSurfaceSize(null));
      final harness = FakeClientHarness();
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
    'surfaces deterministic status when media permission probe fails',
    (WidgetTester tester) async {
      await tester.binding.setSurfaceSize(const Size(1200, 1400));
      addTearDown(() => tester.binding.setSurfaceSize(null));
      final harness = FakeClientHarness();
      await tester.pumpWidget(
        TerminalClientApp(
          clientFactory: harness.createClient,
          mediaEngineFactory: harness.createMediaEngine,
          mediaPermissionProbe: ({required audio, required video}) async {
            throw StateError('permission denied');
          },
        ),
      );
      await tester.tap(find.text('Connect Stream'));
      await tester.pump();

      final localDeviceID = harness.lastClient.requests
          .firstWhere((request) => request.hasHello())
          .hello
          .deviceId;

      harness.lastClient.emitResponse(
        ConnectResponse()
          ..startStream = (iov1.StartStream()
            ..streamId = 'stream-permission-denied'
            ..kind = 'audio'
            ..sourceDeviceId = localDeviceID
            ..targetDeviceId = 'device-2'),
      );
      await tester.pumpAndSettle();

      expect(find.textContaining('Control Stream: Media permission required'),
          findsOneWidget);
      expect(
        find.textContaining(
            'Unable to start media stream stream-permission-denied: Bad state: permission denied'),
        findsOneWidget,
      );

      final localOfferRequests = harness.lastClient.requests
          .where((request) =>
              request.hasWebrtcSignal() &&
              request.webrtcSignal.streamId == 'stream-permission-denied' &&
              request.webrtcSignal.signalType == 'offer')
          .toList();
      expect(localOfferRequests, isEmpty);
    },
  );

  testWidgets(
    'privacy.toggle stops local capture before sending capability delta',
    (WidgetTester tester) async {
      await tester.binding.setSurfaceSize(const Size(1200, 1400));
      addTearDown(() => tester.binding.setSurfaceSize(null));
      final stopBlocker = Completer<void>();
      final harness = FakeClientHarness(stopStreamBlocker: stopBlocker);
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
    final harness = FakeClientHarness();
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
    final harness = FakeClientHarness();
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
    final harness = FakeClientHarness();
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
    final harness = FakeClientHarness();
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
    final harness = FakeClientHarness();
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
      'connect bootstrap sends hello and capability snapshot so metadata and app list hydrate without reconnect',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = FakeClientHarness();
    await tester.pumpWidget(
      TerminalClientApp(
        clientFactory: harness.createClient,
        mediaEngineFactory: harness.createMediaEngine,
      ),
    );

    await tester.tap(find.text('Connect Stream'));
    await tester.pump();

    for (var i = 0; i < 80; i++) {
      if (harness.lastClient.requests.any((request) => request.hasHello()) &&
          harness.lastClient.requests
              .any((request) => request.hasCapabilitySnapshot())) {
        break;
      }
      await tester.pump(const Duration(milliseconds: 25));
    }

    expect(
      harness.lastClient.requests.any((request) => request.hasHello()),
      isTrue,
      reason: 'bootstrap should include hello request on first connect',
    );
    expect(
      harness.lastClient.requests
          .any((request) => request.hasCapabilitySnapshot()),
      isTrue,
      reason: 'bootstrap should include capability snapshot on first connect',
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
      'connect bootstrap still delivers capability snapshot when transport attaches request stream late',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = FakeClientHarness(
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
      harness.lastClient.requests
          .any((request) => request.hasCapabilitySnapshot()),
      isTrue,
      reason: 'client should send capability snapshot during bootstrap',
    );
  });

  testWidgets('server root can hide client chrome',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = FakeClientHarness();
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
            ..props['client_chrome'] = 'hidden'
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

  testWidgets('scoped server root can hide client chrome',
      (WidgetTester tester) async {
    await tester.binding.setSurfaceSize(const Size(1200, 1400));
    addTearDown(() => tester.binding.setSurfaceSize(null));
    final harness = FakeClientHarness();
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
            ..props['client_chrome'] = 'hidden'
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
