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

@$core.Deprecated('Use flowNodeDescriptor instead')
const FlowNode$json = {
  '1': 'FlowNode',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'kind', '3': 2, '4': 1, '5': 9, '10': 'kind'},
    {
      '1': 'args',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.terminals.io.v1.FlowNode.ArgsEntry',
      '10': 'args'
    },
    {'1': 'exec', '3': 4, '4': 1, '5': 9, '10': 'exec'},
  ],
  '3': [FlowNode_ArgsEntry$json],
};

@$core.Deprecated('Use flowNodeDescriptor instead')
const FlowNode_ArgsEntry$json = {
  '1': 'ArgsEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `FlowNode`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List flowNodeDescriptor = $convert.base64Decode(
    'CghGbG93Tm9kZRIOCgJpZBgBIAEoCVICaWQSEgoEa2luZBgCIAEoCVIEa2luZBI3CgRhcmdzGA'
    'MgAygLMiMudGVybWluYWxzLmlvLnYxLkZsb3dOb2RlLkFyZ3NFbnRyeVIEYXJncxISCgRleGVj'
    'GAQgASgJUgRleGVjGjcKCUFyZ3NFbnRyeRIQCgNrZXkYASABKAlSA2tleRIUCgV2YWx1ZRgCIA'
    'EoCVIFdmFsdWU6AjgB');

@$core.Deprecated('Use flowEdgeDescriptor instead')
const FlowEdge$json = {
  '1': 'FlowEdge',
  '2': [
    {'1': 'from', '3': 1, '4': 1, '5': 9, '10': 'from'},
    {'1': 'to', '3': 2, '4': 1, '5': 9, '10': 'to'},
  ],
};

/// Descriptor for `FlowEdge`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List flowEdgeDescriptor = $convert.base64Decode(
    'CghGbG93RWRnZRISCgRmcm9tGAEgASgJUgRmcm9tEg4KAnRvGAIgASgJUgJ0bw==');

@$core.Deprecated('Use flowPlanDescriptor instead')
const FlowPlan$json = {
  '1': 'FlowPlan',
  '2': [
    {
      '1': 'nodes',
      '3': 1,
      '4': 3,
      '5': 11,
      '6': '.terminals.io.v1.FlowNode',
      '10': 'nodes'
    },
    {
      '1': 'edges',
      '3': 2,
      '4': 3,
      '5': 11,
      '6': '.terminals.io.v1.FlowEdge',
      '10': 'edges'
    },
  ],
};

/// Descriptor for `FlowPlan`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List flowPlanDescriptor = $convert.base64Decode(
    'CghGbG93UGxhbhIvCgVub2RlcxgBIAMoCzIZLnRlcm1pbmFscy5pby52MS5GbG93Tm9kZVIFbm'
    '9kZXMSLwoFZWRnZXMYAiADKAsyGS50ZXJtaW5hbHMuaW8udjEuRmxvd0VkZ2VSBWVkZ2Vz');

@$core.Deprecated('Use startFlowDescriptor instead')
const StartFlow$json = {
  '1': 'StartFlow',
  '2': [
    {'1': 'flow_id', '3': 1, '4': 1, '5': 9, '10': 'flowId'},
    {
      '1': 'plan',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.FlowPlan',
      '10': 'plan'
    },
  ],
};

/// Descriptor for `StartFlow`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List startFlowDescriptor = $convert.base64Decode(
    'CglTdGFydEZsb3cSFwoHZmxvd19pZBgBIAEoCVIGZmxvd0lkEi0KBHBsYW4YAiABKAsyGS50ZX'
    'JtaW5hbHMuaW8udjEuRmxvd1BsYW5SBHBsYW4=');

@$core.Deprecated('Use patchFlowDescriptor instead')
const PatchFlow$json = {
  '1': 'PatchFlow',
  '2': [
    {'1': 'flow_id', '3': 1, '4': 1, '5': 9, '10': 'flowId'},
    {
      '1': 'plan',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.FlowPlan',
      '10': 'plan'
    },
  ],
};

