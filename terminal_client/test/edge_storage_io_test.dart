import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:terminal_client/edge/artifact_export.dart';
import 'package:terminal_client/edge/artifact_export_backend_io.dart';
import 'package:terminal_client/edge/bundle_store.dart';
import 'package:terminal_client/edge/bundle_store_backend_io.dart';
import 'package:terminal_client/edge/host.dart';
import 'package:terminal_client/edge/host_state_backend_io.dart';
import 'package:terminal_client/edge/retention.dart';
import 'package:terminal_client/edge/scheduler.dart';

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

    test('edge host reloads active flow state across restart', () async {
      final bundleID = 'pkg/audio.classify/v1';
      final bundlePayload = Uint8List.fromList(<int>[7, 7, 7]);

      final firstStore = await BundleStore.create(
        backend: createIOBundleStoreBackend(rootDir: tempRoot),
      );
      final firstHost = await EdgeHost.create(
        bundleStore: firstStore,
        scheduler: EdgeScheduler(maxCPURealtime: 2, maxMemoryMB: 256),
        retention: RetentionBufferManager(
          audioSec: 10,
          videoSec: 10,
          sensorSec: 10,
          radioSec: 10,
        ),
        stateBackend: createIOEdgeHostStateBackend(rootDir: tempRoot),
      );

      await firstHost.installBundle(bundleID, bundlePayload);
      await firstHost.startFlow('flow-a', bundleId: bundleID);

      final secondStore = await BundleStore.create(
        backend: createIOBundleStoreBackend(rootDir: tempRoot),
      );
      final secondHost = await EdgeHost.create(
        bundleStore: secondStore,
        scheduler: EdgeScheduler(maxCPURealtime: 2, maxMemoryMB: 256),
        retention: RetentionBufferManager(
          audioSec: 10,
          videoSec: 10,
          sensorSec: 10,
          radioSec: 10,
        ),
        stateBackend: createIOEdgeHostStateBackend(rootDir: tempRoot),
      );

      expect(secondHost.activeFlows, contains('flow-a'));
      expect(secondHost.bundleForFlow('flow-a'), bundleID);
    });

    test('edge host ignores corrupt state and prunes flows for removed bundles',
        () async {
      final bundleID = 'pkg/audio.localize/v1';

      final store = await BundleStore.create(
        backend: createIOBundleStoreBackend(rootDir: tempRoot),
      );
      final host = await EdgeHost.create(
        bundleStore: store,
        scheduler: EdgeScheduler(maxCPURealtime: 2, maxMemoryMB: 256),
        retention: RetentionBufferManager(
          audioSec: 10,
          videoSec: 10,
          sensorSec: 10,
          radioSec: 10,
        ),
        stateBackend: createIOEdgeHostStateBackend(rootDir: tempRoot),
      );

      await host.installBundle(bundleID, <int>[1, 2, 3]);
      await host.startFlow('flow-b', bundleId: bundleID);
      await host.removeBundle(bundleID);

      final reloadedStore = await BundleStore.create(
        backend: createIOBundleStoreBackend(rootDir: tempRoot),
      );
      final reloadedHost = await EdgeHost.create(
        bundleStore: reloadedStore,
        scheduler: EdgeScheduler(maxCPURealtime: 2, maxMemoryMB: 256),
        retention: RetentionBufferManager(
          audioSec: 10,
          videoSec: 10,
          sensorSec: 10,
          radioSec: 10,
        ),
        stateBackend: createIOEdgeHostStateBackend(rootDir: tempRoot),
      );
      expect(reloadedHost.activeFlows, isNot(contains('flow-b')));

      final stateFile = File('${tempRoot.path}/host_state.json');
      stateFile.writeAsStringSync('{corrupt-json', flush: true);

      final fromCorruptStore = await BundleStore.create(
        backend: createIOBundleStoreBackend(rootDir: tempRoot),
      );
      final fromCorruptHost = await EdgeHost.create(
        bundleStore: fromCorruptStore,
        scheduler: EdgeScheduler(maxCPURealtime: 2, maxMemoryMB: 256),
        retention: RetentionBufferManager(
          audioSec: 10,
          videoSec: 10,
          sensorSec: 10,
          radioSec: 10,
        ),
        stateBackend: createIOEdgeHostStateBackend(rootDir: tempRoot),
      );
      expect(fromCorruptHost.activeFlows, isEmpty);
    });

    test('edge host patches active flow bundle assignments', () async {
      final initialBundleID = 'pkg/audio.initial/v1';
      final nextBundleID = 'pkg/audio.next/v1';

      final store = await BundleStore.create(
        backend: createIOBundleStoreBackend(rootDir: tempRoot),
      );
      final host = await EdgeHost.create(
        bundleStore: store,
        scheduler: EdgeScheduler(maxCPURealtime: 2, maxMemoryMB: 256),
        retention: RetentionBufferManager(
          audioSec: 10,
          videoSec: 10,
          sensorSec: 10,
          radioSec: 10,
        ),
        stateBackend: createIOEdgeHostStateBackend(rootDir: tempRoot),
      );

      await host.installBundle(initialBundleID, <int>[1]);
      await host.installBundle(nextBundleID, <int>[2]);
      await host.startFlow('flow-c', bundleId: initialBundleID);

      await host.patchFlow('flow-c', bundleId: nextBundleID);
      expect(host.bundleForFlow('flow-c'), nextBundleID);

      await expectLater(
        () => host.patchFlow('missing-flow', bundleId: nextBundleID),
        throwsA(isA<StateError>()),
      );
    });
  });
}
