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

  test('CapabilitySession starts bootstrap at generation one', () {
    final session = CapabilitySession();

    final publication = session.startBootstrap(
      capv1.DeviceCapabilities()..deviceId = 'device-a',
    );

    expect(publication.generation, 1);
    expect(session.generation, 1);
    expect(session.lastAckGeneration, 0);
    expect(session.lastRegisteredCapabilities?.deviceId, 'device-a');
  });

  test('CapabilitySession publishes only changed capabilities by default', () {
    final session = CapabilitySession();
    session.startBootstrap(capv1.DeviceCapabilities()..deviceId = 'device-a');

    final unchanged = session.publishChange(
      capv1.DeviceCapabilities()..deviceId = 'device-a',
    );
    final changed = session.publishChange(
      capv1.DeviceCapabilities()..deviceId = 'device-b',
    );

    expect(unchanged, isNull);
    expect(changed?.generation, 2);
    expect(session.lastRegisteredCapabilities?.deviceId, 'device-b');
  });

  test('CapabilitySession advances after accepted ack generation', () {
    final session = CapabilitySession();
    session.startBootstrap(capv1.DeviceCapabilities()..deviceId = 'device-a');
    session.observeAckGeneration(7);

    final publication = session.publishChange(
      capv1.DeviceCapabilities()..deviceId = 'device-b',
    );

    expect(publication?.generation, 8);
    expect(session.generation, 8);
    expect(session.lastAckGeneration, 7);
  });

  test('CapabilitySession force republishes unchanged capabilities', () {
    final session = CapabilitySession();
    session.startBootstrap(capv1.DeviceCapabilities()..deviceId = 'device-a');

    final publication = session.publishChange(
      capv1.DeviceCapabilities()..deviceId = 'device-a',
      force: true,
    );

    expect(publication?.generation, 2);
  });

  test('CapabilitySession reset clears tracked generation state', () {
    final session = CapabilitySession();
    session.startBootstrap(capv1.DeviceCapabilities()..deviceId = 'device-a');
    session.observeAckGeneration(1);

    session.reset();

    expect(session.lastRegisteredCapabilities, isNull);
    expect(session.generation, 0);
    expect(session.lastAckGeneration, 0);
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
