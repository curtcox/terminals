// This is a generated file - do not edit.
//
// Generated from terminals/capabilities/v1/capabilities.proto.

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

@$core.Deprecated('Use deviceCapabilitiesDescriptor instead')
const DeviceCapabilities$json = {
  '1': 'DeviceCapabilities',
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
    {
      '1': 'screen',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.terminals.capabilities.v1.ScreenCapability',
      '10': 'screen'
    },
    {
      '1': 'keyboard',
      '3': 11,
      '4': 1,
      '5': 11,
      '6': '.terminals.capabilities.v1.KeyboardCapability',
      '10': 'keyboard'
    },
    {
      '1': 'pointer',
      '3': 12,
      '4': 1,
      '5': 11,
      '6': '.terminals.capabilities.v1.PointerCapability',
      '10': 'pointer'
    },
    {
      '1': 'touch',
      '3': 13,
      '4': 1,
      '5': 11,
      '6': '.terminals.capabilities.v1.TouchCapability',
      '10': 'touch'
    },
    {
      '1': 'speakers',
      '3': 14,
      '4': 1,
      '5': 11,
      '6': '.terminals.capabilities.v1.AudioOutputCapability',
      '10': 'speakers'
    },
    {
      '1': 'microphone',
      '3': 15,
      '4': 1,
      '5': 11,
      '6': '.terminals.capabilities.v1.AudioInputCapability',
      '10': 'microphone'
    },
    {
      '1': 'camera',
      '3': 16,
      '4': 1,
      '5': 11,
      '6': '.terminals.capabilities.v1.CameraCapability',
      '10': 'camera'
    },
    {
      '1': 'sensors',
      '3': 17,
      '4': 1,
      '5': 11,
      '6': '.terminals.capabilities.v1.SensorCapability',
      '10': 'sensors'
    },
    {
      '1': 'connectivity',
      '3': 18,
      '4': 1,
      '5': 11,
      '6': '.terminals.capabilities.v1.ConnectivityCapability',
      '10': 'connectivity'
    },
    {
      '1': 'battery',
      '3': 19,
      '4': 1,
      '5': 11,
      '6': '.terminals.capabilities.v1.BatteryCapability',
      '10': 'battery'
    },
    {
      '1': 'edge',
      '3': 20,
      '4': 1,
      '5': 11,
      '6': '.terminals.capabilities.v1.EdgeCapability',
      '10': 'edge'
    },
  ],
};