/// Descriptor for `PatchFlow`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List patchFlowDescriptor = $convert.base64Decode(
    'CglQYXRjaEZsb3cSFwoHZmxvd19pZBgBIAEoCVIGZmxvd0lkEi0KBHBsYW4YAiABKAsyGS50ZX'
    'JtaW5hbHMuaW8udjEuRmxvd1BsYW5SBHBsYW4=');

@$core.Deprecated('Use stopFlowDescriptor instead')
const StopFlow$json = {
  '1': 'StopFlow',
  '2': [
    {'1': 'flow_id', '3': 1, '4': 1, '5': 9, '10': 'flowId'},
  ],
};

/// Descriptor for `StopFlow`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List stopFlowDescriptor =
    $convert.base64Decode('CghTdG9wRmxvdxIXCgdmbG93X2lkGAEgASgJUgZmbG93SWQ=');

@$core.Deprecated('Use deviceRefDescriptor instead')
const DeviceRef$json = {
  '1': 'DeviceRef',
  '2': [
    {'1': 'device_id', '3': 1, '4': 1, '5': 9, '10': 'deviceId'},
  ],
};

/// Descriptor for `DeviceRef`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List deviceRefDescriptor = $convert
    .base64Decode('CglEZXZpY2VSZWYSGwoJZGV2aWNlX2lkGAEgASgJUghkZXZpY2VJZA==');

@$core.Deprecated('Use poseDescriptor instead')
const Pose$json = {
  '1': 'Pose',
  '2': [
    {'1': 'x', '3': 1, '4': 1, '5': 1, '10': 'x'},
    {'1': 'y', '3': 2, '4': 1, '5': 1, '10': 'y'},
    {'1': 'z', '3': 3, '4': 1, '5': 1, '10': 'z'},
    {'1': 'yaw', '3': 4, '4': 1, '5': 1, '10': 'yaw'},
    {'1': 'pitch', '3': 5, '4': 1, '5': 1, '10': 'pitch'},
    {'1': 'roll', '3': 6, '4': 1, '5': 1, '10': 'roll'},
    {'1': 'confidence', '3': 7, '4': 1, '5': 1, '10': 'confidence'},
  ],
};

/// Descriptor for `Pose`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List poseDescriptor = $convert.base64Decode(
    'CgRQb3NlEgwKAXgYASABKAFSAXgSDAoBeRgCIAEoAVIBeRIMCgF6GAMgASgBUgF6EhAKA3lhdx'
    'gEIAEoAVIDeWF3EhQKBXBpdGNoGAUgASgBUgVwaXRjaBISCgRyb2xsGAYgASgBUgRyb2xsEh4K'
    'CmNvbmZpZGVuY2UYByABKAFSCmNvbmZpZGVuY2U=');

@$core.Deprecated('Use locationEstimateDescriptor instead')
const LocationEstimate$json = {
  '1': 'LocationEstimate',
  '2': [
    {'1': 'zone', '3': 1, '4': 1, '5': 9, '10': 'zone'},
    {
      '1': 'pose',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.Pose',
      '10': 'pose'
    },
    {'1': 'radius_m', '3': 3, '4': 1, '5': 1, '10': 'radiusM'},
    {'1': 'confidence', '3': 4, '4': 1, '5': 1, '10': 'confidence'},
    {'1': 'sources', '3': 5, '4': 3, '5': 9, '10': 'sources'},
  ],
};

/// Descriptor for `LocationEstimate`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List locationEstimateDescriptor = $convert.base64Decode(
    'ChBMb2NhdGlvbkVzdGltYXRlEhIKBHpvbmUYASABKAlSBHpvbmUSKQoEcG9zZRgCIAEoCzIVLn'
    'Rlcm1pbmFscy5pby52MS5Qb3NlUgRwb3NlEhkKCHJhZGl1c19tGAMgASgBUgdyYWRpdXNNEh4K'
    'CmNvbmZpZGVuY2UYBCABKAFSCmNvbmZpZGVuY2USGAoHc291cmNlcxgFIAMoCVIHc291cmNlcw'
    '==');

