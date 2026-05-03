import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/connection/control_response_dispatcher.dart';
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';
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
}