/// Descriptor for `DeviceCapabilities`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List deviceCapabilitiesDescriptor = $convert.base64Decode(
    'ChJEZXZpY2VDYXBhYmlsaXRpZXMSGwoJZGV2aWNlX2lkGAEgASgJUghkZXZpY2VJZBJFCghpZG'
    'VudGl0eRgCIAEoCzIpLnRlcm1pbmFscy5jYXBhYmlsaXRpZXMudjEuRGV2aWNlSWRlbnRpdHlS'
    'CGlkZW50aXR5EkMKBnNjcmVlbhgKIAEoCzIrLnRlcm1pbmFscy5jYXBhYmlsaXRpZXMudjEuU2'
    'NyZWVuQ2FwYWJpbGl0eVIGc2NyZWVuEkkKCGtleWJvYXJkGAsgASgLMi0udGVybWluYWxzLmNh'
    'cGFiaWxpdGllcy52MS5LZXlib2FyZENhcGFiaWxpdHlSCGtleWJvYXJkEkYKB3BvaW50ZXIYDC'
    'ABKAsyLC50ZXJtaW5hbHMuY2FwYWJpbGl0aWVzLnYxLlBvaW50ZXJDYXBhYmlsaXR5Ugdwb2lu'
    'dGVyEkAKBXRvdWNoGA0gASgLMioudGVybWluYWxzLmNhcGFiaWxpdGllcy52MS5Ub3VjaENhcG'
    'FiaWxpdHlSBXRvdWNoEkwKCHNwZWFrZXJzGA4gASgLMjAudGVybWluYWxzLmNhcGFiaWxpdGll'
    'cy52MS5BdWRpb091dHB1dENhcGFiaWxpdHlSCHNwZWFrZXJzEk8KCm1pY3JvcGhvbmUYDyABKA'
    'syLy50ZXJtaW5hbHMuY2FwYWJpbGl0aWVzLnYxLkF1ZGlvSW5wdXRDYXBhYmlsaXR5UgptaWNy'
    'b3Bob25lEkMKBmNhbWVyYRgQIAEoCzIrLnRlcm1pbmFscy5jYXBhYmlsaXRpZXMudjEuQ2FtZX'
    'JhQ2FwYWJpbGl0eVIGY2FtZXJhEkUKB3NlbnNvcnMYESABKAsyKy50ZXJtaW5hbHMuY2FwYWJp'
    'bGl0aWVzLnYxLlNlbnNvckNhcGFiaWxpdHlSB3NlbnNvcnMSVQoMY29ubmVjdGl2aXR5GBIgAS'
    'gLMjEudGVybWluYWxzLmNhcGFiaWxpdGllcy52MS5Db25uZWN0aXZpdHlDYXBhYmlsaXR5Ugxj'
    'b25uZWN0aXZpdHkSRgoHYmF0dGVyeRgTIAEoCzIsLnRlcm1pbmFscy5jYXBhYmlsaXRpZXMudj'
    'EuQmF0dGVyeUNhcGFiaWxpdHlSB2JhdHRlcnkSPQoEZWRnZRgUIAEoCzIpLnRlcm1pbmFscy5j'
    'YXBhYmlsaXRpZXMudjEuRWRnZUNhcGFiaWxpdHlSBGVkZ2U=');

@$core.Deprecated('Use deviceIdentityDescriptor instead')
const DeviceIdentity$json = {
  '1': 'DeviceIdentity',
  '2': [
    {'1': 'device_name', '3': 1, '4': 1, '5': 9, '10': 'deviceName'},
    {'1': 'device_type', '3': 2, '4': 1, '5': 9, '10': 'deviceType'},
    {'1': 'platform', '3': 3, '4': 1, '5': 9, '10': 'platform'},
  ],
};

/// Descriptor for `DeviceIdentity`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List deviceIdentityDescriptor = $convert.base64Decode(
    'Cg5EZXZpY2VJZGVudGl0eRIfCgtkZXZpY2VfbmFtZRgBIAEoCVIKZGV2aWNlTmFtZRIfCgtkZX'
    'ZpY2VfdHlwZRgCIAEoCVIKZGV2aWNlVHlwZRIaCghwbGF0Zm9ybRgDIAEoCVIIcGxhdGZvcm0=');

@$core.Deprecated('Use screenCapabilityDescriptor instead')
const ScreenCapability$json = {
  '1': 'ScreenCapability',
  '2': [
    {'1': 'width', '3': 1, '4': 1, '5': 5, '10': 'width'},
    {'1': 'height', '3': 2, '4': 1, '5': 5, '10': 'height'},
    {'1': 'density', '3': 3, '4': 1, '5': 1, '10': 'density'},
    {'1': 'touch', '3': 4, '4': 1, '5': 8, '10': 'touch'},
  ],
};

/// Descriptor for `ScreenCapability`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List screenCapabilityDescriptor = $convert.base64Decode(
    'ChBTY3JlZW5DYXBhYmlsaXR5EhQKBXdpZHRoGAEgASgFUgV3aWR0aBIWCgZoZWlnaHQYAiABKA'
    'VSBmhlaWdodBIYCgdkZW5zaXR5GAMgASgBUgdkZW5zaXR5EhQKBXRvdWNoGAQgASgIUgV0b3Vj'
    'aA==');

@$core.Deprecated('Use keyboardCapabilityDescriptor instead')
const KeyboardCapability$json = {
  '1': 'KeyboardCapability',
  '2': [
    {'1': 'physical', '3': 1, '4': 1, '5': 8, '10': 'physical'},
    {'1': 'layout', '3': 2, '4': 1, '5': 9, '10': 'layout'},
  ],
};

