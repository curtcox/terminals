// This is a generated file - do not edit.
//
// Generated from terminals/control/v1/control.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:async' as $async;
import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

import '../../capabilities/v1/capabilities.pb.dart' as $3;
import '../../diagnostics/v1/diagnostics.pb.dart' as $1;
import '../../io/v1/io.pb.dart' as $0;
import '../../ui/v1/ui.pb.dart' as $2;
import 'control.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'control.pbenum.dart';

class TransportHello extends $pb.GeneratedMessage {
  factory TransportHello({
    $core.int? protocolVersion,
    $core.Iterable<CarrierKind>? supportedCarriers,
    $core.String? desiredDeviceId,
    $core.String? resumeToken,
  }) {
    final result = create();
    if (protocolVersion != null) result.protocolVersion = protocolVersion;
    if (supportedCarriers != null)
      result.supportedCarriers.addAll(supportedCarriers);
    if (desiredDeviceId != null) result.desiredDeviceId = desiredDeviceId;
    if (resumeToken != null) result.resumeToken = resumeToken;
    return result;
  }

  TransportHello._();

  factory TransportHello.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TransportHello.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TransportHello',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.control.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'protocolVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..pc<CarrierKind>(
        2, _omitFieldNames ? '' : 'supportedCarriers', $pb.PbFieldType.KE,
        valueOf: CarrierKind.valueOf,
        enumValues: CarrierKind.values,
        defaultEnumValue: CarrierKind.CARRIER_KIND_UNSPECIFIED)
    ..aOS(3, _omitFieldNames ? '' : 'desiredDeviceId')
    ..aOS(4, _omitFieldNames ? '' : 'resumeToken')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TransportHello clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TransportHello copyWith(void Function(TransportHello) updates) =>
      super.copyWith((message) => updates(message as TransportHello))
          as TransportHello;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TransportHello create() => TransportHello._();
  @$core.override
  TransportHello createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TransportHello getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TransportHello>(create);
  static TransportHello? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get protocolVersion => $_getIZ(0);
  @$pb.TagNumber(1)
  set protocolVersion($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasProtocolVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearProtocolVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  $pb.PbList<CarrierKind> get supportedCarriers => $_getList(1);

  @$pb.TagNumber(3)
  $core.String get desiredDeviceId => $_getSZ(2);
  @$pb.TagNumber(3)
  set desiredDeviceId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDesiredDeviceId() => $_has(2);
  @$pb.TagNumber(3)
  void clearDesiredDeviceId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get resumeToken => $_getSZ(3);
  @$pb.TagNumber(4)
  set resumeToken($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasResumeToken() => $_has(3);
  @$pb.TagNumber(4)
  void clearResumeToken() => $_clearField(4);
}

class TransportHelloAck extends $pb.GeneratedMessage {
  factory TransportHelloAck({
    $core.int? acceptedProtocolVersion,
    CarrierKind? negotiatedCarrier,
    $core.String? sessionId,
    $core.String? resumeToken,
    $fixnum.Int64? heartbeatIntervalMs,
    $core.Iterable<$core.MapEntry<$core.String, $core.String>>? limits,
  }) {
    final result = create();
    if (acceptedProtocolVersion != null)
      result.acceptedProtocolVersion = acceptedProtocolVersion;
    if (negotiatedCarrier != null) result.negotiatedCarrier = negotiatedCarrier;
    if (sessionId != null) result.sessionId = sessionId;
    if (resumeToken != null) result.resumeToken = resumeToken;
    if (heartbeatIntervalMs != null)
      result.heartbeatIntervalMs = heartbeatIntervalMs;
    if (limits != null) result.limits.addEntries(limits);
    return result;
  }

  TransportHelloAck._();

  factory TransportHelloAck.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TransportHelloAck.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TransportHelloAck',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.control.v1'),
      createEmptyInstance: create)
    ..aI(1, _omitFieldNames ? '' : 'acceptedProtocolVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aE<CarrierKind>(2, _omitFieldNames ? '' : 'negotiatedCarrier',
        enumValues: CarrierKind.values)
    ..aOS(3, _omitFieldNames ? '' : 'sessionId')
    ..aOS(4, _omitFieldNames ? '' : 'resumeToken')
    ..aInt64(5, _omitFieldNames ? '' : 'heartbeatIntervalMs')
    ..m<$core.String, $core.String>(6, _omitFieldNames ? '' : 'limits',
        entryClassName: 'TransportHelloAck.LimitsEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OS,
        packageName: const $pb.PackageName('terminals.control.v1'))
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TransportHelloAck clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TransportHelloAck copyWith(void Function(TransportHelloAck) updates) =>
      super.copyWith((message) => updates(message as TransportHelloAck))
          as TransportHelloAck;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TransportHelloAck create() => TransportHelloAck._();
  @$core.override
  TransportHelloAck createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TransportHelloAck getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TransportHelloAck>(create);
  static TransportHelloAck? _defaultInstance;

