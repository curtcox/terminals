import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/capabilities/probe.dart';

void main() {
  test('probe omits media capabilities when no devices are detected', () async {
    final probe = DefaultCapabilityProbe(
      mediaDeviceInventoryProvider: () async => const <MediaDeviceDescriptor>[],
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
    expect(capabilities.hasHaptics(), isFalse);
    expect(capabilities.hasEdge(), isTrue);
    expect(capabilities.edge.hasRetention(), isFalse);
    expect(capabilities.screen.hasFullscreenSupported(), isFalse);
    expect(capabilities.screen.hasMultiWindowSupported(), isFalse);
    expect(capabilities.screen.hasSafeArea(), isFalse);
    expect(capabilities.displays.first.screen.hasSafeArea(), isFalse);
    expect(capabilities.edge.runtimes, contains('dart'));
    expect(capabilities.edge.operators, contains('monitor.foreground_only'));
    expect(
      capabilities.edge.operators,
      contains('monitor.tier.foreground_only'),
    );
    expect(
      capabilities.edge.operators,
      contains('monitor.lifecycle.foreground'),
    );
  });

  test('probe includes media capabilities when devices are detected', () async {
    final probe = DefaultCapabilityProbe(
      mediaDeviceInventoryProvider: () async => const <MediaDeviceDescriptor>[
        MediaDeviceDescriptor(
          kind: 'audioinput',
          deviceId: 'mic-1',
          label: 'Built-in Microphone',
        ),
        MediaDeviceDescriptor(
          kind: 'videoinput',
          deviceId: 'cam-1',
          label: 'USB Camera',
        ),
        MediaDeviceDescriptor(
          kind: 'audiooutput',
          deviceId: 'spk-1',
          label: 'Bluetooth Speaker',
        ),
      ],
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
    expect(capabilities.hasHaptics(), isTrue);
    expect(capabilities.hasTouch(), isTrue);
    expect(capabilities.touch.supported, isTrue);
    expect(capabilities.touch.hasMaxPoints(), isFalse);
    expect(capabilities.edge.hasRetention(), isFalse);
    expect(capabilities.screen.hasFullscreenSupported(), isFalse);
    expect(capabilities.screen.hasMultiWindowSupported(), isFalse);
    expect(capabilities.screen.hasSafeArea(), isFalse);
    expect(capabilities.displays.first.screen.hasSafeArea(), isFalse);
    expect(capabilities.microphone.channels, 1);
    expect(capabilities.microphone.endpoints, isNotEmpty);
    expect(capabilities.camera.endpoints, isNotEmpty);
    expect(capabilities.speakers.endpoints, isNotEmpty);
    expect(capabilities.displays, isNotEmpty);
  });

  test('probe falls back when media enumeration stalls', () async {
    final probe = DefaultCapabilityProbe(
      mediaDeviceInventoryProvider: () async {
        await Future<void>.delayed(const Duration(seconds: 5));
        return const <MediaDeviceDescriptor>[
          MediaDeviceDescriptor(
            kind: 'audioinput',
            deviceId: 'mic-delayed',
            label: 'Delayed Mic',
          ),
        ];
      },
    );

    final capabilities = await probe.probe(
      const CapabilityProbeContext(
        deviceId: 'device-timeout',
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
  });
}
