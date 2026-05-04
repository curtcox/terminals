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

import 'package:protobuf/protobuf.dart' as $pb;

class StreamKind extends $pb.ProtobufEnum {
  static const StreamKind STREAM_KIND_UNSPECIFIED =
      StreamKind._(0, _omitEnumNames ? '' : 'STREAM_KIND_UNSPECIFIED');
  static const StreamKind STREAM_KIND_AUDIO =
      StreamKind._(1, _omitEnumNames ? '' : 'STREAM_KIND_AUDIO');
  static const StreamKind STREAM_KIND_VIDEO =
      StreamKind._(2, _omitEnumNames ? '' : 'STREAM_KIND_VIDEO');
  static const StreamKind STREAM_KIND_SENSOR =
      StreamKind._(3, _omitEnumNames ? '' : 'STREAM_KIND_SENSOR');
  static const StreamKind STREAM_KIND_DATA =
      StreamKind._(4, _omitEnumNames ? '' : 'STREAM_KIND_DATA');

  static const $core.List<StreamKind> values = <StreamKind>[
    STREAM_KIND_UNSPECIFIED,
    STREAM_KIND_AUDIO,
    STREAM_KIND_VIDEO,
    STREAM_KIND_SENSOR,
    STREAM_KIND_DATA,
  ];

  static final $core.List<StreamKind?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 4);
  static StreamKind? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const StreamKind._(super.value, super.name);
}

class FlowState extends $pb.ProtobufEnum {
  static const FlowState FLOW_STATE_UNSPECIFIED =
      FlowState._(0, _omitEnumNames ? '' : 'FLOW_STATE_UNSPECIFIED');
  static const FlowState FLOW_STATE_STARTING =
      FlowState._(1, _omitEnumNames ? '' : 'FLOW_STATE_STARTING');
  static const FlowState FLOW_STATE_RUNNING =
      FlowState._(2, _omitEnumNames ? '' : 'FLOW_STATE_RUNNING');
  static const FlowState FLOW_STATE_DEGRADED =
      FlowState._(3, _omitEnumNames ? '' : 'FLOW_STATE_DEGRADED');
  static const FlowState FLOW_STATE_STOPPING =
      FlowState._(4, _omitEnumNames ? '' : 'FLOW_STATE_STOPPING');
  static const FlowState FLOW_STATE_STOPPED =
      FlowState._(5, _omitEnumNames ? '' : 'FLOW_STATE_STOPPED');
  static const FlowState FLOW_STATE_FAILED =
      FlowState._(6, _omitEnumNames ? '' : 'FLOW_STATE_FAILED');

  static const $core.List<FlowState> values = <FlowState>[
    FLOW_STATE_UNSPECIFIED,
    FLOW_STATE_STARTING,
    FLOW_STATE_RUNNING,
    FLOW_STATE_DEGRADED,
    FLOW_STATE_STOPPING,
    FLOW_STATE_STOPPED,
    FLOW_STATE_FAILED,
  ];

  static final $core.List<FlowState?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 6);
  static FlowState? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const FlowState._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
