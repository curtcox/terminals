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
    $core.String? format,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (deviceId != null) result.deviceId = deviceId;
    if (url != null) result.url = url;
    if (pcmData != null) result.pcmData = pcmData;
    if (ttsText != null) result.ttsText = ttsText;
    if (format != null) result.format = format;
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
    ..aOS(6, _omitFieldNames ? '' : 'format')
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

  @$pb.TagNumber(6)
  $core.String get format => $_getSZ(5);
  @$pb.TagNumber(6)
  set format($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasFormat() => $_has(5);
  @$pb.TagNumber(6)
  void clearFormat() => $_clearField(6);
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

class FlowNode extends $pb.GeneratedMessage {
  factory FlowNode({
    $core.String? id,
    $core.String? kind,
    $core.Iterable<$core.MapEntry<$core.String, $core.String>>? args,
    $core.String? exec,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (kind != null) result.kind = kind;
    if (args != null) result.args.addEntries(args);
    if (exec != null) result.exec = exec;
    return result;
  }

  FlowNode._();

  factory FlowNode.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory FlowNode.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FlowNode',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'kind')
    ..m<$core.String, $core.String>(3, _omitFieldNames ? '' : 'args',
        entryClassName: 'FlowNode.ArgsEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OS,
        packageName: const $pb.PackageName('terminals.io.v1'))
    ..aOS(4, _omitFieldNames ? '' : 'exec')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FlowNode clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FlowNode copyWith(void Function(FlowNode) updates) =>
      super.copyWith((message) => updates(message as FlowNode)) as FlowNode;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static FlowNode create() => FlowNode._();
  @$core.override
  FlowNode createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static FlowNode getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<FlowNode>(create);
  static FlowNode? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get kind => $_getSZ(1);
  @$pb.TagNumber(2)
  set kind($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasKind() => $_has(1);
  @$pb.TagNumber(2)
  void clearKind() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbMap<$core.String, $core.String> get args => $_getMap(2);

  @$pb.TagNumber(4)
  $core.String get exec => $_getSZ(3);
  @$pb.TagNumber(4)
  set exec($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasExec() => $_has(3);
  @$pb.TagNumber(4)
  void clearExec() => $_clearField(4);
}

class FlowEdge extends $pb.GeneratedMessage {
  factory FlowEdge({
    $core.String? from,
    $core.String? to,
  }) {
    final result = create();
    if (from != null) result.from = from;
    if (to != null) result.to = to;
    return result;
  }

  FlowEdge._();

  factory FlowEdge.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory FlowEdge.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FlowEdge',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'from')
    ..aOS(2, _omitFieldNames ? '' : 'to')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FlowEdge clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FlowEdge copyWith(void Function(FlowEdge) updates) =>
      super.copyWith((message) => updates(message as FlowEdge)) as FlowEdge;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static FlowEdge create() => FlowEdge._();
  @$core.override
  FlowEdge createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static FlowEdge getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<FlowEdge>(create);
  static FlowEdge? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get from => $_getSZ(0);
  @$pb.TagNumber(1)
  set from($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasFrom() => $_has(0);
  @$pb.TagNumber(1)
  void clearFrom() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get to => $_getSZ(1);
  @$pb.TagNumber(2)
  set to($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasTo() => $_has(1);
  @$pb.TagNumber(2)
  void clearTo() => $_clearField(2);
}

class FlowPlan extends $pb.GeneratedMessage {
  factory FlowPlan({
    $core.Iterable<FlowNode>? nodes,
    $core.Iterable<FlowEdge>? edges,
  }) {
    final result = create();
    if (nodes != null) result.nodes.addAll(nodes);
    if (edges != null) result.edges.addAll(edges);
    return result;
  }

  FlowPlan._();

  factory FlowPlan.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory FlowPlan.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FlowPlan',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..pPM<FlowNode>(1, _omitFieldNames ? '' : 'nodes',
        subBuilder: FlowNode.create)
    ..pPM<FlowEdge>(2, _omitFieldNames ? '' : 'edges',
        subBuilder: FlowEdge.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FlowPlan clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FlowPlan copyWith(void Function(FlowPlan) updates) =>
      super.copyWith((message) => updates(message as FlowPlan)) as FlowPlan;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static FlowPlan create() => FlowPlan._();
  @$core.override
  FlowPlan createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static FlowPlan getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<FlowPlan>(create);
  static FlowPlan? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<FlowNode> get nodes => $_getList(0);

  @$pb.TagNumber(2)
  $pb.PbList<FlowEdge> get edges => $_getList(1);
}

class StartFlow extends $pb.GeneratedMessage {
  factory StartFlow({
    $core.String? flowId,
    FlowPlan? plan,
  }) {
    final result = create();
    if (flowId != null) result.flowId = flowId;
    if (plan != null) result.plan = plan;
    return result;
  }

  StartFlow._();

  factory StartFlow.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StartFlow.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StartFlow',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'flowId')
    ..aOM<FlowPlan>(2, _omitFieldNames ? '' : 'plan',
        subBuilder: FlowPlan.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StartFlow clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StartFlow copyWith(void Function(StartFlow) updates) =>
      super.copyWith((message) => updates(message as StartFlow)) as StartFlow;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StartFlow create() => StartFlow._();
  @$core.override
  StartFlow createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StartFlow getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<StartFlow>(create);
  static StartFlow? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get flowId => $_getSZ(0);
  @$pb.TagNumber(1)
  set flowId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasFlowId() => $_has(0);
  @$pb.TagNumber(1)
  void clearFlowId() => $_clearField(1);

  @$pb.TagNumber(2)
  FlowPlan get plan => $_getN(1);
  @$pb.TagNumber(2)
  set plan(FlowPlan value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasPlan() => $_has(1);
  @$pb.TagNumber(2)
  void clearPlan() => $_clearField(2);
  @$pb.TagNumber(2)
  FlowPlan ensurePlan() => $_ensure(1);
}

class PatchFlow extends $pb.GeneratedMessage {
  factory PatchFlow({
    $core.String? flowId,
    FlowPlan? plan,
  }) {
    final result = create();
    if (flowId != null) result.flowId = flowId;
    if (plan != null) result.plan = plan;
    return result;
  }

  PatchFlow._();

  factory PatchFlow.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory PatchFlow.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'PatchFlow',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'flowId')
    ..aOM<FlowPlan>(2, _omitFieldNames ? '' : 'plan',
        subBuilder: FlowPlan.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PatchFlow clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  PatchFlow copyWith(void Function(PatchFlow) updates) =>
      super.copyWith((message) => updates(message as PatchFlow)) as PatchFlow;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static PatchFlow create() => PatchFlow._();
  @$core.override
  PatchFlow createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static PatchFlow getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<PatchFlow>(create);
  static PatchFlow? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get flowId => $_getSZ(0);
  @$pb.TagNumber(1)
  set flowId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasFlowId() => $_has(0);
  @$pb.TagNumber(1)
  void clearFlowId() => $_clearField(1);

  @$pb.TagNumber(2)
  FlowPlan get plan => $_getN(1);
  @$pb.TagNumber(2)
  set plan(FlowPlan value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasPlan() => $_has(1);
  @$pb.TagNumber(2)
  void clearPlan() => $_clearField(2);
  @$pb.TagNumber(2)
  FlowPlan ensurePlan() => $_ensure(1);
}

class StopFlow extends $pb.GeneratedMessage {
  factory StopFlow({
    $core.String? flowId,
  }) {
    final result = create();
    if (flowId != null) result.flowId = flowId;
    return result;
  }

  StopFlow._();

  factory StopFlow.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StopFlow.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StopFlow',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'flowId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StopFlow clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StopFlow copyWith(void Function(StopFlow) updates) =>
      super.copyWith((message) => updates(message as StopFlow)) as StopFlow;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StopFlow create() => StopFlow._();
  @$core.override
  StopFlow createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StopFlow getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<StopFlow>(create);
  static StopFlow? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get flowId => $_getSZ(0);
  @$pb.TagNumber(1)
  set flowId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasFlowId() => $_has(0);
  @$pb.TagNumber(1)
  void clearFlowId() => $_clearField(1);
}

class DeviceRef extends $pb.GeneratedMessage {
  factory DeviceRef({
    $core.String? deviceId,
  }) {
    final result = create();
    if (deviceId != null) result.deviceId = deviceId;
    return result;
  }

  DeviceRef._();

  factory DeviceRef.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory DeviceRef.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'DeviceRef',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'deviceId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeviceRef clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  DeviceRef copyWith(void Function(DeviceRef) updates) =>
      super.copyWith((message) => updates(message as DeviceRef)) as DeviceRef;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static DeviceRef create() => DeviceRef._();
  @$core.override
  DeviceRef createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static DeviceRef getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<DeviceRef>(create);
  static DeviceRef? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get deviceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set deviceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDeviceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDeviceId() => $_clearField(1);
}

class Pose extends $pb.GeneratedMessage {
  factory Pose({
    $core.double? x,
    $core.double? y,
    $core.double? z,
    $core.double? yaw,
    $core.double? pitch,
    $core.double? roll,
    $core.double? confidence,
  }) {
    final result = create();
    if (x != null) result.x = x;
    if (y != null) result.y = y;
    if (z != null) result.z = z;
    if (yaw != null) result.yaw = yaw;
    if (pitch != null) result.pitch = pitch;
    if (roll != null) result.roll = roll;
    if (confidence != null) result.confidence = confidence;
    return result;
  }

  Pose._();

  factory Pose.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Pose.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Pose',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aD(1, _omitFieldNames ? '' : 'x')
    ..aD(2, _omitFieldNames ? '' : 'y')
    ..aD(3, _omitFieldNames ? '' : 'z')
    ..aD(4, _omitFieldNames ? '' : 'yaw')
    ..aD(5, _omitFieldNames ? '' : 'pitch')
    ..aD(6, _omitFieldNames ? '' : 'roll')
    ..aD(7, _omitFieldNames ? '' : 'confidence')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Pose clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Pose copyWith(void Function(Pose) updates) =>
      super.copyWith((message) => updates(message as Pose)) as Pose;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Pose create() => Pose._();
  @$core.override
  Pose createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static Pose getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Pose>(create);
  static Pose? _defaultInstance;

  @$pb.TagNumber(1)
  $core.double get x => $_getN(0);
  @$pb.TagNumber(1)
  set x($core.double value) => $_setDouble(0, value);
  @$pb.TagNumber(1)
  $core.bool hasX() => $_has(0);
  @$pb.TagNumber(1)
  void clearX() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.double get y => $_getN(1);
  @$pb.TagNumber(2)
  set y($core.double value) => $_setDouble(1, value);
  @$pb.TagNumber(2)
  $core.bool hasY() => $_has(1);
  @$pb.TagNumber(2)
  void clearY() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.double get z => $_getN(2);
  @$pb.TagNumber(3)
  set z($core.double value) => $_setDouble(2, value);
  @$pb.TagNumber(3)
  $core.bool hasZ() => $_has(2);
  @$pb.TagNumber(3)
  void clearZ() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.double get yaw => $_getN(3);
  @$pb.TagNumber(4)
  set yaw($core.double value) => $_setDouble(3, value);
  @$pb.TagNumber(4)
  $core.bool hasYaw() => $_has(3);
  @$pb.TagNumber(4)
  void clearYaw() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.double get pitch => $_getN(4);
  @$pb.TagNumber(5)
  set pitch($core.double value) => $_setDouble(4, value);
  @$pb.TagNumber(5)
  $core.bool hasPitch() => $_has(4);
  @$pb.TagNumber(5)
  void clearPitch() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.double get roll => $_getN(5);
  @$pb.TagNumber(6)
  set roll($core.double value) => $_setDouble(5, value);
  @$pb.TagNumber(6)
  $core.bool hasRoll() => $_has(5);
  @$pb.TagNumber(6)
  void clearRoll() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.double get confidence => $_getN(6);
  @$pb.TagNumber(7)
  set confidence($core.double value) => $_setDouble(6, value);
  @$pb.TagNumber(7)
  $core.bool hasConfidence() => $_has(6);
  @$pb.TagNumber(7)
  void clearConfidence() => $_clearField(7);
}

class LocationEstimate extends $pb.GeneratedMessage {
  factory LocationEstimate({
    $core.String? zone,
    Pose? pose,
    $core.double? radiusM,
    $core.double? confidence,
    $core.Iterable<$core.String>? sources,
  }) {
    final result = create();
    if (zone != null) result.zone = zone;
    if (pose != null) result.pose = pose;
    if (radiusM != null) result.radiusM = radiusM;
    if (confidence != null) result.confidence = confidence;
    if (sources != null) result.sources.addAll(sources);
    return result;
  }

  LocationEstimate._();

  factory LocationEstimate.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory LocationEstimate.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'LocationEstimate',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'zone')
    ..aOM<Pose>(2, _omitFieldNames ? '' : 'pose', subBuilder: Pose.create)
    ..aD(3, _omitFieldNames ? '' : 'radiusM')
    ..aD(4, _omitFieldNames ? '' : 'confidence')
    ..pPS(5, _omitFieldNames ? '' : 'sources')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LocationEstimate clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LocationEstimate copyWith(void Function(LocationEstimate) updates) =>
      super.copyWith((message) => updates(message as LocationEstimate))
          as LocationEstimate;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static LocationEstimate create() => LocationEstimate._();
  @$core.override
  LocationEstimate createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static LocationEstimate getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<LocationEstimate>(create);
  static LocationEstimate? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get zone => $_getSZ(0);
  @$pb.TagNumber(1)
  set zone($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasZone() => $_has(0);
  @$pb.TagNumber(1)
  void clearZone() => $_clearField(1);

  @$pb.TagNumber(2)
  Pose get pose => $_getN(1);
  @$pb.TagNumber(2)
  set pose(Pose value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasPose() => $_has(1);
  @$pb.TagNumber(2)
  void clearPose() => $_clearField(2);
  @$pb.TagNumber(2)
  Pose ensurePose() => $_ensure(1);

  @$pb.TagNumber(3)
  $core.double get radiusM => $_getN(2);
  @$pb.TagNumber(3)
  set radiusM($core.double value) => $_setDouble(2, value);
  @$pb.TagNumber(3)
  $core.bool hasRadiusM() => $_has(2);
  @$pb.TagNumber(3)
  void clearRadiusM() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.double get confidence => $_getN(3);
  @$pb.TagNumber(4)
  set confidence($core.double value) => $_setDouble(3, value);
  @$pb.TagNumber(4)
  $core.bool hasConfidence() => $_has(3);
  @$pb.TagNumber(4)
  void clearConfidence() => $_clearField(4);

  @$pb.TagNumber(5)
  $pb.PbList<$core.String> get sources => $_getList(4);
}

class ObservationProvenance extends $pb.GeneratedMessage {
  factory ObservationProvenance({
    $core.String? flowId,
    $core.String? nodeId,
    $core.String? execSite,
    $core.String? modelId,
    $core.String? calibrationVersion,
  }) {
    final result = create();
    if (flowId != null) result.flowId = flowId;
    if (nodeId != null) result.nodeId = nodeId;
    if (execSite != null) result.execSite = execSite;
    if (modelId != null) result.modelId = modelId;
    if (calibrationVersion != null)
      result.calibrationVersion = calibrationVersion;
    return result;
  }

  ObservationProvenance._();

  factory ObservationProvenance.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ObservationProvenance.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ObservationProvenance',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'flowId')
    ..aOS(2, _omitFieldNames ? '' : 'nodeId')
    ..aOS(3, _omitFieldNames ? '' : 'execSite')
    ..aOS(4, _omitFieldNames ? '' : 'modelId')
    ..aOS(5, _omitFieldNames ? '' : 'calibrationVersion')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ObservationProvenance clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ObservationProvenance copyWith(
          void Function(ObservationProvenance) updates) =>
      super.copyWith((message) => updates(message as ObservationProvenance))
          as ObservationProvenance;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ObservationProvenance create() => ObservationProvenance._();
  @$core.override
  ObservationProvenance createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ObservationProvenance getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ObservationProvenance>(create);
  static ObservationProvenance? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get flowId => $_getSZ(0);
  @$pb.TagNumber(1)
  set flowId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasFlowId() => $_has(0);
  @$pb.TagNumber(1)
  void clearFlowId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get nodeId => $_getSZ(1);
  @$pb.TagNumber(2)
  set nodeId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasNodeId() => $_has(1);
  @$pb.TagNumber(2)
  void clearNodeId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get execSite => $_getSZ(2);
  @$pb.TagNumber(3)
  set execSite($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasExecSite() => $_has(2);
  @$pb.TagNumber(3)
  void clearExecSite() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get modelId => $_getSZ(3);
  @$pb.TagNumber(4)
  set modelId($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasModelId() => $_has(3);
  @$pb.TagNumber(4)
  void clearModelId() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get calibrationVersion => $_getSZ(4);
  @$pb.TagNumber(5)
  set calibrationVersion($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasCalibrationVersion() => $_has(4);
  @$pb.TagNumber(5)
  void clearCalibrationVersion() => $_clearField(5);
}

class ArtifactRef extends $pb.GeneratedMessage {
  factory ArtifactRef({
    $core.String? id,
    $core.String? kind,
    DeviceRef? source,
    $fixnum.Int64? startUnixMs,
    $fixnum.Int64? endUnixMs,
    $core.String? uri,
  }) {
    final result = create();
    if (id != null) result.id = id;
    if (kind != null) result.kind = kind;
    if (source != null) result.source = source;
    if (startUnixMs != null) result.startUnixMs = startUnixMs;
    if (endUnixMs != null) result.endUnixMs = endUnixMs;
    if (uri != null) result.uri = uri;
    return result;
  }

  ArtifactRef._();

  factory ArtifactRef.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ArtifactRef.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ArtifactRef',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'id')
    ..aOS(2, _omitFieldNames ? '' : 'kind')
    ..aOM<DeviceRef>(3, _omitFieldNames ? '' : 'source',
        subBuilder: DeviceRef.create)
    ..aInt64(4, _omitFieldNames ? '' : 'startUnixMs')
    ..aInt64(5, _omitFieldNames ? '' : 'endUnixMs')
    ..aOS(6, _omitFieldNames ? '' : 'uri')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ArtifactRef clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ArtifactRef copyWith(void Function(ArtifactRef) updates) =>
      super.copyWith((message) => updates(message as ArtifactRef))
          as ArtifactRef;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ArtifactRef create() => ArtifactRef._();
  @$core.override
  ArtifactRef createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ArtifactRef getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ArtifactRef>(create);
  static ArtifactRef? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get id => $_getSZ(0);
  @$pb.TagNumber(1)
  set id($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasId() => $_has(0);
  @$pb.TagNumber(1)
  void clearId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get kind => $_getSZ(1);
  @$pb.TagNumber(2)
  set kind($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasKind() => $_has(1);
  @$pb.TagNumber(2)
  void clearKind() => $_clearField(2);

  @$pb.TagNumber(3)
  DeviceRef get source => $_getN(2);
  @$pb.TagNumber(3)
  set source(DeviceRef value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasSource() => $_has(2);
  @$pb.TagNumber(3)
  void clearSource() => $_clearField(3);
  @$pb.TagNumber(3)
  DeviceRef ensureSource() => $_ensure(2);

  @$pb.TagNumber(4)
  $fixnum.Int64 get startUnixMs => $_getI64(3);
  @$pb.TagNumber(4)
  set startUnixMs($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasStartUnixMs() => $_has(3);
  @$pb.TagNumber(4)
  void clearStartUnixMs() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get endUnixMs => $_getI64(4);
  @$pb.TagNumber(5)
  set endUnixMs($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasEndUnixMs() => $_has(4);
  @$pb.TagNumber(5)
  void clearEndUnixMs() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get uri => $_getSZ(5);
  @$pb.TagNumber(6)
  set uri($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasUri() => $_has(5);
  @$pb.TagNumber(6)
  void clearUri() => $_clearField(6);
}

class Observation extends $pb.GeneratedMessage {
  factory Observation({
    $core.String? kind,
    $core.String? subject,
    DeviceRef? sourceDevice,
    $fixnum.Int64? occurredUnixMs,
    $core.double? confidence,
    $core.String? zone,
    LocationEstimate? location,
    $core.String? trackId,
    $core.Iterable<$core.MapEntry<$core.String, $core.String>>? attributes,
    $core.Iterable<ArtifactRef>? evidence,
    ObservationProvenance? provenance,
  }) {
    final result = create();
    if (kind != null) result.kind = kind;
    if (subject != null) result.subject = subject;
    if (sourceDevice != null) result.sourceDevice = sourceDevice;
    if (occurredUnixMs != null) result.occurredUnixMs = occurredUnixMs;
    if (confidence != null) result.confidence = confidence;
    if (zone != null) result.zone = zone;
    if (location != null) result.location = location;
    if (trackId != null) result.trackId = trackId;
    if (attributes != null) result.attributes.addEntries(attributes);
    if (evidence != null) result.evidence.addAll(evidence);
    if (provenance != null) result.provenance = provenance;
    return result;
  }

  Observation._();

  factory Observation.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Observation.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Observation',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'kind')
    ..aOS(2, _omitFieldNames ? '' : 'subject')
    ..aOM<DeviceRef>(3, _omitFieldNames ? '' : 'sourceDevice',
        subBuilder: DeviceRef.create)
    ..aInt64(4, _omitFieldNames ? '' : 'occurredUnixMs')
    ..aD(5, _omitFieldNames ? '' : 'confidence')
    ..aOS(6, _omitFieldNames ? '' : 'zone')
    ..aOM<LocationEstimate>(7, _omitFieldNames ? '' : 'location',
        subBuilder: LocationEstimate.create)
    ..aOS(8, _omitFieldNames ? '' : 'trackId')
    ..m<$core.String, $core.String>(9, _omitFieldNames ? '' : 'attributes',
        entryClassName: 'Observation.AttributesEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OS,
        packageName: const $pb.PackageName('terminals.io.v1'))
    ..pPM<ArtifactRef>(10, _omitFieldNames ? '' : 'evidence',
        subBuilder: ArtifactRef.create)
    ..aOM<ObservationProvenance>(11, _omitFieldNames ? '' : 'provenance',
        subBuilder: ObservationProvenance.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Observation clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Observation copyWith(void Function(Observation) updates) =>
      super.copyWith((message) => updates(message as Observation))
          as Observation;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Observation create() => Observation._();
  @$core.override
  Observation createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static Observation getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<Observation>(create);
  static Observation? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get kind => $_getSZ(0);
  @$pb.TagNumber(1)
  set kind($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasKind() => $_has(0);
  @$pb.TagNumber(1)
  void clearKind() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get subject => $_getSZ(1);
  @$pb.TagNumber(2)
  set subject($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSubject() => $_has(1);
  @$pb.TagNumber(2)
  void clearSubject() => $_clearField(2);

  @$pb.TagNumber(3)
  DeviceRef get sourceDevice => $_getN(2);
  @$pb.TagNumber(3)
  set sourceDevice(DeviceRef value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasSourceDevice() => $_has(2);
  @$pb.TagNumber(3)
  void clearSourceDevice() => $_clearField(3);
  @$pb.TagNumber(3)
  DeviceRef ensureSourceDevice() => $_ensure(2);

  @$pb.TagNumber(4)
  $fixnum.Int64 get occurredUnixMs => $_getI64(3);
  @$pb.TagNumber(4)
  set occurredUnixMs($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasOccurredUnixMs() => $_has(3);
  @$pb.TagNumber(4)
  void clearOccurredUnixMs() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.double get confidence => $_getN(4);
  @$pb.TagNumber(5)
  set confidence($core.double value) => $_setDouble(4, value);
  @$pb.TagNumber(5)
  $core.bool hasConfidence() => $_has(4);
  @$pb.TagNumber(5)
  void clearConfidence() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get zone => $_getSZ(5);
  @$pb.TagNumber(6)
  set zone($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasZone() => $_has(5);
  @$pb.TagNumber(6)
  void clearZone() => $_clearField(6);

  @$pb.TagNumber(7)
  LocationEstimate get location => $_getN(6);
  @$pb.TagNumber(7)
  set location(LocationEstimate value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasLocation() => $_has(6);
  @$pb.TagNumber(7)
  void clearLocation() => $_clearField(7);
  @$pb.TagNumber(7)
  LocationEstimate ensureLocation() => $_ensure(6);

  @$pb.TagNumber(8)
  $core.String get trackId => $_getSZ(7);
  @$pb.TagNumber(8)
  set trackId($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasTrackId() => $_has(7);
  @$pb.TagNumber(8)
  void clearTrackId() => $_clearField(8);

  @$pb.TagNumber(9)
  $pb.PbMap<$core.String, $core.String> get attributes => $_getMap(8);

  @$pb.TagNumber(10)
  $pb.PbList<ArtifactRef> get evidence => $_getList(9);

  @$pb.TagNumber(11)
  ObservationProvenance get provenance => $_getN(10);
  @$pb.TagNumber(11)
  set provenance(ObservationProvenance value) => $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasProvenance() => $_has(10);
  @$pb.TagNumber(11)
  void clearProvenance() => $_clearField(11);
  @$pb.TagNumber(11)
  ObservationProvenance ensureProvenance() => $_ensure(10);
}

class ObservationMessage extends $pb.GeneratedMessage {
  factory ObservationMessage({
    Observation? observation,
  }) {
    final result = create();
    if (observation != null) result.observation = observation;
    return result;
  }

  ObservationMessage._();

  factory ObservationMessage.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ObservationMessage.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ObservationMessage',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aOM<Observation>(1, _omitFieldNames ? '' : 'observation',
        subBuilder: Observation.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ObservationMessage clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ObservationMessage copyWith(void Function(ObservationMessage) updates) =>
      super.copyWith((message) => updates(message as ObservationMessage))
          as ObservationMessage;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ObservationMessage create() => ObservationMessage._();
  @$core.override
  ObservationMessage createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ObservationMessage getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ObservationMessage>(create);
  static ObservationMessage? _defaultInstance;

  @$pb.TagNumber(1)
  Observation get observation => $_getN(0);
  @$pb.TagNumber(1)
  set observation(Observation value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasObservation() => $_has(0);
  @$pb.TagNumber(1)
  void clearObservation() => $_clearField(1);
  @$pb.TagNumber(1)
  Observation ensureObservation() => $_ensure(0);
}

class ArtifactAvailable extends $pb.GeneratedMessage {
  factory ArtifactAvailable({
    ArtifactRef? artifact,
  }) {
    final result = create();
    if (artifact != null) result.artifact = artifact;
    return result;
  }

  ArtifactAvailable._();

  factory ArtifactAvailable.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ArtifactAvailable.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ArtifactAvailable',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aOM<ArtifactRef>(1, _omitFieldNames ? '' : 'artifact',
        subBuilder: ArtifactRef.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ArtifactAvailable clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ArtifactAvailable copyWith(void Function(ArtifactAvailable) updates) =>
      super.copyWith((message) => updates(message as ArtifactAvailable))
          as ArtifactAvailable;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ArtifactAvailable create() => ArtifactAvailable._();
  @$core.override
  ArtifactAvailable createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ArtifactAvailable getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ArtifactAvailable>(create);
  static ArtifactAvailable? _defaultInstance;

  @$pb.TagNumber(1)
  ArtifactRef get artifact => $_getN(0);
  @$pb.TagNumber(1)
  set artifact(ArtifactRef value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasArtifact() => $_has(0);
  @$pb.TagNumber(1)
  void clearArtifact() => $_clearField(1);
  @$pb.TagNumber(1)
  ArtifactRef ensureArtifact() => $_ensure(0);
}

class RequestArtifact extends $pb.GeneratedMessage {
  factory RequestArtifact({
    $core.String? artifactId,
  }) {
    final result = create();
    if (artifactId != null) result.artifactId = artifactId;
    return result;
  }

  RequestArtifact._();

  factory RequestArtifact.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RequestArtifact.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RequestArtifact',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'artifactId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RequestArtifact clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RequestArtifact copyWith(void Function(RequestArtifact) updates) =>
      super.copyWith((message) => updates(message as RequestArtifact))
          as RequestArtifact;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RequestArtifact create() => RequestArtifact._();
  @$core.override
  RequestArtifact createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RequestArtifact getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RequestArtifact>(create);
  static RequestArtifact? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get artifactId => $_getSZ(0);
  @$pb.TagNumber(1)
  set artifactId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasArtifactId() => $_has(0);
  @$pb.TagNumber(1)
  void clearArtifactId() => $_clearField(1);
}

class FlowStats extends $pb.GeneratedMessage {
  factory FlowStats({
    $core.String? flowId,
    $core.double? cpuPct,
    $core.double? memMb,
    $fixnum.Int64? droppedFrames,
    $core.String? state,
    $core.String? error,
  }) {
    final result = create();
    if (flowId != null) result.flowId = flowId;
    if (cpuPct != null) result.cpuPct = cpuPct;
    if (memMb != null) result.memMb = memMb;
    if (droppedFrames != null) result.droppedFrames = droppedFrames;
    if (state != null) result.state = state;
    if (error != null) result.error = error;
    return result;
  }

  FlowStats._();

  factory FlowStats.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory FlowStats.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'FlowStats',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'flowId')
    ..aD(2, _omitFieldNames ? '' : 'cpuPct')
    ..aD(3, _omitFieldNames ? '' : 'memMb')
    ..a<$fixnum.Int64>(
        4, _omitFieldNames ? '' : 'droppedFrames', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOS(5, _omitFieldNames ? '' : 'state')
    ..aOS(6, _omitFieldNames ? '' : 'error')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FlowStats clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  FlowStats copyWith(void Function(FlowStats) updates) =>
      super.copyWith((message) => updates(message as FlowStats)) as FlowStats;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static FlowStats create() => FlowStats._();
  @$core.override
  FlowStats createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static FlowStats getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<FlowStats>(create);
  static FlowStats? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get flowId => $_getSZ(0);
  @$pb.TagNumber(1)
  set flowId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasFlowId() => $_has(0);
  @$pb.TagNumber(1)
  void clearFlowId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.double get cpuPct => $_getN(1);
  @$pb.TagNumber(2)
  set cpuPct($core.double value) => $_setDouble(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCpuPct() => $_has(1);
  @$pb.TagNumber(2)
  void clearCpuPct() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.double get memMb => $_getN(2);
  @$pb.TagNumber(3)
  set memMb($core.double value) => $_setDouble(2, value);
  @$pb.TagNumber(3)
  $core.bool hasMemMb() => $_has(2);
  @$pb.TagNumber(3)
  void clearMemMb() => $_clearField(3);

  @$pb.TagNumber(4)
  $fixnum.Int64 get droppedFrames => $_getI64(3);
  @$pb.TagNumber(4)
  set droppedFrames($fixnum.Int64 value) => $_setInt64(3, value);
  @$pb.TagNumber(4)
  $core.bool hasDroppedFrames() => $_has(3);
  @$pb.TagNumber(4)
  void clearDroppedFrames() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get state => $_getSZ(4);
  @$pb.TagNumber(5)
  set state($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasState() => $_has(4);
  @$pb.TagNumber(5)
  void clearState() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get error => $_getSZ(5);
  @$pb.TagNumber(6)
  set error($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasError() => $_has(5);
  @$pb.TagNumber(6)
  void clearError() => $_clearField(6);
}

class ClockSample extends $pb.GeneratedMessage {
  factory ClockSample({
    $core.String? deviceId,
    $fixnum.Int64? clientUnixMs,
    $fixnum.Int64? serverUnixMs,
    $core.double? errorMs,
  }) {
    final result = create();
    if (deviceId != null) result.deviceId = deviceId;
    if (clientUnixMs != null) result.clientUnixMs = clientUnixMs;
    if (serverUnixMs != null) result.serverUnixMs = serverUnixMs;
    if (errorMs != null) result.errorMs = errorMs;
    return result;
  }

  ClockSample._();

  factory ClockSample.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClockSample.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClockSample',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'deviceId')
    ..aInt64(2, _omitFieldNames ? '' : 'clientUnixMs')
    ..aInt64(3, _omitFieldNames ? '' : 'serverUnixMs')
    ..aD(4, _omitFieldNames ? '' : 'errorMs')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClockSample clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClockSample copyWith(void Function(ClockSample) updates) =>
      super.copyWith((message) => updates(message as ClockSample))
          as ClockSample;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClockSample create() => ClockSample._();
  @$core.override
  ClockSample createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClockSample getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClockSample>(create);
  static ClockSample? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get deviceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set deviceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDeviceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDeviceId() => $_clearField(1);

  @$pb.TagNumber(2)
  $fixnum.Int64 get clientUnixMs => $_getI64(1);
  @$pb.TagNumber(2)
  set clientUnixMs($fixnum.Int64 value) => $_setInt64(1, value);
  @$pb.TagNumber(2)
  $core.bool hasClientUnixMs() => $_has(1);
  @$pb.TagNumber(2)
  void clearClientUnixMs() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get serverUnixMs => $_getI64(2);
  @$pb.TagNumber(3)
  set serverUnixMs($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasServerUnixMs() => $_has(2);
  @$pb.TagNumber(3)
  void clearServerUnixMs() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.double get errorMs => $_getN(3);
  @$pb.TagNumber(4)
  set errorMs($core.double value) => $_setDouble(3, value);
  @$pb.TagNumber(4)
  $core.bool hasErrorMs() => $_has(3);
  @$pb.TagNumber(4)
  void clearErrorMs() => $_clearField(4);
}

class InstallBundle extends $pb.GeneratedMessage {
  factory InstallBundle({
    $core.String? bundleId,
    $core.String? version,
    $core.List<$core.int>? tarGz,
    $core.String? sha256,
  }) {
    final result = create();
    if (bundleId != null) result.bundleId = bundleId;
    if (version != null) result.version = version;
    if (tarGz != null) result.tarGz = tarGz;
    if (sha256 != null) result.sha256 = sha256;
    return result;
  }

  InstallBundle._();

  factory InstallBundle.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory InstallBundle.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'InstallBundle',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'bundleId')
    ..aOS(2, _omitFieldNames ? '' : 'version')
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'tarGz', $pb.PbFieldType.OY)
    ..aOS(4, _omitFieldNames ? '' : 'sha256')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  InstallBundle clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  InstallBundle copyWith(void Function(InstallBundle) updates) =>
      super.copyWith((message) => updates(message as InstallBundle))
          as InstallBundle;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static InstallBundle create() => InstallBundle._();
  @$core.override
  InstallBundle createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static InstallBundle getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<InstallBundle>(create);
  static InstallBundle? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get bundleId => $_getSZ(0);
  @$pb.TagNumber(1)
  set bundleId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasBundleId() => $_has(0);
  @$pb.TagNumber(1)
  void clearBundleId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get version => $_getSZ(1);
  @$pb.TagNumber(2)
  set version($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasVersion() => $_has(1);
  @$pb.TagNumber(2)
  void clearVersion() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get tarGz => $_getN(2);
  @$pb.TagNumber(3)
  set tarGz($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasTarGz() => $_has(2);
  @$pb.TagNumber(3)
  void clearTarGz() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get sha256 => $_getSZ(3);
  @$pb.TagNumber(4)
  set sha256($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasSha256() => $_has(3);
  @$pb.TagNumber(4)
  void clearSha256() => $_clearField(4);
}

class RemoveBundle extends $pb.GeneratedMessage {
  factory RemoveBundle({
    $core.String? bundleId,
  }) {
    final result = create();
    if (bundleId != null) result.bundleId = bundleId;
    return result;
  }

  RemoveBundle._();

  factory RemoveBundle.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RemoveBundle.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RemoveBundle',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'terminals.io.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'bundleId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoveBundle clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RemoveBundle copyWith(void Function(RemoveBundle) updates) =>
      super.copyWith((message) => updates(message as RemoveBundle))
          as RemoveBundle;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RemoveBundle create() => RemoveBundle._();
  @$core.override
  RemoveBundle createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RemoveBundle getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RemoveBundle>(create);
  static RemoveBundle? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get bundleId => $_getSZ(0);
  @$pb.TagNumber(1)
  set bundleId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasBundleId() => $_has(0);
  @$pb.TagNumber(1)
  void clearBundleId() => $_clearField(1);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
