import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/gen/terminals/ui/v1/ui.pb.dart' as uiv1;
import 'package:terminal_client/ui/idle_main_layer_placeholder.dart';

void main() {
  test('idleMainLayerPlaceholderRoot matches server golden semantics', () {
    final repoRelative =
        '../terminal_server/internal/transport/testdata/idle_main_layer_placeholder_root.pb';
    final golden = File(repoRelative);
    if (!golden.existsSync()) {
      fail(
        'Missing golden $repoRelative (run Go test with '
        'UPDATE_IDLE_MAIN_PLACEHOLDER_GOLDEN=1 to generate)',
      );
    }
    final want = uiv1.Node.fromBuffer(golden.readAsBytesSync());
    final got = idleMainLayerPlaceholderRoot();
    _assertMatchesIdlePlaceholder(got);
    _assertMatchesIdlePlaceholder(want);
    expect(_idlePlaceholderFingerprint(got), _idlePlaceholderFingerprint(want));
  });
}

void _assertMatchesIdlePlaceholder(uiv1.Node n) {
  expect(n.id, '__runtime.main_placeholder.root');
  expect(n.props['client_chrome'], 'hidden');
  expect(n.props['background'], '#101418');
  expect(n.props['type'], 'stack');
  expect(n.hasStack(), isTrue);
  expect(n.children, hasLength(1));
  final child = n.children.first;
  expect(child.id, '__runtime.main_placeholder.message');
  expect(child.props['type'], 'text');
  expect(child.hasText(), isTrue);
  expect(child.text.value, 'Awaiting server UI');
  expect(child.text.style, 'headline');
  expect(child.text.color, '#E7F0F7');
}

/// Stable structural digest independent of protobuf wire map ordering.
String _idlePlaceholderFingerprint(uiv1.Node n) {
  final b = StringBuffer()
    ..write(n.id)
    ..write('|');
  final keys = n.props.keys.toList()..sort();
  for (final k in keys) {
    b.write('$k=${n.props[k]};');
  }
  b.write('|stack=${n.hasStack()}|children=${n.children.length}');
  for (final c in n.children) {
    final ck = c.props.keys.toList()..sort();
    b.write('|child:${c.id}');
    for (final k in ck) {
      b.write(';$k=${c.props[k]}');
    }
    if (c.hasText()) {
      b.write(
        ';text=${c.text.value}/${c.text.style}/${c.text.color}',
      );
    }
  }
  return b.toString();
}
