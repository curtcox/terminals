// This is a generated file - do not edit.
//
// Generated from terminals/control/v1/control.proto.

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

import '../../capabilities/v1/capabilities.pbjson.dart' as $2;
import '../../io/v1/io.pbjson.dart' as $0;
import '../../ui/v1/ui.pbjson.dart' as $1;

@$core.Deprecated('Use commandActionDescriptor instead')
const CommandAction$json = {
  '1': 'CommandAction',
  '2': [
    {'1': 'COMMAND_ACTION_UNSPECIFIED', '2': 0},
    {'1': 'COMMAND_ACTION_START', '2': 1},
    {'1': 'COMMAND_ACTION_STOP', '2': 2},
  ],
};

/// Descriptor for `CommandAction`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List commandActionDescriptor = $convert.base64Decode(
    'Cg1Db21tYW5kQWN0aW9uEh4KGkNPTU1BTkRfQUNUSU9OX1VOU1BFQ0lGSUVEEAASGAoUQ09NTU'
    'FORF9BQ1RJT05fU1RBUlQQARIXChNDT01NQU5EX0FDVElPTl9TVE9QEAI=');

@$core.Deprecated('Use commandKindDescriptor instead')
const CommandKind$json = {
  '1': 'CommandKind',
  '2': [
    {'1': 'COMMAND_KIND_UNSPECIFIED', '2': 0},
    {'1': 'COMMAND_KIND_VOICE', '2': 1},
    {'1': 'COMMAND_KIND_MANUAL', '2': 2},
    {'1': 'COMMAND_KIND_SYSTEM', '2': 3},
  ],
};

/// Descriptor for `CommandKind`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List commandKindDescriptor = $convert.base64Decode(
    'CgtDb21tYW5kS2luZBIcChhDT01NQU5EX0tJTkRfVU5TUEVDSUZJRUQQABIWChJDT01NQU5EX0'
    'tJTkRfVk9JQ0UQARIXChNDT01NQU5EX0tJTkRfTUFOVUFMEAISFwoTQ09NTUFORF9LSU5EX1NZ'
    'U1RFTRAD');

@$core.Deprecated('Use controlErrorCodeDescriptor instead')
const ControlErrorCode$json = {
  '1': 'ControlErrorCode',
  '2': [
    {'1': 'CONTROL_ERROR_CODE_UNSPECIFIED', '2': 0},
    {'1': 'CONTROL_ERROR_CODE_INVALID_CLIENT_MESSAGE', '2': 1},
    {'1': 'CONTROL_ERROR_CODE_INVALID_COMMAND_ACTION', '2': 2},
    {'1': 'CONTROL_ERROR_CODE_INVALID_COMMAND_KIND', '2': 3},
    {'1': 'CONTROL_ERROR_CODE_MISSING_COMMAND_INTENT', '2': 4},
    {'1': 'CONTROL_ERROR_CODE_MISSING_COMMAND_TEXT', '2': 5},
    {'1': 'CONTROL_ERROR_CODE_MISSING_COMMAND_DEVICE_ID', '2': 6},
    {'1': 'CONTROL_ERROR_CODE_PROTOCOL_VIOLATION', '2': 7},
    {'1': 'CONTROL_ERROR_CODE_UNKNOWN', '2': 99},
  ],
};

/// Descriptor for `ControlErrorCode`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List controlErrorCodeDescriptor = $convert.base64Decode(
    'ChBDb250cm9sRXJyb3JDb2RlEiIKHkNPTlRST0xfRVJST1JfQ09ERV9VTlNQRUNJRklFRBAAEi'
    '0KKUNPTlRST0xfRVJST1JfQ09ERV9JTlZBTElEX0NMSUVOVF9NRVNTQUdFEAESLQopQ09OVFJP'
    'TF9FUlJPUl9DT0RFX0lOVkFMSURfQ09NTUFORF9BQ1RJT04QAhIrCidDT05UUk9MX0VSUk9SX0'
    'NPREVfSU5WQUxJRF9DT01NQU5EX0tJTkQQAxItCilDT05UUk9MX0VSUk9SX0NPREVfTUlTU0lO'
    'R19DT01NQU5EX0lOVEVOVBAEEisKJ0NPTlRST0xfRVJST1JfQ09ERV9NSVNTSU5HX0NPTU1BTk'
    'RfVEVYVBAFEjAKLENPTlRST0xfRVJST1JfQ09ERV9NSVNTSU5HX0NPTU1BTkRfREVWSUNFX0lE'
    'EAYSKQolQ09OVFJPTF9FUlJPUl9DT0RFX1BST1RPQ09MX1ZJT0xBVElPThAHEh4KGkNPTlRST0'
    'xfRVJST1JfQ09ERV9VTktOT1dOEGM=');

