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

class PointerAction extends $pb.ProtobufEnum {
  static const PointerAction POINTER_ACTION_UNSPECIFIED =
      PointerAction._(0, _omitEnumNames ? '' : 'POINTER_ACTION_UNSPECIFIED');
  static const PointerAction POINTER_ACTION_DOWN =
      PointerAction._(1, _omitEnumNames ? '' : 'POINTER_ACTION_DOWN');
  static const PointerAction POINTER_ACTION_MOVE =
      PointerAction._(2, _omitEnumNames ? '' : 'POINTER_ACTION_MOVE');
  static const PointerAction POINTER_ACTION_UP =
      PointerAction._(3, _omitEnumNames ? '' : 'POINTER_ACTION_UP');
  static const PointerAction POINTER_ACTION_CANCEL =
      PointerAction._(4, _omitEnumNames ? '' : 'POINTER_ACTION_CANCEL');
  static const PointerAction POINTER_ACTION_SCROLL =
      PointerAction._(5, _omitEnumNames ? '' : 'POINTER_ACTION_SCROLL');

  static const $core.List<PointerAction> values = <PointerAction>[
    POINTER_ACTION_UNSPECIFIED,
    POINTER_ACTION_DOWN,
    POINTER_ACTION_MOVE,
    POINTER_ACTION_UP,
    POINTER_ACTION_CANCEL,
    POINTER_ACTION_SCROLL,
  ];

  static final $core.List<PointerAction?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 5);
  static PointerAction? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const PointerAction._(super.value, super.name);
}

class TouchAction extends $pb.ProtobufEnum {
  static const TouchAction TOUCH_ACTION_UNSPECIFIED =
      TouchAction._(0, _omitEnumNames ? '' : 'TOUCH_ACTION_UNSPECIFIED');
  static const TouchAction TOUCH_ACTION_START =
      TouchAction._(1, _omitEnumNames ? '' : 'TOUCH_ACTION_START');
  static const TouchAction TOUCH_ACTION_MOVE =
      TouchAction._(2, _omitEnumNames ? '' : 'TOUCH_ACTION_MOVE');
  static const TouchAction TOUCH_ACTION_END =
      TouchAction._(3, _omitEnumNames ? '' : 'TOUCH_ACTION_END');
  static const TouchAction TOUCH_ACTION_CANCEL =
      TouchAction._(4, _omitEnumNames ? '' : 'TOUCH_ACTION_CANCEL');

  static const $core.List<TouchAction> values = <TouchAction>[
    TOUCH_ACTION_UNSPECIFIED,
    TOUCH_ACTION_START,
    TOUCH_ACTION_MOVE,
    TOUCH_ACTION_END,
    TOUCH_ACTION_CANCEL,
  ];

  static final $core.List<TouchAction?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 4);
  static TouchAction? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const TouchAction._(super.value, super.name);
}

class ExecPolicy extends $pb.ProtobufEnum {
  static const ExecPolicy EXEC_POLICY_UNSPECIFIED =
      ExecPolicy._(0, _omitEnumNames ? '' : 'EXEC_POLICY_UNSPECIFIED');
  static const ExecPolicy EXEC_POLICY_AUTO =
      ExecPolicy._(1, _omitEnumNames ? '' : 'EXEC_POLICY_AUTO');
  static const ExecPolicy EXEC_POLICY_PREFER_CLIENT =
      ExecPolicy._(2, _omitEnumNames ? '' : 'EXEC_POLICY_PREFER_CLIENT');
  static const ExecPolicy EXEC_POLICY_REQUIRE_CLIENT =
      ExecPolicy._(3, _omitEnumNames ? '' : 'EXEC_POLICY_REQUIRE_CLIENT');
  static const ExecPolicy EXEC_POLICY_SERVER_ONLY =
      ExecPolicy._(4, _omitEnumNames ? '' : 'EXEC_POLICY_SERVER_ONLY');

  static const $core.List<ExecPolicy> values = <ExecPolicy>[
    EXEC_POLICY_UNSPECIFIED,
    EXEC_POLICY_AUTO,
    EXEC_POLICY_PREFER_CLIENT,
    EXEC_POLICY_REQUIRE_CLIENT,
    EXEC_POLICY_SERVER_ONLY,
  ];

  static final $core.List<ExecPolicy?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 4);
  static ExecPolicy? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const ExecPolicy._(super.value, super.name);
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
