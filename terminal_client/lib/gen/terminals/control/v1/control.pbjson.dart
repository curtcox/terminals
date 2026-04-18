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

import '../../capabilities/v1/capabilities.pbjson.dart' as $3;
import '../../diagnostics/v1/diagnostics.pbjson.dart' as $1;
import '../../io/v1/io.pbjson.dart' as $0;
import '../../ui/v1/ui.pbjson.dart' as $2;

@$core.Deprecated('Use carrierKindDescriptor instead')
const CarrierKind$json = {
  '1': 'CarrierKind',
  '2': [
    {'1': 'CARRIER_KIND_UNSPECIFIED', '2': 0},
    {'1': 'CARRIER_KIND_GRPC', '2': 1},
    {'1': 'CARRIER_KIND_WEBSOCKET', '2': 2},
    {'1': 'CARRIER_KIND_TCP', '2': 3},
    {'1': 'CARRIER_KIND_HTTP', '2': 4},
  ],
};

/// Descriptor for `CarrierKind`. Decode as a `google.protobuf.EnumDescriptorProto`.
final $typed_data.Uint8List carrierKindDescriptor = $convert.base64Decode(
    'CgtDYXJyaWVyS2luZBIcChhDQVJSSUVSX0tJTkRfVU5TUEVDSUZJRUQQABIVChFDQVJSSUVSX0'
    'tJTkRfR1JQQxABEhoKFkNBUlJJRVJfS0lORF9XRUJTT0NLRVQQAhIUChBDQVJSSUVSX0tJTkRf'
    'VENQEAMSFQoRQ0FSUklFUl9LSU5EX0hUVFAQBA==');

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

@$core.Deprecated('Use transportHelloDescriptor instead')
const TransportHello$json = {
  '1': 'TransportHello',
  '2': [
    {'1': 'protocol_version', '3': 1, '4': 1, '5': 13, '10': 'protocolVersion'},
    {
      '1': 'supported_carriers',
      '3': 2,
      '4': 3,
      '5': 14,
      '6': '.terminals.control.v1.CarrierKind',
      '10': 'supportedCarriers'
    },
    {'1': 'desired_device_id', '3': 3, '4': 1, '5': 9, '10': 'desiredDeviceId'},
    {'1': 'resume_token', '3': 4, '4': 1, '5': 9, '10': 'resumeToken'},
  ],
};

/// Descriptor for `TransportHello`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List transportHelloDescriptor = $convert.base64Decode(
    'Cg5UcmFuc3BvcnRIZWxsbxIpChBwcm90b2NvbF92ZXJzaW9uGAEgASgNUg9wcm90b2NvbFZlcn'
    'Npb24SUAoSc3VwcG9ydGVkX2NhcnJpZXJzGAIgAygOMiEudGVybWluYWxzLmNvbnRyb2wudjEu'
    'Q2FycmllcktpbmRSEXN1cHBvcnRlZENhcnJpZXJzEioKEWRlc2lyZWRfZGV2aWNlX2lkGAMgAS'
    'gJUg9kZXNpcmVkRGV2aWNlSWQSIQoMcmVzdW1lX3Rva2VuGAQgASgJUgtyZXN1bWVUb2tlbg==');

@$core.Deprecated('Use transportHelloAckDescriptor instead')
const TransportHelloAck$json = {
  '1': 'TransportHelloAck',
  '2': [
    {
      '1': 'accepted_protocol_version',
      '3': 1,
      '4': 1,
      '5': 13,
      '10': 'acceptedProtocolVersion'
    },
    {
      '1': 'negotiated_carrier',
      '3': 2,
      '4': 1,
      '5': 14,
      '6': '.terminals.control.v1.CarrierKind',
      '10': 'negotiatedCarrier'
    },
    {'1': 'session_id', '3': 3, '4': 1, '5': 9, '10': 'sessionId'},
    {'1': 'resume_token', '3': 4, '4': 1, '5': 9, '10': 'resumeToken'},
    {
      '1': 'heartbeat_interval_ms',
      '3': 5,
      '4': 1,
      '5': 3,
      '10': 'heartbeatIntervalMs'
    },
    {
      '1': 'limits',
      '3': 6,
      '4': 3,
      '5': 11,
      '6': '.terminals.control.v1.TransportHelloAck.LimitsEntry',
      '10': 'limits'
    },
  ],
  '3': [TransportHelloAck_LimitsEntry$json],
};

@$core.Deprecated('Use transportHelloAckDescriptor instead')
const TransportHelloAck_LimitsEntry$json = {
  '1': 'LimitsEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `TransportHelloAck`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List transportHelloAckDescriptor = $convert.base64Decode(
    'ChFUcmFuc3BvcnRIZWxsb0FjaxI6ChlhY2NlcHRlZF9wcm90b2NvbF92ZXJzaW9uGAEgASgNUh'
    'dhY2NlcHRlZFByb3RvY29sVmVyc2lvbhJQChJuZWdvdGlhdGVkX2NhcnJpZXIYAiABKA4yIS50'
    'ZXJtaW5hbHMuY29udHJvbC52MS5DYXJyaWVyS2luZFIRbmVnb3RpYXRlZENhcnJpZXISHQoKc2'
    'Vzc2lvbl9pZBgDIAEoCVIJc2Vzc2lvbklkEiEKDHJlc3VtZV90b2tlbhgEIAEoCVILcmVzdW1l'
    'VG9rZW4SMgoVaGVhcnRiZWF0X2ludGVydmFsX21zGAUgASgDUhNoZWFydGJlYXRJbnRlcnZhbE'
    '1zEksKBmxpbWl0cxgGIAMoCzIzLnRlcm1pbmFscy5jb250cm9sLnYxLlRyYW5zcG9ydEhlbGxv'
    'QWNrLkxpbWl0c0VudHJ5UgZsaW1pdHMaOQoLTGltaXRzRW50cnkSEAoDa2V5GAEgASgJUgNrZX'
    'kSFAoFdmFsdWUYAiABKAlSBXZhbHVlOgI4AQ==');

