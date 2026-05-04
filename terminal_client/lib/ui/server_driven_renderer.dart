import 'dart:async';

import 'package:flutter/material.dart';
import 'package:terminal_client/gen/terminals/ui/v1/ui.pb.dart' as uiv1;
import 'package:terminal_client/ui/primitive_props.dart';
import 'package:terminal_client/ui/renderer_policy.dart';
import 'package:terminal_client/ui/server_driven_action.dart';
import 'package:terminal_client/ui/server_driven_node_key.dart';

typedef ServerDrivenActionHandler = void Function(ServerDrivenAction action);

typedef MediaSurfaceBuilder = Widget Function(
  BuildContext context,
  String componentId,
  String trackId,
);

typedef AudioVisualizerBuilder = Widget Function(
  BuildContext context,
  String componentId,
  String streamId,
);

typedef ImageLoader = Widget Function(
  BuildContext context,
  String url,
);

typedef TextInputBindingResolver = ServerDrivenTextInputBinding? Function(
  String componentId,
);

class ServerDrivenTextInputBinding {
  const ServerDrivenTextInputBinding({
    required this.controller,
    this.focusNode,
    this.autofocus,
    this.onChanged,
    this.onSubmitted,
  });

  final TextEditingController controller;
  final FocusNode? focusNode;
  final bool? autofocus;
  final ValueChanged<String>? onChanged;
  final FutureOr<void> Function(String value)? onSubmitted;
}

class ServerDrivenRenderer extends StatelessWidget {
  const ServerDrivenRenderer({
    super.key,
    required this.root,
    required this.onAction,
    this.mediaSurfaceBuilder,
    this.audioVisualizerBuilder,
    this.imageLoader,
    this.textInputBindingResolver,
    this.policy = const RendererPolicy(),
  });

  final uiv1.Node root;
  final ServerDrivenActionHandler onAction;
  final MediaSurfaceBuilder? mediaSurfaceBuilder;
  final AudioVisualizerBuilder? audioVisualizerBuilder;
  final ImageLoader? imageLoader;
  final TextInputBindingResolver? textInputBindingResolver;
  final RendererPolicy policy;

  @override
  Widget build(BuildContext context) => _renderNode(context, root);

