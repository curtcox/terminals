import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/gen/terminals/ui/v1/ui.pb.dart' as uiv1;
import 'package:terminal_client/ui/renderer_policy.dart';
import 'package:terminal_client/ui/server_driven_action.dart';
import 'package:terminal_client/ui/server_driven_renderer.dart';

void main() {
  Widget harness(
    uiv1.Node root, {
    void Function(ServerDrivenAction action)? onAction,
    MediaSurfaceBuilder? mediaSurfaceBuilder,
    AudioVisualizerBuilder? audioVisualizerBuilder,
    ImageLoader? imageLoader,
  }) {
    return MaterialApp(
      home: Scaffold(
        body: ServerDrivenRenderer(
          root: root,
          onAction: onAction ?? (_) {},
          mediaSurfaceBuilder: mediaSurfaceBuilder,
          audioVisualizerBuilder: audioVisualizerBuilder,
          imageLoader: imageLoader,
        ),
      ),
    );
  }

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

  testWidgets('renders structural primitives with stable keys', (tester) async {
    final root = uiv1.Node()
      ..id = 'stack'
      ..stack = (uiv1.StackWidget())
      ..children.addAll([
        uiv1.Node()
          ..id = 'row'
          ..row = (uiv1.RowWidget())
          ..children.addAll([
            uiv1.Node()
              ..id = 'padding'
              ..padding = (uiv1.PaddingWidget()..all = 12)
              ..children.add(
                uiv1.Node()
                  ..id = 'center'
                  ..center = (uiv1.CenterWidget())
                  ..children.add(
                    uiv1.Node()..text = (uiv1.TextWidget()..value = 'Centered'),
                  ),
              ),
            uiv1.Node()
              ..id = 'expand'
              ..expand = (uiv1.ExpandWidget())
              ..children.add(
                uiv1.Node()..text = (uiv1.TextWidget()..value = 'Expanded'),
              ),
          ]),
        uiv1.Node()
          ..id = 'grid'
          ..grid = (uiv1.GridWidget()..columns = 2)
          ..children.addAll([
            uiv1.Node()..text = (uiv1.TextWidget()..value = 'Cell A'),
            uiv1.Node()..text = (uiv1.TextWidget()..value = 'Cell B'),
          ]),
      ]);

    await tester.pumpWidget(harness(root));

    expect(
        find.byKey(const ValueKey<String>('ui-stack-stack')), findsOneWidget);
    expect(find.byKey(const ValueKey<String>('ui-row-row')), findsOneWidget);
    expect(
      find.byKey(const ValueKey<String>('ui-padding-padding')),
      findsOneWidget,
    );
    expect(
        find.byKey(const ValueKey<String>('ui-center-center')), findsOneWidget);
    expect(
        find.byKey(const ValueKey<String>('ui-expand-expand')), findsOneWidget);
    expect(find.byKey(const ValueKey<String>('ui-grid-grid')), findsOneWidget);
    expect(find.byKey(const ValueKey<String>('ui-text-root.0.0.0.0')),
        findsOneWidget);
    expect(find.text('Cell A'), findsOneWidget);
    expect(find.text('Cell B'), findsOneWidget);

    final padding = tester.widget<Padding>(
        find.byKey(const ValueKey<String>('ui-padding-padding')));
    expect(padding.padding, const EdgeInsets.all(12));
  });

  testWidgets('honors horizontal scroll direction', (tester) async {
    final root = uiv1.Node()
      ..id = 'scroll'
      ..scroll = (uiv1.ScrollWidget()..direction = 'horizontal')
      ..children.addAll([
        uiv1.Node()..text = (uiv1.TextWidget()..value = 'One'),
        uiv1.Node()..text = (uiv1.TextWidget()..value = 'Two'),
      ]);

    await tester.pumpWidget(harness(root));

    final scroll = tester.widget<SingleChildScrollView>(
      find.byKey(const ValueKey<String>('ui-scroll-scroll')),
    );
    expect(scroll.scrollDirection, Axis.horizontal);
    expect(find.descendant(of: find.byType(Row), matching: find.text('One')),
        findsOneWidget);
  });

  testWidgets('renders input primitives and emits values', (tester) async {
    final actions = <ServerDrivenAction>[];
    final root = uiv1.Node()
      ..id = 'controls'
      ..stack = (uiv1.StackWidget())
      ..children.addAll([
        uiv1.Node()
          ..id = 'name'
          ..textInput = (uiv1.TextInputWidget()..placeholder = 'Name'),
        uiv1.Node()
          ..id = 'enabled'
          ..toggle = (uiv1.ToggleWidget()..value = false),
        uiv1.Node()
          ..id = 'choice'
          ..dropdown = (uiv1.DropdownWidget()
            ..options.addAll(<String>['a', 'b'])
            ..value = 'a'),
        uiv1.Node()
          ..id = 'tap-zone'
          ..gestureArea = (uiv1.GestureAreaWidget()..action = 'press')
          ..children.add(
            uiv1.Node()..text = (uiv1.TextWidget()..value = 'Tap here'),
          ),
      ]);

    await tester.pumpWidget(harness(root, onAction: actions.add));

    await tester.enterText(
      find.byKey(const ValueKey<String>('ui-text-input-name')),
      'Ada',
    );
    await tester.testTextInput.receiveAction(TextInputAction.done);
    tester
        .widget<SwitchListTile>(
          find.byKey(const ValueKey<String>('ui-toggle-enabled')),
        )
        .onChanged
        ?.call(true);
    tester
        .widget<DropdownButton<String>>(
          find.byKey(const ValueKey<String>('ui-dropdown-choice')),
        )
        .onChanged
        ?.call('b');
    tester
        .widget<GestureDetector>(
          find.byKey(const ValueKey<String>('ui-gesture-tap-zone')),
        )
        .onTap
        ?.call();
    await tester.pump();

    expect(
      actions,
      contains(
        isA<ServerDrivenAction>()
            .having((action) => action.componentId, 'componentId', 'name')
            .having((action) => action.action, 'action', 'submit')
            .having((action) => action.value, 'value', 'Ada'),
      ),
    );
    expect(
      actions,
      contains(
        isA<ServerDrivenAction>()
            .having((action) => action.componentId, 'componentId', 'enabled')
            .having((action) => action.action, 'action', 'toggle')
            .having((action) => action.value, 'value', 'true'),
      ),
    );
    expect(
      actions,
      contains(
        isA<ServerDrivenAction>()
            .having((action) => action.componentId, 'componentId', 'choice')
            .having((action) => action.action, 'action', 'select')
            .having((action) => action.value, 'value', 'b'),
      ),
    );
    expect(
      actions,
      contains(
        isA<ServerDrivenAction>()
            .having((action) => action.componentId, 'componentId', 'tap-zone')
            .having((action) => action.action, 'action', 'press'),
      ),
    );
  });

  testWidgets('renders overlay, progress, and device-control placeholders',
      (tester) async {
    final root = uiv1.Node()
      ..id = 'root'
      ..overlay = (uiv1.OverlayWidget())
      ..children.addAll([
        uiv1.Node()
          ..id = 'progress'
          ..progress = (uiv1.ProgressWidget()..value = 1.8),
        uiv1.Node()
          ..id = 'canvas'
          ..canvas = (uiv1.CanvasWidget()..drawOpsJson = '{"ops":[]}'),
        uiv1.Node()
          ..id = 'fullscreen'
          ..fullscreen = (uiv1.FullscreenWidget()..enabled = true),
        uiv1.Node()
          ..id = 'awake'
          ..keepAwake = (uiv1.KeepAwakeWidget()..enabled = true),
        uiv1.Node()
          ..id = 'brightness'
          ..brightness = (uiv1.BrightnessWidget()..value = -0.5),
      ]);

    await tester.pumpWidget(harness(root));

    expect(
        find.byKey(const ValueKey<String>('ui-overlay-root')), findsOneWidget);
    final progress = tester.widget<LinearProgressIndicator>(
      find.byKey(const ValueKey<String>('ui-progress-progress')),
    );
    expect(progress.value, 1.0);
    expect(
        find.byKey(const ValueKey<String>('ui-canvas-canvas')), findsOneWidget);
    expect(find.byKey(const ValueKey<String>('ui-fullscreen-fullscreen')),
        findsOneWidget);
    expect(find.byKey(const ValueKey<String>('ui-keep-awake-awake')),
        findsOneWidget);
    expect(find.byKey(const ValueKey<String>('ui-brightness-brightness')),
        findsOneWidget);
    expect(find.text('0.00'), findsOneWidget);
  });

  testWidgets('renders unsupported nodes through explicit fallback policy',
      (tester) async {
    await tester.pumpWidget(
      harness(uiv1.Node(), onAction: (_) {}),
    );

    expect(
      find.byKey(const ValueKey<String>('ui-unsupported-root')),
      findsOneWidget,
    );
    expect(find.text('Unsupported UI node'), findsOneWidget);
  });

  testWidgets('can hide unsupported nodes through renderer policy',
      (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        home: ServerDrivenRenderer(
          root: uiv1.Node(),
          onAction: (_) {},
          policy: const RendererPolicy(showFallbackDiagnostics: false),
        ),
      ),
    );

    expect(
      find.byKey(const ValueKey<String>('ui-unsupported-root')),
      findsNothing,
    );
    expect(find.byType(SizedBox), findsOneWidget);
  });
}