/// Descriptor for `KeyboardCapability`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List keyboardCapabilityDescriptor = $convert.base64Decode(
    'ChJLZXlib2FyZENhcGFiaWxpdHkSGgoIcGh5c2ljYWwYASABKAhSCHBoeXNpY2FsEhYKBmxheW'
    '91dBgCIAEoCVIGbGF5b3V0');

@$core.Deprecated('Use pointerCapabilityDescriptor instead')
const PointerCapability$json = {
  '1': 'PointerCapability',
  '2': [
    {'1': 'type', '3': 1, '4': 1, '5': 9, '10': 'type'},
    {'1': 'hover', '3': 2, '4': 1, '5': 8, '10': 'hover'},
  ],
};

/// Descriptor for `PointerCapability`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List pointerCapabilityDescriptor = $convert.base64Decode(
    'ChFQb2ludGVyQ2FwYWJpbGl0eRISCgR0eXBlGAEgASgJUgR0eXBlEhQKBWhvdmVyGAIgASgIUg'
    'Vob3Zlcg==');

@$core.Deprecated('Use touchCapabilityDescriptor instead')
const TouchCapability$json = {
  '1': 'TouchCapability',
  '2': [
    {'1': 'supported', '3': 1, '4': 1, '5': 8, '10': 'supported'},
    {'1': 'max_points', '3': 2, '4': 1, '5': 5, '10': 'maxPoints'},
  ],
};

/// Descriptor for `TouchCapability`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List touchCapabilityDescriptor = $convert.base64Decode(
    'Cg9Ub3VjaENhcGFiaWxpdHkSHAoJc3VwcG9ydGVkGAEgASgIUglzdXBwb3J0ZWQSHQoKbWF4X3'
    'BvaW50cxgCIAEoBVIJbWF4UG9pbnRz');

@$core.Deprecated('Use audioOutputCapabilityDescriptor instead')
const AudioOutputCapability$json = {
  '1': 'AudioOutputCapability',
  '2': [
    {'1': 'channels', '3': 1, '4': 1, '5': 5, '10': 'channels'},
    {'1': 'sample_rates', '3': 2, '4': 3, '5': 5, '10': 'sampleRates'},
  ],
};

/// Descriptor for `AudioOutputCapability`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List audioOutputCapabilityDescriptor = $convert.base64Decode(
    'ChVBdWRpb091dHB1dENhcGFiaWxpdHkSGgoIY2hhbm5lbHMYASABKAVSCGNoYW5uZWxzEiEKDH'
    'NhbXBsZV9yYXRlcxgCIAMoBVILc2FtcGxlUmF0ZXM=');

@$core.Deprecated('Use audioInputCapabilityDescriptor instead')
const AudioInputCapability$json = {
  '1': 'AudioInputCapability',
  '2': [
    {'1': 'channels', '3': 1, '4': 1, '5': 5, '10': 'channels'},
    {'1': 'sample_rates', '3': 2, '4': 3, '5': 5, '10': 'sampleRates'},
  ],
};

/// Descriptor for `AudioInputCapability`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List audioInputCapabilityDescriptor = $convert.base64Decode(
    'ChRBdWRpb0lucHV0Q2FwYWJpbGl0eRIaCghjaGFubmVscxgBIAEoBVIIY2hhbm5lbHMSIQoMc2'
    'FtcGxlX3JhdGVzGAIgAygFUgtzYW1wbGVSYXRlcw==');

@$core.Deprecated('Use cameraLensDescriptor instead')
const CameraLens$json = {
  '1': 'CameraLens',
  '2': [
    {'1': 'width', '3': 1, '4': 1, '5': 5, '10': 'width'},
    {'1': 'height', '3': 2, '4': 1, '5': 5, '10': 'height'},
    {'1': 'fps', '3': 3, '4': 1, '5': 5, '10': 'fps'},
  ],
};