@$core.Deprecated('Use transportHeartbeatDescriptor instead')
const TransportHeartbeat$json = {
  '1': 'TransportHeartbeat',
  '2': [
    {'1': 'unix_ms', '3': 1, '4': 1, '5': 3, '10': 'unixMs'},
  ],
};

/// Descriptor for `TransportHeartbeat`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List transportHeartbeatDescriptor =
    $convert.base64Decode(
        'ChJUcmFuc3BvcnRIZWFydGJlYXQSFwoHdW5peF9tcxgBIAEoA1IGdW5peE1z');

@$core.Deprecated('Use transportErrorDescriptor instead')
const TransportError$json = {
  '1': 'TransportError',
  '2': [
    {'1': 'code', '3': 1, '4': 1, '5': 9, '10': 'code'},
    {'1': 'message', '3': 2, '4': 1, '5': 9, '10': 'message'},
  ],
};

/// Descriptor for `TransportError`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List transportErrorDescriptor = $convert.base64Decode(
    'Cg5UcmFuc3BvcnRFcnJvchISCgRjb2RlGAEgASgJUgRjb2RlEhgKB21lc3NhZ2UYAiABKAlSB2'
    '1lc3NhZ2U=');

@$core.Deprecated('Use wireEnvelopeDescriptor instead')
const WireEnvelope$json = {
  '1': 'WireEnvelope',
  '2': [
    {'1': 'protocol_version', '3': 1, '4': 1, '5': 13, '10': 'protocolVersion'},
    {'1': 'session_id', '3': 2, '4': 1, '5': 9, '10': 'sessionId'},
    {'1': 'sequence', '3': 3, '4': 1, '5': 4, '10': 'sequence'},
    {
      '1': 'client_message',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.ConnectRequest',
      '9': 0,
      '10': 'clientMessage'
    },
    {
      '1': 'server_message',
      '3': 11,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.ConnectResponse',
      '9': 0,
      '10': 'serverMessage'
    },
    {
      '1': 'transport_hello',
      '3': 12,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.TransportHello',
      '9': 0,
      '10': 'transportHello'
    },
    {
      '1': 'transport_hello_ack',
      '3': 13,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.TransportHelloAck',
      '9': 0,
      '10': 'transportHelloAck'
    },
    {
      '1': 'transport_heartbeat',
      '3': 14,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.TransportHeartbeat',
      '9': 0,
      '10': 'transportHeartbeat'
    },
    {
      '1': 'transport_error',
      '3': 15,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.TransportError',
      '9': 0,
      '10': 'transportError'
    },
  ],
  '8': [
    {'1': 'payload'},
  ],
};

/// Descriptor for `WireEnvelope`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List wireEnvelopeDescriptor = $convert.base64Decode(
    'CgxXaXJlRW52ZWxvcGUSKQoQcHJvdG9jb2xfdmVyc2lvbhgBIAEoDVIPcHJvdG9jb2xWZXJzaW'
    '9uEh0KCnNlc3Npb25faWQYAiABKAlSCXNlc3Npb25JZBIaCghzZXF1ZW5jZRgDIAEoBFIIc2Vx'
    'dWVuY2USTQoOY2xpZW50X21lc3NhZ2UYCiABKAsyJC50ZXJtaW5hbHMuY29udHJvbC52MS5Db2'
    '5uZWN0UmVxdWVzdEgAUg1jbGllbnRNZXNzYWdlEk4KDnNlcnZlcl9tZXNzYWdlGAsgASgLMiUu'
    'dGVybWluYWxzLmNvbnRyb2wudjEuQ29ubmVjdFJlc3BvbnNlSABSDXNlcnZlck1lc3NhZ2USTw'
    'oPdHJhbnNwb3J0X2hlbGxvGAwgASgLMiQudGVybWluYWxzLmNvbnRyb2wudjEuVHJhbnNwb3J0'
    'SGVsbG9IAFIOdHJhbnNwb3J0SGVsbG8SWQoTdHJhbnNwb3J0X2hlbGxvX2FjaxgNIAEoCzInLn'
    'Rlcm1pbmFscy5jb250cm9sLnYxLlRyYW5zcG9ydEhlbGxvQWNrSABSEXRyYW5zcG9ydEhlbGxv'
    'QWNrElsKE3RyYW5zcG9ydF9oZWFydGJlYXQYDiABKAsyKC50ZXJtaW5hbHMuY29udHJvbC52MS'
    '5UcmFuc3BvcnRIZWFydGJlYXRIAFISdHJhbnNwb3J0SGVhcnRiZWF0Ek8KD3RyYW5zcG9ydF9l'
    'cnJvchgPIAEoCzIkLnRlcm1pbmFscy5jb250cm9sLnYxLlRyYW5zcG9ydEVycm9ySABSDnRyYW'
    '5zcG9ydEVycm9yQgkKB3BheWxvYWQ=');

