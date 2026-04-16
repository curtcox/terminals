// This is a generated file - do not edit.
//
// Generated from terminals/diagnostics/v1/diagnostics.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports
// ignore_for_file: unused_import

import 'dart:convert' as $convert;
import 'dart:core' as $core;
import 'dart:typed_data' as $typed_data;

@$core.Deprecated('Use bugReportSourceDescriptor instead')
const BugReportSource$json = {
  '1': 'BugReportSource',
  '2': [
    {'1': 'BUG_REPORT_SOURCE_UNSPECIFIED', '2': 0},
    {'1': 'BUG_REPORT_SOURCE_SCREEN_BUTTON', '2': 1},
    {'1': 'BUG_REPORT_SOURCE_GESTURE', '2': 2},
    {'1': 'BUG_REPORT_SOURCE_SHAKE', '2': 3},
    {'1': 'BUG_REPORT_SOURCE_KEYBOARD', '2': 4},
    {'1': 'BUG_REPORT_SOURCE_VOICE', '2': 5},
    {'1': 'BUG_REPORT_SOURCE_QR', '2': 6},
    {'1': 'BUG_REPORT_SOURCE_NFC', '2': 7},
    {'1': 'BUG_REPORT_SOURCE_ADMIN', '2': 8},
    {'1': 'BUG_REPORT_SOURCE_SIP', '2': 9},
    {'1': 'BUG_REPORT_SOURCE_WEBHOOK', '2': 10},
    {'1': 'BUG_REPORT_SOURCE_AUTODETECT', '2': 11},
    {'1': 'BUG_REPORT_SOURCE_OTHER', '2': 99},
  ],
};

/// Descriptor for `BugReportSource`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List bugReportSourceDescriptor = $convert.base64Decode(
    'Cg9CdWdSZXBvcnRTb3VyY2USIQodQlVHX1JFUE9SVF9TT1VSQ0VfVU5TUEVDSUZJRUQQABIjCh'
    '9CVUdfUkVQT1JUX1NPVVJDRV9TQ1JFRU5fQlVUVE9OEAESHQoZQlVHX1JFUE9SVF9TT1VSQ0Vf'
    'R0VTVFVSRRACEhsKF0JVR19SRVBPUlRfU09VUkNFX1NIQUtFEAMSHgoaQlVHX1JFUE9SVF9TT1'
    'VSQ0VfS0VZQk9BUkQQBBIbChdCVUdfUkVQT1JUX1NPVVJDRV9WT0lDRRAFEhgKFEJVR19SRVBP'
    'UlRfU09VUkNFX1FSEAYSGQoVQlVHX1JFUE9SVF9TT1VSQ0VfTkZDEAcSGwoXQlVHX1JFUE9SVF'
    '9TT1VSQ0VfQURNSU4QCBIZChVCVUdfUkVQT1JUX1NPVVJDRV9TSVAQCRIdChlCVUdfUkVQT1JU'
    'X1NPVVJDRV9XRUJIT09LEAoSIAocQlVHX1JFUE9SVF9TT1VSQ0VfQVVUT0RFVEVDVBALEhsKF0'
    'JVR19SRVBPUlRfU09VUkNFX09USEVSEGM=');

@$core.Deprecated('Use bugReportStatusDescriptor instead')
const BugReportStatus$json = {
  '1': 'BugReportStatus',
  '2': [
    {'1': 'BUG_REPORT_STATUS_UNSPECIFIED', '2': 0},
    {'1': 'BUG_REPORT_STATUS_FILED', '2': 1},
    {'1': 'BUG_REPORT_STATUS_MERGED_WITH_AUTODETECT', '2': 2},
    {'1': 'BUG_REPORT_STATUS_REJECTED', '2': 3},
  ],
};

/// Descriptor for `BugReportStatus`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List bugReportStatusDescriptor = $convert.base64Decode(
    'Cg9CdWdSZXBvcnRTdGF0dXMSIQodQlVHX1JFUE9SVF9TVEFUVVNfVU5TUEVDSUZJRUQQABIbCh'
    'dCVUdfUkVQT1JUX1NUQVRVU19GSUxFRBABEiwKKEJVR19SRVBPUlRfU1RBVFVTX01FUkdFRF9X'
    'SVRIX0FVVE9ERVRFQ1QQAhIeChpCVUdfUkVQT1JUX1NUQVRVU19SRUpFQ1RFRBAD');

