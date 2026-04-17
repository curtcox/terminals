import 'dart:typed_data';

import 'bundle_store_backend.dart';

/// Durable bundle store backed by platform storage.
class BundleStore {
  BundleStore._(this._backend);

  static Future<BundleStore> create({BundleStoreBackend? backend}) async {
    final store = BundleStore._(backend ?? createBundleStoreBackend());
    store._bundles.addAll(await store._backend.loadAll());
    return store;
  }

  final BundleStoreBackend _backend;
  final Map<String, Uint8List> _bundles = <String, Uint8List>{};

  Iterable<String> get ids => _bundles.keys;

  Future<void> install(String bundleId, Uint8List payload) async {
    _bundles[bundleId] = payload;
    await _backend.put(bundleId, payload);
  }

  Future<void> remove(String bundleId) async {
    _bundles.remove(bundleId);
    await _backend.remove(bundleId);
  }

  Uint8List? get(String bundleId) => _bundles[bundleId];
}