/// Descriptor for `CameraLens`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List cameraLensDescriptor = $convert.base64Decode(
    'CgpDYW1lcmFMZW5zEhQKBXdpZHRoGAEgASgFUgV3aWR0aBIWCgZoZWlnaHQYAiABKAVSBmhlaW'
    'dodBIQCgNmcHMYAyABKAVSA2Zwcw==');

@$core.Deprecated('Use cameraCapabilityDescriptor instead')
const CameraCapability$json = {
  '1': 'CameraCapability',
  '2': [
    {
      '1': 'front',
      '3': 1,
      '4': 1,
      '5': 11,
      '6': '.terminals.capabilities.v1.CameraLens',
      '10': 'front'
    },
    {
      '1': 'back',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.terminals.capabilities.v1.CameraLens',
      '10': 'back'
    },
  ],
};

/// Descriptor for `CameraCapability`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List cameraCapabilityDescriptor = $convert.base64Decode(
    'ChBDYW1lcmFDYXBhYmlsaXR5EjsKBWZyb250GAEgASgLMiUudGVybWluYWxzLmNhcGFiaWxpdG'
    'llcy52MS5DYW1lcmFMZW5zUgVmcm9udBI5CgRiYWNrGAIgASgLMiUudGVybWluYWxzLmNhcGFi'
    'aWxpdGllcy52MS5DYW1lcmFMZW5zUgRiYWNr');

@$core.Deprecated('Use sensorCapabilityDescriptor instead')
const SensorCapability$json = {
  '1': 'SensorCapability',
  '2': [
    {'1': 'accelerometer', '3': 1, '4': 1, '5': 8, '10': 'accelerometer'},
    {'1': 'gyroscope', '3': 2, '4': 1, '5': 8, '10': 'gyroscope'},
    {'1': 'compass', '3': 3, '4': 1, '5': 8, '10': 'compass'},
    {'1': 'ambient_light', '3': 4, '4': 1, '5': 8, '10': 'ambientLight'},
    {'1': 'proximity', '3': 5, '4': 1, '5': 8, '10': 'proximity'},
    {'1': 'gps', '3': 6, '4': 1, '5': 8, '10': 'gps'},
  ],
};

/// Descriptor for `SensorCapability`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sensorCapabilityDescriptor = $convert.base64Decode(
    'ChBTZW5zb3JDYXBhYmlsaXR5EiQKDWFjY2VsZXJvbWV0ZXIYASABKAhSDWFjY2VsZXJvbWV0ZX'
    'ISHAoJZ3lyb3Njb3BlGAIgASgIUglneXJvc2NvcGUSGAoHY29tcGFzcxgDIAEoCFIHY29tcGFz'
    'cxIjCg1hbWJpZW50X2xpZ2h0GAQgASgIUgxhbWJpZW50TGlnaHQSHAoJcHJveGltaXR5GAUgAS'
    'gIUglwcm94aW1pdHkSEAoDZ3BzGAYgASgIUgNncHM=');

@$core.Deprecated('Use connectivityCapabilityDescriptor instead')
const ConnectivityCapability$json = {
  '1': 'ConnectivityCapability',
  '2': [
    {
      '1': 'bluetooth_version',
      '3': 1,
      '4': 1,
      '5': 9,
      '10': 'bluetoothVersion'
    },
    {
      '1': 'wifi_signal_strength',
      '3': 2,
      '4': 1,
      '5': 8,
      '10': 'wifiSignalStrength'
    },
    {'1': 'usb_host', '3': 3, '4': 1, '5': 8, '10': 'usbHost'},
    {'1': 'usb_ports', '3': 4, '4': 1, '5': 5, '10': 'usbPorts'},
    {'1': 'nfc', '3': 5, '4': 1, '5': 8, '10': 'nfc'},
  ],
};

