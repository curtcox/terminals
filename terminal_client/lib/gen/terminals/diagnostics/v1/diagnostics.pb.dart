// This is a generated file - do not edit.
//
// Generated from terminals/diagnostics/v1/diagnostics.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:fixnum/fixnum.dart' as $fixnum;
import 'package:protobuf/protobuf.dart' as $pb;

import '../../capabilities/v1/capabilities.pb.dart' as $0;
import '../../ui/v1/ui.pb.dart' as $1;
import 'diagnostics.pbenum.dart';

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

export 'diagnostics.pbenum.dart';

/// BugReport is filed by a reporter device against a subject device (which may
/// be the same device, a different device, or unknown). The client contributes
/// ClientContext; the server enriches the record with subject-side state.
class BugReport extends $pb.GeneratedMessage {
  factory BugReport({
    $core.String? reportId,
    $core.String? reporterDeviceId,
    $core.String? subjectDeviceId,
    BugReportSource? source,
    $core.String? description,
    $core.Iterable<$core.String>? tags,
    $fixnum.Int64? timestampUnixMs,
    ClientContext? clientContext,
    $core.List<$core.int>? screenshotPng,
    $core.List<$core.int>? audioWav,
    $core.Iterable<$core.MapEntry<$core.String, $core.String>>? sourceHints,
  }) {
    final result = create();
    if (reportId != null) result.reportId = reportId;
    if (reporterDeviceId != null) result.reporterDeviceId = reporterDeviceId;
    if (subjectDeviceId != null) result.subjectDeviceId = subjectDeviceId;
    if (source != null) result.source = source;
    if (description != null) result.description = description;
    if (tags != null) result.tags.addAll(tags);
    if (timestampUnixMs != null) result.timestampUnixMs = timestampUnixMs;
    if (clientContext != null) result.clientContext = clientContext;
    if (screenshotPng != null) result.screenshotPng = screenshotPng;
    if (audioWav != null) result.audioWav = audioWav;
    if (sourceHints != null) result.sourceHints.addEntries(sourceHints);
    return result;
  }

  BugReport._();

