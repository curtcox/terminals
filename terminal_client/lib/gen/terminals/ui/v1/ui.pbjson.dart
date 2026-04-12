// This is a generated file - do not edit.
//
// Generated from terminals/ui/v1/ui.proto.

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

@$core.Deprecated('Use setUIDescriptor instead')
const SetUI$json = {
  '1': 'SetUI',
  '2': [
    {'1': 'device_id', '3': 1, '4': 1, '5': 9, '10': 'deviceId'},
    {
      '1': 'root',
      '3': 2,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.Node',
      '10': 'root'
    },
  ],
};

/// Descriptor for `SetUI`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List setUIDescriptor = $convert.base64Decode(
    'CgVTZXRVSRIbCglkZXZpY2VfaWQYASABKAlSCGRldmljZUlkEikKBHJvb3QYAiABKAsyFS50ZX'
    'JtaW5hbHMudWkudjEuTm9kZVIEcm9vdA==');

@$core.Deprecated('Use updateUIDescriptor instead')
const UpdateUI$json = {
  '1': 'UpdateUI',
  '2': [
    {'1': 'device_id', '3': 1, '4': 1, '5': 9, '10': 'deviceId'},
    {'1': 'component_id', '3': 2, '4': 1, '5': 9, '10': 'componentId'},
    {
      '1': 'node',
      '3': 3,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.Node',
      '10': 'node'
    },
  ],
};

/// Descriptor for `UpdateUI`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List updateUIDescriptor = $convert.base64Decode(
    'CghVcGRhdGVVSRIbCglkZXZpY2VfaWQYASABKAlSCGRldmljZUlkEiEKDGNvbXBvbmVudF9pZB'
    'gCIAEoCVILY29tcG9uZW50SWQSKQoEbm9kZRgDIAEoCzIVLnRlcm1pbmFscy51aS52MS5Ob2Rl'
    'UgRub2Rl');

@$core.Deprecated('Use transitionUIDescriptor instead')
const TransitionUI$json = {
  '1': 'TransitionUI',
  '2': [
    {'1': 'device_id', '3': 1, '4': 1, '5': 9, '10': 'deviceId'},
    {'1': 'transition', '3': 2, '4': 1, '5': 9, '10': 'transition'},
    {'1': 'duration_ms', '3': 3, '4': 1, '5': 5, '10': 'durationMs'},
  ],
};

/// Descriptor for `TransitionUI`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List transitionUIDescriptor = $convert.base64Decode(
    'CgxUcmFuc2l0aW9uVUkSGwoJZGV2aWNlX2lkGAEgASgJUghkZXZpY2VJZBIeCgp0cmFuc2l0aW'
    '9uGAIgASgJUgp0cmFuc2l0aW9uEh8KC2R1cmF0aW9uX21zGAMgASgFUgpkdXJhdGlvbk1z');

@$core.Deprecated('Use notificationDescriptor instead')
const Notification$json = {
  '1': 'Notification',
  '2': [
    {'1': 'device_id', '3': 1, '4': 1, '5': 9, '10': 'deviceId'},
    {'1': 'title', '3': 2, '4': 1, '5': 9, '10': 'title'},
    {'1': 'body', '3': 3, '4': 1, '5': 9, '10': 'body'},
    {'1': 'level', '3': 4, '4': 1, '5': 9, '10': 'level'},
  ],
};

/// Descriptor for `Notification`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List notificationDescriptor = $convert.base64Decode(
    'CgxOb3RpZmljYXRpb24SGwoJZGV2aWNlX2lkGAEgASgJUghkZXZpY2VJZBIUCgV0aXRsZRgCIA'
    'EoCVIFdGl0bGUSEgoEYm9keRgDIAEoCVIEYm9keRIUCgVsZXZlbBgEIAEoCVIFbGV2ZWw=');