/// Descriptor for `ConnectivityCapability`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List connectivityCapabilityDescriptor = $convert.base64Decode(
    'ChZDb25uZWN0aXZpdHlDYXBhYmlsaXR5EisKEWJsdWV0b290aF92ZXJzaW9uGAEgASgJUhBibH'
    'VldG9vdGhWZXJzaW9uEjAKFHdpZmlfc2lnbmFsX3N0cmVuZ3RoGAIgASgIUhJ3aWZpU2lnbmFs'
    'U3RyZW5ndGgSGQoIdXNiX2hvc3QYAyABKAhSB3VzYkhvc3QSGwoJdXNiX3BvcnRzGAQgASgFUg'
    'h1c2JQb3J0cxIQCgNuZmMYBSABKAhSA25mYw==');

@$core.Deprecated('Use batteryCapabilityDescriptor instead')
const BatteryCapability$json = {
  '1': 'BatteryCapability',
  '2': [
    {'1': 'level', '3': 1, '4': 1, '5': 2, '10': 'level'},
    {'1': 'charging', '3': 2, '4': 1, '5': 8, '10': 'charging'},
  ],
};

/// Descriptor for `BatteryCapability`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List batteryCapabilityDescriptor = $convert.base64Decode(
    'ChFCYXR0ZXJ5Q2FwYWJpbGl0eRIUCgVsZXZlbBgBIAEoAlIFbGV2ZWwSGgoIY2hhcmdpbmcYAi'
    'ABKAhSCGNoYXJnaW5n');

@$core.Deprecated('Use edgeCapabilityDescriptor instead')
const EdgeCapability$json = {
  '1': 'EdgeCapability',
  '2': [
    {'1': 'runtimes', '3': 1, '4': 3, '5': 9, '10': 'runtimes'},
    {
      '1': 'compute',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.terminals.capabilities.v1.EdgeComputeCapability',
      '10': 'compute'
    },
    {'1': 'operators', '3': 3, '4': 3, '5': 9, '10': 'operators'},
    {
      '1': 'retention',
      '3': 4,
      '4': 1,
      '5': 11,
      '6': '.terminals.capabilities.v1.EdgeRetentionCapability',
      '10': 'retention'
    },
    {
      '1': 'timing',
      '3': 5,
      '4': 1,
      '5': 11,
      '6': '.terminals.capabilities.v1.EdgeTimingCapability',
      '10': 'timing'
    },
    {
      '1': 'geometry',
      '3': 6,
      '4': 1,
      '5': 11,
      '6': '.terminals.capabilities.v1.EdgeGeometryCapability',
      '10': 'geometry'
    },
  ],
};

/// Descriptor for `EdgeCapability`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List edgeCapabilityDescriptor = $convert.base64Decode(
    'Cg5FZGdlQ2FwYWJpbGl0eRIaCghydW50aW1lcxgBIAMoCVIIcnVudGltZXMSSgoHY29tcHV0ZR'
    'gCIAEoCzIwLnRlcm1pbmFscy5jYXBhYmlsaXRpZXMudjEuRWRnZUNvbXB1dGVDYXBhYmlsaXR5'
    'Ugdjb21wdXRlEhwKCW9wZXJhdG9ycxgDIAMoCVIJb3BlcmF0b3JzElAKCXJldGVudGlvbhgEIA'
    'EoCzIyLnRlcm1pbmFscy5jYXBhYmlsaXRpZXMudjEuRWRnZVJldGVudGlvbkNhcGFiaWxpdHlS'
    'CXJldGVudGlvbhJHCgZ0aW1pbmcYBSABKAsyLy50ZXJtaW5hbHMuY2FwYWJpbGl0aWVzLnYxLk'
    'VkZ2VUaW1pbmdDYXBhYmlsaXR5UgZ0aW1pbmcSTQoIZ2VvbWV0cnkYBiABKAsyMS50ZXJtaW5h'
    'bHMuY2FwYWJpbGl0aWVzLnYxLkVkZ2VHZW9tZXRyeUNhcGFiaWxpdHlSCGdlb21ldHJ5');

@$core.Deprecated('Use edgeComputeCapabilityDescriptor instead')
const EdgeComputeCapability$json = {
  '1': 'EdgeComputeCapability',
  '2': [
    {'1': 'cpu_realtime', '3': 1, '4': 1, '5': 5, '10': 'cpuRealtime'},
    {'1': 'gpu_realtime', '3': 2, '4': 1, '5': 5, '10': 'gpuRealtime'},
    {'1': 'npu_realtime', '3': 3, '4': 1, '5': 5, '10': 'npuRealtime'},
    {'1': 'mem_mb', '3': 4, '4': 1, '5': 5, '10': 'memMb'},
  ],
};

