// ignore_for_file: avoid_web_libraries_in_flutter, deprecated_member_use

import 'dart:async';
import 'dart:html' as html;
import 'dart:indexed_db' as idb;
import 'dart:typed_data';

import 'bundle_store_backend.dart';

const String _databaseName = 'terminals.edge';
const String _storeName = 'bundles';
const int _databaseVersion = 1;

class _WebBundleStoreBackend implements BundleStoreBackend {
  @override
  Future<Map<String, Uint8List>> loadAll() async {
    final out = <String, Uint8List>{};
    final db = await _openDatabase();
    if (db == null) {
      return out;
    }
    final tx = db.transaction(_storeName, 'readonly');
    final store = tx.objectStore(_storeName);
    await for (final cursor in store.openCursor(autoAdvance: true)) {
      final key = cursor.key;
      final value = cursor.value;
      if (key is String && value is ByteBuffer) {
        out[key] = Uint8List.view(value);
      } else if (key is String && value is Uint8List) {
        out[key] = value;
      }
    }
    await tx.completed;
    return out;
  }

  @override
  Future<void> put(String bundleId, Uint8List payload) async {
    final db = await _openDatabase();
    if (db == null) {
      return;
    }
    final tx = db.transaction(_storeName, 'readwrite');
    tx.objectStore(_storeName).put(payload, bundleId);
    await tx.completed;
  }

  @override
  Future<void> remove(String bundleId) async {
    final db = await _openDatabase();
    if (db == null) {
      return;
    }
    final tx = db.transaction(_storeName, 'readwrite');
    tx.objectStore(_storeName).delete(bundleId);
    await tx.completed;
  }
}

BundleStoreBackend createPlatformBundleStoreBackend() =>
    _WebBundleStoreBackend();

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