@$core.Deprecated('Use nodeDescriptor instead')
const Node$json = {
  '1': 'Node',
  '2': [
    {'1': 'id', '3': 1, '4': 1, '5': 9, '10': 'id'},
    {
      '1': 'props',
      '3': 2,
      '4': 3,
      '5': 11,
      '6': '.terminals.ui.v1.Node.PropsEntry',
      '10': 'props'
    },
    {
      '1': 'children',
      '3': 3,
      '4': 3,
      '5': 11,
      '6': '.terminals.ui.v1.Node',
      '10': 'children'
    },
    {
      '1': 'stack',
      '3': 10,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.StackWidget',
      '9': 0,
      '10': 'stack'
    },
    {
      '1': 'row',
      '3': 11,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.RowWidget',
      '9': 0,
      '10': 'row'
    },
    {
      '1': 'grid',
      '3': 12,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.GridWidget',
      '9': 0,
      '10': 'grid'
    },
    {
      '1': 'scroll',
      '3': 13,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.ScrollWidget',
      '9': 0,
      '10': 'scroll'
    },
    {
      '1': 'padding',
      '3': 14,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.PaddingWidget',
      '9': 0,
      '10': 'padding'
    },
    {
      '1': 'center',
      '3': 15,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.CenterWidget',
      '9': 0,
      '10': 'center'
    },
    {
      '1': 'expand',
      '3': 16,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.ExpandWidget',
      '9': 0,
      '10': 'expand'
    },
    {
      '1': 'text',
      '3': 17,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.TextWidget',
      '9': 0,
      '10': 'text'
    },
    {
      '1': 'image',
      '3': 18,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.ImageWidget',
      '9': 0,
      '10': 'image'
    },
    {
      '1': 'video_surface',
      '3': 19,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.VideoSurfaceWidget',
      '9': 0,
      '10': 'videoSurface'
    },
    {
      '1': 'audio_visualizer',
      '3': 20,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.AudioVisualizerWidget',
      '9': 0,
      '10': 'audioVisualizer'
    },
    {
      '1': 'canvas',
      '3': 21,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.CanvasWidget',
      '9': 0,
      '10': 'canvas'
    },
    {
      '1': 'text_input',
      '3': 22,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.TextInputWidget',
      '9': 0,
      '10': 'textInput'
    },
    {
      '1': 'button',
      '3': 23,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.ButtonWidget',
      '9': 0,
      '10': 'button'
    },
    {
      '1': 'slider',
      '3': 24,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.SliderWidget',
      '9': 0,
      '10': 'slider'
    },
    {
      '1': 'toggle',
      '3': 25,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.ToggleWidget',
      '9': 0,
      '10': 'toggle'
    },
    {
      '1': 'dropdown',
      '3': 26,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.DropdownWidget',
      '9': 0,
      '10': 'dropdown'
    },
    {
      '1': 'gesture_area',
      '3': 27,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.GestureAreaWidget',
      '9': 0,
      '10': 'gestureArea'
    },
    {
      '1': 'overlay',
      '3': 28,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.OverlayWidget',
      '9': 0,
      '10': 'overlay'
    },
    {
      '1': 'progress',
      '3': 29,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.ProgressWidget',
      '9': 0,
      '10': 'progress'
    },
    {
      '1': 'fullscreen',
      '3': 30,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.FullscreenWidget',
      '9': 0,
      '10': 'fullscreen'
    },
    {
      '1': 'keep_awake',
      '3': 31,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.KeepAwakeWidget',
      '9': 0,
      '10': 'keepAwake'
    },
    {
      '1': 'brightness',
      '3': 32,
      '4': 1,
      '5': 11,
      '6': '.terminals.ui.v1.BrightnessWidget',
      '9': 0,
      '10': 'brightness'
    },
  ],
  '3': [Node_PropsEntry$json],
  '8': [
    {'1': 'widget'},
  ],
};

