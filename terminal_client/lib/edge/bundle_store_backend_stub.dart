import 'dart:typed_data';

import 'bundle_store_backend.dart';

class _MemoryBundleStoreBackend implements BundleStoreBackend {
  final Map<String, Uint8List> _storage = <String, Uint8List>{};

  @override
  Future<Map<String, Uint8List>> loadAll() async =>
      Map<String, Uint8List>.from(_storage);

  @override
  Future<void> put(String bundleId, Uint8List payload) async {
    _storage[bundleId] = payload;
  }

  @override
  Future<void> remove(String bundleId) async {
    _storage.remove(bundleId);
  }
}

BundleStoreBackend createPlatformBundleStoreBackend() =>
    _MemoryBundleStoreBackend();