@$core.Deprecated('Use bugReportDescriptor instead')
const BugReport$json = {
  '1': 'BugReport',
  '2': [
    {'1': 'report_id', '3': 1, '4': 1, '5': 9, '10': 'reportId'},
    {
      '1': 'reporter_device_id',
      '3': 2,
      '4': 1,
      '5': 9,
      '10': 'reporterDeviceId'
    },
    {'1': 'subject_device_id', '3': 3, '4': 1, '5': 9, '10': 'subjectDeviceId'},
    {
      '1': 'source',
      '3': 4,
      '4': 1,
      '5': 14,
      '6': '.terminals.diagnostics.v1.BugReportSource',
      '10': 'source'
    },
    {'1': 'description', '3': 5, '4': 1, '5': 9, '10': 'description'},
    {'1': 'tags', '3': 6, '4': 3, '5': 9, '10': 'tags'},
    {'1': 'timestamp_unix_ms', '3': 7, '4': 1, '5': 3, '10': 'timestampUnixMs'},
    {
      '1': 'client_context',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.terminals.diagnostics.v1.ClientContext',
      '10': 'clientContext'
    },
    {'1': 'screenshot_png', '3': 9, '4': 1, '5': 12, '10': 'screenshotPng'},
    {'1': 'audio_wav', '3': 10, '4': 1, '5': 12, '10': 'audioWav'},
    {
      '1': 'source_hints',
      '3': 11,
      '4': 3,
      '5': 11,
      '6': '.terminals.diagnostics.v1.BugReport.SourceHintsEntry',
      '10': 'sourceHints'
    },
  ],
  '3': [BugReport_SourceHintsEntry$json],
};

@$core.Deprecated('Use bugReportDescriptor instead')
const BugReport_SourceHintsEntry$json = {
  '1': 'SourceHintsEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `BugReport`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List bugReportDescriptor = $convert.base64Decode(
    'CglCdWdSZXBvcnQSGwoJcmVwb3J0X2lkGAEgASgJUghyZXBvcnRJZBIsChJyZXBvcnRlcl9kZX'
    'ZpY2VfaWQYAiABKAlSEHJlcG9ydGVyRGV2aWNlSWQSKgoRc3ViamVjdF9kZXZpY2VfaWQYAyAB'
    'KAlSD3N1YmplY3REZXZpY2VJZBJBCgZzb3VyY2UYBCABKA4yKS50ZXJtaW5hbHMuZGlhZ25vc3'
    'RpY3MudjEuQnVnUmVwb3J0U291cmNlUgZzb3VyY2USIAoLZGVzY3JpcHRpb24YBSABKAlSC2Rl'
    'c2NyaXB0aW9uEhIKBHRhZ3MYBiADKAlSBHRhZ3MSKgoRdGltZXN0YW1wX3VuaXhfbXMYByABKA'
    'NSD3RpbWVzdGFtcFVuaXhNcxJOCg5jbGllbnRfY29udGV4dBgIIAEoCzInLnRlcm1pbmFscy5k'
    'aWFnbm9zdGljcy52MS5DbGllbnRDb250ZXh0Ug1jbGllbnRDb250ZXh0EiUKDnNjcmVlbnNob3'
    'RfcG5nGAkgASgMUg1zY3JlZW5zaG90UG5nEhsKCWF1ZGlvX3dhdhgKIAEoDFIIYXVkaW9XYXYS'
    'VwoMc291cmNlX2hpbnRzGAsgAygLMjQudGVybWluYWxzLmRpYWdub3N0aWNzLnYxLkJ1Z1JlcG'
    '9ydC5Tb3VyY2VIaW50c0VudHJ5Ugtzb3VyY2VIaW50cxo+ChBTb3VyY2VIaW50c0VudHJ5EhAK'
    'A2tleRgBIAEoCVIDa2V5EhQKBXZhbHVlGAIgASgJUgV2YWx1ZToCOAE=');

