// This is a generated file - do not edit.
//
// Generated from terminals/control/v1/control.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

class CommandAction extends $pb.ProtobufEnum {
  static const CommandAction COMMAND_ACTION_UNSPECIFIED =
      CommandAction._(0, _omitEnumNames ? '' : 'COMMAND_ACTION_UNSPECIFIED');
  static const CommandAction COMMAND_ACTION_START =
      CommandAction._(1, _omitEnumNames ? '' : 'COMMAND_ACTION_START');
  static const CommandAction COMMAND_ACTION_STOP =
      CommandAction._(2, _omitEnumNames ? '' : 'COMMAND_ACTION_STOP');

  static const $core.List<CommandAction> values = <CommandAction>[
    COMMAND_ACTION_UNSPECIFIED,
    COMMAND_ACTION_START,
    COMMAND_ACTION_STOP,
  ];

  static final $core.List<CommandAction?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 2);
  static CommandAction? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const CommandAction._(super.value, super.name);
}

class CommandKind extends $pb.ProtobufEnum {
  static const CommandKind COMMAND_KIND_UNSPECIFIED =
      CommandKind._(0, _omitEnumNames ? '' : 'COMMAND_KIND_UNSPECIFIED');
  static const CommandKind COMMAND_KIND_VOICE =
      CommandKind._(1, _omitEnumNames ? '' : 'COMMAND_KIND_VOICE');
  static const CommandKind COMMAND_KIND_MANUAL =
      CommandKind._(2, _omitEnumNames ? '' : 'COMMAND_KIND_MANUAL');
  static const CommandKind COMMAND_KIND_SYSTEM =
      CommandKind._(3, _omitEnumNames ? '' : 'COMMAND_KIND_SYSTEM');

  static const $core.List<CommandKind> values = <CommandKind>[
    COMMAND_KIND_UNSPECIFIED,
    COMMAND_KIND_VOICE,
    COMMAND_KIND_MANUAL,
    COMMAND_KIND_SYSTEM,
  ];

  static final $core.List<CommandKind?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static CommandKind? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const CommandKind._(super.value, super.name);
}

class ControlErrorCode extends $pb.ProtobufEnum {
  static const ControlErrorCode CONTROL_ERROR_CODE_UNSPECIFIED =
      ControlErrorCode._(
          0, _omitEnumNames ? '' : 'CONTROL_ERROR_CODE_UNSPECIFIED');
  static const ControlErrorCode CONTROL_ERROR_CODE_INVALID_CLIENT_MESSAGE =
      ControlErrorCode._(
          1, _omitEnumNames ? '' : 'CONTROL_ERROR_CODE_INVALID_CLIENT_MESSAGE');
  static const ControlErrorCode CONTROL_ERROR_CODE_INVALID_COMMAND_ACTION =
      ControlErrorCode._(
          2, _omitEnumNames ? '' : 'CONTROL_ERROR_CODE_INVALID_COMMAND_ACTION');
  static const ControlErrorCode CONTROL_ERROR_CODE_INVALID_COMMAND_KIND =
      ControlErrorCode._(
          3, _omitEnumNames ? '' : 'CONTROL_ERROR_CODE_INVALID_COMMAND_KIND');
  static const ControlErrorCode CONTROL_ERROR_CODE_MISSING_COMMAND_INTENT =
      ControlErrorCode._(
          4, _omitEnumNames ? '' : 'CONTROL_ERROR_CODE_MISSING_COMMAND_INTENT');
  static const ControlErrorCode CONTROL_ERROR_CODE_MISSING_COMMAND_TEXT =
      ControlErrorCode._(
          5, _omitEnumNames ? '' : 'CONTROL_ERROR_CODE_MISSING_COMMAND_TEXT');
  static const ControlErrorCode CONTROL_ERROR_CODE_MISSING_COMMAND_DEVICE_ID =
      ControlErrorCode._(6,
          _omitEnumNames ? '' : 'CONTROL_ERROR_CODE_MISSING_COMMAND_DEVICE_ID');
  static const ControlErrorCode CONTROL_ERROR_CODE_PROTOCOL_VIOLATION =
      ControlErrorCode._(
          7, _omitEnumNames ? '' : 'CONTROL_ERROR_CODE_PROTOCOL_VIOLATION');
  static const ControlErrorCode CONTROL_ERROR_CODE_UNKNOWN = ControlErrorCode._(
      99, _omitEnumNames ? '' : 'CONTROL_ERROR_CODE_UNKNOWN');

  static const $core.List<ControlErrorCode> values = <ControlErrorCode>[
    CONTROL_ERROR_CODE_UNSPECIFIED,
    CONTROL_ERROR_CODE_INVALID_CLIENT_MESSAGE,
    CONTROL_ERROR_CODE_INVALID_COMMAND_ACTION,
    CONTROL_ERROR_CODE_INVALID_COMMAND_KIND,
    CONTROL_ERROR_CODE_MISSING_COMMAND_INTENT,
    CONTROL_ERROR_CODE_MISSING_COMMAND_TEXT,
    CONTROL_ERROR_CODE_MISSING_COMMAND_DEVICE_ID,
    CONTROL_ERROR_CODE_PROTOCOL_VIOLATION,
    CONTROL_ERROR_CODE_UNKNOWN,
  ];

  static final $core.Map<$core.int, ControlErrorCode> _byValue =
      $pb.ProtobufEnum.initByValue(values);
  static ControlErrorCode? valueOf($core.int value) => _byValue[value];

  const ControlErrorCode._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
