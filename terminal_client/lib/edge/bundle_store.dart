import 'dart:typed_data';

import 'bundle_store_backend.dart';

/// Durable bundle store backed by platform storage.
class BundleStore {
  BundleStore({BundleStoreBackend? backend})
      : _backend = backend ?? createBundleStoreBackend() {
    _bundles.addAll(_backend.loadAll());
  }

  final BundleStoreBackend _backend;
  final Map<String, Uint8List> _bundles = <String, Uint8List>{};

  Iterable<String> get ids => _bundles.keys;

  void install(String bundleId, Uint8List payload) {
    _bundles[bundleId] = payload;
    _backend.put(bundleId, payload);
  }

  void remove(String bundleId) {
    _bundles.remove(bundleId);
    _backend.remove(bundleId);
  }

  Uint8List? get(String bundleId) => _bundles[bundleId];
}