@$core.Deprecated('Use connectRequestDescriptor instead')
const ConnectRequest$json = {
  '1': 'ConnectRequest',
  '2': [
    {
      '1': 'register',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.RegisterDevice',
      '9': 0,
      '10': 'register'
    },
    {
      '1': 'capability',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.CapabilityUpdate',
      '9': 0,
      '10': 'capability'
    },
    {
      '1': 'input',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.InputEvent',
      '9': 0,
      '10': 'input'
    },
    {
      '1': 'sensor',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.SensorData',
      '9': 0,
      '10': 'sensor'
    },
    {
      '1': 'stream_ready',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.StreamReady',
      '9': 0,
      '10': 'streamReady'
    },
    {
      '1': 'command',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.CommandRequest',
      '9': 0,
      '10': 'command'
    },
    {
      '1': 'heartbeat',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.Heartbeat',
      '9': 0,
      '10': 'heartbeat'
    },
  ],
  '8': [
    {'1': 'payload'},
  ],
};

/// Descriptor for `ConnectRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List connectRequestDescriptor = $convert.base64Decode(
    'Cg5Db25uZWN0UmVxdWVzdBJCCghyZWdpc3RlchgBIAEoCzIkLnRlcm1pbmFscy5jb250cm9sLn'
    'YxLlJlZ2lzdGVyRGV2aWNlSABSCHJlZ2lzdGVyEkgKCmNhcGFiaWxpdHkYAiABKAsyJi50ZXJt'
    'aW5hbHMuY29udHJvbC52MS5DYXBhYmlsaXR5VXBkYXRlSABSCmNhcGFiaWxpdHkSMwoFaW5wdX'
    'QYAyABKAsyGy50ZXJtaW5hbHMuaW8udjEuSW5wdXRFdmVudEgAUgVpbnB1dBI1CgZzZW5zb3IY'
    'BCABKAsyGy50ZXJtaW5hbHMuaW8udjEuU2Vuc29yRGF0YUgAUgZzZW5zb3ISRgoMc3RyZWFtX3'
    'JlYWR5GAUgASgLMiEudGVybWluYWxzLmNvbnRyb2wudjEuU3RyZWFtUmVhZHlIAFILc3RyZWFt'
    'UmVhZHkSQAoHY29tbWFuZBgGIAEoCzIkLnRlcm1pbmFscy5jb250cm9sLnYxLkNvbW1hbmRSZX'
    'F1ZXN0SABSB2NvbW1hbmQSPwoJaGVhcnRiZWF0GAcgASgLMh8udGVybWluYWxzLmNvbnRyb2wu'
    'djEuSGVhcnRiZWF0SABSCWhlYXJ0YmVhdEIJCgdwYXlsb2Fk');

