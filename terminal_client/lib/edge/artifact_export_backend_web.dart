// ignore_for_file: avoid_web_libraries_in_flutter, deprecated_member_use

import 'dart:async';
import 'dart:html' as html;
import 'dart:indexed_db' as idb;
import 'dart:typed_data';

import 'artifact_export_backend.dart';

const String _databaseName = 'terminals.edge';
const String _storeName = 'artifacts';
const int _databaseVersion = 2;

class _WebArtifactExportBackend implements ArtifactExportBackend {
  @override
  Future<Uint8List?> read(String artifactId) async {
    final db = await _openDatabase();
    if (db == null) {
      return null;
    }
    final tx = db.transaction(_storeName, 'readonly');
    final value = await tx.objectStore(_storeName).getObject(artifactId);
    await tx.completed;
    if (value is ByteBuffer) {
      return Uint8List.view(value);
    }
    if (value is Uint8List) {
      return value;
    }
    if (value is List<int>) {
      return Uint8List.fromList(value);
    }
    return null;
  }

  @override
  Future<void> write(String artifactId, Uint8List payload) async {
    final db = await _openDatabase();
    if (db == null) {
      return;
    }
    final tx = db.transaction(_storeName, 'readwrite');
    tx.objectStore(_storeName).put(payload, artifactId);
    await tx.completed;
  }
}

ArtifactExportBackend createPlatformArtifactExportBackend() =>
    _WebArtifactExportBackend();

Future<idb.Database?> _openDatabase() async {
  final indexedDb = html.window.indexedDB;
  if (indexedDb == null) {
    return null;
  }
  try {
    return await indexedDb.open(
      _databaseName,
      version: _databaseVersion,
      onUpgradeNeeded: (event) {
        final target = event.target;
        if (target is! idb.Request) {
          return;
        }
        final db = target.result;
        if (db is! idb.Database) {
          return;
        }
        if (!(db.objectStoreNames?.contains(_storeName) ?? false)) {
          db.createObjectStore(_storeName);
        }
      },
    );
  } catch (_) {
    return null;
  }
}
