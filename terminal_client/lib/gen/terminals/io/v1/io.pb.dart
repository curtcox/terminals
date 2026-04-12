// This is a generated file - do not edit.
//
// Generated from terminals/io/v1/io.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

class StartStream extends $pb.GeneratedMessage {
  factory StartStream({
    $core.String? streamId,
    $core.String? kind,
    $core.String? sourceDeviceId,
    $core.String? targetDeviceId,
    $core.Iterable<$core.MapEntry<$core.String, $core.String>>? metadata,
  }) {
    final result = create();
    if (streamId != null) result.streamId = streamId;
    if (kind != null) result.kind = kind;
    if (sourceDeviceId != null) result.sourceDeviceId = sourceDeviceId;
    if (targetDeviceId != null) result.targetDeviceId = targetDeviceId;
    if (metadata != null) result.metadata.addEntries(metadata);
    return result;
  }

  StartStream._();

  factory StartStream.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StartStream.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StartStream',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'streamId')
    ..aOS(2, _omitFieldNames ? '' : 'kind')
    ..aOS(3, _omitFieldNames ? '' : 'sourceDeviceId')
    ..aOS(4, _omitFieldNames ? '' : 'targetDeviceId')
    ..m<$core.String, $core.String>(5, _omitFieldNames ? '' : 'metadata',
        entryClassName: 'StartStream.MetadataEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OS,
        packageName: const $pb.PackageName('terminals.io.v1'))
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StartStream clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StartStream copyWith(void Function(StartStream) updates) =>
      super.copyWith((message) => updates(message as StartStream))
          as StartStream;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StartStream create() => StartStream._();
  @$core.override
  StartStream createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StartStream getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StartStream>(create);
  static StartStream? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get streamId => $_getSZ(0);
  @$pb.TagNumber(1)
  set streamId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasStreamId() => $_has(0);
  @$pb.TagNumber(1)
  void clearStreamId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get kind => $_getSZ(1);
  @$pb.TagNumber(2)
  set kind($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasKind() => $_has(1);
  @$pb.TagNumber(2)
  void clearKind() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get sourceDeviceId => $_getSZ(2);
  @$pb.TagNumber(3)
  set sourceDeviceId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSourceDeviceId() => $_has(2);
  @$pb.TagNumber(3)
  void clearSourceDeviceId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get targetDeviceId => $_getSZ(3);
  @$pb.TagNumber(4)
  set targetDeviceId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasTargetDeviceId() => $_has(3);
  @$pb.TagNumber(4)
  void clearTargetDeviceId() => $_clearField(4);

  @$pb.TagNumber(5)
  $pb.PbMap<$core.String, $core.String> get metadata => $_getMap(4);
}

class StopStream extends $pb.GeneratedMessage {
  factory StopStream({
    $core.String? streamId,
  }) {
    final result = create();
    if (streamId != null) result.streamId = streamId;
    return result;
  }

  StopStream._();

  factory StopStream.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StopStream.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StopStream',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'streamId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StopStream clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StopStream copyWith(void Function(StopStream) updates) =>
      super.copyWith((message) => updates(message as StopStream)) as StopStream;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StopStream create() => StopStream._();
  @$core.override
  StopStream createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StopStream getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StopStream>(create);
  static StopStream? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get streamId => $_getSZ(0);
  @$pb.TagNumber(1)
  set streamId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasStreamId() => $_has(0);
  @$pb.TagNumber(1)
  void clearStreamId() => $_clearField(1);
}

class RouteStream extends $pb.GeneratedMessage {
  factory RouteStream({
    $core.String? streamId,
    $core.String? sourceDeviceId,
    $core.String? targetDeviceId,
    $core.String? kind,
  }) {
    final result = create();
    if (streamId != null) result.streamId = streamId;
    if (sourceDeviceId != null) result.sourceDeviceId = sourceDeviceId;
    if (targetDeviceId != null) result.targetDeviceId = targetDeviceId;
    if (kind != null) result.kind = kind;
    return result;
  }