@$core.Deprecated('Use connectRequestDescriptor instead')
const ConnectRequest$json = {
  '1': 'ConnectRequest',
  '2': [
    {
      '1': 'hello',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.Hello',
      '9': 0,
      '10': 'hello'
    },
    {
      '1': 'capability_snapshot',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.CapabilitySnapshot',
      '9': 0,
      '10': 'capabilitySnapshot'
    },
    {
      '1': 'capability_delta',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.CapabilityDelta',
      '9': 0,
      '10': 'capabilityDelta'
    },
    {
      '1': 'register',
      '3': 20,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.RegisterDevice',
      '8': {'3': true},
      '9': 0,
      '10': 'register',
    },
    {
      '1': 'capability',
      '3': 21,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.CapabilityUpdate',
      '8': {'3': true},
      '9': 0,
      '10': 'capability',
    },
    {
      '1': 'input',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.InputEvent',
      '9': 0,
      '10': 'input'
    },
    {
      '1': 'sensor',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.SensorData',
      '9': 0,
      '10': 'sensor'
    },
    {
      '1': 'stream_ready',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.StreamReady',
      '9': 0,
      '10': 'streamReady'
    },
    {
      '1': 'command',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.CommandRequest',
      '9': 0,
      '10': 'command'
    },
    {
      '1': 'heartbeat',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.Heartbeat',
      '9': 0,
      '10': 'heartbeat'
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
      '1': 'voice_audio',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.VoiceAudio',
      '9': 0,
      '10': 'voiceAudio'
    },
    {
      '1': 'observation_message',
      '3': 11,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.ObservationMessage',
      '9': 0,
      '10': 'observationMessage'
    },
    {
      '1': 'artifact_available',
      '3': 12,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.ArtifactAvailable',
      '9': 0,
      '10': 'artifactAvailable'
    },
    {
      '1': 'flow_stats',
      '3': 13,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.FlowStats',
      '9': 0,
      '10': 'flowStats'
    },
    {
      '1': 'clock_sample',
      '3': 14,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.ClockSample',
      '9': 0,
      '10': 'clockSample'
    },
    {
      '1': 'bug_report',
      '3': 15,
      '4': 1,
      '5': 11,
      '6': '.terminals.diagnostics.v1.BugReport',
      '9': 0,
      '10': 'bugReport'
    },
  ],
  '8': [
    {'1': 'payload'},
  ],
};

/// Descriptor for `ConnectRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List connectRequestDescriptor = $convert.base64Decode(
    'Cg5Db25uZWN0UmVxdWVzdBIzCgVoZWxsbxgBIAEoCzIbLnRlcm1pbmFscy5jb250cm9sLnYxLk'
    'hlbGxvSABSBWhlbGxvElsKE2NhcGFiaWxpdHlfc25hcHNob3QYAiABKAsyKC50ZXJtaW5hbHMu'
    'Y29udHJvbC52MS5DYXBhYmlsaXR5U25hcHNob3RIAFISY2FwYWJpbGl0eVNuYXBzaG90ElIKEG'
    'NhcGFiaWxpdHlfZGVsdGEYAyABKAsyJS50ZXJtaW5hbHMuY29udHJvbC52MS5DYXBhYmlsaXR5'
    'RGVsdGFIAFIPY2FwYWJpbGl0eURlbHRhEkYKCHJlZ2lzdGVyGBQgASgLMiQudGVybWluYWxzLm'
    'NvbnRyb2wudjEuUmVnaXN0ZXJEZXZpY2VCAhgBSABSCHJlZ2lzdGVyEkwKCmNhcGFiaWxpdHkY'
    'FSABKAsyJi50ZXJtaW5hbHMuY29udHJvbC52MS5DYXBhYmlsaXR5VXBkYXRlQgIYAUgAUgpjYX'
    'BhYmlsaXR5EjMKBWlucHV0GAQgASgLMhsudGVybWluYWxzLmlvLnYxLklucHV0RXZlbnRIAFIF'
    'aW5wdXQSNQoGc2Vuc29yGAUgASgLMhsudGVybWluYWxzLmlvLnYxLlNlbnNvckRhdGFIAFIGc2'
    'Vuc29yEkYKDHN0cmVhbV9yZWFkeRgGIAEoCzIhLnRlcm1pbmFscy5jb250cm9sLnYxLlN0cmVh'
    'bVJlYWR5SABSC3N0cmVhbVJlYWR5EkAKB2NvbW1hbmQYByABKAsyJC50ZXJtaW5hbHMuY29udH'
    'JvbC52MS5Db21tYW5kUmVxdWVzdEgAUgdjb21tYW5kEj8KCWhlYXJ0YmVhdBgIIAEoCzIfLnRl'
    'cm1pbmFscy5jb250cm9sLnYxLkhlYXJ0YmVhdEgAUgloZWFydGJlYXQSSQoNd2VicnRjX3NpZ2'
    '5hbBgJIAEoCzIiLnRlcm1pbmFscy5jb250cm9sLnYxLldlYlJUQ1NpZ25hbEgAUgx3ZWJydGNT'
    'aWduYWwSQwoLdm9pY2VfYXVkaW8YCiABKAsyIC50ZXJtaW5hbHMuY29udHJvbC52MS5Wb2ljZU'
    'F1ZGlvSABSCnZvaWNlQXVkaW8SVgoTb2JzZXJ2YXRpb25fbWVzc2FnZRgLIAEoCzIjLnRlcm1p'
    'bmFscy5pby52MS5PYnNlcnZhdGlvbk1lc3NhZ2VIAFISb2JzZXJ2YXRpb25NZXNzYWdlElMKEm'
    'FydGlmYWN0X2F2YWlsYWJsZRgMIAEoCzIiLnRlcm1pbmFscy5pby52MS5BcnRpZmFjdEF2YWls'
    'YWJsZUgAUhFhcnRpZmFjdEF2YWlsYWJsZRI7CgpmbG93X3N0YXRzGA0gASgLMhoudGVybWluYW'
    'xzLmlvLnYxLkZsb3dTdGF0c0gAUglmbG93U3RhdHMSQQoMY2xvY2tfc2FtcGxlGA4gASgLMhwu'
    'dGVybWluYWxzLmlvLnYxLkNsb2NrU2FtcGxlSABSC2Nsb2NrU2FtcGxlEkQKCmJ1Z19yZXBvcn'
    'QYDyABKAsyIy50ZXJtaW5hbHMuZGlhZ25vc3RpY3MudjEuQnVnUmVwb3J0SABSCWJ1Z1JlcG9y'
    'dEIJCgdwYXlsb2Fk');

