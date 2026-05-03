import 'package:terminal_client/capabilities/screen_metrics.dart';
import 'package:terminal_client/gen/terminals/capabilities/v1/capabilities.pb.dart'
    as capv1;
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';

class CapabilityPublication {
  const CapabilityPublication({
    required this.generation,
    required this.capabilities,
  });

  final int generation;
  final capv1.DeviceCapabilities capabilities;
}

class CapabilitySession {
  capv1.DeviceCapabilities? _lastRegisteredCapabilities;
  int _generation = 0;
  int _lastAckGeneration = 0;
  String _lastSignature = '';

  capv1.DeviceCapabilities? get lastRegisteredCapabilities =>
      _lastRegisteredCapabilities;

  int get generation => _generation;

  int get lastAckGeneration => _lastAckGeneration;

  void reset() {
    _lastRegisteredCapabilities = null;
    _generation = 0;
    _lastAckGeneration = 0;
    _lastSignature = '';
  }

  CapabilityPublication startBootstrap(
    capv1.DeviceCapabilities capabilities,
  ) {
    return _record(
      capabilities,
      generation: 1,
    );
  }

  CapabilityPublication? publishChange(
    capv1.DeviceCapabilities capabilities, {
    bool force = false,
  }) {
    final nextSignature = capabilitySignature(capabilities);
    if (!force && nextSignature == _lastSignature) {
      return null;
    }
    final nextGeneration = _max(_generation + 1, _lastAckGeneration + 1);
    return _record(
      capabilities,
      generation: nextGeneration,
      signature: nextSignature,
    );
  }

  void observeAckGeneration(int acceptedGeneration) {
    _lastAckGeneration = acceptedGeneration;
  }

  CapabilityPublication _record(
    capv1.DeviceCapabilities capabilities, {
    required int generation,
    String? signature,
  }) {
    final nextCapabilities = capabilities.deepCopy();
    _lastRegisteredCapabilities = nextCapabilities;
    _generation = generation;
    _lastSignature = signature ?? capabilitySignature(nextCapabilities);
    return CapabilityPublication(
      generation: _generation,
      capabilities: nextCapabilities,
    );
  }
}

int _max(int first, int second) => first > second ? first : second;

String capabilitySignature(capv1.DeviceCapabilities capabilities) {
  return capabilities.writeToBuffer().join(',');
}

capv1.DeviceCapabilities applyDisplayMetadataToCapabilities(
  capv1.DeviceCapabilities capabilities, {
  required ScreenMetrics metrics,
}) {
  final screen = capabilities.hasScreen()
      ? capabilities.screen
      : (capabilities.screen = capv1.ScreenCapability());
  screen
    ..width = metrics.logicalSize.width.round()
    ..height = metrics.logicalSize.height.round()
    ..density = metrics.devicePixelRatio
    ..orientation = metrics.orientation
    ..safeArea = (screen.hasSafeArea() ? screen.safeArea : capv1.Insets())
    ..safeArea.left = metrics.safeAreaInsets.left.round()
    ..safeArea.top = metrics.safeAreaInsets.top.round()
    ..safeArea.right = metrics.safeAreaInsets.right.round()
    ..safeArea.bottom = metrics.safeAreaInsets.bottom.round();

  if (capabilities.displays.isEmpty) {
    capabilities.displays.add(capv1.DisplayCapability());
  }
  final display = capabilities.displays.first;
  display.screen = screen.deepCopy();
  return capabilities;
}

bool isStaleCapabilityGenerationError(ControlError error) {
  if (error.code != ControlErrorCode.CONTROL_ERROR_CODE_PROTOCOL_VIOLATION) {
    return false;
  }
  final message = error.message.toLowerCase();
  return message.contains('stale capability generation') ||
      (message.contains('generation') && message.contains('stale'));
}