  RouteStream._();

  factory RouteStream.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RouteStream.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RouteStream',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'streamId')
    ..aOS(2, _omitFieldNames ? '' : 'sourceDeviceId')
    ..aOS(3, _omitFieldNames ? '' : 'targetDeviceId')
    ..aOS(4, _omitFieldNames ? '' : 'kind')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RouteStream clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RouteStream copyWith(void Function(RouteStream) updates) =>
      super.copyWith((message) => updates(message as RouteStream))
          as RouteStream;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RouteStream create() => RouteStream._();
  @$core.override
  RouteStream createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RouteStream getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RouteStream>(create);
  static RouteStream? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get streamId => $_getSZ(0);
  @$pb.TagNumber(1)
  set streamId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasStreamId() => $_has(0);
  @$pb.TagNumber(1)
  void clearStreamId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get sourceDeviceId => $_getSZ(1);
  @$pb.TagNumber(2)
  set sourceDeviceId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSourceDeviceId() => $_has(1);
  @$pb.TagNumber(2)
  void clearSourceDeviceId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get targetDeviceId => $_getSZ(2);
  @$pb.TagNumber(3)
  set targetDeviceId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasTargetDeviceId() => $_has(2);
  @$pb.TagNumber(3)
  void clearTargetDeviceId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get kind => $_getSZ(3);
  @$pb.TagNumber(4)
  set kind($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasKind() => $_has(3);
  @$pb.TagNumber(4)
  void clearKind() => $_clearField(4);
}

enum PlayAudio_Source { url, pcmData, ttsText, notSet }

class PlayAudio extends $pb.GeneratedMessage {
  factory PlayAudio({
    $core.String? requestId,
    $core.String? deviceId,
    $core.String? url,
    $core.List<$core.int>? pcmData,
    $core.String? ttsText,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (deviceId != null) result.deviceId = deviceId;
    if (url != null) result.url = url;
    if (pcmData != null) result.pcmData = pcmData;
    if (ttsText != null) result.ttsText = ttsText;
    return result;
  }

  PlayAudio._();

  factory PlayAudio.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PlayAudio.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, PlayAudio_Source> _PlayAudio_SourceByTag = {
    3: PlayAudio_Source.url,
    4: PlayAudio_Source.pcmData,
    5: PlayAudio_Source.ttsText,
    0: PlayAudio_Source.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PlayAudio',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..oo(0, [3, 4, 5])
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOS(2, _omitFieldNames ? '' : 'deviceId')
    ..aOS(3, _omitFieldNames ? '' : 'url')
    ..a<$core.List<$core.int>>(
        4, _omitFieldNames ? '' : 'pcmData', $pb.PbFieldType.OY)
    ..aOS(5, _omitFieldNames ? '' : 'ttsText')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PlayAudio clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PlayAudio copyWith(void Function(PlayAudio) updates) =>
      super.copyWith((message) => updates(message as PlayAudio)) as PlayAudio;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PlayAudio create() => PlayAudio._();
  @$core.override
  PlayAudio createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PlayAudio getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<PlayAudio>(create);
  static PlayAudio? _defaultInstance;

  @$pb.TagNumber(3)
  @$pb.TagNumber(4)
  @$pb.TagNumber(5)
  PlayAudio_Source whichSource() => _PlayAudio_SourceByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(3)
  @$pb.TagNumber(4)
  @$pb.TagNumber(5)
  void clearSource() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get deviceId => $_getSZ(1);
  @$pb.TagNumber(2)
  set deviceId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDeviceId() => $_has(1);
  @$pb.TagNumber(2)
  void clearDeviceId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get url => $_getSZ(2);
  @$pb.TagNumber(3)
  set url($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasUrl() => $_has(2);
  @$pb.TagNumber(3)
  void clearUrl() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.List<$core.int> get pcmData => $_getN(3);
  @$pb.TagNumber(4)
  set pcmData($core.List<$core.int> value) => $_setBytes(3, value);
  @$pb.TagNumber(4)
  $core.bool hasPcmData() => $_has(3);
  @$pb.TagNumber(4)
  void clearPcmData() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get ttsText => $_getSZ(4);
  @$pb.TagNumber(5)
  set ttsText($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasTtsText() => $_has(4);
  @$pb.TagNumber(5)
  void clearTtsText() => $_clearField(5);
}

class ShowMedia extends $pb.GeneratedMessage {
  factory ShowMedia({
    $core.String? requestId,
    $core.String? deviceId,
    $core.String? mediaUrl,
    $core.String? mediaType,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (deviceId != null) result.deviceId = deviceId;
    if (mediaUrl != null) result.mediaUrl = mediaUrl;
    if (mediaType != null) result.mediaType = mediaType;
    return result;
  }

  ShowMedia._();

  factory ShowMedia.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ShowMedia.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ShowMedia',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOS(2, _omitFieldNames ? '' : 'deviceId')
    ..aOS(3, _omitFieldNames ? '' : 'mediaUrl')
    ..aOS(4, _omitFieldNames ? '' : 'mediaType')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ShowMedia clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ShowMedia copyWith(void Function(ShowMedia) updates) =>
      super.copyWith((message) => updates(message as ShowMedia)) as ShowMedia;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ShowMedia create() => ShowMedia._();
  @$core.override
  ShowMedia createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ShowMedia getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<ShowMedia>(create);
  static ShowMedia? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get deviceId => $_getSZ(1);
  @$pb.TagNumber(2)
  set deviceId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDeviceId() => $_has(1);
  @$pb.TagNumber(2)
  void clearDeviceId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get mediaUrl => $_getSZ(2);
  @$pb.TagNumber(3)
  set mediaUrl($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasMediaUrl() => $_has(2);
  @$pb.TagNumber(3)
  void clearMediaUrl() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get mediaType => $_getSZ(3);
  @$pb.TagNumber(4)
  set mediaType($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasMediaType() => $_has(3);
  @$pb.TagNumber(4)
  void clearMediaType() => $_clearField(4);
}

enum InputEvent_Payload { key, pointer, touch, uiAction, notSet }

class InputEvent extends $pb.GeneratedMessage {
  factory InputEvent({
    $core.String? deviceId,
    KeyEvent? key,
    PointerEvent? pointer,
    TouchEvent? touch,
    UIAction? uiAction,
  }) {
    final result = create();
    if (deviceId != null) result.deviceId = deviceId;
    if (key != null) result.key = key;
    if (pointer != null) result.pointer = pointer;
    if (touch != null) result.touch = touch;
    if (uiAction != null) result.uiAction = uiAction;
    return result;
  }

  InputEvent._();

  factory InputEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory InputEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, InputEvent_Payload>
      _InputEvent_PayloadByTag = {
    2: InputEvent_Payload.key,
    3: InputEvent_Payload.pointer,
    4: InputEvent_Payload.touch,
    5: InputEvent_Payload.uiAction,
    0: InputEvent_Payload.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'InputEvent',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..oo(0, [2, 3, 4, 5])
    ..aOS(1, _omitFieldNames ? '' : 'deviceId')
    ..aOM<KeyEvent>(2, _omitFieldNames ? '' : 'key',
        subBuilder: KeyEvent.create)
    ..aOM<PointerEvent>(3, _omitFieldNames ? '' : 'pointer',
        subBuilder: PointerEvent.create)
    ..aOM<TouchEvent>(4, _omitFieldNames ? '' : 'touch',
        subBuilder: TouchEvent.create)
    ..aOM<UIAction>(5, _omitFieldNames ? '' : 'uiAction',
        subBuilder: UIAction.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  InputEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  InputEvent copyWith(void Function(InputEvent) updates) =>
      super.copyWith((message) => updates(message as InputEvent)) as InputEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static InputEvent create() => InputEvent._();
  @$core.override
  InputEvent createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static InputEvent getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<InputEvent>(create);
  static InputEvent? _defaultInstance;

  @$pb.TagNumber(2)
  @$pb.TagNumber(3)
  @$pb.TagNumber(4)
  @$pb.TagNumber(5)
  InputEvent_Payload whichPayload() =>
      _InputEvent_PayloadByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(2)
  @$pb.TagNumber(3)
  @$pb.TagNumber(4)
  @$pb.TagNumber(5)
  void clearPayload() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  $core.String get deviceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set deviceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDeviceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDeviceId() => $_clearField(1);

  @$pb.TagNumber(2)
  KeyEvent get key => $_getN(1);
  @$pb.TagNumber(2)
  set key(KeyEvent value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasKey() => $_has(1);
  @$pb.TagNumber(2)
  void clearKey() => $_clearField(2);
  @$pb.TagNumber(2)
  KeyEvent ensureKey() => $_ensure(1);

  @$pb.TagNumber(3)
  PointerEvent get pointer => $_getN(2);
  @$pb.TagNumber(3)
  set pointer(PointerEvent value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasPointer() => $_has(2);
  @$pb.TagNumber(3)
  void clearPointer() => $_clearField(3);
  @$pb.TagNumber(3)
  PointerEvent ensurePointer() => $_ensure(2);

  @$pb.TagNumber(4)
  TouchEvent get touch => $_getN(3);
  @$pb.TagNumber(4)
  set touch(TouchEvent value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasTouch() => $_has(3);
  @$pb.TagNumber(4)
  void clearTouch() => $_clearField(4);
  @$pb.TagNumber(4)
  TouchEvent ensureTouch() => $_ensure(3);

  @$pb.TagNumber(5)
  UIAction get uiAction => $_getN(4);
  @$pb.TagNumber(5)
  set uiAction(UIAction value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasUiAction() => $_has(4);
  @$pb.TagNumber(5)
  void clearUiAction() => $_clearField(5);
  @$pb.TagNumber(5)
  UIAction ensureUiAction() => $_ensure(4);
}

class KeyEvent extends $pb.GeneratedMessage {
  factory KeyEvent({
    $core.String? key,
    $core.bool? down,
    $core.bool? up,
    $core.String? text,
  }) {
    final result = create();
    if (key != null) result.key = key;
    if (down != null) result.down = down;
    if (up != null) result.up = up;
    if (text != null) result.text = text;
    return result;
  }

  KeyEvent._();

  factory KeyEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory KeyEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'KeyEvent',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'key')
    ..aOB(2, _omitFieldNames ? '' : 'down')
    ..aOB(3, _omitFieldNames ? '' : 'up')
    ..aOS(4, _omitFieldNames ? '' : 'text')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  KeyEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  KeyEvent copyWith(void Function(KeyEvent) updates) =>
      super.copyWith((message) => updates(message as KeyEvent)) as KeyEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static KeyEvent create() => KeyEvent._();
  @$core.override
  KeyEvent createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static KeyEvent getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<KeyEvent>(create);
  static KeyEvent? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get key => $_getSZ(0);
  @$pb.TagNumber(1)
  set key($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasKey() => $_has(0);
  @$pb.TagNumber(1)
  void clearKey() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get down => $_getBF(1);
  @$pb.TagNumber(2)
  set down($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDown() => $_has(1);
  @$pb.TagNumber(2)
  void clearDown() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.bool get up => $_getBF(2);
  @$pb.TagNumber(3)
  set up($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasUp() => $_has(2);
  @$pb.TagNumber(3)
  void clearUp() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get text => $_getSZ(3);
  @$pb.TagNumber(4)
  set text($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasText() => $_has(3);
  @$pb.TagNumber(4)
  void clearText() => $_clearField(4);
}

class PointerEvent extends $pb.GeneratedMessage {
  factory PointerEvent({
    $core.String? action,
    $core.double? x,
    $core.double? y,
    $core.double? deltaX,
    $core.double? deltaY,
    $core.int? button,
  }) {
    final result = create();
    if (action != null) result.action = action;
    if (x != null) result.x = x;
    if (y != null) result.y = y;
    if (deltaX != null) result.deltaX = deltaX;
    if (deltaY != null) result.deltaY = deltaY;
    if (button != null) result.button = button;
    return result;
  }

  PointerEvent._();

  factory PointerEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PointerEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PointerEvent',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'action')
    ..aD(2, _omitFieldNames ? '' : 'x')
    ..aD(3, _omitFieldNames ? '' : 'y')
    ..aD(4, _omitFieldNames ? '' : 'deltaX')
    ..aD(5, _omitFieldNames ? '' : 'deltaY')
    ..aI(6, _omitFieldNames ? '' : 'button')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PointerEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PointerEvent copyWith(void Function(PointerEvent) updates) =>
      super.copyWith((message) => updates(message as PointerEvent))
          as PointerEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PointerEvent create() => PointerEvent._();
  @$core.override
  PointerEvent createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PointerEvent getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<PointerEvent>(create);
  static PointerEvent? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get action => $_getSZ(0);
  @$pb.TagNumber(1)
  set action($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAction() => $_has(0);
  @$pb.TagNumber(1)
  void clearAction() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.double get x => $_getN(1);
  @$pb.TagNumber(2)
  set x($core.double value) => $_setDouble(1, value);
  @$pb.TagNumber(2)
  $core.bool hasX() => $_has(1);
  @$pb.TagNumber(2)
  void clearX() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.double get y => $_getN(2);
  @$pb.TagNumber(3)
  set y($core.double value) => $_setDouble(2, value);
  @$pb.TagNumber(3)
  $core.bool hasY() => $_has(2);
  @$pb.TagNumber(3)
  void clearY() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.double get deltaX => $_getN(3);
  @$pb.TagNumber(4)
  set deltaX($core.double value) => $_setDouble(3, value);
  @$pb.TagNumber(4)
  $core.bool hasDeltaX() => $_has(3);
  @$pb.TagNumber(4)
  void clearDeltaX() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.double get deltaY => $_getN(4);
  @$pb.TagNumber(5)
  set deltaY($core.double value) => $_setDouble(4, value);
  @$pb.TagNumber(5)
  $core.bool hasDeltaY() => $_has(4);
  @$pb.TagNumber(5)
  void clearDeltaY() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.int get button => $_getIZ(5);
  @$pb.TagNumber(6)
  set button($core.int value) => $_setSignedInt32(5, value);
  @$pb.TagNumber(6)
  $core.bool hasButton() => $_has(5);
  @$pb.TagNumber(6)
  void clearButton() => $_clearField(6);
}

class TouchPoint extends $pb.GeneratedMessage {
  factory TouchPoint({
    $core.int? id,
    $core.double? x,
    $core.double? y,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (x != null) result.x = x;
    if (y != null) result.y = y;
    return result;
  }

  TouchPoint._();

  factory TouchPoint.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TouchPoint.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TouchPoint',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'id')
    ..aD(2, _omitFieldNames ? '' : 'x')
    ..aD(3, _omitFieldNames ? '' : 'y')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TouchPoint clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TouchPoint copyWith(void Function(TouchPoint) updates) =>
      super.copyWith((message) => updates(message as TouchPoint)) as TouchPoint;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TouchPoint create() => TouchPoint._();
  @$core.override
  TouchPoint createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TouchPoint getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TouchPoint>(create);
  static TouchPoint? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get id => $_getIZ(0);
  @$pb.TagNumber(1)
  set id($core.int value) => $_setSignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.double get x => $_getN(1);
  @$pb.TagNumber(2)
  set x($core.double value) => $_setDouble(1, value);
  @$pb.TagNumber(2)
  $core.bool hasX() => $_has(1);
  @$pb.TagNumber(2)
  void clearX() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.double get y => $_getN(2);
  @$pb.TagNumber(3)
  set y($core.double value) => $_setDouble(2, value);
  @$pb.TagNumber(3)
  $core.bool hasY() => $_has(2);
  @$pb.TagNumber(3)
  void clearY() => $_clearField(3);
}

class TouchEvent extends $pb.GeneratedMessage {
  factory TouchEvent({
    $core.String? action,
    $core.Iterable<TouchPoint>? points,
  }) {
    final result = create();
    if (action != null) result.action = action;
    if (points != null) result.points.addAll(points);
    return result;
  }

  TouchEvent._();

  factory TouchEvent.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TouchEvent.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TouchEvent',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'action')
    ..pPM<TouchPoint>(2, _omitFieldNames ? '' : 'points',
        subBuilder: TouchPoint.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TouchEvent clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TouchEvent copyWith(void Function(TouchEvent) updates) =>
      super.copyWith((message) => updates(message as TouchEvent)) as TouchEvent;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TouchEvent create() => TouchEvent._();
  @$core.override
  TouchEvent createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TouchEvent getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TouchEvent>(create);
  static TouchEvent? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get action => $_getSZ(0);
  @$pb.TagNumber(1)
  set action($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAction() => $_has(0);
  @$pb.TagNumber(1)
  void clearAction() => $_clearField(1);

  @$pb.TagNumber(2)
  $pb.PbList<TouchPoint> get points => $_getList(1);
}

class UIAction extends $pb.GeneratedMessage {
  factory UIAction({
    $core.String? componentId,
    $core.String? action,
    $core.String? value,
  }) {
    final result = create();
    if (componentId != null) result.componentId = componentId;
    if (action != null) result.action = action;
    if (value != null) result.value = value;
    return result;
  }

  UIAction._();

  factory UIAction.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory UIAction.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'UIAction',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'componentId')
    ..aOS(2, _omitFieldNames ? '' : 'action')
    ..aOS(3, _omitFieldNames ? '' : 'value')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UIAction clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UIAction copyWith(void Function(UIAction) updates) =>
      super.copyWith((message) => updates(message as UIAction)) as UIAction;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static UIAction create() => UIAction._();
  @$core.override
  UIAction createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static UIAction getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<UIAction>(create);
  static UIAction? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get componentId => $_getSZ(0);
  @$pb.TagNumber(1)
  set componentId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasComponentId() => $_has(0);
  @$pb.TagNumber(1)
  void clearComponentId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get action => $_getSZ(1);
  @$pb.TagNumber(2)
  set action($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAction() => $_has(1);
  @$pb.TagNumber(2)
  void clearAction() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get value => $_getSZ(2);
  @$pb.TagNumber(3)
  set value($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasValue() => $_has(2);
  @$pb.TagNumber(3)
  void clearValue() => $_clearField(3);
}

class SensorData extends $pb.GeneratedMessage {
  factory SensorData({
    $core.String? deviceId,
    $fixnum.Int64? unixMs,
    $core.Iterable<$core.MapEntry<$core.String, $core.double>>? values,
  }) {
    final result = create();
    if (deviceId != null) result.deviceId = deviceId;
    if (unixMs != null) result.unixMs = unixMs;
    if (values != null) result.values.addEntries(values);
    return result;
  }

  SensorData._();

  factory SensorData.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory SensorData.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'SensorData',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'deviceId')
    ..aInt64(2, _omitFieldNames ? '' : 'unixMs')
    ..m<$core.String, $core.double>(3, _omitFieldNames ? '' : 'values',
        entryClassName: 'SensorData.ValuesEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OD,
        packageName: const $pb.PackageName('terminals.io.v1'))
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SensorData clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  SensorData copyWith(void Function(SensorData) updates) =>
      super.copyWith((message) => updates(message as SensorData)) as SensorData;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static SensorData create() => SensorData._();
  @$core.override
  SensorData createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static SensorData getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<SensorData>(create);
  static SensorData? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get deviceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set deviceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDeviceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDeviceId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get unixMs => $_getI64(1);
  @$pb.TagNumber(2)
  set unixMs($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasUnixMs() => $_has(1);
  @$pb.TagNumber(2)
  void clearUnixMs() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbMap<$core.String, $core.double> get values => $_getMap(2);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