  @$pb.TagNumber(1)
  $core.int get acceptedProtocolVersion => $_getIZ(0);
  @$pb.TagNumber(1)
  set acceptedProtocolVersion($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasAcceptedProtocolVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearAcceptedProtocolVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  CarrierKind get negotiatedCarrier => $_getN(1);
  @$pb.TagNumber(2)
  set negotiatedCarrier(CarrierKind value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasNegotiatedCarrier() => $_has(1);
  @$pb.TagNumber(2)
  void clearNegotiatedCarrier() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get sessionId => $_getSZ(2);
  @$pb.TagNumber(3)
  set sessionId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSessionId() => $_has(2);
  @$pb.TagNumber(3)
  void clearSessionId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get resumeToken => $_getSZ(3);
  @$pb.TagNumber(4)
  set resumeToken($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasResumeToken() => $_has(3);
  @$pb.TagNumber(4)
  void clearResumeToken() => $_clearField(4);

  @$pb.TagNumber(5)
  $fixnum.Int64 get heartbeatIntervalMs => $_getI64(4);
  @$pb.TagNumber(5)
  set heartbeatIntervalMs($fixnum.Int64 value) => $_setInt64(4, value);
  @$pb.TagNumber(5)
  $core.bool hasHeartbeatIntervalMs() => $_has(4);
  @$pb.TagNumber(5)
  void clearHeartbeatIntervalMs() => $_clearField(5);

  @$pb.TagNumber(6)
  $pb.PbMap<$core.String, $core.String> get limits => $_getMap(5);
}

class TransportHeartbeat extends $pb.GeneratedMessage {
  factory TransportHeartbeat({
    $fixnum.Int64? unixMs,
  }) {
    final result = create();
    if (unixMs != null) result.unixMs = unixMs;
    return result;
  }

  TransportHeartbeat._();

  factory TransportHeartbeat.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TransportHeartbeat.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TransportHeartbeat',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.control.v1'),
      createEmptyInstance: create)
    ..aInt64(1, _omitFieldNames ? '' : 'unixMs')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TransportHeartbeat clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TransportHeartbeat copyWith(void Function(TransportHeartbeat) updates) =>
      super.copyWith((message) => updates(message as TransportHeartbeat))
          as TransportHeartbeat;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TransportHeartbeat create() => TransportHeartbeat._();
  @$core.override
  TransportHeartbeat createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TransportHeartbeat getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TransportHeartbeat>(create);
  static TransportHeartbeat? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get unixMs => $_getI64(0);
  @$pb.TagNumber(1)
  set unixMs($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasUnixMs() => $_has(0);
  @$pb.TagNumber(1)
  void clearUnixMs() => $_clearField(1);
}

class TransportError extends $pb.GeneratedMessage {
  factory TransportError({
    $core.String? code,
    $core.String? message,
  }) {
    final result = create();
    if (code != null) result.code = code;
    if (message != null) result.message = message;
    return result;
  }

  TransportError._();

  factory TransportError.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory TransportError.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'TransportError',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.control.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'code')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TransportError clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  TransportError copyWith(void Function(TransportError) updates) =>
      super.copyWith((message) => updates(message as TransportError))
          as TransportError;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static TransportError create() => TransportError._();
  @$core.override
  TransportError createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static TransportError getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<TransportError>(create);
  static TransportError? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get code => $_getSZ(0);
  @$pb.TagNumber(1)
  set code($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasCode() => $_has(0);
  @$pb.TagNumber(1)
  void clearCode() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => $_clearField(2);
}

enum WireEnvelope_Payload {
  clientMessage,
  serverMessage,
  transportHello,
  transportHelloAck,
  transportHeartbeat,
  transportError,
  notSet
}

class WireEnvelope extends $pb.GeneratedMessage {
  factory WireEnvelope({
    $core.int? protocolVersion,
    $core.String? sessionId,
    $fixnum.Int64? sequence,
    ConnectRequest? clientMessage,
    ConnectResponse? serverMessage,
    TransportHello? transportHello,
    TransportHelloAck? transportHelloAck,
    TransportHeartbeat? transportHeartbeat,
    TransportError? transportError,
  }) {
    final result = create();
    if (protocolVersion != null) result.protocolVersion = protocolVersion;
    if (sessionId != null) result.sessionId = sessionId;
    if (sequence != null) result.sequence = sequence;
    if (clientMessage != null) result.clientMessage = clientMessage;
    if (serverMessage != null) result.serverMessage = serverMessage;
    if (transportHello != null) result.transportHello = transportHello;
    if (transportHelloAck != null) result.transportHelloAck = transportHelloAck;
    if (transportHeartbeat != null)
      result.transportHeartbeat = transportHeartbeat;
    if (transportError != null) result.transportError = transportError;
    return result;
  }

  WireEnvelope._();

  factory WireEnvelope.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory WireEnvelope.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, WireEnvelope_Payload>
      _WireEnvelope_PayloadByTag = {
    10: WireEnvelope_Payload.clientMessage,
    11: WireEnvelope_Payload.serverMessage,
    12: WireEnvelope_Payload.transportHello,
    13: WireEnvelope_Payload.transportHelloAck,
    14: WireEnvelope_Payload.transportHeartbeat,
    15: WireEnvelope_Payload.transportError,
    0: WireEnvelope_Payload.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'WireEnvelope',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.control.v1'),
      createEmptyInstance: create)
    ..oo(0, [10, 11, 12, 13, 14, 15])
    ..aI(1, _omitFieldNames ? '' : 'protocolVersion',
        fieldType: $pb.PbFieldType.OU3)
    ..aOS(2, _omitFieldNames ? '' : 'sessionId')
    ..a<$fixnum.Int64>(
        3, _omitFieldNames ? '' : 'sequence', $pb.PbFieldType.OU6,
        defaultOrMaker: $fixnum.Int64.ZERO)
    ..aOM<ConnectRequest>(10, _omitFieldNames ? '' : 'clientMessage',
        subBuilder: ConnectRequest.create)
    ..aOM<ConnectResponse>(11, _omitFieldNames ? '' : 'serverMessage',
        subBuilder: ConnectResponse.create)
    ..aOM<TransportHello>(12, _omitFieldNames ? '' : 'transportHello',
        subBuilder: TransportHello.create)
    ..aOM<TransportHelloAck>(13, _omitFieldNames ? '' : 'transportHelloAck',
        subBuilder: TransportHelloAck.create)
    ..aOM<TransportHeartbeat>(14, _omitFieldNames ? '' : 'transportHeartbeat',
        subBuilder: TransportHeartbeat.create)
    ..aOM<TransportError>(15, _omitFieldNames ? '' : 'transportError',
        subBuilder: TransportError.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WireEnvelope clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WireEnvelope copyWith(void Function(WireEnvelope) updates) =>
      super.copyWith((message) => updates(message as WireEnvelope))
          as WireEnvelope;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static WireEnvelope create() => WireEnvelope._();
  @$core.override
  WireEnvelope createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static WireEnvelope getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<WireEnvelope>(create);
  static WireEnvelope? _defaultInstance;

  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(12)
  @$pb.TagNumber(13)
  @$pb.TagNumber(14)
  @$pb.TagNumber(15)
  WireEnvelope_Payload whichPayload() =>
      _WireEnvelope_PayloadByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(12)
  @$pb.TagNumber(13)
  @$pb.TagNumber(14)
  @$pb.TagNumber(15)
  void clearPayload() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  $core.int get protocolVersion => $_getIZ(0);
  @$pb.TagNumber(1)
  set protocolVersion($core.int value) => $_setUnsignedInt32(0, value);
  @$pb.TagNumber(1)
  $core.bool hasProtocolVersion() => $_has(0);
  @$pb.TagNumber(1)
  void clearProtocolVersion() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get sessionId => $_getSZ(1);
  @$pb.TagNumber(2)
  set sessionId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSessionId() => $_has(1);
  @$pb.TagNumber(2)
  void clearSessionId() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get sequence => $_getI64(2);
  @$pb.TagNumber(3)
  set sequence($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSequence() => $_has(2);
  @$pb.TagNumber(3)
  void clearSequence() => $_clearField(3);

  @$pb.TagNumber(10)
  ConnectRequest get clientMessage => $_getN(3);
  @$pb.TagNumber(10)
  set clientMessage(ConnectRequest value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasClientMessage() => $_has(3);
  @$pb.TagNumber(10)
  void clearClientMessage() => $_clearField(10);
  @$pb.TagNumber(10)
  ConnectRequest ensureClientMessage() => $_ensure(3);

  @$pb.TagNumber(11)
  ConnectResponse get serverMessage => $_getN(4);
  @$pb.TagNumber(11)
  set serverMessage(ConnectResponse value) => $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasServerMessage() => $_has(4);
  @$pb.TagNumber(11)
  void clearServerMessage() => $_clearField(11);
  @$pb.TagNumber(11)
  ConnectResponse ensureServerMessage() => $_ensure(4);

  @$pb.TagNumber(12)
  TransportHello get transportHello => $_getN(5);
  @$pb.TagNumber(12)
  set transportHello(TransportHello value) => $_setField(12, value);
  @$pb.TagNumber(12)
  $core.bool hasTransportHello() => $_has(5);
  @$pb.TagNumber(12)
  void clearTransportHello() => $_clearField(12);
  @$pb.TagNumber(12)
  TransportHello ensureTransportHello() => $_ensure(5);

  @$pb.TagNumber(13)
  TransportHelloAck get transportHelloAck => $_getN(6);
  @$pb.TagNumber(13)
  set transportHelloAck(TransportHelloAck value) => $_setField(13, value);
  @$pb.TagNumber(13)
  $core.bool hasTransportHelloAck() => $_has(6);
  @$pb.TagNumber(13)
  void clearTransportHelloAck() => $_clearField(13);
  @$pb.TagNumber(13)
  TransportHelloAck ensureTransportHelloAck() => $_ensure(6);

  @$pb.TagNumber(14)
  TransportHeartbeat get transportHeartbeat => $_getN(7);
  @$pb.TagNumber(14)
  set transportHeartbeat(TransportHeartbeat value) => $_setField(14, value);
  @$pb.TagNumber(14)
  $core.bool hasTransportHeartbeat() => $_has(7);
  @$pb.TagNumber(14)
  void clearTransportHeartbeat() => $_clearField(14);
  @$pb.TagNumber(14)
  TransportHeartbeat ensureTransportHeartbeat() => $_ensure(7);

  @$pb.TagNumber(15)
  TransportError get transportError => $_getN(8);
  @$pb.TagNumber(15)
  set transportError(TransportError value) => $_setField(15, value);
  @$pb.TagNumber(15)
  $core.bool hasTransportError() => $_has(8);
  @$pb.TagNumber(15)
  void clearTransportError() => $_clearField(15);
  @$pb.TagNumber(15)
  TransportError ensureTransportError() => $_ensure(8);
}

enum ConnectRequest_Payload {
  register,
  capability,
  input,
  sensor,
  streamReady,
  command,
  heartbeat,
  webrtcSignal,
  voiceAudio,
  observationMessage,
  artifactAvailable,
  flowStats,
  clockSample,
  bugReport,
  notSet
}

class ConnectRequest extends $pb.GeneratedMessage {
  factory ConnectRequest({
    RegisterDevice? register,
    CapabilityUpdate? capability,
    $0.InputEvent? input,
    $0.SensorData? sensor,
    StreamReady? streamReady,
    CommandRequest? command,
    Heartbeat? heartbeat,
    WebRTCSignal? webrtcSignal,
    VoiceAudio? voiceAudio,
    $0.ObservationMessage? observationMessage,
    $0.ArtifactAvailable? artifactAvailable,
    $0.FlowStats? flowStats,
    $0.ClockSample? clockSample,
    $1.BugReport? bugReport,
  }) {
    final result = create();
    if (register != null) result.register = register;
    if (capability != null) result.capability = capability;
    if (input != null) result.input = input;
    if (sensor != null) result.sensor = sensor;
    if (streamReady != null) result.streamReady = streamReady;
    if (command != null) result.command = command;
    if (heartbeat != null) result.heartbeat = heartbeat;
    if (webrtcSignal != null) result.webrtcSignal = webrtcSignal;
    if (voiceAudio != null) result.voiceAudio = voiceAudio;
    if (observationMessage != null)
      result.observationMessage = observationMessage;
    if (artifactAvailable != null) result.artifactAvailable = artifactAvailable;
    if (flowStats != null) result.flowStats = flowStats;
    if (clockSample != null) result.clockSample = clockSample;
    if (bugReport != null) result.bugReport = bugReport;
    return result;
  }

  ConnectRequest._();

  factory ConnectRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ConnectRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, ConnectRequest_Payload>
      _ConnectRequest_PayloadByTag = {
    1: ConnectRequest_Payload.register,
    2: ConnectRequest_Payload.capability,
    3: ConnectRequest_Payload.input,
    4: ConnectRequest_Payload.sensor,
    5: ConnectRequest_Payload.streamReady,
    6: ConnectRequest_Payload.command,
    7: ConnectRequest_Payload.heartbeat,
    8: ConnectRequest_Payload.webrtcSignal,
    9: ConnectRequest_Payload.voiceAudio,
    10: ConnectRequest_Payload.observationMessage,
    11: ConnectRequest_Payload.artifactAvailable,
    12: ConnectRequest_Payload.flowStats,
    13: ConnectRequest_Payload.clockSample,
    14: ConnectRequest_Payload.bugReport,
    0: ConnectRequest_Payload.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ConnectRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.control.v1'),
      createEmptyInstance: create)
    ..oo(0, [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14])
    ..aOM<RegisterDevice>(1, _omitFieldNames ? '' : 'register',
        subBuilder: RegisterDevice.create)
    ..aOM<CapabilityUpdate>(2, _omitFieldNames ? '' : 'capability',
        subBuilder: CapabilityUpdate.create)
    ..aOM<$0.InputEvent>(3, _omitFieldNames ? '' : 'input',
        subBuilder: $0.InputEvent.create)
    ..aOM<$0.SensorData>(4, _omitFieldNames ? '' : 'sensor',
        subBuilder: $0.SensorData.create)
    ..aOM<StreamReady>(5, _omitFieldNames ? '' : 'streamReady',
        subBuilder: StreamReady.create)
    ..aOM<CommandRequest>(6, _omitFieldNames ? '' : 'command',
        subBuilder: CommandRequest.create)
    ..aOM<Heartbeat>(7, _omitFieldNames ? '' : 'heartbeat',
        subBuilder: Heartbeat.create)
    ..aOM<WebRTCSignal>(8, _omitFieldNames ? '' : 'webrtcSignal',
        subBuilder: WebRTCSignal.create)
    ..aOM<VoiceAudio>(9, _omitFieldNames ? '' : 'voiceAudio',
        subBuilder: VoiceAudio.create)
    ..aOM<$0.ObservationMessage>(
        10, _omitFieldNames ? '' : 'observationMessage',
        subBuilder: $0.ObservationMessage.create)
    ..aOM<$0.ArtifactAvailable>(11, _omitFieldNames ? '' : 'artifactAvailable',
        subBuilder: $0.ArtifactAvailable.create)
    ..aOM<$0.FlowStats>(12, _omitFieldNames ? '' : 'flowStats',
        subBuilder: $0.FlowStats.create)
    ..aOM<$0.ClockSample>(13, _omitFieldNames ? '' : 'clockSample',
        subBuilder: $0.ClockSample.create)
    ..aOM<$1.BugReport>(14, _omitFieldNames ? '' : 'bugReport',
        subBuilder: $1.BugReport.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectRequest copyWith(void Function(ConnectRequest) updates) =>
      super.copyWith((message) => updates(message as ConnectRequest))
          as ConnectRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ConnectRequest create() => ConnectRequest._();
  @$core.override
  ConnectRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ConnectRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ConnectRequest>(create);
  static ConnectRequest? _defaultInstance;

  @$pb.TagNumber(1)
  @$pb.TagNumber(2)
  @$pb.TagNumber(3)
  @$pb.TagNumber(4)
  @$pb.TagNumber(5)
  @$pb.TagNumber(6)
  @$pb.TagNumber(7)
  @$pb.TagNumber(8)
  @$pb.TagNumber(9)
  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(12)
  @$pb.TagNumber(13)
  @$pb.TagNumber(14)
  ConnectRequest_Payload whichPayload() =>
      _ConnectRequest_PayloadByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(1)
  @$pb.TagNumber(2)
  @$pb.TagNumber(3)
  @$pb.TagNumber(4)
  @$pb.TagNumber(5)
  @$pb.TagNumber(6)
  @$pb.TagNumber(7)
  @$pb.TagNumber(8)
  @$pb.TagNumber(9)
  @$pb.TagNumber(10)
  @$pb.TagNumber(11)
  @$pb.TagNumber(12)
  @$pb.TagNumber(13)
  @$pb.TagNumber(14)
  void clearPayload() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  RegisterDevice get register => $_getN(0);
  @$pb.TagNumber(1)
  set register(RegisterDevice value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasRegister() => $_has(0);
  @$pb.TagNumber(1)
  void clearRegister() => $_clearField(1);
  @$pb.TagNumber(1)
  RegisterDevice ensureRegister() => $_ensure(0);

  @$pb.TagNumber(2)
  CapabilityUpdate get capability => $_getN(1);
  @$pb.TagNumber(2)
  set capability(CapabilityUpdate value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasCapability() => $_has(1);
  @$pb.TagNumber(2)
  void clearCapability() => $_clearField(2);
  @$pb.TagNumber(2)
  CapabilityUpdate ensureCapability() => $_ensure(1);

  @$pb.TagNumber(3)
  $0.InputEvent get input => $_getN(2);
  @$pb.TagNumber(3)
  set input($0.InputEvent value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasInput() => $_has(2);
  @$pb.TagNumber(3)
  void clearInput() => $_clearField(3);
  @$pb.TagNumber(3)
  $0.InputEvent ensureInput() => $_ensure(2);

  @$pb.TagNumber(4)
  $0.SensorData get sensor => $_getN(3);
  @$pb.TagNumber(4)
  set sensor($0.SensorData value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasSensor() => $_has(3);
  @$pb.TagNumber(4)
  void clearSensor() => $_clearField(4);
  @$pb.TagNumber(4)
  $0.SensorData ensureSensor() => $_ensure(3);

  @$pb.TagNumber(5)
  StreamReady get streamReady => $_getN(4);
  @$pb.TagNumber(5)
  set streamReady(StreamReady value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasStreamReady() => $_has(4);
  @$pb.TagNumber(5)
  void clearStreamReady() => $_clearField(5);
  @$pb.TagNumber(5)
  StreamReady ensureStreamReady() => $_ensure(4);

  @$pb.TagNumber(6)
  CommandRequest get command => $_getN(5);
  @$pb.TagNumber(6)
  set command(CommandRequest value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasCommand() => $_has(5);
  @$pb.TagNumber(6)
  void clearCommand() => $_clearField(6);
  @$pb.TagNumber(6)
  CommandRequest ensureCommand() => $_ensure(5);

  @$pb.TagNumber(7)
  Heartbeat get heartbeat => $_getN(6);
  @$pb.TagNumber(7)
  set heartbeat(Heartbeat value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasHeartbeat() => $_has(6);
  @$pb.TagNumber(7)
  void clearHeartbeat() => $_clearField(7);
  @$pb.TagNumber(7)
  Heartbeat ensureHeartbeat() => $_ensure(6);

  @$pb.TagNumber(8)
  WebRTCSignal get webrtcSignal => $_getN(7);
  @$pb.TagNumber(8)
  set webrtcSignal(WebRTCSignal value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasWebrtcSignal() => $_has(7);
  @$pb.TagNumber(8)
  void clearWebrtcSignal() => $_clearField(8);
  @$pb.TagNumber(8)
  WebRTCSignal ensureWebrtcSignal() => $_ensure(7);

  @$pb.TagNumber(9)
  VoiceAudio get voiceAudio => $_getN(8);
  @$pb.TagNumber(9)
  set voiceAudio(VoiceAudio value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasVoiceAudio() => $_has(8);
  @$pb.TagNumber(9)
  void clearVoiceAudio() => $_clearField(9);
  @$pb.TagNumber(9)
  VoiceAudio ensureVoiceAudio() => $_ensure(8);

  @$pb.TagNumber(10)
  $0.ObservationMessage get observationMessage => $_getN(9);
  @$pb.TagNumber(10)
  set observationMessage($0.ObservationMessage value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasObservationMessage() => $_has(9);
  @$pb.TagNumber(10)
  void clearObservationMessage() => $_clearField(10);
  @$pb.TagNumber(10)
  $0.ObservationMessage ensureObservationMessage() => $_ensure(9);

  @$pb.TagNumber(11)
  $0.ArtifactAvailable get artifactAvailable => $_getN(10);
  @$pb.TagNumber(11)
  set artifactAvailable($0.ArtifactAvailable value) => $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasArtifactAvailable() => $_has(10);
  @$pb.TagNumber(11)
  void clearArtifactAvailable() => $_clearField(11);
  @$pb.TagNumber(11)
  $0.ArtifactAvailable ensureArtifactAvailable() => $_ensure(10);

  @$pb.TagNumber(12)
  $0.FlowStats get flowStats => $_getN(11);
  @$pb.TagNumber(12)
  set flowStats($0.FlowStats value) => $_setField(12, value);
  @$pb.TagNumber(12)
  $core.bool hasFlowStats() => $_has(11);
  @$pb.TagNumber(12)
  void clearFlowStats() => $_clearField(12);
  @$pb.TagNumber(12)
  $0.FlowStats ensureFlowStats() => $_ensure(11);

  @$pb.TagNumber(13)
  $0.ClockSample get clockSample => $_getN(12);
  @$pb.TagNumber(13)
  set clockSample($0.ClockSample value) => $_setField(13, value);
  @$pb.TagNumber(13)
  $core.bool hasClockSample() => $_has(12);
  @$pb.TagNumber(13)
  void clearClockSample() => $_clearField(13);
  @$pb.TagNumber(13)
  $0.ClockSample ensureClockSample() => $_ensure(12);

  @$pb.TagNumber(14)
  $1.BugReport get bugReport => $_getN(13);
  @$pb.TagNumber(14)
  set bugReport($1.BugReport value) => $_setField(14, value);
  @$pb.TagNumber(14)
  $core.bool hasBugReport() => $_has(13);
  @$pb.TagNumber(14)
  void clearBugReport() => $_clearField(14);
  @$pb.TagNumber(14)
  $1.BugReport ensureBugReport() => $_ensure(13);
}

class VoiceAudio extends $pb.GeneratedMessage {
  factory VoiceAudio({
    $core.String? deviceId,
    $core.List<$core.int>? audio,
    $core.int? sampleRate,
    $core.bool? isFinal,
  }) {
    final result = create();
    if (deviceId != null) result.deviceId = deviceId;
    if (audio != null) result.audio = audio;
    if (sampleRate != null) result.sampleRate = sampleRate;
    if (isFinal != null) result.isFinal = isFinal;
    return result;
  }

  VoiceAudio._();

  factory VoiceAudio.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory VoiceAudio.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'VoiceAudio',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.control.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'deviceId')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'audio', $pb.PbFieldType.OY)
    ..aI(3, _omitFieldNames ? '' : 'sampleRate')
    ..aOB(4, _omitFieldNames ? '' : 'isFinal')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  VoiceAudio clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  VoiceAudio copyWith(void Function(VoiceAudio) updates) =>
      super.copyWith((message) => updates(message as VoiceAudio)) as VoiceAudio;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static VoiceAudio create() => VoiceAudio._();
  @$core.override
  VoiceAudio createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static VoiceAudio getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<VoiceAudio>(create);
  static VoiceAudio? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get deviceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set deviceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDeviceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDeviceId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get audio => $_getN(1);
  @$pb.TagNumber(2)
  set audio($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasAudio() => $_has(1);
  @$pb.TagNumber(2)
  void clearAudio() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get sampleRate => $_getIZ(2);
  @$pb.TagNumber(3)
  set sampleRate($core.int value) => $_setSignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSampleRate() => $_has(2);
  @$pb.TagNumber(3)
  void clearSampleRate() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.bool get isFinal => $_getBF(3);
  @$pb.TagNumber(4)
  set isFinal($core.bool value) => $_setBool(3, value);
  @$pb.TagNumber(4)
  $core.bool hasIsFinal() => $_has(3);
  @$pb.TagNumber(4)
  void clearIsFinal() => $_clearField(4);
}

enum ConnectResponse_Payload {
  registerAck,
  setUi,
  startStream,
  stopStream,
  playAudio,
  showMedia,
  routeStream,
  notification,
  webrtcSignal,
  commandResult,
  heartbeat,
  error,
  updateUi,
  transitionUi,
  installBundle,
  removeBundle,
  startFlow,
  patchFlow,
  stopFlow,
  requestArtifact,
  bugReportAck,
  notSet
}

class ConnectResponse extends $pb.GeneratedMessage {
  factory ConnectResponse({
    RegisterAck? registerAck,
    $2.SetUI? setUi,
    $0.StartStream? startStream,
    $0.StopStream? stopStream,
    $0.PlayAudio? playAudio,
    $0.ShowMedia? showMedia,
    $0.RouteStream? routeStream,
    $2.Notification? notification,
    WebRTCSignal? webrtcSignal,
    CommandResult? commandResult,
    Heartbeat? heartbeat,
    ControlError? error,
    $2.UpdateUI? updateUi,
    $2.TransitionUI? transitionUi,
    $0.InstallBundle? installBundle,
    $0.RemoveBundle? removeBundle,
    $0.StartFlow? startFlow,
    $0.PatchFlow? patchFlow,
    $0.StopFlow? stopFlow,
    $0.RequestArtifact? requestArtifact,
    $1.BugReportAck? bugReportAck,
  }) {
    final result = create();
    if (registerAck != null) result.registerAck = registerAck;
    if (setUi != null) result.setUi = setUi;
    if (startStream != null) result.startStream = startStream;
    if (stopStream != null) result.stopStream = stopStream;
    if (playAudio != null) result.playAudio = playAudio;
    if (showMedia != null) result.showMedia = showMedia;
    if (routeStream != null) result.routeStream = routeStream;
    if (notification != null) result.notification = notification;
    if (webrtcSignal != null) result.webrtcSignal = webrtcSignal;
    if (commandResult != null) result.commandResult = commandResult;
    if (heartbeat != null) result.heartbeat = heartbeat;
    if (error != null) result.error = error;
    if (updateUi != null) result.updateUi = updateUi;
    if (transitionUi != null) result.transitionUi = transitionUi;
    if (installBundle != null) result.installBundle = installBundle;
    if (removeBundle != null) result.removeBundle = removeBundle;
    if (startFlow != null) result.startFlow = startFlow;
    if (patchFlow != null) result.patchFlow = patchFlow;
    if (stopFlow != null) result.stopFlow = stopFlow;
    if (requestArtifact != null) result.requestArtifact = requestArtifact;
    if (bugReportAck != null) result.bugReportAck = bugReportAck;
    return result;
  }

  ConnectResponse._();

  factory ConnectResponse.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ConnectResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static const $core.Map<$core.int, ConnectResponse_Payload>
      _ConnectResponse_PayloadByTag = {
    1: ConnectResponse_Payload.registerAck,
    2: ConnectResponse_Payload.setUi,
    3: ConnectResponse_Payload.startStream,
    4: ConnectResponse_Payload.stopStream,
    5: ConnectResponse_Payload.playAudio,
    6: ConnectResponse_Payload.showMedia,
    7: ConnectResponse_Payload.routeStream,
    8: ConnectResponse_Payload.notification,
    9: ConnectResponse_Payload.webrtcSignal,
    10: ConnectResponse_Payload.commandResult,
    11: ConnectResponse_Payload.heartbeat,
    12: ConnectResponse_Payload.error,
    13: ConnectResponse_Payload.updateUi,
    14: ConnectResponse_Payload.transitionUi,
    15: ConnectResponse_Payload.installBundle,
    16: ConnectResponse_Payload.removeBundle,
    17: ConnectResponse_Payload.startFlow,
    18: ConnectResponse_Payload.patchFlow,
    19: ConnectResponse_Payload.stopFlow,
    20: ConnectResponse_Payload.requestArtifact,
    21: ConnectResponse_Payload.bugReportAck,
    0: ConnectResponse_Payload.notSet
  };
  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ConnectResponse',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.control.v1'),
      createEmptyInstance: create)
    ..oo(0, [
      1,
      2,
      3,
      4,
      5,
      6,
      7,
      8,
      9,
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
      21
    ])
    ..aOM<RegisterAck>(1, _omitFieldNames ? '' : 'registerAck',
        subBuilder: RegisterAck.create)
    ..aOM<$2.SetUI>(2, _omitFieldNames ? '' : 'setUi',
        subBuilder: $2.SetUI.create)
    ..aOM<$0.StartStream>(3, _omitFieldNames ? '' : 'startStream',
        subBuilder: $0.StartStream.create)
    ..aOM<$0.StopStream>(4, _omitFieldNames ? '' : 'stopStream',
        subBuilder: $0.StopStream.create)
    ..aOM<$0.PlayAudio>(5, _omitFieldNames ? '' : 'playAudio',
        subBuilder: $0.PlayAudio.create)
    ..aOM<$0.ShowMedia>(6, _omitFieldNames ? '' : 'showMedia',
        subBuilder: $0.ShowMedia.create)
    ..aOM<$0.RouteStream>(7, _omitFieldNames ? '' : 'routeStream',
        subBuilder: $0.RouteStream.create)
    ..aOM<$2.Notification>(8, _omitFieldNames ? '' : 'notification',
        subBuilder: $2.Notification.create)
    ..aOM<WebRTCSignal>(9, _omitFieldNames ? '' : 'webrtcSignal',
        subBuilder: WebRTCSignal.create)
    ..aOM<CommandResult>(10, _omitFieldNames ? '' : 'commandResult',
        subBuilder: CommandResult.create)
    ..aOM<Heartbeat>(11, _omitFieldNames ? '' : 'heartbeat',
        subBuilder: Heartbeat.create)
    ..aOM<ControlError>(12, _omitFieldNames ? '' : 'error',
        subBuilder: ControlError.create)
    ..aOM<$2.UpdateUI>(13, _omitFieldNames ? '' : 'updateUi',
        subBuilder: $2.UpdateUI.create)
    ..aOM<$2.TransitionUI>(14, _omitFieldNames ? '' : 'transitionUi',
        subBuilder: $2.TransitionUI.create)
    ..aOM<$0.InstallBundle>(15, _omitFieldNames ? '' : 'installBundle',
        subBuilder: $0.InstallBundle.create)
    ..aOM<$0.RemoveBundle>(16, _omitFieldNames ? '' : 'removeBundle',
        subBuilder: $0.RemoveBundle.create)
    ..aOM<$0.StartFlow>(17, _omitFieldNames ? '' : 'startFlow',
        subBuilder: $0.StartFlow.create)
    ..aOM<$0.PatchFlow>(18, _omitFieldNames ? '' : 'patchFlow',
        subBuilder: $0.PatchFlow.create)
    ..aOM<$0.StopFlow>(19, _omitFieldNames ? '' : 'stopFlow',
        subBuilder: $0.StopFlow.create)
    ..aOM<$0.RequestArtifact>(20, _omitFieldNames ? '' : 'requestArtifact',
        subBuilder: $0.RequestArtifact.create)
    ..aOM<$1.BugReportAck>(21, _omitFieldNames ? '' : 'bugReportAck',
        subBuilder: $1.BugReportAck.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectResponse copyWith(void Function(ConnectResponse) updates) =>
      super.copyWith((message) => updates(message as ConnectResponse))
          as ConnectResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ConnectResponse create() => ConnectResponse._();
  @$core.override
  ConnectResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ConnectResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ConnectResponse>(create);
  static ConnectResponse? _defaultInstance;

  @$pb.TagNumber(1)
  @$pb.TagNumber(2)
  @$pb.TagNumber(3)
  @$pb.TagNumber(4)
  @$pb.TagNumber(5)
  @$pb.TagNumber(6)
  @$pb.TagNumber(7)
  @$pb.TagNumber(8)
  @$pb.TagNumber(9)
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
  ConnectResponse_Payload whichPayload() =>
      _ConnectResponse_PayloadByTag[$_whichOneof(0)]!;
  @$pb.TagNumber(1)
  @$pb.TagNumber(2)
  @$pb.TagNumber(3)
  @$pb.TagNumber(4)
  @$pb.TagNumber(5)
  @$pb.TagNumber(6)
  @$pb.TagNumber(7)
  @$pb.TagNumber(8)
  @$pb.TagNumber(9)
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
  void clearPayload() => $_clearField($_whichOneof(0));

  @$pb.TagNumber(1)
  RegisterAck get registerAck => $_getN(0);
  @$pb.TagNumber(1)
  set registerAck(RegisterAck value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasRegisterAck() => $_has(0);
  @$pb.TagNumber(1)
  void clearRegisterAck() => $_clearField(1);
  @$pb.TagNumber(1)
  RegisterAck ensureRegisterAck() => $_ensure(0);

  @$pb.TagNumber(2)
  $2.SetUI get setUi => $_getN(1);
  @$pb.TagNumber(2)
  set setUi($2.SetUI value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasSetUi() => $_has(1);
  @$pb.TagNumber(2)
  void clearSetUi() => $_clearField(2);
  @$pb.TagNumber(2)
  $2.SetUI ensureSetUi() => $_ensure(1);

  @$pb.TagNumber(3)
  $0.StartStream get startStream => $_getN(2);
  @$pb.TagNumber(3)
  set startStream($0.StartStream value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasStartStream() => $_has(2);
  @$pb.TagNumber(3)
  void clearStartStream() => $_clearField(3);
  @$pb.TagNumber(3)
  $0.StartStream ensureStartStream() => $_ensure(2);

  @$pb.TagNumber(4)
  $0.StopStream get stopStream => $_getN(3);
  @$pb.TagNumber(4)
  set stopStream($0.StopStream value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasStopStream() => $_has(3);
  @$pb.TagNumber(4)
  void clearStopStream() => $_clearField(4);
  @$pb.TagNumber(4)
  $0.StopStream ensureStopStream() => $_ensure(3);

  @$pb.TagNumber(5)
  $0.PlayAudio get playAudio => $_getN(4);
  @$pb.TagNumber(5)
  set playAudio($0.PlayAudio value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasPlayAudio() => $_has(4);
  @$pb.TagNumber(5)
  void clearPlayAudio() => $_clearField(5);
  @$pb.TagNumber(5)
  $0.PlayAudio ensurePlayAudio() => $_ensure(4);

  @$pb.TagNumber(6)
  $0.ShowMedia get showMedia => $_getN(5);
  @$pb.TagNumber(6)
  set showMedia($0.ShowMedia value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasShowMedia() => $_has(5);
  @$pb.TagNumber(6)
  void clearShowMedia() => $_clearField(6);
  @$pb.TagNumber(6)
  $0.ShowMedia ensureShowMedia() => $_ensure(5);

  @$pb.TagNumber(7)
  $0.RouteStream get routeStream => $_getN(6);
  @$pb.TagNumber(7)
  set routeStream($0.RouteStream value) => $_setField(7, value);
  @$pb.TagNumber(7)
  $core.bool hasRouteStream() => $_has(6);
  @$pb.TagNumber(7)
  void clearRouteStream() => $_clearField(7);
  @$pb.TagNumber(7)
  $0.RouteStream ensureRouteStream() => $_ensure(6);

  @$pb.TagNumber(8)
  $2.Notification get notification => $_getN(7);
  @$pb.TagNumber(8)
  set notification($2.Notification value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasNotification() => $_has(7);
  @$pb.TagNumber(8)
  void clearNotification() => $_clearField(8);
  @$pb.TagNumber(8)
  $2.Notification ensureNotification() => $_ensure(7);

  @$pb.TagNumber(9)
  WebRTCSignal get webrtcSignal => $_getN(8);
  @$pb.TagNumber(9)
  set webrtcSignal(WebRTCSignal value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasWebrtcSignal() => $_has(8);
  @$pb.TagNumber(9)
  void clearWebrtcSignal() => $_clearField(9);
  @$pb.TagNumber(9)
  WebRTCSignal ensureWebrtcSignal() => $_ensure(8);

  @$pb.TagNumber(10)
  CommandResult get commandResult => $_getN(9);
  @$pb.TagNumber(10)
  set commandResult(CommandResult value) => $_setField(10, value);
  @$pb.TagNumber(10)
  $core.bool hasCommandResult() => $_has(9);
  @$pb.TagNumber(10)
  void clearCommandResult() => $_clearField(10);
  @$pb.TagNumber(10)
  CommandResult ensureCommandResult() => $_ensure(9);

  @$pb.TagNumber(11)
  Heartbeat get heartbeat => $_getN(10);
  @$pb.TagNumber(11)
  set heartbeat(Heartbeat value) => $_setField(11, value);
  @$pb.TagNumber(11)
  $core.bool hasHeartbeat() => $_has(10);
  @$pb.TagNumber(11)
  void clearHeartbeat() => $_clearField(11);
  @$pb.TagNumber(11)
  Heartbeat ensureHeartbeat() => $_ensure(10);

  @$pb.TagNumber(12)
  ControlError get error => $_getN(11);
  @$pb.TagNumber(12)
  set error(ControlError value) => $_setField(12, value);
  @$pb.TagNumber(12)
  $core.bool hasError() => $_has(11);
  @$pb.TagNumber(12)
  void clearError() => $_clearField(12);
  @$pb.TagNumber(12)
  ControlError ensureError() => $_ensure(11);

  @$pb.TagNumber(13)
  $2.UpdateUI get updateUi => $_getN(12);
  @$pb.TagNumber(13)
  set updateUi($2.UpdateUI value) => $_setField(13, value);
  @$pb.TagNumber(13)
  $core.bool hasUpdateUi() => $_has(12);
  @$pb.TagNumber(13)
  void clearUpdateUi() => $_clearField(13);
  @$pb.TagNumber(13)
  $2.UpdateUI ensureUpdateUi() => $_ensure(12);

  @$pb.TagNumber(14)
  $2.TransitionUI get transitionUi => $_getN(13);
  @$pb.TagNumber(14)
  set transitionUi($2.TransitionUI value) => $_setField(14, value);
  @$pb.TagNumber(14)
  $core.bool hasTransitionUi() => $_has(13);
  @$pb.TagNumber(14)
  void clearTransitionUi() => $_clearField(14);
  @$pb.TagNumber(14)
  $2.TransitionUI ensureTransitionUi() => $_ensure(13);

  @$pb.TagNumber(15)
  $0.InstallBundle get installBundle => $_getN(14);
  @$pb.TagNumber(15)
  set installBundle($0.InstallBundle value) => $_setField(15, value);
  @$pb.TagNumber(15)
  $core.bool hasInstallBundle() => $_has(14);
  @$pb.TagNumber(15)
  void clearInstallBundle() => $_clearField(15);
  @$pb.TagNumber(15)
  $0.InstallBundle ensureInstallBundle() => $_ensure(14);

  @$pb.TagNumber(16)
  $0.RemoveBundle get removeBundle => $_getN(15);
  @$pb.TagNumber(16)
  set removeBundle($0.RemoveBundle value) => $_setField(16, value);
  @$pb.TagNumber(16)
  $core.bool hasRemoveBundle() => $_has(15);
  @$pb.TagNumber(16)
  void clearRemoveBundle() => $_clearField(16);
  @$pb.TagNumber(16)
  $0.RemoveBundle ensureRemoveBundle() => $_ensure(15);

  @$pb.TagNumber(17)
  $0.StartFlow get startFlow => $_getN(16);
  @$pb.TagNumber(17)
  set startFlow($0.StartFlow value) => $_setField(17, value);
  @$pb.TagNumber(17)
  $core.bool hasStartFlow() => $_has(16);
  @$pb.TagNumber(17)
  void clearStartFlow() => $_clearField(17);
  @$pb.TagNumber(17)
  $0.StartFlow ensureStartFlow() => $_ensure(16);

  @$pb.TagNumber(18)
  $0.PatchFlow get patchFlow => $_getN(17);
  @$pb.TagNumber(18)
  set patchFlow($0.PatchFlow value) => $_setField(18, value);
  @$pb.TagNumber(18)
  $core.bool hasPatchFlow() => $_has(17);
  @$pb.TagNumber(18)
  void clearPatchFlow() => $_clearField(18);
  @$pb.TagNumber(18)
  $0.PatchFlow ensurePatchFlow() => $_ensure(17);

  @$pb.TagNumber(19)
  $0.StopFlow get stopFlow => $_getN(18);
  @$pb.TagNumber(19)
  set stopFlow($0.StopFlow value) => $_setField(19, value);
  @$pb.TagNumber(19)
  $core.bool hasStopFlow() => $_has(18);
  @$pb.TagNumber(19)
  void clearStopFlow() => $_clearField(19);
  @$pb.TagNumber(19)
  $0.StopFlow ensureStopFlow() => $_ensure(18);

  @$pb.TagNumber(20)
  $0.RequestArtifact get requestArtifact => $_getN(19);
  @$pb.TagNumber(20)
  set requestArtifact($0.RequestArtifact value) => $_setField(20, value);
  @$pb.TagNumber(20)
  $core.bool hasRequestArtifact() => $_has(19);
  @$pb.TagNumber(20)
  void clearRequestArtifact() => $_clearField(20);
  @$pb.TagNumber(20)
  $0.RequestArtifact ensureRequestArtifact() => $_ensure(19);

  @$pb.TagNumber(21)
  $1.BugReportAck get bugReportAck => $_getN(20);
  @$pb.TagNumber(21)
  set bugReportAck($1.BugReportAck value) => $_setField(21, value);
  @$pb.TagNumber(21)
  $core.bool hasBugReportAck() => $_has(20);
  @$pb.TagNumber(21)
  void clearBugReportAck() => $_clearField(21);
  @$pb.TagNumber(21)
  $1.BugReportAck ensureBugReportAck() => $_ensure(20);
}

class RegisterDevice extends $pb.GeneratedMessage {
  factory RegisterDevice({
    $3.DeviceCapabilities? capabilities,
  }) {
    final result = create();
    if (capabilities != null) result.capabilities = capabilities;
    return result;
  }

  RegisterDevice._();

  factory RegisterDevice.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RegisterDevice.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RegisterDevice',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.control.v1'),
      createEmptyInstance: create)
    ..aOM<$3.DeviceCapabilities>(1, _omitFieldNames ? '' : 'capabilities',
        subBuilder: $3.DeviceCapabilities.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RegisterDevice clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RegisterDevice copyWith(void Function(RegisterDevice) updates) =>
      super.copyWith((message) => updates(message as RegisterDevice))
          as RegisterDevice;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RegisterDevice create() => RegisterDevice._();
  @$core.override
  RegisterDevice createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RegisterDevice getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RegisterDevice>(create);
  static RegisterDevice? _defaultInstance;

  @$pb.TagNumber(1)
  $3.DeviceCapabilities get capabilities => $_getN(0);
  @$pb.TagNumber(1)
  set capabilities($3.DeviceCapabilities value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasCapabilities() => $_has(0);
  @$pb.TagNumber(1)
  void clearCapabilities() => $_clearField(1);
  @$pb.TagNumber(1)
  $3.DeviceCapabilities ensureCapabilities() => $_ensure(0);
}

class RegisterAck extends $pb.GeneratedMessage {
  factory RegisterAck({
    $core.String? serverId,
    $core.String? message,
    $core.Iterable<$core.MapEntry<$core.String, $core.String>>? metadata,
  }) {
    final result = create();
    if (serverId != null) result.serverId = serverId;
    if (message != null) result.message = message;
    if (metadata != null) result.metadata.addEntries(metadata);
    return result;
  }

  RegisterAck._();

  factory RegisterAck.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RegisterAck.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RegisterAck',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.control.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'serverId')
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..m<$core.String, $core.String>(3, _omitFieldNames ? '' : 'metadata',
        entryClassName: 'RegisterAck.MetadataEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OS,
        packageName: const $pb.PackageName('terminals.control.v1'))
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RegisterAck clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RegisterAck copyWith(void Function(RegisterAck) updates) =>
      super.copyWith((message) => updates(message as RegisterAck))
          as RegisterAck;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RegisterAck create() => RegisterAck._();
  @$core.override
  RegisterAck createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RegisterAck getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RegisterAck>(create);
  static RegisterAck? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get serverId => $_getSZ(0);
  @$pb.TagNumber(1)
  set serverId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasServerId() => $_has(0);
  @$pb.TagNumber(1)
  void clearServerId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => $_clearField(2);

  @$pb.TagNumber(3)
  $pb.PbMap<$core.String, $core.String> get metadata => $_getMap(2);
}

class CapabilityUpdate extends $pb.GeneratedMessage {
  factory CapabilityUpdate({
    $3.DeviceCapabilities? capabilities,
  }) {
    final result = create();
    if (capabilities != null) result.capabilities = capabilities;
    return result;
  }

  CapabilityUpdate._();

  factory CapabilityUpdate.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CapabilityUpdate.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CapabilityUpdate',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.control.v1'),
      createEmptyInstance: create)
    ..aOM<$3.DeviceCapabilities>(1, _omitFieldNames ? '' : 'capabilities',
        subBuilder: $3.DeviceCapabilities.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CapabilityUpdate clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CapabilityUpdate copyWith(void Function(CapabilityUpdate) updates) =>
      super.copyWith((message) => updates(message as CapabilityUpdate))
          as CapabilityUpdate;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CapabilityUpdate create() => CapabilityUpdate._();
  @$core.override
  CapabilityUpdate createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CapabilityUpdate getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CapabilityUpdate>(create);
  static CapabilityUpdate? _defaultInstance;

  @$pb.TagNumber(1)
  $3.DeviceCapabilities get capabilities => $_getN(0);
  @$pb.TagNumber(1)
  set capabilities($3.DeviceCapabilities value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasCapabilities() => $_has(0);
  @$pb.TagNumber(1)
  void clearCapabilities() => $_clearField(1);
  @$pb.TagNumber(1)
  $3.DeviceCapabilities ensureCapabilities() => $_ensure(0);
}

class StreamReady extends $pb.GeneratedMessage {
  factory StreamReady({
    $core.String? streamId,
  }) {
    final result = create();
    if (streamId != null) result.streamId = streamId;
    return result;
  }

  StreamReady._();

  factory StreamReady.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StreamReady.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StreamReady',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.control.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'streamId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StreamReady clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StreamReady copyWith(void Function(StreamReady) updates) =>
      super.copyWith((message) => updates(message as StreamReady))
          as StreamReady;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StreamReady create() => StreamReady._();
  @$core.override
  StreamReady createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StreamReady getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StreamReady>(create);
  static StreamReady? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get streamId => $_getSZ(0);
  @$pb.TagNumber(1)
  set streamId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasStreamId() => $_has(0);
  @$pb.TagNumber(1)
  void clearStreamId() => $_clearField(1);
}

class CommandRequest extends $pb.GeneratedMessage {
  factory CommandRequest({
    $core.String? requestId,
    $core.String? deviceId,
    CommandAction? action,
    CommandKind? kind,
    $core.String? text,
    $core.String? intent,
    $core.Iterable<$core.MapEntry<$core.String, $core.String>>? arguments,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (deviceId != null) result.deviceId = deviceId;
    if (action != null) result.action = action;
    if (kind != null) result.kind = kind;
    if (text != null) result.text = text;
    if (intent != null) result.intent = intent;
    if (arguments != null) result.arguments.addEntries(arguments);
    return result;
  }

  CommandRequest._();

  factory CommandRequest.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CommandRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CommandRequest',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.control.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOS(2, _omitFieldNames ? '' : 'deviceId')
    ..aE<CommandAction>(3, _omitFieldNames ? '' : 'action',
        enumValues: CommandAction.values)
    ..aE<CommandKind>(4, _omitFieldNames ? '' : 'kind',
        enumValues: CommandKind.values)
    ..aOS(5, _omitFieldNames ? '' : 'text')
    ..aOS(6, _omitFieldNames ? '' : 'intent')
    ..m<$core.String, $core.String>(7, _omitFieldNames ? '' : 'arguments',
        entryClassName: 'CommandRequest.ArgumentsEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OS,
        packageName: const $pb.PackageName('terminals.control.v1'))
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommandRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommandRequest copyWith(void Function(CommandRequest) updates) =>
      super.copyWith((message) => updates(message as CommandRequest))
          as CommandRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CommandRequest create() => CommandRequest._();
  @$core.override
  CommandRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CommandRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CommandRequest>(create);
  static CommandRequest? _defaultInstance;

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
  CommandAction get action => $_getN(2);
  @$pb.TagNumber(3)
  set action(CommandAction value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasAction() => $_has(2);
  @$pb.TagNumber(3)
  void clearAction() => $_clearField(3);

  @$pb.TagNumber(4)
  CommandKind get kind => $_getN(3);
  @$pb.TagNumber(4)
  set kind(CommandKind value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasKind() => $_has(3);
  @$pb.TagNumber(4)
  void clearKind() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get text => $_getSZ(4);
  @$pb.TagNumber(5)
  set text($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasText() => $_has(4);
  @$pb.TagNumber(5)
  void clearText() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get intent => $_getSZ(5);
  @$pb.TagNumber(6)
  set intent($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasIntent() => $_has(5);
  @$pb.TagNumber(6)
  void clearIntent() => $_clearField(6);

  @$pb.TagNumber(7)
  $pb.PbMap<$core.String, $core.String> get arguments => $_getMap(6);
}

class CommandResult extends $pb.GeneratedMessage {
  factory CommandResult({
    $core.String? requestId,
    $core.String? scenarioStart,
    $core.String? scenarioStop,
    $core.String? notification,
    $core.Iterable<$core.MapEntry<$core.String, $core.String>>? data,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (scenarioStart != null) result.scenarioStart = scenarioStart;
    if (scenarioStop != null) result.scenarioStop = scenarioStop;
    if (notification != null) result.notification = notification;
    if (data != null) result.data.addEntries(data);
    return result;
  }

  CommandResult._();

  factory CommandResult.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory CommandResult.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'CommandResult',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.control.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOS(2, _omitFieldNames ? '' : 'scenarioStart')
    ..aOS(3, _omitFieldNames ? '' : 'scenarioStop')
    ..aOS(4, _omitFieldNames ? '' : 'notification')
    ..m<$core.String, $core.String>(5, _omitFieldNames ? '' : 'data',
        entryClassName: 'CommandResult.DataEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OS,
        packageName: const $pb.PackageName('terminals.control.v1'))
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommandResult clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  CommandResult copyWith(void Function(CommandResult) updates) =>
      super.copyWith((message) => updates(message as CommandResult))
          as CommandResult;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static CommandResult create() => CommandResult._();
  @$core.override
  CommandResult createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static CommandResult getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<CommandResult>(create);
  static CommandResult? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get scenarioStart => $_getSZ(1);
  @$pb.TagNumber(2)
  set scenarioStart($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasScenarioStart() => $_has(1);
  @$pb.TagNumber(2)
  void clearScenarioStart() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get scenarioStop => $_getSZ(2);
  @$pb.TagNumber(3)
  set scenarioStop($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasScenarioStop() => $_has(2);
  @$pb.TagNumber(3)
  void clearScenarioStop() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get notification => $_getSZ(3);
  @$pb.TagNumber(4)
  set notification($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasNotification() => $_has(3);
  @$pb.TagNumber(4)
  void clearNotification() => $_clearField(4);

  @$pb.TagNumber(5)
  $pb.PbMap<$core.String, $core.String> get data => $_getMap(4);
}

class ControlError extends $pb.GeneratedMessage {
  factory ControlError({
    ControlErrorCode? code,
    $core.String? message,
  }) {
    final result = create();
    if (code != null) result.code = code;
    if (message != null) result.message = message;
    return result;
  }

  ControlError._();

  factory ControlError.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ControlError.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ControlError',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.control.v1'),
      createEmptyInstance: create)
    ..aE<ControlErrorCode>(1, _omitFieldNames ? '' : 'code',
        enumValues: ControlErrorCode.values)
    ..aOS(2, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ControlError clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ControlError copyWith(void Function(ControlError) updates) =>
      super.copyWith((message) => updates(message as ControlError))
          as ControlError;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ControlError create() => ControlError._();
  @$core.override
  ControlError createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ControlError getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ControlError>(create);
  static ControlError? _defaultInstance;

  @$pb.TagNumber(1)
  ControlErrorCode get code => $_getN(0);
  @$pb.TagNumber(1)
  set code(ControlErrorCode value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasCode() => $_has(0);
  @$pb.TagNumber(1)
  void clearCode() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get message => $_getSZ(1);
  @$pb.TagNumber(2)
  set message($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasMessage() => $_has(1);
  @$pb.TagNumber(2)
  void clearMessage() => $_clearField(2);
}

class WebRTCSignal extends $pb.GeneratedMessage {
  factory WebRTCSignal({
    $core.String? streamId,
    $core.String? signalType,
    $core.String? payload,
  }) {
    final result = create();
    if (streamId != null) result.streamId = streamId;
    if (signalType != null) result.signalType = signalType;
    if (payload != null) result.payload = payload;
    return result;
  }

  WebRTCSignal._();

  factory WebRTCSignal.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory WebRTCSignal.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'WebRTCSignal',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.control.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'streamId')
    ..aOS(2, _omitFieldNames ? '' : 'signalType')
    ..aOS(3, _omitFieldNames ? '' : 'payload')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WebRTCSignal clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WebRTCSignal copyWith(void Function(WebRTCSignal) updates) =>
      super.copyWith((message) => updates(message as WebRTCSignal))
          as WebRTCSignal;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static WebRTCSignal create() => WebRTCSignal._();
  @$core.override
  WebRTCSignal createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static WebRTCSignal getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<WebRTCSignal>(create);
  static WebRTCSignal? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get streamId => $_getSZ(0);
  @$pb.TagNumber(1)
  set streamId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasStreamId() => $_has(0);
  @$pb.TagNumber(1)
  void clearStreamId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get signalType => $_getSZ(1);
  @$pb.TagNumber(2)
  set signalType($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasSignalType() => $_has(1);
  @$pb.TagNumber(2)
  void clearSignalType() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get payload => $_getSZ(2);
  @$pb.TagNumber(3)
  set payload($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasPayload() => $_has(2);
  @$pb.TagNumber(3)
  void clearPayload() => $_clearField(3);
}

class Heartbeat extends $pb.GeneratedMessage {
  factory Heartbeat({
    $core.String? deviceId,
    $fixnum.Int64? unixMs,
  }) {
    final result = create();
    if (deviceId != null) result.deviceId = deviceId;
    if (unixMs != null) result.unixMs = unixMs;
    return result;
  }

  Heartbeat._();

  factory Heartbeat.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory Heartbeat.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'Heartbeat',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.control.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'deviceId')
    ..aInt64(2, _omitFieldNames ? '' : 'unixMs')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Heartbeat clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  Heartbeat copyWith(void Function(Heartbeat) updates) =>
      super.copyWith((message) => updates(message as Heartbeat)) as Heartbeat;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static Heartbeat create() => Heartbeat._();
  @$core.override
  Heartbeat createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static Heartbeat getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<Heartbeat>(create);
  static Heartbeat? _defaultInstance;

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
}

class TerminalControlServiceApi {
  final $pb.RpcClient _client;

  TerminalControlServiceApi(this._client);

  $async.Future<ConnectResponse> connect(
          $pb.ClientContext? ctx, ConnectRequest request) =>
      _client.invoke<ConnectResponse>(
          ctx, 'TerminalControlService', 'Connect', request, ConnectResponse());
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
