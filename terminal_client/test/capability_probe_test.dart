import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/capabilities/probe.dart';

void main() {
  test('probe omits media capabilities when no devices are detected', () async {
    final probe = DefaultCapabilityProbe(
      mediaDeviceKindsProvider: () async => <String>[],
    );

    final capabilities = await probe.probe(
      const CapabilityProbeContext(
        deviceId: 'device-1',
        deviceName: 'Client',
        deviceType: 'desktop',
        platform: 'flutter',
        screenWidth: 1920,
        screenHeight: 1080,
        screenDensity: 2.0,
        touchInputLikely: false,
        targetPlatform: TargetPlatform.macOS,
      ),
    );

    expect(capabilities.hasMicrophone(), isFalse);
    expect(capabilities.hasCamera(), isFalse);
    expect(capabilities.hasSpeakers(), isFalse);
    expect(capabilities.hasEdge(), isFalse);
  });

  test('probe includes media capabilities when devices are detected', () async {
    final probe = DefaultCapabilityProbe(
      mediaDeviceKindsProvider: () async =>
          <String>['audioinput', 'videoinput', 'audiooutput'],
    );

    final capabilities = await probe.probe(
      const CapabilityProbeContext(
        deviceId: 'device-2',
        deviceName: 'Tablet',
        deviceType: 'tablet',
        platform: 'flutter',
        screenWidth: 1080,
        screenHeight: 1920,
        screenDensity: 2.5,
        touchInputLikely: true,
        targetPlatform: TargetPlatform.android,
      ),
    );

    expect(capabilities.hasMicrophone(), isTrue);
    expect(capabilities.hasCamera(), isTrue);
    expect(capabilities.hasSpeakers(), isTrue);
    expect(capabilities.microphone.channels, 1);
  });
}