  factory BugReport.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory BugReport.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'BugReport',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.diagnostics.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'reportId')
    ..aOS(2, _omitFieldNames ? '' : 'reporterDeviceId')
    ..aOS(3, _omitFieldNames ? '' : 'subjectDeviceId')
    ..aE<BugReportSource>(4, _omitFieldNames ? '' : 'source',
        enumValues: BugReportSource.values)
    ..aOS(5, _omitFieldNames ? '' : 'description')
    ..pPS(6, _omitFieldNames ? '' : 'tags')
    ..aInt64(7, _omitFieldNames ? '' : 'timestampUnixMs')
    ..aOM<ClientContext>(8, _omitFieldNames ? '' : 'clientContext',
        subBuilder: ClientContext.create)
    ..a<$core.List<$core.int>>(
        9, _omitFieldNames ? '' : 'screenshotPng', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        10, _omitFieldNames ? '' : 'audioWav', $pb.PbFieldType.OY)
    ..m<$core.String, $core.String>(11, _omitFieldNames ? '' : 'sourceHints',
        entryClassName: 'BugReport.SourceHintsEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OS,
        packageName: const $pb.PackageName('terminals.diagnostics.v1'))
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BugReport clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BugReport copyWith(void Function(BugReport) updates) =>
      super.copyWith((message) => updates(message as BugReport)) as BugReport;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static BugReport create() => BugReport._();
  @$core.override
  BugReport createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static BugReport getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<BugReport>(create);
  static BugReport? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get reportId => $_getSZ(0);
  @$pb.TagNumber(1)
  set reportId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasReportId() => $_has(0);
  @$pb.TagNumber(1)
  void clearReportId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get reporterDeviceId => $_getSZ(1);
  @$pb.TagNumber(2)
  set reporterDeviceId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasReporterDeviceId() => $_has(1);
  @$pb.TagNumber(2)
  void clearReporterDeviceId() => $_clearField(2);

  /// subject_device_id may equal reporter_device_id. Empty means "unknown
  /// subject" and the server will attempt to infer one from tags/description.
  @$pb.TagNumber(3)
  $core.String get subjectDeviceId => $_getSZ(2);
  @$pb.TagNumber(3)
  set subjectDeviceId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSubjectDeviceId() => $_has(2);
  @$pb.TagNumber(3)
  void clearSubjectDeviceId() => $_clearField(3);

  @$pb.TagNumber(4)
  BugReportSource get source => $_getN(3);
  @$pb.TagNumber(4)
  set source(BugReportSource value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasSource() => $_has(3);
  @$pb.TagNumber(4)
  void clearSource() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get description => $_getSZ(4);
  @$pb.TagNumber(5)
  set description($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasDescription() => $_has(4);
  @$pb.TagNumber(5)
  void clearDescription() => $_clearField(5);

  @$pb.TagNumber(6)
  $pb.PbList<$core.String> get tags => $_getList(5);

  @$pb.TagNumber(7)
  $fixnum.Int64 get timestampUnixMs => $_getI64(6);
  @$pb.TagNumber(7)
  set timestampUnixMs($fixnum.Int64 value) => $_setInt64(6, value);
  @$pb.TagNumber(7)
  $core.bool hasTimestampUnixMs() => $_has(6);
  @$pb.TagNumber(7)
  void clearTimestampUnixMs() => $_clearField(7);

  @$pb.TagNumber(8)
  ClientContext get clientContext => $_getN(7);
  @$pb.TagNumber(8)
  set clientContext(ClientContext value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasClientContext() => $_has(7);
  @$pb.TagNumber(8)
  void clearClientContext() => $_clearField(8);
  @$pb.TagNumber(8)
  ClientContext ensureClientContext() => $_ensure(7);

  /// Inline attachments; server MAY move these to sibling files on disk. When
  /// larger than the intake cap, clients SHOULD upload via a separate artifact
  /// request and leave these empty.
  @$pb.TagNumber(9)
  $core.List<$core.int> get screenshotPng => $_getN(8);
  @$pb.TagNumber(9)
  set screenshotPng($core.List<$core.int> value) => $_setBytes(8, value);
  @$pb.TagNumber(9)
  $core.bool hasScreenshotPng() => $_has(8);
  @$pb.TagNumber(9)
  void clearScreenshotPng() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.List<$core.int> get audioWav => $_getN(9);
  @$pb.TagNumber(10)
  set audioWav($core.List<$core.int> value) => $_setBytes(9, value);
  @$pb.TagNumber(10)
  $core.bool hasAudioWav() => $_has(9);
  @$pb.TagNumber(10)
  void clearAudioWav() => $_clearField(10);

  /// Optional free-form structured hints captured by the originating entry
  /// point (e.g. voice transcript confidence, NFC tag id, SIP caller id).
  @$pb.TagNumber(11)
  $pb.PbMap<$core.String, $core.String> get sourceHints => $_getMap(10);
}

/// BugReportAck is the server's persisted acknowledgement of a BugReport.
class BugReportAck extends $pb.GeneratedMessage {
  factory BugReportAck({
    $core.String? reportId,
    $core.String? correlationId,
    BugReportStatus? status,
    $core.String? reportPath,
    $core.String? mergedAutodetectReportId,
    $core.String? message,
  }) {
    final result = create();
    if (reportId != null) result.reportId = reportId;
    if (correlationId != null) result.correlationId = correlationId;
    if (status != null) result.status = status;
    if (reportPath != null) result.reportPath = reportPath;
    if (mergedAutodetectReportId != null)
      result.mergedAutodetectReportId = mergedAutodetectReportId;
    if (message != null) result.message = message;
    return result;
  }

  BugReportAck._();

  factory BugReportAck.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory BugReportAck.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'BugReportAck',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.diagnostics.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'reportId')
    ..aOS(2, _omitFieldNames ? '' : 'correlationId')
    ..aE<BugReportStatus>(3, _omitFieldNames ? '' : 'status',
        enumValues: BugReportStatus.values)
    ..aOS(4, _omitFieldNames ? '' : 'reportPath')
    ..aOS(5, _omitFieldNames ? '' : 'mergedAutodetectReportId')
    ..aOS(6, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BugReportAck clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  BugReportAck copyWith(void Function(BugReportAck) updates) =>
      super.copyWith((message) => updates(message as BugReportAck))
          as BugReportAck;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static BugReportAck create() => BugReportAck._();
  @$core.override
  BugReportAck createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static BugReportAck getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<BugReportAck>(create);
  static BugReportAck? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get reportId => $_getSZ(0);
  @$pb.TagNumber(1)
  set reportId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasReportId() => $_has(0);
  @$pb.TagNumber(1)
  void clearReportId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get correlationId => $_getSZ(1);
  @$pb.TagNumber(2)
  set correlationId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCorrelationId() => $_has(1);
  @$pb.TagNumber(2)
  void clearCorrelationId() => $_clearField(2);

  @$pb.TagNumber(3)
  BugReportStatus get status => $_getN(2);
  @$pb.TagNumber(3)
  set status(BugReportStatus value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasStatus() => $_has(2);
  @$pb.TagNumber(3)
  void clearStatus() => $_clearField(3);

  /// Pointer into the durable JSON record on disk, e.g.
  /// "logs/bug_reports/2026-04-16/<report_id>.json".
  @$pb.TagNumber(4)
  $core.String get reportPath => $_getSZ(3);
  @$pb.TagNumber(4)
  set reportPath($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasReportPath() => $_has(3);
  @$pb.TagNumber(4)
  void clearReportPath() => $_clearField(4);

  /// Populated when status == MERGED_WITH_AUTODETECT.
  @$pb.TagNumber(5)
  $core.String get mergedAutodetectReportId => $_getSZ(4);
  @$pb.TagNumber(5)
  set mergedAutodetectReportId($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasMergedAutodetectReportId() => $_has(4);
  @$pb.TagNumber(5)
  void clearMergedAutodetectReportId() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get message => $_getSZ(5);
  @$pb.TagNumber(6)
  set message($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasMessage() => $_has(5);
  @$pb.TagNumber(6)
  void clearMessage() => $_clearField(6);
}

/// ClientContext is the reporter-side snapshot collected at report time.
/// Every collection is size-capped and redacted before transmission per the
/// rules documented in plans/features/bug-reporting.md.
class ClientContext extends $pb.GeneratedMessage {
  factory ClientContext({
    ClientIdentity? identity,
    $0.DeviceCapabilities? capabilities,
    RuntimeState? runtime,
    ConnectionHealth? connection,
    HardwareState? hardware,
    ErrorCapture? errorCapture,
  }) {
    final result = create();
    if (identity != null) result.identity = identity;
    if (capabilities != null) result.capabilities = capabilities;
    if (runtime != null) result.runtime = runtime;
    if (connection != null) result.connection = connection;
    if (hardware != null) result.hardware = hardware;
    if (errorCapture != null) result.errorCapture = errorCapture;
    return result;
  }

  ClientContext._();

  factory ClientContext.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientContext.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientContext',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.diagnostics.v1'),
      createEmptyInstance: create)
    ..aOM<ClientIdentity>(1, _omitFieldNames ? '' : 'identity',
        subBuilder: ClientIdentity.create)
    ..aOM<$0.DeviceCapabilities>(2, _omitFieldNames ? '' : 'capabilities',
        subBuilder: $0.DeviceCapabilities.create)
    ..aOM<RuntimeState>(3, _omitFieldNames ? '' : 'runtime',
        subBuilder: RuntimeState.create)
    ..aOM<ConnectionHealth>(4, _omitFieldNames ? '' : 'connection',
        subBuilder: ConnectionHealth.create)
    ..aOM<HardwareState>(5, _omitFieldNames ? '' : 'hardware',
        subBuilder: HardwareState.create)
    ..aOM<ErrorCapture>(6, _omitFieldNames ? '' : 'errorCapture',
        subBuilder: ErrorCapture.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientContext clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientContext copyWith(void Function(ClientContext) updates) =>
      super.copyWith((message) => updates(message as ClientContext))
          as ClientContext;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientContext create() => ClientContext._();
  @$core.override
  ClientContext createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientContext getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientContext>(create);
  static ClientContext? _defaultInstance;

  @$pb.TagNumber(1)
  ClientIdentity get identity => $_getN(0);
  @$pb.TagNumber(1)
  set identity(ClientIdentity value) => $_setField(1, value);
  @$pb.TagNumber(1)
  $core.bool hasIdentity() => $_has(0);
  @$pb.TagNumber(1)
  void clearIdentity() => $_clearField(1);
  @$pb.TagNumber(1)
  ClientIdentity ensureIdentity() => $_ensure(0);

  @$pb.TagNumber(2)
  $0.DeviceCapabilities get capabilities => $_getN(1);
  @$pb.TagNumber(2)
  set capabilities($0.DeviceCapabilities value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasCapabilities() => $_has(1);
  @$pb.TagNumber(2)
  void clearCapabilities() => $_clearField(2);
  @$pb.TagNumber(2)
  $0.DeviceCapabilities ensureCapabilities() => $_ensure(1);

  @$pb.TagNumber(3)
  RuntimeState get runtime => $_getN(2);
  @$pb.TagNumber(3)
  set runtime(RuntimeState value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasRuntime() => $_has(2);
  @$pb.TagNumber(3)
  void clearRuntime() => $_clearField(3);
  @$pb.TagNumber(3)
  RuntimeState ensureRuntime() => $_ensure(2);

  @$pb.TagNumber(4)
  ConnectionHealth get connection => $_getN(3);
  @$pb.TagNumber(4)
  set connection(ConnectionHealth value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasConnection() => $_has(3);
  @$pb.TagNumber(4)
  void clearConnection() => $_clearField(4);
  @$pb.TagNumber(4)
  ConnectionHealth ensureConnection() => $_ensure(3);

  @$pb.TagNumber(5)
  HardwareState get hardware => $_getN(4);
  @$pb.TagNumber(5)
  set hardware(HardwareState value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasHardware() => $_has(4);
  @$pb.TagNumber(5)
  void clearHardware() => $_clearField(5);
  @$pb.TagNumber(5)
  HardwareState ensureHardware() => $_ensure(4);

  @$pb.TagNumber(6)
  ErrorCapture get errorCapture => $_getN(5);
  @$pb.TagNumber(6)
  set errorCapture(ErrorCapture value) => $_setField(6, value);
  @$pb.TagNumber(6)
  $core.bool hasErrorCapture() => $_has(5);
  @$pb.TagNumber(6)
  void clearErrorCapture() => $_clearField(6);
  @$pb.TagNumber(6)
  ErrorCapture ensureErrorCapture() => $_ensure(5);
}

class ClientIdentity extends $pb.GeneratedMessage {
  factory ClientIdentity({
    $core.String? deviceId,
    $core.String? deviceName,
    $core.String? deviceType,
    $core.String? platform,
    $core.String? clientVersion,
    $core.String? clientGitSha,
    $fixnum.Int64? clientBuildUnixMs,
    $core.String? osVersion,
    $core.String? locale,
    $core.String? timezone,
    $fixnum.Int64? clockOffsetMs,
  }) {
    final result = create();
    if (deviceId != null) result.deviceId = deviceId;
    if (deviceName != null) result.deviceName = deviceName;
    if (deviceType != null) result.deviceType = deviceType;
    if (platform != null) result.platform = platform;
    if (clientVersion != null) result.clientVersion = clientVersion;
    if (clientGitSha != null) result.clientGitSha = clientGitSha;
    if (clientBuildUnixMs != null) result.clientBuildUnixMs = clientBuildUnixMs;
    if (osVersion != null) result.osVersion = osVersion;
    if (locale != null) result.locale = locale;
    if (timezone != null) result.timezone = timezone;
    if (clockOffsetMs != null) result.clockOffsetMs = clockOffsetMs;
    return result;
  }

  ClientIdentity._();

  factory ClientIdentity.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ClientIdentity.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ClientIdentity',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.diagnostics.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'deviceId')
    ..aOS(2, _omitFieldNames ? '' : 'deviceName')
    ..aOS(3, _omitFieldNames ? '' : 'deviceType')
    ..aOS(4, _omitFieldNames ? '' : 'platform')
    ..aOS(5, _omitFieldNames ? '' : 'clientVersion')
    ..aOS(6, _omitFieldNames ? '' : 'clientGitSha')
    ..aInt64(7, _omitFieldNames ? '' : 'clientBuildUnixMs')
    ..aOS(8, _omitFieldNames ? '' : 'osVersion')
    ..aOS(9, _omitFieldNames ? '' : 'locale')
    ..aOS(10, _omitFieldNames ? '' : 'timezone')
    ..aInt64(11, _omitFieldNames ? '' : 'clockOffsetMs')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientIdentity clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ClientIdentity copyWith(void Function(ClientIdentity) updates) =>
      super.copyWith((message) => updates(message as ClientIdentity))
          as ClientIdentity;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ClientIdentity create() => ClientIdentity._();
  @$core.override
  ClientIdentity createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ClientIdentity getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ClientIdentity>(create);
  static ClientIdentity? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get deviceId => $_getSZ(0);
  @$pb.TagNumber(1)
  set deviceId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasDeviceId() => $_has(0);
  @$pb.TagNumber(1)
  void clearDeviceId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get deviceName => $_getSZ(1);
  @$pb.TagNumber(2)
  set deviceName($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasDeviceName() => $_has(1);
  @$pb.TagNumber(2)
  void clearDeviceName() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get deviceType => $_getSZ(2);
  @$pb.TagNumber(3)
  set deviceType($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasDeviceType() => $_has(2);
  @$pb.TagNumber(3)
  void clearDeviceType() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get platform => $_getSZ(3);
  @$pb.TagNumber(4)
  set platform($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasPlatform() => $_has(3);
  @$pb.TagNumber(4)
  void clearPlatform() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get clientVersion => $_getSZ(4);
  @$pb.TagNumber(5)
  set clientVersion($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasClientVersion() => $_has(4);
  @$pb.TagNumber(5)
  void clearClientVersion() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get clientGitSha => $_getSZ(5);
  @$pb.TagNumber(6)
  set clientGitSha($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasClientGitSha() => $_has(5);
  @$pb.TagNumber(6)
  void clearClientGitSha() => $_clearField(6);

  @$pb.TagNumber(7)
  $fixnum.Int64 get clientBuildUnixMs => $_getI64(6);
  @$pb.TagNumber(7)
  set clientBuildUnixMs($fixnum.Int64 value) => $_setInt64(6, value);
  @$pb.TagNumber(7)
  $core.bool hasClientBuildUnixMs() => $_has(6);
  @$pb.TagNumber(7)
  void clearClientBuildUnixMs() => $_clearField(7);

  @$pb.TagNumber(8)
  $core.String get osVersion => $_getSZ(7);
  @$pb.TagNumber(8)
  set osVersion($core.String value) => $_setString(7, value);
  @$pb.TagNumber(8)
  $core.bool hasOsVersion() => $_has(7);
  @$pb.TagNumber(8)
  void clearOsVersion() => $_clearField(8);

  @$pb.TagNumber(9)
  $core.String get locale => $_getSZ(8);
  @$pb.TagNumber(9)
  set locale($core.String value) => $_setString(8, value);
  @$pb.TagNumber(9)
  $core.bool hasLocale() => $_has(8);
  @$pb.TagNumber(9)
  void clearLocale() => $_clearField(9);

  @$pb.TagNumber(10)
  $core.String get timezone => $_getSZ(9);
  @$pb.TagNumber(10)
  set timezone($core.String value) => $_setString(9, value);
  @$pb.TagNumber(10)
  $core.bool hasTimezone() => $_has(9);
  @$pb.TagNumber(10)
  void clearTimezone() => $_clearField(10);

  /// Device system-clock offset vs server, in milliseconds. Positive means the
  /// device clock is ahead of the server. Zero means unknown.
  @$pb.TagNumber(11)
  $fixnum.Int64 get clockOffsetMs => $_getI64(10);
  @$pb.TagNumber(11)
  set clockOffsetMs($fixnum.Int64 value) => $_setInt64(10, value);
  @$pb.TagNumber(11)
  $core.bool hasClockOffsetMs() => $_has(10);
  @$pb.TagNumber(11)
  void clearClockOffsetMs() => $_clearField(11);
}

class RuntimeState extends $pb.GeneratedMessage {
  factory RuntimeState({
    $core.Iterable<$core.String>? scenarioIds,
    $core.Iterable<$core.String>? activationIds,
    $1.Node? activeUiRoot,
    $core.Iterable<UiEventEntry>? recentUiUpdates,
    $core.Iterable<UiActionEntry>? recentUiActions,
    $core.Iterable<StreamEntry>? activeStreams,
    $core.Iterable<RouteEntry>? activeRoutes,
    $core.Iterable<WebrtcSignalEntry>? recentWebrtcSignals,
    $core.Iterable<LogEntry>? recentLogs,
  }) {
    final result = create();
    if (scenarioIds != null) result.scenarioIds.addAll(scenarioIds);
    if (activationIds != null) result.activationIds.addAll(activationIds);
    if (activeUiRoot != null) result.activeUiRoot = activeUiRoot;
    if (recentUiUpdates != null) result.recentUiUpdates.addAll(recentUiUpdates);
    if (recentUiActions != null) result.recentUiActions.addAll(recentUiActions);
    if (activeStreams != null) result.activeStreams.addAll(activeStreams);
    if (activeRoutes != null) result.activeRoutes.addAll(activeRoutes);
    if (recentWebrtcSignals != null)
      result.recentWebrtcSignals.addAll(recentWebrtcSignals);
    if (recentLogs != null) result.recentLogs.addAll(recentLogs);
    return result;
  }

  RuntimeState._();

  factory RuntimeState.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RuntimeState.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RuntimeState',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.diagnostics.v1'),
      createEmptyInstance: create)
    ..pPS(1, _omitFieldNames ? '' : 'scenarioIds')
    ..pPS(2, _omitFieldNames ? '' : 'activationIds')
    ..aOM<$1.Node>(3, _omitFieldNames ? '' : 'activeUiRoot',
        subBuilder: $1.Node.create)
    ..pPM<UiEventEntry>(4, _omitFieldNames ? '' : 'recentUiUpdates',
        subBuilder: UiEventEntry.create)
    ..pPM<UiActionEntry>(5, _omitFieldNames ? '' : 'recentUiActions',
        subBuilder: UiActionEntry.create)
    ..pPM<StreamEntry>(6, _omitFieldNames ? '' : 'activeStreams',
        subBuilder: StreamEntry.create)
    ..pPM<RouteEntry>(7, _omitFieldNames ? '' : 'activeRoutes',
        subBuilder: RouteEntry.create)
    ..pPM<WebrtcSignalEntry>(8, _omitFieldNames ? '' : 'recentWebrtcSignals',
        subBuilder: WebrtcSignalEntry.create)
    ..pPM<LogEntry>(9, _omitFieldNames ? '' : 'recentLogs',
        subBuilder: LogEntry.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RuntimeState clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RuntimeState copyWith(void Function(RuntimeState) updates) =>
      super.copyWith((message) => updates(message as RuntimeState))
          as RuntimeState;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RuntimeState create() => RuntimeState._();
  @$core.override
  RuntimeState createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RuntimeState getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RuntimeState>(create);
  static RuntimeState? _defaultInstance;

  @$pb.TagNumber(1)
  $pb.PbList<$core.String> get scenarioIds => $_getList(0);

  @$pb.TagNumber(2)
  $pb.PbList<$core.String> get activationIds => $_getList(1);

  /// Serialized UI root exactly as last set by the server.
  @$pb.TagNumber(3)
  $1.Node get activeUiRoot => $_getN(2);
  @$pb.TagNumber(3)
  set activeUiRoot($1.Node value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasActiveUiRoot() => $_has(2);
  @$pb.TagNumber(3)
  void clearActiveUiRoot() => $_clearField(3);
  @$pb.TagNumber(3)
  $1.Node ensureActiveUiRoot() => $_ensure(2);

  @$pb.TagNumber(4)
  $pb.PbList<UiEventEntry> get recentUiUpdates => $_getList(3);

  @$pb.TagNumber(5)
  $pb.PbList<UiActionEntry> get recentUiActions => $_getList(4);

  @$pb.TagNumber(6)
  $pb.PbList<StreamEntry> get activeStreams => $_getList(5);

  @$pb.TagNumber(7)
  $pb.PbList<RouteEntry> get activeRoutes => $_getList(6);

  @$pb.TagNumber(8)
  $pb.PbList<WebrtcSignalEntry> get recentWebrtcSignals => $_getList(7);

  /// Free-form log ring buffer. Values are already redacted by the client.
  @$pb.TagNumber(9)
  $pb.PbList<LogEntry> get recentLogs => $_getList(8);
}

class UiEventEntry extends $pb.GeneratedMessage {
  factory UiEventEntry({
    $fixnum.Int64? unixMs,
    $core.String? kind,
    $core.String? componentId,
    $core.String? detail,
  }) {
    final result = create();
    if (unixMs != null) result.unixMs = unixMs;
    if (kind != null) result.kind = kind;
    if (componentId != null) result.componentId = componentId;
    if (detail != null) result.detail = detail;
    return result;
  }

  UiEventEntry._();

  factory UiEventEntry.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory UiEventEntry.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'UiEventEntry',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.diagnostics.v1'),
      createEmptyInstance: create)
    ..aInt64(1, _omitFieldNames ? '' : 'unixMs')
    ..aOS(2, _omitFieldNames ? '' : 'kind')
    ..aOS(3, _omitFieldNames ? '' : 'componentId')
    ..aOS(4, _omitFieldNames ? '' : 'detail')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UiEventEntry clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UiEventEntry copyWith(void Function(UiEventEntry) updates) =>
      super.copyWith((message) => updates(message as UiEventEntry))
          as UiEventEntry;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static UiEventEntry create() => UiEventEntry._();
  @$core.override
  UiEventEntry createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static UiEventEntry getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<UiEventEntry>(create);
  static UiEventEntry? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get unixMs => $_getI64(0);
  @$pb.TagNumber(1)
  set unixMs($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasUnixMs() => $_has(0);
  @$pb.TagNumber(1)
  void clearUnixMs() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get kind => $_getSZ(1);
  @$pb.TagNumber(2)
  set kind($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasKind() => $_has(1);
  @$pb.TagNumber(2)
  void clearKind() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get componentId => $_getSZ(2);
  @$pb.TagNumber(3)
  set componentId($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasComponentId() => $_has(2);
  @$pb.TagNumber(3)
  void clearComponentId() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get detail => $_getSZ(3);
  @$pb.TagNumber(4)
  set detail($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasDetail() => $_has(3);
  @$pb.TagNumber(4)
  void clearDetail() => $_clearField(4);
}

class UiActionEntry extends $pb.GeneratedMessage {
  factory UiActionEntry({
    $fixnum.Int64? unixMs,
    $core.String? componentId,
    $core.String? action,
    $core.String? value,
  }) {
    final result = create();
    if (unixMs != null) result.unixMs = unixMs;
    if (componentId != null) result.componentId = componentId;
    if (action != null) result.action = action;
    if (value != null) result.value = value;
    return result;
  }

  UiActionEntry._();

  factory UiActionEntry.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory UiActionEntry.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'UiActionEntry',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.diagnostics.v1'),
      createEmptyInstance: create)
    ..aInt64(1, _omitFieldNames ? '' : 'unixMs')
    ..aOS(2, _omitFieldNames ? '' : 'componentId')
    ..aOS(3, _omitFieldNames ? '' : 'action')
    ..aOS(4, _omitFieldNames ? '' : 'value')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UiActionEntry clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  UiActionEntry copyWith(void Function(UiActionEntry) updates) =>
      super.copyWith((message) => updates(message as UiActionEntry))
          as UiActionEntry;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static UiActionEntry create() => UiActionEntry._();
  @$core.override
  UiActionEntry createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static UiActionEntry getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<UiActionEntry>(create);
  static UiActionEntry? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get unixMs => $_getI64(0);
  @$pb.TagNumber(1)
  set unixMs($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasUnixMs() => $_has(0);
  @$pb.TagNumber(1)
  void clearUnixMs() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get componentId => $_getSZ(1);
  @$pb.TagNumber(2)
  set componentId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasComponentId() => $_has(1);
  @$pb.TagNumber(2)
  void clearComponentId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get action => $_getSZ(2);
  @$pb.TagNumber(3)
  set action($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasAction() => $_has(2);
  @$pb.TagNumber(3)
  void clearAction() => $_clearField(3);

  /// Value is redacted on the client before send.
  @$pb.TagNumber(4)
  $core.String get value => $_getSZ(3);
  @$pb.TagNumber(4)
  set value($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasValue() => $_has(3);
  @$pb.TagNumber(4)
  void clearValue() => $_clearField(4);
}

class StreamEntry extends $pb.GeneratedMessage {
  factory StreamEntry({
    $core.String? streamId,
    $core.String? kind,
    $core.String? sourceDeviceId,
    $core.String? targetDeviceId,
  }) {
    final result = create();
    if (streamId != null) result.streamId = streamId;
    if (kind != null) result.kind = kind;
    if (sourceDeviceId != null) result.sourceDeviceId = sourceDeviceId;
    if (targetDeviceId != null) result.targetDeviceId = targetDeviceId;
    return result;
  }

  StreamEntry._();

  factory StreamEntry.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory StreamEntry.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'StreamEntry',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.diagnostics.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'streamId')
    ..aOS(2, _omitFieldNames ? '' : 'kind')
    ..aOS(3, _omitFieldNames ? '' : 'sourceDeviceId')
    ..aOS(4, _omitFieldNames ? '' : 'targetDeviceId')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StreamEntry clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  StreamEntry copyWith(void Function(StreamEntry) updates) =>
      super.copyWith((message) => updates(message as StreamEntry))
          as StreamEntry;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static StreamEntry create() => StreamEntry._();
  @$core.override
  StreamEntry createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static StreamEntry getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<StreamEntry>(create);
  static StreamEntry? _defaultInstance;

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
}

class RouteEntry extends $pb.GeneratedMessage {
  factory RouteEntry({
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

  RouteEntry._();

  factory RouteEntry.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory RouteEntry.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'RouteEntry',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.diagnostics.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'streamId')
    ..aOS(2, _omitFieldNames ? '' : 'sourceDeviceId')
    ..aOS(3, _omitFieldNames ? '' : 'targetDeviceId')
    ..aOS(4, _omitFieldNames ? '' : 'kind')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RouteEntry clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  RouteEntry copyWith(void Function(RouteEntry) updates) =>
      super.copyWith((message) => updates(message as RouteEntry)) as RouteEntry;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static RouteEntry create() => RouteEntry._();
  @$core.override
  RouteEntry createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static RouteEntry getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<RouteEntry>(create);
  static RouteEntry? _defaultInstance;

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

class WebrtcSignalEntry extends $pb.GeneratedMessage {
  factory WebrtcSignalEntry({
    $fixnum.Int64? unixMs,
    $core.String? streamId,
    $core.String? signalType,
  }) {
    final result = create();
    if (unixMs != null) result.unixMs = unixMs;
    if (streamId != null) result.streamId = streamId;
    if (signalType != null) result.signalType = signalType;
    return result;
  }

  WebrtcSignalEntry._();

  factory WebrtcSignalEntry.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory WebrtcSignalEntry.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'WebrtcSignalEntry',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.diagnostics.v1'),
      createEmptyInstance: create)
    ..aInt64(1, _omitFieldNames ? '' : 'unixMs')
    ..aOS(2, _omitFieldNames ? '' : 'streamId')
    ..aOS(3, _omitFieldNames ? '' : 'signalType')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WebrtcSignalEntry clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  WebrtcSignalEntry copyWith(void Function(WebrtcSignalEntry) updates) =>
      super.copyWith((message) => updates(message as WebrtcSignalEntry))
          as WebrtcSignalEntry;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static WebrtcSignalEntry create() => WebrtcSignalEntry._();
  @$core.override
  WebrtcSignalEntry createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static WebrtcSignalEntry getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<WebrtcSignalEntry>(create);
  static WebrtcSignalEntry? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get unixMs => $_getI64(0);
  @$pb.TagNumber(1)
  set unixMs($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasUnixMs() => $_has(0);
  @$pb.TagNumber(1)
  void clearUnixMs() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get streamId => $_getSZ(1);
  @$pb.TagNumber(2)
  set streamId($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasStreamId() => $_has(1);
  @$pb.TagNumber(2)
  void clearStreamId() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get signalType => $_getSZ(2);
  @$pb.TagNumber(3)
  set signalType($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasSignalType() => $_has(2);
  @$pb.TagNumber(3)
  void clearSignalType() => $_clearField(3);
}

class LogEntry extends $pb.GeneratedMessage {
  factory LogEntry({
    $fixnum.Int64? unixMs,
    $core.String? level,
    $core.String? message,
  }) {
    final result = create();
    if (unixMs != null) result.unixMs = unixMs;
    if (level != null) result.level = level;
    if (message != null) result.message = message;
    return result;
  }

  LogEntry._();

  factory LogEntry.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory LogEntry.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'LogEntry',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.diagnostics.v1'),
      createEmptyInstance: create)
    ..aInt64(1, _omitFieldNames ? '' : 'unixMs')
    ..aOS(2, _omitFieldNames ? '' : 'level')
    ..aOS(3, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LogEntry clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  LogEntry copyWith(void Function(LogEntry) updates) =>
      super.copyWith((message) => updates(message as LogEntry)) as LogEntry;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static LogEntry create() => LogEntry._();
  @$core.override
  LogEntry createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static LogEntry getDefault() =>
      _defaultInstance ??= $pb.GeneratedMessage.$_defaultFor<LogEntry>(create);
  static LogEntry? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get unixMs => $_getI64(0);
  @$pb.TagNumber(1)
  set unixMs($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasUnixMs() => $_has(0);
  @$pb.TagNumber(1)
  void clearUnixMs() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get level => $_getSZ(1);
  @$pb.TagNumber(2)
  set level($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasLevel() => $_has(1);
  @$pb.TagNumber(2)
  void clearLevel() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get message => $_getSZ(2);
  @$pb.TagNumber(3)
  set message($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasMessage() => $_has(2);
  @$pb.TagNumber(3)
  void clearMessage() => $_clearField(3);
}

class ConnectionHealth extends $pb.GeneratedMessage {
  factory ConnectionHealth({
    $fixnum.Int64? lastHeartbeatUnixMs,
    $core.int? reconnectAttempt,
    $core.String? lastStatus,
    $core.double? lastRttMs,
    $core.bool? online,
    $core.Iterable<ControlErrorEntry>? recentControlErrors,
  }) {
    final result = create();
    if (lastHeartbeatUnixMs != null)
      result.lastHeartbeatUnixMs = lastHeartbeatUnixMs;
    if (reconnectAttempt != null) result.reconnectAttempt = reconnectAttempt;
    if (lastStatus != null) result.lastStatus = lastStatus;
    if (lastRttMs != null) result.lastRttMs = lastRttMs;
    if (online != null) result.online = online;
    if (recentControlErrors != null)
      result.recentControlErrors.addAll(recentControlErrors);
    return result;
  }

  ConnectionHealth._();

  factory ConnectionHealth.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ConnectionHealth.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ConnectionHealth',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.diagnostics.v1'),
      createEmptyInstance: create)
    ..aInt64(1, _omitFieldNames ? '' : 'lastHeartbeatUnixMs')
    ..aI(2, _omitFieldNames ? '' : 'reconnectAttempt')
    ..aOS(3, _omitFieldNames ? '' : 'lastStatus')
    ..aD(4, _omitFieldNames ? '' : 'lastRttMs')
    ..aOB(5, _omitFieldNames ? '' : 'online')
    ..pPM<ControlErrorEntry>(6, _omitFieldNames ? '' : 'recentControlErrors',
        subBuilder: ControlErrorEntry.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectionHealth clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ConnectionHealth copyWith(void Function(ConnectionHealth) updates) =>
      super.copyWith((message) => updates(message as ConnectionHealth))
          as ConnectionHealth;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ConnectionHealth create() => ConnectionHealth._();
  @$core.override
  ConnectionHealth createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ConnectionHealth getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ConnectionHealth>(create);
  static ConnectionHealth? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get lastHeartbeatUnixMs => $_getI64(0);
  @$pb.TagNumber(1)
  set lastHeartbeatUnixMs($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasLastHeartbeatUnixMs() => $_has(0);
  @$pb.TagNumber(1)
  void clearLastHeartbeatUnixMs() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.int get reconnectAttempt => $_getIZ(1);
  @$pb.TagNumber(2)
  set reconnectAttempt($core.int value) => $_setSignedInt32(1, value);
  @$pb.TagNumber(2)
  $core.bool hasReconnectAttempt() => $_has(1);
  @$pb.TagNumber(2)
  void clearReconnectAttempt() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get lastStatus => $_getSZ(2);
  @$pb.TagNumber(3)
  set lastStatus($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasLastStatus() => $_has(2);
  @$pb.TagNumber(3)
  void clearLastStatus() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.double get lastRttMs => $_getN(3);
  @$pb.TagNumber(4)
  set lastRttMs($core.double value) => $_setDouble(3, value);
  @$pb.TagNumber(4)
  $core.bool hasLastRttMs() => $_has(3);
  @$pb.TagNumber(4)
  void clearLastRttMs() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.bool get online => $_getBF(4);
  @$pb.TagNumber(5)
  set online($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasOnline() => $_has(4);
  @$pb.TagNumber(5)
  void clearOnline() => $_clearField(5);

  @$pb.TagNumber(6)
  $pb.PbList<ControlErrorEntry> get recentControlErrors => $_getList(5);
}

class ControlErrorEntry extends $pb.GeneratedMessage {
  factory ControlErrorEntry({
    $fixnum.Int64? unixMs,
    $core.String? code,
    $core.String? message,
  }) {
    final result = create();
    if (unixMs != null) result.unixMs = unixMs;
    if (code != null) result.code = code;
    if (message != null) result.message = message;
    return result;
  }

  ControlErrorEntry._();

  factory ControlErrorEntry.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ControlErrorEntry.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ControlErrorEntry',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.diagnostics.v1'),
      createEmptyInstance: create)
    ..aInt64(1, _omitFieldNames ? '' : 'unixMs')
    ..aOS(2, _omitFieldNames ? '' : 'code')
    ..aOS(3, _omitFieldNames ? '' : 'message')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ControlErrorEntry clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ControlErrorEntry copyWith(void Function(ControlErrorEntry) updates) =>
      super.copyWith((message) => updates(message as ControlErrorEntry))
          as ControlErrorEntry;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ControlErrorEntry create() => ControlErrorEntry._();
  @$core.override
  ControlErrorEntry createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ControlErrorEntry getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ControlErrorEntry>(create);
  static ControlErrorEntry? _defaultInstance;

  @$pb.TagNumber(1)
  $fixnum.Int64 get unixMs => $_getI64(0);
  @$pb.TagNumber(1)
  set unixMs($fixnum.Int64 value) => $_setInt64(0, value);
  @$pb.TagNumber(1)
  $core.bool hasUnixMs() => $_has(0);
  @$pb.TagNumber(1)
  void clearUnixMs() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get code => $_getSZ(1);
  @$pb.TagNumber(2)
  set code($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCode() => $_has(1);
  @$pb.TagNumber(2)
  void clearCode() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.String get message => $_getSZ(2);
  @$pb.TagNumber(3)
  set message($core.String value) => $_setString(2, value);
  @$pb.TagNumber(3)
  $core.bool hasMessage() => $_has(2);
  @$pb.TagNumber(3)
  void clearMessage() => $_clearField(3);
}

class HardwareState extends $pb.GeneratedMessage {
  factory HardwareState({
    $core.double? batteryLevel,
    $core.bool? batteryCharging,
    $core.int? screenWidthPx,
    $core.int? screenHeightPx,
    $core.double? devicePixelRatio,
    $core.String? orientation,
    $core.Iterable<$core.MapEntry<$core.String, $core.double>>? sensorSnapshot,
  }) {
    final result = create();
    if (batteryLevel != null) result.batteryLevel = batteryLevel;
    if (batteryCharging != null) result.batteryCharging = batteryCharging;
    if (screenWidthPx != null) result.screenWidthPx = screenWidthPx;
    if (screenHeightPx != null) result.screenHeightPx = screenHeightPx;
    if (devicePixelRatio != null) result.devicePixelRatio = devicePixelRatio;
    if (orientation != null) result.orientation = orientation;
    if (sensorSnapshot != null)
      result.sensorSnapshot.addEntries(sensorSnapshot);
    return result;
  }

  HardwareState._();

  factory HardwareState.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory HardwareState.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'HardwareState',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.diagnostics.v1'),
      createEmptyInstance: create)
    ..aD(1, _omitFieldNames ? '' : 'batteryLevel',
        fieldType: $pb.PbFieldType.OF)
    ..aOB(2, _omitFieldNames ? '' : 'batteryCharging')
    ..aI(3, _omitFieldNames ? '' : 'screenWidthPx')
    ..aI(4, _omitFieldNames ? '' : 'screenHeightPx')
    ..aD(5, _omitFieldNames ? '' : 'devicePixelRatio')
    ..aOS(6, _omitFieldNames ? '' : 'orientation')
    ..m<$core.String, $core.double>(7, _omitFieldNames ? '' : 'sensorSnapshot',
        entryClassName: 'HardwareState.SensorSnapshotEntry',
        keyFieldType: $pb.PbFieldType.OS,
        valueFieldType: $pb.PbFieldType.OD,
        packageName: const $pb.PackageName('terminals.diagnostics.v1'))
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HardwareState clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  HardwareState copyWith(void Function(HardwareState) updates) =>
      super.copyWith((message) => updates(message as HardwareState))
          as HardwareState;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static HardwareState create() => HardwareState._();
  @$core.override
  HardwareState createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static HardwareState getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<HardwareState>(create);
  static HardwareState? _defaultInstance;

  @$pb.TagNumber(1)
  $core.double get batteryLevel => $_getN(0);
  @$pb.TagNumber(1)
  set batteryLevel($core.double value) => $_setFloat(0, value);
  @$pb.TagNumber(1)
  $core.bool hasBatteryLevel() => $_has(0);
  @$pb.TagNumber(1)
  void clearBatteryLevel() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.bool get batteryCharging => $_getBF(1);
  @$pb.TagNumber(2)
  set batteryCharging($core.bool value) => $_setBool(1, value);
  @$pb.TagNumber(2)
  $core.bool hasBatteryCharging() => $_has(1);
  @$pb.TagNumber(2)
  void clearBatteryCharging() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.int get screenWidthPx => $_getIZ(2);
  @$pb.TagNumber(3)
  set screenWidthPx($core.int value) => $_setSignedInt32(2, value);
  @$pb.TagNumber(3)
  $core.bool hasScreenWidthPx() => $_has(2);
  @$pb.TagNumber(3)
  void clearScreenWidthPx() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.int get screenHeightPx => $_getIZ(3);
  @$pb.TagNumber(4)
  set screenHeightPx($core.int value) => $_setSignedInt32(3, value);
  @$pb.TagNumber(4)
  $core.bool hasScreenHeightPx() => $_has(3);
  @$pb.TagNumber(4)
  void clearScreenHeightPx() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.double get devicePixelRatio => $_getN(4);
  @$pb.TagNumber(5)
  set devicePixelRatio($core.double value) => $_setDouble(4, value);
  @$pb.TagNumber(5)
  $core.bool hasDevicePixelRatio() => $_has(4);
  @$pb.TagNumber(5)
  void clearDevicePixelRatio() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get orientation => $_getSZ(5);
  @$pb.TagNumber(6)
  set orientation($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasOrientation() => $_has(5);
  @$pb.TagNumber(6)
  void clearOrientation() => $_clearField(6);

  @$pb.TagNumber(7)
  $pb.PbMap<$core.String, $core.double> get sensorSnapshot => $_getMap(6);
}

class ErrorCapture extends $pb.GeneratedMessage {
  factory ErrorCapture({
    $core.String? lastErrorMessage,
    $core.String? lastErrorStack,
    $fixnum.Int64? lastErrorUnixMs,
  }) {
    final result = create();
    if (lastErrorMessage != null) result.lastErrorMessage = lastErrorMessage;
    if (lastErrorStack != null) result.lastErrorStack = lastErrorStack;
    if (lastErrorUnixMs != null) result.lastErrorUnixMs = lastErrorUnixMs;
    return result;
  }

  ErrorCapture._();

  factory ErrorCapture.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory ErrorCapture.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'ErrorCapture',
      package: const $pb.PackageName(
          _omitMessageNames ? '' : 'terminals.diagnostics.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'lastErrorMessage')
    ..aOS(2, _omitFieldNames ? '' : 'lastErrorStack')
    ..aInt64(3, _omitFieldNames ? '' : 'lastErrorUnixMs')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ErrorCapture clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  ErrorCapture copyWith(void Function(ErrorCapture) updates) =>
      super.copyWith((message) => updates(message as ErrorCapture))
          as ErrorCapture;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static ErrorCapture create() => ErrorCapture._();
  @$core.override
  ErrorCapture createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static ErrorCapture getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<ErrorCapture>(create);
  static ErrorCapture? _defaultInstance;

  /// Most recent caught Flutter/UI error, if any.
  @$pb.TagNumber(1)
  $core.String get lastErrorMessage => $_getSZ(0);
  @$pb.TagNumber(1)
  set lastErrorMessage($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasLastErrorMessage() => $_has(0);
  @$pb.TagNumber(1)
  void clearLastErrorMessage() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.String get lastErrorStack => $_getSZ(1);
  @$pb.TagNumber(2)
  set lastErrorStack($core.String value) => $_setString(1, value);
  @$pb.TagNumber(2)
  $core.bool hasLastErrorStack() => $_has(1);
  @$pb.TagNumber(2)
  void clearLastErrorStack() => $_clearField(2);

  @$pb.TagNumber(3)
  $fixnum.Int64 get lastErrorUnixMs => $_getI64(2);
  @$pb.TagNumber(3)
  set lastErrorUnixMs($fixnum.Int64 value) => $_setInt64(2, value);
  @$pb.TagNumber(3)
  $core.bool hasLastErrorUnixMs() => $_has(2);
  @$pb.TagNumber(3)
  void clearLastErrorUnixMs() => $_clearField(3);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
