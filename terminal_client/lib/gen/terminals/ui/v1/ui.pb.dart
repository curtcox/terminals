// This is a generated file - do not edit.
//
// Generated from terminals/ui/v1/ui.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

class SetUI extends $pb.GeneratedMessage {
  factory SetUI({
    $core.String? deviceId,
    Node? root,
  }) {
    final result = create();
    if (deviceId != null) result.deviceId = deviceId;
    if (root != null) result.root = root;
    return result;
  }

  SetUI._();

  factory SetUI.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SetUI.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SetUI',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'deviceId')
    ..aOM<Node>(2, _omitFieldNames ? '' : 'root', subBuilder: Node.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetUI clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SetUI copyWith(void Function(SetUI) updates) =>
      super.copyWith((message) => updates(message as SetUI)) as SetUI;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SetUI create() => SetUI._();
  @$core.override
  SetUI createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SetUI getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<SetUI>(create);
  static SetUI? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get deviceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set deviceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDeviceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDeviceId() => $_clearField(1);

  @$pb.TagNumber(2)
  Node get root => $_getN(1);
  @$pb.TagNumber(2)
  set root(Node value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasRoot() => $_has(1);
  @$pb.TagNumber(2)
  void clearRoot() => $_clearField(2);
  @$pb.TagNumber(2)
  Node ensureRoot() => $_ensure(1);
}

class UpdateUI extends $pb.GeneratedMessage {
  factory UpdateUI({
    $core.String? deviceId,
    $core.String? componentId,
    Node? node,
  }) {
    final result = create();
    if (deviceId != null) result.deviceId = deviceId;
    if (componentId != null) result.componentId = componentId;
    if (node != null) result.node = node;
    return result;
  }

  UpdateUI._();

  factory UpdateUI.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory UpdateUI.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'UpdateUI',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'deviceId')
    ..aOS(2, _omitFieldNames ? '' : 'componentId')
    ..aOM<Node>(3, _omitFieldNames ? '' : 'node', subBuilder: Node.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UpdateUI clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UpdateUI copyWith(void Function(UpdateUI) updates) =>
      super.copyWith((message) => updates(message as UpdateUI)) as UpdateUI;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static UpdateUI create() => UpdateUI._();
  @$core.override
  UpdateUI createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static UpdateUI getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<UpdateUI>(create);
  static UpdateUI? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get deviceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set deviceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDeviceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDeviceId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get componentId => $_getSZ(1);
  @$pb.TagNumber(2)
  set componentId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasComponentId() => $_has(1);
  @$pb.TagNumber(2)
  void clearComponentId() => $_clearField(2);

  @$pb.TagNumber(3)
  Node get node => $_getN(2);
  @$pb.TagNumber(3)
  set node(Node value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasNode() => $_has(2);
  @$pb.TagNumber(3)
  void clearNode() => $_clearField(3);
  @$pb.TagNumber(3)
  Node ensureNode() => $_ensure(2);
}

class TransitionUI extends $pb.GeneratedMessage {
  factory TransitionUI({
    $core.String? deviceId,
    $core.String? transition,
    $core.int? durationMs,
  }) {
    final result = create();
    if (deviceId != null) result.deviceId = deviceId;
    if (transition != null) result.transition = transition;
    if (durationMs != null) result.durationMs = durationMs;
    return result;
  }

  TransitionUI._();

  factory TransitionUI.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TransitionUI.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TransitionUI',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'deviceId')
    ..aOS(2, _omitFieldNames ? '' : 'transition')
    ..aI(3, _omitFieldNames ? '' : 'durationMs')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TransitionUI clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TransitionUI copyWith(void Function(TransitionUI) updates) =>
      super.copyWith((message) => updates(message as TransitionUI))
          as TransitionUI;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TransitionUI create() => TransitionUI._();
  @$core.override
  TransitionUI createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TransitionUI getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TransitionUI>(create);
  static TransitionUI? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get deviceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set deviceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDeviceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDeviceId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get transition => $_getSZ(1);
  @$pb.TagNumber(2)
  set transition($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTransition() => $_has(1);
  @$pb.TagNumber(2)
  void clearTransition() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get durationMs => $_getIZ(2);
  @$pb.TagNumber(3)
  set durationMs($core.int value) => $_setSignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDurationMs() => $_has(2);
  @$pb.TagNumber(3)
  void clearDurationMs() => $_clearField(3);
}

class Notification extends $pb.GeneratedMessage {
  factory Notification({
    $core.String? deviceId,
    $core.String? title,
    $core.String? body,
    $core.String? level,
  }) {
    final result = create();
    if (deviceId != null) result.deviceId = deviceId;
    if (title != null) result.title = title;
    if (body != null) result.body = body;
    if (level != null) result.level = level;
    return result;
  }

  Notification._();

  factory Notification.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Notification.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Notification',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'deviceId')
    ..aOS(2, _omitFieldNames ? '' : 'title')
    ..aOS(3, _omitFieldNames ? '' : 'body')
    ..aOS(4, _omitFieldNames ? '' : 'level')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Notification clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Notification copyWith(void Function(Notification) updates) =>
      super.copyWith((message) => updates(message as Notification))
          as Notification;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Notification create() => Notification._();
  @$core.override
  Notification createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static Notification getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<Notification>(create);
  static Notification? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get deviceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set deviceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDeviceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDeviceId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get title => $_getSZ(1);
  @$pb.TagNumber(2)
  set title($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTitle() => $_has(1);
  @$pb.TagNumber(2)
  void clearTitle() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get body => $_getSZ(2);
  @$pb.TagNumber(3)
  set body($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasBody() => $_has(2);
  @$pb.TagNumber(3)
  void clearBody() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get level => $_getSZ(3);
  @$pb.TagNumber(4)
  set level($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasLevel() => $_has(3);
  @$pb.TagNumber(4)
  void clearLevel() => $_clearField(4);
}

enum Node_Widget {
  stack,
  row,
  grid,
  scroll,
  padding,
  center,
  expand,
  text,
  image,
  videoSurface,
  audioVisualizer,
  canvas,
  textInput,
  button,
  slider,
  toggle,
  dropdown,
  gestureArea,
  overlay,
  progress,
  fullscreen,
  keepAwake,
  brightness,
  notSet
}

class Node extends $pb.GeneratedMessage {
  factory Node({
    $core.String? id,
    $core.Iterable<$core.MapEntry<$core.String, $core.String>>? props,
    $core.Iterable<Node>? children,
    StackWidget? stack,
    RowWidget? row,
    GridWidget? grid,
    ScrollWidget? scroll,
    PaddingWidget? padding,
    CenterWidget? center,
    ExpandWidget? expand,
    TextWidget? text,
    ImageWidget? image,
    VideoSurfaceWidget? videoSurface,
    AudioVisualizerWidget? audioVisualizer,
    CanvasWidget? canvas,
    TextInputWidget? textInput,
    ButtonWidget? button,
    SliderWidget? slider,
    ToggleWidget? toggle,
    DropdownWidget? dropdown,
    GestureAreaWidget? gestureArea,
    OverlayWidget? overlay,
    ProgressWidget? progress,
    FullscreenWidget? fullscreen,
    KeepAwakeWidget? keepAwake,
    BrightnessWidget? brightness,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (props != null) result.props.addEntries(props);
    if (children != null) result.children.addAll(children);
    if (stack != null) result.stack = stack;
    if (row != null) result.row = row;
    if (grid != null) result.grid = grid;
    if (scroll != null) result.scroll = scroll;
    if (padding != null) result.padding = padding;
    if (center != null) result.center = center;
    if (expand != null) result.expand = expand;
    if (text != null) result.text = text;
    if (image != null) result.image = image;
    if (videoSurface != null) result.videoSurface = videoSurface;
    if (audioVisualizer != null) result.audioVisualizer = audioVisualizer;
    if (canvas != null) result.canvas = canvas;
    if (textInput != null) result.textInput = textInput;
    if (button != null) result.button = button;
    if (slider != null) result.slider = slider;
    if (toggle != null) result.toggle = toggle;
    if (dropdown != null) result.dropdown = dropdown;
    if (gestureArea != null) result.gestureArea = gestureArea;
    if (overlay != null) result.overlay = overlay;
    if (progress != null) result.progress = progress;
    if (fullscreen != null) result.fullscreen = fullscreen;
    if (keepAwake != null) result.keepAwake = keepAwake;
    if (brightness != null) result.brightness = brightness;
    return result;
  }

  Node._();

  factory Node.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Node.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, Node_Widget> _Node_WidgetByTag = {
    10: Node_Widget.stack,
    11: Node_Widget.row,
    12: Node_Widget.grid,
    13: Node_Widget.scroll,
    14: Node_Widget.padding,
    15: Node_Widget.center,
    16: Node_Widget.expand,
    17: Node_Widget.text,
    18: Node_Widget.image,
    19: Node_Widget.videoSurface,
    20: Node_Widget.audioVisualizer,
    21: Node_Widget.canvas,
    22: Node_Widget.textInput,
    23: Node_Widget.button,
    24: Node_Widget.slider,
    25: Node_Widget.toggle,
    26: Node_Widget.dropdown,
    27: Node_Widget.gestureArea,
    28: Node_Widget.overlay,
    29: Node_Widget.progress,
    30: Node_Widget.fullscreen,
    31: Node_Widget.keepAwake,
    32: Node_Widget.brightness,
    0: Node_Widget.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Node',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..oo(0, [
      10,
      11,
      12,
      13,
      14,
      15,
      16,
      17,
      18,
      19,
      20,
      21,
      22,
      23,
      24,
      25,
      26,
      27,
      28,
      29,
      30,
      31,
      32
    ])
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..m<$core.String, $core.String>(2, _omitFieldNames ? '' : 'props',
        entryClassName: 'Node.PropsEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OS,
        packageName: const $pb.PackageName('terminals.ui.v1'))
    ..pPM<Node>(3, _omitFieldNames ? '' : 'children', subBuilder: Node.create)
    ..aOM<StackWidget>(10, _omitFieldNames ? '' : 'stack',
        subBuilder: StackWidget.create)
    ..aOM<RowWidget>(11, _omitFieldNames ? '' : 'row',
        subBuilder: RowWidget.create)
    ..aOM<GridWidget>(12, _omitFieldNames ? '' : 'grid',
        subBuilder: GridWidget.create)
    ..aOM<ScrollWidget>(13, _omitFieldNames ? '' : 'scroll',
        subBuilder: ScrollWidget.create)
    ..aOM<PaddingWidget>(14, _omitFieldNames ? '' : 'padding',
        subBuilder: PaddingWidget.create)
    ..aOM<CenterWidget>(15, _omitFieldNames ? '' : 'center',
        subBuilder: CenterWidget.create)
    ..aOM<ExpandWidget>(16, _omitFieldNames ? '' : 'expand',
        subBuilder: ExpandWidget.create)
    ..aOM<TextWidget>(17, _omitFieldNames ? '' : 'text',
        subBuilder: TextWidget.create)
    ..aOM<ImageWidget>(18, _omitFieldNames ? '' : 'image',
        subBuilder: ImageWidget.create)
    ..aOM<VideoSurfaceWidget>(19, _omitFieldNames ? '' : 'videoSurface',
        subBuilder: VideoSurfaceWidget.create)
    ..aOM<AudioVisualizerWidget>(20, _omitFieldNames ? '' : 'audioVisualizer',
        subBuilder: AudioVisualizerWidget.create)
    ..aOM<CanvasWidget>(21, _omitFieldNames ? '' : 'canvas',
        subBuilder: CanvasWidget.create)
    ..aOM<TextInputWidget>(22, _omitFieldNames ? '' : 'textInput',
        subBuilder: TextInputWidget.create)
    ..aOM<ButtonWidget>(23, _omitFieldNames ? '' : 'button',
        subBuilder: ButtonWidget.create)
    ..aOM<SliderWidget>(24, _omitFieldNames ? '' : 'slider',
        subBuilder: SliderWidget.create)
    ..aOM<ToggleWidget>(25, _omitFieldNames ? '' : 'toggle',
        subBuilder: ToggleWidget.create)
    ..aOM<DropdownWidget>(26, _omitFieldNames ? '' : 'dropdown',
        subBuilder: DropdownWidget.create)
    ..aOM<GestureAreaWidget>(27, _omitFieldNames ? '' : 'gestureArea',
        subBuilder: GestureAreaWidget.create)
    ..aOM<OverlayWidget>(28, _omitFieldNames ? '' : 'overlay',
        subBuilder: OverlayWidget.create)
    ..aOM<ProgressWidget>(29, _omitFieldNames ? '' : 'progress',
        subBuilder: ProgressWidget.create)
    ..aOM<FullscreenWidget>(30, _omitFieldNames ? '' : 'fullscreen',
        subBuilder: FullscreenWidget.create)
    ..aOM<KeepAwakeWidget>(31, _omitFieldNames ? '' : 'keepAwake',
        subBuilder: KeepAwakeWidget.create)
    ..aOM<BrightnessWidget>(32, _omitFieldNames ? '' : 'brightness',
        subBuilder: BrightnessWidget.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Node clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Node copyWith(void Function(Node) updates) =>
      super.copyWith((message) => updates(message as Node)) as Node;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Node create() => Node._();
  @$core.override
  Node createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static Node getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Node>(create);
  static Node? _defaultInstance;

  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(12)
  @$pb.TagNumber(13)
  @$pb.TagNumber(14)
  @$pb.TagNumber(15)
  @$pb.TagNumber(16)
  @$pb.TagNumber(17)
  @$pb.TagNumber(18)
  @$pb.TagNumber(19)
  @$pb.TagNumber(20)
  @$pb.TagNumber(21)
  @$pb.TagNumber(22)
  @$pb.TagNumber(23)
  @$pb.TagNumber(24)
  @$pb.TagNumber(25)
  @$pb.TagNumber(26)
  @$pb.TagNumber(27)
  @$pb.TagNumber(28)
  @$pb.TagNumber(29)
  @$pb.TagNumber(30)
  @$pb.TagNumber(31)
  @$pb.TagNumber(32)
  Node_Widget whichWidget() => _Node_WidgetByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(12)
  @$pb.TagNumber(13)
  @$pb.TagNumber(14)
  @$pb.TagNumber(15)
  @$pb.TagNumber(16)
  @$pb.TagNumber(17)
  @$pb.TagNumber(18)
  @$pb.TagNumber(19)
  @$pb.TagNumber(20)
  @$pb.TagNumber(21)
  @$pb.TagNumber(22)
  @$pb.TagNumber(23)
  @$pb.TagNumber(24)
  @$pb.TagNumber(25)
  @$pb.TagNumber(26)
  @$pb.TagNumber(27)
  @$pb.TagNumber(28)
  @$pb.TagNumber(29)
  @$pb.TagNumber(30)
  @$pb.TagNumber(31)
  @$pb.TagNumber(32)
  void clearWidget() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $pb.PbMap<$core.String, $core.String> get props => $_getMap(1);

  @$pb.TagNumber(3)
  $pb.PbList<Node> get children => $_getList(2);

  @$pb.TagNumber(10)
  StackWidget get stack => $_getN(3);
  @$pb.TagNumber(10)
  set stack(StackWidget value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasStack() => $_has(3);
  @$pb.TagNumber(10)
  void clearStack() => $_clearField(10);
  @$pb.TagNumber(10)
  StackWidget ensureStack() => $_ensure(3);

  @$pb.TagNumber(11)
  RowWidget get row => $_getN(4);
  @$pb.TagNumber(11)
  set row(RowWidget value) => $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasRow() => $_has(4);
  @$pb.TagNumber(11)
  void clearRow() => $_clearField(11);
  @$pb.TagNumber(11)
  RowWidget ensureRow() => $_ensure(4);

  @$pb.TagNumber(12)
  GridWidget get grid => $_getN(5);
  @$pb.TagNumber(12)
  set grid(GridWidget value) => $_setField(12, value);
  @$pb.TagNumber(12)
  $core.bool hasGrid() => $_has(5);
  @$pb.TagNumber(12)
  void clearGrid() => $_clearField(12);
  @$pb.TagNumber(12)
  GridWidget ensureGrid() => $_ensure(5);

  @$pb.TagNumber(13)
  ScrollWidget get scroll => $_getN(6);
  @$pb.TagNumber(13)
  set scroll(ScrollWidget value) => $_setField(13, value);
  @$pb.TagNumber(13)
  $core.bool hasScroll() => $_has(6);
  @$pb.TagNumber(13)
  void clearScroll() => $_clearField(13);
  @$pb.TagNumber(13)
  ScrollWidget ensureScroll() => $_ensure(6);

  @$pb.TagNumber(14)
  PaddingWidget get padding => $_getN(7);
  @$pb.TagNumber(14)
  set padding(PaddingWidget value) => $_setField(14, value);
  @$pb.TagNumber(14)
  $core.bool hasPadding() => $_has(7);
  @$pb.TagNumber(14)
  void clearPadding() => $_clearField(14);
  @$pb.TagNumber(14)
  PaddingWidget ensurePadding() => $_ensure(7);

  @$pb.TagNumber(15)
  CenterWidget get center => $_getN(8);
  @$pb.TagNumber(15)
  set center(CenterWidget value) => $_setField(15, value);
  @$pb.TagNumber(15)
  $core.bool hasCenter() => $_has(8);
  @$pb.TagNumber(15)
  void clearCenter() => $_clearField(15);
  @$pb.TagNumber(15)
  CenterWidget ensureCenter() => $_ensure(8);

  @$pb.TagNumber(16)
  ExpandWidget get expand => $_getN(9);
  @$pb.TagNumber(16)
  set expand(ExpandWidget value) => $_setField(16, value);
  @$pb.TagNumber(16)
  $core.bool hasExpand() => $_has(9);
  @$pb.TagNumber(16)
  void clearExpand() => $_clearField(16);
  @$pb.TagNumber(16)
  ExpandWidget ensureExpand() => $_ensure(9);

  @$pb.TagNumber(17)
  TextWidget get text => $_getN(10);
  @$pb.TagNumber(17)
  set text(TextWidget value) => $_setField(17, value);
  @$pb.TagNumber(17)
  $core.bool hasText() => $_has(10);
  @$pb.TagNumber(17)
  void clearText() => $_clearField(17);
  @$pb.TagNumber(17)
  TextWidget ensureText() => $_ensure(10);

  @$pb.TagNumber(18)
  ImageWidget get image => $_getN(11);
  @$pb.TagNumber(18)
  set image(ImageWidget value) => $_setField(18, value);
  @$pb.TagNumber(18)
  $core.bool hasImage() => $_has(11);
  @$pb.TagNumber(18)
  void clearImage() => $_clearField(18);
  @$pb.TagNumber(18)
  ImageWidget ensureImage() => $_ensure(11);

  @$pb.TagNumber(19)
  VideoSurfaceWidget get videoSurface => $_getN(12);
  @$pb.TagNumber(19)
  set videoSurface(VideoSurfaceWidget value) => $_setField(19, value);
  @$pb.TagNumber(19)
  $core.bool hasVideoSurface() => $_has(12);
  @$pb.TagNumber(19)
  void clearVideoSurface() => $_clearField(19);
  @$pb.TagNumber(19)
  VideoSurfaceWidget ensureVideoSurface() => $_ensure(12);

  @$pb.TagNumber(20)
  AudioVisualizerWidget get audioVisualizer => $_getN(13);
  @$pb.TagNumber(20)
  set audioVisualizer(AudioVisualizerWidget value) => $_setField(20, value);
  @$pb.TagNumber(20)
  $core.bool hasAudioVisualizer() => $_has(13);
  @$pb.TagNumber(20)
  void clearAudioVisualizer() => $_clearField(20);
  @$pb.TagNumber(20)
  AudioVisualizerWidget ensureAudioVisualizer() => $_ensure(13);

  @$pb.TagNumber(21)
  CanvasWidget get canvas => $_getN(14);
  @$pb.TagNumber(21)
  set canvas(CanvasWidget value) => $_setField(21, value);
  @$pb.TagNumber(21)
  $core.bool hasCanvas() => $_has(14);
  @$pb.TagNumber(21)
  void clearCanvas() => $_clearField(21);
  @$pb.TagNumber(21)
  CanvasWidget ensureCanvas() => $_ensure(14);

  @$pb.TagNumber(22)
  TextInputWidget get textInput => $_getN(15);
  @$pb.TagNumber(22)
  set textInput(TextInputWidget value) => $_setField(22, value);
  @$pb.TagNumber(22)
  $core.bool hasTextInput() => $_has(15);
  @$pb.TagNumber(22)
  void clearTextInput() => $_clearField(22);
  @$pb.TagNumber(22)
  TextInputWidget ensureTextInput() => $_ensure(15);

  @$pb.TagNumber(23)
  ButtonWidget get button => $_getN(16);
  @$pb.TagNumber(23)
  set button(ButtonWidget value) => $_setField(23, value);
  @$pb.TagNumber(23)
  $core.bool hasButton() => $_has(16);
  @$pb.TagNumber(23)
  void clearButton() => $_clearField(23);
  @$pb.TagNumber(23)
  ButtonWidget ensureButton() => $_ensure(16);

  @$pb.TagNumber(24)
  SliderWidget get slider => $_getN(17);
  @$pb.TagNumber(24)
  set slider(SliderWidget value) => $_setField(24, value);
  @$pb.TagNumber(24)
  $core.bool hasSlider() => $_has(17);
  @$pb.TagNumber(24)
  void clearSlider() => $_clearField(24);
  @$pb.TagNumber(24)
  SliderWidget ensureSlider() => $_ensure(17);

  @$pb.TagNumber(25)
  ToggleWidget get toggle => $_getN(18);
  @$pb.TagNumber(25)
  set toggle(ToggleWidget value) => $_setField(25, value);
  @$pb.TagNumber(25)
  $core.bool hasToggle() => $_has(18);
  @$pb.TagNumber(25)
  void clearToggle() => $_clearField(25);
  @$pb.TagNumber(25)
  ToggleWidget ensureToggle() => $_ensure(18);

  @$pb.TagNumber(26)
  DropdownWidget get dropdown => $_getN(19);
  @$pb.TagNumber(26)
  set dropdown(DropdownWidget value) => $_setField(26, value);
  @$pb.TagNumber(26)
  $core.bool hasDropdown() => $_has(19);
  @$pb.TagNumber(26)
  void clearDropdown() => $_clearField(26);
  @$pb.TagNumber(26)
  DropdownWidget ensureDropdown() => $_ensure(19);

  @$pb.TagNumber(27)
  GestureAreaWidget get gestureArea => $_getN(20);
  @$pb.TagNumber(27)
  set gestureArea(GestureAreaWidget value) => $_setField(27, value);
  @$pb.TagNumber(27)
  $core.bool hasGestureArea() => $_has(20);
  @$pb.TagNumber(27)
  void clearGestureArea() => $_clearField(27);
  @$pb.TagNumber(27)
  GestureAreaWidget ensureGestureArea() => $_ensure(20);

  @$pb.TagNumber(28)
  OverlayWidget get overlay => $_getN(21);
  @$pb.TagNumber(28)
  set overlay(OverlayWidget value) => $_setField(28, value);
  @$pb.TagNumber(28)
  $core.bool hasOverlay() => $_has(21);
  @$pb.TagNumber(28)
  void clearOverlay() => $_clearField(28);
  @$pb.TagNumber(28)
  OverlayWidget ensureOverlay() => $_ensure(21);

  @$pb.TagNumber(29)
  ProgressWidget get progress => $_getN(22);
  @$pb.TagNumber(29)
  set progress(ProgressWidget value) => $_setField(29, value);
  @$pb.TagNumber(29)
  $core.bool hasProgress() => $_has(22);
  @$pb.TagNumber(29)
  void clearProgress() => $_clearField(29);
  @$pb.TagNumber(29)
  ProgressWidget ensureProgress() => $_ensure(22);

  @$pb.TagNumber(30)
  FullscreenWidget get fullscreen => $_getN(23);
  @$pb.TagNumber(30)
  set fullscreen(FullscreenWidget value) => $_setField(30, value);
  @$pb.TagNumber(30)
  $core.bool hasFullscreen() => $_has(23);
  @$pb.TagNumber(30)
  void clearFullscreen() => $_clearField(30);
  @$pb.TagNumber(30)
  FullscreenWidget ensureFullscreen() => $_ensure(23);

  @$pb.TagNumber(31)
  KeepAwakeWidget get keepAwake => $_getN(24);
  @$pb.TagNumber(31)
  set keepAwake(KeepAwakeWidget value) => $_setField(31, value);
  @$pb.TagNumber(31)
  $core.bool hasKeepAwake() => $_has(24);
  @$pb.TagNumber(31)
  void clearKeepAwake() => $_clearField(31);
  @$pb.TagNumber(31)
  KeepAwakeWidget ensureKeepAwake() => $_ensure(24);

  @$pb.TagNumber(32)
  BrightnessWidget get brightness => $_getN(25);
  @$pb.TagNumber(32)
  set brightness(BrightnessWidget value) => $_setField(32, value);
  @$pb.TagNumber(32)
  $core.bool hasBrightness() => $_has(25);
  @$pb.TagNumber(32)
  void clearBrightness() => $_clearField(32);
  @$pb.TagNumber(32)
  BrightnessWidget ensureBrightness() => $_ensure(25);
}

class StackWidget extends $pb.GeneratedMessage {
  factory StackWidget() => create();

  StackWidget._();

  factory StackWidget.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StackWidget.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StackWidget',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StackWidget clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StackWidget copyWith(void Function(StackWidget) updates) =>
      super.copyWith((message) => updates(message as StackWidget))
          as StackWidget;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StackWidget create() => StackWidget._();
  @$core.override
  StackWidget createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StackWidget getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StackWidget>(create);
  static StackWidget? _defaultInstance;
}

class RowWidget extends $pb.GeneratedMessage {
  factory RowWidget() => create();

  RowWidget._();

  factory RowWidget.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RowWidget.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RowWidget',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RowWidget clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RowWidget copyWith(void Function(RowWidget) updates) =>
      super.copyWith((message) => updates(message as RowWidget)) as RowWidget;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RowWidget create() => RowWidget._();
  @$core.override
  RowWidget createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RowWidget getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<RowWidget>(create);
  static RowWidget? _defaultInstance;
}

class GridWidget extends $pb.GeneratedMessage {
  factory GridWidget({
    $core.int? columns,
  }) {
    final result = create();
    if (columns != null) result.columns = columns;
    return result;
  }

  GridWidget._();

  factory GridWidget.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GridWidget.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GridWidget',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'columns')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GridWidget clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GridWidget copyWith(void Function(GridWidget) updates) =>
      super.copyWith((message) => updates(message as GridWidget)) as GridWidget;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GridWidget create() => GridWidget._();
  @$core.override
  GridWidget createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GridWidget getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GridWidget>(create);
  static GridWidget? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get columns => $_getIZ(0);
  @$pb.TagNumber(1)
  set columns($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasColumns() => $_has(0);
  @$pb.TagNumber(1)
  void clearColumns() => $_clearField(1);
}

class ScrollWidget extends $pb.GeneratedMessage {
  factory ScrollWidget({
    $core.String? direction,
  }) {
    final result = create();
    if (direction != null) result.direction = direction;
    return result;
  }

  ScrollWidget._();

  factory ScrollWidget.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ScrollWidget.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ScrollWidget',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'direction')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ScrollWidget clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ScrollWidget copyWith(void Function(ScrollWidget) updates) =>
      super.copyWith((message) => updates(message as ScrollWidget))
          as ScrollWidget;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ScrollWidget create() => ScrollWidget._();
  @$core.override
  ScrollWidget createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ScrollWidget getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ScrollWidget>(create);
  static ScrollWidget? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get direction => $_getSZ(0);
  @$pb.TagNumber(1)
  set direction($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDirection() => $_has(0);
  @$pb.TagNumber(1)
  void clearDirection() => $_clearField(1);
}

class PaddingWidget extends $pb.GeneratedMessage {
  factory PaddingWidget({
    $core.int? all,
  }) {
    final result = create();
    if (all != null) result.all = all;
    return result;
  }

  PaddingWidget._();

  factory PaddingWidget.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PaddingWidget.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PaddingWidget',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'all')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PaddingWidget clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PaddingWidget copyWith(void Function(PaddingWidget) updates) =>
      super.copyWith((message) => updates(message as PaddingWidget))
          as PaddingWidget;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PaddingWidget create() => PaddingWidget._();
  @$core.override
  PaddingWidget createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PaddingWidget getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PaddingWidget>(create);
  static PaddingWidget? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get all => $_getIZ(0);
  @$pb.TagNumber(1)
  set all($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAll() => $_has(0);
  @$pb.TagNumber(1)
  void clearAll() => $_clearField(1);
}

class CenterWidget extends $pb.GeneratedMessage {
  factory CenterWidget() => create();

  CenterWidget._();

  factory CenterWidget.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CenterWidget.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CenterWidget',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CenterWidget clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CenterWidget copyWith(void Function(CenterWidget) updates) =>
      super.copyWith((message) => updates(message as CenterWidget))
          as CenterWidget;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CenterWidget create() => CenterWidget._();
  @$core.override
  CenterWidget createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CenterWidget getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CenterWidget>(create);
  static CenterWidget? _defaultInstance;
}

class ExpandWidget extends $pb.GeneratedMessage {
  factory ExpandWidget() => create();

  ExpandWidget._();

  factory ExpandWidget.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ExpandWidget.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ExpandWidget',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ExpandWidget clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ExpandWidget copyWith(void Function(ExpandWidget) updates) =>
      super.copyWith((message) => updates(message as ExpandWidget))
          as ExpandWidget;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ExpandWidget create() => ExpandWidget._();
  @$core.override
  ExpandWidget createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ExpandWidget getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ExpandWidget>(create);
  static ExpandWidget? _defaultInstance;
}

class TextWidget extends $pb.GeneratedMessage {
  factory TextWidget({
    $core.String? value,
    $core.String? style,
    $core.String? color,
  }) {
    final result = create();
    if (value != null) result.value = value;
    if (style != null) result.style = style;
    if (color != null) result.color = color;
    return result;
  }

  TextWidget._();

  factory TextWidget.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TextWidget.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TextWidget',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'value')
    ..aOS(2, _omitFieldNames ? '' : 'style')
    ..aOS(3, _omitFieldNames ? '' : 'color')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TextWidget clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TextWidget copyWith(void Function(TextWidget) updates) =>
      super.copyWith((message) => updates(message as TextWidget)) as TextWidget;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TextWidget create() => TextWidget._();
  @$core.override
  TextWidget createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TextWidget getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TextWidget>(create);
  static TextWidget? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get value => $_getSZ(0);
  @$pb.TagNumber(1)
  set value($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasValue() => $_has(0);
  @$pb.TagNumber(1)
  void clearValue() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get style => $_getSZ(1);
  @$pb.TagNumber(2)
  set style($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasStyle() => $_has(1);
  @$pb.TagNumber(2)
  void clearStyle() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get color => $_getSZ(2);
  @$pb.TagNumber(3)
  set color($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasColor() => $_has(2);
  @$pb.TagNumber(3)
  void clearColor() => $_clearField(3);
}

class ImageWidget extends $pb.GeneratedMessage {
  factory ImageWidget({
    $core.String? url,
  }) {
    final result = create();
    if (url != null) result.url = url;
    return result;
  }

  ImageWidget._();

  factory ImageWidget.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ImageWidget.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ImageWidget',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'url')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ImageWidget clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ImageWidget copyWith(void Function(ImageWidget) updates) =>
      super.copyWith((message) => updates(message as ImageWidget))
          as ImageWidget;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ImageWidget create() => ImageWidget._();
  @$core.override
  ImageWidget createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ImageWidget getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ImageWidget>(create);
  static ImageWidget? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get url => $_getSZ(0);
  @$pb.TagNumber(1)
  set url($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasUrl() => $_has(0);
  @$pb.TagNumber(1)
  void clearUrl() => $_clearField(1);
}

class VideoSurfaceWidget extends $pb.GeneratedMessage {
  factory VideoSurfaceWidget({
    $core.String? trackId,
  }) {
    final result = create();
    if (trackId != null) result.trackId = trackId;
    return result;
  }

  VideoSurfaceWidget._();

  factory VideoSurfaceWidget.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory VideoSurfaceWidget.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'VideoSurfaceWidget',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'trackId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  VideoSurfaceWidget clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  VideoSurfaceWidget copyWith(void Function(VideoSurfaceWidget) updates) =>
      super.copyWith((message) => updates(message as VideoSurfaceWidget))
          as VideoSurfaceWidget;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static VideoSurfaceWidget create() => VideoSurfaceWidget._();
  @$core.override
  VideoSurfaceWidget createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static VideoSurfaceWidget getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<VideoSurfaceWidget>(create);
  static VideoSurfaceWidget? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get trackId => $_getSZ(0);
  @$pb.TagNumber(1)
  set trackId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasTrackId() => $_has(0);
  @$pb.TagNumber(1)
  void clearTrackId() => $_clearField(1);
}

class AudioVisualizerWidget extends $pb.GeneratedMessage {
  factory AudioVisualizerWidget({
    $core.String? streamId,
  }) {
    final result = create();
    if (streamId != null) result.streamId = streamId;
    return result;
  }

  AudioVisualizerWidget._();

  factory AudioVisualizerWidget.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory AudioVisualizerWidget.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AudioVisualizerWidget',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'streamId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AudioVisualizerWidget clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AudioVisualizerWidget copyWith(
          void Function(AudioVisualizerWidget) updates) =>
      super.copyWith((message) => updates(message as AudioVisualizerWidget))
          as AudioVisualizerWidget;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static AudioVisualizerWidget create() => AudioVisualizerWidget._();
  @$core.override
  AudioVisualizerWidget createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static AudioVisualizerWidget getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AudioVisualizerWidget>(create);
  static AudioVisualizerWidget? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get streamId => $_getSZ(0);
  @$pb.TagNumber(1)
  set streamId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasStreamId() => $_has(0);
  @$pb.TagNumber(1)
  void clearStreamId() => $_clearField(1);
}

class CanvasWidget extends $pb.GeneratedMessage {
  factory CanvasWidget({
    $core.String? drawOpsJson,
  }) {
    final result = create();
    if (drawOpsJson != null) result.drawOpsJson = drawOpsJson;
    return result;
  }

  CanvasWidget._();

  factory CanvasWidget.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CanvasWidget.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CanvasWidget',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'drawOpsJson')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CanvasWidget clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CanvasWidget copyWith(void Function(CanvasWidget) updates) =>
      super.copyWith((message) => updates(message as CanvasWidget))
          as CanvasWidget;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CanvasWidget create() => CanvasWidget._();
  @$core.override
  CanvasWidget createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CanvasWidget getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CanvasWidget>(create);
  static CanvasWidget? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get drawOpsJson => $_getSZ(0);
  @$pb.TagNumber(1)
  set drawOpsJson($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDrawOpsJson() => $_has(0);
  @$pb.TagNumber(1)
  void clearDrawOpsJson() => $_clearField(1);
}

class TextInputWidget extends $pb.GeneratedMessage {
  factory TextInputWidget({
    $core.String? placeholder,
    $core.bool? autofocus,
  }) {
    final result = create();
    if (placeholder != null) result.placeholder = placeholder;
    if (autofocus != null) result.autofocus = autofocus;
    return result;
  }

  TextInputWidget._();

  factory TextInputWidget.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TextInputWidget.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TextInputWidget',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'placeholder')
    ..aOB(2, _omitFieldNames ? '' : 'autofocus')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TextInputWidget clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TextInputWidget copyWith(void Function(TextInputWidget) updates) =>
      super.copyWith((message) => updates(message as TextInputWidget))
          as TextInputWidget;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TextInputWidget create() => TextInputWidget._();
  @$core.override
  TextInputWidget createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TextInputWidget getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TextInputWidget>(create);
  static TextInputWidget? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get placeholder => $_getSZ(0);
  @$pb.TagNumber(1)
  set placeholder($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPlaceholder() => $_has(0);
  @$pb.TagNumber(1)
  void clearPlaceholder() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get autofocus => $_getBF(1);
  @$pb.TagNumber(2)
  set autofocus($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAutofocus() => $_has(1);
  @$pb.TagNumber(2)
  void clearAutofocus() => $_clearField(2);
}

class ButtonWidget extends $pb.GeneratedMessage {
  factory ButtonWidget({
    $core.String? label,
    $core.String? action,
  }) {
    final result = create();
    if (label != null) result.label = label;
    if (action != null) result.action = action;
    return result;
  }

  ButtonWidget._();

  factory ButtonWidget.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ButtonWidget.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ButtonWidget',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'label')
    ..aOS(2, _omitFieldNames ? '' : 'action')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ButtonWidget clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ButtonWidget copyWith(void Function(ButtonWidget) updates) =>
      super.copyWith((message) => updates(message as ButtonWidget))
          as ButtonWidget;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ButtonWidget create() => ButtonWidget._();
  @$core.override
  ButtonWidget createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ButtonWidget getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ButtonWidget>(create);
  static ButtonWidget? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get label => $_getSZ(0);
  @$pb.TagNumber(1)
  set label($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasLabel() => $_has(0);
  @$pb.TagNumber(1)
  void clearLabel() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get action => $_getSZ(1);
  @$pb.TagNumber(2)
  set action($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAction() => $_has(1);
  @$pb.TagNumber(2)
  void clearAction() => $_clearField(2);
}

class SliderWidget extends $pb.GeneratedMessage {
  factory SliderWidget({
    $core.double? min,
    $core.double? max,
    $core.double? value,
  }) {
    final result = create();
    if (min != null) result.min = min;
    if (max != null) result.max = max;
    if (value != null) result.value = value;
    return result;
  }

  SliderWidget._();

  factory SliderWidget.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SliderWidget.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SliderWidget',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..aD(1, _omitFieldNames ? '' : 'min')
    ..aD(2, _omitFieldNames ? '' : 'max')
    ..aD(3, _omitFieldNames ? '' : 'value')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SliderWidget clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SliderWidget copyWith(void Function(SliderWidget) updates) =>
      super.copyWith((message) => updates(message as SliderWidget))
          as SliderWidget;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SliderWidget create() => SliderWidget._();
  @$core.override
  SliderWidget createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SliderWidget getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SliderWidget>(create);
  static SliderWidget? _defaultInstance;

  @$pb.TagNumber(1)
  $core.double get min => $_getN(0);
  @$pb.TagNumber(1)
  set min($core.double value) => $_setDouble(0, value);
  @$pb.TagNumber(1)
  $core.bool hasMin() => $_has(0);
  @$pb.TagNumber(1)
  void clearMin() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.double get max => $_getN(1);
  @$pb.TagNumber(2)
  set max($core.double value) => $_setDouble(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMax() => $_has(1);
  @$pb.TagNumber(2)
  void clearMax() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.double get value => $_getN(2);
  @$pb.TagNumber(3)
  set value($core.double value) => $_setDouble(2, value);
  @$pb.TagNumber(3)
  $core.bool hasValue() => $_has(2);
  @$pb.TagNumber(3)
  void clearValue() => $_clearField(3);
}

class ToggleWidget extends $pb.GeneratedMessage {
  factory ToggleWidget({
    $core.bool? value,
  }) {
    final result = create();
    if (value != null) result.value = value;
    return result;
  }

  ToggleWidget._();

  factory ToggleWidget.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ToggleWidget.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ToggleWidget',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'value')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ToggleWidget clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ToggleWidget copyWith(void Function(ToggleWidget) updates) =>
      super.copyWith((message) => updates(message as ToggleWidget))
          as ToggleWidget;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ToggleWidget create() => ToggleWidget._();
  @$core.override
  ToggleWidget createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ToggleWidget getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ToggleWidget>(create);
  static ToggleWidget? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get value => $_getBF(0);
  @$pb.TagNumber(1)
  set value($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasValue() => $_has(0);
  @$pb.TagNumber(1)
  void clearValue() => $_clearField(1);
}

class DropdownWidget extends $pb.GeneratedMessage {
  factory DropdownWidget({
    $core.Iterable<$core.String>? options,
    $core.String? value,
  }) {
    final result = create();
    if (options != null) result.options.addAll(options);
    if (value != null) result.value = value;
    return result;
  }

  DropdownWidget._();

  factory DropdownWidget.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DropdownWidget.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DropdownWidget',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..pPS(1, _omitFieldNames ? '' : 'options')
    ..aOS(2, _omitFieldNames ? '' : 'value')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DropdownWidget clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DropdownWidget copyWith(void Function(DropdownWidget) updates) =>
      super.copyWith((message) => updates(message as DropdownWidget))
          as DropdownWidget;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DropdownWidget create() => DropdownWidget._();
  @$core.override
  DropdownWidget createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DropdownWidget getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DropdownWidget>(create);
  static DropdownWidget? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<$core.String> get options => $_getList(0);

  @$pb.TagNumber(2)
  $core.String get value => $_getSZ(1);
  @$pb.TagNumber(2)
  set value($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasValue() => $_has(1);
  @$pb.TagNumber(2)
  void clearValue() => $_clearField(2);
}

class GestureAreaWidget extends $pb.GeneratedMessage {
  factory GestureAreaWidget({
    $core.String? action,
  }) {
    final result = create();
    if (action != null) result.action = action;
    return result;
  }

  GestureAreaWidget._();

  factory GestureAreaWidget.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory GestureAreaWidget.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'GestureAreaWidget',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'action')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GestureAreaWidget clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  GestureAreaWidget copyWith(void Function(GestureAreaWidget) updates) =>
      super.copyWith((message) => updates(message as GestureAreaWidget))
          as GestureAreaWidget;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static GestureAreaWidget create() => GestureAreaWidget._();
  @$core.override
  GestureAreaWidget createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static GestureAreaWidget getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<GestureAreaWidget>(create);
  static GestureAreaWidget? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get action => $_getSZ(0);
  @$pb.TagNumber(1)
  set action($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAction() => $_has(0);
  @$pb.TagNumber(1)
  void clearAction() => $_clearField(1);
}

class OverlayWidget extends $pb.GeneratedMessage {
  factory OverlayWidget() => create();

  OverlayWidget._();

  factory OverlayWidget.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory OverlayWidget.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'OverlayWidget',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OverlayWidget clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  OverlayWidget copyWith(void Function(OverlayWidget) updates) =>
      super.copyWith((message) => updates(message as OverlayWidget))
          as OverlayWidget;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static OverlayWidget create() => OverlayWidget._();
  @$core.override
  OverlayWidget createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static OverlayWidget getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<OverlayWidget>(create);
  static OverlayWidget? _defaultInstance;
}

class ProgressWidget extends $pb.GeneratedMessage {
  factory ProgressWidget({
    $core.double? value,
  }) {
    final result = create();
    if (value != null) result.value = value;
    return result;
  }

  ProgressWidget._();

  factory ProgressWidget.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ProgressWidget.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ProgressWidget',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..aD(1, _omitFieldNames ? '' : 'value')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ProgressWidget clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ProgressWidget copyWith(void Function(ProgressWidget) updates) =>
      super.copyWith((message) => updates(message as ProgressWidget))
          as ProgressWidget;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ProgressWidget create() => ProgressWidget._();
  @$core.override
  ProgressWidget createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ProgressWidget getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ProgressWidget>(create);
  static ProgressWidget? _defaultInstance;

  @$pb.TagNumber(1)
  $core.double get value => $_getN(0);
  @$pb.TagNumber(1)
  set value($core.double value) => $_setDouble(0, value);
  @$pb.TagNumber(1)
  $core.bool hasValue() => $_has(0);
  @$pb.TagNumber(1)
  void clearValue() => $_clearField(1);
}

class FullscreenWidget extends $pb.GeneratedMessage {
  factory FullscreenWidget({
    $core.bool? enabled,
  }) {
    final result = create();
    if (enabled != null) result.enabled = enabled;
    return result;
  }

  FullscreenWidget._();

  factory FullscreenWidget.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory FullscreenWidget.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FullscreenWidget',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'enabled')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FullscreenWidget clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FullscreenWidget copyWith(void Function(FullscreenWidget) updates) =>
      super.copyWith((message) => updates(message as FullscreenWidget))
          as FullscreenWidget;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static FullscreenWidget create() => FullscreenWidget._();
  @$core.override
  FullscreenWidget createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static FullscreenWidget getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<FullscreenWidget>(create);
  static FullscreenWidget? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get enabled => $_getBF(0);
  @$pb.TagNumber(1)
  set enabled($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEnabled() => $_has(0);
  @$pb.TagNumber(1)
  void clearEnabled() => $_clearField(1);
}

class KeepAwakeWidget extends $pb.GeneratedMessage {
  factory KeepAwakeWidget({
    $core.bool? enabled,
  }) {
    final result = create();
    if (enabled != null) result.enabled = enabled;
    return result;
  }

  KeepAwakeWidget._();

  factory KeepAwakeWidget.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory KeepAwakeWidget.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'KeepAwakeWidget',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'enabled')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  KeepAwakeWidget clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  KeepAwakeWidget copyWith(void Function(KeepAwakeWidget) updates) =>
      super.copyWith((message) => updates(message as KeepAwakeWidget))
          as KeepAwakeWidget;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static KeepAwakeWidget create() => KeepAwakeWidget._();
  @$core.override
  KeepAwakeWidget createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static KeepAwakeWidget getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<KeepAwakeWidget>(create);
  static KeepAwakeWidget? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get enabled => $_getBF(0);
  @$pb.TagNumber(1)
  set enabled($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEnabled() => $_has(0);
  @$pb.TagNumber(1)
  void clearEnabled() => $_clearField(1);
}

class BrightnessWidget extends $pb.GeneratedMessage {
  factory BrightnessWidget({
    $core.double? value,
  }) {
    final result = create();
    if (value != null) result.value = value;
    return result;
  }

  BrightnessWidget._();

  factory BrightnessWidget.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory BrightnessWidget.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'BrightnessWidget',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.ui.v1'),
      createEmptyInstance: create)
    ..aD(1, _omitFieldNames ? '' : 'value')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BrightnessWidget clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BrightnessWidget copyWith(void Function(BrightnessWidget) updates) =>
      super.copyWith((message) => updates(message as BrightnessWidget))
          as BrightnessWidget;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static BrightnessWidget create() => BrightnessWidget._();
  @$core.override
  BrightnessWidget createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static BrightnessWidget getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<BrightnessWidget>(create);
  static BrightnessWidget? _defaultInstance;

  @$pb.TagNumber(1)
  $core.double get value => $_getN(0);
  @$pb.TagNumber(1)
  set value($core.double value) => $_setDouble(0, value);
  @$pb.TagNumber(1)
  $core.bool hasValue() => $_has(0);
  @$pb.TagNumber(1)
  void clearValue() => $_clearField(1);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