@$core.Deprecated('Use bugReportAckDescriptor instead')
const BugReportAck$json = {
  '1': 'BugReportAck',
  '2': [
    {'1': 'report_id', '3': 1, '4': 1, '5': 9, '10': 'reportId'},
    {'1': 'correlation_id', '3': 2, '4': 1, '5': 9, '10': 'correlationId'},
    {
      '1': 'status',
      '3': 3,
      '4': 1,
      '5': 14,
      '6': '.terminals.diagnostics.v1.BugReportStatus',
      '10': 'status'
    },
    {'1': 'report_path', '3': 4, '4': 1, '5': 9, '10': 'reportPath'},
    {
      '1': 'merged_autodetect_report_id',
      '3': 5,
      '4': 1,
      '5': 9,
      '10': 'mergedAutodetectReportId'
    },
    {'1': 'message', '3': 6, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `BugReportAck`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List bugReportAckDescriptor = $convert.base64Decode(
    'CgxCdWdSZXBvcnRBY2sSGwoJcmVwb3J0X2lkGAEgASgJUghyZXBvcnRJZBIlCg5jb3JyZWxhdG'
    'lvbl9pZBgCIAEoCVINY29ycmVsYXRpb25JZBJBCgZzdGF0dXMYAyABKA4yKS50ZXJtaW5hbHMu'
    'ZGlhZ25vc3RpY3MudjEuQnVnUmVwb3J0U3RhdHVzUgZzdGF0dXMSHwoLcmVwb3J0X3BhdGgYBC'
    'ABKAlSCnJlcG9ydFBhdGgSPQobbWVyZ2VkX2F1dG9kZXRlY3RfcmVwb3J0X2lkGAUgASgJUhht'
    'ZXJnZWRBdXRvZGV0ZWN0UmVwb3J0SWQSGAoHbWVzc2FnZRgGIAEoCVIHbWVzc2FnZQ==');

@$core.Deprecated('Use clientContextDescriptor instead')
const ClientContext$json = {
  '1': 'ClientContext',
  '2': [
    {
      '1': 'identity',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.terminals.diagnostics.v1.ClientIdentity',
      '10': 'identity'
    },
    {
      '1': 'capabilities',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.terminals.capabilities.v1.DeviceCapabilities',
      '10': 'capabilities'
    },
    {
      '1': 'runtime',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.terminals.diagnostics.v1.RuntimeState',
      '10': 'runtime'
    },
    {
      '1': 'connection',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.terminals.diagnostics.v1.ConnectionHealth',
      '10': 'connection'
    },
    {
      '1': 'hardware',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.terminals.diagnostics.v1.HardwareState',
      '10': 'hardware'
    },
    {
      '1': 'error_capture',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.terminals.diagnostics.v1.ErrorCapture',
      '10': 'errorCapture'
    },
  ],
};

/// Descriptor for `ClientContext`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientContextDescriptor = $convert.base64Decode(
    'Cg1DbGllbnRDb250ZXh0EkQKCGlkZW50aXR5GAEgASgLMigudGVybWluYWxzLmRpYWdub3N0aW'
    'NzLnYxLkNsaWVudElkZW50aXR5UghpZGVudGl0eRJRCgxjYXBhYmlsaXRpZXMYAiABKAsyLS50'
    'ZXJtaW5hbHMuY2FwYWJpbGl0aWVzLnYxLkRldmljZUNhcGFiaWxpdGllc1IMY2FwYWJpbGl0aW'
    'VzEkAKB3J1bnRpbWUYAyABKAsyJi50ZXJtaW5hbHMuZGlhZ25vc3RpY3MudjEuUnVudGltZVN0'
    'YXRlUgdydW50aW1lEkoKCmNvbm5lY3Rpb24YBCABKAsyKi50ZXJtaW5hbHMuZGlhZ25vc3RpY3'
    'MudjEuQ29ubmVjdGlvbkhlYWx0aFIKY29ubmVjdGlvbhJDCghoYXJkd2FyZRgFIAEoCzInLnRl'
    'cm1pbmFscy5kaWFnbm9zdGljcy52MS5IYXJkd2FyZVN0YXRlUghoYXJkd2FyZRJLCg1lcnJvcl'
    '9jYXB0dXJlGAYgASgLMiYudGVybWluYWxzLmRpYWdub3N0aWNzLnYxLkVycm9yQ2FwdHVyZVIM'
    'ZXJyb3JDYXB0dXJl');

@$core.Deprecated('Use clientIdentityDescriptor instead')
const ClientIdentity$json = {
  '1': 'ClientIdentity',
  '2': [
    {'1': 'device_id', '3': 1, '4': 1, '5': 9, '10': 'deviceId'},
    {'1': 'device_name', '3': 2, '4': 1, '5': 9, '10': 'deviceName'},
    {'1': 'device_type', '3': 3, '4': 1, '5': 9, '10': 'deviceType'},
    {'1': 'platform', '3': 4, '4': 1, '5': 9, '10': 'platform'},
    {'1': 'client_version', '3': 5, '4': 1, '5': 9, '10': 'clientVersion'},
    {'1': 'client_git_sha', '3': 6, '4': 1, '5': 9, '10': 'clientGitSha'},
    {
      '1': 'client_build_unix_ms',
      '3': 7,
      '4': 1,
      '5': 3,
      '10': 'clientBuildUnixMs'
    },
    {'1': 'os_version', '3': 8, '4': 1, '5': 9, '10': 'osVersion'},
    {'1': 'locale', '3': 9, '4': 1, '5': 9, '10': 'locale'},
    {'1': 'timezone', '3': 10, '4': 1, '5': 9, '10': 'timezone'},
    {'1': 'clock_offset_ms', '3': 11, '4': 1, '5': 3, '10': 'clockOffsetMs'},
  ],
};

/// Descriptor for `ClientIdentity`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clientIdentityDescriptor = $convert.base64Decode(
    'Cg5DbGllbnRJZGVudGl0eRIbCglkZXZpY2VfaWQYASABKAlSCGRldmljZUlkEh8KC2RldmljZV'
    '9uYW1lGAIgASgJUgpkZXZpY2VOYW1lEh8KC2RldmljZV90eXBlGAMgASgJUgpkZXZpY2VUeXBl'
    'EhoKCHBsYXRmb3JtGAQgASgJUghwbGF0Zm9ybRIlCg5jbGllbnRfdmVyc2lvbhgFIAEoCVINY2'
    'xpZW50VmVyc2lvbhIkCg5jbGllbnRfZ2l0X3NoYRgGIAEoCVIMY2xpZW50R2l0U2hhEi8KFGNs'
    'aWVudF9idWlsZF91bml4X21zGAcgASgDUhFjbGllbnRCdWlsZFVuaXhNcxIdCgpvc192ZXJzaW'
    '9uGAggASgJUglvc1ZlcnNpb24SFgoGbG9jYWxlGAkgASgJUgZsb2NhbGUSGgoIdGltZXpvbmUY'
    'CiABKAlSCHRpbWV6b25lEiYKD2Nsb2NrX29mZnNldF9tcxgLIAEoA1INY2xvY2tPZmZzZXRNcw'
    '==');

@$core.Deprecated('Use runtimeStateDescriptor instead')
const RuntimeState$json = {
  '1': 'RuntimeState',
  '2': [
    {'1': 'scenario_ids', '3': 1, '4': 3, '5': 9, '10': 'scenarioIds'},
    {'1': 'activation_ids', '3': 2, '4': 3, '5': 9, '10': 'activationIds'},
    {
      '1': 'active_ui_root',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.Node',
      '10': 'activeUiRoot'
    },
    {
      '1': 'recent_ui_updates',
      '3': 4,
      '4': 3,
      '5': 11,
      '6': '.terminals.diagnostics.v1.UiEventEntry',
      '10': 'recentUiUpdates'
    },
    {
      '1': 'recent_ui_actions',
      '3': 5,
      '4': 3,
      '5': 11,
      '6': '.terminals.diagnostics.v1.UiActionEntry',
      '10': 'recentUiActions'
    },
    {
      '1': 'active_streams',
      '3': 6,
      '4': 3,
      '5': 11,
      '6': '.terminals.diagnostics.v1.StreamEntry',
      '10': 'activeStreams'
    },
    {
      '1': 'active_routes',
      '3': 7,
      '4': 3,
      '5': 11,
      '6': '.terminals.diagnostics.v1.RouteEntry',
      '10': 'activeRoutes'
    },
    {
      '1': 'recent_webrtc_signals',
      '3': 8,
      '4': 3,
      '5': 11,
      '6': '.terminals.diagnostics.v1.WebrtcSignalEntry',
      '10': 'recentWebrtcSignals'
    },
    {
      '1': 'recent_logs',
      '3': 9,
      '4': 3,
      '5': 11,
      '6': '.terminals.diagnostics.v1.LogEntry',
      '10': 'recentLogs'
    },
  ],
};

/// Descriptor for `RuntimeState`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List runtimeStateDescriptor = $convert.base64Decode(
    'CgxSdW50aW1lU3RhdGUSIQoMc2NlbmFyaW9faWRzGAEgAygJUgtzY2VuYXJpb0lkcxIlCg5hY3'
    'RpdmF0aW9uX2lkcxgCIAMoCVINYWN0aXZhdGlvbklkcxI7Cg5hY3RpdmVfdWlfcm9vdBgDIAEo'
    'CzIVLnRlcm1pbmFscy51aS52MS5Ob2RlUgxhY3RpdmVVaVJvb3QSUgoRcmVjZW50X3VpX3VwZG'
    'F0ZXMYBCADKAsyJi50ZXJtaW5hbHMuZGlhZ25vc3RpY3MudjEuVWlFdmVudEVudHJ5Ug9yZWNl'
    'bnRVaVVwZGF0ZXMSUwoRcmVjZW50X3VpX2FjdGlvbnMYBSADKAsyJy50ZXJtaW5hbHMuZGlhZ2'
    '5vc3RpY3MudjEuVWlBY3Rpb25FbnRyeVIPcmVjZW50VWlBY3Rpb25zEkwKDmFjdGl2ZV9zdHJl'
    'YW1zGAYgAygLMiUudGVybWluYWxzLmRpYWdub3N0aWNzLnYxLlN0cmVhbUVudHJ5Ug1hY3Rpdm'
    'VTdHJlYW1zEkkKDWFjdGl2ZV9yb3V0ZXMYByADKAsyJC50ZXJtaW5hbHMuZGlhZ25vc3RpY3Mu'
    'djEuUm91dGVFbnRyeVIMYWN0aXZlUm91dGVzEl8KFXJlY2VudF93ZWJydGNfc2lnbmFscxgIIA'
    'MoCzIrLnRlcm1pbmFscy5kaWFnbm9zdGljcy52MS5XZWJydGNTaWduYWxFbnRyeVITcmVjZW50'
    'V2VicnRjU2lnbmFscxJDCgtyZWNlbnRfbG9ncxgJIAMoCzIiLnRlcm1pbmFscy5kaWFnbm9zdG'
    'ljcy52MS5Mb2dFbnRyeVIKcmVjZW50TG9ncw==');