@$core.Deprecated('Use observationProvenanceDescriptor instead')
const ObservationProvenance$json = {
  '1': 'ObservationProvenance',
  '2': [
    {'1': 'flow_id', '3': 1, '4': 1, '5': 9, '10': 'flowId'},
    {'1': 'node_id', '3': 2, '4': 1, '5': 9, '10': 'nodeId'},
    {'1': 'exec_site', '3': 3, '4': 1, '5': 9, '10': 'execSite'},
    {'1': 'model_id', '3': 4, '4': 1, '5': 9, '10': 'modelId'},
    {
      '1': 'calibration_version',
      '3': 5,
      '4': 1,
      '5': 9,
      '10': 'calibrationVersion'
    },
  ],
};

/// Descriptor for `ObservationProvenance`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List observationProvenanceDescriptor = $convert.base64Decode(
    'ChVPYnNlcnZhdGlvblByb3ZlbmFuY2USFwoHZmxvd19pZBgBIAEoCVIGZmxvd0lkEhcKB25vZG'
    'VfaWQYAiABKAlSBm5vZGVJZBIbCglleGVjX3NpdGUYAyABKAlSCGV4ZWNTaXRlEhkKCG1vZGVs'
    'X2lkGAQgASgJUgdtb2RlbElkEi8KE2NhbGlicmF0aW9uX3ZlcnNpb24YBSABKAlSEmNhbGlicm'
    'F0aW9uVmVyc2lvbg==');

@$core.Deprecated('Use artifactRefDescriptor instead')
const ArtifactRef$json = {
  '1': 'ArtifactRef',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {'1': 'kind', '3': 2, '4': 1, '5': 9, '10': 'kind'},
    {
      '1': 'source',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.DeviceRef',
      '10': 'source'
    },
    {'1': 'start_unix_ms', '3': 4, '4': 1, '5': 3, '10': 'startUnixMs'},
    {'1': 'end_unix_ms', '3': 5, '4': 1, '5': 3, '10': 'endUnixMs'},
    {'1': 'uri', '3': 6, '4': 1, '5': 9, '10': 'uri'},
  ],
};

/// Descriptor for `ArtifactRef`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List artifactRefDescriptor = $convert.base64Decode(
    'CgtBcnRpZmFjdFJlZhIOCgJpZBgBIAEoCVICaWQSEgoEa2luZBgCIAEoCVIEa2luZBIyCgZzb3'
    'VyY2UYAyABKAsyGi50ZXJtaW5hbHMuaW8udjEuRGV2aWNlUmVmUgZzb3VyY2USIgoNc3RhcnRf'
    'dW5peF9tcxgEIAEoA1ILc3RhcnRVbml4TXMSHgoLZW5kX3VuaXhfbXMYBSABKANSCWVuZFVuaX'
    'hNcxIQCgN1cmkYBiABKAlSA3VyaQ==');

@$core.Deprecated('Use observationDescriptor instead')
const Observation$json = {
  '1': 'Observation',
  '2': [
    {'1': 'kind', '3': 1, '4': 1, '5': 9, '10': 'kind'},
    {'1': 'subject', '3': 2, '4': 1, '5': 9, '10': 'subject'},
    {
      '1': 'source_device',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.DeviceRef',
      '10': 'sourceDevice'
    },
    {'1': 'occurred_unix_ms', '3': 4, '4': 1, '5': 3, '10': 'occurredUnixMs'},
    {'1': 'confidence', '3': 5, '4': 1, '5': 1, '10': 'confidence'},
    {'1': 'zone', '3': 6, '4': 1, '5': 9, '10': 'zone'},
    {
      '1': 'location',
      '3': 7,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.LocationEstimate',
      '10': 'location'
    },
    {'1': 'track_id', '3': 8, '4': 1, '5': 9, '10': 'trackId'},
    {
      '1': 'attributes',
      '3': 9,
      '4': 3,
      '5': 11,
      '6': '.terminals.io.v1.Observation.AttributesEntry',
      '10': 'attributes'
    },
    {
      '1': 'evidence',
      '3': 10,
      '4': 3,
      '5': 11,
      '6': '.terminals.io.v1.ArtifactRef',
      '10': 'evidence'
    },
    {
      '1': 'provenance',
      '3': 11,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.ObservationProvenance',
      '10': 'provenance'
    },
  ],
  '3': [Observation_AttributesEntry$json],
};

