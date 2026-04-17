import 'dart:typed_data';

import 'bundle_store_backend.dart';

class _MemoryBundleStoreBackend implements BundleStoreBackend {
  final Map<String, Uint8List> _storage = <String, Uint8List>{};

  @override
  Map<String, Uint8List> loadAll() => Map<String, Uint8List>.from(_storage);

  @override
  void put(String bundleId, Uint8List payload) {
    _storage[bundleId] = payload;
  }

  @override
  void remove(String bundleId) {
    _storage.remove(bundleId);
  }
}

BundleStoreBackend createPlatformBundleStoreBackend() =>
    _MemoryBundleStoreBackend();