@$core.Deprecated('Use connectResponseDescriptor instead')
const ConnectResponse$json = {
  '1': 'ConnectResponse',
  '2': [
    {
      '1': 'register_ack',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.RegisterAck',
      '9': 0,
      '10': 'registerAck'
    },
    {
      '1': 'set_ui',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.SetUI',
      '9': 0,
      '10': 'setUi'
    },
    {
      '1': 'start_stream',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.StartStream',
      '9': 0,
      '10': 'startStream'
    },
    {
      '1': 'stop_stream',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.StopStream',
      '9': 0,
      '10': 'stopStream'
    },
    {
      '1': 'play_audio',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.PlayAudio',
      '9': 0,
      '10': 'playAudio'
    },
    {
      '1': 'show_media',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.ShowMedia',
      '9': 0,
      '10': 'showMedia'
    },
    {
      '1': 'route_stream',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.RouteStream',
      '9': 0,
      '10': 'routeStream'
    },
    {
      '1': 'notification',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.Notification',
      '9': 0,
      '10': 'notification'
    },
    {
      '1': 'webrtc_signal',
      '3': 9,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.WebRTCSignal',
      '9': 0,
      '10': 'webrtcSignal'
    },
    {
      '1': 'command_result',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.CommandResult',
      '9': 0,
      '10': 'commandResult'
    },
    {
      '1': 'heartbeat',
      '3': 11,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.Heartbeat',
      '9': 0,
      '10': 'heartbeat'
    },
    {
      '1': 'error',
      '3': 12,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.ControlError',
      '9': 0,
      '10': 'error'
    },
    {
      '1': 'update_ui',
      '3': 13,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.UpdateUI',
      '9': 0,
      '10': 'updateUi'
    },
    {
      '1': 'transition_ui',
      '3': 14,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.TransitionUI',
      '9': 0,
      '10': 'transitionUi'
    },
  ],
  '8': [
    {'1': 'payload'},
  ],
};

/// Descriptor for `ConnectResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List connectResponseDescriptor = $convert.base64Decode(
    'Cg9Db25uZWN0UmVzcG9uc2USRgoMcmVnaXN0ZXJfYWNrGAEgASgLMiEudGVybWluYWxzLmNvbn'
    'Ryb2wudjEuUmVnaXN0ZXJBY2tIAFILcmVnaXN0ZXJBY2sSLwoGc2V0X3VpGAIgASgLMhYudGVy'
    'bWluYWxzLnVpLnYxLlNldFVJSABSBXNldFVpEkEKDHN0YXJ0X3N0cmVhbRgDIAEoCzIcLnRlcm'
    '1pbmFscy5pby52MS5TdGFydFN0cmVhbUgAUgtzdGFydFN0cmVhbRI+CgtzdG9wX3N0cmVhbRgE'
    'IAEoCzIbLnRlcm1pbmFscy5pby52MS5TdG9wU3RyZWFtSABSCnN0b3BTdHJlYW0SOwoKcGxheV'
    '9hdWRpbxgFIAEoCzIaLnRlcm1pbmFscy5pby52MS5QbGF5QXVkaW9IAFIJcGxheUF1ZGlvEjsK'
    'CnNob3dfbWVkaWEYBiABKAsyGi50ZXJtaW5hbHMuaW8udjEuU2hvd01lZGlhSABSCXNob3dNZW'
    'RpYRJBCgxyb3V0ZV9zdHJlYW0YByABKAsyHC50ZXJtaW5hbHMuaW8udjEuUm91dGVTdHJlYW1I'
    'AFILcm91dGVTdHJlYW0SQwoMbm90aWZpY2F0aW9uGAggASgLMh0udGVybWluYWxzLnVpLnYxLk'
    '5vdGlmaWNhdGlvbkgAUgxub3RpZmljYXRpb24SSQoNd2VicnRjX3NpZ25hbBgJIAEoCzIiLnRl'
    'cm1pbmFscy5jb250cm9sLnYxLldlYlJUQ1NpZ25hbEgAUgx3ZWJydGNTaWduYWwSTAoOY29tbW'
    'FuZF9yZXN1bHQYCiABKAsyIy50ZXJtaW5hbHMuY29udHJvbC52MS5Db21tYW5kUmVzdWx0SABS'
    'DWNvbW1hbmRSZXN1bHQSPwoJaGVhcnRiZWF0GAsgASgLMh8udGVybWluYWxzLmNvbnRyb2wudj'
    'EuSGVhcnRiZWF0SABSCWhlYXJ0YmVhdBI6CgVlcnJvchgMIAEoCzIiLnRlcm1pbmFscy5jb250'
    'cm9sLnYxLkNvbnRyb2xFcnJvckgAUgVlcnJvchI4Cgl1cGRhdGVfdWkYDSABKAsyGS50ZXJtaW'
    '5hbHMudWkudjEuVXBkYXRlVUlIAFIIdXBkYXRlVWkSRAoNdHJhbnNpdGlvbl91aRgOIAEoCzId'
    'LnRlcm1pbmFscy51aS52MS5UcmFuc2l0aW9uVUlIAFIMdHJhbnNpdGlvblVpQgkKB3BheWxvYW'
    'Q=');