@$core.Deprecated('Use observationDescriptor instead')
const Observation_AttributesEntry$json = {
  '1': 'AttributesEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `Observation`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List observationDescriptor = $convert.base64Decode(
    'CgtPYnNlcnZhdGlvbhISCgRraW5kGAEgASgJUgRraW5kEhgKB3N1YmplY3QYAiABKAlSB3N1Ym'
    'plY3QSPwoNc291cmNlX2RldmljZRgDIAEoCzIaLnRlcm1pbmFscy5pby52MS5EZXZpY2VSZWZS'
    'DHNvdXJjZURldmljZRIoChBvY2N1cnJlZF91bml4X21zGAQgASgDUg5vY2N1cnJlZFVuaXhNcx'
    'IeCgpjb25maWRlbmNlGAUgASgBUgpjb25maWRlbmNlEhIKBHpvbmUYBiABKAlSBHpvbmUSPQoI'
    'bG9jYXRpb24YByABKAsyIS50ZXJtaW5hbHMuaW8udjEuTG9jYXRpb25Fc3RpbWF0ZVIIbG9jYX'
    'Rpb24SGQoIdHJhY2tfaWQYCCABKAlSB3RyYWNrSWQSTAoKYXR0cmlidXRlcxgJIAMoCzIsLnRl'
    'cm1pbmFscy5pby52MS5PYnNlcnZhdGlvbi5BdHRyaWJ1dGVzRW50cnlSCmF0dHJpYnV0ZXMSOA'
    'oIZXZpZGVuY2UYCiADKAsyHC50ZXJtaW5hbHMuaW8udjEuQXJ0aWZhY3RSZWZSCGV2aWRlbmNl'
    'EkYKCnByb3ZlbmFuY2UYCyABKAsyJi50ZXJtaW5hbHMuaW8udjEuT2JzZXJ2YXRpb25Qcm92ZW'
    '5hbmNlUgpwcm92ZW5hbmNlGj0KD0F0dHJpYnV0ZXNFbnRyeRIQCgNrZXkYASABKAlSA2tleRIU'
    'CgV2YWx1ZRgCIAEoCVIFdmFsdWU6AjgB');

@$core.Deprecated('Use observationMessageDescriptor instead')
const ObservationMessage$json = {
  '1': 'ObservationMessage',
  '2': [
    {
      '1': 'observation',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.Observation',
      '10': 'observation'
    },
  ],
};

/// Descriptor for `ObservationMessage`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List observationMessageDescriptor = $convert.base64Decode(
    'ChJPYnNlcnZhdGlvbk1lc3NhZ2USPgoLb2JzZXJ2YXRpb24YASABKAsyHC50ZXJtaW5hbHMuaW'
    '8udjEuT2JzZXJ2YXRpb25SC29ic2VydmF0aW9u');

@$core.Deprecated('Use artifactAvailableDescriptor instead')
const ArtifactAvailable$json = {
  '1': 'ArtifactAvailable',
  '2': [
    {
      '1': 'artifact',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.terminals.io.v1.ArtifactRef',
      '10': 'artifact'
    },
  ],
};

/// Descriptor for `ArtifactAvailable`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List artifactAvailableDescriptor = $convert.base64Decode(
    'ChFBcnRpZmFjdEF2YWlsYWJsZRI4CghhcnRpZmFjdBgBIAEoCzIcLnRlcm1pbmFscy5pby52MS'
    '5BcnRpZmFjdFJlZlIIYXJ0aWZhY3Q=');

@$core.Deprecated('Use requestArtifactDescriptor instead')
const RequestArtifact$json = {
  '1': 'RequestArtifact',
  '2': [
    {'1': 'artifact_id', '3': 1, '4': 1, '5': 9, '10': 'artifactId'},
  ],
};

/// Descriptor for `RequestArtifact`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List requestArtifactDescriptor = $convert.base64Decode(
    'Cg9SZXF1ZXN0QXJ0aWZhY3QSHwoLYXJ0aWZhY3RfaWQYASABKAlSCmFydGlmYWN0SWQ=');