@$core.Deprecated('Use uiEventEntryDescriptor instead')
const UiEventEntry$json = {
  '1': 'UiEventEntry',
  '2': [
    {'1': 'unix_ms', '3': 1, '4': 1, '5': 3, '10': 'unixMs'},
    {'1': 'kind', '3': 2, '4': 1, '5': 9, '10': 'kind'},
    {'1': 'component_id', '3': 3, '4': 1, '5': 9, '10': 'componentId'},
    {'1': 'detail', '3': 4, '4': 1, '5': 9, '10': 'detail'},
  ],
};

/// Descriptor for `UiEventEntry`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List uiEventEntryDescriptor = $convert.base64Decode(
    'CgxVaUV2ZW50RW50cnkSFwoHdW5peF9tcxgBIAEoA1IGdW5peE1zEhIKBGtpbmQYAiABKAlSBG'
    'tpbmQSIQoMY29tcG9uZW50X2lkGAMgASgJUgtjb21wb25lbnRJZBIWCgZkZXRhaWwYBCABKAlS'
    'BmRldGFpbA==');

@$core.Deprecated('Use uiActionEntryDescriptor instead')
const UiActionEntry$json = {
  '1': 'UiActionEntry',
  '2': [
    {'1': 'unix_ms', '3': 1, '4': 1, '5': 3, '10': 'unixMs'},
    {'1': 'component_id', '3': 2, '4': 1, '5': 9, '10': 'componentId'},
    {'1': 'action', '3': 3, '4': 1, '5': 9, '10': 'action'},
    {'1': 'value', '3': 4, '4': 1, '5': 9, '10': 'value'},
  ],
};