@$core.Deprecated('Use voiceAudioDescriptor instead')
const VoiceAudio$json = {
  '1': 'VoiceAudio',
  '2': [
    {'1': 'device_id', '3': 1, '4': 1, '5': 9, '10': 'deviceId'},
    {'1': 'audio', '3': 2, '4': 1, '5': 12, '10': 'audio'},
    {'1': 'sample_rate', '3': 3, '4': 1, '5': 5, '10': 'sampleRate'},
    {'1': 'is_final', '3': 4, '4': 1, '5': 8, '10': 'isFinal'},
  ],
};

/// Descriptor for `VoiceAudio`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List voiceAudioDescriptor = $convert.base64Decode(
    'CgpWb2ljZUF1ZGlvEhsKCWRldmljZV9pZBgBIAEoCVIIZGV2aWNlSWQSFAoFYXVkaW8YAiABKA'
    'xSBWF1ZGlvEh8KC3NhbXBsZV9yYXRlGAMgASgFUgpzYW1wbGVSYXRlEhkKCGlzX2ZpbmFsGAQg'
    'ASgIUgdpc0ZpbmFs');

@$core.Deprecated('Use connectResponseDescriptor instead')
const ConnectResponse$json = {
  '1': 'ConnectResponse',
  '2': [
    {
      '1': 'hello_ack',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.HelloAck',
      '9': 0,
      '10': 'helloAck'
    },
    {
      '1': 'capability_ack',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.CapabilityAck',
      '9': 0,
      '10': 'capabilityAck'
    },
    {
      '1': 'register_ack',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.RegisterAck',
      '9': 0,
      '10': 'registerAck'
    },
    {
      '1': 'set_ui',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.SetUI',
      '9': 0,
      '10': 'setUi'
    },
    {
      '1': 'start_stream',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.StartStream',
      '9': 0,
      '10': 'startStream'
    },
    {
      '1': 'stop_stream',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.StopStream',
      '9': 0,
      '10': 'stopStream'
    },
    {
      '1': 'play_audio',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.PlayAudio',
      '9': 0,
      '10': 'playAudio'
    },
    {
      '1': 'show_media',
      '3': 8,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.ShowMedia',
      '9': 0,
      '10': 'showMedia'
    },
    {
      '1': 'route_stream',
      '3': 9,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.RouteStream',
      '9': 0,
      '10': 'routeStream'
    },
    {
      '1': 'notification',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.Notification',
      '9': 0,
      '10': 'notification'
    },
    {
      '1': 'webrtc_signal',
      '3': 11,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.WebRTCSignal',
      '9': 0,
      '10': 'webrtcSignal'
    },
    {
      '1': 'command_result',
      '3': 12,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.CommandResult',
      '9': 0,
      '10': 'commandResult'
    },
    {
      '1': 'heartbeat',
      '3': 13,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.Heartbeat',
      '9': 0,
      '10': 'heartbeat'
    },
    {
      '1': 'error',
      '3': 14,
      '4': 1,
      '5': 11,
      '6': '.terminals.control.v1.ControlError',
      '9': 0,
      '10': 'error'
    },
    {
      '1': 'update_ui',
      '3': 15,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.UpdateUI',
      '9': 0,
      '10': 'updateUi'
    },
    {
      '1': 'transition_ui',
      '3': 16,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.TransitionUI',
      '9': 0,
      '10': 'transitionUi'
    },
    {
      '1': 'install_bundle',
      '3': 17,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.InstallBundle',
      '9': 0,
      '10': 'installBundle'
    },
    {
      '1': 'remove_bundle',
      '3': 18,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.RemoveBundle',
      '9': 0,
      '10': 'removeBundle'
    },
    {
      '1': 'start_flow',
      '3': 19,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.StartFlow',
      '9': 0,
      '10': 'startFlow'
    },
    {
      '1': 'patch_flow',
      '3': 20,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.PatchFlow',
      '9': 0,
      '10': 'patchFlow'
    },
    {
      '1': 'stop_flow',
      '3': 21,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.StopFlow',
      '9': 0,
      '10': 'stopFlow'
    },
    {
      '1': 'request_artifact',
      '3': 22,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.RequestArtifact',
      '9': 0,
      '10': 'requestArtifact'
    },
    {
      '1': 'bug_report_ack',
      '3': 23,
      '4': 1,
      '5': 11,
      '6': '.terminals.diagnostics.v1.BugReportAck',
      '9': 0,
      '10': 'bugReportAck'
    },
  ],
  '8': [
    {'1': 'payload'},
  ],
};