@$core.Deprecated('Use nodeDescriptor instead')
const Node_PropsEntry$json = {
  '1': 'PropsEntry',
  '2': [
    {'1': 'key', '3': 1, '4': 1, '5': 9, '10': 'key'},
    {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
  '7': {'7': true},
};

/// Descriptor for `Node`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List nodeDescriptor = $convert.base64Decode(
    'CgROb2RlEg4KAmlkGAEgASgJUgJpZBI2CgVwcm9wcxgCIAMoCzIgLnRlcm1pbmFscy51aS52MS'
    '5Ob2RlLlByb3BzRW50cnlSBXByb3BzEjEKCGNoaWxkcmVuGAMgAygLMhUudGVybWluYWxzLnVp'
    'LnYxLk5vZGVSCGNoaWxkcmVuEjQKBXN0YWNrGAogASgLMhwudGVybWluYWxzLnVpLnYxLlN0YW'
    'NrV2lkZ2V0SABSBXN0YWNrEi4KA3JvdxgLIAEoCzIaLnRlcm1pbmFscy51aS52MS5Sb3dXaWRn'
    'ZXRIAFIDcm93EjEKBGdyaWQYDCABKAsyGy50ZXJtaW5hbHMudWkudjEuR3JpZFdpZGdldEgAUg'
    'RncmlkEjcKBnNjcm9sbBgNIAEoCzIdLnRlcm1pbmFscy51aS52MS5TY3JvbGxXaWRnZXRIAFIG'
    'c2Nyb2xsEjoKB3BhZGRpbmcYDiABKAsyHi50ZXJtaW5hbHMudWkudjEuUGFkZGluZ1dpZGdldE'
    'gAUgdwYWRkaW5nEjcKBmNlbnRlchgPIAEoCzIdLnRlcm1pbmFscy51aS52MS5DZW50ZXJXaWRn'
    'ZXRIAFIGY2VudGVyEjcKBmV4cGFuZBgQIAEoCzIdLnRlcm1pbmFscy51aS52MS5FeHBhbmRXaW'
    'RnZXRIAFIGZXhwYW5kEjEKBHRleHQYESABKAsyGy50ZXJtaW5hbHMudWkudjEuVGV4dFdpZGdl'
    'dEgAUgR0ZXh0EjQKBWltYWdlGBIgASgLMhwudGVybWluYWxzLnVpLnYxLkltYWdlV2lkZ2V0SA'
    'BSBWltYWdlEkoKDXZpZGVvX3N1cmZhY2UYEyABKAsyIy50ZXJtaW5hbHMudWkudjEuVmlkZW9T'
    'dXJmYWNlV2lkZ2V0SABSDHZpZGVvU3VyZmFjZRJTChBhdWRpb192aXN1YWxpemVyGBQgASgLMi'
    'YudGVybWluYWxzLnVpLnYxLkF1ZGlvVmlzdWFsaXplcldpZGdldEgAUg9hdWRpb1Zpc3VhbGl6'
    'ZXISNwoGY2FudmFzGBUgASgLMh0udGVybWluYWxzLnVpLnYxLkNhbnZhc1dpZGdldEgAUgZjYW'
    '52YXMSQQoKdGV4dF9pbnB1dBgWIAEoCzIgLnRlcm1pbmFscy51aS52MS5UZXh0SW5wdXRXaWRn'
    'ZXRIAFIJdGV4dElucHV0EjcKBmJ1dHRvbhgXIAEoCzIdLnRlcm1pbmFscy51aS52MS5CdXR0b2'
    '5XaWRnZXRIAFIGYnV0dG9uEjcKBnNsaWRlchgYIAEoCzIdLnRlcm1pbmFscy51aS52MS5TbGlk'
    'ZXJXaWRnZXRIAFIGc2xpZGVyEjcKBnRvZ2dsZRgZIAEoCzIdLnRlcm1pbmFscy51aS52MS5Ub2'
    'dnbGVXaWRnZXRIAFIGdG9nZ2xlEj0KCGRyb3Bkb3duGBogASgLMh8udGVybWluYWxzLnVpLnYx'
    'LkRyb3Bkb3duV2lkZ2V0SABSCGRyb3Bkb3duEkcKDGdlc3R1cmVfYXJlYRgbIAEoCzIiLnRlcm'
    '1pbmFscy51aS52MS5HZXN0dXJlQXJlYVdpZGdldEgAUgtnZXN0dXJlQXJlYRI6CgdvdmVybGF5'
    'GBwgASgLMh4udGVybWluYWxzLnVpLnYxLk92ZXJsYXlXaWRnZXRIAFIHb3ZlcmxheRI9Cghwcm'
    '9ncmVzcxgdIAEoCzIfLnRlcm1pbmFscy51aS52MS5Qcm9ncmVzc1dpZGdldEgAUghwcm9ncmVz'
    'cxJDCgpmdWxsc2NyZWVuGB4gASgLMiEudGVybWluYWxzLnVpLnYxLkZ1bGxzY3JlZW5XaWRnZX'
    'RIAFIKZnVsbHNjcmVlbhJBCgprZWVwX2F3YWtlGB8gASgLMiAudGVybWluYWxzLnVpLnYxLktl'
    'ZXBBd2FrZVdpZGdldEgAUglrZWVwQXdha2USQwoKYnJpZ2h0bmVzcxggIAEoCzIhLnRlcm1pbm'
    'Fscy51aS52MS5CcmlnaHRuZXNzV2lkZ2V0SABSCmJyaWdodG5lc3MaOAoKUHJvcHNFbnRyeRIQ'
    'CgNrZXkYASABKAlSA2tleRIUCgV2YWx1ZRgCIAEoCVIFdmFsdWU6AjgBQggKBndpZGdldA==');

@$core.Deprecated('Use stackWidgetDescriptor instead')
const StackWidget$json = {
  '1': 'StackWidget',
};

/// Descriptor for `StackWidget`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List stackWidgetDescriptor =
    $convert.base64Decode('CgtTdGFja1dpZGdldA==');

@$core.Deprecated('Use rowWidgetDescriptor instead')
const RowWidget$json = {
  '1': 'RowWidget',
};

/// Descriptor for `RowWidget`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List rowWidgetDescriptor =
    $convert.base64Decode('CglSb3dXaWRnZXQ=');

@$core.Deprecated('Use gridWidgetDescriptor instead')
const GridWidget$json = {
  '1': 'GridWidget',
  '2': [
    {'1': 'columns', '3': 1, '4': 1, '5': 5, '10': 'columns'},
  ],
};

/// Descriptor for `GridWidget`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List gridWidgetDescriptor = $convert
    .base64Decode('CgpHcmlkV2lkZ2V0EhgKB2NvbHVtbnMYASABKAVSB2NvbHVtbnM=');

@$core.Deprecated('Use scrollWidgetDescriptor instead')
const ScrollWidget$json = {
  '1': 'ScrollWidget',
  '2': [
    {'1': 'direction', '3': 1, '4': 1, '5': 9, '10': 'direction'},
  ],
};

/// Descriptor for `ScrollWidget`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List scrollWidgetDescriptor = $convert.base64Decode(
    'CgxTY3JvbGxXaWRnZXQSHAoJZGlyZWN0aW9uGAEgASgJUglkaXJlY3Rpb24=');

