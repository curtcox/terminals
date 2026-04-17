import 'dart:typed_data';

import 'bundle_store_backend_stub.dart'
    if (dart.library.io) 'bundle_store_backend_io.dart'
    if (dart.library.html) 'bundle_store_backend_web.dart';

abstract class BundleStoreBackend {
  Future<Map<String, Uint8List>> loadAll();
  Future<void> put(String bundleId, Uint8List payload);
  Future<void> remove(String bundleId);
}

BundleStoreBackend createBundleStoreBackend() =>
    createPlatformBundleStoreBackend();