/// Descriptor for `ConnectResponse`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List connectResponseDescriptor = $convert.base64Decode(
    'Cg9Db25uZWN0UmVzcG9uc2USPQoJaGVsbG9fYWNrGAEgASgLMh4udGVybWluYWxzLmNvbnRyb2'
    'wudjEuSGVsbG9BY2tIAFIIaGVsbG9BY2sSTAoOY2FwYWJpbGl0eV9hY2sYAiABKAsyIy50ZXJt'
    'aW5hbHMuY29udHJvbC52MS5DYXBhYmlsaXR5QWNrSABSDWNhcGFiaWxpdHlBY2sSRgoMcmVnaX'
    'N0ZXJfYWNrGAMgASgLMiEudGVybWluYWxzLmNvbnRyb2wudjEuUmVnaXN0ZXJBY2tIAFILcmVn'
    'aXN0ZXJBY2sSLwoGc2V0X3VpGAQgASgLMhYudGVybWluYWxzLnVpLnYxLlNldFVJSABSBXNldF'
    'VpEkEKDHN0YXJ0X3N0cmVhbRgFIAEoCzIcLnRlcm1pbmFscy5pby52MS5TdGFydFN0cmVhbUgA'
    'UgtzdGFydFN0cmVhbRI+CgtzdG9wX3N0cmVhbRgGIAEoCzIbLnRlcm1pbmFscy5pby52MS5TdG'
    '9wU3RyZWFtSABSCnN0b3BTdHJlYW0SOwoKcGxheV9hdWRpbxgHIAEoCzIaLnRlcm1pbmFscy5p'
    'by52MS5QbGF5QXVkaW9IAFIJcGxheUF1ZGlvEjsKCnNob3dfbWVkaWEYCCABKAsyGi50ZXJtaW'
    '5hbHMuaW8udjEuU2hvd01lZGlhSABSCXNob3dNZWRpYRJBCgxyb3V0ZV9zdHJlYW0YCSABKAsy'
    'HC50ZXJtaW5hbHMuaW8udjEuUm91dGVTdHJlYW1IAFILcm91dGVTdHJlYW0SQwoMbm90aWZpY2'
    'F0aW9uGAogASgLMh0udGVybWluYWxzLnVpLnYxLk5vdGlmaWNhdGlvbkgAUgxub3RpZmljYXRp'
    'b24SSQoNd2VicnRjX3NpZ25hbBgLIAEoCzIiLnRlcm1pbmFscy5jb250cm9sLnYxLldlYlJUQ1'
    'NpZ25hbEgAUgx3ZWJydGNTaWduYWwSTAoOY29tbWFuZF9yZXN1bHQYDCABKAsyIy50ZXJtaW5h'
    'bHMuY29udHJvbC52MS5Db21tYW5kUmVzdWx0SABSDWNvbW1hbmRSZXN1bHQSPwoJaGVhcnRiZW'
    'F0GA0gASgLMh8udGVybWluYWxzLmNvbnRyb2wudjEuSGVhcnRiZWF0SABSCWhlYXJ0YmVhdBI6'
    'CgVlcnJvchgOIAEoCzIiLnRlcm1pbmFscy5jb250cm9sLnYxLkNvbnRyb2xFcnJvckgAUgVlcn'
    'JvchI4Cgl1cGRhdGVfdWkYDyABKAsyGS50ZXJtaW5hbHMudWkudjEuVXBkYXRlVUlIAFIIdXBk'
    'YXRlVWkSRAoNdHJhbnNpdGlvbl91aRgQIAEoCzIdLnRlcm1pbmFscy51aS52MS5UcmFuc2l0aW'
    '9uVUlIAFIMdHJhbnNpdGlvblVpEkcKDmluc3RhbGxfYnVuZGxlGBEgASgLMh4udGVybWluYWxz'
    'LmlvLnYxLkluc3RhbGxCdW5kbGVIAFINaW5zdGFsbEJ1bmRsZRJECg1yZW1vdmVfYnVuZGxlGB'
    'IgASgLMh0udGVybWluYWxzLmlvLnYxLlJlbW92ZUJ1bmRsZUgAUgxyZW1vdmVCdW5kbGUSOwoK'
    'c3RhcnRfZmxvdxgTIAEoCzIaLnRlcm1pbmFscy5pby52MS5TdGFydEZsb3dIAFIJc3RhcnRGbG'
    '93EjsKCnBhdGNoX2Zsb3cYFCABKAsyGi50ZXJtaW5hbHMuaW8udjEuUGF0Y2hGbG93SABSCXBh'
    'dGNoRmxvdxI4CglzdG9wX2Zsb3cYFSABKAsyGS50ZXJtaW5hbHMuaW8udjEuU3RvcEZsb3dIAF'
    'IIc3RvcEZsb3cSTQoQcmVxdWVzdF9hcnRpZmFjdBgWIAEoCzIgLnRlcm1pbmFscy5pby52MS5S'
    'ZXF1ZXN0QXJ0aWZhY3RIAFIPcmVxdWVzdEFydGlmYWN0Ek4KDmJ1Z19yZXBvcnRfYWNrGBcgAS'
    'gLMiYudGVybWluYWxzLmRpYWdub3N0aWNzLnYxLkJ1Z1JlcG9ydEFja0gAUgxidWdSZXBvcnRB'
    'Y2tCCQoHcGF5bG9hZA==');

@$core.Deprecated('Use helloDescriptor instead')
const Hello$json = {
  '1': 'Hello',
  '2': [
    {'1': 'device_id', '3': 1, '4': 1, '5': 9, '10': 'deviceId'},
    {
      '1': 'identity',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.terminals.capabilities.v1.DeviceIdentity',
      '10': 'identity'
    },
    {'1': 'client_version', '3': 3, '4': 1, '5': 9, '10': 'clientVersion'},
  ],
};

/// Descriptor for `Hello`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List helloDescriptor = $convert.base64Decode(
    'CgVIZWxsbxIbCglkZXZpY2VfaWQYASABKAlSCGRldmljZUlkEkUKCGlkZW50aXR5GAIgASgLMi'
    'kudGVybWluYWxzLmNhcGFiaWxpdGllcy52MS5EZXZpY2VJZGVudGl0eVIIaWRlbnRpdHkSJQoO'
    'Y2xpZW50X3ZlcnNpb24YAyABKAlSDWNsaWVudFZlcnNpb24=');

@$core.Deprecated('Use helloAckDescriptor instead')
const HelloAck$json = {
  '1': 'HelloAck',
  '2': [
    {'1': 'server_id', '3': 1, '4': 1, '5': 9, '10': 'serverId'},
    {'1': 'session_id', '3': 2, '4': 1, '5': 9, '10': 'sessionId'},
    {
      '1': 'heartbeat_interval_ms',
      '3': 3,
      '4': 1,
      '5': 3,
      '10': 'heartbeatIntervalMs'
    },
  ],
};

/// Descriptor for `HelloAck`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List helloAckDescriptor = $convert.base64Decode(
    'CghIZWxsb0FjaxIbCglzZXJ2ZXJfaWQYASABKAlSCHNlcnZlcklkEh0KCnNlc3Npb25faWQYAi'
    'ABKAlSCXNlc3Npb25JZBIyChVoZWFydGJlYXRfaW50ZXJ2YWxfbXMYAyABKANSE2hlYXJ0YmVh'
    'dEludGVydmFsTXM=');

