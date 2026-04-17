import 'package:flutter/foundation.dart';
import 'package:flutter_webrtc/flutter_webrtc.dart';
import 'package:terminal_client/gen/terminals/capabilities/v1/capabilities.pb.dart'
    as capv1;

typedef MediaDeviceKindsProvider = Future<List<String>> Function();

class CapabilityProbeContext {
  const CapabilityProbeContext({
    required this.deviceId,
    required this.deviceName,
    required this.deviceType,
    required this.platform,
    required this.screenWidth,
    required this.screenHeight,
    required this.screenDensity,
    required this.touchInputLikely,
    required this.targetPlatform,
  });

  final String deviceId;
  final String deviceName;
  final String deviceType;
  final String platform;
  final int screenWidth;
  final int screenHeight;
  final double screenDensity;
  final bool touchInputLikely;
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
    MediaDeviceKindsProvider? mediaDeviceKindsProvider,
  }) : _mediaDeviceKindsProvider =
            mediaDeviceKindsProvider ?? _defaultMediaDeviceKindsProvider;

  final MediaDeviceKindsProvider _mediaDeviceKindsProvider;

  @override
  Future<capv1.DeviceCapabilities> probe(CapabilityProbeContext context) async {
    final mediaKinds = await _probeMediaKinds();
    final hasMicrophone = mediaKinds.contains('audioinput');
    final hasAudioOutput = mediaKinds.contains('audiooutput');
    final hasCamera = mediaKinds.contains('videoinput');
    final hasKeyboard = _isLikelyKeyboardPlatform(context.targetPlatform);
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
        ..touch = context.touchInputLikely);

    if (hasKeyboard) {
      capabilities.keyboard = (capv1.KeyboardCapability()..physical = true);
    }
    capabilities.pointer = (capv1.PointerCapability()
      ..type = context.touchInputLikely ? 'touch' : 'mouse'
      ..hover = !context.touchInputLikely);
    if (context.touchInputLikely) {
      capabilities.touch = (capv1.TouchCapability()
        ..supported = true
        ..maxPoints = 1);
    }
    if (hasAudioOutput) {
      capabilities.speakers = (capv1.AudioOutputCapability()..channels = 2);
    }
    if (hasMicrophone) {
      capabilities.microphone = (capv1.AudioInputCapability()..channels = 1);
    }
    if (hasCamera) {
      capabilities.camera = capv1.CameraCapability();
    }
    capabilities.connectivity = capv1.ConnectivityCapability()
      ..wifiSignalStrength = true;
    capabilities.edge = (capv1.EdgeCapability()
      ..runtimes.addAll(<String>['dart'])
      ..operators.addAll(monitoringTier.operators)
      ..retention = (capv1.EdgeRetentionCapability()
        ..audioSec = 120
        ..videoSec = 120
        ..sensorSec = 600
        ..radioSec = 0));

    return capabilities;
  }

  Future<Set<String>> _probeMediaKinds() async {
    try {
      final kinds = await _mediaDeviceKindsProvider();
      final normalized = kinds
          .map((kind) => kind.trim().toLowerCase())
          .where((kind) => kind.isNotEmpty)
          .toSet();
      return normalized;
    } catch (_) {
      return <String>{};
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

Future<List<String>> _defaultMediaDeviceKindsProvider() async {
  final devices = await navigator.mediaDevices.enumerateDevices();
  return devices
      .map((device) => device.kind)
      .whereType<String>()
      .toList(growable: false);
}

bool _isLikelyKeyboardPlatform(TargetPlatform platform) {
  switch (platform) {
    case TargetPlatform.macOS:
    case TargetPlatform.windows:
    case TargetPlatform.linux:
      return true;
    case TargetPlatform.android:
    case TargetPlatform.iOS:
    case TargetPlatform.fuchsia:
      return false;
  }
}
