import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/connection/control_response_dispatcher.dart';
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';
import 'package:terminal_client/gen/terminals/diagnostics/v1/diagnostics.pb.dart'
    as diagv1;
import 'package:terminal_client/gen/terminals/io/v1/io.pb.dart' as iov1;
import 'package:terminal_client/gen/terminals/ui/v1/ui.pb.dart' as uiv1;

void main() {
  group('statusFromConnectResponse', () {
    test('labels common response payloads without app state', () {
      expect(statusFromConnectResponse(ConnectResponse()), 'Connected');
      expect(
        statusFromConnectResponse(
          ConnectResponse()
            ..setUi = (uiv1.SetUI()
              ..root = (uiv1.Node()
                ..id = 'root'
                ..text = (uiv1.TextWidget()..value = 'Ready'))),
        ),
        'UI updated',
      );
      expect(
        statusFromConnectResponse(
          ConnectResponse()
            ..updateUi = (uiv1.UpdateUI()
              ..componentId = 'target'
              ..node =
                  (uiv1.Node()..text = (uiv1.TextWidget()..value = 'Patched'))),
        ),
        'UI patched',
      );
      expect(
        statusFromConnectResponse(
          ConnectResponse()..registerAck = RegisterAck(),
        ),
        'Registered',
      );
      expect(
        statusFromConnectResponse(
          ConnectResponse()..error = (ControlError()..message = 'boom'),
        ),
        'Server error',
      );
    });

    test('keeps first-match precedence for compound responses', () {
      final response = ConnectResponse()
        ..setUi = (uiv1.SetUI()
          ..root = (uiv1.Node()..text = (uiv1.TextWidget()..value = 'Root')))
        ..updateUi = (uiv1.UpdateUI()
          ..componentId = 'root'
          ..node = (uiv1.Node()..text = (uiv1.TextWidget()..value = 'Patch')));

      expect(statusFromConnectResponse(response), 'UI patched');
    });
  });

  group('applyUpdateUi', () {
    test('returns current root when update has no replacement node', () {
      final root = uiv1.Node()
        ..id = 'root'
        ..text = (uiv1.TextWidget()..value = 'Current');

      expect(
        identical(
          applyUpdateUi(currentRoot: root, update: uiv1.UpdateUI()),
          root,
        ),
        isTrue,
      );
    });

    test('replaces root when component id is empty or targets root id', () {
      final root = uiv1.Node()
        ..id = 'root'
        ..text = (uiv1.TextWidget()..value = 'Current');
      final replacement = uiv1.Node()
        ..id = 'next'
        ..text = (uiv1.TextWidget()..value = 'Next');

      expect(
        applyUpdateUi(
          currentRoot: root,
          update: uiv1.UpdateUI()..node = replacement,
        )!
            .text
            .value,
        'Next',
      );
      expect(
        applyUpdateUi(
          currentRoot: root,
          update: uiv1.UpdateUI()
            ..componentId = 'root'
            ..node = replacement,
        )!
            .id,
        'next',
      );
    });

    test('patches nested children by id or props id without mutating input',
        () {
      final root = uiv1.Node()
        ..id = 'root'
        ..stack = uiv1.StackWidget()
        ..children.add(
          uiv1.Node()
            ..id = 'panel'
            ..row = uiv1.RowWidget()
            ..children.add(
              uiv1.Node()
                ..props['id'] = 'target'
                ..text = (uiv1.TextWidget()..value = 'Old'),
            ),
        );
      final replacement = uiv1.Node()
        ..id = 'target'
        ..text = (uiv1.TextWidget()..value = 'New');

      final updated = applyUpdateUi(
        currentRoot: root,
        update: uiv1.UpdateUI()
          ..componentId = 'target'
          ..node = replacement,
      );

      expect(updated, isNot(same(root)));
      expect(updated!.children.single.children.single.text.value, 'New');
      expect(root.children.single.children.single.text.value, 'Old');
    });

    test('returns existing root if target is missing or root is absent', () {
      final root = uiv1.Node()
        ..id = 'root'
        ..text = (uiv1.TextWidget()..value = 'Current');
      final update = uiv1.UpdateUI()
        ..componentId = 'missing'
        ..node = (uiv1.Node()..text = (uiv1.TextWidget()..value = 'Next'));

      expect(identical(applyUpdateUi(currentRoot: root, update: update), root),
          isTrue);
      expect(applyUpdateUi(currentRoot: null, update: update), isNull);
    });
  });

  group('serverDrivenUiUpdateFromResponse', () {
    test('derives set UI updates outside the shell', () {
      final root = uiv1.Node()
        ..id = 'root'
        ..stack = uiv1.StackWidget()
        ..children.add(
          uiv1.Node()
            ..id = 'target'
            ..text = (uiv1.TextWidget()..value = 'Old'),
        );
      final response = ConnectResponse()..setUi = (uiv1.SetUI()..root = root);

      final update = serverDrivenUiUpdateFromResponse(
        response: response,
        currentRoot: null,
      );

      expect(update, isNotNull);
      expect(update!.activeRoot!.children.single.text.value, 'Old');
      expect(update.uiChanged, isTrue);
      expect(
        update.events.map((event) => event.kind),
        <String>['set_ui'],
      );
      expect(
        update.events.map((event) => event.kindEnum),
        <diagv1.UiEventKind>[diagv1.UiEventKind.UI_EVENT_KIND_SET_UI],
      );
      expect(
        update.events.map((event) => event.componentId),
        <String>['root'],
      );
    });

    test('derives patch UI updates without mutating the current root', () {
      final root = uiv1.Node()
        ..id = 'root'
        ..stack = uiv1.StackWidget()
        ..children.add(
          uiv1.Node()
            ..id = 'target'
            ..text = (uiv1.TextWidget()..value = 'Old'),
        );
      final response = ConnectResponse()
        ..updateUi = (uiv1.UpdateUI()
          ..componentId = 'target'
          ..node = (uiv1.Node()
            ..id = 'target'
            ..text = (uiv1.TextWidget()..value = 'New')));

      final update = serverDrivenUiUpdateFromResponse(
        response: response,
        currentRoot: root,
      );

      expect(update, isNotNull);
      expect(update!.activeRoot!.children.single.text.value, 'New');
      expect(root.children.single.text.value, 'Old');
      expect(update.uiChanged, isTrue);
      expect(update.events.single.kind, 'update_ui');
      expect(
        update.events.single.kindEnum,
        diagv1.UiEventKind.UI_EVENT_KIND_UPDATE_UI,
      );
      expect(update.events.single.componentId, 'target');
    });

    test('derives transition updates and marks existing root changed', () {
      final response = ConnectResponse()
        ..transitionUi = (uiv1.TransitionUI()
          ..transition = 'fade'
          ..durationMs = 120);
      final update = serverDrivenUiUpdateFromResponse(
        response: response,
        currentRoot: uiv1.Node()..id = 'root',
      );

      expect(update, isNotNull);
      expect(update!.uiChanged, isTrue);
      expect(update.events.single.kind, 'transition_ui');
      expect(
        update.events.single.kindEnum,
        diagv1.UiEventKind.UI_EVENT_KIND_TRANSITION_UI,
      );
      expect(update.events.single.componentId, 'root');
      expect(update.transitionHint?.transition, 'fade');
      expect(
          update.transitionHint?.duration, const Duration(milliseconds: 120));
      expect(update.transitionHint?.notification, 'Transition: fade (120ms)');
    });

    test('returns null when response has no UI work', () {
      expect(
        serverDrivenUiUpdateFromResponse(
          response: ConnectResponse(),
          currentRoot: null,
        ),
        isNull,
      );
    });

    test('transition default duration is derived outside the shell', () {
      final hint = transitionHintFromResponse(
        uiv1.TransitionUI()..transition = 'slide_left',
      );

      expect(hint.transition, 'slide_left');
      expect(hint.duration, const Duration(milliseconds: 250));
      expect(hint.notification, 'Transition: slide_left (0ms)');

      final none = transitionHintFromResponse(
        uiv1.TransitionUI()..transition = 'none',
      );
      expect(none.duration, Duration.zero);
    });
  });

  group('commandDiagnosticsFromResponse', () {
    test('classifies command diagnostics by pending request id', () {
      final response = ConnectResponse()
        ..commandResult = (CommandResult()
          ..requestId = 'device-123'
          ..data['battery'] = '93');

      final update = commandDiagnosticsFromResponse(
        response: response,
        pendingRequestIDs: const CommandDiagnosticsRequestIDs(
          deviceStatus: 'device-123',
        ),
      );

      expect(update, isNotNull);
      expect(update!.title, 'device_status');
      expect(update.data, <String, String>{'battery': '93'});
    });

    test('classifies known diagnostic notifications without request ids', () {
      final cases = <String, String>{
        'System query: runtime_status': 'runtime_status',
        'System query: device_status': 'device_status',
        'System query: scenario_registry': 'scenario_registry',
        'System query: list_playback_artifacts': 'list_playback_artifacts',
        'Playback metadata ready': 'playback_metadata',
      };

      for (final entry in cases.entries) {
        final update = commandDiagnosticsFromResponse(
          response: ConnectResponse()
            ..commandResult = (CommandResult()
              ..notification = entry.key
              ..data['ok'] = 'true'),
          pendingRequestIDs: const CommandDiagnosticsRequestIDs(),
        );

        expect(update?.title, entry.value);
        expect(update?.data, <String, String>{'ok': 'true'});
      }
    });

    test('ignores unrelated command results and empty diagnostic payloads', () {
      expect(
        commandDiagnosticsFromResponse(
          response: ConnectResponse(),
          pendingRequestIDs: const CommandDiagnosticsRequestIDs(),
        ),
        isNull,
      );
      expect(
        commandDiagnosticsFromResponse(
          response: ConnectResponse()
            ..commandResult = (CommandResult()
              ..notification = 'System query: runtime_status'),
          pendingRequestIDs: const CommandDiagnosticsRequestIDs(),
        ),
        isNull,
      );
      expect(
        commandDiagnosticsFromResponse(
          response: ConnectResponse()
            ..commandResult = (CommandResult()
              ..notification = 'regular notification'
              ..data['value'] = 'ignored'),
          pendingRequestIDs: const CommandDiagnosticsRequestIDs(),
        ),
        isNull,
      );
    });
  });

  group('registerMetadataFromResponse', () {
    test('prefers typed server metadata build values when present', () {
      final update = registerMetadataFromResponse(
        ConnectResponse()
          ..registerAck = (RegisterAck()
            ..serverMetadata = (ServerMetadata()
              ..build = (BuildMetadata()
                ..sha = 'typed-sha'
                ..dateRfc3339 = '2026-05-03T09:10:11Z'))
            ..metadata[registerMetadataServerBuildShaKey] = 'legacy-sha'
            ..metadata[registerMetadataServerBuildDateKey] = 'legacy-date'),
      );

      expect(update, isNotNull);
      expect(update!.serverBuildSha, 'typed-sha');
      expect(update.serverBuildDate, '2026-05-03T09:10:11Z');
      expect(update.metadata[registerMetadataServerBuildShaKey], 'legacy-sha');
    });

    test('extracts normalized server build metadata', () {
      final update = registerMetadataFromResponse(
        ConnectResponse()
          ..registerAck = (RegisterAck()
            ..metadata[registerMetadataServerBuildShaKey] = ' abc123 '
            ..metadata[registerMetadataServerBuildDateKey] = '2026-05-03'),
      );

      expect(update, isNotNull);
      expect(update!.serverBuildSha, 'abc123');
      expect(update.serverBuildDate, '2026-05-03');
      expect(update.hasDiagnosticsData, isTrue);
      expect(
        update.metadata[registerMetadataServerBuildShaKey],
        ' abc123 ',
      );
    });

    test('returns unknown build values for empty register metadata', () {
      final update = registerMetadataFromResponse(
        ConnectResponse()..registerAck = RegisterAck(),
      );

      expect(update, isNotNull);
      expect(update!.serverBuildSha, 'unknown');
      expect(update.serverBuildDate, 'unknown');
      expect(update.hasDiagnosticsData, isFalse);
    });

    test('ignores responses without register ack', () {
      expect(registerMetadataFromResponse(ConnectResponse()), isNull);
    });
  });

  group('media and edge response helpers', () {
    test('synchronousMediaControlUpdateFromResponse derives stream start data',
        () {
      final update = synchronousMediaControlUpdateFromResponse(
        ConnectResponse()
          ..startStream = (iov1.StartStream()
            ..streamId = 'stream-1'
            ..kind = 'audio'
            ..streamKind = iov1.StreamKind.STREAM_KIND_AUDIO
            ..sourceDeviceId = 'server'
            ..targetDeviceId = 'client'),
      );

      expect(update.startStreamID, 'stream-1');
      expect(update.shouldAcknowledgeStartStream, isTrue);
      expect(update.startStreamNotification, 'Start stream: audio (stream-1)');
      expect(update.lastNotification, 'Start stream: audio (stream-1)');
    });

    test('synchronousMediaControlUpdateFromResponse derives route and signal',
        () {
      final routeUpdate = synchronousMediaControlUpdateFromResponse(
        ConnectResponse()
          ..routeStream = (iov1.RouteStream()
            ..streamId = 'video-1'
            ..sourceDeviceId = 'source'
            ..targetDeviceId = 'target'
            ..kind = 'video'
            ..streamKind = iov1.StreamKind.STREAM_KIND_VIDEO),
      );

      expect(routeUpdate.routeStreamID, 'video-1');
      expect(routeUpdate.routeNotification, 'Route: source -> target (video)');
      expect(routeUpdate.lastNotification, 'Route: source -> target (video)');

      final signalUpdate = synchronousMediaControlUpdateFromResponse(
        ConnectResponse()
          ..webrtcSignal = (WebRTCSignal()
            ..streamId = 'video-1'
            ..signalType = 'answer'
            ..signalTypeEnum =
                WebRTCSignalType.WEB_RTC_SIGNAL_TYPE_ANSWER),
      );

      expect(signalUpdate.webrtcSignalNotification,
          'WebRTC signal: answer (video-1)');
      expect(signalUpdate.lastNotification, 'WebRTC signal: answer (video-1)');
    });

    test('typed enum fields override legacy labels when both are present', () {
      // start_stream, route_stream, and webrtc_signal share a oneof on
      // ConnectResponse, so each typed payload is exercised in its own
      // response.
      final startUpdate = synchronousMediaControlUpdateFromResponse(
        ConnectResponse()
          ..startStream = (iov1.StartStream()
            ..streamId = 'stream-typed'
            ..kind = 'legacy-kind'
            ..streamKind = iov1.StreamKind.STREAM_KIND_SENSOR),
      );
      expect(startUpdate.startStreamNotification,
          'Start stream: sensor (stream-typed)');

      final routeUpdate = synchronousMediaControlUpdateFromResponse(
        ConnectResponse()
          ..routeStream = (iov1.RouteStream()
            ..streamId = 'route-typed'
            ..sourceDeviceId = 'source'
            ..targetDeviceId = 'target'
            ..kind = 'legacy-route'
            ..streamKind = iov1.StreamKind.STREAM_KIND_DATA),
      );
      expect(routeUpdate.routeNotification,
          'Route: source -> target (data)');

      final signalUpdate = synchronousMediaControlUpdateFromResponse(
        ConnectResponse()
          ..webrtcSignal = (WebRTCSignal()
            ..streamId = 'signal-typed'
            ..signalType = 'legacy-signal'
            ..signalTypeEnum =
                WebRTCSignalType.WEB_RTC_SIGNAL_TYPE_ICE_CANDIDATE),
      );
      expect(
        signalUpdate.webrtcSignalNotification,
        'WebRTC signal: candidate (signal-typed)',
      );
    });

    test('synchronousMediaControlUpdateFromResponse derives stop stream data',
        () {
      final update = synchronousMediaControlUpdateFromResponse(
        ConnectResponse()
          ..stopStream = (iov1.StopStream()..streamId = 'stream-2'),
      );

      expect(update.stopStreamID, 'stream-2');
      expect(update.stopStreamNotification, 'Stop stream: stream-2');
      expect(update.lastNotification, 'Stop stream: stream-2');
    });

    test('bundleIDFromFlowPlan returns the first non-empty bundle id', () {
      expect(bundleIDFromFlowPlan(null), isNull);
      expect(bundleIDFromFlowPlan(iov1.FlowPlan()), isNull);
      expect(
        bundleIDFromFlowPlan(
          iov1.FlowPlan()
            ..nodes.addAll(<iov1.FlowNode>[
              iov1.FlowNode()..args['bundle_id'] = '   ',
              iov1.FlowNode()..args['bundle_id'] = ' bundle-a ',
              iov1.FlowNode()..args['bundle_id'] = 'bundle-b',
            ]),
        ),
        'bundle-a',
      );
    });

    test('playAudio helpers preserve source label and pcm byte count', () {
      expect(
        playAudioSourceLabel(iov1.PlayAudio()..pcmData = <int>[1, 2, 3]),
        'pcm_data',
      );
      expect(
        playAudioPcmByteCount(iov1.PlayAudio()..pcmData = <int>[1, 2, 3]),
        3,
      );
      expect(playAudioSourceLabel(iov1.PlayAudio()..url = 'https://x'), 'url');
      expect(playAudioPcmByteCount(iov1.PlayAudio()..url = 'https://x'), 0);
      expect(playAudioSourceLabel(iov1.PlayAudio()..ttsText = 'hello'),
          'tts_text');
      expect(playAudioSourceLabel(iov1.PlayAudio()), 'not_set');
    });

    test('firstPlaybackArtifactID returns first sorted non-empty id', () {
      expect(firstPlaybackArtifactID(const <String, String>{}), '');
      expect(
        firstPlaybackArtifactID(const <String, String>{
          'b': ' artifact-b | meta ',
          'a': '   ',
          'c': 'artifact-c',
        }),
        'artifact-b',
      );
      expect(
        firstPlaybackArtifactID(const <String, String>{
          'a': 'artifact-a|meta',
          'b': 'artifact-b|meta',
        }),
        'artifact-a',
      );
    });

    test('applicationIntentsFromDiagnostics keeps default first and sorts data',
        () {
      expect(
        applicationIntentsFromDiagnostics(const <String, String>{
          'weather': '',
          ' terminal ': '',
          'lights': '',
          '': '',
        }),
        <String>['terminal', 'lights', 'weather'],
      );
      expect(
        applicationIntentsFromDiagnostics(
          const <String, String>{'z': '', 'a': ''},
          defaultIntent: 'launcher',
        ),
        <String>['launcher', 'a', 'z'],
      );
    });
  });
}