/// Descriptor for `UiActionEntry`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List uiActionEntryDescriptor = $convert.base64Decode(
    'Cg1VaUFjdGlvbkVudHJ5EhcKB3VuaXhfbXMYASABKANSBnVuaXhNcxIhCgxjb21wb25lbnRfaW'
    'QYAiABKAlSC2NvbXBvbmVudElkEhYKBmFjdGlvbhgDIAEoCVIGYWN0aW9uEhQKBXZhbHVlGAQg'
    'ASgJUgV2YWx1ZQ==');

@$core.Deprecated('Use streamEntryDescriptor instead')
const StreamEntry$json = {
  '1': 'StreamEntry',
  '2': [
    {'1': 'stream_id', '3': 1, '4': 1, '5': 9, '10': 'streamId'},
    {'1': 'kind', '3': 2, '4': 1, '5': 9, '10': 'kind'},
    {'1': 'source_device_id', '3': 3, '4': 1, '5': 9, '10': 'sourceDeviceId'},
    {'1': 'target_device_id', '3': 4, '4': 1, '5': 9, '10': 'targetDeviceId'},
  ],
};

/// Descriptor for `StreamEntry`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List streamEntryDescriptor = $convert.base64Decode(
    'CgtTdHJlYW1FbnRyeRIbCglzdHJlYW1faWQYASABKAlSCHN0cmVhbUlkEhIKBGtpbmQYAiABKA'
    'lSBGtpbmQSKAoQc291cmNlX2RldmljZV9pZBgDIAEoCVIOc291cmNlRGV2aWNlSWQSKAoQdGFy'
    'Z2V0X2RldmljZV9pZBgEIAEoCVIOdGFyZ2V0RGV2aWNlSWQ=');