@$core.Deprecated('Use paddingWidgetDescriptor instead')
const PaddingWidget$json = {
  '1': 'PaddingWidget',
  '2': [
    {'1': 'all', '3': 1, '4': 1, '5': 5, '10': 'all'},
  ],
};

/// Descriptor for `PaddingWidget`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List paddingWidgetDescriptor =
    $convert.base64Decode('Cg1QYWRkaW5nV2lkZ2V0EhAKA2FsbBgBIAEoBVIDYWxs');

@$core.Deprecated('Use centerWidgetDescriptor instead')
const CenterWidget$json = {
  '1': 'CenterWidget',
};

/// Descriptor for `CenterWidget`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List centerWidgetDescriptor =
    $convert.base64Decode('CgxDZW50ZXJXaWRnZXQ=');

@$core.Deprecated('Use expandWidgetDescriptor instead')
const ExpandWidget$json = {
  '1': 'ExpandWidget',
};

/// Descriptor for `ExpandWidget`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List expandWidgetDescriptor =
    $convert.base64Decode('CgxFeHBhbmRXaWRnZXQ=');

@$core.Deprecated('Use textWidgetDescriptor instead')
const TextWidget$json = {
  '1': 'TextWidget',
  '2': [
    {'1': 'value', '3': 1, '4': 1, '5': 9, '10': 'value'},
    {'1': 'style', '3': 2, '4': 1, '5': 9, '10': 'style'},
    {'1': 'color', '3': 3, '4': 1, '5': 9, '10': 'color'},
  ],
};

/// Descriptor for `TextWidget`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List textWidgetDescriptor = $convert.base64Decode(
    'CgpUZXh0V2lkZ2V0EhQKBXZhbHVlGAEgASgJUgV2YWx1ZRIUCgVzdHlsZRgCIAEoCVIFc3R5bG'
    'USFAoFY29sb3IYAyABKAlSBWNvbG9y');

@$core.Deprecated('Use imageWidgetDescriptor instead')
const ImageWidget$json = {
  '1': 'ImageWidget',
  '2': [
    {'1': 'url', '3': 1, '4': 1, '5': 9, '10': 'url'},
  ],
};

