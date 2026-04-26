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

class DefaultCapabilityProbe implements CapabilityProbe {
  DefaultCapabilityProbe({
    MediaDeviceInventoryProvider? mediaDeviceInventoryProvider,
  }) : _mediaDeviceKindsProvider = mediaDeviceInventoryProvider ??
            _defaultMediaDeviceInventoryProvider;

  final MediaDeviceInventoryProvider _mediaDeviceKindsProvider;

  @override
  Future<capv1.DeviceCapabilities> probe(CapabilityProbeContext context) async {
    final mediaDevices = await _probeMediaDevices();
    final outputEndpoints = _audioEndpointsForKind(mediaDevices, 'audiooutput');
    final inputEndpoints = _audioEndpointsForKind(mediaDevices, 'audioinput');
    final cameraEndpoints = _cameraEndpoints(mediaDevices);

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
          ..primary = true
          ..screen = (capv1.ScreenCapability()
            ..width = context.screenWidth
            ..height = context.screenHeight
            ..density = context.screenDensity
            ..orientation = context.screenWidth >= context.screenHeight
                ? 'landscape'
                : 'portrait'),
      );

    if (outputEndpoints.isNotEmpty) {
      capabilities.speakers =
          (capv1.AudioOutputCapability()..endpoints.addAll(outputEndpoints));
    }
    if (inputEndpoints.isNotEmpty) {
      capabilities.microphone =
          (capv1.AudioInputCapability()..endpoints.addAll(inputEndpoints));
    }
    if (cameraEndpoints.isNotEmpty) {
      capabilities.camera =
          (capv1.CameraCapability()..endpoints.addAll(cameraEndpoints));
    }

    return capabilities;
  }

  List<capv1.AudioEndpoint> _audioEndpointsForKind(
    List<MediaDeviceDescriptor> mediaDevices,
    String kind,
  ) {
    final endpoints = <capv1.AudioEndpoint>[];
    for (final device in mediaDevices) {
      if (device.kind != kind) {
        continue;
      }
      final endpointId = _trimmedOrNull(device.deviceId);
      if (endpointId == null) {
        continue;
      }
      final endpointName = _trimmedOrNull(device.label);
      final endpoint = capv1.AudioEndpoint()
        ..endpointId = endpointId
        ..available = true;
      if (endpointName != null) {
        endpoint.endpointName = endpointName;
      }
      endpoints.add(
        endpoint,
      );
    }
    return endpoints;
  }

  List<capv1.CameraEndpoint> _cameraEndpoints(
    List<MediaDeviceDescriptor> mediaDevices,
  ) {
    final endpoints = <capv1.CameraEndpoint>[];
    for (final device in mediaDevices) {
      if (device.kind != 'videoinput') {
        continue;
      }
      final endpointId = _trimmedOrNull(device.deviceId);
      if (endpointId == null) {
        continue;
      }
      final endpointName = _trimmedOrNull(device.label);
      final endpoint = capv1.CameraEndpoint()
        ..endpointId = endpointId
        ..available = true;
      if (endpointName != null) {
        endpoint.endpointName = endpointName;
      }
      endpoints.add(
        endpoint,
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

String? _trimmedOrNull(String label) {
  final trimmed = label.trim();
  return trimmed.isEmpty ? null : trimmed;
}
