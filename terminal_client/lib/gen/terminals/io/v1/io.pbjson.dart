// This is a generated file - do not edit.
//
// Generated from terminals/io/v1/io.proto.

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

@$core.Deprecated('Use startStreamDescriptor instead')
const StartStream$json = {
  '1': 'StartStream',
  '2': [
    {'1': 'stream_id', '3': 1, '4': 1, '5': 9, '10': 'streamId'},
    {'1': 'kind', '3': 2, '4': 1, '5': 9, '10': 'kind'},
    {'1': 'source_device_id', '3': 3, '4': 1, '5': 9, '10': 'sourceDeviceId'},
    {'1': 'target_device_id', '3': 4, '4': 1, '5': 9, '10': 'targetDeviceId'},
    {
      '1': 'metadata',
      '3': 5,
      '4': 3,
      '5': 11,
      '6': '.terminals.io.v1.StartStream.MetadataEntry',
      '10': 'metadata'
    },
  ],
  '3': [StartStream_MetadataEntry$json],
};

@$core.Deprecated('Use startStreamDescriptor instead')
const StartStream_MetadataEntry$json = {
  '1': 'MetadataEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `StartStream`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List startStreamDescriptor = $convert.base64Decode(
    'CgtTdGFydFN0cmVhbRIbCglzdHJlYW1faWQYASABKAlSCHN0cmVhbUlkEhIKBGtpbmQYAiABKA'
    'lSBGtpbmQSKAoQc291cmNlX2RldmljZV9pZBgDIAEoCVIOc291cmNlRGV2aWNlSWQSKAoQdGFy'
    'Z2V0X2RldmljZV9pZBgEIAEoCVIOdGFyZ2V0RGV2aWNlSWQSRgoIbWV0YWRhdGEYBSADKAsyKi'
    '50ZXJtaW5hbHMuaW8udjEuU3RhcnRTdHJlYW0uTWV0YWRhdGFFbnRyeVIIbWV0YWRhdGEaOwoN'
    'TWV0YWRhdGFFbnRyeRIQCgNrZXkYASABKAlSA2tleRIUCgV2YWx1ZRgCIAEoCVIFdmFsdWU6Aj'
    'gB');

@$core.Deprecated('Use stopStreamDescriptor instead')
const StopStream$json = {
  '1': 'StopStream',
  '2': [
    {'1': 'stream_id', '3': 1, '4': 1, '5': 9, '10': 'streamId'},
  ],
};

/// Descriptor for `StopStream`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List stopStreamDescriptor = $convert
    .base64Decode('CgpTdG9wU3RyZWFtEhsKCXN0cmVhbV9pZBgBIAEoCVIIc3RyZWFtSWQ=');

@$core.Deprecated('Use routeStreamDescriptor instead')
const RouteStream$json = {
  '1': 'RouteStream',
  '2': [
    {'1': 'stream_id', '3': 1, '4': 1, '5': 9, '10': 'streamId'},
    {'1': 'source_device_id', '3': 2, '4': 1, '5': 9, '10': 'sourceDeviceId'},
    {'1': 'target_device_id', '3': 3, '4': 1, '5': 9, '10': 'targetDeviceId'},
    {'1': 'kind', '3': 4, '4': 1, '5': 9, '10': 'kind'},
  ],
};

/// Descriptor for `RouteStream`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List routeStreamDescriptor = $convert.base64Decode(
    'CgtSb3V0ZVN0cmVhbRIbCglzdHJlYW1faWQYASABKAlSCHN0cmVhbUlkEigKEHNvdXJjZV9kZX'
    'ZpY2VfaWQYAiABKAlSDnNvdXJjZURldmljZUlkEigKEHRhcmdldF9kZXZpY2VfaWQYAyABKAlS'
    'DnRhcmdldERldmljZUlkEhIKBGtpbmQYBCABKAlSBGtpbmQ=');

@$core.Deprecated('Use playAudioDescriptor instead')
const PlayAudio$json = {
  '1': 'PlayAudio',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'device_id', '3': 2, '4': 1, '5': 9, '10': 'deviceId'},
    {'1': 'url', '3': 3, '4': 1, '5': 9, '9': 0, '10': 'url'},
    {'1': 'pcm_data', '3': 4, '4': 1, '5': 12, '9': 0, '10': 'pcmData'},
    {'1': 'tts_text', '3': 5, '4': 1, '5': 9, '9': 0, '10': 'ttsText'},
    {'1': 'format', '3': 6, '4': 1, '5': 9, '10': 'format'},
  ],
  '8': [
    {'1': 'source'},
  ],
};