/// Descriptor for `ImageWidget`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List imageWidgetDescriptor =
    $convert.base64Decode('CgtJbWFnZVdpZGdldBIQCgN1cmwYASABKAlSA3VybA==');

@$core.Deprecated('Use videoSurfaceWidgetDescriptor instead')
const VideoSurfaceWidget$json = {
  '1': 'VideoSurfaceWidget',
  '2': [
    {'1': 'track_id', '3': 1, '4': 1, '5': 9, '10': 'trackId'},
  ],
};

/// Descriptor for `VideoSurfaceWidget`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List videoSurfaceWidgetDescriptor =
    $convert.base64Decode(
        'ChJWaWRlb1N1cmZhY2VXaWRnZXQSGQoIdHJhY2tfaWQYASABKAlSB3RyYWNrSWQ=');

@$core.Deprecated('Use audioVisualizerWidgetDescriptor instead')
const AudioVisualizerWidget$json = {
  '1': 'AudioVisualizerWidget',
  '2': [
    {'1': 'stream_id', '3': 1, '4': 1, '5': 9, '10': 'streamId'},
  ],
};

/// Descriptor for `AudioVisualizerWidget`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List audioVisualizerWidgetDescriptor = $convert.base64Decode(
    'ChVBdWRpb1Zpc3VhbGl6ZXJXaWRnZXQSGwoJc3RyZWFtX2lkGAEgASgJUghzdHJlYW1JZA==');

@$core.Deprecated('Use canvasWidgetDescriptor instead')
const CanvasWidget$json = {
  '1': 'CanvasWidget',
  '2': [
    {'1': 'draw_ops_json', '3': 1, '4': 1, '5': 9, '10': 'drawOpsJson'},
  ],
};

/// Descriptor for `CanvasWidget`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List canvasWidgetDescriptor = $convert.base64Decode(
    'CgxDYW52YXNXaWRnZXQSIgoNZHJhd19vcHNfanNvbhgBIAEoCVILZHJhd09wc0pzb24=');

@$core.Deprecated('Use textInputWidgetDescriptor instead')
const TextInputWidget$json = {
  '1': 'TextInputWidget',
  '2': [
    {'1': 'placeholder', '3': 1, '4': 1, '5': 9, '10': 'placeholder'},
    {'1': 'autofocus', '3': 2, '4': 1, '5': 8, '10': 'autofocus'},
  ],
};

/// Descriptor for `TextInputWidget`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List textInputWidgetDescriptor = $convert.base64Decode(
    'Cg9UZXh0SW5wdXRXaWRnZXQSIAoLcGxhY2Vob2xkZXIYASABKAlSC3BsYWNlaG9sZGVyEhwKCW'
    'F1dG9mb2N1cxgCIAEoCFIJYXV0b2ZvY3Vz');

@$core.Deprecated('Use buttonWidgetDescriptor instead')
const ButtonWidget$json = {
  '1': 'ButtonWidget',
  '2': [
    {'1': 'label', '3': 1, '4': 1, '5': 9, '10': 'label'},
    {'1': 'action', '3': 2, '4': 1, '5': 9, '10': 'action'},
  ],
};

/// Descriptor for `ButtonWidget`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List buttonWidgetDescriptor = $convert.base64Decode(
    'CgxCdXR0b25XaWRnZXQSFAoFbGFiZWwYASABKAlSBWxhYmVsEhYKBmFjdGlvbhgCIAEoCVIGYW'
    'N0aW9u');

@$core.Deprecated('Use sliderWidgetDescriptor instead')
const SliderWidget$json = {
  '1': 'SliderWidget',
  '2': [
    {'1': 'min', '3': 1, '4': 1, '5': 1, '10': 'min'},
    {'1': 'max', '3': 2, '4': 1, '5': 1, '10': 'max'},
    {'1': 'value', '3': 3, '4': 1, '5': 1, '10': 'value'},
  ],
};

/// Descriptor for `SliderWidget`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List sliderWidgetDescriptor = $convert.base64Decode(
    'CgxTbGlkZXJXaWRnZXQSEAoDbWluGAEgASgBUgNtaW4SEAoDbWF4GAIgASgBUgNtYXgSFAoFdm'
    'FsdWUYAyABKAFSBXZhbHVl');

