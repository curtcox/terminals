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

import 'package:protobuf/protobuf.dart' as $pb;

/// Source describes which entry point produced a bug report.
class BugReportSource extends $pb.ProtobufEnum {
  static const BugReportSource BUG_REPORT_SOURCE_UNSPECIFIED =
      BugReportSource._(
          0, _omitEnumNames ? '' : 'BUG_REPORT_SOURCE_UNSPECIFIED');
  static const BugReportSource BUG_REPORT_SOURCE_SCREEN_BUTTON =
      BugReportSource._(
          1, _omitEnumNames ? '' : 'BUG_REPORT_SOURCE_SCREEN_BUTTON');
  static const BugReportSource BUG_REPORT_SOURCE_GESTURE =
      BugReportSource._(2, _omitEnumNames ? '' : 'BUG_REPORT_SOURCE_GESTURE');
  static const BugReportSource BUG_REPORT_SOURCE_SHAKE =
      BugReportSource._(3, _omitEnumNames ? '' : 'BUG_REPORT_SOURCE_SHAKE');
  static const BugReportSource BUG_REPORT_SOURCE_KEYBOARD =
      BugReportSource._(4, _omitEnumNames ? '' : 'BUG_REPORT_SOURCE_KEYBOARD');
  static const BugReportSource BUG_REPORT_SOURCE_VOICE =
      BugReportSource._(5, _omitEnumNames ? '' : 'BUG_REPORT_SOURCE_VOICE');
  static const BugReportSource BUG_REPORT_SOURCE_QR =
      BugReportSource._(6, _omitEnumNames ? '' : 'BUG_REPORT_SOURCE_QR');
  static const BugReportSource BUG_REPORT_SOURCE_NFC =
      BugReportSource._(7, _omitEnumNames ? '' : 'BUG_REPORT_SOURCE_NFC');
  static const BugReportSource BUG_REPORT_SOURCE_ADMIN =
      BugReportSource._(8, _omitEnumNames ? '' : 'BUG_REPORT_SOURCE_ADMIN');
  static const BugReportSource BUG_REPORT_SOURCE_SIP =
      BugReportSource._(9, _omitEnumNames ? '' : 'BUG_REPORT_SOURCE_SIP');
  static const BugReportSource BUG_REPORT_SOURCE_WEBHOOK =
      BugReportSource._(10, _omitEnumNames ? '' : 'BUG_REPORT_SOURCE_WEBHOOK');
  static const BugReportSource BUG_REPORT_SOURCE_AUTODETECT = BugReportSource._(
      11, _omitEnumNames ? '' : 'BUG_REPORT_SOURCE_AUTODETECT');
  static const BugReportSource BUG_REPORT_SOURCE_OTHER =
      BugReportSource._(99, _omitEnumNames ? '' : 'BUG_REPORT_SOURCE_OTHER');

  static const $core.List<BugReportSource> values = <BugReportSource>[
    BUG_REPORT_SOURCE_UNSPECIFIED,
    BUG_REPORT_SOURCE_SCREEN_BUTTON,
    BUG_REPORT_SOURCE_GESTURE,
    BUG_REPORT_SOURCE_SHAKE,
    BUG_REPORT_SOURCE_KEYBOARD,
    BUG_REPORT_SOURCE_VOICE,
    BUG_REPORT_SOURCE_QR,
    BUG_REPORT_SOURCE_NFC,
    BUG_REPORT_SOURCE_ADMIN,
    BUG_REPORT_SOURCE_SIP,
    BUG_REPORT_SOURCE_WEBHOOK,
    BUG_REPORT_SOURCE_AUTODETECT,
    BUG_REPORT_SOURCE_OTHER,
  ];

  static final $core.Map<$core.int, BugReportSource> _byValue =
      $pb.ProtobufEnum.initByValue(values);
  static BugReportSource? valueOf($core.int value) => _byValue[value];

  const BugReportSource._(super.value, super.name);
}

/// Status reflects the persisted state of a bug report after server handling.
class BugReportStatus extends $pb.ProtobufEnum {
  static const BugReportStatus BUG_REPORT_STATUS_UNSPECIFIED =
      BugReportStatus._(
          0, _omitEnumNames ? '' : 'BUG_REPORT_STATUS_UNSPECIFIED');
  static const BugReportStatus BUG_REPORT_STATUS_FILED =
      BugReportStatus._(1, _omitEnumNames ? '' : 'BUG_REPORT_STATUS_FILED');
  static const BugReportStatus BUG_REPORT_STATUS_MERGED_WITH_AUTODETECT =
      BugReportStatus._(
          2, _omitEnumNames ? '' : 'BUG_REPORT_STATUS_MERGED_WITH_AUTODETECT');
  static const BugReportStatus BUG_REPORT_STATUS_REJECTED =
      BugReportStatus._(3, _omitEnumNames ? '' : 'BUG_REPORT_STATUS_REJECTED');

  static const $core.List<BugReportStatus> values = <BugReportStatus>[
    BUG_REPORT_STATUS_UNSPECIFIED,
    BUG_REPORT_STATUS_FILED,
    BUG_REPORT_STATUS_MERGED_WITH_AUTODETECT,
    BUG_REPORT_STATUS_REJECTED,
  ];

  static final $core.List<BugReportStatus?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static BugReportStatus? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const BugReportStatus._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