/// Descriptor for `EdgeComputeCapability`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List edgeComputeCapabilityDescriptor = $convert.base64Decode(
    'ChVFZGdlQ29tcHV0ZUNhcGFiaWxpdHkSIQoMY3B1X3JlYWx0aW1lGAEgASgFUgtjcHVSZWFsdG'
    'ltZRIhCgxncHVfcmVhbHRpbWUYAiABKAVSC2dwdVJlYWx0aW1lEiEKDG5wdV9yZWFsdGltZRgD'
    'IAEoBVILbnB1UmVhbHRpbWUSFQoGbWVtX21iGAQgASgFUgVtZW1NYg==');

@$core.Deprecated('Use edgeRetentionCapabilityDescriptor instead')
const EdgeRetentionCapability$json = {
  '1': 'EdgeRetentionCapability',
  '2': [
    {'1': 'audio_sec', '3': 1, '4': 1, '5': 5, '10': 'audioSec'},
    {'1': 'video_sec', '3': 2, '4': 1, '5': 5, '10': 'videoSec'},
    {'1': 'sensor_sec', '3': 3, '4': 1, '5': 5, '10': 'sensorSec'},
    {'1': 'radio_sec', '3': 4, '4': 1, '5': 5, '10': 'radioSec'},
  ],
};

/// Descriptor for `EdgeRetentionCapability`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List edgeRetentionCapabilityDescriptor = $convert.base64Decode(
    'ChdFZGdlUmV0ZW50aW9uQ2FwYWJpbGl0eRIbCglhdWRpb19zZWMYASABKAVSCGF1ZGlvU2VjEh'
    'sKCXZpZGVvX3NlYxgCIAEoBVIIdmlkZW9TZWMSHQoKc2Vuc29yX3NlYxgDIAEoBVIJc2Vuc29y'
    'U2VjEhsKCXJhZGlvX3NlYxgEIAEoBVIIcmFkaW9TZWM=');

@$core.Deprecated('Use edgeTimingCapabilityDescriptor instead')
const EdgeTimingCapability$json = {
  '1': 'EdgeTimingCapability',
  '2': [
    {'1': 'sync_error_ms', '3': 1, '4': 1, '5': 1, '10': 'syncErrorMs'},
  ],
};

/// Descriptor for `EdgeTimingCapability`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List edgeTimingCapabilityDescriptor = $convert.base64Decode(
    'ChRFZGdlVGltaW5nQ2FwYWJpbGl0eRIiCg1zeW5jX2Vycm9yX21zGAEgASgBUgtzeW5jRXJyb3'
    'JNcw==');

@$core.Deprecated('Use edgeGeometryCapabilityDescriptor instead')
const EdgeGeometryCapability$json = {
  '1': 'EdgeGeometryCapability',
  '2': [
    {'1': 'mic_array', '3': 1, '4': 1, '5': 8, '10': 'micArray'},
    {
      '1': 'camera_intrinsics',
      '3': 2,
      '4': 1,
      '5': 8,
      '10': 'cameraIntrinsics'
    },
    {'1': 'compass', '3': 3, '4': 1, '5': 8, '10': 'compass'},
  ],
};

/// Descriptor for `EdgeGeometryCapability`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List edgeGeometryCapabilityDescriptor = $convert.base64Decode(
    'ChZFZGdlR2VvbWV0cnlDYXBhYmlsaXR5EhsKCW1pY19hcnJheRgBIAEoCFIIbWljQXJyYXkSKw'
    'oRY2FtZXJhX2ludHJpbnNpY3MYAiABKAhSEGNhbWVyYUludHJpbnNpY3MSGAoHY29tcGFzcxgD'
    'IAEoCFIHY29tcGFzcw==');
