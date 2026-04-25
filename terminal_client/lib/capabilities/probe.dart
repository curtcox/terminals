import 'package:flutter/foundation.dart';
import 'package:flutter_webrtc/flutter_webrtc.dart';
import 'package:terminal_client/gen/terminals/capabilities/v1/capabilities.pb.dart'
    as capv1;

typedef MediaDeviceInventoryProvider = Future<List<MediaDeviceDescriptor>>
    Function();
const Duration _mediaProbeTimeout = Duration(milliseconds: 750);

class MediaDeviceDescriptor {
  const MediaDeviceDescriptor({
    required this.kind,
    required this.deviceId,
    required this.label,
  });

  final String kind;
  final String deviceId;
  final String label;
}

class CapabilityProbeContext {
  const CapabilityProbeContext({
    required this.deviceId,
    required this.deviceName,
    required this.deviceType,
    required this.platform,
    required this.screenWidth,
    required this.screenHeight,
    required this.screenDensity,
    required this.targetPlatform,
  });

  final String deviceId;
  final String deviceName;
  final String deviceType;
  final String platform;
  final int screenWidth;
  final int screenHeight;
  final double screenDensity;
  final TargetPlatform targetPlatform;
}

abstract class CapabilityProbe {
  Future<capv1.DeviceCapabilities> probe(CapabilityProbeContext context);
}

class MonitoringSupportTier {
  const MonitoringSupportTier({
    required this.supportTier,
    required this.operators,
  });

  final String supportTier;
  final List<String> operators;
}

class DefaultCapabilityProbe implements CapabilityProbe {
  DefaultCapabilityProbe({
    MediaDeviceInventoryProvider? mediaDeviceInventoryProvider,
  }) : _mediaDeviceKindsProvider = mediaDeviceInventoryProvider ??
            _defaultMediaDeviceInventoryProvider;

  final MediaDeviceInventoryProvider _mediaDeviceKindsProvider;

  @override
  Future<capv1.DeviceCapabilities> probe(CapabilityProbeContext context) async {
    final mediaDevices = await _probeMediaDevices();
    final mediaKinds = mediaDevices.map((device) => device.kind).toSet();
    final hasMicrophone = mediaKinds.contains('audioinput');
    final hasAudioOutput = mediaKinds.contains('audiooutput');
    final hasCamera = mediaKinds.contains('videoinput');
    final monitoringTier = _monitoringSupportTierForPlatform(
      context.targetPlatform,
    );

    final capabilities = capv1.DeviceCapabilities()
      ..deviceId = context.deviceId
      ..identity = (capv1.DeviceIdentity()
        ..deviceName = context.deviceName
        ..deviceType = context.deviceType
        ..platform = context.platform)
      ..screen = (capv1.ScreenCapability()
        ..width = context.screenWidth
        ..height = context.screenHeight
        ..density = context.screenDensity
        ..orientation = context.screenWidth >= context.screenHeight
            ? 'landscape'
            : 'portrait')
      ..displays.add(
        capv1.DisplayCapability()
          ..displayId = 'main'
          ..displayName = 'Primary Display'
          ..primary = true
          ..screen = (capv1.ScreenCapability()
            ..width = context.screenWidth
            ..height = context.screenHeight
            ..density = context.screenDensity
            ..orientation = context.screenWidth >= context.screenHeight
                ? 'landscape'
                : 'portrait'),
      );

    if (hasAudioOutput) {
      final outputEndpoints =
          _audioEndpointsForKind(mediaDevices, 'audiooutput');
      capabilities.speakers =
          (capv1.AudioOutputCapability()..endpoints.addAll(outputEndpoints));
    }
    if (hasMicrophone) {
      final inputEndpoints = _audioEndpointsForKind(mediaDevices, 'audioinput');
      capabilities.microphone =
          (capv1.AudioInputCapability()..endpoints.addAll(inputEndpoints));
    }
    if (hasCamera) {
      capabilities.camera = (capv1.CameraCapability()
        ..endpoints.addAll(_cameraEndpoints(mediaDevices)));
    }
    capabilities.edge = (capv1.EdgeCapability()
      ..runtimes.addAll(<String>['dart'])
      ..operators.addAll(monitoringTier.operators));

    return capabilities;
  }

