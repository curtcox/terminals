import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/gen/terminals/ui/v1/ui.pb.dart' as uiv1;
import 'package:terminal_client/ui/server_driven_action.dart';
import 'package:terminal_client/ui/server_driven_renderer.dart';

void main() {
  testWidgets('renders layout and text primitives', (tester) async {
    final root = uiv1.Node()
      ..id = 'root'
      ..stack = (uiv1.StackWidget())
      ..props['background'] = '#112233'
      ..children.add(
        uiv1.Node()
          ..id = 'greeting'
          ..text = (uiv1.TextWidget()
            ..value = 'Hello terminal'
            ..style = 'monospace'
            ..color = '#ffeeaa'),
      );

    await tester.pumpWidget(
      MaterialApp(
        home: ServerDrivenRenderer(
          root: root,
          onAction: (_) {},
        ),
      ),
    );

    expect(find.byKey(const ValueKey<String>('ui-stack-root')), findsOneWidget);
    expect(
      find.byKey(const ValueKey<String>('ui-text-greeting')),
      findsOneWidget,
    );
    expect(find.text('Hello terminal'), findsOneWidget);
  });

  testWidgets('emits generic actions for controls', (tester) async {
    final actions = <ServerDrivenAction>[];
    final root = uiv1.Node()
      ..stack = (uiv1.StackWidget())
      ..children.addAll([
        uiv1.Node()
          ..id = 'go'
          ..button = (uiv1.ButtonWidget()
            ..label = 'Go'
            ..action = 'launch'),
        uiv1.Node()
          ..id = 'volume'
          ..slider = (uiv1.SliderWidget()
            ..min = 0
            ..max = 10
            ..value = 5),
      ]);

    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: ServerDrivenRenderer(
            root: root,
            onAction: actions.add,
          ),
        ),
      ),
    );

    await tester.tap(find.text('Go'));
    tester
        .widget<Slider>(find.byKey(const ValueKey<String>('ui-slider-volume')))
        .onChanged
        ?.call(7);
    await tester.pump();

    expect(
      actions,
      contains(
        isA<ServerDrivenAction>()
            .having((action) => action.componentId, 'componentId', 'go')
            .having((action) => action.action, 'action', 'launch'),
      ),
    );
    expect(
      actions,
      contains(
        isA<ServerDrivenAction>()
            .having((action) => action.componentId, 'componentId', 'volume')
            .having((action) => action.action, 'action', 'change')
            .having((action) => action.value, 'value', '7.0'),
      ),
    );
  });

  testWidgets('delegates media and image surfaces to injected builders',
      (tester) async {
    final requestedMedia = <String>[];
    final requestedImages = <String>[];
    final root = uiv1.Node()
      ..scroll = (uiv1.ScrollWidget())
      ..children.addAll([
        uiv1.Node()
          ..id = 'camera'
          ..videoSurface = (uiv1.VideoSurfaceWidget()..trackId = 'track-1'),
        uiv1.Node()
          ..id = 'photo'
          ..image = (uiv1.ImageWidget()..url = 'https://example.invalid/a.png'),
      ]);

    await tester.pumpWidget(
      MaterialApp(
        home: ServerDrivenRenderer(
          root: root,
          onAction: (_) {},
          mediaSurfaceBuilder: (context, componentId, trackId) {
            requestedMedia.add('$componentId:$trackId');
            return const Text('media slot');
          },
          imageLoader: (context, url) {
            requestedImages.add(url);
            return const Text('image slot');
          },
        ),
      ),
    );

    expect(requestedMedia, <String>['camera:track-1']);
    expect(requestedImages, <String>['https://example.invalid/a.png']);
    expect(find.text('media slot'), findsOneWidget);
    expect(find.text('image slot'), findsOneWidget);
  });
}