/// Descriptor for `PlayAudio`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List playAudioDescriptor = $convert.base64Decode(
    'CglQbGF5QXVkaW8SHQoKcmVxdWVzdF9pZBgBIAEoCVIJcmVxdWVzdElkEhsKCWRldmljZV9pZB'
    'gCIAEoCVIIZGV2aWNlSWQSEgoDdXJsGAMgASgJSABSA3VybBIbCghwY21fZGF0YRgEIAEoDEgA'
    'UgdwY21EYXRhEhsKCHR0c190ZXh0GAUgASgJSABSB3R0c1RleHQSFgoGZm9ybWF0GAYgASgJUg'
    'Zmb3JtYXRCCAoGc291cmNl');

@$core.Deprecated('Use showMediaDescriptor instead')
const ShowMedia$json = {
  '1': 'ShowMedia',
  '2': [
    {'1': 'request_id', '3': 1, '4': 1, '5': 9, '10': 'requestId'},
    {'1': 'device_id', '3': 2, '4': 1, '5': 9, '10': 'deviceId'},
    {'1': 'media_url', '3': 3, '4': 1, '5': 9, '10': 'mediaUrl'},
    {'1': 'media_type', '3': 4, '4': 1, '5': 9, '10': 'mediaType'},
  ],
};

/// Descriptor for `ShowMedia`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List showMediaDescriptor = $convert.base64Decode(
    'CglTaG93TWVkaWESHQoKcmVxdWVzdF9pZBgBIAEoCVIJcmVxdWVzdElkEhsKCWRldmljZV9pZB'
    'gCIAEoCVIIZGV2aWNlSWQSGwoJbWVkaWFfdXJsGAMgASgJUghtZWRpYVVybBIdCgptZWRpYV90'
    'eXBlGAQgASgJUgltZWRpYVR5cGU=');

@$core.Deprecated('Use inputEventDescriptor instead')
const InputEvent$json = {
  '1': 'InputEvent',
  '2': [
    {'1': 'device_id', '3': 1, '4': 1, '5': 9, '10': 'deviceId'},
    {
      '1': 'key',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.KeyEvent',
      '9': 0,
      '10': 'key'
    },
    {
      '1': 'pointer',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.PointerEvent',
      '9': 0,
      '10': 'pointer'
    },
    {
      '1': 'touch',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.TouchEvent',
      '9': 0,
      '10': 'touch'
    },
    {
      '1': 'ui_action',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.UIAction',
      '9': 0,
      '10': 'uiAction'
    },
  ],
  '8': [
    {'1': 'payload'},
  ],
};

/// Descriptor for `InputEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List inputEventDescriptor = $convert.base64Decode(
    'CgpJbnB1dEV2ZW50EhsKCWRldmljZV9pZBgBIAEoCVIIZGV2aWNlSWQSLQoDa2V5GAIgASgLMh'
    'kudGVybWluYWxzLmlvLnYxLktleUV2ZW50SABSA2tleRI5Cgdwb2ludGVyGAMgASgLMh0udGVy'
    'bWluYWxzLmlvLnYxLlBvaW50ZXJFdmVudEgAUgdwb2ludGVyEjMKBXRvdWNoGAQgASgLMhsudG'
    'VybWluYWxzLmlvLnYxLlRvdWNoRXZlbnRIAFIFdG91Y2gSOAoJdWlfYWN0aW9uGAUgASgLMhku'
    'dGVybWluYWxzLmlvLnYxLlVJQWN0aW9uSABSCHVpQWN0aW9uQgkKB3BheWxvYWQ=');

@$core.Deprecated('Use keyEventDescriptor instead')
const KeyEvent$json = {
  '1': 'KeyEvent',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'down', '3': 2, '4': 1, '5': 8, '10': 'down'},
    {'1': 'up', '3': 3, '4': 1, '5': 8, '10': 'up'},
    {'1': 'text', '3': 4, '4': 1, '5': 9, '10': 'text'},
  ],
};

/// Descriptor for `KeyEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List keyEventDescriptor = $convert.base64Decode(
    'CghLZXlFdmVudBIQCgNrZXkYASABKAlSA2tleRISCgRkb3duGAIgASgIUgRkb3duEg4KAnVwGA'
    'MgASgIUgJ1cBISCgR0ZXh0GAQgASgJUgR0ZXh0');

@$core.Deprecated('Use pointerEventDescriptor instead')
const PointerEvent$json = {
  '1': 'PointerEvent',
  '2': [
    {'1': 'action', '3': 1, '4': 1, '5': 9, '10': 'action'},
    {'1': 'x', '3': 2, '4': 1, '5': 1, '10': 'x'},
    {'1': 'y', '3': 3, '4': 1, '5': 1, '10': 'y'},
    {'1': 'delta_x', '3': 4, '4': 1, '5': 1, '10': 'deltaX'},
    {'1': 'delta_y', '3': 5, '4': 1, '5': 1, '10': 'deltaY'},
    {'1': 'button', '3': 6, '4': 1, '5': 5, '10': 'button'},
  ],
};