  List<capv1.AudioEndpoint> _audioEndpointsForKind(
    List<MediaDeviceDescriptor> mediaDevices,
    String kind,
  ) {
    final endpoints = <capv1.AudioEndpoint>[];
    var fallbackIndex = 0;
    for (final device in mediaDevices) {
      if (device.kind != kind) {
        continue;
      }
      final endpointId = device.deviceId.trim().isEmpty
          ? '$kind-$fallbackIndex'
          : device.deviceId.trim();
      fallbackIndex += 1;
      endpoints.add(
        capv1.AudioEndpoint()
          ..endpointId = endpointId
          ..endpointName = _endpointLabelOrDefault(
            device.label,
            kind == 'audioinput' ? 'Microphone' : 'Speaker',
            fallbackIndex,
          )
          ..connectionType = _connectionTypeForEndpoint(device.label)
          ..available = true,
      );
    }
    return endpoints;
  }

  List<capv1.CameraEndpoint> _cameraEndpoints(
    List<MediaDeviceDescriptor> mediaDevices,
  ) {
    final endpoints = <capv1.CameraEndpoint>[];
    var fallbackIndex = 0;
    for (final device in mediaDevices) {
      if (device.kind != 'videoinput') {
        continue;
      }
      final endpointId = device.deviceId.trim().isEmpty
          ? 'camera-$fallbackIndex'
          : device.deviceId.trim();
      fallbackIndex += 1;
      endpoints.add(
        capv1.CameraEndpoint()
          ..endpointId = endpointId
          ..endpointName = _endpointLabelOrDefault(
            device.label,
            'Camera',
            fallbackIndex,
          )
          ..connectionType = _connectionTypeForEndpoint(device.label)
          ..facing = _cameraFacingForLabel(device.label)
          ..available = true,
      );
    }
    return endpoints;
  }

  Future<List<MediaDeviceDescriptor>> _probeMediaDevices() async {
    try {
      final devices = await _mediaDeviceKindsProvider().timeout(
        _mediaProbeTimeout,
        onTimeout: () => const <MediaDeviceDescriptor>[],
      );
      final normalized = <MediaDeviceDescriptor>[];
      for (final device in devices) {
        final kind = device.kind.trim().toLowerCase();
        if (kind.isEmpty) {
          continue;
        }
        normalized.add(
          MediaDeviceDescriptor(
            kind: kind,
            deviceId: device.deviceId,
            label: device.label,
          ),
        );
      }
      return normalized;
    } catch (_) {
      return <MediaDeviceDescriptor>[];
    }
  }
}

MonitoringSupportTier _monitoringSupportTierForPlatform(
  TargetPlatform platform,
) {
  switch (platform) {
    case TargetPlatform.android:
    case TargetPlatform.iOS:
    case TargetPlatform.macOS:
    case TargetPlatform.windows:
    case TargetPlatform.linux:
    case TargetPlatform.fuchsia:
      return const MonitoringSupportTier(
        supportTier: 'foreground_only',
        operators: <String>[
          'monitor.foreground_only',
          'monitor.tier.foreground_only',
          'monitor.lifecycle.foreground',
        ],
      );
  }
}

Future<List<MediaDeviceDescriptor>>
    _defaultMediaDeviceInventoryProvider() async {
  final devices = await navigator.mediaDevices.enumerateDevices();
  return devices
      .map(
        (device) => MediaDeviceDescriptor(
          kind: device.kind ?? '',
          deviceId: device.deviceId,
          label: device.label,
        ),
      )
      .toList(growable: false);
}

String _endpointLabelOrDefault(String label, String prefix, int index) {
  final trimmed = label.trim();
  if (trimmed.isNotEmpty) {
    return trimmed;
  }
  return '$prefix $index';
}

String _connectionTypeForEndpoint(String label) {
  final normalized = label.toLowerCase();
  if (normalized.contains('bluetooth') || normalized.contains('bt')) {
    return 'bluetooth';
  }
  if (normalized.contains('usb')) {
    return 'usb';
  }
  if (normalized.contains('hdmi')) {
    return 'hdmi';
  }
  if (normalized.contains('external')) {
    return 'external';
  }
  return 'built_in';
}

String _cameraFacingForLabel(String label) {
  final normalized = label.toLowerCase();
  if (normalized.contains('back') || normalized.contains('rear')) {
    return 'back';
  }
  if (normalized.contains('front')) {
    return 'front';
  }
  return 'unknown';
}
