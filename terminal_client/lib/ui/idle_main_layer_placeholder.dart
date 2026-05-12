import 'package:terminal_client/gen/terminals/ui/v1/ui.pb.dart' as uiv1;

/// Canonical main-layer placeholder until the first `SetUI` (terminal-ui
/// plan, Phase H). Wire shape must match [ui.IdleMainLayerPlaceholder] on the
/// server; `TestIdleMainLayerPlaceholderGoldenWire` guards the Go side and
/// `idle_main_layer_placeholder_test.dart` asserts semantic parity with the
/// unmarshaled golden protobuf.
uiv1.Node idleMainLayerPlaceholderRoot() {
  return uiv1.Node()
    ..id = '__runtime.main_placeholder.root'
    ..props['client_chrome'] = 'hidden'
    ..props['background'] = '#101418'
    ..props['type'] = 'stack'
    ..stack = uiv1.StackWidget()
    ..children.add(
      uiv1.Node()
        ..id = '__runtime.main_placeholder.message'
        ..props['type'] = 'text'
        ..props['value'] = 'Awaiting server UI'
        ..props['style'] = 'headline'
        ..props['color'] = '#E7F0F7'
        ..text = (uiv1.TextWidget()
          ..value = 'Awaiting server UI'
          ..style = 'headline'
          ..color = '#E7F0F7'),
    );
}
