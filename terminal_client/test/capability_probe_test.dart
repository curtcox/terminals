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
        targetPlatform: TargetPlatform.macOS,
      ),
    );

    expect(capabilities.hasMicrophone(), isFalse);
    expect(capabilities.hasCamera(), isFalse);
    expect(capabilities.hasSpeakers(), isFalse);
    expect(capabilities.hasHaptics(), isFalse);
    expect(capabilities.hasKeyboard(), isFalse);
    expect(capabilities.hasEdge(), isFalse);
    expect(capabilities.hasPointer(), isFalse);
    expect(capabilities.hasTouch(), isFalse);
    expect(capabilities.screen.hasTouch(), isFalse);
    expect(capabilities.displays.first.screen.hasTouch(), isFalse);
    expect(capabilities.screen.hasFullscreenSupported(), isFalse);
    expect(capabilities.screen.hasMultiWindowSupported(), isFalse);
    expect(capabilities.screen.hasSafeArea(), isFalse);
    expect(capabilities.displays.first.screen.hasSafeArea(), isFalse);
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
        targetPlatform: TargetPlatform.android,
      ),
    );

    expect(capabilities.hasMicrophone(), isTrue);
    expect(capabilities.hasCamera(), isTrue);
    expect(capabilities.hasSpeakers(), isTrue);
    expect(capabilities.hasHaptics(), isFalse);
    expect(capabilities.hasKeyboard(), isFalse);
    expect(capabilities.hasPointer(), isFalse);
    expect(capabilities.hasTouch(), isFalse);
    expect(capabilities.screen.hasTouch(), isFalse);
    expect(capabilities.displays.first.screen.hasTouch(), isFalse);
    expect(capabilities.edge.hasRetention(), isFalse);
    expect(capabilities.screen.hasFullscreenSupported(), isFalse);
    expect(capabilities.screen.hasMultiWindowSupported(), isFalse);
    expect(capabilities.screen.hasSafeArea(), isFalse);
    expect(capabilities.displays.first.screen.hasSafeArea(), isFalse);
    expect(capabilities.microphone.channels, 0);
    expect(capabilities.microphone.endpoints.first.channels, 0);
    expect(
        capabilities.microphone.endpoints.first.hasConnectionType(), isFalse);
    expect(capabilities.speakers.channels, 0);
    expect(capabilities.speakers.endpoints.first.channels, 0);
    expect(capabilities.speakers.endpoints.first.hasConnectionType(), isFalse);
    expect(capabilities.camera.endpoints.first.hasConnectionType(), isFalse);
    expect(capabilities.camera.endpoints.first.hasFacing(), isFalse);
    expect(capabilities.microphone.endpoints.first.hasEndpointName(), isTrue);
    expect(capabilities.speakers.endpoints.first.hasEndpointName(), isTrue);
    expect(capabilities.camera.endpoints.first.hasEndpointName(), isTrue);
    expect(capabilities.microphone.endpoints, isNotEmpty);
    expect(capabilities.camera.endpoints, isNotEmpty);
    expect(capabilities.speakers.endpoints, isNotEmpty);
    expect(capabilities.displays, isNotEmpty);
  });

  test('probe omits endpoint names when device labels are unavailable',
      () async {
    final probe = DefaultCapabilityProbe(
      mediaDeviceInventoryProvider: () async => const <MediaDeviceDescriptor>[
        MediaDeviceDescriptor(
          kind: 'audioinput',
          deviceId: 'mic-blank-label',
          label: '   ',
        ),
        MediaDeviceDescriptor(
          kind: 'videoinput',
          deviceId: 'cam-blank-label',
          label: '',
        ),
        MediaDeviceDescriptor(
          kind: 'audiooutput',
          deviceId: 'spk-blank-label',
          label: '\t',
        ),
      ],
    );

    final capabilities = await probe.probe(
      const CapabilityProbeContext(
        deviceId: 'device-3',
        deviceName: 'Unlabeled Devices',
        deviceType: 'desktop',
        platform: 'flutter',
        screenWidth: 1600,
        screenHeight: 900,
        screenDensity: 2.0,
        targetPlatform: TargetPlatform.macOS,
      ),
    );

    expect(capabilities.hasMicrophone(), isTrue);
    expect(capabilities.hasCamera(), isTrue);
    expect(capabilities.hasSpeakers(), isTrue);
    expect(capabilities.microphone.endpoints.first.hasEndpointName(), isFalse);
    expect(capabilities.speakers.endpoints.first.hasEndpointName(), isFalse);
    expect(capabilities.camera.endpoints.first.hasEndpointName(), isFalse);
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
        targetPlatform: TargetPlatform.macOS,
      ),
    );

    expect(capabilities.hasMicrophone(), isFalse);
    expect(capabilities.hasCamera(), isFalse);
    expect(capabilities.hasSpeakers(), isFalse);
  });

  test('probe omits media endpoints without real device IDs', () async {
    final probe = DefaultCapabilityProbe(
      mediaDeviceInventoryProvider: () async => const <MediaDeviceDescriptor>[
        MediaDeviceDescriptor(
          kind: 'audioinput',
          deviceId: '   ',
          label: 'Blank Mic Id',
        ),
        MediaDeviceDescriptor(
          kind: 'audioinput',
          deviceId: 'mic-1',
          label: 'Built-in Microphone',
        ),
        MediaDeviceDescriptor(
          kind: 'videoinput',
          deviceId: '',
          label: 'Blank Cam Id',
        ),
        MediaDeviceDescriptor(
          kind: 'videoinput',
          deviceId: 'cam-1',
          label: 'USB Camera',
        ),
        MediaDeviceDescriptor(
          kind: 'audiooutput',
          deviceId: '\t',
          label: 'Blank Speaker Id',
        ),
        MediaDeviceDescriptor(
          kind: 'audiooutput',
          deviceId: 'spk-1',
          label: 'Built-in Speakers',
        ),
      ],
    );

    final capabilities = await probe.probe(
      const CapabilityProbeContext(
        deviceId: 'device-ids',
        deviceName: 'Endpoint IDs',
        deviceType: 'desktop',
        platform: 'flutter',
        screenWidth: 1920,
        screenHeight: 1080,
        screenDensity: 2.0,
        targetPlatform: TargetPlatform.macOS,
      ),
    );

    expect(capabilities.hasMicrophone(), isTrue);
    expect(capabilities.hasCamera(), isTrue);
    expect(capabilities.hasSpeakers(), isTrue);

    expect(capabilities.microphone.endpoints.length, 1);
    expect(capabilities.microphone.endpoints.first.endpointId, 'mic-1');

    expect(capabilities.camera.endpoints.length, 1);
    expect(capabilities.camera.endpoints.first.endpointId, 'cam-1');

    expect(capabilities.speakers.endpoints.length, 1);
    expect(capabilities.speakers.endpoints.first.endpointId, 'spk-1');
  });

  test('probe omits media capabilities when kind exists without valid IDs',
      () async {
    final probe = DefaultCapabilityProbe(
      mediaDeviceInventoryProvider: () async => const <MediaDeviceDescriptor>[
        MediaDeviceDescriptor(
          kind: 'audioinput',
          deviceId: '   ',
          label: 'Blank Mic Id',
        ),
        MediaDeviceDescriptor(
          kind: 'videoinput',
          deviceId: '',
          label: 'Blank Cam Id',
        ),
        MediaDeviceDescriptor(
          kind: 'audiooutput',
          deviceId: '\t',
          label: 'Blank Speaker Id',
        ),
      ],
    );

    final capabilities = await probe.probe(
      const CapabilityProbeContext(
        deviceId: 'device-invalid-ids-only',
        deviceName: 'Invalid IDs Only',
        deviceType: 'desktop',
        platform: 'flutter',
        screenWidth: 1920,
        screenHeight: 1080,
        screenDensity: 2.0,
        targetPlatform: TargetPlatform.macOS,
      ),
    );

    expect(capabilities.hasMicrophone(), isFalse);
    expect(capabilities.hasCamera(), isFalse);
    expect(capabilities.hasSpeakers(), isFalse);
  });
}
