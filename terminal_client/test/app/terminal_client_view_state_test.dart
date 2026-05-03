import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/app/terminal_client_view_state.dart';
import 'package:terminal_client/gen/terminals/ui/v1/ui.pb.dart' as uiv1;

void main() {
  group('clientChromeModeFromRoot', () {
    test('defaults to standard chrome without a root', () {
      expect(clientChromeModeFromRoot(null), ClientChromeMode.standard);
    });

    test('defaults to standard chrome without a chrome prop', () {
      final root = uiv1.Node()..stack = uiv1.StackWidget();

      expect(clientChromeModeFromRoot(root), ClientChromeMode.standard);
      expect(shouldHideClientChromeForRoot(root), isFalse);
    });

    test('uses generic client_chrome prop to hide chrome', () {
      final root = uiv1.Node()
        ..props['client_chrome'] = 'hidden'
        ..stack = uiv1.StackWidget();

      expect(clientChromeModeFromRoot(root), ClientChromeMode.hidden);
      expect(shouldHideClientChromeForRoot(root), isTrue);
    });

    test('accepts chrome fullscreen as a generic hidden-chrome alias', () {
      final root = uiv1.Node()
        ..props['chrome'] = 'fullscreen'
        ..stack = uiv1.StackWidget();

      expect(clientChromeModeFromRoot(root), ClientChromeMode.hidden);
    });
  });
}
