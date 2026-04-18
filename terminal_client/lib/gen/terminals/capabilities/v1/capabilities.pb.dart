// This is a generated file - do not edit.
//
// Generated from terminals/capabilities/v1/capabilities.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

class DeviceCapabilities extends $pb.GeneratedMessage {
  factory DeviceCapabilities({
    $core.String? deviceId,
    DeviceIdentity? identity,
    ScreenCapability? screen,
    KeyboardCapability? keyboard,
    PointerCapability? pointer,
    TouchCapability? touch,
    AudioOutputCapability? speakers,
    AudioInputCapability? microphone,
    CameraCapability? camera,
    SensorCapability? sensors,
    ConnectivityCapability? connectivity,
    BatteryCapability? battery,
    EdgeCapability? edge,
    $core.Iterable<DisplayCapability>? displays,
    HapticCapability? haptics,
  }) {
    final result = create();
    if (deviceId != null) result.deviceId = deviceId;
    if (identity != null) result.identity = identity;
    if (screen != null) result.screen = screen;
    if (keyboard != null) result.keyboard = keyboard;
    if (pointer != null) result.pointer = pointer;
    if (touch != null) result.touch = touch;
    if (speakers != null) result.speakers = speakers;
    if (microphone != null) result.microphone = microphone;
    if (camera != null) result.camera = camera;
    if (sensors != null) result.sensors = sensors;
    if (connectivity != null) result.connectivity = connectivity;
    if (battery != null) result.battery = battery;
    if (edge != null) result.edge = edge;
    if (displays != null) result.displays.addAll(displays);
    if (haptics != null) result.haptics = haptics;
    return result;
  }

  DeviceCapabilities._();

