import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/capabilities/capability_session.dart';
import 'package:terminal_client/capabilities/screen_metrics.dart';
import 'package:terminal_client/gen/terminals/capabilities/v1/capabilities.pb.dart'
    as capv1;
import 'package:terminal_client/gen/terminals/control/v1/control.pb.dart';

void main() {
  test('applyDisplayMetadataToCapabilities populates screen and display', () {
    final capabilities = applyDisplayMetadataToCapabilities(
      capv1.DeviceCapabilities(),
      metrics: ScreenMetrics(
        logicalSize: const Size(390, 844),
        devicePixelRatio: 3,
        safeAreaInsets: const EdgeInsets.fromLTRB(0, 47, 0, 34),
        orientation: 'portrait',
      ),
    );

    expect(capabilities.screen.width, 390);
    expect(capabilities.screen.height, 844);
    expect(capabilities.screen.density, 3);
    expect(capabilities.screen.orientation, 'portrait');
    expect(capabilities.screen.safeArea.top, 47);
    expect(capabilities.displays, hasLength(1));
    expect(capabilities.displays.first.screen.width, 390);
  });

  test('capabilitySignature changes when capabilities change', () {
    final first = capabilitySignature(
      capv1.DeviceCapabilities()..deviceId = 'device-a',
    );
    final second = capabilitySignature(
      capv1.DeviceCapabilities()..deviceId = 'device-b',
    );

    expect(first, isNot(second));
  });

  test('isStaleCapabilityGenerationError detects protocol stale generations',
      () {
    expect(
      isStaleCapabilityGenerationError(
        ControlError()
          ..code = ControlErrorCode.CONTROL_ERROR_CODE_PROTOCOL_VIOLATION
          ..message = 'stale capability generation: expected newer',
      ),
      isTrue,
    );
    expect(
      isStaleCapabilityGenerationError(
        ControlError()
          ..code = ControlErrorCode.CONTROL_ERROR_CODE_PROTOCOL_VIOLATION
          ..message = 'generation is stale',
      ),
      isTrue,
    );
  });

  test('isStaleCapabilityGenerationError rejects unrelated errors', () {
    expect(
      isStaleCapabilityGenerationError(
        ControlError()
          ..code = ControlErrorCode.CONTROL_ERROR_CODE_UNSPECIFIED
          ..message = 'stale capability generation',
      ),
      isFalse,
    );
    expect(
      isStaleCapabilityGenerationError(
        ControlError()
          ..code = ControlErrorCode.CONTROL_ERROR_CODE_PROTOCOL_VIOLATION
          ..message = 'missing device id',
      ),
      isFalse,
    );
  });
}