@$core.Deprecated('Use toggleWidgetDescriptor instead')
const ToggleWidget$json = {
  '1': 'ToggleWidget',
  '2': [
    {'1': 'value', '3': 1, '4': 1, '5': 8, '10': 'value'},
  ],
};

/// Descriptor for `ToggleWidget`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List toggleWidgetDescriptor =
    $convert.base64Decode('CgxUb2dnbGVXaWRnZXQSFAoFdmFsdWUYASABKAhSBXZhbHVl');

@$core.Deprecated('Use dropdownWidgetDescriptor instead')
const DropdownWidget$json = {
  '1': 'DropdownWidget',
  '2': [
    {'1': 'options', '3': 1, '4': 3, '5': 9, '10': 'options'},
    {'1': 'value', '3': 2, '4': 1, '5': 9, '10': 'value'},
  ],
};

/// Descriptor for `DropdownWidget`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List dropdownWidgetDescriptor = $convert.base64Decode(
    'Cg5Ecm9wZG93bldpZGdldBIYCgdvcHRpb25zGAEgAygJUgdvcHRpb25zEhQKBXZhbHVlGAIgAS'
    'gJUgV2YWx1ZQ==');

@$core.Deprecated('Use gestureAreaWidgetDescriptor instead')
const GestureAreaWidget$json = {
  '1': 'GestureAreaWidget',
  '2': [
    {'1': 'action', '3': 1, '4': 1, '5': 9, '10': 'action'},
  ],
};

/// Descriptor for `GestureAreaWidget`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List gestureAreaWidgetDescriptor = $convert.base64Decode(
    'ChFHZXN0dXJlQXJlYVdpZGdldBIWCgZhY3Rpb24YASABKAlSBmFjdGlvbg==');

@$core.Deprecated('Use overlayWidgetDescriptor instead')
const OverlayWidget$json = {
  '1': 'OverlayWidget',
};

/// Descriptor for `OverlayWidget`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List overlayWidgetDescriptor =
    $convert.base64Decode('Cg1PdmVybGF5V2lkZ2V0');

@$core.Deprecated('Use progressWidgetDescriptor instead')
const ProgressWidget$json = {
  '1': 'ProgressWidget',
  '2': [
    {'1': 'value', '3': 1, '4': 1, '5': 1, '10': 'value'},
  ],
};

/// Descriptor for `ProgressWidget`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List progressWidgetDescriptor = $convert
    .base64Decode('Cg5Qcm9ncmVzc1dpZGdldBIUCgV2YWx1ZRgBIAEoAVIFdmFsdWU=');

@$core.Deprecated('Use fullscreenWidgetDescriptor instead')
const FullscreenWidget$json = {
  '1': 'FullscreenWidget',
  '2': [
    {'1': 'enabled', '3': 1, '4': 1, '5': 8, '10': 'enabled'},
  ],
};

/// Descriptor for `FullscreenWidget`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List fullscreenWidgetDescriptor = $convert.base64Decode(
    'ChBGdWxsc2NyZWVuV2lkZ2V0EhgKB2VuYWJsZWQYASABKAhSB2VuYWJsZWQ=');

@$core.Deprecated('Use keepAwakeWidgetDescriptor instead')
const KeepAwakeWidget$json = {
  '1': 'KeepAwakeWidget',
  '2': [
    {'1': 'enabled', '3': 1, '4': 1, '5': 8, '10': 'enabled'},
  ],
};

/// Descriptor for `KeepAwakeWidget`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List keepAwakeWidgetDescriptor = $convert.base64Decode(
    'Cg9LZWVwQXdha2VXaWRnZXQSGAoHZW5hYmxlZBgBIAEoCFIHZW5hYmxlZA==');

@$core.Deprecated('Use brightnessWidgetDescriptor instead')
const BrightnessWidget$json = {
  '1': 'BrightnessWidget',
  '2': [
    {'1': 'value', '3': 1, '4': 1, '5': 1, '10': 'value'},
  ],
};

/// Descriptor for `BrightnessWidget`. Decode as a `google.protobuf.DescriptorProto`.
final $typed_data.Uint8List brightnessWidgetDescriptor = $convert
    .base64Decode('ChBCcmlnaHRuZXNzV2lkZ2V0EhQKBXZhbHVlGAEgASgBUgV2YWx1ZQ==');
