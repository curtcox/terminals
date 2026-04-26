import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/edge/artifact_export.dart';
import 'package:terminal_client/edge/artifact_export_backend_io.dart';
import 'package:terminal_client/edge/bundle_store.dart';
import 'package:terminal_client/edge/bundle_store_backend_io.dart';

void main() {
  group('edge IO storage durability', () {
    late Directory tempRoot;

    setUp(() {
      tempRoot = Directory.systemTemp.createTempSync('terminals-edge-');
    });

    tearDown(() {
      if (tempRoot.existsSync()) {
        tempRoot.deleteSync(recursive: true);
      }
    });

    test('bundle store reload decodes encoded bundle IDs', () async {
      final bundleID = 'pkg/audio.onset/v1';
      final payload = Uint8List.fromList(<int>[1, 2, 3, 4]);

      final firstStore = await BundleStore.create(
        backend: createIOBundleStoreBackend(rootDir: tempRoot),
      );
      await firstStore.install(bundleID, payload);

      final secondStore = await BundleStore.create(
        backend: createIOBundleStoreBackend(rootDir: tempRoot),
      );

      expect(secondStore.ids, contains(bundleID));
      expect(secondStore.get(bundleID), payload);
    });

    test('bundle store ignores corrupt persisted files and keeps valid entries',
        () async {
      final bundlesDir = Directory('${tempRoot.path}/bundles')
        ..createSync(recursive: true);
      File('${bundlesDir.path}/${Uri.encodeComponent('good/id')}.b64')
          .writeAsStringSync(base64Encode(<int>[9, 8, 7]), flush: true);
      File('${bundlesDir.path}/${Uri.encodeComponent('bad/id')}.b64')
          .writeAsStringSync('not-base64!!', flush: true);

      final store = await BundleStore.create(
        backend: createIOBundleStoreBackend(rootDir: tempRoot),
      );

      expect(store.ids, contains('good/id'));
      expect(store.ids, isNot(contains('bad/id')));
      expect(store.get('good/id'), Uint8List.fromList(<int>[9, 8, 7]));
    });

    test('artifact exporter persists and reloads IDs with separators',
        () async {
      final artifactID = 'play_audio/route:device-a';
      final payload = Uint8List.fromList(<int>[4, 3, 2, 1]);

      final firstExporter = DurableArtifactExporter(
        backend: createIOArtifactExportBackend(rootDir: tempRoot),
      );
      await firstExporter.save(artifactID, payload);

      final secondExporter = DurableArtifactExporter(
        backend: createIOArtifactExportBackend(rootDir: tempRoot),
      );
      final loaded = await secondExporter.exportByID(artifactID);

      expect(loaded, payload);
    });
  });
}
