import 'package:terminal_client/capabilities/screen_metrics.dart';
import 'package:terminal_client/gen/terminals/capabilities/v1/capabilities.pb.dart'
    as capv1;
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';

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