  Widget _renderNode(
    BuildContext context,
    uiv1.Node node, [
    String path = 'root',
  ]) {
    switch (node.whichWidget()) {
      case uiv1.Node_Widget.stack:
        return Container(
          key: _key('stack', node, path),
          color: parseHexColor(node.props['background']),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: _renderChildren(context, node.children, path),
          ),
        );
      case uiv1.Node_Widget.row:
        return Row(
          key: _key('row', node, path),
          children: _renderChildren(context, node.children, path),
        );
      case uiv1.Node_Widget.grid:
        final columns = node.grid.columns > 0 ? node.grid.columns : 1;
        return LayoutBuilder(
          key: _key('grid', node, path),
          builder: (context, constraints) {
            const spacing = 8.0;
            final maxWidth = constraints.maxWidth.isFinite
                ? constraints.maxWidth
                : MediaQuery.of(context).size.width;
            final totalSpacing = spacing * (columns - 1);
            final itemWidth =
                columns <= 1 ? maxWidth : (maxWidth - totalSpacing) / columns;
            return Wrap(
              spacing: spacing,
              runSpacing: spacing,
              children: List<Widget>.generate(
                node.children.length,
                (index) => SizedBox(
                  width: itemWidth,
                  child: _renderNode(
                    context,
                    node.children[index],
                    '$path.$index',
                  ),
                ),
              ),
            );
          },
        );
      case uiv1.Node_Widget.scroll:
        final scroll = node.scroll;
        final bool isHorizontal;
        if (scroll.directionEnum !=
            uiv1.ScrollDirection.SCROLL_DIRECTION_UNSPECIFIED) {
          isHorizontal = scroll.directionEnum ==
              uiv1.ScrollDirection.SCROLL_DIRECTION_HORIZONTAL;
        } else {
          isHorizontal = scroll.direction.trim().toLowerCase() == 'horizontal';
        }
        return SingleChildScrollView(
          key: _key('scroll', node, path),
          scrollDirection: isHorizontal ? Axis.horizontal : Axis.vertical,
          child: isHorizontal
              ? Row(children: _renderChildren(context, node.children, path))
              : Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: _renderChildren(context, node.children, path),
                ),
        );
      case uiv1.Node_Widget.padding:
        return Padding(
          key: _key('padding', node, path),
          padding: EdgeInsets.all(node.padding.all.toDouble()),
          child: _renderNodeChildren(context, node.children, path),
        );
      case uiv1.Node_Widget.center:
        return Center(
          key: _key('center', node, path),
          child: _renderNodeChildren(context, node.children, path),
        );
      case uiv1.Node_Widget.expand:
        return Expanded(
          key: _key('expand', node, path),
          child: _renderNodeChildren(context, node.children, path),
        );
      case uiv1.Node_Widget.text:
        return Padding(
          key: _key('text', node, path),
          padding: const EdgeInsets.symmetric(vertical: 4),
          child: SelectableText(
            node.text.value,
            style: TextStyle(
              color: parseHexColor(node.text.color),
              fontFamily: node.text.style == 'monospace' ? 'monospace' : null,
            ),
          ),
        );
      case uiv1.Node_Widget.textInput:
        return _textInput(node, path);
      case uiv1.Node_Widget.button:
        final componentId = serverDrivenNodeId(node);
        return Padding(
          key: _key('button-padding', node, path),
          padding: const EdgeInsets.symmetric(vertical: 4),
          child: ElevatedButton(
            key: _key('button', node, path),
            onPressed: () => onAction(
              ServerDrivenAction(
                componentId: componentId.isNotEmpty ? componentId : 'button',
                action:
                    node.button.action.isNotEmpty ? node.button.action : 'tap',
              ),
            ),
            child: Text(node.button.label),
          ),
        );
      case uiv1.Node_Widget.slider:
        final componentId = serverDrivenNodeId(node);
        final min = node.slider.min;
        final max = node.slider.max > min ? node.slider.max : min + 1;
        final value = node.slider.value.clamp(min, max).toDouble();
        return Slider(
          key: _key('slider', node, path),
          value: value,
          min: min,
          max: max,
          onChanged: (nextValue) => onAction(
            ServerDrivenAction(
              componentId: componentId.isNotEmpty ? componentId : 'slider',
              action: 'change',
              value: nextValue.toString(),
            ),
          ),
        );
      case uiv1.Node_Widget.toggle:
        final componentId = serverDrivenNodeId(node);
        return SwitchListTile(
          key: _key('toggle', node, path),
          value: node.toggle.value,
          onChanged: (nextValue) => onAction(
            ServerDrivenAction(
              componentId: componentId.isNotEmpty ? componentId : 'toggle',
              action: 'toggle',
              value: nextValue.toString(),
            ),
          ),
        );
      case uiv1.Node_Widget.dropdown:
        return _dropdown(node, path);
      case uiv1.Node_Widget.gestureArea:
        return _gestureArea(context, node, path);
      case uiv1.Node_Widget.overlay:
        return Stack(
          key: _key('overlay', node, path),
          fit: StackFit.loose,
          children: _renderChildren(context, node.children, path),
        );
      case uiv1.Node_Widget.videoSurface:
        final componentId = serverDrivenNodeId(node);
        final trackId = node.videoSurface.trackId.trim();
        final builder = mediaSurfaceBuilder;
        if (builder != null) {
          return KeyedSubtree(
            key: _key('video-surface', node, path),
            child: builder(context, componentId, trackId),
          );
        }
        return _placeholderPrimitive(
          key: _key('video-surface', node, path),
          title: 'Video surface',
          detail: trackId,
        );
      case uiv1.Node_Widget.audioVisualizer:
        final componentId = serverDrivenNodeId(node);
        final streamId = node.audioVisualizer.streamId.trim();
        final builder = audioVisualizerBuilder;
        if (builder != null) {
          return KeyedSubtree(
            key: _key('audio-visualizer', node, path),
            child: builder(context, componentId, streamId),
          );
        }
        return _placeholderPrimitive(
          key: _key('audio-visualizer', node, path),
          title: 'Audio level',
          detail: streamId,
        );
      case uiv1.Node_Widget.canvas:
        final drawOps = node.canvas.drawOpsJson.trim();
        final drawOpsPreview = drawOps.isEmpty
            ? 'No draw ops'
            : (drawOps.length > 64
                ? '${drawOps.substring(0, 64)}...'
                : drawOps);
        return _placeholderPrimitive(
          key: _key('canvas', node, path),
          title: 'Canvas',
          detail: drawOpsPreview,
        );
      case uiv1.Node_Widget.fullscreen:
        return _placeholderPrimitive(
          key: _key('fullscreen', node, path),
          title:
              'Fullscreen ${node.fullscreen.enabled ? 'enabled' : 'disabled'}',
          child: _renderNodeChildren(context, node.children, path),
        );
      case uiv1.Node_Widget.keepAwake:
        return _placeholderPrimitive(
          key: _key('keep-awake', node, path),
          title:
              'Keep awake ${node.keepAwake.enabled ? 'enabled' : 'disabled'}',
          child: _renderNodeChildren(context, node.children, path),
        );
      case uiv1.Node_Widget.brightness:
        final brightness = node.brightness.value.clamp(0.0, 1.0).toDouble();
        return _placeholderPrimitive(
          key: _key('brightness', node, path),
          title: 'Brightness hint',
          detail: brightness.toStringAsFixed(2),
          child: _renderNodeChildren(context, node.children, path),
        );
      case uiv1.Node_Widget.image:
        final loader = imageLoader;
        if (loader != null) {
          return KeyedSubtree(
            key: _key('image', node, path),
            child: loader(context, node.image.url),
          );
        }
        return Image.network(
          node.image.url,
          key: _key('image', node, path),
          fit: BoxFit.cover,
          errorBuilder: (context, error, stackTrace) {
            return const Icon(Icons.broken_image_outlined);
          },
        );
      case uiv1.Node_Widget.progress:
        return LinearProgressIndicator(
          key: _key('progress', node, path),
          value: node.progress.value.clamp(0.0, 1.0).toDouble(),
        );
      case uiv1.Node_Widget.notSet:
        break;
    }
    return _fallback(node, path);
  }

  Widget _textInput(uiv1.Node node, String path) {
    final componentId = serverDrivenNodeId(node);
    final binding = textInputBindingResolver?.call(componentId);
    final controller = binding?.controller;
    return TextField(
      key: _key('text-input', node, path),
      controller: controller,
      focusNode: binding?.focusNode,
      decoration: InputDecoration(
        hintText: node.textInput.placeholder,
      ),
      autofocus: binding?.autofocus ?? node.textInput.autofocus,
      onChanged: binding?.onChanged,
      onSubmitted: (value) async {
        final customSubmit = binding?.onSubmitted;
        if (customSubmit != null) {
          await customSubmit(value);
          return;
        }
        onAction(
          ServerDrivenAction(
            componentId: componentId.isNotEmpty ? componentId : 'text_input',
            action: 'submit',
            value: value,
          ),
        );
        controller?.clear();
      },
    );
  }

  Widget _dropdown(uiv1.Node node, String path) {
    final componentId = serverDrivenNodeId(node);
    final options = node.dropdown.options;
    final selected = options.contains(node.dropdown.value)
        ? node.dropdown.value
        : (options.isNotEmpty ? options.first : null);
    return DropdownButton<String>(
      key: _key('dropdown', node, path),
      isExpanded: true,
      value: selected,
      hint: const Text('Select option'),
      items: options
          .map(
            (option) => DropdownMenuItem<String>(
              value: option,
              child: Text(option),
            ),
          )
          .toList(),
      onChanged: options.isEmpty
          ? null
          : (nextValue) {
              if (nextValue == null) {
                return;
              }
              onAction(
                ServerDrivenAction(
                  componentId:
                      componentId.isNotEmpty ? componentId : 'dropdown',
                  action: 'select',
                  value: nextValue,
                ),
              );
            },
    );
  }

  Widget _gestureArea(BuildContext context, uiv1.Node node, String path) {
    final componentId = serverDrivenNodeId(node);
    final action =
        node.gestureArea.action.isNotEmpty ? node.gestureArea.action : 'tap';
    final child = _renderNodeChildren(context, node.children, path);
    return GestureDetector(
      key: _key('gesture', node, path),
      behavior: HitTestBehavior.opaque,
      onTap: () => onAction(
        ServerDrivenAction(
          componentId: componentId.isNotEmpty ? componentId : 'gesture_area',
          action: action,
        ),
      ),
      child:
          node.children.isEmpty ? const SizedBox(width: 48, height: 48) : child,
    );
  }

  List<Widget> _renderChildren(
    BuildContext context,
    List<uiv1.Node> children,
    String parentPath,
  ) {
    return List<Widget>.generate(
      children.length,
      (index) => _renderNode(context, children[index], '$parentPath.$index'),
    );
  }

  Widget _renderNodeChildren(
    BuildContext context,
    List<uiv1.Node> children,
    String parentPath,
  ) {
    if (children.isEmpty) {
      return const SizedBox.shrink();
    }
    if (children.length == 1) {
      return _renderNode(context, children.first, '$parentPath.0');
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: _renderChildren(context, children, parentPath),
    );
  }

  Widget _fallback(uiv1.Node node, String path) {
    if (!policy.showFallbackDiagnostics) {
      return const SizedBox.shrink();
    }
    return _placeholderPrimitive(
      key: _key('unsupported', node, path),
      title: 'Unsupported UI node',
    );
  }

  Widget _placeholderPrimitive({
    required Key key,
    required String title,
    String? detail,
    Widget? child,
  }) {
    return Container(
      key: key,
      margin: const EdgeInsets.symmetric(vertical: 6),
      padding: const EdgeInsets.all(8),
      decoration: BoxDecoration(
        border: Border.all(color: Colors.blueGrey.shade200),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(title),
          if (detail != null && detail.isNotEmpty) ...[
            const SizedBox(height: 4),
            Text(detail, style: const TextStyle(fontSize: 12)),
          ],
          if (child != null) ...[
            const SizedBox(height: 6),
            child,
          ],
        ],
      ),
    );
  }

  ValueKey<String> _key(String kind, uiv1.Node node, String path) {
    final id = serverDrivenNodeId(node);
    return ValueKey<String>(
      id.isEmpty ? 'ui-$kind-$path' : 'ui-$kind-$id',
    );
  }
}