  factory DeviceCapabilities.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DeviceCapabilities.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DeviceCapabilities',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.capabilities.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'deviceId')
    ..aOM<DeviceIdentity>(2, _omitFieldNames ? '' : 'identity',
        subBuilder: DeviceIdentity.create)
    ..aOM<ScreenCapability>(10, _omitFieldNames ? '' : 'screen',
        subBuilder: ScreenCapability.create)
    ..aOM<KeyboardCapability>(11, _omitFieldNames ? '' : 'keyboard',
        subBuilder: KeyboardCapability.create)
    ..aOM<PointerCapability>(12, _omitFieldNames ? '' : 'pointer',
        subBuilder: PointerCapability.create)
    ..aOM<TouchCapability>(13, _omitFieldNames ? '' : 'touch',
        subBuilder: TouchCapability.create)
    ..aOM<AudioOutputCapability>(14, _omitFieldNames ? '' : 'speakers',
        subBuilder: AudioOutputCapability.create)
    ..aOM<AudioInputCapability>(15, _omitFieldNames ? '' : 'microphone',
        subBuilder: AudioInputCapability.create)
    ..aOM<CameraCapability>(16, _omitFieldNames ? '' : 'camera',
        subBuilder: CameraCapability.create)
    ..aOM<SensorCapability>(17, _omitFieldNames ? '' : 'sensors',
        subBuilder: SensorCapability.create)
    ..aOM<ConnectivityCapability>(18, _omitFieldNames ? '' : 'connectivity',
        subBuilder: ConnectivityCapability.create)
    ..aOM<BatteryCapability>(19, _omitFieldNames ? '' : 'battery',
        subBuilder: BatteryCapability.create)
    ..aOM<EdgeCapability>(20, _omitFieldNames ? '' : 'edge',
        subBuilder: EdgeCapability.create)
    ..pPM<DisplayCapability>(21, _omitFieldNames ? '' : 'displays',
        subBuilder: DisplayCapability.create)
    ..aOM<HapticCapability>(22, _omitFieldNames ? '' : 'haptics',
        subBuilder: HapticCapability.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeviceCapabilities clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeviceCapabilities copyWith(void Function(DeviceCapabilities) updates) =>
      super.copyWith((message) => updates(message as DeviceCapabilities))
          as DeviceCapabilities;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DeviceCapabilities create() => DeviceCapabilities._();
  @$core.override
  DeviceCapabilities createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DeviceCapabilities getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DeviceCapabilities>(create);
  static DeviceCapabilities? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get deviceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set deviceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDeviceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDeviceId() => $_clearField(1);

  @$pb.TagNumber(2)
  DeviceIdentity get identity => $_getN(1);
  @$pb.TagNumber(2)
  set identity(DeviceIdentity value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasIdentity() => $_has(1);
  @$pb.TagNumber(2)
  void clearIdentity() => $_clearField(2);
  @$pb.TagNumber(2)
  DeviceIdentity ensureIdentity() => $_ensure(1);

  @$pb.TagNumber(10)
  ScreenCapability get screen => $_getN(2);
  @$pb.TagNumber(10)
  set screen(ScreenCapability value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasScreen() => $_has(2);
  @$pb.TagNumber(10)
  void clearScreen() => $_clearField(10);
  @$pb.TagNumber(10)
  ScreenCapability ensureScreen() => $_ensure(2);

  @$pb.TagNumber(11)
  KeyboardCapability get keyboard => $_getN(3);
  @$pb.TagNumber(11)
  set keyboard(KeyboardCapability value) => $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasKeyboard() => $_has(3);
  @$pb.TagNumber(11)
  void clearKeyboard() => $_clearField(11);
  @$pb.TagNumber(11)
  KeyboardCapability ensureKeyboard() => $_ensure(3);

  @$pb.TagNumber(12)
  PointerCapability get pointer => $_getN(4);
  @$pb.TagNumber(12)
  set pointer(PointerCapability value) => $_setField(12, value);
  @$pb.TagNumber(12)
  $core.bool hasPointer() => $_has(4);
  @$pb.TagNumber(12)
  void clearPointer() => $_clearField(12);
  @$pb.TagNumber(12)
  PointerCapability ensurePointer() => $_ensure(4);

  @$pb.TagNumber(13)
  TouchCapability get touch => $_getN(5);
  @$pb.TagNumber(13)
  set touch(TouchCapability value) => $_setField(13, value);
  @$pb.TagNumber(13)
  $core.bool hasTouch() => $_has(5);
  @$pb.TagNumber(13)
  void clearTouch() => $_clearField(13);
  @$pb.TagNumber(13)
  TouchCapability ensureTouch() => $_ensure(5);

  @$pb.TagNumber(14)
  AudioOutputCapability get speakers => $_getN(6);
  @$pb.TagNumber(14)
  set speakers(AudioOutputCapability value) => $_setField(14, value);
  @$pb.TagNumber(14)
  $core.bool hasSpeakers() => $_has(6);
  @$pb.TagNumber(14)
  void clearSpeakers() => $_clearField(14);
  @$pb.TagNumber(14)
  AudioOutputCapability ensureSpeakers() => $_ensure(6);

  @$pb.TagNumber(15)
  AudioInputCapability get microphone => $_getN(7);
  @$pb.TagNumber(15)
  set microphone(AudioInputCapability value) => $_setField(15, value);
  @$pb.TagNumber(15)
  $core.bool hasMicrophone() => $_has(7);
  @$pb.TagNumber(15)
  void clearMicrophone() => $_clearField(15);
  @$pb.TagNumber(15)
  AudioInputCapability ensureMicrophone() => $_ensure(7);

  @$pb.TagNumber(16)
  CameraCapability get camera => $_getN(8);
  @$pb.TagNumber(16)
  set camera(CameraCapability value) => $_setField(16, value);
  @$pb.TagNumber(16)
  $core.bool hasCamera() => $_has(8);
  @$pb.TagNumber(16)
  void clearCamera() => $_clearField(16);
  @$pb.TagNumber(16)
  CameraCapability ensureCamera() => $_ensure(8);

  @$pb.TagNumber(17)
  SensorCapability get sensors => $_getN(9);
  @$pb.TagNumber(17)
  set sensors(SensorCapability value) => $_setField(17, value);
  @$pb.TagNumber(17)
  $core.bool hasSensors() => $_has(9);
  @$pb.TagNumber(17)
  void clearSensors() => $_clearField(17);
  @$pb.TagNumber(17)
  SensorCapability ensureSensors() => $_ensure(9);

  @$pb.TagNumber(18)
  ConnectivityCapability get connectivity => $_getN(10);
  @$pb.TagNumber(18)
  set connectivity(ConnectivityCapability value) => $_setField(18, value);
  @$pb.TagNumber(18)
  $core.bool hasConnectivity() => $_has(10);
  @$pb.TagNumber(18)
  void clearConnectivity() => $_clearField(18);
  @$pb.TagNumber(18)
  ConnectivityCapability ensureConnectivity() => $_ensure(10);

  @$pb.TagNumber(19)
  BatteryCapability get battery => $_getN(11);
  @$pb.TagNumber(19)
  set battery(BatteryCapability value) => $_setField(19, value);
  @$pb.TagNumber(19)
  $core.bool hasBattery() => $_has(11);
  @$pb.TagNumber(19)
  void clearBattery() => $_clearField(19);
  @$pb.TagNumber(19)
  BatteryCapability ensureBattery() => $_ensure(11);

  @$pb.TagNumber(20)
  EdgeCapability get edge => $_getN(12);
  @$pb.TagNumber(20)
  set edge(EdgeCapability value) => $_setField(20, value);
  @$pb.TagNumber(20)
  $core.bool hasEdge() => $_has(12);
  @$pb.TagNumber(20)
  void clearEdge() => $_clearField(20);
  @$pb.TagNumber(20)
  EdgeCapability ensureEdge() => $_ensure(12);

  @$pb.TagNumber(21)
  $pb.PbList<DisplayCapability> get displays => $_getList(13);

  @$pb.TagNumber(22)
  HapticCapability get haptics => $_getN(14);
  @$pb.TagNumber(22)
  set haptics(HapticCapability value) => $_setField(22, value);
  @$pb.TagNumber(22)
  $core.bool hasHaptics() => $_has(14);
  @$pb.TagNumber(22)
  void clearHaptics() => $_clearField(22);
  @$pb.TagNumber(22)
  HapticCapability ensureHaptics() => $_ensure(14);
}

class DeviceIdentity extends $pb.GeneratedMessage {
  factory DeviceIdentity({
    $core.String? deviceName,
    $core.String? deviceType,
    $core.String? platform,
  }) {
    final result = create();
    if (deviceName != null) result.deviceName = deviceName;
    if (deviceType != null) result.deviceType = deviceType;
    if (platform != null) result.platform = platform;
    return result;
  }

  DeviceIdentity._();

  factory DeviceIdentity.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DeviceIdentity.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DeviceIdentity',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.capabilities.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'deviceName')
    ..aOS(2, _omitFieldNames ? '' : 'deviceType')
    ..aOS(3, _omitFieldNames ? '' : 'platform')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeviceIdentity clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeviceIdentity copyWith(void Function(DeviceIdentity) updates) =>
      super.copyWith((message) => updates(message as DeviceIdentity))
          as DeviceIdentity;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DeviceIdentity create() => DeviceIdentity._();
  @$core.override
  DeviceIdentity createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DeviceIdentity getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DeviceIdentity>(create);
  static DeviceIdentity? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get deviceName => $_getSZ(0);
  @$pb.TagNumber(1)
  set deviceName($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDeviceName() => $_has(0);
  @$pb.TagNumber(1)
  void clearDeviceName() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get deviceType => $_getSZ(1);
  @$pb.TagNumber(2)
  set deviceType($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDeviceType() => $_has(1);
  @$pb.TagNumber(2)
  void clearDeviceType() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get platform => $_getSZ(2);
  @$pb.TagNumber(3)
  set platform($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasPlatform() => $_has(2);
  @$pb.TagNumber(3)
  void clearPlatform() => $_clearField(3);
}

class ScreenCapability extends $pb.GeneratedMessage {
  factory ScreenCapability({
    $core.int? width,
    $core.int? height,
    $core.double? density,
    $core.bool? touch,
    $core.String? orientation,
    $core.bool? fullscreenSupported,
    $core.bool? multiWindowSupported,
    Insets? safeArea,
  }) {
    final result = create();
    if (width != null) result.width = width;
    if (height != null) result.height = height;
    if (density != null) result.density = density;
    if (touch != null) result.touch = touch;
    if (orientation != null) result.orientation = orientation;
    if (fullscreenSupported != null)
      result.fullscreenSupported = fullscreenSupported;
    if (multiWindowSupported != null)
      result.multiWindowSupported = multiWindowSupported;
    if (safeArea != null) result.safeArea = safeArea;
    return result;
  }

  ScreenCapability._();

  factory ScreenCapability.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ScreenCapability.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ScreenCapability',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.capabilities.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'width')
    ..aI(2, _omitFieldNames ? '' : 'height')
    ..aD(3, _omitFieldNames ? '' : 'density')
    ..aOB(4, _omitFieldNames ? '' : 'touch')
    ..aOS(5, _omitFieldNames ? '' : 'orientation')
    ..aOB(6, _omitFieldNames ? '' : 'fullscreenSupported')
    ..aOB(7, _omitFieldNames ? '' : 'multiWindowSupported')
    ..aOM<Insets>(8, _omitFieldNames ? '' : 'safeArea',
        subBuilder: Insets.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ScreenCapability clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ScreenCapability copyWith(void Function(ScreenCapability) updates) =>
      super.copyWith((message) => updates(message as ScreenCapability))
          as ScreenCapability;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ScreenCapability create() => ScreenCapability._();
  @$core.override
  ScreenCapability createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ScreenCapability getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ScreenCapability>(create);
  static ScreenCapability? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get width => $_getIZ(0);
  @$pb.TagNumber(1)
  set width($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasWidth() => $_has(0);
  @$pb.TagNumber(1)
  void clearWidth() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get height => $_getIZ(1);
  @$pb.TagNumber(2)
  set height($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasHeight() => $_has(1);
  @$pb.TagNumber(2)
  void clearHeight() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.double get density => $_getN(2);
  @$pb.TagNumber(3)
  set density($core.double value) => $_setDouble(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDensity() => $_has(2);
  @$pb.TagNumber(3)
  void clearDensity() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get touch => $_getBF(3);
  @$pb.TagNumber(4)
  set touch($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasTouch() => $_has(3);
  @$pb.TagNumber(4)
  void clearTouch() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get orientation => $_getSZ(4);
  @$pb.TagNumber(5)
  set orientation($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasOrientation() => $_has(4);
  @$pb.TagNumber(5)
  void clearOrientation() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.bool get fullscreenSupported => $_getBF(5);
  @$pb.TagNumber(6)
  set fullscreenSupported($core.bool value) => $_setBool(5, value);
  @$pb.TagNumber(6)
  $core.bool hasFullscreenSupported() => $_has(5);
  @$pb.TagNumber(6)
  void clearFullscreenSupported() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.bool get multiWindowSupported => $_getBF(6);
  @$pb.TagNumber(7)
  set multiWindowSupported($core.bool value) => $_setBool(6, value);
  @$pb.TagNumber(7)
  $core.bool hasMultiWindowSupported() => $_has(6);
  @$pb.TagNumber(7)
  void clearMultiWindowSupported() => $_clearField(7);

  @$pb.TagNumber(8)
  Insets get safeArea => $_getN(7);
  @$pb.TagNumber(8)
  set safeArea(Insets value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasSafeArea() => $_has(7);
  @$pb.TagNumber(8)
  void clearSafeArea() => $_clearField(8);
  @$pb.TagNumber(8)
  Insets ensureSafeArea() => $_ensure(7);
}

class Insets extends $pb.GeneratedMessage {
  factory Insets({
    $core.int? left,
    $core.int? top,
    $core.int? right,
    $core.int? bottom,
  }) {
    final result = create();
    if (left != null) result.left = left;
    if (top != null) result.top = top;
    if (right != null) result.right = right;
    if (bottom != null) result.bottom = bottom;
    return result;
  }

  Insets._();

  factory Insets.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Insets.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Insets',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.capabilities.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'left')
    ..aI(2, _omitFieldNames ? '' : 'top')
    ..aI(3, _omitFieldNames ? '' : 'right')
    ..aI(4, _omitFieldNames ? '' : 'bottom')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Insets clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Insets copyWith(void Function(Insets) updates) =>
      super.copyWith((message) => updates(message as Insets)) as Insets;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Insets create() => Insets._();
  @$core.override
  Insets createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static Insets getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Insets>(create);
  static Insets? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get left => $_getIZ(0);
  @$pb.TagNumber(1)
  set left($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasLeft() => $_has(0);
  @$pb.TagNumber(1)
  void clearLeft() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get top => $_getIZ(1);
  @$pb.TagNumber(2)
  set top($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTop() => $_has(1);
  @$pb.TagNumber(2)
  void clearTop() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get right => $_getIZ(2);
  @$pb.TagNumber(3)
  set right($core.int value) => $_setSignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasRight() => $_has(2);
  @$pb.TagNumber(3)
  void clearRight() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get bottom => $_getIZ(3);
  @$pb.TagNumber(4)
  set bottom($core.int value) => $_setSignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasBottom() => $_has(3);
  @$pb.TagNumber(4)
  void clearBottom() => $_clearField(4);
}

class DisplayCapability extends $pb.GeneratedMessage {
  factory DisplayCapability({
    $core.String? displayId,
    $core.String? displayName,
    $core.bool? primary,
    ScreenCapability? screen,
  }) {
    final result = create();
    if (displayId != null) result.displayId = displayId;
    if (displayName != null) result.displayName = displayName;
    if (primary != null) result.primary = primary;
    if (screen != null) result.screen = screen;
    return result;
  }

  DisplayCapability._();

  factory DisplayCapability.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DisplayCapability.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DisplayCapability',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.capabilities.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'displayId')
    ..aOS(2, _omitFieldNames ? '' : 'displayName')
    ..aOB(3, _omitFieldNames ? '' : 'primary')
    ..aOM<ScreenCapability>(4, _omitFieldNames ? '' : 'screen',
        subBuilder: ScreenCapability.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DisplayCapability clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DisplayCapability copyWith(void Function(DisplayCapability) updates) =>
      super.copyWith((message) => updates(message as DisplayCapability))
          as DisplayCapability;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DisplayCapability create() => DisplayCapability._();
  @$core.override
  DisplayCapability createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DisplayCapability getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<DisplayCapability>(create);
  static DisplayCapability? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get displayId => $_getSZ(0);
  @$pb.TagNumber(1)
  set displayId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDisplayId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDisplayId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get displayName => $_getSZ(1);
  @$pb.TagNumber(2)
  set displayName($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDisplayName() => $_has(1);
  @$pb.TagNumber(2)
  void clearDisplayName() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get primary => $_getBF(2);
  @$pb.TagNumber(3)
  set primary($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasPrimary() => $_has(2);
  @$pb.TagNumber(3)
  void clearPrimary() => $_clearField(3);

  @$pb.TagNumber(4)
  ScreenCapability get screen => $_getN(3);
  @$pb.TagNumber(4)
  set screen(ScreenCapability value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasScreen() => $_has(3);
  @$pb.TagNumber(4)
  void clearScreen() => $_clearField(4);
  @$pb.TagNumber(4)
  ScreenCapability ensureScreen() => $_ensure(3);
}

class KeyboardCapability extends $pb.GeneratedMessage {
  factory KeyboardCapability({
    $core.bool? physical,
    $core.String? layout,
  }) {
    final result = create();
    if (physical != null) result.physical = physical;
    if (layout != null) result.layout = layout;
    return result;
  }

  KeyboardCapability._();

  factory KeyboardCapability.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory KeyboardCapability.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'KeyboardCapability',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.capabilities.v1'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'physical')
    ..aOS(2, _omitFieldNames ? '' : 'layout')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  KeyboardCapability clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  KeyboardCapability copyWith(void Function(KeyboardCapability) updates) =>
      super.copyWith((message) => updates(message as KeyboardCapability))
          as KeyboardCapability;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static KeyboardCapability create() => KeyboardCapability._();
  @$core.override
  KeyboardCapability createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static KeyboardCapability getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<KeyboardCapability>(create);
  static KeyboardCapability? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get physical => $_getBF(0);
  @$pb.TagNumber(1)
  set physical($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPhysical() => $_has(0);
  @$pb.TagNumber(1)
  void clearPhysical() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get layout => $_getSZ(1);
  @$pb.TagNumber(2)
  set layout($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasLayout() => $_has(1);
  @$pb.TagNumber(2)
  void clearLayout() => $_clearField(2);
}

class PointerCapability extends $pb.GeneratedMessage {
  factory PointerCapability({
    $core.String? type,
    $core.bool? hover,
  }) {
    final result = create();
    if (type != null) result.type = type;
    if (hover != null) result.hover = hover;
    return result;
  }

  PointerCapability._();

  factory PointerCapability.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PointerCapability.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PointerCapability',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.capabilities.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'type')
    ..aOB(2, _omitFieldNames ? '' : 'hover')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PointerCapability clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PointerCapability copyWith(void Function(PointerCapability) updates) =>
      super.copyWith((message) => updates(message as PointerCapability))
          as PointerCapability;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PointerCapability create() => PointerCapability._();
  @$core.override
  PointerCapability createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PointerCapability getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PointerCapability>(create);
  static PointerCapability? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get type => $_getSZ(0);
  @$pb.TagNumber(1)
  set type($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasType() => $_has(0);
  @$pb.TagNumber(1)
  void clearType() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get hover => $_getBF(1);
  @$pb.TagNumber(2)
  set hover($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasHover() => $_has(1);
  @$pb.TagNumber(2)
  void clearHover() => $_clearField(2);
}

class TouchCapability extends $pb.GeneratedMessage {
  factory TouchCapability({
    $core.bool? supported,
    $core.int? maxPoints,
  }) {
    final result = create();
    if (supported != null) result.supported = supported;
    if (maxPoints != null) result.maxPoints = maxPoints;
    return result;
  }

  TouchCapability._();

  factory TouchCapability.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TouchCapability.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TouchCapability',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.capabilities.v1'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'supported')
    ..aI(2, _omitFieldNames ? '' : 'maxPoints')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TouchCapability clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TouchCapability copyWith(void Function(TouchCapability) updates) =>
      super.copyWith((message) => updates(message as TouchCapability))
          as TouchCapability;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TouchCapability create() => TouchCapability._();
  @$core.override
  TouchCapability createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TouchCapability getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TouchCapability>(create);
  static TouchCapability? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get supported => $_getBF(0);
  @$pb.TagNumber(1)
  set supported($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSupported() => $_has(0);
  @$pb.TagNumber(1)
  void clearSupported() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get maxPoints => $_getIZ(1);
  @$pb.TagNumber(2)
  set maxPoints($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMaxPoints() => $_has(1);
  @$pb.TagNumber(2)
  void clearMaxPoints() => $_clearField(2);
}

class AudioOutputCapability extends $pb.GeneratedMessage {
  factory AudioOutputCapability({
    $core.int? channels,
    $core.Iterable<$core.int>? sampleRates,
    $core.Iterable<AudioEndpoint>? endpoints,
  }) {
    final result = create();
    if (channels != null) result.channels = channels;
    if (sampleRates != null) result.sampleRates.addAll(sampleRates);
    if (endpoints != null) result.endpoints.addAll(endpoints);
    return result;
  }

  AudioOutputCapability._();

  factory AudioOutputCapability.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory AudioOutputCapability.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AudioOutputCapability',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.capabilities.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'channels')
    ..p<$core.int>(2, _omitFieldNames ? '' : 'sampleRates', $pb.PbFieldType.K3)
    ..pPM<AudioEndpoint>(3, _omitFieldNames ? '' : 'endpoints',
        subBuilder: AudioEndpoint.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AudioOutputCapability clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AudioOutputCapability copyWith(
          void Function(AudioOutputCapability) updates) =>
      super.copyWith((message) => updates(message as AudioOutputCapability))
          as AudioOutputCapability;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static AudioOutputCapability create() => AudioOutputCapability._();
  @$core.override
  AudioOutputCapability createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static AudioOutputCapability getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AudioOutputCapability>(create);
  static AudioOutputCapability? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get channels => $_getIZ(0);
  @$pb.TagNumber(1)
  set channels($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasChannels() => $_has(0);
  @$pb.TagNumber(1)
  void clearChannels() => $_clearField(1);

  @$pb.TagNumber(2)
  $pb.PbList<$core.int> get sampleRates => $_getList(1);

  @$pb.TagNumber(3)
  $pb.PbList<AudioEndpoint> get endpoints => $_getList(2);
}

class AudioInputCapability extends $pb.GeneratedMessage {
  factory AudioInputCapability({
    $core.int? channels,
    $core.Iterable<$core.int>? sampleRates,
    $core.Iterable<AudioEndpoint>? endpoints,
  }) {
    final result = create();
    if (channels != null) result.channels = channels;
    if (sampleRates != null) result.sampleRates.addAll(sampleRates);
    if (endpoints != null) result.endpoints.addAll(endpoints);
    return result;
  }

  AudioInputCapability._();

  factory AudioInputCapability.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory AudioInputCapability.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AudioInputCapability',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.capabilities.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'channels')
    ..p<$core.int>(2, _omitFieldNames ? '' : 'sampleRates', $pb.PbFieldType.K3)
    ..pPM<AudioEndpoint>(3, _omitFieldNames ? '' : 'endpoints',
        subBuilder: AudioEndpoint.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AudioInputCapability clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AudioInputCapability copyWith(void Function(AudioInputCapability) updates) =>
      super.copyWith((message) => updates(message as AudioInputCapability))
          as AudioInputCapability;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static AudioInputCapability create() => AudioInputCapability._();
  @$core.override
  AudioInputCapability createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static AudioInputCapability getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AudioInputCapability>(create);
  static AudioInputCapability? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get channels => $_getIZ(0);
  @$pb.TagNumber(1)
  set channels($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasChannels() => $_has(0);
  @$pb.TagNumber(1)
  void clearChannels() => $_clearField(1);

  @$pb.TagNumber(2)
  $pb.PbList<$core.int> get sampleRates => $_getList(1);

  @$pb.TagNumber(3)
  $pb.PbList<AudioEndpoint> get endpoints => $_getList(2);
}

class AudioEndpoint extends $pb.GeneratedMessage {
  factory AudioEndpoint({
    $core.String? endpointId,
    $core.String? endpointName,
    $core.String? connectionType,
    $core.int? channels,
    $core.Iterable<$core.int>? sampleRates,
    $core.bool? available,
  }) {
    final result = create();
    if (endpointId != null) result.endpointId = endpointId;
    if (endpointName != null) result.endpointName = endpointName;
    if (connectionType != null) result.connectionType = connectionType;
    if (channels != null) result.channels = channels;
    if (sampleRates != null) result.sampleRates.addAll(sampleRates);
    if (available != null) result.available = available;
    return result;
  }

  AudioEndpoint._();

  factory AudioEndpoint.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory AudioEndpoint.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'AudioEndpoint',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.capabilities.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'endpointId')
    ..aOS(2, _omitFieldNames ? '' : 'endpointName')
    ..aOS(3, _omitFieldNames ? '' : 'connectionType')
    ..aI(4, _omitFieldNames ? '' : 'channels')
    ..p<$core.int>(5, _omitFieldNames ? '' : 'sampleRates', $pb.PbFieldType.K3)
    ..aOB(6, _omitFieldNames ? '' : 'available')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AudioEndpoint clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  AudioEndpoint copyWith(void Function(AudioEndpoint) updates) =>
      super.copyWith((message) => updates(message as AudioEndpoint))
          as AudioEndpoint;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static AudioEndpoint create() => AudioEndpoint._();
  @$core.override
  AudioEndpoint createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static AudioEndpoint getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<AudioEndpoint>(create);
  static AudioEndpoint? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get endpointId => $_getSZ(0);
  @$pb.TagNumber(1)
  set endpointId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEndpointId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEndpointId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get endpointName => $_getSZ(1);
  @$pb.TagNumber(2)
  set endpointName($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEndpointName() => $_has(1);
  @$pb.TagNumber(2)
  void clearEndpointName() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get connectionType => $_getSZ(2);
  @$pb.TagNumber(3)
  set connectionType($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasConnectionType() => $_has(2);
  @$pb.TagNumber(3)
  void clearConnectionType() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get channels => $_getIZ(3);
  @$pb.TagNumber(4)
  set channels($core.int value) => $_setSignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasChannels() => $_has(3);
  @$pb.TagNumber(4)
  void clearChannels() => $_clearField(4);

  @$pb.TagNumber(5)
  $pb.PbList<$core.int> get sampleRates => $_getList(4);

  @$pb.TagNumber(6)
  $core.bool get available => $_getBF(5);
  @$pb.TagNumber(6)
  set available($core.bool value) => $_setBool(5, value);
  @$pb.TagNumber(6)
  $core.bool hasAvailable() => $_has(5);
  @$pb.TagNumber(6)
  void clearAvailable() => $_clearField(6);
}

class CameraLens extends $pb.GeneratedMessage {
  factory CameraLens({
    $core.int? width,
    $core.int? height,
    $core.int? fps,
  }) {
    final result = create();
    if (width != null) result.width = width;
    if (height != null) result.height = height;
    if (fps != null) result.fps = fps;
    return result;
  }

  CameraLens._();

  factory CameraLens.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CameraLens.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CameraLens',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.capabilities.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'width')
    ..aI(2, _omitFieldNames ? '' : 'height')
    ..aI(3, _omitFieldNames ? '' : 'fps')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CameraLens clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CameraLens copyWith(void Function(CameraLens) updates) =>
      super.copyWith((message) => updates(message as CameraLens)) as CameraLens;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CameraLens create() => CameraLens._();
  @$core.override
  CameraLens createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CameraLens getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CameraLens>(create);
  static CameraLens? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get width => $_getIZ(0);
  @$pb.TagNumber(1)
  set width($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasWidth() => $_has(0);
  @$pb.TagNumber(1)
  void clearWidth() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get height => $_getIZ(1);
  @$pb.TagNumber(2)
  set height($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasHeight() => $_has(1);
  @$pb.TagNumber(2)
  void clearHeight() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get fps => $_getIZ(2);
  @$pb.TagNumber(3)
  set fps($core.int value) => $_setSignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasFps() => $_has(2);
  @$pb.TagNumber(3)
  void clearFps() => $_clearField(3);
}

class CameraCapability extends $pb.GeneratedMessage {
  factory CameraCapability({
    CameraLens? front,
    CameraLens? back,
    $core.Iterable<CameraEndpoint>? endpoints,
  }) {
    final result = create();
    if (front != null) result.front = front;
    if (back != null) result.back = back;
    if (endpoints != null) result.endpoints.addAll(endpoints);
    return result;
  }

  CameraCapability._();

  factory CameraCapability.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CameraCapability.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CameraCapability',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.capabilities.v1'),
      createEmptyInstance: create)
    ..aOM<CameraLens>(1, _omitFieldNames ? '' : 'front',
        subBuilder: CameraLens.create)
    ..aOM<CameraLens>(2, _omitFieldNames ? '' : 'back',
        subBuilder: CameraLens.create)
    ..pPM<CameraEndpoint>(3, _omitFieldNames ? '' : 'endpoints',
        subBuilder: CameraEndpoint.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CameraCapability clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CameraCapability copyWith(void Function(CameraCapability) updates) =>
      super.copyWith((message) => updates(message as CameraCapability))
          as CameraCapability;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CameraCapability create() => CameraCapability._();
  @$core.override
  CameraCapability createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CameraCapability getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CameraCapability>(create);
  static CameraCapability? _defaultInstance;

  @$pb.TagNumber(1)
  CameraLens get front => $_getN(0);
  @$pb.TagNumber(1)
  set front(CameraLens value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasFront() => $_has(0);
  @$pb.TagNumber(1)
  void clearFront() => $_clearField(1);
  @$pb.TagNumber(1)
  CameraLens ensureFront() => $_ensure(0);

  @$pb.TagNumber(2)
  CameraLens get back => $_getN(1);
  @$pb.TagNumber(2)
  set back(CameraLens value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasBack() => $_has(1);
  @$pb.TagNumber(2)
  void clearBack() => $_clearField(2);
  @$pb.TagNumber(2)
  CameraLens ensureBack() => $_ensure(1);

  @$pb.TagNumber(3)
  $pb.PbList<CameraEndpoint> get endpoints => $_getList(2);
}

class CameraEndpoint extends $pb.GeneratedMessage {
  factory CameraEndpoint({
    $core.String? endpointId,
    $core.String? endpointName,
    $core.String? connectionType,
    $core.String? facing,
    $core.Iterable<CameraLens>? modes,
    $core.bool? available,
  }) {
    final result = create();
    if (endpointId != null) result.endpointId = endpointId;
    if (endpointName != null) result.endpointName = endpointName;
    if (connectionType != null) result.connectionType = connectionType;
    if (facing != null) result.facing = facing;
    if (modes != null) result.modes.addAll(modes);
    if (available != null) result.available = available;
    return result;
  }

  CameraEndpoint._();

  factory CameraEndpoint.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CameraEndpoint.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CameraEndpoint',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.capabilities.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'endpointId')
    ..aOS(2, _omitFieldNames ? '' : 'endpointName')
    ..aOS(3, _omitFieldNames ? '' : 'connectionType')
    ..aOS(4, _omitFieldNames ? '' : 'facing')
    ..pPM<CameraLens>(5, _omitFieldNames ? '' : 'modes',
        subBuilder: CameraLens.create)
    ..aOB(6, _omitFieldNames ? '' : 'available')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CameraEndpoint clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CameraEndpoint copyWith(void Function(CameraEndpoint) updates) =>
      super.copyWith((message) => updates(message as CameraEndpoint))
          as CameraEndpoint;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CameraEndpoint create() => CameraEndpoint._();
  @$core.override
  CameraEndpoint createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CameraEndpoint getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CameraEndpoint>(create);
  static CameraEndpoint? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get endpointId => $_getSZ(0);
  @$pb.TagNumber(1)
  set endpointId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasEndpointId() => $_has(0);
  @$pb.TagNumber(1)
  void clearEndpointId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get endpointName => $_getSZ(1);
  @$pb.TagNumber(2)
  set endpointName($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasEndpointName() => $_has(1);
  @$pb.TagNumber(2)
  void clearEndpointName() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get connectionType => $_getSZ(2);
  @$pb.TagNumber(3)
  set connectionType($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasConnectionType() => $_has(2);
  @$pb.TagNumber(3)
  void clearConnectionType() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get facing => $_getSZ(3);
  @$pb.TagNumber(4)
  set facing($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasFacing() => $_has(3);
  @$pb.TagNumber(4)
  void clearFacing() => $_clearField(4);

  @$pb.TagNumber(5)
  $pb.PbList<CameraLens> get modes => $_getList(4);

  @$pb.TagNumber(6)
  $core.bool get available => $_getBF(5);
  @$pb.TagNumber(6)
  set available($core.bool value) => $_setBool(5, value);
  @$pb.TagNumber(6)
  $core.bool hasAvailable() => $_has(5);
  @$pb.TagNumber(6)
  void clearAvailable() => $_clearField(6);
}

class SensorCapability extends $pb.GeneratedMessage {
  factory SensorCapability({
    $core.bool? accelerometer,
    $core.bool? gyroscope,
    $core.bool? compass,
    $core.bool? ambientLight,
    $core.bool? proximity,
    $core.bool? gps,
  }) {
    final result = create();
    if (accelerometer != null) result.accelerometer = accelerometer;
    if (gyroscope != null) result.gyroscope = gyroscope;
    if (compass != null) result.compass = compass;
    if (ambientLight != null) result.ambientLight = ambientLight;
    if (proximity != null) result.proximity = proximity;
    if (gps != null) result.gps = gps;
    return result;
  }

  SensorCapability._();

  factory SensorCapability.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SensorCapability.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SensorCapability',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.capabilities.v1'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'accelerometer')
    ..aOB(2, _omitFieldNames ? '' : 'gyroscope')
    ..aOB(3, _omitFieldNames ? '' : 'compass')
    ..aOB(4, _omitFieldNames ? '' : 'ambientLight')
    ..aOB(5, _omitFieldNames ? '' : 'proximity')
    ..aOB(6, _omitFieldNames ? '' : 'gps')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SensorCapability clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SensorCapability copyWith(void Function(SensorCapability) updates) =>
      super.copyWith((message) => updates(message as SensorCapability))
          as SensorCapability;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SensorCapability create() => SensorCapability._();
  @$core.override
  SensorCapability createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SensorCapability getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SensorCapability>(create);
  static SensorCapability? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get accelerometer => $_getBF(0);
  @$pb.TagNumber(1)
  set accelerometer($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAccelerometer() => $_has(0);
  @$pb.TagNumber(1)
  void clearAccelerometer() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get gyroscope => $_getBF(1);
  @$pb.TagNumber(2)
  set gyroscope($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasGyroscope() => $_has(1);
  @$pb.TagNumber(2)
  void clearGyroscope() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get compass => $_getBF(2);
  @$pb.TagNumber(3)
  set compass($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasCompass() => $_has(2);
  @$pb.TagNumber(3)
  void clearCompass() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get ambientLight => $_getBF(3);
  @$pb.TagNumber(4)
  set ambientLight($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasAmbientLight() => $_has(3);
  @$pb.TagNumber(4)
  void clearAmbientLight() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.bool get proximity => $_getBF(4);
  @$pb.TagNumber(5)
  set proximity($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasProximity() => $_has(4);
  @$pb.TagNumber(5)
  void clearProximity() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.bool get gps => $_getBF(5);
  @$pb.TagNumber(6)
  set gps($core.bool value) => $_setBool(5, value);
  @$pb.TagNumber(6)
  $core.bool hasGps() => $_has(5);
  @$pb.TagNumber(6)
  void clearGps() => $_clearField(6);
}

class ConnectivityCapability extends $pb.GeneratedMessage {
  factory ConnectivityCapability({
    $core.String? bluetoothVersion,
    $core.bool? wifiSignalStrength,
    $core.bool? usbHost,
    $core.int? usbPorts,
    $core.bool? nfc,
  }) {
    final result = create();
    if (bluetoothVersion != null) result.bluetoothVersion = bluetoothVersion;
    if (wifiSignalStrength != null)
      result.wifiSignalStrength = wifiSignalStrength;
    if (usbHost != null) result.usbHost = usbHost;
    if (usbPorts != null) result.usbPorts = usbPorts;
    if (nfc != null) result.nfc = nfc;
    return result;
  }

  ConnectivityCapability._();

  factory ConnectivityCapability.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ConnectivityCapability.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ConnectivityCapability',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.capabilities.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'bluetoothVersion')
    ..aOB(2, _omitFieldNames ? '' : 'wifiSignalStrength')
    ..aOB(3, _omitFieldNames ? '' : 'usbHost')
    ..aI(4, _omitFieldNames ? '' : 'usbPorts')
    ..aOB(5, _omitFieldNames ? '' : 'nfc')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectivityCapability clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectivityCapability copyWith(
          void Function(ConnectivityCapability) updates) =>
      super.copyWith((message) => updates(message as ConnectivityCapability))
          as ConnectivityCapability;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ConnectivityCapability create() => ConnectivityCapability._();
  @$core.override
  ConnectivityCapability createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ConnectivityCapability getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ConnectivityCapability>(create);
  static ConnectivityCapability? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get bluetoothVersion => $_getSZ(0);
  @$pb.TagNumber(1)
  set bluetoothVersion($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasBluetoothVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearBluetoothVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get wifiSignalStrength => $_getBF(1);
  @$pb.TagNumber(2)
  set wifiSignalStrength($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasWifiSignalStrength() => $_has(1);
  @$pb.TagNumber(2)
  void clearWifiSignalStrength() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get usbHost => $_getBF(2);
  @$pb.TagNumber(3)
  set usbHost($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasUsbHost() => $_has(2);
  @$pb.TagNumber(3)
  void clearUsbHost() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get usbPorts => $_getIZ(3);
  @$pb.TagNumber(4)
  set usbPorts($core.int value) => $_setSignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasUsbPorts() => $_has(3);
  @$pb.TagNumber(4)
  void clearUsbPorts() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.bool get nfc => $_getBF(4);
  @$pb.TagNumber(5)
  set nfc($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasNfc() => $_has(4);
  @$pb.TagNumber(5)
  void clearNfc() => $_clearField(5);
}

class BatteryCapability extends $pb.GeneratedMessage {
  factory BatteryCapability({
    $core.double? level,
    $core.bool? charging,
  }) {
    final result = create();
    if (level != null) result.level = level;
    if (charging != null) result.charging = charging;
    return result;
  }

  BatteryCapability._();

  factory BatteryCapability.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory BatteryCapability.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'BatteryCapability',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.capabilities.v1'),
      createEmptyInstance: create)
    ..aD(1, _omitFieldNames ? '' : 'level', fieldType: $pb.PbFieldType.OF)
    ..aOB(2, _omitFieldNames ? '' : 'charging')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BatteryCapability clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BatteryCapability copyWith(void Function(BatteryCapability) updates) =>
      super.copyWith((message) => updates(message as BatteryCapability))
          as BatteryCapability;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static BatteryCapability create() => BatteryCapability._();
  @$core.override
  BatteryCapability createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static BatteryCapability getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<BatteryCapability>(create);
  static BatteryCapability? _defaultInstance;

  @$pb.TagNumber(1)
  $core.double get level => $_getN(0);
  @$pb.TagNumber(1)
  set level($core.double value) => $_setFloat(0, value);
  @$pb.TagNumber(1)
  $core.bool hasLevel() => $_has(0);
  @$pb.TagNumber(1)
  void clearLevel() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get charging => $_getBF(1);
  @$pb.TagNumber(2)
  set charging($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCharging() => $_has(1);
  @$pb.TagNumber(2)
  void clearCharging() => $_clearField(2);
}

class HapticCapability extends $pb.GeneratedMessage {
  factory HapticCapability({
    $core.bool? supported,
    $core.bool? vibration,
    $core.bool? hapticsEngine,
  }) {
    final result = create();
    if (supported != null) result.supported = supported;
    if (vibration != null) result.vibration = vibration;
    if (hapticsEngine != null) result.hapticsEngine = hapticsEngine;
    return result;
  }

  HapticCapability._();

  factory HapticCapability.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory HapticCapability.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'HapticCapability',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.capabilities.v1'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'supported')
    ..aOB(2, _omitFieldNames ? '' : 'vibration')
    ..aOB(3, _omitFieldNames ? '' : 'hapticsEngine')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HapticCapability clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HapticCapability copyWith(void Function(HapticCapability) updates) =>
      super.copyWith((message) => updates(message as HapticCapability))
          as HapticCapability;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static HapticCapability create() => HapticCapability._();
  @$core.override
  HapticCapability createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static HapticCapability getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<HapticCapability>(create);
  static HapticCapability? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get supported => $_getBF(0);
  @$pb.TagNumber(1)
  set supported($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSupported() => $_has(0);
  @$pb.TagNumber(1)
  void clearSupported() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get vibration => $_getBF(1);
  @$pb.TagNumber(2)
  set vibration($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasVibration() => $_has(1);
  @$pb.TagNumber(2)
  void clearVibration() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get hapticsEngine => $_getBF(2);
  @$pb.TagNumber(3)
  set hapticsEngine($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasHapticsEngine() => $_has(2);
  @$pb.TagNumber(3)
  void clearHapticsEngine() => $_clearField(3);
}

class EdgeCapability extends $pb.GeneratedMessage {
  factory EdgeCapability({
    $core.Iterable<$core.String>? runtimes,
    EdgeComputeCapability? compute,
    $core.Iterable<$core.String>? operators,
    EdgeRetentionCapability? retention,
    EdgeTimingCapability? timing,
    EdgeGeometryCapability? geometry,
  }) {
    final result = create();
    if (runtimes != null) result.runtimes.addAll(runtimes);
    if (compute != null) result.compute = compute;
    if (operators != null) result.operators.addAll(operators);
    if (retention != null) result.retention = retention;
    if (timing != null) result.timing = timing;
    if (geometry != null) result.geometry = geometry;
    return result;
  }

  EdgeCapability._();

  factory EdgeCapability.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EdgeCapability.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EdgeCapability',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.capabilities.v1'),
      createEmptyInstance: create)
    ..pPS(1, _omitFieldNames ? '' : 'runtimes')
    ..aOM<EdgeComputeCapability>(2, _omitFieldNames ? '' : 'compute',
        subBuilder: EdgeComputeCapability.create)
    ..pPS(3, _omitFieldNames ? '' : 'operators')
    ..aOM<EdgeRetentionCapability>(4, _omitFieldNames ? '' : 'retention',
        subBuilder: EdgeRetentionCapability.create)
    ..aOM<EdgeTimingCapability>(5, _omitFieldNames ? '' : 'timing',
        subBuilder: EdgeTimingCapability.create)
    ..aOM<EdgeGeometryCapability>(6, _omitFieldNames ? '' : 'geometry',
        subBuilder: EdgeGeometryCapability.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeCapability clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeCapability copyWith(void Function(EdgeCapability) updates) =>
      super.copyWith((message) => updates(message as EdgeCapability))
          as EdgeCapability;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EdgeCapability create() => EdgeCapability._();
  @$core.override
  EdgeCapability createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EdgeCapability getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EdgeCapability>(create);
  static EdgeCapability? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<$core.String> get runtimes => $_getList(0);

  @$pb.TagNumber(2)
  EdgeComputeCapability get compute => $_getN(1);
  @$pb.TagNumber(2)
  set compute(EdgeComputeCapability value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasCompute() => $_has(1);
  @$pb.TagNumber(2)
  void clearCompute() => $_clearField(2);
  @$pb.TagNumber(2)
  EdgeComputeCapability ensureCompute() => $_ensure(1);

  @$pb.TagNumber(3)
  $pb.PbList<$core.String> get operators => $_getList(2);

  @$pb.TagNumber(4)
  EdgeRetentionCapability get retention => $_getN(3);
  @$pb.TagNumber(4)
  set retention(EdgeRetentionCapability value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasRetention() => $_has(3);
  @$pb.TagNumber(4)
  void clearRetention() => $_clearField(4);
  @$pb.TagNumber(4)
  EdgeRetentionCapability ensureRetention() => $_ensure(3);

  @$pb.TagNumber(5)
  EdgeTimingCapability get timing => $_getN(4);
  @$pb.TagNumber(5)
  set timing(EdgeTimingCapability value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasTiming() => $_has(4);
  @$pb.TagNumber(5)
  void clearTiming() => $_clearField(5);
  @$pb.TagNumber(5)
  EdgeTimingCapability ensureTiming() => $_ensure(4);

  @$pb.TagNumber(6)
  EdgeGeometryCapability get geometry => $_getN(5);
  @$pb.TagNumber(6)
  set geometry(EdgeGeometryCapability value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasGeometry() => $_has(5);
  @$pb.TagNumber(6)
  void clearGeometry() => $_clearField(6);
  @$pb.TagNumber(6)
  EdgeGeometryCapability ensureGeometry() => $_ensure(5);
}

class EdgeComputeCapability extends $pb.GeneratedMessage {
  factory EdgeComputeCapability({
    $core.int? cpuRealtime,
    $core.int? gpuRealtime,
    $core.int? npuRealtime,
    $core.int? memMb,
  }) {
    final result = create();
    if (cpuRealtime != null) result.cpuRealtime = cpuRealtime;
    if (gpuRealtime != null) result.gpuRealtime = gpuRealtime;
    if (npuRealtime != null) result.npuRealtime = npuRealtime;
    if (memMb != null) result.memMb = memMb;
    return result;
  }

  EdgeComputeCapability._();

  factory EdgeComputeCapability.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EdgeComputeCapability.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EdgeComputeCapability',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.capabilities.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'cpuRealtime')
    ..aI(2, _omitFieldNames ? '' : 'gpuRealtime')
    ..aI(3, _omitFieldNames ? '' : 'npuRealtime')
    ..aI(4, _omitFieldNames ? '' : 'memMb')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeComputeCapability clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeComputeCapability copyWith(
          void Function(EdgeComputeCapability) updates) =>
      super.copyWith((message) => updates(message as EdgeComputeCapability))
          as EdgeComputeCapability;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EdgeComputeCapability create() => EdgeComputeCapability._();
  @$core.override
  EdgeComputeCapability createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EdgeComputeCapability getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EdgeComputeCapability>(create);
  static EdgeComputeCapability? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get cpuRealtime => $_getIZ(0);
  @$pb.TagNumber(1)
  set cpuRealtime($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCpuRealtime() => $_has(0);
  @$pb.TagNumber(1)
  void clearCpuRealtime() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get gpuRealtime => $_getIZ(1);
  @$pb.TagNumber(2)
  set gpuRealtime($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasGpuRealtime() => $_has(1);
  @$pb.TagNumber(2)
  void clearGpuRealtime() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get npuRealtime => $_getIZ(2);
  @$pb.TagNumber(3)
  set npuRealtime($core.int value) => $_setSignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasNpuRealtime() => $_has(2);
  @$pb.TagNumber(3)
  void clearNpuRealtime() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get memMb => $_getIZ(3);
  @$pb.TagNumber(4)
  set memMb($core.int value) => $_setSignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasMemMb() => $_has(3);
  @$pb.TagNumber(4)
  void clearMemMb() => $_clearField(4);
}

class EdgeRetentionCapability extends $pb.GeneratedMessage {
  factory EdgeRetentionCapability({
    $core.int? audioSec,
    $core.int? videoSec,
    $core.int? sensorSec,
    $core.int? radioSec,
  }) {
    final result = create();
    if (audioSec != null) result.audioSec = audioSec;
    if (videoSec != null) result.videoSec = videoSec;
    if (sensorSec != null) result.sensorSec = sensorSec;
    if (radioSec != null) result.radioSec = radioSec;
    return result;
  }

  EdgeRetentionCapability._();

  factory EdgeRetentionCapability.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EdgeRetentionCapability.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EdgeRetentionCapability',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.capabilities.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'audioSec')
    ..aI(2, _omitFieldNames ? '' : 'videoSec')
    ..aI(3, _omitFieldNames ? '' : 'sensorSec')
    ..aI(4, _omitFieldNames ? '' : 'radioSec')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeRetentionCapability clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeRetentionCapability copyWith(
          void Function(EdgeRetentionCapability) updates) =>
      super.copyWith((message) => updates(message as EdgeRetentionCapability))
          as EdgeRetentionCapability;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EdgeRetentionCapability create() => EdgeRetentionCapability._();
  @$core.override
  EdgeRetentionCapability createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EdgeRetentionCapability getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EdgeRetentionCapability>(create);
  static EdgeRetentionCapability? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get audioSec => $_getIZ(0);
  @$pb.TagNumber(1)
  set audioSec($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAudioSec() => $_has(0);
  @$pb.TagNumber(1)
  void clearAudioSec() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get videoSec => $_getIZ(1);
  @$pb.TagNumber(2)
  set videoSec($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasVideoSec() => $_has(1);
  @$pb.TagNumber(2)
  void clearVideoSec() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get sensorSec => $_getIZ(2);
  @$pb.TagNumber(3)
  set sensorSec($core.int value) => $_setSignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSensorSec() => $_has(2);
  @$pb.TagNumber(3)
  void clearSensorSec() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get radioSec => $_getIZ(3);
  @$pb.TagNumber(4)
  set radioSec($core.int value) => $_setSignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasRadioSec() => $_has(3);
  @$pb.TagNumber(4)
  void clearRadioSec() => $_clearField(4);
}

class EdgeTimingCapability extends $pb.GeneratedMessage {
  factory EdgeTimingCapability({
    $core.double? syncErrorMs,
  }) {
    final result = create();
    if (syncErrorMs != null) result.syncErrorMs = syncErrorMs;
    return result;
  }

  EdgeTimingCapability._();

  factory EdgeTimingCapability.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EdgeTimingCapability.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EdgeTimingCapability',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.capabilities.v1'),
      createEmptyInstance: create)
    ..aD(1, _omitFieldNames ? '' : 'syncErrorMs')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeTimingCapability clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeTimingCapability copyWith(void Function(EdgeTimingCapability) updates) =>
      super.copyWith((message) => updates(message as EdgeTimingCapability))
          as EdgeTimingCapability;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EdgeTimingCapability create() => EdgeTimingCapability._();
  @$core.override
  EdgeTimingCapability createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EdgeTimingCapability getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EdgeTimingCapability>(create);
  static EdgeTimingCapability? _defaultInstance;

  @$pb.TagNumber(1)
  $core.double get syncErrorMs => $_getN(0);
  @$pb.TagNumber(1)
  set syncErrorMs($core.double value) => $_setDouble(0, value);
  @$pb.TagNumber(1)
  $core.bool hasSyncErrorMs() => $_has(0);
  @$pb.TagNumber(1)
  void clearSyncErrorMs() => $_clearField(1);
}

class EdgeGeometryCapability extends $pb.GeneratedMessage {
  factory EdgeGeometryCapability({
    $core.bool? micArray,
    $core.bool? cameraIntrinsics,
    $core.bool? compass,
  }) {
    final result = create();
    if (micArray != null) result.micArray = micArray;
    if (cameraIntrinsics != null) result.cameraIntrinsics = cameraIntrinsics;
    if (compass != null) result.compass = compass;
    return result;
  }

  EdgeGeometryCapability._();

  factory EdgeGeometryCapability.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EdgeGeometryCapability.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EdgeGeometryCapability',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.capabilities.v1'),
      createEmptyInstance: create)
    ..aOB(1, _omitFieldNames ? '' : 'micArray')
    ..aOB(2, _omitFieldNames ? '' : 'cameraIntrinsics')
    ..aOB(3, _omitFieldNames ? '' : 'compass')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeGeometryCapability clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgeGeometryCapability copyWith(
          void Function(EdgeGeometryCapability) updates) =>
      super.copyWith((message) => updates(message as EdgeGeometryCapability))
          as EdgeGeometryCapability;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EdgeGeometryCapability create() => EdgeGeometryCapability._();
  @$core.override
  EdgeGeometryCapability createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EdgeGeometryCapability getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EdgeGeometryCapability>(create);
  static EdgeGeometryCapability? _defaultInstance;

  @$pb.TagNumber(1)
  $core.bool get micArray => $_getBF(0);
  @$pb.TagNumber(1)
  set micArray($core.bool value) => $_setBool(0, value);
  @$pb.TagNumber(1)
  $core.bool hasMicArray() => $_has(0);
  @$pb.TagNumber(1)
  void clearMicArray() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get cameraIntrinsics => $_getBF(1);
  @$pb.TagNumber(2)
  set cameraIntrinsics($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCameraIntrinsics() => $_has(1);
  @$pb.TagNumber(2)
  void clearCameraIntrinsics() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get compass => $_getBF(2);
  @$pb.TagNumber(3)
  set compass($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasCompass() => $_has(2);
  @$pb.TagNumber(3)
  void clearCompass() => $_clearField(3);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