@$core.Deprecated('Use registerDeviceDescriptor instead')
const RegisterDevice$json = {
  '1': 'RegisterDevice',
  '2': [
    {
      '1': 'capabilities',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.terminals.capabilities.v1.DeviceCapabilities',
      '10': 'capabilities'
    },
  ],
};

/// Descriptor for `RegisterDevice`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List registerDeviceDescriptor = $convert.base64Decode(
    'Cg5SZWdpc3RlckRldmljZRJRCgxjYXBhYmlsaXRpZXMYASABKAsyLS50ZXJtaW5hbHMuY2FwYW'
    'JpbGl0aWVzLnYxLkRldmljZUNhcGFiaWxpdGllc1IMY2FwYWJpbGl0aWVz');

@$core.Deprecated('Use registerAckDescriptor instead')
const RegisterAck$json = {
  '1': 'RegisterAck',
  '2': [
    {'1': 'server_id', '3': 1, '4': 1, '5': 9, '10': 'serverId'},
    {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `RegisterAck`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List registerAckDescriptor = $convert.base64Decode(
    'CgtSZWdpc3RlckFjaxIbCglzZXJ2ZXJfaWQYASABKAlSCHNlcnZlcklkEhgKB21lc3NhZ2UYAi'
    'ABKAlSB21lc3NhZ2U=');

@$core.Deprecated('Use capabilityUpdateDescriptor instead')
const CapabilityUpdate$json = {
  '1': 'CapabilityUpdate',
  '2': [
    {
      '1': 'capabilities',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.terminals.capabilities.v1.DeviceCapabilities',
      '10': 'capabilities'
    },
  ],
};

/// Descriptor for `CapabilityUpdate`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List capabilityUpdateDescriptor = $convert.base64Decode(
    'ChBDYXBhYmlsaXR5VXBkYXRlElEKDGNhcGFiaWxpdGllcxgBIAEoCzItLnRlcm1pbmFscy5jYX'
    'BhYmlsaXRpZXMudjEuRGV2aWNlQ2FwYWJpbGl0aWVzUgxjYXBhYmlsaXRpZXM=');

@$core.Deprecated('Use streamReadyDescriptor instead')
const StreamReady$json = {
  '1': 'StreamReady',
  '2': [
    {'1': 'stream_id', '3': 1, '4': 1, '5': 9, '10': 'streamId'},
  ],
};

/// Descriptor for `StreamReady`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List streamReadyDescriptor = $convert
    .base64Decode('CgtTdHJlYW1SZWFkeRIbCglzdHJlYW1faWQYASABKAlSCHN0cmVhbUlk');

@$core.Deprecated('Use commandRequestDescriptor instead')
const CommandRequest$json = {
  '1': 'CommandRequest',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'device_id', '3': 2, '4': 1, '5': 9, '10': 'deviceId'},
    {
      '1': 'action',
      '3': 3,
      '4': 1,
      '5': 14,
      '6': '.terminals.control.v1.CommandAction',
      '10': 'action'
    },
    {
      '1': 'kind',
      '3': 4,
      '4': 1,
      '5': 14,
      '6': '.terminals.control.v1.CommandKind',
      '10': 'kind'
    },
    {'1': 'text', '3': 5, '4': 1, '5': 9, '10': 'text'},
    {'1': 'intent', '3': 6, '4': 1, '5': 9, '10': 'intent'},
  ],
};