@$core.Deprecated('Use flowStatsDescriptor instead')
const FlowStats$json = {
  '1': 'FlowStats',
  '2': [
    {'1': 'flow_id', '3': 1, '4': 1, '5': 9, '10': 'flowId'},
    {'1': 'cpu_pct', '3': 2, '4': 1, '5': 1, '10': 'cpuPct'},
    {'1': 'mem_mb', '3': 3, '4': 1, '5': 1, '10': 'memMb'},
    {'1': 'dropped_frames', '3': 4, '4': 1, '5': 4, '10': 'droppedFrames'},
    {'1': 'state', '3': 5, '4': 1, '5': 9, '10': 'state'},
    {'1': 'error', '3': 6, '4': 1, '5': 9, '10': 'error'},
  ],
};

/// Descriptor for `FlowStats`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List flowStatsDescriptor = $convert.base64Decode(
    'CglGbG93U3RhdHMSFwoHZmxvd19pZBgBIAEoCVIGZmxvd0lkEhcKB2NwdV9wY3QYAiABKAFSBm'
    'NwdVBjdBIVCgZtZW1fbWIYAyABKAFSBW1lbU1iEiUKDmRyb3BwZWRfZnJhbWVzGAQgASgEUg1k'
    'cm9wcGVkRnJhbWVzEhQKBXN0YXRlGAUgASgJUgVzdGF0ZRIUCgVlcnJvchgGIAEoCVIFZXJyb3'
    'I=');

@$core.Deprecated('Use clockSampleDescriptor instead')
const ClockSample$json = {
  '1': 'ClockSample',
  '2': [
    {'1': 'device_id', '3': 1, '4': 1, '5': 9, '10': 'deviceId'},
    {'1': 'client_unix_ms', '3': 2, '4': 1, '5': 3, '10': 'clientUnixMs'},
    {'1': 'server_unix_ms', '3': 3, '4': 1, '5': 3, '10': 'serverUnixMs'},
    {'1': 'error_ms', '3': 4, '4': 1, '5': 1, '10': 'errorMs'},
  ],
};

/// Descriptor for `ClockSample`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List clockSampleDescriptor = $convert.base64Decode(
    'CgtDbG9ja1NhbXBsZRIbCglkZXZpY2VfaWQYASABKAlSCGRldmljZUlkEiQKDmNsaWVudF91bm'
    'l4X21zGAIgASgDUgxjbGllbnRVbml4TXMSJAoOc2VydmVyX3VuaXhfbXMYAyABKANSDHNlcnZl'
    'clVuaXhNcxIZCghlcnJvcl9tcxgEIAEoAVIHZXJyb3JNcw==');

@$core.Deprecated('Use installBundleDescriptor instead')
const InstallBundle$json = {
  '1': 'InstallBundle',
  '2': [
    {'1': 'bundle_id', '3': 1, '4': 1, '5': 9, '10': 'bundleId'},
    {'1': 'version', '3': 2, '4': 1, '5': 9, '10': 'version'},
    {'1': 'tar_gz', '3': 3, '4': 1, '5': 12, '10': 'tarGz'},
    {'1': 'sha256', '3': 4, '4': 1, '5': 9, '10': 'sha256'},
  ],
};

/// Descriptor for `InstallBundle`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List installBundleDescriptor = $convert.base64Decode(
    'Cg1JbnN0YWxsQnVuZGxlEhsKCWJ1bmRsZV9pZBgBIAEoCVIIYnVuZGxlSWQSGAoHdmVyc2lvbh'
    'gCIAEoCVIHdmVyc2lvbhIVCgZ0YXJfZ3oYAyABKAxSBXRhckd6EhYKBnNoYTI1NhgEIAEoCVIG'
    'c2hhMjU2');

@$core.Deprecated('Use removeBundleDescriptor instead')
const RemoveBundle$json = {
  '1': 'RemoveBundle',
  '2': [
    {'1': 'bundle_id', '3': 1, '4': 1, '5': 9, '10': 'bundleId'},
  ],
};

/// Descriptor for `RemoveBundle`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List removeBundleDescriptor = $convert.base64Decode(
    'CgxSZW1vdmVCdW5kbGUSGwoJYnVuZGxlX2lkGAEgASgJUghidW5kbGVJZA==');