@$core.Deprecated('Use routeEntryDescriptor instead')
const RouteEntry$json = {
  '1': 'RouteEntry',
  '2': [
    {'1': 'stream_id', '3': 1, '4': 1, '5': 9, '10': 'streamId'},
    {'1': 'source_device_id', '3': 2, '4': 1, '5': 9, '10': 'sourceDeviceId'},
    {'1': 'target_device_id', '3': 3, '4': 1, '5': 9, '10': 'targetDeviceId'},
    {'1': 'kind', '3': 4, '4': 1, '5': 9, '10': 'kind'},
  ],
};

/// Descriptor for `RouteEntry`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List routeEntryDescriptor = $convert.base64Decode(
    'CgpSb3V0ZUVudHJ5EhsKCXN0cmVhbV9pZBgBIAEoCVIIc3RyZWFtSWQSKAoQc291cmNlX2Rldm'
    'ljZV9pZBgCIAEoCVIOc291cmNlRGV2aWNlSWQSKAoQdGFyZ2V0X2RldmljZV9pZBgDIAEoCVIO'
    'dGFyZ2V0RGV2aWNlSWQSEgoEa2luZBgEIAEoCVIEa2luZA==');

@$core.Deprecated('Use webrtcSignalEntryDescriptor instead')
const WebrtcSignalEntry$json = {
  '1': 'WebrtcSignalEntry',
  '2': [
    {'1': 'unix_ms', '3': 1, '4': 1, '5': 3, '10': 'unixMs'},
    {'1': 'stream_id', '3': 2, '4': 1, '5': 9, '10': 'streamId'},
    {'1': 'signal_type', '3': 3, '4': 1, '5': 9, '10': 'signalType'},
  ],
};