/// Descriptor for `CommandRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List commandRequestDescriptor = $convert.base64Decode(
    'Cg5Db21tYW5kUmVxdWVzdBIdCgpyZXF1ZXN0X2lkGAEgASgJUglyZXF1ZXN0SWQSGwoJZGV2aW'
    'NlX2lkGAIgASgJUghkZXZpY2VJZBI7CgZhY3Rpb24YAyABKA4yIy50ZXJtaW5hbHMuY29udHJv'
    'bC52MS5Db21tYW5kQWN0aW9uUgZhY3Rpb24SNQoEa2luZBgEIAEoDjIhLnRlcm1pbmFscy5jb2'
    '50cm9sLnYxLkNvbW1hbmRLaW5kUgRraW5kEhIKBHRleHQYBSABKAlSBHRleHQSFgoGaW50ZW50'
    'GAYgASgJUgZpbnRlbnQ=');

@$core.Deprecated('Use commandResultDescriptor instead')
const CommandResult$json = {
  '1': 'CommandResult',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'scenario_start', '3': 2, '4': 1, '5': 9, '10': 'scenarioStart'},
    {'1': 'scenario_stop', '3': 3, '4': 1, '5': 9, '10': 'scenarioStop'},
    {'1': 'notification', '3': 4, '4': 1, '5': 9, '10': 'notification'},
    {
      '1': 'data',
      '3': 5,
      '4': 3,
      '5': 11,
      '6': '.terminals.control.v1.CommandResult.DataEntry',
      '10': 'data'
    },
  ],
  '3': [CommandResult_DataEntry$json],
};

@$core.Deprecated('Use commandResultDescriptor instead')
const CommandResult_DataEntry$json = {
  '1': 'DataEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `CommandResult`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List commandResultDescriptor = $convert.base64Decode(
    'Cg1Db21tYW5kUmVzdWx0Eh0KCnJlcXVlc3RfaWQYASABKAlSCXJlcXVlc3RJZBIlCg5zY2VuYX'
    'Jpb19zdGFydBgCIAEoCVINc2NlbmFyaW9TdGFydBIjCg1zY2VuYXJpb19zdG9wGAMgASgJUgxz'
    'Y2VuYXJpb1N0b3ASIgoMbm90aWZpY2F0aW9uGAQgASgJUgxub3RpZmljYXRpb24SQQoEZGF0YR'
    'gFIAMoCzItLnRlcm1pbmFscy5jb250cm9sLnYxLkNvbW1hbmRSZXN1bHQuRGF0YUVudHJ5UgRk'
    'YXRhGjcKCURhdGFFbnRyeRIQCgNrZXkYASABKAlSA2tleRIUCgV2YWx1ZRgCIAEoCVIFdmFsdW'
    'U6AjgB');