@$core.Deprecated('Use capabilitySnapshotDescriptor instead')
const CapabilitySnapshot$json = {
  '1': 'CapabilitySnapshot',
  '2': [
    {'1': 'device_id', '3': 1, '4': 1, '5': 9, '10': 'deviceId'},
    {'1': 'generation', '3': 2, '4': 1, '5': 4, '10': 'generation'},
    {
      '1': 'capabilities',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.terminals.capabilities.v1.DeviceCapabilities',
      '10': 'capabilities'
    },
  ],
};

/// Descriptor for `CapabilitySnapshot`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List capabilitySnapshotDescriptor = $convert.base64Decode(
    'ChJDYXBhYmlsaXR5U25hcHNob3QSGwoJZGV2aWNlX2lkGAEgASgJUghkZXZpY2VJZBIeCgpnZW'
    '5lcmF0aW9uGAIgASgEUgpnZW5lcmF0aW9uElEKDGNhcGFiaWxpdGllcxgDIAEoCzItLnRlcm1p'
    'bmFscy5jYXBhYmlsaXRpZXMudjEuRGV2aWNlQ2FwYWJpbGl0aWVzUgxjYXBhYmlsaXRpZXM=');

@$core.Deprecated('Use capabilityDeltaDescriptor instead')
const CapabilityDelta$json = {
  '1': 'CapabilityDelta',
  '2': [
    {'1': 'device_id', '3': 1, '4': 1, '5': 9, '10': 'deviceId'},
    {'1': 'generation', '3': 2, '4': 1, '5': 4, '10': 'generation'},
    {
      '1': 'capabilities',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.terminals.capabilities.v1.DeviceCapabilities',
      '10': 'capabilities'
    },
    {'1': 'reason', '3': 4, '4': 1, '5': 9, '10': 'reason'},
  ],
};

/// Descriptor for `CapabilityDelta`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List capabilityDeltaDescriptor = $convert.base64Decode(
    'Cg9DYXBhYmlsaXR5RGVsdGESGwoJZGV2aWNlX2lkGAEgASgJUghkZXZpY2VJZBIeCgpnZW5lcm'
    'F0aW9uGAIgASgEUgpnZW5lcmF0aW9uElEKDGNhcGFiaWxpdGllcxgDIAEoCzItLnRlcm1pbmFs'
    'cy5jYXBhYmlsaXRpZXMudjEuRGV2aWNlQ2FwYWJpbGl0aWVzUgxjYXBhYmlsaXRpZXMSFgoGcm'
    'Vhc29uGAQgASgJUgZyZWFzb24=');

@$core.Deprecated('Use capabilityAckDescriptor instead')
const CapabilityAck$json = {
  '1': 'CapabilityAck',
  '2': [
    {'1': 'device_id', '3': 1, '4': 1, '5': 9, '10': 'deviceId'},
    {
      '1': 'accepted_generation',
      '3': 2,
      '4': 1,
      '5': 4,
      '10': 'acceptedGeneration'
    },
    {'1': 'snapshot_applied', '3': 3, '4': 1, '5': 8, '10': 'snapshotApplied'},
  ],
};

/// Descriptor for `CapabilityAck`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List capabilityAckDescriptor = $convert.base64Decode(
    'Cg1DYXBhYmlsaXR5QWNrEhsKCWRldmljZV9pZBgBIAEoCVIIZGV2aWNlSWQSLwoTYWNjZXB0ZW'
    'RfZ2VuZXJhdGlvbhgCIAEoBFISYWNjZXB0ZWRHZW5lcmF0aW9uEikKEHNuYXBzaG90X2FwcGxp'
    'ZWQYAyABKAhSD3NuYXBzaG90QXBwbGllZA==');

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
    {
      '1': 'metadata',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.terminals.control.v1.RegisterAck.MetadataEntry',
      '10': 'metadata'
    },
  ],
  '3': [RegisterAck_MetadataEntry$json],
};

@$core.Deprecated('Use registerAckDescriptor instead')
const RegisterAck_MetadataEntry$json = {
  '1': 'MetadataEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `RegisterAck`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List registerAckDescriptor = $convert.base64Decode(
    'CgtSZWdpc3RlckFjaxIbCglzZXJ2ZXJfaWQYASABKAlSCHNlcnZlcklkEhgKB21lc3NhZ2UYAi'
    'ABKAlSB21lc3NhZ2USSwoIbWV0YWRhdGEYAyADKAsyLy50ZXJtaW5hbHMuY29udHJvbC52MS5S'
    'ZWdpc3RlckFjay5NZXRhZGF0YUVudHJ5UghtZXRhZGF0YRo7Cg1NZXRhZGF0YUVudHJ5EhAKA2'
    'tleRgBIAEoCVIDa2V5EhQKBXZhbHVlGAIgASgJUgV2YWx1ZToCOAE=');

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
    {
      '1': 'arguments',
      '3': 7,
      '4': 3,
      '5': 11,
      '6': '.terminals.control.v1.CommandRequest.ArgumentsEntry',
      '10': 'arguments'
    },
  ],
  '3': [CommandRequest_ArgumentsEntry$json],
};

