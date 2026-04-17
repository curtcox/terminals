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

  test('android manifest includes camera and microphone permissions', () {
    final manifest =
        File('android/app/src/main/AndroidManifest.xml').readAsStringSync();

    expect(manifest, contains('android.permission.CAMERA'));
    expect(manifest, contains('android.permission.RECORD_AUDIO'));
  });

  test('ios Info.plist includes camera and microphone usage strings', () {
    final plist = File('ios/Runner/Info.plist').readAsStringSync();

    expect(plist, contains('NSCameraUsageDescription'));
    expect(plist, contains('NSMicrophoneUsageDescription'));
  });
}