@$core.Deprecated('Use controlErrorDescriptor instead')
const ControlError$json = {
  '1': 'ControlError',
  '2': [
    {
      '1': 'code',
      '3': 1,
      '4': 1,
      '5': 14,
      '6': '.terminals.control.v1.ControlErrorCode',
      '10': 'code'
    },
    {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `ControlError`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List controlErrorDescriptor = $convert.base64Decode(
    'CgxDb250cm9sRXJyb3ISOgoEY29kZRgBIAEoDjImLnRlcm1pbmFscy5jb250cm9sLnYxLkNvbn'
    'Ryb2xFcnJvckNvZGVSBGNvZGUSGAoHbWVzc2FnZRgCIAEoCVIHbWVzc2FnZQ==');

@$core.Deprecated('Use webRTCSignalDescriptor instead')
const WebRTCSignal$json = {
  '1': 'WebRTCSignal',
  '2': [
    {'1': 'stream_id', '3': 1, '4': 1, '5': 9, '10': 'streamId'},
    {'1': 'signal_type', '3': 2, '4': 1, '5': 9, '10': 'signalType'},
    {'1': 'payload', '3': 3, '4': 1, '5': 9, '10': 'payload'},
  ],
};

/// Descriptor for `WebRTCSignal`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List webRTCSignalDescriptor = $convert.base64Decode(
    'CgxXZWJSVENTaWduYWwSGwoJc3RyZWFtX2lkGAEgASgJUghzdHJlYW1JZBIfCgtzaWduYWxfdH'
    'lwZRgCIAEoCVIKc2lnbmFsVHlwZRIYCgdwYXlsb2FkGAMgASgJUgdwYXlsb2Fk');

@$core.Deprecated('Use heartbeatDescriptor instead')
const Heartbeat$json = {
  '1': 'Heartbeat',
  '2': [
    {'1': 'device_id', '3': 1, '4': 1, '5': 9, '10': 'deviceId'},
    {'1': 'unix_ms', '3': 2, '4': 1, '5': 3, '10': 'unixMs'},
  ],
};

/// Descriptor for `Heartbeat`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List heartbeatDescriptor = $convert.base64Decode(
    'CglIZWFydGJlYXQSGwoJZGV2aWNlX2lkGAEgASgJUghkZXZpY2VJZBIXCgd1bml4X21zGAIgAS'
    'gDUgZ1bml4TXM=');

const $core.Map<$core.String, $core.dynamic> TerminalControlServiceBase$json = {
  '1': 'TerminalControlService',
  '2': [
    {
      '1': 'Connect',
      '2': '.terminals.control.v1.ConnectRequest',
      '3': '.terminals.control.v1.ConnectResponse',
      '5': true,
      '6': true
    },
  ],
};

@$core.Deprecated('Use terminalControlServiceDescriptor instead')
const $core.Map<$core.String, $core.Map<$core.String, $core.dynamic>>
    TerminalControlServiceBase$messageJson = {
  '.terminals.control.v1.ConnectRequest': ConnectRequest$json,
  '.terminals.control.v1.RegisterDevice': RegisterDevice$json,
  '.terminals.capabilities.v1.DeviceCapabilities': $2.DeviceCapabilities$json,
  '.terminals.capabilities.v1.DeviceIdentity': $2.DeviceIdentity$json,
  '.terminals.capabilities.v1.ScreenCapability': $2.ScreenCapability$json,
  '.terminals.capabilities.v1.KeyboardCapability': $2.KeyboardCapability$json,
  '.terminals.capabilities.v1.PointerCapability': $2.PointerCapability$json,
  '.terminals.capabilities.v1.TouchCapability': $2.TouchCapability$json,
  '.terminals.capabilities.v1.AudioOutputCapability':
      $2.AudioOutputCapability$json,
  '.terminals.capabilities.v1.AudioInputCapability':
      $2.AudioInputCapability$json,
  '.terminals.capabilities.v1.CameraCapability': $2.CameraCapability$json,
  '.terminals.capabilities.v1.CameraLens': $2.CameraLens$json,
  '.terminals.capabilities.v1.SensorCapability': $2.SensorCapability$json,
  '.terminals.capabilities.v1.ConnectivityCapability':
      $2.ConnectivityCapability$json,
  '.terminals.capabilities.v1.BatteryCapability': $2.BatteryCapability$json,
  '.terminals.control.v1.CapabilityUpdate': CapabilityUpdate$json,
  '.terminals.io.v1.InputEvent': $0.InputEvent$json,
  '.terminals.io.v1.KeyEvent': $0.KeyEvent$json,
  '.terminals.io.v1.PointerEvent': $0.PointerEvent$json,
  '.terminals.io.v1.TouchEvent': $0.TouchEvent$json,
  '.terminals.io.v1.TouchPoint': $0.TouchPoint$json,
  '.terminals.io.v1.UIAction': $0.UIAction$json,
  '.terminals.io.v1.SensorData': $0.SensorData$json,
  '.terminals.io.v1.SensorData.ValuesEntry': $0.SensorData_ValuesEntry$json,
  '.terminals.control.v1.StreamReady': StreamReady$json,
  '.terminals.control.v1.CommandRequest': CommandRequest$json,
  '.terminals.control.v1.Heartbeat': Heartbeat$json,
  '.terminals.control.v1.ConnectResponse': ConnectResponse$json,
  '.terminals.control.v1.RegisterAck': RegisterAck$json,
  '.terminals.ui.v1.SetUI': $1.SetUI$json,
  '.terminals.ui.v1.Node': $1.Node$json,
  '.terminals.ui.v1.Node.PropsEntry': $1.Node_PropsEntry$json,
  '.terminals.ui.v1.StackWidget': $1.StackWidget$json,
  '.terminals.ui.v1.RowWidget': $1.RowWidget$json,
  '.terminals.ui.v1.GridWidget': $1.GridWidget$json,
  '.terminals.ui.v1.ScrollWidget': $1.ScrollWidget$json,
  '.terminals.ui.v1.PaddingWidget': $1.PaddingWidget$json,
  '.terminals.ui.v1.CenterWidget': $1.CenterWidget$json,
  '.terminals.ui.v1.ExpandWidget': $1.ExpandWidget$json,
  '.terminals.ui.v1.TextWidget': $1.TextWidget$json,
  '.terminals.ui.v1.ImageWidget': $1.ImageWidget$json,
  '.terminals.ui.v1.VideoSurfaceWidget': $1.VideoSurfaceWidget$json,
  '.terminals.ui.v1.AudioVisualizerWidget': $1.AudioVisualizerWidget$json,
  '.terminals.ui.v1.CanvasWidget': $1.CanvasWidget$json,
  '.terminals.ui.v1.TextInputWidget': $1.TextInputWidget$json,
  '.terminals.ui.v1.ButtonWidget': $1.ButtonWidget$json,
  '.terminals.ui.v1.SliderWidget': $1.SliderWidget$json,
  '.terminals.ui.v1.ToggleWidget': $1.ToggleWidget$json,
  '.terminals.ui.v1.DropdownWidget': $1.DropdownWidget$json,
  '.terminals.ui.v1.GestureAreaWidget': $1.GestureAreaWidget$json,
  '.terminals.ui.v1.OverlayWidget': $1.OverlayWidget$json,
  '.terminals.ui.v1.ProgressWidget': $1.ProgressWidget$json,
  '.terminals.ui.v1.FullscreenWidget': $1.FullscreenWidget$json,
  '.terminals.ui.v1.KeepAwakeWidget': $1.KeepAwakeWidget$json,
  '.terminals.ui.v1.BrightnessWidget': $1.BrightnessWidget$json,
  '.terminals.io.v1.StartStream': $0.StartStream$json,
  '.terminals.io.v1.StartStream.MetadataEntry':
      $0.StartStream_MetadataEntry$json,
  '.terminals.io.v1.StopStream': $0.StopStream$json,
  '.terminals.io.v1.PlayAudio': $0.PlayAudio$json,
  '.terminals.io.v1.ShowMedia': $0.ShowMedia$json,
  '.terminals.io.v1.RouteStream': $0.RouteStream$json,
  '.terminals.ui.v1.Notification': $1.Notification$json,
  '.terminals.control.v1.WebRTCSignal': WebRTCSignal$json,
  '.terminals.control.v1.CommandResult': CommandResult$json,
  '.terminals.control.v1.CommandResult.DataEntry': CommandResult_DataEntry$json,
  '.terminals.control.v1.ControlError': ControlError$json,
  '.terminals.ui.v1.UpdateUI': $1.UpdateUI$json,
  '.terminals.ui.v1.TransitionUI': $1.TransitionUI$json,
};

/// Descriptor for `TerminalControlService`. Decode as a `google.protobuf.ServiceDescriptorProto`.
final $typed_data.Uint8List terminalControlServiceDescriptor = $convert.base64Decode(
    'ChZUZXJtaW5hbENvbnRyb2xTZXJ2aWNlEloKB0Nvbm5lY3QSJC50ZXJtaW5hbHMuY29udHJvbC'
    '52MS5Db25uZWN0UmVxdWVzdBolLnRlcm1pbmFscy5jb250cm9sLnYxLkNvbm5lY3RSZXNwb25z'
    'ZSgBMAE=');