@$core.Deprecated('Use commandRequestDescriptor instead')
const CommandRequest_ArgumentsEntry$json = {
  '1': 'ArgumentsEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `CommandRequest`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List commandRequestDescriptor = $convert.base64Decode(
    'Cg5Db21tYW5kUmVxdWVzdBIdCgpyZXF1ZXN0X2lkGAEgASgJUglyZXF1ZXN0SWQSGwoJZGV2aW'
    'NlX2lkGAIgASgJUghkZXZpY2VJZBI7CgZhY3Rpb24YAyABKA4yIy50ZXJtaW5hbHMuY29udHJv'
    'bC52MS5Db21tYW5kQWN0aW9uUgZhY3Rpb24SNQoEa2luZBgEIAEoDjIhLnRlcm1pbmFscy5jb2'
    '50cm9sLnYxLkNvbW1hbmRLaW5kUgRraW5kEhIKBHRleHQYBSABKAlSBHRleHQSFgoGaW50ZW50'
    'GAYgASgJUgZpbnRlbnQSUQoJYXJndW1lbnRzGAcgAygLMjMudGVybWluYWxzLmNvbnRyb2wudj'
    'EuQ29tbWFuZFJlcXVlc3QuQXJndW1lbnRzRW50cnlSCWFyZ3VtZW50cxo8Cg5Bcmd1bWVudHNF'
    'bnRyeRIQCgNrZXkYASABKAlSA2tleRIUCgV2YWx1ZRgCIAEoCVIFdmFsdWU6AjgB');

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
  '.terminals.control.v1.Hello': Hello$json,
  '.terminals.capabilities.v1.DeviceIdentity': $3.DeviceIdentity$json,
  '.terminals.control.v1.CapabilitySnapshot': CapabilitySnapshot$json,
  '.terminals.capabilities.v1.DeviceCapabilities': $3.DeviceCapabilities$json,
  '.terminals.capabilities.v1.ScreenCapability': $3.ScreenCapability$json,
  '.terminals.capabilities.v1.Insets': $3.Insets$json,
  '.terminals.capabilities.v1.KeyboardCapability': $3.KeyboardCapability$json,
  '.terminals.capabilities.v1.PointerCapability': $3.PointerCapability$json,
  '.terminals.capabilities.v1.TouchCapability': $3.TouchCapability$json,
  '.terminals.capabilities.v1.AudioOutputCapability':
      $3.AudioOutputCapability$json,
  '.terminals.capabilities.v1.AudioEndpoint': $3.AudioEndpoint$json,
  '.terminals.capabilities.v1.AudioInputCapability':
      $3.AudioInputCapability$json,
  '.terminals.capabilities.v1.CameraCapability': $3.CameraCapability$json,
  '.terminals.capabilities.v1.CameraLens': $3.CameraLens$json,
  '.terminals.capabilities.v1.CameraEndpoint': $3.CameraEndpoint$json,
  '.terminals.capabilities.v1.SensorCapability': $3.SensorCapability$json,
  '.terminals.capabilities.v1.ConnectivityCapability':
      $3.ConnectivityCapability$json,
  '.terminals.capabilities.v1.BatteryCapability': $3.BatteryCapability$json,
  '.terminals.capabilities.v1.EdgeCapability': $3.EdgeCapability$json,
  '.terminals.capabilities.v1.EdgeComputeCapability':
      $3.EdgeComputeCapability$json,
  '.terminals.capabilities.v1.EdgeRetentionCapability':
      $3.EdgeRetentionCapability$json,
  '.terminals.capabilities.v1.EdgeTimingCapability':
      $3.EdgeTimingCapability$json,
  '.terminals.capabilities.v1.EdgeGeometryCapability':
      $3.EdgeGeometryCapability$json,
  '.terminals.capabilities.v1.DisplayCapability': $3.DisplayCapability$json,
  '.terminals.capabilities.v1.HapticCapability': $3.HapticCapability$json,
  '.terminals.control.v1.CapabilityDelta': CapabilityDelta$json,
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
  '.terminals.control.v1.CommandRequest.ArgumentsEntry':
      CommandRequest_ArgumentsEntry$json,
  '.terminals.control.v1.Heartbeat': Heartbeat$json,
  '.terminals.control.v1.WebRTCSignal': WebRTCSignal$json,
  '.terminals.control.v1.VoiceAudio': VoiceAudio$json,
  '.terminals.io.v1.ObservationMessage': $0.ObservationMessage$json,
  '.terminals.io.v1.Observation': $0.Observation$json,
  '.terminals.io.v1.DeviceRef': $0.DeviceRef$json,
  '.terminals.io.v1.LocationEstimate': $0.LocationEstimate$json,
  '.terminals.io.v1.Pose': $0.Pose$json,
  '.terminals.io.v1.Observation.AttributesEntry':
      $0.Observation_AttributesEntry$json,
  '.terminals.io.v1.ArtifactRef': $0.ArtifactRef$json,
  '.terminals.io.v1.ObservationProvenance': $0.ObservationProvenance$json,
  '.terminals.io.v1.ArtifactAvailable': $0.ArtifactAvailable$json,
  '.terminals.io.v1.FlowStats': $0.FlowStats$json,
  '.terminals.io.v1.ClockSample': $0.ClockSample$json,
  '.terminals.diagnostics.v1.BugReport': $1.BugReport$json,
  '.terminals.diagnostics.v1.ClientContext': $1.ClientContext$json,
  '.terminals.diagnostics.v1.ClientIdentity': $1.ClientIdentity$json,
  '.terminals.diagnostics.v1.RuntimeState': $1.RuntimeState$json,
  '.terminals.ui.v1.Node': $2.Node$json,
  '.terminals.ui.v1.Node.PropsEntry': $2.Node_PropsEntry$json,
  '.terminals.ui.v1.StackWidget': $2.StackWidget$json,
  '.terminals.ui.v1.RowWidget': $2.RowWidget$json,
  '.terminals.ui.v1.GridWidget': $2.GridWidget$json,
  '.terminals.ui.v1.ScrollWidget': $2.ScrollWidget$json,
  '.terminals.ui.v1.PaddingWidget': $2.PaddingWidget$json,
  '.terminals.ui.v1.CenterWidget': $2.CenterWidget$json,
  '.terminals.ui.v1.ExpandWidget': $2.ExpandWidget$json,
  '.terminals.ui.v1.TextWidget': $2.TextWidget$json,
  '.terminals.ui.v1.ImageWidget': $2.ImageWidget$json,
  '.terminals.ui.v1.VideoSurfaceWidget': $2.VideoSurfaceWidget$json,
  '.terminals.ui.v1.AudioVisualizerWidget': $2.AudioVisualizerWidget$json,
  '.terminals.ui.v1.CanvasWidget': $2.CanvasWidget$json,
  '.terminals.ui.v1.TextInputWidget': $2.TextInputWidget$json,
  '.terminals.ui.v1.ButtonWidget': $2.ButtonWidget$json,
  '.terminals.ui.v1.SliderWidget': $2.SliderWidget$json,
  '.terminals.ui.v1.ToggleWidget': $2.ToggleWidget$json,
  '.terminals.ui.v1.DropdownWidget': $2.DropdownWidget$json,
  '.terminals.ui.v1.GestureAreaWidget': $2.GestureAreaWidget$json,
  '.terminals.ui.v1.OverlayWidget': $2.OverlayWidget$json,
  '.terminals.ui.v1.ProgressWidget': $2.ProgressWidget$json,
  '.terminals.ui.v1.FullscreenWidget': $2.FullscreenWidget$json,
  '.terminals.ui.v1.KeepAwakeWidget': $2.KeepAwakeWidget$json,
  '.terminals.ui.v1.BrightnessWidget': $2.BrightnessWidget$json,
  '.terminals.diagnostics.v1.UiEventEntry': $1.UiEventEntry$json,
  '.terminals.diagnostics.v1.UiActionEntry': $1.UiActionEntry$json,
  '.terminals.diagnostics.v1.StreamEntry': $1.StreamEntry$json,
  '.terminals.diagnostics.v1.RouteEntry': $1.RouteEntry$json,
  '.terminals.diagnostics.v1.WebrtcSignalEntry': $1.WebrtcSignalEntry$json,
  '.terminals.diagnostics.v1.LogEntry': $1.LogEntry$json,
  '.terminals.diagnostics.v1.ConnectionHealth': $1.ConnectionHealth$json,
  '.terminals.diagnostics.v1.ControlErrorEntry': $1.ControlErrorEntry$json,
  '.terminals.diagnostics.v1.HardwareState': $1.HardwareState$json,
  '.terminals.diagnostics.v1.HardwareState.SensorSnapshotEntry':
      $1.HardwareState_SensorSnapshotEntry$json,
  '.terminals.diagnostics.v1.ErrorCapture': $1.ErrorCapture$json,
  '.terminals.diagnostics.v1.BugReport.SourceHintsEntry':
      $1.BugReport_SourceHintsEntry$json,
  '.terminals.control.v1.RegisterDevice': RegisterDevice$json,
  '.terminals.control.v1.CapabilityUpdate': CapabilityUpdate$json,
  '.terminals.control.v1.ConnectResponse': ConnectResponse$json,
  '.terminals.control.v1.HelloAck': HelloAck$json,
  '.terminals.control.v1.CapabilityAck': CapabilityAck$json,
  '.terminals.control.v1.RegisterAck': RegisterAck$json,
  '.terminals.control.v1.RegisterAck.MetadataEntry':
      RegisterAck_MetadataEntry$json,
  '.terminals.ui.v1.SetUI': $2.SetUI$json,
  '.terminals.io.v1.StartStream': $0.StartStream$json,
  '.terminals.io.v1.StartStream.MetadataEntry':
      $0.StartStream_MetadataEntry$json,
  '.terminals.io.v1.StopStream': $0.StopStream$json,
  '.terminals.io.v1.PlayAudio': $0.PlayAudio$json,
  '.terminals.io.v1.ShowMedia': $0.ShowMedia$json,
  '.terminals.io.v1.RouteStream': $0.RouteStream$json,
  '.terminals.ui.v1.Notification': $2.Notification$json,
  '.terminals.control.v1.CommandResult': CommandResult$json,
  '.terminals.control.v1.CommandResult.DataEntry': CommandResult_DataEntry$json,
  '.terminals.control.v1.ControlError': ControlError$json,
  '.terminals.ui.v1.UpdateUI': $2.UpdateUI$json,
  '.terminals.ui.v1.TransitionUI': $2.TransitionUI$json,
  '.terminals.io.v1.InstallBundle': $0.InstallBundle$json,
  '.terminals.io.v1.RemoveBundle': $0.RemoveBundle$json,
  '.terminals.io.v1.StartFlow': $0.StartFlow$json,
  '.terminals.io.v1.FlowPlan': $0.FlowPlan$json,
  '.terminals.io.v1.FlowNode': $0.FlowNode$json,
  '.terminals.io.v1.FlowNode.ArgsEntry': $0.FlowNode_ArgsEntry$json,
  '.terminals.io.v1.FlowEdge': $0.FlowEdge$json,
  '.terminals.io.v1.PatchFlow': $0.PatchFlow$json,
  '.terminals.io.v1.StopFlow': $0.StopFlow$json,
  '.terminals.io.v1.RequestArtifact': $0.RequestArtifact$json,
  '.terminals.diagnostics.v1.BugReportAck': $1.BugReportAck$json,
};

/// Descriptor for `TerminalControlService`. Decode as a `google.protobuf.ServiceDescriptorProto`.
final $typed_data.Uint8List terminalControlServiceDescriptor = $convert.base64Decode(
    'ChZUZXJtaW5hbENvbnRyb2xTZXJ2aWNlEloKB0Nvbm5lY3QSJC50ZXJtaW5hbHMuY29udHJvbC'
    '52MS5Db25uZWN0UmVxdWVzdBolLnRlcm1pbmFscy5jb250cm9sLnYxLkNvbm5lY3RSZXNwb25z'
    'ZSgBMAE=');
