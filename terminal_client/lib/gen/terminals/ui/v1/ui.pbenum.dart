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

class ScrollDirection extends $pb.ProtobufEnum {
  static const ScrollDirection SCROLL_DIRECTION_UNSPECIFIED = ScrollDirection._(
      0, _omitEnumNames ? '' : 'SCROLL_DIRECTION_UNSPECIFIED');
  static const ScrollDirection SCROLL_DIRECTION_VERTICAL =
      ScrollDirection._(1, _omitEnumNames ? '' : 'SCROLL_DIRECTION_VERTICAL');
  static const ScrollDirection SCROLL_DIRECTION_HORIZONTAL =
      ScrollDirection._(2, _omitEnumNames ? '' : 'SCROLL_DIRECTION_HORIZONTAL');

  static const $core.List<ScrollDirection> values = <ScrollDirection>[
    SCROLL_DIRECTION_UNSPECIFIED,
    SCROLL_DIRECTION_VERTICAL,
    SCROLL_DIRECTION_HORIZONTAL,
  ];

  static final $core.List<ScrollDirection?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 2);
  static ScrollDirection? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const ScrollDirection._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
