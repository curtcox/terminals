import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  test('all six platform directories exist', () {
    const paths = <String>[
      'android',
      'ios',
      'linux',
      'windows',
      'web',
      'macos',
    ];

    for (final path in paths) {
      expect(Directory(path).existsSync(), isTrue,
          reason: '$path directory is required');
    }
  });

  test('android manifest includes required network and media permissions', () {
    final manifest =
        File('android/app/src/main/AndroidManifest.xml').readAsStringSync();

    expect(manifest, contains('android.permission.INTERNET'));
    expect(
        manifest, contains('android.permission.CHANGE_WIFI_MULTICAST_STATE'));
    expect(manifest, contains('android.permission.CAMERA'));
    expect(manifest, contains('android.permission.RECORD_AUDIO'));
  });

  test('ios Info.plist includes network discovery and media usage strings', () {
    final plist = File('ios/Runner/Info.plist').readAsStringSync();

    expect(plist, contains('NSLocalNetworkUsageDescription'));
    expect(plist, contains('NSBonjourServices'));
    expect(plist, contains('NSCameraUsageDescription'));
    expect(plist, contains('NSMicrophoneUsageDescription'));
  });

  test('macOS metadata includes microphone and camera permissions', () {
    final plist = File('macos/Runner/Info.plist').readAsStringSync();
    final debugEntitlements =
        File('macos/Runner/DebugProfile.entitlements').readAsStringSync();
    final releaseEntitlements =
        File('macos/Runner/Release.entitlements').readAsStringSync();

    expect(plist, contains('NSMicrophoneUsageDescription'));
    expect(plist, contains('NSCameraUsageDescription'));
    expect(
        debugEntitlements, contains('com.apple.security.device.audio-input'));
    expect(debugEntitlements, contains('com.apple.security.device.camera'));
    expect(
        releaseEntitlements, contains('com.apple.security.device.audio-input'));
    expect(releaseEntitlements, contains('com.apple.security.device.camera'));
  });
}