/// Descriptor for `WebrtcSignalEntry`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List webrtcSignalEntryDescriptor = $convert.base64Decode(
    'ChFXZWJydGNTaWduYWxFbnRyeRIXCgd1bml4X21zGAEgASgDUgZ1bml4TXMSGwoJc3RyZWFtX2'
    'lkGAIgASgJUghzdHJlYW1JZBIfCgtzaWduYWxfdHlwZRgDIAEoCVIKc2lnbmFsVHlwZQ==');

@$core.Deprecated('Use logEntryDescriptor instead')
const LogEntry$json = {
  '1': 'LogEntry',
  '2': [
    {'1': 'unix_ms', '3': 1, '4': 1, '5': 3, '10': 'unixMs'},
    {'1': 'level', '3': 2, '4': 1, '5': 9, '10': 'level'},
    {'1': 'message', '3': 3, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `LogEntry`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List logEntryDescriptor = $convert.base64Decode(
    'CghMb2dFbnRyeRIXCgd1bml4X21zGAEgASgDUgZ1bml4TXMSFAoFbGV2ZWwYAiABKAlSBWxldm'
    'VsEhgKB21lc3NhZ2UYAyABKAlSB21lc3NhZ2U=');

@$core.Deprecated('Use connectionHealthDescriptor instead')
const ConnectionHealth$json = {
  '1': 'ConnectionHealth',
  '2': [
    {
      '1': 'last_heartbeat_unix_ms',
      '3': 1,
      '4': 1,
      '5': 3,
      '10': 'lastHeartbeatUnixMs'
    },
    {
      '1': 'reconnect_attempt',
      '3': 2,
      '4': 1,
      '5': 5,
      '10': 'reconnectAttempt'
    },
    {'1': 'last_status', '3': 3, '4': 1, '5': 9, '10': 'lastStatus'},
    {'1': 'last_rtt_ms', '3': 4, '4': 1, '5': 1, '10': 'lastRttMs'},
    {'1': 'online', '3': 5, '4': 1, '5': 8, '10': 'online'},
    {
      '1': 'recent_control_errors',
      '3': 6,
      '4': 3,
      '5': 11,
      '6': '.terminals.diagnostics.v1.ControlErrorEntry',
      '10': 'recentControlErrors'
    },
  ],
};

/// Descriptor for `ConnectionHealth`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List connectionHealthDescriptor = $convert.base64Decode(
    'ChBDb25uZWN0aW9uSGVhbHRoEjMKFmxhc3RfaGVhcnRiZWF0X3VuaXhfbXMYASABKANSE2xhc3'
    'RIZWFydGJlYXRVbml4TXMSKwoRcmVjb25uZWN0X2F0dGVtcHQYAiABKAVSEHJlY29ubmVjdEF0'
    'dGVtcHQSHwoLbGFzdF9zdGF0dXMYAyABKAlSCmxhc3RTdGF0dXMSHgoLbGFzdF9ydHRfbXMYBC'
    'ABKAFSCWxhc3RSdHRNcxIWCgZvbmxpbmUYBSABKAhSBm9ubGluZRJfChVyZWNlbnRfY29udHJv'
    'bF9lcnJvcnMYBiADKAsyKy50ZXJtaW5hbHMuZGlhZ25vc3RpY3MudjEuQ29udHJvbEVycm9yRW'
    '50cnlSE3JlY2VudENvbnRyb2xFcnJvcnM=');

@$core.Deprecated('Use controlErrorEntryDescriptor instead')
const ControlErrorEntry$json = {
  '1': 'ControlErrorEntry',
  '2': [
    {'1': 'unix_ms', '3': 1, '4': 1, '5': 3, '10': 'unixMs'},
    {'1': 'code', '3': 2, '4': 1, '5': 9, '10': 'code'},
    {'1': 'message', '3': 3, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `ControlErrorEntry`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List controlErrorEntryDescriptor = $convert.base64Decode(
    'ChFDb250cm9sRXJyb3JFbnRyeRIXCgd1bml4X21zGAEgASgDUgZ1bml4TXMSEgoEY29kZRgCIA'
    'EoCVIEY29kZRIYCgdtZXNzYWdlGAMgASgJUgdtZXNzYWdl');

@$core.Deprecated('Use hardwareStateDescriptor instead')
const HardwareState$json = {
  '1': 'HardwareState',
  '2': [
    {'1': 'battery_level', '3': 1, '4': 1, '5': 2, '10': 'batteryLevel'},
    {'1': 'battery_charging', '3': 2, '4': 1, '5': 8, '10': 'batteryCharging'},
    {'1': 'screen_width_px', '3': 3, '4': 1, '5': 5, '10': 'screenWidthPx'},
    {'1': 'screen_height_px', '3': 4, '4': 1, '5': 5, '10': 'screenHeightPx'},
    {
      '1': 'device_pixel_ratio',
      '3': 5,
      '4': 1,
      '5': 1,
      '10': 'devicePixelRatio'
    },
    {'1': 'orientation', '3': 6, '4': 1, '5': 9, '10': 'orientation'},
    {
      '1': 'sensor_snapshot',
      '3': 7,
      '4': 3,
      '5': 11,
      '6': '.terminals.diagnostics.v1.HardwareState.SensorSnapshotEntry',
      '10': 'sensorSnapshot'
    },
  ],
  '3': [HardwareState_SensorSnapshotEntry$json],
};

@$core.Deprecated('Use hardwareStateDescriptor instead')
const HardwareState_SensorSnapshotEntry$json = {
  '1': 'SensorSnapshotEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 1, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `HardwareState`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List hardwareStateDescriptor = $convert.base64Decode(
    'Cg1IYXJkd2FyZVN0YXRlEiMKDWJhdHRlcnlfbGV2ZWwYASABKAJSDGJhdHRlcnlMZXZlbBIpCh'
    'BiYXR0ZXJ5X2NoYXJnaW5nGAIgASgIUg9iYXR0ZXJ5Q2hhcmdpbmcSJgoPc2NyZWVuX3dpZHRo'
    'X3B4GAMgASgFUg1zY3JlZW5XaWR0aFB4EigKEHNjcmVlbl9oZWlnaHRfcHgYBCABKAVSDnNjcm'
    'VlbkhlaWdodFB4EiwKEmRldmljZV9waXhlbF9yYXRpbxgFIAEoAVIQZGV2aWNlUGl4ZWxSYXRp'
    'bxIgCgtvcmllbnRhdGlvbhgGIAEoCVILb3JpZW50YXRpb24SZAoPc2Vuc29yX3NuYXBzaG90GA'
    'cgAygLMjsudGVybWluYWxzLmRpYWdub3N0aWNzLnYxLkhhcmR3YXJlU3RhdGUuU2Vuc29yU25h'
    'cHNob3RFbnRyeVIOc2Vuc29yU25hcHNob3QaQQoTU2Vuc29yU25hcHNob3RFbnRyeRIQCgNrZX'
    'kYASABKAlSA2tleRIUCgV2YWx1ZRgCIAEoAVIFdmFsdWU6AjgB');

@$core.Deprecated('Use errorCaptureDescriptor instead')
const ErrorCapture$json = {
  '1': 'ErrorCapture',
  '2': [
    {
      '1': 'last_error_message',
      '3': 1,
      '4': 1,
      '5': 9,
      '10': 'lastErrorMessage'
    },
    {'1': 'last_error_stack', '3': 2, '4': 1, '5': 9, '10': 'lastErrorStack'},
    {
      '1': 'last_error_unix_ms',
      '3': 3,
      '4': 1,
      '5': 3,
      '10': 'lastErrorUnixMs'
    },
  ],
};

/// Descriptor for `ErrorCapture`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List errorCaptureDescriptor = $convert.base64Decode(
    'CgxFcnJvckNhcHR1cmUSLAoSbGFzdF9lcnJvcl9tZXNzYWdlGAEgASgJUhBsYXN0RXJyb3JNZX'
    'NzYWdlEigKEGxhc3RfZXJyb3Jfc3RhY2sYAiABKAlSDmxhc3RFcnJvclN0YWNrEisKEmxhc3Rf'
    'ZXJyb3JfdW5peF9tcxgDIAEoA1IPbGFzdEVycm9yVW5peE1z');
