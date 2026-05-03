import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/connection/control_response_dispatcher.dart';
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';
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
  });
}