/// Descriptor for `PointerEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pointerEventDescriptor = $convert.base64Decode(
    'CgxQb2ludGVyRXZlbnQSFgoGYWN0aW9uGAEgASgJUgZhY3Rpb24SDAoBeBgCIAEoAVIBeBIMCg'
    'F5GAMgASgBUgF5EhcKB2RlbHRhX3gYBCABKAFSBmRlbHRhWBIXCgdkZWx0YV95GAUgASgBUgZk'
    'ZWx0YVkSFgoGYnV0dG9uGAYgASgFUgZidXR0b24=');

@$core.Deprecated('Use touchPointDescriptor instead')
const TouchPoint$json = {
  '1': 'TouchPoint',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 5, '10': 'id'},
    {'1': 'x', '3': 2, '4': 1, '5': 1, '10': 'x'},
    {'1': 'y', '3': 3, '4': 1, '5': 1, '10': 'y'},
  ],
};

/// Descriptor for `TouchPoint`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List touchPointDescriptor = $convert.base64Decode(
    'CgpUb3VjaFBvaW50Eg4KAmlkGAEgASgFUgJpZBIMCgF4GAIgASgBUgF4EgwKAXkYAyABKAFSAX'
    'k=');

@$core.Deprecated('Use touchEventDescriptor instead')
const TouchEvent$json = {
  '1': 'TouchEvent',
  '2': [
    {'1': 'action', '3': 1, '4': 1, '5': 9, '10': 'action'},
    {
      '1': 'points',
      '3': 2,
      '4': 3,
      '5': 11,
      '6': '.terminals.io.v1.TouchPoint',
      '10': 'points'
    },
  ],
};

/// Descriptor for `TouchEvent`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List touchEventDescriptor = $convert.base64Decode(
    'CgpUb3VjaEV2ZW50EhYKBmFjdGlvbhgBIAEoCVIGYWN0aW9uEjMKBnBvaW50cxgCIAMoCzIbLn'
    'Rlcm1pbmFscy5pby52MS5Ub3VjaFBvaW50UgZwb2ludHM=');

@$core.Deprecated('Use uIActionDescriptor instead')
const UIAction$json = {
  '1': 'UIAction',
  '2': [
    {'1': 'component_id', '3': 1, '4': 1, '5': 9, '10': 'componentId'},
    {'1': 'action', '3': 2, '4': 1, '5': 9, '10': 'action'},
    {'1': 'value', '3': 3, '4': 1, '5': 9, '10': 'value'},
  ],
};

/// Descriptor for `UIAction`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List uIActionDescriptor = $convert.base64Decode(
    'CghVSUFjdGlvbhIhCgxjb21wb25lbnRfaWQYASABKAlSC2NvbXBvbmVudElkEhYKBmFjdGlvbh'
    'gCIAEoCVIGYWN0aW9uEhQKBXZhbHVlGAMgASgJUgV2YWx1ZQ==');

@$core.Deprecated('Use sensorDataDescriptor instead')
const SensorData$json = {
  '1': 'SensorData',
  '2': [
    {'1': 'device_id', '3': 1, '4': 1, '5': 9, '10': 'deviceId'},
    {'1': 'unix_ms', '3': 2, '4': 1, '5': 3, '10': 'unixMs'},
    {
      '1': 'values',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.terminals.io.v1.SensorData.ValuesEntry',
      '10': 'values'
    },
  ],
  '3': [SensorData_ValuesEntry$json],
};

@$core.Deprecated('Use sensorDataDescriptor instead')
const SensorData_ValuesEntry$json = {
  '1': 'ValuesEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 1, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `SensorData`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sensorDataDescriptor = $convert.base64Decode(
    'CgpTZW5zb3JEYXRhEhsKCWRldmljZV9pZBgBIAEoCVIIZGV2aWNlSWQSFwoHdW5peF9tcxgCIA'
    'EoA1IGdW5peE1zEj8KBnZhbHVlcxgDIAMoCzInLnRlcm1pbmFscy5pby52MS5TZW5zb3JEYXRh'
    'LlZhbHVlc0VudHJ5UgZ2YWx1ZXMaOQoLVmFsdWVzRW50cnkSEAoDa2V5GAEgASgJUgNrZXkSFA'
    'oFdmFsdWUYAiABKAFSBXZhbHVlOgI4AQ==');
